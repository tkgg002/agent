---
tier: T2
workspace: audit-shadow-master-update-drop-2026-10-20
suffix: shadow_master_update_drop
---

# 04 — Architecture Decision Records (ADRs) & Decision Gate Resolutions

## ADR-01: Đọc động khóa chính (`conflictTarget`) từ `master_binding.transform_spec` (`"pk"`) thay vì hardcode cột xóa hay khóa đối soát
- **Context:** Các bảng Master được sinh bởi `MasterDDLGenerator` không có cột kỹ thuật `_source_id` (`master_ddl_generator.go:137-144`), nhưng cấu hình `master_binding.transform_spec` khai báo khóa chính nghiệp vụ dưới dạng JSON `{"pk": "_id"}` hoặc `{"pk": "id"}` (hoặc mặc định `"_gpay_id"` nếu không khai báo).
- **Decision:** Sử dụng thống nhất hàm `resolveConflictTarget(binding, colTypes)` (`transmuter_utils.go:180-191`) và `parsePKFromSpec(spec)` (`transmuter_utils.go:195-218`) cho toàn bộ:
  1. `deleteFromMasterBySourceAndGpayIDs` (`transmuter.go:1551-1684`) khi xử lý sự kiện `DELETE` hoặc `row.Deleted` trên Master.
  2. `bulkUpsertMaster` (`transmuter.go:1734-1745`) để cho phép cập nhật `"_gpay_id" = EXCLUDED."_gpay_id"` khi `conflictTarget != "_gpay_id"`.
  3. `resolvePKColumnAndType` trong `recon_execute_heal_handler.go:307-336` khi giải mã `sourceIDs` và heal dữ liệu Segment B.

## DG3 Post-Implementation Flag Resolutions (`06_laya_summary_dg3_20261020T152525.json`)
- **`IMPL-F1-DYNAMIC-PK-DELETE-AND-SPLIT` (`FLAG` $\rightarrow$ `RESOLVED`):**
  - Round 1 và Round 3 của Laya đều chọn `CORRECT`/`YES`, được xác nhận trực tiếp tại `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:1621-1683` và `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go:696-699`.
- **`IMPL-F3-INCREMENTAL-RETROACTIVE-SCAN` (`FLAG` $\rightarrow$ `RESOLVED`):**
  - Cả 3 vòng (`r1`, `r2`, `r3`) của Laya đều chọn `CORRECT`/`YES`, được xác nhận trực tiếp tại `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:1508-1535`.
- [2026-10-20T16:08:23+07:00] [Agent:laya_gate] DG3 AF-02-SPLIT-SINGLE-TASK-PARTIAL-ERROR-SWALLOW FLAG resolved: Verified directly in code (r1=CORRECT 0.7675, r2=CORRECT 0.7089): lines 790-794 check if leftErr != nil && rightErr != nil before calling task.complete(leftErr), otherwise calling task.complete(nil). REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go:790-795
