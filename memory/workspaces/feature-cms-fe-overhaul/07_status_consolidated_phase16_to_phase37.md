# Consolidated Status Report — Phase 16 → Phase 37

> **Workspace**: `feature-cms-fe-overhaul`
> **Source log**: `cdc-system/Untitled-2.ini` (raw chat transcript, 2092 dòng)
> **Phạm vi**: Tổng hợp tiến trình refactor CMS FE/BE từ Phase 16 đến Phase 37
> **Author**: Muscle (CC CLI) — tổng hợp lại từ log
> **Created**: 2026-04-28

---

## 0. Bối cảnh chung (Context)

Mục tiêu xuyên suốt: dọn `cms-fe` về kiến trúc V2-native (Debezium-only auto-flow + operator-flow), giảm dần phụ thuộc vào legacy surface `/api/registry` và schema `cdc_internal.*`, kéo write/read path về `cdc_system.*` và `shadow_binding`.

Mô hình 2 luồng chốt từ Phase 8:
- **Auto-flow**: Debezium là luồng chính.
- **Operator-flow** (cms-fe): monitoring / backup / retry / reconcile — phải được giữ lại nhưng API surface gọn.

---

## 1. Bảng tổng hợp 22 Phase

| Phase | Tên | Loại | Trạng thái |
|------:|-----|------|-----------|
| 16 | Sources/Connectors V2 dual-view screen | FE | ✅ Done |
| 17 | cdc_system namespace bridge (model/repo) | BE | ✅ Done |
| 18 | Source Objects read-path V2 | BE+FE | ✅ Done |
| 19 | Shadow Bindings dual-view | BE+FE | ✅ Done |
| 20 | Mapping Context read-model V2 | BE+FE | ✅ Done |
| 21 | Registry Bridge Action Facade | BE+FE | ✅ Done |
| 22 | Transform-status facade | BE+FE | ✅ Done |
| 23 | Dashboard + ActivityManager V2 reads | BE+FE | ✅ Done |
| 24 | Registry mutation facade | BE+FE | ✅ Done |
| 25 | Registry route prune | BE | ✅ Done |
| 26 | Legacy swagger cleanup | BE | ✅ Done |
| 27 | V2 write sync (dual-write follower) | BE | ✅ Done |
| 28 | V2 status visibility (badges UI) | BE+FE | ✅ Done |
| 29 | V2 direct update PATCH | BE+FE | ✅ Done |
| 30 | V2 direct redetect (timestamp) | BE+FE | ✅ Done |
| 31 | Direct scan-fields & transform-status | BE+FE | ✅ Done |
| 32 | Direct standardize | BE+FE | ✅ Done |
| 33 | Schema-aware create-default-columns | BE+Worker+FE | ✅ Done |
| 34 | Shadow schema runtime hardening | Worker | ✅ Done |
| 35 | Runtime schema followthrough (discover/backfill) | Worker | ✅ Done |
| 36 | Schema tail cleanup (event_bridge/transform/repo) | BE+Worker | ✅ Done |
| 37 | Dead-code prune & deprecate | BE+Worker | ⚠️ Partial — session bị block |

---

## 2. Chi tiết từng Phase

### Phase 16 — Sources/Connectors V2 dual-view screen
- **Goal**: Dựng màn V2-native đầu tiên cho cms-fe thay vì tiếp tục vá compatibility layer.
- **Changes**:
  - `SourceConnectors.tsx` refactor thành dual-view: tab Connectors + tab Source Fingerprints.
  - Fetch thêm `/api/v1/sources`; thêm 4 summary cards (Connectors, Fingerprints, Linked, Orphans); cảnh báo Fingerprint-only & Runtime-only.
  - Giữ nguyên destructive actions: create/restart/pause/resume/restart-task/delete.
- **Verify**: `npm run build` pass.
- **Gap**: `source.go` còn bám `cdc_internal.sources`; FE chưa có màn riêng cho `source_object_registry`, `shadow_binding`, `master_binding`.

