# Context: Sửa lỗi healSegmentA/healSegmentB lặp lại do lấy stale report

## Hiện tượng
- Khi tiến hành heal Segment A (`healSegmentA`) hoặc Segment B (`healSegmentB`), hệ thống truy vấn báo cáo đối soát mới nhất (`reportPtr`).
- Nếu báo cáo mới nhất có `HealedAt == nil` (chưa được đánh dấu đã sửa đổi), hệ thống sẽ tái sử dụng lại báo cáo này để tiến hành heal mà không chạy lại tiến trình đối soát (`RunTier2` hoặc `RunSegmentB`).
- Tuy nhiên, nếu báo cáo đó đã quá cũ (stale report) nhưng chưa được heal, thông tin về danh sách ID bị lệch có thể không còn chính xác (dữ liệu đã tự động đồng bộ hoặc có thêm các lệch mới). Việc sử dụng báo cáo cũ này dẫn đến gửi lặp đi lặp lại các tín hiệu heal (debezium signal hoặc transmute cmd) cho các ID không thực sự bị lệch nữa, gây tốn tài nguyên hệ thống và Kafka.

## Nguyên nhân
- Thiếu kiểm tra thời gian hết hạn (expiration/stale check) của báo cáo đối soát cũ trước khi quyết định tái sử dụng báo cáo có `HealedAt == nil`.

## Giải pháp
- Định nghĩa ngưỡng thời gian tối đa để coi một báo cáo đối soát là hợp lệ (ví dụ: 5 phút).
- Nếu báo cáo mới nhất có `HealedAt == nil` nhưng thời gian đối soát (`CheckedAt`) đã vượt quá ngưỡng này, hệ thống phải coi báo cáo đó đã cũ (stale) và tự động chạy lại tiến trình đối soát để thu thập danh sách lệch mới nhất.
