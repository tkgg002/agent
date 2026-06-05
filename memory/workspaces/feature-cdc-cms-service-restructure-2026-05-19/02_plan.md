# 02_plan.md — Plan tổng (Hướng 3 — Vertical Slice / Modular Monolith)

> Đã chọn Hướng 3 từ `09_tasks_solution.md`. Doc này mô tả cấu trúc đích + 11 phase rollout chi tiết.

## 1. Cấu trúc đích (target tree)

```
cdc-cms-service/
├── cmd/
│   ├── server/main.go              ← giữ nguyên
│   └── sync_v2/main.go             ← giữ nguyên (CLI worker)
├── config/config.go                ← giữ nguyên (đã refactor pattern CMS_ envBinds)
├── docs/                           ← swagger generated, giữ nguyên
├── internal/
│   ├── modules/                    ★ NEW — bounded context (mental model A/B/C/D)
│   │   ├── mapping/                  A. Table mapping CRUD
│   │   │   ├── domain.go
│   │   │   ├── repo.go
│   │   │   ├── commands.go
│   │   │   ├── queries.go
│   │   │   ├── handler.go
│   │   │   ├── routes.go
│   │   │   ├── dto.go
│   │   │   └── *_test.go
│   │   ├── registry/                 A+B. Table registry + trigger debezium
│   │   ├── provisioning/             B. Orchestrator (target home cho 729-LOC)
│   │   ├── alerts/                   target home cho 446-LOC alert_manager
│   │   ├── reconciliation/           Reconciliation actions
│   │   ├── connectors/               C. Kafka Connect connectors display
│   │   ├── sources/                  C. Source connections registry
│   │   ├── health/                   D. /health, /ready, /system/health
│   │   ├── jobs/                     Async job tracking
│   │   └── schedules/                Schedule CRUD (transmute, retention, …)
│   ├── platform/                   ★ NEW — cross-cutting
│   │   ├── bus/                      NATS command bus + publisher
│   │   ├── db/                       GORM init, pool, migrate guard
│   │   │   └── gormmodel/            (option) shared GORM struct nếu cross-module
│   │   ├── http/                     Kafka Connect HTTP client
│   │   ├── observability/            logger, OTEL, probes
│   │   ├── middleware/               JWT, RBAC, audit, recover
│   │   └── eventbus/                 (option) In-process pub/sub cross-module
│   ├── server/                     ← composition root
│   │   ├── app.go                    Fiber app bootstrap
│   │   └── wire.go                   DI — assemble modules + platform
│   ├── router/
│   │   └── router.go                 ≤ 50 LOC — delegate sang modules/<X>/routes.go
│   ├── migrate/                    ← giữ — migration guard (đụng cache automigrate)
│   └── naming/                     ← giữ — helper convention (cross-module reusable)
├── migrations/                     ← giữ — SQL + embed
└── pkgs/                           ← giữ — natsconn, logger utilities công khai
```

### Cấu trúc bị XÓA

```
internal/api/                       ❌ (52 file) — di chuyển vào modules/<X>/handler.go
internal/app/commands/              ❌ (32 file) — vào modules/<X>/commands.go
internal/app/queries/               ❌ (24 file) — vào modules/<X>/queries.go
internal/app/ports/                 ❌ (4 file) — inline trong module hoặc xóa
internal/domain/                    ❌ (7 file) — vào modules/<X>/domain.go
internal/model/                     ❌ (11 file) — vào modules/<X>/repo.go hoặc platform/db/gormmodel/
internal/infra/                     ❌
   ├── cache/                       ❌ (chỉ doc.go — xóa luôn)
   ├── http/                        ❌ → platform/http/
   ├── messaging/                   ❌ → platform/bus/
   ├── observability/               ❌ → platform/observability/
   └── persistence/                 ❌ (27 file) — split vào modules/<X>/repo.go
internal/middleware/                ❌ → platform/middleware/
internal/bootstrap/                 ❌ → server/wire.go
```

## 2. Quy ước

### 2.1 Import rules (lint check)

| Rule | Mục đích | Tool |
|------|----------|------|
| `modules/<X>/` KHÔNG import `modules/<Y>/` | Cross-module isolation | `go-arch-lint` hoặc grep + golangci-lint depguard |
| `modules/` import `platform/` OK | Plumbing nội bộ | (allow) |
| `modules/` import `pkgs/` OK | Public utilities | (allow) |
| `platform/` KHÔNG import `modules/` | Layer rule | depguard |
| `router/router.go` chỉ gọi `modules/<X>.RegisterRoutes(app)` | Router thin | code review |

