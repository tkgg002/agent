# 03_implementation — Code demo chi tiết (Brain plan-only §12)

> Code dưới đây là DEMO trong markdown để Muscle copy. Brain KHÔNG sửa file `.go/.tsx` thực.

---

## B1 + B2 — Worker reclaim function + publish helper

### File: `centralized-data-service/internal/handler/snapshot_runner_handler.go` (APPEND cuối file)

```go
// snapshotV2DefaultStaleAfter — a snapshot_progress row in status='running'
// becomes "stale" once its updated_at falls this far behind wall clock.
// Checkpoint updates updated_at every batch (~5s under normal load), so
// 60s leaves a ~12x buffer before reclaim. The constant is overridden by
// env SNAPSHOT_STALE_AFTER_SECONDS at boot.
const snapshotV2DefaultStaleAfter = 60 * time.Second

// ReclaimOrphans scans snapshot_progress for rows stuck in status='running'
// past the stale window (worker killed mid-run, OOM, crash). For each such
// row it (a) demotes status to 'paused' so claimProgress takes the resume
// branch instead of the zombie-window early-return, and (b) re-publishes
// `cdc.cmd.snapshot.v2` so the queue group delivers the run to a live
// worker. Idempotent under DB transaction lock — concurrent boots are safe.
func (r *SnapshotRunner) ReclaimOrphans(ctx context.Context, staleAfter time.Duration) (int, error) {
	if r.natsConn == nil {
		r.logger.Warn("snapshot.v2 reclaim skipped — nats conn nil")
		return 0, nil
	}
	type orphanRow struct {
		ID             int64 `gorm:"column:id"`
		SourceObjectID int64 `gorm:"column:source_object_id"`
	}
	var rows []orphanRow
	cutoff := time.Now().Add(-staleAfter)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id, source_object_id
		FROM cdc_system.snapshot_progress
		WHERE status = 'running' AND updated_at < ?
		ORDER BY id ASC
	`, cutoff).Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("reclaim scan: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}

	reclaimed := 0
	for _, row := range rows {
		// Demote to paused so claimProgress.line:637 resume branch fires;
		// otherwise the zombie-window check at line 629 swallows the claim.
		if err := r.db.WithContext(ctx).Exec(`
			UPDATE cdc_system.snapshot_progress
			SET status = 'paused', updated_at = NOW()
			WHERE id = ? AND status = 'running'
		`, row.ID).Error; err != nil {
			r.logger.Warn("snapshot.v2 reclaim demote failed",
				zap.Int64("progress_id", row.ID),
				zap.Error(err))
			continue
		}
		if err := r.publishResumeMessage(ctx, row.SourceObjectID); err != nil {
			r.logger.Warn("snapshot.v2 reclaim publish failed",
				zap.Int64("progress_id", row.ID),
				zap.Int64("source_object_id", row.SourceObjectID),
				zap.Error(err))
			continue
		}
		reclaimed++
	}
	r.logger.Info("snapshot.v2 boot-reclaim complete",
		zap.Int("orphan_count", len(rows)),
		zap.Int("reclaimed", reclaimed),
		zap.Duration("stale_after", staleAfter))
	return reclaimed, nil
}

