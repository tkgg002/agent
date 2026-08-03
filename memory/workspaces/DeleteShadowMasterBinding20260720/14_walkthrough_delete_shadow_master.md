# Kết quả Triển khai & Walkthrough: Xoá Shadow & Xoá Master

## 1. Các file đã thay đổi

### Backend (`cdc-cms-service`):
- [repository.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/ports/repository.go): Thêm struct `ShadowBindingInfo` và 3 method cho interface `ShadowBindingRepo`.
- [shadow_binding_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/shadow/shadow_binding_repo_gorm.go): Triển khai các method GORM `GetByID`, `ListMasterBindingIDByShadowID`, `DeleteShadowBinding`.
- [delete_shadow_binding.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/shadow/delete_shadow_binding.go): Định nghĩa command, command validation, và handler cho cascade delete shadow.
- [delete_master_binding.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/delete_master_binding.go): Định nghĩa command và handler xoá master binding.
- [master_registry_handler_delete.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_registry_handler_delete.go): API endpoint handler xử lý yêu cầu xoá Master.
- [shadow_binding_actions_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/shadow/shadow_binding_actions_handler.go): Bổ sung API handler xoá Shadow.
- [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go): Khai báo 2 router DELETE đi qua destructive middleware chain.
- [server.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go): Đăng ký command handlers đồng bộ.

### Frontend (`cdc-cms-web`):
- [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx):
  - Tích hợp nút Xoá và ConfirmDestructiveModal.
  - Gọi API `DELETE /api/v1/shadow-bindings/:id` gửi lý do và idempotency key.
- [MasterRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx):
  - Tích hợp nút Xoá và ConfirmDestructiveModal.
  - Gọi API `DELETE /api/v1/masters/:id` gửi lý do và idempotency key.

## 2. Kết quả kiểm tra build
- Backend build thành công: `go build ./internal/...` không phát sinh lỗi.
- Frontend build thành công: `npm run build` tạo bundle production hoàn chỉnh.
