# 00_context — Re-Audit CDC Testing Rerun 2026-06-01

## Date
2026-06-01

## Type
RE-AUDIT (read-only verification). KHÔNG sửa code, chỉ rà soát trạng thái fix sau 2-3 vòng vá P0/P1/P2 + remaining.

## Trigger
User yêu cầu audit lại 5 nhóm × 16 tiêu chí testing CDC sau khi đã ra lệnh fix 2-3 lần. Trích dẫn user: "1 lần nữa yêu cầu audit lại". Note đặc biệt: "không cheat db hay thay đổi các config để đạt đc kết quả".

## Scope (16 tiêu chí gốc — y nguyên audit 2026-05-26)

| Nhóm | Tiêu chí |
|---|---|
| 1. Functional & Correctness | F1 Data Reconciliation / F2 Schema Drift / F3 Event Ordering |
| 2. Stability & Resilience | S1 Failover / S2 Network Flicker / S3 LSN Expire / S4 DLQ |
| 3. Performance & Scalability | P1 Data Lag / P2 Throughput/TPS / P3 Backlog Catch-up / P4 Source DB Overhead |
| 4. Resource Utilization | R1 Memory Leak (soak) / R2 Concurrency & Throttling |
| 5. Metric Monitor | M1 Replication Lag / M2 CPU/Mem / M3 Disk I/O & Network |

## Target Repos
- `data-hub/centralized-data-service` (Go worker plane)
- `data-hub/cdc-cms-service` (Go control plane)
- `data-hub/cdc-cms-web` (TypeScript FE)
- `data-hub/cdc-auth-service` (Go auth plane)

## Prior Audit Reference
- **Audit gốc**: `audit-cdc-qa-process-2026-05-26` → Score 35/64 (54.7%). 4 P0 + 5 P1 + 7 P2 gap.
- **Plan fix**: `plan-cdc-qa-gap-fix-2026-05-27` → 4-phase, target 56/64 (87.5%).
- **Execution claim**: `report_phase_p0_execution_2026-05-27.md` + `report_execute_p1.md` + `report_execute_remaining_gaps_2026-05-27.md`.

## Re-Audit Method
1. Spawn 3 Explore subagent parallel, mỗi agent verify 1 cụm gap (P0 / P1 / P2+NEW).
2. Mỗi agent BẮT BUỘC mở file gốc, đọc thực tế, chứng minh bằng `file_path:line_number`.
3. Cross-verify bằng `go build` + `go vet` + `go test -short` thực tế.
4. Phân loại Verdict: `FIXED` / `PARTIAL` / `FAKE` / `NOT IMPLEMENTED`.
5. Tính lại composite score và delta vs audit gốc.

## Rating Scale (y nguyên audit gốc)
| Level | Nghĩa |
|---|---|
| L0 | Thiếu hoàn toàn |
| L1 | Có dấu vết config/log, chưa có test harness |
| L2 | Code happy-path, thiếu test/metric |
| L3 | Code + 1 test/metric path |
| L4 | Đầy đủ code + test + metric + runbook |

## Constraints
- §7 Full Doc Set 00..10.
- §11 Memory APPEND only.
- §12 Brain Code Prohibition — workspace này KHÔNG sửa code, chỉ audit.
- §14 Pre-flight kiểm tra file vật lý cuối session.
- Không cheat config/DB để đạt PASS.
- Report file vật lý có note "files thay đổi" + "LOC delta" (lấy từ report execution gốc, không tự sửa).
