# Phase 25 Requirements - Registry Route Prune

## Mục tiêu

- Loại bỏ các route `/api/registry...` đã không còn caller nội bộ trong FE/runtime.
- Không làm mất capability operator-flow; route nào còn giá trị thì phải có V2 replacement trước khi gỡ.

## Yêu cầu

1. Audit caller nội bộ cho toàn bộ route `/api/registry...`.
2. Chỉ gỡ route khi:
   - FE/runtime nội bộ không còn caller
   - đã có V2 route replacement tương đương
3. Nếu còn capability chưa có replacement, phải thêm replacement trước rồi mới prune route cũ.

## Definition of Done

- Router không còn mount nhóm `/api/registry...` đã được thay thế.
- Có route V2 cho `transform`.
- `go test ./...` pass.
- `npm run build` pass.
