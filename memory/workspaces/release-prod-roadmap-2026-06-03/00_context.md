# 00_context.md — Release Prod Roadmap (cdc-system)

> **Workspace**: `release-prod-roadmap-2026-06-03`
> **Created**: 2026-06-03
> **Owner**: Brain (Chairman) — planning & coordination only (CLAUDE.md §1, §12)
> **Status**: 🟡 Active (Plan phase)

## Mục tiêu
Lập lộ trình (task + timeline) audit nốt các luồng còn lại của `cdc-system` để đủ điều
kiện **release production**. Phase 1 (Source → Shadow) đã testing xong; cần đóng nốt
4 luồng: **Master (Transmute), SystemHealth, Reconcile, Monitor**, rồi E2E + hardening + release.

## Bản đồ luồng hệ thống (5 luồng cốt lõi)

| # | Luồng | Mô tả | Trạng thái audit |
|---|-------|-------|------------------|
| F1 | **Source → Shadow** | Debezium → Kafka → worker → shadow table (PG 5433/5436) | ✅ Phase 1 testing DONE |
| F2 | **Shadow → Master (Transmute)** | gjson + transform_fn + OCC upsert → master (PG 5434); 3 mode: cron / immediate / post_ingest | 🟡 Audit DONE, **chưa execute** (`feature-masters-page-audit-2026-06-02`) |
| F3 | **SystemHealth** | cms `/api/system/health` + worker probes + slow-SQL guards | 🔴 Chưa audit (nền: `bug-cms-slow-sql-probes`) |
| F4 | **Reconcile** | 3-tier hash window source vs dest → detect missing → heal qua OCC | 🔴 Chưa audit (nền: `bug-reconcile-mongodb-not-configured`) |
| F5 | **Monitor** | JobMonitor close-loop (NATS evt → transmute_schedule.last_status) + Dashboard V2 | 🔴 Chưa audit (nền: `feature-dashboard-v2-monitoring`, `feature-snapshot-monitor`) |

## Service liên quan (4 service monorepo)
- **centralized-data-service** (Go, worker plane) — 4 binary: worker / admin-api / sinkworker / profile_table
- **cdc-cms-service** (Go, control plane) — Fiber + NATS + Redis, hexagonal
- **cdc-auth-service** (Go, auth) — ⚠️ 0 test, chưa chạy local (gap nợ từ B1)
- **cdc-cms-web** (TS/React/Vite) — operator UI

## Tham chiếu workspace nền (đã có sẵn, dùng làm input audit)
- F2: `feature-masters-page-audit-2026-06-02` (audit + verify đã xong)
- F3: `bug-cms-slow-sql-probes-2026-05-26`, `centralized-data-service-config-audit`
- F4: `bug-reconcile-mongodb-not-configured-2026-05-26`
- F5: `feature-dashboard-v2-monitoring-2026-05-27`, `feature-snapshot-monitor-2026-05-25`, `feature-all-flows-trace-aggregation-2026-05-26`

## Giả định (cần User xác nhận để chốt timeline)
1. **Nhân lực**: 1 Muscle (CC CLI) thi công tuần tự + Brain audit/plan song song phase kế tiếp.
2. **Chưa có deadline cứng** từ Boss → timeline đề xuất tương đối từ 2026-06-03.
3. **Staging tồn tại** hoặc dựng được (hiện mới Local smoke — đây là RỦI RO, xem 02_plan §Risks).
4. `cdc-auth-service` 0 test là **prod blocker** phải đóng trước release.
