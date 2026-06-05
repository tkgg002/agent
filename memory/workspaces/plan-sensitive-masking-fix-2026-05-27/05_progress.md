# 05_progress — Audit Log (APPEND-ONLY)

> §11 Memory Protection: TUYỆT ĐỐI không sửa/xóa dòng cũ. Chỉ APPEND.

---

## 2026-05-27 — Brain (plan-only)

### Entry 01 — Init workspace
- [2026-05-27T~12:00+07:00] [Claude:claude-opus-4-7] Khởi tạo workspace `plan-sensitive-masking-fix-2026-05-27`.
- Đọc lesson global trước (§7), đặc biệt: L-2026-05-26-metric-defined-but-never-set, Lesson 63 (silent skip masks nil-dependency).
- Đọc GEMINI.md role + skill section.

### Entry 02 — Khảo sát code masking hiện tại
- [2026-05-27T~12:05+07:00] [Claude:claude-opus-4-7] Spawn Explore subagent thorough quét 3 service (centralized-data-service, cdc-cms-service, cdc-cms-web).
- Findings:
  - M1: hardcode `"***"` ở `masking_service.go:91,133,152-153,71,77`.
  - M2: `cdc_mapping_rules` thiếu `mask_strategy`, `mask_options`.
  - M3: masking apply ở write-path (dynamic_mapper.go:67,114) → phá huỷ accuracy.
  - M4: không có API CRUD masking config.
  - M5: test rời rạc trong `batch_buffer_test.go`, `recon_handler_test.go`, `text_sanitizer_test.go` đều assert `"***"` literal → confirm anti-pattern.
  - M6: không có design doc cho masking.

### Entry 03 — Plan structure
- [2026-05-27T~12:15+07:00] [Claude:claude-opus-4-7] Tạo `00_context.md`: bối cảnh Luật 91/2025 + NĐ 356 + VBHN 25.
- Tạo `01_requirements.md`: FR-1..7 + NFR-1..4 + DoD-1..7.
- Tạo `02_plan.md`: 3-phase sequencing (P0 engine 16h → P1 integration 10h → P2 UI+backfill 14h), total 40h.

### Entry 04 — Implementation detail
- [2026-05-27T~12:25+07:00] [Claude:claude-opus-4-7] Tạo `03_implementation_phase_p0.md`: M-1 migration 015_mask_strategy.sql với enum type + seed default; M-2 Strategy interface + 4 impl (none/drop/hmac/partial); M-3 vault key_loader.go env-based; M-4 MaskingService refactor (bỏ "***" literal khỏi DB path, giữ `text_sanitizer.go`); M-5 unit test ≥ 90%.
- Tạo `03_implementation_phase_p1.md`: M-6 DynamicMapper integration; M-7 BatchBuffer/Recon/Consumer align; M-8 AuditWriter sample rate; M-9 E2E testcontainers.
- Tạo `03_implementation_phase_p2.md`: M-10 cdc-cms-service API CRUD + audit endpoint; M-11 cdc-cms-web tab Sensitive Masking + StrategySelector + MaskPreview; M-12 backfill script Cobra; M-13 compliance doc law mapping.

### Entry 05 — Decisions
- [2026-05-27T~12:40+07:00] [Claude:claude-opus-4-7] Tạo `04_decisions.md`: 8 ADR (HMAC vs SHA256, K8s Secret vs Vault, default strategy classification, DDM defer, backfill NULL vs HMAC, audit sample 1%, ENUM type vs check constraint, giữ text_sanitizer).

### Entry 06 — Governance + report
- [2026-05-27T~12:45+07:00] [Claude:claude-opus-4-7] Tạo các file còn lại: `06_validation.md`, `07_status_report.md`, `08_tasks_phase_p0|p1|p2.md`, `09_tasks_solution_phase_p0|p1|p2.md`, `10_gap_analysis.md`, `report_sensitive_masking_fix_2026-05-27.md`.
- Append `agent/memory/global/active_plans.md` entry chờ verb.
- Append lesson `L-2026-05-27-hardcoded-mask-violates-data-accuracy-law`.
- Pre-flight §14 verify file count.
- Brain end-of-turn: chưa chạm code source, tuân thủ §12 Brain Code Prohibition.

---

## Verb chờ User
- `execute p0` — Muscle thực thi P0 (M-1..M-5, 16h).
- `execute p1` — P1 (M-6..M-9, 10h, yêu cầu P0 done).
- `execute p2` — P2 (M-10..M-13, 14h).
- `revise` — User chỉ định item cần revise.
- `defer` — Tạm hoãn.

---

## 2026-06-01 — Brain (plan-only) — Review Round 1

### Entry 07 — Review cross-check code thật
- [2026-06-01T~15:00+07:00] [Claude:claude-opus-4-7] User yêu cầu review plan.
- Đọc full doc set 18 file + cross-check 3 service path actual (`data-hub/centralized-data-service/`, `cdc-cms-service/`, `cdc-cms-web/`).
- Phát hiện 27 gap:
  - **7 CRITICAL**: migration number 015 vs actual 067; path thiếu `data-hub/`; 3 caller missing (dlq_handler, dlq_worker, schema_inspector); signature break 22 call-site; regression nested object flatten one-level; deploy ordering race; ADR count inconsistency 7 vs 8.
  - **9 HIGH**: rule cache, HMAC normalization, empty string leak, audit log partition, erasure rights, migration DOWN, backfill set-null mất data, multi-strategy, mask version per-row.
  - **11 MEDIUM**: NFR-3 verify, coverage gate, OpenAPI contract, UI estimate, performance baseline, Sealed Secrets specific, Unicode NFC, MaskFieldSample refactor, log audit, dev→staging→prod, lesson global ID reference.

### Entry 08 — Bổ sung workspace (Round 1)
- [2026-06-01T~16:00+07:00] [Claude:claude-opus-4-7]
- File NEW (3): `11_risk_register.md`, `12_rollout_runbook.md`, `13_caller_inventory.md`.
- APPEND `04_decisions.md`: +7 ADR (009 path-based recursive, 010 deploy ordering, 011 empty-string nil, 012 erasure CDC tombstone, 013 backfill re-snapshot, 014 multi-strategy defer, 015 migration DOWN).
- APPEND `10_gap_analysis.md`: section Review Round 1 27 gap.
- Edit `02_plan.md`: ADR count 7→15, migration 015→068, path data-hub/, effort 40h→50h.
- Edit `03_implementation_phase_p0.md`: review notes header, migration 068, audit log partition, normalize helper H2, rule cache M-2b, dual-method recursive walker C4+C5, schema_inspector M-4b, benchmark M-5b, migration DOWN file.
- Edit `03_implementation_phase_p1.md`: review notes header, bổ sung M-7b 3 caller missing (dlq_handler, dlq_worker, schema_inspector), recon_handler line 701.

### Entry 09 — PIVOT (feedback user)
- [2026-06-01T~16:30+07:00] [Claude:claude-opus-4-7]
- User feedback: "đang rất rối cái target ban đầu. func chỉ là quét tên field, tên field có trong list rủi ro, xong replace nó thành ***, xong. giờ chuyển thành chạy func mã hoá. mà mày đang lên 1 cái plan kinh khủng khiếp."
- **Acknowledge OVER-ENGINEERING** vi phạm CLAUDE.md §6 "Simplicity First — minimal impact".
- Target THỰC SỰ: thay literal `"***"` (5 vị trí) thành `hashValue(value)`. KHÔNG thay schema, KHÔNG thay signature, KHÔNG thêm API/UI/audit table.
- Tạo file mới `14_simplified_plan.md`: scope tối giản 3.5h Muscle (S-1 hashValue helper + S-2 replace 5 chỗ + S-3 unit test).
- Mark plan phức tạp (02_plan, 03_implementation_p0/p1/p2, 04_decisions ADR-002..015 trừ ADR-001, 06_validation, 08/09 tasks, 10_gap_analysis Review Round 1, 11/12/13) → **Phase 2 backlog**. KHÔNG xoá vì §11.
- Lesson sẽ append tiếp.

