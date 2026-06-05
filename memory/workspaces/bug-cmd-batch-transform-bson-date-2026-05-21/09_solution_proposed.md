# 09_solution_proposed — bug-cmd-batch-transform-bson-date-2026-05-21

## Decision
Patch `buildCastExpr` để cover 3 form value cho TIMESTAMP target:
1. **Number** (epoch ms) → `to_timestamp(/1000)` — branch hiện có.
2. **Object Extended JSON** `{"$date": "ISO"}` hoặc `{"$date": {"$numberLong": "epoch"}}` — branch MỚI.
3. **String** (ISO) — branch ELSE hiện có.

Tương tự, BSON Int64 cũng có form Extended JSON `{"$numberLong": "123"}` —
cover thêm cho `bigint/int8` để khỏi quay lại fix nữa.

Lựa chọn **SQL-side fix** (không phải write-time unwrap) vì:
- Rows cũ đã lưu sai form trong `_raw_data` — write-time fix chỉ giúp rows mới, rows cũ vẫn fail.
- SQL CASE backward-compatible: cover cả 3 form mà không cần migrate data.
- 1 helper bao tất cả caller (`HandleBatchTransform` + `HandleCreateMaterializedColumn`).

## Edit duy nhất
File: `centralized-data-service/internal/handler/command_handler.go`
Func: `buildCastExpr` (line 1647-1673).

### TIMESTAMP branch — thay bằng:
```go
case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
    // 3 form possible trong _raw_data JSONB:
    //   1. number (epoch ms)                       → to_timestamp(./1000)
    //   2. object {"$date": "ISO"}                 → ::TIMESTAMPTZ
    //   3. object {"$date": {"$numberLong": "ms"}} → to_timestamp(./1000)
    //   4. string ISO                              → ::TIMESTAMP (fallback)
    return fmt.Sprintf(
        "(CASE "+
            "WHEN jsonb_typeof(_raw_data->'%[1]s') = 'number' "+
                "THEN to_timestamp((_raw_data->>'%[1]s')::BIGINT / 1000.0) AT TIME ZONE 'UTC' "+
            "WHEN jsonb_typeof(_raw_data->'%[1]s') = 'object' "+
                 "AND jsonb_typeof(_raw_data->'%[1]s'->'$date') = 'string' "+
                "THEN (_raw_data->'%[1]s'->>'$date')::TIMESTAMPTZ "+
            "WHEN jsonb_typeof(_raw_data->'%[1]s') = 'object' "+
                 "AND jsonb_typeof(_raw_data->'%[1]s'->'$date'->'$numberLong') = 'string' "+
                "THEN to_timestamp((_raw_data->'%[1]s'->'$date'->>'$numberLong')::BIGINT / 1000.0) AT TIME ZONE 'UTC' "+
            "ELSE (_raw_data->>'%[1]s')::TIMESTAMP "+
        "END)",
        field)
```

### BIGINT branch — thêm extended-JSON support:
```go
case "int8", "bigint":
    return fmt.Sprintf(
        "(CASE "+
            "WHEN jsonb_typeof(_raw_data->'%[1]s') = 'object' "+
                 "AND jsonb_typeof(_raw_data->'%[1]s'->'$numberLong') = 'string' "+
                "THEN (_raw_data->'%[1]s'->'$numberLong')::TEXT::BIGINT "+
            "ELSE (_raw_data->>'%[1]s')::BIGINT "+
        "END)",
        field)
```

## Test plan (unit, pure-string assertion)
Tạo file `command_handler_cast_expr_test.go`:
- TIMESTAMP → assert SQL chứa `jsonb_typeof = 'object'` + `'$date'`.
- BIGINT → assert SQL chứa `'$numberLong'`.
- JSONB / JSON / INTEGER / TEXT — assert giữ behavior cũ.

Test pure string compare, không cần DB.

## Verify gates
- `go build ./...`
- `go vet ./...`
- `go test ./internal/handler/...`

## Smoke
1. Restart cdc-worker.
2. Bấm Transform Now (cmd-batch-transform) cho `payment-bill-service.payment-bills`.
3. Expect activity_log row status=success, không còn 22007.

## Out of scope (defer)
- Write-time unwrap BSON trước khi lưu `_raw_data`: invasive, chạm
  snapshot.v2 + kafka_consumer + có thể stream cũng cần coordinate.
  Backward-compatible SQL fix bao cả rows cũ → defer write-time cleanup
  nếu performance trở thành issue.
