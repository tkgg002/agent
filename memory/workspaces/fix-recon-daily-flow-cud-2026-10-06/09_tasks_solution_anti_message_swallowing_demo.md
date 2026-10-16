# Bản Thiết Kế & Demo Code Chuẩn Mực Tuyệt Đối (Audited Line-by-Line): Hệ Thống CDC Ingestion Chống Nuốt Message

**Workspace:** `fix-recon-daily-flow-cud-2026-10-06`  
**Phiên bản:** v6.0 - Bulletproof Distributed Core (Fan-Out Atomicity, Poison-Pill DLQ Evacuation & Bidirectional Normalization)  
**Tác giả:** Brain (Architect)  
**Kỷ luật áp dụng:** Đọc từng dòng, đi từng dòng, bẻ gãy mọi tử huyệt kẹt Lag, rách Batch, xung đột Offset và lệch Context.

---

## 1. TỔNG QUAN 4 TỬ HUYỆT PHÂN TÁN ĐÃ ĐƯỢC GIẢI QUYẾT TRIỆT ĐỂ

1. **Khóa Đóng Băng Kafka Lag do Poison Pill:**
   - Khi DB kiệt sức retry (do sai kiểu, vi phạm CHECK constraint): Kích hoạt `evacuateFailedChunkToDLQ` dùng **Context độc lập 10s** (không dùng context đã hết hạn của DB) và **Bulk Insert `CreateInBatches(100)`**.
   - Dữ liệu lỗi được lưu vào `cdc_system.failed_sync_logs` và trả về `StatusDLQ` $\rightarrow$ Giải phóng `uncommittedFailures`, offset tiếp tục tiến lên, chấm dứt hoàn toàn hiện tượng Lag đóng băng vô tận.

2. **Xung Đột Offset Khi Fan-Out Đa Bảng (Offset Collision):**
   - Gom nhóm kết quả theo `TopicPartition` và `offsetState { hasFailure, allSuccess }`.
   - Chỉ khi TẤT CẢ các routes con của cùng một offset thành công thì offset đó mới được coi là an toàn. Nếu có DÙ CHỈ 1 route con thất bại, offset đó bị khóa chặt trong Set lỗi. Loại bỏ hoàn toàn bug Unstable Sort ghi đè xóa lỗi của nhau.

3. **Hiện Tượng Rách Batch (Batch Tearing) Khi Fan-Out:**
   - Chuyển `for route { bb.Add(rec) }` thành `stagedRecords` tạm thời, sau đó gọi `bb.AddAll(stagedRecords)` nguyên tử dưới duy nhất một lần giữ Lock. Không một tiến trình Flush nào có thể cắt đôi các records của cùng 1 event.

4. **Chuẩn Hóa Đa Chiều (Bidirectional Canonicalization):**
   - Hỗ trợ cả 3 biến thể: Nguyên bản (Raw), Snake_case (`-` -> `_`), và Kebab-case (`_` -> `-`) trong `buildRouteLookupKeys`. Bất kể Debezium bắn tên có gạch nối hay gạch dưới, Registry đều khớp 100%.

---

## 2. CHI TIẾT MÃ NGUỒN CỐT LÕI