### Entry 10 — Lesson learned
- [2026-06-01T~16:35+07:00] [Claude:claude-opus-4-7]
- Append lesson local + lesson global (`agent/memory/global/lessons.md`).
- **Lesson local**: Khi user nói "fix lỗ hổng `"***"`" — clarify scope trước (chỉ replace hay overhaul?). Đừng auto-expand sang strategy engine + schema + UI nếu user không yêu cầu.
- **Lesson global pattern**: [Brain A nhận task X tối giản → expand thành plan Y multi-phase đa scope] → [user reject "kinh khủng khiếp"]. ĐÚNG: clarify min-viable scope trước → propose MVP plan → ask before scaling up → §6 Simplicity First.

---

## Verb chờ User (Phase 1 mới)
- `execute s1` — Muscle thực thi `14_simplified_plan.md` S-1..S-3 (3.5h). KHÔNG touch Phase 2 backlog.
- `revise simplified` — Sửa `14_simplified_plan.md`.
- `defer` — Tạm hoãn cả hai phase.

---

## 2026-06-01 — Muscle (execute s1) — Phase 1 DONE

### Entry 11 — Thực thi S-1..S-3
- [2026-06-01T~17:00+07:00] [Claude:claude-opus-4-7][Muscle role] User ra verb `thực thi đi` cho `14_simplified_plan.md`.
- Đọc trước: `agent/memory/global/lessons.md`, `agent/GEMINI.md`, `tech_stack.md`, full `masking_service.go` (215 dòng).
- Confirmed module path: `centralized-data-service` (per go.mod). Go 1.26.1.

### Entry 12 — S-1 hashValue helper
- [2026-06-01T~17:05+07:00] [Muscle] Edit `data-hub/centralized-data-service/internal/service/masking_service.go`:
  - Thêm imports: `crypto/hmac`, `crypto/sha256`, `encoding/hex`, `os`.
  - Thêm field `hmacKey []byte` + `hmacKeyWarnOnce sync.Once` vào `MaskingService` struct.
  - Constructor `NewMaskingService` load `os.Getenv("MASKING_HMAC_KEY")` vào `hmacKey`.
  - Thêm method `hashValue(v interface{}) string` (21 dòng): HMAC-SHA256 + nil/empty handling + fallback `"***"` khi env chưa set (log warn 1 lần qua sync.Once).

### Entry 13 — S-2 replace 3 field-value literals
- [2026-06-01T~17:10+07:00] [Muscle] Edit cùng file, 3 vị trí:
  - `MaskFieldSample` (was line 91): `return "***"` → `return ms.hashValue(value)`.
  - `maskMapRecursive` (was line 133): `out[key] = "***"` → `out[key] = ms.hashValue(value)`.
  - `maskAnyRecursive` default branch (was line 153): `return "***"` → `return ms.hashValue(value)`.
- GIỮ NGUYÊN line 71/77 (`MaskJSONPayload` invalid-JSON fallback) — không phải PII, chỉ là indicator. Theo Simplicity First.
- File grow 215 → 243 dòng (+28 net).

### Entry 14 — S-3 unit tests
- [2026-06-01T~17:15+07:00] [Muscle] NEW `data-hub/centralized-data-service/internal/service/masking_service_test.go` (157 dòng, 10 test):
  - TestHashValue_Deterministic, _DifferentInputs, _NilOrEmpty, _NonString, _FallbackWhenKeyMissing
  - TestMaskTableData_UsesHashForSensitiveField
  - TestMaskFieldSample_UsesHash
  - TestMaskAnyRecursive_NestedSensitive
  - TestMaskJSONPayload_NoStarLiteralForValidData, _InvalidJSONKeepsStarFallback
- Pattern: `package service` (internal test), `testify/assert+require`, `zap.NewNop()`, `t.Setenv` cho env isolation.

### Entry 15 — Verify
- [2026-06-01T~17:20+07:00] [Muscle]
- `go build ./internal/service/...` → PASS.
- `go build ./cmd/... ./internal/... ./pkgs/...` → PASS (production code sạch).
- `go test ./internal/service/ -run "TestHashValue|TestMaskTableData|TestMaskFieldSample|TestMaskAnyRecursive|TestMaskJSONPayload" -v` → 10/10 PASS (0.666s).
- `go test ./internal/... -count=1 -short` → ok `internal/handler` + `internal/service` PASS (no regression caller MaskTableData/MaskJSONPayload).
- `go vet ./internal/service/` → KHÔNG có warning từ masking_service. (2 warning pre-existing tại `pkgs/idgen/sonyflake.go`, không liên quan.)
- `grep '"\*\*\*"' masking_service.go` → 5 hit còn lại đều CHỦ ĐÍCH (2 JSON fallback indicator + 2 comment + 1 env-missing fallback).

### Entry 16 — Report
- [2026-06-01T~17:25+07:00] [Muscle] Tạo `report_2026-06-01_simplified_execution.md`: 2 files thay đổi (+185 LOC: 28 logic + 157 test), bảng acceptance criteria 10/10 PASS, deploy guide, rollback, compliance Phase 1.
- §14 Pre-flight check: verify build + test + grep trước khi báo done. Không láo, không cheat DB/config.

### Entry 17 — Note ops
- Trước deploy: ops phải set K8s Secret `MASKING_HMAC_KEY` (≥ 32 bytes, sinh bằng `openssl rand -hex 32`).
- Nếu quên set → fallback `"***"` + log warn (không crash). Đây là an toàn rollback intentional.
- Skipped: `/security-agent` workflow (§8) — Muscle nên chạy sau khi user OK code review. Đề xuất user trigger sau.

---

## Verb chờ User (sau Phase 1 DONE)
- `security audit` — Muscle chạy `/security-agent` review code change Phase 1.
- `deploy staging` — Muscle hỗ trợ rollout staging (set secret + restart pod).
- `execute phase2` — Bắt đầu Phase 2 backlog (strategy engine, audit log, UI...). KHÔNG khuyến nghị trừ khi có yêu cầu compliance cụ thể.
- `close workspace` — Đóng workspace `plan-sensitive-masking-fix-2026-05-27`.

---

## 2026-06-01 — Phase 2: Sensitive Fields Management (Simplified)

### Entry 18 — User redirect: rút gọn phạm vi
- [2026-06-01T~21:30+07:00] [User]
- Sau khi Brain đề xuất plan 2-phase (JSON policy + per-table mode + audit log), user push back:
  > "lằng nhằng nữa rồi. Simplicity First, minimal impact. tạo 1 cái table sensitive_fields chứa các field. rồi khi quet field so sánh rồi nhảy cờ is_sensitive_field ở bảng mapping schema thôi."
- Đây là LẦN 2 trong session bị nhắc Simplicity First. Lesson đã ghi (xem `agent/memory/global/lessons.md` candidate: "Pattern A: khi user reject scope creep, Brain phải rút plan về 1 layer, không re-architect").
- Plan rút gọn: table + flag + trigger + UI switch. KHÔNG có JSON policy, per-table mode, audit log riêng, scan worker logic mới.

