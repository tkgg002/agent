# audit_report_mongo_scan_field.md - Báo Cáo Kiểm Toán Toàn Trình (Deep Audit & Self-Improvement)

## 1. Thông Tin Phiên Kiểm Toán
- **Ngày thực hiện**: 2026-08-21
- **Đối tượng kiểm toán**: Luồng quét schema (`scan-fields`, `scan-raw-data`, `introspect-mongo`) giữa MongoDB Source, Debezium CDC, PostgreSQL Shadow Table và CMS Web UI.
- **Dữ liệu thực tế đối soát**: Message Avro Kafka Debezium 3.5.0 (`cdc.goopay.banvietbank-connector-service.bank-requests` / `PROVIDER_BALANCE`).
- **Tiêu chuẩn áp dụng**: Hiến pháp Agent (GEMINI Core Rules), Core Systems Reliability, No Speculation / Adversarial Self-Review.

---

## 2. Phản Biện & Chỉ Ra Các Lỗi Sai Của Đề Xuất Ban Đầu

### ❌ Lỗi Sai #1: Đề xuất trước chỉ sửa `InferTypeFromRawData` nhưng BỎ SÓT trường hợp `$date` dạng Epoch Milliseconds (Number) gây CRASH SQL
- **Phát hiện qua Avro thực tế**: Message thực tế từ Kafka có dạng:
  ```json
  "createdAt": { "$date": 1786503300747 },
  "updatedAt": { "$date": 1786503300747 }
  ```
- **Lỗ hổng nghiêm trọng**:
  Nếu chỉ sửa Go Type Inferrer trả về `TIMESTAMPTZ` mà không sửa `centralized-data-service/internal/service/metadata/mapping_utils.go:91-104`, câu lệnh SQL sinh ra cho mapping sẽ chạy:
  ```sql
  (CASE
      WHEN jsonb_typeof(_raw_data->'createdAt') = 'number' THEN ...
      WHEN jsonb_typeof(_raw_data->'createdAt') = 'object' AND jsonb_typeof(_raw_data->'createdAt'->'$date') = 'string' THEN ...
      WHEN jsonb_typeof(_raw_data->'createdAt') = 'object' AND jsonb_typeof(_raw_data->'createdAt'->'$date'->'$numberLong') = 'string' THEN ...
      ELSE (NULLIF(_raw_data->>'createdAt', ''))::TIMESTAMP
  END)
  ```
  Khi `$date` là số `1786503300747`, nó **rơi vào nhánh `ELSE`** và thực hiện `('{"$date": 1786503300747}')::TIMESTAMP` -> **Postgres quăng Exception crash toàn bộ lệnh Transform/Sync!**
- **Sửa lại cho đúng**: BẮT BUỘC phải bổ sung nhánh:
  ```sql
  WHEN jsonb_typeof(_raw_data->'%[1]s'->'$date') = 'number'
  THEN to_timestamp((NULLIF(_raw_data->'%[1]s'->>'$date', ''))::NUMERIC::BIGINT / 1000.0) AT TIME ZONE 'UTC'
  ```

---

### ❌ Lỗi Sai #2: Bỏ sót Bug Envelope Path trong `scan_service.go:95-98` (`ScanRawData`)
- **Phát hiện**: Trong `scan_service.go:95-98`:
  ```go
  jsonbPath := "_raw_data"
  if engineType == "postgresql" {
      jsonbPath = "_raw_data->'after'"
  }
  ```
  Debezium MongoDB connector emit message vào shadow table cũng có cấu trúc Envelope `{"after": { ... }, "source": { ... }, "op": "c"}`.
  Vì code chỉ kiểm tra `engineType == "postgresql"`, nên khi chạy nút **Scan Raw Data** trên UI cho bảng MongoDB, `jsonbPath` bị đặt thành `"_raw_data"`. Kết quả là nó quét nhầm các key envelope (`after`, `op`, `source`, `ts_ms`) thay vì quét các trường nghiệp vụ của Mongo!
- **Sửa lại cho đúng**: Mở rộng `jsonbPath` áp dụng cho cả `mongodb`, `mysql`, `mariadb` hoặc dùng biểu thức động `COALESCE(_raw_data->'after', _raw_data)`.

---

### ❌ Lỗi Sai #3: Đề xuất lấy 25 đầu / 25 cuối là giải pháp tạm bợ, không giải quyết được tính Polymorphic của MongoDB
- **Bản chất**: Trong collection `bank-requests`, các transaction được chia theo nhiều `requestType` (`PROVIDER_BALANCE`, `BANK_TRANSFER`, `QR_CODE`, `ACCOUNT_INQUIRY`).
- Nếu trong 1 tuần hệ thống chỉ bắn 50,000 giao dịch `PROVIDER_BALANCE`, thì 25 bản ghi mới nhất và 25 bản ghi cũ nhất đều không thể đại diện cho toàn bộ các schema khác nằm ở giữa.
- **Sửa lại cho đúng**: Sử dụng MongoDB native aggregation `$sample` pipeline:
  ```go
  pipeline := mongo.Pipeline{{{Key: "$sample", Value: bson.M{"size": 100}}}}
  ```
  Thuật toán này lấy mẫu ngẫu nhiên đồng đều trên toàn bộ không gian dữ liệu của collection, đảm bảo quét trọn vẹn tất cả các nhánh schema khác nhau.

---

## 3. Bảng Tổng Hợp Kiểm Toán & Đối Soát Mã Nguồn

| Thành phần | Hiện trạng trong Code | Rủi ro / Lỗi | Khắc phục chuẩn Core Systems |
| :--- | :--- | :--- | :--- |
| **Go Type Inferrer** (`source_router.go`) | Map bất kỳ -> `JSONB` | `createdAt`, `updatedAt`, `_id` bị nhận diện thành `JSONB` | Unwrap BSON keys (`$date`, `$oid`, `$numberDecimal`, `$numberLong`) |
| **SQL Date Transform** (`mapping_utils.go`) | Thiếu check `$date` là `number` | Crash Postgres khi gặp epoch millis như message Avro | Thêm `jsonb_typeof(...) = 'number'` -> `to_timestamp(...)` |
| **Scan Raw SQL** (`scan_service.go`) | Chỉ Postgres mới unwrap `->'after'` | Scan MongoDB trên Shadow table ra nhầm envelope keys | Bổ sung `mongodb` vào điều kiện unwrap `->'after'` |
| **Mongo Introspection** (`mongo_introspection.go`) | `Find({}, Limit(10))` không sort | Bỏ lọt các schema của `BANK_TRANSFER` / `PROVIDER_BALANCE` | Dùng `$sample: {size: 100}` aggregation |
| **Explode Layer** (`ScanArrayFields`) | Xử lý mảng `logs` | `logs` ở cấp root là JSONB | Dùng Master Binding Explode `logs[*]` để bóc tách |
