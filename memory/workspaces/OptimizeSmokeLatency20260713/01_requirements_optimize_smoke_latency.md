# Yêu cầu Tối ưu hóa Latency phát hiện Drift trong Smoke Check

Hệ thống đối soát `Recon Engine` thực hiện Smoke Check định kỳ để kiểm tra chênh lệch tổng số lượng bản ghi giữa Source, Shadow và Master DB ở độ phức tạp O(1). 

## Hiện trạng & Vấn đề
- Khi phát hiện drift (diff != 0), hệ thống kích hoạt `runLookbackCheckA` và `runLookbackCheckB` để tìm các bucket (giờ) bị lệch trong lịch sử.
- Hiện tại, việc phát hiện drift và quét bucket check này tiêu tốn **11 giây** cho mỗi lần xảy ra chênh lệch.
- Nguyên nhân:
  1. `runLookbackCheckA` gọi `pickScanRangeWithLag`, hàm này sử dụng `effectiveLookback(ctx)` để xác định mốc thời gian bắt đầu scan. Do lỗi logic trong `effectiveLookback`, hàm luôn trả về `WindowLookback` mặc định là 7 ngày kể cả ở chế độ `hot` (thường trực).
  2. `runLookbackCheckB` được hardcode cứng 7 ngày lookback (`upper.Add(-7 * 24 * time.Hour)`).
  3. Việc thực hiện `BucketCounts` (MongoDB aggregate và Postgres query) trên phạm vi 7 ngày (168 giờ/buckets) của các bảng lớn rất chậm do phải quét lượng lớn dữ liệu hoặc thiếu index tối ưu.

## Yêu cầu
1. **Sửa lỗi logic `effectiveLookback`**:
   - Nếu `RunMode == "cold"`, sử dụng `WindowLookback` (7 ngày).
   - Nếu `RunMode == "hot"` hoặc `""` (mặc định), sử dụng `HotWindowLookback` (mặc định là 2 giờ).
2. **Cập nhật `runLookbackCheckB`**:
   - Thay thế hardcode 7 ngày bằng `rc.effectiveLookback(ctx)` để đồng bộ với cơ chế lookback động của phân đoạn A.
3. **Phân tích hiệu năng câu lệnh `BucketCounts`**:
   - Xác định xem các trường thời gian (`updated_at` hoặc tương đương) đã được đánh index đầy đủ ở MongoDB và Postgres hay chưa.
4. **Bảo toàn tính đúng đắn**:
   - Đảm bảo các unit test (`recon_tier_a_test.go`, v.v.) vẫn hoạt động chính xác và không bị regression.
5. **Chứng minh hiệu quả**:
   - Chạy test thật / log kết quả so sánh trước và sau khi tối ưu.
