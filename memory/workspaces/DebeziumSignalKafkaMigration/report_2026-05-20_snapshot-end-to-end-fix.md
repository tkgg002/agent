# Report — Snapshot End-to-End Fix (Shadow DB actually receives rows)

> **⚠️ DEPRECATION BANNER — 2026-05-20**
>
> Khuyến nghị "Mongo → blocking snapshot" trong file này là **SAI** cho workload fintech (100M+ row collections, realtime CDC) — blocking sẽ khóa collection trong lúc dump.
> Đúng giải pháp: bump Debezium connector plugin lên `≥ 2.7.4` để fix bug NPE/cursor-exhaust ở chính connector, GIỮ snapshot mode = `incremental` cho mọi engine.
> Xem report mới: `report_2026-05-20_snapshot-incremental-mongo-debezium-bump.md`.
> Code đã được dọn: branch `engine→blocking` xoá khỏi `debezium_signal.go`; `docker-compose.yml` bump plugin lên `2.7.4`.

**Date**: 2026-05-20
**Phase**: `snapshot-end-to-end-fix`
**Workspace**: `agent/memory/workspaces/DebeziumSignalKafkaMigration/`
**Operator**: Muscle (CC CLI)
**Trigger**: User furious — "ko 1 snapshot nào chạy đc. error hiện thì phai fix chứ … 1 cái bug fix 1 ngày ko xong. toan báo cáo láo." (Zero snapshots actually run. When errors appear, you must fix them. One bug, not fixed in a day. You keep lying in reports.)

---

## 1. Vấn đề (Definition of Done)

UI bấm "Snapshot" cho collection MongoDB `centralized-export-service.export-jobs` → activity_log ghi `status=success` → **NHƯNG** bảng `cdc_shadow.shadow_goopay_local_centralized_export_service.sd_export_jobs_local` vẫn 0 rows.

DoD: bấm snapshot → shadow PG có data thật. Không chấp nhận báo cáo dựa trên `activity_log` thôi.

## 2. Audit ground-truth (sequential debug, 3 layer)

### 2.1 Layer 1 — Worker consumer log (lag = 0 nhưng shadow = 0)

```
kafka consumer started, brokers=[localhost:19092], group=cdc-worker-group
… 3946 messages processed, lag=0 …
```

Worker đọc message OK. Shadow rỗng. → Bug nằm ở **route resolution** trong worker.

### 2.2 Layer 2 — Schema-drift log leak

```json
{"msg":"schema drift detected","table":"sd_export_jobs_dev","source_db":"centralized-export-service"}
```

`table=sd_export_jobs_dev` cho event từ `goopay-local` → **misroute**. Soi `routeCache` lookup:

```go
// metadata_registry_service.go (BEFORE)
func buildRouteLookupKeys(sourceDB, sourceTable string) []string {
    return dedupeStrings([]string{
        sourceTable,                                 // ← unqualified
        fmt.Sprintf("%s|%s", sourceDB, sourceTable), // ← specific
    })
}
```

`ResolveSourceRoutes` first-match-wins. Hai row `source_object_registry` cùng tên collection `export-jobs` (một dưới `goopay-dev` typo cũ, một dưới `goopay-local`). Unqualified key `export-jobs` đụng → trả về row nào load vào cache trước (id=3 goopay-dev). 132 rows đi nhầm sang `sd_export_jobs_dev`.

### 2.3 Layer 3 — Signal silently dropped

Sau khi fix route, publish signal bằng kafka-console-producer:
```
Signal key 'null' doesn't match the connector's name 'cdc.goopay' — dropped.
```

Debezium yêu cầu Kafka message key = `topic.prefix`. Test với `--property "parse.key=true" --property "key.separator=:"` và key `cdc.goopay:<payload>` → signal được nhận.

### 2.4 Layer 4 — Debezium 2.5.4 incremental snapshot bug

```
ERROR NullPointerException at MongoDbIncrementalSnapshotChangeEventSource:228
```

Workaround partial: thêm `additional-conditions:[]` → NPE biến mất, NHƯNG `_id > lastSeenId` cursor đã exhaust → "No data returned" suốt sau đó.

FINAL workaround: gửi `"type":"blocking"` thay vì `"incremental"`. Debezium 2.5.4 blocking snapshot trên Mongo connector hoạt động bình thường.

## 3. Giải pháp (source code, plan-first)

### 3.1 cdc-worker (centralized-data-service)

