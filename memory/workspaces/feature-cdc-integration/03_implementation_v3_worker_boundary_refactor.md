# Implementation — v3 Worker Boundary Refactor (10 new NATS handlers)

> **Date**: 2026-04-17
> **Trigger**: User approved fix for 12 architectural violations audited in
> `10_gap_analysis_scan_fields_boundary_violation.md` + Explore session
> (A) HTTP sync→NATS async, (B) CMS touches Worker-owned ops, (C) hardcoded
> Airbyte ignoring `sync_engine`/`source_type`.
> **Scope**: Worker-side only. CMS + FE untouched (out of scope this pass).
> **Status**: Code complete. Build / vet / tests pass. Runtime startup verified.

---

## 1. Subject inventory before/after

### Before (12 subjects)
```
cdc.cmd.standardize
cdc.cmd.discover
cdc.cmd.backfill
cdc.cmd.scan-raw-data
cdc.cmd.bridge-airbyte
cdc.cmd.batch-transform
cdc.cmd.bridge-airbyte-batch
cdc.cmd.recon-check
cdc.cmd.recon-heal
cdc.cmd.retry-failed
cdc.cmd.debezium-signal
cdc.cmd.debezium-snapshot
```

### After (22 subjects) — +10 boundary-refactor additions
```
+ cdc.cmd.scan-fields
+ cdc.cmd.scan-source
+ cdc.cmd.refresh-catalog
+ cdc.cmd.airbyte-sync
+ cdc.cmd.sync-register
+ cdc.cmd.sync-state
+ cdc.cmd.restart-debezium
+ cdc.cmd.alter-column
+ cdc.cmd.import-streams
+ cdc.cmd.bulk-sync-from-airbyte
```

Result subjects mirror: `cdc.result.<subject-tail>`.

---

## 2. Per-handler specification

### 2.1 `HandleScanFields` — `cdc.cmd.scan-fields`
- **Payload**: `{registry_id, target_table, source_table, sync_engine, source_type, airbyte_source_id}`.
- **Route**:
  - `airbyte` → `airbyteClient.DiscoverSchema(source_id)` → parse JSON schema
    properties → INSERT mapping rules (`_*` system fields → `status=approved, rule_type=system`; rest → `status=pending`).
  - `debezium` → sample `_raw_data` last 100 rows (ORDER BY `_synced_at` DESC)
    → infer types via `service.InferTypeFromRawData` → INSERT rules.
  - `both` → Debezium primary, Airbyte fallback if Debezium returns zero total.
- **Result**: `{added, total, source_used}` on `cdc.result.scan-fields`.
- **Activity Log**: operation = `scan-fields`, trigger = `nats-command`.
- **Evidence**: `command_handler.go` ~L1100 (`scanFieldsAirbyte` + `scanFieldsDebezium`).

### 2.2 `HandleScanSource` — `cdc.cmd.scan-source`
- **Payload**: `{source_id, source_type}`.
- **Flow**: DiscoverSchema → iterate `catalog.Streams` → INSERT `cdc_table_registry`
  with `is_active=false` for new `source_db + source_table` combinations.
- **Result**: `{added, skipped, total}`.

### 2.3 `HandleRefreshCatalog` — `cdc.cmd.refresh-catalog`
- **Payload**: `{registry_id, airbyte_source_id}`.
- **Flow**: DiscoverSchema → record `schema_changes_log` entry with change_type
  `airbyte_catalog_refresh` and `airbyte_refresh_triggered=true`.
- **Result**: `{source_id, streams}`.

### 2.4 `HandleAirbyteSync` — `cdc.cmd.airbyte-sync`
- **Payload**: `{registry_id, connection_id}`.
- **Flow**: `airbyteClient.TriggerSync(connection_id)` → emit `job_id`.
- **Result**: `{connection_id, job_id}`.

### 2.5 `HandleSyncRegister` — `cdc.cmd.sync-register`
- **Payload**: `{registry_id, target_table, source_table, source_type, sync_engine, airbyte_connection_id, airbyte_source_id}`.
- **Route**:
  - `airbyte` → `syncWithAirbyte()` (DiscoverSchema + UpdateConnection with `selected=true`).
  - `debezium` → `verifyDebeziumConnector()` (GET Kafka Connect `/connectors`).
  - `both` → run both paths; status=error only if both fail.
