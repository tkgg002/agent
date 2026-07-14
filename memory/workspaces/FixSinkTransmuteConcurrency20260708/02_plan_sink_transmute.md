# Lộ trình Triển khai (Plan): Concurrency & Batching Optimization

Tài liệu này vạch ra lộ trình các bước thực thi chi tiết cho chiến dịch tối ưu hóa hiệu năng ghi của Sink và Transmute.

## 1. Danh sách Kỹ năng & Công cụ Sử dụng
*   `view_file`: Đọc hiểu cấu trúc các file Golang.
*   `replace_file_content` / `multi_replace_file_content`: Sửa đổi mã nguồn Golang của worker.
*   `run_command`: Chạy go test và build kiểm thử.

## 2. Các giai đoạn Triển khai (Phases)

### Giai đoạn 1: Song song hóa Sink Flush (`BatchBuffer`)
*   Sửa đổi hàm `Flush()` trong `internal/handler/shadow/batch_buffer.go` để chuyển đổi vòng lặp ghi từ tuần tự sang song song sử dụng `errgroup.Group`.
*   Giới hạn mức độ song song ở mức 20 thông qua `g.SetLimit(20)`.
*   Bảo vệ dữ liệu trả về (`written` và `err`) bằng một `sync.Mutex`.

### Giai đoạn 2: Tích hợp Concurrency Limiter tại Transmute Layer
*   Bổ sung tham số cấu hình giới hạn luồng chạy song song ghi Master DB.
*   Cập nhật `TransmuteHandler` trong `internal/handler/master/transmute_handler.go` để tích hợp một Semaphore Channel giới hạn số Goroutines chạy đồng thời khi thực thi `HandleTransmute`.

### Giai đoạn 3: Xây dựng Debounce Buffer (Micro-batching) & Late ACK
*   Tạo lớp đệm `DebounceBuffer` trong `TransmuteHandler` để gộp các trigger transmute.
*   Cấu hình NATS JetStream với cơ chế Pull Subscription và tích hợp Late ACK cho các worker xử lý transmute.

### Giai đoạn 4: Xác minh và Kiểm thử
*   Chạy unit test toàn bộ thư mục `internal/handler/shadow` và `internal/handler/master`.
*   Chạy integration test giả lập tải cao để đối soát hiệu năng kết nối DB.
