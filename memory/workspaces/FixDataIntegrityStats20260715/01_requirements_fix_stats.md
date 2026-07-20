# Yêu cầu: Sửa lỗi hiển thị thống kê Data Integrity trên UI

Trang đối soát dữ liệu (`http://localhost:5173/data-integrity`) đang hiển thị sai số lượng thống kê (Tổng bảng, Khớp, Lệch).

## Hiện trạng lỗi:
- Hệ thống có 3 bảng đối soát thực tế.
- Trên UI, số lượng "Tổng bảng", "Khớp", "Lệch" đang hiển thị không đúng với số lượng bảng thực tế (ví dụ: hiển thị Tổng bảng lớn hơn 3, số Khớp/Lệch bị nhân đôi hoặc sai lệch).
- Nguyên nhân: Các chỉ số thống kê ở đầu trang `DataIntegrity.tsx` được tính toán trực tiếp từ danh sách report thô (raw reports từ API `/api/reconciliation/report`), nơi chứa các bản ghi phân tách theo từng chặng (segment: `source_shadow` và `shadow_master`). Do đó, một bảng có thể bị tính thành 2 dòng report khác nhau, dẫn đến số liệu thống kê bị đúp.

## Mục tiêu (DoD):
- Đưa các chỉ số thống kê (Tổng bảng, Khớp, Lệch) về đúng số lượng bảng thực tế bằng cách gom nhóm các segment report thành Pipeline đại diện cho mỗi bảng (giống như logic hiển thị Grid Pipelines).
- "Tổng bảng" phải hiển thị đúng số lượng bảng thực tế độc lập.
- "Khớp" phải hiển thị số lượng bảng có trạng thái khớp hoàn toàn ở cả 2 chặng.
- "Lệch" phải hiển thị số lượng bảng có ít nhất một chặng bị lệch/cảnh báo/lag.
- Linter quy trình chạy pass thành công.
