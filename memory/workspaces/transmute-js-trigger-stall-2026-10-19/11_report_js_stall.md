# 11 — Final Implementation Report: Khắc phục Tràn đĩa NATS JetStream (`CDC_TRANSMUTE`) & Đứt luồng Realtime Transmute → Master (`js_stall`)

## 1. Tóm tắt Nguyên nhân & Giải pháp đã Triển khai

1. **Nguyên nhân sự cố trên PRODUCTION (`16/10/2026 13:43:24` → `19/10/2026`):**
   - Stream `CDC_TRANSMUTE` trước đây được cấu hình `Retention: nats.LimitsPolicy`, `MaxAge: 24h`, và **không có giới hạn `MaxBytes`**. Dù consumer `transmute-shadow-consumer-v2` đã xử lý hết và gọi `msg.Ack()` (`Pending = 0`, `Ack Pending = 0`), NATS JetStream vẫn giữ lại toàn bộ `47,612` messages (~1 GB payload chứa `events`) trên đĩa cho tới khi phân vùng `/data/jetstream` bị đầy (`no space left on device` tại `135.blk`).
   - Khi `bb.js.PublishMsg` thất bại do đầy đĩa, `BatchBuffer.publishTransmuteTrigger` fallback sang Core NATS trên cùng subject `"cdc.cmd.transmute-shadow"`, nhưng `server_setup.go` chỉ đăng ký Core NATS subscriber khi `natsClient.JS == nil`. Hệ quả là 100% trigger realtime bị rơi vào hư không và **0 record được ghi vào Master từ ngày 16/10 đến 19/10** (dù dữ liệu vẫn được ghi đầy đủ vào các bảng `shadow_*`).

2. **Giải pháp đã thực thi (Sau khi User `APPROVE`):**
   - **Đổi `CDC_TRANSMUTE` sang `nats.WorkQueuePolicy` + `MaxBytes: 256 MiB` (`pkgs/natsconn/nats_client.go`):** Xóa message khỏi đĩa ngay lập tức khi worker gọi `msg.Ack()` hoặc `msg.Term()` thay vì giữ 24h.
   - **Đặt trần `MaxBytes` + `DiscardOld` cho toàn bộ 4 stream JetStream (`pkgs/natsconn/nats_client.go`):** `CDC_EVENTS` (`512 MiB`), `CDC_TRANSMUTE` (`256 MiB`), `SCHEMA_DRIFT` (`64 MiB`), `SCHEMA_CONFIG` (`64 MiB`).
   - **Tự động Reconcile & Migrate `Retention` lúc khởi động (`pkgs/natsconn/nats_client.go`):** Khi phát hiện `info.Config.Retention != desired.Retention` (ví dụ `CDC_TRANSMUTE` trên PROD đang là `LimitsPolicy`), tự động `DeleteStream` rồi `AddStream` mới (tránh lỗi NATS `10052` và tự động dọn sạch ~1 GB message cũ đã ACK trên PROD ngay khi deploy).
   - **Khắc phục Core NATS Fallback (`transmute_handler.go`, `server_setup.go`, `batch_buffer_fanout.go`):** Khi `bb.js.PublishMsg` lỗi sau 3 lần thử, bắn fallback vào subject riêng `"cdc.cmd.transmute-shadow.core-fallback"` luôn được `server_setup.go` đăng ký `QueueSubscribe` tới `transmuteHandler.HandleTransmuteShadow`.
   - **Đảm bảo End-to-End `Ack()` / `NakWithDelay()` sau khi ghi DB Master & `MaxDeliver = 5` (`debounce.go`, `transmute_handler.go`, `transmute_consumer_pool.go`):**
     - Nối kênh kết quả `DoneCh chan error` vào `NatsMsgTask` (`debounce.go:12-25`).
     - Trong `HandleTransmuteShadowTask` (`transmute_handler.go:138-237,885-919`), đẩy trực tiếp các `NatsMsgTask` của `masterTargets` vào `TableDebouncer` và giữ chờ (`waitForTransmuteDone`, kèm nhịp `msg.InProgress()` mỗi 5s) cho tới khi `runDebouncedGroup` / `binarySearchSplit` / `processSubBatch` thực sự ghi xong xuống DB Master.
     - Chỉ khi ghi DB Master thành công (`err == nil`) mới trả về `nil` để `TransmuteConsumerPool.processMessage` gọi `msg.Ack()` (lúc đó `WorkQueuePolicy` mới xóa message khỏi đĩa).
     - Nếu lỗi lookup `masterBindingRepo` hoặc lỗi ghi DB Master (`err != nil`) → trả về `error` thật để gọi `msg.NakWithDelay(delay)` có backoff (`2s * NumDelivered`), giữ nguyên message trên đĩa cho NATS JetStream phát lại.
     - Khi thử lại đủ `MaxDeliver = 5` lần (`meta.NumDelivered >= uint64(p.cfg.MaxDeliver)`) mà vẫn lỗi → gọi `_ = msg.Term()` và log lỗi poison pill để dừng redeliver vô tận (`transmute_consumer_pool.go:146-175`).
   - **Tự động suy diễn mốc `lastGpayID` từ bảng Master khi bấm "Chạy ngay (tiếp theo)" (`internal/service/master/transmuter.go:346-360,985-1019`):**
     - Giữ nguyên việc xóa `LastCursorJSON = []byte("{}")` ở cuối `Run()` (`transmuter.go:523-528`) để nút **"Chạy ngay (toàn bộ)"** (`runType == "full"`) luôn chạy lại từ đầu và không bao giờ bị kẹt vào mốc cũ.
     - Khi chạy **"Chạy ngay (tiếp theo)"** (`runType != "full"` và `len(chunkIDs) == 0`) mà `lastGpayID == 0`, tự động gọi `t.inferLastGpayIDFromMaster(ctx, masterRow, masterDB, rules)` để suy diễn mốc `lastGpayID` trên Shadow tương ứng với dữ liệu đã có trên bảng Master, cập nhật `totalShadowRows, _ = t.countShadowRows(ctx, masterRow, lastGpayID)` và chạy tiếp từ mốc đó (ví dụ bảng Shadow có `4.1M` dòng, Master đã có `4.0M` dòng → tự động chạy tiếp từ dòng `4.0M` và chỉ quét `100k` dòng còn thiếu).

