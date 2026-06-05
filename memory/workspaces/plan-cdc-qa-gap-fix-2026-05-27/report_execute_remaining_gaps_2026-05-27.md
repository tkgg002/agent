# Report — Execute Remaining Gaps (G-9 refactor, G-NEW-19, G-NEW-24, G-NEW-29)

**Date:** 2026-05-27
**Agent:** Brain plan + Muscle execute (CC CLI)
**Workspace:** `plan-cdc-qa-gap-fix-2026-05-27`
**Driver doc:** `03_implementation_remaining_gaps.md`

---

## 1. Tóm tắt

Đã hoàn thành 4 gap còn lại sau audit, theo plan đã user approve (revise lần 1: bỏ AutoMigrate cho G-9, đổi sang `migrate.Run()`; loại bỏ CI 48h-soak khỏi G-NEW-29).

Phát hiện thêm trong quá trình execute: **audit gốc có 2 finding SAI** → đã drop khỏi plan và document trong `03_implementation_remaining_gaps.md §0` + `05_progress.md` Entry 14.

| Gap | DoD đã đạt | Evidence |
|---|---|---|
| G-9 E2E test load real migrations | ✅ | Test PASS 10.6s với `migrate.Run()`, KHÔNG còn raw CREATE TABLE inline |
| G-NEW-19 Delete ordering | ✅ | 3 test mới + 2 existing PASS (5/5) |
| G-NEW-24 Source DB overhead | ✅ | Metric + GORM callback hoạt động; unit test PASS; postgres/mongo exporter scrape đã có |
| G-NEW-29 Soak orchestrator | ✅ | Script syntax OK; pre-flight gracefully fail; runbook đầy đủ acceptance criteria |

---

## 2. Audit Correction (quan trọng)

| Gap | Audit cũ claim | Thực tế re-verify | Action |
|---|---|---|---|
| **G-11** | `BatchesFlushed` là `NewCounter` không label | Đã LÀ `NewCounterVec[sink,table]` từ trước (`pkgs/metrics/prometheus.go:167-173`); caller dùng `.WithLabelValues("postgres", tableName).Inc()` (`batch_buffer.go:191`) | DROP — không cần fix |
| **G-13** | PerSourcePool là DEAD CODE (NewPerSourcePool không có caller) | Đã wire đầy đủ: `kafka_consumer.go:96` (field), `:165` (constructor param), `:501` (apply); `worker_server.go:608` (instantiate `NewPerSourcePool(s.db, 0)`) | DROP — không cần fix |

**Root cause** của audit cũ sai: grep chỉ 1 file metric, không cross-check toàn repo. Bài học cho audit lần sau.

---

## 3. Chi tiết thay đổi

### 3.1. G-9 — E2E test dùng production migration runner

**File**: `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` (REWRITE)

**Thay đổi**:
- Import thêm `cdc-cms-service/internal/migrate` (production runner đã có sẵn — discovery quan trọng).
- Xoá hàm `runMigrations()` inline (50 dòng raw CREATE TABLE).
- Thay bằng `require.NoError(t, migrate.Run(db, false, true, zap.NewNop()))` — load embed FS `migrations/schema/*/*.sql` qua `migrate.Run()`.
- Thêm `postgres.BasicWaitStrategies()` + `postgres.WithSQLDriver("pgx")` để khắc phục race condition giữa container start và GORM connect (lỗi đầu tiên gặp khi run test: `connection reset by peer`).
- Tạo helper `seedShadowBindingDeps()` insert minimum FK rows (`connection_registry` + `source_object_registry`) — cần thiết vì `cdc_system.shadow_binding` có FK constraints sau khi migration 031 chạy. KHÔNG bypass FK bằng cách drop constraint, để test phản ánh đúng production.
- INSERT vào `cdc_system.schema_proposal` (KHÔNG phải `cdc_internal.schema_proposal`) — migration 037 relocate sang cdc_system.
- Bổ sung assertion `status='approved'` cuối test để kiểm tra UPDATE branch của handler.

**Evidence**:
```
=== RUN   TestApproveSchemaProposal_E2E
--- PASS: TestApproveSchemaProposal_E2E (10.63s)
PASS
ok  cdc-cms-service/internal/app/commands  11.477s
```

### 3.2. G-NEW-19 — Delete event ordering tests

**File**: `centralized-data-service/internal/service/schema_adapter_ordering_test.go` (EDIT)

**Thay đổi**:
- Thêm cột `_gpay_deleted BOOLEAN DEFAULT 0` vào schema `public.test_users` trong `setupTestDB`.
- Thêm helper `readDeletedFlag(t, db, sourceID) bool` đọc cờ tombstone (phân biệt với `readShadowDeleted` đọc absence).
- Thêm helper `deleteAwareSchema()` build `TableSchema` có `_gpay_deleted` column.
- Thêm 3 test mới:
  - `TestEventOrdering_DeleteTombstone`: insert ts=1000 → delete ts=2000 → tombstone flag = true (soft delete via `_gpay_deleted` field như sinkworker.go:123).
  - `TestEventOrdering_InsertAfterDelete_Resurrection`: insert → delete → insert ts mới hơn → row sống lại với `_gpay_deleted=false`.
  - `TestEventOrdering_UpdateAfterDelete_OCCDrop`: insert ts=1000 → delete ts=3000 → update ts=2000 (older replay) → OCC drop, tombstone preserved.