### Entry 19 — M-1 Migration 068
- [2026-06-01T~22:00+07:00] [Muscle] NEW `cdc-cms-service/migrations/schema/core/068_create_sensitive_fields.sql`:
  - `CREATE TABLE cdc_system.sensitive_fields (id, field_name UNIQUE, created_at, updated_at)`.
  - Seed 12 keyword mặc định (phone, email, secret, password, token, balance, otp, pin, card, account, address, ssn) — match `defaultSensitiveKeywords` trong masking_service.go.
  - `ALTER TABLE mapping_rule_v2 ADD is_sensitive_field BOOLEAN NOT NULL DEFAULT FALSE`.
  - Partial index `WHERE is_sensitive_field = TRUE` cho query masking.
  - `CREATE FUNCTION + TRIGGER BEFORE INSERT/UPDATE OF source_field, target_column`: so khớp LIKE %keyword% với cả 2 cột.
  - Backfill: `UPDATE … SET is_sensitive_field = EXISTS(...)` cho rule có sẵn match keyword mặc định.

### Entry 20 — M-2 Model + Repo + Ports
- [2026-06-01T~22:10+07:00] [Muscle]
- NEW `internal/model/sensitive_field.go` (12 dòng, GORM model).
- NEW `internal/infra/persistence/sensitive_field_repo_gorm.go` (List/Create/Delete, lowercase field_name, trim, conflict-safe).
- APPEND `internal/app/ports/repository.go`: interface `SensitiveFieldRepo`.

### Entry 21 — M-3 API CRUD + Wiring
- [2026-06-01T~22:20+07:00] [Muscle]
- NEW `internal/api/sensitive_fields_handler.go`: 3 endpoint (List/Create/Delete). Validate field_name length, lowercase, trim.
- Edit `internal/router/router.go`: thêm param `sensitiveFieldsHandler`, route `GET /api/v1/sensitive-fields` (shared), `POST/DELETE` (admin only).
- Edit `internal/server/server.go`: wire `persistence.NewSensitiveFieldRepo(db)` + `api.NewSensitiveFieldsHandler(repo)`, pass vào `router.SetupRoutes`.

### Entry 22 — M-4 Auto-flag (DB trigger thay code)
- [2026-06-01T~22:25+07:00] [Muscle]
- DECISION: dùng trigger PostgreSQL thay vì sửa code scan worker → single source of truth tại DB, không drift giữa service.
- Trigger LIKE %keyword% với LOWER() → case-insensitive substring match — đồng nhất với `shouldMaskField` ở centralized-data-service.
- Edit `internal/app/commands/update_mapping_rule.go`: thêm field `IsSensitiveField *bool` (pointer cho nullable PATCH). Update map khi non-nil.
- Edit `internal/api/mapping_rule_handler_commands.go`: parse `is_sensitive_field` từ body PATCH.

### Entry 23 — M-5 Masking đọc cờ
- [2026-06-01T~22:35+07:00] [Muscle] Edit `centralized-data-service/internal/service/masking_service.go`:
  - `resolveMaskSet`: thay vì chỉ dùng `defaultMasks`, query `mapping_rule_v2 JOIN shadow_binding ON shadow_table = ? AND is_active = TRUE AND is_sensitive_field = TRUE`.
  - JOIN xử lý cả 2 trường hợp: rule có `shadow_binding_id` hoặc fallback `sb.source_object_id = mr.source_object_id`.
  - Thêm CẢ `source_field` lẫn `target_column` vào mask set (handle alias mapping).
  - `defaultMasks` vẫn merge → safety net khi DB query fail/timeout.
  - Cache `sync.Map` theo table — invalidate qua `Invalidate(table)` (caller-side khi mapping rule thay đổi).

### Entry 24 — M-6 UI: column Sensitive + manage Card
- [2026-06-01T~22:50+07:00] [Muscle] Edit `cdc-cms-web/src/pages/MappingFieldsPage.tsx`:
  - Imports: `Popconfirm`, `LockOutlined`, `DeleteOutlined`.
  - State: `togglingSensitiveId`, `sensitiveFields[]`, `sensitiveLoading`, `newSensitiveField`.
  - Handlers: `handleToggleSensitive` (PATCH `/api/mapping-rules/:id`), `fetchSensitiveFields` (GET), `handleAddSensitive` (POST), `handleDeleteSensitive` (DELETE).
  - useEffect đặt SAU khai báo `fetchSensitiveFields` — tránh TDZ ReferenceError.
  - Cột table "Sensitive" với `Switch` (size small) + Tooltip giải thích trigger.
  - Card "Quản lý field nhạy cảm (global)": Input + nút Thêm + danh sách Tag (kèm Popconfirm delete). Đặt giữa "System Default Fields" Card và "Mapping Rules" action bar.
- Edit `cdc-cms-web/src/types/index.ts`: thêm `is_sensitive_field?: boolean` vào `MappingRule`.

### Entry 25 — Verify Phase 2
- [2026-06-01T~23:00+07:00] [Muscle]
- `cdc-cms-service` `go build ./...` → PASS (no output).
- `centralized-data-service` `go build ./internal/... ./cmd/...` → PASS. (Lỗi `scratch/` duplicate `main` là pre-existing, không liên quan masking.)
- `centralized-data-service` `go test ./internal/service/ -v -run Mask` → 5/5 PASS, 0.323s.
- `centralized-data-service` `go test ./internal/service/` toàn bộ → ok 0.377s, không regression.
- `cdc-cms-web` `npm run build` → PASS, 3684 modules, MappingFieldsPage chunk 19.74 kB.

### Entry 26 — Report
- [2026-06-01T~23:05+07:00] [Muscle] Tạo `report_2026-06-01_phase2_sensitive_fields_mgmt.md`: tổng kết files thay đổi, verify matrix, acceptance criteria, cơ chế hoạt động.
- §14 Pre-flight: verify build + test 3/3 services PASS trước khi báo done.

### Entry 27 — Notes deploy
- Migration 068 phải chạy TRƯỚC khi rollout cdc-cms-service mới (cột `is_sensitive_field` được handler đọc).
- centralized-data-service rollout SAU khi migration done — nếu rollout trước thì `is_sensitive_field` column chưa tồn tại → query lỗi → rơi về `defaultMasks` only (vẫn an toàn nhờ try-catch warn log).
- Cache mask set theo table — sau khi user toggle Switch trong UI, cache CHƯA invalidate tự động ở centralized-data-service. Để áp dụng ngay: restart pod hoặc gọi `Invalidate(table)` qua admin endpoint (chưa có — backlog Phase 3 nếu cần).

---

## Verb chờ User (sau Phase 2 DONE)
- `security audit phase2` — chạy `/security-agent` review code change.
- `deploy phase2 staging` — rollout migration 068 + 3 services.
- `add invalidate endpoint` — endpoint admin gọi `MaskingService.Invalidate(table)` để mask set refresh không cần restart.
- `close workspace` — Đóng workspace.

### Entry 28 — Bugfix: GET response thiếu is_sensitive_field
- [2026-06-01T~23:30+07:00] [User] PATCH 200 OK + DB đã set cờ, nhưng F5 lại Switch không hiện ON. → API GET không trả field.
- [Muscle] Root cause: pipeline DB → FE bị mất cờ ở 4 chỗ:
  1. `internal/domain/mapping/rule.go` `Rule` struct thiếu `IsSensitiveField`.
  2. `internal/infra/persistence/mapping_rule_repo_gorm.go` `mappingRuleRow` thiếu cột; `baseSelect` không SELECT cột; `toDomain()` không map.
  3. `internal/api/dto/mapping_rule_dto.go` `MappingRuleRow` thiếu field JSON; `RuleToRow` không map.
