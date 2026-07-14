# Kế hoạch triển khai chi tiết - Khắc phục logic tự sửa đổi index trong Transmuter & Bổ sung đề xuất trên UI

Kế hoạch này được chuẩn bị bởi Muscle (Chief Engineer) nhằm thực hiện nhiệm vụ chỉnh sửa logic kiểm tra, cảnh báo index `_source_id` tại runtime của Transmuter và bổ sung cơ chế đề xuất index tại tầng Governance API/NATS.

## 1. Các file thay đổi & Nội dung chỉnh sửa

### Bước 1: Sửa đổi `transmuter.go`
- **Đường dẫn:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go`
- **Nội dung:** Gỡ bỏ các dòng code drop/create index `CONCURRENTLY` ngầm trong hàm `ensureShadowSourceIDIndex`. Thay vào đó, sau khi kiểm tra index nếu phát hiện thiếu hoặc invalid, cập nhật cache trạng thái để tránh spam và chỉ ghi nhận log cảnh báo (`Warn`).
- **Mã nguồn thay đổi:**
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

### Bước 2: Sửa đổi `index_manager.go`
- **Đường dẫn:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_manager.go`
- **Nội dung:**
  1. Thêm struct `IndexRecommendation`.
  2. Thêm hàm `GetRecommendations` thực hiện kiểm tra 2 loại index: index trên `_source_id` và partial index trên `_deleted`.
- **Mã nguồn thay đổi:**
  ```go
  // Thêm struct IndexRecommendation
  type IndexRecommendation struct {
  	IndexName   string   `json:"index_name"`
  	Columns     []string `json:"columns"`
  	IsUnique    bool     `json:"is_unique"`
  	IsPartial   bool     `json:"is_partial"`
  	WhereClause string   `json:"where_clause"`
  	Description string   `json:"description"`
  }

  // Thêm GetRecommendations vào IndexManager
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

### Bước 3: Sửa đổi `index_handler.go`
- **Đường dẫn:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/governance/index_handler.go`
- **Nội dung:** Trong hàm `HandleIntrospectIndexes`, gọi hàm `GetRecommendations` để lấy các đề xuất index còn thiếu và đính kèm vào payload phản hồi NATS JSON dưới khóa `recommendations`.
- **Mã nguồn thay đổi:**
  ```go
  	// Tính toán các đề xuất index thiếu
  	recommendations := h.indexManager.GetRecommendations(ctx, targetDB, payload.Schema, payload.Table, indexes)

  	respData, _ := json.Marshal(map[string]interface{}{
  		"command":         "introspect-indexes",
  		"status":          "success",
  		"indexes":         indexes,
  		"recommendations": recommendations,
  	})
  ```

### Bước 4: Viết unit test mới cho logic recommendations
- **Đường dẫn:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_manager_test.go`
- **Nội dung:** Bổ sung `TestIndexManager_GetRecommendations` kiểm tra các trường hợp: thiếu cả 2 index, đã có 1 trong 2 index hợp lệ, và đã có đầy đủ index (không trả về đề xuất nào).

## 2. Kế hoạch kiểm thử & Đánh giá (DoD Verification)
1. **Chạy tests package master:**
   `go test -v ./internal/service/master/...`
   Để kiểm tra xem thay đổi trong transmuter không làm hỏng logic của module master, đồng thời xem các test cũ có bị ảnh hưởng hay không.
2. **Chạy tests package governance:**
   `go test -v ./internal/service/governance/...`
   Đặc biệt là test mới `TestIndexManager_GetRecommendations` phải biên dịch thành công và Pass.
3. **Chạy tests package handler:**
   `go test -v ./internal/handler/...`
   Đảm bảo `index_handler.go` biên dịch thành công và phản hồi cấu trúc JSON đúng.
4. **Kiểm tra linter và biên dịch chung:**
   Đảm bảo toàn bộ codebase biên dịch thành công mà không có lỗi cú pháp hay import.
