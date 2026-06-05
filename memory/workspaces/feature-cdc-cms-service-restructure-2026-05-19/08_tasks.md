# 08_tasks.md — Task break-down chi tiết (Muscle execution)

> 11 phase × tasks chi tiết. Mỗi task: input, action, output, verify, rollback.
> **Brain KHÔNG chạy.** Muscle thực thi sau khi user approve ADR 1-8 + plan.

## Phase 0 — Foundation (0.5 ngày)

| # | Task | Input | Output | Verify |
|---|------|-------|--------|--------|
| 0.1 | `mkdir -p internal/modules` | - | folder rỗng | `ls internal/modules` ok |
| 0.2 | `mkdir -p internal/platform/{bus,db,http,observability,middleware,eventbus}` | - | 6 folder | `ls -d internal/platform/*` ok |
| 0.3 | `mkdir -p internal/server` | - | folder | (Khi tạo `wire.go` ở P1) |
| 0.4 | Tạo `.golangci.yml` rule depguard cấm cross-module import | rules content | file commit | `golangci-lint run` no error |
| 0.5 | Commit `chore: scaffold modular structure (P0)` | - | git tag P0 | `git log -1` |

**Rollback**: `git revert P0`.

---

## Phase 1 — Move platform layer (0.5 ngày)

| # | Task | Cmd |
|---|------|-----|
| 1.1 | Move messaging | `git mv internal/infra/messaging internal/platform/bus` |
| 1.2 | Move observability | `git mv internal/infra/observability internal/platform/observability` |
| 1.3 | Move middleware | `git mv internal/middleware internal/platform/middleware` |
| 1.4 | Move infra/http | `git mv internal/infra/http internal/platform/http` |
| 1.5 | Delete cache (chỉ doc.go) | `rm -rf internal/infra/cache` |
| 1.6 | Bulk replace import: `internal/infra/messaging` → `internal/platform/bus` | `grep -rl "internal/infra/messaging" --include="*.go" \| xargs sed -i ''` (macOS) |
| 1.7 | Bulk replace: `internal/infra/observability` → `internal/platform/observability` | sed |
| 1.8 | Bulk replace: `internal/middleware` → `internal/platform/middleware` | sed |
| 1.9 | Bulk replace: `internal/infra/http` → `internal/platform/http` | sed |
| 1.10 | `go build ./...` | - |
| 1.11 | `go vet ./...` | - |
| 1.12 | `go test ./...` | - |
| 1.13 | Commit `refactor(P1): move plumbing to internal/platform` | - |

**Verify**:
- `ls internal/infra/` → expect `persistence/` only (chưa move).
- `go test ./...` count = baseline.

**Rollback**: `git revert P1`.

---

## Phase 2 — Pilot module `health/` (0.5 ngày) — REVIEW GATE

| # | Task |
|---|------|
| 2.1 | `mkdir internal/modules/health` |
| 2.2 | `git mv internal/api/health_handler.go internal/modules/health/handler.go` |
| 2.3 | `git mv internal/api/system_health_handler.go internal/modules/health/system.go` |
| 2.4 | Đổi package declaration `package api` → `package health` trong 2 file |
| 2.5 | Tạo `internal/modules/health/module.go` (constructor `New`) |
| 2.6 | Tạo `internal/modules/health/routes.go` mount `/health`, `/ready`, `/api/system/health` |
| 2.7 | Sửa `internal/router/router.go` xóa block health route, gọi `healthMod.RegisterRoutes(app)` |
| 2.8 | Sửa `internal/bootstrap/` hoặc tạo `internal/server/wire.go` wire `healthMod` |
| 2.9 | `go build ./...` + `go vet ./...` + `go test ./...` |
| 2.10 | Start server local + smoke test |
| 2.11 | Commit `refactor(P2): pilot module health/` |
| 2.12 | **🛑 STOP — User review pattern + approve trước P3** |

**Smoke test** (bash):
```bash
./cms-server &
sleep 3
curl -fsS localhost:8081/health
curl -fsS localhost:8081/ready
curl -fsS localhost:8081/api/system/health
pkill -f cms-server
```

