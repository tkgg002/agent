# 02_plan.md — Roadmap 4-Phase + Risk + Dependency

## 1. Tổng quan roadmap

```
┌─────────────────────────────────────────────────────────────────────────┐
│  PHASE 0  →  PHASE 1  →  PHASE 2  →  PHASE 3  →  [PHASE 4 OPTIONAL]    │
│  Coverage    Port Split   Comp Root   18 Cmds       Vertical Slice      │
│  3-5d        2-3d         2d          5-7d          5-8d                │
└─────────────────────────────────────────────────────────────────────────┘
   Gate G0      Gate G1      Gate G2      Gate G3      Gate G4
```

**Total estimate (Phase 0-3 mandatory):** 12-17 working days.
**With Phase 4:** 17-25 working days.

---

## 2. Phase dependency graph

```
Phase 0 (test coverage baseline)
   │
   ├── BLOCKS ──> Phase 1 (port split)
   │                 │
   │                 ├── BLOCKS ──> Phase 2 (composition root)
   │                 │                  │
   │                 └── BLOCKS ──> Phase 3 (18 commands refactor)
   │                                    │
   │                                    └── BLOCKS ──> Phase 4 OPTIONAL (vertical slice)
   │
   └── BLOCKS ──> Phase 3 (need test cho 18 cmds trước khi refactor)
```

**Lý do dependency:**
- Phase 1 cần coverage baseline để detect regression khi đổi import path.
- Phase 2 cần port split xong vì `buildRepos()` trả Repos struct chứa interface hẹp.
- Phase 3 cần port split + composition root đã pure-func để inject mock dễ.
- Phase 4 cần Phase 1-3 xong → mọi command đã dùng port hẹp → move folder không break import.

---

## 3. PHASE 0 — Coverage Baseline & High-Risk Test (3-5 ngày)

### 3.1 Mục tiêu
- Đo coverage hiện tại của `internal/app/` + `internal/server/` + `internal/bootstrap/`.
- Bù test để đạt ≥ 60% (app+server) và ≥ 70% (bootstrap).
- Setup `go-arch-lint` skeleton config (chưa enforce strict).

### 3.2 Task breakdown
| # | Task | Owner | Estimate |
|---|---|---|---|
| 0.1 | `make test-cover` → lưu `coverage_baseline_2026-06-01.txt` | Muscle | 30m |
| 0.2 | Phân tích coverage report — list package < 60% | Muscle | 1h |
| 0.3 | Viết test bổ sung cho `internal/app/commands/` (16 file thiếu test) | Muscle | 1-2d |
| 0.4 | Viết test cho `internal/server/server.go` (composition root smoke) | Muscle | 0.5d |
| 0.5 | Viết test cho `internal/bootstrap/registry_mirror.go` + `shadow_connection.go` | Muscle | 1d |
| 0.6 | Re-run `make test-cover` → confirm ≥ 60% (app+server), ≥ 70% (bootstrap) | Muscle | 30m |
| 0.7 | Tạo `tools/lint/arch-lint.yml` skeleton (rule: chỉ `internal/infra/persistence/` import `gorm`) | Muscle | 1h |
| 0.8 | Smoke test 53 routes → baseline `scripts/smoke_routes.sh` | Muscle | 2h |

### 3.3 Acceptance Gate G0
- ✅ `coverage_baseline_2026-06-01.txt` tồn tại + record %
- ✅ `internal/app/` ≥ 60%, `internal/server/` ≥ 60%, `internal/bootstrap/` ≥ 70%
- ✅ `go-arch-lint check` PASS với rule skeleton
- ✅ `scripts/smoke_routes.sh` 53/53 routes return 2xx/4xx-expected

### 3.4 Rollback Phase 0
- N/A — Phase 0 chỉ ADD test, không sửa production code → không có gì rollback.

---

## 4. PHASE 1 — Port Split (2-3 ngày)

### 4.1 Mục tiêu
- Tách `internal/app/ports/repository.go` (151 LOC / 10 interface) thành 9 file port-per-aggregate.
- Mỗi file ≤ 80 LOC, 2-4 interface hẹp ngữ nghĩa.
- XÓA `repository.go` sau khi mọi caller migrate xong.