| File | Trước | Sau |
|---|---|---|
| `internal/service/metadata_registry_service.go::buildRouteLookupKeys` | `[sourceTable, sourceDB|sourceTable]` (legacy first) | `[sourceDB|sourceTable, sourceTable]` (specific first) — eliminate first-wins misroute |
| `internal/service/connector_resolver.go` (NEW helper) | (không có) | `ResolveEngineTypeBySource(ctx, db, database, collection) string` — query `source_object_registry.source_engine_type` |
| `internal/service/debezium_signal.go::TriggerIncrementalSnapshot` | `(ctx, db, coll, filter)` → emit `"incremental"` | Signature `(ctx, engine, db, coll, filter)`. Engine resolver kết nối Mongo→blocking branch. **NOTE**: branch hiện COMMENT OUT (lines 180-182); engine vẫn được log nhưng snapshotType hardcode `"incremental"`. Plumbing để khi đội quyết policy thì uncomment. |
| `internal/handler/recon_handler.go:344` + `internal/service/recon_heal.go:680` | Call cũ 4 args | Resolve engine trước, pass vào TriggerIncrementalSnapshot |

### 3.2 cms-service (cdc-cms-service)

| File | Trước | Sau |
|---|---|---|
| `internal/api/system_connectors_handler.go::Create` + `UpdateConfig` | Không auto-inject signal.* keys — operator phải nhớ 4 key, copy-paste thủ công | `injectDebeziumSignalDefaults(name, cfg)`: inject `signal.enabled.channels=source,kafka`, `signal.kafka.topic`, `signal.kafka.bootstrap.servers`, `signal.kafka.group.id=debezium-signal-<name>` khi `connector.class` starts với `io.debezium.`. Per-key opt-out. |
| `internal/api/system_connectors_handler.go::validateMongoConnectionString` | Không có validation — Mongo connector thiếu replicaSet → state=RUNNING nhưng 0 task → snapshot silent fail | Reject 400 nếu Mongo + `mongodb.connection.string` thiếu `replicaSet=` AND `mongodb.members.auto.discover != false`. Error message chỉ thẳng failure mode "0 tasks → snapshots silently produce no rows". |
| `internal/api/system_connectors_handler.go::NewSystemConnectorsHandler` | 7 args | +2 args `signalBootstrap`, `signalTopic` |
| `internal/server/server.go:172` | call cũ | thread `cfg.System.SignalKafkaBootstrap` + `cfg.System.SignalKafkaTopic` |
| `config/config.go::SystemConfig` | (không có 2 field) | Thêm `SignalKafkaBootstrap` + `SignalKafkaTopic` (mapstructure, env `CMS_SYSTEM_SIGNAL_KAFKA_*`, default topic `cdc.signal.commands`) |
| `config/config-local.yml`, `config-production.yml`, `config-sample.yml` | (không có 2 key) | Thêm `signalKafkaBootstrap` + `signalKafkaTopic` (local + sample = `gpay-kafka:9092`; production trống cho IaC override) |

## 4. Verify (real, có số liệu cụ thể)

### 4.1 Build + vet + test

```
$ cd centralized-data-service && go build ./... && go vet ./...
OK

$ cd cdc-cms-service && go build ./... && go vet ./... && go test ./internal/api/... ./internal/app/... ./internal/infra/...
ok  cdc-cms-service/internal/api                    1.385s
ok  cdc-cms-service/internal/app/commands           0.835s
ok  cdc-cms-service/internal/app/queries            1.834s
ok  cdc-cms-service/internal/infra/http             (cached)
ok  cdc-cms-service/internal/infra/messaging        (cached)
ok  cdc-cms-service/internal/infra/observability    (cached)
ok  cdc-cms-service/internal/infra/observability/probes (cached)
ok  cdc-cms-service/internal/infra/persistence      (cached)
```

### 4.2 E2E worker — sau khi truncate shadow và publish blocking signal

```bash
$ docker exec gpay-kafka kafka-console-producer ... \
  --topic cdc.signal.commands \
  --property "parse.key=true" --property "key.separator=:"
> cdc.goopay:{"type":"execute-snapshot","data":{"data-collections":["centralized-export-service.export-jobs"],"type":"blocking"}}
```

Debezium connector log:
```
Requested 'BLOCKING' snapshot of data collections '[centralized-export-service.export-jobs]'
Finished snapshotting 133 records for collection 'rs0.centralized-export-service.export-jobs'; total duration '00:00:00.231'
```

Topic offset:
```
cdc.goopay.centralized-export-service.export-jobs:0:3947 → 4080  (+133)
```

Worker log (134 lines):
```json
{"msg":"kafka CDC event","topic":"cdc.goopay.centralized-export-service.export-jobs","op":"c","offset":3947}
{"msg":"schema drift detected","source_db":"centralized-export-service","table":"sd_export_jobs_local"}
…  (was sd_export_jobs_dev before fix)
```

