# 09_tasks_solution_snapshot — Code demo cho Plan A

> **Brain phase: DOCUMENT ONLY**. Code dưới đây là demo để Muscle apply sau khi user approve.

## SOL-1 — `BatchBuffer.batchUpsert` return `(written int, err error)`

**File**: `centralized-data-service/internal/handler/batch_buffer.go`
**Range**: line 196-306

### Before (giữ snippet ngắn)
```go
func (bb *BatchBuffer) batchUpsert(records []*model.UpsertRecord) (err error) {
    // ...
    chunkSize := 500
    for i := 0; i < len(records); i += chunkSize {
        // ...
        txErr := db.Transaction(func(tx *gorm.DB) error {
            // ...
            for _, subChunk := range groups {
                query, values := schemaAdapter.BuildBatchUpsertSQLInSchema(
                    schema, schemaName, tableName, effectivePK, subChunk,
                )
                if query == "" {
                    continue
                }
                if err := tx.Exec(query, values...).Error; err != nil {
                    return err
                }
            }
            return nil
        })

        if txErr != nil {
            // sequential fallback ...
            for _, r := range chunk {
                // ...
                if err := db.Exec(query, values...).Error; err != nil {
                    // log + failed_sync_logs
                } else {
                    metrics.SyncSuccess...
                }
            }
        } else {
            // ...
        }
    }

    return nil
}
```

### After
```go
func (bb *BatchBuffer) batchUpsert(records []*model.UpsertRecord) (written int, err error) {
    if len(records) == 0 {
        return 0, nil
    }
    first := records[0]
    tableName := first.TableName
    schemaName := bb.recordSchema(first)

    _, span := observability.ChildSpan(context.Background(), "cdc.batch_upsert",
        attribute.Int("cdc.batch_size", len(records)),
        attribute.String("cdc.target_table", tableName),
        attribute.String("cdc.target_schema", schemaName),
    )
    defer observability.EndSpan(span, &err)

    db := bb.resolveDB(first)
    schemaAdapter := bb.resolveSchemaAdapter(first, db)

    pk := first.PrimaryKeyField
    if err = schemaAdapter.PrepareForCDCInsertInSchema(schemaName, tableName, pk); err != nil {
        bb.logger.Error("prepare table failed",
            zap.String("schema", schemaName),
            zap.String("table", tableName),
            zap.Error(err),
        )
        return 0, err
    }

    schema := schemaAdapter.GetSchemaInSchema(schemaName, tableName)
    if schema == nil {
        return 0, fmt.Errorf("schema not found for %s.%s", schemaName, tableName)
    }

    effectivePK := first.PrimaryKeyField
    if effectivePK == "id" {
        if _, hasSourceID := schema.Columns["source_id"]; hasSourceID {
            effectivePK = "source_id"
        }
    }

    chunkSize := 500
    for i := 0; i < len(records); i += chunkSize {
        end := i + chunkSize
        if end > len(records) {
            end = len(records)
        }
        chunk := records[i:end]

        var chunkWritten int64
        txErr := db.Transaction(func(tx *gorm.DB) error {
            groups := make(map[string][]*model.UpsertRecord)
            for _, r := range chunk {
                sig := getSignature(r)
                groups[sig] = append(groups[sig], r)
            }

            for _, subChunk := range groups {
                query, values := schemaAdapter.BuildBatchUpsertSQLInSchema(
                    schema, schemaName, tableName, effectivePK, subChunk,
                )
                if query == "" {
                    continue
                }
                res := tx.Exec(query, values...)
                if res.Error != nil {
                    return res.Error
                }
                chunkWritten += res.RowsAffected
            }
            return nil
        })

        if txErr != nil {
            // Fallback sequential: đếm per-row thực tế persist được.
            chunkWritten = 0
            for _, r := range chunk {
                query, values := schemaAdapter.BuildUpsertSQLInSchema(
                    schema, bb.recordSchema(r), r.TableName, effectivePK,
                    r.PrimaryKeyValue, r.MappedData,
                    r.RawData, r.Source, r.Hash, r.SourceTsMs,
                )
                res := db.Exec(query, values...)
                if res.Error != nil {
                    bb.logger.Error("upsert failed",
                        zap.String("schema", bb.recordSchema(r)),
                        zap.String("table", tableName),
                        zap.String("pk", r.PrimaryKeyValue),
                        zap.Error(res.Error),
                    )
                    bb.db.Create(bb.buildFailedSyncLog(tableName, r, res.Error))
                    metrics.SyncFailed.WithLabelValues(tableName, "upsert", r.Source).Inc()
                } else {
                    chunkWritten += res.RowsAffected
                    metrics.SyncSuccess.WithLabelValues(tableName, "upsert", r.Source).Inc()
                }
            }
            // Sau fallback: nếu KHÔNG persist được row nào → escalate err.
            if chunkWritten == 0 {
                err = fmt.Errorf("batch upsert chunk failed: %w (fallback persisted 0 rows)", txErr)
                written += int(chunkWritten)
                return written, err
            }
        } else {
            for _, r := range chunk {
                metrics.SyncSuccess.WithLabelValues(tableName, "upsert", r.Source).Inc()
            }
        }

        written += int(chunkWritten)
    }

    return written, nil
}
```

