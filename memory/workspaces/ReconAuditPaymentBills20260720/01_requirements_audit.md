# 01 — Yêu cầu Audit: Recon payment_bills (2h Window)

> Tạo: 2026-07-20T10:16:54+07:00 | Task: Hotfix/Analysis

## Phạm vi

Audit toàn bộ luồng Recon Tier A (source ↔ shadow) cho bảng `payment_bills`
đang chạy với cửa sổ 2h (Hot Mode) trên môi trường **production**.

## Yêu cầu cụ thể

1. Phân tích trace log từ phiên recon thực tế (~90s, 1,952 → 1,952, diff=0)
2. Xác định tại sao có nhiều `drift_drill_down` (8 lần × 5s) mặc dù kết quả khớp
3. Tìm root cause hiệu năng chậm (90s cho 1,952 record là bất thường)
4. Liệt kê các vấn đề cần fix theo thứ tự ưu tiên
5. Gợi ý action items cụ thể

## Dữ liệu đầu vào

- Trace log phiên recon thực tế (production)
- Schema shadow table: `_id Bigint`, `lastUpdatedAt TIMESTAMP`
- Source: MongoDB collection `payment_bills`
- Cấu hình recon: Hot mode = 2h lookback, WindowSize = 15min
- Kết quả cuối: `1,952 → 1,952 (diff=0)` trong ~90s

## Definition of Done

- [ ] Xác định được root cause hiệu năng
- [ ] Liệt kê đủ P1/P2/P3/P4 issues với evidence từ code
- [ ] Đề xuất fix cụ thể, có thể thực hiện ngay
- [ ] Lưu report vật lý vào workspace
