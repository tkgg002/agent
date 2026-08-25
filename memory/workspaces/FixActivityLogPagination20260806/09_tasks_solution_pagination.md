# Technical Solution: Sửa Lỗi Phân Trang API Activity Log

## File Cần Chỉnh Sửa
`cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`

## Chi Tiết Thay Thế
Tại hàm `enrichmentFromClause` (dòng 70-72), thay thế phép join cũ của `master_binding`:

### Nội dung cũ:
```go
		LEFT JOIN cdc_system.master_binding mb
		  ON mb.shadow_binding_id = sb.shadow_binding_id
		 AND mb.is_active = TRUE
```

### Nội dung mới:
```go
		LEFT JOIN LATERAL (
			SELECT mb.master_schema, mb.master_table
			FROM cdc_system.master_binding mb
			WHERE mb.shadow_binding_id = sb.shadow_binding_id
			  AND mb.is_active = TRUE
			ORDER BY mb.updated_at DESC, mb.id DESC
			LIMIT 1
		) mb ON TRUE
```
