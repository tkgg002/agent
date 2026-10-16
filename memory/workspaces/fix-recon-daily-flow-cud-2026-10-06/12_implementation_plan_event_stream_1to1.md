# Kế Hoạch Triển Khai: Luồng CDC 1:1 Không Gộp Sự Kiện (Event Stream 1:1)

## 1. Mục Tiêu Triển Khai
- Đáp ứng chính xác chỉ thị: "ins là ins, update là update, không gộp gì cả, chậm cũng được, cdc chính xác 1:1 giữa các trạm".
- Khắc phục triệt để hiện tượng 71 Update ở Shadow nhưng Master chỉ đếm được 6 Update.
- Sửa lỗi 404 của endpoint `GET /api/reconciliation/daily-flow` (rebuild và restart CMS service).

---

## 2. Chi Tiết Các Bước Kỹ Thuật

### Bước 1: Bổ sung kiểu `TransmuteEventItem` và bắn trọn vẹn events qua NATS JetStream
- **File**: `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go`
- **Thay đổi**:
  1. Định nghĩa `TransmuteEventItem` chứa: `SourceID`, `Op`, `SourceTs`, `RawData`, `IsDelete`.
  2. Trong `publishTransmuteTrigger`:
     - Lặp qua `records []*shadow.UpsertRecord`.
     - Tạo slice `events []TransmuteEventItem`.
     - Đưa `events` vào payload JSON của JetStream bên cạnh `_source_ids`.

### Bước 2: Nâng cấp `HandleTransmuteShadow` và `TableDebouncer` để không gộp event
- **File**: `centralized-data-service/internal/handler/master/transmute_handler.go`
- **Thay đổi**:
  1. Thêm trường `Events []TransmuteEventItem` vào struct nhận payload của `HandleTransmuteShadow`.
  2. Bắn tiếp `events` sang subject `cdc.cmd.transmute`.
  3. Thêm `Events []TransmuteEventItem` vào `TransmuteRequest`.
  4. Trong `runDebouncedGroup`:
     - Nếu trong batch có `Events`: Gom toàn bộ `Events` theo đúng thứ tự mảng tuần tự (preserve sequence), KHÔNG deduplicate qua `idMap`.
     - Gọi module xử lý sự kiện 1:1 `RunEvents(ctx, masterTable, events, ...)`.

### Bước 3: Thêm phương thức `RunEvents` trong `TransmuterModule`
- **File**: `centralized-data-service/internal/service/master/transmuter.go`
- **Thay đổi**:
  1. Hàm `RunEvents(ctx context.Context, masterName string, events []TransmuteEventItem, masterBindingID ...int64)`:
     - Tải thông tin master binding và mapping rules.
     - Với mỗi event trong `events`:
       * Nếu `event.IsDelete` hoặc `event.Op == "d"`: Thực hiện hard delete dòng tương ứng trên Master theo `source_id` -> `res.Updated++` hoặc `res.Deleted++`.
       * Nếu `event.Op == "c"` hoặc `event.Op == "r"`:
         - Build emits từ `event.RawData`.
         - Upsert vào Master table.
         - Tăng `res.Inserted++`.
       * Nếu `event.Op == "u"`:
         - Build emits từ `event.RawData`.
         - Upsert vào Master table.
         - Tăng `res.Updated++`.
     - Trả về `TransmuteResult` với số lượng `Inserted`, `Updated` khớp chính xác 1:1 với số lượng event nhận được từ Shadow/Kafka!

### Bước 4: Kiểm tra biên dịch & Unit Test
- Chạy `go build ./...` trong `centralized-data-service`.
- Chạy `go test ./internal/handler/master/... ./internal/service/master/...`.

### Bước 5: Rebuild & Restart CMS Service
- Biên dịch lại `cdc-cms-service`.
- Khởi động lại service trên cổng 8083 để nạp route `/reconciliation/daily-flow`.

### Bước 6: Kiểm tra xác nhận chất lượng (Definition of Done Gate G1–G8)
- G1: Truy vết yêu cầu: không gộp event, đếm đúng từng op.
- G3: Test thật chạy PASS.
- G6: Output correctness trên dữ liệu thực tế.
- G8: Bằng chứng vật lý trong workspace.
