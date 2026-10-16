# 12 — Implementation Plan: Safe Watermark Offset Commit & Bounded Retry Buffer

> **Mục tiêu**: Xử lý triệt để nguyên nhân gốc rễ gây mất bản ghi CDC giữa Source và Shadow khi scale nhiều consumer.
> **Nguyên tắc**: Brain Lập Kế Hoạch & Thiết Kế Chi Tiết (Chờ APPROVE trước khi delegate Muscle triển khai code).

---

## I. MỤC TIÊU KỸ THUẬT & DEFINITION OF DONE (DoD)

1. **Cam kết Zero Data Loss (RPO = 0)**:
   - Không một bản ghi nào được phép bị Kafka commit offset nếu nó chưa được **PERSIST** an toàn vào PostgreSQL Shadow DB hoặc đã ghi thành công vào **DLQ** (`cdc_system.failed_sync_logs`).
2. **Loại bỏ Hoàn toàn Ack Sớm (No Early Ack)**:
   - Xóa bỏ việc `event_handler` trả về `rows=1` giả tạo khi mới chỉ đưa vào RAM buffer.
3. **Thuật toán Safe Watermark Offset Commit**:
   - Tính toán offset commit theo giá trị nhỏ nhất của bản ghi thất bại (`min(failedOffset) - 1`) trên từng `TopicPartition`.
4. **Cơ chế Chống OOM & Chống Treo (Backpressure & Bounded Retry)**:
   - Hàng đợi retry trong RAM có chặn trần (Bounded buffer). Khi retry vượt quá dung lượng -> Kích hoạt Backpressure tạm dừng `FetchMessage`.
5. **Đồng bộ hóa Chu trình Flush**:
   - Serialize toàn bộ tiến trình Flush và Commit bằng `flushMu` để tránh tranh chấp giữa `timerLoop`, `snapshotRunner` và `reconHeal`.

---

## II. THIẾT KẾ KIẾN TRÚC CHI TIẾT (TECHNICAL SPECIFICATIONS)

### 1. Cấu trúc Trạng thái Thực thi Từng Bản ghi (Per-Record Result)
Thay đổi chữ ký của `batchUpsert` từ:
`func (bb *BatchBuffer) batchUpsert(ctx context.Context, records []*shadow.UpsertRecord) (written int, err error)`
thành:
`func (bb *BatchBuffer) batchUpsert(ctx context.Context, records []*shadow.UpsertRecord) (results []RecordExecutionResult, err error)`

```go
type RecordStatus uint8

const (
    StatusPending   RecordStatus = iota // Chưa xử lý
    StatusPersisted                     // Đã ghi thành công vào Shadow DB
    StatusDLQ                           // Đã ghi thành công vào DLQ failed_sync_logs
    StatusFailed                        // Thất bại, cần giữ offset và retry
    StatusDropped                       // Bản ghi rác được xác nhận (tombstone hợp lệ)
)

type RecordExecutionResult struct {
    Record *shadow.UpsertRecord
    Status RecordStatus
    Err    error
}
```

### 2. Thuật toán Tính toán Safe Offset Commit (Watermark Tracker)

