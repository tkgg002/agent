# Requirements: Masking SQL Query Text in Sensitive DB Traces

## Mục tiêu
Đảm bảo an toàn thông tin và tuân thủ nguyên tắc bảo mật cơ sở dữ liệu:
- **Cho phép** hiển thị đầy đủ câu truy vấn SQL bao gồm cả tham số thực tế (`db.statement` với query variables) đối với Control Plane database (`cdc` role) để phục vụ debug hệ thống nội bộ.
- **Ẩn toàn bộ giá trị thực tế (variables / values) hoặc mask thành `?`** đối với các database nhạy cảm chứa dữ liệu nghiệp vụ của khách hàng: `data source`, `shadow` (`shadow` role), và `master` (`dest` role). Vẫn giữ lại khung xương (skeleton) của câu SQL (ví dụ: `INSERT INTO ... VALUES (?, ?, ...)`) để phục vụ check log/traces cấu trúc truy vấn mà không làm lộ lọt dữ liệu nhạy cảm.

## Các yêu cầu chi tiết
1. **Centralized-Data-Service**:
   - Trong `pkgs/database/multi.go`: Dựa vào tham số `role string` trong `openGorm`, nếu `role == RoleControlPlane` ("cdc") thì đăng ký plugin `tracing.NewPlugin(tracing.WithDBStatement(true))`.
   - Ngược lại (với role `shadow` hoặc `dest`), đăng ký plugin với option loại bỏ tham số: `tracing.NewPlugin(tracing.WithDBStatement(true), tracing.WithoutQueryVariables())`. Option này sẽ tự động parse câu lệnh SQL và che giấu toàn bộ các giá trị thực tế truyền vào, thay bằng `?`.
   - Trong `cmd/admin-api/main.go`: Đăng ký plugin mặc định `tracing.NewPlugin(tracing.WithDBStatement(true))`.

2. **CDC-CMS-Service**:
   - Cập nhật hàm `NewPostgresConnection` trong `pkgs/database/postgres.go` nhận thêm tham số `role string`.
   - Nếu `role == "cdc"` thì đăng ký `tracing.NewPlugin(tracing.WithDBStatement(true))`.
   - Nếu `role == "shadow"` (hoặc các database nhạy cảm khác), đăng ký `tracing.NewPlugin(tracing.WithDBStatement(true), tracing.WithoutQueryVariables())`.
   - Cập nhật các nơi gọi `NewPostgresConnection` trong `cdc-cms-service` (ở `internal/server/server.go` và các tệp test/migration liên quan) để truyền đúng giá trị `role`.