### Phase 17 — cdc_system namespace bridge
- **Goal**: Sửa namespace runtime model trong CMS backend cho khớp end-state migration.
- **Changes**:
  - `source.go`: `TableName()` `cdc_internal.sources` → `cdc_system.sources`.
  - `wizard_session.go`: `cdc_internal.cdc_wizard_sessions` → `cdc_system.cdc_wizard_sessions`.
  - `wizard_repo.go`: raw SQL update sang `cdc_system.cdc_wizard_sessions`.
- **Verify**: grep namespace cũ rỗng; `go test ./...` pass.

### Phase 18 — Source Objects read-path V2
- **Endpoint mới**: `GET /api/v1/source-objects` (đọc từ `cdc_system.source_object_registry`, `cdc_system.shadow_binding`, latest `cdc_reconciliation_report`, join `cdc_table_registry` chỉ để bridge `registry_id`).
- **FE**: `TableRegistry.tsx` chuyển sang `/api/v1/source-objects`; row chưa có `registry_id` vẫn hiển thị nhưng action legacy bị chặn rõ ràng.
- **Verify**: backend test + FE build pass.

### Phase 19 — Shadow Bindings dual-view
- **Endpoint mới**: `GET /api/v1/shadow-bindings`.
- **FE**: `TableRegistry` thành dual-view (Source Objects + Shadow Bindings) với cột practical: source db/table, binding code, shadow schema/table, physical FQN, write mode, ddl status, recon drift, active, last recon.

### Phase 20 — Mapping Context read-model V2
- **Endpoint**: `GET /api/v1/source-objects/registry/{registry_id}` (bridge-aware detail).
- **FE**: `MappingFieldsPage.tsx` không còn fetch full `/api/registry`, dùng `shadow_schema`/`physical_table_fqn` từ read-model mới.

### Phase 21 — Registry Bridge Action Facade
- **Facade endpoints** (`/api/v1/source-objects/registry/{id}/...`):
  - `create-default-columns`, `standardize`, `scan-fields`, `detect-timestamp-field`, `dispatch-status` (GET).
- **Backend chỉ delegate sang `RegistryHandler`** — không đổi semantics.
- **FE**: `useRegistry.ts`, `ReDetectButton.tsx`, `TableRegistry.tsx`, `MappingFieldsPage.tsx` chuyển sang facade.

### Phase 22 — Transform-status facade
- File `source_object_actions_handler.go` + `useAsyncDispatch.ts` mở rộng cho async transform tracking.

### Phase 23 — Dashboard + ActivityManager V2 reads
- Endpoint mới: `GET /api/v1/source-objects/stats`.
- `Dashboard.tsx` đổi label: Registered Tables → Source Objects, Tables Created → Shadow Ready.
- `ActivityManager.tsx` đổi read source sang `/api/v1/source-objects`, dùng `shadow_schema` cho schedule.
- **Note**: Có 1 lần Go build cache bị sandbox chặn → đã rerun ngoài sandbox và pass.

### Phase 24 — Registry mutation facade
- Bọc 3 mutation: `POST /api/v1/source-objects/register`, `PATCH /api/v1/source-objects/registry/:id`, `POST /api/v1/source-objects/register-batch`.

### Phase 25 — Registry route prune
- Thêm `POST /api/v1/source-objects/registry/:id/transform`.
- **Gỡ toàn bộ route `/api/registry...`** đã có replacement khỏi router.

### Phase 26 — Legacy swagger cleanup
- Dọn `@Router /api/registry...` legacy trong `registry_handler.go` → đổi thành internal delegate notes để spec sau này không sinh lại.
- **Verify grep**: FE runtime `/api/registry` rỗng; router legacy `/registry...` rỗng; `@Router /api/registry` rỗng.

