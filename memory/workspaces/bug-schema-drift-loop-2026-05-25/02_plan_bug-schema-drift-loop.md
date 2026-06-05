# Roadmap Kế hoạch (High-level Plan)

## English
1. **Fix Schema Inspector Drift Loop (P1)**: Prevent fallback to `public` schema in `ResolveTargetRoute`. If schema is unresolvable, return an error and skip drift check instead of caching an empty schema and infinitely inserting `pending_fields`.
2. **Enable Batch DB Inserts (P2)**: Since the user approved option 1, we will change `WriteRecordSync` to `Add(record)` in `event_handler.go` to re-enable batch buffer and eliminate the SLOW SQL (1 row/insert) issue.
3. **Reset Kafka Offset (P3)**: Reset the Kafka consumer group offset to stop replaying 10-month-old data and avoid constant `rows:0` hash collisions.

## Tiếng Việt
1. **Sửa lỗi vòng lặp Schema Drift (P1)**: Ngăn chặn fallback về schema `public` trong `ResolveTargetRoute`. Nếu không tìm thấy schema, trả về lỗi và bỏ qua bước kiểm tra drift thay vì cache một mảng rỗng và liên tục báo cáo các trường mới vào `pending_fields`.
2. **Kích hoạt gom nhóm Insert DB (P2)**: Do người dùng đã phê duyệt Option 1, tiến hành thay đổi từ `WriteRecordSync` thành `Add(record)` trong `event_handler.go` để gom batch buffer lại, giải quyết triệt để cảnh báo SLOW SQL.
3. **Reset Kafka Offset (P3)**: Đặt lại mốc offset của Consumer Group Kafka để ngừng replay dữ liệu cũ 10 tháng trước, tránh việc liên tục bị `rows:0` do trùng lặp hash.
