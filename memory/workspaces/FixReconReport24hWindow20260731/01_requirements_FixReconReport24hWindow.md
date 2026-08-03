# Yêu cầu Tối ưu hóa SLOW SQL Reconciliation Report (FixReconReport24hWindow)

## 1. Bối cảnh & Hiện trạng
Sau khi áp dụng window 7 ngày (`WHERE checked_at >= NOW() - INTERVAL '7 days'`), thời gian phản hồi của API `/api/reconciliation/report` đã giảm đáng kể từ **1.648s xuống 483ms**.
Tuy nhiên, mức 483ms vẫn vượt ngưỡng cảnh báo SLOW SQL (>= 200ms).

- **Nguyên nhân chính:**
  Bảng `cdc_system.cdc_recon_smoke_result` ghi log smoke test liên tục (mỗi vài chục giây cho 15 bảng). Trong vòng 7 ngày, số lượng bản ghi lên tới gần 1.000.000 dòng.
  Mệnh đề CTE `smoke_latest` thực hiện `DISTINCT ON` với 6 biểu thức `COALESCE` phức tạp trên ~1 triệu dòng khiến CPU phải thực hiện phép Sort trên memory/disk mất 483ms để chỉ trả về đúng **15 dòng kết quả**!

## 2. Mục tiêu (Definition of Done)
- [ ] Rút ngắn cửa sổ thời gian khoanh vùng smoke test từ 7 ngày xuống **24 giờ** (`WHERE checked_at >= NOW() - INTERVAL '24 hours'`) hoặc 48 giờ. Vì smoke check chạy định kỳ hàng phút, kết quả mới nhất của tất cả 15 bảng chắc chắn nằm trong 24 giờ gần đây. Việc này giúp giảm số lượng bản ghi phải Sort từ ~1.000.000 dòng xuống chỉ còn ~100.000 dòng (giảm 85%+ khối lượng xử lý).
- [ ] Đạt mục tiêu thời gian phản hồi API `/api/reconciliation/report` xuống **< 50ms**.
- [ ] Giữ nguyên 100% wire contract dữ liệu trả về cho Frontend.
