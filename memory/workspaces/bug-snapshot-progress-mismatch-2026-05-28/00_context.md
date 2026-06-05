# 00_context — Bug Snapshot Progress Mismatch (Regression)

## Trigger
User report 2026-05-28 (sáng):

> `select count(*) from events` ra **177,980**, nhưng Snapshot Progress Monitor:
> - Status: **done**
> - Rows Processed: **41,342**
> - Progress: **41,342 / 177,980 = 23.23%**
> - Start: 10:46:59, Finish: 10:48:26 (~90s)
> - Target: `wallet-service / events`
> - Trace ID: `fe-snapshot-eb153f6a-0b62-416b-adfa-e2751fa13b22`
>
> "tao đã nói là số lượng phải là số record add vào success. mày làm gì mà cứ nhầm qua nhầm lại."

## Nature
**Regression / partial fix**. Workspace `snapshot-zero-records-2026-05-27` đã fix layer "BatchBuffer.Flush silent error" (rows_processed = PG RowsAffected). Tuy nhiên còn **2 root cause khác** không được patch:

### Root cause A — Cursor partial batch → break sớm
**File**: `centralized-data-service/internal/handler/snapshot_runner_handler.go:553-555`
```go
if len(batch) < p.BatchSize {
    break   // ← BUG: Mongo cursor trả về < batchSize không có nghĩa là exhausted
}
```
Lý do partial batch:
- `coll.Find(...)` với `SetLimit(int64(p.BatchSize))` + `SetReadPreference(readpref.SecondaryPreferred())`.
- Secondary Mongo replication lag → cursor trả về < limit dù collection còn data.
- `cursor.All(cursorCtx, &batch)` với `cursorCtx = context.WithTimeout(ctx, 2*time.Minute)` — timeout có thể trả partial + nil err.

### Root cause B — Pause break fall-through vào markProgressDone
**File**: `snapshot_runner_handler.go:353-356` + `:569`
```go
if isPaused.Load() {
    r.db.WithContext(ctx).Exec("UPDATE ... SET status='paused' ...")
    break   // ← thoát for loop nhưng tiếp tục chạy đến markProgressDone
}
// ... after for loop:
// :561 FlushBatchBuffer()
// :569 markProgressDone(ctx, progressID, rowsTotal)  ← ghi đè paused → done
```

### Root cause C — `markProgressDone` không guard `rowsTotal == total_rows`
**File**: `snapshot_runner_handler.go:712-721`
```go
func (r *SnapshotRunner) markProgressDone(ctx context.Context, progressID int64, rowsTotal int64) error {
    return r.db.WithContext(ctx).Exec(`
        UPDATE cdc_system.snapshot_progress
        SET status = 'done', rows_processed = ?, finished_at = NOW()
        WHERE id = ?
    `, rowsTotal, progressID).Error
}
```
Không có guard `if rowsTotal < total_rows * threshold → markProgressError`.

## Math chứng minh
- Total: 177,980 docs.
- Batch size: 5000 (line 360 `SetBatchSize(5000) + SetLimit(5000)`).
- Expected: 177,980 / 5000 = **35 batches full + 1 tail (2,980)**.
- Actual: 41,342 / 5000 ≈ **8.3 batches** → cursor return partial ở batch thứ ~8-9.
- Throughput: 41,342 / 90s ≈ 460 rows/s — không phải timeout (5000 docs trong 2 phút = 41 docs/s threshold, 460 dư sức).
- Khả năng cao: **replication lag secondary** trả về < 5000 ở batch nào đó → `len(batch) < BatchSize` → break → markProgressDone.

## Lesson cũ + bài học bị bỏ sót
- Lesson 2026-05-26 `bug-first-snapshot-no-write-2026-05-26`: "Define DoD at the destination".
- Lesson 2026-05-27 `snapshot-zero-records-2026-05-27`: counter ← PG RowsAffected.
- **Bài học bị bỏ sót**: DoD destination KHÔNG đủ — cần thêm DoD **completeness** (rows_processed phải khớp total trước khi mark done).
- Anti-pattern lặp lại: fix layer này → bug "trồi" sang layer khác (whack-a-mole). Cần patch HOLISTIC: invariant "status=done IFF rows_processed >= total * threshold" enforce ở 1 nơi duy nhất.

## Service liên quan
- `centralized-data-service/internal/handler/snapshot_runner_handler.go` (cursor + markProgressDone)
- `centralized-data-service/internal/handler/batch_buffer.go` (đã fix lần trước)
- `cdc-cms-web` snapshot-monitor page (chỉ display, không phải gốc)
- `cdc-cms-service` (NATS publish, không phải gốc)

## In-scope
- Fix cursor exhaustion detection (không dùng `len(batch) < BatchSize`).
- Fix pause fall-through.
- Add invariant guard `markProgressDone` → `markProgressError` nếu thiếu rows.
- Add metric `snapshot_partial_done_total` cho alert.

## Out-of-scope
- Sửa data đã mất.
- Re-architect snapshot.v2 toàn diện.
- Đổi Mongo deployment topology.

## Constraints từ User Note
- ✓ Đọc lesson trước.
- ✓ Theo `/agent` + GEMINI.md (Brain plan-only §1+§12).
- ✓ Plan rõ ràng + code demo chi tiết.
- ✓ Không cheat DB/config.
- ✓ Report cuối có **files thay đổi** + **LOC delta**.
- ✓ Verify service work (Muscle phase) trước khi báo done.
- ✓ Có file `report_*.md`.
