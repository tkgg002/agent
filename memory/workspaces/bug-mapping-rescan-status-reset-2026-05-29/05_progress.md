# 05_progress — APPEND-only audit log

## 2026-05-29 — Session init
- User báo bug `/shadow/20/mappings`: scan shadow tái-tạo không tạo field mới (đúng) nhưng status vẫn approved (sai).
- Yêu cầu thêm 2 feature: cột "In Shadow" trong rules table + System Default Fields hiển thị đủ 11 (hiện 8) kèm trạng thái in shadow.
- Đã đọc lessons.md (lesson mới nhất: L-2026-05-28-enumerate-all-inferrers-before-fix) + GEMINI.md trước khi plan.
- Đã enumerate được 3 entry-points:
  1. `cdc-cms-service ScanRawData` — chỗ duy nhất user trigger scan → reset status tại đây.
  2. `ListMappingRulesHandler` → bổ sung response in_shadow (qua endpoint phụ trợ shadow-columns thay vì JOIN heavy).
  3. FE `SYSTEM_DEFAULT_FIELDS` 8 entries → mở rộng thành 11 entries với PK runtime.
- Tạo workspace docs 00_context, 01_requirements, 02_plan.

## 2026-05-29 — Mid-session correction (§5)
- User báo bug critical: `id` UI=TEXT nhưng tạo shadow column ra BIGINT. Em đang implement FE features → user phải dừng giữa session để cảnh báo.
- **Root cause**: Đã enumerate thiếu entry-point #4 trong fix L-2026-05-28. Path bị miss: `source_object_registry.primary_key_type` (seed legacy = BIGINT) → `payload.PKType` → worker dùng nguyên không apply Mongo policy.
- **Fix**: `centralized-data-service/internal/handler/command_handler.go:706-718` — enforce `pkType = "TEXT"` khi `isMongoPK` bất kể payload (override upstream config).
- **Build verified**: BE worker (`go build ./...` OK), BE cms OK, FE tsc OK.
- **Lesson appended**: `L-2026-05-29-enumerate-includes-upstream-config-payload` vào `agent/memory/global/lessons.md` (extends L-2026-05-28 with upstream-payload pattern + anti-feature-distraction rule).
- **PENDING**: User chưa confirm có muốn keep FE additions (In Shadow column + 11 system fields render). Em dừng feature, chờ user quyết định.

## 2026-05-29 — Clarified scope after user correction
- User clarify: FE additions là **chức năng mới** (giữ); bug int8 là bug cũ (fix). Cả 2 phải clear, không revert.
- Re-verified build cho cả 3 service:
  - `centralized-data-service`: `go build ./...` exit 0
  - `cdc-cms-service`: `go build ./...` exit 0
  - `cdc-cms-web`: `./node_modules/.bin/tsc -b` exit 0
- Report file đã tạo: `data-hub/report_mapping_rescan_status_reset_and_int8_pkType_2026_05_29.md` (5 files changed, ~+126 LOC net).
- Pending: User test golden path để verify runtime (BE em không có quyền touch DB shared).

## 2026-05-29 — Fix System Default Fields (3rd correction)
- User chỉ: "từ lúc nào field ID nó vào System Default Fields". Em đã sai khi push PK source (id) vào danh sách.
- Truth source: `centralized-data-service/internal/sinkworker/schema_manager.go:225-237` — 11 cột system:
  1. `_gpay_id` BIGINT PRIMARY KEY (internal sonyflake, system PK)
  2. `source_id` TEXT NOT NULL
  3. `_raw_data` JSONB
  4. `_source` TEXT
  5. `_synced_at` TIMESTAMPTZ
  6. `_source_ts` BIGINT
  7. `_version` BIGINT
  8. `_hash` TEXT
  9. `_deleted` BOOLEAN
  10. `_created_at` TIMESTAMPTZ
  11. `_updated_at` TIMESTAMPTZ
