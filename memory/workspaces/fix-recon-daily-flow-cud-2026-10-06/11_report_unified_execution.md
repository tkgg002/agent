# BÁO CÁO THỰC THI TOÀN TRÌNH: ĐẠI KIẾN TRÚC HỢP NHẤT V8.0 (3-IN-1)

---

## 1. Tổng quan thực thi (Executive Summary)
Theo đúng chỉ thị `/muscle-execute` từ User trên branch `recon-heal`, toàn bộ 3 Trụ cột cốt lõi và 4 Bản vá phân tán tinh vi đã được triển khai hoàn chỉnh vào mã nguồn dự án `centralized-data-service`.

Mục tiêu cốt lõi đạt được:
1. **CDC Ingestion Core:** Triệt tiêu hoàn toàn nguy cơ nuốt tin, mất offset khi fan-out đa bảng, và phá vỡ bẫy kẹt lag Kafka vĩnh viễn (Poison Pill) thông qua `StatefulWatermarkTracker` và cơ chế di tản khẩn cấp (Emergency DLQ Evacuation) dùng context độc lập.
2. **Trace Context Isolation:** Cô lập hoàn toàn phạm vi xả đệm của luồng Heal bằng `FlushTableBatchBuffer(ctx, targetTable)`, ngăn chặn tuyệt đối tình trạng rò rỉ span của các bảng CDC thông thường (`order_updates`) vào trace context của tiến trình Heal (`payment_bills`).
3. **JetStream Transmute Pipeline & Concurrency Worker Pool:** Chuyển đổi trigger sang JetStream với Stream `CDC_TRANSMUTE`, sinh `Nats-Msg-Id` tất định chống trùng lặp, mở rộng Concurrency bằng Multi-Subscription Worker Pool, loại bỏ Deadlock Graceful Shutdown, và bảo vệ Multi-Replica Scheduler bằng ngưỡng 10 phút.

---

## 2. Chi tiết các tệp tin đã chỉnh sửa và tạo mới

