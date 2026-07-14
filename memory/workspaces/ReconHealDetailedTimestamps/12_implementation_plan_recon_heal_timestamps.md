# Kế Hoạch Triển Khai Chi Tiết - Bổ sung Thời gian Chữa lành Từng Loại Lỗi (Muscle Execution)

Dưới đây là kế hoạch chi tiết của Muscle để thực hiện thay đổi source code ở cả backend và frontend theo đúng Hồ sơ Giải pháp.

## 1. Tạo Bản sao Lưu (Restore-point backups)
Muscle sẽ copy nội dung của các file gốc cần chỉnh sửa sang các file `.bak` trong thư mục `backups/` của workspace này trước khi thay đổi.
Các file cần backup:
1. `centralized-data-service/internal/model/recon/reconciliation_report.go`
2. `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go`
3. `cdc-cms-service/internal/model/recon/reconciliation_report.go`
4. `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
5. `cdc-cms-web/src/hooks/useReconStatus.ts`
6. `cdc-cms-web/src/components/ExecuteHealModal.tsx`

## 2. Chi Tiết Các Bước Chỉnh Sửa Mã Nguồn

### Bước 2.1: Chỉnh sửa `centralized-data-service/internal/model/recon/reconciliation_report.go`
- Thêm 3 trường `HealedMismatchedAt`, `HealedMissingSrcAt`, `HealedMissingDestAt` dạng `*time.Time` vào struct `ReconciliationReport`.

### Bước 2.2: Chỉnh sửa `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go`
- Cập nhật hàm `finalizeReport` để gán giá trị thời gian xử lý khi các trường đếm tương ứng > 0.

### Bước 2.3: Chỉnh sửa `cdc-cms-service/internal/model/recon/reconciliation_report.go`
- Thêm 3 trường `HealedMismatchedAt`, `HealedMissingSrcAt`, `HealedMissingDestAt` vào struct `ReconciliationReport` tương tự bên `centralized-data-service`.

### Bước 2.4: Chỉnh sửa `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- Cập nhật select query `baseQuery` trong hàm `GetTableHistory` để select thêm 3 trường.
- Cập nhật select query thứ 2 trong `unionQuery` (Smoke Check) để select `NULL::timestamp without time zone` cho 3 trường này.

### Bước 2.5: Chỉnh sửa `cdc-cms-web/src/hooks/useReconStatus.ts`
- Thêm 3 trường `healed_mismatched_at`, `healed_missing_src_at`, `healed_missing_dest_at` dạng `string | null` vào interface `UnhealedReport` và `ReconReport`.

### Bước 2.6: Chỉnh sửa `cdc-cms-web/src/components/ExecuteHealModal.tsx`
- Viết helper `formatTimestamp` định dạng timestamp sang `YYYY-MM-DD HH:mm:ss`.
- Thay thế các cột cũ trong `healedReportColumns` bằng 3 cột mới hiển thị thông tin chữa lành chi tiết kèm thời gian tương ứng.

## 3. Kiểm Thử & Xác Minh
- Biên dịch `centralized-data-service`: `go build ./...`
- Biên dịch `cdc-cms-service`: `go build ./cmd/server/...`
- Biên dịch frontend: `npx tsc --noEmit`

## 4. Kiểm Toán Quy Trình (Governance & Linter)
- Chạy linter quy trình `python3 agent/tooling/verify_governance.py` để đảm bảo tuân thủ.
- Cập nhật `05_progress_recon_heal_timestamps.md`.
