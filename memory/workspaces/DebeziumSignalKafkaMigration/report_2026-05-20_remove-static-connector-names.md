# Report — Remove Hardcoded Debezium Connector Names

**Date**: 2026-05-20
**Phase**: `remove-static-connector-names`
**Workspace**: `agent/memory/workspaces/DebeziumSignalKafkaMigration/`
**Operator**: Muscle (CC CLI)

---

## 1. Vấn đề

User báo: trong codebase còn rải rác tên Debezium connector hardcoded:
- `"goopay-mongodb-cdc"` (Mongo)
- `"cdc-pg-source"` (PG)
- `"cdc-mariadb-source"` (MySQL/MariaDB)

Trong khi đó Kafka Connect đăng ký connector ĐỘNG theo `connection_registry.connection_code` (vd `goopay-local`, `goopay-dev`). Hệ quả:

- Probe `/connectors/goopay-mongodb-cdc/status` từ Worker → HTTP 404.
- `activity_log` ghi `error_message=connector status HTTP 404` → debug mislead.
- CMS `/api/v1/system/health` chỉ probe được 1 connector cố định, bỏ sót các connector động khác.

## 2. Audit ground-truth

Quét 4 repo (`cdc-cms-web`, `cdc-cms-service`, `centralized-data-service`, `cdc-control`):

- **cdc-cms-web**: 0 hit.
- **cdc-control**: 0 hit.
- **centralized-data-service**: 9 vị trí (2 code helper + 2 yml + 2 config struct fields + 2 env binding + 1 handler default + JSON deployment artifact — out-of-scope).
- **cdc-cms-service**: 7 vị trí (4 code + 3 yml + 4 test files).

Toàn bộ 3 hit `"goopay-mongodb-cdc"` trong worker đều thuộc commit init `94aa71c3 TraiNguyen 2026-05-13` (xác nhận bằng `git blame`) — pre-existing, không phải do CC introduce.

## 3. Giải pháp (Plan ngắn)

**Nguyên tắc**: Mọi reference tới tên connector phải resolve QUA `connection_registry.connection_code` ở runtime. Không fallback hardcode.

### 3.1 cdc-worker (centralized-data-service)

| Layer | Trước | Sau |
|---|---|---|
| Service helper | (không có) | `service/connector_resolver.go` (NEW): `ResolveConnectorNameBySource(db, collection)` + `ResolveConnectorNameByConnectionID(id)` |
| Signal probe | `cfg.ConnectorStatusURL` (URL có name nhúng cứng) | `cfg.KafkaConnectBaseURL` + connectorName arg tại call site |
| `recon_handler.go` | probe không có connector context | resolve trước, log `connector_name`, embed vào error |
| `recon_heal.go` | `IsConnectorHealthy(ctx)` | `IsConnectorHealthy(ctx, connectorName)` resolved per-entry |
| `command_handler.go::detectConnectorName` | `return "goopay-mongodb-cdc"` | `ResolveConnectorNameBySource(ctx, h.db, entry.SourceDB, entry.SourceTable)`; trả "" → caller error |
| `command_handler.go::HandleSyncState` | implicit hardcode | check `connector==""` → error status, không fallback |
| `command_handler.go::HandleRestartDebezium` | fallback `detectConnectorName(nil)` | REQUIRE `payload.connector_name`, HTTP 400 nếu thiếu |
| `admin/helpers.go::connectorNameFor` | switch engine→hardcoded name | `(s *Server) resolveConnectorByEngine(ctx, engineType)` query `connection_registry` |
| `config/config.go` | `ConnectorStatusURL`+`ConnectorName` fields + env binding | Xóa, chỉ giữ `KafkaConnectURL` |
| `config/*.yml` | `connectorName:`/`connectorStatusUrl:` keys | Xóa |
| `worker_server.go` | tự ý nối URL `<base>/connectors/<name>/status` | Truyền `KafkaConnectBaseURL` thô; nối tại call site theo connectorName resolved |

### 3.2 cms-service (cdc-cms-service)

| Layer | Trước | Sau |
|---|---|---|
| `probes/debezium.go` | chỉ có `Debezium(... debeziumName)` | THÊM `DebeziumAll(...)` enumerate `/connectors` → mỗi name → reuse `Debezium()` → `{status, connectors:[...], count}` |
| `system_health_collector.go` | `CollectorConfig.DebeziumName="goopay-mongodb-cdc"` default | Xóa field DebeziumName; collector luôn gọi `DebeziumAll` |
| `system_health_alerts.go::detectConditions` | đọc `deb["connector"]` single | đọc `deb["connectors"]` slice → emit DebeziumConnectorFailed per-connector; giữ legacy branch backwards-compat |
| `system_health_compute.go` | check `deb["status"]=="FAILED"` | check `StatusDegraded`/`StatusDown` (worst-of từ DebeziumAll); giữ legacy "FAILED" |
| `api/system_health_handler.go::RestartDebezium` | `h.debeziumName` hardcoded default | đọc `connector_name` từ query/body; HTTP 400 nếu thiếu |
| `server/server.go` wiring | truyền `cfg.System.DebeziumConnector` | Xóa argument |
| `config/config.go` | field `DebeziumConnector` + env bind | Xóa |
| `config/*.yml` | `debeziumConnector:` keys | Xóa |
| Tests (4 files) | hardcode "goopay-mongodb-cdc" | đổi sang generic "test-connector" fixture |

