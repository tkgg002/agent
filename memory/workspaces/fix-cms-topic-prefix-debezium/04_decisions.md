# 04 - Architectural Decision Records (ADRs)

## ADR-001: Chuẩn hóa Topic Prefix cho Debezium & SFTP tại CMS Web
- **Bối cảnh:** Debezium tự động nối namespace `{database}.{collection}` vào sau `topic.prefix`. Việc frontend tự ghép `connector_name` vào `topic.prefix` làm topic bị lặp từ (duplicate segment). Ngược lại, plugin `kafka-connect-fs` (SFTP) yêu cầu topic tĩnh mang tên connector.
- **Quyết định:**
  1. **SFTP:** Giữ nguyên quy tắc tự sinh `${TOPIC_PREFIX_SFTP}.${slugify(connector_name)}` (ví dụ `cdc.sftp.my_connector`).
  2. **MongoDB, PostgreSQL, MySQL:** Tự động điền base prefix từ cấu hình ENV (`TOPIC_PREFIX_MONGODB` = `cdc.goopay`, `TOPIC_PREFIX_POSTGRESQL` = `cdc.gpay`, `TOPIC_PREFIX_MYSQL` = `cdc.mariadb`).
  3. **Khóa cứng giao diện (UI Locked):** Toàn bộ các loại database đều bị khóa mờ trường `Topic Prefix` (`<Input disabled />`), không cho phép người dùng sửa tay để đảm bảo tính nhất quán của hệ thống.
  4. **Phạm vi trì hoãn:** Bài toán phân tách đa cụm (Multi-cluster collision khi nhiều Mongo cùng tên DB/Collection) tạm thời chưa triển khai ở phase này.
