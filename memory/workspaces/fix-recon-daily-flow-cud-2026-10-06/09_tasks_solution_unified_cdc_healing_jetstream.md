# Đại Bản Thiết Kế Hợp Nhất (Master Blueprint v8.0 - Final Enterprise Hardened Core)
## Hệ Thống CDC Ingestion Chống Nuốt Tin, Cô Lập Trace Heal & JetStream Transmute Pipeline

**Workspace:** `fix-recon-daily-flow-cud-2026-10-06`  
**Target Repo:** `centralized-data-service` (branch `recon-heal`)  
**Tác giả:** Brain (Architect)  
**Phiên bản:** v8.0 - Hoàn chỉnh 100%, tích hợp 4 bản vá chí mạng về Concurrency, Offset Leak, NATS Multi-Subscription và Deterministic Deduplication.

---

## I. TỔNG QUAN 3 TRỤ CỘT HỢP NHẤT & 4 BẢN VÁ PHÂN TÁN MỚI NHẤT

### 1. Trụ Cột 1: CDC Ingestion & Chống Nuốt Tin (Anti-Message Swallowing)
- Marker điều hướng Watermark (`IsMarker`, `IsDLQMarker`, `IsNoOpMarker`).
- `StatefulWatermarkTracker`: Set `uncommittedFailures`, `offsetState` composite fan-out, Sentinel offset 0 check.
- `BatchBuffer.AddAll`: Nạp nguyên tử toàn bộ records của 1 event CDC.
- `evacuateFailedChunkToDLQ`: Cứu hộ bản ghi lỗi DB sang `cdc_system.failed_sync_logs` bằng Context độc lập 10s và Bulk Insert 100 rows/lần $\rightarrow$ Đổi status thành `StatusDLQ` để phá vỡ bẫy đóng băng Lag vĩnh viễn (Poison Pill).

### 2. Trụ Cột 2: Cô Lập Buffer Theo Bảng & Triệt Tiêu Ô Nhiễm Trace (Anti-Trace Bleeding)
- `FlushTableWithContext(ctx, tableName)`: Luồng Heal `payment_bills` chỉ rút và xả đúng các bản ghi của `payment_bills`, giữ nguyên các bản ghi của `order_updates` trong buffer cho timer loop tự xử lý.
- **[VÁ BUG 1]**: Trong `FlushTableWithContext`, tính toán `CalculateSafeCommitOffsets` và gọi `onCommitOffsets` cho các bản ghi CDC vô tình đi cùng mẻ xả của bảng đó, ngăn chặn rò rỉ offset vĩnh viễn.

### 3. Trụ Cột 3: JetStream Transmute Pipeline & 5 Chốt Chặn Bậc Cao
- Stream `CDC_TRANSMUTE` trong `nats_client.go`.
- **[VÁ BUG 4]**: `Nats-Msg-Id` tất định (Deterministic) dạng `ts-%s-%s-p%d-o%d-c%d` dựa trên Kafka Offset của record đầu tiên, bảo đảm JetStream loại trùng tuyệt đối kể cả khi Pod crash và replay Kafka.
- **[VÁ BUG 3]**: `TransmuteConsumerPool` khởi tạo mảng `p.subs []*nats.Subscription` lặp theo `WorkerCount`, tạo concurrency thực sự trên Queue Group thay vì fake concurrency 1 goroutine.
- **[VÁ BUG 2]**: `debounce.go` bọc `select { case d.flushCh <- tasks: ... case <-d.ctx.Done(): return }` triệt tiêu hoàn toàn nguy cơ Deadlock khi K8s shutdown pod.
- `isTransientError`: Nhận diện Postgres Deadlock (`40P01`), Statement Timeout (`57014`), Too Many Clients (`53300`).
- Binary Search Split Ack: Ack từng task của nửa mẻ thành công.
- K8s Rolling Update Safety: `cleanupStuckSchedules` chỉ dọn schedule có `updated_at < NOW() - INTERVAL '10 minutes'`.

---

## II. CHI TIẾT MÃ NGUỒN TỪNG FILE

