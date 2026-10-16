# Yêu cầu kỹ thuật: NATS Monitor UI (Specs)

## 1. Yêu cầu Chức năng (Functional Requirements)
- **FR1 (Server Overview):** Hiển thị trạng thái máy chủ NATS: Phiên bản, Uptime, Cluster ID, tổng số Active Connections, In/Out Bytes, In/Out Msgs, JetStream Memory & Storage usage.
- **FR2 (Streams Management & Monitoring):**
  - Hiển thị danh sách toàn bộ JetStream streams (`CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `CDC_EVENTS`, `SCHEMA_CONFIG`...).
  - Với mỗi stream: Tên, Subjects đã bind, tổng số Messages lưu trữ, Dung lượng lưu trữ (KB/MB), First Seq, Last Seq, số lượng Consumers.
  - Hỗ trợ xem chi tiết Consumers của từng stream (tên consumer, ack pending, redelivery count, waiting count).
- **FR3 (Active Connections):**
  - Hiển thị danh sách các kết nối client hiện tại: CID, Tên client / Authorized User (`cdc_worker`, `cms_service`, `debezium`), IP, Port, Số Subscriptions, In/Out bytes.
- **FR4 (Auto-refresh & Manual Refresh):**
  - Cho phép người dùng bật/tắt hoặc chọn chu kỳ tự động refresh (5s, 15s, 30s) và nút Refresh thủ công.

## 2. Yêu cầu Phi chức năng (Non-Functional Requirements)
- **NFR1 (Zero Impact on NATS Performance):** Chỉ query cổng HTTP monitor `:18222` với timeout ngắn (2s), không can thiệp luồng publish/subscribe TCP.
- **NFR2 (Error Resilience):** Khi NATS hoặc cổng HTTP monitor tạm thời không truy cập được, backend trả về mã 503 hoặc JSON thể hiện trạng thái `down`/`disconnected` kèm thông báo lỗi thân thiện, không làm sập ứng dụng.
- **NFR3 (Security & Permissions):** Endpoint backend đi qua nhóm route `/api/system/nats/*` được bảo vệ bởi middleware xác thực của CMS.
