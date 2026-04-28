# Phase 29 Solution - V2 Direct Update

## Quyết định chính

- `V2-only` row không nên bị chặn hoàn toàn.
- Nhưng chỉ những field đã có V2 home rõ ràng mới được update trực tiếp.

## Kết quả thực tế

- đã có `PATCH /api/v1/source-objects/:id`
- `is_active` của row `V2-only` update trực tiếp được
- `priority` của row `V2-only` bị disable rõ ràng vì chưa có V2 home