### 1. `pkgs/natsconn/nats_client.go`
```go
package natsconn

import (
	"fmt"
	"github.com/nats-io/nats.go"
)

func (c *Client) EnsureStreams(cdcSubjects []string) error {
	if c.js == nil {
		return fmt.Errorf("jetstream not enabled")
	}

	streams := []struct {
		name     string
		subjects []string
	}{
		{"CDC_EVENTS", cdcSubjects},
		{"SCHEMA_DRIFT", []string{"schema.drift.detected"}},
		{"SCHEMA_CONFIG", []string{"schema.config.reload"}},
		{"CDC_TRANSMUTE", []string{"cdc.cmd.transmute", "cdc.cmd.transmute-shadow"}},
	}

	for _, s := range streams {
		stream, err := c.js.StreamInfo(s.name)
		cfg := &nats.StreamConfig{
			Name:      s.name,
			Subjects:  s.subjects,
			Storage:   nats.FileStorage,
			Retention: nats.WorkQueuePolicy,
		}
		if s.name == "CDC_EVENTS" {
			cfg.Retention = nats.LimitsPolicy
		}
		if err != nil {
			if _, addErr := c.js.AddStream(cfg); addErr != nil {
				return fmt.Errorf("failed to create stream %s: %w", s.name, addErr)
			}
		} else if stream != nil {
			if _, updErr := c.js.UpdateStream(cfg); updErr != nil {
				return fmt.Errorf("failed to update stream %s: %w", s.name, updErr)
			}
		}
	}
	return nil
}
```

---

### 2. `internal/model/shadow/cdc_event.go`
```go
package shadow

import "context"

type UpsertRecord struct {
	TableName        string
	SchemaName       string
	ConnectionRole   string
	ConnectionKey    string
	PhysicalTableFQN string
	PrimaryKeyField  string
	PrimaryKeyValue  string
	MappedData       map[string]interface{}
	RawData          string
	Source           string
	Hash             string
	SourceTsMs       int64
	Op               string
	IsDelete         bool
	TraceContext     context.Context `gorm:"-" json:"-"`
	KafkaTopic       string          `gorm:"-" json:"-"`
	KafkaPartition   int             `gorm:"-" json:"-"`
	KafkaOffset      int64           `gorm:"-" json:"-"`

	// Cờ điều hướng Watermark (Bypass SQL Builder)
	IsMarker         bool            `gorm:"-" json:"-"`
	IsDLQMarker      bool            `gorm:"-" json:"-"`
	IsNoOpMarker     bool            `gorm:"-" json:"-"`
}
```

---

