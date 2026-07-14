# Kế hoạch Audit và Cập nhật Test Cases cho Hệ thống Reconciliation

Kế hoạch này tập trung vào việc sửa đổi các test case bị lỗi biên dịch trong gói `internal/handler/recon`, di chuyển các file test của `scan` về đúng gói của nó, thực thi kiểm thử hồi quy (regression smoke test), và dọn dẹp thư mục legacy `recon_bk`.

## User Review Required

> [!IMPORTANT]
> - Di chuyển `scan_array_path_test.go` và `scan_handler_test.go` từ `internal/handler/recon/` sang `internal/handler/scan/` để chúng có thể kiểm thử trực tiếp các hàm private (`explodePathToPGPath`, `validScanIdent`) của gói `scan`.
> - Cập nhật `recon_heal_v4_test.go` để chuyển đổi sang khởi tạo và gọi `HealHandler` mới thay vì `ReconHandler` cũ.

## Proposed Changes

---

### [Component: central-data-service - Tests]

#### [MODIFY] [recon_heal_v4_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4_test.go)
- Thay đổi `ReconHandler` thành `HealHandler`.
- Thay đổi constructor `NewReconHandler(...)` thành `NewHealHandler(base, reportRepo)`.
- Thay thế `handler.WithBackfill(...)` thành `handler.WithNatsPublisher(nc)`.

#### [NEW] [scan_array_path_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/scan/scan_array_path_test.go)
- Di chuyển file từ `internal/handler/recon/scan_array_path_test.go` sang `internal/handler/scan/scan_array_path_test.go`.
- Cập nhật khai báo `package scan`.

#### [NEW] [scan_handler_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/scan/scan_handler_test.go)
- Di chuyển file từ `internal/handler/recon/scan_handler_test.go` sang `internal/handler/scan/scan_handler_test.go`.
- Cập nhật khai báo `package scan`.

#### [DELETE] [scan_array_path_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_array_path_test.go)
- Xoá file ở vị trí cũ để tránh trùng lặp.

#### [DELETE] [scan_handler_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler_test.go)
- Xoá file ở vị trí cũ để tránh trùng lặp.

#### [DELETE] [recon_bk](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon_bk)
- Xoá hoàn toàn thư mục legacy `recon_bk` cùng các tệp tin con sau khi test cases được chứng minh là thành công và hệ thống chạy ổn định.

---

## Verification Plan

### Automated Tests
1. **Biên dịch và chạy thử nghiệm cho gói `scan`:**
   ```bash
   go test -v ./internal/handler/scan/...
   ```
2. **Biên dịch và chạy thử nghiệm cho gói `recon`:**
   ```bash
   go test -v ./internal/handler/recon/...
   ```
3. **Chạy regression smoke test của hệ thống đối soát:**
   ```bash
   go test -v ./internal/service/recon/...
   ```
