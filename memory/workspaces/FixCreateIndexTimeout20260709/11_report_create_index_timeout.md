# Báo cáo thay đổi - Tối ưu hóa bất đồng bộ Create/Drop Index & Khắc phục lock-storm trong transmuter

## 1. Danh sách các file thay đổi và số dòng code
- **File handler**: `centralized-data-service/internal/handler/governance/index_handler.go` (Thay đổi khoảng 60 dòng code).
- **File logic**: `centralized-data-service/internal/service/master/transmuter.go` (Thay đổi khoảng 70 dòng code).

## 2. Mô tả thay đổi (Overview)
- **Tối ưu hóa IndexHandler**:
  - Chuyển `HandleCreateIndex` và `HandleDropIndex` sang xử lý bất đồng bộ.
  - Sau khi nhận payload và kiểm tra kết nối database thành công, worker gửi phản hồi `CommandResult` thành công lập tức về NATS để giải phóng API client.
  - Tiến trình `CREATE/DROP INDEX CONCURRENTLY` được thực hiện dưới nền trong một goroutine độc lập sử dụng detached context.
- **Khắc phục Lock Storm trong Transmuter**:
  - Thêm cache `ensuredShadowIndexes` dạng `map[string]bool` vào `TransmuterModule` để lưu trữ các index shadow đã được kiểm tra hợp lệ.
  - Sửa đổi logic `ensureShadowSourceIDIndex` để kiểm tra cache trước khi truy vấn Postgres. Nếu đã có trong cache, return ngay lập tức.
  - Khi phát hiện index chưa tồn tại hoặc bị `INVALID` trong DB, gán giá trị cache = true lập tức trước khi chạy tiến trình ngầm `DROP/CREATE INDEX CONCURRENTLY` để ngăn chặn các luồng transmuter song song spawn trùng lặp goroutine tạo index gây nghẽn lock (lock storm).
- **Kiểm thử**:
  - Chạy toàn bộ test suite của handler và transmuter thành công (PASS).
