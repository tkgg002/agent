# Danh sách Task: Loại bỏ Smoke Check khỏi Lịch sử đối soát

## 1. Backend Tasks
- [ ] Task 1.1: Chỉnh sửa interface `ReconReader` trong `recon_reader.go` để thêm tham số `excludeSmoke bool` cho hàm `GetTableHistory`.
- [ ] Task 1.2: Chỉnh sửa concrete struct `reconReadRepoGorm` trong `recon_read_repo_gorm.go` để triển khai logic loại bỏ `cdc_recon_smoke_result` khi `excludeSmoke == true`.
- [ ] Task 1.3: Cập nhật struct `GetTableHistoryQuery` và hàm `Handle` trong `get_table_history.go`.
- [ ] Task 1.4: Cập nhật hàm `TableHistory` trong `reconciliation_handler_reports.go` để parser query string `exclude_smoke`.
- [ ] Task 1.5: Cập nhật `stubReconReader` trong `queries_test.go` để pass build test.
- [ ] Task 1.6: Build test backend Go (`go build ./...` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`).

## 2. Frontend Tasks
- [ ] Task 2.1: Cập nhật `useTableHistory` trong `useReconStatus.ts` để nhận và truyền param `excludeSmoke`.
- [ ] Task 2.2: Cập nhật lời gọi `useTableHistory` trong `ExecuteHealModal.tsx` để truyền `true` cho `excludeSmoke`.
- [ ] Task 2.3: Compile frontend React (`npx tsc --noEmit` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`).
