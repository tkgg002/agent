# Implementation — Phase 10 Bootstrap Seed

## File mới

- [bootstrap_cdc_system_v2_template.sql](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/deployments/sql/bootstrap_cdc_system_v2_template.sql)

## Nội dung chính

File seed này bootstrap một flow mẫu hoàn chỉnh cho V2:

1. `cdc_system.connection_registry`
   - 1 source Mongo
   - 1 shadow Postgres
   - 1 master Postgres
2. `cdc_system.source_object_registry`
   - ví dụ `billing.payments`
3. `cdc_system.shadow_binding`
   - route về `shadow_billing.payments`
4. `cdc_system.master_binding`
   - route về `dw_finance.payment_fact`
5. `cdc_system.mapping_rule_v2`
   - seed 6 mapping rule mẫu
6. `cdc_system.transmute_schedule`
   - seed mode `post_ingest`

## Quyết định thiết kế

- Không nhét seed này vào `migrations/039...`
  - vì seed là dữ liệu vận hành theo môi trường
  - không phải schema bất biến
- Dùng `ON CONFLICT` cho các bảng có natural key rõ ràng:
  - `connection_code`
  - `object_code`
  - `binding_code`
- Với `mapping_rule_v2`, dùng `ON CONFLICT DO NOTHING`
  - vì unique index hiện tại có expression `COALESCE(master_binding_id, 0)`

## Convention đã khóa

- system tables -> `cdc_system`
- shadow schema -> `shadow_<source_db>`
- master table -> schema đích theo binding
