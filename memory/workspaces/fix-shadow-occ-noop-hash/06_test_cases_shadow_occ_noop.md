# Test Cases & Verification Evidence: Fix Shadow Table OCC No-Op Hash Gate

## Test Suite Results
Package: `centralized-data-service/test/internal/service`
Run Command: `go test -v ./test/internal/service/schema_adapter_test.go ./test/internal/service/schema_adapter_ordering_test.go ./test/internal/service/schema_adapter_coerce_test.go`

### Bằng chứng chạy Test thực tế:
```
=== RUN   TestBuildUpsertSQL_PopulatesGpaySourceID
--- PASS: TestBuildUpsertSQL_PopulatesGpaySourceID (0.00s)
=== RUN   TestBuildUpsertSQL_PKIsSourceID_NoDuplicateColumn
--- PASS: TestBuildUpsertSQL_PKIsSourceID_NoDuplicateColumn (0.00s)
=== RUN   TestBuildUpsertSQL_LWWGuard
--- PASS: TestBuildUpsertSQL_LWWGuard (0.00s)
=== RUN   TestBuildSoftDeleteUpdateSQL
--- PASS: TestBuildSoftDeleteUpdateSQL (0.00s)
=== RUN   TestBuildSoftDeleteInsertSQL
--- PASS: TestBuildSoftDeleteInsertSQL (0.00s)
=== RUN   TestEventOrdering_OlderTsIgnored
--- PASS: TestEventOrdering_OlderTsIgnored (0.00s)
=== RUN   TestEventOrdering_HashTiebreaker
--- PASS: TestEventOrdering_HashTiebreaker (0.00s)
=== RUN   TestEventOrdering_DeleteTombstone
--- PASS: TestEventOrdering_DeleteTombstone (0.00s)
=== RUN   TestEventOrdering_InsertAfterDelete_Resurrection
--- PASS: TestEventOrdering_InsertAfterDelete_Resurrection (0.00s)
=== RUN   TestEventOrdering_UpdateAfterDelete_OCCDrop
--- PASS: TestEventOrdering_UpdateAfterDelete_OCCDrop (0.00s)
=== RUN   TestEventOrdering_SameDataSnapshot_NoOp
--- PASS: TestEventOrdering_SameDataSnapshot_NoOp (0.00s)
=== RUN   TestSchemaAdapter_IsJSONB
--- PASS: TestSchemaAdapter_IsJSONB (0.00s)
=== RUN   TestSchemaAdapter_CoerceValue_Text
--- PASS: TestSchemaAdapter_CoerceValue_Text (0.00s)
=== RUN   TestSchemaAdapter_CoerceValue_Int
--- PASS: TestSchemaAdapter_CoerceValue_Int (0.00s)
=== RUN   TestSchemaAdapter_CoerceValue_Float
--- PASS: TestSchemaAdapter_CoerceValue_Float (0.00s)
=== RUN   TestSchemaAdapter_CoerceValue_Bool
--- PASS: TestSchemaAdapter_CoerceValue_Bool (0.00s)
=== RUN   TestSchemaAdapter_CoerceValue_JSON
--- PASS: TestSchemaAdapter_CoerceValue_JSON (0.00s)
=== RUN   TestSchemaAdapter_CoerceValue_Time
--- PASS: TestSchemaAdapter_CoerceValue_Time (0.00s)
PASS
ok  	command-line-arguments	0.691s
```

### Chi tiết kịch bản No-Op đã chứng minh:
- **Kịch bản 1:** Lần đầu Insert snapshot 1 (ts=1000) -> `RowsAffected = 1` (Insert thành công).
- **Kịch bản 2:** Re-snapshot cùng dữ liệu (hash giống hệt), timestamp mới hơn (ts=5000 > 1000) -> `RowsAffected = 0` (No-Op hoàn toàn, 0 disk write, 0 dead tuples).
- **Kịch bản 3:** Re-snapshot dữ liệu có sửa đổi (name đổi -> hash đổi), timestamp mới hơn (ts=6000) -> `RowsAffected = 1` (Update thành công).
