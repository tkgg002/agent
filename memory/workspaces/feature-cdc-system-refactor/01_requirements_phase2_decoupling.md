# Phase 2 v2 — cdc-cms-service Decoupling + CQRS Refactor — Requirements

> **Date**: 2026-05-05 17:00 ICT
> **Owner**: Brain (planning) → Muscle (execution per pillar)
> **Workspace**: `feature-cdc-system-refactor`
> **Supersedes**: `01_requirements_phase2_cms_refactor.md` (Phase 2 v1 — layer-only refactor, **không đủ** decoupling)

---

## 1. Bối cảnh — Tại sao Phase 2 v2

Phase 2 v1 (`phase2_cms_refactor`) chỉ tách layer trong CMS (handler→service→repo), KHÔNG giải quyết được nguyên lý **Separation of Powers** giữa Control Plane và Data Plane. User feedback (2026-05-05 16:45 ICT):

> "Nếu để cdc-cms-service (API) vừa quản lý metadata vừa thực thi các tác vụ nặng (như reconciliation, backfill, sync) thì sẽ vi phạm nguyên tắc Separation of Powers và làm hệ thống rất khó scale."

Audit thực tế:
- **CMS đã dispatch đúng**: 12/14 trigger endpoint publish NATS chuẩn (recon-check, recon-heal, retry-failed, debezium-signal, recon-backfill-source-ts, debezium-snapshot, create-default-columns, standardize, scan-fields, detect-timestamp-field, backfill, alter-column, transmute, master-create).
- **CMS còn INLINE 2 endpoint**:
  1. `master_registry_handler.go:630` `Swap` — chạy `ALTER TABLE ... RENAME TO` trong TX request thread.
  2. `source_object_v2_sync_service.go::SyncFromLegacy` (called inline tại `registry_handler.go:148/268/316`) — UPSERT chuỗi vào `cdc_system.*` blocking request.