### Phase 27 — V2 write sync (dual-write follower)
- Service mới `source_object_v2_sync.go`: normalize source engine, resolve `source_connection_id`/`shadow_connection_id`, upsert `cdc_system.source_object_registry` và `cdc_system.shadow_binding`.
- Cắm vào `registry_handler.go` + `server.go`. `Register`/`Update`/`BulkRegister` sau khi write legacy thành công sẽ sync sang V2 metadata.
- **Blocker thật**: lần đầu fail vì import `gorm.io/datatypes` không có trong module → đã sửa về `[]byte` JSON payload, rerun pass.

### Phase 28 — V2 status visibility
- BE `source_objects_handler.go` thêm: `shadow_binding_id`, `bridge_status`, `metadata_status` (cả List + GetMappingContext).
- FE thêm cột Metadata, render badges: V2 Ready / Shadow Bound / Source Only / Bridge OK / No Bridge.

### Phase 29 — V2 direct update
- `PATCH /api/v1/source-objects/:id` chỉ support `is_active`, `timestamp_field`, `notes`.
- Update trực tiếp `cdc_system.source_object_registry`; sync `is_active` sang `cdc_system.shadow_binding`.
- FE chọn endpoint theo bridge status; row V2-only update trực tiếp được.

### Phase 30 — V2 direct redetect
- Direct routes theo `source_object_id`:
  - `GET /api/v1/source-objects/{id}/dispatch-status`
  - `POST /api/v1/source-objects/{id}/detect-timestamp-field`
- `reconciliation_handler.go` `LatestReport` trả thêm `source_object_id`.
- `ReDetectButton.tsx` ưu tiên direct V2, fallback bridge.
- **Note thẳng thắn**: 1 lần chạy sai gofmt lên file `.ts/.tsx` (lỗi thao tác formatter scope) → verify lại bằng FE build, pass.

### Phase 31 — Direct scan-fields & transform-status
- `POST /api/v1/source-objects/{id}/scan-fields`
- `GET /api/v1/source-objects/{id}/transform-status`
- Resolve active `shadow_binding` từ `source_object_id`.
- **Regression bắt được**: thiếu `import fmt` ở backend; biến TS thừa trong `useRegistry.ts` → đã sửa, pass.
- **Quyết định**: `create-default-columns` chưa direct V2 hóa do worker còn metadata legacy sâu hơn.

### Phase 32 — Direct standardize
- `POST /api/v1/source-objects/{id}/standardize` (worker chỉ cần `target_table`).
- Row V2-only không còn bị chặn vô lý ở action "Tạo Field MĐ".

### Phase 33 — Schema-aware create-default-columns
- **Root cause**: worker hardcode `public` ở nhiều chỗ.
- BE: `POST /api/v1/source-objects/{id}/create-default-columns`; mở rộng dispatch scope với `shadow_schema`, `primary_key_field`, `primary_key_type`.
- Worker `command_handler.go`:
  - Helper schema-aware: `ensureCDCColumnsInSchema`, `tableExistsInSchema`.
  - `HandleStandardize`, `HandleCreateDefaultColumns` giờ hiểu `shadow_schema`.
  - Update `cdc_system.shadow_binding.ddl_status='created'`.
  - Vẫn dual-write `is_table_created` legacy để không gãy.
- FE: `TableRegistry.tsx`, `MappingFieldsPage.tsx` ưu tiên direct route.

### Phase 34 — Shadow schema runtime hardening (Worker)
- Metadata: thêm `ResolveTargetRoute(targetTable)` ở `metadata_registry_service.go` + `registry_service.go` + tests.
- `schema_validator.go`: `introspectColumns()` không còn hardcode `public`, resolve schema theo target route.
- `command_handler.go` schema-aware cho:
  - `HandleBatchTransform`, `HandleScanRawData`, `HandlePeriodicScan`, `HandleDropGINIndex`, `scanFieldsDebezium`.
