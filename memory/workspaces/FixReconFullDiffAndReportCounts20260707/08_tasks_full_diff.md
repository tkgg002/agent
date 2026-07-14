# Tasks: Fix Recon Full Diff and Report Counts

- [ ] Tạo SQL migration `090_recon_smoke_diff_time.sql` tại `cdc-cms-service`.
- [ ] Cập nhật Go Struct `SmokeResult` trong `centralized-data-service` và `cdc-cms-service` để khai báo trường `DiffTime`.
- [ ] Dọn dẹp `internal/service/recon/recon_models.go` loại bỏ `manualLookbackKey` và `coldLookbackKey`.
- [ ] Cập nhật `internal/service/recon/recon_tier_a.go` để tính toán và gán đầy đủ các trường đếm (`SourceCount`, `DestCount`, `Diff`, `TotalSourceCount`, `TotalDestCount`) trong `RunHashWindowCheck`.
- [ ] Cập nhật `internal/service/recon/recon_smoke.go` xây dựng `runLookbackCheckA` / `runLookbackCheckB` và điền `DiffTime` khi Smoke check phát hiện drift.
- [ ] Dọn dẹp `internal/handler/recon/recon_base_handler.go` và cập nhật `recon_check_handler.go` loại bỏ `TypeReconFullDiff` và `LookbackHot`/`LookbackCold`, chuyển case `TypeReconHashWindow` chạy trực tiếp Hash Window check sử dụng context range.
- [ ] Cập nhật `internal/handler/recon/recon_check_heal_handler.go` để hợp nhất và dọn dẹp logic propose heal segment A dùng trực tiếp time range.
- [ ] Cập nhật SQL union trong `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go` để select `diff_time AS stale_ids`.
- [ ] Chạy các bộ unit tests để xác minh tính ổn định:
  - `go test -v ./internal/handler/recon/...`
  - `go test -v ./internal/service/recon/...`
  - `go test ./test/...` ở cdc-cms-service.
- [ ] Đồng bộ hóa walkthrough báo cáo kết quả.
