# 01_requirements_phase_2.md - Yêu cầu chi tiết Phase 2

Tài liệu này đặc tả các yêu cầu nghiệp vụ và kỹ thuật cho Phase 2 của dự án centralized-data-service, tập trung vào nâng cao hiệu năng xử lý song song, khả năng chịu lỗi, dọn dẹp dữ liệu thừa, và đối soát tự động.

---

## 1. P2-1: Tối ưu hóa Concurrency & Throttling (Sink & Transmute)
- **Yêu cầu chặng Sink:**
  - Ghi song song dữ liệu xuống Postgres cho nhiều bảng khác nhau để tránh nghẽn cổ chai tuần tự.
  - Giới hạn tối đa số lượng bảng chạy song song đồng thời (concurrency limit) là 20 để bảo vệ connection pool DB.
  - Đảm bảo lỗi ghi của 1 bảng không được kéo sập hoặc hủy bỏ các giao dịch ghi thành công của các bảng khác trong cùng một batch.
- **Yêu cầu chặng Transmute:**
  - Tích hợp cơ chế Debounce linh hoạt theo Idle Timeout (reset khi có tin nhắn mới) và Max Timeout (bắt buộc flush sau khoảng thời gian tối đa) để gom lô hiệu quả cho các bảng có tần suất ghi thưa.
  - Khống chế Concurrency xử lý transmute theo từng bảng (ví dụ tối đa 10 luồng xử lý song song trên 1 bảng) để tránh tranh chấp lock/deadlock trên Postgres.
  - Tích hợp Backpressure hãm tốc độ pull tin nhắn từ NATS JetStream nếu hàng đợi RAM vượt quá giới hạn an toàn (`2 * maxSize`).
  - Xử lý Poison Pill đệ quy chia để trị (Binary Search Split) với độ phức tạp $O(\log N)$ để tự động cô lập bản ghi lỗi, gửi cảnh báo DLQ và terminate tin nhắn rác, tránh làm nghẽn toàn bộ luồng xử lý.
  - Kéo dài thời gian AckWait bằng cách định kỳ gọi `msg.InProgress()` trong lúc chia nhỏ và xử lý các mẻ con lỗi.
  - Phân loại lỗi mạng/DB (transient errors) để gọi `Nak()` trả lại tin nhắn về NATS JetStream ngay lập tức mà không chạy tuần tự hay cô lập Poison Pill.

---

## 2. P2-2: Dọn dẹp bản ghi mồ côi (Flatten Orphan Cleanup)
- **Bối cảnh:** Đối với các tài liệu CDC có mảng chứa phần tử (ví dụ danh sách line items), hệ thống transmute hoạt động bằng cách explode mảng đó thành nhiều dòng trong master table với PK dạng `<parentID>::idx::<index>`.
- **Yêu cầu:** 
  - Khi một tài liệu cập nhật và thu nhỏ mảng (ví dụ từ 5 phần tử xuống 3 phần tử), hệ thống transmute phải tự động phát hiện và soft-delete (`_deleted = true`, cập nhật `_source_ts`) các dòng master mồ côi dư thừa (trong ví dụ là chỉ số 3 và 4) để tránh trả thừa dữ liệu cho các câu truy vấn aggregate.

---

## 3. P2-3: Đối soát tự động (Kafka Offset vs Shadow DB Count Reconciliation) - HOÃN LẠI (POSTPONED)
- **Yêu cầu:** (Đã hoãn theo yêu cầu của User. Sẽ triển khai ở phase sau).
  - Xây dựng background job định kỳ đối soát lượng dữ liệu đọc từ Kafka và ghi vào shadow table.
  - So sánh `HighWatermark` của các Kafka partition với `MAX(_source_offset)` thực tế đã lưu trong shadow table của partition đó.
  - Nếu độ lệch (lag) vượt quá ngưỡng cấu hình trong thời gian dài (ví dụ 100 offsets trong 10 phút), hệ thống phải nâng cảnh báo qua Prometheus metrics (`cdc_recon_kafka_shadow_lag`) và in log cảnh báo mức độ cao (Critical/High).

---

## 4. P2-4: Giải phóng Scheduler bị kẹt (Scheduler Stuck Cleanup)
- **Yêu cầu:**
  - Khi worker bị crash đột ngột (mất điện, kill process) giữa chừng khi đang chạy transmute job, trạng thái job trong `transmute_schedule` sẽ bị kẹt vĩnh viễn ở trạng thái `'running'`.
  - Hệ thống scheduler phải định kỳ (ở đầu mỗi chu kỳ tick) quét và reset các job kẹt `'running'` quá 2 lần chu kỳ interval (hoặc quá 10 phút) về trạng thái `'failed'` hoặc `'idle'` kèm theo error log để cron có thể tự động trigger chạy lại ở chu kỳ tiếp theo.
