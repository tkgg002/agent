# Requirements — Phase 21 Registry Bridge Action Facade

- Mục tiêu:
  - giảm dependency trực tiếp của FE vào namespace `/api/registry`
  - không phá các action operator thực chiến còn cần `registry_id`
  - tránh tạo logic giả rằng các action này đã hoàn toàn V2-native
- Audit kết luận:
  - `detect-timestamp-field`, `scan-fields`, `standardize`, `create-default-columns` hiện vẫn cần `registry_id`
  - worker payload và xử lý backend cho các action này còn neo vào bridge legacy
  - vấn đề cần xử không phải đổi semantics, mà là đổi FE-facing namespace cho đúng kiến trúc hiện tại
- Requirement kỹ thuật:
  - thêm facade endpoints dưới `/api/v1/source-objects/registry/:id/...`
  - FE chuyển sang gọi facade mới
  - backend vẫn được phép delegate vào `RegistryHandler`
  - cập nhật swagger annotations cùng phase
