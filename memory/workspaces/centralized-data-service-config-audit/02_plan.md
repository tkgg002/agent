# Implementation Plan: Config Audit & Cleanup

## English Version
### Goal Description
Clean up the centralized-data-service configuration files by completing Phase 2 and Phase 3 of the config audit. The current `db:` block in the configuration is poorly named since it only contains connection pool settings, and `config.go` still retains dead code for legacy DB connections.

### Proposed Changes
1. **Yaml Config Files**: Rename the `db:` key to `dbPool:` in `config-local.yml`, `config-production.yml`, and `config-sample.yml` to accurately reflect its usage.
2. **`config/config.go`**:
    - [MODIFY] Change the `DBConfig` struct to `DBPoolConfig` and keep only pool-related fields (`MaxOpenConn`, `MaxIdleConn`, `ConnMaxLifetime`).
    - [MODIFY] Change the `DB` field in `AppConfig` to `DBPool DBPoolConfig` mapped to `dbPool`.
    - [MODIFY] Move the `ReadReplicaDSN` field to the root level of `AppConfig`.
    - [DELETE] Remove dead methods `DSN()` and `PgxDSN()` from `DBConfig`.
    - [MODIFY] Clean up `validateConfig()`, `applyDBFallbacks()`, and `applyEnvOverrides()` to remove legacy `cfg.DB` handling.
3. **`pkgs/database/postgres.go`**:
    - [MODIFY] Update `NewPostgresConnection` to use `cfg.SystemDBURL()` instead of the removed `cfg.DB.DSN()`.
    - [MODIFY] Update pool configurations to reference `cfg.DBPool`.
    - [MODIFY] Update `NewPostgresReadReplica` to use the root-level `cfg.ReadReplicaDSN`.
4. **`internal/server/worker_server.go`**:
    - [MODIFY] Update struct reference to use `cfg.ReadReplicaDSN` instead of `cfg.DB.ReadReplicaDSN`.

### Verification Plan
- Run `go build ./...` to ensure there are no compilation errors.
- Run `make run` or manually boot the service to ensure the application starts correctly with the new `dbPool` block.
- Verify startup logs are clean and without errors.

---

## Phiên bản Tiếng Việt
### Mô tả Mục tiêu
Làm sạch các tệp cấu hình của `centralized-data-service` bằng cách hoàn thành Phase 2 và Phase 3 của đợt audit cấu hình. Khối `db:` hiện tại bị User đánh giá là "vớ vẩn" do nó chỉ chứa các thiết lập kết nối (connection pool) chứ không có thông tin host/port, đồng thời `config.go` vẫn còn giữ các đoạn code chết (dead code) hỗ trợ legacy database connection.

### Đề xuất Thay đổi
1. **Các file Yaml**: Đổi tên key `db:` thành `dbPool:` trong `config-local.yml`, `config-production.yml`, và `config-sample.yml`.
2. **`config/config.go`**:
    - [MODIFY] Đổi `DBConfig` thành struct `DBPoolConfig`, chỉ giữ lại `MaxOpenConn`, `MaxIdleConn`, `ConnMaxLifetime`.
    - [MODIFY] Thay `DB` field trong `AppConfig` bằng `DBPool DBPoolConfig` và map tag `mapstructure:"dbPool"`.
    - [MODIFY] Chuyển `ReadReplicaDSN` lên cấp cao nhất của `AppConfig`.
    - [DELETE] Xóa các method thừa (`DSN()` và `PgxDSN()`).
    - [MODIFY] Dọn dẹp logic ở `validateConfig()`, `applyDBFallbacks()`, và `applyEnvOverrides()` (xóa bỏ đoạn code fallback/legacy liên quan tới `cfg.DB`).
3. **`pkgs/database/postgres.go`**:
    - [MODIFY] Sửa `NewPostgresConnection` để gọi `cfg.SystemDBURL()` thay vì `cfg.DB.DSN()`.
    - [MODIFY] Trỏ các config pool (MaxOpenConn, MaxIdleConn) về `cfg.DBPool`.
    - [MODIFY] Sửa `NewPostgresReadReplica` dùng biến `cfg.ReadReplicaDSN`.
4. **`internal/server/worker_server.go`**:
    - [MODIFY] Sửa thành `cfg.ReadReplicaDSN` thay cho `cfg.DB.ReadReplicaDSN`.

### Kế hoạch Kiểm thử (Verification Plan)
- Chạy `go build ./...` để đảm bảo code dịch không lỗi.
- Khởi động service thông qua `make run` (hoặc start up tương đương) để kiểm chứng ứng dụng đọc `dbPool` và thiết lập kết nối GORM thành công.
- Check log lúc khởi động không có ERROR hoặc cảnh báo liên quan.
