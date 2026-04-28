# Solution — Phase 19 Shadow Bindings Dual View

## Vấn đề

Sau Phase 18, `TableRegistry` đã đọc V2 source objects thật, nhưng operator vẫn chưa có chỗ nhìn trực tiếp binding layer của shadow. Điều này làm `shadow_binding` vẫn là khái niệm backend-only, trong khi thực tế nó là mắt xích quan trọng của operator-flow.

## Giải pháp

- Thêm `GET /api/v1/shadow-bindings`
- Không mở page mới
- Nâng `TableRegistry` thành dual-view:
  - `Source Objects`
  - `Shadow Bindings`

## Kết quả

- FE gọn hơn vì không nở thêm menu/page
- Operator nhìn thấy trực tiếp:
  - source object nào đang route vào shadow nào
  - DDL đã created hay failed
  - drift gần nhất là bao nhiêu
- `/api/registry` tiếp tục bị thu hẹp vai trò dần về compatibility shell
