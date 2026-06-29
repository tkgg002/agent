# Solution Design: Metadata Cascade Masking Scan Fix

Tài liệu này mô tả chi tiết giải pháp kỹ thuật cho 6 vấn đề/bug trong workspace `bug-metadata-cascade-masking-scan-fix-2026-06-23` thuộc dự án `data-hub`.

---

## 1. Vấn đề 1: Bỏ cascade is_active cho shadow_binding
- **Mục tiêu**: Loại bỏ việc tự động cập nhật `is_active` của `shadow_binding` khi cập nhật `is_active` của `source_object`.
- **Giải pháp**: 
  - Khôi phục file [source_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go) về trạng thái sạch sẽ (đã thực hiện).
  - Loại bỏ block code xử lý cascade `is_active` cho `shadow_binding` trong hàm `UpdateMetadata` tại `source_repo_gorm.go`.
- **Chi tiết thay đổi**:
```go
// internal/infra/persistence/source/source_repo_gorm.go
func (r *sourceRepoGorm) UpdateMetadata(ctx context.Context, id int64, updates map[string]interface{}) (int64, error) {
	var rowsAffected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Table("cdc_system.source_object_registry").
			Where("id = ?", id).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ports.ErrRecordNotFound
		}
		rowsAffected = res.RowsAffected

		// Cascade shadow updates removed to keep independent status
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}
```

---

## 2. Vấn đề 2: Cột Shadow hiển thị sai ở snapshot-monitor
- **Mục tiêu**: Sửa query `ListSnapshotProgress` để join chính xác qua `shadow_binding_id` (fallback về active shadow binding mới nhất của `source_object_id` nếu `shadow_binding_id` null).
- **Giải pháp**: Thay thế join cũ bằng `LEFT JOIN LATERAL` để tăng hiệu năng và đảm bảo tính chính xác.
- **Chi tiết thay đổi**:
```go
// internal/infra/persistence/scheduler/snapshot_progress_read_repo_gorm.go
// Trong hàm ListSnapshotProgress, sửa câu SQL baseSQL:
	baseSQL := fmt.Sprintf(`
		SELECT
			sp.*,
			so.source_database,
			so.source_object_name AS source_table,
			cr.connection_code    AS source_connection_code,
			sb.shadow_schema,
			sb.shadow_table
		FROM cdc_system.snapshot_progress sp
		JOIN cdc_system.source_object_registry so ON so.id = sp.source_object_id
		LEFT JOIN cdc_system.connection_registry cr ON cr.id = so.source_connection_id
		LEFT JOIN LATERAL (
			SELECT s.shadow_schema, s.shadow_table
			FROM cdc_system.shadow_binding s
			WHERE (sp.shadow_binding_id IS NOT NULL AND s.id = sp.shadow_binding_id)
			   OR (sp.shadow_binding_id IS NULL AND s.source_object_id = sp.source_object_id AND s.is_active = TRUE)
			ORDER BY s.id DESC
			LIMIT 1
		) sb ON TRUE
		%s
	`, whereSQL)
```

---

## 3. Vấn đề 3 & 4: Masking Strategy không chạy khi snapshot & upstream
- **Mục tiêu**: Loại bỏ việc rò rỉ rule của clone khác trong cache của mapping và masking bằng cách lọc chính xác `v2.ShadowBindingID == bindingID`.
- **Giải pháp**: Thay đổi điều kiện lọc rules trong `ReloadAll` tại `metadata_registry_service.go` để loại bỏ các rules có `ShadowBindingID == nil` (rule chung/chưa scoped) hoặc khác `bindingID`.
- **Chi tiết thay đổi**:
```go
// internal/service/source/metadata_registry_service.go
// Trong hàm ReloadAll:

// 1. Khi nạp mappingCache:
		for _, route := range routes {
			if route.ShadowBinding == nil {
				continue
			}
			bindingID := route.ShadowBinding.ID
			for _, v2 := range v2Rules {
				// Sửa đổi từ: v2.ShadowBindingID != nil && *v2.ShadowBindingID != bindingID
				// Thành: filter chính xác ShadowBindingID khớp với bindingID
				if v2.ShadowBindingID == nil || *v2.ShadowBindingID != bindingID {
					continue
				}
				rs.mappingCache[bindingID] = append(rs.mappingCache[bindingID], convertV2ToLegacyRule(v2, src.SourceObjectName))
			}
		}

// 2. Khi nạp maskMapCache:
		if src != nil {
			v2Rules := rulesBySource[src.ID]
			for _, v2 := range v2Rules {
				// Sửa đổi tương tự:
				if v2.ShadowBindingID == nil || *v2.ShadowBindingID != bindingID {
					continue
				}
				strategy := "none"
				if v2.IsSensitiveField {
					strategy = strings.ToLower(strings.TrimSpace(v2.MaskStrategy))
					if strategy == "" {
						strategy = "hmac"
					}
				}
```

