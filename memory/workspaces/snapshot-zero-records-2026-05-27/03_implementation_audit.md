# 03_implementation_audit — Audit Snapshot Zero Records

## Chain trace (verified bằng đọc code thực tế)

### Layer 1 — Snapshot dispatch
- File: `centralized-data-service/internal/handler/snapshot_runner_handler.go`
- Entry: NATS subscription `cdc.cmd.snapshot.v2` → `Handle()` line 113 → `runSnapshot()` line 165.
- Counter init: `rowsTotal int64` line 170.
- Pre-flight registry reload: line 281-307 (fix từ workspace `bug-first-snapshot-no-write-2026-05-26`, CÒN nguyên).

### Layer 2 — Mongo Find batch loop
- File: `snapshot_runner_handler.go` line 360-545 (`for` loop chính của snapshot).
- Mỗi vòng lặp đọc batch từ Mongo cursor → for mỗi doc:
  - line 463: `bson.MarshalExtJSON(doc, ...)`.
  - line 470: `buildSnapshotEnvelope(...)`.
  - line 471: `written, err := r.eventHandler.HandleRaw(ctx, subject, envelope)`.
  - line 478-489: nếu `written == 0` → `recordDocError("route empty", ...)` → CB. **PHẦN NÀY ĐÚNG.**
  - line 490: `batchWritten += int64(written)`.
- **BUT** — `written` đến từ `processEvent` chỉ đếm enqueue, không phải persist (xem Layer 4).

### Layer 3 — EventHandler dispatch
- File: `centralized-data-service/internal/handler/event_handler.go`
- `HandleRaw` line 66-87 → trả `(rows, err)` (signature đúng từ fix 2026-05-26).
- Implementation thực tế delegate sang `processEvent` line 94-191.

### Layer 4 — processEvent (LỚP BUG 1)
- File: `event_handler.go` line 94-191.
- Block fan-out: line 137-184 lặp qua mỗi route, build `UpsertRecord`, gọi `h.batchBuffer.Add(record)`.
- **Line 173-175** (CRITICAL):
  ```go
  h.batchBuffer.Add(record)
  written := 1
  totalWritten += written
  ```
- `batchBuffer.Add` chỉ append vào slice in-memory (`batch_buffer.go` line 127-139), KHÔNG persist.
- → `processEvent` trả `totalWritten` = số ENQUEUE, không phải số ROW PERSISTED.

### Layer 5 — Per-batch + final flush (LỚP BUG 2 — silent gate)
- File: `snapshot_runner_handler.go`
- Line 516 (per-batch sync flush):
  ```go
  r.eventHandler.FlushBatchBuffer()
  ```
  Không có return value tiêu thụ → kể cả Flush fail cũng không ai biết.
- Line 521: `rowsTotal += batchWritten` → tăng counter bằng số ENQUEUE.
- Line 550 (final flush trước markProgressDone):
  ```go
  r.eventHandler.FlushBatchBuffer()
  ```
  Cùng vấn đề.
- Line 552: `r.markProgressDone(ctx, progressID, rowsTotal)` — report rowsTotal là số đã ENQUEUE = 161, ko phải số đã PERSIST = 0.

### Layer 6 — EventHandler.FlushBatchBuffer (LỚP BUG 3 — proxy void)
- File: `event_handler.go` line 60-63:
  ```go
  func (h *EventHandler) FlushBatchBuffer() {
      h.batchBuffer.Flush()
  }
  ```
- Signature `void` → không thể propagate error cho snapshot_runner.

### Layer 7 — BatchBuffer.Flush (LỚP BUG 4 — silent swallow gốc)
- File: `centralized-data-service/internal/handler/batch_buffer.go` line 158-194.
- Signature `func (bb *BatchBuffer) Flush()` — KHÔNG return error, KHÔNG return count.
- Line 175-193 (CRITICAL):
  ```go
  for groupKey, records := range byTable {
      if err := bb.batchUpsert(records); err != nil {
          observability.Ctx(bb.ctx, bb.logger).Error("batch upsert failed", ...)
          // err bị nuốt ở đây — không propagate
      } else {
          observability.Ctx(bb.ctx, bb.logger).Info("batch upsert ok", ...)
          metrics.BatchesFlushed.WithLabelValues("postgres", records[0].TableName).Inc()
      }
  }
  ```
