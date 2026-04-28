# Phase 24 Plan - Registry Mutation Facade

1. Audit semantics của `Register`, `Update`, `BulkRegister` trong `RegistryHandler`.
2. Xác nhận đây là compatibility mutations thật, chưa nên “V2 hóa logic” ở backend.
3. Thêm facade V2-facing:
   - `POST /api/v1/source-objects/register`
   - `PATCH /api/v1/source-objects/registry/:id`
   - `POST /api/v1/source-objects/register-batch`
4. Chuyển `TableRegistry.tsx` sang 3 endpoint mới.
5. Verify backend tests, frontend build, grep call-sites và trạng thái swagger.
