# Implementation — Phase 8 System Tables Consolidation

- Thêm migration [037_move_system_tables_to_cdc_system.sql](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/037_move_system_tables_to_cdc_system.sql).
- Đổi model system tables sang `cdc_system.*`.
- Vá raw SQL chính cho `recon_runs`, `failed_sync_logs`, `pending_fields`, `enum_types`, `cdc_table_registry`.
- Đổi sinkworker sang shadow schema `shadow_<source_db>` qua topic Debezium.
