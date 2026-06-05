# Report — Reconcile scheduler "skipped (MongoDB not configured)"

**Date**: 2026-05-26
**Workspace**: `bug-reconcile-mongodb-not-configured-2026-05-26`
**Trigger user**:
```
15:18:54 26/5/2026   reconcile   ALL   skipped   scheduler   reconCore not initialized (MongoDB not configured)
15:17:54 26/5/2026   reconcile   ALL   skipped   scheduler   reconCore not initialized (MongoDB not configured)
```

---

## TL;DR
Scheduler reconcile ghi "skipped" mỗi 60s do `cfg.MongoDB.URL` empty → reconCore=nil. Đây là legacy gate vi phạm L3100 (conditional gating bởi feature flag). V2 `connection_registry` đã có thông tin mongo URI nhưng worker không dùng. **Đã fix 3 lớp**:

1. V2 `MetadataRegistryService.ReloadAll` giờ populate `entry.SourceURL` từ `connection_registry` per-source.
2. ReconCore init **luôn** (bỏ guard cfg.MongoDB.URL). ReconSourceAgent lazy-resolve client per-source qua `entry.SourceURL`.
3. Hard-assert defensive trong `ReconSourceAgent.getClient` — error rõ ràng nếu cả entry.SourceURL=="" và defaultClient=nil.

---

## Root cause analysis

### Failure mode chain
```
config-local.yml KHÔNG có block `mongodb:`
  ↓
cfg.MongoDB.URL = "" sau viper.Unmarshal
  ↓
worker_server.go:176 `if cfg.MongoDB.URL != ""` SKIPPED toàn bộ init
  ↓
reconCore = nil
  ↓
Scheduler goroutine tick mỗi 60s (worker_server.go:712)
  ↓
schedule "reconcile" due → runReconcileCycle(now)
  ↓
L846 `if s.reconCore == nil` → activityLogger.Quick(..., "skipped", ...)
  ↓
activity_log ngập "skipped" mỗi phút
  ↓
Operator không thấy feature reconcile chạy → báo bug
```

Thậm chí nếu unblock reconCore (set cfg.MongoDB.URL=""), V2 entries không có SourceURL → `ReconSourceAgent.getClient(ctx, "")` rơi vào defaultClient=nil → panic ở `client.Database(...)`.

### Source of truth — code references
| Layer | File:line | Behavior trước fix |
|-------|-----------|--------------------|
| L1 legacy gate | `worker_server.go:176` | `if cfg.MongoDB.URL != ""` gate cả init |
| L2 V2 không populate URL | `metadata_registry_service.go:525-563` | `synthesizeLegacyTableRegistry` bỏ trống `SourceURL` |
| L3 silent default | `recon_source_agent.go:189-194` | sourceURL=="" → return `sa.defaultClient`, có thể nil |
| L4 silent skip | `worker_server.go:845-879` | `runReconcileCycle` ghi "skipped" mỗi tick |

---

## Fix applied

### File 1: `centralized-data-service/internal/service/metadata_registry_service.go`

**(a) Tách helper `resolveSourceURIFromConn(conn)` từ `GetSourceDSN`** (L378-410):
- `GetSourceDSN` giờ chỉ fetch row + delegate sang helper.
- Helper nhận sẵn `conn` → ReloadAll dùng trực tiếp, không re-query DB per code.

**(b) ReloadAll build `connectionURIByCode` map** (L178-197):
```go
connectionURIByCode := make(map[string]string, len(connections))
resolvedURIs := 0
for i := range connections {
    code := strings.TrimSpace(connections[i].ConnectionCode)
    if code == "" { continue }
    uri, resErr := rs.resolveSourceURIFromConn(&connections[i])
    if resErr != nil {
        rs.logger.Warn("connection_registry: cannot resolve source URI; recon/snapshot will skip sources bound to this connection",
            zap.String("connection_code", code),
            zap.String("engine", connections[i].EngineType),
            zap.Error(resErr))
        continue
    }
    connectionURIByCode[code] = uri
    resolvedURIs++
}
```

**(c) Pass URI vào synthesize** (L210-212):
```go
sourceConnCode := connectionCodeByID[src.SourceConnectionID]
sourceURI := connectionURIByCode[sourceConnCode]
cfg := synthesizeLegacyTableRegistry(src, binding, sourceURI)
```

**(d) Populate `cfg.SourceURL`** (L564-580):
```go
func synthesizeLegacyTableRegistry(..., sourceURI string) *model.TableRegistry {
    ...
    cfg := &model.TableRegistry{
        ...
        SourceURL: strings.TrimSpace(sourceURI),  // ← NEW
        ...
    }
}
```

### File 2: `centralized-data-service/internal/service/metadata_registry_service_test.go`
- Update 4 test call-sites sang signature 3 args (thêm `""` cho sourceURI).

### File 3: `centralized-data-service/internal/server/worker_server.go`

