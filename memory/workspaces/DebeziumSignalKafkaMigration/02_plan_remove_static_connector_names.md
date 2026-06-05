# 02_plan_remove_static_connector_names

## 1. Design decision

### 1.1. Worker resolve flow

Payload `cdc.cmd.debezium-signal` chỉ có `(database, collection, table)`. Resolver SQL:
```sql
SELECT cr.connection_code
FROM cdc_system.source_object_registry so
JOIN cdc_system.connection_registry cr ON cr.id = so.source_connection_id
WHERE (so.source_namespace = $1 OR so.source_database = $1)
  AND so.source_object_name = $2
LIMIT 1
```
- Mongo: `database` lưu trong `source_namespace` ("centralized-export-service") — match `source_namespace`.
- PG/MySQL: `database` lưu trong `source_database` — match `source_database`.
- OR clause handle cả 2 trường hợp trong cùng 1 query.

Resolver return empty string → `CheckConnectorHealth` optimistic skip (KHÔNG block). Visibility design.

### 1.2. CMS-service multi-connector

Approach: **auto-discover via `/connectors` REST** (NOT static list from config).
- `probes.Debezium` đổi từ `(ctx, deps, baseURL, name)` → `(ctx, deps, baseURL)`. Hàm GET `/connectors` để lấy list, sau đó GET `/status` từng cái parallel.
- Return aggregate: `{connectors:[{name,state,tasks_total,tasks_running}], total, failed, healthy}`.
- Snapshot field `debezium` đổi từ object → object có `connectors []`. Schema FE phải đọc lại.
- Alert per-connector: failed → 1 alert/connector (label `connector`).
- `RestartDebezium` endpoint: BẮT BUỘC nhận `connector_name` qua payload (FE chọn). Bỏ default.

### 1.3. Provisioning flow (admin/helpers.go)

`RegisterSourceRequest` đã có `SourceConnectionID int64`. Đổi `connectorNameFor` thành method:
```go
func (s *Server) connectorNameFor(ctx context.Context, req RegisterSourceRequest) (string, error) {
    var connectionCode string
    err := s.deps.DB.WithContext(ctx).Raw(
        `SELECT connection_code FROM cdc_system.connection_registry WHERE id = ?`,
        req.SourceConnectionID,
    ).Scan(&connectionCode).Error
    if err != nil { return "", fmt.Errorf("lookup connection_code: %w", err) }
    if connectionCode == "" {
        return "", fmt.Errorf("no connection found for source_connection_id=%d", req.SourceConnectionID)
    }
    return connectionCode, nil
}
```

### 1.4. Command handler (sync-state, restart-debezium)

`detectConnectorName(entry *TableRegistry)` không thể lookup direct vì `TableRegistry` không có `source_connection_id`. Strategy:
- `HandleSyncState`: lookup qua `(entry.SourceDB, entry.SourceTable)` y hệt worker resolver (helper chung).
- `HandleRestartDebezium`: REQUIRE `payload.connector_name` từ caller. Bỏ fallback hardcode.

### 1.5. Config keys removed

| File | Key | Action |
|---|---|---|
| `centralized-data-service/config/config-local.yml:91` | `connectorName` | DELETE |
| `centralized-data-service/config/config-production.yml:98` | `connectorName: ""` | DELETE |
| `centralized-data-service/config/config.go:54` | `ConnectorName` field | DELETE |
| `cdc-cms-service/config/config-sample.yml:34` | `debeziumConnector` | DELETE |
| `cdc-cms-service/config/config-local.yml:45` | `debeziumConnector` | DELETE |
| `cdc-cms-service/config/config-production.yml:34` | `debeziumConnector` | DELETE |
| `cdc-cms-service/config/config.go:47` | `DebeziumConnector` field | DELETE |
| `cdc-cms-service/config/config.go:155` | `system.debeziumConnector` env binding | DELETE |

## 2. Code demo (chi tiết tới từng method)

### 2.1. `cdc-worker/internal/service/debezium_signal.go`