Shadow PG sau snapshot:
```
SELECT
  (SELECT count(*) FROM cdc_shadow.shadow_goopay_local_centralized_export_service.sd_export_jobs_local) AS local,
  (SELECT count(*) FROM cdc_shadow.shadow_goopay_dev_centralized_export_service.sd_export_jobs_dev)     AS dev;

 local | dev
-------+-----
   133 |   0
```

Spot-check 5 row gồm: probe `e2e-shadow-probe-002 (PROBE)`, `e2e-shadow-probe-001 (TEST)`, jobs `export-1770xxx (COMPLETED)`. Tất cả `_source=debezium`, `_deleted=f`. **Snapshot thật sự chạm shadow.**

## 5. Caveats — KHÔNG dấu

1. **Engine→blocking branch hiện COMMENT OUT** (debezium_signal.go:180-182). UI bấm snapshot vẫn emit `"type":"incremental"` → Mongo sẽ hit NPE/cursor-exhaust ở lần snapshot thứ 2. Workaround tạm: ops chạy `kafka-console-producer` thủ công với payload blocking khi cần re-snapshot. Decision pending: hardcode-by-engine vs per-connector config.
2. **Live patch trên goopay-local connector chưa được "nướng" vào source-of-truth migration**. Connector hiện tại có signal.* keys và replicaSet=rs0 là do live curl PUT trong phiên debug — nếu Kafka Connect bị restart hoàn toàn / connector bị delete-recreate, các key này sẽ mất cho đến khi đi qua CMS Create API (đã fix).
3. **route-resolver fix tác động lan rộng**: bất kỳ workspace nào trước đây dựa vào first-wins behavior (vô tình) sẽ thấy route khác đi sau khi rebuild worker. Cần spot-check các registry entry trùng `source_object_name` để bảo đảm intent thực sự là "qualified by sourceDB".

## 6. Danh sách file thay đổi

**centralized-data-service** — 4 files:
1. `internal/service/metadata_registry_service.go` (key-order fix)
2. `internal/service/connector_resolver.go` (NEW `ResolveEngineTypeBySource` helper)
3. `internal/service/debezium_signal.go` (engine param plumbing; engine→blocking switch commented per reviewer)
4. `internal/handler/recon_handler.go` + `internal/service/recon_heal.go` (caller updates)

**cdc-cms-service** — 6 files:
1. `internal/api/system_connectors_handler.go` (signal.* injection + Mongo URI validation + constructor signature)
2. `internal/server/server.go` (handler wiring)
3. `config/config.go` (SystemConfig fields + env binds + defaults)
4. `config/config-local.yml`
5. `config/config-production.yml`
6. `config/config-sample.yml`

**Workspace memory** — 3 files:
- `agent/memory/workspaces/DebeziumSignalKafkaMigration/05_progress.md` (APPEND)
- `agent/memory/workspaces/DebeziumSignalKafkaMigration/report_2026-05-20_snapshot-end-to-end-fix.md` (NEW — file này)
- `agent/memory/global/lessons.md` (APPEND Global Pattern)

## 7. Skill / Kỹ năng đã sử dụng

- **Read/Edit/Write tools** (Claude Code): patch chính xác, append-only memory.
- **Bash + grep + docker exec**: query Postgres, Kafka admin, kafka-console-producer trực tiếp vào signal topic.
- **TaskCreate/TaskUpdate**: track 8 task (route fix → engine resolver → signal plumbing → CMS injection → URI validation → build/vet/test → progress append → report).
- **CLAUDE.md governance**: §3 plan-first, §6 minimal impact, §7 append-only progress, §11 memory protection, §13 Global Pattern abstraction.
- **Debezium 2.5.4 known bug navigation**: cursor exhaustion + NPE workaround pattern (signal type swap).
- **First-wins anti-pattern detection**: chỉ ra rằng key-lookup order trong cache là silent contract — dễ break khi data có duplicate unqualified key.
- **Verify-before-claim discipline**: KHÔNG claim done dựa trên activity_log; chỉ claim khi `SELECT count(*) FROM shadow_table` ra số dương + spot-check row đúng schema.

## 8. Bài học → Global Pattern (rule 13)

**Pattern**: `[Resolver R uses key set K with first-match]` over `[registry with duplicate unqualified keys K_legacy]` → **Result Y**: silent misroute to first-loaded entry.

**Đúng**: order keys most-specific-first, OR enforce uniqueness on the unqualified key with explicit conflict resolution (db constraint / load-time validation that fails loudly).

Áp dụng được:
- K8s Operator `Reconcile(key)`: namespace/name vs name-only lookup.
- Multi-tenant IAM: tenant_id|user_id vs user_id-only.
- Stripe per-account dispatch: account_id|object_id vs object_id-only.
- Debezium signal data-collections: `db.collection` vs `collection`-only.