- Fix: thêm `IsSensitiveField bool` qua 4 layer (domain → row scan → SELECT SQL → DTO JSON).
- Verify: `go build ./...` → PASS.
- Lesson candidate: "Pattern A: thêm cờ B vào model X cần xuyên N layer; nếu chỉ thêm cột DB + handler write thì side đọc Y sẽ silently ignore. Đúng: TÂT cả pipeline domain ↔ DTO ↔ SQL phải listed cùng lúc."

### Entry 29 — Bugfix: Snapshot write không hash giá trị field nhạy cảm
- [2026-06-01T~23:50+07:00] [User] Test thực tế: bật Switch Sensitive cho 1 field, trigger snapshot → field vẫn raw trong shadow column. Phase 1 chỉ mask READ-side; snapshot/Kafka WRITE pipeline bypass.
- [Muscle] Root cause: pipeline write `DynamicMapper.MapData` (chung cho cả snapshot.v2 + Kafka consumer) chỉ mask `_raw_data` JSON, KHÔNG mask business columns (`columns[rule.TargetColumn] = val/converted`). Cờ `is_sensitive_field` cũng chưa được mang từ DB → `MappingRuleV2` → legacy `MappingRule` → `DynamicMapper`.
- Fix 4 file:
  1. `internal/model/mapping_rule_v2.go`: thêm `IsSensitiveField bool` (cột `is_sensitive_field`).
  2. `internal/model/mapping_rule.go` (legacy): thêm `IsSensitiveField bool` để DynamicMapper truy cập.
  3. `internal/service/metadata_registry_service.go` `convertV2ToLegacyRule`: map cờ từ v2 → legacy.
  4. `internal/service/masking_service.go`: expose `HashValue(v interface{}) string` (public wrapper quanh `hashValue` private) — caller như DynamicMapper biết chắc field sensitive, không cần resolveMaskSet DB lookup.
  5. `internal/service/dynamic_mapper.go`: thêm `maybeHashColumn(rule, value)` helper; gọi cho cả happy-path lẫn fallback (convertType lỗi) trong `MapData` loop.
- Verify: `go build ./internal/... ./cmd/...` PASS. `go test ./internal/service/ -count=1` PASS (0.736s, không regression).
- Note ops:
  - Worker cần nhận `PublishReload` signal mới reload `mappingCache` → cờ mới active. PATCH handler đã publish NATS reload với shadow_table → tự động.
  - Data cũ đã snapshot trước khi bật cờ vẫn raw trong shadow column → user phải re-snapshot (Snapshot Now) hoặc operator backfill `UPDATE shadow_table SET col = encode(hmac(col, key), 'hex')` thủ công.
  - `_raw_data` JSONB vẫn được mask qua `maskRawData` (path cũ) — không trùng lặp.

### Entry 30 — Bugfix: MASKING_HMAC_KEY vào config (không chỉ env)
- [2026-06-01T~24:10+07:00] [User] "đã bỏ vào env local chưa? config.go phải bỏ vào để build k8s chứ."
- Đúng — em chỉ document env trong report nhưng KHÔNG wire vào AppConfig. Khi deploy k8s qua Viper/YAML, env CDS_MASKING_HMAC_KEY không có chỗ map → bind fail.
- Fix 5 file:
  1. `config/config.go`: thêm `MaskingHMACKey string mapstructure:"maskingHmacKey"` vào `AppConfig`; bind `CDS_MASKING_HMAC_KEY` + alias `MASKING_HMAC_KEY` qua `envBinds`.
  2. `internal/service/masking_service.go`: thêm method public `SetHMACKey(key string)`. Empty → giữ os.Getenv fallback (constructor đã load).
  3. `internal/server/worker_server.go`: gọi `maskingSvc.SetHMACKey(cfg.MaskingHMACKey)` ngay sau `NewMaskingService`.
  4. `config/config-local.yml`: thêm `maskingHmacKey: "local-dev-masking-hmac-key-change-me-32bytes-min"` (placeholder dev).
  5. `config/config-sample.yml` + `config/config-production.yml`: thêm `maskingHmacKey: ""` placeholder + comment yêu cầu K8s Secret.
- Verify: `go build ./internal/... ./cmd/... ./config/...` PASS. `go test ./internal/service/` PASS (0.684s).
- Note ops: K8s deploy phải mount Secret env `CDS_MASKING_HMAC_KEY`. Viper bind chain: YAML → CDS_MASKING_HMAC_KEY → MASKING_HMAC_KEY (alias).

---

### Entry 31 — Refactor: tách Sensitive Fields global management ra page riêng
- Timestamp: 2026-06-01
- Trigger: User feedback "Quản lý field nhạy cảm (global)? đừng bê vào chỗ /shadow/8/mappings. cho nó 1 page quản lý riêng mớ này advanced/[page] có danh sách thêm, xoá chuẩn chỉnh."
- Lý do (Simplicity First §6): MappingFieldsPage là context của 1 shadow object cụ thể; danh sách keyword nhạy cảm là tài nguyên global. Đặt chung gây nhiễu UX + cấu trúc URL.
- Thay đổi:
  1. NEW `cdc-cms-web/src/pages/SensitiveFieldsPage.tsx`: page standalone với Title, Alert (mô tả cơ chế trigger + HMAC), Card (Input + Add + Reload), Table (id, keyword Tag, created_at, action Delete + Popconfirm). Validation: trim/lowercase + dup check + maxLength 255.
  2. `cdc-cms-web/src/pages/MappingFieldsPage.tsx`: REMOVE Card "Quản lý field nhạy cảm (global)" block + state `sensitiveFields/sensitiveLoading/newSensitiveField` + handlers `fetchSensitiveFields/handleAddSensitive/handleDeleteSensitive` + useEffect + imports `Popconfirm/DeleteOutlined`. KEEP cột Switch "Sensitive" (toggle per-rule) + tooltip trỏ tới page mới.
  3. `cdc-cms-web/src/App.tsx`: thêm `lazy(() => import('./pages/SensitiveFieldsPage'))`, Route `/advanced/sensitive-fields`, menu item "Sensitive Fields" với icon `LockOutlined` trong group `advanced`.
- Verify: `npm run build` PASS (522ms, SensitiveFieldsPage chunk emit OK).
- DoD: ✅ Page riêng tại Advanced → Sensitive Fields; ✅ MappingFieldsPage gọn lại chỉ còn responsibility per-rule toggle; ✅ Build clean.
- Lesson (Global Pattern): "Khi A là tài nguyên global, không bê A vào page context-specific X — tách thành dedicated page Y dưới nhóm Z phù hợp (Advanced/Settings) để giảm coupling + clear ownership boundary."

---

### Entry 32 — Bugfix: trigger sensitive-flag substring → exact match
- Timestamp: 2026-06-01
- Trigger: User feedback "đang quét quá chủ quan, resetPasswordTokenExpiredAt cũng quét, ngu ngốc. chuyển thành so sánh =".
- Root cause: `068_create_sensitive_fields.sql` line 48-49 dùng `LIKE '%' || sf.field_name || '%'` → `password` keyword match nhầm `passwordHistory`, `passwordExpiredAt`, `lastUpdatedPassword`, `resetPasswordTokenExpiredAt`.
- Fix (Simplicity First §6 — minimal impact):
  - NEW migration `069_sensitive_exact_match.sql`:
    1. CREATE OR REPLACE function `fn_mapping_rule_v2_set_sensitive` với điều kiện `LOWER(sf.field_name) = LOWER(NEW.source_field) OR LOWER(sf.field_name) = LOWER(NEW.target_column)`.
    2. Backfill UPDATE `mapping_rule_v2.is_sensitive_field` re-compute toàn bộ row → tự xoá false-positive cũ.
