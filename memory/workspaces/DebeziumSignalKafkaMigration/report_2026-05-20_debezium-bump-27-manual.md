# Report 2026-05-20 — Debezium bump 2.5.4 → 2.7.4.Final (manual install)

## TL;DR
- **Bump SUCCESS technically**: 3 plugin Debezium 2.7.4.Final tải từ Maven Central (Confluent Hub catalog thiếu) và load vào kafka-connect OK. Cả 2 connector `goopay-local`, `goopay-dev` RUNNING/RUNNING với plugin mới.
- **Bug C KHÔNG fix bằng bump**: NPE vẫn còn ở cùng vị trí (`MongoDbIncrementalSnapshotChangeEventSource.lambda$emitWindowOpen$0`, line 219 trong 2.7.4 vs 228 trong 2.5.4 — cùng root cause). Stack trace verbatim ở dưới.
- **Root cause confirm**: Debezium MongoDB incremental snapshot watermark BUỘC ghi vào source connection (`signal.data.collection`). Không có config nào trong 2.7.4 cho phép watermark trên store khác (Kafka/PG/in-memory). Đây là design intent DBLog, không phải bug.
- **Implication cho prod**: Source DB prod read-only → Debezium incremental snapshot signal infeasible bất kể version. Bump KHÔNG đáng giá cho mục đích fix Bug C.
- **Recommendation**: Custom snapshot worker trong cdc-worker (Go) — bypass Debezium snapshot signal hoàn toàn. Debezium giữ CDC streaming (oplog read = không cần write source).

## Evidence

### Plugin version verify
```
$ curl -s http://127.0.0.1:18083/connector-plugins | python3 -c "import json,sys; [print(p['class'], p['version']) for p in json.load(sys.stdin) if 'Connector' in p['class']]"
io.debezium.connector.mongodb.MongoDbConnector 2.7.4.Final
io.debezium.connector.mysql.MySqlConnector 2.7.4.Final
io.debezium.connector.postgresql.PostgresConnector 2.7.4.Final
```

### Connector state
```json
{"name":"goopay-local","connector":{"state":"RUNNING"},"tasks":[{"id":0,"state":"RUNNING"}]}
{"name":"goopay-dev","  connector":{"state":"RUNNING"},"tasks":[{"id":0,"state":"RUNNING"}]}
```

### Signal nhận OK
```
ERROR ... emit window open for chunk '5365b882-3f67-4f6f-9285-ce04ad355acd'
WARN  Action execute-snapshot failed. The signal SignalRecord{id='signal-bump-verify-1779258749', type='execute-snapshot', ..., additionalData={channelOffset=22}}
```
- Signal `signal-bump-verify-1779258749` arrived và được processSignal đọc → Bug A (key routing) + Bug B (signal topic config) đã fix vẫn work trên 2.7.4.

### NPE stack — 2.7.4.Final (cùng root cause 2.5.4)
```
Caused by: java.lang.NullPointerException
  at io.debezium.connector.mongodb.snapshot.MongoDbIncrementalSnapshotChangeEventSource
     .lambda$emitWindowOpen$0(MongoDbIncrementalSnapshotChangeEventSource.java:219)
  at .emitWindowOpen(:216)
  at .readChunk(:291)
  at .addDataCollectionNamesToSnapshot(:428)
  at ExecuteSnapshot.arrived(:78)
```

### So sánh 2.5.4 vs 2.7.4
| | 2.5.4 | 2.7.4.Final |
|---|---|---|
| Line emitWindowOpen$ | :228 | :219 |
| Root cause | `signal.data.collection` null → NPE khi `.getDatabase().getCollection()` | **identical** |
| Validation upfront | none | none (vẫn lazy fail tại runtime) |
| Workaround dev-only | tạo ghost collection trên source Mongo | **identical, vẫn infeasible cho prod** |

→ Bump KHÔNG fix Bug C cho use case fintech read-only source.

## docker-compose diff (kept — 2.7.4.Final stable)
```yaml
command:
  - bash
  - -c
  - |
    set -euo pipefail
    PLUGIN_DIR=/usr/share/confluent-hub-components
    VERSION=2.7.4.Final
    BASE=https://repo1.maven.org/maven2/io/debezium
    mkdir -p "$$PLUGIN_DIR"
    for c in mongodb postgres mysql; do
      dst="$$PLUGIN_DIR/debezium-connector-$$c"
      if [ -d "$$dst" ] && ls "$$dst" 2>/dev/null | grep -q "$$VERSION"; then
        echo "[plugin] $$c $$VERSION already present, skip"; continue
      fi
      rm -rf "$$dst"
      tarball="debezium-connector-$$c-$$VERSION-plugin.tar.gz"
      url="$$BASE/debezium-connector-$$c/$$VERSION/$$tarball"
      echo "[plugin] download $$url"
      curl -fsSL --retry 3 --retry-delay 2 "$$url" -o "/tmp/$$tarball"
      tar -xzf "/tmp/$$tarball" -C "$$PLUGIN_DIR"
      rm -f "/tmp/$$tarball"
    done
    /etc/confluent/docker/run
```

