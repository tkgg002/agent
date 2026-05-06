# 09 — Solution: Core-Flow Hardening Phase P0+P1

**Phase code**: `core_flow_hardening_p0_p1`
**Created**: 2026-05-04 13:55 (+07)
**Companion**: `01_requirements_*`, `02_plan_*`, `08_tasks_*`.
**Mục đích**: code-level chi tiết cho Muscle copy-execute, kèm exact diff hint, edge case, và verification commands.

---

## P1.1 — `handleDelete` UPSERT (G3)

### Diff hint — `internal/handler/event_handler.go`

**Before** (lines 175-181):
```go
sql := fmt.Sprintf(`UPDATE %s SET _deleted = TRUE, _updated_at = NOW() WHERE %s = ?`,
    qualifiedShadowTable(route),
    quoteEventIdent(pgPKField),
)
if err := db.WithContext(ctx).Exec(sql, pkValue).Error; err != nil {
    return fmt.Errorf("delete fan-out (target=%s): %w", config.TargetTable, err)
}
```

**After**:
```go
// P1.1 (G3) — tombstone-first UPSERT. Handles delete events for rows
// that may not yet exist in shadow (replay / first-touch delete).
// _gpay_source_id stamped here mirrors B11 INSERT/UPDATE branch.
sql := fmt.Sprintf(
    `INSERT INTO %s (%s, _gpay_source_id, _deleted, _created_at, _updated_at, _source)
     VALUES (?, ?::text, TRUE, NOW(), NOW(), 'debezium')
     ON CONFLICT (%s) DO UPDATE SET
        _deleted    = TRUE,
        _updated_at = NOW()`,
    qualifiedShadowTable(route),
    quoteEventIdent(pgPKField),
    quoteEventIdent(pgPKField),
)
if err := db.WithContext(ctx).Exec(sql, pkValue, pkValue).Error; err != nil {
    return fmt.Errorf("delete fan-out (target=%s): %w", config.TargetTable, err)
}
```

**Edge cases**:
- `pkValue` empty string → INSERT vẫn chạy, conflict resolution dựa unique constraint. Nếu shadow chưa có row với `pk=''` → insert tombstone với pk rỗng. Acceptable behavior — caller upstream (`extractPrimaryKey`) trả empty là điều bất thường.
- PK column type không phải TEXT (BIGINT, UUID): pkValue là string → driver auto-cast (gorm + pgx handle). Đã verified ở B11 INSERT branch.
- `_gpay_source_id`: TEXT cast `?::text` đảm bảo consistency với column type ở schema_adapter.

### Test stub — `internal/handler/event_handler_test.go`

```go
func TestHandleDelete_FirstTouch_TombstoneInsert(t *testing.T) {
    sqlDB, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
    defer sqlDB.Close()
    gormDB, _ := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})

    h := &EventHandler{db: gormDB, logger: zap.NewNop()}
    route := &service.ResolvedSourceRoute{
        ShadowBinding: &model.ShadowBinding{ShadowSchema: "shadow_test", ShadowTable: "orders"},
        TableConfig: &service.TableConfig{
            TargetTable:     "orders",
            PrimaryKeyField: "id",
            SourceType:      "postgres",
        },
    }
    event := &model.CDCEvent{Data: model.CDCData{Op: "d", Before: map[string]interface{}{"id": float64(999)}}}

    mock.ExpectExec(`INSERT INTO "shadow_test"."orders".*ON CONFLICT.*DO UPDATE SET.*_deleted\s*=\s*TRUE`).
        WithArgs("999", "999").
        WillReturnResult(sqlmock.NewResult(1, 1))

    err := h.handleDelete(context.Background(), event, []*service.ResolvedSourceRoute{route})
    require.NoError(t, err)
    require.NoError(t, mock.ExpectationsWereMet())
}
```

(Lưu ý: nếu đã có sqlmock import trong test file thì reuse; nếu chưa, add `github.com/DATA-DOG/go-sqlmock` vào go.mod.)

### Smoke command (after build)

