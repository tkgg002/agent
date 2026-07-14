# Phân tích Kỹ thuật - Cập nhật Reconciliation UI & API Pipeline

Tài liệu này ghi nhận kết quả phân tích nguyên nhân gốc rễ và phân tích ảnh hưởng của sự thay đổi đối với luồng đối soát và giao diện điều khiển.

## 1. Vấn đề 1: Tự động đề xuất khoảng thời gian 30 ngày
- **Hiện trạng:** Khi chọn chế độ đối soát `Full Search` hoặc `Deep Check` trên modal yêu cầu đối soát Tier 2, Date Range Picker trống rỗng (`null`). Người dùng bắt buộc phải tự click và chọn ngày, tạo ra trải nghiệm sử dụng không mượt mà và dễ nhầm lẫn.
- **Giải pháp:** Khi state `checkMode` được đổi sang `full_diff` hoặc `deep`, UI sẽ tự động điền giá trị từ `dayjs().subtract(30, 'day')` đến `dayjs()`. Khi người dùng quay trở lại `lookback` mode, range picker sẽ được reset về `null` để quay lại chế độ mặc định (đối soát nhanh lookback).

## 2. Vấn đề 2: Nút "Chữa lành" không xuất hiện khi lệch counts
- **Hiện trạng:** Trước đây, nút "Chữa lành" chỉ hiển thị khi `status` của lookback window đối soát bị lệch (`drift`, `dest_missing`, `warning`). Tuy nhiên, trong thực tế:
  - Nếu dữ liệu trong lookback window (thường là 2 giờ gần nhất) hoàn toàn khớp (trạng thái `ok`), nhưng số lượng active records toàn thời gian của lớp nguồn và lớp đích bị lệch (ví dụ: do một số bản ghi cũ bị thiếu/thừa từ trước khoảng thời gian lookback).
  - Trạng thái tổng thể lúc này là không đồng bộ, nhưng nút "Chữa lành" bị ẩn vì status của lookback window vẫn là `ok`.
- **Giải pháp:**
  - Backend API làm giàu dữ liệu `/api/reconciliation/report` tính toán cờ `HealNeeded` bằng cách kiểm tra:
    - Nếu lookback window phát hiện lệch (`drift`, `dest_missing`, `warning`).
    - Hoặc nếu Smoke Check phát hiện lệch tổng số lượng bản ghi active (`srcActive != dstActive`) hoặc tổng số lượng bản ghi (`srcTotal != dstTotal`).
  - Cờ `heal_needed` được trả về trong payload API và được Frontend sử dụng để hiển thị và kích hoạt nút "Chữa lành" cho chặng tương ứng.
