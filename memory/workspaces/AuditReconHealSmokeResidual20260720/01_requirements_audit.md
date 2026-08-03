# Yêu Cầu Audit: Recon Heal Residual Smoke (Prod)
**Ngày:** 2026-07-20
**Segment:** shadow ↔ master (Segment B)

## Triệu Chứng
1. Check-heal 7 ngày phát hiện master thiếu 5 record (lag 3.3d)
2. Click "Heal" → báo thành công ("healed" / "dispatched")
3. Recon smoke lại vẫn báo lệch (diff != 0)

## Yêu Cầu Audit
- Truy vết luồng heal segment B end-to-end
- Xác định tại sao smoke vẫn báo lệch sau khi heal thành công
- Tìm root cause + đề xuất fix
