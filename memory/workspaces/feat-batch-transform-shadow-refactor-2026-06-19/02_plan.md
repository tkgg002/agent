# Kế hoạch Workspace: Refactor BatchTransformHandler cho Hiệu năng và Bảo mật

## Các bước thực hiện cụ thể

1. **Lập Kế hoạch & Phê duyệt (PLANNING)**:
   - Nghiên cứu mã nguồn của `batch_transform_handler.go`.
   - Viết Implementation Plan chi tiết gửi người dùng xem xét và phê duyệt.
   - Nhận phê duyệt từ người dùng trước khi tiến hành viết code.

2. **Thực thi (IMPLEMENTATION)**:
   - Thay đổi các lệnh SQL thô trong `HandleMasterSwap` để sử dụng `sqlutil.QuoteIdent` phòng chống SQL Injection.
   - Tái cấu trúc truy vấn chunked update trong `HandleBatchTransform` sử dụng dual-CTE để chỉ quét PK bằng Index-Only Scan.
   - Sử dụng `.Row().Scan` để tối ưu hóa việc đọc dữ liệu kết quả của mỗi chunk.
   - Cập nhật điều kiện dừng và cơ chế tích lũy `totalRows` theo đúng semantics.

3. **Kiểm thử & Xác minh (VERIFICATION)**:
   - Tạo file unit test mới `batch_transform_handler_test.go` dùng `go-sqlmock`.
   - Test toàn diện các kịch bản lỗi, chạy thành công cho cả `HandleMasterSwap` và `HandleBatchTransform`.
   - Chạy test suite dự án để đảm bảo tính ổn định.
