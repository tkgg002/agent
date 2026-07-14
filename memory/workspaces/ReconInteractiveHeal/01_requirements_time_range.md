# Yêu cầu chi tiết - Bổ sung khoảng thời gian đối soát trong modal Chữa lành

## 1. Bối cảnh & Mục tiêu
Khi mở modal "Chữa lành đối soát cho [table]", người dùng muốn xem chi tiết khoảng thời gian của phiên đối soát chưa xử lý (thường là dữ liệu đối soát theo khung giờ hoặc theo ngày). Điều này giúp người vận hành biết chính xác phiên lệch thuộc khoảng thời gian nào trước khi quyết định chữa lành.

## 2. Chi tiết yêu cầu
*   **Màn hình hiển thị:** Modal `ExecuteHealModal` (`ExecuteHealModal.tsx`).
*   **Cột mới:** Thêm cột "Khoảng thời gian" vào bảng hiển thị danh sách các phiên đối soát chưa xử lý.
*   **Dữ liệu sử dụng:** 
    *   `recon_start_time` (`ReconStartTime` trong API response).
    *   `recon_end_time` (`ReconEndTime` trong API response).
*   **Định dạng hiển thị:**
    *   Nếu cùng một ngày: `HH:MM - HH:MM (DD/MM/YYYY)` (ví dụ: `10:00 - 11:00 (08/07/2026)`).
    *   Nếu khác ngày: `HH:MM DD/MM/YYYY - HH:MM DD/MM/YYYY`.
    *   Nếu không có dữ liệu: hiển thị `—`.
