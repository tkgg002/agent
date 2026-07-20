# Yêu cầu Chi tiết: Cập nhật biến Total/Active sạch và mốc CheckedAt trong Smoke Recon Segment A

## 1. Bối cảnh
Người dùng muốn cập nhật hiển thị tổng số lượng bản ghi (Total/Active) của Source và Shadow trong Smoke Result chặng A (`source_shadow`) về giá trị đã làm sạch nhiễu của cửa sổ trễ đồng bộ (120s gần nhất). Đồng thời, mốc thời gian kiểm tra `CheckedAt` cần được ghi nhận lùi 120s (tức là `fromTime`).

## 2. Phạm vi Yêu cầu
- Định nghĩa biến `dstTotalClean` trong `RunTotalOnlyA`:
  - `dstTotalClean = dstTotal - dstRecentTotal` (nếu >= 0, ngược lại = 0).
- Cập nhật struct `recon.SmokeResult` trong `RunTotalOnlyA` sử dụng các giá trị làm sạch:
  - `SourceTotal = srcEstClean`
  - `SourceActive = srcEstClean`
  - `ShadowTotal = dstTotalClean`
  - `ShadowActive = dstActiveClean`
  - `CheckedAt = fromTime` (mốc thời gian 120s lùi)

## 3. Definition of Done (DoD)
- [ ] Định nghĩa thành công `dstTotalClean` trong `RunTotalOnlyA`.
- [ ] Gán đúng các giá trị `srcEstClean`, `dstTotalClean`, `dstActiveClean` và `fromTime` vào `SmokeResult`.
- [ ] Dịch (compile) thành công service `centralized-data-service`.
- [ ] Chạy linter quy trình `verify_governance.py` báo PASS.
