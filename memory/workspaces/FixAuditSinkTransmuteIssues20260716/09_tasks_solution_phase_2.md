# 09_tasks_solution_phase_2.md - Hồ sơ giải pháp kỹ thuật chi tiết Phase 2

Tài liệu này đặc tả chi tiết mã nguồn và thuật toán cụ thể cho từng file cần sửa đổi trong Phase 2, được điều chỉnh tương thích 100% với hạ tầng thực tế của dự án.

---

## 1. Hạng mục P2-1: Tối ưu hóa Concurrency (Sink & Transmute)

### A. Sửa đổi `internal/handler/shadow/batch_buffer.go`
- **Mục tiêu:**
  - Chạy ghi DB song song cho các bảng khác nhau bằng `errgroup` để tránh bottle-neck tuần tự.
  - Sử dụng background context timeout 10 giây để bảo vệ giao dịch ghi khi shutdown (không bị cancel/rollback do context hủy của parent).
- **Code thay đổi chi tiết:**

```go
func (bb *BatchBuffer) Flush() (written int, err error) {
	// Tạo timeout context độc lập từ Background để bảo vệ giao dịch ghi DB khi shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
	var g errgroup.Group
	g.SetLimit(20) // Khống chế tối đa 20 bảng chạy song song đồng thời

	for groupKey, records := range byTable {
		gk := groupKey
		recs := records
		g.Go(func() error {
			groupWritten, gerr := bb.batchUpsert(ctx, recs)
			
			mu.Lock()
			defer mu.Unlock()

			if gerr != nil {
				bb.logger.Error("batch upsert failed for table", 
					zap.Error(gerr), 
					zap.String("group", gk),
				)
				if err == nil {
					err = gerr // Lưu lỗi đầu tiên gặp phải
				}
			} else {
				metrics.SyncSuccess.WithLabelValues(recs[0].TableName, "upsert", recs[0].Source).Inc()
			}
			written += groupWritten
			return nil // Luôn trả về nil để bảo vệ các goroutine khác khỏi bị cancel
		})
	}

	_ = g.Wait()

	// Trigger callback commit Kafka offsets sau khi tất cả các bảng ghi song song xong
	if bb.onCommitOffsets != nil {
		groupOffsets := make(map[TopicPartition]int64)
		for _, r := range batch {
			if r.KafkaTopic != "" && r.KafkaOffset > 0 {
				tp := TopicPartition{Topic: r.KafkaTopic, Partition: r.KafkaPartition}
				if r.KafkaOffset > groupOffsets[tp] {
					groupOffsets[tp] = r.KafkaOffset
				}
			}
		}
		if len(groupOffsets) > 0 {
			bb.onCommitOffsets(ctx, groupOffsets)
		}
	}

	return written, err
}
```

---

### B. Sửa đổi `internal/handler/master/transmute_handler.go` và thêm `internal/handler/master/debounce.go`
- **Mục tiêu:**
  - Tích hợp `TableDebouncer` map theo master table trong `TransmuteHandler` để gom mẻ transmute tự động.
  - Tối ưu hóa Backpressure và giải quyết Poison Pill bằng chia để trị đệ quy phù hợp với NATS Core.
- **Tạo mới `internal/handler/master/debounce.go`:**

