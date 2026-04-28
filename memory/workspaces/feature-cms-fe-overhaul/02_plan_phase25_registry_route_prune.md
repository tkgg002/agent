# Phase 25 Plan - Registry Route Prune

1. Audit route `/api/registry...` còn mount trong router.
2. Audit FE/runtime caller thực tế.
3. Thêm `POST /api/v1/source-objects/registry/:id/transform` nếu chưa có.
4. Gỡ các route `/api/registry...` đã được thay thế khỏi router.
5. Verify tests, build, grep route surface, swagger note.
