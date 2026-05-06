# 09_tasks_solution — Multi-Engine Unified Pipeline

> Đối ứng: `08_tasks_multi_engine_unified.md`.
> Mỗi task: code skeleton + verify command. Sẽ APPEND khi thực thi xong (không overwrite).

## L1 cdc-worker

### S1.1 KafkaConfig dual-decode

**File**: `centralized-data-service/config/config.go`

```go
// KafkaConfig … (giữ phần đầu)
type KafkaConfig struct {
    Brokers           []string `mapstructure:"brokers"`
    GroupID           string   `mapstructure:"groupId"`
    TopicPrefix       []string `mapstructure:"-"`              // populated by hook
    SchemaRegistryURL string   `mapstructure:"schemaRegistryUrl"`
    Enabled           bool     `mapstructure:"enabled"`
}

// Viper hook: accept either `topicPrefix: "cdc.gpay"` (string) or
// `topicPrefixes: [cdc.gpay, cdc.goopay]` (list) for backward compat.
func decodeTopicPrefix(raw map[string]any) []string {
    if v, ok := raw["topicPrefixes"]; ok {
        if arr, ok := v.([]any); ok {
            out := make([]string, 0, len(arr))
            for _, x := range arr { out = append(out, fmt.Sprint(x)) }
            return out
        }
    }
    if v, ok := raw["topicPrefix"]; ok {
        switch t := v.(type) {
        case string: return []string{t}
        case []any:
            out := make([]string, 0, len(t))
            for _, x := range t { out = append(out, fmt.Sprint(x)) }
            return out
        }
    }
    return nil
}

// In Load(): cfg.Kafka.TopicPrefix = decodeTopicPrefix(rawKafkaMap)
```

**Test** `config/config_test.go`:

```go
func TestKafkaConfig_TopicPrefix_Scalar(t *testing.T) {
    yaml := `kafka: {topicPrefix: "cdc.gpay"}`
    cfg, _ := loadFromYAML(yaml)
    require.Equal(t, []string{"cdc.gpay"}, cfg.Kafka.TopicPrefix)
}
func TestKafkaConfig_TopicPrefix_List(t *testing.T) {
    yaml := `kafka: {topicPrefixes: [cdc.gpay, cdc.goopay]}`
    cfg, _ := loadFromYAML(yaml)
    require.Equal(t, []string{"cdc.gpay","cdc.goopay"}, cfg.Kafka.TopicPrefix)
}
```

### S1.2 KafkaConsumer multi-prefix

**File**: `internal/handler/kafka_consumer.go` (function `discoverTopics` ~line 440)

```go
// Old: single prefix filter
// New: union over kc.config.TopicPrefix
seen := make(map[string]struct{})
for _, prefix := range kc.config.TopicPrefix {
    for _, topic := range allKafkaTopics {
        if !strings.HasPrefix(topic, prefix) { continue }
        if _, ok := seen[topic]; ok { continue }
        seen[topic] = struct{}{}
        // existing per-topic filter via debeziumNamespaces (S1.3)
    }
}
```

`KafkaConsumerConfig.TopicPrefix` đổi `string` → `[]string`. Update `worker_server.go:509`.

### S1.3 RegistryService.GetDebeziumNamespaces

**File**: `internal/service/registry_service.go` (sau `GetDebeziumTables`)

```go
type DebeziumNamespace struct {
    Engine    string
    Database  string
    Namespace string  // PG schema | Mongo db (== database) | MySQL db
    Object    string  // table | collection
}

func (rs *RegistryService) GetDebeziumNamespaces() []DebeziumNamespace {
    rs.mu.RLock()
    defer rs.mu.RUnlock()
    out := make([]DebeziumNamespace, 0, len(rs.registryCache))
    for _, r := range rs.registryCache {
        if r.SyncEngine != "debezium" && r.SyncEngine != "both" { continue }
        out = append(out, DebeziumNamespace{
            Engine: r.SourceEngineType,
            Database: r.SourceDatabase,
            Namespace: r.SourceNamespace,
            Object: r.SourceTable,  // = source_object_name
        })
    }
    return out
}
```

KafkaConsumer filter từ tableName → tuple `(engine, db, namespace, object)` parsed từ topic `cdc.<prefix>.<db>.<object>`. Giữ helper hiện có (`extractTableNameFromTopic`) — nâng cấp trả tuple.

### S1.4 config-local.yml

```yaml
kafka:
  enabled: true
  brokers: [localhost:19092]
  groupId: cdc-worker-group
  topicPrefixes:
    - cdc.gpay
    - cdc.goopay
    - cdc.mariadb
  schemaRegistryUrl: http://localhost:18081
```

## L4 infra MariaDB

### S4.1 docker-compose service

**File**: `centralized-data-service/docker-compose.yml` (append)

