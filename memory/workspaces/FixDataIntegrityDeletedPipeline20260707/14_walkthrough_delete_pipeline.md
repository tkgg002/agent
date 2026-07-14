# Hướng dẫn Khảo sát & Xác minh (Walkthrough) - Hide Deleted/Inactive Pipelines

Dự án: `cdc-cms-service`
Workspace: `FixDataIntegrityDeletedPipeline20260707`

## Chi tiết các thay đổi trong Codebase

Chúng ta đã sửa đổi file:
- File: [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)

### 1. Thay đổi trong `listLatestPrimary` (Dòng 138):
```diff
-	  LEFT JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table
+	  INNER JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table AND reg.is_active = TRUE
```

### 2. Thay đổi trong `listLatestLegacy` (Dòng 173):
```diff
-	  LEFT JOIN cdc_table_registry reg ON reg.target_table = r.target_table
+	  INNER JOIN cdc_table_registry reg ON reg.target_table = r.target_table AND reg.is_active = TRUE
```

## Các bước kiểm tra và xác minh tiếp theo

Do quá trình chạy test tự động gặp timeout vì lý do phân quyền terminal trên hệ thống máy khách, các bước kiểm tra cần được chạy thủ công:

### Bước 1: Chạy toàn bộ Unit Test của module
Di chuyển vào thư mục `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service` và thực thi:
```bash
go test ./test/...
```
Xác nhận rằng toàn bộ các test cases đều trả về kết quả `PASS`.

### Bước 2: Kiểm tra trực tiếp trên Database (nếu cần)
Nếu muốn xác minh các câu truy vấn SQL chạy chính xác:
1. Chạy câu lệnh SQL `listLatestPrimary` gốc trên database môi trường dev/staging.
2. Kiểm tra xem các bảng có `is_active = FALSE` hoặc không có bản ghi đăng ký trong `cdc_table_registry` có còn xuất hiện trong kết quả hay không.
3. Chạy câu lệnh SQL `listLatestPrimary` mới đã được chỉnh sửa để xác minh rằng các pipeline đó đã bị loại bỏ.
