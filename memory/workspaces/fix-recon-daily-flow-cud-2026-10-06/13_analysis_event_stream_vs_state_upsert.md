# Phân Tích Kỹ Thuật: Gốc Rễ Lệch Số Liệu Master Updated Và Giải Pháp CDC 1:1 Không Gộp

## 1. Hiện Tượng Gốc Rễ Được Chỉ Ra

Khi theo dõi luồng CDC trên dashboard đối soát:
- **Kafka Sink**: Nhận 127 events (56 `c` - Create/Insert, 71 `u` - Update, 0 `d` - Delete).
- **Shadow Table**: Ghi nhận 127 events (56 `c`, 71 `u`, 0 `d`), ghi vật lý vào DB là 62 dòng duy nhất (do 56 ID mới + 6 ID cũ).
- **Master Table**: Chỉ ghi nhận `56 Inserted, 6 Updated, 0 Skipped`! 
  -> Thiếu hụt nghiêm trọng: 71 event Update ở Shadow biến thành chỉ còn 6 Update ở Master (mất 65 Update).

## 2. Truy Tìm Từng Dòng Code Gây Ra Hiện Tượng

### Điểm 1: Mất thông tin Operation (`Op`) tại Trigger Publisher
- **Vị trí**: `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go` dòng 30–68:
  ```go
  sourceIDs := make([]string, 0, len(records))
  for _, r := range records {
      if r.PrimaryKeyValue != "" {
          sourceIDs = append(sourceIDs, r.PrimaryKeyValue)
      }
  }
  // Bắn NATS JetStream với subject cdc.cmd.transmute-shadow:
  payload, _ := json.Marshal(map[string]any{
      "shadow_table":          shadowTable,
      "shadow_schema":         shadowSchema,
      "shadow_connection_key": "default",
      "_source_ids":           sourceIDs,
      "triggered_by":          triggeredBy,
      "correlation_id":        ...,
  })
  ```
- **Hệ quả**: Mặc dù `records` chứa đầy đủ `Op` ("c", "u", "d"), `RawData`, `SourceTs`, nhưng hàm `publishTransmuteTrigger` chỉ gom chuỗi ID thô (`_source_ids`). Toàn bộ bản chất CDC event bị vứt bỏ ngay tại đây!

### Điểm 2: Gộp mẻ (Coalescing / Deduplication) tại NATS Debouncer
- **Vị trí**: `centralized-data-service/internal/handler/master/transmute_handler.go` dòng 448–470:
  ```go
  idMap := make(map[string]bool)
  for _, t := range batch {
      for _, id := range t.Req.SourceIDs {
          if id != "" {
              idMap[id] = true
          }
      }
  }
  var uniqueIDs []string
  for id := range idMap {
      uniqueIDs = append(uniqueIDs, id)
  }
  ```
- **Hệ quả**: Nếu một ID có 1 event Insert và 2 event Update liên tiếp trong mẻ, `idMap` gộp cả 3 event thành đúng 1 ID duy nhất trong `uniqueIDs`.

### Điểm 3: Đọc Snapshot State thay vì Event Log từ Shadow DB
- **Vị trí**: `centralized-data-service/internal/service/master/transmuter.go` dòng 356:
  ```go
  shadowRows, err = t.fetchShadowBatch(ctx, masterRow, lastGpayID, chunkIDs)
  ```
  với câu lệnh SQL:
  ```sql
  SELECT _gpay_id, _source_id, _raw_data, _source_ts, _deleted 
    FROM shadow 
   WHERE _source_id IN (uniqueIDs)
  ```
- **Hệ quả**: Bảng Shadow là một bảng lưu trạng thái hiện tại (State Table) có khóa duy nhất trên `_source_id`. Câu query này chỉ trả về đúng 1 bản ghi mới nhất cho mỗi ID. Toàn bộ các trạng thái trung gian (intermediate updates) đã bị ghi đè ở Shadow DB.

### Điểm 4: Upsert đếm theo `xmax = 0` (Postgres row creation)
- **Vị trí**: `centralized-data-service/internal/service/master/transmuter.go` dòng 1177, 1201–1207:
  ```sql
  INSERT INTO master (...) VALUES (...)
  ON CONFLICT (id) DO UPDATE SET ...
  RETURNING (xmax = 0) AS inserted
  ```
  ```go
  for _, r := range resRows {
      if r.Inserted {
          inserted++
      } else {
          updated++
      }
  }
  ```
- **Hệ quả**:
  - Với 56 ID mới: Postgres chèn mới -> `xmax = 0` -> `inserted = 56`.
  - Với 6 ID đã tồn tại từ trước: Postgres cập nhật -> `xmax != 0` -> `updated = 6`.
  - Kết quả: Master telemetry báo `56 Insert, 6 Update`. 
  - 65 event Update xảy ra trên các ID mới đã bị "nuốt chửng" vào lệnh Insert ban đầu, khiến người vận hành nhìn thấy telemetry bị lệch trầm trọng!

---

## 3. Giải Pháp Triệt Để: Luồng CDC Sự Kiện 1:1 (Event-Driven Stream Không Gộp)

Người dùng đã ra chỉ thị tối cao:
> *"rồi làm sao để có ko gộp nữa. tao cần nó trực quan cho tao, sao mày lằng nhằng quá. tao ko cần nhanh hay tiện, tao cần cdc chinh xác ins là ins, update là update, gộp cái gì mà gộp , chậm cũng đc. mày hiểu ko... thực hiện đi"*

### Thiết Kế Kiến Trúc 1:1:
1. **Định nghĩa `TransmuteEventItem`:**
   ```go
   type TransmuteEventItem struct {
       SourceID string          `json:"source_id"`
       Op       string          `json:"op"` // "c" (insert), "u" (update), "d" (delete), "r" (snapshot)
       SourceTs int64           `json:"source_ts"`
       RawData  json.RawMessage `json:"raw_data,omitempty"`
       IsDelete bool            `json:"is_delete"`
   }
   ```
2. **Gửi trọn vẹn sự kiện qua NATS JetStream:**
   - Trong `publishTransmuteTrigger`, serialize mảng `events: []TransmuteEventItem` từ `records []*shadow.UpsertRecord`.
   - JetStream payload chứa danh sách tuần tự tất cả các event.
3. **Bỏ gộp tại Debouncer khi có Events:**
   - Trong `transmute_handler.go`: Nếu payload mang danh sách `events`, giữ nguyên thứ tự và số lượng event, KHÔNG chạy qua `idMap` deduplicate.
4. **Xử lý sự kiện 1:1 tại Master Transmuter:**
   - Với mỗi event:
     - `op == "c"` hoặc `"r"`: Chuyển đổi và upsert -> Tính vào `res.Inserted++`.
     - `op == "u"`: Chuyển đổi và upsert -> Tính vào `res.Updated++`.
     - `op == "d"` hoặc `is_delete == true`: Xóa khỏi Master -> Tính vào `res.Deleted++` / `res.Updated++`.
   - Kết quả: Nếu Kafka/Shadow có 56 `c` và 71 `u`, Master cũng xử lý và đếm đúng 56 `Inserted` và 71 `Updated`!
