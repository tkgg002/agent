# Requirements — Phase 22 Transform Status Facade

- Mục tiêu:
  - dọn nốt phần status surface thực chiến còn sót dưới `/api/registry`
  - tiếp tục giảm dependency trực tiếp của FE vào compatibility shell
- Audit kết luận:
  - sau Phase 21, phần còn nổi bật nhất ở FE runtime là `transform-status`
  - endpoint này vẫn còn giá trị thực chiến thật cho operator
  - mutation compatibility shell như `PATCH /api/registry/:id` và `POST /api/registry/batch` chưa nên bọc giả ở giai đoạn này
- Requirement kỹ thuật:
  - thêm facade `GET /api/v1/source-objects/registry/{id}/transform-status`
  - FE `TableRegistry` chuyển sang endpoint mới
  - cập nhật swagger annotations cùng phase
