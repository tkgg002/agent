# 02_plan_connector_preflight_visibility — Audit pass 6: Connector pre-flight + visibility

## 1. Vấn đề (user feedback)

> "tao nói log ra, sao replica set thiếu mà ko có log thông báo. thằng chó ngu này. kêu mày làm log mà mày báo cáo láo à"

Worker hiện báo `debezium-signal | success | 1` chỉ vì publish vào Kafka thành công, nhưng **không kiểm Debezium connector có sẵn sàng consume hay không**. Khi connector idle (tasks=[]) — như case hiện tại do Mongo trỏ sai vào standalone instance không phải replica set — signal vào Kafka rồi bị bỏ rơi, không ai consume, snapshot không bao giờ chạy. activity_log "success" → false positive → user gọi là báo cáo láo.

## 2. Root cause logging gap

| Layer | Behavior hiện tại | Vấn đề |
|---|---|---|
| Worker publish | Publish OK → log `debezium signal dispatched` + activity_log success=1 | Không biết connector consume hay không |
| `IsConnectorHealthy(ctx)` đã tồn tại | Chỉ check `state=RUNNING`, không check task count, không return reason chi tiết | Connector "RUNNING + tasks=[]" vẫn pass health check |
| `HandleDebeziumSignal` | Không gọi `IsConnectorHealthy` trước publish | Pre-flight check bị bỏ qua |
| `ConnectorStatusURL` config | Để trống trong `config-local.yml` (chỉ có `kafkaConnectUrl` + `connectorName`) | Default optimistic (assume healthy) — không probe |
| Kafka Connect side | Log "Found no replica sets at mongodb://..." trong `docker logs gpay-kafka-connect` | Không có channel đẩy về CDC `activity_log` → operator phải đi đọc docker logs |

## 3. Giải pháp — pre-flight visibility (no cheat, code-level only)

### 3.1. Đổi `IsConnectorHealthy(ctx) (bool, error)` thành `CheckConnectorHealth(ctx) (ConnectorHealth, error)`

```go
// ConnectorHealth describes the downstream Debezium connector state
// for the purpose of signal delivery pre-flight checks.
type ConnectorHealth struct {
    Healthy   bool   // true only when connector + at least one task are RUNNING
    Reason    string // human-readable diagnostic (always populated)
    State     string // connector state (RUNNING/PAUSED/FAILED/UNASSIGNED/...)
    TaskCount int    // total task count
    TaskState string // state of task[0] when TaskCount > 0
}

func (d *DebeziumSignalClient) CheckConnectorHealth(ctx context.Context) (ConnectorHealth, error) {
    if d.cfg.ConnectorStatusURL == "" {
        return ConnectorHealth{
            Healthy: true,
            Reason:  "connector status URL not configured (assumed healthy)",
        }, nil
    }
    // GET /connectors/<name>/status → parse {connector:{state}, tasks:[{state}]}
    // Decision tree:
    //   connector.state != RUNNING                          → Healthy=false, Reason="connector state=<X> (expected RUNNING)"
    //   len(tasks) == 0                                     → Healthy=false, Reason="connector has 0 tasks (check kafka-connect logs; common cause: source DB unreachable, missing replica set, wrong hostname)"
    //   tasks[0].state != RUNNING                           → Healthy=false, Reason="task[0] state=<X> (expected RUNNING)"
    //   otherwise                                           → Healthy=true,  Reason="connector + task[0] RUNNING"
}
```

Backwards compat: giữ method cũ `IsConnectorHealthy` (thin wrapper trả `(h.Healthy, err)`) — không break test integration nào đang dùng.

### 3.2. `HandleDebeziumSignal` pre-flight

Sau khi resolve `db+collection` nhưng TRƯỚC khi `TriggerIncrementalSnapshot`:

```go
health, herr := h.signal.CheckConnectorHealth(context.Background())
if herr != nil {
    h.logger.Error("debezium signal pre-flight failed: connector status probe unreachable",
        zap.String("trace_id", trace.TraceID),
        zap.String("table", payload.Table),
        zap.String("connector_status_url", "<configured>"),
        zap.Error(herr),
    )
    h.logActivity("debezium-signal", payload.Table, "error", 0,
        fmt.Errorf("connector status probe failed: %w", herr))
    return
}
if !health.Healthy {
    h.logger.Error("debezium signal pre-flight failed: connector not ready",
        zap.String("trace_id", trace.TraceID),
        zap.String("table", payload.Table),
        zap.String("connector_state", health.State),
        zap.Int("task_count", health.TaskCount),
        zap.String("task_state", health.TaskState),
        zap.String("reason", health.Reason),
    )
    h.logActivity("debezium-signal", payload.Table, "error", 0,
        fmt.Errorf("debezium connector not ready: %s", health.Reason))
    return
}
h.logger.Info("debezium signal pre-flight OK",
    zap.String("trace_id", trace.TraceID),
    zap.String("connector_state", health.State),
    zap.Int("task_count", health.TaskCount),
)
```

