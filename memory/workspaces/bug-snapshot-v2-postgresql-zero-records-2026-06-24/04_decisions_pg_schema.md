# Decisions: Fallback Default Schema từ Connection Registry

## 1. Quyết định kiến trúc
Hệ thống sẽ không hardcode schema `"public"` cho PostgreSQL Snapshot V2 nữa, mà sẽ:
- Lấy `DefaultSchema` của Connection Registry làm schema mặc định khi `so.SourceSchema` rỗng.
- Chèn tham số `search_path` trực tiếp vào chuỗi DSN kết nối để driver Postgres (`pgx`) tự động trỏ đúng schema làm việc.

## 2. Rationale (Lý do)
- Tránh lỗi SASL Auth hoặc lỗi đối tượng/bảng không tồn tại khi kết nối tới PostgreSQL nguồn có các bảng snapshot nằm ngoài schema `public` (ví dụ như bảng của object 55 nằm ở schema `cdc_schema_testing`).
- Đảm bảo tính nhất quán giữa thông tin kết nối trong DB registry và quá trình scan/snapshot thực tế của worker.
- Tuân thủ nguyên tắc thiết kế tinh gọn và bám sát kiến trúc cốt lõi của hệ thống CDC.

## 3. Các phương án thay thế đã xem xét
- **Phương án A (Tự động chuyển đổi schema trong từng câu query)**: Rất phức tạp và dễ sót ở các query metadata như pg_class.
- **Phương án B (Dùng search_path trong DSN - Được chọn)**: Rất thanh lịch, driver tự động phân giải schema cho toàn bộ phiên làm việc, không cần thay đổi logic SQL query.
