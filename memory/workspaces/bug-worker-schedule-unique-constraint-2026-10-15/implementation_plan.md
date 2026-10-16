# Implementation Plan — Sửa lỗi Unique Constraint `(operation, target_table)` trên `cdc_worker_schedule`

## 1. Mục tiêu
Giải quyết triệt để lỗi `"Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)"` khi tạo lịch trình (`source_count` hoặc các operation đa tầng) cho cùng tên bảng `payment_bills` trên các tầng (`layer_type`: `source` / `shadow` / `master`), kết nối (`connection_code`) hoặc schema (`container_namespace`) khác nhau.

## 2. Các bước thực hiện
1. Xác minh định nghĩa bảng `cdc_worker_schedule`, các Unique Index/Constraint trong migrations & database thực tế, và handler xử lý lỗi HTTP 409 trong `cdc-cms-service`.
2. Ghi nhận bài học vào `agent/memory/global/lessons.md` theo đúng Rule #5 & Rule #6.
3. Lập hồ sơ giải pháp chi tiết `09_tasks_solution_unique_constraint.md` và trình bày rõ nguyên nhân gốc rễ + phương án sửa tối ưu cho User.
