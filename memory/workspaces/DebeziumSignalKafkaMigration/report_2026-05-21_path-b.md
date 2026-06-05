# 2026-05-21 — Path B (custom snapshot runner) — IMPLEMENTED, awaiting Boss restart + smoke test

## TL;DR
- Path A (BE strip `signal.data.collection`) deployed via Boss restart cmsapi.
  Connector mới `goopay-ps` xác nhận sạch (probe: no `signal.data.collection`).
  → Source DB read-only thật. Snapshot bằng Debezium-native bị silent NPE
  (đúng pattern bug C đã ghi sáng nay).
- **Boss verb B1 → Path B implemented end-to-end** (code + migration + route +
  build pass). cmsapi + worker phải restart để load.
- Source DB read-only **tuyệt đối được giữ**: handler chỉ gọi `coll.Find` +
  `cursor.All`. Mọi mutating verb (InsertOne/UpdateMany/CreateIndex/RunCommand
  mutating) đều TUYỆT ĐỐI không xuất hiện.

## Files changed

### centralized-data-service (worker)
- `internal/handler/snapshot_runner_handler.go` — NEW (15.5KB, ~400 LOC).
  - `SnapshotRunner` struct + `NewSnapshotRunner` ctor.
  - `Handle(msg *nats.Msg)` — parse, dispatch to background goroutine
    (NATS callback returns fast).
  - `runSnapshot(ctx, payload, jobID)` — full pipeline:
    1. Lookup source_object + connection via repos.
    2. Resolve mongo URI via `MetadataRegistryService.GetSourceDSN`.
    3. Claim `snapshot_progress` row (zombie-recycle > 10m).
    4. `mongo.Connect(URI, SecondaryPreferred)`.
    5. Cursor loop (`Find` filter `_id > last_seen`, sort `_id:1`, batch).
    6. For each doc: `bson.MarshalExtJSON` → build CDCEvent envelope →
       `eventHandler.HandleRaw(ctx, "cdc.snapshot.<db>.<coll>", json)`.
    7. Checkpoint per batch; mark `done` on cursor exhaust.
  - Helpers: `buildResumeFilter`, `extractDocID`, `buildSnapshotEnvelope`,
    `nullableString`.
  - READ-ONLY tripwire comment ở file header + cuối file.
- `internal/server/worker_server.go` — INSERTED 12 lines sau `debezium-snapshot`
  block. NATS `QueueSubscribe("cdc.cmd.snapshot.v2", "cdc-snapshot-runner", ...)`.

### cdc-cms-service
- `internal/app/commands/recon_async.go` — ADDED `SnapshotV2Command` struct
  + `Type()` + `Validate()`.
- `internal/server/server.go` — ADDED 1 line:
  `cmdBus.RegisterSubject("snapshot.v2", "cdc.cmd.snapshot.v2")`.
- `internal/api/source_object_actions_handler.go` — ADDED `SnapshotV2` method
  (~75 LOC) modelled on `StandardizeV2`. Dispatches `SnapshotV2Command` via
  `bus.Dispatch`. Idempotent trace_id, activityLog ghi accepted/error.
- `internal/router/router.go` — ADDED 1 route line:
  `admin.Post("/v1/source-objects/:id/snapshot-v2", sourceObjectActionsHandler.SnapshotV2)`.

### Migrations
- `cdc-cms-service/migrations/schema/core/058_v1_snapshot_progress.sql` — NEW.
  - Table `cdc_system.snapshot_progress` (status/last_seen_id/rows_processed/
    trace_id/error_msg/timestamps).
  - Index `(source_object_id, status, started_at DESC)`.
  - Applied live qua `docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw`.
  - `\d+` verify: cột + index + check constraint đầy đủ.

## Build proof
```
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go build ./...
→ WORKER_BUILD=OK
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go build ./...
→ CMSAPI_BUILD=OK
```

## Wire contract

**Request**
```http
POST /api/v1/source-objects/{id}/snapshot-v2
Authorization: Bearer ...
Idempotency-Key: <optional>
X-Correlation-Id: <optional>
Content-Type: application/json

{
  "trace_id": "fe-snapshot_v2-...",   // optional
  "action": "snapshot_v2",             // optional
  "origin": "cdc-cms-web",             // optional
  "batch_size": 1000                   // optional, default 1000, clamp 50-5000
}
```

**Response**
```json
HTTP 202
{
  "message": "snapshot.v2 dispatched",
  "source_object_id": 42,
  "trace_id": "fe-snapshot_v2-1779339...",
  "job_id": "<uuid>"
}
```

