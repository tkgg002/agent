# Requirements: Bổ sung unit tests toàn diện cho centralized-data-service

## 1. Mục tiêu & Phạm vi
Mục tiêu là bổ sung unit test cho tất cả các package nghiệp vụ cốt lõi (internal/service) và các package phụ trợ (pkgs) chưa có hoặc có rất ít kiểm thử. Đảm bảo toàn bộ logic nghiệp vụ quan trọng được bao phủ bởi các ca kiểm thử tự động, chạy độc lập và không phụ thuộc vào live infrastructure (DB, NATS, Kafka).

## 2. Yêu cầu chi tiết theo phân lớp

### A. Tầng Pkgs (Infra & Utilities)
- **R1.1 - pkgs/crypto**: Viết unit test đầy đủ cho `EncryptAES` và `DecryptAES` trong `aes.go`. Kiểm tra các trường hợp: plaintext rỗng, key ngắn, key dài, và quá trình giải mã ngược đảm bảo khôi phục đúng dữ liệu ban đầu.
- **R1.2 - pkgs/utils**: Viết bổ sung unit test cho `hash.go` để kiểm thử hàm `CalculateHash` với các cấu trúc dữ liệu nested phức tạp.
- **R1.3 - pkgs/natsconn**: Mock NATS connection để kiểm thử các helper truyền context và trace `ActionTrace` (`pkgs/natsconn/action_trace.go`).
- **R1.4 - pkgs/rediscache**: Viết test mock cho redis client.

### B. Tầng Internal Services
- **R2.1 - internal/service/metadata**: Bổ sung unit test cho `helpers.go` và `mapping_utils.go` để kiểm thử logic validate cấu hình, phân giải mapping rules từ schema, map dữ liệu kiểu nguồn-đích.
- **R2.2 - internal/service/orchestration**: Bổ sung unit test cho `provisioning_orchestrator.go` và `provisioning_orchestrator_helpers.go` (sử dụng mock repositories) để verify logic state machine chuyển trạng thái của pipeline.
- **R2.3 - internal/service/shadow**: Bổ sung unit test cho `child_explode.go` (xử lý bung mảng JSON) và `type_resolver.go` (phân giải kiểu dữ liệu động).

### C. Tầng Repositories (Database Mocking)
- **R3.1 - GORM Mocking**: Sử dụng `go-sqlmock` hoặc SQLite in-memory để kiểm thử các CRUD/query helpers phức tạp trong `internal/repository/` (như `connection_registry_repo.go`, `schema_log_repo.go`) mà không đòi hỏi kết nối Postgres vật lý.

## 3. Tiêu chí Đạt (DoD)
- Tất cả file test mới biên dịch và chạy pass 100% qua lệnh `go test ./...`.
- Tăng tỷ lệ test coverage thực tế của các package mục tiêu.
- Tuyệt đối không làm gãy các test tích hợp hiện tại.
- Tuân thủ quy ước cấu trúc file test:
  - Unit test thuần túy của package đặt trực tiếp trong package đó hoặc trong thư mục `test/` song song tương ứng.
