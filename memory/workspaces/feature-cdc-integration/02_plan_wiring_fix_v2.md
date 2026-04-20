# Plan: Wiring Fix v2 — Runtime QC Test Results

> Date: 2026-04-16
> Role: Brain QC

## QC Test Results — Mọi thứ đang rỗng

### 1. Reconciliation: KHÔNG BAO GIỜ CHẠY
- `cdc_worker_schedule`: 0 rows cho "reconcile" → schedule chưa seed
- `cdc_reconciliation_report`: 0 rows → không ai check
- Root cause: Worker seed default schedules (bridge, transform, field-scan, partition-check, airbyte-sync) nhưng **KHÔNG có "reconcile"**

### 2. Data lệch thực tế
| Source (MongoDB) | Dest (Postgres) | Lệch |
|:-----------------|:----------------|:------|
| export-jobs: 117 | export_jobs: 116 | -1 (miss) |
| refund-requests: 3 | refund_requests: 1713 | +1710 (Airbyte data cũ) |
| payment-bills: 2 | payment_bills: 1000004 | +1000002 (Airbyte data cũ) |
| 5 collections khác | 0 rows | Chưa sync |

### 3. Failed sync logs: 0 — không biết có fail không

### 4. FE pages: vỏ rỗng vì data = 0

## Fix cần làm

### F1: Seed "reconcile" schedule ✅ DONE
Worker startup → seed thêm schedule "reconcile" vào `cdc_worker_schedule`
- Added to default seeds (30 min interval)
- Also upserts if existing schedules missing "reconcile"

### F2: Wire reconcile vào schedule executor ✅ DONE
- Added `reconCore` to WorkerServer struct
- Added `case "reconcile"` → `s.runReconcileCycle(now)`
- `runReconcileCycle()` calls `reconCore.CheckAll(ctx)`, logs drift count

### F3: Test manual trigger ✅ DONE
- CMS API POST /api/reconciliation/check → Worker nhận → reconCore.RunTier1 → 8 tables checked
- cdc_reconciliation_report: 8 reports (3 drift, 5 ok)
- Tier 2 export_jobs: missing_count=1, missing_ids=["69819fa1e5e5161c3856bbef"]

### F4: Test heal ✅ DONE  
- Trigger: POST /api/reconciliation/heal/export_jobs
- Worker: fetch from MongoDB → BSON unwrap → SchemaAdapter upsert → success
- Result: export_jobs 116 → 117 (match source), record found in PG
- Post-heal Tier 1: export_jobs status=ok, diff=0

### Bug fixes during testing
- **ObjectID format mismatch**: ReconSourceAgent.GetIDs() returned `ObjectID("hex")` but PG stores `hex`. Fixed: `extractMongoID()` using `primitive.ObjectID.Hex()`
- **BSON type encoding**: MongoDB native types (ObjectID, DateTime, int32) can't encode to PG varchar. Fixed: `unwrapBSONValue()` + CoerceValue varchar coercion
- **MongoDB connection**: `host.docker.internal` not resolvable from host. Fixed: `directConnection=true` in config-local.yml

## Definition of Done
- [x] Schedule "reconcile" tồn tại trong cdc_worker_schedule
- [x] Reconciliation tự chạy theo schedule (scheduler triggered, 8 tables)
- [x] cdc_reconciliation_report có data (8 reports)
- [x] FE DataIntegrity hiện report với drift (API verified: 8 tables, 2 drift detected)
- [x] Heal hoạt động: missing record → fetched → inserted (117=117)
