# 09 — Tasks Solution: Path B

## Task list (tracked in TaskCreate #70–#76)

### B0 (#70) ✅ — đọc registry repos + GetSourceDSN
Đã xong. Đã verify shape.

### B1 (#71) ✅ — workspace docs
Đã xong (01_requirements_path_b.md, 02_plan_path_b.md, file này).

### B2 (#72) — Migration 058
File: `cdc-cms-service/migrations/schema/core/058_v1_snapshot_progress.sql`

```sql
-- 058: Path B snapshot runner — control-plane checkpoint table.
-- Used by centralized-data-service/internal/handler/snapshot_runner_handler.go
-- to resume Mongo→shadow snapshots after worker restart and to expose
-- progress to the CMS UI.

CREATE TABLE IF NOT EXISTS cdc_system.snapshot_progress (
    id               BIGSERIAL PRIMARY KEY,
    source_object_id BIGINT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'running'
                     CHECK (status IN ('running','done','error','cancelled')),
    last_seen_id     TEXT,
    rows_processed   BIGINT NOT NULL DEFAULT 0,
    trace_id         TEXT,
    error_msg        TEXT,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_snapshot_progress_source_status
    ON cdc_system.snapshot_progress (source_object_id, status, started_at DESC);

COMMENT ON TABLE cdc_system.snapshot_progress IS
    'Path B (custom snapshot runner) checkpoint — bypasses Debezium signal so source DB stays read-only.';
COMMENT ON COLUMN cdc_system.snapshot_progress.last_seen_id IS
    'Mongo _id (hex ObjectId or string) of the last successfully ACKed doc. NULL = not started.';
```

### B3 (#73) — Worker handler
File: `centralized-data-service/internal/handler/snapshot_runner_handler.go`

Outline (~250 LOC):
- struct `SnapshotRunner{db, eventHandler, registrySvc, connRepo, soRepo, logger}`
- `func NewSnapshotRunner(...) *SnapshotRunner`
- `func (r *SnapshotRunner) Handle(msg *nats.Msg)` — unmarshal payload, kick off `runSnapshot(ctx, payload, header)`
- `func (r *SnapshotRunner) runSnapshot(ctx, payload, jobID, traceID) error`
  1. Lookup source_object + connection via repos.
  2. Resolve mongo URI via `registrySvc.GetSourceDSN(connectionCode)`.
  3. Mongo connect (read-only). Cleanup with `defer client.Disconnect`.
  4. Insert/Update `snapshot_progress` row → status='running', store trace_id, source_object_id.
     Use UPSERT pattern (one active row per source_object_id).
  5. Read `last_seen_id` from row.
  6. Build filter + sort + batch cursor.
  7. Iterate cursor in batches; for each doc → build CDCEvent JSON →
     `eventHandler.HandleRaw(ctx, subject, json)`. On error → mark progress error, return.
  8. Update checkpoint after each batch.
  9. On cursor exhaust → mark status='done', finished_at=NOW.
- Helper `buildCDCEventJSON(doc, srcDB, srcColl) []byte` — minimal shape that
  CDCEventData (Op, After, SourceTsMs) and processEvent need.
- Helper `objectIDFilter(lastSeen string) bson.M` — if hex → ObjectId, else string.

### B4 (#74) — Wire NATS subscribe
File: `centralized-data-service/internal/server/worker_server.go`

Insert after `debezium-snapshot` subscribe (around line 432):

```go
// Path B — custom snapshot runner (source DB read-only). Subject
// cdc.cmd.snapshot.v2 is dispatched by CMS; this worker pulls docs
// from Mongo via Find ONLY and replays them into the shadow apply
// pipeline through eventHandler.HandleRaw.
snapshotRunner := handler.NewSnapshotRunner(db, eventHandler, registrySvc,
    connectionRepo, sourceObjectRepo, logger)
if _, err := natsClient.Conn.QueueSubscribe(
    "cdc.cmd.snapshot.v2", "cdc-snapshot-runner",
    snapshotRunner.Handle,
); err != nil {
    return nil, fmt.Errorf("subscribe cdc.cmd.snapshot.v2: %w", err)
}
logger.Info("snapshot.v2 runner registered (Mongo Find → eventHandler.HandleRaw, source DB read-only)")
```

### B5 (#75) — CMS publish
1. Add command struct in `cdc-cms-service/internal/app/commands/recon_async.go`:
   ```go
   type SnapshotV2Command struct {
       ports.AsyncCommandMixin
       SourceObjectID int64  `json:"source_object_id"`
       TraceID        string `json:"trace_id,omitempty"`
       Action         string `json:"action,omitempty"`
       Origin         string `json:"origin,omitempty"`
       BatchSize      int    `json:"batch_size,omitempty"`
   }
   func (SnapshotV2Command) Type() string { return "snapshot.v2" }
   func (c SnapshotV2Command) Validate() error {
       if c.SourceObjectID <= 0 { return errors.New("snapshot.v2: source_object_id required") }
       return nil
   }
   ```
2. Register subject in `cdc-cms-service/internal/server/server.go` (after line 161):
   ```go
   cmdBus.RegisterSubject("snapshot.v2", "cdc.cmd.snapshot.v2")
   ```
3. Add handler in `cdc-cms-service/internal/api/source_object_actions_handler.go`
   (file already wires source_object_id paths). Endpoint:
   `POST /api/source-objects/:id/snapshot-v2`
   Body: `{"trace_id":"...","action":"snapshot_v2","origin":"cms","batch_size":1000}`
4. Wire route in server router for the new handler method.

### B6 (#76) — Build + test + report
- `go build ./...` cả 2 service.
- `go test ./internal/app/commands/...` (test cho SnapshotV2Command.Validate).
- Migrate live PG `058_v1_snapshot_progress.sql`.
- Append `05_progress.md`, `lessons.md` (Global Pattern dạng "X reuses inverted pipeline Y to avoid mutating Z").
- Viết `report_2026-05-21_path-b.md` tóm tắt.
