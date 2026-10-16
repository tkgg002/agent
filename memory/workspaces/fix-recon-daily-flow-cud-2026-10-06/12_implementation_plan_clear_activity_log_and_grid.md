# Kế hoạch Triển khai Chi tiết (AI Implementation Plan)
## Tính năng Clear Activity Log Động & Tái cấu trúc Bảng Đối Soát 3 Trạm (V3 — Multi-Connector Safe)

### 1. Mục tiêu & Phạm vi
- Triển khai API xóa nhật ký `DELETE /api/activity-log` an toàn, cách ly triệt để theo bộ tứ định vị pipeline: `shadow_schema` + `shadow_table` + `source_database` + `master_table`.
- Kích hoạt Partition Pruning cho bảng `cdc_system.cdc_activity_log` thông qua điều kiện kép: `created_at` và `started_at`.
- Triệt tiêu Guard Clause chống xóa sạch toàn bộ hệ thống hoặc lỗi GORM missing WHERE.
- Thêm giao diện Modal Dọn dẹp với `DatePicker` động trên Ant Design Web CMS.
- Tái cấu trúc Bảng Đối Soát 3 Trạm (`ReconPipelineGrid.tsx`) bóc tách Khối 1: `SNAPSHOT / RE-SYNC` và Khối 2-4: `KAFKA CDC STREAM`.
- Khắc phục lỗi sót param `shadow_schema` trong hook `usePipelineActivityLog`.

### 2. Các file mã nguồn cần chỉnh sửa
1. Backend:
   - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`
   - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/activity_log_handler.go`
   - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go`
2. Frontend:
   - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts`
   - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx`

### 3. Kế hoạch Kiểm tra & Nghiệm thu
- Biên dịch backend Go: `go build ./cmd/... ./internal/...` không có lỗi.
- Typecheck frontend TypeScript: `npx tsc --noEmit` không có lỗi.
- Kiểm tra tính đúng đắn trên dữ liệu thực tế: Pipeline `export_jobs` và `payment_bills`.
