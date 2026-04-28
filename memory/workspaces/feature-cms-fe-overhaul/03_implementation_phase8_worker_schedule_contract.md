# Implementation — Phase 8 Worker Schedule Contract

- Refactor `schedule_handler.go`:
  - thêm DTO `WorkerScheduleScope`, `WorkerScheduleResponse`, `WorkerScheduleCreateRequest`, `WorkerScheduleUpdateRequest`
  - enrich list response bằng join `cdc_system.shadow_binding` + `cdc_system.source_object_registry`
  - thêm resolver cho create path để hỗ trợ payload scope giàu hơn
  - update path trả row mới nhất thay vì message chung chung
  - thêm swagger/comment cho list/create/update
- Refactor `ActivityManager.tsx`:
  - đọc `scope` trực tiếp từ `/api/worker-schedule`
  - create modal gửi thêm metadata scope khi có
  - hiển thị warning nếu scope ambiguous
- Workspace context được cập nhật để chốt mô hình 2 luồng:
  - auto-flow
  - cms-fe operator-flow
