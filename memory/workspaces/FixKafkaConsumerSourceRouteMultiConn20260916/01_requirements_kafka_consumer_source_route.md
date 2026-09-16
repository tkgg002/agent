# 01_requirements_kafka_consumer_source_route.md
## Yêu cầu Kỹ thuật: Khắc phục Lỗi Định tuyến Source -> Shadow của Kafka Consumer khi Trùng Tên Database & Chuẩn hóa _raw_data

### 1. Bối cảnh & Thực trạng
Hệ thống kết nối với 2 nguồn MongoDB độc lập:
- **Connection 1:** Code `traitestmongodevct` | Engine: `mongodb` | Database: `core-trans-proxy-history-service`
- **Connection 2:** Code `traitestctphs` | Engine: `mongodb` | Database: `core-trans-proxy-history-service`

Cả 2 database nguồn đều có cùng tên `core-trans-proxy-history-service` và cùng chứa collection `trans_his`.
- Target Shadow của Connection 1: schema `shadow_traitestmongodevct`, table `trans_his`.
- Target Shadow của Connection 2: schema `shadow_traitestctphs`, table `trans_his`.

### 2. Các Vấn đề Cốt lõi Cần Giải quyết

#### Vấn đề 1: Phân phối nhầm / đụng độ dữ liệu Source -> Shadow tại Kafka Consumer
- Khi Debezium produce message lên Kafka, Kafka Consumer đọc message và gọi `EventHandler.HandleRaw` -> `processEvent`.
- Tại `processEvent` (`internal/handler/shadow/event_handler.go:217`):
  `routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable)`
- Hàm `ResolveSourceRoutes` chỉ nhận `(sourceDB, sourceTable)` mà KHÔNG nhận `sourceConnectionCode`.
- Trong `MetadataRegistryService.ReloadAll`:
  Cache key `core-trans-proxy-history-service|trans_his` được nạp route của CẢ 2 connection (`traitestmongodevct` VÀ `traitestctphs`).
- Khi `ResolveSourceRoutes` tra cứu, nó trả về cả 2 routes!
- Vòng lặp Fan-out (`for _, route := range routes`) ghi nhận sự kiện vào CẢ 2 BẢNG SHADOW của cả 2 kết nối!
- Dữ liệu từ kết nối 1 bị ghi chéo sang kết nối 2 và ngược lại.
- Thêm vào đó, `primaryRoute := routes[0]` luôn bốc route đầu tiên (`traitestmongodevct`), khiến việc giải mã PK, schema inspection của kết nối 2 bị áp đặt theo cấu hình của kết nối 1!

#### Vấn đề 2: Format `_raw_data` lệch nhau giữa Debezium và Snapshot V2
- Snapshot V2 (`snapshot_runner_handler.go:674`): Sử dụng `bson.MarshalExtJSON(doc, false, false)` -> sinh ra MongoDB Extended JSON (`{"$oid": "..."}`, `{"$date": "..."}`).
- Debezium: Đã unwrap BSON ở tầng Kafka Connect converter -> JSON phẳng (`_id` là string, ngày tháng là epoch int64 ms).
- Tại `dynamic_mapper.go:121`: `rawJSON, _ := json.Marshal(dm.maskRawData(bindingID, rawData))` ghi trực tiếp `rawData` vào `_raw_data` mà không unwrap đệ quy MongoDB Extended JSON, làm định dạng trong cột `_raw_data` của PostgreSQL bị đa hình (không đồng nhất).

### 3. Yêu cầu Nghiệm thu (Definition of Done)
1. **Phân lập tuyệt đối Source -> Shadow theo Connection Code:**
   - Kafka Consumer / EventHandler BẮT BUỘC nhận diện được Connection Code từ Kafka Topic hoặc Debezium Source Envelope (`source.name`).
   - `ResolveSourceRoutes` BẮT BUỘC hỗ trợ nhận diện `sourceConnectionCode` để chỉ phân giải đúng route thuộc về kết nối đó.
   - Tuyệt đối không ghi chéo dữ liệu giữa `traitestmongodevct` và `traitestctphs`.
2. **Đồng nhất 100% định dạng `_raw_data`:**
   - Cung cấp hàm chuẩn hóa đệ quy `normalizeMongoExtJSON` trong `dynamic_mapper.go` (hoặc ở pipeline Snapshot V2) để unwrap toàn bộ `{"$oid": "..."}` thành string và `{"$date": "..."}` thành timestamp chuẩn.
3. **Không phá vỡ (Zero-Regression) các luồng hiện tại:**
   - Giữ tương thích ngược với các nguồn SFTP, MySQL, PostgreSQL.
   - Các unit test và integration test pass 100%.
