# Plan: Refactor EmitStepCompleted to BaseHandler Method

## Proposed Steps

1. **Modify `internal/handler/base/provisioning_emit.go`**:
   - Khai báo struct `StepResult`.
   - Thay đổi signature của `EmitStepCompleted` thành phương thức của `*BaseHandler`: `func (h *BaseHandler) EmitStepCompleted(ctx context.Context, result StepResult)`.
   - Cập nhật logic bên trong phương thức để sử dụng các trường `h.NatsConn` và `h.Logger` đã được inject vào `BaseHandler`.

2. **Update Callers**:
   - Sửa đổi lệnh gọi trong `internal/handler/orchestration/discover_handler.go`. Vì `DiscoverHandler` embed `BaseHandler`, ta có thể gọi `h.EmitStepCompleted(ctx, base.StepResult{...})`.
   - Sửa đổi lệnh gọi trong `internal/handler/master/master_ddl_handler.go`. Do `MasterDDLHandler` không embed `BaseHandler`, ta khởi tạo `base.BaseHandler` tạm thời để gọi method này.

3. **Verification**:
   - Biên dịch dự án và chạy các unit test để kiểm tra tính đúng đắn.
