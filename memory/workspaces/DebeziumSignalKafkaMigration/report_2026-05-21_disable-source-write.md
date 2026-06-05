# Report 2026-05-21 — Disable Debezium Source-DB Write

> **Status**: PLAN COMPLETE — AWAIT BOSS VERB
> **No code change yet** (per Rule #3 Plan & Verify, Rule #12 plan-then-execute)

## TL;DR
- Boss complaint xác nhận đúng: Debezium connector ĐANG ghi `debezium_signals` + window markers vào source DB (142 docs trên 3 database).
- Root cause: `signal.data.collection` config (set bởi FE) buộc Debezium DBLog watermark coordination ghi vào source — bất kể `signal.enabled.channels` là `kafka` hay `source,kafka`.
- 2 path khả thi (A — disable, B — custom worker). Path A bỏ ghi source TRONG 30 phút nhưng mất Debezium incremental snapshot. Path B khôi phục snapshot qua custom worker (1-2 ngày).
- Đã viết đầy đủ requirements + plan + tasks-solution. **Chờ Boss verb.**

## What I did (chỉ research + doc, KHÔNG sửa code/cluster)

### 1. Đọc lessons trước (Rule #7)
- `lessons.md` line 53-59: incremental snapshot yêu cầu `signal.data.collection`, thiếu → silent fail.
- Line 3466-3469: source DB read-only → mọi pattern ghi vào source infeasible.
- Line 3433-3445: Kafka signal key routing (đã fix).

### 2. Live state survey (read-only)
```bash
# Connect endpoint
curl http://10.200.186.203:8083/connectors  # → ["goopay-ps","demo","goopay-ces","goopay-pbs"]

# Per-connector signal config
for c in goopay-ps demo goopay-ces goopay-pbs; do
  curl -s "http://10.200.186.203:8083/connectors/$c/config" | jq '
    {channels:."signal.enabled.channels", data:."signal.data.collection"}'
done
```
Kết quả:
| Connector | signal.enabled.channels | signal.data.collection |
|---|---|---|
| `goopay-ps` | `source,kafka` | `payment-service.debezium_signals` |
| `demo` | `<unset>` | `bank-service.debezium_signal` (typo) |
| `goopay-ces` | `source,kafka` | `centrallized-export-service.debezium_signals` |
| `goopay-pbs` | `source,kafka` | `payment-bill-service.debezium_signals` |

### 3. Source DB write evidence
```bash
docker exec gpay-mongo mongosh --quiet \
  "mongodb://root:***@10.200.187.11:27017,...,/?replicaSet=goopay&authSource=admin" \
  --eval '["payment-service","payment-bill-service","centrallized-export-service","bank-service"]
    .forEach(function(name){
      const sdb = db.getSiblingDB(name);
      const colls = sdb.getCollectionNames().filter(c => /debezium|signal|watermark/i.test(c));
      colls.forEach(c => print(name + "." + c + " count=" + sdb.getCollection(c).countDocuments({})));
    });'
```
Kết quả:
```
payment-service.debezium_signals count=38
payment-bill-service.debezium_signals count=42
centrallized-export-service.debezium_signals count=62
(bank-service: 0 — typo singular form không match)
```

Tổng **142 documents** Debezium đã ghi vào source DB. Sample doc xác nhận DBLog watermark pattern:
```js
{
  _id: 'ff7ea550-672c-4e18-a9b6-41bc88903742-open',
  type: 'snapshot-window-open',
  payload: ''
}
```

### 4. Code path map
- BE inject: `cdc-cms-service/internal/api/system_connectors_handler.go:446` (function `injectDebeziumSignalDefaults`).
- FE inject: `cdc-cms-web/src/pages/SourceConnectors.tsx:180-237` (Mongo, MySQL, PG branches).
- KHÔNG có Helm/k8s manifest hay worker code path nào khác inject signal config.

### 5. Workspace docs (Rule #7 No-overwrite)
APPEND vào existing workspace `DebeziumSignalKafkaMigration` với prefix `_disable_source_write`:
- `01_requirements_disable_source_write.md` — DoD, constraints, evidence.
- `02_plan_disable_source_write.md` — root cause re-confirm, options A/B/C/D/E matrix, Path A code demo, Path B architecture, verb dictionary.
- `09_tasks_solution_disable_source_write.md` — T1-T7 chi tiết + Path B sketch.
- `05_progress.md` — APPEND ENTRY 2026-05-21 (không xóa cũ).

## Root cause (technical)
Debezium MongoDB connector (≥1.7) khi nhận signal `execute-snapshot`:
1. Đọc trigger từ kafka channel (nếu `signal.enabled.channels` chứa `kafka`).
2. `MongoDbIncrementalSnapshotChangeEventSource.emitWindowOpen` ghi `{type:'snapshot-window-open'}` vào collection chỉ ra bởi **`signal.data.collection`** trên source.
3. Sau chunk done → ghi `'-close'`.
4. DBLog dùng 2 mốc này để phân biệt event live vs replay (gap-detection).

→ `signal.enabled.channels=kafka` chỉ điều khiển **nơi NHẬN signal**, KHÔNG ảnh hưởng **nơi GHI watermark**. Watermark MẶC ĐỊNH ghi vào source — đây là design intent Debezium DBLog, không phải bug.

## Options matrix (chi tiết trong `02_plan_*.md`)

| | Effort | Stop source write | Snapshot OK | Prod-ready |
|---|---|---|---|---|
| **A. Disable signal config** | 30m | ✅ | ❌ silent fail | ✅ |
| **B. Custom worker** | 1-2d | ✅ | ✅ qua worker | ✅ |
| C. Ghost collection | — | ✅ ghi 1 lần | ✅ | ❌ (loại — lesson) |
| D. Bump 2.7+ | 4h | ❌ vẫn ghi | ✅ | ❌ (đã thử) |
| E. PUT null thủ công | 5m | ✅ | ❌ | ❌ cheat config |

**Recommendation**: A trước (immediate stop), B follow-up (khôi phục snapshot).

## Files dự kiến sửa (khi Boss approve)
| File | Lines | Change |
|---|---|---|
| `cdc-cms-service/internal/api/system_connectors_handler.go` | 446 | `"source,kafka"` → `"kafka"` |
| `cdc-cms-service/internal/api/system_connectors_handler.go` | 451 (new) | Add `delete(cfg, "signal.data.collection")` |
| `cdc-cms-web/src/pages/SourceConnectors.tsx` | 180-181 | Strip Mongo signal.data.collection + channels=kafka |
| `cdc-cms-web/src/pages/SourceConnectors.tsx` | 205-206 | Idem MySQL |
| `cdc-cms-web/src/pages/SourceConnectors.tsx` | 233-235 | Idem PG (schema-scoped) |

## Verb dictionary cho Boss

| Verb | Hành động |
|---|---|
| `ship path A` | Muscle apply T1+T2, build + test, restart cmsapi local. KHÔNG đụng dev cluster. |
| `apply backfill` | Sau ship A, PUT 3 connector dev (10.200.186.203) bỏ source channel. Cần verb riêng vì chạm shared cluster. |
| `cleanup source signals` | Drop existing `debezium_signals` collection trên source — chỉ làm sau khi Boss confirm DBA cho phép write+drop. |
| `start path B` | Bắt đầu plan + implement custom snapshot worker (Mongo cursor + PG checkpoint). |
| `defer` | Document only, không sửa code. |

## Honesty disclosure
- KHÔNG sửa connector config trên 10.200.186.203 (chỉ GET, không PUT).
- KHÔNG drop collection nào trên source Mongo (chỉ countDocuments + find limit:3).
- KHÔNG sửa code FE/BE.
- KHÔNG restart cmsapi.
- Toàn bộ doc + plan đã có file vật lý trong `DebeziumSignalKafkaMigration` workspace.
- Live evidence (count + sample) bộc lộ password connection string trong report — đã redacted bằng `***` khi copy paste lệnh.

## Lessons reused
- 2026-05-20 #debezium #incremental-snapshot #silent-failure — confirm thiếu `signal.data.collection` = silent fail.
- 2026-05-20 #read-only-source — confirm prod source read-only, bỏ pattern ghi.
- 2026-05-20 #kafka #signal-channel #key-routing — đã fix, không liên quan phase này.

## Pending lesson (đề xuất APPEND khi ship path A)
> **Global Pattern**: `[Pipeline A] inject config C buộc store B ghi watermark vào source S]` → nếu S = read-only thì A vẫn ghi vào S bất kể channel khác đã enable. Đúng: tách watermark store khỏi source store (custom runner hoặc Debezium config strategy nếu vendor cung cấp). Áp dụng được cho: Debezium DBLog, Kafka Streams state store, AWS DMS CDC tasks, GoldenGate trail file.

## Definition-of-done (phase này)
- [x] Survey live state (4 connectors + 4 source DBs).
- [x] Map code callsites (BE 1 + FE 3).
- [x] Author 01/02/09 docs.
- [x] APPEND 05_progress.md.
- [x] Tạo report file vật lý.
- [ ] **Pending Boss verb** trước khi code/cluster change.
