# 02_plan_signal_topic_bootstrap — Auto-create Kafka signal topic at worker startup (Audit pass 5)

## 1. Vấn đề

`debezium-signal` lỗi `[3] Unknown Topic Or Partition: cdc.signal.commands` (3 row activity_log mới nhất 18:25:17 UTC).

Mặc dù broker `gpay-kafka` có `auto.create.topics.enable=true` (verify qua `kafka-configs --describe`), kafka-go writer KHÔNG kích hoạt server-side auto-create vì:
- kafka-go v0.4.50 `Writer.WriteMessages` gửi `MetadataRequest` với `allowAutoTopicCreation=false` mặc định.
- Broker trả error code 3 (UnknownTopicOrPartition) mà không tạo topic ngầm cho non-consumer request.

Manual workaround `kafka-topics --create` đã thực hiện trong session trước là CHEAT — vi phạm rule "không cheat db hay thay đổi config để đạt kết quả".

## 2. Giải pháp core-systems: Application-owned topic bootstrap

**Lý do chọn pattern này (so với alternatives)**:

| Alternative | Pros | Cons | Verdict |
|---|---|---|---|
| Broker `auto.create.topics.enable=true` | Zero-code | Production anti-pattern (typo creates phantom topics, can't set partition/RF per-topic) | ❌ Không production-portable |
| docker-compose `KAFKA_CREATE_TOPICS` env | Declarative | Chỉ work cho Bitnami/Wurstmeister image. cp-kafka:7.6.0 không hỗ trợ | ❌ Không match image hiện tại |
| Bootstrap script (kafka-init container) | Visible | Thêm container, race điều phối với worker | ⚠️ Over-engineering |
| **Worker startup EnsureTopic (chosen)** | Single source of truth; application khai báo dependency của nó (parallel với DB migration); idempotent; per-environment portable | Thêm ~30 dòng code trong service đã có | ✅ Best fit |

Pattern này giống cách Debezium connector tự tạo `schema-history` topic ở startup (thấy trong `cdc-mariadb-source.json`: `schema.history.internal.kafka.topic: cdc.mariadb.schema-history`).

## 3. Implementation chi tiết

### 3.1. File `internal/service/debezium_signal.go`

Thêm method `EnsureTopic(ctx)` vào `DebeziumSignalClient`:

```go
// EnsureTopic creates the Debezium signal topic if it does not exist.
// Idempotent — returns nil on TopicAlreadyExists. Called at worker
// startup so the topic is guaranteed to exist before the first publish.
//
// Partition=1 vì signal channel cần ordering chặt (Debezium đọc theo
// offset, không phải key). RF=1 phù hợp dev/single-broker; production
// triển khai qua infra-as-code (terraform/helm) override RF cao hơn
// + tự tạo trước → EnsureTopic return AlreadyExists nhanh.
func (d *DebeziumSignalClient) EnsureTopic(ctx context.Context) error {
    if d == nil || d.writer == nil || len(d.cfg.KafkaBrokers) == 0 {
        return nil
    }
    client := &kafka.Client{
        Addr:    kafka.TCP(d.cfg.KafkaBrokers...),
        Timeout: 10 * time.Second,
    }
    resp, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
        Topics: []kafka.TopicConfig{{
            Topic:             d.cfg.SignalKafkaTopic,
            NumPartitions:     1,
            ReplicationFactor: 1,
        }},
    })
    if err != nil {
        return fmt.Errorf("create signal topic: %w", err)
    }
    if topicErr := resp.Errors[d.cfg.SignalKafkaTopic]; topicErr != nil {
        if errors.Is(topicErr, kafka.TopicAlreadyExists) {
            d.logger.Debug("debezium signal topic already exists",
                zap.String("topic", d.cfg.SignalKafkaTopic))
            return nil
        }
        return fmt.Errorf("ensure signal topic %s: %w",
            d.cfg.SignalKafkaTopic, topicErr)
    }
    d.logger.Info("debezium signal topic ensured",
        zap.String("topic", d.cfg.SignalKafkaTopic),
        zap.Int("partitions", 1),
        zap.Int("replication_factor", 1))
    return nil
}
```

Imports cần thêm: `errors` (cho `errors.Is`).

### 3.2. File `internal/server/worker_server.go`

Sau dòng `signalClient = service.NewDebeziumSignalClient(...)` (line ~388), thêm call EnsureTopic:

```go
if signalClient != nil {
    ensureCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    if err := signalClient.EnsureTopic(ensureCtx); err != nil {
        logger.Warn("failed to ensure debezium signal topic; first publish may fail",
            zap.String("topic", cfg.Debezium.SignalKafkaTopic),
            zap.Error(err))
    }
    cancel()
}
```

Policy fail-soft: WARN không panic. Producer publish lần đầu sẽ fail loudly với rõ "Unknown Topic" → operator tự xử (so với silent: nếu panic, worker boot-loop nhưng có thể là transient Kafka outage → false alarm).

## 4. Verify steps

1. `go build ./... && go vet ./...` clean.
2. Xoá topic `cdc.signal.commands` để verify fix tự tạo lại từ đầu.
3. Restart worker (`make run` hoặc tương đương).
4. Check worker log: "debezium signal topic ensured" (lần đầu) hoặc "already exists" (lần sau).
5. Verify topic tồn tại: `kafka-topics --bootstrap-server localhost:19092 --describe --topic cdc.signal.commands`.
6. Click Snapshot từ UI → activity_log row `debezium-signal` status=success, error_message=NULL.

## 5. Risks

- **EnsureTopic ContextDeadlineExceeded** nếu Kafka chưa ready ở worker boot. Mitigation: 15s timeout, fail-soft. Worker tiếp tục, producer publish retry naturally khi user click.
- **Permission denied** nếu Kafka có ACL strict + worker user không có CREATE TOPIC. Mitigation: production khuyến nghị pre-create topic qua IaC; EnsureTopic chỉ là safety net.

## 6. Files thay đổi

| File | Lý do |
|---|---|
| `internal/service/debezium_signal.go` | Add method `EnsureTopic` + import `errors` |
| `internal/server/worker_server.go` | Call `EnsureTopic` sau khi build signalClient |

## 7. Files KHÔNG thay đổi (verify scope)

- Config YAML, docker-compose: không sửa. Topic name vẫn từ `cfg.Debezium.SignalKafkaTopic`.
- Postgres DB: không touch.
- FE/CMS: không touch (publisher path đã verified ở audit pass 3).
- Debezium connector JSONs: không touch (đã có `signal.kafka.topic` từ pass 1+2).
