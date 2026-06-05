# Kế hoạch thực hiện (Plan)

## 1. Nghiên cứu & Xác định lỗi
- Đọc file `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/kafka_consumer.go` để xác định cách thức hoạt động của `RefreshTopics` và `buildReader`.
- Đọc file `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_queries.go` để xác định các câu lệnh SQL truy vấn bảng `failed_sync_logs`.

## 2. Giải pháp kỹ thuật
### Lỗi 1: Kafka Consumer Panic
- Trong `RefreshTopics` hoặc `buildReader`:
  - Kiểm tra độ dài của mảng topic. Nếu mảng topic rỗng:
    - Đóng reader hiện có (gọi phương thức close reader an toàn).
    - Không khởi tạo reader mới bằng `kafka.NewReader`.
    - Log cảnh báo ở level `warn` hoặc `info`: "No active topics configured, kafka consumer is going to idle state" (hoặc tương tự).
    - Đặt trạng thái reader nội bộ thành `nil`.
- Nếu có topic mới, thực hiện build reader bình thường.

### Lỗi 2: CMS Relation Missing
- Trong `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_queries.go`:
  - Xác định các câu truy vấn truy cập bảng `failed_sync_logs`.
  - Thay thế `"failed_sync_logs"` bằng `"cdc_system"."failed_sync_logs"`.

## 3. Xác minh (Verification)
- Chạy `go build` và `go test` trên cả hai repo `centralized-data-service` và `cdc-cms-service` để đảm bảo code biên dịch và pass toàn bộ tests.
- Chạy `/security-agent` để review an toàn.
