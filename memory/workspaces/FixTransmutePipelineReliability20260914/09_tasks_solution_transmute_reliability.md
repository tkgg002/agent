# Hồ Sơ Giải Pháp Kỹ Thuật Chi Tiết (Technical Solutions)
## Chuyển Đổi Transmute Trigger Pipeline từ NATS Core sang JetStream

Tài liệu này đặc tả chi tiết toàn bộ kiến trúc và các dòng code cần thay đổi trong `centralized-data-service` để chuyển đổi toàn bộ pipeline transmute trigger từ **NATS Core sang JetStream**, giải quyết triệt để 5 Failure Modes rớt trigger.

---

## 1. So Sánh Kiến Trúc & Giải Pháp Gốc Rễ

| Tiêu chí | NATS Core Hiện Tại (Gây Rớt Event) | NATS JetStream (Giải Pháp Chuẩn Hóa) |
| :--- | :--- | :--- |
| **Cơ chế lưu trữ** | In-memory broker, không lưu đĩa. | **FileStorage (Persistent Stream `CDC_TRANSMUTE`)** lưu trên đĩa broker. |
| **Bảo đảm giao vận** | At-most-once (Fire-and-forget). Mất kết nối là mất sạch. | **At-least-once**. Có cơ chế `Ack`, `Nak`, `Term`, `AckWait`, `MaxDeliver`. |
| **Worker Crash / OOM** | Event nằm trên RAM của Debouncer bị xóa trắng vĩnh viễn. | **Không mất event**: Event chỉ được `Ack()` sau khi Master ghi DB thành công. Nếu crash, `AckWait(60s)` hết hạn, JetStream tự động redeliver cho worker sau khi restart. |
| **DB Timeout (5s)** | Lookup fail -> log warn -> return -> mất trigger. | Lookup fail -> gọi `msg.NakWithDelay(2s)` -> JetStream thử lại sau 2 giây. |
| **NATS Reconnect** | Publish lỗi -> log warn -> drop trigger. | `js.PublishMsg` có retry 3 lần với exponential backoff + Message Deduplication ID. |
| **Debouncer Overflow** | Nhồi nhét vào RAM, `flushCh` (100) đầy gây deadlock mutex `td.mu`. | JetStream Pull Consumer (`Fetch` theo batch): Backpressure tự nhiên từ broker, không nghẽn RAM. |
| **Scheduler Crash** | Stuck ở `last_status='running'`, cleanup chỉ chạy khi restart. | Thêm background ticker 5 phút cleanup stuck schedules > 30m; tăng LIMIT lên 50. |

---

## 2. Chi Tiết Các File Cần Thay Đổi

### FILE 1: `pkgs/natsconn/nats_client.go`
**Mục tiêu**: Bổ sung Stream `CDC_TRANSMUTE` vào hàm `EnsureStreams` với FileStorage và Deduplication Window.

```go
// TẠI EnsureStreams (dòng 66-74):
// Thêm CDC_TRANSMUTE vào danh sách streams:
	streams := []struct {
		name     string
		subjects []string
	}{
		{"CDC_EVENTS", cdcSubjects},
		{"SCHEMA_DRIFT", []string{"schema.drift.detected"}},
		{"SCHEMA_CONFIG", []string{"schema.config.reload"}},
		{"CDC_TRANSMUTE", []string{"cdc.cmd.transmute", "cdc.cmd.transmute-shadow"}},
	}
```

---

### FILE 2: `internal/handler/shadow/batch_buffer_fanout.go` & `batch_buffer.go`
**Mục tiêu**: 
1. Bổ sung `js nats.JetStreamContext` vào `BatchBuffer` (`batchBuffer.SetJetStream(natsClient.JS)`).
2. Tại `publishTransmuteTrigger`: Sử dụng `js.PublishMsg` có retry 3 lần.
3. Gắn `Nats-Msg-Id` phản ánh nội dung duy nhất của chính batch đó (Table + BatchUUID + Số lượng bản ghi) để chống drop nhầm các batch kế tiếp.

