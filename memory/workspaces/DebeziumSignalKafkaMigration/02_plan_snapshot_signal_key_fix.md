# 02 — Plan: snapshot-signal-kafka-key-fix (2026-05-20)

## Strategy
Fix **2 bug độc lập** theo thứ tự ưu tiên Bug B (chặn first) → Bug A (key routing). Test end-to-end chỉ valid khi cả 2 fix.

## Step 1 — CMS: force-overwrite + reject Vite placeholder
**File**: `cdc-cms-service/internal/api/system_connectors_handler.go`

**Đổi `injectDebeziumSignalDefaults`**:
- Trước: chỉ set khi cfg chưa có key (`if _, set := cfg[k]; !set`).
- Sau: **luôn overwrite** `signal.kafka.topic` + `signal.kafka.bootstrap.servers` + `signal.enabled.channels` + `signal.kafka.group.id`. Đây là infra config — backend takes ownership, không cho FE control.

**Thêm validator**: nếu cfg chứa key bắt đầu bằng `__VITE_` hoặc `import.meta.env` → trả 400. Đề phòng FE leak placeholder vào key khác.

**Lý do force-overwrite (không chỉ reject)**:
- Reject placeholder mới chỉ cover Vite. Nếu FE đổi placeholder syntax → fail mode khác.
- Backend luôn biết đúng `signalTopic` + `signalBootstrap` từ config CMS (cdc.signal.commands + gpay-kafka:9092). Không có lý do gì để FE override.

## Step 2 — Worker: thêm ResolveTopicPrefix + sửa TriggerIncrementalSnapshot
**Files**:
- `centralized-data-service/internal/service/debezium_signal.go`
- `centralized-data-service/internal/service/connector_resolver.go`
- `centralized-data-service/internal/handler/recon_handler.go`
- `centralized-data-service/internal/service/recon_heal.go`

**Thêm method** trên `DebeziumSignalClient`:
```go
// ResolveTopicPrefix fetches the connector's topic.prefix via Kafka
// Connect REST. Debezium signal channel filters incoming records by
// matching message key against this value — wrong key => silent drop.
func (d *DebeziumSignalClient) ResolveTopicPrefix(ctx context.Context, connectorName string) (string, error)
```
- GET `{KafkaConnectBaseURL}/connectors/{name}/config`
- Decode JSON, return `cfg["topic.prefix"]`.
- Error nếu connector không tồn tại / topic.prefix vắng.

**Sửa `TriggerIncrementalSnapshot` signature**:
```go
func (d *DebeziumSignalClient) TriggerIncrementalSnapshot(
    ctx context.Context,
    connectorName, engine, database, collection, filter string,
) (string, error)
```
- Resolve `topicPrefix := d.ResolveTopicPrefix(ctx, connectorName)`.
- `msg.Key = []byte(topicPrefix)` thay vì `qualified`.
- Update comment lines 206-209 — xoá claim "key not used", thay bằng requirement Debezium 2.5+.

**Update callers**:
- `recon_handler.go:344` — resolve connectorName trước (đã có `ResolveConnectorNameBySource`), pass vào TriggerIncrementalSnapshot.
- `recon_heal.go:680` — tương tự.

## Step 3 — Migrate 2 connector existing
Dùng CMS PATCH /api/v1/system/connectors/{name}/config (sau khi Step 1 deploy) với cfg chứa các keys non-signal — backend sẽ force-inject signal.* đúng.

Hoặc dùng Kafka Connect REST trực tiếp (faster, bypass CMS):
```
curl -X PUT -H 'Content-Type: application/json' \
  http://127.0.0.1:18083/connectors/goopay-local/config \
  -d '{...cfg với signal.kafka.topic: "cdc.signal.commands"...}'
```

Restart sau update để pickup config (Kafka Connect tự reconfigure task).

## Step 4 — Verify
1. `docker logs gpay-kafka-connect | grep "Subscribing to signals topic"` → phải thấy `'cdc.signal.commands'`, không còn `__VITE_*`.
2. Trigger snapshot UI:
   - `count(*)` shadow trước.
   - Trigger.
   - Wait 10s.
   - `count(*)` shadow sau → phải tăng.
   - `docker logs gpay-kafka-connect | grep -i "Requested snapshot\|Snapshot ended"` → phải thấy execute event.
3. Dump `cdc.signal.commands` lần cuối → key mới phải là `cdc.goopay`.

## Risks
- Force-overwrite Bug B có thể phá use case operator nào đó muốn dùng signal topic khác. **Mitigation**: log warning khi overwrite + ghi doc + per-key opt-in qua flag tương lai. Hiện tại không có operator như vậy → safe.
- Resolve topic.prefix qua HTTP mỗi snapshot → latency. **Mitigation**: cache in-memory (TTL 60s) ở phase sau, hiện tại 1 HTTP call/snapshot OK (snapshot rare, không hot path).

## Rollback
- Bug B fix: revert commit CMS, restart CMS container.
- Bug A fix: revert commit worker, rebuild + restart.
- Connector migrate: PUT lại config cũ.
