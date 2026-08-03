# Tasks: OTel Span Error & False-Positive Drift Fix

- [x] Task 1: Create 01_requirements_otel_drift_fix.md and update 05_progress.md with Root Cause Analysis.
- [x] Task 2: Implement dynamic destination timestamp resolution `resolveTSFields` in `ChunkStreamBucketEngine`.
- [x] Task 3: Refactor OpenTelemetry span status logic in `checkDayChunk` and `Execute` of `ChunkStreamBucketEngine` (use `codes.Ok` for business drift, reserve `codes.Error` for system exceptions).
- [x] Task 4: Add unit tests in `recon_stream_bucket_engine_test.go` verifying span status parity and column name resolution.
- [x] Task 5: Run full test suite `go test -v ./internal/service/recon/...` and `go test -v ./internal/handler/recon/...`.
- [x] Task 6: Create `11_report_otel_drift_fix.md`, `12_implementation_plan_otel_drift_fix.md`, `13_analysis_otel_drift_fix.md`, `14_walkthrough_otel_drift_fix.md`.
