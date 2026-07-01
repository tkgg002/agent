# Plan - ReconSmokeTables

## Implementation Plan
1. **Research & Requirements Gathering**:
   - Read schema requirements from `/Users/trainguyen/.gemini/antigravity/brain/99e1440b-4c11-4575-aa43-44f2132e4bcb/implementation_plan.md` (lines 188-288, 294-352, 354-438).
   - Read repository patterns from `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go`.
2. **Database Schema & Migrations**:
   - Create migration SQL file at `migrations/dest/002_recon_smoke_tables.sql`.
3. **Go Models**:
   - Create Go model `internal/model/recon/recon_smoke_model.go` representing `SmokeResult` and `CycleSummary`.
4. **Repository**:
   - Create repository `internal/repository/recon/recon_smoke_repo.go` for GORM database operations.
5. **Service Integration**:
   - Update functions `RunTotalOnlyA`, `RunTotalOnlyB`, `CheckAllUnified` in `internal/service/recon/recon_smoke.go` to return GORM models and save to DB.
6. **Server Jobs Executor Integration**:
   - Update `internal/server/server_jobs.go` to parse the new return types and record cycle summary.
7. **Verification & Testing**:
   - Run `go build ./...` and `go vet ./...` to verify code correctness.
