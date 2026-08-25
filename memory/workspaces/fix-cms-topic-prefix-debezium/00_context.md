# 00 - Context: Fix Debezium Topic Prefix Duplicate in CMS Web

## 1. Bối cảnh
Hệ thống CDC Data Hub sử dụng Debezium Kafka Connectors (MongoDB, PostgreSQL, MySQL) và `kafka-connect-fs` (SFTP) để đẩy dữ liệu từ các nguồn vào Kafka broker.
Tại frontend `cdc-cms-web` (`src/pages/SourceConnectors.tsx`), khi khởi tạo hoặc parse thông tin kết nối, hệ thống tự động gán `topicPrefix` bằng cách ghép thêm `connector_name` (ví dụ `cdc.goopay.payment_service`).

## 2. Vấn đề phát sinh
Debezium MongoDB Connector có quy ước đặt tên topic mặc định là:
`{topic.prefix}.{database}.{collection}`
Khi `topic.prefix` bị gán thừa `connector_name`, Debezium sinh ra topic có dạng:
`cdc.goopay.payment_service.payment-service.payments` (5 segments thay vì 4 segments chuẩn).
Điều này dẫn đến việc lệch tên topic, làm Consumer gặp khó khăn trong việc khớp topic chuẩn hoặc gây trùng lặp tên vô lý. Đồng thời, form input `topicPrefix` bị `disabled` khiến người dùng không thể can thiệp khi có trường hợp nhiều connector trùng DB/Collection name.

## 3. Phạm vi
- File tác động: `cdc-cms-web/src/pages/SourceConnectors.tsx`
- Tương thích: Giữ nguyên cơ chế tự sinh prefix cho SFTP (`kafka-connect-fs`), chỉ sửa cho các connector Debezium (MongoDB, PostgreSQL, MySQL).