// publishResumeMessage publishes a cdc.cmd.snapshot.v2 envelope with
// origin="boot-reclaim". Worker resolves source_object_id and resumes from
// last_seen_id checkpoint — no overwrite, no batch_size override.
func (r *SnapshotRunner) publishResumeMessage(ctx context.Context, sourceObjectID int64) error {
	payload := snapshotV2Payload{
		SourceObjectID: sourceObjectID,
		TraceID:        fmt.Sprintf("boot-reclaim-%d", time.Now().UnixNano()),
		Origin:         "boot-reclaim",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return r.natsConn.Publish("cdc.cmd.snapshot.v2", raw)
}
```

---

## B3 — Worker server wire boot reclaim

### File: `centralized-data-service/internal/server/worker_server.go:496-504`

```go
// BEFORE
snapshotRunner := handler.NewSnapshotRunner(db, eventHandler, registrySvc, connectionRepo, sourceObjectRepo, natsClient.Conn, logger)
if _, err := natsClient.Conn.QueueSubscribe(
    "cdc.cmd.snapshot.v2",
    "cdc-snapshot-runner",
    snapshotRunner.Handle,
); err != nil {
    return nil, fmt.Errorf("subscribe cdc.cmd.snapshot.v2: %w", err)
}
logger.Info("snapshot.v2 runner registered (Mongo Find → eventHandler.HandleRaw, source DB read-only)")
```

```go
// AFTER
snapshotRunner := handler.NewSnapshotRunner(db, eventHandler, registrySvc, connectionRepo, sourceObjectRepo, natsClient.Conn, logger)
if _, err := natsClient.Conn.QueueSubscribe(
    "cdc.cmd.snapshot.v2",
    "cdc-snapshot-runner",
    snapshotRunner.Handle,
); err != nil {
    return nil, fmt.Errorf("subscribe cdc.cmd.snapshot.v2: %w", err)
}
logger.Info("snapshot.v2 runner registered (Mongo Find → eventHandler.HandleRaw, source DB read-only)")

// Boot-time orphan reclaim — a previous worker may have been killed mid-run
// leaving rows stuck in status='running'. Run async so a DB hiccup doesn't
// block the rest of worker startup; the subscriber is already live, so a
// race with new messages is harmless (claimProgress transaction lock).
go func() {
    staleAfter := snapshotV2DefaultStaleAfter
    if envVal := os.Getenv("SNAPSHOT_STALE_AFTER_SECONDS"); envVal != "" {
        if secs, err := strconv.Atoi(envVal); err == nil && secs > 0 {
            staleAfter = time.Duration(secs) * time.Second
        }
    }
    // Tiny delay so the NATS subscriber is registered on the wire before
    // the reclaim publish — avoids the published message racing ahead of
    // the subscriber bind on slow brokers.
    time.Sleep(500 * time.Millisecond)
    if _, err := snapshotRunner.ReclaimOrphans(context.Background(), staleAfter); err != nil {
        logger.Warn("snapshot.v2 boot-reclaim failed (worker still healthy)", zap.Error(err))
    }
}()
```

> **Import bổ sung** ở `worker_server.go`: `"os"`, `"strconv"`, `"time"`, `"context"`, `"go.uber.org/zap"` + `handler` package. Đa số đã import; chỉ `os`+`strconv` có thể cần thêm.

> **Lưu ý**: `snapshotV2DefaultStaleAfter` exported gián tiếp qua method `ReclaimOrphans(ctx, staleAfter)` — caller truyền value, không reference const trong handler package. Để worker_server đọc env và truyền vào.

---

## F1 + F2 + F3 — FE Stale Resume Button

### File: `cdc-cms-web/src/pages/SnapshotMonitor.tsx`

#### F1 — Helper `isStaleRunning` (thêm ngay trước component)

```tsx
// A snapshot row stays in status='running' even after the worker that owns
// it dies. The boot-reclaim job in centralized-data-service catches it
// within ~60s on the next worker restart, but until then the operator
// has no UI escape hatch. Treat updated_at older than this threshold as
// "stale" — show a Resume button with a warning so the operator can
// force-republish the snapshot.v2 message without waiting for a restart.
const STALE_RUNNING_THRESHOLD_MS = 60 * 1000;

function isStaleRunning(record: { status: string; updated_at?: string }): boolean {
  if (record.status !== 'running' || !record.updated_at) return false;
  const updated = Date.parse(record.updated_at);
  if (Number.isNaN(updated)) return false;
  return Date.now() - updated > STALE_RUNNING_THRESHOLD_MS;
}
```

#### F2 — Actions column (line 155-171)

```tsx
// BEFORE
{
  title: 'Actions', key: 'actions', width: 120,
  render: (_, r) => (
    <Space size="small">
      {r.status === 'running' && (
        <Button size="small" danger onClick={() => setActionPending({ action: 'pause', record: r })}>
          Pause
        </Button>
      )}
      {r.status === 'paused' && (
        <Button size="small" type="primary" onClick={() => setActionPending({ action: 'resume', record: r })}>
          Resume
        </Button>
      )}
    </Space>
  ),
},
```

```tsx
// AFTER
{
  title: 'Actions', key: 'actions', width: 160,
  render: (_, r) => {
    const stale = isStaleRunning(r);
    return (
      <Space size="small">
        {r.status === 'running' && !stale && (
          <Button size="small" danger onClick={() => setActionPending({ action: 'pause', record: r })}>
            Pause
          </Button>
        )}
        {r.status === 'paused' && (
          <Button size="small" type="primary" onClick={() => setActionPending({ action: 'resume', record: r })}>
            Resume
          </Button>
        )}
        {r.status === 'running' && stale && (
          <Tooltip title="Snapshot có thể đã orphan — worker chưa heartbeat 60s+">
            <Button
              size="small"
              type="primary"
              icon={<WarningOutlined />}
              onClick={() => setActionPending({ action: 'resume', record: r, stale: true })}>
              Force Resume
            </Button>
          </Tooltip>
        )}
      </Space>
    );
  },
},
```

> **Import bổ sung**: `import { WarningOutlined } from '@ant-design/icons';` + `Tooltip` đã có sẵn ở top imports.

#### F3 — Modal description nhánh stale

```tsx
// BEFORE (line 223-240 dialog block)
<ConfirmDestructiveModal
  open={!!actionPending}
  title={actionPending?.action === 'pause' ? 'Tạm dừng Snapshot' : 'Tiếp tục Snapshot'}
  description={actionPending?.action === 'pause' ? 'Tạm dừng quá trình snapshot sẽ ngưng kéo data từ nguồn.' : 'Tiếp tục quá trình snapshot đang bị tạm dừng.'}
  ...
/>
```

```tsx
// AFTER
<ConfirmDestructiveModal
  open={!!actionPending}
  title={actionPending?.action === 'pause' ? 'Tạm dừng Snapshot' : 'Tiếp tục Snapshot'}
  description={
    actionPending?.action === 'pause'
      ? 'Tạm dừng quá trình snapshot sẽ ngưng kéo data từ nguồn.'
      : actionPending?.stale
        ? 'Force resume: snapshot này có status=running nhưng worker chưa heartbeat 60s+ (có thể đã crash). Bấm Resume sẽ re-publish message snapshot.v2. Lưu ý: nếu worker thật sự còn sống, claimProgress transaction lock sẽ chống double-process.'
        : 'Tiếp tục quá trình snapshot đang bị tạm dừng.'
  }
  ...
/>
```

> **Type extension**: `actionPending` interface cần field optional `stale?: boolean`.

---

## T1 + T2 + T3 — Test demo

### File: `centralized-data-service/internal/handler/snapshot_runner_handler_test.go` (APPEND)

```go
func TestReclaimOrphans_StaleRowsPublished(t *testing.T) {
    r, mock := newRunnerWithMockDB(t)
    // need natsConn — use embed nats-server or skip if nil:
    // simplest: set r.natsConn to a server.Run(nats-test.RunDefaultServer) helper.
    // For unit-level scope, use a stub publisher interface (preferred refactor).

    // Mock SELECT — 2 stale rows + 0 fresh (filter ở SQL nên không return fresh)
    mock.ExpectQuery(`SELECT id, source_object_id\s+FROM cdc_system\.snapshot_progress\s+WHERE status = 'running' AND updated_at < `).
        WithArgs(sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id", "source_object_id"}).
            AddRow(int64(101), int64(11)).
            AddRow(int64(102), int64(12)))

    // 2 UPDATE demote paused expected
    mock.ExpectExec(`UPDATE cdc_system\.snapshot_progress\s+SET status = 'paused'`).
        WithArgs(int64(101)).WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec(`UPDATE cdc_system\.snapshot_progress\s+SET status = 'paused'`).
        WithArgs(int64(102)).WillReturnResult(sqlmock.NewResult(0, 1))

    // natsConn publish — require a stub. Either inject via field or use
    // nats-server.RunRandClientPortServer + client.Connect.
    reclaimed, err := r.ReclaimOrphans(context.Background(), 60*time.Second)
    if err != nil { t.Fatalf("err=%v", err) }
    if reclaimed != 2 { t.Fatalf("reclaimed=%d want 2", reclaimed) }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet: %v", err)
    }
}

