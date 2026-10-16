# BÁO CÁO THAY ĐỔI MÃ NGUỒN (CHANGE REPORT) — BỘ CHỌN NHÁNH PIPELINE CỤ THỂ (SOURCE ➔ SHADOW ➔ MASTER) CHO BÁO CÁO CONFLUENCE

- **Ngày thực hiện:** `2026-10-15`
- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`

---

## 1. Tổng quan các File đã thay đổi

| STT | Repository | File | Số dòng thay đổi | Mô tả tóm tắt |
|---|---|---|---|---|
| 1 | `centralized-data-service` | [`internal/service/confluence/confluence_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go) | `+438 / -81` (Tổng 684 dòng) | Mở rộng `SelectedTable` lưu đầy đủ thông tin 3 trạm (`Source`, `Shadow`, `Master`); thêm `resolveBranchLineage`, `lookupLayerDailyCount` (tra cứu chính xác theo 5 khóa `(report_date, layer_type, connection_code, container_namespace, object_name)`), `lookupBranchCDCFlow` (truy vấn từ `cdc_system.cdc_activity_log` có partition pruning, loại bỏ bảng ảo `recon_hourly_pipeline_stats`), `buildBranchReportRow`, và cập nhật `buildConfluenceXHTML` thành bảng 8 cột hiển thị rõ 3 trạm của từng nhánh. |
| 2 | `centralized-data-service` | [`internal/service/confluence/confluence_runner_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner_test.go) | `+105 / -18` (Tổng 153 dòng) | Cập nhật `TestBuildConfluenceXHTML`, thêm `TestCandidateObjectNames` và `TestLiveBranchLineageAndReportRow` kiểm thử trực tiếp trên DB `cdc_dw` (`localhost:5433`) với 2 nhánh cụ thể của `export-jobs` (`mst:51` và `mst:48`). |
| 3 | `cdc-cms-web` | [`src/pages/ActivityManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx) | `+536 / -44` (Tổng 2065 dòng) | Thêm kiểu `ConfluenceBranchItem`, hàm `normalizeSavedBranches`, và component `ConfluenceBranchSelector` cho phép chọn phân tầng `1. Trạm Nguồn (Source)` ➔ `2. Trạm Trung Gian (Shadow)` ➔ `3. Trạm Đích (Master)` và bấm `+ Thêm nhánh vào báo cáo`. Tích hợp vào cả Modal Tạo mới, Modal Chỉnh sửa và cột hiển thị `Phạm vi / Đối tượng`. |

---

## 2. Chi tiết Thay đổi Kỹ thuật

1. **Triệt tiêu hoàn toàn hành vi phẳng hóa 1-1 và `LIMIT 1` theo tên bảng trần trụi:**
   - Trước khi sửa: UI chỉ cho chọn tên bảng phẳng (`export_jobs`), Backend query `WHERE object_name IN ? LIMIT 1` không phân biệt `layer_type`, `connection_code`, `container_namespace`.
   - Sau khi sửa:
     - Trên UI (`ConfluenceBranchSelector`): Người vận hành chọn cụ thể từng nhánh từ gốc đến ngọn (`Source ➔ Shadow ➔ Master`), có thể thêm nhiều nhánh cụ thể vào danh sách báo cáo và xem/xoá từng nhánh trong bảng danh sách.
     - Dưới Backend (`ConfluenceReportRunner`): Mỗi dòng báo cáo ứng với đúng 1 nhánh `SelectedTable` đã chọn. Số đếm ngày tạo của từng trạm (`source`, `shadow`, `master`) được tra cứu độc lập bằng bộ khóa 5 thành phần `(report_date, layer_type, connection_code, container_namespace, object_name)` trong `cdc_system.daily_source_counts`.
2. **Loại bỏ bảng không tồn tại `cdc_system.recon_hourly_pipeline_stats`:**
   - Thay thế bằng hàm `lookupBranchCDCFlow` truy vấn trực tiếp từ bảng thật `cdc_system.cdc_activity_log` với điều kiện Partition Pruning (`created_at >= startOfDay AND created_at < partitionEnd`) cho đúng cặp `shadow_schema.shadow_table` (`operation = 'kafka-consumer'`) và `master_schema.master_table` (`operation IN ('transmute', 'transmute_sub')`).