```go
// TẠI internal/handler/shadow/batch_buffer.go:
type BatchBuffer struct {
    ...
    natsConn *nats.Conn
    js       nats.JetStreamContext
    ...
}

func (bb *BatchBuffer) SetJetStream(js nats.JetStreamContext) {
    bb.js = js
}

// TẠI internal/handler/shadow/batch_buffer_fanout.go (dòng 68-79):
	msg := &nats.Msg{
		Subject: "cdc.cmd.transmute-shadow",
		Data:    payload,
		Header:  make(nats.Header),
	}
	observability.InjectNATSHeader(ctx, msg.Header)

	// [FIX 3] Dedup ID dựa trên nội dung duy nhất của batch (sinh 1 lần trước retry loop)
	batchUUID := uuid.NewString()
	dedupID := fmt.Sprintf("ts-%s-%s-%d", shadowTable, batchUUID, len(records))
	msg.Header.Set("Nats-Msg-Id", dedupID)

	if bb.js != nil {
		var pubErr error
		for attempt := 1; attempt <= 3; attempt++ {
			_, pubErr = bb.js.PublishMsg(msg)
			if pubErr == nil {
				return
			}
			time.Sleep(time.Duration(attempt*50) * time.Millisecond)
		}
		bb.logger.Error("post-ingest transmute trigger jetstream publish failed after 3 retries",
			zap.String("table", shadowTable), zap.Error(pubErr))
		return
	}

	if err = bb.natsConn.PublishMsg(msg); err != nil {
		bb.logger.Warn("post-ingest transmute trigger publish failed",
			zap.String("table", shadowTable), zap.Error(err))
	}
```

---

### FILE 3: `internal/handler/master/transmute_consumer_pool.go` (FILE MỚI)
**Mục tiêu**: Xây dựng JetStream Worker Pool chuẩn hóa:
- Khai báo tường minh `FilterSubject` để phân tách tuyệt đối giữa `transmute-shadow` và `transmute-master`.
- Cấu hình `MaxDeliver(5)` và `MaxAckPending(500)`.
- Khi `meta.NumDelivered >= 5`: ngắt vòng lặp Poison Pill bằng `msg.Term()` và ghi log DLQ.

```go
package master

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type TransmuteMsgHandler func(ctx context.Context, msg *nats.Msg) error

type TransmuteConsumerPool struct {
	sub           *nats.Subscription
	handler       TransmuteMsgHandler
	poolSize      int
	logger        *zap.Logger
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	processed     atomic.Uint64
	failed        atomic.Uint64
	activeWorkers atomic.Int32
}

func NewTransmuteConsumerPool(
	js nats.JetStreamContext,
	streamName string,
	filterSubject string,
	durableConsumer string,
	handler TransmuteMsgHandler,
	poolSize int,
	ackWait time.Duration,
	logger *zap.Logger,
) (*TransmuteConsumerPool, error) {
	// [FIX 4 & FIX 5] Cấu hình tường minh FilterSubject, MaxDeliver, MaxAckPending
	sub, err := js.PullSubscribe(
		filterSubject,
		durableConsumer,
		nats.BindStream(streamName),
		nats.ManualAck(),
		nats.AckWait(ackWait),
		nats.MaxDeliver(5),
		nats.MaxAckPending(500),
	)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TransmuteConsumerPool{
		sub:      sub,
		handler:  handler,
		poolSize: poolSize,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (cp *TransmuteConsumerPool) Start() {
	cp.logger.Info("starting transmute consumer pool", zap.Int("pool_size", cp.poolSize))
	for i := 0; i < cp.poolSize; i++ {
		cp.wg.Add(1)
		go cp.worker(i)
	}
}

func (cp *TransmuteConsumerPool) worker(id int) {
	defer cp.wg.Done()
	cp.activeWorkers.Add(1)
	defer cp.activeWorkers.Add(-1)

	for {
		select {
		case <-cp.ctx.Done():
			return
		default:
			msgs, err := cp.sub.Fetch(10, nats.MaxWait(2*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				time.Sleep(500 * time.Millisecond)
				continue
			}

			for _, msg := range msgs {
				// [FIX 5] Poison Pill Guard: ngắt vòng lặp retry vô tận
				meta, mErr := msg.Metadata()
				if mErr == nil && meta.NumDelivered >= 5 {
					cp.failed.Add(1)
					cp.logger.Error("Poison pill detected, terminating message to DLQ",
						zap.Uint64("delivered", meta.NumDelivered),
						zap.String("subject", msg.Subject),
						zap.String("data", string(msg.Data)),
					)
					_ = msg.Term() // Chấm dứt vĩnh viễn trên JetStream
					continue
				}

				if err := cp.handler(cp.ctx, msg); err != nil {
					cp.failed.Add(1)
					_ = msg.NakWithDelay(2 * time.Second)
				}
			}
		}
	}
}

func (cp *TransmuteConsumerPool) Stop() {
	cp.cancel()
	cp.wg.Wait()
}
```

