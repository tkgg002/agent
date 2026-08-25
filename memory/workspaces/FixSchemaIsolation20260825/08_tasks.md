# 08_tasks.md — Danh sách Task Chi tiết

- [ ] **Task 1: Chuẩn hóa `recon_check_heal_handler.go`:**
  - Bổ sung `ShadowSchema`, `ShadowTable`, `MasterSchema`, `MasterTable` vào payload và chuẩn hóa `targetTable` trước khi gọi `proposeHealSegmentA` / `proposeHealSegmentB`.
- [ ] **Task 2: Chuẩn hóa `recon_sysops_handler.go`:**
  - Bổ sung `ShadowSchema` vào payload của `HandleRetryFailed`, `HandleDebeziumSignal`, `HandleBackfillSourceTs`, `HandleDetectTimestampField`.
- [ ] **Task 3: Chuẩn hóa `recon_smoke.go` và `recon_tier_b.go`:**
  - Đổi từ `GetByTargetTable(ref.MasterTable)` sang `GetByTargetTableAndSchema(ref.ShadowTable, ref.ShadowSchema)`.
- [ ] **Task 4: Loại bỏ Fallback Đoán mò trong `recon_base_handler.go` & `helpers.go`:**
  - Bỏ các nhánh `GetTableConfig(pureTable)` và `GetTableConfigBySource(pureTable)` khi không có schema prefix.
- [ ] **Task 5: Chuẩn hóa Query trong `schema_validator.go`:**
  - Thêm điều kiện `shadow_schema` khi query table registry.
- [ ] **Task 6: Verification & Test Suite:**
  - Chạy full test suite và build worker.
