# 09 — Tasks Solution: Phân tách Luồng Snapshot và CDC Realtime (PROPOSE ONLY)

> **Trạng thái**: Đề xuất kỹ thuật hoàn chỉnh (Đã kiểm chứng Laya + Demo Go Test PASS 100% + Phản biện chi tiết). Chờ User `APPROVE` trước khi delegate Muscle thực thi.

---

## I. TỔNG HỢP KIẾN TRÚC THAY ĐỔI

Tách biệt hoàn toàn luồng xử lý giữa Realtime CDC (Kafka Consumer) và Bulk Load (Snapshot Runner) thông qua:
1. **Isolated Processing Buffers**: Tách thành `cdcBatchBuffer` (có commit Kafka offset) và `snapshotBatchBuffer` (KHÔNG commit Kafka offset).
2. **Dynamic Concurrency Capping (Semaphore)**: Khống chế Snapshot chỉ được tối đa 25 concurrent connections, đảm bảo CDC luôn có tối thiểu 75 connections sẵn sàng.
3. **Adaptive Throttling / Lag Protection**: Snapshot tự động nhường bước (throttle delay 50ms) khi Kafka Consumer Lag > 1000.

---

## II. DANH MỤC THAY ĐỔI MÃ NGUỒN CHO MUSCLE

| STT | File cần sửa | Chi tiết thay đổi |
|---|---|---|
| 1 | `internal/server/server_setup.go` | - Khởi tạo `cdcBatchBuffer` (dành riêng cho CDC, timeout 500ms, gắn `onCommitOffsets`).<br>- Khởi tạo `snapshotBatchBuffer` (dành riêng cho Snapshot, timeout 5s, KHÔNG gắn `onCommitOffsets`).<br>- Khởi tạo `cdcEventHandler` inject vào `kafkaConsumer`.<br>- Khởi tạo `snapshotEventHandler` inject vào `snapshotRunner`.<br>- Cập nhật NATS subscriber `schema.config.reload` và `connection.config.changed` flush cache trên cả 2 buffers. |
| 2 | `internal/handler/orchestration/snapshot_runner_handler.go` | - Tích hợp cơ chế kiểm tra `metrics.ConsumerLag` trước mỗi batch flush.<br>- Nếu `ConsumerLag > 1000`: Chèn thêm `time.Sleep(50ms)` (Adaptive Throttling).<br>- Giảm chunk size ghi của snapshot xuống 200–500 rows để tránh giữ row lock lâu. |
| 3 | `internal/handler/shadow/batch_buffer.go` | - Thêm cờ `isSnapshotBuffer bool`. Nếu là snapshot buffer -> Bỏ qua hoàn toàn việc tính toán và gọi `onCommitOffsets`.<br>- Giới hạn số lượng concurrent group của snapshot xuống tối đa 4 goroutines (`g.SetLimit(4)`). |

---

## III. DEFINITION OF DONE & TIÊU CHUẨN KIỂM THỬ

1. **Gate G3 (Test thật không giao thoa)**:
   - Chạy đồng thời `SnapshotRunner` bơm 100,000 bản ghi và `KafkaConsumer` bơm 1,000 bản ghi.
   - Xác nhận: Lệnh flush của Snapshot tuyệt đối không làm nhảy hay commit sai offset của Kafka.
2. **Gate G5 (SLA Realtime)**:
   - Đo độ trễ E2E của CDC trong lúc Snapshot đang chạy bulk: Độ trễ P99 của CDC phải duy trì dưới **1.5 giây**.
