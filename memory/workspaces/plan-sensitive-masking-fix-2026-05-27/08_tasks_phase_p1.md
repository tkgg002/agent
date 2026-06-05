# 08_tasks_phase_p1 — Checklist Muscle Phase P1 (Worker Integration)

> Reference: `03_implementation_phase_p1.md`. Effort 10h. Yêu cầu P0 done.

## M-6 — DynamicMapper integration (3h)
- [ ] Sửa `centralized-data-service/internal/service/dynamic_mapper.go`:
  - Line ~67, ~114: thay `dm.maskRawData(targetTable, rawData)` → `dm.maskingSvc.MaskTableData(ctx, evt.ID, evt.SourceCode, targetTable, rawData)`.
  - Xóa helper `maskRawData()` cũ (lines 123-127) — không còn cần.
- [ ] Cập nhật mock + test `dynamic_mapper_test.go`.
- [ ] Verify: `go test ./internal/service -run TestDynamicMapper -v` PASS.

## M-7 — BatchBuffer + ReconHeal + KafkaConsumer align (3h)
- [ ] Sửa `internal/handler/batch_buffer.go`:
  - Line 374-383: `sanitizeRawData()` chạy qua `MaskTableData` thay vì `MaskJSONPayload("***")`.
  - Invalid JSON → return `[]byte("null")` không `"***"`.
- [ ] Sửa `internal/service/recon_heal.go`:
  - Line 771-776: cập nhật test expectation (giá trị mới theo strategy).
- [ ] Sửa `internal/handler/kafka_consumer.go`:
  - Line 1124-1125: cập nhật assert sau khi dispatch qua strategy.
- [ ] Test: `go test ./internal/handler -v && go test ./internal/service -v` PASS.

## M-8 — Audit log writer (2h)
- [ ] Tạo NEW `internal/service/masking/audit_writer.go`:
  - Struct `AuditWriter` với `db`, `ch`, `sampleRate`, `batchSize=500`, `flushEvery=5s`.
  - Method `Run(ctx)` — select case audit channel + ticker.
  - Sample rate qua `rand.Float64()`.
- [ ] Config knob `masking.auditSampleRate` (default 0.01).
- [ ] Wire trong `worker_server.go`: tạo channel buffered, start AuditWriter goroutine.
- [ ] Verify: set rate=1.0, run 100 event, `SELECT COUNT(*) FROM cdc_system.mask_audit_log` ≈ N_field × 100.

## M-9 — E2E integration test (2h)
- [ ] Tạo NEW `internal/service/masking_e2e_test.go` với build tag `//go:build integration`:
  - testcontainers postgres:16-alpine.
  - Apply migration 015_mask_strategy.sql.
  - Seed mapping_rule với 3 strategy (DROP password, HASH_HMAC cccd, PARTIAL phone).
  - Set env `MASKING_HMAC_KEY_V1`.
  - Call `MaskTableData` với data thực + assert đầu ra cho từng strategy.
  - Verify `mask_audit_log` có ≥ 3 record sau flush.
- [ ] Verify: `go test -tags=integration ./internal/service -run TestMaskingE2E -v` PASS.

## Post-phase
- [ ] Build + vet + test all PASS.
- [ ] Verify shadow PG sau khi worker chạy: không có row `_raw_data::text LIKE '%"***"%'`.
- [ ] /security-agent scan PASS.
- [ ] APPEND `05_progress.md`.
