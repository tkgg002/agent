# Báo cáo thay đổi - Sửa lỗi giao diện Master / Shadow Registry reset trạng thái Collapse khi reload trang

## Thay đổi đã thực hiện

### 1. [MasterRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx)
- **Vấn đề:** Khi reload trang `/masters`, các panel `Collapse` hiển thị nhóm Schema Master bị reset trạng thái và tự động đóng lại (collapsed).
- **Giải pháp:** 
  - Khai báo state `activeKeys` lưu trữ mảng các schema đang được mở rộng.
  - Sử dụng `useEffect` để tải trạng thái đã lưu từ `localStorage` dưới key `master-registry-active-keys` khi dữ liệu `groupedMasters` hoàn tất tải lần đầu. Nếu chưa có lịch sử lưu, mặc định mở rộng tất cả các panel.
  - Khi operator thay đổi trạng thái đóng/mở thủ công, cập nhật `activeKeys` và lưu vào `localStorage` qua hàm `handleCollapseChange`.
  - Thay thế `key` động trước đó bằng việc gán trực tiếp `activeKey={activeKeys ?? Object.keys(groupedMasters)}` và `onChange={handleCollapseChange}` cho component `Collapse`.

### 2. [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx)
- **Cập nhật:** Đã hoàn trả (revert) toàn bộ thay đổi liên quan đến `TableRegistry.tsx`. Các panel ở tab Shadow Objects và Shadow Bindings vẫn giữ nguyên hành vi cũ là đóng mặc định khi load trang để giao diện gọn gàng hơn, đúng như thiết kế ban đầu.

---

## Kết quả kiểm tra
- Dự án frontend đã được build lại thành công (`npm run build`) không gặp lỗi TypeScript nào.
- Trạng thái đóng/mở của các Schema Master trên trang `/masters` được duy trì chính xác sau khi reload trang.
