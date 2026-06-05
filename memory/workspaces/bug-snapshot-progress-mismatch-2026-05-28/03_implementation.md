# 03_implementation — Code demo chi tiết (Brain plan-only, §12 compliant)

> **§12 BRAIN CODE PROHIBITION**: Code dưới đây là DEMO trong markdown để Muscle copy. Brain KHÔNG sửa file `.go` thực.

## Patch S1 — Bỏ early-exit `len(batch) < BatchSize`

### File: `centralized-data-service/internal/handler/snapshot_runner_handler.go`
### Range cũ: line 549-555

```go
// BEFORE (lines 549-555)



		// If we got fewer than batch_size rows we are done — Find returned
		// everything that matched.
		if len(batch) < p.BatchSize {
			break
		}
	}
```

```go
// AFTER (xóa 7 dòng — block + comment)


	}
```

**Rationale**: Cursor exhaustion đã có check `if len(batch) == 0 { break }` ở line 383-385 ở đầu mỗi iteration. Khi cursor trả batch cuối có < BatchSize doc, vòng lặp tiếp theo sẽ Find lại với `$gt: lastSeen` filter → trả `[]` → `len(batch) == 0` → exit chuẩn. Block ở line 553-555 là "optimization" sai dẫn đến break sớm khi Mongo secondary trả < limit do replication lag.

---

## Patch S2 — Pause break thành return nil

### File: `snapshot_runner_handler.go` line 352-357

```go
// BEFORE
	for {
		if isPaused.Load() {
			r.logger.Info("snapshot.v2 paused via control plane", zap.Int64("source_object_id", so.ID))
			r.db.WithContext(ctx).Exec("UPDATE cdc_system.snapshot_progress SET status = 'paused', updated_at = NOW() WHERE id = ?", progressID)
			break
		}
```

```go
// AFTER
	for {
		if isPaused.Load() {
			r.logger.Info("snapshot.v2 paused via control plane",
				zap.Int64("source_object_id", so.ID),
				zap.Int64("progress_id", progressID),
				zap.Int64("rows_processed_at_pause", rowsTotal))
			r.db.WithContext(ctx).Exec(
				"UPDATE cdc_system.snapshot_progress SET status = 'paused', updated_at = NOW() WHERE id = ?",
				progressID)
			return nil // do NOT fall through to final flush + markProgressDone
		}
```

**Rationale**: `break` thoát vòng `for` rồi tiếp tục chạy `tailPersisted, tailErr := r.eventHandler.FlushBatchBuffer()` (line 561) → `markProgressDone` (line 569) → ghi đè `status=paused` → `status=done`. `return nil` thoát hàm ngay, giữ nguyên status=paused; khi user resume, snapshot tiếp tục từ `last_seen_id` đã checkpoint.

---

## Patch S3+S4+S5 — markProgressDone guard completeness

### S5 — Capture totalRows vào local var (line 331-333)

```go
// BEFORE
	if estCount, err := coll.EstimatedDocumentCount(ctx); err == nil {
		r.db.WithContext(ctx).Exec("UPDATE cdc_system.snapshot_progress SET total_rows = ? WHERE id = ?", estCount, progressID)
	}
```

```go
// AFTER
	var totalRows int64
	if estCount, err := coll.EstimatedDocumentCount(ctx); err == nil {
		totalRows = estCount
		r.db.WithContext(ctx).Exec(
			"UPDATE cdc_system.snapshot_progress SET total_rows = ? WHERE id = ?",
			estCount, progressID)
	}
```

### S3 — markProgressDone signature + guard (line 712-721)

```go
// BEFORE
func (r *SnapshotRunner) markProgressDone(ctx context.Context, progressID int64, rowsTotal int64) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.snapshot_progress
		SET status = 'done',
		    rows_processed = ?,
		    updated_at = NOW(),
		    finished_at = NOW()
		WHERE id = ?
	`, rowsTotal, progressID).Error
}
```

```go
// AFTER
const snapshotCompletenessThreshold = 0.99 // 1% margin for concurrent inserts + estimate skew

