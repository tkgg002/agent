# Danh sách Task Chi tiết (Recon Smoke Boundary Tasks)

- [x] Thiết lập phương pháp tính toán thời gian `from` và `now` có làm tròn giây lẻ về phút: `now.Add(-120 * time.Second).Truncate(time.Minute)`.
- [x] Cập nhật `RunTotalOnlyA` để thực hiện truy vấn `CountInWindow` trên cả MongoDB Source và Postgres Shadow trong khoảng `[from, now]`.
  - Tính toán số lượng sạch: `srcActiveClean = srcEst - srcRecent` và `dstActiveClean = dstActive - dstRecent`.
  - Thực hiện so sánh hiệu số `diff = srcActiveClean - dstActiveClean` và cập nhật metrics.
- [x] Cập nhật `RunTotalOnlyB` để thực hiện truy vấn `CountInWindow` trên cả Shadow và Master (Postgres) trong khoảng `[from, now]`.
  - Tính toán số lượng sạch: `shadowActiveClean = shadowActive - shadowRecent` và `masterActiveClean = masterActive - masterRecent`.
  - Thực hiện so sánh hiệu số `diff = shadowActiveClean - masterActiveClean` và cập nhật metrics.
- [x] Viết bổ sung Unit Test trong `recon_smoke_test.go` kiểm thử tính đúng đắn của logic trừ bù cửa sổ thời gian gần đây (mock/stub các query CountInWindow).
- [x] Chạy kiểm thử tự động (`go test -v ./internal/service/recon/...`) và kiểm tra linter.
- [x] Báo cáo kết quả (Walkthrough).
