# 01 - Requirements: Chuẩn hóa Topic Prefix cho Debezium Connectors

## 1. Yêu cầu chức năng (Functional Specs)
- **REQ-1:** Khi parse cấu hình MongoDB connector hoặc auto-fill trên Form tạo mới, `topicPrefix` phải lấy prefix gốc từ ENV (`TOPIC_PREFIX_MONGODB` = `cdc.goopay`), KHÔNG tự động nối thêm `.${connector_name}`.
- **REQ-2:** Khi parse cấu hình PostgreSQL/MySQL connector hoặc auto-fill trên Form tạo mới, `topicPrefix` phải lấy prefix gốc từ ENV (`TOPIC_PREFIX_BY_DB[dbKind]`), KHÔNG tự động nối thêm `.${connector_name}`.
- **REQ-3:** Giữ nguyên quy tắc cho SFTP (`kafka-connect-fs`): tiếp tục tự động nối `.${connector_name}` vì plugin này yêu cầu topic định danh theo connector.
- **REQ-4:** Mở khóa (enable) trường nhập liệu `Topic Prefix` trên giao diện cho MongoDB/PostgreSQL/MySQL (`disabled={dbKind === 'sftp'}`) để người dùng có thể tự tùy chỉnh trong trường hợp đặc biệt (nhiều connector kết nối vào cùng 1 DB & Collection).
- **REQ-5:** Bổ sung Tooltip giải thích rõ quy ước đặt tên topic của Debezium và cách xử lý va chạm topic name.

## 2. Tiêu chuẩn nghiệm thu (Definition of Done)
- [x] TypeScript build pass 100% không có lỗi type.
- [x] Form tạo mới MongoDB hiển thị mặc định `cdc.goopay` và cho phép chỉnh sửa.
- [x] Form tạo mới SFTP hiển thị `cdc.sftp.<connector_name>` và bị disabled.
- [x] Không ảnh hưởng tới các cấu hình Debezium khác (`key.converter`, `value.converter`, `signal.kafka.topic`).
