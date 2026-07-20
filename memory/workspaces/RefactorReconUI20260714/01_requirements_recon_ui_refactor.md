# Yêu cầu Chi tiết: Tối ưu hóa UI Đối Soát (Reconciliation UI Refactoring)

## 1. Bối cảnh
Nhằm tinh giản giao diện đối soát, tăng cường trải nghiệm vận hành và đảm bảo tính chính xác trong xử lý dữ liệu theo ngữ cảnh, cần chỉnh sửa lại hai modal chính trong `cdc-cms-web`.

## 2. Phạm vi Yêu cầu

### Yêu cầu 1: Modal "Bắt đầu đối soát" (ConfirmDestructiveModal.tsx)
- **Ẩn Chọn Chặng đối soát (Segment):** Bỏ giao diện chọn chặng (Segment Selector). Thay vào đó, modal sẽ tự động sử dụng loại chặng được truyền trực tiếp từ dòng (row/record) được click ở giao diện danh sách.
- **Chế độ đối soát mặc định:** Chế độ đối soát (`checkMode`) mặc định sẽ được chọn là `2h` (Hot Mode 2 giờ). Khoảng thời gian custom (`customRange`) cũng sẽ mặc định lùi 2 giờ tính từ thời điểm hiện tại.
- **Ẩn tùy chọn Deep Check:** Ẩn (nhưng vẫn giữ lại trong mã nguồn) tùy chọn `Deep Check (Quét toàn collection)` bằng CSS/logic (ví dụ: gán `display: none` cho Radio component tương ứng), không xóa bỏ mã nguồn vì sẽ tái sử dụng trong tương lai.

### Yêu cầu 2: Modal "Chữa lành đối soát" (ExecuteHealModal.tsx)
- **Lọc theo Chặng (Segment Filtering):** Mở ở chặng nào thì danh sách "Phiên chưa xử lý" và "Phiên đã xử lý" chỉ hiển thị các phiên đối soát thuộc chặng đó.
  - Nếu mở từ chặng A (`source_shadow`), chỉ hiện các phiên có `segment === 'source_shadow'` hoặc không có segment.
  - Nếu mở từ chặng B (`shadow_master`), chỉ hiện các phiên có `segment === 'shadow_master'`.

## 3. Definition of Done (DoD)
- [ ] Mở modal "Bắt đầu đối soát" từ chặng A hoặc chặng B, modal tự nhận diện đúng chặng tương ứng và ẩn mục chọn chặng.
- [ ] Chế độ đối soát mặc định hiển thị là 2h, Deep Check được ẩn khỏi UI.
- [ ] Mở modal "Chữa lành đối soát" ở chặng A, danh sách phiên chưa xử lý và đã xử lý chỉ hiển thị các phiên đối soát của chặng A.
- [ ] Mở modal "Chữa lành đối soát" ở chặng B, danh sách tương ứng chỉ hiển thị các phiên đối soát của chặng B.
- [ ] Chạy linter quy trình (`verify_governance.py`) báo PASS.
