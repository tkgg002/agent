# Architecture Decisions - ReconSmokeTables

## ADR 001: Separation of Smoke Test Results and Reconciliation Reports
- **Context**: The existing smoke tests (O(1) count checks) were using the same `ReconciliationReport` model but without persistence. The implementation plan requires separate database tables for smoke test results (`cdc_recon_smoke_result`) and cycle summaries (`cdc_recon_cycle_summary`).
- **Decision**: Created the dedicated structs `SmokeResult` and `CycleSummary` under package `recon` (Go models). Added a repository `ReconSmokeRepo` to encapsulate GORM operations. Updated service layers to return and persist `SmokeResult`/`CycleSummary` records.
- **Consequences**:
  - The model `ReconciliationReport` remains unchanged for detailed tier scans.
  - Smoke tests now persist results explicitly with 3-layer information, simplifying dashboarding.
