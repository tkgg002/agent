# 01 Requirements — Bug: Vi phạm ràng buộc duy nhất (Unique Constraint) khi tạo Worker Schedule cho Shadow và Master có cùng tên bảng (`payment_bills`)

## 1. Bối cảnh & Triệu chứng (User Report)
- **Thông báo lỗi trên UI:** `Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)`
- **Phản ánh của User:** "sao 1 cái shadow, 1 cái master mà chọn tên table giống nhau payment_bills lại báo như này, làm việc hời hợt vậy"
- **Kịch bản lỗi:** Operator tạo 2 tác vụ (ví dụ `source_count`) trên `/activity-manager`:
  - Tác vụ 1: `layer_type = 'shadow'`, `target_table = 'payment_bills'`
  - Tác vụ 2: `layer_type = 'master'`, `target_table = 'payment_bills'`
  - Khi lưu tác vụ thứ 2, hệ thống từ chối với lỗi `Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)` (HTTP 409 Conflict / SQLSTATE 23505).

## 2. Yêu cầu Nghiệp vụ & Kỹ thuật (Specs)
- **REQ-01 (Định danh duy nhất đúng kiến trúc đa tầng 1-N-M):** Trong kiến trúc Data Hub đa tầng (`source`, `shadow`, `master`) và đa kết nối (`connection_code`, `container_namespace`), cùng một `operation` (ví dụ `source_count`) và cùng tên bảng `target_table` (ví dụ `payment_bills`) hoàn toàn có thể tồn tại hợp lệ ở:
  - Các tầng khác nhau (`layer_type`: `source` vs `shadow` vs `master`).
  - Các kết nối khác nhau (`connection_code`: `default_master` vs `master_2`...).
  - Các namespace/schema khác nhau (`container_namespace`).
- **REQ-02 (Loại bỏ ràng buộc phẳng `UNIQUE (operation, target_table)`):** Ràng buộc duy nhất của `cdc_worker_schedule` (hoặc kiểm tra trùng lặp ở API/DB) không được phép chỉ khóa cứng trên 2 trường phẳng `(operation, target_table)`.
- **REQ-03 (Phân tích gốc rễ & Đề xuất giải pháp Propose-Only theo Rule #1 & Rule #13):**
  - Ghi nhận bài học (Mid-Session Fix - Rule #5) vào `agent/memory/global/lessons.md`.
  - Kiểm tra chính xác ràng buộc DB (`cdc_worker_schedule` indexes/constraints) và code backend (`cdc-cms-service` / `centralized-data-service`) + frontend (`cdc-cms-web`).
  - Lập hồ sơ giải pháp chi tiết tại `09_tasks_solution_unique_constraint.md` trình bày rõ Root Cause và giải pháp chuẩn xác chờ User `APPROVE` (Rule #1, Rule #13).
