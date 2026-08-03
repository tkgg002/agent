# Phân Tích Kỹ Thuật - Kafka Connect Reset Offset REST API

## 1. Cơ chế Reset Offset của Kafka Connect (Kafka 3.5+)
Kafka Connect (từ phiên bản 3.5.0 trở đi) hỗ trợ chính thức endpoint REST API:
- `DELETE /connectors/{name}/offsets`: Xóa toàn bộ offset đã commit của connector khỏi topic `connect-offsets`.
- **Ràng buộc của Kafka Connect REST API**: Connector phải đang ở trạng thái **STOPPED** hoặc **PAUSED**. Nếu gọi khi connector đang `RUNNING`, Kafka Connect sẽ ném ra lỗi `HTTP 409 Conflict` (Connector offset reset requests can only be made when the connector is stopped).

## 2. Thiết Kế An Toàn (Safety Gate)
- Trên UI: Thêm cảnh báo lưu ý người dùng nên `Pause` connector trước khi thực hiện nút `Xóa Offset`.
- Trên Backend API: Bọc middleware `destructiveChain` (yêu cầu quyền OpsAdmin, Idempotency-Key và Audit Logging).
