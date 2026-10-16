# Workspace Context: fix-scan-array-nats-timeout-trans-his

## Mục tiêu
Khắc phục lỗi 503 Service Unavailable / NATS Timeout (`Worker timeout or request error: nats: timeout`) khi người dùng bấm scan array cho bảng `trans_his` từ CDC CMS Web.

## Phạm vi (Scope)
1. `cdc-cms-service`:
   - `internal/api/system/introspection_handler.go`: Endpoint `ScanArray` và `ScanArrayAll`.
2. `centralized-data-service`:
   - `internal/service/source/scan_service.go`: Hàm `ScanArrayFields`.
   - `internal/handler/scan/scan_handler.go`: NATS handler `HandleScanArrayFields`.
3. Hạ tầng vận hành:
   - Worker service (`centralized-data-service`) trên k8s / docker môi trường Goopay.
