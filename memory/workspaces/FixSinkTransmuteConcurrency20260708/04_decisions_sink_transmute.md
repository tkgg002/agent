# Nhật ký Quyết định Kiến trúc (ADRs): Concurrency & Batching Optimization

## ADR-1: Nâng cấp NATS từ Push sang Pull JetStream
*   **Bối cảnh:** Push subscription không có backpressure làm tràn bộ đệm worker dưới tải cao.
*   **Quyết định:** Chuyển đổi sang Pull JetStream. Worker chủ động kéo (Fetch) tin nhắn khi rảnh.
*   **Trạng thái:** APPROVED.

## ADR-2: Song song hóa Flush
*   **Bối cảnh:** Flush tuần tự 200 bảng gây nghẽn cục bộ.
*   **Quyết định:** Sử dụng `errgroup` với giới hạn `SetLimit(20)`.
*   **Trạng thái:** APPROVED.

## ADR-3: Poison Pill Fallback (Sequential Write + Term)
*   **Bối cảnh:** Lỗi của một bản ghi làm hỏng cả mẻ Bulk Transmute gây retry vô hạn.
*   **Quyết định:** Fallback ghi tuần tự, lưu dòng lỗi vào DLQ và gọi `Term()` trên NATS.
*   **Trạng thái:** APPROVED.
