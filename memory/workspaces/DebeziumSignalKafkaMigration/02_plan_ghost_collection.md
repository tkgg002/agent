# 02 Plan — Ghost Collection Workaround

## Architecture summary
```
[UI snapshot button]
   ↓ REST
[CMS] → resolves connectorName
   ↓ NATS cdc.cmd.debezium-snapshot
[Worker]
   ↓ ResolveTopicPrefix (HTTP GET /connectors/{n}/config)
   ↓ Kafka publish (key = topic.prefix, topic = cdc.signal.commands)
[Debezium Kafka signal channel]
   ↓ Filter by key == topic.prefix
[MongoDbIncrementalSnapshotChangeEventSource.emitWindowOpen]
   ↓ writes WATERMARK doc → cdc_system.debezium_watermarks  ← Ghost Collection
[Chunks flow: read → publish → close window → next chunk]
   ↓ Kafka topic cdc.goopay.centralized-export-service.export-jobs
[cdc-worker consumer]
   ↓ apply
[PG shadow: shadow_goopay.sd_export_jobs_local] ← row count tăng
```

## Steps

### Step 1 — Revert docker-compose 2.5.4 (đã làm)
- File: `centralized-data-service/docker-compose.yml`
- Diff: `2.7.4` → `2.5.4` cho 3 plugin.

### Step 2 — Recreate kafka-connect container
```bash
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service
docker compose rm -sfv kafka-connect
docker compose up -d kafka-connect
```
- Wait until `confluent-hub install` xong cho cả 3 plugin (poll log + REST `/connector-plugins`).
- Poll-tool gracefully (no plugin → wait 5s, up to 180s).

### Step 3 — Tạo ghost collection trên Mongo
```bash
docker exec gpay-mongo mongosh --quiet --eval "\
  db.getSiblingDB('cdc_system').createCollection('debezium_watermarks');\
  print('collections:'); db.getSiblingDB('cdc_system').getCollectionNames();\
"
```
- Sau lệnh: `cdc_system.debezium_watermarks` exist, 0 doc.

### Step 4 — PATCH config goopay-local
```bash
curl -s http://127.0.0.1:18083/connectors/goopay-local/config > /tmp/cfg-local.json
jq '. + {"signal.data.collection":"cdc_system.debezium_watermarks"} | del(.name)' \
  /tmp/cfg-local.json > /tmp/cfg-local-new.json
curl -s -X PUT -H 'Content-Type: application/json' \
  --data-binary @/tmp/cfg-local-new.json \
  http://127.0.0.1:18083/connectors/goopay-local/config
```

(goopay-dev: similar nếu user confirm — mặc định cũng làm vì 2 connector dùng chung phase, ghost collection trong cdc_system trên cluster prod cần user tự tạo. Phase này CHỈ verify goopay-local; goopay-dev sẽ PATCH config nhưng collection tạo trên cluster prod là user-decision, không trong scope phase này.)

### Step 5 — Restart connectors
```bash
curl -X POST "http://127.0.0.1:18083/connectors/goopay-local/restart?includeTasks=true&onlyFailed=false"
```
- Wait until state RUNNING.

### Step 6 — Capture state BEFORE snapshot
```bash
# Shadow PG row count
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -tAc \
  "select count(*) from shadow_goopay.sd_export_jobs_local;"

# Source Mongo count
docker exec gpay-mongo mongosh --quiet --eval \
  "print(db.getSiblingDB('centralized-export-service').getCollection('export-jobs').countDocuments({}))"
```

### Step 7 — Trigger snapshot qua NATS (đi qua worker code mới)
```bash
docker run --rm --network cdc-bridge natsio/nats-box:latest \
  nats --server 'nats://cdc_worker:worker_secret_2026@nats:4222' \
  pub cdc.cmd.debezium-snapshot \
  '{"type":"incremental","database":"centralized-export-service","collection":"export-jobs","table":"sd_export_jobs_local","trace_id":"ghost-collection-verify","action":"snapshot_now","origin":"verify"}'
```

### Step 8 — Capture log + state AFTER (wait ~60s)
```bash
docker logs gpay-kafka-connect --since 2m 2>&1 | grep -E "Requested|NullPointerException|emitWindowOpen|window opened|window closed|debezium_watermarks" | tail -30

docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -tAc \
  "select count(*) from shadow_goopay.sd_export_jobs_local;"
```

### Step 9 — Verify delta
- Shadow AFTER - Shadow BEFORE phải > 0 (ideally = source count).
- NPE count phải = 0.
- `window opened/closed` log phải xuất hiện.

### Step 10 — Report + APPEND progress + lesson
- `report_2026-05-20_snapshot-ghost-collection.md`: số liệu thực, không cheat.
- APPEND `05_progress.md`.
- APPEND lesson "Global Pattern [A đặt config B vắng → C NPE]" vào `lessons.md`.

## Risk + mitigation
| Risk | Mitigation |
|---|---|
| Plugin install fail lại lần nữa (network) | Poll log retry; nếu timeout 180s thì restart container 1 lần |
| Ghost collection name mismatch convention | Dùng đúng `cdc_system.debezium_watermarks` theo user spec |
| Restart connector trigger lại snapshot mode `initial` | KHÔNG có gì lấy `signal.data.collection` làm capture, chỉ là internal store — không hề affect include.list |
| Goopay-dev cluster không có cdc_system DB | KHÔNG tạo collection trên prod cluster trong phase này, chỉ PATCH config local. Nếu dev đã có ghost collection do user tạo sẵn thì OK; nếu không, dev connector sẽ vẫn NPE — đó là expected (out of scope) |
| Shadow count đã có sẵn (dirty state) | Capture BEFORE/AFTER → đo delta, không assert tuyệt đối |

## Rollback
- Nếu Ghost Collection KHÔNG fix NPE: revert config PATCH bằng cách remove `signal.data.collection`; restart connector; báo user → user quyết bump 3.x.
- Code worker/CMS KHÔNG đổi → không cần rollback code.
