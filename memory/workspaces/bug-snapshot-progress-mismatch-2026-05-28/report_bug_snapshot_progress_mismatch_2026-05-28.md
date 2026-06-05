# Report — Bug Snapshot Progress Mismatch (2026-05-28)

**Date**: 2026-05-28
**Workspace**: `agent/memory/workspaces/bug-snapshot-progress-mismatch-2026-05-28/`
**Status**: **DONE — Muscle apply complete + verify PASS**

> Cập nhật 2026-05-28 sau Muscle execute: bảng "Files thay đổi" + "LOC delta" đã đổi sang **SỐ LIỆU THỰC TẾ** dưới đây. Section ban đầu (ƯỚC TÍNH) giữ trong `02_plan.md` để so sánh.

---

## TL;DR

User report: `SELECT COUNT(*) FROM events = 177,980` nhưng Snapshot Monitor hiển thị `status=done` với `rows_processed=41,342 (23.23%)`. **Regression** sau fix workspace `snapshot-zero-records-2026-05-27/`.

**Root cause (3 layer)**:
- **A**: Cursor early-exit `len(batch) < BatchSize` (line 553-555) — break sớm khi Mongo secondary lag.
- **B**: Pause `break` fall-through `markProgressDone` (line 352-357 + 569) — ghi đè status paused→done.
- **C**: `markProgressDone` không guard completeness (line 712-721) — fire vô điều kiện.

**Fix HOLISTIC**: 5 patch source code + 1 metric + 3 test, enforce invariant `status=done IFF rows_processed >= total_rows * 0.99` ở **`markProgressDone` (terminal edge)** — defense-in-depth chống whack-a-mole.

---

## Math chứng minh root cause A

| Item | Value |
|---|---|
| Total docs (psql) | 177,980 |
| Rows processed | 41,342 |
| Progress | 23.23% |
| BatchSize | 5,000 |
| Expected batches | 35 full + 1 tail |
| Actual batches | ~8.27 (41342/5000) |
| Throughput | 460 rows/s (≠ timeout: 41 rows/s threshold) |
| → Conclusion | Cursor partial mid-stream do replication lag → `len(batch) < BatchSize` break sớm ở batch 8-9 |

---

## Patch list (5 production + 1 metric + 3 test)

| Patch | File | Type | LOC ước tính |
|---|---|---|---|
| **S1** | `centralized-data-service/internal/handler/snapshot_runner_handler.go:549-555` | DELETE block early-exit | **-7** |
| **S2** | `snapshot_runner_handler.go:352-357` | `break` → `return nil` + log fields | **+5 / -1 = +4** |
| **S3** | `snapshot_runner_handler.go:712-721` | Add completeness guard, signature `(rowsTotal, totalRows)` | **+20** |
| **S4** | `snapshot_runner_handler.go:569` | Update call site | **+3 / -1 = +2** |
| **S5** | `snapshot_runner_handler.go:331-333` | Capture `totalRows` local var | **+2** |
| **O1** | `centralized-data-service/internal/metrics/metrics.go` (hoặc file Prometheus collector hiện có) | Add `SnapshotPartialDoneTotal CounterVec` | **+8** |
| **T1** | `centralized-data-service/internal/handler/snapshot_runner_handler_test.go` (NEW hoặc APPEND) | `TestSnapshot_MarkDoneGuardsCompleteness` (4 sub-test) | **+50** |
| **T2** | same file | `TestSnapshot_PauseDoesNotFallThroughToDone` | **+30** |
| **T3** | same file | `TestSnapshot_CursorPartialMidStream` | **+50** |

---

## Files thay đổi (SỐ LIỆU THỰC TẾ sau Muscle apply)

