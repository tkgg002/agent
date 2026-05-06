# Phase 2 — cdc-cms-service Refactor — Plan

> **Date**: 2026-05-05 | **Pre-req**: 01_requirements_phase2_cms_refactor.md (DoD + constraints).
> **Status**: Planning only. Code chưa chạm. Awaiting user approval per pillar.

## Tóm tắt

8 pillar độc lập (P0-P7), execute tuần tự từ low-risk → high-risk. Mỗi pillar = 1 commit, có thể revert riêng. Tổng effort ước tính: **~2-3 tuần** nếu chạy tuần tự với 1 engineer. Có thể parallel P5 (health probe split) + P6 (V2 sync atomicity) với P3-P4 vì không đụng cùng file.

| Pillar | Title | Risk | Effort | Blast radius |
|---|---|---|---|---|
| P0 | Dead-code prune | LOW | 2h | -1 service struct, -50 line server.go |
| P1 | V2 repository layer | MEDIUM | 1.5d | +6 repo file, refactor 6 handler read paths |
| P2 | Service layer extraction (godfile split) | HIGH | 3-4d | Refactor 4 godfile (894+693+689+667 dòng) |
| P3 | ActivityLog helper dedup | LOW | 4h | Replace 8+ inline write call site |
| P4 | V1↔V2 surface dedup | MEDIUM | 1d | Eliminate `h.registry.X` từ V2 handler |
| P5 | Health collector probe split | MEDIUM | 1d | Split 781-line collector → 7 probe file |
| P6 | V2 sync atomicity (TX hoặc outbox) | HIGH | 1.5d | Touch register/update flow |
| P7 | Test coverage uplift | MEDIUM | 2d | +20-30 test file |

---

## Current State Snapshot (live 2026-05-05)

- Process: `/tmp/cms-server` PID 18563 listening :8083 — 5 days uptime per project_context (đã restart trong session prep).
- Service health: `/health` 200 + `/ready` 200 ✅
- File count: 76 .go file, 10 _test.go file, 13% coverage.
- 4 godfile pain: `reconciliation_handler.go` (894), `source_object_actions_handler.go` (693), `mapping_rule_handler.go` (689), `master_registry_handler.go` (667).
- Single Postgres instance, dual-namespace: `public.cdc_*` (V1 legacy) + `cdc_system.*` (V2).

---

## P0 — Dead-code prune

**Why**: `ReconciliationService` đã retire phase F (Airbyte→Debezium) nhưng vẫn:
- Wire vào `server.go:85` (struct alloc)
- Goroutine launch `server.go:187` (`<-ctx.Done()` no-op)
- Hold `RegistryRepo` + `MappingRuleRepo` reference (mislead reader)

**Files**:
- DELETE `internal/service/reconciliation_service.go`
- EDIT `internal/server/server.go` — remove field + alloc + Start goroutine

**Approach**:
1. Verify NO callsite ngoài `server.go`: `grep -r "ReconciliationService\|reconSvc" --include='*.go'`
2. Audit `internal/model/cdc_event.go` — model dùng cho deserialize CDC event ở worker, CMS không ingest event → likely dead. Verify grep.
3. Remove cả 2 nếu confirmed dead.

**Verification**: `go build ./...` PASS + 8 endpoint smoke PASS.

**Risk**: LOW. Subtractive, no behavioral change.

**Lesson invoked**: #160 Simplicity First (don't keep retired code "just in case").

---

## P1 — V2 repository layer

