# Report: Refactor and Drainage of DB & NATS from API and App Layers

## 1. Summary of Changes
- **Tổng số files thay đổi**: 79
- **Tổng số dòng code thêm**: 1837 insertions
- **Tổng số dòng code xóa/thay đổi**: 1354 deletions

## 2. Chi tiết Drainage theo Layer

### API Layer (`internal/api/`)
- Loại bỏ toàn bộ `gorm.ErrRecordNotFound` và các tham chiếu trực tiếp đến `*gorm.DB` / `h.db`.
- Map toàn bộ lỗi GORM sang `ports.ErrRecordNotFound` thông qua Port Interface.
- Loại bỏ direct NATS connection (`*nats.Conn`) và thay thế bằng `ports.ReloadPublisher` hoặc `ports.Publisher`.

### Application Layer (`internal/app/`)
- Di chuyển/refactor các Command Handlers (như `register_registry.go`, `bulk_register_registry.go`, `update_registry.go`, `update_mapping_rule.go`) để sử dụng `ports.ReloadPublisher` thay cho direct dependency `*natsconn.NatsClient`.
- Cập nhật saga steps để truyền context (`ctx`) phục vụ Trace Propagation.

### Infrastructure Layer (`internal/infra/`)
- **Persistence**: Cập nhật các repository concrete implementations trong `internal/infra/persistence/...` để thực hiện mapping lỗi `gorm.ErrRecordNotFound` thành `ports.ErrRecordNotFound`.
- **Messaging**: Triển khai `messaging.NewReloadPublisher` trong `nats_publisher.go` thực thi port interface `ports.ReloadPublisher`.

### Server Wiring (`internal/server/server.go`)
- Khởi tạo adapter `reloadPublisher := messaging.NewReloadPublisher(natsClient.Conn)` tại composition root.
- Cập nhật dependency injection để truyền `reloadPublisher` thay thế `natsClient` cho các command handlers và handlers có nhu cầu.

## 3. Kết quả Xác minh (Verification)
- Biên dịch thành công: `go build ./...` passes
- Kiểm thử thành công: `go test ./...` passes (tất cả các package test suites đều ok)