## 4. Verify kết quả thực tế

### 4.1 Build + Vet
```
$ cd centralized-data-service && go build ./... && go vet ./...   # CLEAN
$ cd cdc-cms-service && go build ./... && go vet ./...            # CLEAN
$ cd cdc-cms-service && go test ./internal/infra/observability/... ./internal/infra/persistence/... ./internal/api/...
ok  cdc-cms-service/internal/infra/observability           1.275s
ok  cdc-cms-service/internal/infra/observability/probes    0.748s
ok  cdc-cms-service/internal/infra/persistence             1.765s
ok  cdc-cms-service/internal/api                           0.556s
```

### 4.2 E2E worker (Kafka signal → probe)

Trước fix (log cũ activity_log 19:42):
```
operation=debezium-signal  status=error
error_message=signal published to kafka but connector not ready: state= task_count=0 reason=connector status HTTP 404
```

Sau fix (publish signal lúc 03:19 với `database=centralized-export-service collection=export-jobs`):

Worker log:
```json
{"msg":"debezium signal published BUT connector not ready — snapshot will NOT execute",
 "trace_id":"e2e-removestatic-001",
 "connector_name":"goopay-local",
 "connector_state":"RUNNING",
 "task_count":0,
 "reason":"connector has 0 tasks (...)"}
```

Activity log:
```
operation=debezium-signal  status=error
error_message=signal published to kafka but connector "goopay-local" not ready: state=RUNNING task_count=0 task_state= reason=connector has 0 tasks (...)
```

⇒ Resolver dynamic ĐÚNG: `(database=centralized-export-service, collection=export-jobs)` → tra `source_object_registry` JOIN `connection_registry` → `connection_code="goopay-local"`. KHÔNG còn HTTP 404. (Task count = 0 là issue source DB unreachable, không thuộc phạm vi này.)

### 4.3 E2E cms-service (system-health)

```
$ curl :8083/api/v1/system/health
...
"debezium":{
  "connectors":[
    {"connector":"goopay-dev","status":"RUNNING","tasks":[{"id":0,"state":"RUNNING"}]},
    {"connector":"goopay-local","status":"RUNNING","tasks":[]}
  ],
  "count":2,
  "status":"ok"
}
```

⇒ Enumerate ĐỘNG cả 2 connector từ `GET /connectors`. Không còn ràng buộc 1 connector hardcoded.

## 5. Danh sách file thay đổi

**centralized-data-service (cdc-worker)** — 10 files:
1. `internal/service/connector_resolver.go` *(NEW)*
2. `internal/service/debezium_signal.go`
3. `internal/handler/recon_handler.go`
4. `internal/service/recon_heal.go`
5. `internal/handler/command_handler.go`
6. `internal/admin/helpers.go`
7. `internal/server/worker_server.go`
8. `config/config.go`
9. `config/config-local.yml`
10. `config/config-production.yml`

**cdc-cms-service** — 14 files:
1. `internal/infra/observability/probes/debezium.go` *(thêm DebeziumAll)*
2. `internal/infra/observability/probes/debezium_test.go`
3. `internal/infra/observability/system_health_collector.go`
4. `internal/infra/observability/system_health_alerts.go`
5. `internal/infra/observability/system_health_alerts_test.go`
6. `internal/infra/observability/system_health_collector_test.go`
7. `internal/infra/observability/system_health_compute.go`
8. `internal/infra/persistence/alert_manager_test.go`
9. `internal/api/system_health_handler.go`
10. `internal/server/server.go`
11. `config/config.go`
12. `config/config-sample.yml`
13. `config/config-local.yml`
14. `config/config-production.yml`

**Workspace memory** — 3 files:
- `agent/memory/workspaces/DebeziumSignalKafkaMigration/05_progress.md` *(APPEND)*
- `agent/memory/workspaces/DebeziumSignalKafkaMigration/report_2026-05-20_remove-static-connector-names.md` *(NEW — file này)*
- `agent/memory/global/lessons.md` *(APPEND Global Pattern)*

## 6. Skill / Kỹ năng đã sử dụng

- **Read/Edit/Write tools** (Claude Code): patch file chính xác, append-only memory.
- **Bash + grep**: audit hardcoded strings; verify build/vet/test; query Postgres qua `docker exec`; publish NATS qua `nats` CLI.
- **TaskCreate/TaskUpdate**: track 8 sub-task phases (admin refactor → command_handler → worker build → cms batch → cms build → worker E2E → cms verify → report).
- **CLAUDE.md governance**: §3 plan-first, §6 minimal impact, §7 immutable progress append, §11 memory protection, §13 Global Pattern lesson.
- **Global Pattern abstraction** (rule 13): trừu tượng case Debezium → pattern chung cho mọi control-plane dynamic resource (K8s Operator, multi-tenant IAM, Stripe per-account…).
- **Atomic resolver pattern**: 1 helper file (`connector_resolver.go`) phục vụ 3 handler + 1 admin server → tránh duplicate query 4 nơi.
- **Backwards-compat slice/single-shape parsing**: alerts.go đọc cả new `connectors:[...]` lẫn legacy single shape → an toàn rolling deploy.