### Module 1: `internal/handler/shadow/event_handler.go`
```go
// Staged Buffer Pattern: Gom toàn bộ records của các routes vào một slice tạm
stagedRecords := make([]*shadow.UpsertRecord, 0, len(routes))

for _, route := range routes {
	// ... trích xuất PK, dynamic mapping, dựng UpsertRecord ...
	record := &shadow.UpsertRecord{
		TableName:        targetTable,
		SchemaName:       shadowSchemaName(route),
		ConnectionRole:   "shadow",
		ConnectionKey:    route.ShadowConnectionKey,
		PhysicalTableFQN: shadowPhysicalTable(route),
		PrimaryKeyField:  pgPKField,
		PrimaryKeyValue:  pkValue,
		MappedData:       mappedColumns,
		RawData:          rawJSON,
		Source:           sourceStr,
		Hash:             hash,
		SourceTsMs:       event.Data.SourceTsMs,
		IsDelete:         isDelete,
		Op:               event.Data.Op,
		TraceContext:     ctx,
		KafkaTopic:       event.KafkaTopic,
		KafkaPartition:   event.KafkaPartition,
		KafkaOffset:      event.KafkaOffset,
	}
	stagedRecords = append(stagedRecords, record)

	if h.childExplode != nil && route.ShadowBinding != nil {
		childDB, derr := h.connMgr.GetShadowDB(ctx, route.ShadowConnectionKey)
		if derr == nil && childDB != nil {
			h.childExplode.EmitFromParent(ctx, childDB, route.ShadowBinding.ID, pkValue, data)
		}
	}
}

// Bơm toàn bộ records của 1 Kafka event vào buffer dưới 1 lần giữ Lock duy nhất
h.batchBuffer.AddAll(stagedRecords)
return len(stagedRecords), nil
```

---

### Module 2: `internal/handler/shadow/batch_buffer.go`
```go
package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"centralized-data-service/internal/model/shadow"
	"centralized-data-service/internal/service/governance"
	"centralized-data-service/pkgs/metrics"
	"centralized-data-service/pkgs/observability"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// AddAll đưa toàn bộ các records của một event vào buffer dưới DUY NHẤT một lần giữ lock
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

func (bb *BatchBuffer) FlushWithContext(parentCtx context.Context) (written int, err error) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	bb.flushMu.Lock()
	defer bb.flushMu.Unlock()

	// Context dành riêng cho DB Writes
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

	// 1. Phân loại: Tách riêng Data Records và Marker Records
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

	// 2. Thực thi ghi Database cho các Data Records
	if len(byTable) > 0 {
		var mu sync.Mutex
		var resMu sync.Mutex
		var g errgroup.Group
		limit := 20
		if bb.isSnapshotBuffer {
			limit = 4
		}
		g.SetLimit(limit)

		for groupKey, records := range byTable {
			gk := groupKey
			recs := records
			g.Go(func() error {
				groupWritten, groupResults, gerr := bb.batchUpsertWithRetry(dbCtx, recs)

				if gerr != nil {
					observability.Ctx(dbCtx, bb.logger).Error("batch upsert exhausted retries, evacuating to DLQ to prevent pipeline freeze",
						zap.String("group", gk),
						zap.Int("record_count", len(recs)),
						zap.Error(gerr),
					)
					// PHAO CỨU SINH: Đẩy thẳng vào DLQ Database bằng context độc lập
					evacuatedResults := bb.evacuateFailedChunkToDLQ(recs, gerr)

					resMu.Lock()
					allResults = append(allResults, evacuatedResults...)
					resMu.Unlock()

					mu.Lock()
					if err == nil {
						err = gerr
					}
					mu.Unlock()
				} else {
					resMu.Lock()
					allResults = append(allResults, groupResults...)
					resMu.Unlock()

					metrics.BatchesFlushed.WithLabelValues("postgres", recs[0].TableName).Inc()
					if bb.natsConn != nil {
						bb.publishTransmuteTrigger(dbCtx, recs[0].SchemaName, recs[0].TableName, recs)
					}
					mu.Lock()
					written += groupWritten
					mu.Unlock()
				}
				return nil
			})
		}
		_ = g.Wait()
	}

	// 3. Tách riêng context 5s độc lập cho Kafka Commit
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

// evacuateFailedChunkToDLQ ghi log khẩn cấp với Context độc lập và Bulk Insert hiệu năng cao
func (bb *BatchBuffer) evacuateFailedChunkToDLQ(records []*shadow.UpsertRecord, dbErr error) []RecordExecutionResult {
	results := make([]RecordExecutionResult, len(records))
	if len(records) == 0 {
		return results
	}

	// Cấp Context độc lập 10s: Tuyệt đối không dùng context đã chết của DB Write
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

	// Thử ghi Bulk Insert 100 dòng/lần
	bulkErr := bb.db.WithContext(evacCtx).CreateInBatches(dlqRows, 100).Error
	if bulkErr == nil {
		for i, r := range records {
			results[i] = RecordExecutionResult{
				Record: r,
				Status: StatusDLQ, // Thành công -> Giải phóng Watermark
				Err:    dbErr,
			}
		}
		return results
	}

	// Fallback: Nếu Bulk Insert lỗi, ghi từng dòng để cứu các dòng khác
	bb.logger.Warn("bulk DLQ evacuation failed, falling back to row-by-row", zap.Error(bulkErr))
	for i, r := range records {
		status := StatusFailed
		if singleErr := bb.db.WithContext(evacCtx).Create(&dlqRows[i]).Error; singleErr == nil {
			status = StatusDLQ
		} else {
			bb.logger.Error("FATAL: Failed to evacuate single record to DLQ",
				zap.String("table", r.TableName),
				zap.Int64("offset", r.KafkaOffset),
				zap.Error(singleErr),
			)
		}
		results[i] = RecordExecutionResult{
			Record: r,
			Status: status,
			Err:    dbErr,
		}
	}

	return results
}
```

