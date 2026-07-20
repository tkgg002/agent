# Walkthrough — Fix Audit Sink & Transmute Issues (Phase 0 & Phase 1)

Đã hoàn thành toàn bộ các tasks trong **Phase 0** và **Phase 1** của kế hoạch hành động. Code đã được biên dịch thành công và vượt qua tất cả unit tests của shadow handler, master handler, và master service.

## Thay đổi chi tiết

### 1. [NEW / MODIFIED] [cdc_event.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/shadow/cdc_event.go)
- Thêm `KafkaTopic`, `KafkaPartition`, `KafkaOffset` vào `CDCEvent` và `UpsertRecord` struct để liên kết từng record với offset của nó trong Kafka.

### 2. [MODIFIED] [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)
- Map `KafkaTopic`, `KafkaPartition`, `KafkaOffset` từ `CDCEvent` sang `UpsertRecord` khi tạo record.
- Thêm proxy method `SetOnCommitOffsets` để liên kết callback từ `KafkaConsumer` xuống `BatchBuffer`.
- Bổ sung metrics tracking cho 2 điểm silent drop:
  - **Drop 3 (Source not registered):** `metrics.EventsDropped.WithLabelValues("source_not_registered", sourceTable).Inc()`
  - **Drop 4 (Missing PK):** `metrics.EventsDropped.WithLabelValues("missing_pk", sourceTable).Inc()`

### 3. [MODIFIED] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
- Thêm callback `onCommitOffsets` vào `BatchBuffer`.
- Trong `Flush()`, sau khi `batchUpsert` thành công cho một group, gom các offset cao nhất của từng partition và gọi `onCommitOffsets` để trigger commit lên Kafka.
- Thêm helper `writeFailedSyncLog` để handle log error và counter `metrics.DLQWriteFail.Inc()` khi failedSyncLogRepo/DB ghi DLQ thất bại (thay vì ignore error bằng `_`).

### 4. [MODIFIED] [prometheus.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go)
- Định nghĩa metric `EventsDropped`: `cdc_sink_events_dropped_total` với các label `reason` và `topic`.

### 5. [MODIFIED] [kafka_consumer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go)
- Đặt `CommitInterval: 0` trong `buildReader` (chỉ cho phép manual commit).
- Gán callback `SetOnCommitOffsets` trong `NewKafkaConsumer`.
- Loại bỏ block commit offset tức thời trong consume loop.
- Thêm helper `commitOffsets` để manual commit danh sách offsets nhận được từ BatchBuffer callback (hoặc DLQ write).
- Khi record bị lỗi và ghi DLQ thành công, commit offset của record đó ngay lập tức (vì đã an toàn trong DLQ).
- Bổ sung log và metrics cho 2 silent drop points:
  - **Drop 1 (Empty value):** `metrics.EventsDropped.WithLabelValues("empty_value", msg.Topic).Inc()`
  - **Drop 2 (Nil afterData):** `metrics.EventsDropped.WithLabelValues("nil_after_data", msg.Topic).Inc()`

### 6. [MODIFIED] [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)
- Thêm recover block trong background goroutine của transmute, tự động gọi `cancel()` context khi panic để giải phóng DB connection resources đang pending.

### 7. [MODIFIED] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
- Sửa bare type assertions trong dedup sang `switch type` để cover cả `int64` và `float64` (chống panic/drop dữ liệu do JSON float64 parsing trap).
- Tích hợp retry backoff logic cho `bulkUpsertMaster` (max 3 retries, exponential backoff với retryable DB errors).
- Bổ sung helper `isRetryableDBError` check các lỗi: deadlock, connection refused/reset/closed, bad connection, pg error codes `40001`, `40P01`, `55000`, `57P01`.
- Thêm log Warn chi tiết lý do khi rules skip records.
- Fallback gán `DefaultValue` nếu mapping/validation lỗi hoặc missing field cho non-nullable rules mà có default value.

### 8. [MODIFIED] [server_setup.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go)
- Chuyển NATS Subscribe của `cdc.cmd.transmute` và `cdc.cmd.transmute-shadow` sang `QueueSubscribe` với queue group `"transmute-workers"`.

---

## Kết quả kiểm thử

Đã chạy thành công toàn bộ unit tests cho shadow package, master handler, và master service:
- `go test -v ./internal/handler/shadow/...` → **PASS**
- `go test -v ./internal/service/master/...` → **PASS**
- `go test -v ./internal/handler/master/...` → **PASS**
- `go build -o /dev/null ./cmd/...` → **SUCCESS** (biên dịch sạch không lỗi)
