# Yêu cầu: Khắc phục lỗi lệch pha đối soát cho các bảng có dữ liệu ghi liên tục

## Bối cảnh
Khi đối soát (reconciliation) chạy trên các bảng có dữ liệu ghi liên tục (realtime inserts/updates), do luôn có độ trễ đồng bộ (replication lag / CDC lag) nhất định giữa MongoDB (source) và Postgres shadow (destination), việc đối soát dữ liệu sát mốc thời gian hiện tại (`now`) sẽ dẫn đến các cảnh báo lệch pha giả (false-positive drift).

## Yêu cầu chi tiết
1. **Áp dụng mốc chặn trên thời gian (Upper Bound / Freeze Margin)**:
   - Mọi cơ chế đối soát (bao gồm đối soát theo cửa sổ thời gian, đối soát toàn bộ ID, dọn dẹp orphan, và đối soát fingerprint qua bucket hash) đều PHẢI giới hạn dữ liệu đối soát nhỏ hơn mốc chặn trên: `upper = now - lag_time`.
   - `lag_time` được tính động dựa trên replication lag thực tế (`adaptiveFreeze`), hoặc tối thiểu là một mốc an toàn (ví dụ: `WindowFreezeMargin` mặc định 5 phút, hoặc cấu hình mốc cách đây 10 phút).
2. **Các cơ chế cần chỉnh sửa**:
   - `FullIDDiffMissingFromShadow`: Lọc shadow IDs và stream source IDs có timestamp < `upper`.
   - `RunOrphanPrune`: Lọc shadow IDs và stream source IDs có timestamp < `upper`.
   - `RunDeepCheck` (Bucket Hash): Cập nhật `BucketHash` của Mongo và Postgres để lọc dữ liệu < `upper`.
3. **Đảm bảo hiệu năng và tính đúng đắn**:
   - Việc lọc dữ liệu trong MongoDB stream không được gây ra lỗi sort in-memory (RAM limit) khi keyset pagination chạy trên các bảng lớn.
   - Các unit test và integration test cần được cập nhật và chạy pass.
