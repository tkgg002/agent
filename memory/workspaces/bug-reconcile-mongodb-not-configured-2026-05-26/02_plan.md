# 02_plan — Fix reconcile scheduler skipped

## Root cause chain
```
1. config-local.yml KHÔNG có block `mongodb:`  → cfg.MongoDB.URL = ""
2. worker_server.go L176: `if cfg.MongoDB.URL != ""` SKIPPED → reconCore = nil
3. Scheduler ticker (60s) → runReconcileCycle()
4. L846: `if s.reconCore == nil` → WARN log + activity_log "skipped"
5. Feature reconcile DEAD trong local dev, mặc dù V2 connection_registry đã có
   mongo source data (admin nhập qua UI).
```

## Architecture decision
ReconSourceAgent **đã được thiết kế** cho multi-source:
- `sa.clients[sourceURL]` map — lazy create client per-URL.
- `getClient(ctx, sourceURL)` — empty URL → fallback defaultClient.
- `entry.SourceURL` được pass vào `CountInWindowWithFallback`, etc.

V2 `MetadataRegistryService.ListTableConfigs()` build `TableRegistry` synthetic
nhưng **bỏ trống `SourceURL` field**. → ReconSourceAgent rơi vào defaultClient
nil path → panic OR silent 0-count.

`MetadataRegistryService` đã có `GetSourceDSN(connectionCode)` đủ thông tin để
resolve URI cho mọi engine (mongo / postgres / mysql). Chỉ cần **wire qua**.

## Phase 1 — Populate `SourceURL` cho V2 entries
File: `internal/service/metadata_registry_service.go`

**Change A** (~L160-209 ReloadAll): build `sourceURIByCode` map ngay sau khi
build `connectionCodeByID`:
```go
connectionURIByCode := make(map[string]string, len(connections))
for i := range connections {
    code := strings.TrimSpace(connections[i].ConnectionCode)
    if code == "" {
        continue
    }
    uri, err := rs.GetSourceDSN(ctx, code)
    if err != nil {
        rs.logger.Warn("connection_registry: cannot resolve source DSN; recon will skip this source",
            zap.String("connection_code", code),
            zap.String("engine", connections[i].EngineType),
            zap.Error(err))
        continue
    }
    connectionURIByCode[code] = uri
}
```

**Change B** (~L183): pass URI vào `synthesizeLegacyTableRegistry`:
```go
sourceConnCode := connectionCodeByID[src.SourceConnectionID]
sourceURI := connectionURIByCode[sourceConnCode]
cfg := synthesizeLegacyTableRegistry(src, binding, sourceURI)
```

**Change C** (~L525): cập nhật signature + populate field:
```go
func synthesizeLegacyTableRegistry(src *model.SourceObjectRegistry, binding *model.ShadowBinding, sourceURI string) *model.TableRegistry {
    ...
    cfg := &model.TableRegistry{
        ID:        uint(src.ID),
        SourceDB:  sourceDB,
        SourceURL: strings.TrimSpace(sourceURI), // ← NEW
        ...
    }
    ...
}
```

## Phase 2 — Bỏ guard `cfg.MongoDB.URL` quanh ReconCore init
File: `internal/server/worker_server.go`

**Change D** (~L174-198): tách reconCore init thành 2 nhánh:
- Default client: `if cfg.MongoDB.URL != ""` → tạo `mongoClientShared`. Else: `mongoClientShared = nil`.
- ReconCore: init **luôn** với `(sourceAgent, destAgent, db, mongoClientShared (có thể nil), schemaAdapter, registryRepo, redisCache, ...)`. Log INFO khi defaultClient=nil — "ReconCore initialized in V2-only mode (defaultClient unavailable, per-source mongo URLs from connection_registry)".

**Change E**: giữ guard `if mongoClientShared != nil` cho ReconHealer + Backfill + TimestampDetector + FullCountAgg (chúng phụ thuộc shared default client — defer refactor sau). WARN log rõ "X disabled — set MongoDB.URL OR refactor X to use V2 connection_registry".

**Change F**: scheduler handler `runReconcileCycle` đã check `reconCore == nil`. Giữ nguyên Warn log nhưng update message phản ánh trạng thái mới ("reconCore=nil — bug fix 2026-04-20 chain still active, đáng lẽ không xảy ra sau Phase 2").

## Phase 3 — Hard-assert trong ReconSourceAgent
File: `internal/service/recon_source_agent.go`

**Change G** (~L189-216 `getClient`): nếu `sourceURL == "" && sa.defaultClient == nil` → return error rõ ràng:
```go
func (sa *ReconSourceAgent) getClient(ctx context.Context, sourceURL string) (*mongo.Client, error) {
    if sourceURL == "" {
        if sa.defaultClient == nil {
            return nil, fmt.Errorf("recon: no mongo client available — entry.SourceURL is empty AND default client not configured. Check V2 connection_registry has an active mongo connection bound to this source_object, or set cfg.MongoDB.URL for legacy fallback.")
        }
        return sa.defaultClient, nil
    }
    ...
}
```
Lý do: silent panic → operator visible error → report rõ trong reconciliation_report.error_message.

## Phase 4 — Verify
- `go build ./...` ở `centralized-data-service/`
- `go vet ./internal/...`
- `go test ./internal/service/ -count=1 -run Recon`
- Optional: `go test ./internal/handler/ -count=1`

## Risk audit
- **R1**: ReconCore.RunTier3 (Tier 3 bucket hash) có dùng `rc.mongoClient` không? — Đã grep: only L890 comment, no active read. SAFE.
- **R2**: ListTableConfigs() được gọi từ EventHandler / DynamicMapper path — populate thêm SourceURL không ảnh hưởng các consumer hiện tại (extra field, không break).
- **R3**: GetSourceDSN có thể timeout với connection_registry rows broken — đã có WARN log + skip, không block ReloadAll.
- **R4**: Backward compat: nếu cfg.MongoDB.URL được set + V2 chưa active → defaultClient được dùng cho entries có SourceURL="" → behavior cũ giữ nguyên.
- **R5**: Local dev không có Mongo running → reconCore CheckAll sẽ thử connect mongo (V2 connection_registry URI có thể trỏ docker hostname không reach từ host) → vẫn fail nhưng giờ với error message rõ "cannot connect to mongodb://..." thay vì "MongoDB not configured" — accurate operator-facing message.

## Cộng diff dự kiến
- `metadata_registry_service.go`: +20 / -3
- `worker_server.go`: +15 / -10 (restructure guard)
- `recon_source_agent.go`: +5 / -1
- **Tổng**: ~50 dòng net change.
