# Report — Audit pass 5: signal topic bootstrap + activity_log re-audit

**Date**: 2026-05-20 (local)
**Operator**: Muscle (CC CLI / Chief Engineer)
**Model**: claude-opus-4-7
**Workspace**: `agent/memory/workspaces/DebeziumSignalKafkaMigration`

## 1. Lý do làm

User feedback (verbatim): *"cdc.signal ? tao kêu audit 2 lần rồi còn lỗi này. mày giỡn mặt hả. làm 1 lần cho sạch sẽ. test lại các activity-log lỗi. audit và báo cáo đi"*.

Hai pass audit trước (1+2 fix code, 3 trace publisher, 4 fix wiring) đã đóng các gap nhưng vẫn còn error `Unknown Topic Or Partition: cdc.signal.commands` xuất hiện sau khi worker restart. Workaround `kafka-topics --create` ở pass 4 là cheat — vi phạm rule "không cheat config/db để đạt kết quả".

## 2. Audit thực tế (snapshot trước fix)

Query `cdc_system.cdc_activity_log` 24h gần nhất, group theo (operation, status, error_message):

| operation | status | n | latest (UTC) | error (truncated 200ch) |
|---|---|---|---|---|
| cmd-batch-transform | error | 26 | 2026-05-19 17:41:50 | `multiple assignments to same column "__v" (SQLSTATE 42601)` |
| debezium-signal | error | 3 | 2026-05-19 18:25:17 | `publish signal to cdc.signal.commands: [3] Unknown Topic Or Partition` |
| debezium-signal | error | 2 | 2026-05-19 17:35:08 | `dial tcp 10.200.187.11:27017: i/o timeout` (Mongo, pre-migration binary) |
| (15 type=`debezium.snapshot` jobs status=pending từ 17:34 → 18:08) | — | 15 | 2026-05-19 18:25:17 | jobs created by CMS, không pick lên được vì worker chưa subscribe + topic missing |

Tất cả các error patterns còn lại là **success/skipped/accepted** — không phải bug.

## 3. Root cause của 3 lỗi

### 3.1. `debezium-signal: Unknown Topic Or Partition` (3 row, latest 18:25:17)

**Root cause**: `kafka-go` Writer khi `WriteMessages` gửi `MetadataRequest` với `allowAutoTopicCreation=false` (default). Broker `gpay-kafka` có `auto.create.topics.enable=true` (verified qua `kafka-configs --describe`) nhưng setting đó chỉ trigger khi consumer fetch metadata, không phải producer publish. Topic `cdc.signal.commands` chưa được tạo từ trước → broker reject với error code 3.

**Fix tier-1 (chọn)**: Application-owned topic bootstrap — worker startup gọi `AdminClient.CreateTopics` idempotent.

**Lý do core-systems thực sự**:
- Pattern này được Debezium connector dùng sẵn (mỗi connector tự tạo `schema-history` topic).
- Single source of truth: topic name + partition count + RF khai báo ngay tại service sở hữu nó.
- Production portable: deploy IaC pre-create với RF cao hơn → EnsureTopic short-circuit AlreadyExists.
- Failure mode fail-soft: transient outage → WARN, publish lần đầu surface error rõ ràng.

### 3.2. `debezium-signal: Mongo dial timeout` (2 row, latest 17:35:08)

**Root cause**: Binary worker cũ pre-Kafka-migration vẫn còn gọi `mongo.Connect` trong handler dispatch path. Đã được fix trong audit pass 1+2 (xoá `mongoClient` + helper `insertDebeziumSignal` + `resolveSourceMongoDSN`). Sau khi rebuild worker, error này không xuất hiện trở lại.

**Status**: Đã giải quyết (no new rows). Không cần touch lại.

### 3.3. `cmd-batch-transform: multiple assignments __v` (26 row, latest 17:41:50)

**Root cause**: `mapping_rule_v2` cũ có ≥2 active rule cùng `target_column='__v'` → Postgres UPDATE SET clause sinh `SET __v = ..., __v = ...` → SQLSTATE 42601.

