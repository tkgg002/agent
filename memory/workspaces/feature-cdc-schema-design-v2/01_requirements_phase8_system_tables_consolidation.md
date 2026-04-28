# Requirements — Phase 8 System Tables Consolidation

## Scope

- Dời toàn bộ system tables còn ở public/cdc_internal vào cdc_system.
- Đổi runtime/model chính sang cdc_system.
- Đổi shadow physical schema sang shadow_<source_db> thay cho cdc_internal.
