# KẾ HOẠCH TRIỂN KHAI CHI TIẾT CỦA AI (AI IMPLEMENTATION PLAN) — BỘ CHỌN NHÁNH PIPELINE CỤ THỂ CHO BÁO CÁO CONFLUENCE

- **Ngày thực hiện:** `2026-10-15`
- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`

---

## 1. Mục tiêu Kiến trúc
Giải quyết triệt để bài toán quan hệ phân nhánh `1 Source ➔ N Shadows ➔ M Masters` khi cấu hình và xuất Báo cáo Đối soát Hằng ngày lên Atlassian Confluence:
1. Người dùng chọn đích danh từng nhánh (`Source ➔ Shadow ➔ Master`) trên giao diện `ActivityManager.tsx`.
2. Backend `ConfluenceReportRunner` chỉ tổng hợp đúng các nhánh được chọn, tra cứu số đếm ngày tạo tại cả 3 trạm (`source`, `shadow`, `master`) theo bộ khóa 5 thành phần và lưu lượng CDC theo đúng nhánh đó.

---

## 2. Các bước đã triển khai
1. **Bước 1 — Phân tích dữ liệu thật trên `cdc_dw` (`13_analysis_confluence_1_to_n_lineage.md`):**
   - Xác minh cây huyết thống thực tế của `export-jobs` (`1 Source ➔ 1 Shadow ➔ 3 Masters`) và `payment-bills` (`1 Source ➔ 2 Shadows ➔ 2 Masters`).
2. **Bước 2 — Thiết kế giải pháp và trình duyệt (`09_tasks_solution_confluence_branch_selection.md`):**
   - Thiết kế cấu trúc JSON `SelectedTable` / `ConfluenceBranchItem` lưu trọn vẹn định danh 3 trạm.
   - Chờ và nhận lệnh `APPROVE` từ User theo đúng Rule #13.
3. **Bước 3 — Thực thi Backend (`centralized-data-service`):**
   - Cập nhật `internal/service/confluence/confluence_runner.go`: `resolveBranchLineage`, `lookupLayerDailyCount`, `lookupBranchCDCFlow`, `buildBranchReportRow`, `buildConfluenceXHTML`.
   - Cập nhật `internal/service/confluence/confluence_runner_test.go`: bổ sung unit test và live DB integration test trên `cdc_dw`.
4. **Bước 4 — Thực thi Frontend (`cdc-cms-web`):**
   - Xây dựng `ConfluenceBranchSelector` trong `src/pages/ActivityManager.tsx` với toàn bộ React Hooks ở top-level.
   - Tích hợp vào Modal Tạo mới, Modal Chỉnh sửa và bảng danh sách lịch trình.
5. **Bước 5 — Kiểm định Gates G1–G8 (`06_validation_confluence_branch_selection.md`):**
   - Chạy `go test -v ./internal/service/confluence/...` (PASS 100% cùng DB thật `cdc_dw`).
   - Chạy `go build -o /tmp/cds-worker ./cmd/worker` (Exit 0).
   - Chạy `npm run build` (Exit 0).
