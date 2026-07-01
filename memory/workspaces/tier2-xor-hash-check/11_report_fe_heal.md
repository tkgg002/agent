# Báo Cáo Thay Đổi Frontend (Heal Mode Routing)

Báo cáo chi tiết các thay đổi trong các file Frontend phục vụ tính năng kích hoạt heal theo chế độ chọn (Window vs Full-diff).

## 1. Danh sách các file thay đổi
- [src/hooks/useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts):
  - Thay đổi hook `useHealMutation` để truyền thêm các tham số `mode`, `startTime`, và `endTime` trong payload khi gọi API `/api/reconciliation/heal`.
  - Thay đổi mutation parameter type để chấp nhận các optional fields `mode?: string`, `startTime?: string`, `endTime?: string`.
  - Số lượng dòng thay đổi: ~15 dòng.
  
- [src/components/ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx):
  - Bổ sung prop `isHeal?: boolean` vào interface `ConfirmDestructiveModalProps`.
  - Cập nhật signature `onConfirm` hỗ trợ truyền các tham số: `onConfirm: (reason: string, mode?: string, startTime?: string, endTime?: string) => Promise<void> | void;`
  - Thêm state: `mode` (mặc định `'window'`), `startTime`, `endTime`, `timeError`.
  - Bổ sung `useEffect` khi open modal để tính toán mặc định thời gian Window 7 ngày trước.
  - Bổ sung hàm `handleTimeChange` để validate khoảng thời gian Full-diff không được vượt quá 30 ngày và thời gian kết thúc không nhỏ hơn bắt đầu.
  - Cập nhật logic `isFormValid` kết hợp kiểm tra `timeError` nếu ở chế độ Full-diff.
  - Bổ sung phần giao diện radio buttons (Window vs Full-diff) và Inputs datetime-local cho start/end time khi prop `isHeal` là `true`.
  - Số lượng dòng thay đổi: ~120 dòng.

- [src/pages/DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx):
  - Cập nhật type `ModalAction` hỗ trợ thêm optional `isHeal?: boolean` trong kind `'heal'`.
  - Cập nhật hàm `openHeal` để truyền thêm `isHeal: true` khi thiết lập modal plan.
  - Cập nhật `handleConfirm` để nhận thêm `mode`, `startTime`, `endTime` và truyền chúng xuống `heal.mutateAsync`.
  - Cập nhật render `ConfirmDestructiveModal` để truyền prop `isHeal={modalPlan.action.kind === 'heal'}`.
  - Số lượng dòng thay đổi: ~15 dòng.

## 2. Kết quả Xác thực
- Chạy biên dịch `npm run build` thành công hoàn chỉnh, không gặp bất cứ lỗi TypeScript hay linter nào.