```bash
# Source PG: delete row id=64 (đã exist từ B11 verify)
docker exec gpay-postgres-source psql -U src_user -d goopay_source \
  -c "DELETE FROM public.orders WHERE id = 64;"
sleep 5
# Verify shadow tombstone
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  -c "SELECT id, _deleted, _gpay_source_id, _updated_at FROM shadow_goopay_source.orders WHERE id = 64;"
# Expect: _deleted=t, _gpay_source_id='64', _updated_at fresh.
```

---

## P0.1 — Reader manager NATS-driven refresh (G1)

### File `internal/handler/kafka_consumer.go` — full struct + method updates

**Add imports** (nếu chưa có):
```go
"sync"
```

**Modify struct** (find `type KafkaConsumer struct`):
```go
type KafkaConsumer struct {
    config    KafkaConfig
    handler   *EventHandler
    validator *service.SchemaValidator
    masking   *service.MaskingService
    registry  service.MetadataRegistry
    logger    *zap.Logger
    readers   []*kafka.Reader
    batches   map[string]*topicBatch
    batchMu   sync.Mutex
    // P0.1 additions:
    refreshMu     sync.Mutex
    currentTopics []string
}
```

(Nếu existing struct đã có `mu sync.Mutex` cho mục đích khác, dùng tên `refreshMu` để tránh đụng.)

**New method `buildReader`** (insert sau method tồn tại nào hợp lý, e.g., trước `Start`):
```go
func (kc *KafkaConsumer) buildReader(topics []string) *kafka.Reader {
    return kafka.NewReader(kafka.ReaderConfig{
        Brokers:          kc.config.Brokers,
        GroupID:          kc.config.GroupID,
        GroupTopics:      topics,
        MinBytes:         10e3,
        MaxBytes:         10e6,
        CommitInterval:   time.Second,
        SessionTimeout:   30 * time.Second,
        RebalanceTimeout: 30 * time.Second,
        StartOffset:      kafka.FirstOffset,
        Logger:           nil,
    })
}
```

**New method `RefreshTopics`**:
```go
// RefreshTopics re-discovers Kafka topics and recreates the reader if
// the topic set has changed. Idempotent. Safe for concurrent callers
// (NATS handler + background ticker + test).
func (kc *KafkaConsumer) RefreshTopics(ctx context.Context) error {
    newTopics, err := kc.discoverTopics(ctx)
    if err != nil {
        return fmt.Errorf("discover topics: %w", err)
    }

    kc.refreshMu.Lock()
    defer kc.refreshMu.Unlock()

    if topicSetEqual(kc.currentTopics, newTopics) {
        kc.logger.Debug("topic set unchanged, skipping refresh",
            zap.Int("count", len(newTopics)))
        return nil
    }

    kc.logger.Info("topic set changed, recreating reader",
        zap.Strings("old", kc.currentTopics),
        zap.Strings("new", newTopics))

    kc.flushAllBatches()

    for _, r := range kc.readers {
        if err := r.Close(); err != nil {
            kc.logger.Warn("close old reader", zap.Error(err))
        }
    }
    kc.readers = nil

    newReader := kc.buildReader(newTopics)
    kc.readers = append(kc.readers, newReader)
    kc.currentTopics = append([]string(nil), newTopics...)

    return nil
}

func topicSetEqual(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    set := make(map[string]struct{}, len(a))
    for _, t := range a {
        set[t] = struct{}{}
    }
    for _, t := range b {
        if _, ok := set[t]; !ok {
            return false
        }
    }
    return true
}
```

**Modify `Start` method** — thay đoạn từ line 140 trở đi:

