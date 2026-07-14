# Báo cáo thay đổi (Report) - Khắc phục hiển thị dữ liệu chưa Heal và Bổ sung thông tin đối soát

## Danh sách các tệp thay đổi

### 1. [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)
- **Vị trí thay đổi:** Hàm `openHeal` (Dòng 225 - 234).
- **Mô tả:** Thay đổi logic từ thiết lập `modalPlan` (mở ConfirmDestructiveModal) sang thiết lập `executeHealTarget` (mở trực tiếp ExecuteHealModal).
- **Số lượng dòng thay đổi:** Xóa 9 dòng, thêm 6 dòng (Chênh lệch: -3 dòng).

### 2. [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- **Vị trí thay đổi:** Title modal, table columns (`reportColumns`), getDiffIDs helper, and executeHeal function.
- **Mô tả:** 
  - Thay đổi tiêu đề hiển thị từ `"Chữa lành drift — "` thành `"Chữa lành đối soát cho "`.
  - Bổ sung cột **Loại kiểm tra** (thông tin `check_type` hiển thị thân thiện dạng Tag).
  - Bổ sung cột **ID lệch** (hiển thị danh sách ID bị lệch. Nếu có trên 2 ID, hiển thị 2 ID đầu kèm nút 👁️ Popover hiển thị toàn bộ ID kèm nút Copy nhanh).
  - Tăng độ rộng của modal `width` lên `960` để vừa vặn cho các cột mới.
  - Sử dụng `Array.from(new Set(...))` để lọc trùng (unique) `reportIds` truyền lên.
- **Số lượng dòng thay đổi:** Thêm mới và chỉnh sửa khoảng 120 dòng code.

### 3. [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
- **Vị trí thay đổi:** `executeHeal`, `executeHealSegA`, `executeHealSegB`, và thêm helper ở cuối file.
- **Mô tả:**
  - Lọc trùng `ReportIDs` ở hàm `executeHeal` bằng helper `uniqueUint64s`.
  - Lọc trùng danh sách các record IDs lệch (mismatched, missing, orphan) trong `executeHealSegA` và `executeHealSegB` bằng helper `uniqueStrings` để tránh chạy đi chạy lại 1 cái nhiều lần.
  - Thêm các hàm helper `uniqueStrings` và `uniqueUint64s`.
- **Số lượng dòng thay đổi:** Thêm mới và chỉnh sửa khoảng 50 dòng code.

---
## Tổng kết số lượng dòng thay đổi
- **Tổng số tệp thay đổi:** 3 tệp.
- **Tổng số dòng code được chỉnh sửa/thêm mới:** ~177 dòng.