```yaml
  gpay-mariadb:
    image: mariadb:10.11
    container_name: gpay-mariadb
    restart: unless-stopped
    environment:
      MARIADB_ROOT_PASSWORD: gpay_pass
      MARIADB_DATABASE: goopay_legacy_maria
      MARIADB_USER: src_maria
      MARIADB_PASSWORD: src_maria_pass
    command:
      - --server-id=10
      - --log-bin=mysql-bin
      - --binlog-format=ROW
      - --binlog-row-image=FULL
      - --gtid-strict-mode=1
      - --bind-address=0.0.0.0
    ports: ["13306:3306"]
    volumes:
      - ./deployments/mariadb/init:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "mariadb -uroot -p$$MARIADB_ROOT_PASSWORD -e 'SELECT 1'"]
      interval: 10s
      timeout: 5s
      retries: 10
```

### S4.2 Init SQL

**File**: `centralized-data-service/deployments/mariadb/init/01_seed.sql`

```sql
CREATE DATABASE IF NOT EXISTS goopay_legacy_maria;
USE goopay_legacy_maria;
CREATE TABLE IF NOT EXISTS legacy_orders (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  amount INT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO legacy_orders (user_id, amount, status) VALUES
  (1001, 100, 'pending'), (1002, 200, 'paid'),
  (1003, 300, 'pending'), (1004, 400, 'paid'),
  (1005, 500, 'failed');
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'src_maria'@'%';
GRANT SELECT ON goopay_legacy_maria.* TO 'src_maria'@'%';
FLUSH PRIVILEGES;
```

### S4.3 Debezium connector spec

**File**: `centralized-data-service/deployments/connectors/cdc-mariadb-source.json`

```json
{
  "name": "cdc-mariadb-source",
  "config": {
    "connector.class": "io.debezium.connector.mysql.MySqlConnector",
    "tasks.max": "1",
    "database.hostname": "gpay-mariadb",
    "database.port": "3306",
    "database.user": "src_maria",
    "database.password": "src_maria_pass",
    "database.server.id": "184054",
    "topic.prefix": "cdc.mariadb",
    "database.include.list": "goopay_legacy_maria",
    "table.include.list": "goopay_legacy_maria.legacy_orders",
    "schema.history.internal.kafka.bootstrap.servers": "gpay-kafka:9092",
    "schema.history.internal.kafka.topic": "schema-history.cdc.mariadb",
    "snapshot.mode": "initial",
    "key.converter": "io.confluent.connect.avro.AvroConverter",
    "key.converter.schema.registry.url": "http://gpay-schema-registry:8081",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "http://gpay-schema-registry:8081"
  }
}
```

Manual deploy:

```bash
curl -X POST -H 'Content-Type: application/json' \
  --data @deployments/connectors/cdc-mariadb-source.json \
  http://localhost:18083/connectors
```

### S4.4 Migration seed

**File**: `centralized-data-service/migrations/cdc/049_mariadb_seed_legacy_orders.sql`

```sql
BEGIN;
INSERT INTO cdc_system.source_object_registry
  (object_code, source_connection_id, source_engine_type, source_database,
   source_namespace, source_object_name, source_object_type, source_locator_json,
   normalized_source_key, primary_key_field, sync_engine, is_active, profile_status,
   provisioning_mode, provisioning_state, notes)
VALUES
  ('mariadb_legacy_orders_v2', 1, 'mysql', 'goopay_legacy_maria',
   'goopay_legacy_maria', 'legacy_orders', 'table',
   '{"engine":"mariadb","host":"gpay-mariadb","port":3306}',
   'mariadb:goopay_legacy_maria.legacy_orders', 'id',
   'debezium', false, 'draft', 'manual', 'draft',
   'seed for multi_engine_unified phase')
ON CONFLICT (object_code) DO NOTHING;
COMMIT;
```

`is_active=false` ban đầu — operator dùng FE Toggle để đổi sang `auto` + activate sau khi smoke.

## L2 cms-api

### S2.1 SourceListResponse expand

Khi vào việc sẽ grep handler hiện tại; expected diff: add 3 field vào DTO + struct query SELECT.

### S2.2 Idempotency middleware

**File** (new): `cdc-cms-service/internal/middleware/idempotency.go`

```go
type idempotencyEntry struct {
    statusCode int
    body       []byte
    expiresAt  time.Time
}

type IdempotencyStore struct {
    mu    sync.Mutex
    cache map[string]idempotencyEntry
    ttl   time.Duration
}

func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore { ... }

func (s *IdempotencyStore) Middleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := c.Get("Idempotency-Key")
        if key == "" { return c.Next() }
        // … cache hit → replay; miss → run handler, capture, store
    }
}
```

Wire ở router: chỉ apply cho route `/provisioning/mode`.

### S2.3 Smoke command

