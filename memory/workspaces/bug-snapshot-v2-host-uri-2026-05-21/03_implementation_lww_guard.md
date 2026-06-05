# 03_implementation_lww_guard — Technical Design

> **Phase**: `lww_guard`
> **Strategy**: Phương án D
> **Audience**: Muscle thực thi + reviewer.

## 1. High-level data flow (sau khi fix)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Mongo source (read-only)                              │
│                                                                          │
│  ┌──────────────────┐         ┌────────────────────────┐                │
│  │  Oplog stream    │         │  Mongo Find chunks     │                │
│  │  (Debezium)      │         │  (snapshot runner)     │                │
│  └────────┬─────────┘         └───────────┬────────────┘                │
│           │                               │                              │
│           │ source.ts_ms                  │ clusterTime (db.hello)       │
│           │ (oplog real ts)               │ (logical clock, NOT walltime)│
└───────────┼───────────────────────────────┼──────────────────────────────┘
            │                               │
            ▼                               ▼
   ┌────────────────┐              ┌────────────────────┐
   │ Kafka topic    │              │ NATS subject       │
   │ cdc.X.Y.Z      │              │ snapshot.v2.X.Y    │
   └────────┬───────┘              └─────────┬──────────┘
            │                                │
            ▼                                ▼
   ┌─────────────────┐               ┌────────────────────────────┐
   │ kafka_consumer  │               │ snapshot_runner_handler    │
   │ (envelope)      │               │ buildSnapshotEnvelope(     │
   │                 │               │   afterJSON, now,          │
   │                 │               │   clusterTimeMs            │ ←── NEW param
   │                 │               │ )                          │
   └────────┬────────┘               └────────────┬───────────────┘
            │                                     │
            └─────────────────┬───────────────────┘
                              ▼
            ┌────────────────────────────────────────┐
            │     EventHandler.HandleRaw (shared)    │
            │                                        │
            │  - parse envelope.source               │
            │  - parse envelope.data.source_ts_ms    │
            │  - record.Source = envelope.source     │  ←── NEW: propagate
            │  - record.SourceTsMs = source_ts_ms    │
            └────────────────────┬───────────────────┘
                                 ▼
            ┌────────────────────────────────────────┐
            │       BatchBuffer.batchUpsert          │
            │                                        │
            │  schemaAdapter.BuildUpsertSQLInSchema( │
            │     ..., source=record.Source,         │
            │     sourceTsMs=record.SourceTsMs )     │
            └────────────────────┬───────────────────┘
                                 ▼
            ┌────────────────────────────────────────────────────┐
            │       PG Shadow table (cdc_internal.<table>)       │
            │                                                    │
            │   INSERT ... ON CONFLICT (pk) DO UPDATE SET ...    │
            │   WHERE _source_ts IS NULL                         │
            │      OR _source_ts < EXCLUDED._source_ts           │ ←── NEW: strict <
            │      OR (_source_ts = EXCLUDED._source_ts          │ ←── NEW: tiebreaker
            │          AND _source = 'snapshot:v2'               │
            │          AND EXCLUDED._source <> 'snapshot:v2')    │
            └────────────────────────────────────────────────────┘
