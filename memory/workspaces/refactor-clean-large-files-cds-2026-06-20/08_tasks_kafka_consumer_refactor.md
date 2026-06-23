# Tasks List - Phân rã file kafka_consumer.go

Checklist chi tiết thực hiện các bước cho Muscle:

- [ ] Task 1: Khởi tạo các file phụ trợ mới trong `internal/handler/shadow/`:
  - [ ] `adaptive_batcher.go` (Adaptive batching & stats)
  - [ ] `avro_helper.go` (Avro codec & unwrap helpers)
  - [ ] `dlq_helper.go` (DLQ writing & diagnostics)
  - [ ] `topic_helper.go` (Topic discovery & auto-refresh)
  - [ ] `utils.go` (Tracing, transient errors & error classification)
- [ ] Task 2: Cập nhật file `kafka_consumer.go` gốc (lược bỏ các phần code đã di chuyển sang các file helper).
- [ ] Task 3: Biên dịch kiểm tra cú pháp và sửa lỗi biên dịch (nếu có):
  ```bash
  go build ./...
  ```
- [ ] Task 4: Chạy unit tests cho shadow package:
  ```bash
  go test -v ./internal/handler/shadow/...
  ```
- [ ] Task 5: Chạy unit tests cho toàn bộ dự án:
  ```bash
  go test ./...
  ```
- [ ] Task 6: Kiểm tra hoạt động của service bằng log/smoketest (sử dụng service đang chạy sẵn ở background `make run`).
- [ ] Task 7: Chạy kiểm tra bảo mật `/security-agent` trước khi báo cáo kết thúc.
- [ ] Task 8: Tạo báo cáo `report_kafka_consumer_refactor.md` ghi nhận số dòng code thay đổi và audit lại quá trình.
