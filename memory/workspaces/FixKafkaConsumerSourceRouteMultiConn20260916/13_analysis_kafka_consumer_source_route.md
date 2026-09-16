# 13_analysis_kafka_consumer_source_route.md
## Báo Cáo Phân Tích Chuyên Sâu: Cơ Chế Routing Source -> Shadow Tại Kafka Consumer Khi Nhiều Nguồn Trùng Tên Database

---

### 1. Kiến trúc Luồng Dữ liệu Source -> Shadow
Dữ liệu CDC từ nguồn MongoDB truyền về PostgreSQL Shadow thông qua các trạm:
```
MongoDB Cluster 1 (traitestmongodevct) ──┐
                                         ├──> Kafka Broker (Topic: <prefix>.<db>.<coll>)
MongoDB Cluster 2 (traitestctphs)       ──┘         │
                                                    ▼
                                          KafkaConsumer (CDS Engine)
                                                    │
                                                    ▼
                                          EventHandler.HandleRaw
                                                    │
                                                    ▼
                                          EventHandler.processEvent
                                                    │
                                                    ├──> MetadataRegistryService.ResolveSourceRoutes(db, coll, [conn])
                                                    │
                                                    ├──> DynamicMapper (Chuyển đổi type, mask data, _raw_data)
                                                    │
                                                    ▼
                                          BatchBuffer ──> PostgreSQL Shadow Tables
                                              - shadow_traitestmongodevct.trans_his
                                              - shadow_traitestctphs.trans_his
```

---

### 2. Chi Tiết Các Lỗ Hổng Kiến Trúc Đang Gây Xáo Trộn Dữ Liệu

#### Lỗ hổng 1: Đụng độ Key tại `MetadataRegistryService`
- Trong `metadata_registry_service.go:ReloadAll`:
  ```go
  for _, sourceKey := range buildSourceLookupKeys(src, sourceConnCode) {
      rs.routeCache[sourceKey] = append(rs.routeCache[sourceKey], route)
  }
  ```
- `buildSourceLookupKeys` tạo các key:
  1. `objectName` (ví dụ `trans_his`)
  2. `sourceDB|objectName` (ví dụ `core-trans-proxy-history-service|trans_his`)
  3. `connCode:objectName` (ví dụ `traitestmongodevct:trans_his`)
  4. `connCode:sourceDB|objectName` (ví dụ `traitestmongodevct:core-trans-proxy-history-service|trans_his`)
- **Vấn đề:** Cả Connection 1 (`traitestmongodevct`) và Connection 2 (`traitestctphs`) đều có `sourceDB = "core-trans-proxy-history-service"` và `objectName = "trans_his"`.
  Do đó, key `core-trans-proxy-history-service|trans_his` bị append CẢ 2 ROUTES:
  `rs.routeCache["core-trans-proxy-history-service|trans_his"] = [Route_traitestmongodevct, Route_traitestctphs]`!

#### Lỗ hổng 2: `ResolveSourceRoutes` chỉ tìm kiếm theo 2 tham số `(sourceDB, sourceTable)`
- Chữ ký hàm:
  ```go
  func (rs *MetadataRegistryService) ResolveSourceRoutes(sourceDB, sourceTable string) []*metadata.ResolvedSourceRoute
  ```
- Hàm này gọi `buildRouteLookupKeys(sourceDB, sourceTable)`:
  ```go
  func buildRouteLookupKeys(sourceDB, sourceTable string) []string {
      return dedupeStrings([]string{
          fmt.Sprintf("%s|%s", sourceDB, sourceTable),
          fmt.Sprintf("%s:%s", sourceDB, sourceTable),
          sourceTable,
      })
  }
  ```
- `buildRouteLookupKeys` HOÀN TOÀN KHÔNG CÓ `connCode`!
- Do đó, khi gọi `ResolveSourceRoutes("core-trans-proxy-history-service", "trans_his")`, nó trả về MẢNG GỒM CẢ 2 ROUTES!

#### Lỗ hổng 3: `EventHandler` Fan-out nhầm vào cả 2 bảng Shadow
- Tại `event_handler.go:217`:
  ```go
  routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable)
  ...
  for _, route := range routes {
      tableConfig := route.TableConfig
      targetTable := tableConfig.TargetTable
      ...
      h.batchBuffer.Add(ctx, upsertRec)
  }
  ```
