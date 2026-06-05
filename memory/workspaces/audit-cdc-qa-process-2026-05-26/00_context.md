# 00_context — Audit CDC QA Process

**Date**: 2026-05-26
**Workspace**: `audit-cdc-qa-process-2026-05-26`
**Type**: AUDIT (read-only) — KHÔNG sửa code, chỉ rà soát hiện trạng.

## Scope
User cung cấp 5 nhóm tiêu chí QA cho hệ thống CDC:
1. **Functional & Correctness**: Data Reconciliation, Schema Drift, Event Ordering.
2. **Stability & Resilience**: Failover/Self-Healing, Network Flicker, LSN/Offset Expire, DLQ.
3. **Performance & Scalability**: Data Lag, Throughput/TPS, Backlog Catch-up, Source DB Overhead.
4. **Resource Utilization**: Memory Leak (soak), Concurrency/Throttling.
5. **Metric Monitor**: Replication Lag, CPU/Mem, Disk I/O, Network Thruput.

## Target Repo
- `data-hub/centralized-data-service` (Go worker plane).
- `data-hub/cdc-cms-service` (control plane).
- `data-hub/cdc-cms-web` (operator FE).
- `data-hub/cdc-auth-service` (auth plane).

## Audit Output
- `06_validation.md` — Matrix tiêu chí × {L0..L4} × file evidence.
- `10_gap_analysis.md` — Gap rõ ràng + recommend (test harness, metric, runbook).
- `report_cdc_qa_process_audit_2026-05-26.md` — Executive summary cho User.

## Constraints (User directive)
- Đọc lessons trước (DONE — lessons L985/L3100/L-CDC-circuit-breaker/L-CDC-route-empty-silent-skip relevant).
- Đọc `agent/GEMINI.md` (DONE).
- KHÔNG sửa code/config/db để đạt kết quả audit.
- Plan rõ ràng, code demo cho recommend (nếu có).
- Report dựa trên evidence thực tế từ codebase, không bịa.
- Có 1 file report_*.md.

## Rating Scale
| Level | Nghĩa |
|---|---|
| L0 | Thiếu hoàn toàn (no code/no test/no metric) |
| L1 | Có dấu vết (config/log/scaffold) nhưng chưa có test harness |
| L2 | Một phần (code hoạt động trong happy path, thiếu test/metric) |
| L3 | Đáp ứng cơ bản (code + 1 test/metric path) |
| L4 | Đáp ứng đầy đủ (code + test harness + metric + runbook) |

## Quy trình
1. Spawn 2 Explore subagent parallel để scan codebase tìm bằng chứng từng tiêu chí.
2. Tổng hợp matrix.
3. Gap analysis + recommend test harness (không sửa code).
4. Report + append lessons + active_plans.
