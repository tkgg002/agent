# 00_context — FE Humanize API Errors

## Trigger
User báo lỗi raw từ BE rò ra UI:
```json
{
  "error": "failed to register table: ERROR: duplicate key value violates unique constraint \"cdc_table_registry_conn_source_db_table_target_key\" (SQLSTATE 23505)"
}
```
Hiển thị nguyên văn cho operator → khó đọc, không nói được lỗi gì.

## Yêu cầu user
- Thêm bộ handler cho lỗi này.
- **FE only**, không sửa API.
- Quét toàn bộ chỗ show error → format lại cho hợp lý.

## Stack
- FE: `data-hub/cdc-cms-web` (Vite + React 19 + AntD v6 + Axios).
- BE: trả raw GORM/Postgres error qua `{ "error": "..." }` (đa số endpoint) hoặc `{ "detail": "..." }` (vài endpoint Wizard/Connectors).
- Lỗi thường gặp:
  - SQLSTATE 23505 unique violation (kèm constraint name).
  - SQLSTATE 23503 FK, 23502 NOT NULL, 23514 CHECK.
  - 42P01 undefined_table, 42703 undefined_column, 42P07 duplicate_table.
  - 40P01 deadlock, 40001 serialization.
  - 22P02 invalid input, 22001 string overflow.
  - 08006 connection failure, 57014 query canceled, 53300 too many connections.
  - HTTP 4xx/5xx khi BE không trả body.
  - Network: ECONNABORTED (timeout), ERR_NETWORK (down).

## Root cause cách show lỗi cũ
- Pattern `e.response?.data?.error || 'fallback'` chỉ relay raw text → không decode SQLSTATE, không biết constraint nào.
- Một số nơi (`Connectors`, `Wizard`) ưu tiên `detail`; một số ưu tiên `error`. Không nhất quán.
- Một số onError chỉ in fallback static ('Toggle failed') → mất thông tin lỗi BE.