- → BUG GỐC: Flush log error rồi return void. Snapshot vẫn báo done với rowsTotal=161.

### Layer 8 — BatchBuffer.batchUpsert (đã có error return — không cần thay đổi nhiều)
- File: `batch_buffer.go` line 196-306.
- Signature `func (bb *BatchBuffer) batchUpsert(records []*model.UpsertRecord) (err error)` — đã return error.
- TX path line 252-275 → rollback toàn chunk nếu lỗi.
- Sequential fallback line 277-302 → per-row `db.Exec`; nếu fail thì write vào `failed_sync_logs` + bump `metrics.SyncFailed`. **Không bubble error lên Flush.**
- → batchUpsert có error return nhưng đơn giản: nếu tx fail VÀ fallback hoàn tất, hàm vẫn `return nil` ở line 305. Nên đếm `RowsAffected` để biết thực số persist.

## Failure-mode matrix (theo Layer)

| Layer | Counter | Reality | Khoảng cách |
|---|---|---|---|
| 4 processEvent | `totalWritten = số route` | enqueue async | OK nếu định nghĩa là "đã giao cho buffer" |
| 5 runSnapshot line 521 | `rowsTotal += batchWritten` | enqueue async | **mismatch** với persist |
| 7 Flush | (không có counter) | err log only | **silent swallow** |
| 8 batchUpsert | (chỉ return err) | tx OR fallback | tx fail + fallback success → return nil; tx fail + fallback per-row fail → vẫn return nil (do fallback không break) |

## Bằng chứng đối chiếu

| File | Line | Bằng chứng |
|---|---|---|
| `snapshot_runner_handler.go` | 516, 550 | Gọi `FlushBatchBuffer()` không tiêu thụ return |
| `snapshot_runner_handler.go` | 521 | `rowsTotal += batchWritten` (counter từ enqueue) |
| `snapshot_runner_handler.go` | 552 | `markProgressDone(ctx, progressID, rowsTotal)` |
| `event_handler.go` | 61-63 | `FlushBatchBuffer()` void proxy |
| `event_handler.go` | 173-175 | `Add(record); written := 1; totalWritten += written` |
| `batch_buffer.go` | 158-194 | `Flush()` void; err log only |
| `batch_buffer.go` | 277-302 | sequential fallback không bubble err |
| `batch_buffer.go` | 305 | `return nil` cuối hàm bất kể tx/fallback có lỗi từng chunk |

## Vì sao Bug 2 (audit-shadow-create-bugs) liên quan
- Trước fix 17:30 ICT hôm nay, shadow table tạo từ FE `/shadow` THIẾU `_source_ts`, `_gpay_source_id`, `_gpay_deleted`.
- INSERT của BatchBuffer dùng `BuildBatchUpsertSQLInSchema` → SQL build cột metadata từ `cdcMetadataColumns` của `schema_adapter.go` line 19-32 — slice này CÓ `_source_ts` nhưng KHÔNG có `_gpay_source_id`/`_gpay_deleted`.
- `PrepareForCDCInsertInSchema` line 121-225 lazy-add các cột trong `cdcMetadataColumns` (bao gồm `_source_ts`) → có thể đã tự fix runtime.
- ON CONFLICT dùng `pkField` (id), không phải `_gpay_source_id` → bug missing `_gpay_source_id` UNIQUE KHÔNG break insert chính.
- **Tóm lại**: Bug 2 KHÔNG nhất thiết là nguyên nhân INSERT fail. Nhưng silent-swallow chain (Layer 7) làm che giấu MỌI nguyên nhân fail. Plan A surface error → operator sẽ biết exact SQL error.
