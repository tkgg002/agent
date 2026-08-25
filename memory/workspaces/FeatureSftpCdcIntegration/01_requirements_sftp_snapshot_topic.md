# Yêu cầu nghiệp vụ: Trì hoãn tạo Kafka Topic cho nguồn SFTP sang nút Snapshot

Tài liệu đặc tả yêu cầu nghiệp vụ điều chỉnh cách thức đồng bộ dữ liệu SFTP.

---

## 1. Yêu cầu chi tiết

- **Tạo Connection (Bước 1):** Khi người dùng tạo connection SFTP, hệ thống khởi tạo connector trên Kafka Connect ngay lập tức (giữ nguyên luồng chuẩn của Mongo/Postgres). Tuy nhiên, **KHÔNG** tự động tạo Kafka topic tại bước này.
- **Tạo Table Shadow & Mapping (Bước 2 + 3):** Hoạt động bình thường offline.
- **Khởi chạy đồng bộ (Kích hoạt):** 
  - Trạng thái `is_active` của Table Registry / Shadow Binding được toggle bình thường trên UI (chỉ cập nhật DB, không can thiệp Kafka/Kafka Connect).
  - Khi người dùng nhấn nút **"Snapshot"** đối với nguồn SFTP, Backend nhận yêu cầu và tự động tạo Kafka Topic (`autoCreateKafkaTopic`).
  - Khi topic được tạo, connector (đang chạy từ Bước 1) sẽ tự động nhận diện và đẩy dữ liệu file SFTP lên Kafka. Worker bắt đầu tiêu thụ từ offset 0 sạch sẽ.