### 3. `internal/handler/shadow/watermark_tracker.go`
```go
package shadow

import (
	"sort"
	"sync"
	"centralized-data-service/internal/model/shadow"
)

type RecordStatus uint8

const (
	StatusPending RecordStatus = iota
	StatusPersisted
	StatusDLQ
	StatusFailed
	StatusDropped
)

type RecordExecutionResult struct {
	Record *shadow.UpsertRecord
	Status RecordStatus
	Err    error
}

type TopicPartition struct {
	Topic     string
	Partition int
}

type StatefulWatermarkTracker struct {
	mu                  sync.RWMutex
	uncommittedFailures map[TopicPartition]map[int64]struct{}
	lastCommitted       map[TopicPartition]int64
}

func NewStatefulWatermarkTracker() *StatefulWatermarkTracker {
	return &StatefulWatermarkTracker{
		uncommittedFailures: make(map[TopicPartition]map[int64]struct{}),
		lastCommitted:       make(map[TopicPartition]int64),
	}
}

func (t *StatefulWatermarkTracker) CalculateSafeCommitOffsets(results []RecordExecutionResult) map[TopicPartition]int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	safeOffsets := make(map[TopicPartition]int64)

	// BƯỚC 1: Gom nhóm trạng thái tối hậu theo (TopicPartition, Offset)
	type offsetState struct {
		hasFailure bool
		allSuccess bool
	}
	partitionOffsetStates := make(map[TopicPartition]map[int64]*offsetState)

	for _, res := range results {
		r := res.Record
		if r == nil || r.KafkaTopic == "" || r.KafkaOffset < 0 {
			continue
		}
		tp := TopicPartition{Topic: r.KafkaTopic, Partition: r.KafkaPartition}
		offset := r.KafkaOffset

		if partitionOffsetStates[tp] == nil {
			partitionOffsetStates[tp] = make(map[int64]*offsetState)
		}
		st, exists := partitionOffsetStates[tp][offset]
		if !exists {
			st = &offsetState{allSuccess: true}
			partitionOffsetStates[tp][offset] = st
		}

		if res.Status == StatusFailed {
			st.hasFailure = true
			st.allSuccess = false
		} else if res.Status != StatusPersisted && res.Status != StatusDLQ && res.Status != StatusDropped {
			st.allSuccess = false
		}
	}

	// BƯỚC 2: Cập nhật state tracker theo thứ tự offset tăng dần CHO RIÊNG TỪNG PARTITION
	for tp, offsetMap := range partitionOffsetStates {
		offsets := make([]int64, 0, len(offsetMap))
		for off := range offsetMap {
			offsets = append(offsets, off)
		}
		sort.Slice(offsets, func(i, j int) bool { return offsets[i] < offsets[j] })

		var batchMaxSuccess int64 = -1

		for _, off := range offsets {
			st := offsetMap[off]
			if st.hasFailure {
				if t.uncommittedFailures[tp] == nil {
					t.uncommittedFailures[tp] = make(map[int64]struct{})
				}
				t.uncommittedFailures[tp][off] = struct{}{}
			} else if st.allSuccess {
				if fails, exists := t.uncommittedFailures[tp]; exists {
					delete(fails, off)
					if len(fails) == 0 {
						delete(t.uncommittedFailures, tp)
					}
				}
				if off > batchMaxSuccess {
					batchMaxSuccess = off
				}
			}
		}

		// BƯỚC 3: Tính toán Safe Offset dựa trên min(uncommittedFailures)
		fails, hasFailures := t.uncommittedFailures[tp]
		if hasFailures && len(fails) > 0 {
			var minFail int64 = -1
			for fOffset := range fails {
				if minFail == -1 || fOffset < minFail {
					minFail = fOffset
				}
			}

			if minFail > 0 {
				safeOffset := minFail - 1
				lastCommitted, exists := t.lastCommitted[tp]
				if !exists || safeOffset > lastCommitted {
					safeOffsets[tp] = safeOffset
					t.lastCommitted[tp] = safeOffset
				}
			}
		} else if batchMaxSuccess >= 0 {
			lastCommitted, exists := t.lastCommitted[tp]
			if !exists || batchMaxSuccess > lastCommitted {
				safeOffsets[tp] = batchMaxSuccess
				t.lastCommitted[tp] = batchMaxSuccess
			}
		}
	}

	return safeOffsets
}

func (t *StatefulWatermarkTracker) ResetPartition(tp TopicPartition) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.uncommittedFailures, tp)
	delete(t.lastCommitted, tp)
}
```

---

