# Yêu cầu Tính Năng: Xóa Offset Connector trên Kafka Connect (Reset Debezium Offset)

## 1. Bối cảnh & Mục tiêu
Khi Connector bị ngắt kết nối/dừng quá lâu khiến PostgreSQL purge WAL (gây lỗi `PostgresOffsetContext ... but this is no longer available on the server`), người vận hành cần khả năng Reset (xóa) Offset cũ của Debezium Connector trực tiếp trên UI để Debezium bắt đầu streaming lại hoặc snapshot lại từ đầu mà không cần thao tác curl CLI phức tạp.

## 2. Chi tiết Yêu cầu (Scope & Specs)
- **Backend (`cdc-cms-service`)**:
  - Hỗ trợ REST API `DELETE /connectors/{name}/offsets` thông qua KafkaConnectClient (Kafka Connect 3.5+ REST API).
  - Khai báo API Handler `ResetOffsets` (hoặc `DeleteOffsets`) trong `SystemConnectorsHandler`.
  - Đăng ký Route Destructive trong `router.go`: `registerDestructive("/v1/system/connectors/:name/offsets", h.Source.SystemConnectors.ResetOffsets)` hoặc tương đương.
  - Đảm bảo ghi audit log đầy đủ (với actor, timestamp, reason ≥ 10 ký tự).

- **Frontend UI (`cdc-cms-web`)**:
  - Trang `/sources` (`SourceConnectors.tsx`):
  - Thêm nút "Xóa Offset" (Reset Offset) ở cả bảng **Connections** và bảng **Connectors**.
  - Mở Modal xác nhận khi bấm nút "Xóa Offset" kèm theo Cảnh báo (Alert Warning):
    > *Lưu ý: Connector nên ở trạng thái PAUSED hoặc STOPPED trước khi xóa offset. Việc xóa offset sẽ làm Debezium quên vị trí LSN/offset cũ và bắt đầu stream lại từ LSN mới hoặc trigger snapshot dựa trên cấu hình snapshot.mode.*
  - Yêu cầu nhập lý do ≥ 10 ký tự trước khi xác nhận.
  - Gửi API request `POST` hoặc `DELETE` tới `/api/v1/system/connectors/:name/offsets` với header `Idempotency-Key`.
