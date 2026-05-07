# 10 — Gap Analysis: Flow 1 (Source → Shadow DB)

> **Author**: x2 (Muscle, cms-lane) | **Date**: 2026-05-07 ICT
> **Method**: Read-only audit cms `internal/api/`, `internal/infra/persistence/{shadow_automator, provisioning_*}.go`, `internal/router/router.go` + worker `internal/handler/provisioning_step_handlers.go` + `internal/server/worker_server.go` (NATS Subscribe).

## 1. Surface đã có (works today)

| Step | Endpoint / Subject | Handler | Status |
|---|---|---|---|
| Step 1 — Connector create | `POST /api/v1/system/connectors` | `SystemConnectorsHandler.Create` → `CreateSystemConnectorCommand` | ✅ |
| Step 1b — Connector list | `GET /api/v1/system/connectors`, `GET /api/v1/system/connector-plugins` | `SystemConnectorsHandler.List/Plugins` | ✅ |
| Step 2 — Connector status | `GET /api/v1/system/connectors/:name` | `SystemConnectorsHandler.Get` | ✅ |
| Step 3 — Register source object | `POST /api/v1/source-objects/register` | `RegistryHandler.Register` (qua bus `RegisterRegistryCommand`) — inline `ShadowAutomator.EnsureShadowTable` | ✅ (legacy path) |
| Step 4 — Set manual mode | `POST /api/v1/cms/sources/:id/provisioning/mode {mode:manual}` | `ProvisioningHandler.SetMode` | ✅ |
| Step 4 — Advance shadow_bind | `POST /api/v1/cms/sources/:id/provisioning/advance` | `ProvisioningHandler.Advance` → publish `cdc.cmd.shadow.bind` | ✅ |
| Step 5 — Get state | `GET /api/v1/cms/sources/:id/provisioning` | `ProvisioningHandler.GetState` | ✅ |
| Step 5 — List shadow bindings | `GET /api/v1/shadow-bindings` | `SourceObjectsHandler.ListShadowBindings` | ✅ |
| Step 6 — Restart Debezium (re-snapshot) | `POST /api/v1/system/connectors/:name/restart` | `SystemConnectorsHandler.Restart` | ✅ |
| Step 7 — Transform status (data check) | `GET /api/v1/source-objects/:id/transform-status` | `SourceObjectActionsHandler.TransformStatusV2` | ✅ |
| Worker subscriber | `cdc.cmd.shadow.bind` | `ProvisioningStepHandler.HandleShadowBind` (Mongo pre-flight + DDL + binding upsert) | ✅ |

## 2. Gaps phát hiện

### G1 — Thiếu endpoint "ping source" độc lập (Step 0)

**Symptom**: Operator chỉ biết source reachable sau khi tạo connector + đợi Kafka Connect feedback. Nếu sai connection string → connector status FAILED, phải xoá + tạo lại. Tốn 1 Kafka Connect REST call + 1 row dirty trong `system_connector_registry`.

**Want**: `POST /api/v1/sources/probe` (admin tier) — body `{kind, connection_string, db, sample_object}` → return `{tcp_reachable, auth_ok, sample_row_count}` mà KHÔNG persist gì.

**Risk if skip**: noise in connector registry, audit churn. Acceptable for v0.

**Priority**: P3 (nice-to-have).

### G2 — 2 đường tạo shadow song song có khả năng drift

**Symptom**:
- Path A (Register-time inline): `RegisterRegistryCommand` → `ShadowAutomator.EnsureShadowTable` — schema convention = caller-resolved (cms) trong `shadow_<source_db>` (theo comment `shadow_automator.go:20`); 8-col layout.
- Path B (provisioning advance): NATS `cdc.cmd.shadow.bind` → worker `HandleShadowBind` — schema = `naming.ShadowSchemaName(connection_code)` = `shadow_<connection_code>`; columns inferred từ source.

