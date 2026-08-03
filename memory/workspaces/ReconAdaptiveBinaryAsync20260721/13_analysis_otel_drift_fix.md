# Architectural Analysis: OpenTelemetry Trace Semantics & Cross-Database Column Resolution

## 1. Trace Semantics: Business State vs. System Error
OpenTelemetry standard dictates that Span Status Code `codes.Error` should be reserved for system failures (unhandled exceptions, database disconnections, syntax errors). When a reconciliation engine identifies a data mismatch (drift):
- The reconciliation process itself executed **successfully**.
- Marking the span as `codes.Error` pollutes observability metrics (SLOs, error rates, alerting spikes on Jaeger/Grafana).
- **Correct Pattern:** Set span status to `codes.Ok` and annotate business semantics using span attributes (`recon.is_drift = true`, `recon.total_drift_count = N`).

## 2. Cross-Database Timestamp Resolution
When source data resides in MongoDB (e.g. `payment_bills`) with camelCase timestamp fields (`updatedAt`) and destination data is stored in PostgreSQL shadow tables (`shadow_testpbs.payment_bills`) with snake_case columns (`updated_at` / `_source_ts`):
- Passing `dstTS = srcTS` blindly results in PostgreSQL executing `SELECT ... WHERE "updatedAt" >= ...`.
- PostgreSQL either fails or returns 0 rows due to column absence.
- **Correct Pattern:** Probing `destAgent.ColumnExists` against `[primary, camelToSnake(primary), candidates, "_source_ts"]` resolves the true physical column name, ensuring identical timestamp windowing on both sides.