**Why**: Mọi truy cập `cdc_system.*` (V2 schema) hiện qua `h.db.Raw(...)` trực tiếp trong handler. Hậu quả:
- Cannot unit-test handler (cần real DB).
- SQL string lặp lại nhiều file (vd `resolveDispatchScopeBySourceObjectID` trong source_object_actions_handler.go gọi 6 lần).
- Schema drift risk (Lesson #1310 — model↔DB drift): không có 1 chỗ thấy hết SQL.

**Files mới** (under `internal/repository/`):
- `v2_source_object_repo.go` — `SourceObjectRegistryRepo` (source_object_registry + shadow_binding)
- `v2_mapping_rule_repo.go` — `MappingRuleV2Repo` (mapping_rule_v2)
- `v2_master_binding_repo.go` — `MasterBindingRepo`
- `v2_connection_registry_repo.go` — `ConnectionRegistryRepo`
- `v2_wizard_session_repo.go` — `WizardSessionRepo`
- `v2_alert_repo.go` — `AlertRepo` (cdc_alerts)
- `v2_admin_action_repo.go` — `AdminActionRepo` (cdc_system.admin_actions, audit log)

**Approach**:
1. Mỗi repo: define interface với method theo verb thực tế đang dùng (List/Get/Upsert/UpdateStatus/...).
2. Implementation: di chuyển SQL nguyên xi từ handler → method, return typed result struct (không nested struct, theo Lesson #1253 — flat scan).
3. Wire: `server.go` instantiate repo, inject vào handler constructor.
4. Refactor handler: replace `h.db.Raw(...).Scan(&dst)` → `h.repo.MethodName(ctx, params)`.
5. Mock interface cho test.

**Order**: làm 1 repo + 1 handler refactor demo trước (e.g. `MappingRuleV2Repo` + `mapping_rule_handler.go`) để verify pattern, sau mới fan-out 6 repo còn lại.

**Verification**: `go build` PASS + endpoint smoke 4-5 (source-objects, mapping-rules) + repo unit test (sqlmock).

**Risk**: MEDIUM. Touch nhiều handler file nhưng diff cơ học.

**Lesson invoked**: #1253 (flat scan struct, không nested), #1310 (schema drift detection).

---

## P2 — Service layer extraction (godfile split)

**Why**: 4 godfile handler 600-900 dòng trộn responsibilities (parse request + business logic + SQL + NATS publish + activity log). Vi phạm separation of concerns. Test impossible.

**Files affected**:

### P2.1 — `reconciliation_handler.go` (894 dòng)
Extract sang `internal/service/reconciliation/`:
- `drift_calculator.go` — `ComputeDriftStatus(srcCount, dstCount, recon)` (line 34-73)
- `recon_dispatcher.go` — NATS dispatch helper cho `cdc.cmd.recon-check`, `cdc.cmd.recon-heal`, `cdc.cmd.retry-failed`, `cdc.cmd.debezium-signal`, `cdc.cmd.recon-backfill-source-ts`, `cdc.cmd.debezium-snapshot`
- `failed_log_query.go` — pagination + scope-enrich query (line 538-624)
- `report_query.go` — report pagination + drift enrich (line 175-450)
- `pgident.go` (utility) — Postgres identifier sanitizer (line 873-886) → move to `pkgs/utils/pg_ident.go`

Handler còn lại ≤180 dòng: parse request → call service → return.

### P2.2 — `source_object_actions_handler.go` (693 dòng)
Extract sang `internal/service/source_object/`:
- `dispatch_service.go` — generic dispatch cho 7 NATS subject (`scan-fields`, `standardize`, `create-default-columns`, `detect-timestamp-field`, ...). Method nhận `(ctx, subject, scope, payload)` — handler chỉ pass through.
- `scope_resolver.go` — `ResolveDispatchScopeBySourceObjectID` (line 39-73, đang gọi 6 lần) + `ResolveByRegistryID` (V1 bridge)
- 12+ V2 + V1-bridge action method gọi service uniformly.

Handler còn lại ≤200 dòng.

### P2.3 — `mapping_rule_handler.go` (689 dòng)
Extract sang `internal/service/mapping/`:
- `mapping_rule_service.go` — `Create`, `UpdateStatus`, `BatchUpdate`, `Backfill`
- `scope_resolver.go` — `ResolveScope` (line 105-179)
- `list_query.go` — multi-join read query (line 244-357)
- `batch_dispatcher.go` — `BatchUpdate` per-rule NATS publish (giải quyết N+1 pattern line 615-689) → bulk publish

Handler còn lại ≤180 dòng.

### P2.4 — `master_registry_handler.go` (667 dòng)
Extract sang `internal/service/master_registry/`:
- `master_service.go` — `Create`, `Approve`, `Reject`, `ToggleActive`, `Swap`
- `swap_orchestrator.go` — atomic RENAME TX (đã có `MasterSwap` service — verify reuse)

Handler còn lại ≤150 dòng.

**Approach**: pillar-by-pillar (P2.1 → P2.2 → P2.3 → P2.4), commit từng cái. Mỗi sub-pillar:
1. Extract logic → service file mới (paste-as-is, đặt method trên struct mới).
2. Service depend vào repo (P1) thay vì `*gorm.DB`.
3. Refactor handler → call service.
4. Build PASS + smoke endpoint cụ thể.
5. Add 1 test cho service (ít nhất golden path).

**Verification**: `wc -l internal/api/*.go` mọi file ≤200 + 8 smoke endpoint PASS.

**Risk**: HIGH. Lớn nhất, nhiều file, nhiều logic. Nhưng diff mỗi sub-pillar độc lập commit → revert được.

**Lesson invoked**: #258 No Cross-Domain Model in Handler, #475 Forgotten Field Assignment (paranoid review patch handler), #160 Simplicity First (extract only what's tangled, don't refactor stable utility helpers).

---

## P3 — ActivityLog helper dedup

**Why**: `model.ActivityLog{...}` được khởi tạo + GORM Create inline 8+ chỗ trong `registry_handler.go` (line 50-61, 253-264, ...) với pattern gần giống nhau (operation, target_table, status, details, triggered_by). Copy-paste = drift waiting to happen.

**Approach**:
1. Tạo `internal/service/activity_logger.go`:
   ```go
   type ActivityLogger struct { db *gorm.DB; logger *zap.Logger }
   func (a *ActivityLogger) Log(ctx, ActivityLogEvent) error
   func (a *ActivityLogger) LogAsync(ctx, ActivityLogEvent)  // fire-and-forget, drain qua chan
   type ActivityLogEvent struct { Operation, TargetTable, Status, Details, TriggeredBy string; Scope V2Scope }
   ```
2. Replace 8+ inline call site → `h.activityLogger.Log(c.Context(), event)`.
3. Wire `ActivityLogger` ở `server.go`, inject vào handler.

**Verification**: `grep "model.ActivityLog{" internal/api/*.go` = 0 + 8 smoke endpoint PASS.

**Risk**: LOW. Mechanical replace.

**Lesson invoked**: DRY. Cũng giảm bug-surface cho Lesson #475 (forgotten field) — 1 chỗ khai báo struct = 1 chỗ review.

---

## P4 — V1↔V2 surface dedup

**Why**: `RegistryHandler` (V1) marked "compatibility delegate" trong comment nhưng `SourceObjectActionsHandler` (V2) vẫn gọi `h.registry.Standardize(c)`, `h.registry.ScanFields(c)`, etc. để tiết kiệm code. Hậu quả: V1 path = production code path qua V2 → V1 không thể deprecate.

**Approach**:
1. Ý nghĩa: cả V1 RegistryHandler và V2 SourceObjectActionsHandler PHẢI gọi cùng `service.SourceObjectDispatchService` (đã tạo P2.2). V1 handler nhận `registry_id`, resolve qua `ResolveByRegistryID`; V2 handler nhận `source_object_id`, resolve qua `ResolveDispatchScopeBySourceObjectID`. Cả 2 sau đó gọi cùng `dispatch.Standardize(scope)`.
2. Verify scope: kiểm tra phase 27-32 cms-fe-overhaul đã chuyển V2 sang direct-write path chưa. Nếu rồi, P4 scope giảm — chỉ còn dedup helper, không cần đụng V1 handler.

**Verification**: `grep "h.registry\." source_object_actions_handler.go` = 0.

**Risk**: MEDIUM. Phụ thuộc P2.2 done.

---

## P5 — Health collector probe split

**Why**: `system_health_collector.go` 781 dòng — 7 probe + 3 DB query + alert + serialize. Probe pattern: `func probeX(ctx) (status, latency, err)` lặp 7 lần, không thể unit-test 1 probe riêng.

**Approach**:
1. Tạo `internal/service/health/probes/` với 7 file:
   - `worker.go` — `ProbeWorker(ctx, url, client) (Result, error)`
   - `kafka_connect.go` — `ProbeKafkaConnect`
   - `debezium.go` — `ProbeDebezium`
   - `kafka_lag.go` — `ProbeKafkaLag`
   - `nats.go` — `ProbeNATS`
   - `postgres.go` — `ProbePostgres`
   - `redis.go` — `ProbeRedis`
2. Each probe = pure function (no DI mess) — chỉ nhận URL + http.Client → struct kết quả.
3. Collector orchestrate qua `errgroup.Group`, fan-out parallel (cải thiện latency: 7 probe sequential → parallel).
4. Alert eval (`computeAlerts`, `computeOverall`) tách `internal/service/health/alerts/evaluator.go`.
5. Serialize JSON snapshot → `internal/service/health/snapshot.go`.

**Verification**: `wc -l internal/service/system_health_collector.go` ≤300 + `/api/system/health` smoke PASS + cache key `system_health:snapshot` populated trong Redis.

**Risk**: MEDIUM. Concurrent fan-out introduce timing change (latency giảm, nhưng cache snapshot semantics giữ nguyên).

**Lesson invoked**: Single Responsibility. Cũng prepare cho test fan-out của Pillar 7.

---

## P6 — V2 sync atomicity

**Why**: `SourceObjectV2SyncService.SyncFromLegacy` được gọi sau `repo.Create(&entry)` ngoài TX:
```go
if err := h.repo.Create(&entry); err != nil { ... }  // V1 commit
if err := h.v2sync.SyncFromLegacy(ctx, &entry); err != nil {  // V2 outside TX
    h.logger.Error("post-register v2 sync failed", ...)  // SILENT SWALLOW
}
```
Hậu quả: V1 success + V2 fail = silent diverge. Không alerting, không retry.

**Approach** (pick one, pillar split decision):
- **Option A — Same TX** (preferred):
  ```go
  err := h.db.Transaction(func(tx *gorm.DB) error {
      if err := tx.Create(&entry).Error; err != nil { return err }
      if err := h.v2sync.SyncFromLegacyTx(tx, &entry); err != nil { return err }
      return nil
  })
  ```
  V1+V2 atomic. Rollback nếu V2 fail.

- **Option B — Outbox pattern**:
  - Handler ghi V1 + outbox row trong cùng TX.
  - Background drainer (`internal/service/outbox_drainer.go`) tick mỗi 5s, đọc outbox pending → call V2 sync → mark done.
  - Idempotent qua `outbox.id`.
  - Phù hợp nếu V2 sync chậm hoặc remote (sau này V2 service tách ra microservice).

**Recommended**: Option A. Đơn giản, đủ cho hiện tại single-DB. Outbox khi V2 thành remote service.

**Verification**: Test path register source object với V2 sync force-fail (mock SyncFromLegacy return err) → assert V1 row KHÔNG commit (rollback) + V2 row KHÔNG insert.

**Risk**: HIGH. Touch register flow — entry point quan trọng cho operator wizard.

---

## P7 — Test coverage uplift

**Why**: Sau P0-P6, code đã layered → test-able. Hiện 13% coverage. Target ≥35% trên `internal/service/` + `internal/repository/`.

**Approach**:
1. Repo tests qua `sqlmock` hoặc `dockertest` (live PG container test ổn hơn).
2. Service tests: mock repo + nats client + redis. Test golden path + 2-3 edge case mỗi service.
3. Handler tests: existing pattern (`fiber_test.go` infra) — mock service.
4. CI gate: `go test -cover` threshold 35% cho 2 directory trên.

**Verification**: `go test -cover ./internal/service/... ./internal/repository/...` ≥ 35% combined.

**Risk**: LOW. Pure additive (test code).

---

## Execution methodology

Per CLAUDE.md §4 (Deep Execution): mỗi pillar invoke `/refactor-coordinator` (nếu workflow tồn tại) hoặc manual coordinate via subject contract:
- P0: simple subtractive — direct execute.
- P1: invoke `/refactor-coordinator` để parallelize 6 repo skeleton.
- P2: invoke `/debug-agent` nếu smoke fail; `/qa-agent` cho regression sweep.
- P3-P6: similar pattern.
- Mỗi pillar end: `/security-agent` BẮT BUỘC (CLAUDE.md §8).

## Rollback strategy

Mỗi pillar = 1 commit. Revert = `git revert <pillar-commit>`. Vì:
- API contract giữ nguyên → FE không cần redeploy.
- DB schema giữ nguyên → migration không cần rollback.
- NATS subject giữ nguyên → worker không bị ảnh hưởng.
- Service alive guarantee qua endpoint smoke POST-revert.

## Decision log

- **Why không tạo workspace mới** (e.g. `feature-cdc-cms-refactor-2026-05`)? → Phase 1 của `feature-cdc-system-refactor` đã refactor centralized-data-service (sister service); Phase 2 = sister cdc-cms-service là continuation tự nhiên cùng đề tài "system refactor". Tránh fragment workspace.
- **Why không nhập vào `feature-cms-fe-overhaul`**? → 39 phase đó scope = FE-driven contract changes (path, response shape). P2 đây là architectural refactor backend-internal, không touch contract → không phù hợp với theme.
- **Why không bắt đầu từ P2 (godfile split) — biggest pain**? → Risk highest. Cần P0+P1 chuẩn bị (repo abstraction) trước, để service layer có dep target sạch.
- **Why P3 sau P2 mà không trước**? → ActivityLog helper sẽ được dùng trong service (P2). Nếu làm P3 trước, helper vẫn ở handler scope, sau P2 lại phải refactor lần 2.
- **Why không split P2 thành 4 pillar riêng**? → Có thể, P2.1-P2.4 là sub-pillar độc lập commit. Plan giữ chung umbrella P2 vì cùng pattern (extract handler→service).
