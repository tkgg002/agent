# Context: Scan Fields Dispatch Status Fix

## Problem
Khi thực hiện gọi API `scan-fields` cho source object:
1. FE gửi `POST /api/v1/source-objects/:id/scan-fields?binding_id=25` và nhận được phản hồi.
2. FE tiến hành poll status qua API `GET /api/v1/source-objects/:id/dispatch-status?subject=scan-fields`.
3. Tuy nhiên, FE không bao giờ dừng polling mặc dù lệnh scan fields đã chạy thành công trên worker.

## Root Cause
- Khi nhận request `scan-fields`, API service (`cdc-cms-service`) ghi nhận một dòng `accepted` vào `cdc_activity_log` với `operation = "scan-fields"`.
- Khi worker (`centralized-data-service`) xử lý xong command `scan-fields` trong `discover_handler.go`, nó tự build map response JSON và gọi `h.NatsPublish` trực tiếp thay vì `h.PublishResult`. Do đó, callback `OnPublishResult` (vốn dùng để ghi nhận status `success`/`error` của command vào `cdc_activity_log`) **không được gọi**.
- Kết quả là `cdc_activity_log` chỉ giữ trạng thái `accepted` của `scan-fields`, khiến FE tiếp tục poll vô tận.

## Strategy
- Sửa đổi worker `discover_handler.go` tại `HandleScanFields` để ghi nhận trực tiếp kết quả chạy (`success` hoặc `error`) vào `cdc_activity_log` với tên `operation = "scan-fields"`.
- Việc ghi log này được thực hiện thông qua `governance.NewActivityLogger`.
