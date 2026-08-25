# 13 - Root Cause & Technical Analysis

## 1. Phân tích nguyên nhân gốc rễ (Root Cause Analysis)
Khi tạo connector từ CMS Web, lập trình viên trước đây đã sử dụng template string:
`${TOPIC_PREFIX_MONGODB}.${slugifyForShadow(connectorName)}`
với giả định rằng `topic.prefix` cần chứa tên connector để phân biệt.

Tuy nhiên:
- Debezium framework có cơ chế tự động ghép tên Database và Collection vào sau `topic.prefix`:
  `{topic.prefix}.{database}.{collection}`
- Vì tên connector thường được đặt trùng hoặc gần giống tên database (ví dụ connector `payment_service` đọc DB `payment-service`), việc ghép này tạo ra topic 5 thành phần: `cdc.goopay.payment_service.payment-service.payments`.
- Topic chuẩn mà hệ thống và consumer mong đợi là: `cdc.goopay.payment-service.payments`.

## 2. Rủi ro va chạm (Collision Risk)
Nếu có 2 cluster khác nhau cùng chứa DB `payment-service` và collection `payments`:
- Nếu cả 2 đều dùng `topic.prefix = cdc.goopay`, chúng sẽ cùng ghi vào 1 topic Kafka `cdc.goopay.payment-service.payments`.
- Giải pháp đúng đắn là mở khóa ô nhập liệu `Topic Prefix` trên CMS, để mặc định là `cdc.goopay` nhưng cho phép người dùng chủ động điều chỉnh thành `cdc.goopay.cluster2` khi có nhu cầu.
