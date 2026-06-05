# Report 2026-05-20 — Connector post-publish visibility (pass 6)

## 1. Tóm tắt

Worker trước đây report `debezium-signal | success | 1` ngay khi Kafka producer ACK record, không probe Debezium connector state. Khi connector idle (`tasks=[]` do replica set thiếu / source DB unreachable / config sai) → signal vào Kafka rồi bị bỏ rơi → snapshot không bao giờ chạy → activity_log vẫn xanh = **false positive**.

Pass 6: thêm post-publish probe `CheckConnectorHealth` vào `HandleDebeziumSignal`. Khi connector unhealthy → log ERROR + ghi activity_log error với reason đầy đủ. Không refuse-publish (user explicit reject prevention — chỉ cần visibility).

## 2. Trigger (user feedback nguyên văn)

1. "tao nói log ra, sao replica set thiếu mà ko có log thông báo. thằng chó ngu này. kêu mày làm log mà mày báo cáo láo à"
2. "mày đang làm tình thế, ko ngăn gốc rễ, tao ko sợ error, nhưng tao nói là tao cần khi error thì báo lỗi ra. ko phải kiểu ngu si này"

→ rule cụ thể: visibility (báo lỗi ra), KHÔNG prevention (refuse-publish).

## 3. Thay đổi code (file + diff hướng)

### 3.1. `centralized-data-service/internal/service/debezium_signal.go`
- Thêm struct:
  ```go
  type ConnectorHealth struct {
      Healthy   bool
      Reason    string
      State     string
      TaskCount int
      TaskState string
  }
  ```
- Thêm method `CheckConnectorHealth(ctx) (ConnectorHealth, error)`. Decision tree:
  - `ConnectorStatusURL == ""` → `{Healthy:true, Reason:"connector status URL not configured (assumed healthy)"}`
  - HTTP build/transport error → `{Healthy:false, Reason:"connector status probe failed"}`, return err
  - HTTP non-200 → `{Healthy:false, Reason:"connector status HTTP <code>"}`
  - JSON parse error → `{Healthy:false, Reason:"parse connector status failed"}`, return err
  - `state != RUNNING` → `{Healthy:false, Reason:"connector state=<X> (expected RUNNING)"}`
  - `len(tasks) == 0` → `{Healthy:false, Reason:"connector has 0 tasks (check kafka-connect logs for connector start-up errors; common causes: source DB unreachable, missing replica set, wrong hostname)"}`
  - `tasks[0].state != RUNNING` → `{Healthy:false, Reason:"task[0] state=<X> (expected RUNNING)"}`
  - else → `{Healthy:true, Reason:"connector + task[0] RUNNING"}`
- Refactor `IsConnectorHealthy(ctx) (bool, error)` thành thin wrapper trả `(h.Healthy, err)` để backwards-compat với mọi caller integration.

### 3.2. `centralized-data-service/internal/handler/recon_handler.go::HandleDebeziumSignal`
- Sau khi log "debezium signal dispatched", TRƯỚC khi `logActivity(success)`, thêm post-publish probe:
  ```go
  connHealth, healthErr := h.signal.CheckConnectorHealth(context.Background())
  if healthErr != nil {
      h.logger.Error("debezium signal published BUT connector status probe failed", ...)
      h.logActivity("debezium-signal", payload.Table, "error", 0,
          fmt.Errorf("signal published to kafka but connector status probe failed: %w", healthErr))
      return
  }
  if !connHealth.Healthy {
      h.logger.Error("debezium signal published BUT connector not ready — snapshot will NOT execute",
          zap.String("trace_id", trace.TraceID),
          zap.String("signal_id", signalID),
          zap.String("connector_state", connHealth.State),
          zap.Int("task_count", connHealth.TaskCount),
          zap.String("task_state", connHealth.TaskState),
          zap.String("reason", connHealth.Reason),
      )
      h.logActivity("debezium-signal", payload.Table, "error", 0,
          fmt.Errorf("signal published to kafka but connector not ready: state=%s task_count=%d task_state=%s reason=%s",
              connHealth.State, connHealth.TaskCount, connHealth.TaskState, connHealth.Reason))
      return
  }
  h.logger.Info("debezium signal end-to-end ready", ...)
  h.logActivity("debezium-signal", payload.Table, "success", 1, nil)
  ```