```go
// === P0.1 — initial reader build via shared helper ===
kc.refreshMu.Lock()
reader := kc.buildReader(topics)
kc.readers = append(kc.readers, reader)
kc.currentTopics = append([]string(nil), topics...)
kc.refreshMu.Unlock()

kc.logger.Info("kafka consumer started",
    zap.Strings("topics", topics),
    zap.String("group", kc.config.GroupID),
)

flushTicker := time.NewTicker(5 * time.Second)
defer flushTicker.Stop()

// P0.1 safety net — auto-refresh every 60s.
refreshTicker := time.NewTicker(60 * time.Second)
defer refreshTicker.Stop()

for {
    select {
    case <-ctx.Done():
        kc.flushAllBatches()
        kc.Stop()
        return
    case <-flushTicker.C:
        kc.flushAllBatches()
    case <-refreshTicker.C:
        if err := kc.RefreshTopics(ctx); err != nil {
            kc.logger.Warn("auto refresh topics failed", zap.Error(err))
        }
    default:
        kc.refreshMu.Lock()
        if len(kc.readers) == 0 {
            kc.refreshMu.Unlock()
            time.Sleep(100 * time.Millisecond)
            continue
        }
        currentReader := kc.readers[0]
        kc.refreshMu.Unlock()

        msg, err := currentReader.FetchMessage(ctx)
        if err != nil {
            if ctx.Err() != nil {
                return
            }
            // If reader was recreated mid-fetch, error is expected.
            if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "reader closed") {
                kc.logger.Debug("reader closed during refresh, retrying", zap.Error(err))
                time.Sleep(200 * time.Millisecond)
                continue
            }
            kc.logger.Error("kafka fetch error", zap.Error(err))
            time.Sleep(time.Second)
            continue
        }

        // ... existing per-msg processing (lines 184-264) UNCHANGED
    }
}
```

(Note: must `import "errors"` và `"io"` nếu chưa có, để check reader-closed condition.)

### File `internal/server/worker_server.go` — add subscribe

Tìm block subscribe (sau line 261 cdc.cmd.alter-column), thêm:

```go
// P0.1 (G1) — kafka topic dynamic refresh trigger.
natsClient.Conn.Subscribe("cdc.cmd.kafka.refresh-topics", func(msg *nats.Msg) {
    if err := kafkaConsumer.RefreshTopics(context.Background()); err != nil {
        logger.Warn("nats-triggered topic refresh failed", zap.Error(err))
        return
    }
    logger.Info("nats-triggered topic refresh ok")
})
```

(Lưu ý: variable `kafkaConsumer` phải tồn tại ở scope này — verify nơi tạo. Nếu currently named khác, đổi cho khớp.)

### Test stubs — `internal/handler/kafka_consumer_test.go`

```go
func TestRefreshTopics_NoChange(t *testing.T) {
    kc := &KafkaConsumer{
        logger:        zap.NewNop(),
        currentTopics: []string{"a", "b"},
        readers:       []*kafka.Reader{nil},
    }
    // Stub discoverTopics: monkeypatch via interface OR refactor discoverTopics
    // to accept injectable provider. For minimal invasiveness, use a
    // test-only override field:
    kc.discoverFunc = func(ctx context.Context) ([]string, error) {
        return []string{"a", "b"}, nil
    }
    err := kc.RefreshTopics(context.Background())
    require.NoError(t, err)
    require.Len(t, kc.readers, 1) // unchanged
}

func TestRefreshTopics_AddTopic(t *testing.T) {
    kc := &KafkaConsumer{
        logger:        zap.NewNop(),
        currentTopics: []string{"a"},
        readers:       []*kafka.Reader{},
        config:        KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
    }
    kc.discoverFunc = func(ctx context.Context) ([]string, error) {
        return []string{"a", "b"}, nil
    }
    err := kc.RefreshTopics(context.Background())
    require.NoError(t, err)
    require.ElementsMatch(t, []string{"a", "b"}, kc.currentTopics)
}
```

(Note: cần thêm field test-only `discoverFunc func(ctx context.Context) ([]string, error)` vào struct, và refactor `discoverTopics` method để delegate qua nó nếu set. Pattern đã proven ở các Go codebase khác. Hoặc dùng interface wrapper — Muscle quyết.)

### Smoke command