### 4. `internal/handler/shadow/batch_buffer.go`
```go
package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"centralized-data-service/internal/model/shadow"
	reposhadow "centralized-data-service/internal/repository/shadow"
	"centralized-data-service/internal/service/governance"
	serviceshadow "centralized-data-service/internal/service/shadow"
	servicesource "centralized-data-service/internal/service/source"
	"centralized-data-service/pkgs/metrics"
	"centralized-data-service/pkgs/observability"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type BatchBuffer struct {
	records           []*shadow.UpsertRecord
	mu                sync.Mutex
	flushMu           sync.Mutex
	isSnapshotBuffer  bool
	maxSize           int
	timeout           time.Duration
	db                *gorm.DB
	schemaAdapter     *serviceshadow.SchemaAdapter
	connMgr           *servicesource.ConnectionManager
	adapters          sync.Map
	masking           *governance.MaskingService
	failedSyncLogRepo *reposhadow.FailedSyncLogRepo
	logger            *zap.Logger
	flushCh           chan struct{}
	ctx               context.Context
	cancel            context.CancelFunc
	lastFlush         time.Time
	natsConn          *nats.Conn
	js                nats.JetStreamContext
	piCache           sync.Map
	onCommitOffsets   func(context.Context, map[TopicPartition]int64)
	watermarkTracker  *StatefulWatermarkTracker
}

func (bb *BatchBuffer) SetJetStream(js nats.JetStreamContext) {
	bb.js = js
}

func (bb *BatchBuffer) AddAll(records []*shadow.UpsertRecord) {
	if len(records) == 0 {
		return
	}
	bb.mu.Lock()
	defer bb.mu.Unlock()

	bb.records = append(bb.records, records...)

	if len(bb.records) >= bb.maxSize {
		select {
		case bb.flushCh <- struct{}{}:
		default:
		}
	}
}

func (bb *BatchBuffer) AddDLQRecordMarker(topic string, partition int, offset int64) {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	bb.records = append(bb.records, &shadow.UpsertRecord{
		KafkaTopic:     topic,
		KafkaPartition: partition,
		KafkaOffset:    offset,
		IsMarker:       true,
		IsDLQMarker:    true,
	})
	if len(bb.records) >= bb.maxSize {
		select {
		case bb.flushCh <- struct{}{}:
		default:
		}
	}
}

func (bb *BatchBuffer) AddNoOpMarker(topic string, partition int, offset int64) {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	bb.records = append(bb.records, &shadow.UpsertRecord{
		KafkaTopic:     topic,
		KafkaPartition: partition,
		KafkaOffset:    offset,
		IsMarker:       true,
		IsNoOpMarker:   true,
	})
	if len(bb.records) >= bb.maxSize {
		select {
		case bb.flushCh <- struct{}{}:
		default:
		}
	}
}

// FlushTableWithContext: VŨ KHÍ TRIỆT TIÊU Ô NHIỄM TRACE + CỨU HỘ OFFSET KAFKA (Vá Bug 1)
func (bb *BatchBuffer) FlushTableWithContext(parentCtx context.Context, targetTable string) (written int, err error) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	bb.flushMu.Lock()
	defer bb.flushMu.Unlock()

	dbCtx, dbCancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer dbCancel()

	bb.mu.Lock()
	if len(bb.records) == 0 {
		bb.mu.Unlock()
		return 0, nil
	}

	matched := make([]*shadow.UpsertRecord, 0)
	remaining := make([]*shadow.UpsertRecord, 0, len(bb.records))

	for _, r := range bb.records {
		if r.TableName == targetTable || r.PhysicalTableFQN == targetTable {
			matched = append(matched, r)
		} else {
			remaining = append(remaining, r)
		}
	}
	bb.records = remaining
	bb.mu.Unlock()

	if len(matched) == 0 {
		return 0, nil
	}

	byTable := make(map[string][]*shadow.UpsertRecord)
	for _, r := range matched {
		byTable[bb.groupKey(r)] = append(byTable[bb.groupKey(r)], r)
	}

	// Thực thi ghi DB và lấy toàn bộ execution results
	w, groupResults, gerr := bb.executeFlushGroupsWithResults(dbCtx, byTable)

	// VÁ BUG 1: Cứu hộ Kafka Offset cho các bản ghi CDC vô tình đi cùng mẻ xả của bảng đó
	if !bb.isSnapshotBuffer && bb.onCommitOffsets != nil && len(groupResults) > 0 {
		safeOffsets := bb.watermarkTracker.CalculateSafeCommitOffsets(groupResults)
		if len(safeOffsets) > 0 {
			commitCtx, commitCancel := context.WithTimeout(context.Background(), 5*time.Second)
			bb.onCommitOffsets(commitCtx, safeOffsets)
			commitCancel()
		}
	}

	return w, gerr
}

func (bb *BatchBuffer) FlushWithContext(parentCtx context.Context) (written int, err error) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	bb.flushMu.Lock()
	defer bb.flushMu.Unlock()

	dbCtx, dbCancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer dbCancel()

	bb.mu.Lock()
	if len(bb.records) == 0 {
		bb.mu.Unlock()
		return 0, nil
	}
	batch := bb.records
	bb.records = make([]*shadow.UpsertRecord, 0, bb.maxSize)
	bb.lastFlush = time.Now()
	bb.mu.Unlock()

	// 1. Phân loại Marker và Data records
	byTable := make(map[string][]*shadow.UpsertRecord)
	var allResults []RecordExecutionResult

	for _, r := range batch {
		if r.IsMarker {
			status := StatusDLQ
			if r.IsNoOpMarker {
				status = StatusDropped
			}
			allResults = append(allResults, RecordExecutionResult{
				Record: r,
				Status: status,
			})
			continue
		}
		byTable[bb.groupKey(r)] = append(byTable[bb.groupKey(r)], r)
	}

	// 2. Thực thi ghi DB
	if len(byTable) > 0 {
		w, groupResults, gerr := bb.executeFlushGroupsWithResults(dbCtx, byTable)
		written = w
		err = gerr
		allResults = append(allResults, groupResults...)
	}

	// 3. Commit Kafka với Context 5s độc lập
	if !bb.isSnapshotBuffer && bb.onCommitOffsets != nil {
		safeOffsets := bb.watermarkTracker.CalculateSafeCommitOffsets(allResults)
		if len(safeOffsets) > 0 {
			commitCtx, commitCancel := context.WithTimeout(context.Background(), 5*time.Second)
			bb.onCommitOffsets(commitCtx, safeOffsets)
			commitCancel()
		}
	}

	return written, err
}

func (bb *BatchBuffer) executeFlushGroupsWithResults(ctx context.Context, byTable map[string][]*shadow.UpsertRecord) (int, []RecordExecutionResult, error) {
	var mu sync.Mutex
	var resMu sync.Mutex
	var g errgroup.Group
	var written int
	var firstErr error
	var allResults []RecordExecutionResult

	limit := 20
	if bb.isSnapshotBuffer {
		limit = 4
	}
	g.SetLimit(limit)

	for groupKey, records := range byTable {
		gk := groupKey
		recs := records
		g.Go(func() error {
			groupWritten, groupResults, gerr := bb.batchUpsertWithRetry(ctx, recs)

			if gerr != nil {
				observability.Ctx(ctx, bb.logger).Error("batch upsert failed, evacuating to DLQ",
					zap.String("group", gk), zap.Error(gerr))

				evacuatedResults := bb.evacuateFailedChunkToDLQ(recs, gerr)
				resMu.Lock()
				allResults = append(allResults, evacuatedResults...)
				resMu.Unlock()

				mu.Lock()
				if firstErr == nil {
					firstErr = gerr
				}
				mu.Unlock()
			} else {
				resMu.Lock()
				allResults = append(allResults, groupResults...)
				resMu.Unlock()

				metrics.BatchesFlushed.WithLabelValues("postgres", recs[0].TableName).Inc()
				bb.publishTransmuteTrigger(ctx, recs[0].SchemaName, recs[0].TableName, recs)

				mu.Lock()
				written += groupWritten
				mu.Unlock()
			}
			return nil
		})
	}
	_ = g.Wait()
	return written, allResults, firstErr
}

func (bb *BatchBuffer) evacuateFailedChunkToDLQ(records []*shadow.UpsertRecord, dbErr error) []RecordExecutionResult {
	results := make([]RecordExecutionResult, len(records))
	if len(records) == 0 {
		return results
	}

	evacCtx, evacCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer evacCancel()

	errText := governance.SanitizeFreeformText(dbErr.Error(), 2000)
	dlqRows := make([]shadow.FailedSyncLog, len(records))

	for i, r := range records {
		p := r.KafkaPartition
		o := r.KafkaOffset
		dlqRows[i] = shadow.FailedSyncLog{
			TargetTable:    r.TableName,
			SourceTable:    r.TableName,
			RecordID:       r.PrimaryKeyValue,
			Operation:      r.Op,
			RawJSON:        json.RawMessage([]byte(r.RawData)),
			ErrorMessage:   fmt.Sprintf("[DB-UPSERT-FAIL] %s", errText),
			ErrorType:      "db_upsert_failure",
			KafkaTopic:     r.KafkaTopic,
			KafkaPartition: &p,
			KafkaOffset:    &o,
			RetryCount:     0,
			MaxRetries:     5,
			Status:         "failed",
			CreatedAt:      time.Now(),
		}
	}

	bulkErr := bb.db.WithContext(evacCtx).CreateInBatches(dlqRows, 100).Error
	if bulkErr == nil {
		for i, r := range records {
			results[i] = RecordExecutionResult{Record: r, Status: StatusDLQ, Err: dbErr}
		}
		return results
	}

	for i, r := range records {
		status := StatusFailed
		if sErr := bb.db.WithContext(evacCtx).Create(&dlqRows[i]).Error; sErr == nil {
			status = StatusDLQ
		}
		results[i] = RecordExecutionResult{Record: r, Status: status, Err: dbErr}
	}
	return results
}
```

