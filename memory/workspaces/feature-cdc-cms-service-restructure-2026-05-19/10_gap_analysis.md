# 10_gap_analysis.md — Phân tích gap cấu trúc hiện tại

> Ground truth bằng `Explore` agent (thoroughness=very thorough) + `Bash` verify file size/import. Mọi con số có file:line.

## A. Cây thư mục hiện tại + đếm file

```
cdc-cms-service/
├── cmd/
│   ├── server/                          1 file
│   └── sync_v2/                         1 file
├── config/                              1 file  (config.go ăn ENV prefix CMS_)
├── docs/                                1 file  (swagger generated)
├── internal/
│   ├── api/                            52 file  ⚠️ FLAT NAMESPACE BLOAT
│   │   └── dto/                         1 file  (mapping_rule_dto.go)
│   ├── app/
│   │   ├── commands/                   32 file  ⚠️ FLAT
│   │   ├── ports/                       4 file  (command_bus, publisher, query_bus, repository)
│   │   └── queries/                    24 file  ⚠️ FLAT
│   ├── bootstrap/                       2 file
│   ├── domain/
│   │   ├── job/                         1 file
│   │   ├── mapping/                     2 file
│   │   ├── master/                      1 file
│   │   ├── reconciliation/              2 file
│   │   └── source/                      1 file
│   ├── infra/
│   │   ├── cache/                       1 file  (chỉ doc.go — empty package)
│   │   ├── http/                        3 file  (kafka_connect client)
│   │   ├── messaging/                   4 file  (NATS bus + publisher)
│   │   ├── observability/               9 file + probes/ 10 file
│   │   └── persistence/                27 file  ⚠️ FLAT + chứa application service
│   ├── middleware/                     10 file  (jwt, rbac, audit, …)
│   ├── migrate/                         1 file
│   ├── model/                          11 file  ⚠️ CHỒNG VỚI domain/
│   ├── naming/                          1 file
│   ├── router/                          1 file  (408 dòng)
│   └── server/                          1 file
├── migrations/                          1 .go file (embed.go) + SQL files
└── pkgs/                                6 file  (natsconn, logger, …)
```

**Tổng: ~76 .go production file + ~14 test file.**

## B. 14 mismatch / pain point cụ thể

### B.1 — File quá dài (> 300 LOC)

| # | File | LOC | Vấn đề |
|---|------|-----|--------|
| 1 | `internal/infra/persistence/provisioning_orchestrator.go` | **729** | State machine + CAS write + NATS publish + step dispatch — tất cả 1 file trong **persistence** layer |
| 2 | `internal/api/source_object_actions_handler.go` | **550** | Handler đa hành động + business logic |
| 3 | `internal/infra/persistence/source_object_v2_sync.go` | **495** | Sync logic V2 chèn vào persistence |
| 4 | `internal/infra/persistence/alert_manager.go` | **446** | State machine alert nhưng nằm trong persistence |
| 5 | `internal/router/router.go` | **408** | 1 file mount tất cả route — không group theo module |
| 6 | `internal/api/system_connectors_handler.go` | **367** | Kafka Connect CRUD + status — 1 file |
| 7 | `internal/app/commands/create_mapping_rule.go` | **338** | Command + DTO struct + raw SQL INSERT inline |
| 8 | `internal/api/schedule_handler.go` | **329** | Schedule CRUD đa case |

### B.2 — Package bloat (FLAT, > 15 file)

| # | Package | File count | Hệ quả |
|---|---------|-----------|--------|
| 9 | `internal/api/` | **52** | Khi dev mở folder, thấy 52 file mặt phẳng. Mapping_rule có 5 file `mapping_rule_handler*.go` (base, batch, commands, create, list). Registry có 9 file `registry_handler*.go`. Tìm theo tên prefix không trực quan. |
| 10 | `internal/app/commands/` | **32** | Không group theo aggregate. Command `create_mapping_rule` + `update_mapping_rule` + `delete_mapping_rule` nằm cạnh `ack_alert` + `register_registry` + `v2_sync`. |
| 11 | `internal/infra/persistence/` | **27** | Chứa lẫn lộn repository thuần + state machine + orchestrator + alert manager + approval service. |

### B.3 — File nằm sai chỗ (architectural smell)