### 3.3. Wire connector status URL từ `kafkaConnectUrl + connectorName`

`worker_server.go`: trước khi gọi `NewDebeziumSignalClient`, build `ConnectorStatusURL` nếu user chỉ set `kafkaConnectUrl + connectorName`:

```go
connectorStatusURL := cfg.Debezium.ConnectorStatusURL
if connectorStatusURL == "" && cfg.Debezium.KafkaConnectURL != "" && cfg.Debezium.ConnectorName != "" {
    connectorStatusURL = strings.TrimRight(cfg.Debezium.KafkaConnectURL, "/") +
        "/connectors/" + cfg.Debezium.ConnectorName + "/status"
}
```

Pass `connectorStatusURL` (not `cfg.Debezium.ConnectorStatusURL`) vào `DebeziumSignalConfig`.

Optimistic-by-default behavior giữ nguyên cho mọi env chưa set `kafkaConnectUrl + connectorName`. Hiện tại config-local.yml đã set cả 2 → pre-flight tự enable.

## 4. Verify steps

1. `go build ./... && go vet ./...` clean.
2. Restart worker → log "debezium signal subscribers registered" (vẫn OK).
3. Publish test signal → expected:
   - Worker log ERROR "debezium signal pre-flight failed: connector not ready, connector_state=RUNNING, task_count=0, reason=connector has 0 tasks (...)"
   - activity_log row `debezium-signal | error | 0 | debezium connector not ready: connector has 0 tasks ...`
4. activity_log UI / Postgres query: error message rõ ràng tại sao snapshot không chạy.
5. Operator giờ có thể grep `kafka-connect-logs` cho root cause cụ thể (missing replica set).

## 5. Files thay đổi

| File | Loại sửa |
|---|---|
| `centralized-data-service/internal/service/debezium_signal.go` | Add `ConnectorHealth` struct + `CheckConnectorHealth` method; refactor `IsConnectorHealthy` thành thin wrapper |
| `centralized-data-service/internal/handler/recon_handler.go` | Pre-flight call `CheckConnectorHealth` trước `TriggerIncrementalSnapshot` |
| `centralized-data-service/internal/server/worker_server.go` | Derive `ConnectorStatusURL` từ `kafkaConnectUrl + connectorName` nếu trống |

KHÔNG sửa:
- DB rows, docker-compose, connector JSON, FE/CMS, MongoDB instances.
- `IsConnectorHealthy` (giữ làm wrapper backwards-compat).

## 6. Risks

- **Probe latency**: thêm 1 HTTP GET ~5ms vào mỗi signal publish. Acceptable: signal là user-initiated action không phải hot loop.
- **Connect REST timeout**: client đã có `Timeout: 5 * time.Second` → fail-loud → activity_log error rõ ràng.
- **Connector name single-config**: dev có 1 connector cho Mongo (`goopay-mongodb-cdc` / `goopay-local`). Production multi-connector → cần discriminator theo source. Out of scope pass 6 — flag tại `00_followups.md` nếu cần.

## 7. Bài học rút ra (sẽ append lessons.md)

**Global Pattern [Publisher P báo `success` ngay sau khi commit operation Op vào transport T, mà không probe consumer C đang/sẽ consume]** → false positive: T accept → P báo success → C silent drop → end-to-end fail nhưng activity_log/metrics đều green. User mất tín nhiệm.

**Đúng**: với MỌI fire-and-forget publish vào transport có downstream consumer phụ thuộc state (Debezium connector, Kafka consumer group, NATS subscriber, RabbitMQ binding), publisher MUST pre-flight probe consumer health TRƯỚC publish. Probe return cấu trúc có Reason rõ ràng (không chỉ bool). Khi unhealthy → log ERROR + activity_log error với reason → operator có 1 dòng grep-able thay vì phải đào docker logs.
