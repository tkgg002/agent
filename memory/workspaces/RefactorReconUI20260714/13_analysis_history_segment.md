# Phân tích kỹ thuật: Bổ sung cột Chặng vào Nhật ký đối soát

## 1. Hiện trạng trước khi thay đổi
- Bảng "Nhật ký đối soát (30 phiên gần nhất)" hiển thị lịch sử đối soát từ API `GetTableHistory`.
- Phản hồi từ API này (kiểu dữ liệu `ReconReport`) đã chứa trường `segment`, nhưng UI Table chưa định nghĩa cột cho trường này nên người dùng không quan sát được chặng cụ thể trực tiếp từ danh sách phiên.

## 2. Giải pháp kỹ thuật
- Thêm cột `Chặng` vào định nghĩa `columns` của Ant Design `<Table>` trong component `ReconPipelineGrid.tsx`.
- Cột này maps với thuộc tính `segment` của `ReconReport`.
- Nếu giá trị `segment` là `'shadow_master'`, hiển thị màu tím đại diện cho chặng B (Shadow ➔ Master).
- Ngược lại, hiển thị màu xanh nước biển đại diện cho chặng A (Source ➔ Shadow).

## 3. Đánh giá rủi ro & Tác động
- Thay đổi hoàn toàn cục bộ trên lớp React/UI.
- Không thay đổi data contract hay API endpoints.
- Không gây ảnh hưởng tới hiệu năng do chỉ hiển thị thêm một trường dữ liệu đã có sẵn trong payload.
