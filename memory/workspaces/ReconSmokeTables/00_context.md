# Context - ReconSmokeTables

## Goal
Implement reconciliation smoke test database persistence in centralized-data-service.

## Scope
1. Migration for `cdc_system.cdc_recon_smoke_result` and `cdc_system.cdc_recon_cycle_summary`.
2. Go model `recon_smoke_model.go` with GORM and JSON tags.
3. Repository `recon_smoke_repo.go` to support GORM Create/Save operations.
4. Service updates in `recon_smoke.go`.
5. Job executor updates in `server_jobs.go`.
6. Go build/vet validation.
