# Decisions: Sửa lỗi healSegmentA/healSegmentB lặp lại do lấy stale report

## 1. Ngưỡng thời gian Max Age của Report (5 phút)
- **Quyết định**: Sử dụng `5 * time.Minute` làm thời hạn tối đa của một bản ghi báo cáo đối soát.
- **Lý do**:
  - Đủ ngắn để tránh việc hệ thống tái sử dụng báo cáo lỗi cũ khi dữ liệu thật đã được đồng bộ tự nhiên qua CDC.
  - Đủ dài để không làm quá tải hệ thống nếu người dùng vô tình bấm trigger heal nhiều lần liên tiếp trong khoảng thời gian rất ngắn.

## 2. Chiến lược fallback khi stale
- **Quyết định**: Khi phát hiện báo cáo đã stale, hệ thống sẽ thực hiện check động (Segment B check hoặc Segment A Tier-2 check) đồng bộ trước khi quyết định gửi tập hợp ID nào đi heal.
- **Lý do**:
  - Đảm bảo tính chính xác và an toàn của hệ thống Core. Không bao giờ gửi các bản ghi đã khớp đi heal lặp lại.
  - Ngăn ngừa tình trạng trễ hoặc nghẽn hàng đợi Kafka/Debezium.
