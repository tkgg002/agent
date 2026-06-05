# Report — Fix Snapshot Zero Records (2026-05-27)

## TL;DR
- 5 SOL patch site applied per `09_tasks_solution_snapshot.md`.
- Snapshot path bây giờ đo persistence-accurate counter từ PG `RowsAffected`.
- Flush error bubble lên `markProgressError` thay vì silent done.
- **Build + vet (handler) + test PASS** trên cả 3 service.
- Timer loop callers vẫn ignore return (best-effort drain, error đã log trong Flush).

## Patch applied (5 SOL)

### SOL-1: `BatchBuffer.batchUpsert` — signature `(written int, err error)`
- File: `centralized-data-service/internal/handler/batch_buffer.go`
- Line: 210-318 (was 196-306).
- TX path đếm `tx.Exec(...).RowsAffected` cộng dồn vào `chunkWritten`.
- Fallback sequential: per-row `db.Exec(...).RowsAffected`, chỉ cộng khi err == nil.
- Nếu fallback persist 0 row → return err (escalate). Trước: silent return nil.

### SOL-2: `BatchBuffer.Flush` — signature `(written int, err error)`
- File: `batch_buffer.go`
- Line: 158-208 (was 158-194).
- Consume `groupWritten, gerr` từ batchUpsert; cộng dồn `written`; capture first `err`.
- Log `persisted` per group cho operator triage.

### SOL-3: `EventHandler.FlushBatchBuffer` — signature `(written int, err error)`
- File: `centralized-data-service/internal/handler/event_handler.go`
- Line: 60-65 (was 60-63).
- Pass-through return từ `BatchBuffer.Flush`.

### SOL-4: `runSnapshot` consume Flush return (per-batch + final)
- File: `centralized-data-service/internal/handler/snapshot_runner_handler.go`
- Per-batch site: line 516-531 (was 516 + 521).
  - Flush err → `tripBreaker` (đã sẵn cơ chế DLQ + markProgressError).
  - `persisted < batchWritten` → log warn "partial persistence".
  - `rowsTotal += int64(persisted)` thay cho `rowsTotal += batchWritten`.
- Final flush: line 558-572 (was 550-555).
  - Flush err → `markProgressError` + return err.
  - `rowsTotal += int64(tailPersisted)` cho tail records còn trong buffer.

### SOL-5: Timer loop ignore với `_, _ =`
- File: `batch_buffer.go` line 145-155.
- 3 caller (`<-bb.ctx.Done()`, `<-ticker.C`, `<-bb.flushCh`) bọc `_, _ = bb.Flush()`.
- Best-effort drain, error đã log trong Flush.

## Files changed (3)

| File | Before LOC | After LOC | Delta |
|---|---|---|---|
| `centralized-data-service/internal/handler/batch_buffer.go` | ~415 | 452 | **+37** |
| `centralized-data-service/internal/handler/event_handler.go` | ~344 | 345 | **+1** |
| `centralized-data-service/internal/handler/snapshot_runner_handler.go` | ~862 | 878 | **+16** |
| **Tổng** | — | — | **+54 NET** |

## Verify

### Build
| Service | Command | Result |
|---|---|---|
| `centralized-data-service` | `go build ./...` | **PASS** (exit 0, no output) |
| `cdc-cms-service` | `go build ./...` | **PASS** (CMS_BUILD_OK) |
| `cdc-cms-web` | `npx vite build` | **PASS** (built in 718ms, 9 chunks) |

### Vet
- `go vet ./internal/handler/`: errors trong `pkgs/idgen/sonyflake.go:77,82` (`ResetForTest` copy `sync.Once`) — **pre-existing, không liên quan patch**. Code không sửa, lỗi đã tồn tại trước.

### Test
- `go test ./internal/handler/... -count=1 -timeout 60s` → **PASS** (`ok centralized-data-service/internal/handler 3.769s`).
- Tất cả test cases PASS, không có `--- FAIL`.

## Behavior diff (trước/sau fix)

### Trước (bug)
- Counter `rowsTotal` ← `batchWritten` (enqueue count từ `processEvent`).
- `BatchBuffer.Flush()` log err rồi drop.
- `markProgressDone(rowsTotal=161)` → activity_log status=success, shadow=0 rows.

### Sau (fix)
- Counter `rowsTotal` ← `persisted` (PG `RowsAffected`).
- `Flush()` return `(written, err)`; snapshot_runner consume cả 2.
- Nếu Flush err → `tripBreaker` (per-batch) hoặc `markProgressError` (final).
- `markProgressDone` chỉ fire khi Flush nil err **và** persisted reflect đúng.

## Cross-reference
- **Lesson** `lessons.md` 2026-05-26 line 3417-3421 "Define DoD at the destination" — bug hôm nay là case study trực tiếp, không cần lesson mới.
- **Workspace** `bug-first-snapshot-no-write-2026-05-26` đã fix layer 1 (`HandleRaw` return rows); workspace hôm nay fix layer 2-4 (Flush chain).
- **Workspace** `audit-shadow-create-bugs-2026-05-27` đã fix DDL Bug #2 (system cols). Fix Plan A độc lập nhưng cộng hưởng: nếu DDL còn thiếu cột, Plan A surface lỗi INSERT thay vì silent done.

## Files report-only (workspace docs)
| File | Trạng thái |
|---|---|
| `00_context.md` ... `10_gap_analysis.md` | Tạo phase audit |
| `report_audit_snapshot_zero_records_2026-05-27.md` | Tạo phase audit |
| `report_fix_snapshot_zero_records_2026-05-27.md` | **File này (fix phase)** |
| `05_progress.md` | APPEND Entry 4 |

## Sign-off
- [x] §6 Simplicity First: Plan A, 5 patch site, minimal-impact signature change.
- [x] §11 Memory Protection: `05_progress.md` APPEND only.
- [x] §12 Brain Code Prohibition: Muscle apply sau user verb "làm đi" — đúng quy trình.
- [x] §13 Lesson cross-check: reuse lesson hiện có, không nhân bản.
- [x] §14 Pre-flight: tất cả file workspace đã tạo, build + test PASS.

## Next step (chờ user)
- Optional: chạy `/security-agent` (§8 GEMINI gate).
- Optional: runtime verify bằng cách tạo snapshot mới trên FE → check log + count `psql`.
