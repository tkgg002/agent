# Progress: bug-metadata-cascade-masking-scan-fix-2026-06-23

## Audit & Governance
- **Root Cause Analysis (RCA)**:
  1. **Cascade toggle active**: Trong `UpdateMetadata` tại `source_repo_gorm.go`, logic tự động update `is_active` cho `shadow_binding` dẫn đến khi toggle source registry hoạt động, shadow binding bị toggle active theo ngoài ý muốn.
  2. **Sai tên table Shadow**: Query trong `ListSnapshotProgress` join lateral `shadow_binding` bằng `source_object_id` và lấy record active mới nhất, dẫn đến khi hiển thị snapshot progress của các binding cũ/khác, nó hiển thị sai tên table shadow.
  3. **Masking không chạy**: Logic reload registry trong `metadata_registry_service.go` append rule mapping vào cache của binding mà không lọc theo `v2.ShadowBindingID == bindingID`, dẫn đến nạp chéo rule của clone/binding khác đè lên làm mất cấu hình mask.
  4. **Scan-fields polling không stop**: Clock skew/drift giữa Client FE và Database Docker khiến filter `started_at >= since` lọc mất log hoàn thành. Cần đồng bộ qua `server_time` từ response 202 của backend.
  5. **Transmute không chạy**: Query `ListMasterTablesByShadowIdentity` trong `master_binding_repo.go` bị dư 1 argument (truyền 8 arguments cho 7 placeholders), gây lỗi query và làm đứt gãy flow post-ingest transmute trigger.

## Progress Log
- [2026-06-23T03:45:00Z] [Antigravity] Khởi tạo workspace và phân tích context.
- [2026-06-23T03:47:00Z] [Antigravity] Nhận thêm feedback từ user về lỗi transmute không chạy, điều gia và xác định nguyên nhân lệch query parameters.
- [2026-06-23T04:04:00Z] [Antigravity:Gemini] Bắt đầu thực thi kế hoạch sửa đổi. Cập nhật checklist thực thi.
- [2026-06-23T04:06:00Z] [Antigravity:Gemini] Hoàn thành fix code cho các file: metadata_registry_service.go, source_object_actions_handler.go, registry_handler_tools_scan.go, useAsyncDispatch.ts. Xác minh master_binding_repo.go hoạt động chính xác và build thành công.

## Checklist thực thi của Muscle
- [x] Bổ sung `server_time` vào `source_object_actions_handler.go` và `registry_handler_tools_scan.go` (cdc-cms-service)
- [x] Cập nhật hook `useAsyncDispatch.ts` (cdc-cms-web)
- [x] Fix logic nạp cache rules trong `metadata_registry_service.go` (centralized-data-service)
- [x] Fix đối số dư thừa trong `master_binding_repo.go` (centralized-data-service)
- [x] Rebuild và chạy kiểm thử tự động
