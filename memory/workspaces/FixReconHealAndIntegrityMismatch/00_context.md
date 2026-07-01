# Context - FixReconHealAndIntegrityMismatch

## Problem Statement
Người dùng báo cáo 3 vấn đề chính liên quan đến Data Integrity & Healing:
1. **Lệnh Heal trả về `noop`**: Khi heal bảng `payment_bills` bị lệch 1 bản ghi giữa MongoDB source (39,991) và Postgres shadow (39,990), tiến trình heal segment A qua NATS command `cdc.cmd.recon-heal` kết thúc với trạng thái `noop` và không bù lại bản ghi thiếu.

## Current Findings & Research
- **Heal trả về `noop`**:
  - `healSegmentA` trong `recon_heal_v4.go` lấy báo cáo Tier 2 gần nhất của bảng. Do báo cáo Tier 2 gần nhất có `MissingCount = 0` (vì không tìm thấy ID bị thiếu trong destination qua `diffIDs`), heal trả về `noop`.
  - Lý do `MissingCount = 0` trong báo cáo Tier 2:
    - Trong MongoDB source, `updated_at` (kiểu ISODate) được dùng làm trường mốc thời gian để chia window.
    - Trong Postgres shadow, `_source_ts` (kiểu bigint, lưu milliseconds) được dùng làm trường mốc thời gian.
    - Sai lệch milliseconds giữa `updated_at` của MongoDB và `_source_ts` của Debezium dẫn tới việc cùng một bản ghi rơi vào hai window khác nhau ở source và destination. Điều này tạo ra hàng nghìn false drift IDs trong `stale_ids` (missing_from_src) và có thể bỏ sót bản ghi thực sự bị thiếu nếu nó nằm ngoài phạm vi quét.
    - Khoảng thời gian quét của `RunTier2` (lookback) mặc định chỉ là 2 giờ (Hot mode) hoặc 7 ngày (Cold mode). Nếu bản ghi thiếu quá cũ, `RunTier2` không quét tới mốc thời gian của nó.