---

### FILE 4: `internal/handler/master/transmute_handler.go`
**Mục tiêu**:
1. `HandleTransmuteShadowMsg`: trả về `error` khi DB lookup timeout/fail để consumer Nak.
2. `HandleTransmuteMsg`:
   - Với CDC Realtime (`len(req.SourceIDs) > 0`): đưa vào Debouncer, chỉ Ack sau khi Master upsert thành công.
   - Với Full Transmute (`len(req.SourceIDs) == 0`): [FIX 2] Khởi tạo Heartbeat Ticker 15s gửi `msg.InProgress()`, BẮT BUỘC bọc trong `context.WithTimeout(maxDeadline)` để triệt tiêu nguy cơ Zombie Worker.
   - [FIX 6] Cập nhật định kỳ `updated_at = NOW()` vào bảng schedule để làm heartbeat cho Scheduler.

```go
// [FIX 2 & FIX 6] Full Transmute an toàn với DeadLine + Heartbeat Cột:
func (h *TransmuteHandler) runFullTransmuteAsync(ctx context.Context, msg *nats.Msg, req TransmuteRequest) {
	maxDeadline := 2 * time.Hour
	jobCtx, jobCancel := context.WithTimeout(ctx, maxDeadline)
	defer jobCancel()

	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	dbHeartbeatTicker := time.NewTicker(1 * time.Minute)
	defer dbHeartbeatTicker.Stop()

	doneCh := make(chan struct{})

	// Heartbeat Goroutine có Deadline chặn trên
	go func() {
		for {
			select {
			case <-doneCh:
				return
			case <-jobCtx.Done():
				// Đã vượt quá maxDeadline (2h) -> Ngừng heartbeat ngay lập tức!
				observability.Ctx(jobCtx, h.logger).Error("transmute job exceeded max deadline, stopping heartbeat",
					zap.String("master", req.MasterTable),
					zap.Duration("deadline", maxDeadline))
				if msg != nil {
					_ = msg.NakWithDelay(10 * time.Second)
				}
				return
			case <-heartbeatTicker.C:
				if msg != nil {
					_ = msg.InProgress()
				}
			case <-dbHeartbeatTicker.C:
				// [FIX 6 & AUDIT FINDING 4] Cập nhật DB Heartbeat theo ID hoặc fallback theo MasterTable
				if req.ScheduleID > 0 {
					_ = h.svc.UpdateScheduleHeartbeat(jobCtx, req.ScheduleID)
				} else if req.MasterTable != "" {
					_ = h.svc.UpdateScheduleHeartbeatByTable(jobCtx, req.MasterTable)
				}
			}
		}
	}()

	res, err := h.svc.Run(jobCtx, req.MasterTable, nil, req.JobID)
	close(doneCh) // Dừng Heartbeat

	if err != nil {
		if errors.Is(jobCtx.Err(), context.DeadlineExceeded) {
			metrics.TransmuteErrorTotal.WithLabelValues(req.MasterTable).Inc()
			return
		}
		if isTransientError(err) {
			if msg != nil { _ = msg.NakWithDelay(10 * time.Second) }
			return
		}
		// Poison / Fatal error -> Term
		if msg != nil { _ = msg.Term() }
		return
	}

	// Thành công -> Ack
	if msg != nil {
		_ = msg.Ack()
	}
}

// [AUDIT FINDING 3] processSubBatch: BẮT BUỘC Ack cho nửa batch thành công khi chia để trị
func (h *TransmuteHandler) processSubBatch(...) {
    ...
	if err == nil {
		for _, t := range subBatch {
			resp.CorrelationID = t.Req.CorrelationID
			h.reply(t.Msg, resp)
			h.publishCompleted(runCtx, t.Req, res, nil, traceID)
			if t.Msg != nil {
				_ = t.Msg.Ack() // ✅ XÁC NHẬN JETSTREAM: Nửa batch thành công không bị redeliver
			}
		}
	}
}

// [AUDIT FINDING 1] isTransientError: Mở rộng đầy đủ Postgres Deadlock & Statement Timeout
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection pool") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "canceling statement due to statement timeout") ||
		strings.Contains(msg, "sqlstate 57014") ||
		strings.Contains(msg, "deadlock detected") ||
		strings.Contains(msg, "sqlstate 40p01") ||
		strings.Contains(msg, "too many clients") ||
		strings.Contains(msg, "sqlstate 53300") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "sqlstate 08") ||
		strings.Contains(msg, "out of memory") ||
		strings.Contains(msg, "sqlstate 53200")
}
```

