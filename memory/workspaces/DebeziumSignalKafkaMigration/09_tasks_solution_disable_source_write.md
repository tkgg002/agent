# 09 Tasks-Solution — Disable Debezium Source-DB Write

> **Phase**: `_disable_source_write`
> **Status**: AWAIT BOSS VERB (`ship path A` / `apply backfill` / `start path B` / `defer`)

## Task breakdown (Path A — execute when verb `ship path A`)

### T1. BE: Patch `injectDebeziumSignalDefaults`
- File: `cdc-cms-service/internal/api/system_connectors_handler.go`
- Line: 438-461
- Change:
  - `signal.enabled.channels` value: `"source,kafka"` → `"kafka"`
  - Add `delete(cfg, "signal.data.collection")` before override loop
- Test: any handler_test.go covering injectDebeziumSignalDefaults — update assertion.

### T2. FE: Strip signal.data.collection + drop "source" channel
- File: `cdc-cms-web/src/pages/SourceConnectors.tsx`
- Lines:
  - 180-181 (Mongo create branch)
  - 205-206 (MySQL create branch)
  - 233-235 (PG create branch)
- Change:
  - Remove `'signal.data.collection': ...` line
  - Change `'signal.enabled.channels': 'source,kafka'` → `'signal.enabled.channels': 'kafka'`
- Test: `npx tsc --noEmit` PASS.

### T3. Build verification (Muscle gates)
- `cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go vet ./... && go build ./... && go test -count=1 ./...` PASS
- `cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-web && npx tsc --noEmit` PASS

### T4. Restart cmsapi local
- `pkill -f 'go-build.*exe/main'` → re-run `go run cmd/server/main.go`
- Wait readiness probe.

### T5. Smoke test
- Re-POST connector demo qua FE → `curl /api/v1/system/connectors/demo/config` → verify keys.
- Trigger snapshot via FE button → check Mongo source count.

## Task breakdown (Path A.3 — execute when verb `apply backfill`, AFTER `ship path A`)

### T6. Backfill 3 dev connectors
- Script:
```bash
for c in goopay-ps goopay-pbs goopay-ces; do
  CFG=$(curl -s "http://10.200.186.203:8083/connectors/$c/config" \
    | python3 -c "import json,sys; d=json.load(sys.stdin); d.pop('signal.data.collection',None); d['signal.enabled.channels']='kafka'; print(json.dumps(d))")
  echo "=== PUT $c ==="
  echo "$CFG" | python3 -m json.tool | head -5
  curl -s -X PUT -H 'Content-Type: application/json' \
    "http://10.200.186.203:8083/connectors/$c/config" -d "$CFG" \
    | python3 -m json.tool | head -10
  sleep 1
done
```
- Verify per-connector RUNNING after PUT.

### T7. Capture before/after evidence
- BEFORE: 142 documents (38+42+62) snapshot-window-* already in source.
- AFTER (post-backfill + trigger snapshot once): count phải KHÔNG tăng.
- Optional cleanup (Boss verb `cleanup source signals`): `db.debezium_signals.drop()` per database — KHÔNG TỰ LÀM mà không có verb riêng (data destruction risk).

## Task breakdown (Path B — execute when verb `start path B`)

Document only — KHÔNG code trong scope phase này. Plan ngắn:

### B1. Migration cdc_system.snapshot_progress
- PG table: `(source_object_id BIGINT PK, last_seen_id TEXT, last_seen_at TIMESTAMPTZ, status TEXT, chunk_size INT, updated_at TIMESTAMPTZ)`.

### B2. Worker `internal/snapshot/runner.go`
- Subscribe NATS `cdc.cmd.snapshot.request`.
- Open Mongo read-only client cho source.
- Cursor + batch publish Kafka.
- Checkpoint PG `snapshot_progress`.

### B3. CMS API → publish NATS
- Replace `system-connector.lifecycle restart-task` path cho snapshot.
- FE button gọi mới.

### B4. Envelope schema compat
- Debezium envelope: `{op, ts_ms, source, before, after}` — emulate cho replay consumer.

### B5. Tests
- Unit: cursor pagination.
- Integration: e2e dev cluster — 1k docs snapshot vào shadow.

## Definition-of-done

- [ ] Boss approve verb path A or A+B.
- [ ] Code edits committed (per verb).
- [ ] Build + test PASS.
- [ ] Live evidence captured (before/after Mongo count).
- [ ] `05_progress.md` APPEND row.
- [ ] `report_2026-05-21_disable-source-write.md` created.
- [ ] Lesson APPEND nếu phát hiện thêm pattern mới.
