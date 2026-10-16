# 02 Plan: Chuẩn hóa Debezium CDC Payload về BSON Extended JSON và Đồng bộ Wrapper MongoDB BSON cho JSONB

## 1. Mục tiêu (Objective)
Đưa định dạng dữ liệu `_raw_data` của luồng Debezium CDC streaming về đồng nhất với luồng Snapshot:v2 (sử dụng MongoDB Relaxed Extended JSON: `$oid`, `$date`, v.v.). Đồng thời, đánh giá và thiết kế cơ chế lưu giữ wrapper BSON trong các trường kiểu `JSONB` tại Shadow và Master table mà không phá vỡ tính tương thích ngược của hệ thống.

## 2. Bối cảnh & Hiện trạng
- **Snapshot:v2:** Sử dụng Go `mongo-driver` với `bson.MarshalExtJSON(doc, false, false)` $\rightarrow$ tạo BSON Extended JSON:
  - `_id`: `{"$oid": "..."}`
  - `createdAt`, `lastUpdatedAt`: `{"$date": "..."}`
- **Debezium CDC:** Đi qua Kafka Connect với Avro/JSON converter $\rightarrow$ làm phẳng các kiểu BSON:
  - `_id`: chuỗi hex phẳng `"693f..."`
  - `createdAt`, `lastUpdatedAt`: số nguyên epoch ms `176577...`
- **Các trường JSONB hiện tại ở Shadow/Master:**
  - Shadow `schema_adapter_coerce.go`: hàm `CoerceValue` đang gọi `normalizeMongoExtendedJSON` để bóc tách `$oid`, `$date` về primitive string/date trong JSONB.
  - Master `transmuter_utils.go`: hàm `unwrapMongoExtJSON` bóc tách scalar wrapper BSON.

## 3. Lộ trình Triển khai (High-Level Roadmap)
- **Giai đoạn 1: Chuẩn hóa Debezium CDC Ingestion sang BSON ExtJSON tại Consumer**
  - Vị trí: `internal/handler/shadow/kafka_consumer.go` / `event_handler.go`.
  - Nhận diện MongoDB CDC events: tự động bọc lại `_id` (nếu là hex 24 ký tự) thành `{"$oid": "..."}` và các trường timestamp epoch thành `{"$date": "ISO-8601"}` trước khi lưu vào `_raw_data`.
- **Giai đoạn 2: Điều chỉnh CoerceValue cho cột JSONB tại Shadow DB**
  - Cung cấp tùy chọn / cấu hình `preserve_bson_wrappers` hoặc điều chỉnh `CoerceValue` để không tự động bóc bỏ `$oid`/`$date` trong các cột JSONB.
- **Giai đoạn 3: Điều chỉnh Transmuter sang Master DB cho cột JSONB**
  - Giữ nguyên cấu trúc ExtJSON khi chuyển tiếp dữ liệu JSONB sang Master DB.
- **Giai đoạn 4: Kiểm thử Tương thích Toàn diện (Backward Compatibility & Regression Matrix)**
  - Kiểm thử 4 góc: Bản ghi cũ (phẳng) vs Bản ghi mới (ExtJSON) trên cả Debezium và Snapshot.
  - Kiểm thử các câu query SQL / Recon / Heal / Transmute.