---

### FILE 5: `internal/handler/master/debounce.go`
**Mục tiêu**: [FIX 1] Thay thế `go func` bừa bãi bằng Go Idiomatic FIFO: Lấy dữ liệu ra dưới Lock, nhả Lock rồi mới blocking send vào Channel. Đảm bảo 100% thứ tự CDC và backpressure tự nhiên.

```go
func (td *TableDebouncer) extractTasksLocked() []NatsMsgTask {
	if len(td.tasks) == 0 {
		return nil
	}
	if td.idleTimer != nil {
		td.idleTimer.Stop()
		td.idleTimer = nil
	}
	if td.maxTimer != nil {
		td.maxTimer.Stop()
		td.maxTimer = nil
	}
	tasks := td.tasks
	td.tasks = make([]NatsMsgTask, 0, td.maxSize)
	return tasks
}

func (td *TableDebouncer) flushBatch(tasks []NatsMsgTask) {
	if len(tasks) == 0 {
		return
	}
	// Blocking send ngoài lock: đảm bảo tuyệt đối thứ tự FIFO, tạo backpressure lành mạnh
	select {
	case td.flushCh <- tasks:
	case <-td.ctx.Done():
		for _, t := range tasks {
			if t.Msg != nil {
				_ = t.Msg.Nak()
			}
		}
	}
}

func (td *TableDebouncer) Add(msg *nats.Msg, req TransmuteRequest) {
	var toFlush []NatsMsgTask

	td.mu.Lock()
	td.tasks = append(td.tasks, NatsMsgTask{Msg: msg, Req: req})

	if len(td.tasks) >= td.maxSize {
		toFlush = td.extractTasksLocked()
	} else {
		td.setupTimersLocked()
	}
	td.mu.Unlock()

	// Gửi ngoài lock: không bao giờ deadlock Mutex, giữ đúng thứ tự FIFO
	if len(toFlush) > 0 {
		td.flushBatch(toFlush)
	}
}
```

---

### FILE 6: `internal/server/server_setup.go`
**Mục tiêu**: [FIX 4] Khởi tạo Consumer Pools với `FilterSubject` rõ ràng.

```go
	batchBuffer.SetJetStream(natsClient.JS)
	transmuteHandler.SetJetStream(natsClient.JS)

	// 1. Shadow Consumer: CHỈ nhận cdc.cmd.transmute-shadow
	shadowConsumer, err := handlermaster.NewTransmuteConsumerPool(
		natsClient.JS,
		"CDC_TRANSMUTE",
		"cdc.cmd.transmute-shadow", // FilterSubject
		"cds-transmute-shadow-workers",
		transmuteHandler.HandleTransmuteShadowMsg,
		3,
		30*time.Second,
		logger,
	)
	if err == nil {
		ws.RegisterOnStart(func() { shadowConsumer.Start() })
		ws.RegisterOnStop(func() { shadowConsumer.Stop() })
	}

	// 2. Master Consumer: CHỈ nhận cdc.cmd.transmute
	masterConsumer, err := handlermaster.NewTransmuteConsumerPool(
		natsClient.JS,
		"CDC_TRANSMUTE",
		"cdc.cmd.transmute", // FilterSubject
		"cds-transmute-master-workers",
		transmuteHandler.HandleTransmuteMsg,
		cfg.Worker.PoolSize,
		180*time.Second,
		logger,
	)
	if err == nil {
		ws.RegisterOnStart(func() { masterConsumer.Start() })
		ws.RegisterOnStop(func() { masterConsumer.Stop() })
	}
```

