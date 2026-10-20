# 08 — Tasks & Autonomous Full-Stack Checklist: Chuẩn Hóa Đếm Bảng Theo Ngày & `:8082/metrics`

- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`
- **Task Suffix:** `metrics_daily_table_count_8082`
- **Trạng thái hiện tại:** `PROPOSE-ONLY` (Chờ User chốt đặc tả 4 trạm và ban hành lệnh `APPROVE` — Rule #1, Rule #13).

---

## Task 1 — Bổ sung cấu hình Cột Ngày Tạo (`created_at_field`) cho Bảng Source và liên kết huyết thống sang Shadow & Master (Không đoán mò)
- **[Mô tả & Phạm vi]:**
  - Bổ sung trường cấu hình cột ngày tạo bản ghi (`created_at_field`) trên `source_object_registry` ([`source_object_registry.go:8-49`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/source_object_registry.go#L8-L49)) và Form đăng ký/sửa Bảng Source tại [`TableRegistry.tsx:1647-1801`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx#L1647-L1801).
  - Liên kết cột ngày tạo từ **Bảng Source** $\rightarrow$ qua `mapping_rule_v2` sang cột tương ứng trên **Bảng Shadow** $\rightarrow$ qua `master_mapping_rules` sang cột tương ứng trên **Bảng Master** để khi tạo lịch `source_count` trên [`ActivityManager.tsx:221-234,444-446`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L221-L234) hệ thống lấy đúng cột đã khai báo (không dùng regex đoán mò và không gán nhầm `_created_at` cho bảng Master).
- **[Context & Logs]:**
  - Hiện tại `source_object_registry` chỉ có `primary_key_field` và `timestamp_field` (`lastUpdatedAt` cho recon window). Lịch `#23` và `#24` bị lỗi `ERROR: column "createdAt" / "_created_at" does not exist (SQLSTATE 42703)`.
- **[Definition of Done]:**
  - Không còn logic đoán mò cột ngày tạo; Bảng Source, Shadow, Master xác định đúng cột ngày tạo đã cấu hình; pass `go test` và `npx tsc --noEmit`.

## Task 2 — Lọc bản ghi xóa mềm (`_deleted = false`) khi đếm theo ngày trên Shadow/Master & Xuất Metric Đếm Theo Ngày ra `:8082/metrics`
- **[Mô tả & Phạm vi]:**
  - Tại [`source_count_runner.go:288-295`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L288-L295) (`countSQL`), bổ sung điều kiện `AND "_deleted" = false` khi `params.LayerType` là `shadow` hoặc `master` trên PostgreSQL để số đếm theo ngày chỉ tính bản ghi đang active, khớp với Source.
  - Khai báo `DailyTableCount` (`cdc_daily_table_count`) trong [`prometheus.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go) và cập nhật trong [`source_count_runner.go:255-270`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L255-L270) (kèm hydrate các bản ghi ngày gần nhất khi khởi động worker, tuyệt đối không quét tổng toàn bảng).
- **[Context & Logs]:**
  - Hiện tại `source_count_runner.go` chưa đẩy metric nào ra `:8082/metrics` và `countSQL` chưa lọc `_deleted = false`.
- **[Definition of Done]:**
  - `curl -s http://localhost:8082/metrics | grep cdc_daily_table_count` hiển thị đúng số đếm theo ngày cho từng trạm (`source`, `shadow`, `master`), khớp 100% với `cdc_system.daily_source_counts`.
