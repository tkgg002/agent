# Plan — Phase 2: Debezium Incremental Snapshot + Parallel Backfill

> **Date**: 2026-04-21
> **Author**: Muscle (claude-opus-4-7[1m]) — 7-stage SOP Stage 2 PLAN
> **Scope**: Phase 2 T2.1 + T2.2 per v7.2 Section 8
> **Pre-approved context**: User chọn Option A cho legacy race → deprecate backfill button, migrate sang `cdc_internal`

---

## 0. Pre-flight findings (Stage 1+pre-flight)

### 0.1 Debezium connector config state
| Key | Current | Correct |
|---|---|---|
| `signal.data.collection` | `goopay.debezium_signals` ← SAI | `payment-bill-service.debezium_signal` |
| `database.include.list` | `payment-bill-service,centralized-export-service` | keep |
| `snapshot.mode` | `initial` | keep |
| `signal.enabled.channels` | (unset → default `source`) | keep |
| `capture.mode` | `change_streams_update_full` | keep |
| `collection.include.list` | 9 collections | keep |

**Root cause**: `goopay` database không tồn tại trong Mongo, collection `goopay.debezium_signals` không có → connector không watch được, mọi signal doc fired trước đây silently ignored.

### 0.2 Mongo signal collections đã tồn tại
- `payment-bill-service.debezium_signal` (singular) — 3 historical docs format đúng Debezium 2.5
- `centralized-export-service.debezium_signal` — 0 docs, collection mới

### 0.3 cdc_internal starting state
| Table | Rows | _source_ts > 0 | Mongo src |
|---|---|---|---|
| `cdc_internal.payment_bills` | 2 | 0 | 2 ✓ |
| `cdc_internal.export_jobs` | 117 | 0 | 117 ✓ |
| `cdc_internal.refund_requests` | 1719 | 0 | 1719 ✓ |

**Count đã match 100%** từ Phase 1. Vấn đề: `_source_ts = 0` toàn bộ (Phase 1 bug extract `source.ts_ms`). OCC baseline bị hỏng.

### 0.4 cdc_internal.table_registry state
- **0 rows**. Chưa có table nào được registered. `profile_status` CHECK ∈ `{pending_data, syncing, active, failed}`.

### 0.5 SinkWorker runtime state
- **KHÔNG đang chạy** (grep `ps aux` rỗng).

---

## 1. Goal (DoD per user order)

1. `cdc_internal.{export_jobs, refund_requests}` = status `active` trong `table_registry`.
2. 100% dữ liệu sạch = rows match Mongo source + `_source_ts > 0` (OCC baseline hợp lệ).
3. Streaming CDC tiếp tục hoạt động song song snapshot (không downtime).
4. Snapshot không ghi đè stream updates mới hơn (`ON CONFLICT DO NOTHING` per user prescription).

---

## 2. Strategy — 2 hợp lệ options, chọn B

### Option A — Re-snapshot with OCC DO UPDATE (current SinkWorker logic)
- Fire incremental snapshot → SinkWorker dùng logic hiện tại `ON CONFLICT DO UPDATE WHERE EXCLUDED._source_ts > target._source_ts`.
- Vì tất cả target `_source_ts = 0`, snapshot events sẽ UPDATE hết → ts_ms mới.
- **Rủi ro**: Race — nếu stream update row A tại T=100 (cdc_internal.A._source_ts=100) rồi snapshot fire tại T=200, snapshot event sẽ có ts_ms ≥ 100 (snapshot read-time) → UPDATE với data có thể stale hơn stream update.
- **Ưu điểm**: Không cần code change.
- **Nhược điểm**: Violate user prescription "ON CONFLICT DO NOTHING".

### **Option B — Snapshot-aware dispatch (RECOMMENDED per user prescription)** ✓
- Detect snapshot event qua `envelope.source.snapshot ∈ {"true", "last", "incremental"}`.
- Snapshot path → `ON CONFLICT (_gpay_source_id) DO NOTHING` (chỉ fill missing rows).
- Streaming path → giữ OCC logic hiện tại.
- **Ưu điểm**: Snapshot không thể ghi đè stream; semantic an toàn.
- **Nhược điểm**: Với existing rows đã có ts=0, DO NOTHING sẽ KHÔNG fix `_source_ts`. Cần riêng 1-shot UPDATE sau khi snapshot drain xong (nếu user cần fix baseline).

**Chọn Option B** vì user prescribe rõ + architecture an toàn hơn.

---

## 3. Execution sequence (Stage 3 EXECUTE)

