# Requirements — Worker-side `connection_overrides` (URI overlay theo connection_code)

## Origin

User feedback (2026-05-18, sau khi fix Snapshot lazy-resolve + Sync Fields):
> "cdc_system.connection_registry của tôi là 1 bảng chứa rất nhiều connect, server db mongo, docker db mongo, db posgres, mýql ... tìm giải phap. ko thể fix kiểu A đc. vì db nguồn là admin tự nhập vào"
>
> "ok, làm đi và áp dụng overlay cho mọi service worker (recon, source ingest, …)"

Constraints user đã set:
- Đọc lesson + GEMINI.md trước.
- Chỉ làm đúng yêu cầu.
- Không cheat DB/config để "fake" success.
- Plan phải rõ ràng + code demo cụ thể.
- Report dựa trên kết quả thực tế, list đầy đủ file thay đổi.
- Verify services hoạt động trước khi báo done.
- Output 1 file `report_*.md` ghi lại tất cả changes.

## Bối cảnh kỹ thuật

`cdc_system.connection_registry` là bảng do admin nhập trực tiếp qua CMS UI cho mọi loại source/shadow/master (mongo, postgres, mysql). Column `host` chứa **FULL URI** (`mongodb://gpay-mongo:27017/?replicaSet=rs0`) hoặc **bare host** (`mongo-host`) + `port` riêng. Worker dev chạy local → không phải lúc nào cũng reach được hostname admin nhập (docker network, VPN).

Worker hiện có 3 site translate row → URI:

| # | File:line | Function | Kiểu lookup |
|---|---|---|---|
| A | `internal/service/metadata_registry_service.go:331` | `GetSourceDSN(connectionCode)` | `ConnectionRegistryRepo.GetByCode(code)` |
| B | `internal/handler/command_handler.go:277` | `scanFieldsMongoSource` | `db.First(&conn, sourceConnectionID)` |
| C | `internal/handler/recon_handler.go:439` | `resolveSourceMongoDSN(table)` | `db.First(&conn, connID)` |

Site A gọi từ: `ProvisioningStepHandler.pickSourceDSN` (shadow.bind PG / MySQL).
Site B gọi từ: NATS `cdc.cmd.scan-fields`, `cdc.cmd.create-default-columns` → fallback Mongo path.
Site C gọi từ: NATS `cdc.cmd.debezium-signal`, `cdc.cmd.debezium-snapshot` (Snapshot Now FE button).

Mọi site khác trong codebase đọc `connection_registry` chỉ để lấy `connection_code`/`id` cho JOIN/FK (transmuter, master DDL, admin source register, etc.) — KHÔNG build DSN → không cần overlay.

## Functional Requirements

- **FR-1** Mỗi `connection_registry.connection_code` (string, e.g., `"goopay"`, `"goopay1"`, `"default_shadow"`) là KEY của overlay map. Worker đọc URI override từ config (`connectionOverrides:`), KHÔNG đụng DB.
- **FR-2** Overlay áp dụng ở CẢ 3 site A/B/C. Override miss → fallback đúng nguyên behavior cũ (DSN từ secret_ref / host+port).
- **FR-3** Empty override map (production) → identical với current code (zero diff at runtime).
- **FR-4** Khi override hit → log `INFO` 1 lần: `connection override applied connection_code=<code> origin=config`.
- **FR-5** Generic theo engine: override URI có thể là `mongodb://`, `postgres://`, `mysql://`, hay raw DSN — worker pass-through, không parse/validate scheme.

## Non-Functional Requirements

- **NFR-1** Config layer: YAML `connectionOverrides:` map[string]string. Env override pattern `CONNECTION_OVERRIDE_<CODE>` cho per-key (case-insensitive code).
- **NFR-2** Single helper `ApplyConnectionOverride(conn, overrides, logger) (uri, ok)` — duplicate-free.
- **NFR-3** Lesson: `Caller-Resolver Wiring Verification` lần này phải áp dụng — trace từng caller chain trước khi báo done.

## Out of Scope

- Đổi schema `connection_registry` (giữ host/port nguyên).
- Đổi FE/CMS API (admin tiếp tục nhập URI qua UI).
- Override theo `id` (chỉ theo `connection_code` — semantic stable hơn).
- Override shadow/master config (`cfg.ShadowDB`, `cfg.MasterDB` đã từ static config, không từ `connection_registry`).

## Definition of Done

- [ ] `AppConfig.ConnectionOverrides map[string]string` parse từ YAML + env.
- [ ] Helper `ApplyConnectionOverride` ở `internal/service/connection_overrides.go`.
- [ ] Site A `GetSourceDSN`: apply overlay đầu tiên, log hit.
- [ ] Site B `scanFieldsMongoSource`: apply overlay sau db.First.
- [ ] Site C `resolveSourceMongoDSN`: apply overlay sau db.First.
- [ ] Wiring đầy đủ ở `worker_server.go`.
- [ ] `go build ./...` PASS.
- [ ] `go vet ./...` PASS.
- [ ] `go test ./internal/handler/... ./internal/service/...` PASS.
- [ ] `config-local.yml` có example block `connectionOverrides:` (commented hoặc minimal).
- [ ] User test: restart worker → click Snapshot Now `export-jobs` → log `dispatch_path=mongo_lazy_resolve signal_id=<ObjectID>` (override `goopay` → `mongodb://localhost:17017/...`).
- [ ] `report_connection_overrides.md` ghi đủ file:line thay đổi + verification.
- [ ] APPEND `05_progress.md` mỗi file thay đổi.
- [ ] APPEND global lesson.
