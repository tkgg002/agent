# Report — Worker-side `connection_overrides` (URI overlay theo `connection_code`)

**Phase**: fe-api-worker-action-tracer-2026-05-18 / connection_overrides
**Author**: Claude Code (Muscle, claude-opus-4-7)
**Date**: 2026-05-19

## Mục tiêu

User feedback: `cdc_system.connection_registry` chứa URI do admin nhập qua CMS UI cho nhiều loại source (mongo server, mongo docker, postgres, mysql). Worker dev không reach được hostname admin nhập (docker/VPN). Cần solution: worker override URI theo `connection_code` mà không sửa DB và áp dụng cho MỌI service worker (recon, source ingest, debezium signal, scan fields, ...).

## Cách giải quyết

3 site bypass trong codebase được xác định bởi Explore agent. Overlay áp dụng tại CẢ 3:

| Site | File:line | Function | Trigger |
|---|---|---|---|
| A | `internal/service/metadata_registry_service.go:341` | `GetSourceDSN(code)` | `ProvisioningStepHandler.pickSourceDSN` (shadow.bind PG/MySQL) |
| B | `internal/handler/command_handler.go:311` | `scanFieldsMongoSource` | NATS `cdc.cmd.scan-fields`, `cdc.cmd.create-default-columns` → Mongo fallback |
| C | `internal/handler/recon_handler.go:471` | `resolveSourceMongoDSN(table)` | NATS `cdc.cmd.debezium-signal`, `cdc.cmd.debezium-snapshot` (Snapshot Now) |

Logic: ngay sau khi load `model.ConnectionRegistry` row, gọi `ApplyConnectionOverride(&conn, overrides, logger)`. Hit → return override URI; miss → fallback nguyên path cũ.

## File thay đổi (7 file)

### 1. `config/config.go`
- **Thêm field** `AppConfig.ConnectionOverrides map[string]string` (mapstructure tag `connectionOverrides`).
- **Env scanner** trong `applyEnvOverrides`: prefix `CONNECTION_OVERRIDE_<CODE>` → upsert vào map.

### 2. `config/config-local.yml`
- Thêm block ví dụ:
```yaml
connectionOverrides:
  goopay: "mongodb://localhost:17017/?replicaSet=rs0&directConnection=true"
  # goopay1: "..."
  # default_shadow: "..."
```

### 3. `internal/service/connection_overrides.go` (NEW)
- `ApplyConnectionOverride(conn, overrides, logger) (uri, ok)` — case-insensitive lookup theo `conn.ConnectionCode`. Log 1 dòng INFO khi hit (audit trail).
- `NormalizeConnectionOverrides(in) map` — lowercase keys, trim values, drop empties.

### 4. `internal/service/metadata_registry_service.go`
- Struct thêm `connectionOverrides map[string]string`.
- Ctor `NewMetadataRegistryService` nhận thêm param `connectionOverrides`, normalize qua helper.
- `GetSourceDSN`: check overlay đầu tiên (sau khi load conn, trước tryPlainDSN/env/AES chain).

### 5. `internal/handler/command_handler.go`
- Struct thêm `connectionOverrides map[string]string`.
- Setter `SetConnectionOverrides(map)` — normalize qua helper.
- `scanFieldsMongoSource`: check overlay sau `db.First(&conn, ...)`. Hit → assign dsn từ override; miss → giữ nguyên hostRaw/port assembly.

### 6. `internal/handler/recon_handler.go`
- Struct thêm `connectionOverrides map[string]string`.
- Builder `WithConnectionOverrides(map) *ReconHandler` (fluent style, đồng nhất với `WithHealer`/`WithMetadataRegistry`/...).
- `resolveSourceMongoDSN`: check overlay sau `db.First(&conn, connID)`. Hit → return uri; miss → fallback.

### 7. `internal/server/worker_server.go`
- Normalize overrides 1 lần ở top services section.
- Log `connection overrides loaded` với danh sách codes (audit).
- Pass overrides vào `NewMetadataRegistryService` constructor.
- `cmdHandler.SetConnectionOverrides(connectionOverrides)`.
- Cả 2 nhánh ReconHandler (`reconCore != nil` + `signalOnlyHandler` lazy resolve mode) gọi `.WithConnectionOverrides(connectionOverrides)`.

