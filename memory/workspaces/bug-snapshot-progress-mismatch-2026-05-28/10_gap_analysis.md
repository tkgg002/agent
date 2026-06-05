# 10_gap_analysis — Bug Snapshot Progress Mismatch

## Gap → Fix → Verify map

| # | Gap | File:line evidence | Fix (patch ID) | Verify command |
|---|---|---|---|---|
| **A** | Cursor early-exit `len(batch) < BatchSize` break sớm khi Mongo secondary trả partial do replication lag | `snapshot_runner_handler.go:553-555` | **S1** — XÓA block 549-555 | `go test ./internal/handler/ -run TestSnapshot_CursorPartialMidStream` PASS + `grep -n 'len(batch) < p.BatchSize' snapshot_runner_handler.go` = 0 |
| **B** | Pause `break` fall-through xuống `markProgressDone` ghi đè status paused→done | `snapshot_runner_handler.go:352-357` + `:569` | **S2** — `break` → `return nil` | `go test ./internal/handler/ -run TestSnapshot_PauseDoesNotFallThroughToDone` PASS + DB row status=`paused` |
| **C** | `markProgressDone` không guard `rowsTotal vs total_rows` | `snapshot_runner_handler.go:712-721` | **S3** — Thêm completeness guard threshold 0.99 + signature `(rowsTotal, totalRows)` | `go test ./internal/handler/ -run TestSnapshot_MarkDoneGuardsCompleteness` PASS (4 sub-test) |
| **D** | `totalRows` từ `EstimatedDocumentCount` chỉ UPDATE DB, không lưu local → callsite không có | `snapshot_runner_handler.go:331-333` | **S5** — Capture local var `totalRows` | `git diff snapshot_runner_handler.go` thấy `var totalRows int64` |
| **E** | Call site `markProgressDone` không truyền `totalRows` | `snapshot_runner_handler.go:569` | **S4** — Update call site | `git diff` thấy `markProgressDone(ctx, progressID, rowsTotal, totalRows)` |
| **F** | Không có observability cho partial-done event → không alert | (không có metric) | **O1** — Add `SnapshotPartialDoneTotal CounterVec` | `curl :8080/metrics \| grep snapshot_partial_done_total` xuất hiện |
| **G** | Test suite không cover scenario cursor partial mid-stream + pause + guard | (không có test) | **T1+T2+T3** — 3 test function | `go test ./internal/handler/... -count=1` PASS toàn bộ |

## Math verification (proof gap A là root cause)

| Số liệu | Giá trị | Tính toán |
|---|---|---|
| Total docs (psql) | 177,980 | `SELECT count(*) FROM events` |
| Rows processed (snapshot_progress) | 41,342 | UI display |
| Progress % | 23.23% | 41342 / 177980 |
| BatchSize | 5,000 | line 363 `SetLimit(5000)` |
| Expected batches | 35 full + 1 tail (2,980) | 177980 / 5000 |
| Actual batches | ~8-9 | 41342 / 5000 ≈ 8.27 |
| Duration | ~90s | 10:46:59 → 10:48:26 |
| Throughput | 460 rows/s | 41342 / 90 |
| 2-min timeout threshold | 41 docs/s | 5000 / 120 |
| → **Throughput dư xa timeout** | 460 >> 41 | ✓ Không phải timeout |
| → **Phải là `len(batch) < BatchSize` break sớm** | Gap A confirmed | Mongo secondary replication lag trả < 5000 ở batch ~8-9 |

## Strategy distribution sau fix

| Trước fix | Sau fix |
|---|---|
| Mọi snapshot có thể bị early-exit ngẫu nhiên do replication lag | Snapshot exit chỉ khi cursor thực sự cạn (`len(batch) == 0`) |
| Pause → status=done (bug) | Pause → status=paused (chính xác) |
| Mọi markProgressDone fire vô điều kiện | markProgressDone guard threshold; reject sang markProgressError nếu thiếu |
| Không có metric | `cdc_snapshot_partial_done_total{reason}` cho alert |

## Compliance evidence post-fix

Sau khi Muscle apply, hệ thống PHẢI có:
- ✓ `grep -n 'len(batch) < p.BatchSize' snapshot_runner_handler.go` = 0 match.
- ✓ Test integration cursor partial PASS — kết thúc với `rows_processed == total_rows ± 1%`.
- ✓ Test integration pause PASS — status=`paused` cho tới khi resume.
- ✓ Test integration completeness guard PASS — under-threshold → status=`error` reason="incomplete".
- ✓ Metric `cdc_snapshot_partial_done_total` expose ở `/metrics`.
- ✓ Lesson `L-2026-05-28-mark-done-without-completeness-guard` ghi vào `agent/memory/global/lessons.md`.

## Out-of-scope (defer roadmap)

- **DLQ tự retry snapshot incomplete**: cần queue + scheduler — defer phase 2.
- **UI "Retry incomplete snapshot" button**: cần FE work; auditor có thể restart manual qua CMS API.
- **Auto-fallback ReadPreference Primary nếu lag > threshold**: phức tạp; defer.
- **`CountDocuments` exact thay vì `EstimatedDocumentCount`**: cost 30s+ trên 177k docs; defer benchmark.