- Helpers mới: `quoteCommandQualifiedTable()`, `hasColumnInSchema()`, `resolveTargetRoute()`, `resolveTargetSchema()`.

### Phase 35 — Runtime schema followthrough
- `pending_field_repo.go`: thêm `GetTableColumnsInSchema(ctx, schema, table)`; giữ wrapper public.
- `schema_inspector.go`: inject `MetadataRegistry`, `SetMetadataRegistry()`; cache key đổi thành `schema.table`.
- `worker_server.go`: inject `registrySvc` vào `SchemaInspector`.
- `command_handler.go`: `HandleDiscover` introspect đúng schema; `HandleBackfill` update theo `schema.table` + quote `target_column`.

### Phase 36 — Schema tail cleanup
- `event_bridge.go`: `quoteEventBridgeIdent()`, `quoteEventBridgeQualifiedTable()`, `resolveTargetSchema()`; `pollChanges()` theo `schema.table`.
- `transform_service.go`: thêm metadata-aware schema resolution; `BatchTransform()` update theo `schema.table`.
- `registry_repo.go` (CMS): wrappers `ScanRawKeysInSchema()`, `PerformBackfillInSchema()`, `GetDBColumnsInSchema()`; methods cũ giữ làm fallback.

### Phase 37 — Dead-code prune & deprecate (PARTIAL)
- **Quyết định dứt điểm** (không thêm lớp đệm nữa):
  - `TransformService`: dead code, không có caller → **prune** (xóa file, -112 dòng).
  - Helper raw SQL trong CMS `registry_repo`: không có caller → **prune** (-49 dòng).
  - `EventBridge`: giữ làm **compatibility reserve** (vì còn test + giá trị nếu poller quay lại) → đóng dấu rõ không thuộc runtime chính.
- **Files changed**: 3 (registry_repo.go, event_bridge.go, transform_service.go xóa).
- ⚠️ **Session bị block**: Hit usage limit lúc 8:51 AM, không kịp tạo bộ docs `01..09_phase37_*`.

---

## 3. Tổng số files đã thay đổi

Theo log, mỗi phase có một block "X files changed +Y -Z". Tóm tắt:
- **Phase 16**: 11 files, +323/-24
- **Phase 17**: 13 files, +122/-13
- **Phase 18**: 15 files, +453/-28
- **Phase 19**: 14 files, +430/-32
- **Phase 20**: 14 files, +287/-15
- **Phase 21**: 17 files, +250/-8
- **Phase 22+23**: 34 files, +457/-36 (gộp 1 block)
- **Phase 24+25+26**: 49 files, +484/-146 (gộp 1 block)
- **Phase 27**: 19 files, +438/-16
- **Phase 28**: 18 files, +168/-15
- **Phase 29**: 18 files, +237/-19
- **Phase 30**: 16 files, +324/-8
- **Phase 31**: 15 files, +291/-14
- **Phase 32**: 12 files, +155/-5
- **Phase 33**: 18 files, +333/-34
- **Phase 34**: 16 files, +263/-12
- **Phase 35**: 13 files, +193/-7
- **Phase 36**: 12 files, +267/-11
- **Phase 37**: 3 files, +11/-164 (prune)

---

## 4. Chiến lược kỹ thuật xuyên suốt

1. **Read-path trước, write-path sau**: Cắt FE khỏi `/api/registry` ở read-path để an toàn (Phase 18-23) trước khi đụng mutation.
2. **Facade trước, semantics sau**: Phase 21-22 bọc namespace V2 cho action mà chưa đổi backend semantics — tránh "V2 giả".
3. **Dual-write follower**: Phase 27 sync sang V2 metadata làm follower trước, rồi mới mở direct path V2 (Phase 29-32).
4. **Direct-V2 hóa tuần tự theo độ an toàn**: timestamp-detect → scan-fields/transform-status → standardize → create-default-columns (cần cả worker hardening).
5. **Worker schema hardening theo waves**: Phase 34 (operator path nóng) → 35 (discover/backfill) → 36 (tail) → 37 (prune dead code).
6. **Audit-first, prune-when-replacement-ready**: Chỉ prune route/SQL khi đã có replacement V2 verified.