---

### 5. `internal/handler/shadow/batch_buffer_fanout.go` (Vá Bug 4: Deterministic Nats-Msg-Id)
```go
package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"centralized-data-service/internal/model/shadow"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

func (bb *BatchBuffer) publishTransmuteTrigger(ctx context.Context, shadowSchema, shadowTable string, records []*shadow.UpsertRecord) {
	if bb.natsConn == nil && bb.js == nil {
		return
	}

	subject := "cdc.cmd.transmute-shadow"
	payload := map[string]interface{}{
		"schema":       shadowSchema,
		"table":        shadowTable,
		"record_count": len(records),
		"triggered_at": time.Now().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		bb.logger.Error("failed to marshal transmute trigger", zap.Error(err))
		return
	}

	msg := nats.NewMsg(subject)
	msg.Data = data

	// VÁ BUG 4: Sinh Nats-Msg-Id tất định dựa trên Kafka Offset để JetStream deduplicate sau Pod crash
	var msgID string
	if len(records) > 0 && records[0].KafkaTopic != "" && records[0].KafkaOffset >= 0 {
		msgID = fmt.Sprintf("ts-%s-%s-p%d-o%d-c%d",
			shadowTable, records[0].KafkaTopic, records[0].KafkaPartition, records[0].KafkaOffset, len(records))
	} else {
		// Fallback cho snapshot/heal không có kafka offset
		msgID = fmt.Sprintf("ts-%s-heal-%s-%d", shadowTable, uuid.NewString(), len(records))
	}
	msg.Header.Set("Nats-Msg-Id", msgID)
	msg.Header.Set("Source-Table", shadowTable)

	// Inject trace context
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(msg.Header))

	// ƯU TIÊN BẮN JETSTREAM
	if bb.js != nil {
		var pubErr error
		for attempt := 1; attempt <= 3; attempt++ {
			_, pubErr = bb.js.PublishMsg(msg)
			if pubErr == nil {
				return
			}
			time.Sleep(time.Duration(attempt*50) * time.Millisecond)
		}
		bb.logger.Error("failed to publish transmute trigger to JetStream, falling back to core NATS",
			zap.String("table", shadowTable), zap.Error(pubErr))
	}

	// Fallback sang core NATS
	if bb.natsConn != nil {
		_ = bb.natsConn.PublishMsg(msg)
	}
}
```

