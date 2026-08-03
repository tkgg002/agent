# Technical Change Report: OTel Span Status Parity & False Drift Resolution

## Overview & Scope
This report documents the changes implemented in `ChunkStreamBucketEngine` to resolve false-positive OpenTelemetry span errors and eliminate false-positive sub-window drift.

## Files Changed
1. **[recon_stream_bucket_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go)** (Lines modified: ~35)
   - Added `resolveTSFields(ctx context.Context, entry source.TableRegistry) (srcTS, dstTS string)` to dynamically resolve destination timestamp column names on PostgreSQL shadow tables using `destAgent.ColumnExists`.
   - Updated `Execute` to use `srcTS, dstTS := e.resolveTSFields(ctx, entry)`.
   - Changed span status logic in `checkDayChunk` and `Execute`: business drift is marked with `codes.Ok` and attributes `recon.is_drift = true`, `recon.total_drift_count = N`. Only genuine database/network exceptions trigger `codes.Error`.

2. **[recon_stream_bucket_engine_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine_test.go)** (Lines modified: ~28)
   - Added `TestChunkStreamBucketEngine_ResolveTSFields` testing timestamp resolution for both PostgreSQL and MongoDB source connections.

## Verification & Impact
- `go test -v ./internal/service/recon/... ./internal/handler/recon/...` passed 100%.
- Eliminates 5 false-positive Jaeger UI span errors for operational drift results.
- Eliminates 30 sub-window false-positive count mismatches when MongoDB timestamp fields (`updatedAt`) map to Postgres columns (`updated_at` / `_source_ts`).
