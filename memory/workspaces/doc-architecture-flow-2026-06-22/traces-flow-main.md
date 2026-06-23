# Implementation Plan - Hardening CDC End-to-End Tracing & Context Propagation

## Goal / Mục tiêu
Triển khai bọc spans và truyền trace context (Context Propagation) xuyên suốt 5 luồng xử lý chính trong `centralized-data-service` nhằm đảm bảo tính liên tục của bản đồ vết trace (traces map) và phục vụ quan sát hệ thống.
Implement trace span encapsulation and propagate context across 5 core workflows in `centralized-data-service` to ensure trace map continuity and end-to-end observability.

---

## User Review Required / Các điểm cần User xem xét
> [!IMPORTANT]
> - Các thay đổi này chủ yếu tác động lên phân hệ **worker** chạy ngầm và **NATS message handlers**. Mọi thay đổi đều được bảo vệ bằng cơ chế xử lý lỗi (error capturing) trong các spans.
> - Luồng **BatchBuffer** gom cụm bất đồng bộ nên span cha của nó là `"cdc.batchbuffer.flush"` sẽ được tạo ở cấp độ Flush và lan truyền xuống các child spans `"cdc.batchbuffer.upsert"` cùng NATS trigger `"cdc.cmd.transmute-shadow"`.
> - These changes primarily impact the **worker daemons** and **NATS message handlers**. All modifications are wrapped with robust error-capturing inside spans.
> - Since **BatchBuffer** flushes asynchronously, the parent span `"cdc.batchbuffer.flush"` will be initialized at the Flush level and propagated down to child spans `"cdc.batchbuffer.upsert"` and NATS trigger `"cdc.cmd.transmute-shadow"`.

---

## Proposed Changes / Các thay đổi đề xuất

### 1. Ingestion & Transmutation Flows / Luồng Ingestion & Transmutation

#### [MODIFY] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
- **Thay đổi / Changes**:
  - Cập nhật phương thức `Flush()` để tạo span cha `"cdc.batchbuffer.flush"` bọc quanh toàn bộ chu kỳ flush.
  - Truyền context này xuống hàm `batchUpsert` và `publishTransmuteTrigger`.
  - Trong `batchUpsert`, nhận `ctx context.Context` và đổi tên child span từ `"cdc.batch_upsert"` thành `"cdc.batchbuffer.upsert"`.
  - Update `Flush()` to create a parent span `"cdc.batchbuffer.flush"` around the flush cycle.
  - Propagate this context to `batchUpsert` and `publishTransmuteTrigger`.
  - In `batchUpsert`, accept `ctx context.Context` and rename the span from `"cdc.batch_upsert"` to `"cdc.batchbuffer.upsert"`.

#### [MODIFY] [batch_buffer_fanout.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go)
- **Thay đổi / Changes**:
  - Cập nhật hàm `publishTransmuteTrigger` để nhận `ctx context.Context`.
  - Thay thế `bb.natsConn.Publish` bằng `bb.natsConn.PublishMsg` đính kèm trace context qua NATS Headers.
  - Update `publishTransmuteTrigger` to accept `ctx context.Context`.
  - Replace `bb.natsConn.Publish` with `bb.natsConn.PublishMsg` containing trace context injected via NATS Headers.

#### [MODIFY] [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)
- **Thay đổi / Changes**:
  - Trong `HandleTransmuteShadow`, trích xuất trace context từ NATS Header của tin nhắn đầu vào, sau đó thay thế `h.natsConn.Publish` khi gửi NATS message tiếp theo ở dòng 122 bằng `PublishMsg` đính kèm trace context đã trích xuất.
  - Trong `HandleTransmute`, đổi tên span từ `"nats.HandleTransmute"` thành `"cdc.worker.transmute.process"`.
  - In `HandleTransmuteShadow`, extract context from the incoming message header and replace `h.natsConn.Publish` at line 122 with `PublishMsg` to propagate trace context.
  - In `HandleTransmute`, rename span from `"nats.HandleTransmute"` to `"cdc.worker.transmute.process"`.

#### [MODIFY] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
- **Thay đổi / Changes**:
  - Bọc logic xử lý chính của hàm `Run` bằng child span `"cdc.service.transmute"`.
  - Wrap `Run` method's main execution logic with child span `"cdc.service.transmute"`.