- PK source (id/_id/wallet_id) là BUSINESS field — không phải system default.
- Fixed `MappingFieldsPage.tsx`: bỏ `buildSystemDefaultFields(pkField)` dynamic, dùng `SYSTEM_DEFAULT_FIELDS` static 11 entries với `_gpay_id` là PK.
- FE tsc verified: exit 0.

---

### 2026-05-29 — Phase 2 closed: align 3 shadow-bootstrap paths + rename `source_id` → `_source_id`

- **Trigger từ user**:
  1. "thằng ngu, đã chuyển id -> _gpay_id thì phải làm full luồng cho nó chứ sao tao tạo shadow mới vẫn là id và _gpay_id (BIGINT) ◯ pending."
  2. "rồi lúc chạy snapshot nữa, tao nhắc rồi đó, lát snapshot lại ko thiếu thì đung nói nhé."
  3. "nếu đc chuyển source_id thành _source_id luôn đi, tránh sau này bị trùng với field của các table source luôn."
  4. "đã nói 1 ngàn lần là anh sẽ xoá db để làm lại toàn bộ, sp chưa realease. em cứ làm sao để chạy mượt sau này."
- **Root cause**: 3 path tạo shadow table (sinkworker runtime, command_handler NATS bootstrap, shadow_automator CMS bootstrap) drift spec với nhau. Đồng thời column `source_id` trùng namespace business field source.
- **Thay đổi (append-only audit)**:
  - Truth source: `centralized-data-service/internal/sinkworker/schema_manager.go` (lines 225-269, 390) — rename + partial UNIQUE INDEX.
  - Sinkworker upsert: `internal/sinkworker/upsert.go` — `ON CONFLICT (_source_id)`, immutable map.
  - Sinkworker record key: `internal/sinkworker/sinkworker.go:147,263`.
  - NATS bootstrap: `centralized-data-service/internal/handler/command_handler.go` — `ensureCDCColumnsInSchema` (149-213) + `HandleCreateDefaultColumns` (689-710), bỏ source-PK column, dùng `_gpay_id` PK + `_source_id` partial UNIQUE INDEX.
  - CMS sync bootstrap: `cdc-cms-service/internal/infra/persistence/shadow_automator.go:75-104` — DDL full V2, thêm `_source_ts BIGINT` đã thiếu, sonyflake trigger trên `NEW._gpay_id`.
  - Schema adapter: `centralized-data-service/internal/service/schema_adapter.go` — rename `_source_id`.
  - Master DDL + transmuter: `master_ddl_generator.go:89,100,136,148` + `transmuter.go:87,328,335,362,449,456`.
  - Soft-delete simplification: `event_handler.go:233-249` — V2 single-path INSERT ... ON CONFLICT (_source_id).
  - Batch buffer: `batch_buffer.go:106-111,251-256`.
  - CMS API: `mapping_preview_handler.go` struct + SELECT; `transmute_schedule_handler.go:228` JSON tag.
  - FE: `MappingFieldsPage.tsx:53,83` (SYSTEM_DEFAULT_FIELDS); `MasterRegistry.tsx:68,425` (placeholder example).
  - Tests: `approve_schema_proposal_integration_test.go:100` updated to `_gpay_id BIGINT PRIMARY KEY, _source_id TEXT NOT NULL`.
- **Đã verify**:
  - `go build ./...` exit 0 — centralized-data-service.
  - `go build ./...` exit 0 — cdc-cms-service.
  - `tsc -b` exit 0 — cdc-cms-web.
  - Snapshot path: routes qua sinkworker upsert → tự động kế thừa spec mới (đã verify đường đi, không thêm code).
- **Lesson liên kết**: `L-2026-05-29-three-shadow-bootstrap-paths-must-align` (lessons.md).
- **DB action cho user**: drop schema cũ + chạy lại migrate.Run() → tự sinh schema mới đồng nhất. Không có dual-read, không có shim.
- **Status**: Phase 2 done. Pending tay user: wipe DB + test E2E lần nữa qua FE flow tạo shadow mới.