**Rollback**: `git revert P2` → các module sau pending.

---

## Phase 3 — Module `mapping/` (1 ngày)

| # | Task |
|---|------|
| 3.1 | `mkdir internal/modules/mapping` |
| 3.2 | `git mv internal/domain/mapping/rule.go internal/modules/mapping/domain.go` |
| 3.3 | `git mv internal/domain/mapping/errors.go` → append vào `domain.go` (rm file cũ) |
| 3.4 | `git mv internal/api/dto/mapping_rule_dto.go internal/modules/mapping/dto.go` |
| 3.5 | `git mv internal/api/mapping_rule_handler*.go` (5 file) → `internal/modules/mapping/handler.go` (merge) |
| 3.6 | `git mv internal/app/commands/create_mapping_rule.go` + `update_mapping_rule.go` + `delete_mapping_rule.go` → `internal/modules/mapping/commands.go` (merge) |
| 3.7 | `git mv internal/app/queries/list_mapping_rules.go` + `get_mapping_rule.go` → `internal/modules/mapping/queries.go` (merge) |
| 3.8 | `git mv internal/infra/persistence/mapping_rule_repo_gorm.go internal/modules/mapping/repo.go` |
| 3.9 | Thêm method `Validate()`, `CanTransitionTo()`, `Apply()` cho `Rule` (thicken domain — xem `03_implementation.md`) |
| 3.10 | Tạo `module.go` + `routes.go` theo template |
| 3.11 | Đổi package `api` → `mapping`, `commands` → `mapping`, `queries` → `mapping`, `persistence` → `mapping` trong các file đã move |
| 3.12 | Wire `mappingMod` trong `server/wire.go` |
| 3.13 | Sửa `router.go` xóa block mapping route |
| 3.14 | Build + test + smoke |
| 3.15 | Commit `refactor(P3): module mapping/` |
| 3.16 | **🛑 STOP — User review pattern lock** |

**Smoke test**:
```bash
TOKEN=$(jwt-helper)
curl -fsS -H "Authorization: Bearer $TOKEN" \
     localhost:8081/api/v1/mapping-rules?limit=5
curl -fsS -X POST -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"source_field":"name","target_column":"name","data_type":"text","source_table":"users"}' \
     localhost:8081/api/v1/mapping-rules
```

**Rollback**: `git revert P3`.

---

## Phase 4 — Module `registry/` (1 ngày)

| # | Task | Note |
|---|------|------|
| 4.1 | `mkdir internal/modules/registry` | |
| 4.2 | `git mv` 9 file `internal/api/registry_handler*.go` → `internal/modules/registry/handler.go` (merge) | Giữ comment swagger |
| 4.3 | **Tách business "register → restart debezium"** từ `registry_handler_register.go:82-87` sang `internal/modules/registry/commands.go` (method `Register` tự dispatch internal) | FR-9 |
| 4.4 | **Tách raw NATS publish** `registry_handler_transform.go:17` → dùng `bus.Dispatch()` | FR-8 |
| 4.5 | `git mv internal/app/commands/register_registry.go` + `v2_sync.go` → `commands.go` | |
| 4.6 | `git mv internal/domain/source/object.go internal/modules/registry/domain.go` | |
| 4.7 | `git mv internal/model/table_registry.go` → unexported `tableRegistryRow` trong `repo.go` | |
| 4.8 | `git mv internal/infra/persistence/source_object_*.go` → `repo.go` (merge) | |
| 4.9 | Wire + routes + build + test + smoke | |
| 4.10 | Commit `refactor(P4): module registry/` | |
| 4.11 | **🛑 User review** | |

**Smoke**: `curl POST /api/v1/registry`, `GET /api/v1/registry`, `POST /api/v1/registry/transform` (mock).

---

## Phase 5 — Module `connectors/` + `sources/` (1 ngày)

### 5A — connectors