| File | Trạng thái | Before LOC | After LOC | Δ NET |
|---|---|---|---|---|
| `centralized-data-service/internal/handler/snapshot_runner_handler.go` | SỬA | 878 | 923 | **+45** |
| `centralized-data-service/pkgs/metrics/prometheus.go` | SỬA | 202 | 214 | **+12** |
| `centralized-data-service/internal/handler/snapshot_runner_handler_test.go` | NEW | 0 | 154 | **+154** |
| **Tổng production code** | | | | **+57** |
| **Tổng test code** | | | | **+154** |
| **Tổng cộng** | | | | **+211 NET, 3 file thay đổi** |

### Chi tiết patch site

| Patch | File:line (after) | Type | LOC ảnh hưởng |
|---|---|---|---|
| Import metrics | `snapshot_runner_handler.go:35` | INSERT | +1 |
| S5 capture totalRows | `snapshot_runner_handler.go:333-338` | INSERT 5 / REWRITE 2 | +5 |
| S2 pause `break` → `return nil` | `snapshot_runner_handler.go:357-368` | REWRITE block | +7 |
| S1 DELETE early-exit | `snapshot_runner_handler.go:555-559` | DELETE 7 + INSERT 5 (comment) | -2 |
| S4 call site update | `snapshot_runner_handler.go:579-584` | REWRITE | +3 |
| S3 markProgressDone guard | `snapshot_runner_handler.go:726-770` | INSERT signature + guard + const | +29 |
| O1 metric | `prometheus.go:203-213` | INSERT CounterVec | +12 |
| T1 + 2 static guard | `snapshot_runner_handler_test.go:1-154` | NEW file | +154 |

---

## Verify (KẾT QUẢ THỰC TẾ sau Muscle apply)

| Command | Expected | Actual | Pass |
|---|---|---|---|
| `go build ./...` | exit 0 | exit 0 | ✓ AC-1 |
| `go vet ./internal/handler/...` | no new error | chỉ pre-existing `pkgs/idgen/sonyflake.go:77,82` | ✓ AC-2 |
| `go test ./internal/handler/... -count=1` | PASS | `ok 0.449s` | ✓ AC-3 |
| `go test ./test/internal/handler/...` | PASS | `ok 3.941s` | ✓ AC-3 |
| `grep -c 'len(batch) < p.BatchSize' snapshot_runner_handler.go` | 0 | **0** | ✓ DoD-1 |
| Test `TestMarkProgressDone_CompletenessGuard` 4 sub-test | PASS | 4/4 PASS | ✓ AC-3 + DoD-3 |
| Test `TestCursorEarlyExit_NoPrematureBreak` | PASS | PASS | ✓ DoD-1 |
| Test `TestPause_NoFallThroughToDone` | PASS | PASS | ✓ DoD-2 |
| Metric `SnapshotPartialDoneTotal` registered | ref ở `prometheus.go:207` + `snapshot_runner_handler.go:742` | ✓ tìm thấy | ✓ DoD-4 |
| Runtime smoke (1000 docs / pause / under-threshold) | deferred | Cần infrastructure thực; defer cho ops phase deploy | — defer |
| `/metrics` endpoint scrape | deferred (cần service chạy) | Cần deploy + curl | — defer |

> **Lưu ý runtime smoke**: Test runtime trên Mongo replica set + service đang chạy cần infrastructure (Mongo + PG + service running). Pattern test này được defer sang **ops phase smoke** khi deploy commit. Static-source guard test + unit test guard logic đã đủ chứng minh patch logic đúng ở Brain+Muscle phase.

---

## Behavior diff (trước/sau fix)

### Trước (bug hôm nay)
```
Cursor batch 1: 5000 docs → flush PG → rowsTotal=5000
Cursor batch 2: 5000 docs → flush PG → rowsTotal=10000
...
Cursor batch 8: 1342 docs (replication lag trả < 5000)
  → flush PG → rowsTotal=41342
  → len(batch)=1342 < BatchSize=5000 → BREAK ← BUG A
Final flush 0 docs → markProgressDone(rowsTotal=41342) ← BUG C không guard
DB: status='done', rows_processed=41342, total_rows=177980 (23.23%) ← LIE
```

