# 03_implementation_remaining_gaps — Khắc phục Remaining Gaps (Revised)

**Date:** 2026-05-27
**Phase:** Remaining (sau Phase P0/P1/P2/UI đã execute)
**Status:** Brain plan → Muscle execute

---

## 0. Audit Correction (so với `report_audit_workspace_2026-05-27.md`)

Re-verify code thực tế phát hiện audit cũ có **2 điểm sai**:

| Gap | Audit cũ claim | Thực tế (re-verify) | Action |
|---|---|---|---|
| **G-11** BatchesFlushed | `NewCounter` không label | `NewCounterVec [sink, table]` (prometheus.go:167-173); caller `WithLabelValues("postgres", tableName).Inc()` (batch_buffer.go:191) | ✅ KHÔNG cần làm — drop khỏi plan |
| **G-13** PerSourcePool | DEAD CODE | Đã wire: `kafka_consumer.go:96,165,501` (constructor + apply); `worker_server.go:608` (instantiate); semaphore + metric chạy | ✅ KHÔNG cần làm — drop khỏi plan |

**Lesson**: audit phải double-check grep callers, không chỉ đọc 1 file metric.

---

## 1. Scope (4 gap còn lại — đã user approve)

| Gap | Mô tả | File chính | Effort |
|---|---|---|---|
| **G-9 (refactor)** | E2E test load real SQL migrations qua `migrate.Run()` thay raw CREATE TABLE | `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` | 1h |
| **G-NEW-19** | Delete event ordering test (insert→delete, delete→insert resurrection, OCC drop older update) | `centralized-data-service/internal/service/schema_adapter_ordering_test.go` | 1.5h |
| **G-NEW-24** | Source DB overhead metric: histogram `cdc_source_query_duration_seconds` + GORM callback per-role | `pkgs/metrics/prometheus.go`, NEW `pkgs/database/metrics_callback.go`, `multi.go` | 2h |
| **G-NEW-29** | Soak orchestrator script (k6 + pprof heap snapshot + diff) + runbook | NEW `scripts/soak_test.sh`, NEW `docs/runbooks/soak-test.md` | 2h |

**Total**: ~6.5h muscle work.

---

## 2. Detail per Gap

### G-9 — E2E test load real SQL migrations

**Hiện trạng** (file `approve_schema_proposal_e2e_test.go:28-81`): hàm `runMigrations(t, db)` chạy inline `CREATE TABLE cdc_system.schema_proposal (...)` raw SQL, không match production schema.

**Discovery quan trọng**: `cdc-cms-service` đã có production migration runner: `internal/migrate/runner.go:33` — `func Run(gdb, includeSeeds, skipCluster, logger)` đọc embed FS `migrations.SchemaFiles` (`//go:embed schema/*/*.sql`).

**Plan**:
1. Import `cdc-cms-service/internal/migrate`.
2. Trong `TestApproveSchemaProposal_E2E`, sau khi `connectGORM`:
   ```go
   require.NoError(t, migrate.Run(db, false, true, zap.NewNop()))
   ```
   - `includeSeeds=false`: skip seed files (giữ test deterministic).
   - `skipCluster=true`: tránh cluster bootstrap (PG container đã bootstrap).
3. Sau đó manual create `cdc_internal.shadow_users` (ShadowAutomator tạo runtime trong production; test simulate).
4. Xoá hàm `runMigrations()` cũ.

**DoD G-9**:
- Test PASS với `go test -tags=integration ./internal/app/commands/ -run TestApproveSchemaProposal_E2E -count=1`.
- Test sử dụng schema giống hệt production (migration tracker + advisory lock).
- File `runMigrations()` đã bị xoá; không còn `CREATE TABLE` inline.

---

### G-NEW-19 — Delete event ordering test

**Hiện trạng**: `schema_adapter_ordering_test.go` chỉ test Insert + Update OCC, **không có Delete event**. Sink worker (`sinkworker.go:123`) xử lý op="d" bằng cách set `_gpay_deleted=true` (soft delete pattern, không DROP row).

