# 00_context — Bug Snapshot Orphan Sau Kill Worker, FE Không Có Resume

## Trigger
User report 2026-05-28 (sau khi đã apply fix `bug-snapshot-progress-mismatch-2026-05-28`):

> "snapshot đang chạy tôi kill cdc-worker, start lại thì ko chạy nữa. trong http://localhost:5173/snapshot-monitor các nút chức năng resume ko xuất hiện để kích hoạt lại"

## Symptom
| Layer | Trạng thái |
|---|---|
| DB `cdc_system.snapshot_progress` | row `status='running'`, `updated_at` cũ (worker đã chết) |
| Worker mới (restarted) | Không tự pick up; chỉ react message `cdc.cmd.snapshot.v2` tới |
| NATS `cdc.cmd.snapshot.v2` | Không có message mới publish — message cũ đã được ACK trước khi worker chết |
| FE `/snapshot-monitor` | Chỉ hiện nút **Pause** (status=running) hoặc **Resume** (status=paused) — KHÔNG có Resume cho `running` stale |
| User option | KHÔNG có cách nào qua UI; phải đợi 10 phút zombie window + tự gọi API, hoặc UPDATE DB tay |

## Root cause (2 layer)

### Layer A — Worker boot KHÔNG có orphan reclaim
**Evidence**: `centralized-data-service/internal/server/worker_server.go:496-503`
```go
snapshotRunner := handler.NewSnapshotRunner(...)
if _, err := natsClient.Conn.QueueSubscribe(
    "cdc.cmd.snapshot.v2",
    "cdc-snapshot-runner",
    snapshotRunner.Handle,
); err != nil { ... }
// ↑ Chỉ subscribe. KHÔNG scan snapshot_progress tìm orphan.
```

`claimProgress` (`snapshot_runner_handler.go:628-633`):
```go
if existing.Status == "running" && !p.Overwrite {
    if time.Since(existing.UpdatedAt) < snapshotV2ZombieAfter { // 10 phút
        claim.acquired = false  // ← bỏ qua, dù có message tới
        claim.rowID = existing.ID
        return nil
    }
}
```

→ Sau khi worker restart: không có ai publish `cdc.cmd.snapshot.v2`, nên claim.acquired logic không bao giờ chạy. Row "running" nằm im. Sau 10 phút zombie → vẫn cần message external trigger.

### Layer B — FE chỉ render Resume khi status='paused'
**Evidence**: `cdc-cms-web/src/pages/SnapshotMonitor.tsx:155-171`
```tsx
{r.status === 'running' && (
  <Button size="small" danger onClick={...}>Pause</Button>
)}
{r.status === 'paused' && (
  <Button size="small" type="primary" onClick={...}>Resume</Button>
)}
```

→ Row stuck ở `status='running'` (mặc dù worker đã chết) → user chỉ thấy nút **Pause** (không hữu ích — chính nó đang đứng yên), không có **Resume**.

### Layer C — Resume API endpoint hoạt động đúng nếu được gọi
**Evidence**: `cdc-cms-service/internal/api/snapshot_progress_handler.go:66-77` — Resume handler publish `cdc.cmd.snapshot.v2` với `overwrite=false`. Worker nhận message → claimProgress thấy `paused` (hoặc `running` stale > 10min) → resume từ `last_seen_id`. Endpoint không phải bug; chỉ là **không có way gọi nó từ UI** cho row `running` stale.

## Workspace cũ liên quan (đã fix)
- `bug-first-snapshot-no-write-2026-05-26/` — fix layer HandleRaw.
- `snapshot-zero-records-2026-05-27/` — fix Flush chain counter.
- `bug-snapshot-progress-mismatch-2026-05-28/` — fix cursor/pause/markDone (vừa apply, 3 file +211 LOC).

Bug hôm nay là **vector khác**: orphan recovery, không phải data correctness.

## Lesson liên quan
- `lessons.md` L-2026-05-28-mark-done-without-completeness-guard: enforce invariant ở terminal edge.
- **Bài học mở rộng (anti-pattern phát hiện)**: Long-running job chỉ react message-driven, không có **boot-time orphan scan** → khi process bị kill (SIGKILL, OOM, crash) row in-flight stuck vĩnh viễn cho đến khi external trigger. Cần Global Pattern: "Process P with in-flight rows S, on boot must scan stale(S) > τ → re-claim or notify operator".

## Service liên quan
- `centralized-data-service/internal/server/worker_server.go` — startup hook.
- `centralized-data-service/internal/handler/snapshot_runner_handler.go` — claim logic + add reclaim function.
- `cdc-cms-web/src/pages/SnapshotMonitor.tsx` — Resume button render logic.
- `cdc-cms-service/internal/api/snapshot_progress_handler.go` — Resume API (no change cần — chỉ cần FE gọi).

## In-scope
- Worker boot: tự scan + re-publish `cdc.cmd.snapshot.v2` cho row `running` stale > τ.
- FE: hiển thị nút **Resume** cho `running` stale > τ với confirm dialog cảnh báo "snapshot có thể đã orphan".
- Threshold `staleAfter` = 60s (configurable) — cấp dưới `snapshotV2ZombieAfter` = 10 phút để giữ chống double-claim của claimProgress.

## Out-of-scope
- Đổi NATS sang JetStream (durable subscription) — re-architect lớn, defer.
- Auto-retry exponential backoff cho row `error` — không phải bug user report.
- Heartbeat health check liveness probe — infrastructure work, defer.

## Constraints từ User (verbatim style)
- ✓ Đọc lesson trước (đã đọc 3 workspace cũ + lessons.md).
- ✓ Theo `/agent` + GEMINI.md (Brain plan-only §1+§12).
- ✓ Plan rõ ràng + code demo chi tiết.
- ✓ KHÔNG cheat DB, KHÔNG đổi config.
- ✓ Report cuối có **files thay đổi** + **số dòng code thay đổi**.
- ✓ Verify service work trước khi báo done.
- ✓ Luôn có file `report_*.md`.