| # | Task |
|---|------|
| 5A.1 | `git mv internal/api/system_connectors_handler.go` → `internal/modules/connectors/handler.go` |
| 5A.2 | `git mv internal/app/queries/list_connectors.go` → `internal/modules/connectors/queries.go` |
| 5A.3 | Decide: `internal/infra/http/kafka_connect.go` → `platform/http/kafka_connect.go` (shared) hay `modules/connectors/client.go` (module-only)? **Default: `platform/http/`** vì có khả năng module khác cũng dùng. |
| 5A.4 | Wire + routes |

### 5B — sources

| # | Task |
|---|------|
| 5B.1 | `git mv internal/api/sources_handler.go` → `internal/modules/sources/handler.go` |
| 5B.2 | `git mv internal/model/source.go` → unexported struct trong `repo.go` |
| 5B.3 | Wire + routes |

| 5.X | Build + test + smoke + commit `refactor(P5): connectors + sources` |
| 5.Y | **🛑 User review** |

---

## Phase 6 — Module `alerts/` (1 ngày) — sensitive

| # | Task |
|---|------|
| 6.1 | `git mv internal/infra/persistence/alert_manager.go` (446 LOC) → `internal/modules/alerts/manager.go` |
| 6.2 | Refactor structural: tách state machine logic ra `state_machine.go` (KHÔNG đổi logic, chỉ chia file) |
| 6.3 | `git mv internal/api/alerts_handler.go` → `handler.go` |
| 6.4 | `git mv internal/app/commands/ack_alert.go` + `silence_alert.go` → `commands.go` |
| 6.5 | `git mv internal/app/queries/list_alerts.go` → `queries.go` |
| 6.6 | Wire + routes |
| 6.7 | Build + test + smoke `curl /api/v1/alerts` |
| 6.8 | Commit `refactor(P6): module alerts/` |
| 6.9 | **🛑 User review** |

---

## Phase 7 — Module `provisioning/` (1.5 ngày) — **CRITICAL**

| # | Task | Note |
|---|------|------|
| 7.1 | Run `go test ./internal/infra/persistence/...` — ghi baseline | |
| 7.2 | `git mv internal/infra/persistence/provisioning_orchestrator.go` (729 LOC) → `internal/modules/provisioning/orchestrator.go` | |
| 7.3 | `git mv internal/infra/persistence/provisioning_state_machine.go` → `internal/modules/provisioning/state_machine.go` | |
| 7.4 | `git mv internal/infra/persistence/approval_service.go` → `internal/modules/provisioning/approval.go` | |
| 7.5 | `git mv internal/api/provisioning_handler.go` → `internal/modules/provisioning/handler.go` | |
| 7.6 | Tạo `module.go` + `routes.go` | |
| 7.7 | Wire | |
| 7.8 | Run lại unit test → expect 100% match với baseline 7.1 | |
| 7.9 | Staging smoke test: chạy full provisioning flow | |
| 7.10 | Commit `refactor(P7): module provisioning/` | |
| 7.11 | **🛑 CRITICAL User review** | |

**Rollback** nếu state machine không match: `git revert P7` rồi đào sâu lỗi.

---

## Phase 8 — Module `reconciliation/` + `jobs/` + `schedules/` (1 ngày)

### 8A — reconciliation

`git mv` 5 file `api/reconciliation_handler*.go` + 2 file `domain/reconciliation/` + commands → `internal/modules/reconciliation/`.

### 8B — jobs

`git mv` `domain/job/job.go` + commands liên quan → `internal/modules/jobs/`.

### 8C — schedules

`git mv` `api/schedule_handler.go` (329 LOC) + `api/transmute_schedule_handler.go` → `internal/modules/schedules/`.

| 8.X | Wire + routes + build + test + smoke + commit `refactor(P8): reconciliation+jobs+schedules` |
| 8.Y | **🛑 User review** |

---

## Phase 9 — Cleanup (0.5 ngày)