---

## 2. Danh sách File Thay đổi & Thống kê (`git diff --stat`)

```text
 internal/handler/master/debounce.go                |  23 +-
 internal/handler/master/transmute_consumer_pool.go |  26 +-
 .../master/transmute_debounce_group_test.go        | 306 ++++++++++++++++++++-
 internal/handler/master/transmute_handler.go       | 266 ++++++++++++++----
 internal/handler/shadow/batch_buffer_fanout.go     |   9 +-
 .../handler/shadow/snapshot_cdc_isolation_test.go  |  76 +++++
 internal/server/server_setup.go                    |   1 +
 internal/service/master/transmuter.go              |  50 ++++
 internal/service/master/transmuter_gpayid_test.go  | 162 ++++++++++-
 pkgs/natsconn/nats_client.go                       | 123 ++++++---
 pkgs/natsconn/nats_client_test.go                  | 100 +++++++
 11 files changed, 1044 insertions(+), 106 deletions(-)
```

| File | Loại thay đổi | Mô tả chi tiết |
|------|---------------|----------------|
| [`pkgs/natsconn/nats_client.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/natsconn/nats_client.go#L63-L167) | Code | Cấu hình `CDC_TRANSMUTE` với `WorkQueuePolicy` + `MaxBytes: 256 MiB`, đặt `MaxBytes` cho cả 4 stream, và tự động reconcile (`DeleteStream`/`AddStream` khi đổi `Retention`, `UpdateStream` khi đổi `MaxBytes`/`MaxAge`/`Subjects`/`Discard`). |
| [`internal/handler/master/debounce.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/debounce.go#L12-L96) | Code | Bổ sung `DoneCh chan error`, hàm `complete(err error)` cho `NatsMsgTask` và `AddTask(task NatsMsgTask)` cho `TableDebouncer`. |
| [`internal/handler/master/transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L65-L260) | Code | Thực thi End-to-End `Ack`/`NakWithDelay` qua `handleTransmuteShadowWithOutcome`, `waitForTransmuteDone`, báo kết quả `t.complete(err)` tại `runDebouncedGroup` / `binarySearchSplit` / `processSubBatch`, và khai báo `SubjectTransmuteShadowCoreFallback`. |
| [`internal/handler/master/transmute_consumer_pool.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_consumer_pool.go#L130-L175) | Code | Kiểm tra `meta.NumDelivered >= uint64(p.cfg.MaxDeliver)` khi handler trả lỗi để gọi `msg.Term()` dừng poison pill ở lần thử thứ 5, và áp dụng backoff cho `msg.NakWithDelay`. |
| [`internal/server/server_setup.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L359-L363) | Code | Luôn đăng ký `QueueSubscribe` cho `SubjectTransmuteShadowCoreFallback` tới `transmuteHandler.HandleTransmuteShadow` kể cả khi `natsClient.JS != nil`. |
| [`internal/handler/shadow/batch_buffer_fanout.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go#L118-L129) | Code | Khi `bb.js.PublishMsg` lỗi sau 3 lần thử, publish fallback qua `cdc.cmd.transmute-shadow.core-fallback`. |
| [`internal/service/master/transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L346-L360) | Code | Tự động suy diễn `lastGpayID` từ bảng Master (`inferLastGpayIDFromMaster`, dòng `985-1019`) và cập nhật `totalShadowRows` khi chạy `"Chạy ngay (tiếp theo)"` (`runType != "full"`). |
| [`internal/service/master/transmuter_gpayid_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_gpayid_test.go#L45-L181) | Test | Kiểm thử `TestInferLastGpayIDFromMaster` trên PostgreSQL thật (kết hợp Sonyflake `_gpay_id`, SHA-256 `RunEvents` hash, lọc bỏ hash ngẫu nhiên, và kiểm chứng trên bảng `payment_bills` `>1M` dòng). |
| [`pkgs/natsconn/nats_client_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/natsconn/nats_client_test.go#L13-L184) | Test | Kiểm thử `WorkQueuePolicy`, `MaxBytes`, xóa message ngay sau `AckSync()`, và migrate từ `LimitsPolicy` sang `WorkQueuePolicy`. |
| [`internal/handler/master/transmute_debounce_group_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_debounce_group_test.go#L64-L350) | Test | Kiểm thử tích hợp `WorkQueueImmediateDeleteOnAck`, `EndToEndAckAndNakRetryOnMasterDBFailure` (retry khi lỗi ghi Master), và `MaxDeliverExhaustionTerminatesPoisonPill` (dừng sau 5 lần retry). |
| [`internal/handler/shadow/snapshot_cdc_isolation_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/snapshot_cdc_isolation_test.go#L156-L229) | Test | Kiểm thử `BatchBuffer.Flush` fallback sang `cdc.cmd.transmute-shadow.core-fallback` khi JetStream publish thất bại. |

---

## 3. Hướng dẫn Vận hành sau khi Deploy lên PRODUCTION

1. **Khi deploy `centralized-data-service` mới lên PROD:**
   - Lúc khởi động, `EnsureStreams` sẽ tự động xóa stream `CDC_TRANSMUTE` cũ (`LimitsPolicy` đang giữ ~1 GB / 47,612 messages đã ACK) và tạo lại `CDC_TRANSMUTE` với `WorkQueuePolicy` + `MaxBytes: 256 MiB`, tiếp theo `TransmuteConsumerPool.Start()` tự tạo lại consumer `transmute-shadow-consumer-v2`.
   - Ổ đĩa `/data/jetstream` được giải phóng ngay lập tức và luồng realtime `Shadow → Master` tự động chạy lại cho toàn bộ các bản ghi mới với cơ chế End-to-End `Ack()` sau khi ghi Master / `NakWithDelay()` khi lỗi DB / `Term()` sau `MaxDeliver = 5` lần thử.
2. **Đồng bộ bù dữ liệu từ `16/10 13:43` đến lúc deploy:**
   - Vì dữ liệu từ 16/10 đã nằm sẵn đầy đủ trong các bảng `shadow_*` (`cdc_shadow`), sau khi deploy xong chỉ cần bấm **"Chạy ngay (tiếp theo)"** cho các bảng Master active: hệ thống sẽ tự động suy diễn mốc `lastGpayID` từ bảng Master (kèm `Gap-Safe Guard` không nhảy cóc qua vùng trũng Outage 16/10–19/10) và chỉ quét bù đúng các bản ghi còn thiếu từ Shadow sang Master.

---

## 4. Báo Cáo Khắc Phục Sau Kiểm Toán Phản Biện (`T13 → T16` / `F1 → F7`)

### 4.1. Thống Kê Thay Đổi (`git diff --stat` so với `78c20a9`)
```text
 internal/handler/master/debounce.go                |   5 +
 internal/handler/master/transmute_consumer_pool.go |  95 +++-
 .../master/transmute_debounce_group_test.go        | 626 ++++++++++++++++++---
 internal/handler/master/transmute_handler.go       | 160 ++++--
 internal/service/master/transmuter.go              |  50 +-
 internal/service/master/transmuter_gpayid_test.go  |  70 ++-
 6 files changed, 820 insertions(+), 186 deletions(-)
```

### 4.2. Tóm Tắt Các Lỗ Hổng Đã Khắc Phục Triệt Để
1. **`F1` + `F2` (`transmute_consumer_pool.go:167-230`, `transmute_handler.go:194-231, 890-960`):** Thay thế việc chờ đồng bộ trong callback `QueueSubscribe` bằng `ErrAsyncAckPending` + `shadowMsgTracker` (atomic fan-out counter + goroutine gửi `msg.InProgress()` định kỳ tối đa 35 phút). Giải phóng ngay 10 worker để `TableDebouncer` gom lô không giới hạn và giữ sống message qua các mẻ SQL dài hơn `AckWait = 30s`.
2. **`F3` (`transmuter.go:530, 1028-1054`):** Bổ sung `Gap-Safe Guard` trong `inferLastGpayIDFromMaster` (kiểm tra 100% mẫu `gapSourceIDs` ngay sau `inferred` đã tồn tại trong Master và `remAfterCand >= activeShadowRows - masterCount`) để không bao giờ nhảy cóc qua vùng trũng Outage khi CDC Realtime (`RunEvents`) ghi các dòng mới vào cuối bảng trước.
3. **`F4` (`debounce.go:180-189`):** Khi `<-td.ctx.Done()` xảy ra lúc đang chờ `td.sem`, `workerLoop` chạy nốt `td.runTransmute(context.Background(), batch)` thay vì bỏ rơi `batch`.
4. **`F5` (`transmute_handler.go:40-48`, `transmute_debounce_group_test.go`):** Xóa sạch 3 trường closure mock (`lookupMasterTargetsFn`, `runEventsFn`, `runFn`) khỏi struct production `TransmuteHandler`; viết lại toàn bộ integration test chạy trên **Postgres thật (`:5433`, `:5436`, `:5434`) + NATS JetStream thật**.
5. **`F6` (`transmute_debounce_group_test.go:762-818`):** Xóa stream rác `TEST_WQ_CHECK` trên NATS Docker local (`:14222` / `:18222`) và migrate trực tiếp `CDC_TRANSMUTE` trên broker local sang `"retention": "workqueue", "max_bytes": 268435456`.
6. **`F7` (`transmuter.go:668-755`):** Sửa `TransmuterModule.RunEvents` lưu `lastEventErr` khi `DELETE` hoặc `bulkUpsertMaster` trả lỗi và trả về `res, lastEventErr` (thay vì nuốt vào `res.Skipped++` và trả về `nil`), đảm bảo kích hoạt `NakWithDelay` khi DB Master lỗi.
