# Plan: Core Fix for Batch Transform Schema Drift

## Phase 1: Research & Code Modification
1. Rà soát file `internal/handler/shadow/batch_transform_handler.go` để xác định chính xác vị trí lặp qua mapping rules.
2. Thêm logic kiểm tra cột tồn tại bằng cách gọi `h.HasColumnInSchema`.
3. Sửa hàm `IsSafeType` trong `internal/handler/base/base_handler.go` dùng biểu thức chính quy (Regex) đồng bộ với `TypeResolver` của `shadow`.
4. Build lại code của `centralized-data-service`.

## Phase 2: Verification (Red -> Green)
1. Khôi phục cấu hình database registry gốc của User (đã khôi phục).
2. Chạy lệnh batch-transform cho bảng V1 (`export_jobs`) thông qua script test NATS command.
3. Verify xem worker có in ra log warning bỏ qua cột `__v` và thực thi transform thành công các cột còn lại không.
4. Gửi lệnh `create-default-columns` cho bảng `export_jobs_4` có PK type là `VARCHAR(24)` để verify quá trình chạy không còn lỗi `invalid primary key type "VARCHAR(24)"`.
5. Đảm bảo kết quả trả về NATS của cả hai lệnh là `success`.


## Phase 3: Governance Audit & Cleanup
1. Chạy `/security-agent` để rà soát thay đổi code (nếu có, nhưng hiện tại Brain không tự sửa code).
2. Tạo file `report_core_fix.md` ghi nhận toàn bộ file thay đổi, dòng code thay đổi theo quy tắc số 10.
3. Cập nhật `05_progress.md` và `active_plans.md`.
