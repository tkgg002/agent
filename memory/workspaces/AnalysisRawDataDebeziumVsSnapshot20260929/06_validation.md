# 06 Validation: Bằng Chứng Kiểm Thử Đồng Bộ Debezium và Bảo Toàn BSON Wrapper cho JSONB

## 1. Kết Quả Kiểm Thử Tự Động (Automated Test Execution)

### Test Suite 1: `internal/handler/shadow/avro_helper_test.go`
- **Lệnh thực thi:**
  ```bash
  go test -v ./internal/handler/shadow -run TestUnwrapAvroUnion
  ```
- **Kết quả (PASS 100%):**
  ```text
  === RUN   TestUnwrapAvroUnion_StandardUnion
  === RUN   TestUnwrapAvroUnion_StandardUnion/nil
  === RUN   TestUnwrapAvroUnion_StandardUnion/scalar_string
  === RUN   TestUnwrapAvroUnion_StandardUnion/scalar_int
  === RUN   TestUnwrapAvroUnion_StandardUnion/avro_string_union
  === RUN   TestUnwrapAvroUnion_StandardUnion/avro_long_union
  === RUN   TestUnwrapAvroUnion_StandardUnion/avro_int_union
  === RUN   TestUnwrapAvroUnion_StandardUnion/avro_double_union
  === RUN   TestUnwrapAvroUnion_StandardUnion/multi-key_map_untouched
  --- PASS: TestUnwrapAvroUnion_StandardUnion (0.00s)
  === RUN   TestUnwrapAvroUnion_BSONExtJSONPreserved
  === RUN   TestUnwrapAvroUnion_BSONExtJSONPreserved/$oid_wrapper
  === RUN   TestUnwrapAvroUnion_BSONExtJSONPreserved/$date_epoch_wrapper
  === RUN   TestUnwrapAvroUnion_BSONExtJSONPreserved/$date_ISO_string_wrapper
  === RUN   TestUnwrapAvroUnion_BSONExtJSONPreserved/$numberLong_wrapper
  === RUN   TestUnwrapAvroUnion_BSONExtJSONPreserved/$numberDecimal_wrapper
  --- PASS: TestUnwrapAvroUnion_BSONExtJSONPreserved (0.00s)
  === RUN   TestUnwrapAvroUnionMap_MixedPayload
  --- PASS: TestUnwrapAvroUnionMap_MixedPayload (0.00s)
  PASS
  ok  	centralized-data-service/internal/handler/shadow	1.052s
  ```

### Test Suite 2: `test/internal/service/schema_adapter_coerce_test.go`
- **Lệnh thực thi:**
  ```bash
  go test -v ./test/internal/service/schema_adapter_coerce_test.go
  ```
- **Kết quả (PASS 100%):**
  ```text
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
  ok  	command-line-arguments	0.701s
  ```

### Test Suite 3: `internal/service/master/transmute/...`
- **Lệnh thực thi:**
  ```bash
  go test -v ./internal/service/master/transmute/...
  ```
- **Kết quả (PASS 100%):**
  ```text
  === RUN   TestRegistryHasBuiltins
  --- PASS: TestRegistryHasBuiltins (0.00s)
  === RUN   TestGetFallsBackToCopy
  --- PASS: TestGetFallsBackToCopy (0.00s)
  === RUN   TestCopyOneToOne
  --- PASS: TestCopyOneToOne (0.00s)
  === RUN   TestFlattenExplodes
  --- PASS: TestFlattenExplodes (0.00s)
  === RUN   TestFlattenDotNotationExtJSON
  --- PASS: TestFlattenDotNotationExtJSON (0.00s)
  === RUN   TestFlattenDualMasterParentAndElem
  --- PASS: TestFlattenDualMasterParentAndElem (0.00s)
  === RUN   TestFlattenStringifiedJSONArray
  --- PASS: TestFlattenStringifiedJSONArray (0.00s)
  === RUN   TestFlattenValidateSpec
  --- PASS: TestFlattenValidateSpec (0.00s)
  === RUN   TestListSorted
  --- PASS: TestListSorted (0.00s)
  PASS
  ok  	centralized-data-service/internal/service/master/transmute	(cached)
  ```

### Test Suite 4: Kiểm tra biên dịch (Compiler Check)
- **Lệnh thực thi:**
  ```bash
  go build ./cmd/worker
  go build ./cmd/admin-api
  go build ./cmd/sinkworker
  ```
- **Kết quả:** Cả 3 service nhị phân đều biên dịch thành công 100% không cảnh báo (Exit Code = 0).
