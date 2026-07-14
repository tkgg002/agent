# Kế hoạch Triển khai Kỹ thuật: Concurrency & Batching Optimization (Bullet-Proof Version)

Tài liệu này đặc tả chi tiết thiết kế mã nguồn, cấu trúc dữ liệu và logic xử lý của Sink Worker và Transmute Worker, tích hợp giải pháp triệt để cho **12 lỗ hổng phân tán (issues)** được phát hiện từ quá trình Red Teaming.

---

## 1. Thành phần: Shadow Sink Layer (`batch_buffer.go`)

### Giải quyết các Issues chặng Sink:
*   **Issue 1 (Tự sát chùm của `errgroup.WithContext`):** Sử dụng `errgroup.Group` thuần (không có context hủy dây chuyền). Hàm con chạy song song cho từng bảng ghi nhận lỗi riêng biệt và luôn trả về `nil` để tránh gây huỷ chùm sang các bảng thành công khác.
*   **Issue 5 (Batch Dilution):** Không lạm dụng cấu hình batch quá lớn (ví dụ 5000) gây trễ vô ích. Thay vào đó, đặt cấu hình vừa phải (`BatchSize = 500–1000`, `BatchTimeout = 50ms`) và tối ưu hóa ghi bằng cú pháp **Multi-row Upsert** của Postgres.

### Thiết kế mã nguồn `batch_buffer.go`:

```go
package shadow

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

func (bb *BatchBuffer) Flush() (written int, err error) {
	ctx := bb.ctx

	bb.mu.Lock()
	if len(bb.records) == 0 {
		bb.mu.Unlock()
		return 0, nil
	}
	batch := bb.records
	bb.records = make([]*shadow.UpsertRecord, 0, bb.maxSize)
	bb.lastFlush = time.Now()
	bb.mu.Unlock()

	// Phân nhóm bản ghi theo bảng
	byTable := make(map[string][]*shadow.UpsertRecord)
	for _, r := range batch {
		byTable[bb.groupKey(r)] = append(byTable[bb.groupKey(r)], r)
	}

	var mu sync.Mutex
	var g errgroup.Group // Sử dụng errgroup.Group thuần (không WithContext)
	g.SetLimit(20)       // Khống chế số lượng bảng ghi song song đồng thời

	for groupKey, records := range byTable {
		gk := groupKey
		recs := records
		g.Go(func() error {
			// Sử dụng parent ctx, không dùng cancelCtx bị liên đới lỗi của bảng khác
			groupWritten, gerr := bb.batchUpsert(ctx, recs)
			
			mu.Lock()
			defer mu.Unlock()

			if gerr != nil {
				// Ghi nhận lỗi cục bộ của bảng đó
				bb.logger.Error("batch upsert failed for table", 
					zap.Error(gerr), 
					zap.String("group", gk),
				)
				if err == nil {
					err = gerr // Lưu lỗi đầu tiên gặp phải
				}
			} else {
				metrics.BatchesFlushed.WithLabelValues("postgres", recs[0].TableName).Inc()
				if bb.natsConn != nil {
					ids := make([]string, 0, len(recs))
					for _, r := range recs {
						if r.PrimaryKeyValue != "" {
							ids = append(ids, r.PrimaryKeyValue)
						}
					}
					bb.publishTransmuteTrigger(ctx, recs[0].SchemaName, recs[0].TableName, ids)
				}
			}
			written += groupWritten
			return nil // Luôn trả về nil để bảo vệ các goroutine khác khỏi bị cancel
		})
	}

	_ = g.Wait()
	return written, err
}
```

---

## 2. Thành phần: Master Transmute Layer (`debounce.go` & `transmute_handler.go`)

