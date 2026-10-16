# 13 Analysis: So sánh và Phân tích Bản chất _raw_data giữa Debezium và Snapshot:v2

## I. TỔNG QUAN HIỆN TRẠNG DỮ LIỆU

Hai mẫu payload `_raw_data` được trích xuất từ cùng collection MongoDB (`centralized-export-service.export-jobs`):

| Trường (Field) | Khi chạy qua **Debezium CDC** | Khi chạy qua **Snapshot:v2** | Bản chất khác biệt |
| :--- | :--- | :--- | :--- |
| `_id` | `"693f8ee6190816bcb7188bc7"` (String hex trần) | `{"$oid": "6940d414190816bcb7188c13"}` (BSON Extended JSON wrapper) | Debezium unwrap thành scalar string, Snapshot:v2 giữ nguyên BSON `$oid` object |
| `createdAt` | `1765773013801` (Number - Epoch milliseconds) | `{"$date": "2025-12-16T03:37:56.735Z"}` (BSON Extended JSON wrapper) | Debezium map Date thành epoch timestamp (int64), Snapshot:v2 giữ nguyên BSON `$date` object dạng ISO-8601 |
| `lastUpdatedAt` | `1765773014983` (Number - Epoch milliseconds) | `{"$date": "2025-12-16T03:37:57.992Z"}` (BSON Extended JSON wrapper) | Tương tự `createdAt` |
| `params` | Object lồng nhau (`dateFr`, `dateTo`, ...) | Object lồng nhau (`dateFr`, `dateTo`, ...) | Giống nhau (JSON object) |
| `progress`, `__v` | `100`, `0` (Number) | `100`, `0` (Number) | Giống nhau (Number) |
| `jobId`, `fileUrl`, `exportType`, `userId`, `status`, `merchantId` | String scalar | String scalar | Giống nhau |

---

## II. NGUYÊN NHÂN GỐC RỄ (ROOT CAUSE) TỪ MÃ NGUỒN VÀ KIẾN TRÚC

### 1. Luồng Debezium (`io.debezium.connector.mongodb.MongoDbConnector`)
- **Cơ chế thu thập:** Debezium mở MongoDB Change Stream trên Replica Set.
- **Tầng Debezium Converter / Schema:**
  - Debezium áp dụng type mapping chuẩn của mình:
    - BSON `ObjectId` được ép kiểu về kiểu chuỗi phẳng (String) đại diện cho 24-byte hex: `"693f8ee6190816bcb7188bc7"`.
    - BSON `Date` (`ISODate`) được chuyển đổi theo schema `io.debezium.time.Timestamp`, biểu diễn dưới dạng số nguyên `int64` (epoch milliseconds tính từ 1970-01-01): `1765773013801`.
- **Tầng Consumer (`kafka_consumer.go`):**
  - Khi `centralized-data-service` đọc message từ Kafka (thông qua Avro hoặc JSON unmarshal), trường `after` được giải mã thành Go map: `map[string]interface{}`.
  - Các trường này là kiểu dữ liệu chuẩn của JSON: `_id` là `string`, `createdAt` là `float64` / `int64`.
  - Khi ghi xuống `_raw_data` qua `DynamicMapper`: dữ liệu nguồn này được giữ nguyên bản dưới dạng JSON phẳng.

### 2. Luồng Snapshot:v2 (`snapshot_runner_handler.go`)
- **Cơ chế thu thập:** Snapshot:v2 kết nối trực tiếp vào MongoDB thông qua Go official mongo-driver (`go.mongodb.org/mongo-driver/mongo`), thực hiện phân trang `coll.Find()`.
- **Tầng Serialization trong mã nguồn (Dòng 674 `snapshot_runner_handler.go`):**
  ```go
  mongoBatch := make([]bson.M, 0, p.BatchSize)
  cursor.All(cursorCtx, &mongoBatch)
  for _, doc := range mongoBatch {
      idStr, _ := extractDocID(doc)
      afterJSON, mErr := bson.MarshalExtJSON(doc, false, false)
      records = append(records, record{idStr: idStr, afterJSON: afterJSON, err: mErr})
  }
  ```
- **Hàm `bson.MarshalExtJSON(doc, canonical, escapeHTML)`:**
  - Được cấu hình với `canonical = false`, tức là chế độ **MongoDB Relaxed Extended JSON format (BSON ExtJSON)**:
    - `primitive.ObjectID` luôn được đóng gói thành: `{"$oid": "..."}`.
    - `primitive.DateTime` luôn được đóng gói thành: `{"$date": "YYYY-MM-DDTHH:mm:ss.sssZ"}`.
  - Sau đó `afterJSON` này được đóng gói vào CDC envelope `buildSnapshotEnvelope(...)` và truyền vào `EventHandler.HandleRaw` -> `DynamicMapper` -> lưu trực tiếp vào trường `_raw_data`.

---

## III. HỆ THỐNG ĐANG XỬ LÝ SỰ KHÁC BIỆT NÀY NHƯ THẾ NÀO? (DUAL-STACK ADAPTER)

