# Gap Analysis — ScanFields Architectural Violation

> **Date**: 2026-04-20
> **Trigger**: User architectural review 3 câu về `ScanFields`
> **Severity**: HIGH (3 violations compound)
> **Status**: ✅ **CLOSED 2026-04-20** — User approved Option A full refactor. 3 Muscle parallel (Worker + CMS + FE) executed. Evidence in `03_implementation_v3_{worker,cms,fe}_boundary_refactor.md` + `03_implementation_v3_fe_async_dispatch.md`.

---

## 1. Violations identified

| # | Rule violated | Evidence | Severity |
|:--|:--------------|:---------|:---------|
| 1 | **NATS async pattern** (ADR-015) | `registry_handler.go:866-885` HTTP sync block 5-30s | HIGH |
| 2 | **Service boundary** (ADR service_boundary_analysis_1) | API layer calls Airbyte Discovery + INSERTs mapping_rules | HIGH |
| 3 | **Multi-source support** (registry schema) | Hardcoded `AirbyteSourceID` check (line 877), ignores `SyncEngine`/`SourceType` | HIGH |

All 3 violations compound: ScanFields broken for Debezium-only tables, blocks UI, bypasses Worker boundary.

---

## 2. Correct pattern (already established)

- **Discover** (`registry_handler.go:488`): Publish NATS `cdc.cmd.discover` → 202 → Worker handles.
- **Introspect** (`introspection_handler.go:36-51`): NATS Request-Reply dual path Debezium-first + Airbyte-fallback.
- **Scan-raw-data** (`command_handler.go:290`): Worker reads `_raw_data` JSONB for Debezium tables.

ScanFields is outlier duy nhất.

---

## 3. Refactor proposal

### 3.1 Move logic CMS → Worker
- New Worker handler: `HandleScanFields(msg *nats.Msg)` trong `command_handler.go`.
- Subscribe subject `cdc.cmd.scan-fields` trong `worker_server.go:281`.
- Logic route theo `entry.SyncEngine`:
  - `airbyte` → reuse `DiscoverSchema` Airbyte client (move từ CMS).
  - `debezium` → scan `_raw_data` JSONB (reuse logic từ `scan-raw-data` handler).
  - `both` → Debezium primary, Airbyte fallback (pattern từ Introspect).

### 3.2 CMS handler giữ thin layer
```go
// cdc-cms-service/internal/api/registry_handler.go
func (h *RegistryHandler) ScanFields(c *fiber.Ctx) error {
    entry, _ := h.repo.GetByID(...)
    payload, _ := json.Marshal(map[string]any{
        "registry_id":  entry.ID,
        "target_table": entry.TargetTable,
        "source_table": entry.SourceTable,
        "sync_engine":  entry.SyncEngine,
        "source_type":  entry.SourceType,
    })
    h.natsClient.Conn.Publish("cdc.cmd.scan-fields", payload)
    h.logAction("scan-fields", entry.TargetTable, "accepted", ...)
    return c.Status(202).JSON(fiber.Map{"message": "scan-fields accepted"})
}
```

### 3.3 FE polling / status query
- FE dùng `useMutation` để fire POST → nhận 202.
- Poll `GET /api/registry/:id/scan-status` mỗi 3s cho đến `done`.
- Hoặc show toast "scan dispatched, view progress in Activity Log".

### 3.4 Worker result publish
- After scan complete → `cdc.result.scan-fields` với `{added, total, skipped}`.
- Activity Log entry operation=`scan-fields`.
- Prom metric `cdc_scan_fields_duration_seconds{sync_engine, source_type}`.

---

## 4. Files impact

### Worker
- `internal/handler/command_handler.go` — NEW `HandleScanFields` (+80 LOC)
- `internal/server/worker_server.go` — subscribe subject (+2 LOC)
- `pkgs/airbyte/client.go` — verify Airbyte client accessible (có thể cần move)
- `internal/service/scan_service.go` — reuse `_raw_data` JSONB scanner

### CMS
- `internal/api/registry_handler.go` — `ScanFields` reduce từ 105 LOC → ~20 LOC (publish + 202)
- `internal/api/registry_handler.go` — NEW `ScanFieldsStatus(c)` GET endpoint cho polling

### FE
- `src/hooks/useRegistry.ts` — NEW `useScanFieldsMutation` + `useScanFieldsStatus`
- `src/pages/TableRegistry.tsx` — poll status thay chờ blocking response

---

## 5. Effort estimate

~3-4 hours:
- Worker handler + Airbyte migration: 2h
- CMS thin layer: 30m
- FE polling hook: 1h
- Test + verify: 30m

---

## 6. Risk nếu giữ current

- Bảng Debezium-only (Mongo) **không thể scan-fields** qua UI.
- UI timeout nếu Airbyte Discovery chậm (>30s default HTTP timeout).
- Scaling: N tables × scan-fields concurrent → CMS thread exhaustion.
- Audit: không có activity log per scan run.
- Consistency: khác pattern 4 other `cdc.cmd.*` operations → confusing for contributors.

---

## 7. Risk nếu refactor

- Thêm async complexity (polling/websocket FE).
- Test coverage cần viết mới cho Worker handler + dual-source routing.
- Migration: existing ScanFields API callers cần update (nếu có — current chỉ FE gọi).

**Tradeoff**: refactor win, chấp nhận complexity để đúng architecture.

---

## 8. Decision required

User approve 1 trong 3 options:

**A. Full refactor** (recommended): Move Worker + multi-source routing + FE polling. 3-4h.

**B. Partial fix** (minimum): Keep CMS-side, add `sync_engine` routing within CMS (unblocks Debezium tables nhưng vẫn HTTP sync + boundary violation). 1h.

**C. Document only** (defer): Ghi lại technical debt, không fix ngay. 0h.

---

## 9. Related

- `service_boundary_analysis_1.md` — 4 violations cũ đã fix, ScanFields là #5 chưa fix
- `04_decisions_nats_vs_kafka.md` — ADR-015 NATS async pattern
- `03_implementation_debezium_nats.md` — NATS subject catalog
- Lesson global 2026-03-31 "Service Boundary Analysis" — chưa enforce check cho new endpoints
