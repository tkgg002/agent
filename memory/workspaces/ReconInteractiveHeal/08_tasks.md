# Danh sách Task chi tiết (Tasks) - Hiệu chỉnh Chữa lành tương tác (Rev.3)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal)

- `[x]` Hiệu chỉnh tài liệu workspace (`ReconInteractiveHeal`).
- `[x]` Triển khai chỉnh sửa mã nguồn (Ủy quyền hoàn toàn cho Muscle):
  - `[x]` RESTORE và cấu hình API Gateway (khai báo `ExecuteHealCommand`, `/execute-heal`, khôi phục `/heal` nguyên bản).
  - `[x]` RESTORE và cấu hình Frontend:
    - `[x]` Ẩn lựa chọn quét/thời gian ở `ConfirmDestructiveModal` khi `isHeal = true`.
    - `[x]` Cấu hình nút "Chữa lành" mở `ConfirmDestructiveModal` chạy `useHealMutation`.
    - `[x]` Thêm nút "Thực thi chữa lành" bên cạnh "Chữa lành" trong `ReconPipelineGrid.tsx` (DrillDown) và prop `onExecuteHeal` trong `DataIntegrity.tsx`.
    - `[x]` Cấu hình nút "Thực thi chữa lành" mở `ExecuteHealModal` chạy `useExecuteHealMutation`.
  - `[x]` RESTORE và cấu hình Worker:
    - `[x]` Khôi phục `HandleReconHeal` nguyên bản trong `recon_handler_run.go` (giữ nguyên logic check Tier 2).
    - `[x]` Đăng ký lại `cdc.cmd.execute-heal` gọi `HandleExecuteHeal` (trong `recon_execute_heal.go`).
- `[x]` Xác minh toàn diện (Definition of Done):
  - `[x]` Chạy `go test ./...` trên worker thành công.
  - `[x]` Build frontend và backend thành công.
  - `[x]` Báo cáo kết quả bằng Walkthrough.

## Phase 6: Khôi phục cấu hình thời gian quét đối soát (healcheck)
- `[ ]` Cấu hình API Gateway (`cdc-cms-service`):
  - `[ ]` Thêm `start_time` và `end_time` vào `ReconCheckCommand`.
  - `[ ]` Cập nhật `TriggerCheck` và `TriggerCheckAll` để parse và gán thời gian.
- `[ ]` Cấu hình Frontend (`cdc-cms-web`):
  - `[ ]` Cập nhật `useCheckTableMutation` nhận `startTime`/`endTime`.
  - `[ ]` Cập nhật `ConfirmDestructiveModal.tsx` hiển thị radio chọn Window/Full Scan và RangePicker cho check table.
  - `[ ]` Cập nhật `DataIntegrity.tsx` chuyển tiếp `startTime`/`endTime` đến mutation.
- `[ ]` Xác minh biên dịch & hoạt động.
