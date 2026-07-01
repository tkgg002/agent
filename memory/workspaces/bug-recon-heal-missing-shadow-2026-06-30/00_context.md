# Context: Lỗi Record 41063 Bị Thiếu ở Shadow DB Dù Đã Gửi Debezium Signal

## Hiện tượng
- Quá trình tự chữa lành (Self-Healing) của Recon V4 P2 đã kích hoạt thành công cho bảng `payment_bills`.
- Hệ thống phát hiện ID bị thiếu `41063` (mã hex/string trong mongo hoặc shadow) và đã bắn tín hiệu snapshot:
  - `msg: "recon heal-a dispatched snapshot signal", table: "payment_bills", ids: 1`
  - `msg: "debezium signal published", topic: "cdc.signal.commands", filter: "{\"_id\": {\"$in\": [\"41063\"]}}"`
  - Debezium đã xác nhận trạng thái connector `RUNNING` và sẵn sàng nhận lệnh: `msg: "debezium signal end-to-end ready", connector_state: "RUNNING"`.
- Tuy nhiên, khi kiểm tra bảng `payment_bills` ở Shadow DB, record ID `41063` vẫn không xuất hiện.

## Vấn đề cần giải quyết
- Tìm ra điểm nghẽn hoặc lỗi trong luồng đi của dữ liệu từ:
  **Source (MongoDB) -> Debezium (Kafka Connect) -> Kafka Topic (cdc.signal.commands / data topic) -> Sinkworker -> Shadow DB (PostgreSQL)**
- Khắc phục lỗi để record được đồng bộ thành công sang Shadow DB.