### 2.2 Module canonical file structure

```go
// modules/<X>/

domain.go      Entity struct + behavior (Validate, CanXxx, Transition)
repo.go        Repository struct + interface RepoPort (port nội bộ)
commands.go    Command handlers (write)
queries.go     Query handlers (read)
handler.go     HTTP Fiber handlers
routes.go      func RegisterRoutes(app *fiber.App, m *Module)
dto.go         Request/response payload struct + mapper
module.go      func New(deps Deps) *Module
*_test.go      Unit + integration test
```

### 2.3 Module constructor pattern

```go
package mapping

type Deps struct {
    DB        *gorm.DB
    Bus       bus.CommandBus
    Logger    *zap.Logger
}

type Module struct {
    repo    *Repository
    cmd     *CommandHandlers
    qry     *QueryHandlers
    h       *HTTPHandlers
}

func New(deps Deps) *Module {
    repo := NewRepository(deps.DB)
    cmd := NewCommandHandlers(repo, deps.Bus, deps.Logger)
    qry := NewQueryHandlers(repo, deps.Logger)
    h := NewHTTPHandlers(cmd, qry)
    return &Module{repo: repo, cmd: cmd, qry: qry, h: h}
}

func (m *Module) RegisterRoutes(app *fiber.App) {
    g := app.Group("/api/v1/mapping-rules")
    g.Get("/", m.h.List)
    g.Post("/", m.h.Create)
    g.Get("/:id", m.h.Get)
    g.Put("/:id", m.h.Update)
    g.Delete("/:id", m.h.Delete)
}
```

## 3. Roadmap 11 phase

### Phase 0 — Foundation (0.5 ngày)

**Mục tiêu**: Tạo skeleton, lint rule, DI wire-up trống.

**Steps**:
1. `mkdir -p internal/{modules,platform/{bus,db,http,observability,middleware,eventbus},server}`
2. Tạo `internal/server/wire.go` rỗng (chưa wire module).
3. Cài `golangci-lint` rule depguard cấm import cross-module (chuẩn bị file `.golangci.yml`).
4. Build sạch.

**Acceptance**: `go build ./...` PASS, `go vet ./...` clean.

**Review gate**: User approve skeleton.

---

### Phase 1 — Move platform (0.5 ngày)

**Mục tiêu**: Chuyển plumbing cross-cutting về `platform/`.

**Steps**:
1. `git mv internal/infra/messaging internal/platform/bus`
2. `git mv internal/infra/observability internal/platform/observability`
3. `git mv internal/middleware internal/platform/middleware`
4. `git mv internal/infra/http internal/platform/http`
5. `rm -rf internal/infra/cache` (chỉ doc.go rỗng).
6. Bulk update import: `cdc-cms-service/internal/infra/messaging` → `cdc-cms-service/internal/platform/bus`, …
7. Build + test.

**Acceptance**: `go build ./...` + `go test ./...` PASS.

**Review gate**: User approve.

---

### Phase 2 — Module `health/` (pilot — 0.5 ngày)

**Mục tiêu**: Test full pattern trên module đơn giản nhất.

**Files chạm**:
- `internal/api/health_handler.go` → `internal/modules/health/handler.go`
- `internal/api/system_health_handler.go` → `internal/modules/health/system_handler.go`
- Tạo `internal/modules/health/routes.go` + `module.go`.

**Steps**:
1. `git mv` 2 file vào `modules/health/`.
2. Tách `health_handler` + `system_health_handler` → `modules/health/handler.go` (gộp), `system.go`.
3. Tạo `modules/health/routes.go` mount `/health`, `/ready`, `/api/system/health`.
4. Sửa `router/router.go` xóa block health, gọi `healthModule.RegisterRoutes(app)`.
5. Wire vào `server/wire.go`.
6. Smoke test: `curl localhost:8081/health`.

**Acceptance**:
- `wc -l internal/modules/health/*.go` reasonable.
- `curl /health` → 200 OK.
- `curl /api/system/health` → 200 OK.

**Review gate**: ⚠️ Critical — User review pattern trước khi nhân rộng các module khác.

---

### Phase 3 — Module `mapping/` (pattern mẫu — 1 ngày)

**Mục tiêu**: Module table CRUD (mental model A).

**Files chạm**:

| Old path | New path |
|----------|---------|
| `internal/api/mapping_rule_handler.go` | `internal/modules/mapping/handler.go` |
| `internal/api/mapping_rule_handler_batch.go` | (merge vào handler.go) |
| `internal/api/mapping_rule_handler_commands.go` | (merge) |
| `internal/api/mapping_rule_handler_create.go` | (merge) |
| `internal/api/mapping_rule_handler_list.go` | (merge) |
| `internal/api/dto/mapping_rule_dto.go` | `internal/modules/mapping/dto.go` |
| `internal/app/commands/create_mapping_rule.go` | `internal/modules/mapping/commands.go` (Create method) |
| `internal/app/commands/update_mapping_rule.go` | `internal/modules/mapping/commands.go` (Update method) |
| `internal/app/commands/delete_mapping_rule.go` | `internal/modules/mapping/commands.go` (Delete method) |
| `internal/app/queries/list_mapping_rules.go` | `internal/modules/mapping/queries.go` |
| `internal/app/queries/get_mapping_rule.go` | (merge) |
| `internal/domain/mapping/rule.go` | `internal/modules/mapping/domain.go` |
| `internal/domain/mapping/errors.go` | (merge vào domain.go) |
| `internal/infra/persistence/mapping_rule_repo_gorm.go` | `internal/modules/mapping/repo.go` |

**Code demo chi tiết**: xem `03_implementation.md`.

**Acceptance**:
- `ls internal/modules/mapping/` → 7-8 file canonical.
- `curl /api/v1/mapping-rules` PASS.
- `go test ./internal/modules/mapping/...` PASS.

**Review gate**: User review module mẫu — chốt pattern.

---

### Phase 4 — Module `registry/` (1 ngày)

**Files chạm** (9 file `api/registry_handler*.go` + 4 file command + 1 model + 1 domain):

| Old | New |
|-----|-----|
| `api/registry_handler.go` | `modules/registry/handler.go` |
| `api/registry_handler_register.go` | (merge — chuyển business "register → restart debezium" sang `modules/registry/commands.go`) |
| `api/registry_handler_bulk.go` | (merge) |
| `api/registry_handler_dispatch.go` | (merge) |
| `api/registry_handler_tools_columns.go` | (merge) |
| `api/registry_handler_tools_scan.go` | (merge) |
| `api/registry_handler_transform.go` | (merge — **xóa raw NATS publish**, dùng `bus.Dispatch()`) |
| `app/commands/register_registry.go` | `modules/registry/commands.go` |
| `app/commands/v2_sync.go` | `modules/registry/commands.go` (Sync method) |
| `domain/source/object.go` | `modules/registry/domain.go` |
| `model/table_registry.go` | `modules/registry/repo.go` (GORM struct) |
| `infra/persistence/source_object_*.go` | `modules/registry/repo.go` |

**Acceptance**: `curl /api/v1/registry` + `POST /api/v1/registry` + `POST /api/v1/registry/transform` PASS.

**Review gate**: User approve.

---

### Phase 5 — Module `connectors/` + `sources/` (1 ngày)

| Old | New |
|-----|-----|
| `api/system_connectors_handler.go` (367 LOC) | `modules/connectors/handler.go` |
| `app/queries/list_connectors.go` | `modules/connectors/queries.go` |
| `infra/http/kafka_connect.go` | `platform/http/kafka_connect.go` (shared) hoặc `modules/connectors/client.go` |
| `api/sources_handler.go` | `modules/sources/handler.go` |
| `model/source.go` | `modules/sources/repo.go` |

**Acceptance**: `curl /api/system/connectors` + `curl /api/v1/sources` PASS.

**Review gate**: User approve.

---

### Phase 6 — Module `alerts/` (1 ngày)

**Critical**: tách `infra/persistence/alert_manager.go` (446 LOC) khỏi persistence.

| Old | New |
|-----|-----|
| `infra/persistence/alert_manager.go` | `modules/alerts/manager.go` |
| `api/alerts_handler.go` | `modules/alerts/handler.go` |
| `app/commands/ack_alert.go` | `modules/alerts/commands.go` |
| `app/commands/silence_alert.go` | (merge vào commands.go) |
| `app/queries/list_alerts.go` | `modules/alerts/queries.go` |

**Acceptance**: `curl /api/v1/alerts` PASS, ack + silence PASS.

**Review gate**: User approve.

---

### Phase 7 — Module `provisioning/` (1.5 ngày — **phase nhạy nhất**)

**Critical**: tách `infra/persistence/provisioning_orchestrator.go` (729 LOC) — state machine + CAS + NATS publish + step dispatch.

