# 08_tasks — Checklist Muscle Execute

> Muscle BẮT BUỘC tick từng task. KHÔNG báo done nếu chưa verify.

## Phase 1 — Core Patch (~2h)

- [ ] **S1**: Xóa block `snapshot_runner_handler.go` line 549-555 (`if len(batch) < p.BatchSize { break }`).
- [ ] **S2**: Sửa `snapshot_runner_handler.go` line 352-357 — đổi `break` thành `return nil` + thêm log `rows_processed_at_pause`.
- [ ] **S5**: Capture `totalRows` vào local var ở line 331-333 (trước UPDATE).
- [ ] **S3**: Đổi signature `markProgressDone(ctx, progressID, rowsTotal, totalRows int64) error` ở line 712-721 + thêm guard completeness threshold 0.99.
- [ ] **S4**: Update call site line 569 truyền `totalRows`.

## Phase 2 — Observability (~1h)

- [ ] **O0**: Locate file chứa Prometheus collector (likely `internal/metrics/metrics.go` hoặc tương đương; nếu không có, tạo mới).
- [ ] **O1**: Add `SnapshotPartialDoneTotal = promauto.NewCounterVec(...)` với label `reason`.
- [ ] **O2**: Inc metric `reason="persist_mismatch"` trong `markProgressDone` guard branch.
- [ ] **O3**: (Optional) Inc metric `reason="pause_fallthrough"` nếu phát hiện trường hợp pause xảy ra sát threshold (low priority).

## Phase 3 — Test (~2h)

- [ ] **T1**: Thêm `TestSnapshot_MarkDoneGuardsCompleteness` table-driven 4 case (complete, 99pct_pass, under_99pct, zero_total_skip_guard).
- [ ] **T2**: Thêm `TestSnapshot_PauseDoesNotFallThroughToDone` với mock NATS pause.
- [ ] **T3**: Thêm `TestSnapshot_CursorPartialMidStream` với mock Mongo batches `[5000, 4999, 5000, 2981, 0]`.
- [ ] **T4**: Implement helper `newTestRunnerWithSQLiteMock` / `newMongoMockWithBatches` (reuse pattern từ `event_handler_test.go` nếu có).

## Phase 4 — Verify (~30min)

- [ ] **V1**: `go build ./...` trong `centralized-data-service` — exit 0.
- [ ] **V2**: `go vet ./internal/handler/...` — no new error (pre-existing `pkgs/idgen/sonyflake.go` ignored).
- [ ] **V3**: `go test ./internal/handler/... -count=1 -timeout 120s` — PASS toàn bộ.
- [ ] **V4**: Runtime smoke test 1000 docs (xem `06_validation.md`).
- [ ] **V5**: Verify metric expose ở `/metrics` endpoint.

## Phase 5 — Report (~30min)

- [ ] **R1**: Append entry mới vào `05_progress.md` (Entry 5 — Muscle apply complete) với LOC thực + timestamp.
- [ ] **R2**: Update `report_bug_snapshot_progress_mismatch_2026-05-28.md` — đổi bảng từ "ước tính" sang "thực tế":
  - Files thay đổi: list cụ thể.
  - LOC delta: `git diff --stat` output.
- [ ] **R3**: (§8) Chạy `/security-agent` quét patch.
- [ ] **R4**: Update `agent/memory/global/active_plans.md` đổi status PLAN_READY → DONE.

## Gating verb `done`

Muscle CHỈ báo `done` khi:
- ✓ Tất cả Phase 1-5 checklist tick.
- ✓ AC-1 đến AC-8 trong `06_validation.md` PASS.
- ✓ `report_*.md` có **bảng files thay đổi** + **LOC delta thực** (không phải ước tính).
