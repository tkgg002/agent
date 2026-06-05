# 05_progress — bug-cmd-batch-transform-bson-date-2026-05-21 (APPEND-ONLY)

## 2026-05-21 15:30 ICT — Fix applied, gates PASS

**Symptom**: User report log activity_log:
```
cmd-batch-transform  payment-bill-service.payment-bills → shadow_..sd_payment_bills
status=error  trigger=nats-command
ERROR: invalid input syntax for type timestamp:
       "{"$date": "***T08:53:41.741Z"}" (SQLSTATE 22007)
```
Same minute, `transform` (per-row trigger=scheduler) success rows=1.

**Root cause** (see 00_context.md + 09_solution_proposed.md):
`buildCastExpr` cho TIMESTAMP target chỉ handle 2 form trong _raw_data JSONB:
1. `number` (epoch ms) — branch ok.
2. ELSE → `(text)::TIMESTAMP` — fail trên object Extended-JSON `{"$date": "..."}`.

MongoDB BSON Date qua canonical encoder lưu vào _raw_data ở form object →
PG `->>` serialize thành string `{"$date": "..."}` → cast TIMESTAMP fail 22007.

**Vì sao 2 path khác nhau**:
- scheduler/transform: chạy ngay sau insert mới (rows có epoch ms hoặc ISO string).
- nats-command/cmd-batch-transform: bulk re-transform trên rows cũ với form object.

**Edits landed**:
- `centralized-data-service/internal/handler/command_handler.go` —
  `buildCastExpr` thêm CASE branches cho:
  - TIMESTAMP: object `{"$date": "ISO"}` → `::TIMESTAMPTZ`;
    object `{"$date": {"$numberLong": "ms"}}` → `to_timestamp(/1000)`.
  - BIGINT/INT8: object `{"$numberLong": "123"}` → `($numberLong-text)::BIGINT`.
  Backward-compatible: branch number/text giữ nguyên, chỉ thêm CASE WHEN mới
  ở đầu để cover form object trước.
- `centralized-data-service/internal/handler/command_handler_cast_expr_test.go` (NEW) —
  pure-string unit tests assert SQL contains `$date` / `$numberLong` branches
  cho timestamp/timestamptz/bigint/int8; UnchangedTypes test confirm
  jsonb/json/integer/numeric/boolean/text không bị regress.

**Verify gates** (cdc-cms-worker == centralized-data-service):
- `go build ./...` → EXIT 0.
- `go vet ./...` → EXIT 0.
- `go test -run TestBuildCastExpr ./internal/handler/...` → ok 0.772s, EXIT 0.
- `go test ./internal/handler/...` (full suite) → ok 3.342s, EXIT 0.

**Cần user smoke**:
1. Ctrl-C cdc-worker hiện tại + `go run cmd/worker/main.go` lại (binary mới).
2. Bấm "Transform Now" (cmd-batch-transform) trên row
   `payment-bill-service.payment-bills` → `sd_payment_bills`.
3. Expect:
   - `cdc_activity_log` row mới: `operation='cmd-batch-transform'`
     `status='success'` `rows_affected=N`, KHÔNG còn 22007.
   - PG column timestamp đích (vd `_updated_at`, `created_at`) có giá trị
     ISO/epoch hợp lệ cho rows trước đây fail.

**Halt**: chờ user restart worker + smoke.

**Out of scope (đã defer)**:
- Write-time unwrap BSON tại snapshot.v2 / kafka_consumer trước khi lưu
  _raw_data. SQL fix bao cả rows cũ; defer write-time cleanup nếu sau này
  performance trở thành issue (object lookup ~2-3 jsonb_typeof calls/row).