| # | File:line | Hiện tại | Đáng lẽ |
|---|-----------|----------|---------|
| 12 | `infra/persistence/provisioning_orchestrator.go:79` | Comment line 1: `// Package service —` nhưng `package persistence`. Logic = state machine + NATS publish + step dispatch | Application service ở `modules/provisioning/orchestrator.go` |
| 13 | `infra/persistence/provisioning_state_machine.go:19` | Comment thừa nhận `"CMS-side copy of domain type"` | Pure domain → `modules/provisioning/state_machine.go` hoặc `domain/provisioning/` |
| 14 | `infra/persistence/approval_service.go:16` | NATS publish + multi-table write trong persistence | Application service ở `modules/approval/service.go` |
| 15 | `infra/persistence/alert_manager.go:64` | State machine alert trong persistence | `modules/alerts/manager.go` |
| 16 | `api/alerts_handler.go:20` `api/alerts_handler.go:38` | Handler nhận `*persistence.AlertManager` trực tiếp (concrete type) | Phải inject qua port interface |
| 17 | `api/provisioning_handler.go:30` `api/provisioning_handler.go:38` | Handler nhận `*persistence.ProvisioningOrchestrator` trực tiếp | Qua port interface |

### B.4 — Domain anemic

| File | Methods | Đánh giá |
|------|---------|---------|
| `domain/job/job.go` | `New()` (1 method) | Chỉ factory, không có behavior |
| `domain/mapping/rule.go` | 0 method | Pure struct |
| `domain/mapping/errors.go` | 0 method | Chỉ là sentinel error vars |
| `domain/master/*.go` | 0 method | Pure struct |
| `domain/reconciliation/*.go` | 0 method | Pure struct |
| `domain/source/object.go` | 0 method | Pure struct |

→ **Anemic domain** — toàn bộ behavior (validate, transition, can_advance) đang nằm ở `infra/persistence/provisioning_state_machine.go:58-73` và rải rác trong các command. Đây là anti-pattern: khi đặt invariant ở persistence, bất cứ command nào ghi DB đều phải nhớ check lại — dễ bypass.

### B.5 — Naming chồng chéo: `model/` vs `domain/`

| Concept | `model/` | `domain/` |
|---------|----------|-----------|
| Table registry | `model/table_registry.go` (GORM tag đầy đủ) | `domain/source/object.go` (clean struct) — V2 |
| Source | `model/source.go` | `domain/source/object.go` |
| Mapping rule | (không có ở model/, định nghĩa trong command) | `domain/mapping/rule.go` |
| Reconciliation | `model/reconciliation_*.go` (≥2 file) | `domain/reconciliation/*.go` |
| Job | `model/job.go` | `domain/job/job.go` |

→ Dev mới phải hỏi: *"Mở model/ hay domain/ trước?"*. Đây chính là "ko trực quan" mà user phàn nàn.

### B.6 — Bypass abstraction (import violations)

**17 file `api/` import `infra/persistence` trực tiếp**:

```text
api/activity_log_handler.go            api/registry_handler_dispatch.go
api/alerts_handler.go                  api/registry_handler_tools_scan.go
api/provisioning_handler.go            api/registry_handler.go
api/reconciliation_handler.go          api/schema_change_handler.go
api/reconciliation_handler_backfill.go api/registry_handler_bulk.go
api/reconciliation_handler_commands.go api/registry_handler_transform.go
api/reconciliation_handler_heal.go     api/registry_handler_register.go
api/source_object_actions_handler.go   api/registry_handler_tools_columns.go
api/source_objects_handler.go
```

→ Hexagonal ports tồn tại (`internal/app/ports/repository.go`) nhưng api/ thường skip → gọi thẳng concrete repo. Khó test, khó swap.

**4 file `app/commands/` import `infra/persistence` trực tiếp**:

```text
commands/ack_alert.go        commands/register_registry.go
commands/silence_alert.go    commands/v2_sync.go
```

→ Command (application) phụ thuộc persistence (infra) — vi phạm Dependency Inversion.

**7 file `api/` import `pkgs/natsconn` trực tiếp**:

```text
api/introspection_handler.go        api/reconciliation_handler.go
api/mapping_rule_handler.go         api/registry_handler.go
api/master_registry_handler.go      api/system_health_handler.go
api/transmute_schedule_handler.go
```

→ Handler giữ raw NATS client SONG SONG với CommandBus → 2 đường ra event. `registry_handler_transform.go:17` publish raw NATS `cdc.cmd.batch-transform` không qua bus → mất idempotency.

### B.7 — Business rule rò vào HTTP handler