---

### FILE 7: `internal/service/master/transmute_scheduler.go`
**Mục tiêu**: 
1. [FIX 6] Thay thế fixed 30m reset bằng dynamic heartbeat check (`updated_at < NOW() - INTERVAL '10 minutes'`).
2. [AUDIT FINDING 2] Sửa cả startup cleanup `cleanupStuckSchedules` kiểm tra `updated_at < NOW() - 10m` để chống race condition khi Rolling Update Pods trên K8s.
3. Tăng `LIMIT 10` lên `LIMIT 50`.

```go
// TẠI cleanupStuckSchedules (Startup - dòng 216-237):
// TRƯỚC: WHERE last_status = 'running' (Giết nhầm pod khác khi rolling update!)
// SAU:
func (s *TransmuteScheduler) cleanupStuckSchedules(ctx context.Context) {
	res := s.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.transmute_schedule
		   SET last_status = 'failed',
		       last_error  = 'Worker restarted — previous session heartbeat timed out (>10m)',
		       updated_at  = NOW()
		 WHERE last_status = 'running'
		   AND updated_at < NOW() - INTERVAL '10 minutes'`,
	)
	if res.RowsAffected > 0 {
		s.logger.Warn("cleaned up stale transmute schedules on restart", zap.Int64("count", res.RowsAffected))
	}
}

