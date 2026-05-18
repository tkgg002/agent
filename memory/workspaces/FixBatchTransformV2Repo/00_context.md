# 00_context — FixBatchTransformV2Repo

## Trigger
User report (2026-05-13 09:38 UTC):
```
09:38:22 13/5/2026 cmd-batch-transform centralized-export-service.export-jobs
shadow_centralized_export_service.sd_export_jobs error - - nats-command
no active mapping rules for table sd_export_jobs (source: export-jobs)
```
Test path: chạy transform từ FE / NATS publish `cdc.cmd.batch-transform` →
worker `HandleBatchTransform` → 0 rule match → fail.

## Surface
- Service: centralized-data-service worker (port 8082, /health 9090)
- File: `centralized-data-service/internal/handler/command_handler.go`
  → `HandleBatchTransform`
- Tables involved:
  - `cdc_system.cdc_mapping_rules` (V1, legacy, đã không còn ai ghi)
  - `cdc_system.mapping_rule_v2` (V2, FE/CMS ghi vào đây)
  - `shadow_centralized_export_service.sd_export_jobs` (target shadow)
  - `cdc_system.cdc_activity_log` (kết quả ghi vào partition daily)

## Constraint từ user
- Đọc lessons trước.
- Theo `agent/GEMINI.md` (role/skill).
- Chỉ làm đúng scope user yêu cầu.
- Report = số liệu thật, không bịa.
- Service phải chạy OK mới được báo done.
- Bắt buộc có file `report_*.md`.

## Liên hệ tới fix trước
Workspace `FixAlterColumnShadowSchema` (cùng ngày, sớm hơn) đã fix:
- `h.shadowDB` routing cho ALTER TABLE.
- Schema-qualification cho DDL.
- Per-command idempotency suffix.

Cùng pattern xuất hiện ở `HandleBatchTransform`:
1. Query `cdc_mapping_rules` (V1) thay vì `mapping_rule_v2` (V2) → 0 row.
2. `h.db.Exec` (destination plane 5434) thay vì `h.shadowDB` (5436) →
   `relation does not exist`.
3. Bare identifier `exportType` → PG fold lowercase → column not found.
4. JSONB column với cast `::TEXT` → `expression is of type text` 42804.
5. Mongo BSON Date stored as JSON number (epoch-ms) → `::TIMESTAMP` →
   `timestamp out of range` 22008.

## Definition of Done
- `nats pub cdc.cmd.batch-transform 'sd_export_jobs'` → activity log status=success,
  rows_affected > 0, error_message empty.
- Cột target trong `shadow_centralized_export_service.sd_export_jobs` được populate
  (không phải toàn NULL).
- `params` thực sự là JSONB (`pg_typeof = jsonb`), không phải text.
- `createdAt`/`lastUpdatedAt` được decode từ epoch-ms thành PG TIMESTAMP đúng.
- Cả 2 service (worker + cms) report `{"status":"ok"}` post-fix.
