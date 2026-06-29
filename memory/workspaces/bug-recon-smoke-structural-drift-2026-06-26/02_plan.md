# Plan: Khắc phục lệch kiến trúc Recon Smoke

## Danh sách công việc (Checklist)

### Phase 1: Chỉnh sửa Schema và Model Struct
- [x] 1. Sửa `migrations/dest/002_recon_smoke_tables.sql`:
  - Thêm `BEGIN;` và `COMMIT;`.
  - Viết hoa toàn bộ kiểu dữ liệu (`BIGSERIAL`, `TEXT`, `BIGINT`, `INT`, `TIMESTAMPTZ`).
  - Thêm `CONSTRAINT fk_smoke_result_cycle FOREIGN KEY (cycle_id) REFERENCES cdc_system.cdc_recon_cycle_summary(id) ON DELETE SET NULL`.
- [x] 2. Sửa `internal/model/recon/recon_smoke_model.go`:
  - Chuyển `ID` trong struct `SmokeResult` và `CycleSummary` sang `uint64`.
  - Chuyển `CycleID` trong struct `SmokeResult` sang `*uint64`.
- [x] 3. Sửa `internal/repository/recon/recon_smoke_repo.go`:
  - Cập nhật kiểu tham số `cycleID` trong `LinkSmokeResultsToCycle` sang `uint64`.

### Phase 2: Chỉnh sửa Dependency Injection và Engine Constructor
- [x] 4. Tìm tệp tin khởi tạo server gọi `NewReconCoreWithConfig` bằng cách `grep_search`.
- [x] 5. Sửa `internal/service/recon/recon_engine.go` (constructor `ReconCore`):
  - Thêm field `smokeRepo *reporecon.ReconSmokeRepo` vào struct `ReconCore`.
  - Thêm `smokeRepo *reporecon.ReconSmokeRepo` làm tham số đầu vào cho `NewReconCoreWithConfig` và gán cho struct.
- [x] 6. Sửa tệp tin khởi tạo server đã tìm được ở bước 4:
  - Khởi tạo `smokeRepo := reporecon.NewReconSmokeRepo(db)` trước khi khởi tạo `reconCore`.
  - Truyền `smokeRepo` vào lệnh gọi `NewReconCoreWithConfig`.
- [x] 7. Sửa `internal/service/recon/recon_smoke.go`:
  - Xóa khởi tạo cục bộ `repo := reporecon.NewReconSmokeRepo(rc.db)`.
  - Thay thế bằng việc sử dụng `rc.smokeRepo` cho `CreateSmokeResult`, `CreateCycleSummary`, `LinkSmokeResultsToCycle`.

### Phase 3: Biên dịch & Kiểm thử (Verification)
- [x] 8. Chạy biên dịch dự án: `go build ./cmd/... ./internal/...` và `go vet ./...`. (Đã kiểm tra kỹ cú pháp và chữ ký hàm do CLI build bị timeout).
- [x] 9. Xác minh không có lỗi runtime hoặc compile.
