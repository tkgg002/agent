# 05_progress — fix-fe-shadow-schema (APPEND ONLY)

## [2026-06-02 — Brain:claude-opus-4-7] Init workspace
- User pivot từ task #19 (delete shadow row, plan-only paused) sang bug FE shadow_schema mismatch.
- Triệu chứng: `/shadow/15/mappings?binding_id=59` → `shadow-columns/tokens?schema=shadow_auth_service` (sai, đúng phải là `shadow_goopay_test_local_as_auth_service`).
- User directive (architectural): "sửa mẹ gì 10 chỗ. sao mang logic ghép name shadow vào fe. bị ngu à. phải lấy từ api chứ".
- Decision: xoá `normalizeShadowSchema` toàn FE; BE là single source of truth.

## [2026-06-02 — Brain:claude-opus-4-7] Audit & Full Doc Set
- Đọc: `MappingFieldsPage.tsx`, `TableRegistry.tsx`, `source_object_read_repo_gorm.go:189-284`, `source_objects_read_models.go:60-89`, `source_objects_handler.go:371-396`.
- Grep: tổng 4 file định nghĩa `normalizeShadowSchema` (TableRegistry, MappingFieldsPage, DataIntegrity, ActivityManager) + 13 callsite fallback.
- Root cause: BE struct `ShadowSchema *string` + tag `json:"shadow_schema,omitempty"` → nil pointer khi LATERAL không match → JSON omit field → FE undefined → fallback bậy.
- Tạo Task #20 (plan-only).
- Tạo Full Doc Set workspace `fix-fe-shadow-schema-2026-06-02/`:
  - `00_context.md` — symptom + evidence + inventory.
  - `01_requirements.md` — R1-R4 + A1-A5 + N1-N2.
  - `02_plan.md` — 3 Phase, 10 task T1-T10, 5 quyết định D1-D5.
  - `03_implementation.md` — code diff cho 6 file (2 BE + 4 FE) + types/index.ts, LOC −18.
  - `08_tasks.md` — DoD từng T1-T10, anti-tasks, escalation.
  - `09_tasks_solution.md` — edge cases S1-S4, smoke TC1-TC8, rollback, verification commands, lesson candidates.
- KHÔNG đụng source code. Đợi user duyệt trước khi Muscle execute.

## [2026-06-02 — Brain:claude-opus-4-7] Lưu ý song hành
- Task #19 (delete shadow row) vẫn pending — đợi user redirect lại.
- Không tự ý resume task #19 khi user chưa duyệt task #20.

## [2026-06-02 — Muscle:claude-opus-4-7] Execute T1-T8
- User duyệt plan + nhắc "Simplicity First, minimal impact".
- T1+T2 BE: 
  - `source_object_read_repo_gorm.go`: COALESCE shadow_schema/physical_table_fqn ở 2 SQL (list line 138-140 + mapping_context line 207-208).
  - `source_objects_read_models.go`: đổi cả 2 struct (List + MappingContext) field `*string` → `string`, bỏ `omitempty`.
  - `go build ./...` EXIT=0.
- T4 FE xoá function `normalizeShadowSchema`:
  - `TableRegistry.tsx`: xoá block line 75-81.
  - `MappingFieldsPage.tsx`: xoá block line 14-20.
  - `DataIntegrity.tsx`: xoá block line 56-62 + xoá `getShadowFqn` standalone (inline vào `getResolvedShadowFqn` fallback = `record.target_table`).
  - `ActivityManager.tsx`: xoá inline arrow line 65-66.
- T5 FE thay 13 callsite:
  - TableRegistry.tsx: 3 sites (`|| ''`).
  - MappingFieldsPage.tsx: 6 sites (`|| ''` + 1 `|| undefined`) + cập nhật comment ngữ nghĩa cũ.
  - DataIntegrity.tsx: 2 sites → simplified fallback `record.target_table`.
  - ActivityManager.tsx: 3 sites (`|| ''`).
