# Kế Hoạch Triển Khai: Sửa Lỗi Phân Trang API Activity Log

## 1. Phương án
Thay thế phép join thông thường bảng `master_binding` bằng `LEFT JOIN LATERAL ... LIMIT 1` để tránh nhân bản kết quả khi quan hệ `shadow_binding` ↔ `master_binding` là 1-N.

## 2. File thay đổi
- File: `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`
- Vị trí: Hàm `enrichmentFromClause` (dòng 70-72)

```sql
		LEFT JOIN LATERAL (
			SELECT mb.master_schema, mb.master_table
			FROM cdc_system.master_binding mb
			WHERE mb.shadow_binding_id = sb.shadow_binding_id
			  AND mb.is_active = TRUE
			ORDER BY mb.updated_at DESC, mb.id DESC
			LIMIT 1
		) mb ON TRUE
```

## 3. Xác thực
- Chạy unit tests: `go test ./test/...`
- Chạy script kiểm tra `test_db.go` để đối soát số lượng dòng logs của Page 1 và Page 2.