---

## 4. Vấn đề 5: Scan fields chạy trên table rỗng và báo lỗi
- **Mục tiêu**: Tránh báo lỗi không cần thiết khi scan fields trên shadow table rỗng và giúp frontend dừng polling trạng thái thành công.
- **Giải pháp**: Sửa `ScanFieldsDebezium` trong `discover_handler.go` để trả về `0, 0, nil` thay vì ném lỗi khi table rỗng.
- **Chi tiết thay đổi**:
```go
// internal/handler/source/discover_handler.go
// Trong hàm ScanFieldsDebezium:
	if len(rows) == 0 {
		if sourceType == "mongodb" {
			return h.scanFieldsMongoSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		h.Logger.Info("scan-fields: shadow table is empty, returning success with 0 fields", 
			zap.String("table", targetTable))
		return 0, 0, nil
	}
```

---

## 5. Vấn đề 6: Transmute query parameters mismatch
- **Mục tiêu**: Khắc phục lỗi lệch parameters trong `ListMasterTablesByShadowIdentity` khiến transmuter không chạy.
- **Giải pháp**: Refactor câu query sang **GORM Named Arguments** (`@arg_name`) để loại bỏ hoàn toàn positional arguments và tối giản tham số truyền vào.
- **Chi tiết thay đổi**:
```go
// internal/repository/master/master_binding_repo.go
func (r *MasterBindingRepo) ListMasterTablesByShadowIdentity(ctx context.Context, shadowTable, shadowSchema, shadowConnectionKey, shadowBindingCode string) ([]string, error) {
	var result []string
	err := r.db.WithContext(ctx).Raw(
		`SELECT mb.master_table
		   FROM cdc_system.master_binding mb
		   JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
		   LEFT JOIN cdc_system.connection_registry cr ON cr.id = sb.shadow_connection_id
		  WHERE mb.is_active = true
		    AND mb.schema_status = 'approved'
		    AND (
		      (@binding_code <> '' AND sb.binding_code = @binding_code)
		      OR
		      (@binding_code = '' AND sb.shadow_table = @shadow_table 
		        AND COALESCE(sb.shadow_schema, '') = COALESCE(NULLIF(@shadow_schema, ''), COALESCE(sb.shadow_schema, '')) 
		        AND (@shadow_connection_key IN ('', 'default') OR COALESCE(cr.connection_code, 'default') = @shadow_connection_key))
		    )`,
		map[string]interface{}{
			"binding_code":          shadowBindingCode,
			"shadow_table":          shadowTable,
			"shadow_schema":         shadowSchema,
			"shadow_connection_key": shadowConnectionKey,
		},
	).Scan(&result).Error
	return result, err
}
```

---

## Verification Plan

### Automated Tests
1. Chạy bộ unit tests hiện có tại `cdc-cms-service` để đảm bảo logic `ListSnapshotProgress` hoạt động bình thường:
   ```bash
   go test -v ./...
   ```
2. Chạy bộ unit tests hiện có tại `centralized-data-service` để kiểm chứng:
   - Metadata reload (`metadata_registry_service_test.go`).
   - Discovery / Scan fields (`kafka_consumer_discover_test.go` hoặc tương đương).
   ```bash
   go test -v ./internal/service/source/...
   ```
3. Viết unit test mới để mô phỏng kịch bản:
   - Nạp mappingCache with rules thuộc các bindings khác nhau và kiểm tra sự cô lập.
   - Quét fields trên table rỗng trả về `0, 0, nil`.
