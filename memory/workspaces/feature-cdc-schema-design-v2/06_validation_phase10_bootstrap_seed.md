# Validation — Phase 10 Bootstrap Seed

## Xác minh cấu trúc

Đã đối chiếu file seed với:

- `029_v2_connection_registry.sql`
- `030_v2_source_object_registry.sql`
- `031_v2_shadow_binding.sql`
- `032_v2_master_binding.sql`
- `033_v2_mapping_rule.sql`
- `036_v2_transmute_schedule.sql`

## Xác minh logic

Đã tự rà và vá một lỗi tiềm tàng:

- `mapping_rule_v2` không nên dùng `ON CONFLICT` target theo expression index
- đã đổi thành `ON CONFLICT DO NOTHING`

## Kết luận

Seed template hiện phù hợp để:

1. copy ra file môi trường riêng
2. sửa connection code / secret_ref / db/schema/table thật
3. chạy tay sau khi migrate xong
