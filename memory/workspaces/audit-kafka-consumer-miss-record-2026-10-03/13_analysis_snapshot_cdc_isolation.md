# 13 — Analysis: Đánh giá Xung đột giữa Luồng Snapshot và Kafka Consumer CDC

> **Bối cảnh**: Hệ thống hiện tại có 2 luồng nghiệp vụ cốt lõi:
> 1. **Kafka Consumer (CDC Stream)**: Bản chất Realtime, độ trễ thấp (<1s), yêu cầu ổn định và **ưu tiên tuyệt đối (Strict P0 SLA)**.
> 2. **Snapshot Runner (Bulk Load)**: Bản chất Batch, khối lượng cực lớn (hàng triệu bản ghi), throughput-oriented, có thể chạy chậm, **ưu tiên thấp hơn (Background P1/P2)**.

---

## I. HIỆN TRẠNG MÃ NGUỒN: CHƯA HỀ TÁCH BIỆT MESSAGER & BUFFER

Đối soát trực tiếp mã nguồn tại `internal/server/server_setup.go`:

1. **Dùng chung một instance BatchBuffer duy nhất**:
   - `server_setup.go:152`:
     ```go
     batchBuffer := handlershadow.NewBatchBuffer(cfg.Worker.BatchSize, cfg.Worker.BatchTimeout, db, schemaAdapter, logger)
     ```
   - `server_setup.go:202`: Khởi tạo `eventHandler` và truyền `batchBuffer` này vào.
   - `server_setup.go:375-387`: `snapshotRunner` nhận `eventHandler` dùng chung này.
   - `server_setup.go:493-496`: `kafkaConsumer` cũng nhận `eventHandler` dùng chung này!

2. **Gom lẫn lộn bản ghi Bulk và bản ghi Realtime vào chung RAM Slice**:
   - Khi `KafkaConsumer` nhận message: Đẩy vào `batchBuffer.Add(record)` (`event_handler.go:460`).
   - Khi `SnapshotRunner` đọc dữ liệu lịch sử từ MongoDB/Source DB: Từng bản ghi cũng được đẩy vào cùng hàm `event_handler.HandleRaw(...)` và nhét vào cùng slice `batchBuffer.records`!
   - Không hề có cơ chế phân loại (tagging/routing) để cách ly 2 luồng.

---

## II. 4 TÁC HẠI CHÍ MAT KHI GOM CHUNG 2 LUỒNG

### 1. Nhiễm bẩn Offset Kafka (Kafka Offset Corruption & False Commit)
- Tại `snapshot_runner_handler.go:827` và `:889`, Snapshot Runner định kỳ gọi `r.eventHandler.FlushBatchBuffer(ctx)`.
- Hàm này gọi `batchBuffer.FlushWithContext(ctx)`.
- Khi đó, toàn bộ slice `bb.records` bị rút ra để ghi DB. Nếu trong slice đó có tình cờ chứa các bản ghi của Kafka Consumer:
  - Dòng 279–291 của `batch_buffer.go` sẽ duyệt qua toàn bộ batch và **gọi callback `onCommitOffsets` commit offset lên Kafka**!
  - Nghĩa là: **Một tác vụ nền của Snapshot lại trực tiếp ra lệnh commit offset cho Kafka Consumer!**
  - Nếu trong mớ bản ghi đó, bảng của Snapshot bị lỗi DB hoặc timeout, các bản ghi Kafka có thể bị fail oan hoặc bị commit offset trôi qua khi dữ liệu chưa thực sự an toàn.

### 2. Chiếm đoạt và Cạn kiệt Connection Pool DB (Resource Starvation)
- Cấu hình `dbPool.maxOpenConn = 100` (`config-production.yml:18`).
- Cả Snapshot và Kafka CDC dùng chung một pool kết nối này.
- Khi Snapshot chạy hàng chục nghìn bản ghi, `FlushWithContext` mở tối đa 20 goroutines (`g.SetLimit(20)`) thực thi các transaction bulk insert lớn.
- Hệ quả: Snapshot chiếm trọn 100 connections của DB.
- Khi Kafka Consumer cần ghi một sự kiện realtime, nó bị chặn (connection pool wait) -> Dẫn đến timeout hoặc kích hoạt lỗi `SQLSTATE 53300: sorry, too many clients already`.
- Kafka Consumer rơi vào đường mã P2 / P3 (đã audit trước đó) -> **Gây mất bản ghi realtime!**

### 3. Nghẽn Hàng đợi Đầu hàng (Head-of-Line Queue Contention)
- Dung lượng buffer `maxSize = 1000`.
- Khi Snapshot bơm dữ liệu với tốc độ cao, buffer bị lấp đầy bởi bản ghi Snapshot trong vài mili-giây.
- Các message CDC realtime từ Kafka bị kẹt lại phía sau, không được flush kịp thời, khiến **Consumer Lag tăng vọt giả tạo**.

### 4. Vi phạm SLA Nghiệp vụ
- Khách hàng và hệ thống vận hành cần dữ liệu biến động số dư / giao dịch hiển thị trong vòng 1-2 giây.
- Việc Snapshot chạy song song và "dìm" luồng CDC khiến dữ liệu nghiệp vụ bị trễ hàng chục phút, gây gián đoạn quy trình đối soát realtime.