```bash
# 1. Pre-stage: tạo collection Mongo mới
docker exec gpay-mongo-source mongosh goopay --eval \
  'db.smoke_p01.insertOne({_id:ObjectId(),tag:"P01-pre"})'

# 2. PUT include list extend
curl -X GET http://localhost:8083/connectors/mongo-source/config \
  | jq '.["collection.include.list"] += ",goopay.smoke_p01"' \
  | curl -X PUT http://localhost:8083/connectors/mongo-source/config \
        -H "Content-Type: application/json" --data @-

# 3. Trigger refresh
docker exec nats-cdc nats pub cdc.cmd.kafka.refresh-topics ""

# 4. Watch worker logs
docker logs --tail 20 cdc-worker | grep -i "topic set"
# Expect: "topic set changed, recreating reader" với new=[..., "cdc.goopay.goopay.smoke_p01"]

# 5. INSERT trigger CDC
docker exec gpay-mongo-source mongosh goopay --eval \
  'db.smoke_p01.insertOne({_id:ObjectId(),tag:"P01-after-refresh"})'
sleep 10

# 6. Verify shadow
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  -c "\dt shadow_goopay_mongo.smoke_p01*"
# Expect: shadow table tồn tại với rows.
```

---

## P0.2 — `cdc-admin-api` (G6)

### Directory structure

```
cmd/admin-api/
└── main.go

internal/admin/
├── server.go           # Gin router + middleware
├── types.go            # DTOs
├── source_register.go  # POST /v2/sources/register
├── helpers.go          # Debezium + Schema Registry HTTP clients
└── server_test.go
```

### File `cmd/admin-api/main.go`

```go
package main

import (
    "context"
    "log"
    "os"

    "centralized-data-service/internal/admin"
    "centralized-data-service/internal/config"

    "github.com/nats-io/nats.go"
    "go.uber.org/zap"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("config load: %v", err)
    }

    logger, _ := zap.NewProduction()
    defer logger.Sync()

    db, err := gorm.Open(postgres.Open(cfg.MasterDB.Default.DSN()), &gorm.Config{})
    if err != nil {
        logger.Fatal("open master db", zap.Error(err))
    }

    nc, err := nats.Connect(cfg.NATS.URL)
    if err != nil {
        logger.Fatal("nats connect", zap.Error(err))
    }
    defer nc.Drain()

    addr := os.Getenv("ADMIN_API_LISTEN_ADDR")
    if addr == "" {
        addr = "127.0.0.1:8090"
    }

    srv := admin.NewServer(admin.Deps{
        DB:                db,
        NATS:              nc,
        DebeziumBaseURL:   getEnvOr("DEBEZIUM_URL", "http://kafka-connect:8083"),
        SchemaRegistryURL: getEnvOr("SCHEMA_REGISTRY_URL", "http://schema-registry:8081"),
        AuthToken:         os.Getenv("ADMIN_API_TOKEN"),
        Logger:            logger,
    })

    logger.Info("admin-api starting", zap.String("addr", addr))
    if err := srv.Run(context.Background(), addr); err != nil {
        logger.Fatal("admin-api crashed", zap.Error(err))
    }
}

func getEnvOr(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}
```

### File `internal/admin/types.go`

```go
package admin

type RegisterSourceRequest struct {
    ObjectCode        string                 `json:"object_code"          binding:"required"`
    SourceEngineType  string                 `json:"source_engine_type"   binding:"required,oneof=postgres mongodb mariadb"`
    SyncEngine        string                 `json:"sync_engine"          binding:"required,oneof=debezium"`
    SourceObjectName  string                 `json:"source_object_name"   binding:"required"`
    SourceLocator     map[string]interface{} `json:"source_locator"       binding:"required"`
    TargetMasterTable string                 `json:"target_master_table"  binding:"required"`
    Notes             string                 `json:"notes"`
}

type RegisterSourceResponse struct {
    SourceObjectID    int64    `json:"source_object_id"`
    ProvisioningState string   `json:"provisioning_state"`
    StepsCompleted    []string `json:"steps_completed"`
    LastStepError     string   `json:"last_step_error,omitempty"`
}
```

