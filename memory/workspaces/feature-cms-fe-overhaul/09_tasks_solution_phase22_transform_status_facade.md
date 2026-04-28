# Solution — Phase 22 Transform Status Facade

## Vấn đề

Sau Phase 21, FE runtime gần như đã bỏ được `/api/registry` cho các action/operator surface chính, nhưng `transform-status` vẫn còn trỏ trực tiếp vào namespace cũ.

## Giải pháp

- Thêm facade:
  - `GET /api/v1/source-objects/registry/{id}/transform-status`
- `TableRegistry` chuyển sang endpoint này
- giữ nguyên các mutation compatibility shell chưa nên đổi

## Kết quả

- FE-facing status surface nhất quán hơn với namespace `source-objects`
- giảm thêm dependency trực tiếp vào `/api/registry`
- không over-engineer phần mutation path
