# 03_implementation_v3_cms_boundary_refactor

> **Scope**: CDC-CMS-Service — refactor 11 endpoints + 2 fixes sang thin-layer
> publish NATS (cdc.cmd.*), đảm bảo Rule B (config-write sync, external mutate
> async) + Rule C (registry-aware routing theo `sync_engine`).
>
> **Agent**: Muscle (CC CLI)
> **Date**: 2026-04-17
> **Worker**: Parallel agent tạo handler `cdc.cmd.{scan-fields,sync-register,sync-state,scan-source,refresh-catalog,airbyte-sync,restart-debezium,alter-column,import-streams,bulk-sync-from-airbyte,create-default-columns}`
>
> Tuân Rule 3 (Plan & Verify), Rule 6 (Simplicity First + Minimal Impact),
> Rule 7 (Persist in workspace), Rule 8 (Security Gate sau khi xong).

---

## 1. Pattern chuẩn (per Discover handler ref `registry_handler.go:488` pre-refactor)

```go
func (h *Handler) XxxAction(c *fiber.Ctx) error {
    entry, _ := h.repo.GetByID(...)                      // DB read
    payload, _ := json.Marshal(map[string]any{
        "registry_id": entry.ID,
        "sync_engine": entry.SyncEngine,
        "source_type": entry.SourceType,
        // ... fields worker cần
    })
    if err := h.natsClient.Conn.Publish("cdc.cmd.xxx", payload); err != nil {
        h.logAction("xxx", entry.TargetTable, "error", nil, err.Error())
        return c.Status(500).JSON(fiber.Map{"error": "dispatch failed: "+err.Error()})
    }
    h.logAction("xxx", entry.TargetTable, "accepted", map[string]interface{}{"user": middleware.GetUsername(c)}, "")
    return c.Status(202).JSON(fiber.Map{
        "message":      "xxx command accepted",
        "target_table": entry.TargetTable,
    })
}
```

Hai nhóm:
- **Pure external** (scan-fields, sync, refresh-catalog, scan-source, restart-debezium, bulk-sync-from-airbyte, import-streams) → publish → 202.
- **Hybrid** (register, PATCH registry, batch, mapping-rules/batch) → DB write sync + publish NATS cho external side → 202 với `{entry, dispatched: [...]}`.

---

## 2. Diff summary per endpoint

### (1) POST /api/registry/:id/scan-fields — `registry_handler.go:ScanFields`
- **Before**: `h.airbyteClient.DiscoverSchema` đồng bộ → parse JSONSchema → insert hàng loạt `cdc_mapping_rules` → return 200. Latency 5-30s.
- **After**: payload `{registry_id, sync_engine, source_type, source_db, source_table, target_table, airbyte_source_id}` → publish `cdc.cmd.scan-fields` → 202.
- **Response**: `{message, target_table, sync_engine}`.

### (2) POST /api/registry — `RegistryHandler.Register`
- **Before**: `syncWithAirbyte` block 5-15s (list sources, discover, list connections, get+update connection) → `repo.Create` → `SELECT create_cdc_table(...)` → `PublishReload` → 201.
- **After**: `repo.Create` (sync config-write) → publish `cdc.cmd.create-default-columns` → nếu `sync_engine ∈ {airbyte,both}` publish `cdc.cmd.sync-register` → `PublishReload` → 202.
- **Response**: `{message, entry, dispatched: [...]}`.
- **Helper xóa**: `syncWithAirbyte` (74 LoC) — moved to Worker.

### (3) PATCH /api/registry/:id — `RegistryHandler.Update`
- **Before**: Selective update (sync) + `syncRegistryStateToAirbyte` block 3-10s (discover schema, merge catalog, UpdateConnection) → 200.
- **After**: Selective update (sync) + publish `cdc.cmd.sync-state` nếu `IsActive|SyncEngine` đổi & entry là airbyte/both & có `AirbyteConnectionID` → 202.
- **Response**: `{message, entry, dispatched: [...]}`.
- **Helper xóa**: `syncRegistryStateToAirbyte` (156 LoC) — moved to Worker.

### (4) POST /api/registry/batch — `RegistryHandler.BulkRegister`
- **Before**: `BulkCreate` + blocking `SELECT create_all_pending_cdc_tables()` → 201.
- **After**: `BulkCreate` (sync) → re-query để lấy ID → loop publish `cdc.cmd.create-default-columns` per entry → 202.
- **Response**: `{message, created, dispatched}`.

### (5) POST /api/registry/scan-source — `RegistryHandler.ScanSource`
- **Before**: `DiscoverSchema` block → loop create registry entries in DB → 200. Latency 2-10s.
- **After**: publish `cdc.cmd.scan-source` với `{airbyte_source_id, triggered_by}` → 202.

