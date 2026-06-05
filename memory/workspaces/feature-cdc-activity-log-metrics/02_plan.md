# Plan: Fixing CDC Activity Log Metrics / Kế hoạch sửa đổi Activity Log Metrics

## English Version
### Phase 1: Implement Synchronous DB Upsert in BatchBuffer
1. Create `WriteRecordSync(record *model.UpsertRecord) (int, error)` in `internal/handler/batch_buffer.go`.
2. This method prepares the table and runs the DB upsert SQL synchronously.
3. If the DB operation fails, it returns the error without inserting into `failed_sync_logs` internally to prevent duplicate writes (since `KafkaConsumer` will write to DLQ).
4. If it succeeds, it increments `SyncSuccess` metrics and returns `1, nil`.

### Phase 2: Refactor EventHandler to execute synchronously
1. Update `processEvent` in `internal/handler/event_handler.go` to call `h.batchBuffer.WriteRecordSync(record)` instead of `h.batchBuffer.Add(record)`.
2. Accumulate `written` rows from each target route and return the sum.
3. Propagate any database errors immediately to the caller (`HandleRaw`).

### Phase 3: Refactor KafkaConsumer message processing
1. Update `KafkaConsumer.processMessage` to return the number of rows affected and an error.
2. In the consumption loop, adjust the metrics and statistics:
   - If error occurs: increment `batch.failed` and invoke `writeDLQ` (which persists the event in `failed_sync_logs`).
   - If successful: increment `batch.success` and accumulate `batch.rowsAffected` using the returned value.
3. Verify that logs for individual events remain at `DEBUG` level while the batch-level logs print at `INFO` level.

### Phase 4: Verification and Smoke Testing
1. Restart the worker using `make run`.
2. Trigger CDC events (e.g. via Debezium snapshot signals or DB mutations) and monitor logs.
3. Check `cdc_activity_log` to verify `rows_affected` reflects the actual database materializations.
4. Verify `failed_sync_logs` contains single entries for failed events.

---

## Tiếng Việt Version
### Phase 1: Triển khai DB Upsert Đồng bộ trong BatchBuffer
1. Thêm phương thức `WriteRecordSync(record *model.UpsertRecord) (int, error)` trong `internal/handler/batch_buffer.go`.
2. Phương thức này sẽ chuẩn bị bảng và thực thi SQL upsert đồng bộ.
3. Nếu DB operation lỗi, trả về lỗi mà không ghi vào `failed_sync_logs` để tránh trùng lặp log (vì `KafkaConsumer` sẽ ghi vào DLQ).
4. Nếu thành công, tăng metric `SyncSuccess` và trả về `1, nil`.

### Phase 2: Cấu trúc lại EventHandler để xử lý đồng bộ
1. Cập nhật `processEvent` trong `internal/handler/event_handler.go` để gọi `h.batchBuffer.WriteRecordSync(record)` thay vì `h.batchBuffer.Add(record)`.
2. Cộng dồn số dòng `written` từ các route và trả về tổng số.
3. Lan truyền (propagate) lỗi database lập tức về phía caller (`HandleRaw`).

### Phase 3: Cấu trúc lại xử lý tin nhắn của KafkaConsumer
1. Cập nhật `KafkaConsumer.processMessage` để trả về số dòng bị ảnh hưởng và lỗi.
2. Trong vòng lặp consume, cập nhật số liệu thống kê:
   - Nếu xảy ra lỗi: tăng `batch.failed` và gọi `writeDLQ` (lưu vào `failed_sync_logs`).
   - Nếu thành công: tăng `batch.success` và cộng dồn `batch.rowsAffected` từ giá trị trả về.
3. Đảm bảo log của từng message riêng lẻ giữ mức `DEBUG` và log của batch giữ mức `INFO`.

### Phase 4: Xác thực và Smoke Test
1. Khởi động lại worker bằng `make run`.
2. Kích hoạt các CDC event (ví dụ bằng debezium snapshot hoặc thay đổi DB) và theo dõi log.
3. Kiểm tra bảng `cdc_activity_log` để xác nhận `rows_affected` phản ánh đúng số dòng thực tế được ghi.
4. Xác minh `failed_sync_logs` chỉ chứa 1 dòng duy nhất cho mỗi bản ghi lỗi.
