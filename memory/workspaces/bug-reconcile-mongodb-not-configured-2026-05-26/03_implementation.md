# 03_implementation — Reconcile MongoDB-not-configured fix

## File 1 — `internal/service/metadata_registry_service.go`

### Change A (~L341-405) — Tách helper `resolveSourceURIFromConn`
- `GetSourceDSN(ctx, code)` giờ chỉ fetch conn rồi delegate sang
  `resolveSourceURIFromConn(conn)`.
- `resolveSourceURIFromConn(conn *model.ConnectionRegistry) (string, error)`
  giữ nguyên resolution chain (ApplyConnectionOverride → tryPlainDSN(Host) →
  tryEnvPointer(Host) → tryPlainDSN(SecretRef) → tryEnvPointer(SecretRef) →
  buildDSNFromFields → DecryptAES).
- Lợi ích: caller đã có `conn` (vd ReloadAll) không cần re-fetch by code.

### Change B (~L178-197) — ReloadAll build `connectionURIByCode`
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

### Change C (~L210-212) — Pass URI vào synthesize
```go
sourceConnCode := connectionCodeByID[src.SourceConnectionID]
sourceURI := connectionURIByCode[sourceConnCode]
cfg := synthesizeLegacyTableRegistry(src, binding, sourceURI)
```
(và xóa dòng `sourceConnCode :=` ở L225 cũ vì đã khai báo trên).

### Change D (~L564-580) — Update signature + populate field
```go
func synthesizeLegacyTableRegistry(src *model.SourceObjectRegistry, binding *model.ShadowBinding, sourceURI string) *model.TableRegistry {
    ...
    cfg := &model.TableRegistry{
        ID:        uint(src.ID),
        SourceDB:  sourceDB,
        ...
        SourceURL: strings.TrimSpace(sourceURI),  // ← NEW
        ...
    }
    ...
}
```

### Change E — Update log INFO
Thêm `connection_uris_resolved` count vào "V2 metadata registry reloaded" line.

### Change F — Update test signatures
`metadata_registry_service_test.go` 4 call-sites đổi sang 3-arg form
(thêm `""` cho sourceURI — test path không cần URI thật).

## File 2 — `internal/server/worker_server.go`

### Change G (~L174-213) — Tách reconCore khỏi guard `cfg.MongoDB.URL`
**Trước**: `if cfg.MongoDB.URL != "" { mc, err := ...; reconCore = ... }` →
toàn bộ init reconCore bị skip khi URL empty.

**Sau**:
```go
var mongoClientShared *mongo.Client
if cfg.MongoDB.URL != "" {
    mc, err := mongodb.NewClient(...)
    if err != nil {
        logger.Warn("MongoDB default client connection failed; ReconCore will operate in V2-only mode (per-source URIs from connection_registry)", zap.Error(err))
    } else {
        mongoClientShared = mc
        logger.Info("MongoDB default client connected (legacy cfg.MongoDB.URL)")
    }
} else {
    logger.Info("cfg.MongoDB.URL not set; ReconCore will operate in V2-only mode (per-source URIs from connection_registry). ReconHealer / Backfill / TimestampDetector / FullCountAgg remain disabled until a default client is available.")
}

sourceAgent := service.NewReconSourceAgent(mongoClientShared, logger)
destAgent := service.NewReconDestAgentWithConfig(...)
reconCore := service.NewReconCoreWithConfig(
    sourceAgent, destAgent, db, mongoClientShared, schemaAdapter, registryRepo,
    redisCache, service.ReconCoreConfig{}, logger,
)
reconCore.SetMetadataRegistry(registrySvc)
logger.Info("Reconciliation Core initialized",
    zap.Bool("default_mongo_client", mongoClientShared != nil),
    zap.String("source_uri_resolution", "per-source via connection_registry (V2)"),
)
```

### Change H (~L467-525) — Restructure reconHandler wiring
- `if mongoClientShared != nil { /* init Healer / Backfill / TsDetector / FullCountAgg */ }`.
- Else: WARN log rõ các feature nào bị disabled.
- ReconHandler **luôn** đăng ký với reconCore + 5 NATS subjects.
- ReconHandler đã có nil-check ở L154 (heal), L457 (backfill), L536 (tsDetect)
  → handler tự return structured error khi service nil.
- Bỏ stub subscriber block 511-548 (đã thay thế bằng cùng handler với nil-safety).
- Log INFO mới: `recon_check_available`, `recon_heal_available`, etc.

### Change I (~L860-880) — Update runReconcileCycle defensive log
- Cũ: "reconCore not initialized (MongoDB not configured)" — Warn.
- Mới: Error log — nếu reconCore nil bây giờ thì là wiring regression
  (bug nghiêm trọng), không phải config missing.

## File 3 — `internal/service/recon_source_agent.go`

### Change J (~L189-203) — Hard-assert trong getClient
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

## Verify
- `go build ./...` → EXIT=0
- `go vet ./internal/...` → EXIT=0
- `go test ./internal/service/ -count=1 -run Recon` → PASS (0.771s)
- `go test ./internal/service/ -count=1` (full) → PASS (0.525s)
- `go test ./internal/handler/ -count=1` (full) → PASS (3.771s)

## Side-effect & Risk audit
- **R1 — rc.mongoClient dead reference**: grep confirmed L890 chỉ trong comment,
  không có active read. Truyền nil → safe.
- **R2 — ListTableConfigs() consumers**: thêm SourceURL non-empty không break
  consumer hiện tại (extra field, các path khác ignore).
- **R3 — GetSourceDSN per-connection overhead**: ReloadAll giờ thêm O(N)
  resolution. N nhỏ (10-50 connections typically). Mỗi resolve không query DB
  (đã refactor sang resolveSourceURIFromConn). OK.
- **R4 — Backward compat**: cfg.MongoDB.URL set + V2 chưa active → entries
  có SourceURL="" rơi vào defaultClient → behavior cũ giữ nguyên.
- **R5 — Local dev Mongo unreachable**: nếu V2 connection_registry URI trỏ
  docker hostname không reach → ReconCore CheckAll vẫn fail nhưng giờ với
  error rõ "connect to mongodb://docker-host:27017 timeout" thay vì
  "MongoDB not configured" — operator hành động đúng (set CONNECTION_OVERRIDE
  env var per L3126 lesson).

## Diff summary (số dòng net)
- `metadata_registry_service.go`: +35 / -3
- `metadata_registry_service_test.go`: +4 / -4 (signature only)
- `worker_server.go`: +35 / -55 (restructure)
- `recon_source_agent.go`: +6 / -1

**Tổng**: ~80 dòng net change. Minimal, focused, không over-engineer.
