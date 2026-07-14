# Yêu cầu Chi tiết (Specs) - Khắc phục hiển thị dữ liệu chưa Heal trong modal "Chữa lành đối soát"

## Mục tiêu
Khi người dùng bấm nút "Chữa lành" trên màn hình Data Integrity:
1. Hệ thống hiển thị modal `ExecuteHealModal` thay vì `ConfirmDestructiveModal`.
2. Modal hiển thị đầy đủ danh sách các phiên chưa được heal (unhealed reports) cho bảng đích tương ứng.
3. Tiêu đề của modal hiển thị chính xác: "Chữa lành đối soát cho [table_name]".

## Definition of Done (DoD)
- Nút "Chữa lành" trong component `DataIntegrity.tsx` gọi `openHeal` và mở modal `ExecuteHealModal`.
- Tiêu đề của `ExecuteHealModal` được cập nhật từ `"Chữa lành drift — "` thành `"Chữa lành đối soát cho "`.
- Dự án `cdc-cms-web` biên dịch thành công không có lỗi (`npx tsc --noEmit`).
