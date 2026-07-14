# Yêu cầu cải tiến logic thời gian đối soát: Sử dụng 1 quy tắc Time Range, loại bỏ Full Diff & Lookback Options

## 1. Vấn đề hiện tại
1. Hệ thống đối soát đang tồn tại quá nhiều option phức tạp chồng chéo nhau giữa `lookback` (hot, cold, deep) và `full_diff` (quét toàn bộ).
2. Sự khác biệt giữa `hash_window` (Tier 2) và `full_diff` thực chất chỉ là khoảng thời gian quét. Việc chia làm 2 check type riêng biệt gây dư thừa mã nguồn.
3. Smoke check khi phát hiện chênh lệch (`drift`) không lưu lại vết các giờ bị lệch, gây khó khăn cho việc tra cứu.

## 2. Yêu cầu chi tiết mới
- **Quy tắc thời gian thống nhất**:
  - Loại bỏ hoàn toàn các option lookback `hot` và `cold` ở tầng backend.
  - Phía backend chỉ chạy duy nhất 1 quy tắc cấu hình khoảng thời gian đối soát: Sử dụng `WithReconTimeRange(start, end)` trong context.
  - Trên Frontend (FE):
    - Người dùng có 3 lựa chọn: `2h` (hot), `7 ngày` (cold), và `Custom` (tự chọn).
    - Khi chọn `2h` (hot): Frontend tự tính toán `StartTime = now - 2h` và `EndTime = now` rồi truyền xuống backend.
    - Khi chọn `7 ngày` (cold): Frontend tự tính toán `StartTime = now - 7d` và `EndTime = now` rồi truyền xuống backend.
    - Khi chọn `Custom`: Cho người dùng tự chọn khoảng thời gian tùy ý (với validator giới hạn tối đa 30 ngày ở cả FE và BE).
- **Loại bỏ `full_diff` (TypeReconFullDiff)**:
  - Loại bỏ hoàn toàn check type `full_diff` khỏi mã nguồn backend (chỉ giữ lại `hash_window` cho Tier 2 check).
  - Mọi yêu cầu chạy đối soát theo khoảng thời gian từ FE (kể cả 2h, 7 ngày hay custom) đều truyền `TypeRecon = "hash_window"`, kèm theo `StartTime` và `EndTime`.
- **Validator khoảng thời gian tối đa 30 ngày**:
  - Tại `validateAndEnrichContext` của check handler, bắt buộc phải có `StartTime` và `EndTime` đối với `hash_window`.
  - Thực hiện kiểm tra tính hợp lệ: `EndTime >= StartTime` và `EndTime - StartTime <= 30 ngày` (30 * 24 * 3600 * 1000 mili-giây).
- **Lưu vết `diff_time` khi Smoke Drift**:
  - Khi Smoke check Segment A hoặc B phát hiện lệch số lượng, tự động chạy hàm lookback check (`runLookbackCheckA` / `runLookbackCheckB`) trên phạm vi mặc định (7 ngày) để bóc tách các giờ bị lệch, serialize thành mảng JSON lưu vào cột `diff_time` mới của bảng `cdc_recon_smoke_result`.
  - Phía `cdc-cms-service` select `diff_time AS stale_ids` để hiển thị danh sách giờ bị lệch trực tiếp lên cột "lệch" trên Dashboard.
