# Tasks Checklist: Đối soát và Heal bảng export-jobs bị lệch và noop

- `[ ]` **Phase 1: Research & Diagnose**
  - `[ ]` Kiểm tra cấu hình `timestamp_field` của `export-jobs` trong Table Registry.
  - `[ ]` Xác minh tên database MongoDB chính xác và đếm số lượng record.
  - `[ ]` Tìm bản ghi lệch cụ thể giữa MongoDB và shadow PostgreSQL.
  - `[ ]` Phân tích code reconciliation để hiểu tại sao trả về `noop`.
- `[ ]` **Phase 2: Technical Design & Solution Approval**
  - `[ ]` Thiết kế giải pháp kỹ thuật cụ thể và ghi vào `09_tasks_solution_recon_export_jobs.md`.
  - `[ ]` Chờ User phê duyệt giải pháp trước khi thực hiện code/config thay đổi.
- `[ ]` **Phase 3: Implementation & Verification**
  - `[ ]` Thực hiện sửa code/config theo giải pháp đã duyệt.
  - `[ ]` Chạy đối soát và heal lại bảng `export-jobs`.
  - `[ ]` Xác minh trạng thái đối soát sau heal báo `ok` và bản ghi đã khớp.