func TestReclaimOrphans_NoStale_NoOp(t *testing.T) {
    r, mock := newRunnerWithMockDB(t)
    mock.ExpectQuery(`SELECT id, source_object_id\s+FROM cdc_system\.snapshot_progress`).
        WithArgs(sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id", "source_object_id"}))
    reclaimed, err := r.ReclaimOrphans(context.Background(), 60*time.Second)
    if err != nil || reclaimed != 0 {
        t.Fatalf("err=%v reclaimed=%d", err, reclaimed)
    }
}

func TestReclaimOrphans_DBError_Propagates(t *testing.T) {
    r, mock := newRunnerWithMockDB(t)
    mock.ExpectQuery(`SELECT id, source_object_id`).
        WithArgs(sqlmock.AnyArg()).
        WillReturnError(errors.New("conn refused"))
    _, err := r.ReclaimOrphans(context.Background(), 60*time.Second)
    if err == nil {
        t.Fatalf("expected err, got nil")
    }
}
```

> **NATS stub**: cách đơn giản nhất cho unit test → inject `natsConn` field thành interface (vd `type natsPublisher interface { Publish(string, []byte) error }`) — chấp nhận `*nats.Conn` hoặc mock. Refactor 5-10 LOC trong `snapshot_runner_handler.go`. Hoặc dùng `nats-server/test.RunRandClientPortServer` cho embed in-memory.

---

## Summary Patch Plan

| Patch | File | Type | LOC ước tính |
|---|---|---|---|
| B1+B2+B4 | `snapshot_runner_handler.go` | APPEND `ReclaimOrphans` + `publishResumeMessage` + const | +90 |
| B3 | `internal/server/worker_server.go` | APPEND goroutine boot reclaim | +12 |
| F1+F2+F3 | `cdc-cms-web/src/pages/SnapshotMonitor.tsx` | APPEND helper + Actions render + Modal text | +29 |
| T1+T2+T3 | `internal/handler/snapshot_runner_handler_test.go` (APPEND) | 3 test function | +130 |

**Total**: ~+261 LOC; **4 file thay đổi**.