**Plan** — bổ sung 3 test case:
1. `TestEventOrdering_DeleteEventTombstone`:
   - Insert ts=1000 → row exists, `_gpay_deleted=false`
   - Delete ts=2000 (mappedData `{_gpay_deleted: true}`) → row vẫn tồn tại nhưng cờ `_gpay_deleted=true`
2. `TestEventOrdering_InsertAfterDelete_Resurrection`:
   - Insert ts=1000 → `_gpay_deleted=false`
   - Delete ts=2000 → `_gpay_deleted=true`
   - Insert ts=3000 (newer) → `_gpay_deleted=false` (row sống lại)
3. `TestEventOrdering_UpdateAfterDelete_OCCDrop`:
   - Insert ts=1000
   - Delete ts=3000 → `_gpay_deleted=true`
   - Update ts=2000 (older than delete) → KHÔNG cập nhật (OCC drop), `_gpay_deleted` vẫn true.

**Implementation note**: schema test cần thêm cột `_gpay_deleted BOOLEAN`; mappedData truyền `{"_gpay_deleted": true/false}` thông qua `BuildUpsertSQL`.

**DoD G-NEW-19**:
- 3 test mới PASS.
- Existing tests `TestEventOrdering_OlderTsIgnored`, `TestEventOrdering_HashTiebreaker` vẫn PASS (no regression).

---

### G-NEW-24 — Source DB overhead metric

**Hiện trạng**: KHÔNG có metric đo độ trễ query từ CDC service tới Source DB. User requirement: "tránh overload mà không hay biết".

**Plan**:
1. Add metric vào `pkgs/metrics/prometheus.go`:
   ```go
   SourceQueryDuration = promauto.NewHistogramVec(
       prometheus.HistogramOpts{
           Name:    "cdc_source_query_duration_seconds",
           Help:    "Duration of queries against source DBs by role and operation",
           Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms .. ~4s
       },
       []string{"role", "operation"},
   )
   ```
2. Tạo NEW file `pkgs/database/metrics_callback.go`:
   - Func `RegisterQueryMetrics(db *gorm.DB, role string) error`
   - Đăng ký GORM callbacks `Before/After` cho `query`, `create`, `update`, `delete`
   - Trong Before: set `start_time` vào `db.InstanceSet`
   - Trong After: get start_time, compute elapsed, gọi `metrics.SourceQueryDuration.WithLabelValues(role, op).Observe(elapsed.Seconds())`
3. Edit `pkgs/database/multi.go`:
   - Đổi `openGorm(dsn)` → `openGorm(dsn, role string)`
   - Sau `gorm.Open`, gọi `RegisterQueryMetrics(db, role)`
   - Update callsite line 89: `r.openGorm(dsn, role)`
4. Verify `deployments/prometheus/prometheus.yml` — đã có job `postgres-exporter` (line 41-44) và `mongodb-exporter` (line 47-49). Append annotation comment làm rõ purpose.

**DoD G-NEW-24**:
- Build PASS `go build ./...`.
- Run `centralized-data-service` local, query metric endpoint → thấy `cdc_source_query_duration_seconds{role,operation}` xuất hiện.
- Histogram có data sau khi gọi 1 GORM query.
- Unit test: thêm `pkgs/database/metrics_callback_test.go` verify callback chạy đúng (sqlite in-memory).

---

### G-NEW-29 — Soak orchestrator script + runbook

**Hiện trạng**: Không có script soak. `load_test.js` chỉ chạy 9 phút (1m+5m+2m+1m).

**Plan**:
1. NEW `scripts/soak_test.sh`:
   - Env vars: `SOAK_DURATION_HOURS` (default 48), `K6_TARGET_URL`, `PPROF_URL` (default `http://localhost:6060/debug/pprof/heap`), `SNAPSHOT_INTERVAL_MIN` (default 30), `OUT_DIR` (default `./soak_artifacts`).
   - Pre-flight: kiểm tra `k6`, `go tool pprof`, `curl` có sẵn.
   - Start `k6 run --duration ${SOAK_DURATION_HOURS}h scripts/load_test.js` (background, log → `$OUT_DIR/k6.log`).
   - Loop: mỗi `SNAPSHOT_INTERVAL_MIN` phút, `curl -s $PPROF_URL > $OUT_DIR/heap_$(date +%s).pb.gz` và `go tool pprof -top -unit=mb $heap_file > $OUT_DIR/heap_top_$timestamp.txt`.
   - End: diff first/last snapshot, output `$OUT_DIR/heap_diff.txt`, exit code = 1 nếu heap growth > threshold (default 50MB).
   - Dry-run mode: `--dry-run` chỉ chạy 5 phút (smoke validation script syntax).
