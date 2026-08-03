# Requirement Spec: OTel Span Error & False-Positive Drift Fix

## 1. Problem Statement
When running reconciliation on real payloads (e.g. `payment_bills` with custom time range `1784018220000` to `1784623020000` — 7 days lookback):
1. **Jaeger Tracing Spans Error (5 Span Errors):** The system recorded 5 span errors on Jaeger trace `0bb92d9e53f68588f73d0501cb91d14d` when drift was detected.
2. **False-Positive Drift (30 Sub-Windows):** 30 sub-windows were reported as drifted even though actual database records in MongoDB and PostgreSQL shadow table were completely aligned.

## 2. Requirements & Acceptance Criteria
- **[R1] OTel Span Status Parity:** Business data drift must NOT mark OpenTelemetry spans with `codes.Error`. Spans for drift detection must have status `codes.Ok` with attributes `recon.is_drift = true` and `recon.total_drift_count = N`. `codes.Error` must only be set when genuine system/query errors occur (`err != nil`).
- **[R2] Timestamp Field Resolution:** `ChunkStreamBucketEngine` must resolve the destination Postgres shadow table timestamp column dynamically using `destAgent.ColumnExists` (probing primary, `camelToSnake`, candidates, and fallback to `_source_ts`), preventing column mismatch queries.
- **[R3] Zero False Drift:** Matching records between MongoDB source and PostgreSQL shadow table must return 0 drift windows and status `clean`.
- **[R4] Unit & Integration Tests:** All unit tests in `service/recon` and `handler/recon` must pass 100%.
