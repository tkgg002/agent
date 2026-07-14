# Yêu cầu Kỹ thuật - Khắc phục Hồi quy Đối soát Smoke

## Phạm vi (Scope)
Khắc phục các bài kiểm thử bị lỗi sau khi thực hiện tối ưu hóa hiệu năng đối soát smoke check, đồng thời tái kích hoạt các luồng kiểm tra lookback (phát hiện lệch cụ thể theo bucket giờ) cho cả hai Segment A và Segment B.

## Yêu cầu Chi tiết
1. **Khắc phục `TestBuildCastExpr`:**
   - Cập nhật biểu thức kiểm tra chuỗi cho kiểu dữ liệu `integer` và `boolean` trong test để phù hợp với hàm `BuildCastExpr` hiện tại (bọc `NULLIF` và `::NUMERIC`).
2. **Khắc phục `TestHashWindowDriftDetection`:**
   - Điều chỉnh mức độ drift trong test từ `1ms` thành `1000ms` (1 giây) vì hàm băm `hashIDPlusTsMs` thực hiện làm tròn thời gian về hàng giây nhằm tránh false-positives từ các nguồn dữ liệu không đồng nhất.
3. **Tối ưu hóa & Tái kích hoạt Lookback Checks:**
   - Tái kích hoạt `runLookbackCheckA` trong `RunTotalOnlyA`.
   - Tái kích hoạt `runLookbackCheckB` trong `RunTotalOnlyB`.
   - Để tránh việc `runLookbackCheckA` gọi lại `pickScanRangeWithLag` (làm phát sinh thêm 2 truy vấn DB Max Window Timestamp chậm), truyền trực tiếp `lo, hi, srcTS, dstTS` từ `RunTotalOnlyA` vào `runLookbackCheckA`.
