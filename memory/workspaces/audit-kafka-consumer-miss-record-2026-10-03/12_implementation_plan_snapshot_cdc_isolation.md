# 12 — Implementation Plan: Phân tách Triệt để Pipeline Realtime CDC và Snapshot Bulk Load

> **Mục tiêu**: Tách rời hoàn toàn hàng đợi, bộ đệm ghi và tài nguyên database giữa Kafka Consumer (Ưu tiên P0) và Snapshot Runner (Nền P1/P2).

---

## I. KIẾN TRÚC MỤC TIÊU (TARGET ARCHITECTURE)

```
[Kafka Cluster]                                  [Source DB / MongoDB]
      │                                                   │
      ▼                                                   ▼
[KafkaConsumer (P0 Realtime)]                    [SnapshotRunner (P1 Bulk)]
      │                                                   │
      ▼                                                   ▼
[CDCEventHandler]                                [SnapshotEventHandler]
      │                                                   │
      ▼                                                   ▼
[CDC_BatchBuffer]                                [Snapshot_BatchBuffer]
  - batchSize: 200                                 - batchSize: 2000
  - timeout: 500ms                                 - timeout: 5s
  - with onCommitOffsets                           - NO onCommitOffsets
      │                                                   │
      ▼ (Dedicated Quota: 70%)                            ▼ (Dedicated Quota: 30%)
[CDC DB Connection Pool]                         [Snapshot DB Connection Pool]
      │                                                   │
      └─────────────────┬─────────────────────────────────┘
                        ▼
               [Shadow PostgreSQL]
```

---

## II. 3 TRỤ CỘT GIẢI PHÁP KỸ THUẬT

### Trụ cột 1: Tách biệt Bộ đệm & Event Handler (Buffer & Handler Isolation)
1. **Khởi tạo 2 BatchBuffer riêng biệt trong `server_setup.go`**:
   - `cdcBatchBuffer`: Cấu hình cho độ trễ thấp (`batchSize = 200`, `timeout = 500ms`). Gắn callback `onCommitOffsets` với thuật toán Safe Watermark Offset.
   - `snapshotBatchBuffer`: Cấu hình cho throughput cao (`batchSize = 2000`, `timeout = 5s`). **TUYỆT ĐỐI KHÔNG GẮN `onCommitOffsets`**.
2. **Khởi tạo 2 EventHandler riêng biệt**:
   - `cdcEventHandler`: Phục vụ Kafka Consumer và NATS CDC pools.
   - `snapshotEventHandler`: Chỉ phục vụ `SnapshotRunner` và `ReconHealer`.
   - Kết quả: Bản ghi snapshot không bao giờ lọt vào buffer của CDC và ngược lại.

### Trụ cột 2: Cách ly Tài nguyên Kết nối DB (Connection Pool Quota Isolation)
1. **Phân bổ Hạn ngạch (Quota Reservation)**:
   - Tổng kết nối `maxOpenConn = 100`:
     + **CDC Pool**: 70 connections (Bảo lưu ưu tiên cao nhất, luôn sẵn sàng cho giao dịch realtime).
     + **Snapshot Pool**: 30 connections (Giới hạn trần cứng - Hard Cap). Dù snapshot có muốn mở thêm cũng không thể vượt quá 30 kết nối, triệt tiêu 100% nguy cơ snapshot nuốt trọn DB gây lỗi 53300 cho CDC.
2. **Concurrency Limit độc lập**:
   - `cdcBatchBuffer`: Cho phép tối đa 16-20 concurrent table workers (`g.SetLimit(20)`).
   - `snapshotBatchBuffer`: Khống chế tối đa 2-4 concurrent table workers (`g.SetLimit(4)`).

### Trụ cột 3: Cơ chế Điều tiết Thích ứng & Nhường quyền (Adaptive Throttling / Dynamic Yield)
1. **Theo dõi Consumer Lag realtime**:
   - `SnapshotRunner` được inject interface `LagInspector` để đọc `metrics.ConsumerLag`.
2. **Nguyên tắc nhường quyền**:
   - Khi `KafkaConsumerLag <= 1000`: Snapshot chạy ở tốc độ bình thường.
   - Khi `KafkaConsumerLag > 1000` (CDC đang bị dồn ứ):
     + Snapshot tự động kích hoạt chế độ **Throttling** (chèn thêm delay `time.Sleep(50ms)` giữa các bulk flush).
     + Nếu `KafkaConsumerLag > 10000`: Snapshot tạm dừng hoàn toàn (Pause) cho đến khi CDC giải phóng hết hàng đợi, bảo vệ tuyệt đối SLA vận hành của hệ thống.

---

## III. KẾ HOẠCH TRIỂN KHAI CHO MUSCLE

- **Task 1: Khởi tạo Hai Buffer Riêng Biệt** (`internal/server/server_setup.go`).
- **Task 2: Phân tách Handler & Inject đúng Interface cho SnapshotRunner**.
- **Task 3: Triển khai Quota Pool cho DB** (`pkgs/database/multi.go` hoặc `PerSourcePool`).
- **Task 4: Thêm Hook Điều tiết Adaptive Throttling vào `SnapshotRunner`** (`snapshot_runner_handler.go`).
- **Task 5: Unit & Integration Test**: Verify không có sự giao thoa offset giữa 2 luồng.
