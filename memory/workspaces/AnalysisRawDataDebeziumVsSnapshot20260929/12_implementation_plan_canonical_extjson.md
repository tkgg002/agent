# Kế Hoạch Triển Khai: Chuyển Đổi Canonical Extended JSON (v2) Cho Cả Debezium CDC & Snapshot

## 1. Tổng Quan
- **Nhu cầu:** User yêu cầu đưa cả Debezium CDC và Snapshot:v2 về định dạng MongoDB Canonical Extended JSON (v2) (giữ các thẻ `$numberInt`, `$numberLong`, `$date.{"$numberLong"}`).
- **Nguyên tắc:** 
  - Không phá vỡ tương thích của Typed Columns trong Shadow & Master DB.
  - Xử lý tại tầng Ingestion để `_raw_data` lưu trữ thống nhất 100%.

## 2. Kế Hoạch Chi Tiết Các Bước Triển Khai
- [ ] **Bước 1 (Snapshot Engine):**
  - Chỉnh sửa `snapshot_runner_handler.go` (L674): `bson.MarshalExtJSON(doc, true, false)`
  - Chỉnh sửa `snapshot_runner_utils.go` (L135): `bson.MarshalExtJSON(bson.M{"_id": raw}, true, false)`
  - Chỉnh sửa `recon_heal_fetch.go` (L76): `bson.MarshalExtJSON(doc, true, false)`
- [ ] **Bước 2 (Debezium CDC Consumer):**
  - Trong `kafka_consumer.go`: xây dựng hàm chuẩn hóa `toCanonicalMongoExtJSON`. Khi `afterRaw` hoặc `beforeRaw` là `string` từ MongoDB, chuyển đổi qua Canonical ExtJSON trước khi unmarshal vào `afterData` / `beforeData`.
- [ ] **Bước 3 (Shadow Dynamic Mapper Safeguard):**
  - Trong `dynamic_mapper.go`: bổ sung unwrap `$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal` vào `unwrapMongoTypes` để bảo vệ việc ánh xạ giá trị vào các cột định kiểu (int, bigint, numeric) trong PostgreSQL.
- [ ] **Bước 4 (Sinkworker Protection):**
  - Bổ sung `toCanonicalMongoExtJSON` vào `internal/sinkworker/utils.go` và guard prefix `$` trong `unwrapAvroUnion`.
- [ ] **Bước 5 (Biên dịch & Xác Thực):**
  - Compile toàn bộ dự án (`go build ./...`).
  - Chạy `worker` và verify lại dữ liệu thực tế trên PostgreSQL Shadow DB.
