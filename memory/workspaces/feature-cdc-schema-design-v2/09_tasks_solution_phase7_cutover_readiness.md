# Solution — Phase 7 Cutover Readiness

## Delivered

Pha này chốt readiness cho cutover:

- lookup runtime của `command/recon/backfill/full-count/schema validation` đã ưu tiên metadata V2
- worker wiring đã cấp metadata provider tới các runtime service quan trọng
- đường `source -> shadow -> master` và operational metadata chính đã có thể chạy dựa vào `cdc_system`

## Cutover Checklist

1. Chạy migrations tới `036_v2_transmute_schedule.sql`.
2. Seed `cdc_system.connection_registry`.
3. Seed `cdc_system.source_object_registry`.
4. Seed `cdc_system.shadow_binding`.
5. Seed `cdc_system.master_binding`.
6. Seed `cdc_system.mapping_rule_v2`.
7. Nếu cần schedule tự động: seed `cdc_system.transmute_schedule`.
8. Xoá data shadow/master cũ theo kế hoạch vận hành của bạn.
9. Restart worker/service.
10. Verify:
    - ingest vào shadow được
    - `cdc.cmd.transmute-shadow` fanout đúng
    - master DDL auto-create đúng schema
    - transmute ghi được sang master DB đích
    - `sync_runtime_state` có cập nhật success/failure

## Remaining Legacy, But Not Immediate Cutover Blockers

- `cdc_internal.transmute_schedule` cũ chưa drop
- một số update side-effect vẫn ghi vào `cdc_table_registry`
- các service/phần test cũ còn model `TableRegistry` để giữ compatibility
