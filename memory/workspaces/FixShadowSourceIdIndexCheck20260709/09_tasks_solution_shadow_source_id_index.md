# Giải pháp kỹ thuật chi tiết - Khắc phục logic tự sửa đổi index trong Transmuter & Bổ sung đề xuất trên UI

## 1. Các file cần thay đổi

### File 1: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go`

Gỡ bỏ hoàn toàn phần code tự động thực thi DDL `DROP INDEX CONCURRENTLY` và `CREATE INDEX CONCURRENTLY` trong hàm `ensureShadowSourceIDIndex`. Thay đổi để chỉ kiểm tra và ghi nhận cảnh báo (`Warn` log):

```go
func (t *TransmuterModule) ensureShadowSourceIDIndex(ctx context.Context, shadowDB *gorm.DB, row *masterBindingRuntime) {
	if shadowDB.Dialector.Name() != "postgres" {
		return
	}

	indexName := fmt.Sprintf("idx_%s_source_id", row.ShadowTable)

	// RLock để check cache xem đã kiểm tra chưa
	t.mu.RLock()
	var isEnsured bool
	if t.ensuredShadowIndexes != nil {
		isEnsured = t.ensuredShadowIndexes[indexName]
	}
	t.mu.RUnlock()
	if isEnsured {
		return
	}

	var validCount int64
	errValid := shadowDB.WithContext(ctx).Raw(`
		SELECT COUNT(*) 
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = ? AND t.relname = ? AND c.relname = ? AND i.indisvalid = true`,
		row.ShadowSchema, row.ShadowTable, indexName).Scan(&validCount).Error

	if errValid == nil && validCount > 0 {
		// Index hợp lệ đã tồn tại, cập nhật cache và bỏ qua
		t.mu.Lock()
		if t.ensuredShadowIndexes == nil {
			t.ensuredShadowIndexes = make(map[string]bool)
		}
		t.ensuredShadowIndexes[indexName] = true
		t.mu.Unlock()
		return
	}

	if errValid == nil {
		// Đánh dấu cache để tránh spam log cảnh báo trong các lần transmute tiếp theo
		t.mu.Lock()
		if t.ensuredShadowIndexes == nil {
			t.ensuredShadowIndexes = make(map[string]bool)
		}
		t.ensuredShadowIndexes[indexName] = true
		t.mu.Unlock()

		// Chỉ ghi log cảnh báo, tuyệt đối KHÔNG chạy DDL gây lock-storm tại runtime
		t.logger.Warn("transmuter: missing or invalid core index on _source_id. Please create it manually from CMS UI beforehand to prevent deadlocks and lock contention under sync load",
			zap.String("schema", row.ShadowSchema),
			zap.String("table", row.ShadowTable),
			zap.String("index", indexName))
	}
}
```

### File 2: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_manager.go`

Bổ sung cấu trúc `IndexRecommendation` và phương thức `GetRecommendations` vào `IndexManager` để tự động kiểm tra và gợi ý các index quan trọng còn thiếu (index `_source_id` cho transmuter và index `_deleted` cho CountDeletedRows):

```go
// Thêm struct IndexRecommendation vào index_manager.go
type IndexRecommendation struct {
	IndexName   string   `json:"index_name"`
	Columns     []string `json:"columns"`
	IsUnique    bool     `json:"is_unique"`
	IsPartial   bool     `json:"is_partial"`
	WhereClause string   `json:"where_clause"`
	Description string   `json:"description"`
}

func (im *IndexManager) GetRecommendations(ctx context.Context, db *gorm.DB, schema, table string, indexes []IndexInfo) []IndexRecommendation {
	var recs []IndexRecommendation

	// 1. Kiểm tra index trên _source_id (bắt buộc cho transmuter)
	sourceIdIndexName := fmt.Sprintf("idx_%s_source_id", table)
	hasSourceIdIndex := false
	for _, idx := range indexes {
		if idx.IndexName == sourceIdIndexName && idx.IsValid {
			hasSourceIdIndex = true
			break
		}
	}
	if !hasSourceIdIndex {
		recs = append(recs, IndexRecommendation{
			IndexName:   sourceIdIndexName,
			Columns:     []string{"_source_id"},
			IsUnique:    false,
			IsPartial:   false,
			Description: "Index cốt lõi trên cột _source_id bắt buộc phải có để transmuter hoạt động đồng bộ hiệu năng cao.",
		})
	}

	// 2. Kiểm tra partial index trên _deleted (tối ưu hóa CountDeletedRows cho Recon)
	deletedIndexName := fmt.Sprintf("idx_%s_deleted_partial", table)
	hasDeletedIndex := false
	for _, idx := range indexes {
		// Chấp nhận cả idx_..._deleted_partial hoặc idx_...__deleted (do người dùng tự tạo từ đề xuất trước)
		if (idx.IndexName == deletedIndexName || idx.IndexName == fmt.Sprintf("idx_%s__deleted", table)) && idx.IsValid {
			hasDeletedIndex = true
			break
		}
	}
	if !hasDeletedIndex {
		recs = append(recs, IndexRecommendation{
			IndexName:   deletedIndexName,
			Columns:     []string{"_deleted"},
			IsUnique:    false,
			IsPartial:   true,
			WhereClause: "_deleted = true",
			Description: "Tối ưu CountDeletedRows: Tạo partial index trên cột _deleted để tối ưu hóa truy vấn đối soát dòng đã xóa cho Recon.",
		})
	}

	return recs
}
```

### File 3: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/governance/index_handler.go`

Cập nhật `HandleIntrospectIndexes` để trả về thêm mảng `recommendations` trong payload kết quả NATS:

```go
	indexes, err := h.indexManager.ListIndexes(ctx, targetDB, payload.Schema, payload.Table)
	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "introspect-indexes", Status: "error", Error: err.Error()})
		return
	}

	// Tính toán các đề xuất index thiếu
	recommendations := h.indexManager.GetRecommendations(ctx, targetDB, payload.Schema, payload.Table, indexes)

	respData, _ := json.Marshal(map[string]interface{}{
		"command":         "introspect-indexes",
		"status":          "success",
		"indexes":         indexes,
		"recommendations": recommendations,
	})
	h.NatsPublish(msg, "cdc.result.introspect-indexes", respData)
```