```

## 2. Schema changes

### 2.1 `cdcCols` map (Go source)

| Before | After | Type |
|---|---|---|
| 8 entries | 9 entries (`+_source_ts`) | `map[string]string` |

```go
"_source_ts": "BIGINT"
```

Lý do KHÔNG `DEFAULT 0` hay `DEFAULT NOW()`:
- `0` sẽ gây guard `IS NULL OR <` trigger update vô điều kiện cho row legacy.
- `NOW()` là wall clock PG, không phải oplog ts.
- `NULL` semantically đúng: "chưa biết ts source" → fall vào `IS NULL` branch → first write thắng.

### 2.2 DDL inline `createShadowTableV1WithCols`

Thêm dòng `"_source_ts" BIGINT,` SAU `"_source" VARCHAR(20) DEFAULT 'airbyte',`.

### 2.3 Migration 060

Apply ADD COLUMN cho bảng đã tồn tại. PG 11+ ADD COLUMN không default = metadata-only operation, instant. Không cần `CONCURRENTLY`.

Backfill block để comment (optional, không required cho correctness — guard `IS NULL` đã handle).

### 2.4 Migration 061

`snapshot_progress` thêm 2 column:
- `mongo_cluster_time_start_ms BIGINT NULL`
- `mongo_cluster_time_capture_method VARCHAR(32) NULL`

## 3. Code changes

### 3.1 `BuildUpsertSQLInSchema` WHERE clause (`schema_adapter.go:513-518`)

**Behavioral change**: `<=` → `<` + tiebreaker.

Before:
```go
whereClause = fmt.Sprintf(
    `WHERE %s."_source_ts" IS NULL OR %s."_source_ts" <= EXCLUDED."_source_ts"`,
    qualifiedTable, qualifiedTable,
)
```

After (xem code đầy đủ trong `09_tasks_solution_lww_guard.md` Edit #3):
```go
if hasSourceCol {
    whereClause = fmt.Sprintf(
        `WHERE %s."_source_ts" IS NULL `+
            `OR %s."_source_ts" < EXCLUDED."_source_ts" `+
            `OR (%s."_source_ts" = EXCLUDED."_source_ts" `+
            `    AND %s."_source" = 'snapshot:v2' `+
            `    AND EXCLUDED."_source" <> 'snapshot:v2')`,
        qualifiedTable, qualifiedTable, qualifiedTable, qualifiedTable,
    )
} else {
    // Schema cũ chưa có _source → guard ts-only
    whereClause = fmt.Sprintf(
        `WHERE %s."_source_ts" IS NULL OR %s."_source_ts" < EXCLUDED."_source_ts"`,
        qualifiedTable, qualifiedTable,
    )
}
```

**Lưu ý**: hash-dedup fallback (khi `sourceTsMs=0`) **giữ nguyên** — không đổi behavior cho legacy bridge/retry path.

### 3.2 `buildSnapshotEnvelope` signature (`snapshot_runner_handler.go:498`)

Before:
```go
func buildSnapshotEnvelope(afterJSON []byte, now time.Time) []byte
```

After:
```go
func buildSnapshotEnvelope(afterJSON []byte, now time.Time, clusterTimeMs int64) []byte
```

Caller update bắt buộc — Muscle phải `grep -n buildSnapshotEnvelope` để tìm tất cả call site và truyền `clusterTimeMs`.

### 3.3 `captureClusterTime` helper mới

Đặt trong `snapshot_runner_handler.go` cùng package. Signature:

```go
func captureClusterTime(ctx context.Context, db *mongo.Database, log *zap.Logger) (clusterTimeMs int64, method string)
```

Trả tuple để caller bind `mongo_cluster_time_capture_method` vào `snapshot_progress`.

Fallback chain xem `09_tasks_solution_lww_guard.md` Edit #5.

### 3.4 `EventHandler.HandleRaw` propagate `_source` từ envelope

**Audit step bắt buộc trước khi sửa** — `09_tasks_solution_lww_guard.md` Edit #6.

Hypothesis (Muscle verify trước khi code):
- Envelope có field `source` (vd `"source":"snapshot:v2"` cho snapshot, `"source":"debezium-mongodb"` cho realtime).
- HandleRaw parse envelope → tạo `Record` struct → fill `Source` field.
- BatchBuffer call `BuildUpsertSQLInSchema(..., source=record.Source, ...)`.

Nếu chain này đã đúng → không cần edit, chỉ verify trace.
Nếu chain hỏng (vd realtime đang hardcode `'debezium-v125'`) → minimal fix tại điểm break.

## 4. Backward compatibility

| Aspect | Status |
|---|---|
| Bảng V1 legacy chưa có `_source_ts` | ✅ Migration 060 ADD COLUMN NULL, không break row cũ |
| Row legacy `_source_ts = NULL` | ✅ Guard `IS NULL` branch — first write thắng |
| Bảng V2 đã có `_source_ts` | ✅ Migration 060 `IF NOT EXISTS` skip, không double-add |
| Snapshot run cũ trong `snapshot_progress` không có `mongo_cluster_time_start_ms` | ✅ Migration 061 ADD COLUMN NULL — row cũ NULL, không break read |
| `BuildUpsertSQLInSchema` caller hiện tại với `sourceTsMs=0` (bridge/retry) | ✅ Fallback hash dedup giữ nguyên |
| `_source` column chưa được set (NULL) | ⚠️ NULL không match `= 'snapshot:v2'` → KHÔNG trigger tiebreaker → behavior giống case "ts bằng nhau, không tiebreak" → no-op (keep current). Acceptable. |

## 5. Performance impact

| Aspect | Impact | Note |
|---|---|---|
| Migration 060 lock time | < 100ms/table cho ADD COLUMN no-default | PG 11+ metadata-only |
| Storage overhead | +8 bytes/row (BIGINT) | 50M row ≈ 400MB |
| Query plan WHERE clause | Constant string compare, no scan change | Negligible |
| Snapshot start latency | +1 RTT to Mongo cho `db.hello()` | < 50ms typical |
| `_source` column write | Đã có sẵn, chỉ value khác | 0 cost |

## 6. Observability

Log lines mới (Muscle add khi implement):

```
INFO  snapshot.v2 cluster time captured  cluster_time_ms=1779348000000 method=hello connection_code=goopay-pbs
WARN  snapshot.v2 clusterTime capture fallback to wall clock  error=...
INFO  snapshot.v2 row written  source=snapshot:v2 source_ts_ms=1779348000000
```

Metrics (nếu có Prometheus exporter):
- `cdc_snapshot_cluster_time_capture_method_total{method="hello|replSetGetStatus|walltime"}` — counter
- `cdc_upsert_skipped_by_occ_total{table=...,reason="ts_older|tiebreak_snapshot_loses"}` — counter (optional, phase sau)

## 7. Security

- Mongo `db.hello()` là read-only command, không write source store → tuân thủ L-CDC-golden-rule.
- KHÔNG đụng đến credentials, secret_ref, hay DSN resolution path.
- Migration script chạy `EXECUTE format(...)` với `%I` identifier escape → an toàn injection.
- `/security-agent` review BẮT BUỘC trước báo Done.

## 8. Future work (phase sau, KHÔNG trong scope `lww_guard`)

- Unify V1 + V2 schema generator về 1 nguồn truth.
- Consolidate `_deleted` vs `_gpay_deleted`.
- Nới `_source` VARCHAR(20) → VARCHAR(64).
- `_version` Lamport refactor (nếu nhu cầu xuất hiện).
- Snapshot recovery dùng `mongo_cluster_time_start_ms` để resume at consistent point.
