# Kế hoạch Triển khai: Tối ưu hóa UI Đối Soát (Reconciliation UI Refactoring)

## 1. Yêu cầu & Mục tiêu
Tinh chỉnh giao diện đối soát trong ứng dụng web `cdc-cms-web` để tối ưu hóa quy trình vận hành:
1. **Modal "Bắt đầu đối soát" (`ConfirmDestructiveModal.tsx`)**:
   - Mặc định chọn chế độ đối soát `2h`.
   - Ẩn tùy chọn `Deep Check` (Quét toàn collection) bằng CSS (`display: none`), giữ nguyên logic để tái sử dụng sau này.
   - Bỏ/Ẩn giao diện chọn chặng (Segment Selector), tự động dùng chặng được truyền vào qua `initialSegment`.
2. **Modal "Chữa lành đối soát" (`ExecuteHealModal.tsx`)**:
   - Lọc danh sách "Phiên chưa xử lý" và "Phiên đã xử lý" dựa trên chặng của dòng dữ liệu đang được mở (`segment` prop).

---

## 2. Thiết kế Kỹ thuật Chi tiết

### Thay đổi 1: `ConfirmDestructiveModal.tsx`
- **Chế độ quét mặc định:** Thay đổi state `checkMode` và `useEffect` khởi tạo mặc định về `2h` thay vì `7d`. Cập nhật `customRange` mặc định tương ứng là `[endTime.subtract(2, 'hour'), endTime]`.
- **Ẩn Deep Check:** Thêm style `display: 'none'` vào component `<Radio value="deep">`.
- **Ẩn Segment Selector:** 
  - Giao diện chọn chặng nằm trong khối `Chọn chặng đối soát` (khoảng dòng 168-178).
  - Chúng ta sẽ comment out toàn bộ phần giao diện này (hoặc thêm điều kiện `isManualRecon && false` / ẩn bằng CSS), để người dùng không thay đổi được chặng. Khi submit, modal vẫn gửi giá trị `segment` nhận được từ `initialSegment || ''`.

### Thay đổi 2: `ExecuteHealModal.tsx`
- Nhận prop `segment?: string`.
- Lọc danh sách `reports` (phiên chưa xử lý) và `healedReports` (phiên đã xử lý) bằng filter:
  - Nếu `segment` là `'source_shadow'`: Lọc `r.segment === 'source_shadow' || !r.segment` (do các bản ghi cũ của chặng A có thể có `segment` null/rỗng).
  - Nếu `segment` là `'shadow_master'`: Lọc `r.segment === 'shadow_master'`.
  - Nếu `segment` không được truyền hoặc rỗng, hiển thị toàn bộ không lọc.
- Do `reports` được lọc trước khi tính các biến phụ trợ (`healMismatched`, `healMissingDest`, `pruneMissingSrc`), các checkboxes hành động và badge count của tab sẽ tự động cập nhật chính xác theo dữ liệu đã lọc của chặng đó.

---

## 3. Kế hoạch Kiểm tra (DoD Verification)
1. **Chạy linter quy trình:**
   - Chạy `python3 agent/tooling/verify_governance.py` để đảm bảo tài liệu đầy đủ và đúng quy trình.
2. **Kiểm tra Frontend:**
   - Đảm bảo code TSX build thành công không lỗi syntax/type.
   - Nhờ User hoặc tự test (nếu có môi trường local dev server chạy) để xác minh giao diện hoạt động chính xác.
