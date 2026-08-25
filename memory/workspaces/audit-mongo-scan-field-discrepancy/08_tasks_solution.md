# 08_tasks_solution.md - Hồ sơ Giải pháp Kỹ thuật Chi tiết (Updated with Avro Deep-Dive)

## 1. Vấn đề 1: Type Inference và SQL Casting cho MongoDB Extended JSON (`$date`, `$oid`)
### Gốc rễ:
1. **Ở tầng Go Type Inferrer (`source_router.go`)**:
   Khi `json.Unmarshal` đọc chuỗi BSON Extended JSON v2:
   - `createdAt`: `{"$date": 1786503300747}` (số) hoặc `{"$date": "2026-05-07T04:01:17.830Z"}` (chuỗi ISO)
   - `_id`: `{"$oid": "6a7be084eeb3c73b3c96f2c1"}`
   - `updatedAt`: `{"$date": 1786503300747}`
   Các trường này có kiểu Go `map[string]interface{}`. Hàm `InferTypeFromRawData` hiện tại coi mọi map là `JSONB`.
2. **Ở tầng SQL Mapping Expression (`mapping_utils.go:91-104`)**:
   Hàm sinh biểu thức SQL cho kiểu `TIMESTAMPTZ` hiện đang **THIẾU** trường hợp `jsonb_typeof(_raw_data->'createdAt'->'$date') = 'number'`. Khi `$date` là epoch millis (1786503300747) như trong message Avro thực tế, SQL fallback về `(NULLIF(_raw_data->>'createdAt', ''))::TIMESTAMP` gây crash SQL (`invalid input syntax for type timestamp: "{"$date": 1786503300747}"`).

### Giải pháp tối ưu toàn diện:
1. Sửa `InferTypeFromRawData` trong `source_router.go`:
   ```go
   case map[string]interface{}:
       if len(v) == 1 {
           if _, ok := v["$date"]; ok {
               return "TIMESTAMPTZ"
           }
           if _, ok := v["$oid"]; ok {
               return "TEXT"
           }
           if _, ok := v["$numberDecimal"]; ok {
               return "NUMERIC"
           }
           if _, ok := v["$numberLong"]; ok {
               return "BIGINT"
           }
           if _, ok := v["$numberInt"]; ok {
               return "INTEGER"
           }
       }
       return "JSONB"
   ```
2. Bổ sung nhánh `number` cho `$date` trong `mapping_utils.go:91-104`:
   ```sql
   WHEN jsonb_typeof(_raw_data->'%[1]s'->'$date') = 'number'
   THEN to_timestamp((NULLIF(_raw_data->'%[1]s'->>'$date', ''))::NUMERIC::BIGINT / 1000.0) AT TIME ZONE 'UTC'
   ```

---

## 2. Vấn đề 2: Lỗi Envelope JSONB Path trong `scan_service.go:95-98` (`ScanRawData`)
### Gốc rễ:
Trong `scan_service.go`:
```go
jsonbPath := "_raw_data"
if engineType == "postgresql" {
    jsonbPath = "_raw_data->'after'"
}
```
Khi source là MongoDB, Debezium CDC cũng đóng gói dữ liệu vào envelope `{"after": { ... }}`. Việc hardcode chỉ Postgres mới dùng `_raw_data->'after'` làm tính năng Scan Raw Data trên MongoDB quét nhầm các trường cấp envelope (`after`, `before`, `source`, `op`, `ts_ms`) thay vì các business fields!

### Giải pháp tối ưu:
```go
jsonbPath := "_raw_data"
if engineType == "postgresql" || engineType == "mongodb" || engineType == "mysql" || engineType == "mariadb" {
    jsonbPath = "_raw_data->'after'"
}
```
Hoặc kiểm tra sự tồn tại của key `after` bằng SQL `CASE WHEN _raw_data ? 'after' THEN _raw_data->'after' ELSE _raw_data END`.

---

## 3. Vấn đề 3: Sampling Engine cho MongoDB Polymorphic Collection
### Gốc rễ:
Lệnh `collection.Find(..., Limit(10))` lấy 10 document đầu theo insertion order. Vì 10 document đầu là các request API cũ (`requestData`, `responseData`), nên bỏ lọt các giao dịch `BANK_TRANSFER` / `PROVIDER_BALANCE` (`bankTransactionId`, `logs`).

### Giải pháp tối ưu:
Sử dụng MongoDB Aggregation `$sample` để lấy mẫu ngẫu nhiên phân bổ đều trên toàn collection:
```go
pipeline := mongo.Pipeline{
    {{Key: "$sample", Value: bson.M{"size": 100}}},
}
cursor, err := collection.Aggregate(ctx, pipeline)
```
Kết hợp với việc merge các field từ toàn bộ các sample documents.
