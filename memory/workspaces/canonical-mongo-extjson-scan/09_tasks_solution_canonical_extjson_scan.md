# 09 Tasks Solution: Canonical MongoDB Extended JSON Scan & Flatten Fix

## 1. Giải pháp kỹ thuật

### File 1: `centralized-data-service/internal/service/source/scan_service.go`
- Bổ sung `isCanonicalMongoExtJSON(m map[string]interface{}) (string, bool)`:
  - Nhận diện các key Canonical ExtJSON: `$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, `$oid`, `$date`, `$binary`, `$timestamp`, `$regularExpression`, `$symbol`, `$code`, `$undefined`.
  - Trả về kiểu PostgreSQL tương ứng: `INTEGER`, `BIGINT`, `NUMERIC`, `TEXT`, `TIMESTAMPTZ`.
- Cập nhật `flattenJSONWithTypes(prefix string, value interface{}, result map[string]string, allowArrayRecurse bool)`:
  - Kiểm tra `isCanonicalMongoExtJSON(v)` trước khi duyệt map. Nếu đúng, gán `result[prefix] = pgType` và `return`.

### File 2: `centralized-data-service/internal/handler/scan/scan_handler.go`
- Bổ sung `isCanonicalMongoExtJSON` và áp dụng guard trong `flattenJSONWithTypes(prefix string, value interface{}, result map[string]string)`.

### File 3: `centralized-data-service/internal/service/source/source_router.go`
- Cập nhật `InferTypeFromRawData`:
  - Thêm `$numberDouble` -> `NUMERIC`
  - Thêm `$timestamp` -> `TIMESTAMPTZ`
  - Thêm `$binary` -> `TEXT`

### File 4: `centralized-data-service/internal/service/master/transmute/flatten.go`
- Cập nhật `unwrapExtJSON`:
  - Thêm `if n, ok := m["$numberInt"]; ok { return n }`
  - Thêm `if n, ok := m["$numberDouble"]; ok { return n }`

## 2. Kế hoạch kiểm thử
- Bổ sung test case trong `scan_handler_test.go` hoặc `scan_service_test.go` với payload lồng chứa `{"info": {"fee": {"$numberInt": "100"}}}`.
- Kiểm tra `result["info.fee"] == "INTEGER"` và không có key `info.fee.$numberInt`.

---

## Phase 2: Refactor DRY & Unwrapping Coercion Solution

### 1. Refactor DRY (Loại bỏ code trùng lặp và gom về Single Source of Truth)
- **File:** `centralized-data-service/internal/service/source/scan_service.go`
  - Đổi `isCanonicalMongoExtJSON` thành hàm Public export: `IsCanonicalMongoExtJSON(m map[string]interface{}) (string, bool)`.
  - Giữ nguyên toàn bộ 14 nhánh nhận diện BSON ExtJSON.
- **File:** `centralized-data-service/internal/handler/scan/scan_handler.go`
  - Xóa bỏ toàn bộ 55 dòng hàm trùng lặp `isCanonicalMongoExtJSON`.
  - Trong `flattenJSONWithTypes(prefix string, value interface{}, result map[string]string)`: gọi `source.IsCanonicalMongoExtJSON(v)`.
- **File:** `centralized-data-service/internal/service/source/source_router.go`
  - Trong `InferTypeFromRawData`: thay thế 50 dòng `switch/if` thủ công cho `map[string]interface{}` bằng lời gọi trực tiếp:
    ```go
    case map[string]interface{}:
        if pgType, ok := IsCanonicalMongoExtJSON(v); ok {
            return pgType
        }
        return "JSONB"
    ```

### 2. Coercion & Complete Unwrapping trong Transmute Flatten
- **File:** `centralized-data-service/internal/service/master/transmute/flatten.go`
  - Thêm helper `mongoNumberToGo(n any) any` chuyển string number sang `int64`.
  - Nâng cấp `unwrapExtJSON(v any) any`:
    - `$numberInt` và `$numberLong`: gọi `mongoNumberToGo(n)` trả về `int64`.
    - `$numberDouble`: dùng `strconv.ParseFloat` trả về `float64`.
    - `$date`: bóc tách string ISO-8601, timestamp float64, hoặc map lồng `$numberLong` (epoch millis) thành `time.Time` UTC qua `time.UnixMilli`.
    - `$timestamp`: bóc tách `{"$timestamp": {"t": ..., "i": ...}}` thành `time.Unix(sec, 0).UTC()`.

### 3. Kế hoạch kiểm thử & Verification
- Cập nhật `TestUnwrapExtJSON` trong `strategy_test.go` khẳng định giá trị unwrap đúng kiểu dữ liệu (`int64`, `float64`, `time.Time`).
- Chạy `go test ./internal/handler/scan/... ./internal/service/source/... ./internal/service/master/transmute/...`.
- Chạy `go build ./internal/... ./cmd/admin-api ./cmd/sinkworker ./cmd/worker`.
