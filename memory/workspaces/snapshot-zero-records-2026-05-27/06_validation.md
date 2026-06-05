# 06_validation — Audit Snapshot Zero Records

## Pre-fix baseline (audit phase)
*Phase audit: chỉ đọc code + verify chain, KHÔNG sửa code → baseline build chưa cần đo.*

## Post-fix verify plan (CHỜ user approve + Muscle apply)

### Step 1 — Build verify
| Service | Command | Expected |
|---|---|---|
| `centralized-data-service` | `go build ./...` | exit 0, no output |
| `centralized-data-service` | `go vet ./...` | exit 0, no warning |
| `cdc-cms-service` | `go build ./...` | exit 0 (zero-touch nhưng sanity) |
| `cdc-cms-web` | `npx vite build` | exit 0 (zero-touch nhưng sanity) |

### Step 2 — Test verify
| Suite | Command | Expected |
|---|---|---|
| handler package | `go test ./internal/handler/... -run 'BatchBuffer\|EventHandler\|Snapshot' -v` | tất cả test cases PASS (giữ baseline) |
| Full handler | `go test ./internal/handler/... -count=1` | KHÔNG có `--- FAIL` test case (goleak pre-existing không tính) |

### Step 3 — Runtime verify (sau khi deploy)
1. Tạo registry mới `export-jobs` → `export_jobs_3`.
2. Trigger snapshot từ FE `/snapshot-monitor`.
3. Quan sát log `centralized-data-service`:
   - Nếu Flush success → log `"batch upsert ok"` + counter persisted = batchWritten.
   - Nếu Flush fail → log `"batch upsert failed"` VÀ snapshot status → `error` (KHÔNG `done`).
4. Query `psql -h localhost -p 5436 -U postgres -d shadow -c "SELECT count(*) FROM shadow_xxx.export_jobs_3"` — kỳ vọng = số rows MongoDB collection.
5. Negative test: cố tình ngắt PG (hoặc inject SQL error qua bad column type) → snapshot phải FAIL với markProgressError chứa root cause SQL, không silent done.

### Step 4 — Counter consistency check
- `snapshot_progress.rows_processed` SAU FIX = số PG rows thực tế.
- `activity_log.rows_processed` SAU FIX = số PG rows thực tế.
- Mismatch giữa 2 bảng và shadow table count = bug regression.

## Files dự kiến thay đổi (preview)
| File | LOC delta dự kiến |
|---|---|
| `centralized-data-service/internal/handler/batch_buffer.go` | ~+15 / ~-3 |
| `centralized-data-service/internal/handler/event_handler.go` | ~+3 / ~-2 |
| `centralized-data-service/internal/handler/snapshot_runner_handler.go` | ~+12 / ~-2 |
| **Tổng** | **~30 LOC delta** |
