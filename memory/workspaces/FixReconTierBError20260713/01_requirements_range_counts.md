# Yêu cầu: Điều chỉnh chỉ số đối soát Segment B theo dải thời gian

## 1. Yêu cầu chi tiết
- **Loại bỏ TotalSourceCount và TotalDestCount:**
  - Không truy vấn và không gán hai trường `TotalSourceCount` và `TotalDestCount` trong `ReconciliationReport` ở cả Tier A và Tier B (ở Tier A đã tắt sẵn, cần tắt hoàn toàn ở Tier B).
  - Loại bỏ hoàn toàn việc gọi hàm `runCountCheckB` (truy vấn `COUNT(*)` toàn bảng) để cải thiện hiệu năng đối soát.
- **Cập nhật SourceCount và DestCount ở Segment B:**
  - Hai chỉ số `SourceCount` và `DestCount` trong báo cáo đối soát Segment B (`RunHashWindowCheckB` và `RunDeepCheckB`) phải phản ánh số lượng record nằm trong **khoảng thời gian được quét (watermark range)** chứ không phải tổng số dòng Active toàn cục.
  - Cụ thể: 
    - Với `RunHashWindowCheckB`: Trả về tổng số dòng quét được qua các window (`totalShadow` và `totalMaster`).
    - Với `RunDeepCheckB`: Trả về tổng số dòng quét qua các bucket (`totalShadow` và `totalMaster`).

## 2. Tiêu chí hoàn thành (Definition of Done)
- Loại bỏ hoàn toàn `runCountCheckB` và các truy vấn `COUNT(*)` toàn bảng trong Tier B.
- Gán đúng `SourceCount` và `DestCount` theo tổng số lượng dòng quét trong dải thời gian.
- Biên dịch dự án thành công không có lỗi.
- Linter quy trình governance chạy qua (PASS).
