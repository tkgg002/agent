# Requirements — Phase 19 Shadow Bindings Dual View

- Mục tiêu:
  - giảm thêm dependency của FE vào `/api/registry`
  - đưa `shadow_binding` thành concept operator thấy trực tiếp
  - không tạo thêm page mới nếu có thể dùng lại IA hiện có
- Audit kết luận:
  - chưa có API dedicated cho list `shadow_binding`
  - `TableRegistry` đã đọc V2 source objects, nhưng operator vẫn chưa có màn nhìn trực tiếp binding layer
  - `shadow_binding` hiện đã được dùng rải rác trong `masters`, `reconciliation`, `worker-schedule`, nhưng chưa có monitoring surface riêng
- Requirement kỹ thuật:
  - thêm `GET /api/v1/shadow-bindings`
  - enrich với source object metadata + latest recon info
  - FE nên hiển thị trong cùng màn `Source Objects` dưới dạng dual-view/tab, tránh phình thêm page
  - cập nhật swagger annotations cùng phase
