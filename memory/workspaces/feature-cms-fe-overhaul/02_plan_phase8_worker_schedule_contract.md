# Plan — Phase 8 Worker Schedule Contract

1. Audit `schedule_handler.go`, `ActivityManager.tsx`, model `WorkerSchedule`, và metadata V2 (`source_object_registry`, `shadow_binding`).
2. Thiết kế response DTO mới cho `GET /api/worker-schedule`:
   - giữ field legacy
   - thêm scope source/shadow phục vụ operator-flow
3. Thiết kế create contract compatibility-first:
   - vẫn nhận `target_table`
   - hỗ trợ thêm `source_database`, `source_table`, `shadow_schema`, `shadow_table`
   - resolve về target table thực tế từ metadata V2
4. Refactor `ActivityManager` để ưu tiên dùng context từ API schedule thay vì tự suy diễn cho list view.
5. Cập nhật swagger/comment cho API bị thay đổi.
6. Verify bằng `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web`.