**(e) Bỏ guard `cfg.MongoDB.URL` quanh reconCore init** (L170-213):
- `mongoClientShared` gate by URL (legacy default client).
- ReconCore init **luôn** — defaultClient có thể nil.
- Log INFO "ReconCore initialized default_mongo_client=false source_uri_resolution=per-source via connection_registry (V2)".

**(f) Restructure reconHandler wiring** (L467-525):
- Healer / Backfill / TsDetector / FullCountAgg vẫn gate `mongoClientShared != nil`.
- ReconHandler luôn register 5 NATS subjects (handler tự return structured error khi service nil — nil-check sẵn có ở recon_handler.go L154, L457, L536).
- Bỏ stub subscriber block 511-548 cũ (thay bằng same handler với nil-safety).
- Log INFO mới: `recon_check_available`, `recon_heal_available`, etc.

**(g) Update runReconcileCycle defensive log** (L860-880):
- Cũ: `Warn("reconcile skipped — MongoDB not configured")`.
- Mới: `Error("wiring regression: ReconCore should be initialized unconditionally since 2026-05-26")`. Nếu trigger giờ là bug nghiêm trọng, không phải config missing.

### File 4: `centralized-data-service/internal/service/recon_source_agent.go`

**(h) Hard-assert trong getClient** (L189-203):
```go
func (sa *ReconSourceAgent) getClient(ctx context.Context, sourceURL string) (*mongo.Client, error) {
    if sourceURL == "" {
        if sa.defaultClient == nil {
            return nil, fmt.Errorf("recon: no mongo client available — entry.SourceURL is empty AND default client not configured. Verify the source_object_registry row's connection_code resolves to a valid Mongo URI in cdc_system.connection_registry, or set cfg.MongoDB.URL as a legacy default")
        }
        return sa.defaultClient, nil
    }
    ...
}
```

---

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` (centralized-data-service) | EXIT=0 |
| `go vet ./internal/...` | EXIT=0 |
| `go test ./internal/service/ -count=1 -run Recon` | PASS 0.771s |
| `go test ./internal/service/ -count=1` (full) | PASS 0.525s |
| `go test ./internal/handler/ -count=1` (full) | PASS 3.771s |
| Backward compat: cfg.MongoDB.URL set | preserved (default client init như cũ) |
| Bug A fix 2026-04-20 (Warn log silent-skip) | preserved + upgraded sang Error (defensive) |
| Lesson L3100 (every subject has subscriber) | preserved (recon-* luôn register) |

### ⚠️ Limitation — không thể test runtime live
Môi trường này không có:
- MongoDB live để test ReconCore.CheckAll thực sự gọi vào.
- Worker process running để observe scheduler tick.
- NATS broker để test subscribers.

**Manual smoke (cần operator chạy)**:
1. Start worker với config-local.yml hiện tại (KHÔNG sửa config).
2. Confirm startup log: `Reconciliation Core initialized default_mongo_client=false source_uri_resolution=per-source via connection_registry (V2)`.
3. Confirm `V2 metadata registry reloaded ... connection_uris_resolved=N` với N>0.
4. Đợi >30 phút (hoặc set `cdc_worker_schedule.interval_minutes=1` cho "reconcile" entry tạm thời) → quan sát activity_log:
   - **Trước fix**: liên tục "reconcile ALL skipped".
   - **Sau fix**: dispatch thật. Nếu Mongo URI từ V2 không reach được → error message rõ thay vì "skipped".

---

## Files changed (audit)
```
data-hub/centralized-data-service/internal/service/metadata_registry_service.go
data-hub/centralized-data-service/internal/service/metadata_registry_service_test.go
data-hub/centralized-data-service/internal/server/worker_server.go
data-hub/centralized-data-service/internal/service/recon_source_agent.go
```
**Tổng diff**: ~80 dòng net change.

## Memory files appended
```
agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26/00_context.md
agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26/01_requirements.md
agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26/02_plan.md
agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26/03_implementation.md
agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26/05_progress.md
agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26/08_tasks.md
agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26/09_tasks_solution.md
agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26/report_reconcile_mongodb_not_configured_2026-05-26.md (file này)
agent/memory/global/lessons.md (sẽ append)
agent/memory/global/active_plans.md (sẽ append)
```

## Skills sử dụng
- Read / Edit / Write / Bash
- Codebase tracing (worker_server → reconCore → reconSourceAgent → metadataRegistry → connection_registry)
- Lessons retrieval (L985 silent-skip pattern, L3100 conditional subscriber gating, L-CDC-route-empty-silent-skip-2026-05-26)
- TaskCreate / TaskUpdate (progress tracking)
- go build + go vet + go test (verify pipeline)
- Memory file APPEND-only (§11 GEMINI.md)
- Workspace prefix structure (§7 GEMINI.md)
- 2-layer fix + defense-in-depth (lesson pattern)
