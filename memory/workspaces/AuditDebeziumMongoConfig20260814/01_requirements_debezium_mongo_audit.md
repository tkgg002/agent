# Requirements: Audit Debezium MongoDB Source Connector Configuration

## Overview
Đánh giá, đối soát và chuẩn hóa file cấu hình JSON cho **Debezium MongoDB Source Connector** kết hợp với Schema Registry (Avro / JSON), khắc phục triệt để 7 Bẫy Nguy Hiểm (Tripwires) đã được nhận diện.

## Detail Requirements
1. Xác nhận và phân tích chi tiết 7 Tripwires trong cấu hình MongoDB CDC Connector:
   - Tripwire 1: Plain-text password rò rỉ trong Connection String -> dùng ConfigProvider (`${file:...}`) hoặc vault.
   - Tripwire 2: Signal Kafka Bootstrap Servers chỉ vào `localhost:29092` -> gây Connection Refused khi chạy trên Docker/K8s/Server.
   - Tripwire 3: Cấu hình `schema.history.internal.*` dư thừa (chỉ dành cho RDBMS DDL history, không áp dụng cho MongoDB NoSQL).
   - Tripwire 4: Sai chính tả name filtering (`centrallized-export-service` thừa chữ `l` -> Silent Failure).
   - Tripwire 5: Giới hạn `producer.override.max.request.size` 2MB dễ dính Poison Pill (Mongo document tối đa 16MB).
   - Tripwire 6: `snapshot.mode: no_data` bỏ qua dữ liệu cũ.
   - Tripwire 7: `signal.kafka.group.id` / `signal.kafka.consumer.group.id` dùng chung gây tranh giành/load-balance signal message.
2. Đánh giá cảnh báo về `AvroConverter` vs `JsonConverter` đối với MongoDB Schema-less:
   - MongoDB có thể có dynamic type (e.g. `status` lúc string, lúc int), Avro sẽ văng `Incompatible Schema`.
   - Đề xuất giải pháp an toàn (JsonConverter không schema hoặc dùng SMT / Mongo Schema Evolution control).
3. Trình bày file JSON cấu hình chuẩn hoá hoàn chỉnh (Production-Ready) mẫu cho Debezium MongoDB Source Connector.