| Old | New |
|-----|-----|
| `infra/persistence/provisioning_orchestrator.go` (729) | `modules/provisioning/orchestrator.go` |
| `infra/persistence/provisioning_state_machine.go` | `modules/provisioning/state_machine.go` |
| `infra/persistence/approval_service.go` | `modules/provisioning/approval.go` |
| `api/provisioning_handler.go` | `modules/provisioning/handler.go` |
| `app/commands/provisioning_*.go` | `modules/provisioning/commands.go` |

**Risk mitigation**:
- Run unit test `provisioning_*_test.go` trước + sau move → 100% match.
- Smoke test full provisioning flow trên staging.
- Code review bắt buộc trước merge.

**Acceptance**: provisioning state machine pass tất cả test có sẵn.

**Review gate**: ⚠️ Critical — User approve.

---

### Phase 8 — Module `reconciliation/` + `jobs/` + `schedules/` (1 ngày)

| Old | New |
|-----|-----|
| `api/reconciliation_handler*.go` (5 file) | `modules/reconciliation/handler.go` |
| `app/commands/reconcile_*.go` | `modules/reconciliation/commands.go` |
| `domain/reconciliation/*.go` | `modules/reconciliation/domain.go` |
| `app/commands/v2_sync.go` (đã move ở P4) | (đã xong) |
| `domain/job/job.go` | `modules/jobs/domain.go` |
| `api/schedule_handler.go` (329) | `modules/schedules/handler.go` |
| `api/transmute_schedule_handler.go` | (merge) |

**Acceptance**: smoke test các endpoint reconciliation + schedule.

**Review gate**: User approve.

---

### Phase 9 — Cleanup (0.5 ngày)

**Steps**:
1. `ls internal/api/` → expect empty.
2. `ls internal/app/` → expect empty.
3. `ls internal/domain/` → expect empty.
4. `ls internal/model/` → expect empty (hoặc giữ `platform/db/gormmodel/` nếu shared).
5. `ls internal/infra/` → expect empty.
6. `ls internal/bootstrap/` → expect empty.
7. `rm -rf` các folder rỗng.
8. Giảm `router/router.go` ≤ 50 LOC.
9. Cập nhật `internal/server/wire.go` wire đủ tất cả module.

**Acceptance**:
- AC-5..AC-8 trong `01_requirements.md` PASS.
- `wc -l internal/router/router.go` ≤ 50.

**Review gate**: User approve cleanup.

---

### Phase 10 — Verify (0.5 ngày)

**Steps**:
1. `go vet ./...` clean.
2. `go build ./...` ok.
3. `go test ./...` PASS — count = baseline.
4. Smoke test full 10+ endpoint qua curl script.
5. Deploy staging → canary 1% production traffic → 100%.
6. Monitor 24h.
7. Cập nhật `agent/memory/global/lessons.md` với pattern Vertical Slice.
8. Cập nhật `agent/memory/global/project_context.md` profile mới.
9. Tag `git tag refactor/v1-modular-monolith`.

**Acceptance**: AC-9 + AC-10 PASS.

**Review gate**: Final user approve.

---

## 4. Gantt / dependency

```
P0 → P1 → P2 (pilot — gate)
                ↓
                ├──→ P3 (mapping — pattern lock)
                ├──→ P4 (registry)
                ├──→ P5 (connectors+sources)
                ├──→ P6 (alerts)
                ├──→ P7 (provisioning — sensitive)
                └──→ P8 (reconciliation+jobs+schedules)
                              ↓
                              P9 (cleanup) → P10 (verify)
```

P2 = gate quan trọng nhất. User approve pattern xong, P3..P8 có thể chạy song song nếu nhiều Muscle.

## 5. Rollback strategy

Mỗi phase = 1 git commit có message `refactor(P<n>): move <module> to modular structure`.
Nếu phase fail → `git revert <commit>` đưa về trạng thái trước phase đó. Vì mỗi phase chỉ `git mv` + sửa import (không refactor logic), revert an toàn.

## 6. Tham chiếu

| Tài liệu | Sử dụng |
|---------|---------|
| `00_context.md` | scope + user mental model |
| `10_gap_analysis.md` | 14 mismatch + import violation |
| `09_tasks_solution.md` | 4 hướng đi + chọn Hướng 3 |
| `03_implementation.md` | code demo Go module `mapping/` |
| `04_decisions.md` | ADR — 6 quyết định kiến trúc |
| `08_tasks.md` | break-down task chi tiết |
