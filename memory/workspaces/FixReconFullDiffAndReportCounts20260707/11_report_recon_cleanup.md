# Báo cáo thực thi - muscle_execute: Recon Cleanup

Báo cáo chi tiết các file đã sửa đổi và kết quả chạy kiểm thử xác minh cho tác vụ dọn dẹp các cột dư thừa (`tier`/`target_table`) và tiêu chuẩn hóa thông tin Metadata nguồn (`source_type`/`source_host`/`source_table`) cho hệ thống đối soát dữ liệu (Reconciliation).

## 1. Danh sách các file đã sửa đổi & Thống kê dòng thay đổi
Dưới đây là thống kê diff của các file liên quan trực tiếp đến phase dọn dẹp:

### Component: `cdc-cms-service`
* **File đã sửa**:
  - [reconciliation_report.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/model/recon/reconciliation_report.go)
  - [source_object_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/source_object_read_repo_gorm.go)
  - [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
* **Tổng số dòng thay đổi**: +217 insertions, -58 deletions.

### Component: `centralized-data-service`
* **File đã sửa**:
  - [reconciliation_report.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go)
  - [recon_engine_segment_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_segment_b.go)
  - [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
  - [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
  - [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
  - [recon_engine_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_run.go)
* **Tổng số dòng thay đổi**: +591 insertions, -547 deletions.

### Component: `cdc-cms-web`
* **File đã sửa**:
  - [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
  - [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
* **Tổng số dòng thay đổi**: +96 insertions, -17 deletions.

---

## 2. Chi tiết các thay đổi chính

### Loại bỏ trường `Tier`
- Đã xóa trường `Tier` khỏi struct `ReconciliationReport` ở cả hai component `cdc-cms-service` và `centralized-data-service`.
- Xóa các gán trường `Tier: ...` khỏi các struct literal khởi tạo `ReconciliationReport` trong `recon_tier_a.go`, `recon_tier_b.go`, `recon_engine_run.go`.

### Thay đổi tag GORM và JOIN cho `TargetTable`
- Đổi tag GORM của `TargetTable` thành `gorm:"-"` trong `centralized-data-service` (không persist nữa).
- Cập nhật tag GORM cho `TargetTable` thành `gorm:"column:target_table"` trong `cdc-cms-service` để giữ tương thích ngược khi scan động.
- Sửa đổi các câu SQL JOIN trong `source_object_read_repo_gorm.go` để chuyển các liên kết JOIN của bảng `cdc_reconciliation_report rr` từ `rr.target_table` sang `rr.shadow_table`.

### Thêm thông tin Metadata nguồn
- Thêm `SourceType`, `SourceHost`, `SourceTable` vào `ReconciliationReport`.
- Cập nhật logic gán thông tin nguồn trong `stampA` (sử dụng helper `extractHost(entry.SourceURL)`) và `stampB` (gán `"postgresql"` / `"shadow_plane"` / `ref.ShadowTable`).
- Cập nhật các câu SQL trong `recon_read_repo_gorm.go` (`listLatestPrimary` và `GetTableHistory`) để lấy trực tiếp các cột `source_type`, `source_host`, `source_table`.

### Loại bỏ Smoke Check dư thừa
- Xóa bỏ hoàn toàn hàm `RunSmokeCheck` khỏi `recon_tier_a.go` và hàm `RunSmokeCheckB` khỏi `recon_tier_b.go`.
- Cập nhật `CheckAllSegmentA` trong `recon_engine_run.go` chuyển từ chạy `RunSmokeCheck` sang `RunDeepCheck`.
- Loại bỏ logic xử lý `TypeReconSmoke` khỏi `validateAndEnrichContext` và `executeGenericCheck` trong `recon_check_handler.go`.

### Cập nhật Frontend
- Cập nhật interface `ReconReport` trong `useReconStatus.ts`: loại bỏ `tier`, bổ sung `source_host` và `source_table`.
- Cập nhật `levelLabel` trong `ReconPipelineGrid.tsx` ánh xạ theo `check_type` thay vì `tier` (tránh lỗi do không tồn tại `tier`).
- Thêm helper `getSourceDisplayName` để ghép và định dạng chuỗi tên nguồn một cách đầy đủ thông tin: `[source_type] source_host / source_db . source_table`.

---

## 3. Kết quả kiểm thử xác minh

Tất cả các test suite liên quan đều được thực thi và vượt qua 100%:

1. **Centralized Data Service (Service Tests)**:
   - Lệnh: `go test -count=1 ./internal/service/recon/...`
   - Kết quả: `ok  centralized-data-service/internal/service/recon  0.753s` (PASS)

2. **Centralized Data Service (Handler Tests)**:
   - Lệnh: `go test -count=1 ./internal/handler/recon/...`
   - Kết quả: `ok  centralized-data-service/internal/handler/recon  0.817s` (PASS)

3. **CDC CMS Service (Integration Tests)**:
   - Lệnh: `go test ./test/...`
   - Kết quả: Tất cả packages test compile và chạy thành công (PASS)

4. **Frontend Type Check (CDC CMS Web)**:
   - Lệnh: `npx tsc --noEmit`
   - Kết quả: Biên dịch thành công, 0 lỗi TypeScript (PASS)