### Step 3.1 — Patch connector config (Brain/Ops action)
```bash
curl -X PUT -H "Content-Type: application/json" localhost:18083/connectors/goopay-mongodb-cdc/config \
  -d '{
    "connector.class":"io.debezium.connector.mongodb.MongoDbConnector",
    ...(keep existing)...
    "signal.data.collection":"payment-bill-service.debezium_signal"
  }'
```
- Debezium Kafka Connect sẽ reload task, giữ offset → không re-snapshot ngoài ý muốn.
- **Verify**: `curl /connectors/.../status` = RUNNING; `/config` confirms new signal.data.collection.

**Rủi ro**: task restart có thể re-snapshot nếu offset reset. Cần watch Kafka consumer lag trước/sau.

### Step 3.2 — Ensure target signal collection tồn tại + writable
- `payment-bill-service.debezium_signal` đã có — skip create.
- Verify grant: current connector user có `find + changeStream` permission (RUNNING state implies OK).

### Step 3.3 — Code change SinkWorker: snapshot-aware dispatch (T2.2)
**Files to modify** (Muscle sẽ thực thi khi user duyệt plan):
- `cdc-system/centralized-data-service/internal/sinkworker/envelope.go`:
  - Add field `Source.Snapshot string` to envelope struct (nếu chưa có).
  - Helper: `IsSnapshotEvent(env) bool` — trả true khi `source.snapshot ∈ {"true","last","incremental"}`.
- `cdc-system/centralized-data-service/internal/sinkworker/upsert.go`:
  - Add second builder `buildUpsertSQLSnapshot(table, record)` → `INSERT ... ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted DO NOTHING`.
- `cdc-system/centralized-data-service/internal/sinkworker/sinkworker.go`:
  - `HandleMessage`: dispatch based on `IsSnapshotEvent(envelope)` → select SQL builder.
  - Metrics: counter `cdc_sinkworker_snapshot_events_total{table,outcome=inserted|skipped}`.
- Tests:
  - Add unit test `TestSnapshotDispatch` — 3 cases: streaming (uses OCC), snapshot new row (INSERT), snapshot existing row (DO NOTHING).

**Verify**: `go build ./... && go vet ./... && go test ./internal/sinkworker/... -count=1`.

### Step 3.4 — Register tables trong cdc_internal.table_registry
Before firing snapshot, create registry rows với status=`syncing`:
```sql
INSERT INTO cdc_internal.table_registry (target_table, source_db, source_collection, is_financial, profile_status, ...)
VALUES
  ('export_jobs', 'centralized-export-service', 'export-jobs', false, 'syncing', ...),
  ('refund_requests', 'payment-bill-service', 'refund-requests', true, 'syncing', ...);
```
- Schema_approved_at: NULL (admin review pending).
- Sau snapshot drain xong → UPDATE status='active'.

### Step 3.5 — Start SinkWorker
```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
CONFIG_PATH=config/config-local.yml nohup go run ./cmd/sinkworker > /tmp/sinkworker.log 2>&1 &
```
- Verify claim machine_id + fencing_token log.
- Verify subscribe 3 Debezium topics with consumer group `cdc-v125-sink-worker`.

### Step 3.6 — Fire incremental snapshot signals
Insert signal docs into `payment-bill-service.debezium_signal`:
```javascript
// Export_jobs (cross-db snapshot)
db.debezium_signal.insertOne({
  type: "execute-snapshot",
  data: {
    type: "incremental",
    "data-collections": ["centralized-export-service.export-jobs"]
  }
});

// Refund_requests
db.debezium_signal.insertOne({
  type: "execute-snapshot",
  data: {
    type: "incremental",
    "data-collections": ["payment-bill-service.refund-requests"]
  }
});
```
**Monitor**:
- Kafka Connect log — confirm `Signal 'execute-snapshot' received` + `Starting incremental snapshot`.
- Kafka consumer lag — `kafka_consumergroup_lag{group=cdc-v125-sink-worker}` spike then drain.
- SinkWorker log — snapshot events dispatched via new path.

### Step 3.7 — Drain verification
- After lag=0, check cdc_internal.* rows still match Mongo counts (no duplicates, no losses).
- Spot-check 3 rows per table to see if `_source_ts` updated (will remain 0 vì DO NOTHING trên existing; expected given Option B).

### Step 3.8 — Optional: Baseline fix cho `_source_ts = 0`
Nếu user cần fix OCC baseline (tôi recommend):
- 1-shot admin SQL: UPDATE cdc_internal.* SET `_source_ts = <Mongo updated_at_ms>` WHERE `_source_ts = 0` — requires Mongo lookup.
- Or: stop Sink → truncate → re-snapshot với DO UPDATE semantics một lần (Option A) → restart Sink với Option B → ongoing stream.

**Decision**: skip Step 3.8 this round, mark registry `active` với ghi chú `_source_ts baseline=0 documented`.

