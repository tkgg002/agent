# Context: Debezium Delete Flow Debugging

## Vấn đề hiện tại
- **Mô tả**: Khi thực hiện hành động DELETE trên database nguồn, dữ liệu thay đổi không được cập nhật/xóa ở shadow table hoặc database đích. Luồng delete đang không chạy.
- **Mục tiêu**: Điều tra lý do tại sao delete event từ Debezium không được xử lý hoặc không được gửi đi, sau đó đề xuất và thực hiện giải pháp khắc phục.

## Các thành phần liên quan
1. **Debezium Connectors**:
   - PostgreSQL Source Connector (`deployments/debezium/pg-source-connector.json`)
   - MongoDB Source/Sink Connectors (đối với môi trường demo)
2. **Sink Worker (centralized-data-service)**:
   - Consumer lắng nghe Kafka topics và thực hiện ghi dữ liệu xuống shadow tables.
   - Logic xử lý delete event trong parser/handler (cần check xem có bị skip, hoặc do format message delete đặc thù của Debezium).
3. **Kafka Topics & Messages**:
   - Format của delete event trong Debezium (thường có payload `op: "d"` và tombstone message với value `null`).
