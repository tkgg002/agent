# Plan: Điều tra lỗi master_connection_not_found

## Các bước thực hiện:
1. **Research & Triage (Muscle)**: Tìm kiếm chuỗi `"master_connection_not_found"` trong toàn bộ codebase để xác định chính xác file và hàm sinh ra lỗi này.
2. **Phân tích Root Cause (Muscle/Brain)**: Xem xét điều kiện kích hoạt lỗi. Kiểm tra cấu hình database master, connection registry, các bảng liên quan như `cdc_connection_registry` hoặc cấu hình biến môi trường.
3. **Đề xuất giải pháp (Brain)**: Lập phương án sửa lỗi chi tiết (code config, migration, code backend...) và trình user.
4. **Thực thi sửa lỗi (Muscle)**: Sửa code hoặc cấu hình theo phương án đã duyệt.
5. **Verify (Muscle)**: Chạy test, kiểm tra logs, xác nhận lỗi đã biến mất.
