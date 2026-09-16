# Tasks — Transmute Pipeline Reliability (NATS Core -> JetStream Migration)

## Phase 1: Stream Provisioning & Content-Hash Publisher
- [ ] **T1**: `pkgs/natsconn/nats_client.go` — Bổ sung Stream `CDC_TRANSMUTE` (`cdc.cmd.transmute`, `cdc.cmd.transmute-shadow`) vào `EnsureStreams`.
- [ ] **T2**: `internal/handler/shadow/batch_buffer.go` — Bổ sung `js nats.JetStreamContext` và `SetJetStream`.
- [ ] **T3**: `internal/handler/shadow/batch_buffer_fanout.go` — Cập nhật `publishTransmuteTrigger` dùng `js.PublishMsg` có retry 3 lần + `Nats-Msg-Id` gắn liền nội dung batch (`ts-<table_name>-<batch_uuid>-<count>`).

## Phase 2: JetStream Consumer Pools & Handlers
- [ ] **T4**: `internal/handler/master/transmute_consumer_pool.go` — Tạo mới `TransmuteConsumerPool` với `FilterSubject`, `MaxDeliver(5)`, `MaxAckPending(500)` và circuit breaker `msg.Term()` khi `meta.NumDelivered >= 5`.
- [ ] **T5**: `internal/handler/master/transmute_handler.go` — Viết `HandleTransmuteShadowMsg` trả về error (NakWithDelay khi DB fail); cập nhật `runDebouncedTransmute` chỉ `Msg.Ack()` sau khi Master commit thành công; Full Transmute bổ sung Heartbeat Ticker 15s có deadline `context.WithTimeout(2h)` và DB Heartbeat 1m.
- [ ] **T6**: `internal/handler/master/debounce.go` — Sửa `Add` & `flushBatch`: trích xuất slice dưới Lock, nhả Lock rồi thực hiện blocking send ra channel để bảo toàn tuyệt đối thứ tự FIFO và tạo backpressure tự nhiên.

## Phase 3: Wiring, Scheduler & Transmuter Engine
- [ ] **T7**: `internal/server/server_setup.go` — Thay thế `QueueSubscribe` bằng 2 `TransmuteConsumerPool` có `FilterSubject` riêng biệt và quản lý lifecycle.
- [ ] **T8**: `internal/service/master/transmute_scheduler.go` — Bổ sung định kỳ 5m cleanup schedule stale heartbeat (`updated_at < NOW() - 10m`); tăng `LIMIT 10` lên `LIMIT 50`.
- [ ] **T9**: `internal/service/master/transmuter.go` — Thêm `UpdateScheduleHeartbeat`; sửa `bulkUpsertMaster` không nuốt lỗi; nâng OCC clock skew tolerance lên 60s.

## Phase 4: Verification
- [ ] **T10**: `go build ./...` — Compile check toàn bộ centralized-data-service.
- [ ] **T11**: `go test ./internal/handler/master/... ./internal/handler/shadow/... ./internal/service/master/...` — Unit & integration tests.
- [ ] **T12**: Chaos test — Xác minh thứ tự FIFO khi burst traffic, timeout zombie heartbeat, và poison pill termination.