**Evidence**:
```
=== RUN   TestEventOrdering_OlderTsIgnored ... PASS
=== RUN   TestEventOrdering_HashTiebreaker ... PASS
=== RUN   TestEventOrdering_DeleteTombstone ... PASS
=== RUN   TestEventOrdering_InsertAfterDelete_Resurrection ... PASS
=== RUN   TestEventOrdering_UpdateAfterDelete_OCCDrop ... PASS
PASS  5/5 ordering tests
```

### 3.3. G-NEW-24 — Source DB overhead metric

**Files**:
- EDIT `pkgs/metrics/prometheus.go` — append metric `SourceQueryDuration` histogram (12 exponential buckets từ 1ms tới ~4s, labels `role` + `operation`).
- NEW `pkgs/database/metrics_callback.go` — `RegisterQueryMetrics(db, role)` đăng ký Before/After hooks cho 6 GORM verbs (query, create, update, delete, row, raw). Trước query: `tx.InstanceSet(startTimeKey, time.Now())`. Sau query: compute `time.Since(start)`, observe vào histogram. KHÔNG dùng các unexported types `*gorm.processor` (đầu tiên gặp lỗi `undefined: gorm.Callback` và `undefined: gorm.ProcessorImpl`) — inline registration cho từng verb.
- EDIT `pkgs/database/multi.go` — đổi signature `openGorm(dsn) → openGorm(dsn, role)` và gọi `RegisterQueryMetrics(db, role)` sau pool tuning. Update callsite duy nhất tại line 89.
- NEW `pkgs/database/metrics_callback_test.go` — 2 unit test PASS: histogram nhận observation khi query, error khi truyền nil db hoặc empty role.
- EDIT `deployments/prometheus/prometheus.yml` — viết lại comment `postgres-exporter` và `mongodb-exporter` job để làm rõ pairing với metric mới + yêu cầu scrape `pg_stat_statements`/`pg_stat_database`.

**Evidence**:
```
ok  centralized-data-service/pkgs/database  0.651s
   - TestRegisterQueryMetrics_RecordsHistogram PASS
   - TestRegisterQueryMetrics_RejectsBadInput PASS
go build ./... → exit 0
```

### 3.4. G-NEW-29 — Soak orchestrator + runbook

**Files**:
- NEW `scripts/soak_test.sh` (130 dòng, executable):
  - Env vars: `SOAK_DURATION_HOURS` (48), `K6_TARGET_URL`, `PPROF_URL`, `SNAPSHOT_INTERVAL_MIN` (30), `OUT_DIR`, `HEAP_GROWTH_LIMIT_MB` (50).
  - Pre-flight: kiểm tra `curl`, `k6` (FATAL nếu thiếu, exit 2); `go` optional (fallback raw heap nếu thiếu).
  - Khởi k6 background với `--duration ${SOAK_DURATION_HOURS}h`, log → `$OUT_DIR/k6.log`.
  - Loop: mỗi `SNAPSHOT_INTERVAL_SECONDS` curl pprof endpoint → `heap_<timestamp>.pb.gz` + `go tool pprof -top -unit=mb` → `heap_top_<timestamp>.txt`.
  - End-of-run: `go tool pprof -base first last` diff → `heap_diff.txt`. Parse total MB từ "Showing nodes" line. Exit code 1 nếu growth > limit.
  - `--dry-run`: chạy 5 phút (300s) snapshot mỗi 60s, dành cho CI smoke (KHÔNG check growth do baseline quá ngắn).
  - Trap EXIT cleanup k6 process.
- NEW `docs/runbooks/soak-test.md`:
  - Khi nào chạy: quarterly cadence, trước bump major version, sau OOM pager.
  - Tại sao KHÔNG CI: timeout ≤ 6h; staging only.
  - Pre-requisites: staging cluster, k6/go/curl, ≥ 5 GB disk.
  - Quy trình full run.
  - Acceptance criteria (heap growth ≤ 50MB, 0 panic, goroutine growth ≤ 10%, p99 < 5s, lag < 5k).
  - Smoke pattern cho CI.

**Evidence**:
```
bash -n scripts/soak_test.sh → exit 0 (syntax OK)
bash scripts/soak_test.sh --dry-run → exit 2 (pre-flight FATAL: k6 not installed local) — đúng behavior thiết kế
```

---

## 4. Files thay đổi (8 total)

### Edit (4)
1. `data-hub/cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` (REWRITE)
2. `data-hub/centralized-data-service/internal/service/schema_adapter_ordering_test.go` (+helpers +3 test +column)
3. `data-hub/centralized-data-service/pkgs/metrics/prometheus.go` (append `SourceQueryDuration`)
4. `data-hub/centralized-data-service/pkgs/database/multi.go` (openGorm signature + RegisterQueryMetrics call)
5. `data-hub/centralized-data-service/deployments/prometheus/prometheus.yml` (comments only)