### Giải quyết các Issues chặng Transmute:
*   **Issue 2 (Fallback mù quáng khi DB sập):** Sử dụng bộ lọc lỗi `isTransientError(err)`. Nếu DB sập hoặc timeout kết nối, gọi `Nak()` trả lại tin nhắn về NATS JetStream ngay lập tức. Chỉ fallback tìm Poison Pill khi gặp lỗi dữ liệu (Data/Constraint Error).
*   **Issue 3 (Trôi AckWait khi fallback tuần tự):** Trong luồng chạy fallback, định kỳ gọi `msg.InProgress()` để báo hiệu NATS kéo dài AckWait.
*   **Issue 4 (Tràn RAM/OOM tại TableDebouncer):** Thiết lập cơ chế Backpressure trong hàm `Add()`. Nếu bộ đệm vượt quá `maxSize * 2`, tạm giải phóng khóa Mutex và sleep ngắn để hãm tốc độ Fetch của NATS Pull.
*   **Issue 6 (Thiếu tính Idempotency):** Transmute logic bắt buộc thực hiện **Upsert Idempotent** (sử dụng cú pháp `ON CONFLICT (primary_key) DO UPDATE SET ...`).
*   **Issue 7 (Idle Debounce cho bảng thưa):** Thay vì debounce 1s cứng, sử dụng cơ chế **Flush after Idle** (nếu sau sự kiện cuối cùng X ms không có gì mới, hoặc tổng thời gian từ sự kiện đầu tiên đạt ngưỡng max thì flush ngay).
*   **Issue 8 (Thuật toán Fallback Poison Pill $O(N)$):** Triển khai thuật toán **Chia để trị (Binary Search Split)**. Chia đôi mẻ lỗi làm hai, chạy bulk-write cho mỗi nửa. Nửa thành công -> ACK nhanh. Nửa thất bại -> chia đôi tiếp cho tới khi cô lập được dòng lỗi ở kích thước 1.
*   **Issue 9 (Trôi AckWait do nghẽn Semaphore):** Luồng Fetch của NATS Pull chỉ fetch tin nhắn khi số lượng tin nhắn đang xử lý trong RAM chưa vượt quá ngưỡng giới hạn (`MaxAckPending` của NATS Consumer kết hợp với kiểm soát số lượng slot khả dụng).
*   **Issue 10 (Sập Pool kết nối DB khi Scale ngang):** Giới hạn pool kết nối theo công thức và khuyến nghị triển khai PgBouncer ở chế độ Transaction Pooling.
*   **Issue 11 (Mất thứ tự sự kiện - Event Ordering):** 
    *   Hạ tầng NATS cấu hình chia phân vùng (Partition Stream) theo Hashing khóa chính để đảm bảo các bản ghi của cùng một khóa chính luôn đi vào một Consumer duy nhất theo thứ tự FIFO.
    *   Phía DB: Áp dụng kiểm tra phiên bản/thời gian cập nhật `WHERE EXCLUDED.updated_at > master.updated_at` trong câu lệnh Upsert.
*   **Issue 12 (AckWait & MaxDeliver chưa đủ rộng):** Điều chỉnh cấu hình NATS Stream: `AckWait = 60s`, `MaxDeliver = 5`.

### Thiết kế mã nguồn đệm `debounce.go` (Gia cố toàn diện):

