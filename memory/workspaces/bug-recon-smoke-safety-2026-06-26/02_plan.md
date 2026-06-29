# Implementation Plan: Recon Smoke Safety Hardening

## 1. Objectives
- Tăng tính an toàn cho luồng xử lý đồng thời (concurrency) trong `CheckAllUnified`.
- Ngăn chặn panic làm sập ứng dụng và quản lý tài nguyên (semaphore, connection pool) tốt hơn khi context bị hủy.
- Đồng bộ cơ chế timeout (SmokeQueryTimeout) cho Segment B trong hàm `RunTotalOnlyB`.
- Đảm bảo mã nguồn biên dịch thành công và không vi phạm các kiểm tra tĩnh của Go compiler.

## 2. Detailed Steps

### Phase 1: Investigation & Context Restoring
1. Đọc nội dung file `internal/service/recon/recon_smoke.go` để hiểu cấu trúc hiện tại của `CheckAllUnified` và `RunTotalOnlyB`.
2. Xác định các điểm cần can thiệp:
   - Trong `CheckAllUnified`: Các goroutine gọi `RunTotalOnlyA` và `RunTotalOnlyB`.
   - Trong `RunTotalOnlyB`: Cách khởi tạo context và cách truyền context vào các hàm truy vấn DB.

### Phase 2: Implementation of Safety Upgrades
1. **Goroutine check trong `CheckAllUnified`**:
   - Thêm `defer recover()` bảo vệ bên trong các goroutine chạy song song.
   - Khi phát hiện panic, log lại qua `rc.logger.Error`.
   - Bọc việc acquire semaphore (`globalSem.Acquire` và `connSem.Acquire`) bằng `select` kết hợp với `ctx.Done()`. Nếu context bị đóng trước khi có semaphore, thoát ngay lập tức.
2. **Timeout trong `RunTotalOnlyB`**:
   - Khởi tạo `fastCtx, cancel := context.WithTimeout(ctx, timeout)` với timeout lấy từ `rc.cfg.SmokeQueryTimeout` (nếu bằng 0 thì mặc định là 15s).
   - Truyền `fastCtx` thay cho `ctx` vào các lệnh DB như `MaxWindowTs`, `getOrQuery`, và `repo.CreateSmokeResult`.
   - Gọi `defer cancel()`.

### Phase 3: Verification
1. Chạy biên dịch toàn bộ code: `go build ./cmd/... ./internal/...`
2. Chạy kiểm tra tĩnh: `go vet ./...`
3. Kiểm tra các unit test liên quan đến recon smoke (nếu có).

## 3. Definition of Done (DoD)
- [ ] File `internal/service/recon/recon_smoke.go` đã được cập nhật thành công.
- [ ] Các goroutine trong `CheckAllUnified` có `recover()` để bắt panic.
- [ ] Việc tranh chấp semaphore sử dụng `select-case` với `ctx.Done()`.
- [ ] `RunTotalOnlyB` sử dụng `fastCtx` với timeout (SmokeQueryTimeout) cho tất cả các thao tác DB và lưu kết quả.
- [ ] Lệnh `go build` và `go vet` hoàn thành không có lỗi.