### Step 3.9 — Mark registry status=active
```sql
UPDATE cdc_internal.table_registry
SET profile_status='active', schema_approved_at=NOW(), schema_approved_by='admin-local'
WHERE target_table IN ('export_jobs','refund_requests');
```

---

## 4. Verification checklist (Stage 4 VERIFY — DoD)

- [ ] `curl /connectors/.../config | grep signal.data.collection` → `payment-bill-service.debezium_signal`
- [ ] `docker exec gpay-mongo mongosh --eval '...find signal'` → signal docs present
- [ ] SinkWorker log: `kafka consumer started group=cdc-v125-sink-worker`
- [ ] Kafka Connect log: `Starting incremental snapshot for collection=<name>`
- [ ] Kafka Connect log: `Incremental snapshot completed`
- [ ] `cdc_sinkworker_snapshot_events_total{outcome=skipped}` > 0 (existing rows path)
- [ ] `cdc_internal.export_jobs` row count = 117, `cdc_internal.refund_requests` = 1719 (unchanged — good)
- [ ] No duplicate rows: `COUNT(*) = COUNT(DISTINCT _gpay_source_id)` per table
- [ ] Registry rows both `profile_status='active'`
- [ ] Live test: `mongosh insert 1 new doc` into `export-jobs` → see it arrive in `cdc_internal.export_jobs` via stream (not snapshot)
- [ ] Security gate: SinkWorker fencing still enforced (direct psql INSERT without session vars → ERROR)

---

## 5. Risks & mitigations

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | Connector config PUT triggers full re-snapshot (snapshot.mode=initial) | Medium | Debezium Connect REST PUT keeps offsets by default; verify consumer group offsets unchanged before/after |
| 2 | Signal collection not watched after config patch | Low | PUT applies + restart task; Debezium 2.5 auto-adds signal collection to monitored set |
| 3 | SinkWorker code change breaks streaming path | Medium | Keep streaming path SQL template bit-exact; unit test dispatch branch |
| 4 | `ON CONFLICT DO NOTHING` masks data-divergence (e.g., row mutated during snapshot) | Low | Log counter `outcome=skipped` emit at debug level per row (or aggregate); operator alert if skipped spike |
| 5 | Snapshot emits events for all 9 collections (collection.include.list), not just 2 requested | None | `data-collections` array in signal doc restricts scope — Debezium honors it |
| 6 | `_source_ts` remains 0 after snapshot (baseline bug not fixed) | Known | Explicit design decision (Option B). Document as tech-debt; revisit if needed |
| 7 | Fencing token expired during long snapshot drain | Low | SinkWorker heartbeat every 30s (existing); token refresh logic TBD if drain > 2min threshold |

---

## 6. Out-of-scope (deferred per user order)

- Multi-collection auto-routing (plan §8 T2.2 original scope → simplified here to 2 explicit tables)
- DLQ persistence (plan §4)
- Admin UI for financial schema approval (plan §11 #4)
- `_source_ts` baseline fix for existing 0-rows (Step 3.8 skipped)
- FE display of snapshot progress (separate task)
- Deprecation of legacy backfill button FE (separate task)

---

## 7. Effort estimate

| Step | Time |
|---|---|
| 3.1 Patch connector | 5 min |
| 3.3 SinkWorker code + tests | 45 min |
| 3.4 Registry rows | 5 min |
| 3.5 Start Sink | 5 min |
| 3.6 Fire signals | 5 min |
| 3.7 Drain verify | 10 min |
| 3.9 Mark active | 5 min |
| Stage 5 docs | 15 min |
| **Total** | **~95 min** |

---

## 8. Rollback plan

Nếu snapshot gây regression:
1. **Stop SinkWorker** (pkill sinkworker).
2. **Rollback connector config** — PUT lại `signal.data.collection: goopay.debezium_signals` (vô hiệu hóa signals again).
3. **Revert SinkWorker code** — `git checkout HEAD -- internal/sinkworker/{envelope,upsert,sinkworker}.go`.
4. **Delete registry rows** — `DELETE FROM cdc_internal.table_registry WHERE target_table IN (...)`.
5. **No DB data rollback needed** — snapshot với DO NOTHING không thay đổi rows.

---

## 9. Approval gate

**Awaiting user approval on**:
1. **Chọn Option A hay B** cho dispatch logic (plan mặc định B).
2. **Step 3.8 — `_source_ts` baseline fix** có thực hiện không? (plan mặc định SKIP).
3. **Run order các collections** — simultaneously hay tuần tự (export_jobs trước refund_requests)? (plan mặc định simultaneous, 2 signal docs).

Muscle **KHÔNG execute Stage 3** cho đến khi user trả lời ít nhất 3 câu trên hoặc duyệt toàn bộ plan.
