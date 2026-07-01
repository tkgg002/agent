# Plan: Sửa lỗi healSegmentA/healSegmentB lặp lại do lấy stale report

## 1. Nghiên cứu & Thiết kế (Research & Design)
- **Hành động 1.1**: Đọc `internal/handler/recon/recon_heal_v4.go` để xác định logic check `HealedAt` của cả `healSegmentA` và `healSegmentB`.
- **Hành động 1.2**: Thiết kế ngưỡng thời gian hết hạn (ví dụ: `healReportMaxAge = 5 * time.Minute`).
- **Hành động 1.3**: Bổ sung điều kiện kiểm tra `time.Since(reportPtr.CheckedAt) > healReportMaxAge` để tự động kích hoạt tiến trình đối soát mới.

## 2. Thực thi & Sửa code (Execution)
- **Hành động 2.1**: Sửa code trong `internal/handler/recon/recon_heal_v4.go` cho cả `healSegmentA` và `healSegmentB`.
- **Hành động 2.2**: Cập nhật unit test trong `recon_heal_v4_test.go` nếu cần để kiểm nghiệm logic stale report.

## 3. Xác minh & Báo cáo (Verification)
- **Hành động 3.1**: Chạy `go build ./...` và `go test -v ./internal/handler/recon/...` để đảm bảo code compile và pass tất cả test cases.
- **Hành động 3.2**: Báo cáo kết quả và kết thúc workspace.
