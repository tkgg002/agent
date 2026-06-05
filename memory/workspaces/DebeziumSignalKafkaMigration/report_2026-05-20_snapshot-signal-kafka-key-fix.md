# Report 2026-05-20 — snapshot signal Kafka Key fix (Bug A + B)

## TL;DR
- **Bug A FIX**: worker `TriggerIncrementalSnapshot` đặt `Kafka Key = topic.prefix` (resolved từ Kafka Connect REST) thay vì `<db>.<collection>`. Debezium 2.5+ KafkaSignalChannel silently drop messages có key ≠ topic.prefix.
- **Bug B FIX**: CMS `injectDebeziumSignalDefaults` force-overwrite `signal.kafka.topic` + `signal.kafka.bootstrap.servers` (thay vì chỉ inject khi vắng) để chặn FE Vite placeholder leak (`__VITE_SIGNAL_KAFKA_TOPIC__`).
- **Verified**: sau fix, Debezium log `Requested 'INCREMENTAL' snapshot of data collections '[centralized-export-service.export-jobs]'`. Trước fix: 0 line như vậy bất kể bao nhiêu signal worker publish.
- **Bug C NEW (block shadow rows)**: Debezium 2.5.4 Mongo connector NPE tại `MongoDbIncrementalSnapshotChangeEventSource.lambda$emitWindowOpen$2(:228)` → snapshot fail right at chunk emit. Khác với 2 bug A/B (đã fix), Bug C là plugin bug — cần user quyết bump version hoặc đổi cách khác.

## Evidence chain

### Bug A: worker key sai
**Trước**:
- Dump `cdc.signal.commands`: 17 message worker publish với key `centralized-export-service.export-jobs`.
- 4 message manual test với key `cdc.goopay` → trước đó snapshot thành công 133 rows.
- Connector `topic.prefix` = `cdc.goopay`.
- Debezium 2.5+ KafkaSignalChannel#process compare `record.key()` với `topic.prefix` → mismatch → silently drop.

**Sau** (worker log):
```
debezium signal published  topic=cdc.signal.commands  connector=goopay-local
                            topic_prefix=cdc.goopay   signal_id=signal-1779249406874349000
```

### Bug B: CMS placeholder leak
**Trước** (connector config dump):
```json
"signal.kafka.topic": "__VITE_SIGNAL_KAFKA_TOPIC__"
```
**Trước** (Kafka Connect log):
```
Subscribing to signals topic '__VITE_SIGNAL_KAFKA_TOPIC__'
Error while fetching metadata: {__VITE_SIGNAL_KAFKA_TOPIC__=UNKNOWN_TOPIC_OR_PARTITION}
```
**Sau migrate + restart connector**:
```
signal.kafka.topic = cdc.signal.commands
Subscribing to signals topic 'cdc.signal.commands'
```

### End-to-end (sau cả A+B)
```
[03:56:53] Adding discovered server host.docker.internal:17017 to client view of cluster
[03:56:53] Requested 'INCREMENTAL' snapshot of data collections '[centralized-export-service.export-jobs]'
[03:56:53] ERROR Error while attempting to emit window open for chunk '6b1256f6...': null
           java.lang.NullPointerException
             at MongoDbIncrementalSnapshotChangeEventSource.lambda$emitWindowOpen$2(:228)
[03:56:54] Action execute-snapshot failed.
```

Signal đến đúng (Bug A+B đã thông). Snapshot fail ở plugin level (Bug C).

## Diff (key files)

### `cdc-cms-service/internal/api/system_connectors_handler.go`
- `injectDebeziumSignalDefaults` chuyển từ "inject if missing" → "force overwrite + log warn".
- Lý do: signal.* là infra config backend phải own, không cho FE/operator override. Defends 2 failure mode thực: (1) Vite placeholder, (2) operator typo.

### `centralized-data-service/internal/service/debezium_signal.go`
- Thêm method `ResolveTopicPrefix(ctx, connectorName)` — GET Kafka Connect REST `/connectors/{name}/config`.
- `TriggerIncrementalSnapshot` signature mới: `(ctx, connectorName, engine, database, collection, filter)`.
- `msg.Key = []byte(topicPrefix)` thay vì `qualified`. Comment giải thích Debezium 2.5+ requirement.

### Callers
- `internal/handler/recon_handler.go:344` — resolve connectorName trước call, fail-fast nếu vắng (no connector → no topic.prefix → impossible to build correct key).
- `internal/service/recon_heal.go:680` — reuse connectorName đã resolve cho health probe phía trên.

### Connector migrate (one-shot)
```bash
curl -s http://127.0.0.1:18083/connectors/goopay-{local,dev}/config | \
  jq '. + {"signal.kafka.topic":"cdc.signal.commands","signal.enabled.channels":"source,kafka"} | del(.name)' | \
  curl -X PUT -H 'Content-Type: application/json' --data-binary @- \
    http://127.0.0.1:18083/connectors/goopay-{local,dev}/config
curl -X POST "http://127.0.0.1:18083/connectors/goopay-{local,dev}/restart?includeTasks=true&onlyFailed=false"
```

