# Phạm vi & Bối cảnh (Scope & Context)

- **Vấn đề**: `centralized-data-service` gặp lỗi "schema drift detected" liên tục mỗi 5s và văng cảnh báo "SLOW SQL" do insert vào bảng `pending_fields` và bảng đích (upsert 1 row/lần). Worker cũng đang re-process data cũ (snapshot) khiến hash collision diễn ra liên tục (`rows:0`).
- **Phạm vi xử lý**:
  1. `schema_inspector.go`: Sửa logic fallback schema `public` sang trả về lỗi để ngừng drift check nếu không resolve được shadow_schema.
  2. `event_handler.go`: Đổi `WriteRecordSync` thành `Add(record)` để kích hoạt tính năng Batching của DB, giảm thiểu SLOW SQL (theo quyết định của User chấp nhận thay đổi metric đếm).
  3. Xử lý hạ tầng (Kafka Offset): Reset offset để tránh re-process backlog cũ (sẽ thực hiện qua CLI/NATS tùy chọn).
