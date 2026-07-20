# Báo cáo thay đổi chi tiết (v1.13 - Hotfix hiển thị FE)

## 1. Danh sách file thay đổi
* **File:** [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
* **Số lượng dòng thay đổi:** ~80 dòng code.

## 2. Chi tiết các thay đổi
* **Hàm `getDiffIDs`:**
  * Sửa logic parse JSON cho cả hai Segment A (`source_shadow`) và Segment B (`shadow_master`).
  * Đọc chính xác các trường `missing_from_shadow`, `missing_from_master`, `mismatched` (đối với segment B) và `missing_from_dest`, `missing_from_src`, `mismatched` (đối với segment A).
* **Cột `ID lệch` (diff_ids):**
  * Sửa logic render cột để chỉ hiển thị duy nhất một nút tròn nhỏ có icon `UnorderedListOutlined` (giống bảng đã heal), ẩn hoàn toàn các tag ID trực tiếp ngoài cột.
  * Khi click vào nút list này, Popover chi tiết (chứa 3 danh sách ID phân tách theo 3 loại: Thiếu ở Shadow (màu đỏ), Thiếu ở Master/Source (màu cam), Lệch dữ liệu (màu vàng)) sẽ được hiển thị.
  * Hỗ trợ nút "Copy tất cả" danh sách ID trong Popover.
  * Xóa bỏ icon `EyeOutlined` và hàm helper `getTagColor` không còn sử dụng để giữ code sạch sẽ và tránh lỗi tsc.

## 3. Kết quả xác minh
* Build frontend (`npm run build`) thành công 100%, không có lỗi biên dịch.
* Đã chạy linter quy trình (`verify_governance.py`) và kết quả **PASSED 🟢**.

