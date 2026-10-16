# 09 — Tasks Solution: Chi tiết Kỹ thuật Single Best Approach (PROPOSE ONLY)

> **Trạng thái**: Đề xuất kỹ thuật hoàn chỉnh (Đã kèm Code Demo Unit Test & Phản Biện Chống Suy Diễn). Chờ User `APPROVE` trước khi uỷ quyền Muscle thực thi.

---

## I. TỔNG HỢP KIẾN TRÚC GIẢI PHÁP TỐI ƯU DUY NHẤT

Giải pháp mang tên: **Safe Watermark Offset Commit with Bounded In-Memory Retry Queue & Backpressure**.

### Sơ đồ luồng xử lý mới (New Execution Flow)
```
[Kafka Broker]
      │
      ▼ (Fetch message)
[kafka_consumer.go] ── Msg ──► [event_handler.go] ── Enqueue RAM ──► (Chỉ trả về PENDING, không tăng metric success!)
                                         │
                                         ▼ (Ticker / Batch full)
[batch_buffer.go: FlushWithContext] ◄── [flushMu Lock: serialize toàn bộ tiến trình flush]
      │
      ├───► Gọi batchUpsert(ctx, recs) trả về []RecordExecutionResult
      │         │
      │         ├─► Chunk Tx OK: Đánh dấu tất cả là StatusPersisted
      │         └─► Chunk Tx Fail: Fallback tuần tự
      │                   ├─► OK: StatusPersisted
      │                   ├─► Non-retryable: writeDLQ OK -> StatusDLQ; fail -> StatusFailed
      │                   └─► Retryable (bao gồm 53300, 40P01, 57P01...): Đánh dấu StatusFailed (KHÔNG RETURN sớm!)
      │
      ├───► CalculateSafeCommitOffsets(results)
      │         └─► Với mỗi partition: Commit SafeOffset = (minFailedOffset - 1)
      │
      ├───► Requeue các bản ghi có StatusFailed (hoặc offset > SafeOffset) vào RetryBuffer
      │         ├─► retryCount++
      │         ├─► Nếu retryCount > 5 -> Buộc ghi DLQ khẩn cấp / Spillover
      │         └─► Nếu RetryBuffer >= pauseThreshold (5000) -> Kích hoạt Backpressure (pause Kafka fetch)
      │
      └───► commitOffsets(SafeOffsets) lên Kafka Broker
```

---

## II. DANH MỤC FILE & CÁC THAY ĐỔI CẦN THỰC THI (ACTION ITEMS CHO MUSCLE)

| STT | File cần sửa | Chức năng thay đổi |
|---|---|---|
| 1 | `internal/handler/shadow/batch_buffer.go` | - Thêm `RecordStatus`, `RecordExecutionResult`.<br>- Refactor `batchUpsert` trả về `([]RecordExecutionResult, error)`.<br>- Trong fallback, không `return` ngay khi gặp retryable error mà gắn cờ `StatusFailed`.<br>- Thêm `isRetryableDBError`: bổ sung SQLSTATE `53300`, `53200`, `53400`, `57014`.<br>- Thêm `flushMu sync.Mutex` cho `FlushWithContext`. |
| 2 | `internal/handler/shadow/watermark_tracker.go` *(File mới)* | - Triển khai hàm `CalculateSafeCommitOffsets(results)`.<br>- Triển khai `RetryBuffer` với logic Exponential Backoff + Jitter và Backpressure flag. |
| 3 | `internal/handler/shadow/kafka_consumer.go` | - Tích hợp cờ Backpressure: Nếu `batchBuffer.IsPaused()` -> `time.Sleep(50ms)` hoặc pause fetch.<br>- Khi nhận sự kiện Rebalance Revoke: Gọi `batchBuffer.ClearPartitionRetry(tp)`.<br>- Sửa silent drop tại dòng 631 (`nil_after_data`): Thay vì `return 0, nil`, ghi vào DLQ. |
| 4 | `internal/handler/shadow/event_handler.go` | - Sửa silent drop tại dòng 343 (`missing_pk`): Đẩy vào DLQ thay vì `return 0, nil`.<br>- Dòng 460–462: Không tăng metric processed success khi mới chỉ đưa vào RAM buffer. |
| 5 | `config/config-production.yml` | - Khuyến nghị hạ `dbPool.maxOpenConn` từ `100` xuống `25` (nếu chạy 4–6 pods) để tổng kết nối không vượt quá `max_connections` của PostgreSQL. |

---

## III. DEFINITION OF DONE & TIÊU CHUẨN KIỂM ĐỊNH (GATES G1–G8)

1. **Gate G2 (Reproduce trước khi Fix - Red -> Green)**:
   - Dựng test mô phỏng lỗi DB giữa chừng (như đã demo tại `scratch/demo_safe_offset_commit_test.go`).
   - Chứng minh: Code cũ commit offset 14 khi offset 12 fail (làm mất record 12). Code mới chỉ commit offset 11.
2. **Gate G3 (Test thật)**:
   - Viết Integration test thật kết nối PostgreSQL testcontainer, ngắt kết nối giả lập lỗi 53300 và deadlock.
3. **Gate G6 (Output Correctness)**:
   - Đảm bảo `Count(Source Records) == Count(Shadow Records)` sau khi chu kỳ retry hoàn tất, không bị thiếu 1 bản ghi nào.