### File `internal/admin/server.go`

```go
package admin

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/nats-io/nats.go"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

type Deps struct {
    DB                *gorm.DB
    NATS              *nats.Conn
    DebeziumBaseURL   string
    SchemaRegistryURL string
    AuthToken         string
    Logger            *zap.Logger
}

type Server struct {
    deps   Deps
    engine *gin.Engine
}

func NewServer(deps Deps) *Server {
    if deps.AuthToken == "" {
        deps.Logger.Warn("ADMIN_API_TOKEN empty — auth disabled (dev mode only)")
    }
    s := &Server{deps: deps}
    s.engine = s.buildEngine()
    return s
}

func (s *Server) buildEngine() *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(s.authMiddleware())
    r.POST("/v2/sources/register", s.handleRegisterSource)
    r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
    return r
}

func (s *Server) authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.URL.Path == "/healthz" {
            c.Next()
            return
        }
        if s.deps.AuthToken == "" {
            c.Next() // dev mode
            return
        }
        got := c.GetHeader("Authorization")
        want := "Bearer " + s.deps.AuthToken
        if got != want {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }
        c.Next()
    }
}

func (s *Server) Run(ctx context.Context, addr string) error {
    httpSrv := &http.Server{
        Addr:              addr,
        Handler:           s.engine,
        ReadHeaderTimeout: 10 * time.Second,
    }
    go func() {
        <-ctx.Done()
        shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        httpSrv.Shutdown(shutCtx)
    }()
    return httpSrv.ListenAndServe()
}
```

### File `internal/admin/source_register.go`

```go
package admin

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

func (s *Server) handleRegisterSource(c *gin.Context) {
    var req RegisterSourceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var sourceID int64
    var stepsCompleted []string

    // === Step 1: registry insert (idempotent) ===
    err := s.deps.DB.Transaction(func(tx *gorm.DB) error {
        var masterID int64
        if err := tx.Raw(`SELECT id FROM cdc_system.master_binding
                          WHERE master_table = ? AND is_active = true
                          LIMIT 1`, req.TargetMasterTable).Scan(&masterID).Error; err != nil {
            return err
        }
        if masterID == 0 {
            return fmt.Errorf("master_table %q not found in master_binding", req.TargetMasterTable)
        }

        locatorJSON, _ := json.Marshal(req.SourceLocator)
        if err := tx.Raw(`
            INSERT INTO cdc_system.source_object_registry
                (object_code, source_engine_type, sync_engine, source_object_name,
                 source_locator_json, is_active, provisioning_state, notes,
                 created_at, updated_at)
            VALUES (?, ?, ?, ?, ?::jsonb, true, 'pending', ?, NOW(), NOW())
            ON CONFLICT (object_code) DO UPDATE SET
                source_locator_json = EXCLUDED.source_locator_json,
                provisioning_state  = 'pending',
                last_step_error     = NULL,
                updated_at          = NOW()
            RETURNING id`,
            req.ObjectCode, req.SourceEngineType, req.SyncEngine,
            req.SourceObjectName, string(locatorJSON), req.Notes).
            Scan(&sourceID).Error; err != nil {
            return err
        }

        // shadow_binding (idempotent on (source_object_id, shadow_schema, shadow_table))
        shadowSchema := shadowSchemaFor(req)
        shadowTable := req.SourceObjectName
        if err := tx.Exec(`
            INSERT INTO cdc_system.shadow_binding
                (source_object_id, shadow_schema, shadow_table, is_active, ddl_status, created_at, updated_at)
            VALUES (?, ?, ?, true, 'pending', NOW(), NOW())
            ON CONFLICT (source_object_id, shadow_schema, shadow_table) DO UPDATE SET
                is_active = true,
                ddl_status = 'pending',
                updated_at = NOW()`,
            sourceID, shadowSchema, shadowTable).Error; err != nil {
            return err
        }
        return nil
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "step1 (registry insert) failed: " + err.Error(),
        })
        return
    }
    stepsCompleted = append(stepsCompleted, "registry_insert")

    // === Step 2: Debezium include list extend ===
    if err := s.extendDebeziumInclude(c.Request.Context(), req); err != nil {
        s.markProvisioningFailed(sourceID, "step2_debezium", err)
        c.JSON(http.StatusMultiStatus, RegisterSourceResponse{
            SourceObjectID:    sourceID,
            ProvisioningState: "step2_failed",
            StepsCompleted:    stepsCompleted,
            LastStepError:     err.Error(),
        })
        return
    }
    stepsCompleted = append(stepsCompleted, "debezium_include_extend")

    // === Step 3: Schema Registry preempt compat=NONE ===
    if err := s.preemptSchemaRegistry(c.Request.Context(), req); err != nil {
        s.markProvisioningFailed(sourceID, "step3_schema_registry", err)
        c.JSON(http.StatusMultiStatus, RegisterSourceResponse{
            SourceObjectID:    sourceID,
            ProvisioningState: "step3_failed",
            StepsCompleted:    stepsCompleted,
            LastStepError:     err.Error(),
        })
        return
    }
    stepsCompleted = append(stepsCompleted, "schema_registry_preempt")

    // === Step 4: NATS signal worker reload ===
    if err := s.deps.NATS.Publish("cdc.cmd.kafka.refresh-topics", []byte("{}")); err != nil {
        s.deps.Logger.Warn("nats publish refresh-topics failed", zap.Error(err))
        // non-fatal
    }
    stepsCompleted = append(stepsCompleted, "worker_signal")

    // === Step 5: mark active ===
    s.deps.DB.Exec(`UPDATE cdc_system.source_object_registry
                    SET provisioning_state = 'active',
                        last_step_error    = NULL,
                        updated_at         = NOW()
                    WHERE id = ?`, sourceID)

    c.JSON(http.StatusOK, RegisterSourceResponse{
        SourceObjectID:    sourceID,
        ProvisioningState: "active",
        StepsCompleted:    stepsCompleted,
    })
}

func (s *Server) markProvisioningFailed(sourceID int64, step string, err error) {
    s.deps.DB.Exec(`UPDATE cdc_system.source_object_registry
                    SET provisioning_state = ?,
                        last_step_error    = ?,
                        updated_at         = NOW()
                    WHERE id = ?`, step, err.Error(), sourceID)
}
```