2. NEW `docs/runbooks/soak-test.md`:
   - Pre-req: staging cluster, pprof endpoint exposed, k6 installed.
   - Quy trình quarterly: chạy 48-72h, kết quả nộp về `report_soak_QYYYY_QN.md`.
   - Acceptance criteria: heap growth < 50MB, no panic in k6 log, p99 latency < 5s end-of-test.
   - CI chỉ smoke 5min: `bash scripts/soak_test.sh --dry-run` để verify syntax.

**DoD G-NEW-29**:
- Script chạy `--dry-run` PASS (5 phút, không error syntax).
- Runbook có acceptance criteria rõ ràng.
- KHÔNG claim CI soak 48-72h (false claim) — chỉ smoke 5min.

---

## 3. Sequencing

```
G-NEW-19 (test only, isolated)  ─┐
G-9 (test refactor, isolated)   ─┼─►  Build + go vet ./...
G-NEW-24 (metric + callback)    ─┘     │
                                       ▼
G-NEW-29 (script, no Go impact) ──►  Smoke dry-run 5min
                                       │
                                       ▼
                                  Doc + report
```

Parallel-safe vì 4 gap touching disjoint files (trừ prometheus.go nhưng chỉ append). Execute tuần tự theo task list để dễ verify.

---

## 4. Verification Plan

```bash
# Phase 1 — unit tests
cd centralized-data-service
go vet ./... && go build ./...
go test ./internal/service/ -run TestEventOrdering -count=1
go test ./pkgs/database/ -run TestRegisterQueryMetrics -count=1

# Phase 2 — integration test
cd ../cdc-cms-service
go test -tags=integration ./internal/app/commands/ -run TestApproveSchemaProposal_E2E -count=1

# Phase 3 — soak smoke
cd ../centralized-data-service
bash scripts/soak_test.sh --dry-run
```

---

## 5. Files Touched (predicted)

**Edits**:
1. `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` — G-9
2. `centralized-data-service/internal/service/schema_adapter_ordering_test.go` — G-NEW-19
3. `centralized-data-service/pkgs/metrics/prometheus.go` — G-NEW-24 (append)
4. `centralized-data-service/pkgs/database/multi.go` — G-NEW-24 (openGorm signature)

**New files**:
1. `centralized-data-service/pkgs/database/metrics_callback.go` — G-NEW-24
2. `centralized-data-service/pkgs/database/metrics_callback_test.go` — G-NEW-24 unit
3. `centralized-data-service/scripts/soak_test.sh` — G-NEW-29
4. `centralized-data-service/docs/runbooks/soak-test.md` — G-NEW-29

**Estimated total**: 4 edit + 4 new = 8 files touched.

---

## 6. Risk + Mitigation

| Risk | Mitigation |
|---|---|
| `migrate.Run()` cần env permissions (CREATE SCHEMA) | testcontainers postgres mặc định superuser → OK |
| GORM callback overhead làm chậm hot path | Histogram observe ~ns scale, negligible vs ms query latency |
| openGorm signature change breaks other callers | grep confirm chỉ 1 callsite tại line 89 |
| Soak script chạy CI thực tế 48h | KHÔNG đưa vào CI; CI chỉ smoke 5min |

---

## 7. Brain Code Prohibition §12

Brain (Antigravity) viết doc này; Muscle (CC CLI) sẽ:
- Edit 4 file + tạo 4 file mới (8 files Go/SH/MD).
- Build + test verify.
- Append `05_progress.md` Entry 11.
- Tạo `report_execute_remaining_gaps_2026-05-27.md`.
