# 01_requirements — Debezium Signal Kafka Migration

## Functional
1. Khi user nhấn "Snapshot Now" trên CMS (FE), backend phải **publish 1 message Kafka** lên topic signal (mặc định `cdc.signal.commands`).
2. Mọi Debezium connector đang chạy phải consume topic này và tự filter theo `data-collections` trong payload.
3. **TUYỆT ĐỐI KHÔNG** ghi/insert vào bất kỳ collection nào của source DB.
4. Nếu Kafka brokers chưa được cấu hình (`cfg.Kafka.Brokers` rỗng): backend phải từ chối lệnh với log rõ ràng, **không fallback** sang ghi source.

## Non-functional
- Backward compat: env var mới `CDS_DEBEZIUM_SIGNAL_KAFKA_TOPIC` (đã có trong `config.go:303`).
- File mẫu `deployments/debezium/mongodb-connector.json` phải dùng Kafka signal channel.
- CMS `buildConnectorConfig` (cho cả mongodb/mysql/postgres) phải thêm 3 key: `signal.enabled.channels`, `signal.kafka.topic`, `signal.kafka.bootstrap.servers`.

## Definition of Done
- [x] `go build ./...` sạch.
- [x] `go vet ./...` sạch.
- [x] `npx tsc -b` cho `SourceConnectors.tsx` không lỗi (các file khác như `TableRegistry.tsx` có lỗi unused-var pre-existing — out of scope).
- [x] Không còn dòng nào trong codebase ghi vào `debezium_signal` collection của source.
- [x] `mongodb-connector.json` không còn `signal.data.collection`; có 3 key Kafka signal.
- [x] `SourceConnectors.tsx::buildConnectorConfig` có 3 key Kafka signal ở cả 3 nhánh DB.
- [x] Workspace docs đầy đủ theo CLAUDE.md §7.

## Out of scope
- `TableRegistry.tsx` lint errors (TS6133) — pre-existing, không thuộc thay đổi này.
- Thay đổi behavior `recon_heal.go` (Phase A heal) — đã match chữ ký mới sẵn.
