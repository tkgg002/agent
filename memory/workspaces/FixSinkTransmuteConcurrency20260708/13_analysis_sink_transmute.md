# Phân tích Kiến trúc (Analysis): Concurrency & Batching Optimization

Tài liệu này phân tích chi tiết nguyên nhân gốc rễ và đề xuất giải pháp kỹ thuật giải quyết bài toán tải cao (5000 msg/s).

## 1. Phân tích Nguyên nhân Gốc rễ (Root Cause Analysis)
*   **Vấn đề 1 (Sequential Flush):** Khi có tải cao trên nhiều bảng (tối đa 200 bảng), `BatchBuffer` gom dữ liệu thành công nhưng khi Flush lại duyệt tuần tự qua từng bảng để thực hiện `batchUpsert`. Việc này dẫn đến nếu một bảng bị phản hồi chậm từ database, toàn bộ hàng đợi flush của các bảng khác sẽ bị nghẽn (Head-of-Line Blocking), làm tăng đột biến Consumer Lag trên Kafka.
*   **Vấn đề 2 (Connection Storm & Lock Contention):** Khi nhận tin nhắn trigger transmutation từ NATS, `TransmuteHandler` sử dụng cơ chế `go func()` không giới hạn cho mỗi tin nhắn. Khi tải burst đạt 5000 msg/s, hàng ngàn Goroutines sẽ được sinh ra đồng thời, tranh chấp kết nối xuống Postgres Master, gây cạn kiệt Connection Pool và tăng thời gian lock hàng đợi của DB.

## 2. Đánh giá Giải pháp Đề xuất
*   **Giải pháp 1 (Parallel Flush sử dụng `errgroup`):**
    *   *Ưu điểm:* Song song hóa việc ghi xuống DB, giảm thời gian khóa luồng Kafka Consumer từ $O(N)$ xuống $O(N/20)$.
    *   *Rủi ro:* Cần cấu hình Connection Pool của Postgres (`MaxOpenConns`) lớn hơn giới hạn SetLimit (20) để tránh nghẽn kết nối nội bộ.
*   **Giải pháp 2 (Semaphore Concurrency Limiter):**
    *   *Ưu điểm:* Giới hạn số lượng Goroutines hoạt động đồng thời ghi Master DB, bảo vệ CPU và RAM của Master DB.
    *   *Rủi ro:* Các Goroutines bị chặn chờ slot sẽ tiêu tốn một phần nhỏ RAM, cần quản lý vòng đời context để tránh rò rỉ (goroutine leak).
