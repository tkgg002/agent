# 00_context — Plan vá gap CDC QA + CMS-FE Admin Audit UI

**Date**: 2026-05-27
**Workspace**: `plan-cdc-qa-gap-fix-2026-05-27`
**Type**: PLAN-ONLY (Brain Chairman). KHÔNG sửa code (§12 GEMINI).
**Source audit**: `agent/memory/workspaces/audit-cdc-qa-process-2026-05-26/` (16 gap → 4 P0 + 5 P1 + 7 P2).

## Mục tiêu
1. Vá toàn bộ 16 gap theo priority P0 → P1 → P2.
2. Thêm UI `/audit` trên `cdc-cms-web` cho Admin xem trạng thái QA Process realtime.
3. Endpoint backend `/api/v1/audit/*` trên `cdc-cms-service` cấp data cho UI.

## Phạm vi
- `centralized-data-service/` (worker plane) — metric set call, OTel exporter, DLQ circuit breaker, pprof endpoint, smoke test scripts.
- `cdc-cms-service/` (control plane) — endpoint audit, query layer, đọc `admin_actions` + scrape audit summary.
- `cdc-cms-web/` (FE) — page `/audit` + hook `useAuditStatus` + menu item.
- `deployments/` — OTel collector config + Prometheus scrape config + Grafana dashboard JSON + Alertmanager rules.

## Constraint (user directive)
- Đọc lessons trước → DONE (L985 silent-skip, L3100 conditional subscriber, L-CDC-circuit-breaker-2026-05-22, L-2026-05-26-metric-defined-but-never-set lesson mới từ audit, L-2026-05-26-trace, L-2026-05-26-log-sampling).
- Đọc `agent/GEMINI.md` → DONE (§1 Brain Chairman, §7 Full Doc Set, §11 APPEND-only, §12 Brain Code Prohibition).
- KHÔNG cheat db/config để đạt result.
- Plan rõ ràng + code demo chi tiết.
- Report dựa trên evidence thực tế.
- Mỗi service work check trước khi báo done (khi Muscle execute).
- Có 1 file `report_*.md`.

## Phases
- **Phase P0** (4 gap, ~6.5h Muscle): G-1 metric Set, G-2 OTel exporter, G-3 Prometheus scrape, G-4 DLQ circuit breaker.
- **Phase P1** (5 gap, ~20h): G-5 failover smoke, G-6 WAL slot alert, G-7 pprof+goleak, G-8 ordering test, G-9 drift E2E test.
- **Phase P2** (7 gap, ~16h): G-10 Tier3 config, G-11 batches counter, G-12 burst mode, G-13 per-source pool, G-14 runbook, G-15 chaos test, G-16 load test.
- **Phase UI** (~16h): CMS-FE `/audit` page + backend `/api/v1/audit/*` endpoint + types + tests.

**Tổng effort**: ~58.5h Muscle work. Theo §1 GEMINI: Brain plan + document → User approve → Muscle execute. Brain KHÔNG sửa code trong plan này.

## Evidence base (from audit + explore)
- Audit composite score 35/64 = 54.7% (mục tiêu sau plan: ~85% = 54/64).
- `centralized-data-service/pkgs/metrics/prometheus.go:73-79` — ConsumerLag gauge defined, no `.Set()` call.
- `deployments/otel-collector-config.yml` — exporter `debug` stdout only.
- `cdc-cms-service/internal/router/router.go` — Fiber CQRS, dualGet/dualPost pattern.
- `cdc-cms-web/src/App.tsx:17-30,187-206` — React Router v7 + lazy load pattern.
- `cdc-cms-web/src/services/api.ts:14-36` — axios + JWT interceptor.
- `cdc-cms-web/src/pages/SystemHealth.tsx` — dashboard reference pattern.
