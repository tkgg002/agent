# 11 — Báo Cáo Thay Đổi Mã Nguồn: Fix Lỗi Miss Record & Phân Tách Pipeline Snapshot vs CDC

> **Thời gian thực hiện**: 2026-10-06  
> **Trạng thái**: Đã triển khai hoàn tất (Muscle Execution Done). 100% test pass.

---

## I. TỔNG QUAN THAY ĐỔI

Giải quyết triệt để 2 nguyên nhân cốt lõi gây thất thoát dữ liệu (miss record) khi scale Kafka consumer và chạy snapshot:
1. **Safe Watermark Offset Commit**: Chấm dứt tình trạng commit offset mù quáng khi batch bị lỗi (đặc biệt lỗi 53300 do nghẽn DB pool). Chỉ commit đến `min(failedOffset) - 1`.
2. **Buffer & Pipeline Isolation giữa Snapshot và CDC**: Tách rời hoàn toàn bộ đệm và handler xử lý giữa luồng Bulk Snapshot và luồng Realtime CDC. Snapshot chạy qua buffer riêng với concurrency cap = 4, timeout 5s, và tuyệt đối không trigger callback commit Kafka offset.
3. **Triệt tiêu Silent Drop**: Sửa lỗi nuốt lỗi tại `kafka_consumer.go:639` (`nil_after_data`) và `event_handler.go:349` (`missing_pk`). Trả lỗi tường minh để message được chuyển sang DLQ hoặc retry thay vì commit mất hút.
4. **Adaptive Lag Throttling**: Snapshot Runner tự động đo Kafka Consumer Lag realtime, chèn delay 50ms khi lag > 1000 và dừng tạm thời 500ms khi lag > 10000 để bảo vệ độ trễ realtime cho CDC.

---

## II. CHI TIẾT TỪNG FILE THAY ĐỔI

| STT | File thay đổi | Loại | Số dòng (+/-) | Mô tả chi tiết thay đổi |
|---|---|---|---|---|
| 1 | `internal/handler/shadow/watermark_tracker.go` | Tạo mới | +110 lines | Định nghĩa trạng thái bản ghi (`StatusPersisted`, `StatusDLQ`, `StatusFailed`), thuật toán `CalculateSafeCommitOffsets` bảo vệ watermark, và hàm test `IsRetryableDBErrorForTest`. |
| 2 | `internal/handler/shadow/watermark_tracker_test.go` | Tạo mới | +99 lines | Unit test cho thuật toán safe watermark: kiểm tra case tất cả pass, case fail ở giữa chặn nhảy offset, case DLQ cho phép offset tiếp tục tiến, và test mở rộng các mã lỗi retryable DB. |
| 3 | `internal/handler/shadow/snapshot_cdc_isolation_test.go` | Tạo mới | +77 lines | Unit test chứng minh Snapshot buffer tuyệt đối không commit Kafka offset khi flush, và CDC buffer thực hiện commit bình thường. |
| 4 | `internal/handler/shadow/batch_buffer.go` | Sửa | +127 / -37 | - Thêm cờ `isSnapshotBuffer` và phương thức `SetSnapshotMode(bool)`.<br>- Bổ sung mutex `flushMu` chống race condition khi gọi Flush thủ công đồng thời với timer.<br>- Khống chế `g.SetLimit(4)` cho snapshot buffer, 20 cho CDC buffer.<br>- Mở rộng `isRetryableDBError` bắt Class 53 (53300 too many clients, 53200, 53400), 57014 (query canceled), và text lỗi connection pool exhaustion.<br>- Sửa `batchUpsert` trả về `results []RecordExecutionResult`. Khi gặp retryable error, đánh dấu tất cả record còn lại trong chunk/batch là `StatusFailed`.<br>- Khi flush xong, chỉ commit offset nếu `!isSnapshotBuffer` và sử dụng `CalculateSafeCommitOffsets`. |
| 5 | `internal/handler/shadow/kafka_consumer.go` | Sửa | +18 / -3 | - Sửa `afterData == nil && opStr != "d"`: Trả lỗi tường minh `fmt.Errorf("kafka message has no 'after' data (op=%s)", opStr)` thay vì `return 0, nil` (loại bỏ silent drop, kích hoạt luồng đẩy DLQ).<br>- Thêm phương thức `GetMaxLag() int64` đọc an toàn lag tối đa của các partition reader. |
| 6 | `internal/handler/shadow/event_handler.go` | Sửa | +2 / -2 | Sửa `pkValue == ""` trả lỗi tường minh thay vì `return 0, nil` để consumer ghi nhận lỗi và chuyển tiếp DLQ. |
| 7 | `internal/handler/orchestration/snapshot_runner_handler.go` | Sửa | +30 / -0 | - Bổ sung interface `lagInspector` (`GetMaxLag() int64`).<br>- Thêm phương thức `checkAdaptiveLagThrottling(ctx)`: Kiểm tra lag của Kafka Consumer trước mỗi vòng lặp batch mới.<br>- Tự động delay 50ms khi lag > 1000; pause 500ms khi lag > 10000. |
| 8 | `internal/server/server_setup.go` | Sửa | +55 / -22 | - Tách `cdcBatchBuffer` (batch 200, timeout 500ms, commit offset) và `snapshotBatchBuffer` (batch 2000, timeout 5s, `SetSnapshotMode(true)`).<br>- Tách `cdcEventHandler` và `snapshotEventHandler`.<br>- Inject `snapshotEventHandler` vào `snapshotRunner` và `executeHealHandler`.<br>- Inject `cdcEventHandler` vào `kafkaConsumer` và `consumerPools`.<br>- Cập nhật NATS subscriber `schema.config.reload` và `connection.config.changed` để invalidate cache trên cả 2 buffers.<br>- Cập nhật endpoint `/api/v1/internal/stats` xuất trạng thái của cả 2 buffers. |

---

## III. KẾT QUẢ KIỂM THỬ (VERIFICATION RESULTS)

1. **Unit Tests mới**:
   - `watermark_tracker_test.go`:
     - `TestCalculateSafeCommitOffsets_AllPass`: **PASS** (0.00s)
     - `TestCalculateSafeCommitOffsets_MiddleFailurePreventsDataLoss`: **PASS** (0.00s)
     - `TestCalculateSafeCommitOffsets_DLQAllowsCommitProgress`: **PASS** (0.00s)
     - `TestIsRetryableDBError_ExpandedCodes`: **PASS** (0.00s)
   - `snapshot_cdc_isolation_test.go`:
     - `TestSnapshotBuffer_NeverCommitsKafkaOffsets`: **PASS** (0.00s)
     - `TestCDCBuffer_CommitsKafkaOffsets`: **PASS** (0.00s)
2. **Regression Tests**:
   - `internal/handler/shadow/...`: **PASS** (100% tests pass, 0 regressions)
   - `internal/handler/orchestration/...`: **PASS** (100% tests pass)
   - `internal/handler/source/...`: **PASS** (100% tests pass)
3. **Build Binary**:
   - `go build ./cmd/...`: **PASS** (Không có lỗi compile)
