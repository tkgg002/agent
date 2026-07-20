# Yêu cầu Chi tiết: Thêm Chặng (Segment) vào Nhật ký đối soát

## 1. Bối cảnh
Nhằm giúp người dùng dễ dàng theo dõi và phân biệt giữa các phiên đối soát thuộc Chặng A (Source ➔ Shadow) và Chặng B (Shadow ➔ Master), cần hiển thị cột "Chặng" (segment) trong bảng "Nhật ký đối soát (30 phiên gần nhất)" tại component `ReconPipelineGrid.tsx`.

## 2. Phạm vi Yêu cầu
- Bổ sung cột "Chặng" vào Table columns trong component `ReconPipelineGrid.tsx`.
- Cột "Chặng" sẽ hiển thị:
  - `<Tag color="purple">Shadow → Master</Tag>` nếu `segment === 'shadow_master'`.
  - `<Tag color="blue">Source → Shadow</Tag>` nếu `segment !== 'shadow_master'`.
- Độ rộng cột (width) là 120px, khớp với thiết kế cột Chặng ở các modal khác để tạo sự đồng bộ.

## 3. Definition of Done (DoD)
- [ ] Thêm cột "Chặng" vào bảng "Nhật ký đối soát" trong `ReconPipelineGrid.tsx`.
- [ ] Dữ liệu hiển thị đúng tag màu tương ứng với từng chặng (chặng A màu xanh, chặng B màu tím).
- [ ] Dự án `cdc-cms-web` build thành công 100%.
- [ ] Chạy linter quy trình `verify_governance.py` báo PASS.
