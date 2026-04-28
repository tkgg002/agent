# Implementation — Phase 15 Retry Failed Scope Enrichment

## Audit kết luận

- `retry failed` khác `check/heal`:
  - `check/heal` cần resolve theo scope vì caller chọn target theo ngữ cảnh
  - `retry failed` đã có `failed_log_id` là canonical identity
- Downstream worker `HandleRetryFailed` chỉ đọc:
  - `failed_log_id`
  - `target_table`
  - `record_id`
  - `raw_json`
  nên có thể thêm field mới vào payload mà không phá compatibility

## Thay đổi đã áp dụng

- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/reconciliation_handler.go`
  - thêm swagger annotation cho `POST /api/failed-sync-logs/{id}/retry`
  - enrich retry response với:
    - `source_database`
    - `source_table`
    - `shadow_schema`
    - `shadow_table`
    - `scope_ambiguous`
  - enrich payload NATS `cdc.cmd.retry-failed` với source/shadow metadata tương tự
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/hooks/useReconStatus.ts`
  - cập nhật docs/comment cho retry mutation
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/DataIntegrity.tsx`
  - failed logs table ưu tiên render source/shadow metadata từ chính failed log record
  - chỉ fallback sang `reportByTarget` khi cần

## Kết luận

Retry endpoint **không cần** đổi sang scope-aware input.
Thứ cần là **scope-aware output + payload enrichment** để operator-flow rõ nghĩa hơn.
