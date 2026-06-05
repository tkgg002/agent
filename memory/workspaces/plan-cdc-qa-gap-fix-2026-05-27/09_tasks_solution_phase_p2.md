# 09_tasks_solution_phase_p2 — Hồ sơ giải pháp P2

## G-10: Tier3 off-peak config
- **Root cause**: Off-peak window hardcoded 02-05 trong `selectTier3` → không phù hợp đa region.
- **Solution**: Đưa vào ReconCoreConfig với default unchanged.
- **Trade-off**: Tăng complexity nhỏ, đổi flexibility.

## G-11: Batches counter
- **Root cause**: Thiếu metric đo throughput batch flush → khó tính TPS chính xác.
- **Solution**: CounterVec label (shadow_db, table, status).
- **Lý do 3 label**: Đủ phân tích, không cardinality explosion.

## G-12: Adaptive batch
- **Root cause**: Static batch size không tận dụng burst capacity → consumer lag tăng khi spike.
- **Solution**: Monitor lag, tăng size 2x khi vượt threshold; revert khi normal.
- **Lý do cap maxMultiplier=4**: Quá lớn → memory pressure + transaction time out.
- **Anti-thrashing**: `lastAdjust` 30s window tránh oscillate.

## G-13: Per-source pool semaphore
- **Root cause**: 1 source spike (large source backfill) → chiếm hết pool, source khác starve.
- **Solution**: Per-source semaphore cap concurrent acquire.
- **Lý do semaphore vs separate pool**: ADR-006 (share resource, đơn giản config).
- **Metric saturation**: Operator nhìn được source nào đang full.

## G-14: Runbooks
- **Root cause**: Alert fire nhưng on-call không biết flow xử lý → MTTR cao.
- **Solution**: 4 runbook critical paths + link `runbook_url` trong alert annotation.
- **Lý do markdown vs Confluence**: Code-as-doc, đi cùng repo, không phụ thuộc external.

## G-15: Chaos network
- **Root cause**: Chưa validate network flicker scenario → assumption resilience không có evidence.
- **Solution**: iptables DROP 10 phút trên staging.
- **Acceptance định lượng**: `AFTER_LAG < 2x BEFORE_LAG` sau 1 phút catchup.

## G-16: k6 load test
- **Root cause**: Chưa có baseline performance @1000 TPS → không phát hiện regression.
- **Solution**: k6 script với threshold P99 < 5s, chạy weekly CI.
- **Lý do k6 vs Locust**: Native JS, threshold built-in, dễ CI integrate.

## Tổng impact P2
- Score: +5 → 56/64 (87.5%).
- Criteria cover: 1.1 Reconcile, 3.2 TPS, 3.3 Backlog, 4.2 Concurrency, 2.3 LSN runbook, 2.2 Network, 3.1 Data Lag.