```go
type DebeziumSignalConfig struct {
    SignalKafkaTopic     string
    KafkaBrokers         []string
    KafkaConnectBaseURL  string // CHANGED from ConnectorStatusURL — chỉ là base http://host:port
    IncrementalChunkSize int
}

// CheckConnectorHealth — accept connector name động per-call.
// Empty connectorName OR empty KafkaConnectBaseURL → optimistic skip.
func (d *DebeziumSignalClient) CheckConnectorHealth(
    ctx context.Context,
    connectorName string,
) (ConnectorHealth, error) {
    if d.cfg.KafkaConnectBaseURL == "" {
        return ConnectorHealth{Healthy: true, Reason: "kafka connect base URL not configured (probe skipped)"}, nil
    }
    if connectorName == "" {
        return ConnectorHealth{Healthy: true, Reason: "connector name unresolved (probe skipped)"}, nil
    }
    url := strings.TrimRight(d.cfg.KafkaConnectBaseURL, "/") + "/connectors/" + connectorName + "/status"
    // ... rest unchanged (HTTP GET + parse + decision tree)
}

// IsConnectorHealthy REMOVED — không còn caller (worker dùng CheckConnectorHealth trực tiếp).
```

### 2.2. `cdc-worker/internal/handler/recon_handler.go`

```go
// resolveConnectorName — DB lookup (source_object_registry JOIN connection_registry)
// để map (database, collection) → connection_code = Kafka Connect connector name.
// Return empty string khi không resolve (caller treat as optimistic skip).
func (h *ReconHandler) resolveConnectorName(ctx context.Context, database, collection string) string {
    if h.db == nil || database == "" || collection == "" {
        return ""
    }
    var connectionCode string
    err := h.db.WithContext(ctx).Raw(`
        SELECT cr.connection_code
        FROM cdc_system.source_object_registry so
        JOIN cdc_system.connection_registry cr ON cr.id = so.source_connection_id
        WHERE (so.source_namespace = ? OR so.source_database = ?)
          AND so.source_object_name = ?
        LIMIT 1
    `, database, database, collection).Scan(&connectionCode).Error
    if err != nil {
        h.logger.Warn("resolve connector name failed",
            zap.String("database", database), zap.String("collection", collection), zap.Error(err))
        return ""
    }
    return connectionCode
}

// Trong HandleDebeziumSignal, sau "debezium signal dispatched":
connectorName := h.resolveConnectorName(context.Background(), db, collection)
connHealth, healthErr := h.signal.CheckConnectorHealth(context.Background(), connectorName)
// ... rest unchanged + LOG `connector_name` field cho operator trace
```

### 2.3. `cdc-worker/internal/server/worker_server.go`

```go
// XÓA block derive ConnectorStatusURL.
signalClient = service.NewDebeziumSignalClient(
    service.DebeziumSignalConfig{
        SignalKafkaTopic:     cfg.Debezium.SignalKafkaTopic,
        KafkaBrokers:         cfg.Kafka.Brokers,
        KafkaConnectBaseURL:  cfg.Debezium.KafkaConnectURL,  // CHANGED
        IncrementalChunkSize: cfg.Debezium.IncrementalChunkSize,
    },
    logger,
)
// XÓA import "strings" nếu không còn dùng.
```

### 2.4. `cdc-worker/internal/admin/helpers.go`

```go
// XÓA func connectorNameFor (engine-based hardcode).
// XÓA comment block lines 102-109.

// Caller `extendDebeziumInclude` đổi:
func (s *Server) extendDebeziumInclude(ctx context.Context, req RegisterSourceRequest) (*ExtendResult, error) {
    connector, err := s.connectorNameFor(ctx, req)  // method mới, DB lookup
    if err != nil { return nil, err }
    // ... rest unchanged
}

// New method:
func (s *Server) connectorNameFor(ctx context.Context, req RegisterSourceRequest) (string, error) {
    if req.SourceConnectionID == 0 {
        return "", fmt.Errorf("source_connection_id missing in request")
    }
    var connectionCode string
    err := s.deps.DB.WithContext(ctx).Raw(
        `SELECT connection_code FROM cdc_system.connection_registry WHERE id = ? AND status = 'active'`,
        req.SourceConnectionID,
    ).Scan(&connectionCode).Error
    if err != nil { return "", fmt.Errorf("lookup connection_code by id=%d: %w", req.SourceConnectionID, err) }
    if connectionCode == "" {
        return "", fmt.Errorf("no active connection_registry row for source_connection_id=%d", req.SourceConnectionID)
    }
    return connectionCode, nil
}
```

### 2.5. `cdc-worker/internal/handler/command_handler.go`