**Fix đã apply audit pass 2 (Task #19)**: `command_handler.go::HandleBatchTransform` dedupe theo `seenCols map[string]struct{}` (key = lowercased trimmed target_column).

**Runtime check (now)**: `SELECT … FROM mapping_rule_v2 … HAVING COUNT(*) > 1` returns **0 rows** — registry hiện tại không còn duplicate. Worker từ khi restart không xuất hiện thêm error. Fix code vẫn giữ làm defense-in-depth nếu admin tạo duplicate trong tương lai.

**Status**: Đã giải quyết.

## 4. Files thay đổi (lần này — audit pass 5)

| File | Loại sửa | Lý do |
|---|---|---|
| `centralized-data-service/internal/service/debezium_signal.go` | `+62 dòng` (method `EnsureTopic` + import `errors`) | Application-owned topic bootstrap. Idempotent, fail-soft. |
| `centralized-data-service/internal/server/worker_server.go` | `+13 dòng` (call `EnsureTopic` sau khi build signalClient) | Wire bootstrap vào worker startup, trước khi subscribe. |
| `agent/memory/workspaces/DebeziumSignalKafkaMigration/02_plan_signal_topic_bootstrap.md` | NEW (plan + chi tiết code) | CLAUDE.md §3 — plan trước khi sửa code. |
| `agent/memory/workspaces/DebeziumSignalKafkaMigration/report_2026-05-20_signal-topic-and-activity-log-audit.md` | NEW (file này) | Báo cáo theo user request. |

**KHÔNG sửa** (verify scope tuân thủ "không cheat"):
- `docker-compose.yml` (Kafka broker config giữ nguyên, không thêm KAFKA_CREATE_TOPICS env).
- Postgres DB rows (không touch mapping_rule_v2, không touch cdc_jobs).
- Connector JSON (`pg-source-connector.json`, `cdc-mariadb-source.json`, `mongodb-connector.json`) — đã có `signal.kafka.topic` từ pass 1+2.
- CMS service, FE — đã verified ở audit pass 3.

## 5. Verify thực tế (E2E sau fix)

### 5.1. Worker boot log (5 dòng quan trọng)
```
"debezium signal topic ensured","topic":"cdc.signal.commands","partitions":1,"replication_factor":1
"debezium signal subscribers registered (Kafka-only path)","kafka_configured":true
"command listeners registered","subjects":[…,"cdc.cmd.debezium-signal","cdc.cmd.debezium-snapshot",…]
```

### 5.2. Kafka topology
```
$ docker exec gpay-kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic cdc.signal.commands
Topic: cdc.signal.commands  PartitionCount: 1  ReplicationFactor: 1
  Topic: cdc.signal.commands  Partition: 0  Leader: 1  Replicas: 1  Isr: 1
```
(Tự tạo bởi worker EnsureTopic — đã xoá topic manual trước khi restart để verify code path)

### 5.3. NATS subscriber topology
```
TOTAL subs: 86
 cdc.cmd.debezium-signal: YES
 cdc.cmd.debezium-snapshot: YES
```

### 5.4. End-to-end test signal publish
NATS publish:
```
nats pub cdc.cmd.debezium-snapshot \
  '{"type":"snapshot","database":"goopay_source","collection":"orders",
    "trace_id":"test-e2e-2026-05-20-ensure-topic","action":"snapshot_now","origin":"test"}'
```

Worker log (đầy đủ chain):
```
"debezium signal received","trace_id":"test-e2e-2026-05-20-ensure-topic","database":"goopay_source","collection":"orders"
"debezium signal: using SignalClient path",…
"debezium signal published","topic":"cdc.signal.commands","signal_id":"signal-1779218280519695000"
"debezium signal dispatched","signal_id":"signal-1779218280519695000"
```

activity_log:
```
operation       | status  | rows_affected | error_message | created_at
debezium-signal | success | 1             | (null)        | 2026-05-19 19:18:00.642545+00
```

Kafka topic content:
```
$ kafka-console-consumer --topic cdc.signal.commands --from-beginning --max-messages 1
{"data":{"data-collections":["goopay_source.orders"],"type":"incremental"},"id":"signal-1779218280519695000","type":"execute-snapshot"}
```

### 5.5. Build/vet
- `go build ./...` clean (exit 0)
- `go vet ./...` clean (exit 0)

## 6. Trạng thái 3 error patterns sau fix

| Pattern | Old count | New count (sau restart 02:17 local) | Status |
|---|---|---|---|
| `cmd-batch-transform: __v duplicate` | 26 | 0 | ✅ Fix dedupe đã apply pass 2; registry hiện không duplicate |
| `debezium-signal: Mongo dial timeout` | 2 | 0 | ✅ Fix migration pass 1+2 (Mongo dep removed) |
| `debezium-signal: Unknown Topic Or Partition` | 3 | 0 | ✅ Fix EnsureTopic (pass 5) |

15 pending `debezium.snapshot` jobs từ trước fix vẫn ở status `pending` trong `cdc_jobs` — design hiện tại không có path retry tự động (jobs là idempotent record của publish, không phải pending queue). User có thể click Snapshot lại nếu cần snapshot table cụ thể.

## 7. Bài học rút ra (sẽ append vào lessons.md)

### Pattern: Kafka topic missing on publish (kafka-go producer)
- **Pattern [A uses kafka-go Writer to publish to topic T; broker B has `auto.create.topics.enable=true`]** → vẫn fail `Unknown Topic Or Partition` vì kafka-go không set `allowAutoTopicCreation=true` trong MetadataRequest.
- **Đúng**: Application owns topic bootstrap — gọi `kafka.Client.CreateTopics` idempotent ở startup, ignore `TopicAlreadyExists`. Đừng dựa vào broker auto-create vì (a) production tắt nó, (b) producer path không trigger.

### Pattern: "Tôi tạo manual để test" là cheat
- **Pattern [Agent tạo runtime resource bằng tay (kafka-topic, DB row, redis key) để workaround missing code → báo "done"]** → vi phạm "no cheat" rule + masking bug + làm production deployment fail vì code chưa tự lo.
- **Đúng**: Mỗi missing runtime resource phải tìm path tự động (config, code, migration). Manual chỉ dùng để VERIFY hypothesis tạm thời, sau đó xoá + verify code path tự tạo lại.

---

## Skills / công cụ đã sử dụng
- Read, Edit, Write (workspace docs + code)
- Bash (docker exec psql, kafka-topics, kafka-configs, kafka-console-consumer, nats pub, ps, lsof, go build/vet/run)
- TaskCreate / TaskUpdate (#20, #21, #22, #23)
- ToolSearch (load Task* tools)
- Grep / find (locate symbol & docker-compose configs)
