# Kế hoạch Triển khai (AI Implementation Plan) - Sửa đổi Đối soát & Báo cáo Số liệu

Tài liệu này chi tiết kế hoạch thực hiện của Muscle (Chief Engineer) để thay đổi code đối soát thời gian, thống nhất các option lookback, tính toán số lượng báo cáo, và cập nhật view db.

## 1. Danh sách các file cần chỉnh sửa & tạo mới

1. **[Tạo mới]** `cdc-cms-service/migrations/schema/recon_dlq/090_recon_smoke_diff_time.sql`
2. **[Chỉnh sửa]** `cdc-cms-service/internal/model/recon_smoke.go`
3. **[Chỉnh sửa]** `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
4. **[Chỉnh sửa]** `centralized-data-service/internal/model/recon/recon_smoke_model.go`
5. **[Chỉnh sửa]** `centralized-data-service/internal/service/recon/recon_models.go`
6. **[Chỉnh sửa]** `centralized-data-service/internal/service/recon/recon_engine.go`
7. **[Chỉnh sửa]** `centralized-data-service/internal/service/recon/recon_tier_a.go`
8. **[Chỉnh sửa]** `centralized-data-service/internal/service/recon/recon_smoke.go`
9. **[Chỉnh sửa]** `centralized-data-service/internal/handler/recon/recon_base_handler.go`
10. **[Chỉnh sửa]** `centralized-data-service/internal/handler/recon/recon_check_handler.go`
11. **[Chỉnh sửa]** `centralized-data-service/internal/handler/recon/recon_check_heal_handler.go`

## 2. Các bước triển khai chi tiết

### Bước 1: Khởi động & Tạo SQL Migration
- Tạo file SQL migration `090_recon_smoke_diff_time.sql` với câu lệnh `ALTER TABLE cdc_system.cdc_recon_smoke_result ADD COLUMN IF NOT EXISTS diff_time JSONB;`.
- Ghi log tiến độ vào `05_progress_full_diff.md`.

### Bước 2: Cập nhật Struct Model `SmokeResult`
- Cập nhật struct `SmokeResult` trong `cdc-cms-service/internal/model/recon_smoke.go` và `centralized-data-service/internal/model/recon/recon_smoke_model.go`.
- Thêm trường `DiffTime json.RawMessage` map với cột `diff_time`.

### Bước 3: Dọn dẹp & Thống nhất cấu hình thời gian tại `centralized-data-service/internal/service/recon/`
- **`recon_models.go`**: Xóa `manualLookbackKey` và `coldLookbackKey`. Đảm bảo chỉ giữ lại `WithReconTimeRange` và `GetReconTimeRange`.
- **`recon_engine.go`**: Đơn giản hóa hàm `effectiveLookback` chỉ trả về cấu hình mặc định (2 giờ) mà không còn phân biệt `hot` / `cold`.
- **`recon_tier_a.go`**: Trong `RunHashWindowCheck`, tính toán và trả về đầy đủ `SourceCount`, `DestCount`, `Diff`, `TotalSourceCount`, `TotalDestCount` cho report.
- **`recon_smoke.go`**: Thêm `runLookbackCheckA` và `runLookbackCheckB` để lấy các buckets bị lệch thời gian khi phát hiện có drift, sau đó gán vào trường `DiffTime` của `SmokeResult`.

### Bước 4: Sửa đổi tầng Route & Handler trong `centralized-data-service/internal/handler/recon/`
- **`recon_base_handler.go`**: Xóa các hằng số `TypeReconFullDiff`, `LookbackHot`, `LookbackCold`.
- **`recon_check_handler.go`**: Cập nhật validator và `executeGenericCheck` loại bỏ `TypeReconFullDiff`.
- **`recon_check_heal_handler.go`**: Hợp nhất logic propose heal, xóa `proposeFullDiffHealA` và `proposeWindowHealA`, thay bằng `proposeHealSegmentA` nhận range thời gian trực tiếp.

### Bước 5: Map hiển thị Read-side trong `cdc-cms-service`
- **`recon_read_repo_gorm.go`**: Cập nhật query UNION để lấy `diff_time` thay vì `NULL::jsonb` cho cột `stale_ids` của Smoke check.

### Bước 6: Chạy test & xác minh
- Chạy unit tests trong `centralized-data-service`:
  - `go test -v ./internal/handler/recon/...`
  - `go test -v ./internal/service/recon/...`
- Chạy unit tests trong `cdc-cms-service`:
  - `go test ./test/...`

## 3. Quản lý Rủi ro & Khôi phục (Restore-points)
- Sẽ thực hiện git commit cục bộ (Restore-point) trước khi sửa code và sau khi sửa xong để bảo đảm an toàn.
