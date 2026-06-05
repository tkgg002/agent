# Danh sách Task (Task List)

- [x] **Task 1**: Cập nhật `internal/service/schema_inspector.go` để xử lý logic fallback schema.
- [x] **Task 2**: Cập nhật `internal/handler/event_handler.go` đổi `WriteRecordSync(record)` thành `Add(record)` và xử lý kết quả trả về `totalWritten` cho `processEvent` sao cho không làm vỡ code (metric rows written bây giờ chỉ có thể trả về số lượng event được enqueue vào buffer thay vì real affected rows).
- [x] **Task 3**: Build và Restart `centralized-data-service` (làm bằng `go build ./...` và kiểm tra `make run`).
- [ ] **Task 4**: Thao tác Reset Kafka Consumer Group Offset cho topic `cdc.goopay.scheduler-service.schedule_histories`.