### 4.2 File mới (tạo)
| File | Interface | Caller cũ |
|---|---|---|
| `master_port.go` | `MasterReader`, `MasterWriter`, `MasterApprover`, `MasterSwapper` | `MasterRepo` |
| `mapping_port.go` | `MappingRuleReader`, `MappingRuleWriter`, `MappingRuleApprover` | `MappingRuleRepo` |
| `source_port.go` | `SourceReader`, `SourceWriter`, `SourceLifecycleWriter` | `SourceRepo` |
| `job_port.go` | `JobReader`, `JobWriter`, `JobLogReader` | `JobRepo`, `JobLogRepo` |
| `recon_port.go` | `ReconReader`, `ReconWriter`, `ReconCheckpointer` | `ReconRepo` |
| `wizard_port.go` | `WizardReader`, `WizardWriter`, `WizardStepper` | `WizardRepo` |
| `system_connector_port.go` | `ConnectorReader`, `ConnectorWriter`, `ConnectorRunner` | `ConnectorRepo` |
| `registry_port.go` | `RegistryReader`, `RegistryWriter`, `RegistryMirror` | `RegistryRepo` |
| `transform_port.go` | `TransformReader`, `TransformWriter`, `BackfillRunner` | `TransformRepo` |

### 4.3 Task breakdown
| # | Task | Estimate |
|---|---|---|
| 1.1 | Tạo 9 file port mới (interface declaration only, KHÔNG đụng implementation) | 0.5d |
| 1.2 | Update import của handler/command/query (replace `ports.MasterRepo` → `ports.MasterReader/Writer/...`) | 1d |
| 1.3 | Update `server.go` wiring (composition root tạm thời) | 0.5d |
| 1.4 | XÓA `internal/app/ports/repository.go` | 5m |
| 1.5 | Chạy `go build ./...` + `make test` PASS | 30m |
| 1.6 | `grep -rn "ports.MasterRepo\|ports.MappingRuleRepo\|..." internal/` → empty | 10m |

### 4.4 Acceptance Gate G1 (FR-1 + AC-2/3)
- ✅ AC-2: 0 file import legacy interface
- ✅ AC-3: `repository.go` không tồn tại
- ✅ `make test` + `make test-integration` PASS
- ✅ Coverage ≥ baseline (NFR-1)
- ✅ 53 routes smoke PASS

### 4.5 Rollback Phase 1
- `git revert <commit-range>` — Phase 1 chỉ rename/move interface, không đổi runtime behavior → safe revert.
- Trước revert: `git tag pre-phase1` để mark mốc.

---

## 5. PHASE 2 — Composition Root Split (2 ngày)

### 5.1 Mục tiêu
- `internal/server/server.go` từ 333 LOC → ≤ 80 LOC (orchestrate only).
- Tách 5 file sub-builder pure function: `infra.go`, `repos.go`, `bus.go`, `routes.go`, `workers.go`.

### 5.2 File breakdown target
| File | LOC ước tính | Trách nhiệm |
|---|---|---|
| `server.go` | ≤ 80 | Orchestrate: gọi tuần tự buildInfra → buildRepos → buildBus → buildRoutes → buildWorkers → Start() |
| `infra.go` | ~ 60 | `buildInfra(cfg) (*Infra, error)` — DB, ShadowDB, NATS, Redis, OTel |
| `repos.go` | ~ 50 | `buildRepos(infra) Repos` — typed struct chứa concrete repo impl |
| `bus.go` | ~ 120 | `buildCommandBus(repos, infra) (*bus.CommandBus, error)` + `registerCommandHandlers(...)` |
| `routes.go` | ~ 80 | `buildRouter(handlers, mw) *fiber.App` |
| `workers.go` | ~ 70 | `buildWorkers(infra, bus) []Worker` (NATS subscribers, cron jobs) |

### 5.3 Task breakdown
| # | Task | Estimate |
|---|---|---|
| 2.1 | Tạo `infra.go` + move logic DB/NATS/Redis init từ `server.go` | 2h |
| 2.2 | Tạo `repos.go` + define `Repos` struct typed | 2h |
| 2.3 | Tạo `bus.go` + move 100 dòng `RegisterSubject`/`RegisterSync` | 4h |
| 2.4 | Tạo `routes.go` + move router building | 2h |
| 2.5 | Tạo `workers.go` + move worker/subscriber bootstrap | 2h |
| 2.6 | Refactor `server.go` còn ≤ 80 LOC, orchestrate tuần tự | 2h |
| 2.7 | `make test-integration` PASS | 1h |
| 2.8 | Verify NFR-5 (start time ≤ baseline + 100ms) | 30m |

### 5.4 Acceptance Gate G2 (FR-2 + AC-4/5)
- ✅ AC-4: `wc -l internal/server/server.go ≤ 80`
- ✅ AC-5: 5 file mới tồn tại
- ✅ Mọi sub-builder pure func (không receiver-state)
- ✅ `make test-integration` PASS
- ✅ Service start ≤ baseline + 100ms

### 5.5 Rollback Phase 2
- `git revert <commit-range>` — pure refactor, không đổi behavior.
- Risk gãy NATS subscriber: smoke test integration cover.

---

## 6. PHASE 3 — Refactor 18 Commands Raw `*gorm.DB` (5-7 ngày)

