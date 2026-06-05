# Report — Plan vá Gap CDC QA + UI Admin Audit

**Date**: 2026-05-27
**Workspace**: `agent/memory/workspaces/plan-cdc-qa-gap-fix-2026-05-27/`
**Status**: PLAN READY — chờ User verb

---

## TL;DR

- Baseline audit: **35/64 (54.7%)** với 16 gap (4 P0 + 5 P1 + 7 P2).
- Plan 4 phase tổng **58.5h** đưa score lên **56/64 (87.5%)**.
- UI Admin Audit (16h) trên cdc-cms-web cho phép admin theo dõi composite + 16 gap + 4 metric health real-time.
- 21 file workspace đã tạo theo §7 Full Doc Set, tuân thủ §11 Memory Protection + §12 Brain Code Prohibition.

---

## Phase summary

| Phase | Effort | Gap | Score | Dependency |
|---|---|---|---|---|
| **P0 — Blocker** | 6.5h | G-1 ConsumerLag, G-2 OTel exporter, G-3 Prom scrape+alert, G-4 DLQ Circuit Breaker | 35 → 44 (68.75%) | — |
| **P1 — Pre-release** | 20h | G-5 failover smoke, G-6 WAL alert, G-7 pprof+goleak, G-8 ordering test, G-9 drift E2E | 44 → 51 (79.7%) | P0 |
| **P2 — Backlog** | 16h | G-10 Tier3 config, G-11 batches counter, G-12 adaptive batch, G-13 per-source pool, G-14 4 runbook, G-15 chaos test, G-16 k6 load | 51 → 56 (87.5%) | — (parallel) |
| **UI — Admin Audit** | 16h | BE 3 endpoint + FE 4 component + migration seed | 56 (no delta) | — (parallel P1) |

---

## Evidence base (file:line reference)

Plan dựa trên audit cũ `audit-cdc-qa-process-2026-05-26`, với code evidence cụ thể:

- **G-1 ConsumerLag chưa Set**: `centralized-data-service/internal/handler/kafka_consumer.go:~162` (reader init, không gọi `.Set()`).
- **G-2 OTel collector thiếu exporter**: `deployments/otel-collector-config.yml` (chỉ có receiver).
- **G-4 DLQ không có CB**: `kafka_consumer.go:~460` (CommitMessages không check rate).
- **G-7 pprof missing**: `cmd/worker/main.go` (không import `net/http/pprof`).
- **G-9 schema approve flow**: `cdc-cms-service/internal/app/commands/approve_schema_proposal.go` (unit test only).
- (Chi tiết file:line đầy đủ trong `03_implementation_phase_*.md` mỗi gap.)

---

## Architecture decisions (ADR)

| ADR | Topic | Choice |
|---|---|---|
| 001 | Prom scrape interval | Adaptive 15s/30s/60s |
| 002 | DLQ CB threshold | Configurable (worker.dlqCircuitBreakerRPS) |
| 003 | Audit gap state storage | DB (2 table) thay vì YAML |
| 004 | UI refresh strategy | React Query polling (30s/15s) |
| 005 | Audit page permission | Admin-only (RequireRole) |
| 006 | Per-source connection control | Semaphore on shared pool |
| 007 | Goroutine leak detection lib | go.uber.org/goleak |

Chi tiết: `04_decisions.md`.

---

## Files in workspace (21 file, tất cả vật lý)

```
plan-cdc-qa-gap-fix-2026-05-27/
├── 00_context.md
├── 01_requirements.md
├── 02_plan.md
├── 03_implementation_phase_p0.md
├── 03_implementation_phase_p1.md
├── 03_implementation_phase_p2.md
├── 03_implementation_phase_ui.md
├── 04_decisions.md
├── 05_progress.md
├── 06_validation.md
├── 07_status_report.md
├── 08_tasks_phase_p0.md
├── 08_tasks_phase_p1.md
├── 08_tasks_phase_p2.md
├── 08_tasks_phase_ui.md
├── 09_tasks_solution_phase_p0.md
├── 09_tasks_solution_phase_p1.md
├── 09_tasks_solution_phase_p2.md
├── 09_tasks_solution_phase_ui.md
├── 10_gap_analysis.md
└── report_plan_cdc_qa_gap_fix_2026-05-27.md  ← file này
```

---

## Governance compliance

| Quy tắc | Status |
|---|---|
| §1 Brain plan-only | ✓ Không touch code source |
| §7 Full Doc Set 21 file | ✓ |
| §11 Memory APPEND-only | ✓ 05_progress.md chỉ APPEND |
| §12 Brain Code Prohibition | ✓ Code demo trong MD block, không sửa .go/.ts/.sql |
| §14 Pre-flight Governance | ✓ Verify 21 file vật lý exist |

---

## Verb chờ User (next action)

| Verb | Hành động |
|---|---|
| `execute p0` | Muscle bắt đầu Phase P0 (4 gap blocker, 6.5h) |
| `execute p1` | Muscle bắt đầu Phase P1 (5 gap, 20h, yêu cầu P0 done) |
| `execute p2` | Muscle bắt đầu Phase P2 (7 gap, 16h, parallel) |
| `execute ui` | Muscle bắt đầu Phase UI Admin Audit (16h, parallel với P1) |
| `revise` | User chỉ định gap/phase cụ thể cần plan lại |
| `defer` | Tạm hoãn, lưu trạng thái |

---

## Skill đã sử dụng (theo §0.3)
- Memory governance (workspace init, append progress, doc set).
- Brain plan-only delegation (§1, §12).
- Composite score audit framework (5 nhóm × 16 criterion × L0-L4).
- ADR documentation pattern.
- Read tool (đọc audit cũ, evidence file:line).
- Write tool (21 file workspace).
- Code demo trong markdown block (Go, TypeScript, SQL, YAML, Bash).
- Dependency sequencing (P0 → P1, P2/UI parallel).
- Verification mapping (mỗi gap có verify command cụ thể).
