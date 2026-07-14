# Kế hoạch Triển khai Chi tiết - Lọc bỏ Smoke Check ở Database Lịch sử

Kế hoạch thực thi chi tiết dưới vai trò Muscle (Chief Engineer) nhằm triển khai lọc bỏ Smoke Check ở database lịch sử đối soát theo thiết kế tại [09_tasks_solution_recon_history.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconHistoryExcludeSmoke/09_tasks_solution_recon_history.md).

## Các bước thực hiện

### 1. Thay đổi Backend Go
1. Sửa `cdc-cms-service/internal/app/queries/recon/recon_reader.go`:
   - Thêm tham số `excludeSmoke bool` vào hàm `GetTableHistory`.
2. Sửa `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`:
   - Cập nhật hàm `GetTableHistory` để nhận tham số `excludeSmoke`.
   - Nếu `excludeSmoke` bằng `true`, dùng `baseQuery` (chỉ bảng `cdc_reconciliation_report`).
   - Nếu bằng `false`, dùng `unionQuery` (UNION ALL với `cdc_recon_smoke_result`).
   - Thay `unionQuery` bằng `queryToUse` trong `countQuery` và `selectQuery`.
3. Sửa `cdc-cms-service/internal/app/queries/recon/get_table_history.go`:
   - Thêm trường `ExcludeSmoke bool` vào struct `GetTableHistoryQuery`.
   - Trong `Handle`, truyền `q.ExcludeSmoke` vào repository call.
4. Sửa `cdc-cms-service/internal/api/recon/reconciliation_handler_reports.go`:
   - Lấy query parameter `exclude_smoke == "true"` và gán vào `ExcludeSmoke`.
5. Sửa `cdc-cms-service/test/internal/app/queries/queries_test.go`:
   - Cập nhật mock `stubReconReader.GetTableHistory` để khớp signature interface mới.

### 2. Thay đổi Frontend React
1. Sửa `cdc-cms-web/src/hooks/useReconStatus.ts`:
   - Cập nhật hook `useTableHistory` để nhận thêm tham số `excludeSmoke = false`.
   - Truyền query parameter `exclude_smoke: 'true'` lên API nếu `excludeSmoke` bằng `true`.
2. Sửa `cdc-cms-web/src/components/ExecuteHealModal.tsx`:
   - Truyền tham số `true` vào đối số thứ 5 của `useTableHistory`.

### 3. Biên dịch và Kiểm chứng
1. Biên dịch Go Backend: `go build ./cmd/server/...` tại `cdc-cms-service`.
2. Biên dịch Web Frontend: `npx tsc --noEmit` tại `cdc-cms-web`.
3. Ghi chép tiến trình thực thi vào `05_progress_recon_history.md`.
