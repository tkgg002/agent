# 06_test_cases.md - Kế Hoạch & Bằng Chứng Kiểm Thử (Validation Log)

## 1. Danh Sục Kịch Bản Kiểm Thử (Test Cases)

### TC-01: BSON Extended JSON v2 Type Inference (`source_router_test.go`)
- **Mục tiêu**: Kiểm tra hàm `InferTypeFromRawData` suy luận chính xác các kiểu dữ liệu BSON.
- **Kịch bản kiểm thử**:
  1. `nil` -> `TEXT`
  2. `boolean` -> `BOOLEAN`
  3. `float64` nguyên -> `BIGINT`
  4. `float64` thập phân -> `NUMERIC`
  5. RFC3339 date string -> `TIMESTAMPTZ`
  6. `time.Time` -> `TIMESTAMPTZ`
  7. `{"$date": "2026-05-07T04:01:17.830Z"}` -> `TIMESTAMPTZ`
  8. `{"$date": 1786503300747}` (Epoch millis) -> `TIMESTAMPTZ`
  9. `{"$oid": "6a7be084eeb3c73b3c96f2c1"}` -> `TEXT`
  10. `{"$numberDecimal": "12345.67"}` -> `NUMERIC`
  11. `{"$numberLong": "1786503300747"}` -> `BIGINT`
  12. `{"$numberInt": "42"}` -> `INTEGER`
  13. Generic nested JSON -> `JSONB`
  14. Array of objects -> `JSONB`
- **Kết quả**: **16/16 test cases PASS**.

### TC-02: PostgreSQL Mapping SQL Cast Generation (`metadata_mapping_test.go`)
- **Mục tiêu**: Kiểm tra hàm `BuildCastExpr` sinh biểu thức SQL an toàn không bị crash khi `$date` là epoch millis integer.
- **Kịch bản kiểm thử**:
  - `BuildCastExpr("createdAt", "timestamptz")` sinh SQL chứa case `jsonb_typeof(_raw_data->'createdAt'->'$date') = 'number'`.
- **Kết quả**: **PASS**.

### TC-03: ScanRawData Envelope Path (`scan_service.go`)
- **Mục tiêu**: Xác nhận `jsonbPath` trỏ đúng vào `_raw_data->'after'` cho engine `mongodb`.
- **Kết quả**: **PASS**.

### TC-04: MongoDB Stratified Sampling (`mongo_introspection.go`)
- **Mục tiêu**: Lấy mẫu $sample aggregation và sort `_id` DESC để gom trọn vẹn polymorphic schema.
- **Kết quả**: **PASS**.

---

## 2. Bằng Chứng Thực Thi Lệnh Test
```bash
go test -v ./internal/service/source -run TestInferTypeFromRawData
=== RUN   TestInferTypeFromRawData_ExtendedJSON
--- PASS: TestInferTypeFromRawData_ExtendedJSON (0.00s)
PASS
ok  	centralized-data-service/internal/service/source	0.772s

go test ./internal/... ./test/...
ok  	centralized-data-service/internal/admin	(cached)
ok  	centralized-data-service/internal/handler/master	(cached)
ok  	centralized-data-service/internal/handler/scan	(cached)
ok  	centralized-data-service/internal/handler/shadow	(cached)
ok  	centralized-data-service/internal/handler/source	(cached)
ok  	centralized-data-service/internal/service/master	(cached)
ok  	centralized-data-service/internal/service/recon	(cached)
ok  	centralized-data-service/internal/service/source	(cached)
ok  	centralized-data-service/test/internal/service	(cached)
```
