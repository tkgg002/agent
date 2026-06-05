# 03_implementation_audit — Audit Shadow Create Bugs

> Audit-only phase. KHÔNG thực thi code change. Tài liệu này ghi rõ HOW của fix để Muscle thực hiện ở phase sau (sau khi user approve `09_tasks_solution_audit.md`).

## Implementation map (per layer)

| Layer | File | Action |
|---|---|---|
| Worker | `centralized-data-service/internal/handler/command_handler.go` | Patch 2 chỗ: line 586-602 (CREATE TABLE) + line 149-179 (`ensureCDCColumnsInSchema`) |
| Worker | `centralized-data-service/internal/handler/command_handler.go` | Patch line 649: đổi caller `GetActiveRulesBySourceTable` → `ListActiveBySourceObject` |
| Worker | `centralized-data-service/internal/repository/mapping_rule_v2_repo.go` | (Optional) Deprecate `GetActiveRulesBySourceTable` hoặc thêm doc-comment cảnh báo cross-entity bleed |
| Service | `cdc-cms-service/internal/api/registry_handler_register.go` | (Verify) `CreateDefaultColumnsCommand` payload đã có `SourceObjectID` để worker dùng — KHÔNG sửa nếu đã có |

## Files NOT touched (đã verify đúng)
- `cdc-cms-web/**` — FE không leak field, không thiếu cột; chỉ submit form raw.
- `cdc-cms-service/internal/app/commands/register_registry.go` — flow đúng, dispatch đúng payload.
- `centralized-data-service/internal/sinkworker/schema_manager.go` — DDL builder runtime đã đúng (`_source_ts` line 231).
- `centralized-data-service/internal/sinkworker/upsert.go` — OCC guard đúng (line 69-122).
- DB schema `cdc_system.shadow_binding` / `mapping_rule_v2` — không cần migrate.

## DB cheat blocked (§6 Demand Elegance)
- ❌ KHÔNG `ALTER TABLE shadow_<conn>_<db>.sd_export_jobs_1 ADD COLUMN _source_ts BIGINT` thủ công cho từng shadow → workaround.
- ✅ FIX tại core path `HandleCreateDefaultColumns` để mọi shadow tương lai đều có. Shadow đã tạo lỗi: phase migration riêng (out-of-scope ở đây).

## Verification plan (sau khi áp dụng fix ở phase sau)
1. `go build ./...` ở 2 service (cdc-cms-service + centralized-data-service) — phải PASS.
2. `go vet ./...` — phải PASS.
3. Tạo shadow mới qua FE `/shadow`, kiểm tra schema PG: `psql -c "\d+ shadow_xxx.sd_test_yyy"` — phải có đủ 9 cột system (`_gpay_source_id`, `_raw_data`, `_source`, `_source_ts`, `_synced_at`, `_version`, `_hash`, `_gpay_deleted`, `_deleted`, `_created_at`, `_updated_at`).
4. Tạo 2 shadow với cùng `source_table` nhưng khác `target_table` → cả 2 shadow KHÔNG được cross-leak business columns của nhau (chỉ có cột system).
5. Khi user approve mapping_rule_v2 cho registry A → chỉ shadow của A được ALTER ADD; shadow của B (cùng source_table, khác registry) KHÔNG bị thêm.

## Pre-flight verify (§14)
- Tất cả file workspace đã tạo vật lý: `00_context.md`, `01_requirements.md`, `02_plan.md`, `03_implementation_audit.md`, `04_decisions.md`, `05_progress.md`, `06_validation.md`, `07_status.md`, `08_tasks_audit.md`, `09_tasks_solution_audit.md`, `10_gap_analysis.md`, `report_audit_shadow_create_bugs_2026-05-27.md`.
- §11: `05_progress.md` chỉ append, không overwrite.
- §12: phase này Muscle ở chế độ AUDIT — không sửa source code. Chờ user approve trước khi execute.
