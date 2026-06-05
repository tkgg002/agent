# Requirements — Fix Snapshot source-key + Sync Fields rows_affected miscount

## Origin
User feedback (2026-05-18, sau lazy resolve phase):
> "thằng chó, snapshot thì bị no source route for target table 'export-jobs'... rows_affected: 19 nhưng ko tạo 1 field nào. mày làm kiểu chó gì vâyh"

## 2 Bug song song

### Bug A — Snapshot Now lookup mismatch
- FE `cdc-cms-web/src/pages/TableRegistry.tsx:490` gửi `record.source_table` (e.g., `export-jobs`).
- API `cdc-cms-service/internal/api/reconciliation_handler_tools.go:36-77` pass nguyên `c.Params("table")` → NATS payload `Table=export-jobs`.
- Worker `recon_handler.go` `resolveSourceMongoDSN` gọi `metadata.ResolveTargetRoute("export-jobs")` → nil vì `targetRouteMap` keyed bằng `cfg.TargetTable` (e.g., `sd_export_jobs`).
- → Log: `no source route for target table "export-jobs"`. Snapshot Now luôn fail.

### Bug B — Sync Fields rows_affected miscount
- `command_handler.go` `HandleCreateDefaultColumns` chạy `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` cho từng rule.
- Postgres `ADD COLUMN IF NOT EXISTS` không error khi column đã tồn tại → `Exec` PASS → `columnsAdded++`.
- Khi shadow đã có sẵn full schema (đã sync trước đó), 19 rule đi qua, `columnsAdded=19`, FE thấy success/rows_affected=19 nhưng không có field mới nào thật sự được tạo.

## Functional Requirements

- FR-1 (snapshot): `HandleDebeziumSignal` phải resolve DSN khi `payload.Table` là source name HOẶC target name.
- FR-2 (snapshot): Dùng cùng fallback chain với `resolveTargetTableConfig` (target → sd_+target → source → DB).
- FR-3 (sync fields): `columnsAdded` chỉ tăng khi column thật sự được thêm vào shadow (không có sẵn trước ALTER).
- FR-4 (sync fields): Log thêm metric `columns_already_exist` để debug user-visible 0 fields được tạo.

## Out of Scope

- Đổi shape của FE/API payload (giữ nguyên `source_table` chuỗi).
- Refactor route key model (giữ targetRouteMap keyed by target).

## Definition of Done

- [ ] `resolveSourceMongoDSN(table)` accept source/target name, fail-safe rõ khi cả 2 không match.
- [ ] `HandleCreateDefaultColumns` count chính xác columns thật sự được tạo + log `columns_already_exist`.
- [ ] `go build ./...` PASS.
- [ ] `go vet ./...` PASS.
- [ ] `go test ./internal/handler/... ./internal/server/...` PASS.
- [ ] User test:
  - Snapshot Now `export-jobs` → log `dispatch_path=mongo_lazy_resolve` + `signal_id=<ObjectID>`.
  - Sync Fields với shadow đã đủ field → `rows_affected=0` + `columns_already_exist=19`. Khi mapping_rule có column mới → `rows_affected > 0`.
