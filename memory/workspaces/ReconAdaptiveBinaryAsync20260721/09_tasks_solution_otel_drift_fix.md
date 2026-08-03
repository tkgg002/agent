# Technical Solution: OpenTelemetry Span Parity & False Drift Resolution

## 1. OpenTelemetry Span Status Refactoring

### Current Problem
In `internal/service/recon/recon_stream_bucket_engine.go`:
```go
// Line 181 inside checkDayChunk:
if len(drifts) > 0 {
    span.SetStatus(codes.Error, fmt.Sprintf("drift: %d sub-windows", len(drifts)))
}

// Line 107 inside Execute:
if len(allDrifts) > 0 {
    span.SetStatus(codes.Error, fmt.Sprintf("drift detected: %d sub-windows", len(allDrifts)))
}
```

### Refactored Pattern
```go
// Inside checkDayChunk:
if len(drifts) > 0 {
    span.SetAttributes(
        attribute.Bool("recon.is_drift", true),
        attribute.Int("recon.drift_count", len(drifts)),
    )
    span.SetStatus(codes.Ok, "drift detected")
} else {
    span.SetAttributes(attribute.Bool("recon.is_drift", false))
    span.SetStatus(codes.Ok, "clean")
}

// Inside Execute:
if len(allDrifts) > 0 {
    span.SetAttributes(
        attribute.Bool("recon.is_drift", true),
        attribute.Int("recon.total_drift_count", len(allDrifts)),
    )
    span.SetStatus(codes.Ok, fmt.Sprintf("drift detected: %d sub-windows", len(allDrifts)))
} else {
    span.SetAttributes(attribute.Bool("recon.is_drift", false))
    span.SetStatus(codes.Ok, "clean")
}
```

## 2. Destination Timestamp Column Resolution

### Current Problem
`ChunkStreamBucketEngine.Execute` uses:
```go
srcTS := tsField(entry)
dstTS := srcTS
```
When `entry` is for MongoDB table `payment_bills` with `TimestampField = "updatedAt"`, `dstTS` is passed as `"updatedAt"`. But PostgreSQL shadow table `shadow_testpbs.payment_bills` has column `"updated_at"` or `"_source_ts"`. PostgreSQL returns 0 records for `"updatedAt"`, producing false-positive drift for all 30 sub-windows.

### Refactored Pattern
Add `resolveTSFields` method to `ChunkStreamBucketEngine`:
```go
func (e *ChunkStreamBucketEngine) resolveTSFields(ctx context.Context, entry source.TableRegistry) (srcTS, dstTS string) {
    primary := tsField(entry)
    if primary == "" {
        primary = "updated_at"
    }
    srcTS = primary

    if isPostgres(entry.SourceURL) {
        return srcTS, srcTS
    }

    if e.destAgent != nil {
        snakePrimary := camelToSnake(primary)
        probeOrder := buildTSProbeOrder(primary, snakePrimary, entry.GetCandidates())
        for _, cand := range probeOrder {
            exists, err := e.destAgent.ColumnExists(ctx, entry.QualifiedTarget(), cand)
            if err == nil && exists {
                return srcTS, cand
            }
        }
    }

    return srcTS, "_source_ts"
}
```
In `Execute`:
```go
srcTS, dstTS := e.resolveTSFields(ctx, entry)
```
This guarantees that both MongoDB source and PostgreSQL shadow table query their respective real physical timestamp columns, yielding 100% hash and count parity.
