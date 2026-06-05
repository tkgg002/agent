# Report — Audit Snapshot Zero Records (2026-05-27)

## TL;DR
- **Symptom**: Snapshot UI báo `done 161/161 100%` nhưng `export_jobs_2` 0 rows.
- **Root cause**: 4-layer silent-swallow chain ở `centralized-data-service`. Counter `rowsTotal` đo enqueue (BatchBuffer.Add), không đo persist (PG INSERT). `BatchBuffer.Flush()` log error rồi drop, không propagate cho snapshot_runner.
- **Fix proposal (Plan A)**: Plumb `(written int, err error)` qua chain `Flush → FlushBatchBuffer → runSnapshot`. Counter `rowsTotal` lấy từ `RowsAffected`. Nếu err → `markProgressError`, không silent done.
- **Impact**: 3 file, ~+57 / ~-24 LOC NET.
- **Trạng thái**: AUDIT COMPLETE. Chờ user verb "làm đi" → Muscle apply.

## Bug Chain (theo Layer)
| # | File | Line | Bug |
|---|---|---|---|
| 1 | `event_handler.go` | 173-175 | `Add(record); written := 1` — đếm enqueue, không persist |
| 2 | `snapshot_runner_handler.go` | 516, 521, 550 | `FlushBatchBuffer()` không tiêu thụ return; `rowsTotal += batchWritten` |
| 3 | `event_handler.go` | 61-63 | `FlushBatchBuffer()` void proxy, không có return |
| 4 | `batch_buffer.go` | 158-194 | `Flush()` log err rồi drop; `batchUpsert` return err nhưng bị nuốt |

## Code Demo
Xem `09_tasks_solution_snapshot.md` — 5 SOL patch site với code before/after.

## Files dự kiến thay đổi (Muscle phase)
| File | LOC delta dự kiến |
|---|---|
| `centralized-data-service/internal/handler/batch_buffer.go` | ~+33 / ~-17 |
| `centralized-data-service/internal/handler/event_handler.go` | ~+4 / ~-3 |
| `centralized-data-service/internal/handler/snapshot_runner_handler.go` | ~+20 / ~-4 |
| **Tổng** | **~+57 / ~-24 (NET +33)** |

## Files đã thay đổi (Audit phase)
| File | LOC delta |
|---|---|
| (source code) | **0 dòng** — audit-only theo §12 Brain Code Prohibition |
| workspace docs | +11 file (00..10 + report) |

## Verify plan (chạy ở Muscle phase)
1. `go build ./...` cho 3 service → PASS.
2. `go vet ./...` cho centralized-data-service → PASS.
3. `go test ./internal/handler/... -count=1` → test cases PASS (ignore pre-existing goleak).
4. Runtime: chạy snapshot trên registry mới, kỳ vọng:
   - Flush success → log `"batch upsert ok" persisted=N`, status=done với `rows_processed=N (PG thực tế)`.
   - Flush fail → log `"batch upsert failed"`, status=**error** với root cause SQL.

## Cross-reference
- **Lesson** `lessons.md` 2026-05-26 line 3417-3421 "Define DoD at the destination" — bug hôm nay là case study trực tiếp.
- **Workspace** `bug-first-snapshot-no-write-2026-05-26` — đã sửa Layer 1 (`HandleRaw` trả `(rows, err)` + route-empty CB). Workspace hôm nay sửa Layer 2-4 (Flush chain).
- **Workspace** `audit-shadow-create-bugs-2026-05-27` — fix DDL Bug #2 (thêm `_source_ts`, `_gpay_source_id`, `_gpay_deleted`). KHÔNG là nguyên nhân trực tiếp của bug hôm nay nhưng Plan A sẽ surface lỗi insert nếu có constraint failure.

## Sign-off
- [x] §7 Full Doc Set: 11 files trong workspace.
- [x] §11 Memory Protection: `05_progress.md` APPEND only.
- [x] §12 Brain Code Prohibition: zero source change phase audit.
- [x] §13 Lesson cross-check: dùng lesson cũ "Define DoD at the destination", không cần lesson mới.
- [x] §14 Pre-flight: tất cả file vật lý đã tạo, không có shadow doc.

## Next step
1. User approve "làm đi" → Muscle apply 5 SOL.
2. Build + vet + test verify.
3. Ghi `report_fix_snapshot_zero_records_2026-05-27.md` + APPEND Entry 4 vào `05_progress.md`.
4. Optional `/security-agent` per §8.
