# Plan: Research & Fix Export Jobs Reconciliation Drift

## Phase 1: Research & Diagnosis
1. **Tra cứu Cấu hình Registry**: Kiểm tra cấu hình trong bảng `cdc_system.cdc_table_registry` đối với `export-jobs` (ví dụ: `timestamp_field`, `source_object_name`, `source_object_id`).
2. **Khảo sát Schema MongoDB & PostgreSQL**: Kiểm tra cấu trúc bản ghi thực tế trong MongoDB collection `export-jobs` và bảng `export_jobs` trong shadow PostgreSQL.
3. **Phân tích Code Reconciliation**: Kiểm tra logic đối soát tại `internal/service/recon/recon_tier_b.go` và `internal/service/recon/recon_dest_query.go` để xem cách lọc window thời gian và so sánh dữ liệu.
4. **Tìm bản ghi lệch**: Tìm ID của bản ghi bị lệch giữa MongoDB và shadow PostgreSQL để xác định lý do nó bị bỏ sót.

## Phase 2: Implementation & Fix
1. **Thiết kế giải pháp**: Đề xuất cách sửa (trong registry hoặc mã nguồn) và viết vào `09_tasks_solution_recon_export_jobs.md`.
2. **Review & Approve**: Chờ User duyệt giải pháp.
3. **Thực thi sửa đổi**: Thực hiện thay đổi cấu hình hoặc code.

## Phase 3: Verification
1. **Chạy đối soát (Recon)**: Trigger chạy đối soát qua API để xác minh lệch dữ liệu.
2. **Đồng bộ (Heal)**: Trigger heal qua API để sửa lệch dữ liệu.
3. **Kiểm tra kết quả**: Xác minh trạng thái đối soát sau heal báo `ok`.