### 6.1 Mục tiêu
- Mọi command trong list 18 (xem `09_tasks_solution.md` §3) đổi từ `db *gorm.DB` → port hẹp.
- Implementation port hẹp đặt trong `internal/infra/persistence/<aggregate>_repo_gorm.go` (file đã có sẵn).
- 1 commit / 1 command để dễ revert.

### 6.2 Thứ tự refactor (đơn giản → phức tạp)
Reference `09_tasks_solution.md` §3 cho 18 commands cụ thể. Thứ tự:

**Batch 1 (đơn giản — 1 query, ít state):**
1. `mark_failed_log_retrying.go` → `JobLogWriter`
2. `update_master_table.go` → `MasterWriter`
3. `update_mapping_rule.go` → `MappingRuleWriter`
4. `cancel_job.go` → `JobWriter`

**Batch 2 (medium — multi-query, có validation):**
5. `register_source.go` → `SourceWriter` + `RegistryReader`
6. `approve_mapping.go` → `MappingRuleApprover` + `MasterReader`
7. `swap_master.go` → `MasterSwapper`
8. `provision_source.go` → `SourceLifecycleWriter`
9. `start_backfill.go` → `BackfillRunner` + `JobWriter`
10. `pause_connector.go` → `ConnectorRunner`
11. `resume_connector.go` → `ConnectorRunner`
12. `restart_connector.go` → `ConnectorRunner`

**Batch 3 (complex — saga, transaction multi-aggregate):**
13. `wizard_step_complete.go` → `WizardStepper` + `SourceReader` + `MappingRuleReader`
14. `trigger_reconciliation.go` → `ReconWriter` + `JobWriter`
15. `heal_drift.go` → `MappingRuleApprover` + `ReconCheckpointer`
16. `register_master.go` → `MasterWriter` + `MappingRuleReader` + `SourceReader`
17. `snapshot_transmute.go` → `TransformWriter` + `BackfillRunner`
18. `bulk_register_registry.go` → `RegistryWriter` + `SourceWriter` (highest risk)

### 6.3 Task breakdown (lặp lại cho mỗi command)
| # | Task | Estimate |
|---|---|---|
| 3.X.1 | Đọc command, identify SQL operation cần wrap | 15m |
| 3.X.2 | Thêm method vào port hẹp tương ứng (vd: `MasterWriter.SetActive(...)`) | 15m |
| 3.X.3 | Implement method trong `*_repo_gorm.go` (wrap raw SQL, không đổi logic) | 30m |
| 3.X.4 | Đổi command: `db *gorm.DB` → `repo ports.MasterWriter` | 30m |
| 3.X.5 | Update test command với mock port hẹp | 30m |
| 3.X.6 | `go build ./... && make test` PASS | 10m |
| 3.X.7 | Commit `refactor(cmd): <name> → port` | 5m |

**Total per command:** ~ 2h × 18 = 36h = ~ 5 ngày.
**Buffer:** +2 ngày cho Batch 3 (saga complex).

### 6.4 Acceptance Gate G3 (FR-3 + AC-6)
- ✅ AC-6: `grep -rn "gorm.io/gorm" internal/app/commands/` → empty
- ✅ Mỗi commit có test PASS riêng
- ✅ Coverage ≥ baseline
- ✅ NFR-2/3/4 không đổi (route, NATS subject, DB schema)

### 6.5 Rollback Phase 3
- Mỗi commit độc lập → revert riêng từng command nếu cần.
- Stop rule: nếu 3 command liên tiếp fail test → STOP, escalate Brain re-plan (Risk R1).

---

## 7. PHASE 4 — Vertical Slice (OPTIONAL, 5-8 ngày)

### 7.1 Mục tiêu
- Move file theo 8 BC vào `internal/bc/<bc_name>/`.
- Tạo `internal/platform/` cho cross-cutting (bus, auth, ratelimit, audit, deprecation, observability).
- Tạo `internal/shared/` cho technical primitives (naming, pg, id) — KHÔNG domain enum.
- Enforce cycle import qua `go-arch-lint`.

### 7.2 Move order (1 BC / iteration)
| # | BC | LOC ước tính | Estimate |
|---|---|---|---|
| 4.1 | `source` (8 cmd + 6 query + 18 api file) | ~ 2,500 | 1d |
| 4.2 | `master` (5 cmd + 1 query + 4 api file) | ~ 1,200 | 0.5d |
| 4.3 | `mapping` (4 cmd + 4 query + 6 api file) | ~ 1,800 | 1d |
| 4.4 | `transform` (4 cmd + 3 query + 5 api file) | ~ 1,500 | 1d |
| 4.5 | `reconciliation` (3 cmd + 4 query + 5 api file) | ~ 1,400 | 1d |
| 4.6 | `wizard` (3 cmd + 2 query + 3 api file) — **được phép cross-BC** | ~ 1,000 | 1d |
| 4.7 | `system_control` (4 cmd + 3 query + 6 api file) | ~ 1,700 | 1d |
| 4.8 | `observability` (2 cmd + 5 query + 5 api file) | ~ 1,400 | 0.5d |
| 4.9 | Tạo `internal/platform/` + `internal/shared/` | ~ 800 | 1d |

