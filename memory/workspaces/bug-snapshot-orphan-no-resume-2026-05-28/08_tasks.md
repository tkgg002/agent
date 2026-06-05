# 08_tasks — Checklist Muscle Execute

## Phase 1 — BE Worker Boot Reclaim (~2h)

- [ ] **B4 const**: Thêm `snapshotV2DefaultStaleAfter = 60 * time.Second` ở const block `snapshot_runner_handler.go:45`.
- [ ] **B1 ReclaimOrphans**: APPEND method `(r *SnapshotRunner) ReclaimOrphans(ctx, staleAfter) (int, error)` ở cuối `snapshot_runner_handler.go`.
- [ ] **B2 publishResumeMessage**: APPEND helper `publishResumeMessage(ctx, sourceObjectID)`.
- [ ] **B3 worker_server wire**: Sau `QueueSubscribe` ở `internal/server/worker_server.go:504`, thêm goroutine async đọc env `SNAPSHOT_STALE_AFTER_SECONDS` rồi gọi `ReclaimOrphans`. Thêm import `os`, `strconv`, `context` nếu chưa có.

## Phase 2 — FE Stale Resume (~1h)

- [ ] **F1 helper**: APPEND `isStaleRunning` + const `STALE_RUNNING_THRESHOLD_MS` ở `cdc-cms-web/src/pages/SnapshotMonitor.tsx` (ngay trước component default export).
- [ ] **F2 Actions column**: Update render block line 155-171 hiển thị thêm nút "Force Resume" cho `running && stale`. Import `WarningOutlined` từ `@ant-design/icons`.
- [ ] **F3 Modal warning**: Branch description theo `actionPending?.stale` ở Modal block line 223-240. Mở rộng type `actionPending` thêm field optional `stale?: boolean`.

## Phase 3 — Test (~1.5h)

- [ ] **Refactor (small)**: Đổi `r.natsConn *nats.Conn` thành interface để mock được:
  ```go
  type natsPublisher interface { Publish(string, []byte) error }
  ```
  hoặc giữ nguyên `*nats.Conn` + dùng `nats-server/test` embed server.
- [ ] **T1**: `TestReclaimOrphans_StaleRowsPublished` APPEND `snapshot_runner_handler_test.go`.
- [ ] **T2**: `TestReclaimOrphans_NoStale_NoOp`.
- [ ] **T3**: `TestReclaimOrphans_DBError_Propagates`.

## Phase 4 — Verify (~30min)

- [ ] **V1 Build BE**: `cd centralized-data-service && go build ./...` exit 0.
- [ ] **V2 Vet BE**: `go vet ./internal/handler/... ./internal/server/...` no new error.
- [ ] **V3 Test BE**: `go test ./internal/handler/... ./internal/server/... -count=1 -timeout 120s` PASS.
- [ ] **V4 Build FE**: `cd cdc-cms-web && npx vite build` PASS.
- [ ] **V5 Runtime smoke** (06_validation.md):
  - Seed orphan row `status='running' updated_at=NOW()-120s'`.
  - Restart worker.
  - Trong 10s row chuyển paused → running.
  - Open `/snapshot-monitor`, thấy Force Resume button, click → ok.
- [ ] **V6 Regression**: `go test ./internal/handler/ -run 'MarkProgress|CursorEarly|Pause_No' -count=1` PASS (fix cũ không break).

## Phase 5 — Report + Governance (~30min)

- [ ] **R1**: APPEND Entry 4-5 `05_progress.md` với LOC thực + verify result.
- [ ] **R2**: Update `report_bug_snapshot_orphan_no_resume_2026-05-28.md` đổi "ước tính" → "thực tế".
- [ ] **R3**: APPEND `agent/memory/global/active_plans.md` entry workspace.
- [ ] **R4**: APPEND `agent/memory/global/lessons.md` lesson `L-2026-05-28-boot-reclaim-missing-for-message-driven-runner`.
- [ ] **R5**: Chạy `/security-agent` (§8).

## Gating verb `done`

Muscle CHỈ báo `done` khi:
- ✓ Phase 1+2+3+4+5 checklist tick đầy đủ.
- ✓ V5 runtime smoke (BE reclaim + FE button) thấy hoạt động trên dataset thực.
- ✓ Report có **bảng files thay đổi** + **LOC delta thực** (git diff --stat).
- ✓ Regression test fix cũ PASS.