| File:line | Vấn đề |
|-----------|--------|
| `api/registry_handler_register.go:82-87` | Sau khi command `RegisterRegistry` sync thành công, **handler tự gọi** `bus.Dispatch(RestartDebeziumCommand{})`. Quy luật "nếu register thì restart debezium" thuộc về application layer, không phải HTTP. |
| `api/registry_handler_transform.go:17` | Raw `natsClient.Conn.Publish("cdc.cmd.batch-transform", ...)` — bypass CommandBus, không có job row, không retry, không idempotency. |

## C. Bảng tóm tắt 4 nhóm mental model user vs cấu trúc hiện tại

| Nhóm | User mong đợi | Hiện tại (rải rác ở) |
|------|---------------|---------------------|
| **A. Table CRUD** | 1 folder chứa: handler + command + query + repo + domain entity của "table mapping" | `api/mapping_rule_handler*.go` (5 file) + `app/commands/create_mapping_rule.go` + `app/queries/list_mapping_rules.go` + `domain/mapping/rule.go` + `model/` (??) → 4 thư mục khác nhau |
| **B. Trigger dispatch** | 1 folder chứa: NATS bus + publisher + command dispatcher + handler hành động "register → trigger debezium" | `infra/messaging/` (bus) + `api/registry_handler_register.go:82` (rò business rule) + `pkgs/natsconn` (raw) + `app/commands/register_registry.go` |
| **C. Connection display** | 1 folder chứa: list connectors + status + connectors action | `api/system_connectors_handler.go` (367 LOC) + `app/queries/list_connectors.go` + `infra/http/kafka_connect.go` + `api/sources_handler.go` |
| **D. Health service** | 1 folder chứa: health + ready + system health + introspection | `api/health_handler.go` + `api/system_health_handler.go` (149) + `api/introspection_handler.go` |

→ **Để hiểu nhóm A**, dev mở 4 folder + jump ~7 file.
→ **Để hiểu nhóm B**, dev mở 4 folder + jump ~5 file.
→ **Để hiểu nhóm C**, dev mở 4 folder + jump ~4 file.

## D. So sánh với pattern khác trong dự án

| Pattern | Service | Đặc điểm |
|---------|---------|----------|
| Hexagonal flat (hiện tại) | `cdc-cms-service` | api/52 + commands/32 + queries/24 + persistence/27 — flat, gộp theo layer |
| Layered đơn giản | `cdc-auth-service` | 9 .go, không tách port — đơn giản nhưng quá nhỏ để học |
| Layered + envBinds | `centralized-data-service` (post refactor 2026-05-19) | 144 .go, config tách rõ, business gộp theo function name |
| **Vertical slice / Module** | (chưa có service nào) | Đề xuất cho `cdc-cms-service` |

## E. Lesson liên quan từ `lessons.md` (grep)

| Line | Tag | Nội dung |
|------|-----|---------|
| 753 | AutoMigrate conflict | `cdc-cms-service/internal/server/server.go:52` — Migrate guard cần ở 1 chỗ duy nhất, dễ trùng lặp khi spawn module |
| 1253 | GORM nested struct | `WorkerScheduleResponse` projection issue → infra phải có DTO riêng, không chia sẻ với domain |
| 1422 | Cross-repo workflow | Refactor đụng cả cms ↔ centralized-data-service → cần boundary rõ ràng |
| 2181 | CQRS Phase 3 scope | Discipline: refactor PHẢI có phase nhỏ + acceptance criteria từng phase |
| 2399 | Task #19 backup | "Đang làm rất lâu và rối kinh khủng" — bài học refactor mơ hồ → fail |
| 2755 | Migrations refactor pattern | 7 ràng buộc khi lên kế hoạch refactor `migrations/` — áp dụng tương tự ở đây |
| 3055 | V1 vs V2 drift | Có 2 thế hệ schema (cdc_table_registry vs source_object_registry + shadow_binding) → cấu trúc đích phải có chỗ cho cả hai |

## F. Root cause summary

```
Hexagonal đúng layer + sai granularity
        │
        ├── Layer dọc (api/app/domain/infra) gộp toàn service vào 4 thư mục
        │
        ├── Mỗi thư mục lại flat → 27/32/52 file mặt phẳng
        │
        ├── Dev không nhìn ra "module mapping nằm đâu" trong 1 click
        │
        └── Khi sửa 1 feature → jump 4-7 file, dễ rò business vào sai layer
```

**Kết luận**: Cần thêm tầng **module-by-feature (vertical slice)** ở trên hoặc thay thế tầng layer dọc — xem `09_tasks_solution.md` để chọn hướng.
