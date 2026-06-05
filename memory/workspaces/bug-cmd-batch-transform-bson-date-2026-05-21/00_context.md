# 00_context — bug-cmd-batch-transform-bson-date-2026-05-21

## Symptom
User dán activity_log row:

```
15:13:43 21/5/2026  cmd-batch-transform
  payment-bill-service.payment-bills
  → shadow_goopay_pbs_payment_bill_service.sd_payment_bills
  status=error  trigger=nats-command
  error: ERROR: invalid input syntax for type timestamp:
         "{"$date": "***T08:53:41.741Z"}" (SQLSTATE 22007)

15:13:43 21/5/2026  transform   (cùng table cùng giờ)
  status=success  rows=1  trigger=scheduler
```

## Root cause
`centralized-data-service/internal/handler/command_handler.go:1647-1673` →
`buildCastExpr(field, "timestamp")` emit:

```sql
CASE WHEN jsonb_typeof(_raw_data->'field') = 'number'
     THEN to_timestamp((_raw_data->>'field')::BIGINT / 1000.0) AT TIME ZONE 'UTC'
     ELSE (_raw_data->>'field')::TIMESTAMP
END
```

Chỉ handle 2 form:
1. `number` (epoch ms) → ok
2. ELSE → cast text → ok cho ISO string, FAIL cho object `{"$date": "..."}`

MongoDB BSON Date sau khi qua mongo-driver/bsonjson Marshal sinh ra form
**Extended JSON** = JSON object `{"$date": "2026-05-21T08:53:41.741Z"}`.
PG `->>` trên object trả về string serialized `{"$date": "..."}` → cast
TIMESTAMP fail SQLSTATE 22007.

## Vì sao 2 path khác nhau
- `transform` (scheduler / per-row): chạy trên rows MỚI insert qua
  snapshot.v2/Debezium path. Tại đó BSON Date đã được driver Go unwrap
  → JSON field là **number** (epoch ms) → branch #1 đi qua được.
- `cmd-batch-transform` (nats-command / bulk re-transform): chạy lại trên
  rows CŨ trong `_raw_data` đã lưu trước đây ở form object → branch #2 fail.

## Affected file
- `centralized-data-service/internal/handler/command_handler.go` —
  func `buildCastExpr` (line 1647-1673).

Caller cùng helper:
- `command_handler.go:1048` — `HandleCreateMaterializedColumn` (single col).
- `command_handler.go:1303` — `HandleBatchTransform` (bulk SET).

→ Fix tại helper bao cover cả 2 đường gọi.

## Scope
1 helper, 1 file. Không đụng schema, không đụng FE, không cần migration.