## Verify (kết quả thực tế)

| Bước | Command | Kết quả |
|---|---|---|
| Build | `go build ./...` | EXIT=0 |
| Vet | `go vet ./...` | EXIT=0 |
| Test handler+service | `go test -count=1 ./internal/handler/... ./internal/service/...` | PASS (handler 3.780s, service 1.369s) |
| Test config | `go test -count=1 ./config/...` | PASS (0.215s) |
| Runtime probe | YAML `goopay` + env `CONNECTION_OVERRIDE_GOOPAY1=mongodb://override-via-env:27017/` | Map có cả 2 keys: `goopay`, `goopay1` — đúng |

## Hành động user cần làm (verify end-to-end)

1. **Confirm `connection_registry`** có row `connection_code='goopay'` (engine=mongodb).
   ```sql
   SELECT id, connection_code, engine_type, host, port FROM cdc_system.connection_registry WHERE connection_code='goopay';
   ```
2. **Restart worker**:
   - Ctrl-C tty003 (cũ).
   - `go run cmd/worker/main.go`.
3. **Worker startup expected log**:
   - `connection overrides loaded connection_codes=[goopay]`
4. **Click Snapshot Now** cho `export-jobs` từ FE TableRegistry:
   - Worker log expected:
     ```
     connection override applied connection_code=goopay engine=mongodb origin=config
     debezium signal inserted dispatch_path=mongo_lazy_resolve signal_id=<ObjectID>
     ```
   - KHÔNG còn `no such host: gpay-mongo` hoặc `no source route for target table "export-jobs"`.
5. **Optional** — nếu mapping_rule_v2 cho `sd_export_jobs` có column mới, click Sync Fields → expected `columns_added=N` (Site B hoặc Debezium HTTP path).

## Out of scope (đã document trong requirements)

- Đổi schema `connection_registry`.
- Đổi FE/CMS API.
- Override theo `id` (chỉ theo `connection_code` cho semantic stability).
- Override `cfg.ShadowDB`/`cfg.MasterDB` static config (đã có sẵn nguyên path config-driven, không từ `connection_registry`).

## Risk + Mitigation

| Risk | Mitigation |
|---|---|
| Override leak vào prod | Empty map = no-op; env var pattern `CONNECTION_OVERRIDE_*` rõ ràng; log INFO mỗi hit để audit |
| Caller-Resolver Wiring miss site nào | Explore agent enumerate đã liệt kê đủ 3 site bypass + Site A canonical. Mọi `connection_registry` JOIN khác chỉ đọc `connection_code`/`id` cho FK — không build DSN |
| ConnectionCode trùng cross-engine | Helper pass-through string nên không corrupt — admin tự đảm bảo URI scheme phù hợp engine |

## Skills sử dụng

- Explore agent (very thorough) → enumerate 3 site bypass + canonical resolver
- Edit/Read tools → modify 6 file Go + 1 file YAML
- Bash tool → go build/vet/test + runtime config probe
- TaskCreate/TaskUpdate → 9 task tracking
- File system (Workspace) → Full Doc Set + APPEND `05_progress.md`

## Files changed checklist (Pre-flight Governance)

- [x] `config/config.go` (modified)
- [x] `config/config-local.yml` (modified)
- [x] `internal/service/connection_overrides.go` (created)
- [x] `internal/service/metadata_registry_service.go` (modified)
- [x] `internal/handler/command_handler.go` (modified)
- [x] `internal/handler/recon_handler.go` (modified)
- [x] `internal/server/worker_server.go` (modified)
- [x] Workspace docs: `01_requirements_connection_overrides.md`, `02_plan_connection_overrides.md`, `08_tasks_connection_overrides.md`, `09_tasks_solution_connection_overrides.md`
- [x] `05_progress.md` APPENDED
- [x] `report_connection_overrides.md` (this file)
- [x] Global lesson APPENDED (xem `agent/memory/global/lessons.md`)