### (6) POST /api/registry/:id/sync — `RegistryHandler.Sync`
- **Before**: `TriggerSync` Airbyte block → 200 với `{job_id}`. Latency 1-5s.
- **After**: Validate `sync_engine ∈ {airbyte,both}` (400 nếu Debezium) + có `AirbyteConnectionID` → publish `cdc.cmd.airbyte-sync` → 202.

### (7) POST /api/registry/:id/refresh-catalog — `RegistryHandler.RefreshCatalog`
- **Before**: `DiscoverSchema` block → 202 nhưng vẫn wait Airbyte. Latency 2-10s.
- **After**: Validate engine + source_id → publish `cdc.cmd.refresh-catalog` → 202 immediate.

### (8) POST /api/registry/sync-from-airbyte — `RegistryHandler.SyncFromAirbyte`
- **Before**: `ListConnections` + `ListSources` loop + tạo registry entries + mapping rules + `PublishReload`. 60+ LoC helper `createMappingRulesFromSchema` + `inferDataTypeFromSchema`. Latency 10-60s.
- **After**: publish `cdc.cmd.bulk-sync-from-airbyte` với `{workspace_id, triggered_by}` → 202.
- **Helpers xóa**: `createMappingRulesFromSchema`, `inferDataTypeFromSchema`, `inferSQLType`.

### (9) POST /api/airbyte/import/execute — `AirbyteHandler.ExecuteImport`
- **Before**: `GetConnection` → loop: `Create(entry)` + `Create(rule)` + `SELECT create_cdc_table(...)`. 5-20s.
- **After**: publish `cdc.cmd.import-streams` với `{connection_id, stream_names, triggered_by}` → 202.

### (10) PATCH /api/mapping-rules/batch — `MappingRuleHandler.BatchUpdate`
- **Before**: Loop `UpdateStatus` (sync OK) + nếu approved → blocking `ALTER TABLE ... ADD COLUMN`. 2-10s tùy số rules.
- **After**: Loop `UpdateStatus` (sync) + nếu approved → publish `cdc.cmd.alter-column` per rule → 202.
- **Response**: `{message, updated, dispatched, backfilled, total}`.

### (11) POST /api/tools/restart-debezium — `SystemHealthHandler.RestartDebezium`
- **Before**: `http.Client` POST tới `kafkaConnectURL/connectors/.../restart`. 1-10s.
- **After**: publish `cdc.cmd.restart-debezium` với `{connector_name, kafka_connect_url}` → 202.
- **Constructor change**: `NewSystemHealthHandler` nhận thêm `*natsconn.NatsClient`. Updated `server.go` wiring.

### (12) GET /api/registry/:id/status — `RegistryHandler.GetStatus`
- **Before**: Call Airbyte `GetConnection` không check `SyncEngine`. Với entry Debezium gây 500 (không có Airbyte connection).
- **After**: Validate `sync_engine ∈ {airbyte,both}` ngay từ đầu. Debezium entries → 400 với hint redirect `/api/system/health`. Response thêm `sync_engine`.

### (13) GET /api/sync/health — `RegistryHandler.SyncHealth`
- **Before**: `ListConnections` loop ALL Airbyte connections không filter → double-count các connection không thuộc registry.
- **After**: Pluck `airbyte_connection_id` distinct từ registry có `sync_engine ∈ {airbyte,both}` → filter ListConnections qua whitelist. Thêm metrics `total_registered_airbyte`.

### NEW: GET /api/registry/:id/dispatch-status — `RegistryHandler.DispatchStatus`
- Query `cdc_activity_log WHERE target_table = entry.target_table AND operation = subject AND started_at >= since`, ORDER BY started_at DESC LIMIT 50.
- Subject chấp nhận cả `cdc.cmd.scan-fields` lẫn `scan-fields`.
- Response: `{target_table, operation, since, entries, count}` — FE poll status transitions `accepted → running → success|error`.
- Mount: `shared.Get("/registry/:id/dispatch-status", ...)` (admin + operator).

---

## 3. Registry-aware routing (Rule C) examples

```http
# Debezium entry (id=42, sync_engine=debezium)
POST /api/registry/42/sync → 400
{"error":"sync requires sync_engine=airbyte|both","sync_engine":"debezium"}

POST /api/registry/42/refresh-catalog → 400
{"error":"refresh-catalog requires sync_engine=airbyte|both","sync_engine":"debezium"}

GET /api/registry/42/status → 400
{"error":"status endpoint requires sync_engine=airbyte|both",
 "hint":"for debezium entries, use /api/system/health or Debezium connector endpoint",
 "sync_engine":"debezium"}

# Airbyte entry (id=41) — everything allowed
POST /api/registry/41/sync → 202
{"message":"airbyte-sync command accepted","target_table":"identitycounters",
 "connection_id":"0cd18604-..."}
```