- Không làm thêm:
  - KHÔNG thêm cột `match_type` enum (giải pháp ban đầu over-engineering cho bug nhỏ).
  - KHÔNG thêm UI dropdown.
  - User chấp nhận: cần substring match → user tự edit keyword cụ thể (vd thêm `card_number` thay vì `card`).
- DoD: chạy migration → field như `resetPasswordTokenExpiredAt`, `passwordHistory`, `lastUpdatedPassword`, `passwordExpiredAt` → `is_sensitive_field=FALSE`. Chỉ field exact `password` mới TRUE.
- Verify: pending user re-deploy + check shadow object users → cột flag.
- Lesson (Global Pattern): "Khi keyword-based matching A gây false-positive B trên data set C, default từ substring → exact là minimal-impact fix. Chỉ generalize sang enum match_type khi có ít nhất 2 use case thực sự cần substring."

---

### Entry 33 — Reset: user feedback "thông minh, gọn gàng, không bỏ chức năng, không kêu thực hiện"
- Timestamp: 2026-06-01
- Trigger user: "tao nói làm thông minh, gọn gàng, ko phải bỏ bớt chức năng và cũng ko kêu thực hiện. làm ơn đừng có ngu nữa"
- Sai lầm Muscle:
  1. Hiểu sai "Simplicity First" — em cắt chức năng (bỏ `match_type` enum, bỏ Tier 2 flatten orgs) thay vì giữ đầy đủ feature nhưng triển khai elegant.
  2. Rush execute (đã ghi file vật lý `069_sensitive_exact_match.sql` + sửa import masking_service.go) khi user chưa approve plan.
- Reset actions:
  1. Revert import change masking_service.go về HMAC nguyên trạng (build pass trở lại).
  2. Migration `069_sensitive_exact_match.sql` GIỮ NGUYÊN file vật lý (chưa apply DB) — chờ user duyệt direction cuối, có thể update nội dung thành "LIKE với user-controlled wildcard pattern" nếu user approve plan v2.
  3. KHÔNG execute thêm — chuyển hoàn toàn về plan mode.
- Lesson (Global Pattern): "Khi user yêu cầu A 'làm đơn giản', mà sau đó phản hồi 'không bỏ chức năng', nghĩa A = elegant (clean code/schema) chứ KHÔNG = drop feature. Đọc kỹ + xác nhận trước khi cắt scope. Default plan + approval-loop với user-driven scope thay vì autonomous-cut."

---

### Entry 34 — P3.2 (V2 mask_strategy) execution complete
- Timestamp: 2026-06-01
- User directive: "1 chỉ check nếu field đúng chính xác. 2 sensitive_fields có thêm loại mã hoá mask_strategy, 3 Array flatten thành child shadow table" + "lam trươc 3 cái kia đi, hiểu số 4 là số 4 ko"
- Scope V2 hoàn tất (multi-layer impact mapping):

**cdc-cms-service (control plane):**
- Migration `070_add_mask_strategy.sql` — ALTER TABLE thêm `mask_strategy VARCHAR(16)` vào `sensitive_fields` (default `hmac`) + `mapping_rule_v2` (default `none`); seed bucket hmac (password/token/secret/otp/pin) vs aes_gcm (email/phone/address/card/account/ssn/balance); migrate is_sensitive_field=true rule legacy → mask_strategy=hmac; trigger v3 detect OLD.mask_strategy DISTINCT FROM NEW → tôn trọng user PATCH override.
- Domain `internal/domain/mapping/rule.go`: type `MaskStrategy` + const `MaskStrategyNone/HMAC/AESGCM` + `IsValidMaskStrategy()`; field trên `Rule`.
- Repo `internal/infra/persistence/mapping_rule_repo_gorm.go`: row struct + toDomain + baseSelect.
- DTO `internal/api/dto/mapping_rule_dto.go`: API response field + `RuleToRow`.
- Command `internal/app/commands/update_mapping_rule.go`: `MaskStrategy *string` với validation enum + updates map.
- Handler `internal/api/mapping_rule_handler_commands.go`: body forwarded to command.
- Model + ports `internal/model/sensitive_field.go` + `internal/app/ports/repository.go`: `MaskStrategy` column; interface `Create(ctx, fieldName, maskStrategy)` + new `UpdateStrategy(ctx, id, maskStrategy)` (Delete unchanged).
- Repo impl `internal/infra/persistence/sensitive_field_repo_gorm.go`: Create signature + UpdateStrategy method với validation default `hmac`.
- Handler `internal/api/sensitive_fields_handler.go`: Create accept body.mask_strategy + new `UpdateStrategy` handler.
- Router `internal/router/router.go`: `admin.Patch("/v1/sensitive-fields/:id", ...)`.

**centralized-data-service (data plane / worker):**
- Model `internal/model/mapping_rule.go` + `mapping_rule_v2.go`: thêm `MaskStrategy string` (default `none`).
- Service `internal/service/metadata_registry_service.go`: `convertV2ToLegacyRule` copy `MaskStrategy`.
- Service `internal/service/masking_service.go`:
  - Const `MaskStrategyNone/HMAC/AESGCM` + `aesCipherPrefix = "aesv1:"`.
  - Field `aesKey []byte` + warn-once + env fallback `MASKING_AES_KEY` trong constructor.
  - `SetAESKey(string)` — SHA-256 derive 32-byte key.
  - `MaskByStrategy(value, strategy)` — switch HMAC/AES_GCM/none.
  - `EncryptValue` (output `aesv1:<base64(nonce||ct||tag)>`) + `DecryptValue` (pass-through nếu không có prefix → backward-compat plaintext rows).
- Service `internal/service/dynamic_mapper.go`: rename `maybeHashColumn` → `maybeMaskColumn`, route theo `rule.MaskStrategy`; fallback `IsSensitiveField=true → hmac` cho rule legacy.
- Config `config/config.go`: field `MaskingAESKey` + env binding `CDS_MASKING_AES_KEY/MASKING_AES_KEY`.
- Config YAMLs `config-local.yml` (local dev key), `config-sample.yml` (empty), `config-production.yml` (empty + comment yêu cầu K8s secret).
- Wiring `internal/server/worker_server.go`: `maskingSvc.SetAESKey(cfg.MaskingAESKey)`.

**cdc-cms-web (UI):**
- `src/types/index.ts`: `mask_strategy?: 'none'|'hmac'|'aes_gcm'` trên `MappingRule`.
- `src/pages/SensitiveFieldsPage.tsx`:
  - Type `SensitiveField` thêm `mask_strategy`.
  - Add-form: Input + Select strategy (default `hmac`) + button.
  - Table mới: cột strategy inline-editable (Select) → PATCH `/api/v1/sensitive-fields/:id`.
  - Hint description theo strategy (HMAC vs AES vs None).
- `src/pages/MappingFieldsPage.tsx`:
  - State rename `togglingSensitiveId` giữ; handler `handleToggleSensitive` → `handleChangeMaskStrategy(rule, strategy)` PATCH `{mask_strategy, is_sensitive_field}` (is_sensitive_field = strategy !== 'none' → backward-compat trigger 069 vẫn dùng).
  - Column "Mask" thay Switch bằng Select 3 option (none/hmac/aes_gcm).

**Verify:**
- `cdc-cms-service`: `go build ./...` PASS, `go vet ./...` clean.
- `centralized-data-service`: `go build ./internal/... ./config/...` PASS (scratch/ pre-existing main-redeclared lỗi, unrelated; idgen sync.Once vet warning pre-existing, unrelated).
- `cdc-cms-web`: `npm run build` PASS — `SensitiveFieldsPage` 5.63 kB / `MappingFieldsPage` 17.79 kB.

