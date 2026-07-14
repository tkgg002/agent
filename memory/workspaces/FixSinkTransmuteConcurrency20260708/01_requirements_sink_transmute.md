# Yêu cầu Chi tiết: Tối ưu hóa Concurrency & Batching cho Sink và Transmute

Tài liệu này xác định các yêu cầu kỹ thuật cần đạt được (Definition of Done) cho chiến dịch tối ưu hóa hiệu năng ghi của lớp Shadow DB (Sink Worker) và Master DB (Transmute Worker).

## 1. Yêu cầu Hiệu năng & Khả năng chịu tải (SLA)
*   **Throughput:** Hệ thống phải chịu tải tối thiểu 5000 messages/giây từ Kafka mà không xảy ra hiện tượng tràn bộ đệm hoặc phình RAM do bùng nổ Goroutines.
*   **Độ trễ ghi (Flush Latency):** Thời gian flush một lô ghi Shadow DB (chứa 200 bảng khác nhau) giảm từ mức $O(N)$ (tuần tự) xuống còn $O(N/P)$ (song song), với $P \le 20$ là giới hạn luồng ghi song song.
*   **Bảo vệ Database:** Số lượng kết nối đồng thời từ Worker ghi xuống Postgres (cả Shadow và Master) không vượt quá giới hạn tài nguyên cấu hình, ngăn ngừa lỗi "too many connections" hoặc "connection timeout".

## 2. Yêu cầu Kỹ thuật
*   **Song song hóa Sink Flush:**
    *   Sử dụng `errgroup` với cấu hình semaphore giới hạn (max 20) để flush đồng thời các nhóm bảng trong `BatchBuffer`.
    *   Tự cô lập lỗi: Nếu bảng A lỗi kết nối hoặc lỗi cú pháp, quá trình ghi của bảng B vẫn phải tiếp tục và hoàn tất.
*   **Giới hạn Concurrency Transmute:**
    *   Tích hợp bộ điều phối (Semaphore/Worker Pool) tại `TransmuteHandler` để giới hạn số lượng luồng ghi Master DB đồng thời.
*   **Debounce Buffer & Late ACK (NATS):**
    *   Nâng cấp cơ chế trigger Transmute: Gom các tin nhắn trigger `cdc.cmd.transmute` trong bộ nhớ RAM trong thời gian tối đa 1s hoặc đạt size 500 records/bảng.
    *   Chỉ ACK tin nhắn NATS sau khi mẻ Transmute ghi Master DB thành công. Nếu lỗi hoặc sập, tin nhắn tự động redeliver về NATS.

## 3. Definition of Done (DoD)
*   [ ] Thực hiện song song hóa vòng lặp flush tại `batch_buffer.go` thành công.
*   [ ] Tích hợp semaphore giới hạn Goroutine tại `transmute_handler.go` thành công.
*   [ ] Cơ chế Late ACK hoạt động chính xác khi có lỗi giả lập (đứt mạng Postgres Master).
*   [ ] Kiểm thử tích hợp (Unit test/Integration test) chạy qua thành công.
