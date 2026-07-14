# Danh sách Task — Backend Interactive Heal (ĐÃ HOÀN THÀNH)

## Phase Backend — 13 files ✅ BUILD PASS

### Gateway (`cdc-cms-service`) — 9 files ✅
- [x] `model/recon/reconciliation_report.go` — +6 fields thống kê granular
- [x] `commands/recon/recon_async.go` — +`ExecuteHealCommand` struct (Type, Validate)
- [x] `queries/recon/recon_reader.go` — +`ListUnhealedReports` vào interface ReconReader
- [x] `queries/recon/list_unhealed_reports.go` — [NEW] CQRS Query Handler (theo pattern ListLatestReportsHandler)
- [x] `persistence/recon/recon_read_repo_gorm.go` — Implement `ListUnhealedReports` (shadow_table OR master_table)
- [x] `api/recon/reconciliation_handler.go` — Inject `listUnhealedQ` vào struct + constructor
- [x] `api/recon/reconciliation_handler_execute_heal.go` — [NEW] `TriggerExecuteHeal` (POST) + `GetUnhealedReports` (GET)
- [x] `server/server.go` — `RegisterSubject("execute-heal")` + instantiate query handler + update constructor
- [x] `router/router.go` — `registerDestructive` + `dual("GET")` routes

### Worker (`centralized-data-service`) — 4 files ✅
- [x] `model/recon/reconciliation_report.go` — +6 fields thống kê granular
- [x] `handler/recon/recon_execute_heal.go` — [NEW] `HandleExecuteHeal` + executeHealSegA/B + publishTransmuteChunked
- [x] `server/server_setup.go` — Subscribe `cdc.cmd.execute-heal`
- [x] `handler/recon/recon_handler_run.go` — +Deprecation warning trên `HandleReconHeal`

### DB Migration ✅
- [x] `migrations/schema/recon_dlq/088_recon_interactive_heal_stats.sql` — 6 cột mới

## Verification ✅
- [x] Gateway `go build ./...` — PASS
- [x] Worker `go build ./internal/...` — PASS
