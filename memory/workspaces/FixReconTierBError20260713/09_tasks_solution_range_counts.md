# Giải pháp kỹ thuật - Range Counts

## 1. Loại bỏ `runCountCheckB`
Hàm `runCountCheckB` thực hiện các câu lệnh `COUNT(*)` toàn bảng rất tốn kém:
- Loại bỏ định nghĩa hàm `runCountCheckB`.
- Xóa bỏ việc gọi `runCountCheckB` trong cả `RunHashWindowCheckB` và `RunDeepCheckB`.
- Xóa bỏ logic tối ưu hóa kiểm tra count trước khi chạy hash (vì không còn count toàn cục).

## 2. Cập nhật `RunHashWindowCheckB`
- Tại các điểm trả về báo cáo đối soát:
  - Khi khớp global hash: Đã sử dụng đúng số lượng trong dải thời gian quét (`shadowGlobal.Count` / `masterGlobal.Count`).
  - Khi rơi vào window loop (có drift):
    - Đổi `SourceCount: &shadowActive` thành `SourceCount: &totalShadow`.
    - Đổi `DestCount: masterActive` thành `DestCount: totalMaster`.
    - Đổi `Diff: shadowActive - masterActive` thành `Diff: totalShadow - totalMaster`.
    - Không gán `TotalSourceCount` và `TotalDestCount`.

## 3. Cập nhật `RunDeepCheckB`
- Xóa bỏ block kiểm tra count toàn cục `if errSF == nil && errMF == nil && shadowActive == masterActive && transmuteLagMs == 0`.
- Tại điểm trả về báo cáo cuối cùng:
  - Đổi `SourceCount: &shadowActive` thành `SourceCount: &totalShadow`.
  - Đổi `DestCount: masterActive` thành `DestCount: totalMaster`.
  - Đổi `Diff: shadowActive - masterActive` thành `Diff: totalShadow - totalMaster`.
  - Không gán `TotalSourceCount` và `TotalDestCount`.
