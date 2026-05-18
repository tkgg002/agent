# 00 — Context: FixAlterColumnShadowSchema

## Yêu cầu user (2026-05-12 17:10 ICT)
> POST `http://localhost:8083/api/mapping-rules/batch` — duyệt field mới ở
> `http://localhost:5173/shadow/1/mappings`. Không thấy chạy.
>
> Worker activity log (17:10:56 12/5/2026):
> `alter-column | centralized-export-service.export-jobs | shadow_centralized_export_service.sd_export_jobs | error | nats-command | ERROR: relation "sd_export_jobs" does not exist (SQLSTATE 42P01)`

## Triệu chứng
Worker khi nhận `cdc.cmd.alter-column` cố `ALTER TABLE "sd_export_jobs"` (bare,
không có schema) → fallback `search_path` không có `shadow_centralized_export_service`
→ relation không tồn tại trong `public` → fail.

## Lessons match (đã đọc trước khi sửa, theo CLAUDE.md §7)
- **2026-04-28** (line 1238) — Schema rename ↔ search_path coupling: bare SQL
  fall back về search_path mặc định. **Đúng**: schema-qualify rõ ràng.
- **2026-05-11** (line 2682) — GORM TableName mixed qualification + role
  search_path trap. **Global Pattern X**: đặt search_path ở DSN/session level,
  migrations LUÔN schema-qualify rõ ràng (`schema.table`).

## Root cause hypothesis
1. CMS `BatchUpdate` (mapping_rule_handler_batch.go:48) dispatch
   `AlterColumnCommand{TargetTable: *rule.ShadowTable}` — KHÔNG truyền schema.
2. NATS subject `cdc.cmd.alter-column` → worker `HandleAlterColumn`
   (`command_handler.go:1672`) build `ALTER TABLE "%s"` không qualify.
3. Worker DB connection → `shadow_centralized_export_service.sd_export_jobs`
   tồn tại thật, nhưng search_path mặc định không bao gồm schema này.

## Definition of Done
- POST `/api/mapping-rules/batch` trả 202 (giữ nguyên contract)
- Worker thực thi `ALTER TABLE "shadow_centralized_export_service"."sd_export_jobs" ADD COLUMN ...` thành công
- Activity log mới có entry `alter-column | success`
- Cột mới xuất hiện trong DB (verify bằng `\d` hoặc `information_schema.columns`)
- KHÔNG break các flow khác (build pass, no test regression)

## Files dự kiến sửa
1. `cdc-cms-service/internal/app/commands/source_async.go` — thêm `TargetSchema` vào `AlterColumnCommand`
2. `cdc-cms-service/internal/api/mapping_rule_handler_batch.go` — pass `*rule.ShadowSchema`
3. `centralized-data-service/internal/handler/command_handler.go` — payload thêm `TargetSchema`, build qualified SQL

## Files KHÔNG sửa
- Search_path trên role/DSN — không can thiệp vì lesson 2026-05-11 nói role-level setting nguy hiểm; per-statement qualify là đường an toàn nhất.
