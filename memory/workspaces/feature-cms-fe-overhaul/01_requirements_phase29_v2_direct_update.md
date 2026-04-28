# Phase 29 Requirements - V2 Direct Update

## Mục tiêu

- Cho row `V2-only` update được trực tiếp theo `source_object_id`.
- Không ép mọi row phải có `registry_id`.

## Yêu cầu

1. Có mutation path `PATCH /api/v1/source-objects/:id`.
2. Chỉ support những field đã có home rõ ràng ở V2:
   - `is_active`
   - `timestamp_field`
   - `notes`
3. `priority` và `sync_interval` chưa có V2 home thì không được giả vờ update.
4. FE phải route đúng:
   - row bridged -> bridge patch
   - row V2-only -> V2 direct patch

## Definition of Done

- Backend route/direct update hoạt động.
- `TableRegistry` không chặn `is_active` cho row `V2-only`.
- `go test ./...` pass.
- `npm run build` pass.