### New (4)
1. `data-hub/centralized-data-service/pkgs/database/metrics_callback.go`
2. `data-hub/centralized-data-service/pkgs/database/metrics_callback_test.go`
3. `data-hub/centralized-data-service/scripts/soak_test.sh` (+x)
4. `data-hub/centralized-data-service/docs/runbooks/soak-test.md`

### Workspace docs
1. `agent/memory/workspaces/plan-cdc-qa-gap-fix-2026-05-27/03_implementation_remaining_gaps.md` (NEW)
2. `agent/memory/workspaces/plan-cdc-qa-gap-fix-2026-05-27/05_progress.md` (APPEND Entry 14)
3. `agent/memory/workspaces/plan-cdc-qa-gap-fix-2026-05-27/report_execute_remaining_gaps_2026-05-27.md` (NEW, file này)

---

## 5. Verification

```bash
# 1. Build + vet
cd data-hub/centralized-data-service
go vet ./...           # exit 0 ✅
go build ./...         # exit 0 ✅

# 2. Unit + service tests (mới)
go test ./pkgs/database/ -count=1
   # ok centralized-data-service/pkgs/database 0.651s ✅
go test ./internal/service/ -run TestEventOrdering -count=1 -v
   # 5/5 PASS ✅

# 3. E2E integration test (G-9)
cd ../cdc-cms-service
go test -tags=integration ./internal/app/commands/ -run TestApproveSchemaProposal_E2E -count=1
   # PASS 10.63s ✅

# 4. Soak script
cd ../centralized-data-service
bash -n scripts/soak_test.sh   # syntax OK ✅
bash scripts/soak_test.sh --dry-run   # exit 2 (k6 not installed) → pre-flight đúng ✅
```

---

## 6. Pre-existing failures KHÔNG do thay đổi này

Khi chạy `go test ./internal/service/ -count=1` (toàn package) phát hiện:
- `TestSanitizeMongoDSN/{no_creds,basic_auth,srv_auth,only_host_no_at}` FAIL — liên quan Entry 11 (DSN sanitization) đã có vấn đề trước; KHÔNG touch file `mongo_introspection.go`.
- `internal/handler` goleak FAIL trên `kafka-go.NewReader` goroutine — pre-existing kafka-go leak documented từ Entry 07/08; KHÔNG touch handler package.

Cần riêng task fix cho 2 issue trên (out-of-scope phase này).

---

## 7. Brain ↔ Muscle separation (§12)

- Brain (Antigravity) responsibility: viết `03_implementation_remaining_gaps.md`, không sửa code.
- Muscle (CC CLI) responsibility: 4 file edit + 4 file new, tất cả build/test passed.
- Trong session này, vai trò Brain + Muscle thực hiện trong cùng CC CLI process — user đã approve plan revised và yêu cầu execute (`vậy làm đi`), không có vi phạm separation vì sequence vẫn là Plan-doc → Approve → Execute.

---

## 8. Skills đã sử dụng

- **3-way verification** — Audit ↔ Source code (grep) ↔ Test output để phát hiện audit cũ sai (G-11, G-13).
- **golang-migrate alternative discovery** — Tìm `cdc-cms-service/internal/migrate/runner.go` đã có production runner thay vì add dependency mới.
- **GORM Callback API** — Đăng ký Before/After hooks cho 6 verbs, `tx.InstanceSet/InstanceGet` để pass start time.
- **testcontainers wait strategy** — `postgres.BasicWaitStrategies()` để khắc phục race condition.
- **Bash strict mode + trap** — `set -euo pipefail`, EXIT trap cleanup background k6 PID.
- **pprof diff analysis** — `go tool pprof -base first last -unit=mb` để tính heap growth giữa 2 snapshot.
- **Prometheus HistogramVec design** — Exponential buckets 1ms..4s, low-cardinality labels (role, operation).
- **Brain Code Prohibition §12** — Brain viết doc trước, Muscle execute sau.
- **Memory APPEND-only §11** — Tạo Entry 14 mới trong `05_progress.md`, không sửa Entry cũ.
- **TaskCreate/Update tracking** — 6 task end-to-end, mark completed khi DoD đạt.

---

## 9. Bước tiếp theo (đề xuất cho user)

1. **Fix pre-existing failures** (out-of-scope phase này, cần task riêng):
   - `TestSanitizeMongoDSN` 4 case fail (Entry 11 sanitization logic regression).
   - `internal/handler` kafka-go goroutine leak.
2. **Chạy `/security-agent`** — workspace vẫn chưa có `report_security_*` theo audit gốc §8.
3. **Bổ sung gap còn lại của 12 G-NEW** (G-NEW-20..23, 25..28, 30) nếu user còn quan tâm Full coverage 5 nhóm requirement.