```go
// XÓA func detectConnectorName.

// HandleSyncState (line 2052):
connector := h.resolveConnectorNameFromEntry(ctx, entry)
if connector == "" {
    debeziumStatus = "error"
    firstErr = "cannot resolve connector name from registry (source DB/object missing in source_object_registry)"
    // ... unchanged
}

// New helper:
func (h *CommandHandler) resolveConnectorNameFromEntry(ctx context.Context, entry *model.TableRegistry) string {
    if entry == nil || h.db == nil { return "" }
    var connectionCode string
    h.db.WithContext(ctx).Raw(`
        SELECT cr.connection_code
        FROM cdc_system.source_object_registry so
        JOIN cdc_system.connection_registry cr ON cr.id = so.source_connection_id
        WHERE (so.source_namespace = ? OR so.source_database = ?) AND so.source_object_name = ?
        LIMIT 1
    `, entry.SourceDB, entry.SourceDB, entry.SourceTable).Scan(&connectionCode)
    return connectionCode
}

// HandleRestartDebezium (line 2089):
connector := strings.TrimSpace(payload.ConnectorName)
if connector == "" {
    // REJECT — không có fallback hardcode.
    h.publishResultWithSubject(msg, "cdc.result.restart-debezium",
        CommandResult{Command: "restart-debezium", Status: "error",
            Error: "connector_name required in payload (no implicit default)"})
    return
}
```

### 2.6. Config files (cdc-worker)

```yaml
# config-local.yml, config-production.yml: XÓA debezium.connectorName key
debezium:
  kafkaConnectUrl: http://127.0.0.1:18083
  signalKafkaTopic: cdc.signal.commands
  incrementalChunkSize: 1000
```

```go
// config.go: XÓA field ConnectorName + env binding.
type DebeziumConfig struct {
    SignalKafkaTopic     string `mapstructure:"signalKafkaTopic"`
    KafkaConnectURL      string `mapstructure:"kafkaConnectUrl"`
    // ConnectorName REMOVED
    ConnectorStatusURL   string `mapstructure:"connectorStatusUrl"` // KEEP — explicit override
    IncrementalChunkSize int    `mapstructure:"incrementalChunkSize"`
}
```

Actually `ConnectorStatusURL` cũng nên xoá luôn vì giờ build từ base + name động. **Xoá cả 2.**

### 2.7. `cdc-cms-service/internal/infra/observability/probes/debezium.go`

(Cần đọc nội dung trước khi sửa — dự kiến).

```go
// Đổi signature:
func Debezium(ctx context.Context, deps HTTPDeps, baseURL string) map[string]any {
    if baseURL == "" {
        return map[string]any{"status": "unknown", "reason": "kafka_connect_url empty"}
    }
    // GET /connectors → []string names
    listURL := strings.TrimRight(baseURL, "/") + "/connectors"
    // ... HTTP GET, parse, iterate parallel với errgroup, return aggregate
    return map[string]any{
        "status": "ok"|"degraded"|"down",
        "total_count": N,
        "running_count": M,
        "failed_count": K,
        "connectors": []map[string]any{ {name, state, tasks_running, tasks_total}, ... },
    }
}
```

### 2.8. `cdc-cms-service/internal/infra/observability/system_health_collector.go`

```go
type CollectorConfig struct {
    // ... unchanged fields ...
    // DebeziumName REMOVED
    // ... unchanged ...
}

// Call site đổi:
g.Go(func() error {
    set("pipeline", "debezium", probes.Debezium(gCtx, hd, c.cfg.KafkaConnectURL))
    return nil
})

// NewCollector: bỏ default `cfg.DebeziumName = "goopay-mongodb-cdc"`.
```

### 2.9. `cdc-cms-service/internal/infra/observability/system_health_alerts.go`

```go
// Đổi logic fire alert: thay vì 1 alert single connector, iterate list:
if dbz, ok := snap.CDCPipeline["debezium"].(map[string]any); ok {
    if connectors, ok := dbz["connectors"].([]any); ok {
        for _, raw := range connectors {
            c, _ := raw.(map[string]any)
            state, _ := c["state"].(string)
            name, _ := c["name"].(string)
            if !strings.EqualFold(state, "RUNNING") {
                out = append(out, detectedCondition{req: persistence.FireRequest{
                    Name: "DebeziumConnectorFailed", Severity: "critical",
                    Labels: map[string]string{"component":"debezium", "connector": name},
                    Description: fmt.Sprintf("Debezium connector %s state=%s", name, state),
                }})
            }
        }
    }
}
```

### 2.10. `cdc-cms-service/internal/api/system_health_handler.go`