---

### 2. Reconciliation & Ticker Flows / Luồng Reconciliation & Tickers

#### [MODIFY] [worker_server_tickers.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/worker_server_tickers.go)
- **Thay đổi / Changes**:
  - (Đã có span `"cdc.worker.reconcile_cycle"`). Đảm bảo context này được truyền xuống các method con của `ReconCore`.
  - (Already has `"cdc.worker.reconcile_cycle"`). Ensure this context is passed down to `ReconCore` child methods.

#### [MODIFY] [scan_handler_discover.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler_discover.go)
- **Thay đổi / Changes**:
  - Trong `HandlePeriodicScan`, khởi tạo `subMsg.Header = make(nats.Header)` và gọi `observability.InjectNATSHeader(ctx, subMsg.Header)` trước khi gọi `h.HandleScanRawData(subMsg)` để tránh đứt gãy trace flow.
  - In `HandlePeriodicScan`, initialize `subMsg.Header = make(nats.Header)` and inject context via `observability.InjectNATSHeader` before calling `h.HandleScanRawData` to avoid trace continuity breakage.

---

### 3. Startup Reaper Flows / Luồng Startup Reaper

#### [MODIFY] [worker_server.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/worker_server.go)
- **Thay đổi / Changes**:
  - Bọc tác vụ startup dọn dẹp `s.reconCore.ReapOrphanRunsFromDeadInstances(...)` bằng anonymous function tự thực thi và span `"cdc.worker.startup_reap"`.
  - Wrap startup task `s.reconCore.ReapOrphanRunsFromDeadInstances(...)` with an immediately invoked anonymous function and span `"cdc.worker.startup_reap"`.

#### [MODIFY] [recon_engine_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_run.go)
- **Thay đổi / Changes**:
  - Bọc các phương thức `ReapStaleRuns` và `ReapOrphanRunsFromDeadInstances` bằng child span `"cdc.service.reap"` kèm attribute `reap.type` (giá trị tương ứng `"stale"` và `"orphan"`).
  - Wrap `ReapStaleRuns` and `ReapOrphanRunsFromDeadInstances` with child span `"cdc.service.reap"` and attribute `reap.type` (`"stale"` or `"orphan"`).

---

### 4. DLQ & Retry Engine Flows / Luồng DLQ & Retry Engine

#### [MODIFY] [dlq_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/dlq_handler.go)
- **Thay đổi / Changes**:
  - Bọc logic `HandleWithRetryContext` bằng child span `"cdc.worker.dlq.process"`.
  - Trong `sendToDLQ` và `ReplayDLQ`, chuyển đổi `nats.Conn.Publish` thô sang `PublishMsg` đính kèm trace context qua NATS Header.
  - Wrap `HandleWithRetryContext` with child span `"cdc.worker.dlq.process"`.
  - In `sendToDLQ` and `ReplayDLQ`, switch from raw `nats.Conn.Publish` to `PublishMsg` and inject context via NATS Header.

#### [MODIFY] [dlq_state_machine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/dlq_state_machine.go)
- **Thay đổi / Changes**:
  - Trong `RunOnce`, bọc logic bằng child span `"cdc.worker.dlq.retry"`.
  - Trong `retryOne`, thay thế `sm.nats.Conn.Publish` thô bằng `PublishMsg` đính kèm trace context qua NATS Header.
  - In `RunOnce`, wrap with child span `"cdc.worker.dlq.retry"`.
  - In `retryOne`, replace raw `sm.nats.Conn.Publish` with `PublishMsg` and propagate trace context.

---

## Verification Plan / Kế hoạch Xác minh

### Automated Verification / Xác minh Tự động
- Biên dịch toàn bộ codebase để đảm bảo tính đúng đắn về cú pháp:
  `go build ./...` trong `centralized-data-service`.
- Compile the entire codebase to ensure syntactical correctness:
  `go build ./...` in `centralized-data-service`.

### Manual Verification / Xác minh Thủ công
- Chạy rà soát bảo mật bằng `/security-agent` để đảm bảo code sạch trước khi đánh dấu Hoàn thành.
- Run security scanning using `/security-agent` to ensure clean code before marking the task as Done.
