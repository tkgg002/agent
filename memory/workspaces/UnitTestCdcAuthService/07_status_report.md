# 07_status_report — UnitTestCdcAuthService

> **Status**: ✅ DONE — all tests green, build clean, vet clean.
> **Date**: 2026-05-14 09:46 ICT (baseline) · 2026-05-15 ICT (test folder reorg — coverage parity preserved)
> **Owner**: Muscle (CC CLI, claude-opus-4-7)
>
> **Update 2026-05-15**: Toàn bộ test đã move sang `cdc-auth-service/test/` mirror folder với external `*_test` packages (ADR-005). Coverage parity 100% với baseline bên dưới. Chi tiết: `report_test_folder_reorg_20260515.md`.

## Summary
Bổ sung bộ unit test cho `cdc-auth-service`. Trước đây service có 0 test (lesson `project_context.md` đánh dấu là HIGH gap của bucket B1). Sau task này: 49 sub-tests pass, coverage non-DB-layer 81.8% — 100%.

## Test results — actual, from `go test ./... -count=1 -cover`
| Package | Status | Coverage | Số test parent / sub |
|---|---|---|---|
| `cdc-auth-service/config` | PASS | **81.8%** | 5 / 14 |
| `cdc-auth-service/internal/api` | PASS | **100.0%** | 4 / 16 |
| `cdc-auth-service/internal/model` | PASS | **100.0%** | 1 / 1 |
| `cdc-auth-service/internal/service` | PASS | **90.7%** | 4 / 18 |
| `cdc-auth-service/internal/repository` | no test files | 0.0% (intentional, L-2305) | — |
| `cdc-auth-service/internal/server` | no test files | 0.0% (intentional) | — |
| `cdc-auth-service/pkgs/database` | no test files | 0.0% (intentional) | — |
| `cdc-auth-service/cmd/server` | no test files | 0.0% (intentional) | — |
| `cdc-auth-service/docs` | no test files | 0.0% (generated, intentional) | — |

**Verification commands executed**:
- `go test ./... -count=1` → all PASS (no FAIL lines).
- `go test ./... -count=1 -cover` → coverage report above.
- `go build ./...` → 0 errors.
- `go vet ./...` → 0 errors.

## Convention compliance
- ✅ No sqlmock added (lesson L-2258 project convention).
- ✅ No testcontainers added.
- ✅ Repository skipped per lesson L-2305 (adapter layer ≠ unit test target via mock).
- ✅ Service tested via interface stub (`UserRepository`) — fake handcrafted.
- ✅ Handler tested via interface stub (`AuthSvc`) + fiber's `app.Test()`.
- ✅ Config: pure-fn `validateConfig` + table-driven cases.
