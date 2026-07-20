# Danh sách task chi tiết - Fix lỗi hiển thị chênh lệch đối soát trên CMS FE

- [x] Task 1: Sửa hàm `getDiffIDs` trong `ExecuteHealModal.tsx` để parse đúng các trường JSON của Segment B (`missing_from_shadow`, `missing_from_master`, `mismatched`).
- [x] Task 2: Nâng cấp `popoverContent` trong `ExecuteHealModal.tsx` để hiển thị phân chia theo 3 loại (Thiếu ở Shadow, Thiếu ở Master, Lệch dữ liệu) đối với cả Segment A và Segment B.
- [x] Task 3: Chạy linter quy trình verify_governance.py để đảm bảo tính hợp lệ.
- [/] Task 4: Sửa cột ID lệch trong bảng chưa heal để dùng icon list `UnorderedListOutlined` và ẩn tag ID ngoài bảng (nhận feedback từ User).