**Total:** ~ 8 ngày (best case 5d nếu auto-script).

### 7.3 Method: `git mv` (NFR-8 — preserve blame)
```bash
git mv internal/app/commands/register_source.go \
       internal/bc/source/commands/register_source.go
# UPDATE import path bằng sed/gofmt
git commit -m "refactor(bc): move source commands"
```

### 7.4 Acceptance Gate G4 (FR-4 + AC-7)
- ✅ AC-7: 0 cross-BC import (trừ wizard)
- ✅ `go-arch-lint check` PASS với rule strict
- ✅ Git blame preserve (test: `git log --follow internal/bc/source/commands/register_source.go`)
- ✅ Mọi BC move xong: full test + integration PASS
- ✅ Swagger regenerate PASS

### 7.5 Rollback Phase 4
- Risk cao (R2): move folder phá import path → CI fail kéo dài.
- Mitigation: move 1 BC, test sau mỗi BC. Nếu BC X fail → revert BC X only, BC trước đó keep.
- Worst case: revert toàn bộ Phase 4, giữ Phase 1-3 (vẫn 80% benefit).

---

## 8. Risk Register tổng (link `01_requirements.md` §6)

| ID | Risk | Phase | Mitigation áp dụng |
|---|---|---|---|
| R1 | 18 commands silent regression | 3 | 1 commit/cmd + test riêng + stop rule 3 fail |
| R2 | Phase 4 move phá import | 4 | Move từng BC, test giữa các bước, dùng `git mv` |
| R3 | Linter false positive | 0/4 | Test config trên 1 BC trước |
| R4 | Bootstrap bug hidden | 0 | Bug fix riêng workspace, không nhập vào refactor |
| R5 | User reject Phase 4 | 4 | Phase 4 OPTIONAL — Phase 1-3 standalone value |
| R6 | NATS subject/route typo | 1/2 | NFR-2/3 enforce + integration test |
| R7 | Phase 2 cycle giữa server_*.go | 2 | Pure func + `go vet` detect |
| R8 | Coverage không đạt 60% | 0 | Estimate 3-5d, > 1 tuần thì re-plan |

---

## 9. Timeline tổng (Gantt-style)

```
Week 1  │ ██████ Phase 0 (5d)
Week 2  │ ███ Phase 1 (3d) │ ██ Phase 2 (2d)
Week 3  │ ███████ Phase 3 batch 1+2 (5d)
Week 4  │ ██ Phase 3 batch 3 (2d) │ ████ Phase 4 BC 1-2 (4d)  [OPTIONAL]
Week 5  │ ████ Phase 4 BC 3-8 + lint (4d)                       [OPTIONAL]
```

**Critical path:** Phase 0 → Phase 1 → Phase 3 (Phase 2 chạy song song Phase 3 batch 1).
**Phase 4** chỉ start sau khi user approve Phase 3 hoàn tất.

---

## 10. Definition of Done — Mỗi Phase

(Reference `01_requirements.md` §5)

✅ Tất cả FR phase đạt
✅ Tất cả AC phase verify PASS
✅ `make test` + `make test-integration` PASS
✅ Coverage ≥ baseline (NFR-1)
✅ `go vet` + `golangci-lint` PASS
✅ Swagger regenerate PASS
✅ Smoke 53 routes PASS
✅ `05_progress.md` append entry phase đó
✅ `report_*.md` cập nhật files thay đổi + LOC
✅ User approve trước khi sang phase tiếp
✅ `/security-agent` PASS

---

## 11. Dependency vào tài liệu khác workspace

| File | Lý do tham chiếu |
|---|---|
| `00_context.md` | 8 BC mapping + lý do v2 |
| `01_requirements.md` | FR/NFR/AC chi tiết |
| `10_gap_analysis.md` | 7 gap evidence |
| `09_tasks_solution.md` | Solution code cho từng phase |
| `03_implementation.md` (sẽ tạo) | Code demo Go chi tiết |
| `04_decisions.md` (sẽ tạo) | ADR cho quyết định khó |
| `06_validation.md` (sẽ tạo) | Test plan + linter config |
| `08_tasks.md` (sẽ tạo) | Task breakdown chi tiết hơn |

---

## 12. Approval gate (link `01_requirements.md` §8)

**Mỗi phase cần user approve TRƯỚC khi Muscle thực thi:**
1. Brain present plan phase → User review
2. User approve → Muscle execute
3. Muscle complete + report → User verify (AC + smoke test)
4. User approve → Sang phase tiếp

**Brain (Antigravity) không bao giờ tự ý sang phase tiếp mà không có user approve.**
