# 01 — Requirements: snapshot-signal-kafka-key-fix (2026-05-20)

## Context
- User trigger 2 snapshot qua UI (`goopay-local`, `goopay-dev`). Worker log báo "debezium signal published" + "debezium signal end-to-end ready" (state=RUNNING, task_count=1) NHƯNG **0 row** vào shadow PG.
- User cáo buộc cheating sau khi từng có "133 rows success" — phải verify bằng evidence cứng.

## Evidence (đã thu thập)
1. **Dump Kafka topic `cdc.signal.commands`** (`docker exec gpay-kafka kafka-console-consumer ... --from-beginning`):
   - 17 messages từ worker với key = `centralized-export-service.export-jobs` (qualified `<db>.<collection>`)
   - 4 messages từ manual test với key = `cdc.goopay` (test thủ công bằng `kafka-console-producer`)
   - 3 messages từ debug test trước với key = `null`

2. **Connector config** (`curl /connectors/goopay-{local,dev}/config`):
   - `topic.prefix` = `cdc.goopay`
   - `signal.kafka.topic` = `__VITE_SIGNAL_KAFKA_TOPIC__` ← **PLACEHOLDER VITE LITERAL**
   - `signal.enabled.channels` = `kafka`

3. **Kafka Connect log**:
   ```
   Subscribing to signals topic '__VITE_SIGNAL_KAFKA_TOPIC__'
   Error while fetching metadata: {__VITE_SIGNAL_KAFKA_TOPIC__=UNKNOWN_TOPIC_OR_PARTITION}
   Assigned to partition(s): __VITE_SIGNAL_KAFKA_TOPIC__-0
   ```

4. **Topic list**: `__VITE_SIGNAL_KAFKA_TOPIC__` được Kafka auto-create (do consumer subscribe) nhưng KHÔNG có producer → vĩnh viễn 0 message.

## Root cause (2 bug độc lập, cùng tồn tại)
### Bug B (chặn first): signal.kafka.topic = placeholder Vite
- FE Vite (`import.meta.env.VITE_SIGNAL_KAFKA_TOPIC`) gửi POST với value chưa resolve = literal `__VITE_SIGNAL_KAFKA_TOPIC__`.
- CMS `injectDebeziumSignalDefaults` chỉ inject default khi key vắng (`if _, set := cfg[k]; !set { cfg[k] = v }`) → respect placeholder.
- Debezium subscribe topic không tồn tại → MỌI signal worker publish vào `cdc.signal.commands` đều bị bỏ qua.

### Bug A: worker key sai
- `debezium_signal.go:210-214` set `Key = qualified` (`<db>.<collection>`).
- Comment lines 206-209 viết: "Debezium does not use the key for routing" — **SAI**. Debezium 2.5+ KafkaSignalChannel filter message theo key matching `topic.prefix`. Key sai → drop silently.
- Bằng chứng đối chứng: 4 message thủ công với key=`cdc.goopay` từng snapshot thành công (trước khi Bug B xảy ra).

## Definition of Done
1. CMS reject hoặc auto-replace placeholder Vite (`__VITE_*__`) trong signal.kafka.topic.
2. CMS force-overwrite signal.kafka.topic/bootstrap.servers (backend takes ownership của infra config, không cho FE control).
3. Worker set Kafka Key = topic.prefix resolved từ Kafka Connect REST `/connectors/{name}/config`.
4. 2 connector existing được migrate (config update + restart) để subscribe `cdc.signal.commands`.
5. End-to-end test: user trigger snapshot → connector log "Snapshot — N records" → `count(*) shadow.<table>` tăng đúng số documents trong source collection.
6. Report + APPEND 05_progress + lesson Global Pattern.

## Out of scope
- FE typo `centrallized-export-service` (3 L's) — đã ghi nhận, để FE team xử lý.
- Bump Debezium plugin version (đã revert ở phase trước, giữ 2.5.4).
