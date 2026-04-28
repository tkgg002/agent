# Phase 25 Solution - Registry Route Prune

## Quyết định chính

- Không cần giữ route `/api/registry...` trong router nếu FE/runtime nội bộ đã bỏ hoàn toàn và đã có replacement V2.
- Giữ handler delegate bên dưới để tránh làm refactor phase này lan sang logic xử lý sâu hơn.

## Kết quả mong muốn

- API surface gọn hơn.
- Không còn “route thừa chỉ để hoài niệm” trong router.

## Kết quả thực tế

- Đã thêm `POST /api/v1/source-objects/registry/:id/transform`.
- Router không còn mount nhóm `/registry...` legacy đã có replacement.
- Tests và build đều pass.
