# Context - FixReconColumnNotExist

## Goal
Fix the GORM execution error `column "updated_at" does not exist (SQLSTATE 42703)` in `recon_dest_query.go:392` which occurs during the reconciliation cycle when querying the destination or shadow database for maximum watermark.

## Scope
1. Address the root cause in `MaxWindowTs` of `internal/service/recon/recon_dest_query.go`.
2. Implement fallback logic to metadata columns `_updated_at` and `_source_ts` if the requested timestamp field does not exist in the destination or shadow Postgres table.
3. Validate overall compilation and run unit tests.
