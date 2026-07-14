# Yêu cầu cập nhật Reconciliation UI & API Pipeline

Dự án yêu cầu bổ sung/sửa đổi các tính năng sau:

1. **Frontend: Đề xuất khoảng thời gian mặc định khi chọn Full Search (Full Diff)**
   - Khi người dùng chọn chế độ "Full Search (Full Diff)" hoặc "Deep Check" trong modal xác nhận đối soát (`ConfirmDestructiveModal.tsx`), hệ thống phải tự động đề xuất/điền khoảng thời gian 30 ngày gần nhất (từ `dayjs().subtract(30, 'day')` đến `dayjs()`).
   - Nếu người dùng chuyển về chế độ "Lookback Mode", khoảng thời gian tùy chọn sẽ được reset về `null` để tránh gửi nhầm dữ liệu.

2. **Backend & Frontend: Hiển thị nút "Chữa lành" (Heal) dựa trên trạng thái heal-needed**
   - Hiện tại, nút "Chữa lành" trên UI (`DataIntegrity.tsx` và `ReconPipelineGrid.tsx`) chỉ hiển thị hoặc được enable khi status của bản ghi thuộc các giá trị cụ thể như `drift`, `dest_missing`, hoặc `warning`.
   - Tuy nhiên, trạng thái không khớp (cần chữa lành) có thể xuất hiện từ cả smoke check (đối soát tổng thể active count của 3 lớp Source/Shadow/Master) và recon check (đối soát chi tiết trong lookback window).
   - Backend cần tính toán và trả về một trường flag rõ ràng (ví dụ: `heal_needed: boolean`) biểu thị bản ghi này có cần chữa lành hay không bằng cách kiểm tra toàn bộ kết quả của smoke check và recon check.
   - Frontend sẽ sử dụng trường `heal_needed` này để quyết định việc hiển thị và active nút "Chữa lành" một cách chính xác.
