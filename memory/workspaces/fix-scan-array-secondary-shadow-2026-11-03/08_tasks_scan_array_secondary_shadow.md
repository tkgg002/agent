# 08 Tasks: Kiểm tra & Thiết kế Giải pháp Lỗi Scan Array (Flatten) Trên Bảng Secondary Shadow (`binding_id=77`)

tier: T2
suffix: scan_array_secondary_shadow
date: 2026-11-03

## Danh sách Tasks

### TASK-01: Điều tra End-to-End Callchain của `Scan Array (Flatten)` với `binding_id=77` (S2 Investigate & DG0) — [DONE]
- **Mô tả:** Truy vết luồng bấm nút `Scan Array (Flatten) → Tạo rules PENDING để duyệt` trên trang `/masters/transaction_history/mappings?binding_id=77` từ Frontend (`cdc-cms-web`) xuống Backend (`cdc-cms-service` và `centralized-data-service`), xác định chính xác câu thông báo lỗi "không có db shadow" phát sinh tại `file:line` nào và vì sao field nằm ở bảng phụ `transaction_history_payload` lại gây lỗi này.
- **Context `file:line`:**
  - `data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:391-426, 1235-1273`
  - `data-hub/cdc-cms-web/src/components/master/ShadowSourceBadge.tsx:60-110`
  - `data-hub/cdc-cms-service/internal/api/system/introspection_handler.go:318-464`
  - `data-hub/cdc-cms-service/internal/infra/persistence/master/mapping_rule_repo_gorm.go:359-372`
  - `data-hub/centralized-data-service/internal/service/source/scan_service.go:191-492`
  - `data-hub/centralized-data-service/internal/service/source/scan_master_binding_resolver.go:1-33`
  - `data-hub/centralized-data-service/internal/service/master/transmuter.go:1768-1782`
- **Logs / Test:** Kiểm chứng qua mã nguồn thực tế tại HEAD và chạy cổng `DG0` (`06_laya_summary_dg0_20261103T095727.json` — `gate: PASS`).
- **Definition of Done (DoD):** Xác định rõ 100% `file:line` gây lỗi ở cả FE và BE (`F1`–`F4`), vượt qua `DG0` (`PASS`).

### TASK-02: Thiết kế Giải pháp Kỹ thuật & Code Demo (S3 Solution & DG1/DG2) — [DONE]
- **Mô tả:** Thiết kế giải pháp duy nhất tối ưu nhất (ánh xạ 1-1 với từng phát hiện `F1..F4`) để `Scan Array (Flatten)` hỗ trợ đầy đủ cả trường nằm trên bảng chính (`transaction_history`) lẫn trường nằm trên bảng phụ (`transaction_history_payload`) trong mô hình `multi_shadow` (`binding_id=77`).
- **Context `file:line`:** `09_tasks_solution_scan_array_secondary_shadow.md`.
- **Logs / Test:** Chạy `laya_gate.py run --stage dg1` (gộp DG1 + DG2) đạt `PASS`.
- **Definition of Done (DoD):** Hoàn tất `09_tasks_solution_scan_array_secondary_shadow.md`, qua cổng `DG1/DG2` (`PASS`), trình bày trực tiếp Code Demo trên khung chat và dừng chờ User `APPROVE`.

### TASK-03: Thực thi Giải pháp Backend tại `centralized-data-service` & `cdc-cms-service` (S4 Execute — Sau `APPROVE`) — [DONE]
- **Mô tả:**
  1. Tạo file mới đơn trách nhiệm `centralized-data-service/internal/service/source/scan_array_target_resolver.go` (`311` dòng) với `resolveShadowTargetForScanArray`, `resolveParentV2IDForScanArray`, `cleanLogicalExplodePath`, `parseScanNodeJSON` (giữ nguyên `scan_master_binding_resolver.go` = `33` dòng).
  2. Cập nhật và rút gọn `ScanService.ScanArrayFields` (`scan_service.go:191-452`) sử dụng helper mới và hỗ trợ stringified JSON (`NOW = 758 < 772` dòng, giảm `14` dòng).
  3. Tạo file mới đơn trách nhiệm `centralized-data-service/internal/service/master/transmuter_gjson_path.go` (`52` dòng, `resolveRuleGJSONValue`) và rút gọn `TransmuterModule.extractColumns` (`transmuter.go:1802`) xuống 1 dòng (`NOW = 1874 < 1888` dòng, giảm `14` dòng).
  4. Cập nhật `GetApprovedJSONFieldsByMasterBindingID` (`mapping_rule_repo_gorm.go:359-394`) lấy đủ JSON fields của mọi bảng Shadow thuộc `master_binding_id` kèm fallback tương thích ngược.
- **Context `file:line`:**
  - `data-hub/centralized-data-service/internal/service/source/scan_array_target_resolver.go:1-311` *(File Mới — SRP)*
  - `data-hub/centralized-data-service/internal/service/source/scan_service.go:191-452`
  - `data-hub/centralized-data-service/internal/service/master/transmuter_gjson_path.go:1-52` *(File Mới — SRP)*
  - `data-hub/centralized-data-service/internal/service/master/transmuter.go:1802`
  - `data-hub/cdc-cms-service/internal/infra/persistence/master/mapping_rule_repo_gorm.go:359-394`
- **Logs / Test:** Chạy unit/integration test qua `evidence_run.py` lấy biên lai `EV-20261103T110607-9063`, `EV-20261103T110718-9536`, `EV-20261103T110723-0564`, `EV-20261103T105533-e97a`, `EV-20261103T110728-cc95`, `EV-20261103T110844-3700`, `EV-20261103T110952-ffc4`, `EV-20261103T111001-0671`.
- **Definition of Done (DoD):** Tất cả test xanh (`PASS`), `wc -l` giảm dòng trên cả `scan_service.go` (`-14` dòng) và `transmuter.go` (`-14` dòng).

### TASK-04: Thực thi Giải pháp Frontend tại `cdc-cms-web` & Kiểm định Cuối (S4–S6 — Sau `APPROVE`) — [DONE]
- **Mô tả:**
  1. Tạo file mới đơn trách nhiệm `cdc-cms-web/src/utils/masterScanArray.ts` (`55` dòng: `resolveShadowTableForExplodePath`, `buildScanArrayFieldOptions`).
  2. Cập nhật `loadAllBindingShadowRules` trong `ShadowSourceBadge.tsx:60-108` bảo toàn `shadow_table`.
  3. Cập nhật và rút gọn `handleScanArray` + `Select.options` trong `MasterMappingFieldsPage.tsx:392-433, 1251` (`NOW = 1388 < 1403` dòng, giảm `15` dòng).
  4. Chạy build `npx vite build` (`EV-20261103T105542-9217`), chạy `DG3` (`06_laya_summary_dg3_20261103T111143.json` & `06_laya_summary_dg3review_20261103T111143.json` — `PASS`), xuất `audit_report_scan_array_secondary_shadow.md` và chạy `verify_governance.py`.
- **Context `file:line`:**
  - `data-hub/cdc-cms-web/src/utils/masterScanArray.ts:1-55` *(File Mới — SRP)*
  - `data-hub/cdc-cms-web/src/components/master/ShadowSourceBadge.tsx:60-108`
  - `data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:392-433, 1251`
- **Logs / Test:** Biên lai `EV-20261103T105542-9217` cho `npx vite build` và `DG3 PASS` (`06_laya_summary_dg3_20261103T111143.json`).
- **Definition of Done (DoD):** Build sạch 0 lỗi, vượt qua đủ 8 cổng G1–G8, `verify_governance.py` `PASSED` 🟢.
