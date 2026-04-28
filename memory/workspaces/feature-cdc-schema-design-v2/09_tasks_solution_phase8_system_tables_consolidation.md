# Solution — Phase 8 System Tables Consolidation

## Delivered

- System models/runtime chính đã chuyển sang `cdc_system`.
- Legacy system tables ở public/cdc_internal có migration dời về `cdc_system`.
- Shadow namespace không còn mặc định `cdc_internal`, mà là `shadow_<source_db>`.

## Note

- Một số reference legacy còn tồn tại trong comments/migrations cũ hoặc compatibility paths chưa bị xoá hẳn.