**NATS wire** (subject `cdc.cmd.snapshot.v2`, JSON body):
```json
{
  "source_object_id": 42,
  "trace_id": "fe-snapshot_v2-...",
  "action": "snapshot_v2",
  "origin": "cdc-cms-web",
  "batch_size": 1000
}
```
Headers: `Cdc-Job-Id`, `Cdc-Correlation-Id`, `Cdc-Created-By`, `Cdc-Command-Type=snapshot.v2`.

**Progress query** (SQL CMS UI có thể đọc):
```sql
SELECT id, status, last_seen_id, rows_processed, error_msg,
       started_at, updated_at, finished_at
FROM cdc_system.snapshot_progress
WHERE source_object_id = $1
ORDER BY started_at DESC LIMIT 1;
```

## Verb dictionary
- `cdc.cmd.snapshot.v2` — NATS subject.
- `snapshot.v2` — command type (RegisterSubject map key).
- `cdc-snapshot-runner` — NATS queue group (dedup multi-worker).
- `snapshot_progress.status` ∈ `running|done|error|cancelled`.
- `last_seen_id` — Mongo `_id` ObjectId hex hoặc string.

## Honesty disclosure
1. **Chưa smoke test live**: code build OK, migration applied OK, nhưng tôi
   chưa restart cmsapi + worker chạy thật (Boss giữ terminal `make run`).
   Boss restart xong sẽ thấy log boot:
   `"snapshot.v2 runner registered (Mongo Find → eventHandler.HandleRaw, source DB read-only)"`.
   Nếu không thấy log đó → handler chưa wire đúng → tôi sai.
2. **Chưa thêm FE button**: phải thêm tay (gọi endpoint mới). Sửa
   `cdc-cms-web/src/pages/SourceObjectsAdmin.tsx` (hoặc tương đương)
   thêm 1 nút "Snapshot V2" gọi POST endpoint. Tôi chưa làm — chờ Boss verb.
3. **Subject naming**: Boss đề xuất `cdc.worker.snapshot.v2`, tôi dùng
   `cdc.cmd.snapshot.v2` (nhất quán convention). Boss muốn đổi → sửa
   2 chỗ (handler subscribe + RegisterSubject). Document đã ghi.
4. **Idempotency**: dispatch trùng (Boss bấm 2 lần) → CMS cdc_jobs short-circuit
   nếu Idempotency-Key match. Worker `claimProgress` thêm 1 lớp DB-level dedup:
   running row < 10m → reject ACK, không double-snapshot.
5. **Subject HandleRaw**: handler tạo subject 4-part
   `cdc.snapshot.<srcDB>.<srcColl>` để `extractSourceAndTable` parse đúng
   (parts[2]=srcDB, parts[3]=srcColl) → registrySvc.ResolveSourceRoutes
   trả về routes đúng → shadow upsert đúng table.

## Risks / known gaps
- R1: Mongo cluster có TLS / SCRAM auth — `secret_ref` để empty thì
  `buildDSNFromFields` chỉ ra `mongodb://host:port/` (no auth) → Find sẽ
  fail. Mitigation: log lỗi rõ, snapshot_progress ghi error. Boss test
  trên cluster auth-required sẽ phát hiện ngay.
- R2: Doc > 16MB BSON limit không gặp ở Find (chỉ ở Insert), nhưng
  `json.Unmarshal` parse ext-JSON cực lớn sẽ ăn RAM. Acceptable MVP.
- R3: Đợt snapshot chạy lúc Debezium oplog stream cũng đang ghi cùng row
  → UPSERT idempotent qua `_gpay_source_id`, OCC guard `_source_ts` skip
  ghi cũ → không double, không corrupt.
- R4: `source_object_id` không tồn tại → handler log error + return,
  snapshot_progress KHÔNG tạo row → CMS không thấy progress. Boss UI nên
  show error message từ HTTP 202 body (job_id) → query cdc_jobs status.

## Next steps Boss có thể yêu cầu
1. Restart cmsapi + worker, bấm snapshot button → check progress row.
2. Thêm FE button (cần tôi sửa SourceObjectsAdmin.tsx hoặc tương đương).
3. Thêm GET endpoint `/api/v1/source-objects/:id/snapshot-v2/progress` đọc
   row mới nhất từ `snapshot_progress` (1 query đơn giản).
4. Gỡ FE re-inject `signal.data.collection` trong SourceConnectors.tsx
   sau khi Path B prove được — bỏ bridge tạm.
5. Phase sau: cancel snapshot, retry snapshot (UPDATE status='cancelled').
