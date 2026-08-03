# Implementation Plan: OTel Span Status Refactoring & False Drift Elimination

## Summary
This plan details the technical steps to refactor `ChunkStreamBucketEngine` to:
1. Fix false-positive OpenTelemetry span errors on Jaeger UI by marking business data drifts with `codes.Ok` and reserve `codes.Error` strictly for DB/system execution failures.
2. Fix false-positive sub-window drift by adding dynamic Postgres shadow table timestamp field resolution (`destAgent.ColumnExists`).

## Proposed Code Changes

### Component: Service Recon Engine
#### [MODIFY] [recon_stream_bucket_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go)
- Add `resolveTSFields(ctx context.Context, entry source.TableRegistry) (srcTS, dstTS string)` method.
- Update `Execute` to use `srcTS, dstTS := e.resolveTSFields(ctx, entry)`.
- Update OTel span status calls in `Execute` and `checkDayChunk`:
  - `codes.Ok` on clean / drift completion.
  - `codes.Error` only on `srcErr != nil` or `dstErr != nil`.

#### [MODIFY] [recon_stream_bucket_engine_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine_test.go)
- Add unit test `TestChunkStreamBucketEngine_OTelSpanStatus` verifying `codes.Ok` status when drift occurs.
- Add unit test `TestChunkStreamBucketEngine_ResolveTSFields` verifying timestamp field resolution.

## Verification Plan

### Automated Tests
```bash
go test -v ./internal/service/recon/...
go test -v ./internal/handler/recon/...
```

### Manual Verification
- Execute payload test against database using HTTP POST `/api/reconciliation/check?type_recon=hash_window`.
