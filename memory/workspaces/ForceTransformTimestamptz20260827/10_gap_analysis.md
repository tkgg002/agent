# 10 Gap Analysis

## Hiện trạng & Đánh giá rủi ro
1. **Khả năng tương thích ngược (Backward Compatibility):**
   - API `POST /api/v1/source-objects/:id/transform` hoàn toàn tương thích với các client cũ không gửi body hoặc gửi body rỗng (`force=false`).
   - NATS command `cdc.cmd.batch-transform` vẫn xử lý trơn tru cả plain string table name (legacy) và JSON payload mới.
2. **Hiệu năng trên dữ liệu lớn (Performance & Scalability):**
   - Không thực hiện bất kỳ câu lệnh DDL hoặc full-table locking nào.
   - Force transform sử dụng lại toàn bộ cơ chế Chunked CTE UPDATE phân trang theo Primary Key (1000 rows/batch), tự điều chỉnh kích thước chunk động theo độ trễ thực thi.
3. **An toàn dữ liệu (Data Integrity):**
   - Yêu cầu bắt buộc `force_fields` không rỗng ngăn chặn việc vô tình ghi đè toàn bộ bảng.
   - Chỉ các trường được chỉ định rõ ràng trong danh sách mới được trích xuất lại từ `_raw_data`.