```go
package master

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsMsgTask struct {
	Msg *nats.Msg
	Req TransmuteRequest
}

type TableDebouncer struct {
	mu           sync.Mutex
	tableName    string
	tasks        []NatsMsgTask
	maxSize      int
	idleTimeout  time.Duration
	maxTimeout   time.Duration
	idleTimer    *time.Timer
	maxTimer     *time.Timer
	flushCh      chan []NatsMsgTask
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	sem          chan struct{} // Khống chế Concurrency
	runTransmute func(ctx context.Context, table string, ids []string) error
	failedLogger func(ctx context.Context, table string, id string, err error)
}

func NewTableDebouncer(
	ctx context.Context,
	tableName string,
	maxSize int,
	idleTimeout time.Duration,
	maxTimeout time.Duration,
	concurrencyLimit int,
	runTransmute func(ctx context.Context, table string, ids []string) error,
	failedLogger func(ctx context.Context, table string, id string, err error),
) *TableDebouncer {
	subCtx, cancel := context.WithCancel(ctx)
	td := &TableDebouncer{
		tableName:    tableName,
		maxSize:      maxSize,
		idleTimeout:  idleTimeout,
		maxTimeout:   maxTimeout,
		flushCh:      make(chan []NatsMsgTask, 50),
		ctx:          subCtx,
		cancel:       cancel,
		sem:          make(chan struct{}, concurrencyLimit),
		runTransmute: runTransmute,
		failedLogger: failedLogger,
	}

	td.wg.Add(1)
	go td.workerLoop()

	return td
}

// Add tích hợp Backpressure (Fix Issue 4)
func (td *TableDebouncer) Add(msg *nats.Msg, req TransmuteRequest) {
	for {
		td.mu.Lock()
		
		// Backpressure: Nếu đệm RAM đang quá tải (> 2 * maxSize), dừng nhẹ đầu vào
		if len(td.tasks) >= td.maxSize*2 {
			td.mu.Unlock()
			select {
			case <-td.ctx.Done():
				msg.Nak()
				return
			case <-time.After(10 * time.Millisecond):
				continue // Thử lại sau 10ms hãm phanh
			}
		}

		td.tasks = append(td.tasks, NatsMsgTask{Msg: msg, Req: req})

		// Gom đạt maxSize -> Kích hoạt flush ngay lập tức
		if len(td.tasks) >= td.maxSize {
			td.triggerFlushLocked()
			td.mu.Unlock()
			break
		}

		// Khởi tạo các timer phục vụ Flush after Idle (Fix Issue 7)
		td.setupTimersLocked()
		
		td.mu.Unlock()
		break
	}
}

func (td *TableDebouncer) setupTimersLocked() {
	// 1. Idle Timer: Reset mỗi khi có sự kiện mới
	if td.idleTimer != nil {
		td.idleTimer.Stop()
	}
	td.idleTimer = time.AfterFunc(td.idleTimeout, func() {
		td.mu.Lock()
		td.triggerFlushLocked()
		td.mu.Unlock()
	})

	// 2. Max Timer: Bắt đầu từ sự kiện đầu tiên của mẻ, không reset giữa chừng
	if td.maxTimer == nil {
		td.maxTimer = time.AfterFunc(td.maxTimeout, func() {
			td.mu.Lock()
			td.triggerFlushLocked()
			td.mu.Unlock()
		})
	}
}

func (td *TableDebouncer) triggerFlushLocked() {
	if len(td.tasks) == 0 {
		return
	}

	// Dừng các Timer
	if td.idleTimer != nil {
		td.idleTimer.Stop()
		td.idleTimer = nil
	}
	if td.maxTimer != nil {
		td.maxTimer.Stop()
		td.maxTimer = nil
	}

	// Gửi mẻ gom vào channel flush
	td.flushCh <- td.tasks
	td.tasks = make([]NatsMsgTask, 0, td.maxSize)
}

func (td *TableDebouncer) workerLoop() {
	defer td.wg.Done()

	for {
		select {
		case <-td.ctx.Done():
			// Flush nốt những gì còn trong RAM trước khi shutdown
			td.mu.Lock()
			td.triggerFlushLocked()
			td.mu.Unlock()
			return
		case batch := <-td.flushCh:
			td.processBatch(batch)
		}
	}
}

func (td *TableDebouncer) processBatch(batch []NatsMsgTask) {
	// Khống chế concurrency bằng Semaphore (Fix Issue 9)
	select {
	case td.sem <- struct{}{}:
	case <-td.ctx.Done():
		for _, t := range batch {
			t.Msg.Nak()
		}
		return
	}

	go func() {
		defer func() { <-td.sem }()

		// Gom tập các source ID duy nhất trong mẻ
		idSet := make(map[string]struct{})
		for _, t := range batch {
			for _, id := range t.Req.SourceIDs {
				if id != "" {
					idSet[id] = struct{}{}
				}
			}
		}

		uniqueIDs := make([]string, 0, len(idSet))
		for id := range idSet {
			uniqueIDs = append(uniqueIDs, id)
		}

		if len(uniqueIDs) == 0 {
			for _, t := range batch {
				t.Msg.Ack()
			}
			return
		}

		// Bước 1: Bulk Transmute song song (Idempotent ON CONFLICT)
		err := td.runTransmute(td.ctx, td.tableName, uniqueIDs)
		if err == nil {
			// Thành công toàn bộ lô
			for _, t := range batch {
				t.Msg.Ack()
			}
			return
		}

		// Bước 2: Phân loại lỗi kết nối DB / mạng để Fail-Fast (Fix Issue 2)
		if isTransientError(err) {
			td.logger.Warn("Transient DB error encountered. NAKing batch to retry later.", zap.Error(err))
			for _, t := range batch {
				t.Msg.Nak() // Trả lại NATS ngay, tuyệt đối không lùi về chạy tuần tự
			}
			return
		}

		// Bước 3: Lỗi dữ liệu / Poison Pill -> Kích hoạt thuật toán Chia Để Trị (Fix Issue 8)
		td.logger.Error("Poison pill detected in batch. Initiating Binary Search Split...", zap.Error(err))
		td.binarySearchSplit(batch)
	}()
}

// Thuật toán chia để trị (Binary Search Split) độ phức tạp O(log N) (Fix Issue 8)
func (td *TableDebouncer) binarySearchSplit(batch []NatsMsgTask) {
	// Nếu mẻ chỉ còn 1 bản ghi -> Bản ghi lỗi cụ thể (Poison Pill)
	if len(batch) == 1 {
		task := batch[0]
		td.logger.Error("Poison pill isolated successfully!", 
			zap.String("table", td.tableName),
			zap.Any("source_ids", task.Req.SourceIDs),
		)
		
		// Ghi nhận lỗi vào DLQ table
		for _, id := range task.Req.SourceIDs {
			td.failedLogger(td.ctx, td.tableName, id, fmt.Errorf("poison pill isolated"))
		}
		
		// Terminate tin nhắn rác vĩnh viễn trên NATS
		task.Msg.Term()
		return
	}

	// Chia đôi mẻ tin nhắn
	mid := len(batch) / 2
	left := batch[:mid]
	right := batch[mid:]

	// Định kỳ báo cáo InProgress kéo dài AckWait (Fix Issue 3)
	for _, t := range batch {
		t.Msg.InProgress()
	}

	// Xử lý song song / tuần tự hai nửa
	td.processSubBatch(left)
	td.processSubBatch(right)
}

func (td *TableDebouncer) processSubBatch(subBatch []NatsMsgTask) {
	idSet := make(map[string]struct{})
	for _, t := range subBatch {
		for _, id := range t.Req.SourceIDs {
			idSet[id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	err := td.runTransmute(td.ctx, td.tableName, ids)
	if err == nil {
		// Nửa này thành công -> ACK toàn bộ tin nhắn trong nửa này
		for _, t := range subBatch {
			t.Msg.Ack()
		}
		return
	}

	// Nếu nửa này thất bại, tiếp tục chia đôi đệ quy
	td.binarySearchSplit(subBatch)
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection pool") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "sqlstate 08")
}
```

