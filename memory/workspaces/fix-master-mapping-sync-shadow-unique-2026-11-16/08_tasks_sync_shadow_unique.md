---
tier: T2
suffix: sync_shadow_unique
workspace: fix-master-mapping-sync-shadow-unique-2026-11-16
---
# 08 Tasks — Điều Tra & Thiết Kế Giải Pháp Sửa Lỗi "Sync Từ Shadow" Vi Phạm Ràng Buộc Duy Nhất (`binding_id=61`)

## Task List

### T1. [S2 Investigate] Truy vết End-to-End luồng "Sync từ Shadow" từ Frontend -> CMS API -> Application Command -> Persistence & Schema Migrations
- **Context (`file:line`):**
  - Frontend: `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx`, `cdc-cms-web/src/services/api.ts` (tìm cả thông báo lỗi tiếng Việt `"Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)"` ở FE hoặc BE).
  - Backend CMS: `cdc-cms-service/internal/api/master/master_mapping_rule_handler.go`, `cdc-cms-service/internal/app/commands/master/*`, `cdc-cms-service/internal/infra/persistence/master/*`.
  - Migrations & DB Schema: toàn bộ chuỗi migrations của `mapping_rule_master` và `master_binding` trong `cdc-cms-service` và `centralized-data-service`.
- **Logs / Test:** Kiểm tra log runtime hiện có của `cdc-cms-service` (nếu đang chạy local/docker) hoặc tái hiện chính xác điều kiện gây `SQLSTATE 23505` qua phân tích tĩnh + unit test.
- **Definition of Done:** Xác định chính xác 100% (`VERIFIED` kèm `file:line`):
  1. Nơi phát sinh thông báo `"Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)"`.
  2. Câu lệnh `INSERT` / `UPSERT` cụ thể trong `SyncFromShadow` và các Unique Constraint / Unique Index thực tế trên bảng `cdc_system.mapping_rule_master`.
  3. Kịch bản dữ liệu cụ thể trên `master_table = cdc_payment_bill` (`binding_id=61`) kích hoạt lỗi trùng khóa (ví dụ: nhiều `master_binding` cùng tên bảng `cdc_payment_bill` như `binding_id=61` vs `binding_id=95`, hoặc xung đột `target_column` khi đổi tên cột `camelCase -> snake_case`, hoặc `ON CONFLICT` thiếu/sai khóa).

### T2. [S2.5 Pre-Plan Critique (DG0) + S3 Solution + DG1/DG2] Phản biện Giả định & Thiết kế Giải pháp Tối ưu Nhất
- **Context (`file:line`):** Toàn bộ callers đọc/ghi `mapping_rule_master` ở cả `cdc-cms-service` và `centralized-data-service` (`TransmuterModule`, `MasterDDLGenerator`, `SaveMasterMappingRule`, `SyncMasterRulesFromShadow`).
- **Logs / Test:** Chạy `$T/laya_gate.py run --stage dg0` và `$T/laya_gate.py run --stage dg1` (`PASS`).
- **Definition of Done:**
  - Đối chiếu đủ 5 bước Blast-Radius = 0 (`system_overview_and_design_pattern.md` Mục 5).
  - Trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** ánh xạ 1-1 với từng phát hiện `F1..F4` kèm **Code Demo** trong `09_tasks_solution_sync_shadow_unique.md` và hiển thị trực tiếp **Code Demo** trên chat; dừng chờ User `APPROVE`.

### T3. [S4–S6 Execute, Test & Final Audit — Chỉ thực thi sau khi có User `APPROVE`]
- **Context (`file:line`):**
  - Backend Persistence: `cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:209-281` (`syncRulesFromAllShadowsTx`)
  - Backend Unit Tests: `cdc-cms-service/internal/infra/persistence/master/master_repo_gorm_test.go`
  - Frontend Error Map: `cdc-cms-web/src/utils/apiError.ts:41-57`
- **Logs / Test:**
  - Chạy qua `$T/evidence_run.py`: `go build ./internal/... ./cmd/...`, `go vet ./internal/... ./cmd/...`, `go test -v ./internal/infra/persistence/master/... ./internal/app/commands/master/...` (gồm test tái hiện Red->Green cho rename `camelCase -> snake_case`, `1 shadow -> N master fields`, `multi_shadow` trùng cột, `status = 'pending'`), và `npm run build` tại `cdc-cms-web`.
- **Definition of Done:**
  - Pass đủ Gates `G1–G8`, `DG3` (`PASS`), ghi biên lai `EV-...` vào `audit_report_sync_shadow_unique.md`, append `DONE` vào `05_progress_sync_shadow_unique.md`, và `verify_governance.py` đạt `PASSED` 🟢.