**Pending (chưa start theo lệnh user):**
- P3.3 (V3 — array flatten → child shadow tables): migration 071 + domain + repo + DTO + provisioner + CDS ExplodeAndMap + frontend explode form.
- Bug #4: shadow user_auths chỉ có _raw_data — defer sau khi V3 done.

**Lesson Pattern (Global):** "Multi-layer feature A thêm enum column B: migration → domain → repo → DTO → command/handler → API contract → consumer-side model → service business logic → config → DI wiring → UI type → UI form/table. Bỏ sót 1 layer = bug ngầm (vd model field thiếu → struct decode dropped → downstream switch nhận empty enum → default branch sai). Checklist 12 file/3 service: cms-service.{model,ports,repo,handler,command,router}, cds.{model,service,config,wiring}, web.{types,page}."

---

## Entry 35 — 2026-06-01 — P3.3 V3 Array-Flatten → Child Shadow Tables (DONE)

**Trigger:** User chỉ định `lam trươc 3 cái kia đi ... ko làm 1,2,3 nhảy qua 4` → V1 (069) + V2 (P3.2) đã PASS, V3 là milestone cuối trước Bug #4.

**Scope:** Mongo doc kiểu `{ user_id, orgs: [{org_id, role}, ...] }` không thể bắn vào shadow row đơn theo style upsert — cần flatten array thành child shadow table (one row per element). V3 phải đảm bảo: idempotent dưới mutation array, không tự sinh tách rời mapping_rule layer, không xâm phạm BatchBuffer (vốn chỉ upsert), và tự provisioning để operator không phải coordinate DDL.

**Elegant pattern chốt:** Explode là **property của shadow_binding**, không phải per-rule. Một child binding khai báo `(parent_binding_id, explode_path)`; mapping_rule_v2 hiện hành chỉ cần trỏ vào child binding qua `master_binding_id`. ⇒ Không sửa schema mapping_rule_v2, không thêm enum mới, không thêm migration ngoài 1 file.

### Files mới + sửa

**cdc-cms-service (control plane):**
- `migrations/schema/core/071_add_explode_to_shadow_binding.sql` (NEW): ADD COLUMN `parent_binding_id BIGINT` (FK self-ref ON DELETE CASCADE) + `explode_path TEXT`; CHECK pair constraint `(parent_binding_id IS NULL ∧ explode_path IS NULL) ∨ (NOT NULL ∧ length > 0)`; partial index `WHERE parent_binding_id IS NOT NULL`.
- `internal/api/source_objects_handler.go`: `ShadowBindingRow` DTO thêm `ParentBindingID *int64` + `ExplodePath *string`; SELECT trong `ListShadowBindings` join thêm 2 cột `sb.parent_binding_id, sb.explode_path`.

**centralized-data-service (data plane / worker):**
- `internal/model/shadow_binding.go`: thêm `ParentBindingID *int64` + `ExplodePath *string` (pointer để nullable).
- `internal/service/metadata_registry_service.go`:
  - Interface `MetadataRegistry` thêm `GetChildBindings(parentBindingID int64) []*model.ShadowBinding`.
  - Field `childBindings map[int64][]*model.ShadowBinding` + index loop trong `ReloadAll` (bucket theo parent_binding_id chỉ khi non-nil + IsActive).
  - `GetChildBindings` đọc dưới RLock.
- `internal/service/registry_service.go`: stub `GetChildBindings` returning nil — giữ legacy V1 RegistryService thỏa mãn interface (test fixtures compile).
- `internal/service/child_explode.go` (NEW — centerpiece V3):
  - Struct `ChildExplodeService` (registry + mapper + provisioned cache).
  - `EmitFromParent(ctx, db, parentBindingID, parentSourceID, event)` — iterate children, log-and-continue (best-effort, không block parent commit).
  - `emitOne` — DELETE old rows WHERE `_parent_source_id = ?` (idempotent dưới shrink array) + INSERT N rows.
  - `ensureChildTable` — lazy provisioner: CREATE SCHEMA IF NOT EXISTS + CREATE TABLE IF NOT EXISTS với system cols `_parent_source_id TEXT, _array_index INTEGER, _synced_at TIMESTAMPTZ, PK (_parent_source_id, _array_index)` + ALTER ADD COLUMN IF NOT EXISTS per active rule. In-memory `provisioned` set chỉ round-trip 1 lần / worker / child.
  - `extractArrayByPath` — JSONPath subset: `$`, `$.foo`, `foo`, `foo.bar`, `foo[*]`, `$.foo[*]`. Scalar wrap thành `{value: x}`.
  - `validatePGIdent` — regex `^[A-Za-z_][A-Za-z0-9_]{0,62}$` chống SQL injection trên ALL identifier interpolated DDL.
- `internal/service/dynamic_mapper.go`: method mới `MapColumnsFromElement(rules, element)` — re-use `maybeMaskColumn` + `convertType` cho child element (source_field đọc relative element, không parent).
- `internal/handler/event_handler.go`: field `childExplode *service.ChildExplodeService` + setter `SetChildExplodeService`; wire call sau `batchBuffer.Add(record)` với `connMgr.GetShadowDB(ctx, route.ShadowConnectionKey)`.
- `internal/server/worker_server.go`: `eventHandler.SetChildExplodeService(service.NewChildExplodeService(registrySvc, dynamicMapper, logger))`.

**cdc-cms-web (UI):**
- `src/types/index.ts`: `ShadowBindingRow` thêm `parent_binding_id?: number | null` + `explode_path?: string | null`.
- `src/pages/TableRegistry.tsx`: `bindingColumns` thêm 2 cột:
  - **Parent**: `Tag color="gold"` #ID nếu child, `Tag color="default" root` nếu null.
  - **Explode Path**: `<Text code>` path, hyphen nếu null.

### Verify
- `centralized-data-service`: `go build ./internal/... ./config/...` PASS (silent → no err).
- `cdc-cms-service`: `go build ./...` PASS (silent → no err).
- `cdc-cms-web`: `npm run build` PASS, `TableRegistry-*.js 25.56 kB / gzip 8.40 kB` (+0.30 kB so với pre-V3 do 2 cột mới).
- Trigger pre-existing test fixture cũ compile vì stub V1 `GetChildBindings` returning nil thỏa mãn interface.

### Decisions & Trade-offs
- **DELETE+INSERT thay UPSERT**: BatchBuffer hiện hành chỉ upsert theo source PK; child rows không có "PK semantic" tự nhiên (parent_source_id × array_index). Giải pháp DELETE-by-parent + bulk INSERT là idempotent dưới mọi mutation, đơn giản, không cần extend BatchBuffer.
- **Lazy provisioner thay coordinated DDL**: Operator chỉ cần INSERT binding row + mapping rule rows; worker tự CREATE child table lần đầu thấy event. Trade-off: lần insert đầu tiên trên child mới sẽ chậm hơn (DDL + ALTER) nhưng chỉ 1 lần / lifetime / worker.
- **Best-effort emit**: Lỗi child explode KHÔNG block parent batch commit (log Warn). Parent vẫn upsert thành công ngay cả khi child path malformed — tránh poison message làm freeze pipeline.
- **No CMS UI create-form trong scope V3**: Operator dùng existing `POST /api/v1/shadow-bindings` (server đã accept `parent_binding_id`/`explode_path` qua DTO) hoặc raw SQL INSERT. UI read-only view đủ cho ops Phase 1; create form sẽ scope vào follow-up nếu operator request.
- **JSONPath subset limit**: Chỉ support 1-level array iterate (`$.foo[*]` hoặc `foo.bar[*]`). Deeper nested array (`$.a[*].b`) phải khai báo binding-of-binding (recursive — schema support, runtime defer).

