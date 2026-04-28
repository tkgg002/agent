# Phase 23 Plan - Dashboard + ActivityManager V2 Reads

1. Audit call-sites FE còn đọc trực tiếp `/api/registry`.
2. Audit `registry/stats` để xác định có thể thay bằng read-model V2 hay không.
3. Thêm `GET /api/v1/source-objects/stats` từ `cdc_system.source_object_registry` + `shadow_binding` + bridge fields cần thiết.
4. Chuyển `Dashboard` sang endpoint stats mới và sửa copy cho đúng semantics `Source Objects`.
5. Chuyển `ActivityManager` sang `GET /api/v1/source-objects`.
6. Giữ nguyên mutation compatibility shell ở `PATCH /api/registry/:id` và `POST /api/registry/batch`.
7. Verify bằng backend tests, frontend build và note trạng thái swagger.
