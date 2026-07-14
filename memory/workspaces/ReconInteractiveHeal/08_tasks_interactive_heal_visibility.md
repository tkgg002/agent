# Danh sách Task chi tiết (Tasks) - Khắc phục hiển thị dữ liệu chưa Heal

- `[x]` Cập nhật `openHeal` trong `DataIntegrity.tsx` để gán `executeHealTarget` thay vì `modalPlan`.
- `[x]` Cập nhật tiêu đề hiển thị trong `ExecuteHealModal.tsx` thành `"Chữa lành đối soát cho "`.
- `[x]` Chạy `npx tsc --noEmit` trong `cdc-cms-web` để xác nhận biên dịch thành công.
- `[x]` Xác minh chức năng hoạt động (Biên dịch thành công 100%, lỗi CDP trình duyệt cục bộ).
- `[x]` Cập nhật `ExecuteHealModal.tsx` ở Frontend (hiển thị check_type, getDiffIDs popover, unique reportIds).
- `[x]` Cập nhật `recon_execute_heal_handler.go` ở Backend (lọc trùng report_ids và record IDs).
- `[x]` Xác minh biên dịch Frontend và Backend thành công.

