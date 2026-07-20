# Kế hoạch sửa lỗi hiển thị chênh lệch đối soát trên CMS FE

## User Review Required
> [!IMPORTANT]
> Thay đổi này chỉ sửa đổi logic hiển thị ở Frontend (`cdc-cms-web`), cụ thể là parse đúng JSON dạng `{missing_from_shadow, missing_from_master, mismatched}` và hiển thị chi tiết 3 danh sách này trên Popover khi click vào cột ID lệch. 
> Logic tính toán count ở Backend (`centralized-data-service`) vẫn được giữ nguyên (Stale = Mismatched + Missing from Shadow, Thừa = Missing from Shadow) để tránh làm lệch các báo cáo và logic chữa lành đã chạy ổn định.

## Proposed Changes

### Component `cdc-cms-web`

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
* Sửa hàm `getDiffIDs` ở dòng 85-128 để map đúng key JSON:
  * Segment B (`shadow_master`): Đọc `missing_from_shadow`, `missing_from_master`, `mismatched`.
  * Segment A (`source_shadow`): Đọc `missing_from_dest`, `missing_from_src`, `mismatched`.
* Nâng cấp `popoverContent` ở cột `ID lệch` (dòng 320-356) hiển thị phân nhóm rõ ràng 3 loại:
  * **Thiếu ở Shadow** (màu đỏ)
  * **Thiếu ở Master** (màu cam)
  * **Lệch dữ liệu** (màu vàng)
* Sửa logic render cột `ID lệch` của bảng **chưa heal**:
  * Ẩn render tag ID trực tiếp ngoài cột.
  * Chỉ hiển thị duy nhất một nút tròn nhỏ có icon `UnorderedListOutlined` (giống hệt bên bảng đã heal).
  * Khi click vào nút list này, Popover chi tiết phân tách 3 loại sẽ được hiển thị.

## Verification Plan

### Manual Verification
* Chạy service CMS Web và click vào xem chi tiết ID lệch của bản ghi có chênh lệch Segment B (như bản ghi ID 39 ở DB).
* Xác nhận ID lệch `70799479065416231` hiển thị đúng dưới mục **Thiếu ở Shadow (Missing from Shadow)**.
* Xác nhận không còn hiển thị `—` ở cột ID lệch.