## Bug C — Debezium 2.5.4 Mongo incremental NPE (chưa fix)

### Stack trace
```
Caused by: java.lang.NullPointerException
    at io.debezium.connector.mongodb.snapshot.MongoDbIncrementalSnapshotChangeEventSource
       .lambda$emitWindowOpen$2(MongoDbIncrementalSnapshotChangeEventSource.java:228)
    at MongoDbConnection.execute(...)
    at MongoDbIncrementalSnapshotChangeEventSource.emitWindowOpen(:225)
    at .readChunk(:301)
    at .addDataCollectionNamesToSnapshot(:445)
    at ExecuteSnapshot.arrived(:78)
```

### Phát hiện
- Cả 2 connector (`goopay-local` → mongo host:17017, `goopay-dev` → cluster 10.200.187.x) đều reproduce cùng NPE.
- Reproduce với payload tối giản (không filter, full snapshot).
- Snapshot init mode `initial` đã xong từ trước (resume token present), snapshot mới là incremental signal-driven.

### Phase trước & lý do tôi rút lại bump
Phase `snapshot-incremental-mongo-debezium-bump` đã propose bump 2.5.4 → 2.7.4 vì NPE này. User pushback "2.5.4 incremental đc ko, sao bump làm gì" → tôi xác nhận "có thể hallucinate, revert". Lần đó **tôi sai khi rút lại** — stack trace lần này confirm bug có thật, 100% deterministic.

### Options
| Option | Pros | Cons |
|---|---|---|
| Bump connector → 2.7.4+ | Fix tận gốc, official Debezium fix | Phải re-test toàn bộ pipeline với 2.7.4 (capture.mode option, signal action API có thể đổi) |
| Patch monkey-fix lambda$emitWindowOpen$2 | Không đổi version | High risk, không official, mất khi rebuild image |
| Dùng `snapshot.mode=initial` + `snapshot.collection.filter.overrides` thay vì signal | Tránh hẳn incremental signal path | Block toàn DB khi reset, mâu thuẫn với constraint "fintech 100M+ rows không được block" |

### Khuyến nghị
Bump 2.7.4 + re-test (mở phase mới, không tự ý bump). Đây là decision của user, không phải Muscle.

## Verification commands

```bash
# 1. Signal dump (key column)
docker exec gpay-kafka kafka-console-consumer \
  --bootstrap-server kafka:9092 --topic cdc.signal.commands \
  --property print.key=true --property key.separator='|' \
  --from-beginning --timeout-ms 3000

# 2. Connector config (signal.kafka.topic should be cdc.signal.commands)
curl -s http://127.0.0.1:18083/connectors/goopay-local/config | \
  jq '{"signal.kafka.topic", "signal.kafka.bootstrap.servers", "topic.prefix"}'

# 3. Connect log subscribe line
docker logs gpay-kafka-connect 2>&1 | grep "Subscribing to signals topic" | tail -3

# 4. End-to-end (worker → NATS → Kafka → connector)
docker run --rm --network cdc-bridge natsio/nats-box:latest \
  nats --server 'nats://cdc_worker:worker_secret_2026@nats:4222' \
  pub cdc.cmd.debezium-snapshot \
  '{"type":"incremental","database":"centralized-export-service","collection":"export-jobs","table":"sd_export_jobs_local","trace_id":"e2e","action":"snapshot_now","origin":"verify"}'

# 5. Worker log (should show topic_prefix=cdc.goopay)
tail -50 /tmp/cdc-worker.log | grep "debezium signal published"

# 6. Connector log (should show Requested INCREMENTAL — then NPE until Bug C resolved)
docker logs gpay-kafka-connect --since 1m | grep -E "Requested|NullPointerException" | tail -10
```

## Definition-of-done (phase này)
- [x] CMS force-overwrite signal.* → build pass
- [x] Worker ResolveTopicPrefix + key fix → build pass
- [x] Migrate 2 connector existing → PUT 200 + log "Subscribing to signals topic 'cdc.signal.commands'"
- [x] Worker restart với code mới → "debezium signal published topic_prefix=cdc.goopay"
- [x] Debezium nhận signal → "Requested 'INCREMENTAL' snapshot of data collections"
- [ ] Shadow row count tăng → **BLOCKED bởi Bug C** (Debezium 2.5.4 Mongo NPE)

## Honesty disclosure
- Tôi đã hứa user "không cheat DB" — báo cáo này không cheat. Shadow count CHƯA tăng, đã ghi nhận rõ và chỉ ra Bug C blocking.
- Phase trước tôi rút lại bump khi user pushback. Lần này có evidence (stack trace) chứng minh bump cần thiết. User vẫn quyết — Muscle không tự bump.