```go
package master

import (
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
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
	sem          chan struct{}
	logger       *zap.Logger
	runTransmute func(ctx context.Context, batch []NatsMsgTask)
}

func NewTableDebouncer(
	ctx context.Context,
	tableName string,
	maxSize int,
	idleTimeout time.Duration,
	maxTimeout time.Duration,
	concurrencyLimit int,
	logger *zap.Logger,
	runTransmute func(ctx context.Context, batch []NatsMsgTask),
) *TableDebouncer {
	subCtx, cancel := context.WithCancel(ctx)
	td := &TableDebouncer{
		tableName:    tableName,
		maxSize:      maxSize,
		idleTimeout:  idleTimeout,
		maxTimeout:   maxTimeout,
		flushCh:      make(chan []NatsMsgTask, 100),
		ctx:          subCtx,
		cancel:       cancel,
		sem:          make(chan struct{}, concurrencyLimit),
		logger:       logger,
		runTransmute: runTransmute,
	}

	td.wg.Add(1)
	go td.workerLoop()

	return td
}

func (td *TableDebouncer) Add(msg *nats.Msg, req TransmuteRequest) {
	for {
		td.mu.Lock()
		
		// Backpressure: Hãm đầu vào nếu RAM quá tải
		if len(td.tasks) >= td.maxSize*2 {
			td.mu.Unlock()
			select {
			case <-td.ctx.Done():
				return
			case <-time.After(10 * time.Millisecond):
				continue
			}
		}

		td.tasks = append(td.tasks, NatsMsgTask{Msg: msg, Req: req})

		if len(td.tasks) >= td.maxSize {
			td.triggerFlushLocked()
			td.mu.Unlock()
			break
		}

		td.setupTimersLocked()
		td.mu.Unlock()
		break
	}
}

func (td *TableDebouncer) setupTimersLocked() {
	if td.idleTimer != nil {
		td.idleTimer.Stop()
	}
	td.idleTimer = time.AfterFunc(td.idleTimeout, func() {
		td.mu.Lock()
		td.triggerFlushLocked()
		td.mu.Unlock()
	})

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
	if td.idleTimer != nil {
		td.idleTimer.Stop()
		td.idleTimer = nil
	}
	if td.maxTimer != nil {
		td.maxTimer.Stop()
		td.maxTimer = nil
	}
	td.flushCh <- td.tasks
	td.tasks = make([]NatsMsgTask, 0, td.maxSize)
}

func (td *TableDebouncer) workerLoop() {
	defer td.wg.Done()
	for {
		select {
		case <-td.ctx.Done():
			td.mu.Lock()
			td.triggerFlushLocked()
			td.mu.Unlock()
			return
		case batch := <-td.flushCh:
			select {
			case td.sem <- struct{}{}:
				go func() {
					defer func() { <-td.sem }()
					td.runTransmute(td.ctx, batch)
				}()
			case <-td.ctx.Done():
				return
			}
		}
	}
}

func (td *TableDebouncer) Stop() {
	td.cancel()
	td.wg.Wait()
}
```

- **Sửa đổi `internal/handler/master/transmute_handler.go` để tích hợp debouncer & Poison Pill Binary Search:**
  - Khai báo map `debouncers map[string]*TableDebouncer` và mutex bảo vệ nó.
  - Trong `HandleTransmute`, thay vì gọi goroutine xử lý ngay, ta đẩy vào debouncer tương ứng.
  - Viết method `runDebouncedTransmute` thực hiện bulk transmute và đệ quy `binarySearchSplit` khi gặp lỗi dữ liệu (constraint violation).

---

## 2. Hạng mục P2-2: Dọn dẹp bản ghi mồ côi (Flatten Orphan Cleanup)

### Sửa đổi `internal/service/master/transmute/flatten.go`
- Bổ sung logic tìm và soft-delete các dòng mồ côi khi mảng co rút:

```go
// flatten.go - logic bổ sung trong hàm Transform hoặc sau bulk upsert
// Giả định hàm có thông tin parent ID và danh sách master keys được tạo ra
func (f flatten) PruneOrphans(ctx context.Context, db *gorm.DB, tableName string, parentID string, activeKeys []string) error {
	var dbKeys []string
	err := db.WithContext(ctx).Table(tableName).
		Where("_source_id LIKE ? AND _deleted = false", parentID+fanoutSep+"%").
		Pluck("_source_id", &dbKeys).Error
	if err != nil {
		return err
	}

	activeMap := make(map[string]bool)
	for _, k := range activeKeys {
		activeMap[k] = true
	}

	var orphans []string
	for _, k := range dbKeys {
		if !activeMap[k] {
			orphans = append(orphans, k)
		}
	}

	if len(orphans) > 0 {
		return db.WithContext(ctx).Table(tableName).
			Where("_source_id IN (?)", orphans).
			Updates(map[string]any{
				"_deleted":     true,
				"_source_ts":   time.Now().UnixNano() / int64(time.Millisecond),
				"_updated_at":  time.Now(),
			}).Error
	}
	return nil
}
```

---

## 3. Hạng mục P2-4: Giải phóng Scheduler kẹt (Scheduler Stuck Cleanup)

### Sửa đổi `internal/service/master/transmute_scheduler.go`
- Bổ sung phương thức `cleanupStuckSchedules()` và gọi ở đầu mỗi hàm `tick()`:

```go
func (s *TransmuteScheduler) cleanupStuckSchedules(ctx context.Context) {
	// Reset các job bị kẹt 'running' quá 10 phút (2x interval)
	timeoutThreshold := time.Now().Add(-10 * time.Minute)
	
	res := s.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.transmute_schedule
		   SET last_status = 'failed',
		       last_error = 'Job execution timed out (worker lost or stuck)',
		       updated_at = NOW()
		 WHERE last_status = 'running'
		   AND last_run_at < ?`,
		timeoutThreshold,
	)
	
	if res.Error != nil {
		s.logger.Error("failed to cleanup stuck transmute schedules", zap.Error(res.Error))
		return
	}
	
	if res.RowsAffected > 0 {
		s.logger.Warn("cleaned up stuck transmute schedules", zap.Int64("count", res.RowsAffected))
	}
}
```