### File `internal/admin/helpers.go`

```go
package admin

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

func shadowSchemaFor(req RegisterSourceRequest) string {
    switch req.SourceEngineType {
    case "postgres":
        return "shadow_" + stringFromLocator(req.SourceLocator, "database")
    case "mongodb":
        return "shadow_" + stringFromLocator(req.SourceLocator, "database") + "_mongo"
    case "mariadb":
        return "shadow_" + stringFromLocator(req.SourceLocator, "database") + "_mariadb"
    }
    return "shadow_default"
}

func stringFromLocator(loc map[string]interface{}, key string) string {
    if v, ok := loc[key].(string); ok {
        return v
    }
    return ""
}

func qualifiedSourceObjectName(req RegisterSourceRequest) string {
    switch req.SourceEngineType {
    case "postgres":
        // public.orders / app_schema.users — use namespace from locator if present
        ns := stringFromLocator(req.SourceLocator, "schema")
        if ns == "" {
            ns = "public"
        }
        return ns + "." + stringFromLocator(req.SourceLocator, "table")
    case "mongodb":
        return stringFromLocator(req.SourceLocator, "database") + "." +
            stringFromLocator(req.SourceLocator, "collection")
    case "mariadb":
        return stringFromLocator(req.SourceLocator, "database") + "." +
            stringFromLocator(req.SourceLocator, "table")
    }
    return req.SourceObjectName
}

func includeKeyFor(engineType string) string {
    switch engineType {
    case "postgres":
        return "table.include.list"
    case "mongodb":
        return "collection.include.list"
    case "mariadb":
        return "table.include.list"
    }
    return "table.include.list"
}

func connectorNameFor(engineType string, locator map[string]interface{}) string {
    // Convention from existing operations:
    //   postgres: pg-source-<dbname>
    //   mongodb:  mongo-source
    //   mariadb:  mariadb-source-<dbname>
    db := stringFromLocator(locator, "database")
    switch engineType {
    case "postgres":
        return "pg-source-" + db
    case "mongodb":
        return "mongo-source"
    case "mariadb":
        return "mariadb-source-" + db
    }
    return ""
}

func topicNameFor(req RegisterSourceRequest) string {
    // Convention: cdc.<topic-prefix>.<source-id>.<object>
    // e.g. cdc.goopay.public.orders OR cdc.goopay.payment-bill-service.payment_bills
    // For exact resolution, fetch connector config first. Here we approximate:
    // cdc.<prefix>.<db>.<object>
    prefix := "goopay"
    db := stringFromLocator(req.SourceLocator, "database")
    obj := req.SourceObjectName
    return fmt.Sprintf("cdc.%s.%s.%s", prefix, db, obj)
}

func (s *Server) extendDebeziumInclude(ctx context.Context, req RegisterSourceRequest) error {
    connector := connectorNameFor(req.SourceEngineType, req.SourceLocator)
    if connector == "" {
        return fmt.Errorf("cannot derive connector name for engine %q", req.SourceEngineType)
    }
    url := fmt.Sprintf("%s/connectors/%s/config", s.deps.DebeziumBaseURL, connector)

    getReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := http.DefaultClient.Do(getReq)
    if err != nil {
        return fmt.Errorf("get connector config: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("get connector config %d: %s", resp.StatusCode, b)
    }
    var cfg map[string]string
    if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
        return err
    }

    key := includeKeyFor(req.SourceEngineType)
    qualified := qualifiedSourceObjectName(req)
    current := cfg[key]

    if containsCSV(current, qualified) {
        return nil
    }
    if current == "" {
        cfg[key] = qualified
    } else {
        cfg[key] = current + "," + qualified
    }

    body, _ := json.Marshal(cfg)
    putReq, _ := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
    putReq.Header.Set("Content-Type", "application/json")
    putResp, err := http.DefaultClient.Do(putReq)
    if err != nil {
        return fmt.Errorf("put connector config: %w", err)
    }
    defer putResp.Body.Close()
    if putResp.StatusCode >= 300 {
        b, _ := io.ReadAll(putResp.Body)
        return fmt.Errorf("put connector config %d: %s", putResp.StatusCode, b)
    }
    return nil
}

func (s *Server) preemptSchemaRegistry(ctx context.Context, req RegisterSourceRequest) error {
    topic := topicNameFor(req)
    subject := topic + "-value"
    url := fmt.Sprintf("%s/config/%s", s.deps.SchemaRegistryURL, subject)
    body := []byte(`{"compatibility":"NONE"}`)
    putReq, _ := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
    putReq.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
    resp, err := http.DefaultClient.Do(putReq)
    if err != nil {
        return fmt.Errorf("put compat: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode == 404 {
        // Subject chưa tồn tại — Schema Registry sẽ inherit từ global default.
        // Caller phải đảm bảo global compat=NONE; hoặc subject sẽ được tạo
        // sau khi message đầu xuất hiện (lúc đó global default applies).
        return nil
    }
    if resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("put compat %d: %s", resp.StatusCode, b)
    }
    return nil
}

func containsCSV(csv, needle string) bool {
    for _, p := range strings.Split(csv, ",") {
        if strings.TrimSpace(p) == needle {
            return true
        }
    }
    return false
}
```