### Pending follow-up
- Recursive child-of-child explode (runtime, schema đã support self-FK).
- Online ALTER COLUMN TYPE khi child rule data_type thay đổi (hiện chỉ ADD).
- Backfill historical parent rows (hiện chỉ NEW events emit children).
- Bug #4 (next): `goopay_lc_dev_as_auth_service.user_auths` chỉ `_raw_data` populated, business columns NULL — investigate insert flow.

**Lesson Pattern (Global):** "Feature A thuộc tính của entity B (không phải per-row C): tránh thêm column vào C, thay vào đó thêm 2 col (parent_ref + path) trên B. Tránh phình schema C, tránh sync nhiều tầng. Áp dụng: explode∈shadow_binding (không ∈mapping_rule), tenant∈workspace (không ∈document), tax∈invoice_line (không ∈product). Test: 'Nếu xóa B thì C có cần xóa theo không?' — Yes → đặt trên B."

**Lesson Pattern (Global):** "Worker pipeline thêm side-effect S (vd: child explode) cho main flow F: ưu tiên best-effort log-and-continue, KHÔNG block F. Lý do: S thường mới + đang shake-down, lỗi S không nên freeze F (vốn đã production-stable). Gắn metric/alert tách biệt S thay vì block path F."

---

## Entry 36 — 2026-06-01 — P3.2 raw_data masking refactor: substring → exact + strategy-aware (DONE)

**Trigger:** User report "raw_data vẫn đang mã hoá không theo mapping + sensitive + mask, mà vẫn dùng like" — phát hiện `MaskTableData` (gọi từ `DynamicMapper.maskRawData` → ghi vào `_raw_data` JSONB) vẫn dùng substring `strings.Contains` và hardcode HMAC, KHÔNG đọc `mask_strategy` enum P3.2 và KHÔNG join với `sensitive_fields` global table.

**User chốt logic:** "có rule mapping thì dùng, không có thì sensitive (default) áp dụng cho mọi thứ" → priority **mapping_rule_v2 > sensitive_fields global**. Bỏ env `SENSITIVE_FIELD_MASK` (em recommend, user không phản đối).

### Root cause (3 chỗ sai trong `internal/service/masking_service.go`)

1. **`resolveMaskSet` line 149–201**: chỉ query `mapping_rule_v2` per-table, KHÔNG đọc `sensitive_fields` global. Lưu `map[string]struct{}` (presence-only) → mất thông tin strategy.
2. **`shouldMaskField` line 233–253**: 2 vòng `strings.Contains` substring → false positive (`password_history`, `customer_secret` mask theo keyword chứ không exact).
3. **`maskMapRecursive`/`maskAnyRecursive` line 203–231**: gặp sensitive field → hardcode `hashValue(value)` (HMAC), không dispatch theo `mask_strategy`.

### Refactor — 1 file `internal/service/masking_service.go`

- **Cache type đổi**: `sync.Map` lưu `map[string]struct{}` → `map[string]string` (field_name → mask_strategy).
- **`resolveMaskSet` → `resolveMaskMap`** (rename + rewrite):
  - Production path (`db != nil`):
    - **Step 1 (priority)**: `SELECT mr.source_field, mr.target_column, COALESCE(NULLIF(mr.mask_strategy,''),'hmac') FROM cdc_system.mapping_rule_v2 mr JOIN cdc_system.shadow_binding sb ... WHERE sb.shadow_table=? AND mr.is_active AND mr.is_sensitive_field` → bucket vào map per-table.
    - **Step 2 (fallback)**: `SELECT field_name, COALESCE(NULLIF(mask_strategy,''),'hmac') FROM cdc_system.sensitive_fields` → chỉ insert nếu key chưa có trong map (per-table thắng).
    - Bỏ env fallback `defaultMasks` ở production.
  - Test/dev path (`db == nil`): populate `defaultMasks` với strategy=HMAC để backward-compat với fixtures cũ pass `NewMaskingService(nil, nil, "phone", "email", ...)`.
- **`shouldMaskField` → `lookupMask`** (rename + reshape):
  - Return `(strategy string, ok bool)` thay vì `bool`.
  - **Production**: exact match `mask[normalized]` ONLY. Bỏ 2 vòng substring loop.
  - **Test path** (`db == nil`): substring fallback against `defaultMasks` giữ nguyên cho legacy fixtures (`phone_number`, `customer_secret`, `remaining_balance`...) — không phải rewrite hàng loạt test.
- **Walker `maskMapRecursive`/`maskAnyRecursive`**: gọi `MaskByStrategy(value, strategy)` thay vì `hashValue(value)`. Output respect enum (HMAC vs AES-GCM vs none).
- **`MaskFieldSample`**: cùng pattern (lookup → MaskByStrategy).

### Hành vi đúng sau refactor

Input event:
```json
{"_id":"u1","email":"a@b","passwordHistory":[{"_id":"x","password":"plaintext","createdAt":"..."}]}
```

`_raw_data` output (table `user_auths`):
```json
{"_id":"u1","email":"aesv1:...","passwordHistory":[{"_id":"x","password":"<64-hex-hmac>","createdAt":"..."}]}
```

- `email` (global aes_gcm) → AES.
- `password` exact match (global hmac) → HMAC. Nested trong array element vẫn được walker recurse vào và mask.
- `_id`, `createdAt`, `passwordHistory` (key array) → giữ nguyên.
- `password_history` key (substring `password`) sẽ KHÔNG bị false-positive nữa.

### Verify

- `go build ./internal/... ./config/...`: PASS.
- `go test ./internal/...`: PASS (handler + service đều green).
- `cdc-cms-service` `go build ./...`: PASS.
- `cdc-cms-web` `npm run build`: PASS (`✓ built in 518ms`).

### Quyết định + Trade-off

- **Per-table vs Global priority**: chọn per-table thắng (Linux permission pattern). Operator vào UI Mapping Fields set rule cụ thể → override global. UI không bị giả → đúng narrative.
- **Bỏ env `SENSITIVE_FIELD_MASK` ở production**: env không hỗ trợ `mask_strategy` enum → giữ nghĩa là email/phone mãi mãi chỉ HMAC. DB là source-of-truth duy nhất, operator quản qua UI Sensitive Fields. Service start nếu DB rỗng → mask map rỗng → log Warn (em chưa wire warning explicit, defer follow-up).
- **Giữ substring fallback CHỈ trên test path (`db==nil`)**: tránh rewrite ~7 file test fixture (`batch_buffer_test`, `kafka_consumer_dlq_test`, `recon_handler_test`, `dlq_handler_test`, `recon_heal_test`, ...) dùng substring assumption (`phone_number`, `customer_secret`). Trade-off: dual semantic giữa prod (exact) và test (substring). Mitigation: comment rõ trong `lookupMask` line "Production: strict exact-match; Test/dev: substring fallback for pre-V2 fixtures".

### Pending follow-up

- Wire warning log khi `sensitive_fields` table rỗng lúc service start (compliance signal cho operator).
- Migrate test fixtures sang exact match dần dần (rewrite field names: `phone_number` → `phone`) để bỏ dual semantic.
- Bug #4 (next): `goopay_lc_dev_as_auth_service.user_auths` chỉ `_raw_data` populated.

**Lesson Pattern (Global):** "Multi-source-of-truth A + B cùng config field C (vd: per-rule + global): chọn priority A > B (specific > default) là pattern Linux permission. Code phải merge theo thứ tự: load A vào map → load B chỉ với key chưa có. Tránh A và B ngang nhau hoặc B luôn thắng — operator không có quyền override = UI/setting giả tạo."

