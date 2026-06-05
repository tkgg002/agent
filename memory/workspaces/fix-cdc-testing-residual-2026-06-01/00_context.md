# 00_context — Fix CDC Testing Residual (2026-06-01)

## Nguồn gốc
- Workspace nguồn: `agent/memory/workspaces/audit-cdc-testing-rerun-2026-06-01/`
- Re-audit kết quả: 50/64 (78.1%) — KHÔNG đạt 100%.
- User feedback: "test cái mẹ gì mà toàn ko đc 100% vậy" → kích hoạt §2 Bug Fixing Tự chủ (Full-loop).

## Hệ thống
- CDC pipeline: Mongo/PG/MariaDB → Debezium → Kafka → Worker → Shadow → Master.
- Service chính: `data-hub/centralized-data-service`, `data-hub/cdc-cms-service`.
- Stack QA: testcontainers-go, k6, prometheus, otel collector, goleak.

## Gap list residual (sau audit-rerun)
| ID      | Tên                                        | Tier | Effort |
|---------|--------------------------------------------|------|--------|
| G1-RES  | ConsumerOffset gauge + Set() call          | P0   | 1h     |
| G3-RES  | metrics_callback_test.go FAKE fix          | P0   | 1h     |
| G5-RES  | 2 FAIL cms mapping_rule assertion          | P0   | 0.5h   |
| G8-RES  | APPEND lessons về path correction + FAKE   | P0   | 0.5h   |
| G2-RES  | WAL auto snapshot resume                   | P1   | 4h     |
| G4-RES  | k6 CDC data path (sql ext)                 | P1   | 3h     |
| G6-RES  | Chaos pumba thay iptables                  | P2   | 2h     |
| G7-RES  | Adaptive batch throttle-down dest unhealthy| P2   | 3h     |

## Định nghĩa "Done"
- Mỗi gap PHẢI có: code patch + verify command + log thật (không fake).
- Không nhúng config/db cheating §0.
- Build/Vet/Test PASS trước khi mark done §3.