### Test stub — `internal/admin/server_test.go`

```go
func TestRegisterSource_HappyPath(t *testing.T) {
    sqlDB, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
    defer sqlDB.Close()
    gormDB, _ := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})

    // Mock Debezium endpoint
    debezium := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "GET" {
            w.Write([]byte(`{"collection.include.list":"goopay.payment_bills"}`))
            return
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer debezium.Close()

    // Mock Schema Registry endpoint
    schemaReg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer schemaReg.Close()

    nc := mockNATS(t) // helper to spin up embedded NATS or use mock
    defer nc.Close()

    srv := NewServer(Deps{
        DB:                gormDB,
        NATS:              nc,
        DebeziumBaseURL:   debezium.URL,
        SchemaRegistryURL: schemaReg.URL,
        Logger:            zap.NewNop(),
    })

    // Mock SQL expectations
    mock.ExpectBegin()
    mock.ExpectQuery(`SELECT id FROM cdc_system.master_binding`).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
    mock.ExpectQuery(`INSERT INTO cdc_system.source_object_registry`).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))
    mock.ExpectExec(`INSERT INTO cdc_system.shadow_binding`).
        WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()
    mock.ExpectExec(`UPDATE cdc_system.source_object_registry`).
        WillReturnResult(sqlmock.NewResult(0, 1))

    body := `{
        "object_code":"smoke",
        "source_engine_type":"mongodb",
        "sync_engine":"debezium",
        "source_object_name":"smoke_collection",
        "source_locator":{"database":"goopay","collection":"smoke_collection"},
        "target_master_table":"payment_bills_addtest"
    }`
    req := httptest.NewRequest("POST", "/v2/sources/register", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    srv.engine.ServeHTTP(rec, req)
    require.Equal(t, http.StatusOK, rec.Code)
    var resp RegisterSourceResponse
    json.Unmarshal(rec.Body.Bytes(), &resp)
    require.Equal(t, "active", resp.ProvisioningState)
    require.Contains(t, resp.StepsCompleted, "registry_insert")
    require.Contains(t, resp.StepsCompleted, "debezium_include_extend")
    require.Contains(t, resp.StepsCompleted, "schema_registry_preempt")
    require.Contains(t, resp.StepsCompleted, "worker_signal")
}
```

