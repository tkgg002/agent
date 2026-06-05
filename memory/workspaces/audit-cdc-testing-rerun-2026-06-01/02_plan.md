# 02_plan — Plan Re-Audit Execution

## Phase A — Context Load (15 min)
- A1. Đọc `agent/GEMINI.md` + `CLAUDE.md`.
- A2. Đọc `agent/memory/global/{project_context.md, active_plans.md, tech_stack.md}`.
- A3. Đọc 3 report execution: P0 / P1 / remaining.
- A4. Identify code root: `data-hub/centralized-data-service`, `data-hub/cdc-cms-service`, `data-hub/cdc-auth-service`.

## Phase B — Workspace Setup (5 min)
- B1. Tạo workspace `audit-cdc-testing-rerun-2026-06-01`.
- B2. Tạo TaskList 8 task (6 audit + 1 verify + 1 report).
- B3. Khởi tạo doc 00_context, 01_requirements, 02_plan, 05_progress.

## Phase C — Parallel Verification (45 min)
3 Explore subagent chạy parallel, không chia sẻ context:

- **Agent C1**: Verify P0 G-1..G-4 (ConsumerLag.Set, OTel exporter, Prometheus scrape, DLQ Circuit Breaker).
- **Agent C2**: Verify P1 G-5..G-9 (Failover smoke, WAL alert, pprof+goleak, ordering test, drift E2E).
- **Agent C3**: Verify P2 G-10..G-16 + G-NEW (Tier3, BatchesFlushed, Adaptive batch, PerSourcePool, Runbooks, Chaos, k6, Delete ordering, Source DB metric, Soak script).

Mỗi agent xuất:
- Rating L0..L4 + evidence file:line.
- Verdict FIXED / PARTIAL / FAKE / NOT IMPLEMENTED.
- Risk 1 câu.
- Delta vs audit gốc.

## Phase D — Build/Test Verification (15 min)
- D1. `go build + go vet` cho 3 service Go.
- D2. `go test -short` cho package có file test.
- D3. Verify pre-existing failures (TestSanitizeMongoDSN + handler goleak) status.
- D4. Detect regression mới.

## Phase E — Synthesis (20 min)
- E1. Tổng hợp matrix 16 tiêu chí × rating mới.
- E2. Tính composite score + delta.
- E3. Tạo 06_validation.md matrix.
- E4. Tạo 10_gap_analysis.md với gap residual.
- E5. Tạo 07_status_report.md.
- E6. Tạo report_audit_testing_rerun_2026-06-01.md.
- E7. APPEND active_plans.md (không sửa).

## Phase F — Pre-flight Check §14 (5 min)
- F1. Liệt kê file vật lý đã tạo.
- F2. Confirm §11 APPEND-only.
- F3. Confirm §12 không sửa source code.
- F4. Verb chờ user: `re-execute` (fix gap residual) / `revise` / `accept`.

## Definition of Done
- Tất cả 8 task TaskList completed.
- 8 file workspace tồn tại trong filesystem.
- Build/vet/test status documented (PASS hoặc FAIL có lý do).
- Composite score mới có math chứng minh.
- Verdict cho từng gap có evidence.
