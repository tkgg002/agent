# Tasks Checklist: Research, Diagnose & Fix Export Jobs Reconciliation Drift

- `[ ]` **Phase 1: Research & Diagnose**
  - `[ ]` Kiểm tra cấu hình `timestamp_field` của `export-jobs` trong Table Registry.
  - `[ ]` Kiểm tra dữ liệu thực tế tại MongoDB collection `export-jobs` và shadow PostgreSQL table `export_jobs`.
  - `[ ]` Đọc hiểu logic tại `recon_tier_b.go` và `recon_dest_query.go` để xem query hai bên được sinh thế nào.
  - `[ ]` Tìm bản ghi lệch cụ thể bằng cách chạy script phân tích.
- `[ ]` **Phase 2: Technical Design & Solution Approval**
  - `[ ]` Thiết kế giải pháp kỹ thuật cụ thể và ghi vào `09_tasks_solution_recon_export_jobs.md`.
  - `[ ]` Chờ User phê duyệt giải pháp trước khi thực hiện code/config thay đổi.
- `[ ]` **Phase 3: Implementation & Verification**
  - `[ ]` Thực hiện sửa code/config theo giải pháp đã duyệt.
  - `[ ]` Khởi động lại các service nếu cần thiết.
  - `[ ]` Chạy đối soát và heal lại bảng `export-jobs`.
  - `[ ]` Xác minh trạng thái đối soát sau heal báo `ok` và bản ghi đã khớp.