### Smoke command (after build + boot)

```bash
# Boot admin-api locally (assume worker đã chạy với P0.1 rồi)
ADMIN_API_TOKEN=secret123 go run ./cmd/admin-api &

curl -X POST http://localhost:8090/v2/sources/register \
  -H "Authorization: Bearer secret123" \
  -H "Content-Type: application/json" \
  -d '{
    "object_code":"mongo_smoke_p02",
    "source_engine_type":"mongodb",
    "sync_engine":"debezium",
    "source_object_name":"smoke_p02",
    "source_locator":{"database":"goopay","collection":"smoke_p02"},
    "target_master_table":"payment_bills_addtest",
    "notes":"P0.2 smoke"
  }'
# Expect: 200, provisioning_state=active

# Trigger ingest
docker exec gpay-mongo-source mongosh goopay --eval \
  'db.smoke_p02.insertOne({_id:ObjectId(),amount:777,note:"P0.2-after-register"})'

# After 30s
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  -c "SELECT count(*) FROM shadow_goopay_mongo.smoke_p02;"
# Expect: 1
```

---

## Rollback procedures (per-task)

| Task | Rollback |
|------|----------|
| P1.1 | `git revert <commit>` → SQL behavior trở về UPDATE-only. Shadow rows đã upsert tombstone không cần undo (chỉ thêm row, không xóa data hợp lệ). |
| P0.1 | `git revert <commit>`. Worker sẽ trở lại restart-required mode. Không có data corruption risk. |
| P0.2 | `git revert <commit>` + xóa `cmd/admin-api`. KHÔNG ảnh hưởng worker. Source rows đã register vẫn còn — manual cleanup nếu cần. |

---

## Pre-flight check (CLAUDE.md §14)

- [x] Solution file vật lý đã ghi (file này).
- [x] Diff hint cụ thể cho từng task.
- [x] Test stub cho từng task.
- [x] Smoke command verify.
- [x] Rollback procedure.
- [ ] Brain ĐÃ KHÔNG sửa code (CLAUDE.md §12) — verified.
- [ ] Awaiting user approve để delegate Muscle.
