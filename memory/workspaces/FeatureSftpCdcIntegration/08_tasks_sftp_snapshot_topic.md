# Danh sách Task chi tiết: Di chuyển tạo Topic Kafka SFTP sang nút Snapshot

Bản Checklist thực thi kỹ thuật.

---

- `[x]` **Task 1: Revert các thay đổi cũ về Active Binding**
  - `[x]` Khôi phục `update_shadow_binding.go` về nguyên bản (chỉ cập nhật status `is_active` vào DB).
  - `[x]` Khôi phục đăng ký `shadow-binding.update` trong `server.go` về nguyên bản.
  - `[x]` Khôi phục `update_registry.go` về nguyên bản.
  - `[x]` Khôi phục đăng ký `registry.update` trong `server.go` về nguyên bản.
- `[x]` **Task 2: Cấu hình lại Debeizum Connector (Bước Connection)**
  - `[x]` Bỏ logic skip connector `Create` trong `debezium_connector.go` để connector SFTP được tạo ngay lập tức khi tạo Connection.
  - `[x]` Đảm bảo `debezium_connector.go` không gọi `autoCreateKafkaTopic`.
- `[x]` **Task 3: Triển khai rẽ nhánh Snapshot cho SFTP**
  - `[x]` Sửa file `source_object_actions_handler.go` tại API `SnapshotV2`.
  - `[x]` Bổ sung dependency `db *gorm.DB` vào `SourceObjectActionsHandler`.
  - `[x]` Nếu `source_engine_type` của object là `sftp` (hoặc `file`, `csv`), thực hiện:
    1. Query tên topic và Kafka brokers.
    2. Chạy `autoCreateKafkaTopic` để tạo topic Kafka.
    3. Trả về response thành công (bypass việc gửi command NATS `snapshot.v2`).
- `[ ]` **Task 4: Bổ sung kiểm tra an toàn nil pointer & warning log**
  - `[ ]` Thêm nil check cho `h.db` ở hàm `SnapshotV2` để tránh panics trong môi trường unit tests.
  - `[ ]` Thêm check và log warning khi không tìm thấy connection code của sftp source object.
- `[x]` **Task 5: Đăng ký dependency trong `server.go`**
  - `[x]` Truyền `db` vào hàm khởi tạo `NewSourceObjectActionsHandler` tại `server.go`.
- `[x]` **Task 6: Biên dịch & Kiểm thử**
  - `[x]` Kiểm tra biên dịch dự án `cdc-cms-service`.
  - `[x]` Chạy `go test ./internal/...` để verify.
