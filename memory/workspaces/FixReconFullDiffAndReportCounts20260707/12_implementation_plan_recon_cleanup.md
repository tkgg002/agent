# Kế hoạch triển khai - muscle_execute: Recon Cleanup

Kế hoạch thực thi các thay đổi mã nguồn nhằm dọn dẹp các trường redundancy (`tier`/`target_table`) và tiêu chuẩn hóa thông tin Metadata nguồn (`source_type`/`source_host`/`source_table`) cho hệ thống Reconciliation.

## 1. Mục tiêu
- Cập nhật định nghĩa struct `ReconciliationReport` ở cả `cdc-cms-service` và `centralized-data-service`.
- Sửa đổi các câu query GORM và các hàm JOIN trong repo của `cdc-cms-service`.
- Cập nhật logic gán thông tin metadata trong `recon_engine_segment_b.go`.
- Loại bỏ các hàm kiểm tra smoke check dư thừa ở Tier A/B và handler check.
- Cập nhật Frontend Hook và Component để hiển thị chính xác tên nguồn.
- Kiểm thử và bảo đảm tất cả các unit test đều pass.

## 2. Các bước thực hiện chi tiết

### Phase 1: Chuẩn bị & Audit logs
- Khởi tạo file `05_progress_recon_cleanup.md` để ghi nhận tiến độ thực thi.

### Phase 2: Chỉnh sửa `cdc-cms-service`
1. Sửa `internal/model/recon/reconciliation_report.go`
   - Loại bỏ trường `Tier`.
   - Cập nhật `TargetTable` tag GORM thành `gorm:"column:target_table"` để tương thích ngược.
   - Thêm `SourceType`, `SourceHost`, `SourceTable`.
2. Sửa `internal/infra/persistence/source/source_object_read_repo_gorm.go`
   - Cập nhật các câu JOIN đến `cdc_reconciliation_report rr` tại dòng 84, 268, 404 để JOIN dựa trên `shadow_table`.
3. Sửa `internal/infra/persistence/recon/recon_read_repo_gorm.go`
   - Cập nhật `GetTableHistory` và `listLatestPrimary` để loại bỏ `tier` và chọn trực tiếp `source_type`, `source_host`, `source_table`.

### Phase 3: Chỉnh sửa `centralized-data-service`
1. Sửa `internal/model/recon/reconciliation_report.go`
   - Loại bỏ `Tier`.
   - Cập nhật `TargetTable` tag GORM thành `gorm:"-"`.
   - Thêm `SourceType`, `SourceHost`, `SourceTable`.
2. Sửa `internal/service/recon/recon_engine_segment_b.go`
   - Cập nhật hàm `stampA` và `stampB` để gán thông tin `SourceType`, `SourceHost`, `SourceTable`.
3. Sửa `internal/service/recon/recon_tier_a.go`
   - Xóa hàm `RunSmokeCheck`.
4. Sửa `internal/service/recon/recon_tier_b.go`
   - Xóa hàm `RunSmokeCheckB`.
5. Sửa `internal/handler/recon/recon_check_handler.go`
   - Xóa case `TypeReconSmoke` khỏi `validateAndEnrichContext`.
   - Xóa logic gọi `RunSmokeCheck`/`RunSmokeCheckB` khỏi `executeGenericCheck`.

### Phase 4: Chỉnh sửa `cdc-cms-web`
1. Sửa `src/hooks/useReconStatus.ts`
   - Thay đổi kiểu dữ liệu `ReconReport`: loại bỏ `tier`, thêm `source_host`, `source_table`.
2. Sửa `src/components/ReconPipelineGrid.tsx`
   - Cập nhật `levelLabel` hiển thị theo `check_type`.
   - Thêm helper `getSourceDisplayName` và cập nhật cách hiển thị `sourceName`.

### Phase 5: Xác minh & Kiểm thử
- Chạy các lệnh test trong `centralized-data-service` và `cdc-cms-service`.
- Chạy biên dịch TS trong `cdc-cms-web` với `npx tsc --noEmit`.

### Phase 6: Tổng kết & Báo cáo
- Ghi nhận kết quả, so sánh diff, tổng hợp báo cáo vào `11_report_recon_cleanup.md` và `05_progress_recon_cleanup.md`.