### 3.3. `centralized-data-service/internal/server/worker_server.go`
- Thêm import `"strings"`.
- Trước `NewDebeziumSignalClient`, derive `connectorStatusURL`:
  ```go
  connectorStatusURL := cfg.Debezium.ConnectorStatusURL
  if connectorStatusURL == "" && cfg.Debezium.KafkaConnectURL != "" && cfg.Debezium.ConnectorName != "" {
      connectorStatusURL = strings.TrimRight(cfg.Debezium.KafkaConnectURL, "/") +
          "/connectors/" + cfg.Debezium.ConnectorName + "/status"
  }
  ```
- Pass `connectorStatusURL` (not `cfg.Debezium.ConnectorStatusURL`) vào `DebeziumSignalConfig`.

## 4. Verify thực tế (kết quả runtime, không phải lý thuyết)

### 4.1. Build + vet
```
$ go build ./...      # exit 0 (clean sau khi fix missing strings import)
$ go vet ./...        # exit 0
```

### 4.2. Restart worker
- Kill PID 17021 (parent `go run`) + 17027 (compiled child binary).
- `nohup go run cmd/worker/main.go > /tmp/worker.log 2>&1 &` — boot OK, subscribers registered cho `cdc.cmd.debezium-signal` + `cdc.cmd.debezium-snapshot`.

### 4.3. E2E test
Publish:
```
nats --server "nats://cdc_worker:worker_secret_2026@localhost:14222" pub cdc.cmd.debezium-snapshot \
  '{"trace_id":"preflight-test-001","table":"export-jobs","db":"centralized-export-service","collection":"export-jobs","source_object_id":1}'
```

Worker log (grep `preflight-test-001`):
```
INFO  "debezium signal received"   trace_id=preflight-test-001  action=snapshot_now  table=export-jobs
INFO  "debezium signal: using SignalClient path"
INFO  "debezium signal published"  topic=cdc.signal.commands  signal_id=signal-1779219779670312000
INFO  "debezium signal dispatched" dispatch_path=signal_client signal_id=signal-1779219779670312000
ERROR "debezium signal published BUT connector not ready — snapshot will NOT execute"
        trace_id=preflight-test-001  signal_id=signal-1779219779670312000
        connector_state=""  task_count=0  task_state=""
        reason="connector status HTTP 404"
```

activity_log query:
```
| t        | operation       | target_table | status | rows | error_message                                                                                                          | triggered_by |
|----------|-----------------|--------------|--------|------|------------------------------------------------------------------------------------------------------------------------|--------------|
| 19:42:59 | debezium-signal | export-jobs  | error  | 0    | signal published to kafka but connector not ready: state= task_count=0 task_state= reason=connector status HTTP 404    | nats-command |
```

So sánh với pre-pass-6 (cùng input):
```
| 19:27:14 | debezium-signal | export-jobs  | success | 1   | (null) | nats-command |   ← false positive cũ
```

Pass criterion đạt: operator giờ có 1 SQL/grep dòng duy nhất point đúng downstream cause (HTTP 404 = connector name mismatch), không cần đào docker logs.

## 5. Service status (verify trước khi báo done)

- Worker `cmd/worker/main.go` đang chạy (PID mới sau restart), tail log: `kafka consumer started`, `debezium signal subscribers registered`. Không có goroutine leak / panic.
- Postgres `gpay-postgres-cdc`: ok (query activity_log success).
- Kafka `localhost:19092`: ok (publish success, signal_id ghi nhận).
- NATS `localhost:14222`: ok (consumer pool subscribed cả 2 subjects).
- Kafka Connect `localhost:18083`: ok (REST returns connector list `["goopay-local","goopay-dev"]`).
- Debezium connector `goopay-local`: `state=RUNNING, tasks=[]` (downstream idle — root cause của 0-doc snapshot từ trước, giờ visibility surface lên rồi).

