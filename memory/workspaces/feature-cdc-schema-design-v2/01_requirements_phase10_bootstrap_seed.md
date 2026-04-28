# Requirements — Phase 10 Bootstrap Seed

## Mục tiêu

- Tạo bộ seed SQL tối thiểu cho `cdc_system` để bootstrap V2 sau khi wipe dữ liệu.
- Seed phải bám đúng control-plane schema hiện tại.
- Seed không được nằm trong migration chain mặc định vì đây là dữ liệu vận hành theo môi trường.

## Phạm vi

- Tạo file SQL template trong repo `centralized-data-service`
- Ghi rõ cách dùng cho cutover `wipe & bootstrap`

## Definition of Done

1. Có file SQL template seed cho:
   - `connection_registry`
   - `source_object_registry`
   - `shadow_binding`
   - `master_binding`
   - `mapping_rule_v2`
   - `transmute_schedule`
2. Seed phản ánh đúng convention:
   - system -> `cdc_system`
   - shadow -> `shadow_<source_db>`
3. Seed an toàn để copy/chỉnh theo từng môi trường.
