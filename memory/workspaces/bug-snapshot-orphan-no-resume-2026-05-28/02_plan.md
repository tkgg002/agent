# 02_plan — Sequencing Fix Bug Snapshot Orphan No-Resume

## Approach
Patch HOLISTIC 1 PR multi-service (BE + FE) cùng 1 plan. BE chính (auto-recovery), FE phụ trợ (manual escape hatch).

## Phase sequencing

### Phase 1 — BE Worker Boot Reclaim (~2h)
| Step | File | Hành động | LOC ước tính |
|---|---|---|---|
| B1 | `snapshot_runner_handler.go` (cuối file) | Thêm method `ReclaimOrphans(ctx, staleAfter time.Duration) (n int, err error)` — query running rows stale → UPDATE status='paused' → publish `cdc.cmd.snapshot.v2` từng row | +50 |
| B2 | `snapshot_runner_handler.go` | Thêm helper `publishResumeMessage(ctx, sourceObjectID int64) error` — marshal payload `{source_object_id, trace_id: "boot-reclaim-<ts>", origin: "boot-reclaim"}` và `natsConn.Publish` | +20 |
| B3 | `internal/server/worker_server.go:496-504` | Sau khi `QueueSubscribe`, thêm goroutine async: `go snapshotRunner.ReclaimOrphans(...)` với log warn nếu fail | +12 |
| B4 | Add const | `snapshot_runner_handler.go` const block | `snapshotV2DefaultStaleAfter = 60 * time.Second` + env reader | +8 |
| **Sub-total** | | | **+90 NET** |

### Phase 2 — FE Stale Resume Button (~1h)
| Step | File | Hành động | LOC ước tính |
|---|---|---|---|
| F1 | `cdc-cms-web/src/pages/SnapshotMonitor.tsx` | Thêm helper `isStaleRunning(updatedAt: string): boolean` (delta > 60s) | +8 |
| F2 | Same file `:155-171` Actions column | Render thêm nút Resume khi `r.status === 'running' && isStaleRunning(r.updated_at)` với icon `<WarningOutlined />` + tooltip | +15 |
| F3 | Modal `:223-240` description | Branch text: nếu source row đang stale-running → cảnh báo "có thể đã orphan" | +6 |
| **Sub-total** | | | **+29 NET** |

### Phase 3 — Test (~1.5h)
| Step | File | Hành động | LOC ước tính |
|---|---|---|---|
| T1 | `snapshot_runner_handler_test.go` (APPEND) | `TestReclaimOrphans_StaleRowsPublished` — sqlmock 3 row (2 stale + 1 fresh) → assert 2 message publish + 2 UPDATE paused | +80 |
| T2 | Same | `TestReclaimOrphans_NoStale_NoOp` — không có row stale → 0 publish | +25 |
| T3 | Same | `TestReclaimOrphans_DBError_LogsNoSpread` — query lỗi → return err, không panic | +25 |
| **Sub-total** | | | **+130 NET (test)** |

### Phase 4 — Verify (~30min)
- `cd centralized-data-service && go build ./...` exit 0.
- `go vet ./internal/handler/...` no new error.
- `go test ./internal/handler/... -count=1 -timeout 120s` PASS.
- `cd cdc-cms-web && npx vite build` PASS (FE static check).
- Optional runtime smoke: tạo row `status='running' updated_at=NOW() - 90s'`, restart worker → row phải chuyển paused→running trong 5s.

### Phase 5 — Report + governance (~30min)
- Update `report_*.md` với LOC thực + files thay đổi.
- APPEND Entry 5 `05_progress.md`.
- APPEND `active_plans.md` + lesson global `L-2026-05-28-boot-reclaim-missing-for-message-driven-runner`.

## Total LOC estimate
- Production BE: ~+90.
- Production FE: ~+29.
- Test BE: ~+130.
- **TOTAL**: ~+249 LOC, **4 file thay đổi** (2 BE + 1 FE + 1 test mới hoặc append).

## Risk + Mitigation
| Risk | Mitigation |
|---|---|
| Boot reclaim publish trong khi queue subscriber chưa sẵn sàng | Spawn goroutine sau khi `QueueSubscribe` return success; thêm `time.Sleep(500ms)` để JetStream/NATS sync (đã đủ với non-jetstream subscriber tạm) |
| Double-claim race (2 worker boot cùng lúc) | claimProgress đã có DB transaction lock (line 607). Lock-by-DB an toàn cho race. |
| `staleAfter=60s` quá strict — batch flush > 60s sẽ false-reclaim | Checkpoint hiện update updated_at mỗi batch (line 535). Batch flush ~5s. 60s margin = 12x batch interval. Vẫn an toàn. |
| FE `isStaleRunning` chạy ở browser tz mismatch | Dùng `Date.parse(updated_at)` ISO, không phụ thuộc tz local |

## Rollback plan
- Phase 1 fail → revert worker_server.go change → chỉ subscribe NATS như cũ.
- Phase 2 fail → revert FE Resume button → user lại đợi 10 phút zombie + UPDATE DB tay.
- Phase 3 test fail → revert toàn bộ commit.
