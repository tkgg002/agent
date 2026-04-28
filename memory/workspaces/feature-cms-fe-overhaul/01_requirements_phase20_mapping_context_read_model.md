# Requirements — Phase 20 Mapping Context Read Model

- Mục tiêu:
  - cắt dependency của `MappingFieldsPage` vào `GET /api/registry`
  - giữ nguyên operator actions còn sống thật
  - không làm FE phải fetch toàn bộ registry rồi tự `find()`
- Audit kết luận:
  - page mappings hiện chỉ cần một context duy nhất theo `registry_id`
  - create-default-columns vẫn đang đi qua legacy path `/api/registry/:id/create-default-columns`
  - list/create/reload/backfill mapping rules đã đi theo V2 metadata rồi
- Requirement kỹ thuật:
  - thêm detail API bridge-aware:
    - `GET /api/v1/source-objects/registry/{registry_id}`
  - response phải đủ cho page mappings:
    - source db/type/table
    - shadow schema/table/FQN
    - sync/ddl/recon status
    - giữ `registry_id` để operator actions legacy còn chạy được
  - cập nhật swagger annotations cùng phase
