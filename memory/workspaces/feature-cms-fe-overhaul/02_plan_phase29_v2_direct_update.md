# Phase 29 Plan - V2 Direct Update

1. Audit `update` call-site FE và `Update` mutation hiện tại.
2. Thêm `PATCH /api/v1/source-objects/:id`.
3. Implement direct V2 update cho field subset hợp lệ.
4. Chuyển `TableRegistry` sang chọn đúng endpoint theo trạng thái bridge.
5. Disable/update-copy cho `priority` khi row chưa có bridge.
6. Verify backend tests + frontend build.