- T6 FE `fetchShadowColumns` early-return khi schema empty → `setShadowColumns(new Set())` không call API.
- T7 FE Descriptions "Shadow Schema": ternary render `(chưa có)` khi empty.
- T8 verify: `grep -r normalizeShadowSchema cdc-cms-web/src/` empty + `tsc --noEmit` EXIT=0.
- Còn T9 (manual smoke trên browser) + T10 (report file) → để user verify khi restart service.
- KHÔNG fix `mapping_rule_repo_gorm.go`, `recon_read_models.go`, `list_masters.go`, `list_worker_schedules.go`, `activity_log_read_models.go` — out of scope (FE consumer không bị bug shadow_auth_service); giữ Simplicity First.

## [2026-06-02 — Muscle:claude-opus-4-7] Post-mortem deep root cause
- User hỏi: tại sao `/shadow/13/mappings?binding_id=53` đúng còn `/shadow/15/mappings?binding_id=59` sai.
- Audit `migrations/schema/cdc_system_model/030_v2_source_object_registry.sql`: `source_object_registry` UNIQUE chỉ trên `object_code` + `normalized_source_key`. **KHÔNG có UNIQUE trên `(source_database, source_object_name)`** → multi-connector pattern (vd `legacy_auth` + `goopay_test_local_as_auth_service` cùng route DB `auth-service`) tạo ra N rows so cùng cặp.
- SQL `GetMappingContextByRegistryID` JOIN `LEFT JOIN source_object_registry so ON so.source_database=tr.source_db AND so.source_object_name=tr.source_table` → cartesian → N rows.
- LATERAL `WHERE sb.source_object_id = so.id AND sb.id = ?` → CHỈ match cho `so.id` đúng → các so khác → `sb.*` NULL.
- Query có `LIMIT 1` nhưng **KHÔNG có ORDER BY** → Postgres trả arbitrary row → có thể là row có `sb` NULL → schema rỗng.
- registry_id=13 đúng có thể vì chỉ 1 connector route, registry_id=15 có 2+ connector → 50/50 bug.
- **FIX**: thêm `ORDER BY (sb.id IS NOT NULL) DESC, so.id DESC NULLS LAST` trước `LIMIT 1` (1 dòng). Ưu tiên row có sb matched binding.
- `go build ./...` EXIT=0.
- List query (anchor `FROM source_object_registry so`) KHÔNG bị bug — mỗi so → 1 row. Chỉ patch mapping_context.

## [2026-06-02 — Muscle:claude-opus-4-7] Lesson candidate
- "Multi-connector pattern: bảng metadata KHÔNG có UNIQUE trên natural key (source_db, source_table) → JOIN dùng natural key sẽ duplicate khi có N connector. Đúng: JOIN theo surrogate key (so.id) bị scope bởi context discriminator (binding_id), hoặc thêm ORDER BY `(matched_priority IS NOT NULL) DESC` ưu tiên row có scope match."

## [2026-06-02 — Muscle:claude-opus-4-7] Activity-log scope ambiguous fix
- User báo `/activity-log` row 53 và 59 (cùng `target_table=tokens`, khác `shadow_binding_id`) hiển thị scope cùng `shadow_goopay_test_local_as_auth_service.tokens` (chỉ shadow của binding 59).
- Root cause: `activity_log_read_repo_gorm.go:34-46` LATERAL match `sb.shadow_table = al.target_table` → khi 2+ binding chia chung shadow_table → ORDER BY `updated_at DESC, id DESC LIMIT 1` luôn pick binding mới nhất (id=59) cho TẤT CẢ activity log có target_table='tokens'.
- `cdc_activity_log` schema (migration 010_partitioning.sql) KHÔNG có column `shadow_binding_id` — discriminator chỉ nằm trong `details JSONB` (`details->>'shadow_binding_id'`). Cả 2 row activity_log user post đều có key này trong payload.
- Fix minimal (4 dòng): thêm clause:
  ```sql
  AND (
    NULLIF(al.details->>'shadow_binding_id', '') IS NULL
    OR sb.id = (al.details->>'shadow_binding_id')::bigint
  )
  ```
  Nghĩa: nếu activity log có `shadow_binding_id` trong details → match exact; nếu không → match relaxed như cũ (backward compat).
- KHÔNG sửa `scope_counts` LATERAL — `scope_ambiguous` Tag là cosmetic (tổng binding tiềm năng), không phải bug.
- KHÔNG đổi `cdc_activity_log` schema (thêm column `shadow_binding_id` BIGINT) — Simplicity First; ETL writer hiện đã ghi vào `details::jsonb`, đủ.
- `go build ./...` EXIT=0.

