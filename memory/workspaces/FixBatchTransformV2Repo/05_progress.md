# 05_progress — FixBatchTransformV2Repo (APPEND ONLY per CLAUDE.md §11)

- [2026-05-13 09:38] [Muscle:CC] Bug nhận: `HandleBatchTransform` báo
  `no active mapping rules for table sd_export_jobs (source: export-jobs)`.
  Trace `command_handler.go:993` → đang query V1 repo (`cdc_mapping_rules`).
  CMS đang ghi V2 (`mapping_rule_v2`). Lệch source-of-truth.
- [2026-05-13 09:42] [Muscle:CC] Edit #1 — `command_handler.go:993` đổi
  `h.mappingRepo.GetActiveRulesByTargetTable` → `h.mappingV2Repo.GetActiveRulesBySourceTable`.
  Verify: V2 repo filter `is_active=true AND status='approved'` JOIN
  `source_object_registry` ON `source_object_name`. Đúng semantics.
- [2026-05-13 09:43] [Muscle:CC] Restart worker. Test: `nats pub
  cdc.cmd.batch-transform sd_export_jobs` → log "found 17 active rules".
  Tốt, nhưng nhảy sang lỗi mới.
- [2026-05-13 09:45] [Muscle:CC] Lỗi #2: `relation
  "shadow_centralized_export_service.sd_export_jobs" does not exist`.
  Cùng pattern FixAlterColumnShadowSchema: `h.db` (5434 dest) thay vì
  `h.shadowDB` (5436 cdc_shadow). Edit #2 — line 1037 thêm
  `execDB := h.db; if h.shadowDB != nil { execDB = h.shadowDB }`.
  Build pass.
- [2026-05-13 09:47] [Muscle:CC] Restart + retest. Lỗi #3: `column
  "exporttype" does not exist`. PG fold-lowercase identifier.
  Edit #3 — line 1027/1031 wrap với `quoteCommandIdent(rule.TargetColumn)`
  cho cả SET clause và WHERE NULL clause. Build pass.
- [2026-05-13 09:50] [Muscle:CC] Restart + retest. Lỗi #4: `column "params"
  is of type jsonb but expression is of type text (SQLSTATE 42804)`.
  `buildCastExpr` không có nhánh JSONB. Edit #4 — `command_handler.go:1224`
  thêm `case "jsonb": return (_raw_data->'%s')` (dùng `->` thay vì `->>`).
  Build pass.
- [2026-05-13 09:57] [Muscle:CC] Restart + retest. Lỗi #5: `timestamp out
  of range: "1778482050803" (SQLSTATE 22008)`. Inspect
  `_raw_data->'createdAt'` → `jsonb_typeof = number`. Mongo BSON Date
  serialized thành JSON number (epoch-ms).
- [2026-05-13 09:58] [Muscle:CC] Edit #5 — `buildCastExpr` nhánh
  timestamp đổi sang `CASE WHEN jsonb_typeof = 'number' THEN
  to_timestamp(::BIGINT/1000.0) AT TIME ZONE 'UTC' ELSE ::TIMESTAMP END`.
  Đồng thời tách `int4` vs `int8` (rule data_type=BIGINT bị cast thành
  INTEGER → có thể overflow với epoch-ms). Build pass.
- [2026-05-13 10:01] [Muscle:CC] Restart + retest cuối: NATS publish OK
  → log `batch transform completed table=sd_export_jobs rows_affected=129`
  → activity_log entry id=34 `status=success rows=129 error=null`. ✓
- [2026-05-13 10:02] [Muscle:CC] DB verify:
  - `pg_typeof(params)=jsonb`, `jsonb_typeof(params)=string` (giá trị
    `"testValue"` etc) → JSONB hợp lệ.
  - `createdAt = 2026-04-16 01:58:45.976` (decode từ `1778482050803` ms,
    1778482050.803s ÷ 86400 → 2026-04-16, đúng).
  - 129 row, 128/129 cột exportType, 127/129 params, 1/129 jobName, 2/129
    add_field_alter_2 (số thấp do field mới approve, chỉ có vài row test
    có key đó trong `_raw_data`).
- [2026-05-13 10:02] [Muscle:CC] Service health: worker (8082)
  `{"service":"cdc-worker","status":"ok"}`, CMS (8083)
  `{"service":"cdc-cms","status":"ok"}`. ✓
