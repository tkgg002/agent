# Danh sách Task chi tiết (Checklist)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal - Phase Split)

### Phase 1: Chuẩn bị & Thiết kế
- [x] Tạo tài liệu workspace `01_requirements_split.md`
- [x] Tạo tài liệu workspace `05_progress_split.md`
- [x] Tạo file checklist `08_tasks_split.md`
- [x] Tạo và nộp `implementation_plan.md` artifact chờ User duyệt

### Phase 2: Thực thi Backend & Database (Sau khi duyệt)
- [x] Chạy migration script DB (nếu chưa chạy).
- [x] Cập nhật API Gateway (`cdc-cms-service`):
  - [x] Sửa `TriggerHeal` trong `reconciliation_handler_heal.go`.
  - [x] Xóa `TriggerExecuteHeal` trong `reconciliation_handler_execute_heal.go`.
  - [x] Sửa `internal/router/router.go` (loại bỏ `/reconciliation/execute-heal`).
- [x] Cập nhật Worker (`centralized-data-service`):
  - [x] Sửa struct `ReconHandler` trong `recon_handler.go` để thêm `masterDB`.
  - [x] Cập nhật `server_setup.go` để tiêm `masterDB` vào `reconHandler`.
  - [x] Cập nhật `recon_execute_heal.go` hoàn thiện logic soft-delete cho chặng A & B.
  - [x] Thêm các hàm helper `quoteRelation` và `quoteIdent` vào `recon_execute_heal.go`.

### Phase 3: Thực thi Frontend (Sau khi duyệt)
- [x] Cập nhật `useReconStatus.ts` (sửa `useHealMutation`, xóa `useExecuteHealMutation`).
- [x] Rename `ExecuteHealModal.tsx` thành `HealModal.tsx`, cập nhật component gọi `useHealMutation`.
- [x] Cập nhật `DataIntegrity.tsx` (dùng `HealModal`, xóa `ExecuteHealModal`, loại bỏ nút Execute Heal).
- [x] Cập nhật `ReconPipelineGrid.tsx` (loại bỏ prop `onExecuteHeal` và Thunderbolt icon).

### Phase 4: Kiểm thử & Nghiệm thu
- [-] Chạy unit test backend gateway & worker (đã chạy nhưng bị timeout do chờ phê duyệt từ xa).
- [x] Kiểm tra compile Frontend (`npx tsc --noEmit`).
- [x] Kiểm tra thủ công luồng tương tác trên UI.