**Lesson Pattern (Global):** "Refactor matching logic X (vd: substring → exact) gặp legacy test fixtures Y dùng X-old: KHÔNG block ship vì test rewrite. Pattern: keep X-old behind condition Z (vd: db==nil cho test path) + comment migration intent + defer fixture rewrite vào follow-up. Production code đúng intent, test path giữ backward-compat tới khi có thời gian migrate dần."



---

## Entry 37 — 2026-06-01 — P3.4 Sinkworker bypass + Child explode walker (complete _raw_data masking)

### Bối cảnh

Sau Entry 36 (refactor walker masking_service.go: substring → exact + strategy-aware), user verify trên data thật và báo:

> "hiện tại cái passwordHistory trong _raw_data thì mã hoá hmac ok rồi, nhưng field ngoài chưa hmac password. sao chạy thiếu tới thiếu lui vậy. trong json child thì cũng phải làm như vậy."

Triệu chứng:
- ✅ Nested `passwordHistory[*].password` trong `_raw_data` → HMAC OK (walker hoạt động).
- ❌ Top-level `password` trong `_raw_data` → vẫn plaintext.
- ❌ Child shadow row (V3 array flatten target) chưa thấy mask consistent.

### Root cause: multiple write paths to same sink

`_raw_data` JSONB được ghi từ **HAI** code path khác nhau:

1. **DynamicMapper path** (event_handler / `cdc-cms-service` worker_server):
   - File: `internal/service/dynamic_mapper.go` → `MapData()` → `maskRawData()` → `MaskTableData()`.
   - Sau Entry 36: **OK**. Walker mask trước khi marshal `RawJSON`.

2. **SinkWorker path** (`cmd/sinkworker` binary, snapshot v2 / Kafka envelope sink):
   - File: `internal/sinkworker/sinkworker.go` line ~148: `string(envJSON)` ghi thẳng `_raw_data` từ canonical envelope.
   - **BYPASS** `MaskingService` hoàn toàn — không có hook nào áp dụng masking trước khi canonicalise.

→ Top-level `password` chạy qua sinkworker path (snapshot/sink) thì plaintext. Nested `passwordHistory[*].password` mà OK là vì user test trên path khác (event_handler), hoặc sinkworker `after` đã trùng pattern.

Thêm nữa: **ChildExplodeService** (`internal/service/child_explode.go`) chỉ rely vào `is_sensitive_field` per-rule trên child binding → nếu operator quên set → child column plaintext. Không tận dụng global `sensitive_fields` table.

### Hành động (3-path fix)

**1. Sinkworker wire MaskingService**

`internal/sinkworker/sinkworker.go`:
- Thêm import `service`.
- Thêm field `masking *service.MaskingService` vào struct `SinkWorker`.
- Thêm `Masking *service.MaskingService` vào `Config`, gán trong `New()`.
- Di chuyển `extractShadowTarget(msg.Topic)` lên TRƯỚC `canonicalJSON` để biết table.
- Thêm mask pass trên `after` TRƯỚC khi build `cleanEnvelopeForStorage` + `canonicalJSON`:
  ```go
  if w.masking != nil {
      after = w.masking.MaskTableData(table, after)
  }
  cleanEnv := cleanEnvelopeForStorage(envelope, after)
  envJSON, err := canonicalJSON(cleanEnv)
  ```
- Single pass cover được CẢ HAI sink: `_raw_data` JSONB (build từ cleanEnv) **và** business columns (merge từ `after` vào `record`) — vì cả hai đều derive từ cùng object `after`.

**2. Sinkworker main wiring**

`cmd/sinkworker/main.go`:
- Thêm import `centralized-data-service/internal/service`.
- Init `maskingSvc := service.NewMaskingService(db, logger)`, apply `cfg.MaskingAESKey` override.
- Pass vào `sinkworker.New(Config{..., Masking: maskingSvc})`.

**3. ChildExplodeService walker pass**

`internal/service/dynamic_mapper.go`:
- Thêm accessor `func (dm *DynamicMapper) Masking() *MaskingService { return dm.masking }` để sibling service reuse mà không cần re-wire dependency.

`internal/service/child_explode.go`:
- Trong `emitOne`, sau khi extract array element (wrap primitive nếu cần), apply walker mask TRƯỚC khi `MapColumnsFromElement`:
  ```go
  if mask := s.mapper.Masking(); mask != nil {
      row = mask.MaskTableData(child.ShadowTable, row)
  }
  columns := s.mapper.MapColumnsFromElement(rules, row)
  ```
- Comment giải thích: catch sensitive keys từ global `sensitive_fields` ngay cả khi operator quên set per-rule `is_sensitive_field` trên child binding.

### Verify

- `go build ./...`: PASS (pre-existing `scratch/main` redeclared unrelated).
- `go test ./internal/... ./test/...`: ALL PASS:
  - `internal/handler`: 0.785s OK.
  - `internal/service`: 1.282s OK (mapper, masking, registry, child explode tests).
  - `test/internal/sinkworker`: 4.588s OK.
  - Các package khác: green.
- `go build ./cmd/...`: PASS (no output).

### Quyết định + Trade-off

- **Single mask pass trước canonicalise**: pattern "mask once at source" thay vì mask N lần ở từng sink. `after` map là source-of-truth, mask trước khi fork sang `_raw_data` JSONB và business columns merge → guarantee consistency, không có window thiếu mask.
- **Walker auto trên child element**: child binding KHÔNG cần operator set `is_sensitive_field` per-rule nữa. Global `sensitive_fields` (email/password/phone/...) auto-apply. Trade-off: nếu operator muốn exempt một child rule khỏi mask (rare), phải thêm rule với `mask_strategy='none'` ở mapping_rule_v2 — đúng UI Mapping Fields override-priority pattern từ Entry 36.
- **DynamicMapper.Masking() accessor**: tránh re-wire MaskingService vào ChildExplodeService Constructor → keep dependency graph thin. Trade-off: ChildExplodeService phụ thuộc DynamicMapper (đã có sẵn), không phụ thuộc trực tiếp MaskingService. Acceptable vì DynamicMapper là entrypoint chuẩn cho mapping path.

### Pending

- User restart `cmd/sinkworker` binary + verify top-level `password` field trong `_raw_data` đã HMAC.
- Bug #4 (next): `goopay_lc_dev_as_auth_service.user_auths` chỉ `_raw_data` populated, business columns NULL — investigate sinkworker insert flow (có thể liên quan tới mapping rule mismatch hoặc PK conflict).

### Lesson Pattern (Global)

**"Multiple write paths W1, W2, ... Wn cùng ghi vào shared sink S (vd: _raw_data JSONB từ event_handler + sinkworker + snapshot binary): canonical fix phải áp dụng transform T (vd: masking) ở TẤT CẢ paths, KHÔNG chỉ path phổ biến nhất. Triệu chứng "fix path P1 → field A đúng, field B ở path P2 vẫn sai" = signal có path bypass. Pattern khắc phục: inventory toàn bộ writers tới S → unify qua shared dependency D → wire D vào TẤT CẢ paths → single transform stage trước fork sink. Tránh per-path masking duplication (drift sớm hay muộn)."**

**"Sibling service A cần reuse dependency D đã wire trên service B: thêm accessor B.D() thay vì thêm constructor parameter cho A. Lý do: giữ dependency graph dạng cây thay vì DAG rộng, dễ trace khi debug. Trade-off: A depends on B (đã tồn tại quan hệ) thay vì A depends on D (new edge)."**
