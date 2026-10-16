# 09 Technical Solution: Quăng Lỗi Tường Minh, Chặn Commit Offset Khi Lỗi & Tua Ngược Offset Replay 490 Messages

---

## 1. Bản Chất Kỹ Thuật Của Vấn Đề

### A. Vì sao "lỗi vẫn ko đc quăng ra"?
1. **Lỗ hổng commit offset mù quáng khi có lỗi (`kafka_consumer.go:476-503`)**:
   - Khi `processMessage` gặp lỗi (`procErr != nil`), consumer gọi `kc.writeDLQ(ctx, msg, procErr)` đẩy vào bảng `failed_sync_logs`.
   - Ngay sau đó, tại dòng 499:
     ```go
     kc.commitOffsets(ctx, map[TopicPartition]int64{
         {Topic: msg.Topic, Partition: msg.Partition}: msg.Offset,
     })
     ```
     Code tự động commit offset của message lỗi lên broker! Kafka broker đánh dấu message đã hoàn tất và không bao giờ replay lại.
   - Consumer không in ra `os.Stderr`, không dừng vòng lặp, không crash.
2. **Activity Log không ghi nhận lỗi**:
   - `batch_buffer.go` chỉ ghi vào `cdc_activity_log` khi `batchUpsert` chạy trong database. Khi lỗi xảy ra sớm ở router (`source route not found in registry cache`), `batch_buffer` không bao giờ được gọi.
   - `adaptive_batcher.go` không hề insert vào database `cdc_activity_log`.
   - Do đó, trên CMS (tab "Log Kafka" của Data Integrity gọi `/api/activity-log?operation=kafka-consumer&target_table=trans_his`), bảng `cdc_activity_log` trống trơn -> User không thấy bất kỳ lỗi nào được quăng ra!

### B. Vì sao "bây giờ vẫn bị ko có kafka-consumer thành công"?
1. **Kafka Broker Offset đã ở cuối (Lag = 0)**:
   - Trên Kafka broker `10.200.186.203:9092`: Topic `cdc.goopaylocal.core-trans-proxy-history-service.trans-his` có committed offset là **478523**.
   - Toàn bộ 490 messages lịch sử (chứa các event lúc 10:20) đã bị commit trôi qua broker từ phiên trước.
   - Không có message mới nào trên broker, nên worker chỉ đứng chờ (`DeadlineExceeded` trong loop 200ms), không có sự kiện consume nào chạy.
2. **Tiến trình worker thật trên máy vẫn là binary cũ**:
   - Tiến trình `go run cmd/worker/main.go` (PID 7693) đang chạy từ 13:15 chưa được restart để nạp code mới.

---

## 2. Giải Pháp Triệt Để (3 Điểm Chốt)

### Chốt 1: Quăng Lỗi Tường Minh & Chặn Tuyệt Đối Commit Offset Khi Có Lỗi (`kafka_consumer.go`)
1. Khi `procErr != nil`:
   - **In trực tiếp ra `os.Stderr`**:
     ```go
     fmt.Fprintf(os.Stderr, "\n[KAFKA CONSUMER ERROR] topic=%s partition=%d offset=%d err=%s\n\n",
         msg.Topic, msg.Partition, msg.Offset, procErr.Error())
     ```
   - **Ghi ngay bản ghi `status: error` vào `cdc_activity_log`**:
     Khởi tạo hoặc dùng `ActivityLogger` để ghi trực tiếp vào `cdc_activity_log` với:
     - `operation`: `"kafka-consumer"`
     - `target_table`: `sourceTable` (hoặc `msg.Topic`)
     - `status`: `"error"`
     - `error_message`: `procErr.Error()`
     - `source_database`: `sourceDB`
     - `source_table`: `sourceTable`
     - `triggered_by`: `"kafka-consumer"`
   - **CHẶN COMMIT OFFSET**:
     TUYỆT ĐỐI KHÔNG gọi `kc.commitOffsets` khi `procErr != nil`! Message lỗi phải giữ nguyên offset để Kafka có thể replay lại khi operator sửa lỗi xong.

### Chốt 2: Khôi Phục Hoàn Toàn Tương Thích Ngược Trong `ResolveSourceRoutes`
1. Trong `metadata_registry_service.go:ResolveSourceRoutes`:
   - Nếu `conn` được truyền vào (ví dụ `conn = "core-trans-proxy-history-service"`):
     Lọc candidate connections.
   - **Nếu không tìm thấy route nào khớp với `conn` (`len(filtered) == 0`)**:
     Thay vì `return nil` gây drop toàn bộ traffic của các connector cũ, **fallback trả về `routes` ban đầu** nếu các routes này không có sự mâu thuẫn destination, hoặc log cảnh báo và fallback bảo toàn tương thích ngược!

### Chốt 3: Script Tua Ngược Offset (Rewind Offset) Cho Topic `trans-his`
- Cung cấp script chạy ngay:
  Reset offset của consumer group `cdc-worker-group-local` cho topic `cdc.goopaylocal.core-trans-proxy-history-service.trans-his` về offset **478033** (trước thời điểm 490 messages lúc 10:20).
- Sau khi reset offset và chạy worker mới:
  Kafka sẽ tự động phát lại toàn bộ 490 messages đó.
  Worker sẽ consume thành công, ghi vào Shadow DB, và đẩy Activity Log thành công hiển thị xanh lè trên CMS!
