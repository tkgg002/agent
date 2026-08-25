# 09_tasks_solution_sftp_clean_consumer_architecture.md

## Hồ sơ Thiết kế Kiến trúc Chuẩn: SFTP Topic Re-consumption qua KafkaConsumer

### 1. Phân tích Phá vỡ Kiến trúc của Giải pháp Ad-hoc cũ
- **Lỗi thiết kế (Code Smell / Cheat-code)**: Việc `SnapshotRunner` tự tạo một `kafka.Reader` độc lập rồi chạy vòng lặp thủ công đọc Kafka message đã vi phạm nguyên lý **Single Responsibility** và **Single Source of Truth** của hệ thống.
- **Tác hại**:
  - Tách rời luồng tiêu thụ khỏi `KafkaConsumer` (nơi quản lý circuit breaker, schema validator, metrics và activity log).
  - Nhân bản code xử lý message ở 2 nơi (`SnapshotRunner` và `KafkaConsumer`).

---

### 2. Thiết kế Kiến trúc Chuẩn Enterprise (Clean Architecture)

#### A. Phân định Trách nhiệm (Separation of Concerns)
1. **`KafkaConsumer` (Tầng Ingestion Engine)**:
   - Quản lý 100% vòng đời của các `kafka.Reader`.
   - Bổ sung phương thức chính thống: `ReconsumeTopic(ctx context.Context, topic string) error`.
   - Khi được gọi `ReconsumeTopic(topic)`, `KafkaConsumer` đóng reader hiện tại của topic đó, thiết lập cấu hình đọc lại từ `kafka.FirstOffset` (offset 0), và cho phép pipeline chính thống (`processMessage` -> `validator` -> `HandleRaw` -> `batchBuffer` -> DB) tiêu thụ lại toàn bộ dữ liệu.

2. **`SnapshotRunner` (Tầng Orchestrator)**:
   - Chỉ đóng vai trò điều phối và ghi nhận tiến độ (`snapshot_progress`).
   - Khi nhận lệnh Snapshot SFTP: Yêu cầu `KafkaConsumer` thực hiện `ReconsumeTopic(topic)`.
   - Tuyệt đối KHÔNG tự mở reader hay chạy vòng lặp tiêu thụ ad-hoc.

---

### 3. Kế hoạch Thực thi (Dành cho Muscle sau khi được Approve)
1. Bổ sung `ReconsumeTopic(ctx, topic)` vào `KafkaConsumer` (`internal/handler/shadow/kafka_consumer.go`).
2. Ủy quyền lệnh SFTP snapshot từ `SnapshotRunner` sang `KafkaConsumer.ReconsumeTopic`.
3. Kiểm thử biên dịch và unit test suite.
