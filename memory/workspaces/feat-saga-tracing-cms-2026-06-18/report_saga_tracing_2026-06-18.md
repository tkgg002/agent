# Report: Saga & OTel Tracing — Post-Audit Fix Round
**Feature**: feat-saga-tracing-cms-2026-06-18  
**Session Date**: 2026-06-18 (Audit + Fix round)  
**Status**: ✅ COMPLETE — Build clean, 8/8 tests PASS  
**Previous report**: `report_saga_tracing_2026-06-18.md` (initial implementation)

---

## Tổng quan thay đổi (Toàn bộ từ đầu)

| Hạng mục | Số lượng |
|---|---|
| Files thay đổi (tổng) | 17 source + 1 new test file + 2 new source files |
| Tổng dòng | +479 / -125 |
| Unit tests | 8/8 PASS (5 behavioral + 3 OTel integration) |

---

## Fix round: 12 issues từ Audit Report

### 🔴 CRITICAL — 3/3 fixed

| Issue | Fix | File |
|---|---|---|
| **C1** — Span name `saga.run` thay vì `saga.{name}` | Đổi `"saga.run"` → `"saga."+r.name` | `saga/saga.go:58` |
| **C2** — Compensation dùng `RejectSchema` (business reject) thay vì revert | Thêm `RevertSchemaTx()` method (set `schema_status='pending_review'`, clear reviewer fields). Cập nhật compensation trong `approve_master.go` | `ports/repository.go`, `master_repo_gorm.go`, `approve_master.go` |
| **C3** — `registry_handler_update.go` không có tracing (A4 bị bỏ sót) | Thêm span `api.registry.update` bao phủ multi-op flow | `registry_handler_update.go` |

### 🟡 WARN — 5/5 fixed

| Issue | Fix | File |
|---|---|---|
| **W1** — `10_gap_analysis.md` S5 chưa update theo Q2 decision | Cập nhật S5 với HTTP-first order + design decision note | `10_gap_analysis.md` |
| **W2** — S6 `approve_ddl_executor.go` chưa có saga | Audit confirm NATS sync-blocking → thêm saga 2 steps: `nats-publish-reconcile → db-clear-pending-flags`. Compensation=nil (DDL không undo được) | `approve_ddl_executor.go` |
| **W3** — S7 `drop_column.go` NATS→DB pattern chưa có saga | Thêm saga 2 steps: `nats-publish-drop → db-update-in-master-status`. Compensation=nil (DROP đã xảy ra) | `drop_column.go` |
| **W4** — `fiberHeaderCarrier` thiếu comment lý giải | Thêm comment giải thích O(n) allocation trade-off vs spec's MapCarrier | `otel_propagator.go` |
| **W5** — Thiếu `attribute.Int("saga.steps")` trong saga span | Thêm vào `StartSpan()` call | `saga/saga.go` |

### 🔵 INFO — 4/4 fixed

| Issue | Fix | File |
|---|---|---|
| **I1** — Comment misleading TC2 "failed before executing" | Đổi thành "executed but returned error → never added to executed list" | `saga_test.go` |
| **I2** — Compensation steps không có OTel span | Thêm span `saga.compensate` per step với `RecordError` khi fail | `saga/saga.go` |
| **I3** — `masterBindingID` closure thiếu comment thread-safety | Thêm comment "sequential only, safe without mutex" | `approve_master.go` |
| **I4** — Thiếu integration test OTel propagation | Tạo `saga_otel_test.go` với 3 TCs: span names, compensation span, parent-child | `saga/saga_otel_test.go` [NEW] |

---

## Files thực tế đã thay đổi (audit round)

| File | Dòng +/- | Nội dung thay đổi |
|---|---|---|
| `internal/app/saga/saga.go` | +15/-4 | C1: span name, W5: saga.steps attr, I2: compensation span |
| `internal/app/saga/saga_test.go` | +2/-1 | I1: fix comment TC2 |
| `internal/app/saga/saga_otel_test.go` | +120/0 | I4: 3 OTel integration tests [NEW] |
| `internal/app/ports/repository.go` | +5/0 | C2: RevertSchemaTx() interface method |
| `internal/infra/persistence/master/master_repo_gorm.go` | +41/0 | C2: RevertSchemaTx() GORM implementation |
| `internal/app/commands/governance/approve_master.go` | +9/-3 | C2: dùng RevertSchemaTx, I3: thread-safety comment |
| `internal/api/source/registry_handler_update.go` | +11/-1 | C3: span api.registry.update |
| `internal/app/commands/master/approve_ddl_executor.go` | +40/-10 | W2: saga 2 steps + NewApproveDDLExecutorWithLogger |
| `internal/app/commands/master/drop_column.go` | +30/-10 | W3: saga 2 steps |
| `internal/middleware/otel_propagator.go` | +11/0 | W4: design comment |
| `agent/memory/workspaces/.../10_gap_analysis.md` | +10/-6 | W1: S5 section update |

---

## Saga Coverage Summary (Final)

| ID | Command | Saga | Steps | Compensation |
|---|---|---|---|---|
| S1 | `register_registry` | ✅ | 3 | DeleteRegistry / nil / nil |
| S2 | `approve_master` | ✅ | 2 | **RevertSchemaTx** / nil |
| S3 | `approve_schema_proposal` | ✅ | 0 (single-tx) | Auto DB rollback |
| S4 | `create_master` | ✅ | 1 | DeleteClonedRules + DeleteMasterBinding |
| S5 | `debezium_connector` | ✅ | 2+2 | HTTP.Delete / FullCleanup |
| S6 | `approve_ddl_executor` | ✅ | 2 | nil (DDL irreversible) |
| S7 | `drop_column` | ✅ | 2 | nil (DROP irreversible) |

## API Tracing Coverage (Final)

| Handler | Span | Status |
|---|---|---|
| `registry_handler_register.go` | `api.registry.register` | ✅ |
| `registry_handler_bulk.go` | `api.registry.bulk_register` | ✅ |
| `registry_handler_update.go` | `api.registry.update` | ✅ **[Fixed C3]** |
| `mapping_rule_handler_batch.go` | `api.mapping_rule.batch_update` | ✅ |
| `master_registry_handler_approve.go` | `api.master.approve` | ✅ |

---

## Verification

```
✅ go build ./... — PASS (0 errors)
✅ go test ./internal/app/saga/... -v — 8/8 PASS
   Behavioral:
   - TestRunner_AllPass
   - TestRunner_Step1Fail_NoCompensation
   - TestRunner_Step2Fail_CompensatesStep1
   - TestRunner_NilCompensate_Skipped
   - TestRunner_CompensationFail_OriginalErrPreserved
   OTel Integration:
   - TestRunner_OTel_AllPass_EmitsSpans
   - TestRunner_OTel_Failure_EmitsCompensationSpan
   - TestRunner_OTel_SpanParentChild
```