- **Result**: `{airbyte_ok, debezium_ok, airbyte_error, debezium_error}`.

### 2.6 `HandleSyncState` — `cdc.cmd.sync-state`
- **Payload**: `{registry_id, action}` where action = `activate|deactivate`.
- **Flow**: Fetch registry entry. For Airbyte (via `ShouldUseAirbyte`) — toggle
  stream's `Selected` flag via UpdateConnection. For Debezium (via
  `ShouldUseDebezium`) — PUT Kafka Connect `/connectors/:name/pause` or
  `/resume`.
- **Result**: `{airbyte_status, debezium_status, error}`.

### 2.7 `HandleRestartDebezium` — `cdc.cmd.restart-debezium`
- **Payload**: `{connector_name}` (defaults to `goopay-mongodb-cdc`).
- **Flow**: POST `/connectors/:name/restart?includeTasks=true&onlyFailed=false`.
- **Security** (Rule 3): 10s timeout, single retry on 5xx, error otherwise.
- **Result**: `{connector_name}`.

### 2.8 `HandleAlterColumn` — `cdc.cmd.alter-column`
- **Payload**: `{target_table, column_name, data_type, action}` where action = `add|drop|alter_type`.
- **Security** (Rule 3):
  - Identifier whitelist `[A-Za-z0-9_-]` + length ≤ 64 (see `isSafeIdent`).
  - Data type allowlist (TEXT/BIGINT/INTEGER/NUMERIC/BOOLEAN/TIMESTAMP/JSONB/… via `isSafeType`).
- **SQL**: `ALTER TABLE "t" ADD/DROP/ALTER COLUMN "c" TYPE {safe-type}`.

### 2.9 `HandleImportStreams` — `cdc.cmd.import-streams`
- **Payload**: `{connection_id, streams:[{source_table, target_table, source_db, source_type}]}`.
- **Flow**: For each stream — registry insert if missing → `SELECT create_cdc_table(...)`
  on DW → seed default mapping rule for `id` column.
- **Result**: `{connection_id, added, skipped, total}`.

### 2.10 `HandleBulkSyncFromAirbyte` — `cdc.cmd.bulk-sync-from-airbyte`
- **Payload**: `{}` (walks all connections in workspace).
- **Flow**: `ListConnections` → `ListSources` → for each stream, insert registry
  row with Airbyte metadata + seed mapping rules from `JsonSchema.properties`
  for streams where `Config.Selected=true`.
- **Result**: `{connections, added, already_exists}`.

---

## 3. Registry helper service

File: `internal/service/source_router.go` (new).

- `ShouldUseAirbyte(entry) bool` — returns true when `sync_engine` ∈ {`airbyte`, `both`}.
- `ShouldUseDebezium(entry) bool` — true when ∈ {`debezium`, `both`}.
- `InferTypeFromRawData(value) string` — infers Postgres type from JSON sample:
  - `bool` → BOOLEAN
  - integer-valued `float64` → BIGINT; fractional → NUMERIC
  - RFC3339-like string → TIMESTAMP; else → TEXT
  - object/array → JSONB

Tests: `internal/service/source_router_test.go` — 3 test funcs, 20+ cases. All pass.

---

## 4. Wiring in `worker_server.go`

- Added config field `Debezium.KafkaConnectURL` + env override `KAFKA_CONNECT_URL`.
- New injection points:
  - `cmdHandler.SetKafkaConnectURL(cfg.Debezium.KafkaConnectURL)`
  - `cmdHandler.SetNATSConn(natsClient.Conn)` — needed so handlers can publish
    `cdc.result.*` events (the NATS library does not expose the connection from
    `*nats.Subscription`, forcing explicit wiring).
- 10 `Subscribe(...)` calls registered contiguously after the legacy block.
- Log-line subject list at line ~290 extended to include all 10 new subjects.

---

## 5. Airbyte client extensions (`pkgs/airbyte/client.go`)

