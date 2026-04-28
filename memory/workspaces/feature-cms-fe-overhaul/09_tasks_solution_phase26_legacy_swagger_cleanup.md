# Phase 26 Solution - Legacy Swagger Cleanup

## Quyết định chính

- Router và annotations phải cùng nói một ngôn ngữ.
- Nếu route legacy đã bị gỡ khỏi router, comment swagger của nó cũng phải bị gỡ hoặc đổi nghĩa tương ứng.

## Kết quả thực tế

- `registry_handler.go` không còn source-level `@Router /api/registry...`.
- Các method legacy được mô tả lại như delegate nội bộ cho facade V2.
