# Yêu cầu Refactor recon_dest_agent.go

Tài liệu này xác định mục tiêu và yêu cầu kỹ thuật cho việc tái cấu trúc file `recon_dest_agent.go` (652 dòng) tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go`.

## 1. Mục tiêu
- Giảm kích thước file `recon_dest_agent.go` bằng cách chuyển các phần logic không thuộc trách nhiệm quản lý client/connection sang các file helper chuyên biệt.
- Giữ vững cấu trúc code hiện tại, bảo đảm không thay đổi logic nghiệp vụ, thuật toán băm XOR hoặc các câu lệnh SQL Postgres.
- Đảm bảo biên dịch thành công và vượt qua 100% các bài kiểm thử unit tests của dự án.

## 2. Phạm vi thay đổi
Tất cả các thay đổi sẽ diễn ra cục bộ bên trong package `recon` (`internal/service/recon/`).
Không làm thay đổi API interface / signature của `ReconDestAgent` để tránh ảnh hưởng đến các package bên ngoài gọi vào (ví dụ: `recon_handler.go`, `recon_engine.go`).

## 3. Các ràng buộc kỹ thuật
- **Read-Only Enforce**: Giữ nguyên cơ chế `SET TRANSACTION READ ONLY` trong `readOnlyDB` để bảo vệ cơ sở dữ liệu Postgres replica/primary.
- **SQL Quoting**: Tất cả các tên bảng, tên cột phải được kiểm tra qua `validateIdent` và escaped qua `quoteIdent` / `quoteRelation` trước khi đưa vào SQL string.
- **XOR Hashing**: Đảm bảo sử dụng chung hàm `hashIDPlusTsMs` và `bucketIndex` để tạo mã băm XOR tương thích chéo với MongoDB side (`ReconSourceAgent`).
