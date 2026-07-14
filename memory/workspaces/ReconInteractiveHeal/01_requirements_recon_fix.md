# Yêu cầu Chi tiết (Specs) - Khắc phục lỗi Biên dịch & Cấu trúc của Package Recon Handler

## Mục tiêu
Khắc phục triệt để các lỗi biên dịch và cấu trúc phát sinh sau khi tách biệt `HealHandler` thành `CheckHealHandler` (đề xuất/check) và `ExecuteHealHandler` (thực thi). Đảm bảo toàn bộ gói `internal/handler/recon` biên dịch thành công, chạy kiểm thử (unit tests) vượt qua 100% và hệ thống cdc-worker khởi động chính xác.

## Yêu cầu chi tiết

### 1. Sửa lỗi biên dịch trong `recon_heal_fetch.go`
- Thay đổi receiver của hàm `FetchAndWriteByIDs` từ `*HealHandler` sang `*ExecuteHealHandler` để khớp với logic thực thi được gọi từ `CheckHealHandler` (`h.executeHeal.FetchAndWriteByIDs`).

### 2. Tái cấu trúc Unit Test trong `recon_heal_v4_test.go`
- Cập nhật alias `ReconHandler` trỏ về `CheckHealHandler`.
- Thay đổi `NewReconHandler` để khởi tạo cả `ExecuteHealHandler` và `CheckHealHandler`, sau đó liên kết chúng với nhau.
- Cập nhật phương thức `WithBackfill` trên `CheckHealHandler` (thông qua alias) để tiêm `NatsPublisher` vào `ExecuteHealHandler` tương ứng nhằm thực hiện các lệnh transmute.

### 3. Điều chỉnh Server Setup trong `internal/server/server_setup.go`
- Loại bỏ khởi tạo `NewHealHandler` cũ.
- Khởi tạo `ExecuteHealHandler` với các dependency: `shadowDB`, `NatsPublisher`, `EventHandler`.
- Khởi tạo `CheckHealHandler` với `ReconBase`, `reportRepo`, `ExecuteHealHandler` và `healer`.
- Cập nhật đăng ký NATS subscriptions:
  - `cdc.cmd.recon-heal` -> Đăng ký nhận bằng `checkHealHandler.HandleReconHeal`.
  - `cdc.cmd.execute-heal` -> Đăng ký nhận bằng `executeHealHandler.HandleExecuteHeal`.

### 4. Đảm bảo chất lượng & Kiểm định (Definition of Done)
- Toàn bộ gói `internal/handler/recon` biên dịch thành công.
- Các unit tests trong gói `internal/handler/recon` chạy vượt qua 100%.
- Kiểm tra toàn bộ mã nguồn của dự án (`go build ./...`) không bị lỗi biên dịch.
