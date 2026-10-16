# 11 Report: Báo Cáo Thay Đổi Mã Nguồn Chuẩn Hóa Debezium & Giữ Wrapper BSON Cho JSONB

## I. TỔNG QUAN THAY ĐỔI
Đã hoàn thành chuẩn hóa việc tiếp nhận dữ liệu MongoDB từ Debezium CDC streaming khớp 100% với Snapshot:v2 (chuẩn MongoDB BSON Extended JSON: `$oid`, `$date`, `$numberLong`...) và cấu hình giữ nguyên wrapper BSON cho các trường kiểu `JSONB` ở cả Shadow DB và Master DB.

---

## II. DANH SÁCH TỆP TIN THAY ĐỔI VÀ DÒNG CODE

### 1. `centralized-data-service/internal/handler/shadow/kafka_consumer.go`
- **Số dòng thay đổi:** ~14 dòng (bỏ 7 dòng thừa, tinh chỉnh 7 dòng switch case).
- **Chi tiết thay đổi:**
  - Phân nhánh chính xác: Chỉ gọi `unwrapAvroUnionMap` khi `afterRaw` / `beforeRaw` là `map[string]interface{}` (Avro record của nguồn quan hệ Postgres/MySQL).
  - Khi `afterRaw` / `beforeRaw` là `string` (MongoDB Change Streams JSON): unmarshal JSON trực tiếp và **không** gọi `unwrapAvroUnionMap`, bảo toàn nguyên vẹn `$oid` và `$date`.

### 2. `centralized-data-service/internal/handler/shadow/avro_helper.go`
- **Số dòng thay đổi:** +6 dòng.
- **Chi tiết thay đổi:**
  - Bổ sung chốt chặn an toàn (Defensive Guard) trong `unwrapAvroUnion`: nếu map có 1 key và key đó bắt đầu bằng `"$"` (ví dụ `$oid`, `$date`, `$numberLong`), hàm nhận diện đó là MongoDB BSON ExtJSON và trả về nguyên vẹn thay vì bóc tách như Avro union.

### 3. `centralized-data-service/internal/service/shadow/schema_adapter_coerce.go`
- **Số dòng thay đổi:** ~20 dòng.
- **Chi tiết thay đổi:**
  - Trong `CoerceValue` cho cột kiểu `JSONB`: Loại bỏ bước gọi `normalizeMongoExtendedJSON(v)`, marshal trực tiếp `json.Marshal(v)` để bảo toàn 100% cấu trúc BSON wrapper bên trong cột `JSONB`.
  - Trong `decodeBase64JSON`: decode trực tiếp chuỗi JSON mà không bóc trần BSON wrapper.

### 4. `centralized-data-service/internal/service/master/transmuter.go`
- **Số dòng thay đổi:** +3 dòng.
- **Chi tiết thay đổi:**
  - Trong `extractColumns`: Chỉ gọi `unwrapMongoExtJSON` khi cột đích không phải kiểu `JSONB`/`JSON` (`!isJSONColumnType(r.DataType)`). Nhờ đó, các trường JSONB được chuyển tiếp nguyên vẹn wrapper BSON sang Master DB.

### 5. `centralized-data-service/internal/handler/shadow/avro_helper_test.go` [NEW]
- **Số dòng:** 75 dòng.
- **Chi tiết thay đổi:**
  - File test mới kiểm chứng `unwrapAvroUnion` với Avro unions thông thường và đảm bảo không bóc nhầm các wrapper BSON ExtJSON.

### 6. `centralized-data-service/test/internal/service/schema_adapter_coerce_test.go`
- **Số dòng thay đổi:** 16 dòng.
- **Chi tiết thay đổi:**
  - Cập nhật test case `TestSchemaAdapter_CoerceValue_JSON` để assert rằng các wrapper `$oid` và `$date` được lưu giữ nguyên vẹn trong cột `JSONB`.

---

## III. KẾT QUẢ NGHIỆM THU
- Toàn bộ unit tests mới và liên quan đều **PASS 100%**.
- Các binary `cmd/worker`, `cmd/admin-api`, `cmd/sinkworker` đều biên dịch thành công (Exit Code = 0).
