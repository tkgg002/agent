# 10 — Gap Analysis v2: Full System Audit (All Layers)

> **Cập nhật**: 2026-06-18  
> **Phương pháp**: Scan toàn bộ API handlers + Commands + Infra — không chỉ nhóm command

---

## Định nghĩa Saga (đã làm rõ)

> **Saga cần khi**: Operation chạm vào ≥2 data stores (DB, NATS, External HTTP, Redis...) theo thứ tự,  
> và store thứ nhất đã committed nhưng store thứ hai có thể fail.  
> **Không cần** DB constraint check, chỉ cần: "store A thành công + store B fail → state inconsistent"

---

## LAYER 1: API Handlers — Multi-Store Operations

### 🔴 SOURCE — registry_handler_register.go (CRITICAL)

```
bus.Execute(registerCmd)      ← DB write (register + shadow DDL + NATS reload)
  ↓ success
sourceRepo.ResolveShadowSchema()  ← DB read
  ↓
bus.Dispatch(createDefaultColumnsCmd)  ← NATS publish (worker)
  ↓
bus.Execute(V2SyncCmd)            ← DB write (v2 sync)
  ↓
bus.Dispatch(RestartDebeziumCmd)  ← NATS publish (worker)
```
**Saga cần**: Nếu V2Sync fail sau khi Register xong → registry exists nhưng v2 sync corrupt  
**Action**: Saga ở API handler level, KHÔNG chỉ ở command level

---

### 🔴 SOURCE — registry_handler_bulk.go

```
for each entry:
  bus.Execute(registerCmd)         ← DB write per entry
  bus.Dispatch(createDefaultCols)  ← NATS per entry
  bus.Execute(V2SyncCmd)           ← DB write per entry
```
**Saga cần**: Batch partial fail — entry 3/10 fail, entries 1-2 đã committed

---

### 🔴 SHADOW — mapping_rule_handler_batch.go — BatchUpdate (CRITICAL)

```
for each ruleID:
  bus.Execute(UpdateMappingRuleCommand)   ← DB write (status=approved)
  bus.Dispatch(AlterColumnCommand)        ← NATS → worker ALTER TABLE
  bus.Dispatch(BackfillCommand)           ← NATS → worker backfill data
  natsClient.PublishReload(table)         ← NATS fire-and-forget
```
**Saga cần**: DB update xong mà AlterColumn fail → rule `approved` nhưng column không có trong shadow table

---

### 🔴 SOURCE — registry_handler_update.go

```
bus.Execute(UpdateRegistryCmd)     ← DB write
  ↓ success (nếu v2 sync needed)
bus.Execute(V2SyncCmd)             ← DB write (v2 sync)
bus.Dispatch(RestartDebeziumCmd)   ← NATS publish
```
**Saga cần**: V2Sync fail sau Update → registry state và v2 mapping lệch nhau

---

### 🔴 SCHEDULER — snapshot_progress_handler.go

```
natsConn.Publish("pause")      ← NATS fire-and-forget
natsConn.Publish("snapshot.v2")  ← NATS publish
```
**Saga**: N/A vì không có DB write trước — chỉ cần tracing

---

## LAYER 2: Commands — Multi-Store Operations (revised)

### S1 🔴 — `source/register_registry.go`
```
sourceRepo.Register()          ← DB write
  → Compensate: DeleteRegistry()
automator.EnsureShadowTable()  ← External DDL
  → Compensate: cleanup (best-effort)
nats.PublishReload()           ← NATS
  → Compensate: nil
```

### S2 🔴 — `governance/approve_master.go`
```
repo.ApproveSchemaTx()         ← DB write
  → Compensate: RevertSchemaTx()
publisher.Publish()            ← NATS
  → Compensate: nil
```

### S3 🔴 — `governance/approve_schema_proposal.go`
```
[multi-step DDL chain — xem 03_implementation_saga.md]
```

### S4 🔴 — `master/create_master.go`
```
masterRepo.CreateMasterBinding()  ← DB write
  → Compensate: DeleteMasterBinding()
masterRepo.CloneMappingRules()    ← DB write
  → Compensate: DeleteClonedRules()
```

### S5 🔴 — `source/debezium_connector.go` (Create + Delete)
```
Create:
  w.CreateConnector()  ← External HTTP (KafkaConnect)  [STEP 1 — source of truth]
    → Compensate: w.DeleteConnector()
  repo.SaveFingerprint()  ← DB write (audit trail, best-effort)  [STEP 2]
    → Compensate: FullCleanup()

Delete:
  w.DeleteConnector()  ← External HTTP (404 = idempotent OK)  [STEP 1]
    → Compensate: nil (cannot safely re-create without full config)
  repo.FullCleanup()   ← DB write  [STEP 2]
    → Compensate: nil (idempotent)
```
> **Design decision (Q2)**: HTTP-first, DB-second — HTTP is source of truth for connector state.
> DB fingerprint is audit/metadata; best-effort failure on DB does NOT fail the saga.
> **Status**: ✅ Implemented in `debezium_connector.go` via saga.Runner.