Theo bài học hệ thống ngày **2026-09-16** (`lessons.md`):
> **Nguyên tắc:** Raw Data (`_raw_data`) là **Ground Truth bất biến** đại diện trung thực cho những gì nguồn gửi đến. Tuyệt đối không can thiệp biến dạng `_raw_data` tại tầng Ingestion. Thay vào đó, toàn bộ hạ tầng hạ lưu được trang bị **Dual-Stack Adapter / Polymorphic Handling** để xử lý mượt mà cả hai định dạng.

Cụ thể, hệ thống xử lý ở 3 tầng:

### 1. Tầng Typed Columns tại Shadow DB (`dynamic_mapper.go`)
Trước khi trích xuất giá trị cho các cột định kiểu (ví dụ cột `created_at` kiểu `TIMESTAMPTZ` trên PostgreSQL), hàm `unwrapMongoTypes` (dòng 341) tự động nhận diện:
- **Xử lý `_id`:**
  - Nếu là Debezium: nhận string `"693f..."` -> giữ nguyên.
  - Nếu là Snapshot:v2: nhận map `{"$oid": "6940..."}` -> trích xuất chuỗi hex `"6940..."`.
- **Xử lý `createdAt` / `lastUpdatedAt`:**
  - Nếu là Debezium: nhận number `1765773013801` -> giữ nguyên -> hàm `toTimestamp` nhận `int64`/`float64` > 1e12 và chuyển thành `time.UnixMilli(...)`.
  - Nếu là Snapshot:v2: nhận map `{"$date": "2025-12-16T03:37:56.735Z"}` -> `unwrapMongoTypes` parse `time.RFC3339` thành `time.Time`.
=> **Kết quả:** Trên các cột typed column (`created_at`, `id`, `_source_id`), cả 2 luồng đều cho ra dữ liệu đồng nhất 100%.

### 2. Tầng SQL Expression / Metadata Views (`mapping_utils.go` - `BuildCastExpr`)
Khi query trực tiếp từ trường `_raw_data` bằng PostgreSQL SQL:
- **Biểu thức ép kiểu cho Text/ID (Dòng 127):**
  ```sql
  (CASE
    WHEN jsonb_typeof(_raw_data->'my_field') = 'object'
     AND jsonb_typeof(_raw_data->'my_field'->'$oid') = 'string'
    THEN (_raw_data->'my_field'->>'$oid')
    ELSE (_raw_data->>'my_field')
  END)
  ```
- **Biểu thức ép kiểu cho Timestamp (Dòng 91 & 109):**
  ```sql
  (CASE
    WHEN jsonb_typeof(_raw_data->'my_field') = 'number'
    THEN to_timestamp((NULLIF(_raw_data->>'my_field', ''))::NUMERIC::BIGINT / 1000.0)
    WHEN jsonb_typeof(_raw_data->'my_field') = 'object'
     AND jsonb_typeof(_raw_data->'my_field'->'$date') = 'string'
    THEN (NULLIF(_raw_data->'my_field'->>'$date', ''))::TIMESTAMPTZ
    ELSE (NULLIF(_raw_data->>'my_field', ''))::TIMESTAMPTZ
  END)
  ```
=> Hệ thống tự động phân nhánh SQL runtime: nếu gặp number (Debezium) thì chia 1000 để ra giây, nếu gặp object `$date` (Snapshot:v2) thì cast thẳng từ ISO-8601 string.

### 3. Tầng Transmuter đẩy dữ liệu sang Master DB (`transmuter_utils.go` & `flatten.go`)
- Hàm `unwrapMongoExtJSON` (dòng 45 `transmuter_utils.go`) và `unwrapExtJSON` (dòng 213 `flatten.go`) kiểm tra:
  - Nếu đối tượng là map 1 key có `$oid`, bóc ra string.
  - Nếu là `$date`, bóc ra ISO-8601 string để PostgreSQL master tự parse.
  - Nếu đã là primitive scalar (từ Debezium), giữ nguyên.

---

## IV. ĐÁNH GIÁ VÀ KHUYẾN NGHỊ VẬN HÀNH

1. **Tính tương thích:** Hệ thống hiện tại đã được bọc Dual-Stack toàn diện từ Ingestion, Shadow Typed Column, Shadow SQL View cho đến Master Transmuter. Dữ liệu dù đến từ Debezium hay Snapshot:v2 đều được xử lý chính xác và không bị gãy type.
2. **Nguyên tắc Bất biến:** Tuyệt đối **KHÔNG** can thiệp viết code để "làm phẳng hóa" `_raw_data` của Snapshot:v2 thành kiểu Debezium (hoặc ngược lại) tại tầng lưu trữ Shadow, vì sẽ vi phạm tôn chỉ **Ground Truth** và dễ dẫn đến rủi ro hồi quy (Regression) với các mapping rules lịch sử.
3. **Lưu ý khi cấu hình Mapping Rules thủ công:**
   - Trong `mapping_rule_v2`, `source_field` nên luôn để tên trường gốc (`_id`, `createdAt`) thay vì trỏ lồng sâu (`_id.$oid`, `createdAt.$date`), vì tên trường gốc được hệ sinh thái Adapter hỗ trợ tự động trên cả 2 nguồn Debezium và Snapshot:v2.
