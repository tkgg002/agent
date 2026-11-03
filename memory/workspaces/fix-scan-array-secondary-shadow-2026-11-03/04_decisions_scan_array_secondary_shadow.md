# 04 Decisions Log — fix-scan-array-secondary-shadow-2026-11-03

- [2026-11-03T10:00:21+07:00] **DG0 `C-REALITY` (`FLAG` → `RESOLVED`):**
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_service.go:198-215`
  - **Lý do:** Tại `scan_service.go:198-215`, khi `masterBindingID != nil`, hàm thực hiện `SELECT sb.shadow_schema, sb.shadow_table FROM cdc_system.master_binding mb JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id WHERE mb.id = ?` và gán đè `targetTable = sb.ShadowTable`, `schemaName = sb.ShadowSchema`.
- [2026-11-03T10:02:08+07:00] [Agent:laya_gate] DG1 S-RC FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_service.go:198-215 assigns targetTable = sb.ShadowTable from mb.shadow_binding_id
- [2026-11-03T10:04:18+07:00] [Agent:laya_gate] DG2 S-RC FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_service.go:198-215 assigns targetTable = sb.ShadowTable from mb.shadow_binding_id
- [2026-11-03T10:07:34+07:00] [Agent:laya_gate] DG3 S-RC FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_service.go:198-215 assigns targetTable = sb.ShadowTable from mb.shadow_binding_id
- [2026-11-03T10:23:16+07:00] [Agent:laya_gate] DG1 S-RC FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_service.go:198-215 assigns targetTable = sb.ShadowTable from mb.shadow_binding_id
- [2026-11-03T10:23:16+07:00] [Agent:laya_gate] DG2 S-RC FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_service.go:198-215 assigns targetTable = sb.ShadowTable from mb.shadow_binding_id
- [2026-11-03T10:23:16+07:00] [Agent:laya_gate] DG3 S-RC FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_service.go:198-215 assigns targetTable = sb.ShadowTable from mb.shadow_binding_id
