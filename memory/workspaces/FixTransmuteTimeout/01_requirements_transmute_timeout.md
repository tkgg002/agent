# Yêu cầu: Tối ưu Hóa Batch Processing và Khắc phục Timeout Transmuter

## 1. Vấn đề Hiện tại
- Lỗi `context deadline exceeded` xảy ra khi thực hiện đồng bộ hóa dữ liệu từ bảng Shadow sang bảng Master (Transmute) cho các bảng lớn (khoảng 100M+ dòng).
- Nguyên nhân: `TransmuteHandler.HandleTransmute` áp dụng một timeout cứng là 5 phút (`context.WithTimeout(ctx, 5*time.Minute)`). Mọi hoạt động đồng bộ hóa toàn bộ bảng (full sync) đều bị hủy bỏ và log lỗi sau 300 giây.
- Luồng NATS nhận message và xử lý đồng bộ trong cùng một thread đăng ký, điều này có thể chặn các lệnh đồng bộ khác và làm tăng nguy cơ quá tải nếu tiến trình kéo dài.

## 2. Mục tiêu kỹ thuật
- Tối ưu hóa truy vấn incremental/heal (`len(onlyIDs) > 0`): bỏ cast và bỏ `ORDER BY PK` không cần thiết, giúp tránh tình trạng quét tuần tự toàn bộ index hoặc quét toàn bộ bảng.
- Tự động tạo index non-partial (`CREATE INDEX CONCURRENTLY`) trên bảng Shadow cho cột `_source_id` nếu chưa tồn tại để tăng tốc độ truy vấn incremental/heal.
- Triển khai cơ chế checkpoint/resume cho tiến trình Full Sync dựa trên cột `last_cursor_json` trong bảng `cdc_system.sync_runtime_state` để tránh phải chạy lại từ đầu khi bị gián đoạn.
- Đẩy việc thực thi transmuter sang goroutine riêng biệt (asynchronous) để tránh chặn luồng NATS subscription.
- Thiết lập timeout linh hoạt cho các tiến trình: 30 phút cho incremental/heal, 24 giờ cho full sync.
- Duy trì tính nhất quán của hệ thống log hoạt động (`ActivityLog`), metrics và tracing (`observability`).
- Đảm bảo các test suite hiện tại vẫn hoạt động bình thường.