func (r *SnapshotRunner) markProgressDone(
	ctx context.Context,
	progressID int64,
	rowsTotal int64,
	totalRows int64,
) error {
	// Completeness guard — status=done IFF rows_processed >= totalRows * threshold.
	// EstimatedDocumentCount can drift slightly under concurrent insert load,
	// so we keep a 1% margin instead of strict equality.
	if totalRows > 0 && float64(rowsTotal) < float64(totalRows)*snapshotCompletenessThreshold {
		metrics.SnapshotPartialDoneTotal.WithLabelValues("persist_mismatch").Inc()
		reason := fmt.Sprintf(
			"incomplete: rows_processed=%d expected>=%.0f (total_rows=%d, threshold=%.2f)",
			rowsTotal, float64(totalRows)*snapshotCompletenessThreshold, totalRows, snapshotCompletenessThreshold)
		r.logger.Error("snapshot.v2 mark-done rejected — completeness guard tripped",
			zap.Int64("progress_id", progressID),
			zap.Int64("rows_processed", rowsTotal),
			zap.Int64("total_rows", totalRows),
			zap.String("reason", reason))
		r.markProgressError(ctx, progressID, reason)
		return fmt.Errorf("completeness guard: %s", reason)
	}
	return r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.snapshot_progress
		SET status = 'done',
		    rows_processed = ?,
		    updated_at = NOW(),
		    finished_at = NOW()
		WHERE id = ?
	`, rowsTotal, progressID).Error
}
```

### S4 — Call site update (line 569)

```go
// BEFORE
	if err := r.markProgressDone(ctx, progressID, rowsTotal); err != nil {
		r.logger.Warn("snapshot.v2 mark-done failed",
			zap.Int64("progress_id", progressID), zap.Error(err))
	}
```

```go
// AFTER
	if err := r.markProgressDone(ctx, progressID, rowsTotal, totalRows); err != nil {
		r.logger.Warn("snapshot.v2 mark-done failed (or guard tripped)",
			zap.Int64("progress_id", progressID),
			zap.Int64("rows_processed", rowsTotal),
			zap.Int64("total_rows", totalRows),
			zap.Error(err))
	}
```

---

## Patch O1 — Prometheus metric

### File: `centralized-data-service/internal/metrics/metrics.go` (kiểm tra nếu tồn tại; nếu chưa có metrics package thì add vào file chứa Prometheus collector hiện có)

```go
// Add to var block
var (
	// ... existing metrics ...

	// SnapshotPartialDoneTotal counts mark-done attempts that fell short of
	// completeness threshold or were caught by pause-fallthrough guard.
	// Label "reason": cursor_short, pause_fallthrough, persist_mismatch.
	SnapshotPartialDoneTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cdc_snapshot_partial_done_total",
			Help: "Snapshot mark-done attempts blocked by completeness guard, by reason.",
		},
		[]string{"reason"},
	)
)
```

---

## Patch T1+T2+T3 — Test demo

### File: `centralized-data-service/internal/handler/snapshot_runner_handler_test.go` (APPEND)

```go
func TestSnapshot_MarkDoneGuardsCompleteness(t *testing.T) {
	cases := []struct {
		name       string
		rowsTotal  int64
		totalRows  int64
		wantStatus string
		wantErr    bool
	}{
		{"complete", 1000, 1000, "done", false},
		{"99pct_pass", 990, 1000, "done", false},
		{"under_99pct", 41342, 177980, "error", true},
		{"zero_total_skip_guard", 100, 0, "done", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, db := newTestRunnerWithSQLiteMock(t)
			progressID := seedSnapshotProgressRow(t, db, tc.totalRows)
			err := r.markProgressDone(context.Background(), progressID, tc.rowsTotal, tc.totalRows)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			gotStatus := readSnapshotStatus(t, db, progressID)
			if gotStatus != tc.wantStatus {
				t.Fatalf("status=%s want %s", gotStatus, tc.wantStatus)
			}
		})
	}
}

func TestSnapshot_PauseDoesNotFallThroughToDone(t *testing.T) {
	r, db := newTestRunnerWithSQLiteMock(t)
	mongoMock := newMongoMockWithDocs(t, 10000) // 2 batches of 5000
	natsPause := make(chan struct{})

	go func() { time.Sleep(50 * time.Millisecond); close(natsPause) }()
	go func() { <-natsPause; r.simulatePauseSignal() }()

	err := r.runSnapshotWith(mongoMock, &snapshotV2Payload{BatchSize: 5000})
	if err != nil { t.Fatalf("unexpected err: %v", err) }

	status := readSnapshotStatus(t, db, r.lastProgressID)
	if status != "paused" {
		t.Fatalf("status=%s want paused (pause fall-through bug)", status)
	}
}

func TestSnapshot_CursorPartialMidStream(t *testing.T) {
	// Mongo secondary replication lag: trả 4999 ở batch 2 dù còn data
	r, db := newTestRunnerWithSQLiteMock(t)
	mongoMock := newMongoMockWithBatches(t, [][]int{
		mkBatch(5000), mkBatch(4999), mkBatch(5000), mkBatch(2981), mkBatch(0),
	})
	err := r.runSnapshotWith(mongoMock, &snapshotV2Payload{BatchSize: 5000})
	if err != nil { t.Fatalf("unexpected err: %v", err) }

	rows := readSnapshotRowsProcessed(t, db, r.lastProgressID)
	if rows != 17980 {
		t.Fatalf("rows_processed=%d want 17980 (early-exit bug)", rows)
	}
	status := readSnapshotStatus(t, db, r.lastProgressID)
	if status != "done" {
		t.Fatalf("status=%s want done", status)
	}
}
```

> Helper `newTestRunnerWithSQLiteMock`, `newMongoMockWithBatches`, etc. — Muscle phase implement với SQLite in-memory + Mongo mock interface đã có pattern trong `event_handler_test.go`.

---

## Summary Patch Plan

| Patch | File | Hành động | LOC delta |
|---|---|---|---|
| S1 | snapshot_runner_handler.go:549-555 | DELETE early-exit block | -7 |
| S2 | snapshot_runner_handler.go:352-357 | `break` → `return nil` + log | +5 / -1 |
| S3 | snapshot_runner_handler.go:712-721 | Add completeness guard + new signature | +20 |
| S4 | snapshot_runner_handler.go:569 | Update call site | +3 / -1 |
| S5 | snapshot_runner_handler.go:331-333 | Capture totalRows local var | +2 |
| O1 | metrics package | Add `SnapshotPartialDoneTotal` CounterVec | +8 |
| T1+T2+T3 | snapshot_runner_handler_test.go | 3 test function | +130 (test) |

**Total**: ~+159 LOC production+test, **3 files thay đổi** (snapshot_runner_handler.go, metrics.go, snapshot_runner_handler_test.go).