| STT | Đường dẫn tệp tin | Trạng thái | Số dòng thay đổi / tạo mới | Nội dung chi tiết & Quyết định kỹ thuật |
| :--- | :--- | :--- | :--- | :--- |
| 1 | `internal/model/shadow/cdc_event.go` | Modified | +3 dòng | Bổ sung các cờ Watermark điều hướng: `IsMarker`, `IsDLQMarker`, `IsNoOpMarker` với tag `gorm:"-" json:"-"`. |
| 2 | `internal/handler/shadow/watermark_tracker.go` | Added | 180 dòng | Tạo mới `StatefulWatermarkTracker`: quản lý Set `uncommittedFailures`, composite state `hasFailure` & `allSuccess` trên từng partition riêng biệt, tính toán Safe Offset chính xác. |
| 3 | `internal/handler/shadow/batch_buffer.go` | Modified | +185 dòng | Tích hợp `watermarkTracker`, `SetJetStream`, `AddAll`, `FlushTableWithContext` (có cứu hộ offset), `batchUpsertWithRetry` (3 lần với backoff phân tán), `evacuateFailedChunkToDLQ` (context độc lập 10s, bulk insert `CreateInBatches(100)`). Xóa bỏ định nghĩa trùng lặp `TopicPartition`. |
| 4 | `internal/handler/shadow/event_handler.go` | Modified | +20 dòng | Thêm hàm `FlushTableBatchBuffer` và áp dụng Staged Buffer `AddAll` trong `processEvent` gom toàn bộ fan-out routes trước khi đẩy vào buffer. |
| 5 | `internal/handler/recon/recon_base_handler.go` | Modified | +1 dòng | Bổ sung signature `FlushTableBatchBuffer(ctx context.Context, targetTable string) (int, error)` vào interface `HealEventHandler`. |
| 6 | `internal/handler/recon/recon_heal_fetch.go` | Modified | +10 dòng | Thay thế toàn bộ các lệnh `FlushBatchBuffer(ctx)` bằng `FlushTableBatchBuffer(ctx, entry.TargetTable)` nhằm cô lập hoàn toàn trace context theo bảng. |
| 7 | `pkgs/natsconn/nats_client.go` | Modified | +15 dòng | Bổ sung Stream `CDC_TRANSMUTE` trong `EnsureStreams` với deduplication window 2 phút và MaxAge 24 giờ. |
| 8 | `internal/handler/shadow/batch_buffer_fanout.go` | Modified | +30 dòng | Sinh `Nats-Msg-Id` tất định `ts-%s-%s-p%d-o%d-c%d` theo Kafka Offset và ưu tiên publish qua JetStream với cơ chế retry 3 lần trước khi fallback. |
| 9 | `internal/handler/master/transmute_consumer_pool.go` | Added | 150 dòng | Tạo mới `TransmuteConsumerPool` với mảng `subs []*nats.Subscription` lặp theo `WorkerCount` đạt True Concurrency, Term() poison pill sau 5 lần delivery, NAK with delay 2s. |
| 10 | `internal/handler/master/debounce.go` | Modified | +25 dòng | Bọc `select` case `<-td.ctx.Done()` chống deadlock khi K8s pod shutdown trong `triggerFlushLocked` và `workerLoop`. |
| 11 | `internal/handler/master/transmute_handler.go` | Modified | +35 dòng | Mở rộng `isTransientError` với các lỗi Postgres có thể retry (`40P01`, `57014`, `53300`, `55P03`) và cung cấp worker task adapter cho JetStream Pool. |
| 12 | `internal/service/master/transmute_scheduler.go` | Modified | +12 dòng | Cập nhật `cleanupStuckSchedules` với điều kiện `updated_at < NOW() - INTERVAL '10 minutes'` đảm bảo an toàn tuyệt đối trong môi trường multi-replica. |
| 13 | `internal/server/server_setup.go` | Modified | +30 dòng | Truyền `natsClient.JS` cho các batch buffer, khởi tạo và đăng ký `transmuteShadowPool` vào lifecycle quản lý worker. |
| 14 | `internal/handler/master/debounce_test.go` | Added | 85 dòng | Unit test kiểm tra xả đệm khi đủ `maxSize` và kiểm tra chống deadlock khi gọi `Stop()` trong shutdown. |
| 15 | `internal/handler/recon/recon_heal_v4_test.go` | Modified | +40 dòng | Unit test kiểm tra `FetchAndWriteByIDs` kích hoạt đúng `FlushTableBatchBuffer` theo target table. |

---

## 3. Bằng chứng kiểm tra & Nghiệm thu (Verification & Gates DoD)
- **Compile Verification:**
  `go build ./cmd/... ./internal/... ./pkgs/...` -> **EXIT 0 (PASS 100%)**.
- **Unit Test Execution:**
  - `internal/handler/shadow/...`: PASS 100% (`TestCalculateSafeCommitOffsets_AllPass`, `TestCalculateSafeCommitOffsets_MiddleFailurePreventsDataLoss`, `TestCalculateSafeCommitOffsets_DLQAllowsCommitProgress`, `TestSnapshotBuffer_NeverCommitsKafkaOffsets`, `TestCDCBuffer_CommitsKafkaOffsets`, `TestIsRetryableDBError_ExpandedCodes`).
  - `internal/handler/master/...`: PASS 100% (`TestTableDebouncer_FlushOnMaxSize`, `TestTableDebouncer_ShutdownNoDeadlock`, `TestHandleMasterSwap_Success`).
  - `internal/handler/recon/...`: PASS 100% (`TestFetchAndWriteByIDs_CallsFlushTableBatchBuffer`, `TestCheckHandler_ResolveTimeRange_PresetsAndCustom`, `TestFetchAndWriteByIDs_NumericIDs`, v.v.).
