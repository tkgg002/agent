# Phân tích kỹ thuật: Phương án rẽ nhánh nút Snapshot cho nguồn SFTP

Tài liệu phân tích kiến trúc và phân rã thiết kế.

---

## 1. Phân tích thiết kế

- **Hạn chế thay đổi Core Flow:** Tránh được việc sửa đổi logic toggle `is_active` ở tầng Registry (`update_registry.go`) và Shadow Binding (`update_shadow_binding.go`) giúp bảo toàn 100% tính toàn vẹn và độ tin cậy của luồng xử lý PostgreSQL và MongoDB CDC hiện tại.
- **Tận dụng UI Flow sẵn có:** 
  - Nút **"Snapshot"** trên giao diện Table Registry (`TableRegistry.tsx`) được hiển thị và cho phép nhấn khi `record.is_active` ở trạng thái bật (đã active).
  - Đối với cơ sở dữ liệu (Postgres, Mongo), nút này sẽ kích hoạt cơ chế đọc trực tiếp Oplog/Table để seed dữ liệu.
  - Đối với file/SFTP stream, nút này sẽ kích hoạt việc tạo Topic Kafka. Khi topic Kafka được tạo, connector (đã chạy từ bước tạo connection) sẽ tự động đẩy toàn bộ file mẫu/snapshot ban đầu vào Kafka, và Worker sẽ tiêu thụ ngay lập tức từ offset 0.
- **Bypass NATS:** Nguồn SFTP không cần một NATS queue runner để quét hay điều phối vì Kafka Connect Connector đảm nhiệm toàn bộ việc đọc/ghi tập tin. Việc rẽ nhánh tạo topic trực tiếp tại API giúp tối giản tài nguyên mạng và giảm độ trễ.