// TẠI ticker định kỳ 5 phút:
func (s *TransmuteScheduler) cleanupStuckRunningSchedules(ctx context.Context) {
	res := s.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.transmute_schedule
		   SET last_status = 'failed',
		       last_error  = 'Heartbeat timeout — worker unresponsive for over 10 minutes',
		       updated_at  = NOW()
		 WHERE last_status = 'running'
		   AND updated_at < NOW() - INTERVAL '10 minutes'`,
	)
	if res.RowsAffected > 0 {
		s.logger.Warn("cleaned up dead transmute schedules by heartbeat timeout", zap.Int64("count", res.RowsAffected))
	}
}
```


---

### FILE 8: `internal/service/master/transmuter.go`


#### Thay đổi 3.1: Chặn Nuốt Lỗi Âm Thầm (Silent Skip) Trong `processBatch`
- **Vị trí**: Dòng 925-957
- **Mục tiêu**: Khi `bulkUpsertMaster` thất bại vĩnh viễn (sau 3 lần retry), KHÔNG được phép âm thầm `continue` và trả về `err = nil`. Phải trả `err` về cho `Run()` để kích hoạt cơ chế `binarySearchSplit` (chia nhỏ mẻ cô lập Poison Pill) hoặc trả lỗi về cho debouncer/NATS retry.
```go
// TRƯỚC (Dòng 949-957):
			if err != nil {
				out.skipped += int64(end - i)
				continue
			}
			out.inserted += ins
			out.updated += upd
			out.occSkipped += occSkip
			out.skipped += occSkip

// SAU:
			if err != nil {
				out.lastErr = err
				out.skipped += int64(end - i)
				// Không nuốt lỗi! Return ngay để Run() báo lỗi cho caller xử lý chia để trị (Binary Search Split)
				return out
			}
			out.inserted += ins
			out.updated += upd
			out.occSkipped += occSkip
			out.skipped += occSkip
```
- **Kèm theo cập nhật struct `processBatchResult` và xử lý tại `Run()`**:
```go
// Tại khai báo processBatchResult (Dòng 747-759):
type processBatchResult struct {
	scanned       int64
	inserted      int64
	updated       int64
	skipped       int64
	occSkipped    int64
	ruleMisses    int64
	typeErrors    int64
	lastGpayID    int64
	rowEmitCounts map[string]int
	lastErr       error // Bổ sung để truyền lỗi lên Run()
}

// Tại vòng lặp Run() (Dòng 340-349):
			batchRes := t.processBatch(ctx, masterRow, rules, shadowRows, strat, rc)
			res.Scanned += batchRes.scanned
			res.Inserted += batchRes.inserted
			res.Updated += batchRes.updated
			res.Skipped += batchRes.skipped
			res.OccSkipped += batchRes.occSkipped
			res.RuleMisses += batchRes.ruleMisses
			res.TypeErrors += batchRes.typeErrors
			lastGpayID = batchRes.lastGpayID

			if batchRes.lastErr != nil {
				t.markRuntimeFailure(ctx, masterRow.ID, batchRes.lastErr)
				t.finishTransmuteJob(ctx, jobID, "FAILED", res.Inserted+res.Updated, totalShadowRows, batchRes.lastErr.Error())
				return res, fmt.Errorf("bulk upsert master failed: %w", batchRes.lastErr)
			}
```

#### Thay đổi 3.2: Tăng dung sai Clock Skew OCC chống drop update do lệch giờ
- **Vị trí**: Dòng 1075-1077
- **Mục tiêu**: Nâng dung sai từ 2 giây (2000ms) lên 60 giây (60000ms). Khi Debezium emit batch hoặc Kafka consumer xử lý lag, clock skew giữa các node MongoDB/Postgres có thể vượt 2s dẫn đến mệnh đề `WHERE EXCLUDED._source_ts >= master._source_ts - 2000` bị FALSE và Postgres bỏ qua update (drop update âm thầm).
```go
// TRƯỚC (Dòng 1075-1077):
	// Dung sai clock skew mặc định là 2 giây (2000ms) để khắc phục lệch giờ ở các node nguồn CDC
	const clockSkewToleranceMs = 2000

// SAU:
	// Dung sai clock skew nâng lên 60 giây (60000ms) để chống drop update hợp lệ khi CDC consumer xử lý mẻ lệch micro-second hoặc Kafka lag
	const clockSkewToleranceMs = 60000
```

---

### 4. File: `centralized-data-service/internal/repository/master/master_binding_repo.go`

#### Thay đổi 4.1: Thêm fallback liên kết `source_object_id` khi tra cứu Master Tables
- **Vị trí**: Dòng 72-84 (`ListMasterTablesByShadowTable`) và Dòng 86-111 (`ListMasterTablesByShadowIdentity`)
- **Mục tiêu**: Chống rớt trigger khi `mb.shadow_binding_id` là NULL nhưng `mb.source_object_id = sb.source_object_id`.
```go
// TRƯỚC (Dòng 78-80):
		`SELECT COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table AS master_fqn
		   FROM cdc_system.master_binding mb
		   JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
		  WHERE sb.shadow_table = ? AND mb.is_active = true AND mb.schema_status = 'approved'`,

// SAU:
		`SELECT COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table AS master_fqn
		   FROM cdc_system.master_binding mb
		   JOIN cdc_system.shadow_binding sb 
		     ON (mb.shadow_binding_id IS NOT NULL AND sb.id = mb.shadow_binding_id)
		     OR (mb.shadow_binding_id IS NULL AND sb.source_object_id = mb.source_object_id)
		  WHERE sb.shadow_table = ? AND mb.is_active = true AND mb.schema_status = 'approved'`,
```
Tương tự trong `ListMasterTablesByShadowIdentity` (Dòng 92-93):
```go
// TRƯỚC (Dòng 92-93):
		   FROM cdc_system.master_binding mb
		   JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id

// SAU:
		   FROM cdc_system.master_binding mb
		   JOIN cdc_system.shadow_binding sb 
		     ON (mb.shadow_binding_id IS NOT NULL AND sb.id = mb.shadow_binding_id)
		     OR (mb.shadow_binding_id IS NULL AND sb.source_object_id = mb.source_object_id)
```

---

### 5. File: `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go`

#### Thay đổi 5.1: Thêm fallback liên kết `source_object_id` trong `hasPostIngestSchedule`
- **Vị trí**: Dòng 97-105
- **Mục tiêu**: Đảm bảo gate `hasPostIngestSchedule` không trả về 0 khi `mb.shadow_binding_id` là NULL nhưng liên kết qua `source_object_id`.
```go
// TRƯỚC (Dòng 99-100):
		  FROM cdc_system.transmute_schedule ts
		  JOIN cdc_system.master_binding mb ON mb.id = ts.master_binding_id
		  JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
		 WHERE ts.mode = 'post_ingest' AND ts.is_enabled = true

// SAU:
		  FROM cdc_system.transmute_schedule ts
		  JOIN cdc_system.master_binding mb ON mb.id = ts.master_binding_id
		  JOIN cdc_system.shadow_binding sb 
		    ON (mb.shadow_binding_id IS NOT NULL AND sb.id = mb.shadow_binding_id)
		    OR (mb.shadow_binding_id IS NULL AND sb.source_object_id = mb.source_object_id)
		 WHERE ts.mode = 'post_ingest' AND ts.is_enabled = true
```

---

### 6. File: `centralized-data-service/internal/handler/master/debounce.go`

#### Thay đổi 6.1: Chống Deadlock & Nghẽn Channel trong `triggerFlushLocked`
- **Vị trí**: Dòng 125
- **Mục tiêu**: `td.flushCh <- td.tasks` khi buffer đầy (100 batches) sẽ làm hàm `triggerFlushLocked` bị block vĩnh viễn trong khi đang giữ `td.mu.Lock()`. Khi `td.mu` bị kẹt, toàn bộ `td.Add()` từ NATS subscriber bị treo cứng.
```go
// TRƯỚC (Dòng 125):
	td.flushCh <- td.tasks
	td.tasks = make([]NatsMsgTask, 0, td.maxSize)

// SAU:
	select {
	case td.flushCh <- td.tasks:
	default:
		td.logger.Warn("transmute debouncer flush channel saturated — spawning async worker dispatch",
			zap.String("master", td.tableName),
			zap.Int("tasks_count", len(td.tasks)))
		flushedTasks := td.tasks
		go func(tasks []NatsMsgTask) {
			select {
			case td.flushCh <- tasks:
			case <-time.After(5 * time.Second):
				td.logger.Error("transmute debouncer channel timeout — direct running batch to prevent data loss",
					zap.String("master", td.tableName))
				td.runTransmute(td.ctx, tasks)
			}
		}(flushedTasks)
	}
	td.tasks = make([]NatsMsgTask, 0, td.maxSize)
```

---

## TỔNG HỢP DANH SÁCH FILE THAY ĐỔI
| STT | Repository | Đường dẫn File | Loại thay đổi | Chức năng giải quyết |
|:---|:---|:---|:---|:---|
| 1 | `cdc-cms-web` | `src/components/ReconPipelineGrid.tsx` | Sửa (L282) | Tab Log Transmute: Bỏ param `sourceDb` thừa, kích hoạt fast index query (<5ms) |
| 2 | `cdc-cms-service` | `internal/infra/persistence/system/activity_log_read_repo_gorm.go` | Sửa (L50-67, L88-90, L94) | Sửa LATERAL join `tm_so` lấy `source_object_id` từ `master_binding`, bổ sung 30-day partition pruning |
| 3 | `centralized-data-service` | `internal/service/master/transmuter.go` | Sửa (L747-759, L340-349, L949-957, L1076) | Ngừng nuốt lỗi ở `bulkUpsertMaster`, kích hoạt `binarySearchSplit`, nâng dung sai OCC skew 60s |
| 4 | `centralized-data-service` | `internal/repository/master/master_binding_repo.go` | Sửa (L78-80, L92-93) | Fallback `source_object_id` khi lookup master bindings từ shadow table |
| 5 | `centralized-data-service` | `internal/handler/shadow/batch_buffer_fanout.go` | Sửa (L99-100) | Fallback `source_object_id` trong gate check `hasPostIngestSchedule` |
| 6 | `centralized-data-service` | `internal/handler/master/debounce.go` | Sửa (L125) | Non-blocking `flushCh` send để chống kẹt `td.mu` và chặn NATS subscriber |