```go
func CalculateSafeCommitOffsets(batch []*shadow.UpsertRecord, results []RecordExecutionResult) map[TopicPartition]int64 {
    // 1. Phân nhóm kết quả theo TopicPartition
    type partitionTracking struct {
        minFailedOffset int64
        maxOffset       int64
        hasFailures     bool
    }
    
    tracking := make(map[TopicPartition]*partitionTracking)
    
    for _, res := range results {
        r := res.Record
        if r.KafkaTopic == "" || r.KafkaOffset <= 0 {
            continue
        }
        tp := TopicPartition{Topic: r.KafkaTopic, Partition: r.KafkaPartition}
        t, exists := tracking[tp]
        if !exists {
            t = &partitionTracking{
                minFailedOffset: -1,
                maxOffset:       r.KafkaOffset,
                hasFailures:     false,
            }
            tracking[tp] = t
        }
        if r.KafkaOffset > t.maxOffset {
            t.maxOffset = r.KafkaOffset
        }
        
        // Nếu bản ghi thất bại (chưa persist và chưa vào DLQ)
        if res.Status == StatusFailed {
            t.hasFailures = true
            if t.minFailedOffset == -1 || r.KafkaOffset < t.minFailedOffset {
                t.minFailedOffset = r.KafkaOffset
            }
        }
    }
    
    // 2. Tính toán offset an toàn để commit
    safeOffsets := make(map[TopicPartition]int64)
    for tp, t := range tracking {
        if !t.hasFailures {
            // Toàn bộ các bản ghi trong batch của partition này thành công
            safeOffsets[tp] = t.maxOffset
        } else {
            // Có lỗi -> Chỉ được commit đến minFailedOffset - 1
            if t.minFailedOffset > 1 {
                safeOffsets[tp] = t.minFailedOffset - 1
            }
            // Nếu minFailedOffset <= 1, không commit gì cả cho partition này!
        }
    }
    return safeOffsets
}
```

### 3. Sửa đổi Fallback trong `batchUpsert` (`batch_buffer.go`)
- Khi chunk transaction thất bại và chuyển sang tuần tự:
  - Nếu gặp `isRetryableDBError(err)`:
    - **CẤM `return` ngay làm rơi các bản ghi phía sau.**
    - Ghi nhận bản ghi hiện tại là `StatusFailed`.
    - Đánh dấu tất cả các bản ghi còn lại trong chunk và các chunk tiếp theo của bảng đó là `StatusFailed`.
    - Trả kết quả về cho `FlushWithContext` để tính toán Safe Offset.

### 4. Mở rộng Bảng tra cứu Lỗi `isRetryableDBError`
Bổ sung:
- PostgreSQL SQLSTATE `53300` (too_many_clients / connection limit exceeded).
- PostgreSQL SQLSTATE `53200` (out_of_memory).
- PostgreSQL SQLSTATE `53400` (configuration_limit_exceeded).
- PostgreSQL SQLSTATE `57014` (query_canceled / statement timeout).
- Các chuỗi text: `"too many clients"`, `"remaining connection slots"`, `"connection pool timeout"`.

### 5. Xử lý Dứt điểm Silent Drops
- `kafka_consumer.go:631` (`nil_after_data`): Thay vì `return 0, nil`, tạo payload DLQ với `errorType="nil_after_data"`.
- `event_handler.go:343` (`missing_pk`): Đẩy vào DLQ `errorType="missing_pk"`, không `return 0, nil`.

---

## III. KẾ HOẠCH TRIỂN KHAI THEO TỪNG TASK (PHASE BREAKDOWN)

- **Task 1: Core Domain & Types Refactor** (`batch_buffer.go`): Định nghĩa `RecordStatus`, `RecordExecutionResult`.
- **Task 2: Watermark Calculator Implementation**: Viết hàm `CalculateSafeCommitOffsets` độc lập, bao phủ 100% unit tests.
- **Task 3: Refactor `batchUpsert` Fallback**: Trả về slice kết quả, loại bỏ exit đột ngột khi gặp retryable error.
- **Task 4: Sửa `FlushWithContext`**: Tích hợp Watermark Calculator, serialize bằng `flushMu`, gom các bản ghi `StatusFailed` vào hàng đợi retry.
- **Task 5: Backpressure Controller**: Tích hợp cơ chế pause/resume reader khi buffer retry chạm ngưỡng trần (`maxRetryQueueSize = 5000`).
- **Task 6: Mở rộng `isRetryableDBError` & Chặn Silent Drop**.
- **Task 7: Integration Test (Red -> Green)**: Dựng kịch bản lỗi DB giả lập, kiểm chứng 0 record bị miss.
