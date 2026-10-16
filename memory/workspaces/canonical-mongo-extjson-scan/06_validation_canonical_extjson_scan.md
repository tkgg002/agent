# 06 Validation: Canonical MongoDB Extended JSON Scan & Flatten Fix

## 1. Mục tiêu kiểm thử
Xác nhận rằng:
1. Khi scan/flatten các đối tượng lồng chứa MongoDB Canonical Extended JSON (đặc biệt là `info.fee: {"$numberInt": "100"}` và array `items[*].sub_fee: {"$numberInt": "50"}`), bộ scan không còn bóc tách sâu thành `info.fee.$numberInt`.
2. Kiểu dữ liệu PostgreSQL được suy luận chính xác: `$numberInt` -> `INTEGER`, `$numberLong` -> `BIGINT`, `$numberDouble` -> `NUMERIC`, `$timestamp` -> `TIMESTAMPTZ`, v.v.
3. Hàm `unwrapExtJSON` trong Transmuter Flatten tháo vỏ `$numberInt` và `$numberDouble` thành công.
4. Toàn bộ test suite và build binary của project không phát sinh lỗi hoặc regression.

## 2. Kết quả Unit Tests

### A. Test Flatten & Chặn đệ quy ExtJSON
Lệnh chạy:
```bash
go test -v -run "TestFlattenJSONWithTypes_CanonicalMongoExtJSON|TestScanService_FlattenJSONWithTypes_CanonicalMongoExtJSON" ./internal/handler/scan/... ./internal/service/source/...
```
Kết quả:
```text
=== RUN   TestFlattenJSONWithTypes_CanonicalMongoExtJSON
--- PASS: TestFlattenJSONWithTypes_CanonicalMongoExtJSON (0.00s)
=== RUN   TestScanService_FlattenJSONWithTypes_CanonicalMongoExtJSON
--- PASS: TestScanService_FlattenJSONWithTypes_CanonicalMongoExtJSON (0.00s)
PASS
ok  	centralized-data-service/internal/handler/scan	0.024s
ok  	centralized-data-service/internal/service/source	0.031s
```
Khẳng định:
- `result["info.fee"] == "INTEGER"`: PASS.
- `result["items[*].sub_fee"] == "INTEGER"`: PASS.
- Hoàn toàn KHÔNG tồn tại bất kỳ key nào chứa `.$numberInt` hoặc `.$oid` hay `.$date`.

### B. Test Suy luận kiểu ExtJSON (InferTypeFromRawData)
Lệnh chạy:
```bash
go test -v -run "TestInferTypeFromRawData" ./internal/service/source/...
```
Kết quả:
```text
=== RUN   TestInferTypeFromRawData_ExtendedJSON
--- PASS: TestInferTypeFromRawData_ExtendedJSON (0.00s)
PASS
ok  	centralized-data-service/internal/service/source	0.029s
```
Bao phủ 25 kịch bản bao gồm `$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, `$oid`, `$date`, `$binary`, `$timestamp`, `$regularExpression`, `$symbol`, `$code`, `$undefined`.

### C. Test Unwrap ExtJSON trong Transmuter Flatten
Lệnh chạy:
```bash
go test -v -run "TestUnwrapExtJSON" ./internal/service/master/transmute/...
```
Kết quả:
```text
=== RUN   TestUnwrapExtJSON
--- PASS: TestUnwrapExtJSON (0.00s)
PASS
ok  	centralized-data-service/internal/service/master/transmute	0.018s
```

## 3. Build Verification
Lệnh chạy:
```bash
go build ./internal/... ./cmd/admin-api ./cmd/sinkworker ./cmd/worker
```
Kết quả: **Exit Code 0** (Biên dịch thành công 100%, không cảnh báo, không lỗi cú pháp).
