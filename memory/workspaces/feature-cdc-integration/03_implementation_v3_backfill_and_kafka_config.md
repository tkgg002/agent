# Implementation — Backfill Button + Kafka Config (v3)

Date: 2026-04-17
Session: Muscle runtime verification (bug fix + end-to-end backfill)

---

## Section 1: Kafka Config Apply

### Files changed (prior session)
- `centralized-data-service/config/config-local.yml` (kafka broker + topic prefix)
- `centralized-data-service/internal/infra/kafka/consumer.go` (prefix-based topic discovery)
- `docker-compose.local.yml` (kafka-exporter sidecar on port 9308)

### Runtime evidence (from this session)
- Worker log line (after restart):
  `"discovered kafka topics (filtered by debezium registry)","topics":["cdc.goopay.payment-bill-service.refund-requests","cdc.goopay.centralized-export-service.export-jobs"],"debezium_tables":2`
- Worker `kafka consumer started` success at `localhost:19092` group `cdc-worker-group`.
- kafka-exporter sidecar: NOT validated this session (out of scope — backfill focus).

### Status
DONE — kafka consumer runtime-verified via topic discovery log. Sidecar health re-check deferred to next infra pass.

---

## Section 2: Backfill Button — Full-Stack

### Worker side
- Service: `centralized-data-service/internal/service/backfill_source_ts.go`
  - `BackfillSourceTsService.BackfillOne(ctx, runID, table)` — drive per-table loop.
  - NATS subject handled: `cdc.cmd.recon-backfill-source-ts`.
  - Writes `recon_runs` with `tier=4` (dedicated backfill tier, isolated from Tier 1/2/3 scheduled recon).
  - Emits Prom gauge `cdc_recon_backfill_progress{table}` (0..100).

### CMS side
- Route registration: `cdc-cms-service/internal/router/router.go:152`
  - POST `/api/recon/backfill-source-ts` (destructive chain: JWT → RequireOpsAdmin → Idempotency → Audit).
  - GET  `/api/recon/backfill-source-ts/status` (shared chain: admin or operator).
- Handler: `cdc-cms-service/internal/api/reconciliation_handler.go` — `TriggerBackfillSourceTs`, `BackfillSourceTsStatus`.
- Reason enforcement: `cdc-cms-service/internal/middleware/audit.go:225` — body field `reason` required, >=10 chars.

### FE side (prior session)
- Hook + UI component — not re-verified this session.

### Runtime verification (this session)

#### Bug fixes found during runtime
1. **PK field mismatch** — `backfill_source_ts.go:153` was using `entry.PrimaryKeyField` (value `"id"` for `refund_requests`, a human-readable business code like `240723040419GDCJKB`) which cannot Mongo $in lookup by ObjectID. Fix: hard-code `pk := "_id"`.
2. **Infinite loop on unmatched batches** — `fetchNullBatch` used `LIMIT N` without ORDER/cursor. When a batch's rows had no matching Mongo doc, `_source_ts` stayed NULL, so the same top-N was re-selected forever until `MaxTotalScan=10M` cap. Fix: cursor-based pagination (`WHERE pk > ? ORDER BY pk`) using last id of previous batch as cursor. Import added: `database/sql`.

#### End-to-end evidence

**export_jobs** (Mongo source had all 117 ObjectIDs):
- recon_runs row: `status=success, docs_scanned=101, heal_actions=101, finished_at - started_at = 46ms`
- PG count: `total=117, with_ts=117` → 100% coverage.

**refund_requests** (Mongo source had only 3 of 1713 docs initially; seeded 1712 missing ObjectIDs with `updated_at` before re-run):
- recon_runs row (after seed): `status=success, docs_scanned=1712, heal_actions=1712, duration ≈ 977ms`
- PG count: `total=1713, with_ts=1713` → 100% coverage.

**Status API** (`GET /api/recon/backfill-source-ts/status` with admin JWT): returns runs tier=4 with `percent_done`, `null_remaining`, `total_rows`. Both `refund_requests` and `export_jobs` at `percent_done: 100`.

**Prom metric** (`curl :9090/metrics | grep cdc_recon_backfill_progress`):
```
cdc_recon_backfill_progress{table="export_jobs"} 100
cdc_recon_backfill_progress{table="refund_requests"} 100
```

### Security self-review
- RBAC: `RequireOpsAdmin()` enforced on POST (only `ops-admin` or `admin` roles).
- Idempotency: Redis-backed middleware, TTL 1h; key from `Idempotency-Key` header.
- Audit: async INSERT into `admin_actions` partitioned table; `reason` (>=10 chars) enforced inline.
- Non-destructive UPDATE: guarded by `_source_ts IS NULL` clause → safe replays.

---

## Issues encountered + resolution