**LOC delta**: ~+18 / ~-8.

**Tinh thần**: TX path đếm `RowsAffected`. Fallback path chỉ đếm thành công. Nếu fallback fail hoàn toàn → escalate error thay vì silent return nil.

---

## SOL-2 — `BatchBuffer.Flush` return `(written int, err error)`

**File**: `centralized-data-service/internal/handler/batch_buffer.go`
**Range**: line 158-194

### Before
```go
func (bb *BatchBuffer) Flush() {
    bb.mu.Lock()
    if len(bb.records) == 0 {
        bb.mu.Unlock()
        return
    }
    batch := bb.records
    bb.records = make([]*model.UpsertRecord, 0, bb.maxSize)
    bb.lastFlush = time.Now()
    bb.mu.Unlock()

    byTable := make(map[string][]*model.UpsertRecord)
    for _, r := range batch {
        byTable[bb.groupKey(r)] = append(byTable[bb.groupKey(r)], r)
    }

    for groupKey, records := range byTable {
        if err := bb.batchUpsert(records); err != nil {
            observability.Ctx(bb.ctx, bb.logger).Error("batch upsert failed",
                observability.ErrorField(err),
                observability.Attrs(
                    zap.String("group", groupKey),
                    zap.Int("count", len(records)),
                ),
            )
        } else {
            observability.Ctx(bb.ctx, bb.logger).Info("batch upsert ok",
                observability.Attrs(
                    zap.String("group", groupKey),
                    zap.Int("count", len(records)),
                ),
            )
            metrics.BatchesFlushed.WithLabelValues("postgres", records[0].TableName).Inc()
        }
    }
}
```

### After
```go
// Flush forces a synchronous upsert of the current batch.
// Returns (written, err):
//   - written: total rows actually persisted across all groups
//     (TX RowsAffected OR sequential fallback success count).
//   - err: first error encountered. Non-nil err does NOT mean written == 0;
//     some groups may have succeeded before the failing one.
//
// Callers that need persistence-accurate counter (snapshot runner) MUST
// consume both values. Timer loop callers may ignore via `_, _ = bb.Flush()`.
func (bb *BatchBuffer) Flush() (written int, err error) {
    bb.mu.Lock()
    if len(bb.records) == 0 {
        bb.mu.Unlock()
        return 0, nil
    }
    batch := bb.records
    bb.records = make([]*model.UpsertRecord, 0, bb.maxSize)
    bb.lastFlush = time.Now()
    bb.mu.Unlock()

    byTable := make(map[string][]*model.UpsertRecord)
    for _, r := range batch {
        byTable[bb.groupKey(r)] = append(byTable[bb.groupKey(r)], r)
    }

    for groupKey, records := range byTable {
        groupWritten, gerr := bb.batchUpsert(records)
        if gerr != nil {
            observability.Ctx(bb.ctx, bb.logger).Error("batch upsert failed",
                observability.ErrorField(gerr),
                observability.Attrs(
                    zap.String("group", groupKey),
                    zap.Int("count", len(records)),
                    zap.Int("persisted", groupWritten),
                ),
            )
            if err == nil {
                err = gerr // capture first error; continue draining other groups
            }
        } else {
            observability.Ctx(bb.ctx, bb.logger).Info("batch upsert ok",
                observability.Attrs(
                    zap.String("group", groupKey),
                    zap.Int("count", len(records)),
                    zap.Int("persisted", groupWritten),
                ),
            )
            metrics.BatchesFlushed.WithLabelValues("postgres", records[0].TableName).Inc()
        }
        written += groupWritten
    }
    return written, err
}
```

**LOC delta**: ~+14 / ~-8.

---

## SOL-3 — `EventHandler.FlushBatchBuffer` return `(written int, err error)`

**File**: `centralized-data-service/internal/handler/event_handler.go`
**Range**: line 60-63

### Before
```go
// FlushBatchBuffer forces the batch buffer to flush immediately.
func (h *EventHandler) FlushBatchBuffer() {
    h.batchBuffer.Flush()
}
```

### After
```go
// FlushBatchBuffer forces the batch buffer to flush immediately and reports
// the persistence-accurate count plus the first error encountered.
// Snapshot runner consumes both to mark progress with destination reality.
func (h *EventHandler) FlushBatchBuffer() (written int, err error) {
    return h.batchBuffer.Flush()
}
```

**LOC delta**: ~+4 / ~-3.

---

## SOL-4 — `runSnapshot` consume Flush return ở per-batch + final

**File**: `centralized-data-service/internal/handler/snapshot_runner_handler.go`

### Patch site 1 — per-batch flush (line 510-528)