| # | Task | Verify |
|---|------|--------|
| 9.1 | `ls internal/api/` → empty | `rm -rf internal/api` |
| 9.2 | `ls internal/app/` → empty | `rm -rf internal/app` |
| 9.3 | `ls internal/domain/` → empty | `rm -rf internal/domain` |
| 9.4 | `ls internal/model/` → empty (hoặc giữ `platform/db/gormmodel/`) | `rm -rf internal/model` |
| 9.5 | `ls internal/infra/` → empty | `rm -rf internal/infra` |
| 9.6 | `ls internal/bootstrap/` → empty | `rm -rf internal/bootstrap` |
| 9.7 | Giảm `internal/router/router.go` ≤ 50 LOC | `wc -l internal/router/router.go` |
| 9.8 | `internal/server/wire.go` wire đủ 9-10 module | grep module imports |
| 9.9 | Build + test + smoke full suite | |
| 9.10 | Commit `refactor(P9): cleanup legacy folders` | |
| 9.11 | **🛑 User review** | |

---

## Phase 10 — Verify + deploy + report (0.5 ngày)

| # | Task |
|---|------|
| 10.1 | `go vet ./...` clean |
| 10.2 | `go build ./...` ok |
| 10.3 | `go test ./...` PASS — count = baseline |
| 10.4 | Smoke test full endpoint list (xem `06_test.md`): 15-20 endpoint qua bash script |
| 10.5 | Build binary `go build -o cms-server ./cmd/server` |
| 10.6 | Start binary → check log "ready" ≤ 5s |
| 10.7 | Deploy staging |
| 10.8 | Canary 1% production traffic |
| 10.9 | Monitor 24h: error rate, latency p99, NATS subscribe count |
| 10.10 | Promote 100% production |
| 10.11 | Cập nhật `agent/memory/global/lessons.md` (lesson "Vertical Slice pattern + 7 ràng buộc refactor lớn") |
| 10.12 | Cập nhật `agent/memory/global/project_context.md` profile mới (76 file → ~80 file phân bổ trong modules/) |
| 10.13 | Tag `git tag refactor/v1-modular-monolith` |
| 10.14 | Final user approve |

---

## Risk → Mitigation matrix (tham chiếu)

| Phase | Risk hot spot | Mitigation |
|-------|--------------|-----------|
| P0 | golangci config sai → CI red | Test rule local trước khi push |
| P1 | Bulk sed sai pattern → import vỡ | Diff review từng bulk replace, không tự tin thì commit từng file |
| P2 | Health endpoint regression → liveness probe fail → K8s rollback | Smoke test trước commit, có rollback tag |
| P3 | Test mapping fail do package đổi | `go test -count=1` để bypass cache, fix import path test file |
| P4 | Raw NATS publish thay bằng bus → behavior khác (sync vs async) | Đọc kỹ `nats_command_bus.go` Dispatch semantic, log đầy đủ |
| P5 | Module nào sở hữu `kafka_connect.go`? | Theo ADR-006, default `platform/http/` shared |
| P6 | `alert_manager.go` 446 LOC chứa background goroutine? | Grep `go func` trước move, đảm bảo lifecycle ổn |
| P7 | `provisioning_orchestrator.go` 729 LOC có shared state? | Baseline unit test bắt buộc trước & sau |
| P8 | `schedule_handler.go` đa case → tách module có làm vỡ logic? | Pure structural — chỉ move, không sửa |
| P9 | Còn file thừa? | `find internal -name "*.go" \| xargs grep -l "package (api\|domain\|model)"` |
| P10 | Production canary sai? | Feature flag rollback ngay |

---

## Summary effort

| Phase | Effort (Muscle) | Risk |
|-------|----------------|------|
| P0 | 0.5d | Thấp |
| P1 | 0.5d | Thấp |
| P2 (gate) | 0.5d | Thấp |
| P3 | 1d | Vừa |
| P4 | 1d | Vừa |
| P5 | 1d | Vừa |
| P6 | 1d | Cao |
| P7 (critical) | 1.5d | Cao |
| P8 | 1d | Vừa |
| P9 | 0.5d | Thấp |
| P10 | 0.5d | Vừa |
| **Tổng** | **~9 ngày** | |
