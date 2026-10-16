# Technical Solution: Chuẩn Hóa Cả Debezium CDC và Snapshot:v2 sang Canonical Extended JSON (v2)

## 1. Mục Tiêu Nghiệp Vụ
Đồng bộ hóa 100% định dạng `_raw_data` của MongoDB trên cả 2 kênh Ingestion (Debezium CDC streaming và Snapshot:v2 backfill) sang chuẩn **MongoDB Canonical Extended JSON (v2)**:
- Trường ID: `{"_id": {"$oid": "..."}}`
- Trường số nguyên / long / double:
  - `{"$numberInt": "198250"}`
  - `{"$numberLong": "..."}`
  - `{"$numberDouble": "..."}`
- Trường thời gian:
  - `{"$date": {"$numberLong": "1790749815450"}}`
- Trường phiên bản:
  - `{"__v": {"$numberInt": "0"}}`

---

## 2. Điểm Kỹ Thuật Cốt Lõi (Architecture Core)

### A. Snapshot:v2 (`snapshot_runner_handler.go` & `recon_heal_fetch.go`)
- Dữ liệu từ MongoDB cursor được nạp vào Go struct `bson.M`.
- Hiện tại: gọi `bson.MarshalExtJSON(doc, false, false)` -> sinh ra Relaxed mode (số nguyên và ngày tháng bị unwrap thành JSON scalar/integer).
- **Giải pháp:** Chuyển tham số `canonical` từ `false` sang `true`:
  `bson.MarshalExtJSON(doc, true, false)`
  -> Tự động sinh 100% Canonical Extended JSON.

### B. Debezium CDC (`kafka_consumer.go` & `sinkworker/utils.go`)
- Debezium MongoDB Connector capture Change Stream và đóng gói vào `event["after"]` dưới dạng Avro union `{"string": "..."}`.
- Chuỗi JSON trong `string` do Debezium sinh ra vốn ở dạng Relaxed ExtJSON (chứa số thô `210840` và ngày phẳng `{"$date": 1790132406754}`).
- **Giải pháp:** Khi nhận chuỗi JSON `v` từ Debezium `afterRaw`:
  Dùng `bson.UnmarshalExtJSON([]byte(v), false, &doc)` để phục hồi BSON type metadata (Int32, Int64, DateTime), sau đó gọi `bson.MarshalExtJSON(doc, true, false)` để re-serialize sang **Canonical Extended JSON**.
  Thao tác này đảm bảo tính Idempotent (nếu payload đã là Canonical thì giữ nguyên, nếu là Relaxed thì nâng cấp thành Canonical).

### C. Khả Năng Tương Thích Ngược & An Toàn Tầng Dưới
1. **DynamicMapper (`dynamic_mapper.go`):**
   - Cột `_raw_data` lấy trực tiếp `rawData` (Canonical ExtJSON).
   - Các cột nghiệp vụ định kiểu (Typed Columns: INT, BIGINT, DOUBLE) trong Shadow DB: bổ sung vào `unwrapMongoTypes` khả năng bóc tách `$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal` để `convertType` map giá trị sang PostgreSQL column mà không bị lỗi.
2. **Primary Key Extraction (`event_handler.go`):**
   - Hỗ trợ trích xuất PK an toàn khi `_id` hoặc PK field là `$oid`, `$numberInt`, `$numberLong`.
3. **Master Transmuter (`transmuter_utils.go`):**
   - Đã có sẵn hỗ trợ `$numberInt`, `$numberLong`, `$numberDouble`, `$date.{"$numberLong"}`. Hoạt động tương thích 100%.

---

## 3. Danh Sách File Cần Chỉnh Sửa

1. `internal/handler/orchestration/snapshot_runner_handler.go`: Bật `canonical = true` khi marshal `doc`.
2. `internal/handler/orchestration/snapshot_runner_utils.go`: Bật `canonical = true` trong fallback marshal.
3. `internal/handler/recon/recon_heal_fetch.go`: Bật `canonical = true` trong heal fetch.
4. `internal/handler/shadow/kafka_consumer.go`: Thêm `toCanonicalMongoExtJSON` và áp dụng cho `afterRaw` / `beforeRaw`.
5. `internal/service/shadow/dynamic_mapper.go`: Bổ sung unwrap các thẻ số `$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal` cho Typed Columns.
6. `internal/sinkworker/utils.go`: Bổ sung `toCanonicalMongoExtJSON` cho `decodeAfter` và guard `$` cho `unwrapAvroUnion`.
