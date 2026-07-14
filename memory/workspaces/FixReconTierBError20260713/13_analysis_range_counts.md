# Phân tích kết quả - Range Counts

Chúng ta đã tiến hành loại bỏ toàn bộ truy vấn `COUNT(*)` toàn bảng từ `runCountCheckB` và tối ưu hóa các dải đếm trong Segment B.

## Phân tích Hiệu Năng
- Trước đây: Mỗi lần chạy `RunHashWindowCheckB` hoặc `RunDeepCheckB` đều gọi `runCountCheckB` thực thi:
  1. `SELECT COUNT(*)` trên bảng Shadow
  2. `SELECT COUNT(*)` trên bảng Master
  3. `SELECT COUNT(*) WHERE _deleted = true` trên bảng Shadow
  4. `SELECT COUNT(*) WHERE _deleted = true` trên bảng Master
  - Tổng cộng 4 câu truy vấn `COUNT(*)` toàn bảng. Trên các database cỡ lớn (hàng triệu/chục triệu dòng), điều này gây ra tình trạng khóa bảng hoặc làm CPU tăng vọt, kéo dài thời gian đối soát.
- Sau khi tối ưu:
  - Loại bỏ hoàn toàn 4 câu lệnh truy vấn toàn bảng.
  - Các chỉ số `SourceCount`, `DestCount` và `Diff` giờ đây được tính dựa trên `totalShadow` và `totalMaster` (lấy qua window scan hoặc bucket scan), vốn chỉ giới hạn trong dải thời gian quét của watermark (đã được đánh index thời gian).
  - Điều này giúp giảm độ trễ của CDC recon xuống mức tối đa.
