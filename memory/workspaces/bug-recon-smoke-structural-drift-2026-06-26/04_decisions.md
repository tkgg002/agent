# Architectural Decisions: Khắc phục lệch kiến trúc Recon Smoke

## ADR-01: Áp dụng Dependency Injection cho ReconSmokeRepo vào ReconCore

### Bối cảnh (Context)
- Trước đây, `ReconSmokeRepo` được khởi tạo cục bộ bên trong service `recon_smoke.go` bằng cách gọi `repo := reporecon.NewReconSmokeRepo(rc.db)` tại mỗi hàm nghiệp vụ (`RunTotalOnlyA`, `RunTotalOnlyB`, `CheckAllUnified`, v.v.).
- Điều này tạo ra sự kết hợp chặt chẽ (tight coupling) giữa service layer và repository layer, gây khó khăn cho việc viết unit test/mocking DB connection và vi phạm nguyên lý Clean Architecture / Dependency Injection.

### Quyết định (Decision)
- Chuyển `ReconSmokeRepo` thành một trường trong struct `ReconCore`: `smokeRepo *reporecon.ReconSmokeRepo`.
- Truyền `smokeRepo` thông qua constructor `NewReconCoreWithConfig` tại thời điểm khởi tạo server (`server_setup.go`, `worker_server_init.go`, `server_setup.go` backup 2).
- Loại bỏ toàn bộ các khởi tạo cục bộ bên trong service `recon_smoke.go` và chuyển sang sử dụng `rc.smokeRepo`.

### Hậu quả (Consequences)
- **Tích cực**:
  - Tách biệt hoàn toàn (decoupling) việc khởi tạo repository khỏi nghiệp vụ logic của service.
  - Hỗ trợ tốt hơn cho việc testing (có thể mock repository dễ dàng).
  - Giảm thiểu việc khởi tạo đi khởi tạo lại các Repo instance không cần thiết.
- **Tiêu cực**:
  - Cần phải chỉnh sửa chữ ký hàm khởi tạo `NewReconCoreWithConfig` dẫn đến phải cập nhật tất cả các callsites (wiring setup) trong dự án. Tuy nhiên, thay đổi này đã được thực hiện đồng bộ.
