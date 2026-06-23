# Context: Refactor EmitStepCompleted to BaseHandler Method

## Goal
Chuyển đổi hàm helper `EmitStepCompleted` trong `internal/handler/base/provisioning_emit.go` thành một phương thức (method) thuộc struct `BaseHandler`. Thiết lập struct `StepResult` để đóng gói các tham số đầu vào (giải quyết code smell Long Parameter List).

## Active Files
- [provisioning_emit.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/base/provisioning_emit.go)
- [discover_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/discover_handler.go)
- [master_ddl_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/master_ddl_handler.go)

## Current Status
Hàm `EmitStepCompleted` hiện đang là standalone function nhận 9 tham số. Cần chuyển sang method của `BaseHandler` nhận `context.Context` và `StepResult`.
