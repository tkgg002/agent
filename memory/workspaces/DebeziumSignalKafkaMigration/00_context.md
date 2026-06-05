# 00_context — Debezium Signal Kafka Migration

## Bối cảnh
Trước migration, `centralized-data-service` kích hoạt incremental snapshot của Debezium bằng cách **InsertOne** một document `{ type: "execute-snapshot", data: { data-collections: [...] } }` vào collection `debezium_signal` **của database nguồn (source MongoDB)**.

Vấn đề:
1. Vi phạm nguyên tắc **source DB là read-only**: worker phải có quyền write trên DB nguồn.
2. Lỗi production thực tế: `not authorized on bank-service to execute command { insert: "debezium_signal", ... }` khi user nhấn nút "Snapshot Now" trên FE.
3. Đối với mỗi source mới phải cấp quyền write — risk lan tỏa, audit phức tạp.

## Đường đi cũ (đã loại bỏ)
- FE `/api/recon/debezium-signal` → NATS `cdc.cmd.debezium-signal` →
  `internal/handler/recon_handler.go::HandleDebeziumSignal` →
  3 dispatch path:
  1. `signalClient` (Kafka, đúng): chỉ áp dụng khi `cfg.Kafka.Brokers` không rỗng.
  2. `mongo_shared_client` (sai): `client.Database(db).Collection("debezium_signal").InsertOne(...)`.
  3. `mongo_lazy_resolve` (sai): tự resolve URI source từ `connection_registry` rồi InsertOne.

Mọi connector Debezium (file mẫu `deployments/debezium/mongodb-connector.json` + CMS `SourceConnectors.tsx::buildConnectorConfig`) đều đang dùng `signal.data.collection = "<source-db>.debezium_signal"` — yêu cầu Debezium scan source DB để đọc signal.

## Mục tiêu
Loại bỏ hoàn toàn việc ghi/đọc signal trên source DB. Chuyển sang **Debezium 2.x Kafka signal channel** (`signal.enabled.channels=kafka`, `signal.kafka.topic`). Worker publish lên topic; Debezium connectors consume từ topic — source DB hoàn toàn không bị chạm.

## Stack
- Go 1.22 (`centralized-data-service`)
- segmentio/kafka-go v0.4.50
- Debezium MongoDB Connector 2.x
- React + Ant Design (`cdc-cms-web`)

## Liên quan
- Lesson tham chiếu: `bug-mongo-url-dynamic-source-2026-05-18` (audit dynamic source — không liên quan dây chuyền write nhưng cùng cấu hình `connection_registry`).
- Cũ: trước đây có biến `connectionOverrides` truyền vào `ReconHandler` chỉ phục vụ `resolveSourceMongoDSN` → đã loại.