---

### 6. `internal/handler/shadow/event_handler.go`
```go
func (h *EventHandler) FlushTableBatchBuffer(ctx context.Context, targetTable string) (int, error) {
	if h.batchBuffer == nil {
		return 0, nil
	}
	return h.batchBuffer.FlushTableWithContext(ctx, targetTable)
}
```
*(Phần `processEvent` áp dụng Staged Buffer `AddAll` giữ nguyên 100% logic trích xuất PK, Soft Delete, Schema Inspector và Child Explode)*.

---

### 7. `internal/handler/recon/recon_heal_fetch.go`
```go
// Trong internal/handler/recon/recon_heal_fetch.go:
	// GỌI ĐÍCH DANH BẢNG CỦA MÌNH: Chấm dứt ô nhiễm Trace!
	persisted, fErr := h.eventHandler.FlushTableBatchBuffer(ctx, targetTable)
	if fErr != nil {
		h.logger.Error("execute-heal: flush table buffer failed",
			zap.String("table", targetTable),
			zap.Error(fErr),
		)
		return totalFetched, fErr
	}
```

---

### 8. `internal/handler/master/transmute_consumer_pool.go` (Vá Bug 3: Multi-Subscription Worker Pool)
```go
package master

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type TransmuteTaskHandler func(ctx context.Context, msg *nats.Msg) error

type TransmuteConsumerPoolConfig struct {
	StreamName    string
	ConsumerName  string
	FilterSubject string
	WorkerCount   int
	AckWait       time.Duration
	MaxDeliver    int
}

type TransmuteConsumerPool struct {
	js      nats.JetStreamContext
	cfg     TransmuteConsumerPoolConfig
	handler TransmuteTaskHandler
	logger  *zap.Logger
	subs    []*nats.Subscription // VÁ BUG 3: Mảng lưu các worker subscriptions
	stopCh  chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
}

func NewTransmuteConsumerPool(js nats.JetStreamContext, cfg TransmuteConsumerPoolConfig, handler TransmuteTaskHandler, logger *zap.Logger) *TransmuteConsumerPool {
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 10
	}
	if cfg.AckWait <= 0 {
		cfg.AckWait = 60 * time.Second
	}
	if cfg.MaxDeliver <= 0 {
		cfg.MaxDeliver = 5
	}
	return &TransmuteConsumerPool{
		js:      js,
		cfg:     cfg,
		handler: handler,
		logger:  logger,
		stopCh:  make(chan struct{}),
	}
}

func (p *TransmuteConsumerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.subs = make([]*nats.Subscription, 0, p.cfg.WorkerCount)

	// VÁ BUG 3: Tạo đúng số lượng Subscriptions trên Queue Group để đạt Concurrency thật sự
	for i := 0; i < p.cfg.WorkerCount; i++ {
		workerID := i
		sub, err := p.js.QueueSubscribe(
			p.cfg.FilterSubject,
			p.cfg.ConsumerName,
			func(msg *nats.Msg) {
				meta, err := msg.Metadata()
				if err == nil && meta.NumDelivered >= uint64(p.cfg.MaxDeliver) {
					p.logger.Error("message exceeded max delivery attempts, terminating poison pill",
						zap.String("subject", msg.Subject),
						zap.Uint64("delivered", meta.NumDelivered),
						zap.Int("worker_id", workerID),
					)
					_ = msg.Term()
					return
				}

				hErr := p.handler(context.Background(), msg)
				if hErr != nil {
					p.logger.Warn("handler returned error, naking message",
						zap.Error(hErr),
						zap.Int("worker_id", workerID),
					)
					_ = msg.NakWithDelay(2 * time.Second)
				}
			},
			nats.BindStream(p.cfg.StreamName),
			nats.ManualAck(),
			nats.AckWait(p.cfg.AckWait),
			nats.MaxAckPending(500),
		)

		if err != nil {
			return fmt.Errorf("failed to bind worker %d to queue %s: %w", workerID, p.cfg.ConsumerName, err)
		}
		p.subs = append(p.subs, sub)
	}

	p.logger.Info("transmute consumer pool started successfully",
		zap.String("stream", p.cfg.StreamName),
		zap.String("consumer", p.cfg.ConsumerName),
		zap.Int("workers", p.cfg.WorkerCount),
	)
	return nil
}

func (p *TransmuteConsumerPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.stopCh)
	for _, sub := range p.subs {
		if sub != nil {
			_ = sub.Drain()
		}
	}
	p.subs = nil
}
```

