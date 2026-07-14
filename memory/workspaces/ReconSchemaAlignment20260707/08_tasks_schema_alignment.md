# Danh sách Task - Đồng bộ Schema Đối soát Shadow/Master

- [x] Áp dụng migration `089_recon_master_metadata.sql` để tạo cột trong DB
- [x] Cập nhật struct `ReconciliationReport` ở `cdc-cms-service` và `centralized-data-service`
- [x] Cập nhật logic `stampB` trong `centralized-data-service` để lưu master metadata
- [x] Cập nhật query `GetTableHistory` trong `recon_read_repo_gorm.go` để lấy `master_schema`
- [x] Chạy build dự án đảm bảo compile thành công
- [x] Xác minh API trả về trường `master_schema` qua cURL/jq
