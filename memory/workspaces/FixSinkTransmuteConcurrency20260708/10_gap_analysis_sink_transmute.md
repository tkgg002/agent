# Phân tích Lỗ hổng Kiến trúc (Gap Analysis): Concurrency & Batching Optimization

Tài liệu này ghi nhận sự khác biệt giữa thiết kế cũ và thiết kế mới được đề xuất.

## 1. Hiện trạng (Current State)
*   **Flush chặng Sink:** Ghi tuần tự từng bảng. Nếu bảng A trễ, bảng B bị chặn.
*   **Transmute Handler:** Sinh goroutine tự do cho mỗi tin nhắn NATS nhận được. Gây Connection Storm xuống Master DB khi có burst.
*   **NATS Subscription:** Cơ chế Push truyền thống, không có backpressure và Late ACK đúng nghĩa.

## 2. Trạng thái Đích (Target State)
*   **Flush chặng Sink:** Ghi song song tối đa 20 bảng. Cô lập hoàn toàn lỗi giữa các bảng.
*   **Transmute Handler:** Sử dụng Debounce Buffer gom tin nhắn trong RAM (1s / 500 records), giới hạn concurrency ghi Master DB bằng semaphore.
*   **NATS Subscription:** Cơ chế Pull JetStream hỗ trợ Late ACK và Poison Pill Isolation (Sequential fallback + Term + DLQ).
