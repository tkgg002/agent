# 01 Requirements — Bump Debezium 2.5.4 → 2.7.4.Final (manual install)

## Context
- Phase trước (`ghost-collection`) **INVALID** (read-only source constraint).
- Phase trước nữa (`signal-kafka-key-fix`) fix Bug A (worker key) + Bug B (CMS placeholder). Bug C (Mongo incremental NPE) chưa fix.
- Confluent Hub catalog không có 2.6/2.7/2.8/2.9 (chỉ tới 2.5.4 rồi nhảy 3.0.8).
- User quyết: bump 2.7.4 — Confluent Hub thiếu thì tải binary trực tiếp.

## Source binary
- Maven Central có sẵn `2.7.4.Final` cho 3 plugin (HEAD 200):
  - `https://repo1.maven.org/maven2/io/debezium/debezium-connector-mongodb/2.7.4.Final/debezium-connector-mongodb-2.7.4.Final-plugin.tar.gz`
  - `https://repo1.maven.org/maven2/io/debezium/debezium-connector-postgres/2.7.4.Final/debezium-connector-postgres-2.7.4.Final-plugin.tar.gz` (artifact name = `postgres`, không `postgresql`)
  - `https://repo1.maven.org/maven2/io/debezium/debezium-connector-mysql/2.7.4.Final/debezium-connector-mysql-2.7.4.Final-plugin.tar.gz`

## Functional Requirements
1. Update `docker-compose.yml` kafka-connect command: thay 3 lệnh `confluent-hub install` bằng curl + tar -xzf vào `/usr/share/confluent-hub-components/`.
2. Recreate `gpay-kafka-connect`; verify plugin classes của 2.7.4.Final loaded (REST `/connector-plugins` chứa `MongoDbConnector` với version `2.7.4.Final`).
3. Verify 2 connector (`goopay-local`, `goopay-dev`) RUNNING/RUNNING với plugin mới (config persisted trong Kafka `_connect-configs`, không cần re-register).
4. Trigger snapshot via worker NATS publish (worker code Bug A đã đúng).
5. Capture log Connect — verify (a) signal nhận được, (b) behavior khi không có `signal.data.collection`: KHÔNG được phép NPE; expect validation error rõ ràng hoặc fallback hợp lệ.
6. **NOTE constraint chưa giải quyết**: Debezium 2.7.x vẫn dùng watermark trên source collection cho Mongo incremental snapshot (chưa có incremental snapshot read-only-source-friendly variant). Bump 2.7.4 chỉ fix NPE/validation, KHÔNG fix root read-only constraint cho prod. Tuy nhiên user yêu cầu bump trước → tôi bump trước, sau đó báo behavior + bàn next step.

## Non-Functional Requirements
- KHÔNG đụng source DB (không tạo collection nào trên gpay-mongo).
- Tải binary qua HTTPS Maven Central (trusted).
- Persistent volume: KHÔNG cần — plugin re-download mỗi lần recreate, chấp nhận trade-off để giữ image gốc Confluent.
- Recreate idempotent: nếu plugin folder đã có, skip download (script check).

## Definition of Done
- [ ] docker-compose plugin install command thay curl + tar
- [ ] kafka-connect UP, REST `/connector-plugins` báo version `2.7.4.Final` cho cả 3
- [ ] 2 connector goopay-* state RUNNING/RUNNING với plugin 2.7.4
- [ ] Trigger snapshot → capture log Connect 60s
- [ ] Report behavior: (a) NPE biến mất / (b) validation error / (c) snapshot work
- [ ] Report file `report_2026-05-20_debezium-bump-27-manual.md`
- [ ] APPEND `05_progress.md`
- [ ] APPEND lesson Global Pattern (Confluent Hub gap → manual install Maven)
