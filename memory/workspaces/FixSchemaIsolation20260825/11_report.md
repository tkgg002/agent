# 11_report.md — Báo Cáo Thay Đổi Toàn Trình (E2E Schema Isolation)

## 1. Các File Đã Thay Đổi:
- **`cdc-cms-service`**:
  - `internal/app/commands/recon/recon_check.go`: Thêm `ShadowSchema`, `ShadowTable`, `SourceDatabase`, `SourceTable` vào struct `ReconCheckCommand`.
  - `internal/app/commands/recon/recon_async.go`: Thêm schema fields vào `ReconHealCommand`, `ExecuteHealCommand`.
  - `internal/api/recon/reconciliation_handler_commands.go`: Ghép `scope.ShadowSchema` vào `table` và truyền đầy đủ metadata trong `TriggerCheck`.
  - `internal/api/recon/reconciliation_handler_heal.go`: Ghép `scope.ShadowSchema` và truyền metadata trong `TriggerHeal`.
  - `internal/api/recon/reconciliation_handler_execute_heal.go`: Ghép `req.ShadowSchema` và truyền metadata trong `TriggerExecuteHeal`.
- **`centralized-data-service`**:
  - `internal/handler/recon/recon_check_handler.go`: Bổ sung metadata vào `reconCheckPayload` và `ReconJobCreatedEvent`.
  - `internal/handler/recon/recon_check_heal_handler.go`: Bổ sung metadata vào `reconHealPayload` và chuẩn hóa `targetTable = shadow_schema.table`.
  - `internal/handler/recon/recon_base_handler.go`: Xóa bỏ các nhánh fallback đoán mò sang `pureTable`.
  - `internal/service/recon/recon_job_worker.go`: Lookup tường minh `shadow_schema.target_table`, xóa bỏ fallback.
  - `internal/service/recon/recon_smoke.go`: Gọi `GetByTargetTableAndSchema`.
  - `internal/service/recon/recon_tier_b.go`: Gọi `GetByTargetTableAndSchema`.
  - `internal/service/metadata/helpers.go`: Xóa bỏ fallback sang `pureTable`.

## 2. Kết Quả:
- Cả 2 service (`cdc-cms-service` server và `centralized-data-service` worker) compile thành công 100%.
- Luồng End-to-End: `UI (CMS)` $\rightarrow$ `API (cdc-cms-service)` $\rightarrow$ `NATS` $\rightarrow$ `Worker (centralized-data-service)` được bảo đảm mang đầy đủ `shadow_schema` từ đầu đến cuối.
