# Kế hoạch Kiểm thử & Xác minh: Concurrency & Batching Optimization

## 1. Kịch bản Unit Test
*   **TestBatchBufferParallelFlush:** Giả lập 10 bảng khác nhau, DB mock delay 50ms mỗi bảng. Tổng thời gian hoàn thành flush phải `< 100ms` (song song) thay vì `500ms` (tuần tự).
*   **TestDebounceBufferSizeLimit:** Gửi 500 tin nhắn của cùng một bảng, verify mẻ flush được kích hoạt ngay lập tức.
*   **TestDebounceBufferTimeout:** Gửi 10 tin nhắn, đợi 1.1s (timeout = 1s), verify mẻ flush được kích hoạt đúng giờ.
*   **TestPoisonPillIsolation:** Ghi một mẻ có 1 tin nhắn lỗi và 4 tin nhắn đúng. Verify 4 tin nhắn đúng được ACK, 1 tin nhắn lỗi bị Term và đẩy vào DLQ.

## 2. Kiểm thử Tải (Load/Stress Test)
*   Bắn tải 5000 msg/s qua Kafka.
*   Theo dõi `cdc_master_active_row_count` và Postgres connections.
