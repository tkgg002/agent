# 08_tasks_activity_log_fix.md

## Danh sách Tác vụ Chi tiết (Writer Data Standardization)

- [ ] Task 1: Sửa `batch_buffer.go` (`internal/handler/shadow/batch_buffer.go`) truyền `tableName` thuần vào `act.Start("kafka-consumer", tableName, "kafka-consumer")` và bổ sung `shadow_schema` vào `details`.
- [ ] Task 2: Sửa `sinkworker/worker.go` (`internal/sinkworker/worker.go`) truyền `tableName` thuần vào `w.activity.Start("sink-upsert", tableName, "kafka-consumer")`.
- [ ] Task 3: Chuẩn hóa `transmute_handler.go` (`internal/handler/master/transmute_handler.go`):
  - Đổi `"triggered_by": "kafka-consumer-hook"` thành `"triggered_by": "kafka-consumer"`.
  - Thêm `"correlation_id"` vào `details` JSON của Transmute Log.
  - Loại bỏ `"duration_ms"` dư thừa khỏi `details` JSON.
- [ ] Task 4: Chạy `go test ./...` kiểm tra toàn bộ suite `centralized-data-service`.
- [ ] Task 5: Verify API Read ở `cdc-cms-service` đảm bảo hiển thị 100% metadata mà KHÔNG sửa bất kỳ dòng SQL Read nào.
- [ ] Task 6: Tạo file báo cáo `11_report_activity_log_fix.md` và `14_walkthrough_activity_log_fix.md`.