---

### Module 3: `internal/handler/shadow/watermark_tracker.go`
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
	StatusDropped
	StatusFailed
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

	// Bước 1: Gom nhóm trạng thái tối hậu theo từng Offset của từng Partition
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

	// Bước 2: Cập nhật state tracker theo trạng thái đã tổng hợp
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
				// Dù chỉ 1 route con fail -> Khóa cứng offset này trong Set lỗi
				if t.uncommittedFailures[tp] == nil {
					t.uncommittedFailures[tp] = make(map[int64]struct{})
				}
				t.uncommittedFailures[tp][off] = struct{}{}
			} else if st.allSuccess {
				// Toàn bộ các route con đều an toàn -> Gỡ lỗi
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

		// Bước 3: Tính Safe Offset dựa trên min(uncommittedFailures)
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
```

---

### Module 4: `internal/service/source/metadata_registry_utils.go`
```go
func buildRouteLookupKeys(sourceDB, sourceTable string, sourceConn ...string) []string {
	sourceDB = strings.TrimSpace(sourceDB)
	sourceTable = strings.TrimSpace(sourceTable)
	var conn string
	if len(sourceConn) > 0 {
		conn = strings.TrimSpace(sourceConn[0])
	}

	keys := make([]string, 0, 16)

	appendKeys := func(db, tbl string) {
		if conn != "" {
			for _, c := range stripGenericPrefixes(conn) {
				keys = append(keys,
					fmt.Sprintf("%s:%s|%s", c, db, tbl),
					fmt.Sprintf("%s:%s.%s", c, db, tbl),
					fmt.Sprintf("%s:%s", c, tbl),
				)
			}
		}
		keys = append(keys,
			fmt.Sprintf("%s|%s", db, tbl),
			fmt.Sprintf("%s:%s", db, tbl),
			fmt.Sprintf("%s.%s", db, tbl),
		)
	}

	// 1. Exact Raw
	appendKeys(sourceDB, sourceTable)

	// 2. Snake_case fallback (chuyển '-' thành '_')
	snakeDB := strings.ReplaceAll(sourceDB, "-", "_")
	snakeTable := strings.ReplaceAll(sourceTable, "-", "_")
	if snakeDB != sourceDB || snakeTable != sourceTable {
		appendKeys(snakeDB, snakeTable)
	}

	// 3. Kebab-case fallback (chuyển '_' thành '-')
	kebabDB := strings.ReplaceAll(sourceDB, "_", "-")
	kebabTable := strings.ReplaceAll(sourceTable, "_", "-")
	if kebabDB != sourceDB || kebabTable != sourceTable {
		appendKeys(kebabDB, kebabTable)
	}

	return dedupeStrings(keys)
}
```