---

## 5. Risks / Gaps còn lại sau Phase 37

- `is_table_created` vẫn **dual-write** (legacy + `shadow_binding.ddl_status`). Chưa cắt bridge.
- `cdc_table_registry` vẫn là **write gate chính** dù sync follower V2 đã chạy.
- `priority`, `sync_interval` còn **bridge-only**.
- Một số mapping/operator action vẫn neo `registry_id`.
- **Generated swagger docs** chưa regen được do local thiếu binary `swag` (báo lặp ở mọi phase từ 18 trở đi).
- Bộ docs Phase 37 (`01..09_phase37_*`) **chưa tồn tại** trong workspace.
- `EventBridge` đang ở trạng thái compatibility reserve, chưa có quyết định cuối cho runtime chính.

---

## 6. Đề xuất Next Step (Phase 38+)

Theo logic Phase 37 đã chọn "ra quyết định dứt điểm":
1. **Phase 38**: Tạo bộ docs Phase 37 hồi tố (`01..09_phase37_dead_code_prune.md`) để workspace không thiếu.
2. **Phase 39**: Cắt `is_table_created` dual-write — chỉ giữ `shadow_binding.ddl_status` làm SoT.
3. **Phase 40**: Tách write-path V2 khỏi `cdc_table_registry` cho row V2-only (mở `priority`/`sync_interval` ở `source_object_registry`).
4. **Bonus**: Cài `swag` binary trên máy build để regen swagger docs.

---

## 7. Verification matrix (theo log)

| Phase | go test cms | go test worker | npm run build | grep audit |
|------:|:---:|:---:|:---:|:---:|
| 16 | — | — | ✅ | — |
| 17 | ✅ | — | — | ✅ |
| 18 | ✅ | — | ✅ | — |
| 19 | ✅ | — | ✅ | — |
| 20 | ✅ | — | ✅ | — |
| 21 | ✅ | — | ✅ | — |
| 22 | ✅ | — | ✅ | — |
| 23 | ✅ (rerun ngoài sandbox) | — | ✅ | ✅ |
| 24-26 | ✅ | — | ✅ | ✅ |
| 27 | ✅ (sau khi sửa import) | — | — | — |
| 28 | ✅ | — | ✅ | — |
| 29 | ✅ | — | ✅ | — |
| 30 | ✅ | — | ✅ | — |
| 31 | ✅ (sau khi sửa import fmt) | — | ✅ | — |
| 32 | ✅ | — | ✅ | — |
| 33 | ✅ | ✅ | ✅ | — |
| 34 | — | ✅ | — | — |
| 35 | — | ✅ | — | — |
| 36 | ✅ | ✅ (rerun ngoài sandbox) | — | — |
| 37 | — | — | — | — |

---

## 8. Skills đã sử dụng (xuyên suốt)

API contract audit, golang backend refactor, worker runtime hardening, schema-qualified SQL design, metadata-driven routing, swagger annotation update/cleanup, React/TypeScript refactor, route pruning, gofmt, go test, npm build, grep audit, security self-check, workspace documentation, debug compile blocker, data-model sync design.

---

## Skills đã sử dụng (cho task này)

- File reading & log parsing (Untitled-2.ini, 2092 dòng)
- Memory governance (đọc lessons.md, project_context.md, active_plans.md)
- Workspace audit (kiểm tra docs đã có cho Phase 1-36)
- Workspace documentation (tạo report tổng hợp theo prefix chuẩn `07_status_*`)
- Append-only progress log (chuẩn bị append entry Phase 37 vào `05_progress.md`)
- Task tracking (TaskCreate/TaskUpdate)
