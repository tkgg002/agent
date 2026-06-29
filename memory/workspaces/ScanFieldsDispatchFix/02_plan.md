# Plan: Scan Fields Dispatch Status Fix

## Objective
Sửa lỗi polling vô tận trên FE đối với action `scan-fields`. Đảm bảo worker ghi nhận trạng thái `success`/`error` khớp với `operation = "scan-fields"` vào bảng `cdc_system.cdc_activity_log`.

## Proposed Changes
Chúng ta sẽ sửa đổi file `internal/handler/source/discover_handler.go` tại hàm `HandleScanFields`.
Không dùng tool sửa trực tiếp, thay vào đó tạo file script python để vá file từ `scratch/`.

### Modification details for `discover_handler.go`
1. Import package:
   `"centralized-data-service/internal/service/governance"`
2. Sửa trong `HandleScanFields`:
   - Khởi tạo `activityLogger := governance.NewActivityLogger(h.DB, h.Logger)`
   - Khi chạy thành công: Gọi `activityLogger.Quick` để ghi record `"scan-fields"` với status `"success"`.
   - Khi chạy thất bại: Gọi `activityLogger.Quick` để ghi record `"scan-fields"` với status `"error"`.

## Verification Plan
1. **Compile Verification**: Chạy `make build` hoặc `go build` cho worker.
2. **Behavior Verification**:
   - Trigger `scan-fields` qua REST API hoặc dispatch NATS.
   - Kiểm tra xem trong bảng `cdc_system.cdc_activity_log` có xuất hiện record:
     - `operation = "scan-fields"`, `status = "success"` (hoặc `error` nếu lỗi).
   - Kiểm tra API `dispatch-status` trả về đúng status `success` giúp FE dừng polling.
