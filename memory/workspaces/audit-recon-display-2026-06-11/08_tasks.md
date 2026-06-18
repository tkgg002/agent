# Tasks: Fix recon display (trọn gói)
- [x] T1 Migration 085 ADD shadow_schema/shadow_table/run_id + index
- [x] T2 Worker model +3 field
- [x] T3 Worker recon_core: run_id + set keys mọi write site (A+B) → build PASS
- [x] T4 API model +3 field
- [x] T5 API recon_read_repo + ResolveTargetTableByScope query theo khóa pipeline + gom phiên run_id → build PASS
- [x] T6 FE truyền khóa + hiển thị phiên
- [x] T7 (build+migration; runtime chờ deploy) Verify build 2 service + test + runtime