- **12/14 cmd subject** thiếu companion `cdc.evt.X.completed` → status không close-loop (Lesson #1292 violation).
- **State tracking phân mảnh**: 5 table khác nhau lưu execution status (master_binding.schema_status, transmute_schedule.last_status, recon_runs.status, failed_sync_logs.status, source_object_registry.provisioning_state) → không có view tổng hợp "job đang chạy".

## 2. User mandate (4 Pillar — chốt thứ tự)

| Pillar | Title | Mục tiêu |
|---|---|---|
| **P1** | Khởi tạo cấu trúc `domain/` và `app/` | Định nghĩa interface Repository + Command Bus + Query Bus + Publisher |
| **P2** | Di chuyển logic Handler → `app/queries/` | Read paths trước (low risk), 1-1 mapping HTTP endpoint ↔ Query handler |
| **P3** | Chuyển Action Handler → `app/commands/` + NATS Command Dispatcher | Write paths, 202 Accepted async, fire-and-forget cmd có companion event |
| **P4** | Chuẩn hóa `infra/persistence/` | Raw SQL CHỈ ở infra layer; domain + app KHÔNG còn `db.Raw()`/`db.Exec()` |

**Nguyên tắc**: thực thi tuần tự P1→P2→P3→P4. Mỗi pillar là 1 commit/PR standalone, có thể revert riêng.

## 3. Outer boundary — Control Plane vs Data Plane (ranh giới đỏ)

### A. cdc-cms-service (Control Plane / Brain)
**Chỉ làm**:
1. CRUD metadata: mapping_rule, source_object, master_binding, connection, alert_rule.
2. Validate nghiệp vụ + RBAC (admin/operator/ops-admin/destructive).
3. Read aggregate cho UI: list, detail, drift report, health snapshot.
4. **Dispatch command** qua NATS. Trả 202 Accepted + `job_id` cho action heavy.
5. Read job status từ `cdc_system.*` để hiển thị progress.

**Không làm**:
- ALTER TABLE / RENAME / DDL trên DB đích.
- Cross-DB scan (source DB, master DB).
- Reconciliation drift compute (chỉ READ report đã được worker viết).
- Backfill / sync data row.

### B. centralized-data-service (Data Plane / Muscle)
**Chỉ làm**:
1. Subscribe NATS cmd → execute heavy work.
2. Tương tác Source DB (Mongo/Postgres) + Destination DB.
3. UPDATE state về `cdc_system.*` để CMS đọc.
4. Publish `cdc.evt.X.completed` companion event sau mỗi cmd handler.

**Không làm**:
- Expose HTTP endpoint cho FE (trừ /healthz, /metrics).
- Validate RBAC user (worker không có user context).

## 4. Inner structure — Clean Architecture / CQRS (CMS sau refactor)

```
cdc-cms-service/
├── cmd/server/main.go               # bootstrap, DI wire
├── internal/
│   ├── domain/                      # P1 — pure business
│   │   ├── mapping/                 #   entity + value object
│   │   ├── source/
│   │   ├── master/
│   │   ├── reconciliation/
│   │   └── job/                     #   Job entity (id, type, status, result)
│   ├── app/                         # P2 + P3 — application layer
│   │   ├── ports/                   # P1 — interface
│   │   │   ├── repository.go        #   MappingRuleRepo, SourceRepo, MasterRepo, JobRepo
│   │   │   ├── command_bus.go       #   CommandBus.Dispatch(cmd) error
│   │   │   ├── query_bus.go         #   QueryBus.Ask(query) (result, error)
│   │   │   └── publisher.go         #   Publisher.Publish(subject, payload) error
│   │   ├── queries/                 # P2 — read
│   │   │   ├── list_mapping_rules.go
│   │   │   ├── get_master_by_name.go
│   │   │   ├── get_recon_report.go
│   │   │   └── ...                  #   1 query per use-case
│   │   └── commands/                # P3 — write
│   │       ├── register_source.go
│   │       ├── approve_master.go
│   │       ├── trigger_recon_check.go
│   │       ├── master_swap.go       #   moved from inline → cmd → worker
│   │       ├── v2_sync.go           #   moved from inline → cmd → worker
│   │       └── ...
│   ├── infra/                       # P4 — outer ring
│   │   ├── persistence/             #   GORM impl — raw SQL CHỈ Ở ĐÂY
│   │   │   ├── mapping_rule_repo_gorm.go
│   │   │   ├── source_repo_gorm.go
│   │   │   ├── master_repo_gorm.go
│   │   │   └── job_repo_gorm.go
│   │   ├── messaging/               #   NATS adapter
│   │   │   ├── nats_publisher.go
│   │   │   └── nats_command_bus.go  #   CommandBus impl wrap NATS publish
│   │   ├── http/                    #   Kafka Connect REST client
│   │   └── cache/                   #   Redis adapter
│   └── api/                         # thin Fiber adapter
│       ├── mapping_rule_handler.go  #   ≤100 dòng — HTTP marshal + call query/command bus
│       ├── master_registry_handler.go
│       └── ...
└── pkgs/                            # cross-cutting (logger, validator)
```

## 5. Definition of Done

| # | Criterion | Verify command |
|---|---|---|
| 1 | `internal/domain/`, `internal/app/{ports,queries,commands}`, `internal/infra/{persistence,messaging,http,cache}` đã tạo | `ls internal/domain internal/app internal/infra` |
| 2 | Mỗi handler `internal/api/*.go` ≤100 dòng, KHÔNG có business logic | `wc -l internal/api/*.go` mọi file ≤100 |
| 3 | Mọi raw SQL (`db.Raw`, `db.Exec`) chỉ tồn tại trong `internal/infra/persistence/` | `grep -r "db.Raw\|db.Exec" internal/{api,domain,app}/` = 0 |
| 4 | Mỗi handler READ gọi `query.Bus.Ask(...)`, không trực tiếp gọi repo | `grep -r "Repo\." internal/api/` = 0 trong handler READ |
| 5 | Mỗi handler WRITE gọi `cmd.Bus.Dispatch(...)`, không trực tiếp gọi service | `grep -r "Service\." internal/api/` = 0 trong handler WRITE |
| 6 | `cdc.cmd.master-swap` cmd subject + worker handler tồn tại; CMS publish thay vì inline ALTER | `grep "ALTER TABLE.*RENAME" internal/api/ internal/app/` = 0 |
| 7 | `cdc.cmd.v2-sync` cmd subject + worker handler tồn tại; CMS publish thay vì inline UPSERT | `grep "SyncFromLegacy" internal/api/` = 0 (chỉ ở cmd handler) |
| 8 | Mỗi `cdc.cmd.X` có companion `cdc.evt.X.completed` subject + JobMonitor subscribe | mapping table 1-1 trong `02_plan_phase2_decoupling.md` |
| 9 | Bảng `cdc_system.cdc_jobs` (NEW) hoặc per-domain status column track mọi job | migration audit |
| 10 | Test coverage `internal/domain/` + `internal/app/` ≥ 50% | `go test -cover ./internal/domain/... ./internal/app/...` |
| 11 | Zero regression — 8 endpoint smoke PASS sau mỗi pillar | `curl` thực tế (xem §6) |
| 12 | Mỗi pillar `/security-agent` PASS trước commit | report log |

## 6. Verification — 8 endpoint smoke (exercise-driven, Lesson #1264)

| # | Endpoint | Expect |
|---|---|---|
| 1 | `GET /health` | 200 `{"status":"ok"}` |
| 2 | `GET /api/system/health` (auth) | 200, JSON snapshot, `overall.status` field |
| 3 | `GET /api/sync/health` (auth) | 200, `total_registered ≥ 0` |
| 4 | `GET /api/v1/source-objects` (auth) | 200, `data: []` |
| 5 | `GET /api/mapping-rules` (auth) | 200, mảng object có `id,target_table,source_field,status` |
| 6 | `GET /api/v1/system/connectors` (auth) | 200 hoặc 502 (Kafka Connect down OK miễn không 500) |
| 7 | `GET /api/reconciliation/report` (auth) | 200, mảng |
| 8 | `GET /api/v1/masters` (auth) | 200, mảng |

**Bonus action smoke** (P3+):
| # | Endpoint | Expect |
|---|---|---|
| A1 | `POST /api/reconciliation/check` body `{"target":"orders"}` | 202 + `{"job_id":"..."}` |
| A2 | `GET /api/jobs/{job_id}` | 200, status `pending|running|success|failed` |
| A3 | `POST /api/v1/masters/{name}/swap` (admin) | 202 + `job_id`, sau worker chạy → 200 với status=success |

## 7. Constraints (must hold)

- **No API contract change**: path / request / response shape giữ nguyên. Trường thêm `job_id` chỉ append vào response 202, không break shape cũ.
- **No NATS subject removal**: 18 subject hiện tại GIỮ nguyên. Subject MỚI có thể thêm.
- **No DB schema breaking change**: chỉ ADD COLUMN/TABLE, không DROP/RENAME. `cdc_system.cdc_jobs` table mới (P3) cần migration riêng — coordinate với `centralized-data-service/migrations/`.
- **Per-pillar commit**: 1 pillar = 1 PR/commit standalone, có thể revert riêng. KHÔNG mix.
- **Per-pillar gate**: build PASS + unit test PASS + 8 endpoint smoke PASS + `/security-agent` PASS + APPEND `05_progress.md` trước commit.
- **APPEND-only memory** (CLAUDE.md §11).
- **Real verification** (CLAUDE.md §3 + Lesson #1264): exercise endpoint thực tế.
- **Brain-Code Prohibition** (CLAUDE.md §12): plan này = Brain; code chờ user approve → Muscle.
- **Idempotency**: cmd handler must guard `WHERE status='pending'` để retry không double-execute (Lesson #1292).
- **Cascade safety** (Lesson #1399): event-driven cascade phải có circuit breaker — JobMonitor KHÔNG chain trigger cmd khác trong handler completion.

## 8. Out of scope (KHÔNG làm Phase 2 v2)

- FE changes (`cdc-cms-web`).
- Auth service changes (`cdc-auth-service`).
- DB migration mới ngoài `cdc_system.cdc_jobs`.
- New features (chỉ refactor architecture).
- Worker scheduler/cron changes (giữ nguyên transmute_scheduler, dlq_worker, recon leader, partition_dropper).

## 9. Reference lessons (đã re-read trước plan v2)

| Lesson | Áp dụng |
|---|---|
| #160 Simplicity First | Không over-engineer; reuse existing JobMonitor pattern, không tạo Saga framework. |
| #258 No Cross-Domain Model in CQRS Handler | Query handler chỉ đọc 1 aggregate, command handler chỉ write 1 aggregate. |
| #475 Forgotten Field Assignment | Patch command handler phải gán mọi field. |
| #1240 Schema rename ↔ search_path | Qualify SQL `cdc_system.tab` hoặc set search_path. |
| #1253 GORM Raw().Scan no nested struct | Query result struct flat, không nested. |
| #1264 PASS exercise-driven | 8 endpoint smoke + 3 action smoke (A1-A3). |
| #1277 Tuân thủ user rule literal | 4 pillar order P1→P2→P3→P4 không hoán đổi. |
| #1292 Fire-and-forget cmd cần companion event | 12/14 cmd cần thêm `cdc.evt.X.completed`; JobMonitor subscribe. |
| #1399 Event-Driven Cascade Liability | JobMonitor KHÔNG fire cmd khác trong handler completion → tránh runaway. |
| 2026-04-29 (line 828-852) NATS async + service boundary | Precedent: 12 ADR-015 violation đã fix; plan v2 đi tiếp đường lối này. |
| 2026-05-05 (line 1817) Cross-repo decoupling | CMS không mount/import worker code; chỉ qua NATS subject contract. |

## 10. Effort estimate

| Pillar | Effort | Risk | Parallel-able |
|---|---|---|---|
| P1 Setup interface | 2d | LOW | — |
| P2 Queries migration | 3d | LOW (read only) | parts parallel |
| P3 Commands + Bus + Move 2 INLINE | 5d | HIGH (write + worker change) | sequential |
| P4 Infra persistence cleanup | 3d | MEDIUM | parts parallel |
| **Total sequential** | **13d** | | |
| **With parallel within pillar** | **~10d** | | |
| **Pre-commit gate overhead +20%** | **~12-13d** | | |
| **Realistic** | **3 tuần** với 1 engineer | | |
