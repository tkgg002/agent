# Yêu cầu bổ sung: Fix sftp consumer sync to shadow

## Bối cảnh
Sau khi kích hoạt đệ quy (`policy.recursive: true`) và tạo connector thành công, Kafka topic `cdc.sftplocal.testsftp12.reconcile_final` đã nhận được 162 records (oplog phẳng từ file CSV). Tuy nhiên:
1. Các sự kiện này không tự động được lưu chuyển sang Shadow DB.
2. Dịch vụ log cảnh báo `kafka message has no 'after' data, dropping (non-delete)` và bỏ qua toàn bộ message.
3. Dịch vụ log cảnh báo `cannot resolve DSN for connection "testsftp12"`.

## Yêu cầu sửa đổi
1. Bổ sung cơ chế tự phát hiện và bypass lớp giải mã Debezium trong `kafka_consumer.go` đối với các topic SFTP (`IsSFTPTopic`).
2. Giải nén payload phẳng trực tiếp thành `afterData` và gọi thẳng sang `HandleRaw` trong `event_handler.go` để chuyển hóa dữ liệu.
3. Hỗ trợ động DSN resolution cho engine type `"sftp"` trong `metadata_registry_service.go` để loại bỏ log cảnh báo không mong muốn.
