# 02_plan — Roadmap vá gap CDC QA + UI Audit

## Sequencing (dependency order)

```
Phase P0 (blocker)
  ├── G-1 metric Set ────┐
  ├── G-3 Prom scrape ───┼─→ Phase UI (depends on metric live)
  ├── G-2 OTel exporter ─┘
  ├── G-4 DLQ CB
  ├── G-17 DLQ Multi-pod
  └── G-18 Sched Multi-pod
         ↓
Phase P1 (pre-release)
  ├── G-5 failover smoke
  ├── G-6 WAL alert
  ├── G-7 pprof+goleak
  ├── G-8 ordering test
  └── G-9 drift E2E test
         ↓
Phase P2 (backlog) — parallel sprints
  └── G-10..G-16
         ↓
Phase UI Admin Audit (parallel with P1, ~16h)
  ├── BE: /api/v1/audit/qa-summary + /gaps + /metric-health
  └── FE: AuditPage + useAuditStatus + menu item
```

## Phase P0 — 9h Muscle effort

| Gap | Effort | Files thay đổi | Detail file |
|---|---|---|---|
| G-1 metric Set | 0.5h | `centralized-data-service/internal/handler/kafka_consumer.go` | `03_implementation_phase_p0.md` §G-1 |
| G-2 OTel exporter | 1h | `centralized-data-service/deployments/otel-collector-config.yml` | §G-2 |
| G-3 Prom scrape | 1h | `centralized-data-service/deployments/prometheus/prometheus.yml` (NEW) + K8s ServiceMonitor | §G-3 |
| G-4 DLQ CB | 4h | `centralized-data-service/internal/handler/dlq_circuit_breaker.go` (NEW) + `kafka_consumer.go` integration | §G-4 |
| G-17 DLQ Multi-pod | 0.5h | `centralized-data-service/internal/service/dlq_worker.go` | §G-17 |
| G-18 Sched Multi-pod | 1.5h | `centralized-data-service/internal/server/worker_server.go` | §G-18 |

## Phase P1 — 20h Muscle effort

| Gap | Effort | Files thay đổi |
|---|---|---|
| G-5 failover smoke | 4h | `centralized-data-service/scripts/smoke_failover.sh` (NEW) + CI workflow |
| G-6 WAL slot alert | 4h | `deployments/prometheus/alerts/wal_slot.yml` (NEW) + postgres_exporter deploy |
| G-7 pprof + goleak | 2h | `centralized-data-service/cmd/worker/main.go` (import pprof) + 3 test file TestMain |
| G-8 ordering test | 2h | `internal/service/schema_adapter_ordering_test.go` (NEW) |
| G-9 drift E2E test | 8h | `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` (NEW) + testcontainers |

## Phase P2 — 16h Muscle effort

| Gap | Effort | Files thay đổi |
|---|---|---|
| G-10 Tier3 config | 1h | `internal/service/recon_core.go` (off-peak window từ ReconCoreConfig) |
| G-11 batches counter | 0.5h | `pkgs/metrics/prometheus.go` + `batch_buffer.go` |
| G-12 burst adaptive | 2h | `kafka_consumer.go` (adaptive batch size khi lag > threshold) |
| G-13 per-source pool | 4h | `pkgs/database/postgres.go` (per-source semaphore) |
| G-14 runbook | 2h | `docs/runbooks/recon-drift.md` + `docs/runbooks/wal-slot-expire.md` (NEW) |
| G-15 chaos test | 4h | `scripts/chaos_network.sh` (NEW) — iptables drop |
| G-16 load test | 2.5h | `scripts/load_test.js` k6 script + report |

## Phase UI — 16h Muscle effort

### Backend (`cdc-cms-service`) — 8h
| Item | File |
|---|---|
| Handler | `internal/api/audit_handler.go` (NEW) |
| DTO | `internal/api/dto/audit_dto.go` (NEW) |
| Query layer | `internal/app/queries/get_qa_summary.go`, `list_gaps.go`, `get_metric_health.go` (NEW) |
| Persistence (gap state) | `internal/infra/persistence/gap_state_repo_gorm.go` (NEW) — đọc từ `cdc_system.qa_gap_state` |
| Router wire | `internal/router/router.go` thêm `dualGet(adminGroup, "/audit/...")` |
| DI wire | `internal/server/server.go` inject AuditHandler |
| Migration | `cdc-cms-service/migrations/0060_qa_gap_state.sql` (NEW) — table seed 16 gap |
| Test | `audit_handler_test.go` (NEW) |

### Frontend (`cdc-cms-web`) — 8h
| Item | File |
|---|---|
| Page | `src/pages/AuditPage.tsx` (NEW) |
| Hook | `src/hooks/useAuditStatus.ts` (NEW) |
| Types | `src/types/audit.ts` (NEW) — `AuditSummary`, `Gap`, `MetricHealth` interface |
| Component | `src/components/audit/RatingMatrix.tsx`, `GapList.tsx`, `MetricHealthCards.tsx` (NEW) |
| Route + Menu | `src/App.tsx` (lazy import + Route + menu item) |
| i18n strings | inline (theo pattern hiện có, không có i18n lib) |

## Detail files (Full Doc Set §7)

Mỗi phase có bộ:
- `03_implementation_{phase}.md` — thiết kế kỹ thuật chi tiết với code demo.
- `08_tasks_{phase}.md` — checklist task cho Muscle.
- `09_tasks_solution_{phase}.md` — hồ sơ giải pháp.

4 phase × 3 file = 12 file detail + bộ common (00/01/02/04/05/06/07/10) + report = 21 file.

## ADRs (sẽ ghi trong `04_decisions.md`)
- ADR-001: Adaptive metric scrape interval (15s/30s/60s)
- ADR-002: DLQ circuit breaker threshold (configurable vs hard-coded)
- ADR-003: Audit gap state lưu vào DB hay file YAML?
- ADR-004: UI refresh interval 30s vs SSE realtime
- ADR-005: Admin-only audit page (RequireRole("admin")) vs operator readable

## Verification per phase
- Mỗi gap có acceptance check (unit test / integration test / smoke check).
- Composite score recalculate sau từng phase.
- Service work verify (build + vet + test) trước khi báo done.

## Brain workflow
1. Brain (file này) tạo doc set đầy đủ.
2. User review + approve specific phase (verb: `execute p0`, `execute p1`, `execute ui`, `revise`, `defer`).
3. Muscle execute phase được approved → APPEND `05_progress.md`.
4. Sau mỗi phase: re-audit score + update `07_status_report.md`.
