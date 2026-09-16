# Requirements: Fix Kafka Consumer Oplog UPDATE Event Processing

## 1. Overview
Hệ thống CDC (Centralized Data Service) sử dụng Debezium Kafka Consumer để tiêu thụ các sự kiện CDC từ MongoDB Oplog / Change Streams về DB Shadow. Qua kiểm tra thực tế, luồng Kafka Consumer đang bị bỏ sót / DROP các sự kiện UPDATE (`op == "u"`).

## 2. Root Cause Summary
1. **Connector Configuration (`cdc_system.sources`)**: Debezium Mongo Connector sử dụng `"capture.mode": "change_streams"`, khiến Debezium KHÔNG gửi kèm full document ở trường `after` đối với các sự kiện UPDATE (`op: "u"`).
2. **Worker Logic (`kafka_consumer.go` line 596-605)**: Khi `opStr == "u"` và `afterData == nil`, `kafka_consumer` đánh giá điều kiện `afterData == nil && opStr != "d"` thành `TRUE`, log cảnh báo `kafka message has no 'after' data, dropping (non-delete)` và DROP luôn tin nhắn UPDATE.
3. **Event Handler (`event_handler.go` line 308-313)**: Không có fallback parse dữ liệu cho sự kiện UPDATE khi `after` bị nil.

## 3. Scope of Fix
- **Cấu hình Connector**: Đổi `"capture.mode"` thành `"change_streams_with_full_update"` cho toàn bộ MongoDB Debezium Connectors.
- **Code Handling**: Thêm cơ chế phòng thủ (defense in depth) cho `kafka_consumer.go` và `event_handler.go` đối với sự kiện `UPDATE` (`op == "u"`).

## 4. Acceptance Criteria (Definition of Done)
- [ ] Mọi sự kiện UPDATE (`op: "u"`) phát sinh trên MongoDB nguồn đều được Debezium gửi đủ `after` full document.
- [ ] `kafka_consumer.go` không còn drop bừa bãi sự kiện UPDATE khi `afterData` được populate đầy đủ hoặc qua fallback.
- [ ] DB Shadow (`cdc_shadow`) nhận đầy đủ các câu UPDATE (`_updated_at > _created_at`) khi có mutation UPDATE từ nguồn.
- [ ] Chạy test verify end-to-end thành công.
