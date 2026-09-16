# 14 Walkthrough — Force Transform & Bug Fix

## Hướng dẫn sử dụng tính năng mới

### 1. Khắc phục hiển thị Transform In-Progress trên Table Registry
- Truy cập `Table Registry` (`/tables` hoặc `/shadow/tables`).
- Khi bấm chạy Transform ở bảng cha (ví dụ `payment_bills`), bảng con (`payment_bills_1`) sẽ không còn bị hiển thị trạng thái "Đang chạy" nhầm lẫn nữa nhờ việc mapping key bằng `shadow_binding_id` độc lập.

### 2. Sử dụng tính năng "Force Transform" khi đổi type cột hoặc cần bóc lại dữ liệu
1. **Bước 1 — Bấm Transform:**
   - Tại trang `Table Registry`, bấm nút **Transform** ở cột Action của bảng tương ứng (hoặc bảng con).
2. **Bước 2 — Modal Transform xuất hiện:**
   - Modal hiển thị thông tin bảng shadow đích.
   - **Tùy chọn Force Transform:** Bật công tắc `Force Transform (Ghi đè field)`.
   - **Chọn field cần ghi đè:** Dropdown multi-select tự động hiển thị danh sách các trường active của bảng. Operator có thể chọn cụ thể các trường (ví dụ `updated_at`, `created_at`...) hoặc bấm "Chọn tất cả".
3. **Bước 3 — Xác nhận chạy:**
   - Bấm **Chạy Force Transform** (nút màu đỏ nổi bật).
   - Job được điều phối xuống Worker và chạy ngầm theo từng chunk 1000 rows.
   - Tiến độ thực thi được cập nhật realtime trong bảng `cdc_system.transform_jobs` và hiển thị trên CMS.