Added to keep CMS parity per Rule 4 (no improvements, just port):

- `Stream.SourceDefinedPrimaryKey [][]string` — used by `HandleImportStreams`
  + `HandleBulkSyncFromAirbyte` for PK inference.
- `Stream.DefaultCursorField []string`.
- `Config.CursorField`, `Config.AliasName`.
- `JobStatus` struct.
- `DiscoverSchema(...)` — alias for existing `DiscoverSourceSchema` so CMS-style
  call sites work unchanged.
- `GetWorkspaceID() string` — best-effort via `getDefaultWorkspaceID`.
- `ListJobs(ctx, connectionID) ([]JobStatus, error)` — ported verbatim from CMS.

No existing methods changed (Rule 2 minimal impact).

---

## 6. Security gates (Rule 3)

| Risk | Mitigation | Location |
|:-----|:-----------|:---------|
| SQL injection via identifier | `isSafeIdent` whitelist | `HandleAlterColumn` |
| SQL injection via TYPE | `isSafeType` allowlist | `HandleAlterColumn` |
| Kafka Connect hangs | 10s `http.Client.Timeout` | `connectCall` |
| Kafka Connect transient 5xx | 1 retry on 5xx or transport error | `connectCall` |
| Airbyte secret leak | Reuses existing `airbyteClient` (no new creds handling) | — |

---

## 7. Verification

### Build
```
go build ./...    → 0 errors
go vet ./...      → 0 errors
```

### Unit tests
```
go test ./internal/handler/... -count=1
  ok  centralized-data-service/internal/handler  0.433s

go test ./internal/service/... -count=1
  ok  centralized-data-service/internal/service  0.809s
```

### Runtime startup verify
```bash
pkill -f "cdc-worker|cmd/worker"; sleep 2
nohup go run ./cmd/worker > /tmp/worker.log 2>&1 &
sleep 15
grep "command listeners registered" /tmp/worker.log
```

Result (trimmed):
```
"msg":"command listeners registered",
"subjects":[
  "cdc.cmd.standardize","cdc.cmd.discover","cdc.cmd.backfill",
  "cdc.cmd.scan-raw-data","cdc.cmd.bridge-airbyte","cdc.cmd.batch-transform",
  "cdc.cmd.bridge-airbyte-batch","cdc.cmd.recon-check","cdc.cmd.recon-heal",
  "cdc.cmd.retry-failed","cdc.cmd.debezium-signal","cdc.cmd.debezium-snapshot",
  "cdc.cmd.scan-fields","cdc.cmd.scan-source","cdc.cmd.refresh-catalog",
  "cdc.cmd.airbyte-sync","cdc.cmd.sync-register","cdc.cmd.sync-state",
  "cdc.cmd.restart-debezium","cdc.cmd.alter-column","cdc.cmd.import-streams",
  "cdc.cmd.bulk-sync-from-airbyte"
]
```

No `panic`, `fatal`, `ERROR` or `SQLSTATE` entries in startup window.

---

## 8. Out-of-scope (next waves)

- CMS thin-layer update: replace direct API logic with NATS publish (30m).
- FE polling hook for scan-fields async pattern.
- Property test suite using embedded NATS server (optional — CI deps).
- Per-handler Prom metrics (`cdc_<handler>_duration_seconds`).
- Kafka Connect circuit-breaker refinement (current: 1 retry; richer policy TBD).

---

## 9. Files touched

```
pkgs/airbyte/client.go                          (+3 structs, +3 funcs)
config/config.go                                (+2 fields, +2 env overrides)
internal/service/source_router.go               (NEW, 80 LOC)
internal/service/source_router_test.go          (NEW, 80 LOC tests)
internal/handler/command_handler.go             (+10 handlers, +helpers, ~650 LOC)
internal/server/worker_server.go                (+12 subscribe calls, +1 log field)
```

Nothing under `cdc-cms-service/` or `cdc-cms-web/` was modified (Rule 2, scope).

---

## 10. Rollback plan

1. `git revert` the single commit (this implementation).
2. Remove boundary-refactor subject subscriptions from `worker_server.go`.
3. Worker resumes 12-subject topology — no schema migration to roll back.