**Risk**: Nếu `source_database != connection_code`, 2 schema khác nhau cho cùng 1 source. `cdc_system.shadow_binding` chỉ có 1 row trỏ Path B; Path A tạo physical table không có binding row → orphan.

**Want clarify** (max-Brain decide):
- Boss muốn dùng path nào cho Flow 1 manual?
- Nếu Path B (V2 chính thức) → operator KHÔNG nên gọi `Register` qua RegisterRegistryCommand path (vì nó kích shadow_automator inline). Cần endpoint V2 riêng "register source object metadata only" không tạo shadow table.
- Nếu Path A (legacy + faster) → đơn giản hơn (1 call Register là xong shadow table) nhưng không tích hợp state machine, không gate Mongo pre-flight, không inferSourceColumns.

**Priority**: P0 — phải clarify trước khi viết test plan.

### G3 — Thiếu "verify streaming arrived" check sau Step 4

**Per L-multi-tier-filter** (lesson 2026-05-04): "config write 200 OK ≠ resource streaming". Sau Step 4 advance, operator chỉ biết state = `shadow_active` nhưng không biết Debezium thực sự stream được data tới shadow.

**Want**: Endpoint hoặc query helper kiểm tra:
- Last `_synced_at` trong shadow table < N seconds.
- Hoặc Kafka offset tăng.
- Hoặc `cdc.evt.transmute.completed` nhận trong N giây.

Hiện có `GET /api/v1/source-objects/:id/transform-status` trả `total_rows` — đủ basic. Nhưng không phân biệt "0 rows vì source rỗng" vs "0 rows vì pipeline tắc".

**Priority**: P2 — tăng trust mà không block Flow 1.

### G4 — Wizard FE-driven, không server-side orchestrate

**Symptom**: `WizardHandler.Execute` chỉ flips status=running + ghi progress. Pipeline thực do FE gọi từng endpoint (Patch + system-connectors + register + provisioning). Không có 1 endpoint "atomic Flow 1" cho operator gọi 1 lần ra shadow.

**Want clarify**: Boss có muốn 1 endpoint server-side dispatch toàn Flow 1 (Step 1→7)? Hay giữ manual step-by-step?

Boss directive nói "(manual) các bước" → manual = OK, không cần atomic endpoint. Nhưng có thể Boss muốn 1 wizard step group "shadow-only" (Step 1+3+4) để bớt 3 lần handshake.

**Priority**: P2 — chỉ build nếu Boss confirm.

### G5 — CMS-side mongoClient cho pre-flight (mirror worker)

**Symptom**: Worker `HandleShadowBind` có Mongo pre-flight (`EstimatedDocumentCount > 0`). CMS không có client Mongo để pre-validate trước khi advance. Nếu source rỗng → worker fail step → operator phải Retry. Vòng round-trip thêm.

**Want**: CMS thêm probe `POST /api/v1/sources/:id/source-preflight` (admin tier) — gọi sang Mongo/PG/MariaDB trực tiếp (không qua worker) để fail-fast nếu source rỗng/unreachable.

**Priority**: P3 (worker đã gate, chỉ là UX enhancement).

### G6 — Thiếu doc tổng hợp Flow 1 cho operator

**Want**: 1 trang doc/handbook ghi rõ 7 step + payload mẫu + verify SQL. Hiện tại operator phải đọc rải rác qua handler comments.

**Priority**: P1 — sau khi confirm gap G2.

## 3. Kết luận

**Flow 1 manual hoạt động được TODAY** với 7 step ở §1. Không cần thêm code lớn — chỉ cần Boss/max chốt G2 (chọn Path A hoặc Path B, hay dùng cả hai có doc rõ).

**Gap critical**: G2 (Path A vs B). Các gap còn lại P2-P3, có thể xử lý ở phase sau.

**Test plan pre-execute**: 1 source Mongo `goopay.users` (đã có data) — chạy 7 step, verify DoD ở `01_requirements_flow1`. Toàn bộ via curl + JWT admin.

— x2