### S6 ✅ — `master/approve_ddl_executor.go` — Implemented
```
Saga: ddl.approve
  nats-publish-reconcile — PublishMasterReconcile (SYNC blocking, 60s timeout)
    → Compensate: nil (DDL already executed by worker — irreversible)
  db-clear-pending-flags — ClearPendingDDLFlags
    → Compensate: nil (idempotent on retry: worker ALTER is idempotent)
```
> **Status**: ✅ Implemented via `saga.New("ddl.approve", ...)`. Build + tests pass.

### S7 ✅ — `master/drop_column.go` — Implemented
```
Saga: column.drop
  nats-publish-drop — PublishMasterDropColumn (SYNC blocking)
    → Compensate: nil (DROP already executed — cannot re-add column)
  db-update-in-master-status — UpdateInMasterStatus
    → Compensate: nil (idempotent: next DROP will be IF EXISTS)
```
> **Status**: ✅ Implemented via `saga.New("column.drop", ...)`. Build + tests pass.
> ⚠️ `drop_rejected_columns.go` S7b: NOT IMPLEMENTED — uses fire-and-forget per-column pattern.
> This is acceptable because it already retries safely (next call re-attempts failed columns).

---

## LAYER 3: Tracing Coverage — Tất cả luồng cần span

### HTTP Entry Points (OtelPropagator middleware)
- Mọi Fiber handler → auto covered khi middleware được register

### Command Bus
- `Execute()` → span `command_bus.execute` ✅
- `Dispatch()` → span `command_bus.dispatch` ✅

### API Handler Level (tracing chủ động tại handler)
Một số handler gọi nhiều bus ops — cần span per-handler để thấy rõ hơn:

| Handler | Flow | Span cần |
|---------|------|---------|
| `registry_handler_register.go` | Execute + Dispatch + Execute + Dispatch | `api.source.register` parent span |
| `registry_handler_bulk.go` | Loop N×(Execute + Dispatch + Execute) | `api.source.bulk-register` parent span |
| `mapping_rule_handler_batch.go` | Loop N×(Execute + Dispatch + Dispatch + Publish) | `api.shadow.batch-update` parent span |
| `registry_handler_update.go` | Execute + Execute + Dispatch | `api.source.update-registry` parent span |

### Infra Layer (Tracing thụ động — qua ctx propagation)
- `nats_command_bus.go` → đã có span ở Execute/Dispatch
- `KafkaConnectWriter` → cần span nếu có HTTP calls
- `ShadowAutomator.EnsureShadowTable()` → cần span (DDL operation)

### Background Jobs (không qua HTTP)
- `stuck_job_reaper.go` → không cần span (internal background)
- `system_health_collector.go` → không cần span (metrics collection)

---

## Saga Risk Summary (Revised — All Layers)

| ID | Layer | Location | Risk |
|----|-------|----------|------|
| S1 | Command | `source/register_registry.go` | 🔴 DB + DDL + NATS |
| S2 | Command | `governance/approve_master.go` | 🔴 DB + NATS |
| S3 | Command | `governance/approve_schema_proposal.go` | 🔴 DB + DDL + DB + DB |
| S4 | Command | `master/create_master.go` | 🔴 DB + DB |
| S5 | Command | `source/debezium_connector.go` | 🔴 DB + External HTTP |
| A1 | API Handler | `source/registry_handler_register.go` | 🔴 3 operations sequentially |
| A2 | API Handler | `source/registry_handler_bulk.go` | 🔴 N×3 operations in loop |
| A3 | API Handler | `shadow/mapping_rule_handler_batch.go` | 🔴 N×(DB+NATS+NATS+NATS) |
| A4 | API Handler | `source/registry_handler_update.go` | 🟡 DB + DB + NATS |

---

## Saga Strategy theo Layer

### Command Level (S1-S5): saga.Runner
→ Local compensation trong handler

### API Handler Level (A1-A4): Error logging + partial rollback hints
→ API handlers gọi nhiều commands sequentially — nếu step 2 fail không có cơ chế rollback step 1  
→ **Strategy**: Thêm rollback command dispatch khi bước sau fail  
→ **Không** dùng saga.Runner tại API layer (quá phức tạp cho layer này, vi phạm separation of concerns)

---

## Tracing Gap Summary (All Layers)

| Gap | Layer | Severity |
|-----|-------|---------|
| G1: Không extract W3C traceparent | HTTP Middleware | 🟡 |
| G2: CommandBus không tạo span | Infra/Messaging | 🟡 |
| G3: Saga không có span | App/Saga | 🟡 |
| G4: API handler multi-op không có parent span | API | 🟡 |
| G5: TextMapPropagator chưa set | pkgs/observability | 🔴 (blocks all tracing) |
