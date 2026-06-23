# Plan: Wire DI & Initialize Sub-handlers in Worker Server

We have successfully moved all handlers and services into domain-specific sub-packages. The final step is to update the worker server's dependency injection (DI) wiring in `worker_server.go` and `worker_server_init.go`, wire NATS subscriptions to the new handler types, delete the legacy root handler files, and verify the build.

## User Review Required

> [!IMPORTANT]
> - Sửa đổi kiểu dữ liệu của các handler/service trong struct `WorkerServer` tại `internal/server/worker_server.go` sang sub-package tương ứng.
> - Sửa đổi logic khởi tạo tại `internal/server/worker_server_init.go` để import chính xác các package: `handlershadow`, `handlerorchestration`, `handlermaster`, `handlersource`, `handlerrecon`, và `handlercommon`.
> - Đăng ký các route NATS subscriber cho các sub-handler mới.
> - Xóa bỏ hoàn toàn các file phẳng cũ ở root `internal/handler/` (`command_handler.go`, `command_handler_ddl.go`, `command_handler_discover.go`, `command_handler_scan.go`, `command_handler_sync.go`, `command_handler_transform.go`).
> - GP-230: Không tự động commit từng bước để nguyên working tree để user có thể review diff tổng thể trên IDE.

## Proposed Changes

### [Worker Server DI Wiring]

#### [MODIFY] [worker_server.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/worker_server.go)
- Sửa đổi các trường handler trong struct `WorkerServer` trỏ tới kiểu dữ liệu sub-package mới thay cho package `handler` cũ.

#### [MODIFY] [worker_server_init.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/worker_server_init.go)
- Thêm các alias imports cho các sub-package:
  - `"centralized-data-service/internal/handler/shadow"` -> `handlershadow`
  - `"centralized-data-service/internal/handler/orchestration"` -> `handlerorchestration`
  - `"centralized-data-service/internal/handler/master"` -> `handlermaster`
  - `"centralized-data-service/internal/handler/source"` -> `handlersource`
  - `"centralized-data-service/internal/handler/recon"` -> `handlerrecon`
  - `"centralized-data-service/internal/handler/common"` -> `handlercommon`
- Thay thế khởi tạo `commandHandler` cũ bằng việc khởi tạo 6 sub-handlers riêng biệt.
- Wire các NATS subscriptions của client trỏ đến các method handler tương ứng trong sub-packages.
- Cập nhật hằng số `SubjectTransmuteCompleted` sang `handlermaster.SubjectTransmuteCompleted`.
- Cập nhật hằng số `SubjectProvisioningStepCompleted` sang `handlercommon.SubjectProvisioningStepCompleted`.

#### [DELETE] [command_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/command_handler.go)
#### [DELETE] [command_handler_ddl.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/command_handler_ddl.go)
#### [DELETE] [command_handler_discover.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/command_handler_discover.go)
#### [DELETE] [command_handler_scan.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/command_handler_scan.go)
#### [DELETE] [command_handler_sync.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/command_handler_sync.go)
#### [DELETE] [command_handler_transform.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/command_handler_transform.go)

## Verification Plan

### Automated Tests
- Chạy `go build ./...` trong `centralized-data-service` để kiểm tra biên dịch không có lỗi.
- Chạy `go test ./...` để đảm bảo tất cả tests hiện tại chạy pass.