---

## 4. Latency matrix (runtime smoke 2026-04-17)

| # | Endpoint | Before (est.) | After (measured) | Speedup |
|:--|:---------|:--------------|:------------------|:-------|
| 1 | scan-fields | 5-30s | 22ms | ~227-1364x |
| 2 | POST /registry | 5-15s | 38ms | ~131-394x |
| 3 | PATCH /registry/:id | 3-10s | 4ms | ~750-2500x |
| 4 | POST /registry/batch | 5-30s | 3ms | ~1666-10000x |
| 5 | scan-source | 2-10s | 4ms | ~500-2500x |
| 6 | sync | 1-5s | 4ms | ~250-1250x |
| 7 | refresh-catalog | 2-10s | 4ms | ~500-2500x |
| 8 | bulk-sync-from-airbyte | 10-60s | 3ms | ~3333-20000x |
| 9 | import/execute | 5-20s | 0.4ms | ~12500-50000x |
| 10 | mapping-rules/batch | 2-10s | 3ms | ~666-3333x |
| 11 | restart-debezium | 1-10s | 3ms | ~333-3333x |
| 12 | GET /registry/:id/status (airbyte) | 100-500ms | 139ms | unchanged (read Airbyte) |
| 12b | GET /registry/:id/status (debezium) | often 500 | 1ms (400) | root-caused |
| 13 | GET /sync/health | 100-500ms | 59ms | filtered, faster |

Runtime check target `HTTP=202 time<0.1s` cho tất cả async endpoints — ĐẠT.

---

## 5. Files changed

| File | Lines before | Lines after | Change |
|:-----|:-------------|:------------|:------|
| `internal/api/registry_handler.go` | 1515 | ~1150 | -365 (removed 3 helpers; rewrote 9 handlers) |
| `internal/api/airbyte_handler.go` | 317 | ~266 | -51 (rewrote ExecuteImport, add json import) |
| `internal/api/mapping_rule_handler.go` | 290 | ~295 | +5 (alter async, remove fmt import) |
| `internal/api/system_health_handler.go` | 135 | ~130 | -5 (http removed, nats added) |
| `internal/server/server.go` | ~160 | ~161 | +1 (pass natsClient) |
| `internal/router/router.go` | ~234 | ~235 | +1 (dispatch-status route) |

Total: ~-415 LoC (CMS thinner).

---

## 6. Security (Rule 8)

Giữ nguyên Phase 4 chain cho destructive endpoints:
- `POST /tools/restart-debezium`: `RequireOpsAdmin` + `RateRestart (3/h)` + `Idempotency` + `Audit`.
- Các destructive khác (`registerDestructive`): `RequireOpsAdmin` + `Idempotency` + `Audit`.
- Verify: `POST /tools/restart-debezium` không có `reason` → 400 `missing or too-short reason` (middleware chain intact).
- Verify: `POST /tools/restart-debezium` có reason ≥10 chars → 202 + audit insert.

RBAC unchanged: shared (admin+operator) cho read + `dispatch-status`; admin-only cho register/update/batch/scan-source/sync-from-airbyte/sync/refresh-catalog/scan-fields/import/execute.

---

## 7. Fire-and-forget mode (Rule brief #6)

Nếu Worker handler chưa subscribe → NATS publish vẫn thành công (JetStream persist / core NATS drops). CMS response 202 không phụ thuộc Worker. Parallel Muscle (Worker) sẽ pick up.

Failure mode documentation:
- Publish error (NATS down) → 500 với `dispatch failed: <err>`.
- Worker down nhưng NATS up → 202 OK, message sẽ queue hoặc drop tùy subject config.
- FE polling `GET /registry/:id/dispatch-status?subject=<subject>&since=<ts>` để observe `accepted → running → success|error` transitions.

---

## 8. Test evidence

```
go build ./...                 → PASS (silent)
go vet ./...                   → PASS (silent)
go test ./internal/api/...     → ok cdc-cms-service/internal/api 0.527s
```

Runtime (server on :8083):
```
startup log clean (no WARN/FATAL on boot)
all 11 endpoints → HTTP=202, time < 140ms
#12b (debezium status) → HTTP=400 (before: often 500)
#13 (sync/health) → filtered by sync_engine
NEW dispatch-status → HTTP=200 returning activity log entries
```

---

## 9. Known follow-ups (out of this refactor scope)

- Worker side: handlers `cdc.cmd.alter-column`, `cdc.cmd.restart-debezium`, `cdc.cmd.import-streams` etc. (tracked in worker workspace).
- FE: switch từ 200 reading response data → 202 + polling `dispatch-status` (tracked in FE workspace).
- Delete dead code `Kafka Connect REST URL` config nếu Worker hoàn toàn take over.
