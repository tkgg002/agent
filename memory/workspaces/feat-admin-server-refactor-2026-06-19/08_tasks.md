# Tasks: Refactor Admin HTTP Server

## Task: Refactor Server.go & Server_test.go
- **Phase**: GĐ2 Safety net
- **Service Group**: Gateways
- **Service(s)**: centralized-data-service
- **Mô tả**: Tái cấu trúc logic HTTP Server trong `internal/admin/server.go` và cập nhật các unit test tương ứng trong `internal/admin/server_test.go` theo đúng thiết kế an toàn, hiệu suất, tránh rò rỉ bộ nhớ (memory leak) ở rate limiter và bypass auth middleware.
- **Trạng thái**: [ ] TODO

### [Context]
- Current state: Hoàn thành refactor admin/helpers.go, tất cả unit test đều pass. Các thay đổi chưa commit.
- Dependencies: Không có service bên ngoài nào bị ảnh hưởng trực tiếp do chỉ sửa đổi cấu trúc server Gin và HTTP handlers nội bộ của admin API.
- ADR liên quan: N/A
- Logs/Error: N/A

### [Definition of Done]
- [ ] `rateLimiterStore` sử dụng IP thay vì token, giới hạn bộ nhớ qua cache TTL / cleanup routine chạy định kỳ, chống OOM.
- [ ] Đổi thứ tự middleware trong Gin: Body Limit -> Rate Limit -> Auth Middleware.
- [ ] Đổi `engine` thành public `Router` trong struct `Server`, loại bỏ hàm `EngineForTest()`.
- [ ] Cập nhật toàn bộ các lời gọi `EngineForTest()` trong `server_test.go` thành `Router`.
- [ ] Đảm bảo dự án biên dịch thành công (`go build ./...`).
- [ ] Đảm bảo 100% unit tests chạy qua (`go test -v ./internal/admin/...`).
- [ ] Chạy `/security-agent` để đảm bảo an toàn bảo mật.
- [ ] Ghi nhận đầy đủ tiến trình thực hiện vào `05_progress.md`.