---

## 3. Cấu hình NATS JetStream & DB Connection Pool (Fix Issue 6, 8, 9, 10, 11, 12)

### A. Cấu hình NATS Stream & Consumer (`server_setup.go`):
*   **MaxAckPending:** Đặt giới hạn `MaxAckPending = 1000` trên Consumer để hãm luồng NATS Pull. NATS sẽ không phân phối tin nhắn mới nếu số lượng tin nhắn chưa ACK vượt quá ngưỡng này.
*   **AckWait & MaxDeliver:** Nâng cấu hình Stream:
    *   `AckWait = 60s` (Đảm bảo biên an toàn khi DB bị chậm hoặc chạy đệ quy chia đôi mẻ).
    *   `MaxDeliver = 5` (Giới hạn tối đa số lần retry trước khi đưa tin nhắn vào JetStream DLQ mặc định nếu có).
*   **Event Ordering Hashing:** Cấu hình partition key trong Kafka và NATS Stream theo `hash(table_name + primary_key)` để đảm bảo tính tuần tự FIFO trên từng khóa chính.

### B. Cấu hình Connection Pool Postgres (Postgres Pool):
*   **Công thức giới hạn kết nối:**
    $$\text{Tổng kết nối} = \text{Số instances} \times (bb.\text{SetLimit}(20) + td.\text{SetLimit}(10))$$
    Nếu chạy 3 instances, tổng số kết nối mở ra có thể đạt $3 \times (20 + 10) = 90$ kết nối.
*   **Khuyên dùng PgBouncer:** Triển khai **PgBouncer** ở chế độ **Transaction Pooling** cho Master DB để chia sẻ connection pool tập trung và an toàn, bảo vệ Master DB khỏi sập kết nối khi scale ngang.