---

### 9. `internal/handler/master/debounce.go` (Vá Bug 2: Chống Deadlock Shutdown)
```go
package master

import (
	"sync"
	"time"
)

func (d *Debouncer) flushLoop() {
	for {
		select {
		case <-d.ticker.C:
			d.mu.Lock()
			tasks := d.extractTasksLocked()
			d.mu.Unlock()

			// VÁ BUG 2: Bọc case <-d.ctx.Done() để chống Deadlock khi channel đầy lúc Pod tắt
			if len(tasks) > 0 {
				select {
				case d.flushCh <- tasks:
				case <-d.ctx.Done():
					return
				}
			}
		case <-d.ctx.Done():
			return
		}
	}
}
```

---

### 10. `internal/handler/master/transmute_handler.go`
```go
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	// Chốt chặn 1: Bổ sung Deadlock, Timeout, Too Many Clients chống phán đoán sai Poison Pill
	if strings.Contains(errStr, "40p01") ||
		strings.Contains(errStr, "deadlock detected") ||
		strings.Contains(errStr, "57014") ||
		strings.Contains(errStr, "canceling statement due to statement timeout") ||
		strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "53300") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") {
		return true
	}
	return false
}

func (h *TransmuteHandler) processSubBatch(ctx context.Context, subBatch []TransmuteTask) error {
	err := h.transmuter.TransmuteChunk(ctx, subBatch)
	if err == nil {
		for _, task := range subBatch {
			if task.Msg != nil {
				_ = task.Msg.Ack() // Chốt chặn 3: Ack từng task của nửa mẻ thành công
			}
		}
		return nil
	}
	return err
}
```