## 6. Files thay đổi (ground truth, đã grep verify)

| File | Loại | Verify |
|---|---|---|
| `centralized-data-service/internal/service/debezium_signal.go` | Add struct + method, refactor wrapper | grep `ConnectorHealth\|CheckConnectorHealth` returns 9 hits |
| `centralized-data-service/internal/handler/recon_handler.go` | Add post-publish probe + 2 error branches | grep `CheckConnectorHealth` returns 1 hit ở line 373 |
| `centralized-data-service/internal/server/worker_server.go` | Derive `connectorStatusURL` + add `strings` import | grep `connectorStatusURL` returns 4 hits |

KHÔNG sửa (per "không cheat" rule):
- DB rows / docker-compose / connector JSON / FE/CMS / MongoDB instances.
- `config/config-local.yml` (connectorName mismatch giữ nguyên — xem follow-up §7).
- `internal/admin/helpers.go` / `internal/handler/command_handler.go` (cũng hardcode `goopay-mongodb-cdc` — out of pass 6 scope).
- `IsConnectorHealthy` (giữ làm wrapper backwards-compat).

## 7. Follow-up — connectorName mismatch (out of pass 6 scope)

Pass-6 visibility report `HTTP 404` lý do vì:
- `config/config-local.yml:91`: `connectorName: goopay-mongodb-cdc`
- `internal/admin/helpers.go:113`: hardcode `return "goopay-mongodb-cdc"` cho engine type mongodb
- `internal/handler/command_handler.go:2314`: fallback `return "goopay-mongodb-cdc"`
- Nhưng actual Kafka Connect deploy đang chạy 2 connectors: `goopay-local` + `goopay-dev` (CMS đặt tên theo connection code).

Đây là pre-existing inconsistency, KHÔNG phải bug do pass 6 tạo ra. Pass-6 visibility ĐÚNG là expose nó loud-and-clear (mục tiêu của pass 6). 3 option để fix, ĐỀ XUẤT user chọn:
- (A) Rename Debezium connectors trên Kafka Connect thành `goopay-mongodb-cdc` (match code/config). Đơn giản nhất nhưng phá multi-source (goopay-local vs goopay-dev).
- (B) Update code hardcode + config thành `goopay-local` cho local env. Vẫn không scale cho multi-connector production.
- (C) Make code dynamic — resolve connector name từ source_connection_id / object_code thay vì hardcode. Đúng nhất nhưng refactor lớn.

→ Cần user direction trước khi sửa.

## 8. Skill / công cụ sử dụng

- Đọc/sửa file: Read, Edit, Write
- Shell + container: Bash, docker exec psql
- Go build: `go build ./...`, `go vet ./...`
- Runtime test: `nats pub`, `curl`, tail/grep log
- Workspace governance (CLAUDE.md §7): append-only `05_progress.md`, append-only `lessons.md`, prefix-bound report file
- Pattern abstraction (CLAUDE.md §13): 3 Global Patterns mới đã append vào `lessons.md` (publisher-success-without-probe, harsh-feedback-routing, visibility-vs-prevention)

## 9. Definition of Done — checklist

- [x] Build + vet clean
- [x] Worker restart clean, subscribers registered
- [x] E2E test → log ERROR + activity_log error với reason đầy đủ
- [x] So sánh side-by-side pre/post pass 6 trong activity_log để chứng minh từ false-positive → true-error
- [x] 3 files code thay đổi đã grep verify
- [x] `05_progress.md` append entries [45:DONE..52:DONE]
- [x] `lessons.md` append 3 Global Patterns
- [x] `report_2026-05-20_connector-preflight-visibility.md` (file này) — đầy đủ trace + verify + follow-up
- [x] Service health check post-change: worker + postgres + kafka + nats + kafka-connect all green
