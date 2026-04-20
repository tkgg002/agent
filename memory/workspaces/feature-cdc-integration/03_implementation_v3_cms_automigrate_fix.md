# 03 — Implementation v3 — CMS AutoMigrate cross-service fix

**Date**: 2026-04-17
**Author**: Muscle (claude-opus-4-7[1m])
**Scope**: Fix SQLSTATE 42P16 trên cả 2 service (Worker đã fix session trước, CMS miss cross-service search)

---

## Bug report (user)

```
/Users/trainguyen/Documents/work/cdc-cms-service/internal/server/server.go:52
ERROR: column "created_at" is in a primary key (SQLSTATE 42P16)
[1.345ms] [rows:0] ALTER TABLE "cdc_activity_log" ALTER COLUMN "created_at" DROP NOT NULL
```

## Root cause

`cdc_activity_log` được define trong SQL migration `centralized-data-service/migrations/010_partitioning.sql` với **composite PRIMARY KEY (created_at, id)** cho RANGE partitioning theo ngày. GORM AutoMigrate so sánh struct Go `model.ActivityLog` (chỉ có `ID` là primary key) vs DB schema, detect `created_at` có `NOT NULL` thêm → phát sinh `ALTER TABLE ... DROP NOT NULL`. Postgres reject vì column thuộc PK (42P16).

Root cause thực sự = **conflict giữa 2 source of truth cho schema**: GORM struct tag vs SQL migration file. Khi đã chọn SQL migration cho partitioned tables (bắt buộc vì GORM không support PARTITION BY), PHẢI remove AutoMigrate.

## Cross-service pattern search

```bash
rg "AutoMigrate" --type go -l
```

Kết quả 8 file match toàn monorepo:

| File | Trạng thái | Hành động |
|------|------------|-----------|
| `centralized-data-service/internal/server/worker_server.go:80` | Đã fix session trước (comment-only, đánh dấu remove) | Verify không regression (build + runtime) |
| `cdc-cms-service/internal/server/server.go:52` | **ACTIVE — root cause fire bug này** | **REMOVE + comment** |
| `reconcile-service/pkgs/database/orm/gorm.go:105` | Service riêng, migrate `ReconcileLogs/ExportHistory/...` | KEEP (không partitioned, không shared DB với CDC) |
| `simo-integration-service/pkgs/database/orm/gorm.go:77` | Service riêng, `SimoReportTemplate/SubmitReport` | KEEP |
| `distribution-hub/dihub-booking-service/.../orm_instance.go:56` | Service riêng, `BookingInfo/Passenger` | KEEP |
| `distribution-hub/dihub-booking-futa-connector/.../orm_instance.go:74` | Service riêng, `RequestLog` | KEEP |
| `distribution-hub/dihub-core-service/.../orm_instance.go:55` | Service riêng, `Partner/MigrationSeeds` | KEEP |
| `distribution-hub/dihub-service-template/.../orm_instance.go:56` | Template | KEEP |

**Conclusion**: Chỉ 2 service trong CDC stack (Worker + CMS) share partitioned PG schema. CMS là callsite MISSING → nguyên nhân bug hôm nay.

## Fix applied — CMS

**File**: `/Users/trainguyen/Documents/work/cdc-cms-service/internal/server/server.go`

### Before (line 51-53)

```go
// Auto-migrate: ensure required tables exist (prevents runtime errors)
db.AutoMigrate(&model.ActivityLog{}, &model.TableRegistry{}, &model.MappingRule{}, &model.WorkerSchedule{}, &model.ReconciliationReport{}, &model.FailedSyncLog{}, &model.Alert{})
logger.Info("Auto-migrate completed")
```

### After

```go
// Schema managed via SQL migrations (centralized-data-service/migrations/001-014
// + cdc-cms-service/migrations/003-013) — NOT auto-migrated.
// Reason: GORM AutoMigrate conflicts with partitioned tables (e.g. cdc_activity_log
// has composite PRIMARY KEY (created_at, id) for RANGE partitioning — GORM tries
// to DROP NOT NULL on created_at which Postgres rejects with SQLSTATE 42P16).
// Tables & owning migrations:
//   - cdc_table_registry         -> 001_init_schema.sql, 013_table_registry_expected_fields.sql
//   - cdc_mapping_rules          -> 001_init_schema.sql, cms/003_add_mapping_rule_status.sql
//   - cdc_activity_log           -> 006_activity_log.sql, 010_partitioning.sql (PARTITIONED)
//   - cdc_worker_schedule        -> 007_worker_schedule.sql
//   - cdc_reconciliation_reports -> 008_reconciliation.sql
//   - cdc_failed_sync_logs       -> 008_reconciliation.sql, 012_dlq_state_machine.sql
//   - cdc_alerts / cdc_silences  -> cms/013_alerts.sql
```

Phụ: remove import `"cdc-cms-service/internal/model"` khỏi `server.go` (không còn ref trực tiếp — repos/handlers tự import model trong file riêng).

## Build verify

```bash
cd /Users/trainguyen/Documents/work/cdc-cms-service && go build ./...
# exit 0 — no output

cd /Users/trainguyen/Documents/work/centralized-data-service && go build ./...
# exit 0 — no output
```

## Runtime verify (startup CLEAN cả 2)

Filter real errors: `grep -E '"level":"(error|fatal|panic)"|panic:|SQLSTATE'`

**Worker** (`/tmp/worker.log`):
```
(no match) CLEAN
```
Tail: `CDC Worker started :8082`, `partition dropper started`, `kafka consumer started topics:[cdc.goopay.payment-bill-service.refund-requests, cdc.goopay.centralized-export-service.export-jobs]`.

**CMS** (`/tmp/cms.log`):
```
(no match) CLEAN
```
Tail: `PostgreSQL connected`, `NATS JetStream connected`, `Redis connected`, `CMS Service started :8083`, `system health collector started`, `audit logger started`, `alert background resolver started`, `Starting Airbyte Reconciliation Worker interval=300`.

Không còn `ALTER TABLE ... DROP NOT NULL` + không còn SQLSTATE 42P16.

## Lesson — global pattern candidate

**Global Pattern [Fix pattern X in service A → Grep X toàn monorepo trước khi close]** → Result: cover N service với chi phí 2s search. Đúng: sau khi sửa 1 callsite, **LUÔN** `rg "pattern" --type go -l` trước khi mark task done. Session trước chỉ fix Worker → miss CMS → user báo bug same-pattern ở service khác.

Áp dụng cho: AutoMigrate conflicts, hardcoded secret, deprecated API, migration pattern, config key rename, log format... Bất kỳ fix code pattern nào.

## Files changed

- `/Users/trainguyen/Documents/work/cdc-cms-service/internal/server/server.go` (line 7-24 import block + line 51-66 migration comment; **removed line 52 `db.AutoMigrate(...)` + line 53 logger.Info + removed import `"cdc-cms-service/internal/model"`**)

## Files NOT changed (verified clean)

- `/Users/trainguyen/Documents/work/centralized-data-service/internal/server/worker_server.go` — already comment-only (session trước); build + runtime CLEAN confirm no regression.

## Security gate

No secret touched. No SQL injection surface. No auth bypass. Only remove ORM migration call + reorganize imports. Scope minimal.

## Next candidate work (out of scope this fix)

- Xoá `&model.Alert{}`, `&model.FailedSyncLog{}` etc khỏi CMS models nếu không dùng nơi khác? → cần scan repos/handlers — defer, không minimal impact.
- Wire CI guard: lint rule cấm `AutoMigrate` trong 2 service CDC (Worker + CMS) để ngăn regression future.
