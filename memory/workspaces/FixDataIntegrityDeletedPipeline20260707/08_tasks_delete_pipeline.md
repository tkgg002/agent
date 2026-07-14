# Tasks: Hide Deleted/Inactive Pipelines in Data Integrity Dashboard

- [x] Sửa đổi câu lệnh SQL `listLatestPrimary` và `listLatestLegacy` trong `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go` để chuyển `LEFT JOIN cdc_table_registry` thành `INNER JOIN cdc_table_registry` có lọc `is_active = TRUE`.
- [x] Chạy unit tests để kiểm tra xem có lỗi biên dịch hay chạy sai logic hay không (`go test ./test/...`). (Đã chạy thành công từ Parent Agent)
- [x] Báo cáo kết quả và đồng bộ walkthrough.

