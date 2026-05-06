# Status

- Current phase: Implemented and validated locally.

---

## 2026-05-07 02:42 ICT — Workspace CLOSE-OUT

**Status**: ✅ ALL P-tasks closed (Phase 2 cms refactor + dedup + test uplift complete).

### P-task ledger (final)
| Task   | Subject                            | Status | cms commit |
|--------|------------------------------------|--------|------------|
| T13 P3 | ActivityLog helper extraction      | ✅     | `7ea23d7`  |
| T14 P4 | V1↔V2 dedup (router-level swap)    | ✅     | `084a4a1`  |
| T15 P5 | Health collector probe split       | ✅     | `477ba19`  |
| T16 P6 | V2 sync atomicity (db.Transaction) | ✅     | `4a2a6e7`  |
| T17 P7 | Test uplift (5 đợt, 60 tests)      | ✅     | `48567b8` → `5804fe6` |

### T17 P7 final accounting
- **File-level DoD**: 12/13 service files có ≥1 test (= **92%**). `system_health_queries.go` (DB-only) defer to integration phase per project convention.
- **Coverage delta**:
  - `internal/service`: 20.0% → 21.4% (capped — adapter/orchestrator paths cần real DB)
  - `internal/service/health/probes`: 0% → 82.8% (chỉ Postgres + Redis ping uncovered, cần real client)
- **35% combined coverage DoD**: **NOT MET — accepted as partial DoD per architect ruling**. Reason: adapter layer (6 repository file: `mapping_rule_repo.go`, `pending_field_repo.go`, `registry_repo.go`, `schema_log_repo.go`, `source_repo.go`, `wizard_repo.go`) đúng spec phải cover bằng integration test (testcontainers) — không phải sqlmock unit mock. sqlmock = brittle hard-coded SQL strings, không validate GORM clause builder / transaction semantics / DB-side default.
- **Net new test surface**: 60 tests / 11 file across 4 sqlmock-free lanes:
  1. **Pure-fn tests** (predicates, validators, type-coerce): provisioning_state_machine, system_health_compute, shadow_automator (validateIdent), system_health_alerts (toFloat64/ownsAlertName/detectConditions), provisioning_orchestrator (correlation ID + trace stamping).
  2. **HTTP wire-contract tests** (httptest.NewServer): worker, nats, kafka_connect, debezium, kafka_lag.
  3. **Nil-receiver / no-op tests**: activity_logger (existing), reconciliation_service (Start/Stop contract).
  4. **JSON wire-contract tests**: approval_service (ApproveRequest/RejectRequest field tags).

### Deferred to backlog (NOT blocking workspace close)
- **T18 (proposed, not scheduled)**: Repository integration tests via testcontainers (~0.5d). Cover 6 repository file với real Postgres + GORM clause builder validation + transaction semantic verification. Schedule: enter sau khi vào phase integration testing chung của repo.

### Decision audit trail
- **Architect ruling Q3**: Accept partial DoD now → close T17 vĩnh viễn. Coverage gap documented above; adapter layer = wrong target for unit mock (sqlmock anti-pattern).
- **Project-convention Global Pattern lesson APPEND** (`agent/memory/global/lessons.md`): "Repository adapter layer ≠ unit test target" — see lesson section dated 2026-05-07.
- **Effort/value ratio**: ~1d sqlmock cho 6 file → ~18 brittle false-positive tests vs ~0.5d testcontainers → 1 lane real GORM cover toàn bộ. Value ratio ≥ 3x in favor of integration approach.

### Workspace artifacts (final inventory)
- `00_context.md`, `01_requirements_phase1.md`, `01_requirements_phase2_cms_refactor.md`, `01_requirements_phase2_decoupling.md`
- `02_plan_phase1.md`, `02_plan_phase2_cms_refactor.md`, `02_plan_phase2_decoupling.md`
- `03_implementation_phase1.md`
- `04_decisions.md`, `04_decisions_p3_critique_round2_2026-05-06.md`
- `05_progress.md` (immutable APPEND-only audit log; T13-T17 + 5 T17 đợt = 9 entries since session start)
- `06_validation_phase1.md`
- `07_status.md` (this file — close-out section just appended)
- `08_tasks_phase1.md`, `08_tasks_phase2_cms_refactor.md`, `08_tasks_phase2_decoupling.md`
- `09_tasks_solution_phase1.md`, `09_tasks_solution_phase2_cms_refactor.md`, `09_tasks_solution_phase2_decoupling.md`
- `10_gap_analysis.md`

**Workspace `feature-cdc-system-refactor` STATUS = CLOSED.**