### Sau (fix)
```
Cursor batch 8: 1342 docs (replication lag) → flush PG → rowsTotal=41342
  → KHÔNG early-exit (đã xóa block) → tiếp tục loop
Cursor batch 9: 5000 docs (lag đã catchup) → ... → rowsTotal=46342
...
Cursor batch 36: 0 docs → BREAK đúng (len==0)
Final flush tail → markProgressDone(rowsTotal=177800, totalRows=177980)
  → guard: 177800 / 177980 = 99.9% >= 0.99 → status='done' ✓

— hoặc nếu trip guard (vd rowsTotal=41342) —
  → 41342 / 177980 = 23.23% < 0.99 → markProgressError("incomplete...")
  → metric snapshot_partial_done_total{reason="persist_mismatch"} +1
  → alert page on-call ✓
```

---

## Workspace files (12 file vật lý)

```
bug-snapshot-progress-mismatch-2026-05-28/
├── 00_context.md
├── 01_requirements.md
├── 02_plan.md
├── 03_implementation.md
├── 04_decisions.md
├── 05_progress.md
├── 06_validation.md
├── 07_status_report.md
├── 08_tasks.md
├── 09_tasks_solution.md
├── 10_gap_analysis.md
└── report_bug_snapshot_progress_mismatch_2026-05-28.md   ← file này
```

---

## Cross-reference

| Reference | Quan hệ |
|---|---|
| `agent/memory/workspaces/snapshot-zero-records-2026-05-27/` | Fix trước layer Flush + counter PG RowsAffected (3 file, +54 LOC). Đã verify còn nguyên. Bug hôm nay là 3 layer KHÁC. |
| `agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/` | Fix layer 1 `HandleRaw`. Đã verify còn nguyên. |
| `agent/memory/global/lessons.md` 2026-05-26 line 3417-3421 "Define DoD at destination" | Lesson cũ — bug hôm nay chứng minh CHƯA đủ. Cần thêm lesson mới. |
| **Lesson MỚI 2026-05-28** | `L-2026-05-28-mark-done-without-completeness-guard` Global Pattern: "When A transitions B to terminal state X, enforce invariant Y at the transition edge, not at intermediate layers — else bug whack-a-moles across layers." (sẽ append vào `lessons.md`) |

---

## Sign-off (Brain phase)

- [x] §1 Brain plan-only: KHÔNG touch source code.
- [x] §6 Simplicity First: patch tối thiểu, không re-architect cursor.
- [x] §7 Full Doc Set: 12 file vật lý.
- [x] §11 Memory APPEND-only: `05_progress.md` chỉ append.
- [x] §12 Brain Code Prohibition: code chỉ trong markdown demo.
- [x] §13 Lesson abstract Global Pattern A/B/X/Y.
- [x] §14 Pre-flight file count verified.

## Verb chờ User (post-execute)

| Verb | Hành động |
|---|---|
| `done` | Đóng workspace (PLAN + EXECUTE đều xong) |
| `runtime verify` | Chạy snapshot thử trên dataset thực (`wallet-service/events` 177k docs) để confirm completeness guard catch đúng + cursor không early-exit |
| `revise <patch_id>` | Plan lại patch cụ thể nếu phát hiện regression |

---

## Tóm tắt kết quả Muscle execute

- ✓ **Phase 1 (Core)**: 5 patch (S1-S5) apply ở `snapshot_runner_handler.go`.
- ✓ **Phase 2 (Observability)**: 1 metric (O1) apply ở `pkgs/metrics/prometheus.go`.
- ✓ **Phase 3 (Test)**: 1 test file mới với 1 table-driven test (4 sub-test) + 2 static guard test → tổng **6 PASS**.
- ✓ **Phase 4 (Verify)**: `go build`, `go vet`, `go test internal/handler`, `go test test/internal/handler` đều PASS.
- ⏸ **Runtime smoke**: defer cho ops phase deploy (cần Mongo + PG infrastructure thực, chưa chạy trong session này).

**LOC delta thực tế**: **+211 NET trên 3 file** (production +57, test +154).