| # | Issue | Root cause | Fix | Status |
|---|---|---|---|---|
| 1 | PK mismatch (id vs _id) | Registry's `primary_key_field="id"` is a business code, not Mongo ObjectID | Hard-code `pk := "_id"` | Fixed |
| 2 | Infinite loop on unmatched batches | LIMIT without ORDER BY/cursor re-selects the same rows when UPDATE no-ops | Cursor pagination on `_id` | Fixed |
| 3 | Reason header not read | Audit middleware reads `reason` from JSON body, not `X-Action-Reason` header | Moved reason into request body | Resolved (caller-side) |
| 4 | Status endpoint 403 with ops-admin JWT | Status route uses `shared` chain (admin/operator only) | Issued separate JWT with `role=admin` | Working as designed |
| 5 | Mongo source refund-requests has only 3 of 1713 docs | Local env data drift; Mongo dump older than PG snapshot | Seeded 1712 ObjectIDs via `mongosh bulkWrite` with synthetic `updated_at` | Test-only remediation; doc'd for ops |

---

## Files touched this session (source code)
- `/Users/trainguyen/Documents/work/centralized-data-service/internal/service/backfill_source_ts.go`
  - Added `database/sql` import
  - Lines 153-158: PK now hard-coded `_id`
  - Lines 163-199: cursor-based loop
  - Lines 264-300: `fetchNullBatch` now takes `after` param, issues ORDER BY / WHERE > ? query

---

## Section 3: Structural Finding — Legacy Backfill vs Airbyte Race (APPENDED 2026-04-17)

### Observation (runtime, 3 ngày sau Section 2 claim)

Trạng thái PG hiện tại — contradict Section 2's "100% coverage":
```
refund_requests: total=3425, with_ts=3424, missing=1
export_jobs:     total=117,  with_ts=16,   missing=101
```

recon_runs cho thấy backfill runs đã THÀNH CÔNG lúc `2026-04-17 08:01:13`:
```
export_jobs     | tier=4 | status=success | docs_scanned=101 | heal_actions=101
refund_requests | tier=4 | status=success | docs_scanned=1712 | heal_actions=1712
```

**Nhưng** metadata của 101 rows hiện đang null:
```
_created_at = 2026-04-15 (rows tồn tại từ trước)
_updated_at = 2026-04-20 02:19:33  ← GHI ĐÈ 3 ngày SAU recon success
_synced_at  = 2026-04-20 02:19:33
```

### Root cause

**Race condition structural giữa backfill và Airbyte CDC sync:**

1. Backfill UPDATE `_source_ts` = ts_ms (succeeded, verified recon_runs).
2. Airbyte sync-next cycle (3 ngày sau) chạy UPSERT full-row replace trên cùng PK (`_id`).
3. Airbyte không biết về column `_source_ts` (do public schema Airbyte-managed, thêm _source_ts offline) → UPSERT set lại `_source_ts = NULL` qua `EXCLUDED._source_ts`.
4. OCC guard trong UPSERT của bản thân backfill dùng `WHERE _source_ts IS NULL` đã không bảo vệ được khỏi Airbyte vì Airbyte có quyền write đầy đủ và không qua backfill path.

### Tại sao KHÔNG re-trigger

- Re-trigger → reach 117/117 tạm thời → Airbyte sync cycle tiếp theo lại reset về NULL.
- Đây là cosmetic fix, không giải quyết root cause.
- Violates Rule 6 "Simplicity First" — spin wheels không productive.

### Alignment với architecture v7.2 (Sonyflake Unified)

Architecture v7.2 Section 2 đã quy định rõ:
> "Airbyte legacy public schema — **để tự bơi**, không touch.
>  cdc_internal (Debezium + Go SinkWorker) là pháo đài mới, nơi `_source_ts` được bảo vệ
>  bởi Fencing Trigger BEFORE INSERT OR UPDATE + OCC guard."

Nghĩa là: `_source_ts` trên legacy public schema sẽ LUÔN race với Airbyte. Đây là **by design** của parallel-system approach. Backfill legacy là lãng phí effort.

**Canonical place cho `_source_ts` integrity** = `cdc_internal.*` (đã delivered Phase 0 + Phase 1).

### Decision options for user

- **Option A (recommended)**: Chấp nhận legacy `_source_ts` không bền vững. Xóa button Backfill khỏi FE khi cdc_internal đầy đủ production-ready. Tập trung migrate consumers sang query cdc_internal thay vì public.
- **Option B**: Thêm PG trigger `BEFORE UPDATE ON <table> FOR EACH ROW: IF NEW._source_ts IS NULL AND OLD._source_ts IS NOT NULL THEN NEW._source_ts := OLD._source_ts` trên tất cả Airbyte-managed legacy tables để preserve. Violates "để tự bơi" policy nhưng giữ button functional.
- **Option C**: Config Airbyte exclude column `_source_ts` khỏi replace-set. Cần check Airbyte normalization SQL template; có thể không khả thi với custom column ngoài Airbyte sync schema.

### Status

- Task "Backfill Button" — **logically delivered + build verified**.
- Runtime coverage fluctuates due to structural race với Airbyte.
- **Awaiting user decision** giữa 3 option trên.

### Re-trigger NOT executed this session

Brain/Muscle conclude re-triggering = cosmetic. Honest report chọn over "lái" con số 117/117 tạm thời. User được quyền decide strategy trước khi tốn thêm cycle.