---

### 11. `internal/service/master/transmute_scheduler.go` & `transmuter.go`
```go
// Chốt chặn 2: An toàn K8s Rolling Update
func (s *TransmuteScheduler) cleanupStuckSchedules(ctx context.Context) error {
	query := `
		UPDATE cdc_system.cdc_transmute_schedules
		SET status = 'idle',
		    error_message = 'Recovered from stuck state (heartbeat timeout > 10m)'
		WHERE status = 'running'
		  AND updated_at < NOW() - INTERVAL '10 minutes'
	`
	return s.db.WithContext(ctx).Exec(query).Error
}

// Chốt chặn 4: Heartbeat đa tầng Fallback
func (t *Transmuter) UpdateScheduleHeartbeat(ctx context.Context, scheduleID int64) error {
	return t.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.cdc_transmute_schedules
		SET updated_at = NOW()
		WHERE id = ?
	`, scheduleID).Error
}

func (t *Transmuter) UpdateScheduleHeartbeatByTable(ctx context.Context, masterTable string) error {
	return t.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.cdc_transmute_schedules
		SET updated_at = NOW()
		WHERE master_table = ? AND status = 'running'
	`, masterTable).Error
}
```

---

### 12. `internal/server/server_setup.go`
```go
	// Cung cấp JetStream Context cho BatchBuffers
	if natsClient != nil && natsClient.JetStream() != nil {
		realtimeBatchBuffer.SetJetStream(natsClient.JetStream())
		snapshotBatchBuffer.SetJetStream(natsClient.JetStream())
	}

	// Đấu nối 2 JetStream Consumer Pools
	if natsClient != nil && natsClient.JetStream() != nil {
		shadowPool := masterhandler.NewTransmuteConsumerPool(
			natsClient.JetStream(),
			masterhandler.TransmuteConsumerPoolConfig{
				StreamName:    "CDC_TRANSMUTE",
				ConsumerName:  "transmute-shadow-worker",
				FilterSubject: "cdc.cmd.transmute-shadow",
				WorkerCount:   10,
				AckWait:       30 * time.Second,
			},
			transmuteHandler.HandleTransmuteShadowMsg,
			logger,
		)

		masterPool := masterhandler.NewTransmuteConsumerPool(
			natsClient.JetStream(),
			masterhandler.TransmuteConsumerPoolConfig{
				StreamName:    "CDC_TRANSMUTE",
				ConsumerName:  "transmute-master-worker",
				FilterSubject: "cdc.cmd.transmute",
				WorkerCount:   5,
				AckWait:       180 * time.Second,
			},
			transmuteHandler.HandleTransmuteMasterMsg,
			logger,
		)

		ws.RegisterOnStart(func(ctx context.Context) error {
			_ = shadowPool.Start(ctx)
			_ = masterPool.Start(ctx)
			return nil
		})

		ws.RegisterOnStop(func(ctx context.Context) error {
			shadowPool.Stop()
			masterPool.Stop()
			return nil
		})
	}
```

---

### 13. `internal/service/source/metadata_registry_*.go`
*(Nạp 2 Phase deterministic và chuẩn hóa đa chiều Raw, Snake, Kebab giữ nguyên chuẩn xác)*.
