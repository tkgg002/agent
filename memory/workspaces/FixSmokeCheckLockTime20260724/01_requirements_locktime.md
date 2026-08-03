# Requirements — Fix Smoke Check LockTime & Drift Discrepancy (Shadow ↔ Master)

## Bối cảnh (Problem Context)
Lúc 16:10:00 24/7/2026, kết quả Smoke Check ghi nhận:
1. `Source → Shadow`: KHỚP (9.37s : 4,795,901 → 4,795,901) (0)
2. `Shadow → Master`: LỆCH (61ms : 4,795,907 → 4,795,905) (-2)

Nhận xét của User: *"có vấn đề ở chỗ locktime để quét này."*

## Yêu cầu (Specifications & Requirements)
1. **Phân tích nguyên nhân gốc rễ (Root Cause Analysis)**:
   - Làm rõ tại sao `Shadow → Master` bị báo lệch 2 bản ghi (-2) trong khi `Source → Shadow` lại khớp 0 bản ghi.
   - Trace chính xác cơ chế LockTime / Scan Range / Window Subtract trong `centralized-data-service/internal/service/recon/recon_smoke.go`.
   - Xác định bất đồng bộ giữa mốc thời gian chốt `COUNT(*)` (thời điểm `scanExact`) và mốc thời gian `CountInWindow` (thời điểm `RunTotalOnlyB` tính `nowTime`).
2. **Đề xuất giải pháp kiến trúc tối ưu (Architectural Proposal)**:
   - Đảm bảo tính nhất quán của mốc thời gian `lockTime` (Cutoff LockTime / Snapshot Time) trên toàn bộ chu kỳ Smoke Check (`CheckAllUnified`).
   - Khắc phục lỗ hổng lệch trượt thời gian giữa các goroutine scan parallel.
   - Bổ sung cơ chế Fallback HashWindow cho Segment B (tương tự Segment A) hoặc đồng bộ Cutoff LockTime giữa `scanExact` và `CountInWindow`.