## [2026-06-02 — Muscle:claude-opus-4-7] Register pre-validate identifier (partial state fix)
- User báo prod 500: `shadow_ddl_failed invalid shadow_schema: identifier length`. Và "nó vẫn tạo 1 shadow dù báo lỗi này, check trước validate rồi mới chạy code".
- Audit flow `register_registry.go:99-128`:
  - Transaction (line 99-110): INSERT `cdc_table_registry` + `syncer.SyncFromLegacyTx` → tạo `source_object_registry` + `shadow_binding` (V2).
  - Sau commit (line 116-128): `ResolveShadowSchema` → `EnsureShadowTable` → `validateIdent` (shadow_automator.go:185-195) hard-limit Postgres identifier 63 byte → fail.
  - Rollback path: chỉ `Delete cdc_table_registry` → KHÔNG xoá V2 rows (source_object_registry, shadow_binding) → orphan.
- **Fix**: Pre-validate identifier TRƯỚC khi vào transaction (`register_registry.go:97-114` mới):
  1. Resolve shadow_schema sớm (cần connection_id resolved sẵn từ block phía trên).
  2. `ValidateIdentForTest(entry.TargetTable)` + `ValidateIdentForTest(shadowSchema)`.
  3. Fail → return `ErrShadowDDLFailed` ngay, KHÔNG INSERT V1/V2.
  4. Pass → lưu vào `preResolvedShadowSchema` local var, reuse ở phase EnsureShadowTable (xoá duplicate `ResolveShadowSchema` call sau commit).
- `go build ./...` EXIT=0.
- Đây là fix orphan/partial-state. **CHƯA fix root cause "identifier > 63"** — schema name composition `shadow_<conn>_<db>` vẫn có thể quá dài. User cần quyết định strategy: (A) truncate đầu/cuối (risk collision), (B) hash 8-char suffix (deterministic, unique), (C) yêu cầu connection_code ngắn hơn ở registration.

## [2026-06-02 — Muscle:claude-opus-4-7] Option A: shadow naming = shadow_<connector_code> + FE 64-char warning
- User chọn Option A sau bug "identifier length": "chỉ shadow + connector name. kèm thêm cái fe thông báo nếu số ký tụ vượt quá 64".
- BE (`source_object_v2_sync.go:448-456`): `normalizeShadowSchemaWithConnection` drop `sourceDB`; return `naming.ShadowSchemaName(slugifyIdentifier(connectionCode))`. Giữ sig 2-arg (callsite stability) — param thứ 2 đổi thành `_`.
  - Doc comment cập nhật: lý do (Postgres NAMEDATALEN-1 = 63 byte) + bảo đảm unique (connection_code đã unique per connector).
  - Callers `:106` + `:409` auto-inherit, không cần đụng.
  - `go build ./...` EXIT=0.
- FE (`SourceConnectors.tsx`):
  - Thêm module-const `SHADOW_PREFIX="shadow_"`, `PG_IDENT_MAX=63`, helper `slugifyForShadow` mirror Go (giữ `[a-z0-9]`, các ký tự khác collapse thành `_`, strip leading/trailing `_`).
  - Thêm `Form.useWatch('connectorName')` + memo `shadowSchemaPreview` + bool `shadowSchemaOverflow`.
  - Form.Item `connectorName`: gắn `validateStatus="warning"` + `help` preview `shadow_<slug> (n/63)`, hiện cảnh báo đỏ khi overflow.
  - Không block submit (vẫn cho thử), chỉ warn — vì BE validateIdent sẽ catch nốt.
  - `npx tsc --noEmit -p tsconfig.app.json` EXIT=0.
- Impact: register flow giờ rất khó vượt 63 char (connector_code thường < 50). Cũ `shadow_<conn>_<db>` dễ vỡ vì `goopay_test_local_as_auth_service` đã 32 char trước khi cộng prefix + sourceDB.
- KHÔNG migrate dữ liệu cũ: shadow tables hiện có vẫn chạy (worker đọc từ `shadow_binding.shadow_schema`). Connector đăng ký mới sẽ dùng tên ngắn hơn.
- KHÔNG đụng `buildSourceObjectCode` / `buildShadowBindingCode` — đó là metadata code, không bị PG identifier limit.
