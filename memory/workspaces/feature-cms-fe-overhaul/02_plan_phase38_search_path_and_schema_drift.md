# Phase 38 — Plan: search_path + schema-drift recovery

## Bước 1 — Patch search_path (DB role)
- Apply: `ALTER ROLE "user" SET search_path = cdc_system, public;`
- Lưu thành migration `centralized-data-service/migrations/039_set_search_path.sql`.
- Restart cả CMS + Worker để session connection áp dụng search_path mới.
- Expected: nhóm lỗi 1 (cdc_table_registry, failed_sync_logs, cdc_activity_log,
  cdc_reconciliation_report) tan biến.

## Bước 2 — Replace `cdc_internal` → `cdc_system` (code-only)
- Phạm vi giới hạn vào 2 handler có endpoint trong list 11:
  - `internal/api/schema_proposal_handler.go` — 7 occurrence, dùng `Table()` +
    raw `UPDATE`.
  - `internal/api/transmute_schedule_handler.go` — 4 occurrence trong List/
    Create/Toggle/RunNow.
- Không đụng `shadow_automator.go`, `mapping_preview_handler.go` (out of scope,
  ghi vào debt).

## Bước 3 — Reshape `transmute_schedule` List + RunNow
- Bảng có `master_binding_id` (FK), không có `master_table`. Rewrite List và
  RunNow thành raw SQL JOIN `cdc_system.master_binding` để expose
  `mb.master_table` về struct `ScheduleRow.MasterTable`.
- Giữ nguyên contract API.

## Bước 4 — Dọn `tr.sync_status` & `tr.recon_drift`
- Patch `source_objects_handler.go`:
  - Xóa `tr.sync_status,` trong COALESCE. Để CASE fallback (`source_error|drift|
    healthy|unknown`) làm nguồn duy nhất.
  - Replace `COALESCE(rr.diff, tr.recon_drift, 0)` → `COALESCE(rr.diff, 0)`.
- Áp dụng cho 3 query (List, ListShadowBindings, GetMappingContext).

## Bước 5 — Fix GORM nested-struct scan cho WorkerScheduleResponse
- Thêm struct flat `workerScheduleScanRow` với 22 field tương ứng SELECT.
- Trong `listResponses`: `Scan(&scan)` rồi map sang `[]WorkerScheduleResponse`
  với sub-struct `Scope` (transpose tay).
- Qualify `cdc_worker_schedule` thành `cdc_system.cdc_worker_schedule` cho
  rõ ràng (không phụ thuộc search_path).

## Bước 6 — Verify
- `go build ./...` cả 2 service.
- Restart CMS, login `admin/admin123` (auth-service yêu cầu `username`, không
  phải email).
- Curl 11 URLs với header `Authorization: Bearer $TOKEN`. Kỳ vọng 11/11 = 200.
- Auto-flow probe:
  - `curl http://localhost:18083/connectors/<name>/status` → RUNNING.
  - `docker exec gpay-kafka kafka-topics --list` → có topic `cdc.goopay.*`.
  - `tail /tmp/cdc-worker.log` → không panic, có loop `discoverTopics`.
- Document mismatch giữa Debezium `collection.include.list` và registry như debt.

## Bước 7 — Document & Lesson
- Phase 38 docs (00..09) prefix đúng convention.
- Append `05_progress.md` (immutable log).
- 2 lesson mới vào `agent/memory/global/lessons.md`:
  - Global Pattern: schema rename phải đi kèm search_path (hoặc qualify mọi
    GORM `TableName`).
  - Global Pattern: GORM `Raw().Scan` không lan vào nested struct → flat scan
    + transpose.
