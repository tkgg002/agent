# 08 Tasks: Danh sách công việc tối ưu GetTableHistory

- [x] Task 1: Sửa logic xây dựng `where` và `args` trong `GetTableHistory` (`recon_read_repo_gorm.go`).
- [x] Task 2: Tối ưu `countQuery` nhánh `excludeSmoke = true` loại bỏ subquery 40 cột.
- [x] Task 3: Tối ưu `countQuery` nhánh `excludeSmoke = false` thành UNION ALL `SELECT 1` siêu nhẹ.
- [x] Task 4: Kiểm tra và xác nhận nội dung file migration `102_fix_slow_table_history.sql`.
- [x] Task 5: Xác nhận nội dung SQL migration 102 và cung cấp lệnh chạy vào container Docker `gpay-postgres-cdc` (`cdc_dw`).
- [x] Task 6: Kiểm tra `git diff` và chạy `go build` xác nhận không có lỗi biên dịch.
- [x] Task 7: Cập nhật tài liệu báo cáo `11_report_fix_slow_table_history.md` và `14_walkthrough_fix_slow_table_history.md`.
- [x] Task 8 (Phase 2): Pushdown `ORDER BY checked_at DESC LIMIT fetchLimit` vào `reportBranch` và `smokeBranch` trong `selectQuery`.
- [x] Task 9 (Phase 2): Cập nhật migration 102 với composite covering index `idx_smoke_schema_table_segment_master` (`INCLUDE (segment, master_table)`).