- `event_handler` coi tất cả các route trong `routes` là các logical clones cần fan-out.
- Hậu quả: Mỗi message từ `traitestmongodevct` được ghi vào CẢ 2 bảng shadow (`shadow_traitestmongodevct.trans_his` VÀ `shadow_traitestctphs.trans_his`). Message từ `traitestctphs` cũng tương tự!
- Thêm vào đó: `primaryRoute := routes[0]` luôn lấy `routes[0]` (luôn là `traitestmongodevct`), khiến việc phân tích PK và Schema Inspection của `traitestctphs` bị áp sai cấu hình!

#### Lỗ hổng 4: `KafkaConsumer` vứt bỏ thông tin Connector Name
- Trong `kafka_consumer.go:552`:
  `sourceRaw := event["source"]`
  `sourceTsMs = extractSourceTsMs(sourceRaw)`
- Debezium gửi kèm metadata nguồn cực kỳ chi tiết trong `event["source"]`:
  - `source["name"]`: chính là `topic.prefix` / tên connector (ví dụ `traitestmongodevct` hoặc `traitestctphs`).
  - `source["db"]`: tên database (`core-trans-proxy-history-service`).
  - `source["collection"]`: tên collection (`trans_his`).
- Nhưng `kafka_consumer.go` chỉ lấy `ts_ms`, không truyền `source["name"]` vào `cdcEvent` đóng gói!

#### Lỗ hổng 5: `_raw_data` đa hình (Extended JSON vs Plain JSON)
- Snapshot V2 dùng `bson.MarshalExtJSON(doc, false, false)` -> sinh ra Extended JSON:
  `{"_id": {"$oid": "6868aa0bcf6e97965bb6c351"}, "createdAt": {"$date": "2025-07-05T04:28:59.729Z"}}`.
- Debezium: Converter của Kafka Connect đã unwrap BSON phẳng:
  `{"_id": "6aaa3b467742982d574e2817", "createdAt": 1789541190671}`.
- `dynamic_mapper.go:121` ghi thẳng `rawData` vào `_raw_data` mà không có hàm chuẩn hóa đệ quy.

---

### 3. Giải Pháp Kỹ Thuật Tối Ưu

1. **Chuẩn hóa Truyền & Nhận diện Connection Code:**
   - Trong `kafka_consumer.go`: Trích xuất `connectorName := extractSourceConnectorName(sourceRaw)` và gắn vào `cdcEvent["source_conn"] = connectorName`.
   - Trong `event_handler.go`: Đọc `sourceConn := event.SourceConn` (hoặc phân giải từ `subject` nếu là topic có prefix).
   - Truyền `sourceConn` vào `ResolveSourceRoutes(sourceDB, sourceTable, sourceConn)`.

2. **Nâng cấp `ResolveSourceRoutes` tại `MetadataRegistryService`:**
   - Mở rộng signature: `ResolveSourceRoutes(sourceDB, sourceTable string, sourceConn ...string) []*metadata.ResolvedSourceRoute`.
   - Nếu `sourceConn` có giá trị: Ưu tiên tìm kiếm key theo `sourceConn:sourceDB|sourceTable`, `sourceConn:sourceTable`.
   - Nhờ vậy, `rs.routeCache["traitestmongodevct:core-trans-proxy-history-service|trans_his"]` CHỈ trả về đúng route của `traitestmongodevct`.
   - `rs.routeCache["traitestctphs:core-trans-proxy-history-service|trans_his"]` CHỈ trả về đúng route của `traitestctphs`.
   - Giữ tương thích ngược (fallback) nếu không truyền `sourceConn`.

3. **Chuẩn hóa `_raw_data` trong `DynamicMapper`:**
   - Cài đặt hàm đệ quy `normalizeMongoExtJSON(data map[string]interface{}) map[string]interface{}`:
     * `{"$oid": "..."}` -> `"..."`
     * `{"$date": ...}` -> số ms (int64) hoặc chuỗi ISO nếu là string.
   - Áp dụng vào `rawData` trước khi lưu vào `RawJSON` để cả Snapshot V2 lẫn Debezium đều sinh ra `_raw_data` phẳng 100%.