## Why I don't propose Ghost Collection again
User chỉ ra constraint cốt lõi: **source DB prod read-only**. Ghost Collection (tạo `cdc_system.debezium_watermarks` trên source Mongo) chỉ work trên dev — không deploy được lên prod. Bài học đã ghi vào `lessons.md` (2026-05-20 entry: "Source DB Read-Only Constraint").

## Recommendation: bỏ Debezium incremental signal, làm custom snapshot trong cdc-worker

### Architecture
```
[UI snapshot button] → CMS → NATS cdc.cmd.snapshot
                                       ↓
[cdc-worker:SnapshotRunner (NEW)]
   ↓ open MongoClient (read-only credential to source)
   ↓ cursor: find({_id:{$gt:lastSeenId}}).sort({_id:1}).limit(batchSize)
   ↓ for each doc → transform → publish to Kafka cdc.goopay.<db>.<coll>
                                  (same envelope schema Debezium emits → existing apply consumer reuse)
                  → checkpoint lastSeenId vào PG cdc_system.snapshot_progress (control plane — pipeline OWN)
[existing shadow apply consumer]
   ↓ upsert shadow_<src>.<table>
```

### Pros
- KHÔNG đụng source (read-only credential, cursor read = không ghi)
- KHÔNG phụ thuộc Debezium plugin internal
- Checkpoint trong PG control plane mà pipeline đã own → resume sau restart
- Reuse existing Kafka topic + envelope → zero downstream code change
- Có thể chạy song song với Debezium CDC streaming (Debezium continue oplog read cho live changes)

### Cons
- Phải implement envelope schema Debezium tương thích (1 lần). Hoặc bypass Kafka, ghi shadow PG trực tiếp.
- Snapshot không có DBLog watermark gap-detection (live writes lúc snapshot có thể duplicate — nhưng worker apply UPSERT idempotent đã handle).

### Estimate
- ~1-2 ngày Muscle work: SnapshotRunner service, PG migration `cdc_system.snapshot_progress` table (control plane, KHÔNG đụng source/dest schema), NATS handler, integration test.

## Verification commands tôi đã chạy
```bash
# 1. Plugin tải Maven
for c in mongodb postgres mysql; do
  curl -sI -o /dev/null -w "%{http_code} $c\n" \
    "https://repo1.maven.org/maven2/io/debezium/debezium-connector-$c/2.7.4.Final/debezium-connector-$c-2.7.4.Final-plugin.tar.gz"
done
# 200 mongodb / 200 postgres / 200 mysql

# 2. Compose recreate
docker compose rm -sfv kafka-connect && docker compose up -d kafka-connect

# 3. Wait plugins available + version check
curl -s http://127.0.0.1:18083/connector-plugins | grep 2.7.4.Final

# 4. Connector status RUNNING/RUNNING (both)

# 5. Direct Kafka signal publish (no worker)
docker exec gpay-kafka bash -c "echo 'cdc.goopay|{...}' | kafka-console-producer ... --property parse.key=true --property key.separator='|'"

# 6. Log Connect → SAME NPE as 2.5.4
```

## Definition-of-done (phase này)
- [x] docker-compose manual-install pattern
- [x] Plugin 2.7.4.Final loaded (verified REST)
- [x] Connector RUNNING/RUNNING với plugin mới
- [x] Trigger snapshot direct Kafka publish → signal nhận
- [ ] ~~Shadow rows tăng~~ → **infeasible**: NPE same as 2.5.4, design constraint
- [x] Report file vật lý
- [x] Lesson APPEND (`#confluent-hub`, `#debezium`, `#read-only-source`)

## Honesty disclosure
- KHÔNG tạo collection nào trên source Mongo cho test này (rút kinh nghiệm phase Ghost Collection bị user chửi đúng).
- Đã rollback ghost collection trước test này: `cdc_system` database absent trên Mongo source.
- Bump 2.7.4 confirmed KHÔNG fix Bug C — đây là design constraint Debezium, không phải bug 2.5.4 specific.
- Plugin 2.7.4 keep nguyên (stable, không break CDC streaming) — đợi user quyết direction tiếp.
