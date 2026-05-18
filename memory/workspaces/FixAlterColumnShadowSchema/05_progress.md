# Progress Log — FixAlterColumnShadowSchema

## Progress History (APPEND ONLY per CLAUDE.md §11)
- [2026-05-12 17:14] [Muscle:CC] Init workspace. Đọc lessons.md (2026-04-28 line 1238 + 2026-05-11 line 2682) — match đúng pattern: bare table name + search_path fallback → relation not found. Plan: schema-qualify ALTER TABLE statement.
- [2026-05-12 17:14] [Muscle:CC] Trace flow: FE `/shadow/1/mappings` → CMS `BatchUpdate` (mapping_rule_handler_batch.go:48) dispatch `AlterColumnCommand{TargetTable}` (KHÔNG có schema) → NATS `cdc.cmd.alter-column` → worker `HandleAlterColumn` (command_handler.go:1672) build `ALTER TABLE "<table>"` bare → fail.
- [2026-05-12 17:14] [Muscle:CC] Verified: `Rule.ShadowSchema *string` đã có sẵn trong domain (rule.go:42), repo populate (mapping_rule_repo_gorm.go:71). Chỉ thiếu việc truyền xuống command + payload + SQL builder.
- [2026-05-13 02:20] [Muscle:CC] Edit #1 — `cdc-cms-service/internal/app/commands/source_async.go:116-127` thêm field `TargetSchema` vào `AlterColumnCommand`.
- [2026-05-13 02:20] [Muscle:CC] Edit #2 — `cdc-cms-service/internal/api/mapping_rule_handler_batch.go:48-50` truyền `*rule.ShadowSchema` xuống command. Build pass.
- [2026-05-13 02:20] [Muscle:CC] Edit #3 — `centralized-data-service/internal/handler/command_handler.go:1672-1734` add `TargetSchema` payload, validate `isSafeIdent`, build `qualifiedTable = "schema"."table"` cho 3 actions (add/drop/alter_type). Build pass.
- [2026-05-13 02:20] [Muscle:CC] Restart cả 2 service (kill + nohup go run). Cả 2 lên healthy.
- [2026-05-13 02:14] [Muscle:CC] Test #1 (with Idempotency-Key): POST batch ko thấy alter-column job mới. Trace → discovered `cdc_jobs.idempotency_key` UNIQUE → cùng key cho update + alter → second Dispatch short-circuit silently.
- [2026-05-13 02:14] [Muscle:CC] Test #2 (without Idempotency-Key): alter-column job được tạo, worker exec → activity log error đổi từ `relation does not exist` → `schema "shadow_centralized_export_service" does not exist`. Confirm fix qualify schema OK; bug mới: worker `h.db` (port 5434 goopay_dest) không có schema này — chỉ tồn tại ở `h.shadowDB` (port 5436 cdc_shadow).
- [2026-05-13 02:18] [Muscle:CC] Edit #4 — worker `HandleAlterColumn` đổi `h.db.Exec(sql)` → `execDB := h.shadowDB if h.shadowDB != nil else h.db` (consistency với line 150).
- [2026-05-13 02:18] [Muscle:CC] Edit #5 — CMS `BatchUpdate` thêm per-command idempotency suffix: `:status`, `:alter`, `:backfill`. Thêm `context` import. Build pass.
- [2026-05-13 02:20] [Muscle:CC] Test #3 (with Idempotency-Key, all fixes): rule 17 approve → activity log entry 54 status=success, action=add, column=add_field_alter_2 → `\d shadow_centralized_export_service.sd_export_jobs` xác nhận cột mới. ✓
- [2026-05-13 02:21] [Muscle:CC] Discovered ANOTHER pre-existing bug: rule status vẫn `pending` sau approve. Trace: `update_mapping_rule.go:67` query `SELECT shadow_table FROM cdc_system.mapping_rule_v2` → 42703 column does not exist (column moved to `shadow_binding` table).
- [2026-05-13 02:22] [Muscle:CC] Edit #6 — `update_mapping_rule.go:62-79` đổi query dùng JOIN `shadow_binding sb ON sb.source_object_id = mr.source_object_id`. Build pass.
- [2026-05-13 02:23] [Muscle:CC] Restart CMS + Test #4: rule 16 approve → status=`approved`, column `jobName` xuất hiện, update-status job = success. ✓
- [2026-05-13 02:24] [Muscle:CC] Test #5 (rule 17): cũng approved thành công. Service health: worker (8082+9090) + cms (8083) đều `{"status":"ok"}`. ✓
