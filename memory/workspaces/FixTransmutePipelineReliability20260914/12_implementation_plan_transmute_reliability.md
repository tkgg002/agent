# Kế Hoạch Triển Khai Kỹ Thuật (Implementation Plan)

## 1. Thông Tin Chung
- **Dự Án**: `centralized-data-service`
- **Mục Tiêu**: Giải quyết tận gốc các nguyên nhân gây rớt transmute trigger và timeout recon.
- **Vai Trò Phụ Trách**: Brain (Lập kế hoạch & Review) -> Chờ User Approve -> Muscle (Thực thi code).

## 2. Thứ Tự Triển Khai (Step-by-Step Execution)

### Bước 1: Sửa file `internal/handler/master/transmute_handler.go`
- Tăng timeout context tra cứu binding: 5s -> 10s (Dòng 102).
- Thêm metric `TransmuteErrorTotal.WithLabelValues("__shadow_lookup_" + req.ShadowTable).Inc()` khi lookup fail (Dòng 116-130).
- Thêm log debug khi không có active master binding (Dòng 133-136).
- Thêm error handling khi publish NATS msg `cdc.cmd.transmute` (Dòng 149-152).

### Bước 2: Sửa file `internal/handler/master/debounce.go`
- Import package `centralized-data-service/pkgs/metrics`.
- Thêm log cảnh báo và metric `__backpressure_<master>` khi hàng đợi vượt ngưỡng quá tải `maxSize * 2` (Dòng 69-78).
- Dùng `select ... default` bảo vệ channel `flushCh` chống kẹt goroutine và log lỗi drop mẻ (Dòng 125).

### Bước 3: Sửa file `internal/service/master/transmute_scheduler.go`
- Bổ sung ticker chạy ngầm định kỳ 10 phút/lần trong vòng lặp `Start()` (Dòng 73-89).
- Bổ sung hàm `cleanupLongRunningSchedules(ctx context.Context)` để reset schedule bị kẹt > 30 phút.

### Bước 4: Sửa file `internal/service/recon/recon_smoke.go`
- Đổi gọi `CountRows` sang `EstimatedCountRows` tại dòng 112 để triệt tiêu full table scan 30s.

### Bước 5: Kiểm Tra Biên Dịch & Test
- Chạy: `go build ./internal/...`
- Chạy: `go test ./internal/handler/master/... -v`
- Chạy: `go test ./internal/service/master/... -v`

### Bước 6: Tài Liệu Hóa & Kiểm Tra Governance
- Cập nhật nhật ký `05_progress.md`.
- Ghi nhận `11_report_transmute_reliability.md` sau khi hoàn thành.
- Rà soát các bài học trong `lessons.md`.
