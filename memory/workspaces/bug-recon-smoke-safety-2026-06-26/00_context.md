# Context: Recon Smoke Safety Hardening

## 1. Background
Hệ thống reconciliation (recon) thực hiện kiểm tra khói (smoke check) giữa các nguồn dữ liệu khác nhau. File `internal/service/recon/recon_smoke.go` thực thi các goroutine chạy song song để so sánh Segment A và Segment B.
Hiện tại, các goroutine này thiếu cơ chế bắt panic (`recover()`), và việc tranh chấp tài nguyên (semaphore `globalSem` và `connSem`) chưa an toàn nếu context kết thúc trước.
Ngoài ra, hàm `RunTotalOnlyB` cần thiết lập timeout `fastCtx` giống như Segment A để tránh treo query và rò rỉ tài nguyên DB.

## 2. Requirements
1. **Nâng cấp goroutine chạy Segment A và Segment B trong `CheckAllUnified`**:
   - Thêm block `recover()` để bắt panic, log lại qua `rc.logger.Error`.
   - Sử dụng `select-case` để acquire `globalSem` và `connSem` an toàn, kiểm tra `ctx.Done()` để thoát ngay lập tức nếu ctx bị hủy/timeout.
2. **Nâng cấp `RunTotalOnlyB`**:
   - Thiết lập `fastCtx` với timeout sử dụng `rc.cfg.SmokeQueryTimeout` (mặc định 15s nếu bằng 0).
   - Truyền `fastCtx` vào tất cả các câu lệnh DB (`MaxWindowTs`, `getOrQuery`) và `repo.CreateSmokeResult` của Segment B.
3. **Biên dịch & Kiểm tra tĩnh**:
   - `go build ./cmd/... ./internal/...`
   - `go vet ./...`

## 3. Scope & Constraints
- Sửa đổi trực tiếp trong `internal/service/recon/recon_smoke.go`.
- Không thay đổi hành vi nghiệp vụ ngoài việc tăng tính an toàn và quản lý tài nguyên (panic recovery, context cancellation, timeout).
