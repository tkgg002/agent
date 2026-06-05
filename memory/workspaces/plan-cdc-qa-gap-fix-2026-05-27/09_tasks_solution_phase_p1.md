# 09_tasks_solution_phase_p1 — Hồ sơ giải pháp P1

## G-5: Failover smoke test
- **Root cause**: Chưa có test chứng minh "worker crash → restart không mất event không dup" — chỉ có unit test cho logic OCC.
- **Solution**: Bash script E2E + CI workflow.
- **Lý do bash + CI**:
  - Bash gần thực tế ops hơn Go test.
  - CI weekly tự động chống regression.
- **Định lượng**: 10k document là sweet spot (đủ để gặp scenario, không quá tải runner).

## G-6: WAL slot alert
- **Root cause**: PG replication slot expire silently → CDC stop nhưng không alert.
- **Solution**: 2 alert (lag > 1GB; inactive > 5m) + postgres-exporter deploy + runbook.
- **Lý do 1GB threshold**:
  - Dưới 1GB là noise normal.
  - Trên 1GB cần action trong 10 phút trước khi disk full.
- **Runbook**: Operator có sẵn flow drop+recreate slot khi cần.

## G-7: pprof + goleak
- **Root cause**: Service không có pprof endpoint → khi prod gặp memory leak/goroutine leak không debug được. Test không phát hiện leak ở dev.
- **Solution**: pprof endpoint (gated by config) + goleak.VerifyTestMain ở 3 critical package.
- **Lý do gate pprof bằng config**:
  - Pprof expose thông tin runtime nhạy cảm.
  - Default off, enable khi cần debug.
- **IgnoreTopFunction**: kafka-go + otel batch processor có background goroutine intended, không phải leak.

## G-8: Ordering test
- **Root cause**: OCC logic (`_source_ts older → reject`) có ở code path nhưng không có test case explicit.
- **Solution**: 2 test scenario — older ts reject + hash tiebreaker.
- **Lý do test riêng thay vì gộp vào existing**:
  - Tách biệt giúp dễ debug khi regress.
  - Naming explicit theo BDD style.

## G-9: Drift E2E test (testcontainers)
- **Root cause**: ApproveSchemaProposal command có unit test nhưng chưa có E2E chứng minh chuỗi: command → ALTER TABLE → mapping_rule → NATS publish.
- **Solution**: testcontainers postgres + nats + 8-step assertion flow.
- **Lý do testcontainers vs in-memory**:
  - Mapping_rule + ALTER TABLE phải chạy thật trên Postgres (regex, JSON column).
  - In-memory không hỗ trợ DDL.
- **Build tag `integration`**: tránh tốn time CI khi chạy unit test thường.

## Tổng impact P1
- Score: +7 → 51/64 (79.7%).
- Criteria cover: 1.2 Failover, 2.1 WAL slot, 4.1 Memory leak, 1.3 Event ordering, 1.4 Schema drift E2E. (Chi tiết `10_gap_analysis.md`.)
