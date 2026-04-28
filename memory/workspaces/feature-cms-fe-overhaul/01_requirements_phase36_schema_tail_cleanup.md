# Requirements — Phase 36 Schema Tail Cleanup

## Mục tiêu

- Audit và xử lý nhóm “đuôi” schema assumptions còn lại sau Phase 35.
- Tập trung vào:
  - `event_bridge`
  - `transform_service`
  - `cms-service/internal/repository/registry_repo`

## Ràng buộc

- Không mở rộng scope sang refactor toàn bộ compatibility shell.
- Phân biệt rõ:
  - path có caller thật
  - path dormant/ít dùng chỉ cần hardening nhẹ
- Không đổi contract external API.

## Definition of Done

- `event_bridge` poll path không còn assume `public`.
- `transform_service` có đường resolve schema nếu được dùng lại.
- `registry_repo` có helper schema-aware wrappers cho compatibility shell.
- Verify pass ở:
  - `centralized-data-service`
  - `cdc-cms-service`
