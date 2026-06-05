# 02_plan — Sequencing Fix Bug Snapshot Progress Mismatch

## Approach: HOLISTIC patch single-PR

Patch cả 3 root cause A+B+C trong 1 PR để tránh whack-a-mole (lesson 2026-05-28 mới — xem `agent/memory/global/lessons.md`).

## Phase sequencing

### Phase 1 — Core Patch (Muscle, ~2h)
| Step | File | Hành động | LOC ước tính |
|---|---|---|---|
| S1 | `snapshot_runner_handler.go:553-555` | XÓA block `if len(batch) < p.BatchSize { break }` | -3 |
| S2 | `snapshot_runner_handler.go:352-357` | Sau pause break, thêm `return nil` trước khi loop kết thúc — chuyển `break` thành `return nil` (vì pause = terminal cho run này, không phải tạm pause cursor) | +1 |
| S3 | `snapshot_runner_handler.go:712-721` | Đổi signature `markProgressDone(ctx, progressID, rowsTotal, totalRows int64) error`; add guard: nếu `rowsTotal < totalRows * completenessThreshold` → gọi `markProgressError` thay vì UPDATE status='done' | +15 |
| S4 | `snapshot_runner_handler.go:569` | Cập nhật call site `markProgressDone` truyền `totalRows` (capture từ EstimatedDocumentCount line 331-333; lưu vào local var thay vì chỉ UPDATE DB) | +3 |
| S5 | `snapshot_runner_handler.go:328-333` | Lưu `totalRows` vào local var (hiện tại chỉ UPDATE DB rồi quên) | +2 |
| **Sub-total** | | | **+18, -3 = +15 NET** |

### Phase 2 — Observability (Muscle, ~1h)
| Step | File | Hành động | LOC ước tính |
|---|---|---|---|
| O1 | `centralized-data-service/internal/metrics/metrics.go` (hoặc tương đương) | Add Prometheus counter `SnapshotPartialDoneTotal` với label `reason` | +6 |
| O2 | `snapshot_runner_handler.go:712-721` markProgressDone | Increment metric khi guard trip với `reason="persist_mismatch"` | +1 |
| O3 | `snapshot_runner_handler.go` pause path | Increment metric `reason="pause_fallthrough"` chỉ khi pause xảy ra trước khi đạt threshold | +1 |
| **Sub-total** | | | **+8 NET** |

### Phase 3 — Test (Muscle, ~2h)
| Step | File | Hành động | LOC ước tính |
|---|---|---|---|
| T1 | `snapshot_runner_handler_test.go` (NEW hoặc append) | `TestSnapshot_MarkDoneGuardsCompleteness` table-driven 3 case: complete / partial / under-threshold | +50 |
| T2 | Same file | `TestSnapshot_PauseDoesNotFallThroughToDone` mock NATS pause | +30 |
| T3 | Same file | `TestSnapshot_CursorPartialMidStream` mock Mongo cursor trả 4999 → 5000 → 5000 → 0 | +50 |
| **Sub-total** | | | **+130 NET (test only)** |

### Phase 4 — Verify (Muscle, ~30min)
- `go build ./...` → exit 0.
- `go vet ./internal/handler/...` → no new error.
- `go test ./internal/handler/... -count=1 -timeout 120s` → PASS.
- Runtime smoke: chạy snapshot trên dataset test `wallet-service/events` 1000 docs, kiểm tra `snapshot_progress.status = 'done'` IFF `rows_processed = 1000`.

## Total LOC estimate
- **Production code**: +23 NET (Phase 1+2).
- **Test code**: +130 NET (Phase 3).
- **TOTAL**: ~+153 LOC.
- **Files thay đổi**: 2-3 file (`snapshot_runner_handler.go`, `metrics.go` nếu chưa có, `snapshot_runner_handler_test.go`).

## Decision sequencing
- KHÔNG đổi ReadPreference (ADR-001 defer).
- KHÔNG đổi cursor sang `cursor.Next()` loop (overkill, `cursor.All` + `len(batch) == 0` đã handle đúng — root cause là check `< BatchSize`).
- Completeness threshold mặc định 0.99 (configurable qua env `SNAPSHOT_COMPLETENESS_THRESHOLD`); 1% margin để absorb concurrent insert race trong dataset đang ghi liên tục.

## Rollback plan
- Nếu Phase 1 fix gây regression (vd: snapshot không bao giờ tự exit do cursor.All hang) → revert commit; bug cũ vẫn tồn tại nhưng không xấu hơn.
- Metric Phase 2 độc lập, có thể giữ nguyên.

## Risk
| Risk | Mitigation |
|---|---|
| Mongo `cursor.All` với SetLimit có thể "vĩnh viễn" trả 0 nếu primary lag — nhưng `len(batch) == 0` đã handle (exit normal) | Đã có sẵn ở line 383-385, fix chỉ bỏ early-exit thừa |
| Threshold 0.99 quá strict khi dataset đang ghi nhanh (insert > snapshot rate) | Configurable; Muscle phase verify trên dataset thực |
| `totalRows` từ EstimatedDocumentCount là estimate (sai số ±1%) — chính lý do chọn threshold 0.99 thay vì 1.00 | ADR-002 ghi rõ tradeoff |
