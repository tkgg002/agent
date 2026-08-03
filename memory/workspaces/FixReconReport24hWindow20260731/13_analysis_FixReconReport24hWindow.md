# Phân Tích Sâu Nguyên Nhân & Giải Pháp Tối Ưu Tốc Độ Reconciliation Report

## I. Đánh Giá Kết Quả Bước 1
- API `/api/recon/backfill-source-ts/status`: Latency giảm từ **690ms -> 5.8ms** (`0.005861s`). Đã hoàn toàn tối ưu thành công!
- API `/api/reconciliation/report`: Latency giảm từ **1.648s -> 483ms** (`0.4844s`). Giảm 70% thời gian nhưng vẫn ở mức 483ms (vượt ngưỡng cảnh báo 200ms).

## II. Phân Tích Nguyên Nhân Mức Latency 483ms
- Phép `DISTINCT ON` trong CTE `smoke_latest` của `recon_read_repo_gorm.go` sử dụng 6 biểu thức `COALESCE` phức tạp:
  `ORDER BY COALESCE(shadow_schema, ''), shadow_table, COALESCE(NULLIF(master_schema, ''), ''), COALESCE(NULLIF(master_table, ''), ''), COALESCE(segment, 'source_shadow'), checked_at DESC`
- Vì smoke test ghi log kết quả liên tục vài giây/lần cho 15 bảng, cửa sổ 7 ngày chứa ~900.000 bản ghi.
- Phép Sort 6 biểu thức `COALESCE` trên ~900.000 dòng mất 483ms để trích xuất ra vỏn vẹn **15 dòng kết quả**.

## III. Giải Pháp Tối Ưu Cửa Sổ 24 Giờ (24h Window Pruning)
- Smoke test vận hành tự động liên tục. Kết quả mới nhất của tất cả 15 bảng active chắc chắn có mặt trong **24 giờ gần đây**.
- Thu hẹp `checked_at >= NOW() - INTERVAL '24 hours'` (hoặc 48 hours) giúp giảm quy mô dữ liệu cần Sort từ ~900.000 dòng xuống chỉ còn ~120.000 dòng.
- Latency sẽ giảm từ 483ms xuống **< 50ms**.