#### Before
```go
// Happy-path DLQ flush (errors below CB threshold are still persisted
// for operator triage even when the snapshot continues).
flushDLQ()

// SYNC FLUSH: Force the batch buffer to write to Postgres before checkpointing.
// This guarantees that the UI progress bar accurately reflects the data
// actually persisted in the target DB, not just the data read from Mongo.
// It eliminates the "100% but still running" illusion.
r.eventHandler.FlushBatchBuffer()

// Count rows that processEvent actually routed (per-route fan-out).
// Using len(batch) here was the regression that hid the "first
// snapshot writes 0 rows" failure mode behind a success activity log.
rowsTotal += batchWritten
if batchTail != "" {
    lastSeen = batchTail
}
if err := r.checkpoint(ctx, progressID, lastSeen, rowsTotal); err != nil {
    r.logger.Warn("snapshot.v2 checkpoint failed",
        zap.Int64("progress_id", progressID), zap.Error(err))
}
```

#### After
```go
flushDLQ()

// SYNC FLUSH: persistence-accurate. Counter `batchWritten` chỉ đếm enqueue
// (processEvent fan-out); chỉ Flush mới biết PG persist thực sự bao nhiêu.
// Nếu Flush err → trip breaker (markProgressError) thay vì silent done.
persisted, flushErr := r.eventHandler.FlushBatchBuffer()
if flushErr != nil {
    return tripBreaker(fmt.Sprintf("flush after batch (enqueued=%d, persisted=%d): %v",
        batchWritten, persisted, flushErr))
}
if int64(persisted) < batchWritten {
    r.logger.Warn("snapshot.v2 partial persistence",
        zap.Int64("progress_id", progressID),
        zap.Int64("enqueued", batchWritten),
        zap.Int("persisted", persisted),
    )
}

// Counter từ destination reality, không từ enqueue.
rowsTotal += int64(persisted)
if batchTail != "" {
    lastSeen = batchTail
}
if err := r.checkpoint(ctx, progressID, lastSeen, rowsTotal); err != nil {
    r.logger.Warn("snapshot.v2 checkpoint failed",
        zap.Int64("progress_id", progressID), zap.Error(err))
}
```

### Patch site 2 — final flush trước markProgressDone (line 547-555)

#### Before
```go
// Force a flush of any remaining records in the batch buffer so we
// guarantee all inserts are complete before marking the progress as done.
r.eventHandler.FlushBatchBuffer()

if err := r.markProgressDone(ctx, progressID, rowsTotal); err != nil {
    r.logger.Warn("snapshot.v2 mark-done failed",
        zap.Int64("progress_id", progressID), zap.Error(err))
}
```

#### After
```go
// Final flush — drain tail records. Counter destination-accurate.
persisted, flushErr := r.eventHandler.FlushBatchBuffer()
if flushErr != nil {
    r.markProgressError(ctx, progressID,
        fmt.Sprintf("final flush failed (persisted=%d): %v", persisted, flushErr))
    return fmt.Errorf("final flush: %w", flushErr)
}
rowsTotal += int64(persisted)

if err := r.markProgressDone(ctx, progressID, rowsTotal); err != nil {
    r.logger.Warn("snapshot.v2 mark-done failed",
        zap.Int64("progress_id", progressID), zap.Error(err))
}
```

**LOC delta**: ~+20 / ~-4.

---

## SOL-5 — Timer loop / other callers ignore với `_, _ =`

**File**: `centralized-data-service/internal/handler/batch_buffer.go`
**Range**: line 141-156 (timer loop)

### Trước
```go
case <-bb.flushTicker.C:
    bb.Flush()
```

### Sau
```go
case <-bb.flushTicker.C:
    _, _ = bb.Flush() // timer loop: best-effort, lỗi đã log trong Flush
```

**LOC delta**: ~+1 / ~-1.

**Lưu ý**: Cần grep `Flush()` + `FlushBatchBuffer()` callers tránh sót.
- `grep -rn "\.Flush()" centralized-data-service/internal/handler/batch_buffer.go centralized-data-service/internal/handler/*.go`.
- Mọi caller ngoài snapshot path → wrap `_, _ =`.

---

## Tóm tắt LOC

| Patch | File | LOC delta dự kiến |
|---|---|---|
| SOL-1 | batch_buffer.go (batchUpsert) | +18 / -8 |
| SOL-2 | batch_buffer.go (Flush) | +14 / -8 |
| SOL-3 | event_handler.go (FlushBatchBuffer) | +4 / -3 |
| SOL-4 | snapshot_runner_handler.go (lines 510-528 + 547-555) | +20 / -4 |
| SOL-5 | batch_buffer.go (timer loop) | +1 / -1 |
| **Total** | **3 file** | **~+57 / ~-24 = +33 NET** |

## Compile-time checklist sau Muscle apply
- [ ] `go build ./...` PASS (no error).
- [ ] `go vet ./...` PASS.
- [ ] `go test ./internal/handler/... -count=1` test cases PASS (ignore pre-existing goleak).
- [ ] Grep còn caller `Flush()` / `FlushBatchBuffer()` chưa wrap → 0 kết quả ngoài SOL list.
