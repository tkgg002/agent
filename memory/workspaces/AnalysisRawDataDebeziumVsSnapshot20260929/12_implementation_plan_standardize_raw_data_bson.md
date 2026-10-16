# 12 Implementation Plan: Đồng bộ Debezium CDC về Chuẩn MongoDB BSON ExtJSON và Lưu giữ BSON Wrapper cho JSONB Fields (Cập nhật)

## I. NGUYÊN NHÂN GỐC RỄ & PHÁT HIỆN THỰC TẾ
- Chuỗi payload `after.string` nhận từ Kafka do Debezium MongoDB Connector phát ra **VỐN ĐÃ ĐƯỢC BỌC SẴN BSON EXTJSON**:
  ```json
  {"_id": {"$oid": "693f8ee6190816bcb7188bc7"}, "createdAt": {"$date": 1765773013801}, "lastUpdatedAt": {"$date": 1765773014983}, ...}
  ```
- Tại `kafka_consumer.go:613`, sau khi parse chuỗi JSON này, code lại gọi hàm `unwrapAvroUnionMap(afterData)`.
- Hàm `unwrapAvroUnion` trong `avro_helper.go` thấy map có đúng 1 key (`len(m) == 1`) nên tưởng là Avro Union (`{"string": ...}`) và **bóc nhầm mất** các key `$oid` và `$date`.
- Ở tầng Shadow, `schema_adapter_coerce.go` lại chạy qua `normalizeMongoExtendedJSON` làm bóc tiếp các wrapper BSON bên trong cột `JSONB`.

## II. GIẢI PHÁP KỸ THUẬT TINH GỌN (SIMPLICITY FIRST)
1. `kafka_consumer.go`: Không gọi `unwrapAvroUnionMap` đối với payload đã unmarshal từ JSON string.
2. `avro_helper.go`: `unwrapAvroUnion` bỏ qua mọi map có key bắt đầu bằng `$` (BSON ExtJSON).
3. `schema_adapter_coerce.go`: Bỏ `normalizeMongoExtendedJSON` đối với các cột kiểu `JSONB` để giữ nguyên BSON wrapper ở Shadow.
4. `transmuter_utils.go`: Bảo toàn BSON wrapper khi chuyển tiếp sang Master DB.

## III. PHẠM VI ÁP DỤNG
- Áp dụng cho các sự kiện mới phát sinh (In-flight CDC events).
- Giữ nguyên các bản ghi cũ, Dual-Stack Adapter tiếp tục tương thích hoàn toàn.