```go
// RestartDebezium: REQUIRE connector_name từ payload (FE pass).
func (h *SystemHealthHandler) RestartDebezium(c *fiber.Ctx) error {
    var body struct{ ConnectorName string `json:"connector_name"` }
    if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.ConnectorName) == "" {
        return c.Status(400).JSON(fiber.Map{"error": "connector_name required in request body"})
    }
    // ... dispatch unchanged, dùng body.ConnectorName
}

// Constructor bỏ debeziumName param luôn.
```

### 2.11. Config + test cleanup (cms-service)

- `config.go`: xóa `DebeziumConnector` field + env binding.
- `config-{sample,local,production}.yml`: xóa key `debeziumConnector`.
- 3 test files: thay hardcode `"goopay-mongodb-cdc"` bằng `"test-connector"` hoặc xóa nếu test scope đã đổi.

## 3. Files thay đổi (final list)

### cdc-worker (8 files)
1. `internal/service/debezium_signal.go` — config field + CheckConnectorHealth sig.
2. `internal/handler/recon_handler.go` — add resolveConnectorName, pass into probe.
3. `internal/server/worker_server.go` — xóa derive logic + strings import.
4. `internal/admin/helpers.go` — xóa connectorNameFor, đổi extendDebeziumInclude.
5. `internal/handler/command_handler.go` — xóa detectConnectorName, đổi HandleSyncState + HandleRestartDebezium.
6. `config/config.go` — xóa ConnectorName + ConnectorStatusURL field + env bindings.
7. `config/config-local.yml` — xóa connectorName key.
8. `config/config-production.yml` — xóa connectorName key.

### cdc-cms-service (8 files)
1. `internal/infra/observability/probes/debezium.go` — đổi signature, auto-discover.
2. `internal/infra/observability/system_health_collector.go` — xóa DebeziumName.
3. `internal/infra/observability/system_health_alerts.go` — per-connector loop.
4. `internal/api/system_health_handler.go` — RestartDebezium body parse.
5. `internal/server/server.go` — bỏ pass DebeziumName/DebeziumConnector.
6. `config/config.go` — xóa DebeziumConnector field.
7. `config/config-{sample,local,production}.yml` — xóa key.
8. Tests: `system_health_alerts_test.go`, `probes/debezium_test.go`, `alert_manager_test.go`, `system_health_collector_test.go` — parameterize/rename.

## 4. Verify plan

1. `go build ./... && go vet ./...` ở cả 2 repos clean.
2. `go test ./...` ở 2 repos pass.
3. Restart worker (kill + nohup go run).
4. Boot log clean, no error/warning bất ngờ.
5. E2E test: `nats pub cdc.cmd.debezium-snapshot '{"trace_id":"final-removed-hardcode-001","database":"centralized-export-service","collection":"export-jobs"}'`.
6. Expected worker log: `connector_name="goopay-local"` (resolved từ DB!) + ERROR `reason="connector has 0 tasks (...)"` (NOT 404).
7. Expected activity_log row: `error_message` chứa `state=RUNNING task_count=0 ...`.
8. Restart cms-service (nếu đang chạy). Verify `/api/v1/system-health` JSON response có `debezium.connectors: [{name:"goopay-local"...},{name:"goopay-dev"...}]`.

## 5. Risks + mitigation

- **Risk**: `source_object_registry` chưa có row tương ứng → resolver trả empty → probe skip. **Mitigation**: log INFO "connector name unresolved (probe skipped)" để operator biết cần đăng ký source. NOT block publish.
- **Risk**: `connection_registry.status != 'active'` filter trong admin/helpers.go có thể loại bỏ connection user vừa tạo nhưng pending. **Mitigation**: chỉ filter `active` cho **admin provisioning** (RegisterSource). Worker probe path KHÔNG filter status (xem connector đang trong real state thế nào).
- **Risk**: CMS-service FE expect `snap.CDCPipeline.debezium` là object có field `state` (single) → schema break. **Mitigation**: thêm `state` rollup ở object level (`"ok"|"degraded"|"down"`) để FE không vỡ + thêm `connectors[]` array cho FE đọc chi tiết.
- **Risk**: tests dùng hardcode tên cũ → mock URL không match. **Mitigation**: đổi mock URL + test data parallel.

## 6. Out of scope

- 4 deployments JSON files (sample artifact).
- DB schema migration (không add field nào).
- NATS payload schema change.
- cdc-cms-web (0 hit).
