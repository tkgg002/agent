# Báo cáo thay đổi: Trì hoãn khởi tạo SFTP Connector & Topic Kafka theo sự kiện Active Binding

Báo cáo chi tiết các file đã thay đổi, số lượng dòng code và tóm tắt thay đổi trong phiên làm việc ngày 2026-08-12.

---

## 1. Danh sách file đã sửa đổi (Modified Files)

### 1. [`debezium_connector.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/source/debezium_connector.go)
- **Hành động:** `[MODIFY]`
- **Mô tả:** 
  - Sửa đổi hàm `Handle` của `CreateSystemConnectorHandler`. Bỏ qua hoàn toàn bước gọi HTTP sang Kafka Connect để tạo connector đối với nguồn SFTP.
  - Xóa bỏ hàm `autoCreateKafkaTopic` và logic tự tạo topic khi tạo connection (xóa block lines 159-171).
  - Dọn dẹp các import `"github.com/segmentio/kafka-go"` và `"os"` không còn sử dụng.
- **Số dòng thay đổi:** Xóa 18 dòng, sửa 15 dòng.

### 2. [`update_shadow_binding.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/shadow/update_shadow_binding.go)
- **Hành động:** `[MODIFY]`
- **Mô tả:**
  - Định nghĩa hàm `autoCreateKafkaTopic` và thêm các import `"github.com/segmentio/kafka-go"`, `"os"`.
  - Bổ sung các dependency: `ports.SourceRepo`, `ports.SystemConnectorRepo`, `KafkaConnectorWriter`, `*gorm.DB` vào `UpdateShadowBindingHandler`.
  - Triển khai logic: Khi bật liên kết (`IsActive = true`), hệ thống tự động gọi `autoCreateKafkaTopic` tạo topic Kafka trước, sau đó dựng lại cấu hình đầy đủ từ database (`options_json`) và khởi chạy connector trên Kafka Connect. Khi tắt liên kết (`IsActive = false`), gửi yêu cầu `Delete` tới Kafka Connect để xóa connector.
- **Số dòng thay đổi:** Ghi đè toàn bộ tệp (từ 79 dòng lên 187 dòng).

### 3. [`server.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go)
- **Hành động:** `[MODIFY]`
- **Mô tả:** Cập nhật đăng ký Handler cho `shadow-binding.update` bằng cách truyền các dependency mới vào `NewUpdateShadowBindingHandler`.
- **Số dòng thay đổi:** 8 dòng.

### 4. [`kafka_connect.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/http/kafka_connect.go)
- **Hành động:** `[MODIFY]`
- **Mô tả:** Bổ sung helper `sanitizeSFTPURI` để lọc che (sanitize) mật khẩu nhạy cảm của SFTP URI trước khi lưu trữ vào DB cdc_sources, tránh rò rỉ credential.
- **Số dòng thay đổi:** 22 dòng.

### 5. [`source_object_v2_sync.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/source_object_v2_sync.go)
- **Hành động:** `[MODIFY]`
- **Mô tả:** Bổ sung case `"sftp"`, `"file"`, `"csv"`, `"json"`, `"kafka"` vào hàm `normalizeSourceEngine` để định danh đúng loại nguồn SFTP thay vì bị fallback nhầm sang `postgresql`.
- **Số dòng thay đổi:** 6 dòng.

---

## 2. Kết quả Biên dịch và Kiểm thử (Verification)
- Lệnh biên dịch: `go build ./cmd/... ./internal/...` trong `cdc-cms-service` và `centralized-data-service` thành công 100%.
- Lệnh unit test: `go test ./internal/...` thành công 100%.
- Log chạy runtime của API Server (`task-3889`) và Worker Daemon (`task-3856`) đều kết nối tốt, hoạt động trơn tru.
