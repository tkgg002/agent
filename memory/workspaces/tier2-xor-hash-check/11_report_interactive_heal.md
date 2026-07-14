# Báo cáo thay đổi — Backend Interactive Heal (2026-07-03)

## Tổng quan
Tách biệt luồng Đối soát (Recon Check) và Thực thi (Heal/Prune). Tạo command mới `ExecuteHealCommand` với 3 checkboxes granular. Deprecate `ReconHealCommand` cũ.

## Files đã thay đổi

### Gateway (`cdc-cms-service`) — 9 files

#### [MODIFY] reconciliation_report.go (model)
- **Đường dẫn**: `internal/model/recon/reconciliation_report.go`
- **Thay đổi**: +6 fields (+7 dòng)
- **Chi tiết**: Thêm `HealedMismatchedCount`, `HealedMismatchedDurationMs`, `HealedMissingDestCount`, `HealedMissingDestDurationMs`, `PrunedMissingSrcCount`, `PrunedMissingSrcDurationMs` — thống kê granular cho từng checkbox heal.

#### [MODIFY] recon_async.go (command)
- **Đường dẫn**: `internal/app/commands/recon/recon_async.go`
- **Thay đổi**: +20 dòng
- **Chi tiết**: Thêm `ExecuteHealCommand` struct với `ReportIDs []uint64`, `HealMismatched`, `HealMissingDest`, `PruneMissingSrc` flags. `Type() = "execute-heal"`, `Validate()` yêu cầu `report_ids`.

#### [MODIFY] recon_reader.go (interface)
- **Đường dẫn**: `internal/app/queries/recon/recon_reader.go`
- **Thay đổi**: +5 dòng
- **Chi tiết**: Thêm `ListUnhealedReports(ctx, table, shadowSchema string) ([]ReconciliationReport, error)` vào interface `ReconReader`.

#### [NEW] list_unhealed_reports.go (CQRS query handler)
- **Đường dẫn**: `internal/app/queries/recon/list_unhealed_reports.go`
- **Thay đổi**: ~42 dòng (file mới)
- **Chi tiết**: `ListUnhealedReportsQuery` + `ListUnhealedReportsHandler` theo đúng pattern `ListLatestReportsHandler`.

#### [MODIFY] recon_read_repo_gorm.go (persistence)
- **Đường dẫn**: `internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Thay đổi**: +15 dòng
- **Chi tiết**: Implement `ListUnhealedReports`. Query: `WHERE (shadow_table = ? OR master_table = ?) AND healed_at IS NULL AND (missing_count > 0 OR stale_count > 0 OR orphan_count > 0)`. Dùng Migration 085 key (shadow_table/master_table) thay vì target_table.

#### [MODIFY] reconciliation_handler.go (API handler struct)
- **Đường dẫn**: `internal/api/recon/reconciliation_handler.go`
- **Thay đổi**: +3 dòng
- **Chi tiết**: Thêm field `listUnhealedQ *recon.ListUnhealedReportsHandler`, thêm param constructor, thêm vào struct init.

#### [NEW] reconciliation_handler_execute_heal.go (API handler methods)
- **Đường dẫn**: `internal/api/recon/reconciliation_handler_execute_heal.go`
- **Thay đổi**: ~62 dòng (file mới)
- **Chi tiết**: `TriggerExecuteHeal` (POST, destructive) dispatch `ExecuteHealCommand` qua NATS. `GetUnhealedReports` (GET) trả danh sách report chưa heal.

#### [MODIFY] server.go (wiring)
- **Đường dẫn**: `internal/server/server.go`
- **Thay đổi**: +3 dòng
- **Chi tiết**: `RegisterSubject("execute-heal", "cdc.cmd.execute-heal")`, instantiate `listUnhealedReportsH`, update `NewReconciliationHandler` call.

#### [MODIFY] router.go (routing)
- **Đường dẫn**: `internal/router/router.go`
- **Thay đổi**: +2 dòng
- **Chi tiết**: `registerDestructive("/reconciliation/execute-heal", ...)` + `dual("GET", shared, "/reconciliation/report/:table/unhealed", ...)`.

---

### Worker (`centralized-data-service`) — 4 files

#### [MODIFY] reconciliation_report.go (model)
- **Đường dẫn**: `internal/model/recon/reconciliation_report.go`
- **Thay đổi**: +8 dòng
- **Chi tiết**: Cùng 6 fields như gateway, giữ nguyên type `*int64` cho SourceCount (khác gateway dùng `int64`).

#### [NEW] recon_execute_heal.go (NATS handler)
- **Đường dẫn**: `internal/handler/recon/recon_execute_heal.go`
- **Thay đổi**: ~261 dòng (file mới)
- **Chi tiết**: `HandleExecuteHeal` (subscribe `cdc.cmd.execute-heal`). `executeHeal` loop qua report IDs, phân luồng theo segment. `executeHealSegA` dùng `FetchAndWriteByIDs`. `executeHealSegB` dùng `mapGpayToSourceIDs` + `publishTransmuteChunked` (reuse pattern healSegmentB). Fallback flat array cho stale_ids Segment B.

#### [MODIFY] server_setup.go (subscription)
- **Đường dẫn**: `internal/server/server_setup.go`
- **Thay đổi**: +1 dòng
- **Chi tiết**: `natsClient.Conn.Subscribe("cdc.cmd.execute-heal", reconHandler.HandleExecuteHeal)`.

#### [MODIFY] recon_handler_run.go (deprecation)
- **Đường dẫn**: `internal/handler/recon/recon_handler_run.go`
- **Thay đổi**: +2 dòng
- **Chi tiết**: Thêm `[DEPRECATED]` warning log ở đầu `HandleReconHeal`.

---

### DB Migration — 1 file

#### [NEW] 088_recon_interactive_heal_stats.sql
- **Đường dẫn**: `migrations/schema/recon_dlq/088_recon_interactive_heal_stats.sql`
- **Thay đổi**: 17 dòng (file mới)
- **Chi tiết**: `ALTER TABLE cdc_system.cdc_reconciliation_report ADD COLUMN IF NOT EXISTS ...` — 6 cột INT DEFAULT 0.

## Tổng kết
- **Files thay đổi**: 14 (9 gateway + 4 worker + 1 migration)
- **Files mới**: 4 (list_unhealed_reports.go, reconciliation_handler_execute_heal.go, recon_execute_heal.go, 088_*.sql)
- **Dòng code thêm**: ~400 dòng
- **Build**: Gateway ✅ PASS | Worker ✅ PASS
