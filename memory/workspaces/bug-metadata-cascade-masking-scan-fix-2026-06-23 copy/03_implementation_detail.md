# Technical Design

## 1. Loại bỏ cascade active
File: `cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go`
- Xóa bỏ block logic kiểm tra và cập nhật `cdc_system.shadow_binding` khi `is_active` được cập nhật cho `source_object_registry`.

## 2. Sửa lateral join hiển thị sai tên bảng shadow
File: `cdc-cms-service/internal/infra/persistence/scheduler/snapshot_progress_read_repo_gorm.go`
- Thay thế đoạn LEFT JOIN LATERAL cũ bằng LEFT JOIN trực tiếp:
```sql
		LEFT JOIN cdc_system.shadow_binding sb ON sb.id = COALESCE(
			sp.shadow_binding_id,
			(
				SELECT s.id FROM cdc_system.shadow_binding s
				WHERE s.source_object_id = sp.source_object_id
				ORDER BY s.updated_at DESC, s.id DESC
				LIMIT 1
			)
		)
```

## 3. Sửa nạp chéo rules trong cache
File: `centralized-data-service/internal/service/source/metadata_registry_service.go`
- Trong vòng lặp `for _, v2 := range v2Rules`, thêm điều kiện so sánh `v2.ShadowBindingID` với `bindingID`:
```go
			bindingID := route.ShadowBinding.ID
			for _, v2 := range v2Rules {
				if v2.ShadowBindingID != nil && *v2.ShadowBindingID != bindingID {
					continue
				}
				rs.mappingCache[bindingID] = append(rs.mappingCache[bindingID], convertV2ToLegacyRule(v2, src.SourceObjectName))
			}
```