```bash
# (Sau khi cms-service started + có JWT của ops admin)
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -d '{"username":"ops","password":"ops"}' | jq -r .access_token)

for ID in 26 <MONGO_ID> <MARIA_ID>; do
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: smoke-$ID-$(date +%s)" \
    -d '{"mode":"auto"}' \
    http://<CMS_PORT>/api/v1/cms/sources/$ID/provisioning/mode
done
```

## L3 cms-fe

### S3.1 Type

**File**: `cdc-cms-web/src/types/index.ts` (location TBD khi vào việc)

```ts
export interface SourceObjectRow {
  // … existing
  provisioning_mode?: 'auto' | 'manual';
  provisioning_state?: 'draft' | 'shadow_pending' | 'shadow_active' |
                       'master_pending' | 'master_active' |
                       'mapping_pending' | 'mapping_ready' |
                       'schedule_pending' | 'running' | 'paused' |
                       'failed' | 'archived';
  source_engine_type?: 'postgresql' | 'mongodb' | 'mysql' | 'mariadb';
}
```

### S3.2 Hook

**File**: `cdc-cms-web/src/hooks/useProvisioningMode.ts`

```ts
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { cmsApi } from '../api/cmsApi';

export function useProvisioningMode() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, mode }: { id: number; mode: 'auto'|'manual' }) =>
      cmsApi.post(`/api/v1/cms/sources/${id}/provisioning/mode`,
        { mode },
        { headers: { 'Idempotency-Key': `prov-mode-${id}-${Date.now()}` } }
      ),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['sources'] }); },
  });
}
```

### S3.3 Columns + Switch + Filter

**File**: `cdc-cms-web/src/pages/TableRegistry.tsx`

```tsx
const engineColor: Record<string,string> = {
  postgresql: 'blue', mongodb: 'green', mysql: 'orange', mariadb: 'orange',
};
const stateColor: Record<string,string> = {
  draft: 'default', running: 'success', paused: 'warning',
  failed: 'error', archived: 'default',
  // *_pending → 'processing', *_active → 'cyan', mapping_ready → 'cyan'
};

// Add columns:
{ title: 'Engine', dataIndex: 'source_engine_type',
  filters: [{text:'PG',value:'postgresql'},{text:'Mongo',value:'mongodb'},{text:'MariaDB',value:'mariadb'},{text:'MySQL',value:'mysql'}],
  onFilter: (v, r) => r.source_engine_type === v,
  render: (v: string) => v ? <Tag color={engineColor[v]}>{v}</Tag> : <Tag>—</Tag> },
{ title: 'Mode', dataIndex: 'provisioning_mode',
  render: (v: string|undefined, r: SourceObjectRow) => (
    <Switch
      checked={v === 'auto'}
      checkedChildren="Auto" unCheckedChildren="Manual"
      onChange={(checked) => handleToggleMode(r, checked ? 'auto' : 'manual')}
    />
  ) },
{ title: 'State', dataIndex: 'provisioning_state',
  render: (v: string|undefined) => v ? <Tag color={stateColor[v] ?? 'processing'}>{v}</Tag> : <Tag>—</Tag> },
```

`handleToggleMode`:

```tsx
const modeMut = useProvisioningMode();
const handleToggleMode = (record: SourceObjectRow, mode: 'auto'|'manual') => {
  const inFlight = ['shadow_pending','master_pending','mapping_pending','schedule_pending','failed']
    .includes(record.provisioning_state ?? '');
  const fire = () => modeMut.mutate({ id: record.id!, mode },
    { onSuccess: () => message.success(`Mode → ${mode}`),
      onError: (e: any) => {
        if (e?.response?.status === 409) message.error('CAS conflict — refresh and retry');
        else if (e?.response?.status === 422) message.error('Invalid transition');
        else message.error('Toggle failed');
      } });
  if (inFlight && mode === 'manual') {
    Modal.confirm({
      title: 'Switch to Manual mid-flow?',
      content: `Current state: ${record.provisioning_state}. Manual mode keeps current state but stops auto fan-out. In-flight commands are NOT cancelled.`,
      onOk: fire, okText: 'Switch', cancelText: 'Keep Auto',
    });
  } else {
    fire();
  }
};
```

## L5 E2E

Đã liệt kê ở `08_tasks_multi_engine_unified.md` §L5. Sẽ APPEND log thực thi vào `05_progress_multi_engine_unified.md` (file mới sẽ tạo ở step verify cuối).

## Tham chiếu chéo

- Lesson `lessons.md:1347` — phase mới trong existing workspace.
- Architect ruling `04_decisions_provisioning_mode.md` D1, D5, D6, D8 — đã áp dụng cho Toggle mode flow.
- `agent/workflows/feature-dev.md` — Phases 1–7 — em đang ở Phase 4 (Architecture Design) → sẽ chuyển Phase 5 (Implementation) sau khi user approve.
