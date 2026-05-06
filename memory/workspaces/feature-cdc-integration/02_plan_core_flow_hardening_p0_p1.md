# 02 — Plan: Core-Flow Hardening Phase P0+P1

**Phase code**: `core_flow_hardening_p0_p1`
**Created**: 2026-05-04 13:55 (+07)
**Companion docs**: `01_requirements_core_flow_hardening_p0_p1.md`, `08_tasks_core_flow_hardening_p0_p1.md`, `09_tasks_solution_core_flow_hardening_p0_p1.md`.

---

## Strategic order (architect ruling)

1. **P1.1 (G3)** trước — blast radius nhỏ nhất (1 function delete branch), độc lập, dễ rollback.
2. **P0.1 (G1)** thứ 2 — refactor consume layer, blast radius trung (1 file kafka_consumer.go + 1 NATS subscription mới), không tạo cmd mới.
3. **P0.2 (G6)** cuối — blast radius lớn (cmd mới + package mới + endpoint HTTP + rollback compensation), phụ thuộc P0.1 (cần subject NATS để publish refresh).

Reasoning: small → big để mỗi bước land độc lập, smoke test riêng, không bị nghẽn.

---

## P1.1 — Delete tombstone-first (G3)

### Critical files

- `internal/handler/event_handler.go:145-184` — `handleDelete` function body.
- `internal/handler/event_handler.go:175-181` — UPDATE statement to replace.
- `internal/handler/event_handler_test.go` — pattern integration test cho delete branch (nếu chưa có thì tạo).

### Approach

Đổi từ:
```sql
UPDATE shadow.tbl SET _deleted=TRUE, _updated_at=NOW() WHERE pk=?
```
sang INSERT…ON CONFLICT pattern. Tham khảo schema_adapter.go::BuildUpsertSQLInSchema để dùng đúng column set V1 (`_gpay_source_id`, `_deleted`, `_created_at`, `_updated_at`, `_source`).

```go
sql := fmt.Sprintf(`INSERT INTO %s (%s, _gpay_source_id, _deleted, _created_at, _updated_at, _source)
VALUES (?, ?::text, TRUE, NOW(), NOW(), 'debezium')
ON CONFLICT (%s) DO UPDATE SET _deleted=TRUE, _updated_at=NOW()`,
    qualifiedShadowTable(route),
    quoteEventIdent(pgPKField),
    quoteEventIdent(pgPKField),
)
db.Exec(sql, pkValue, pkValue)
```

**Lưu ý**:
- Pass `pkValue` 2 lần (1 cho PK column, 1 cho `_gpay_source_id`) vì `_gpay_source_id` luôn TEXT — `?::text` cast.
- Conflict target = PK column (đã có UNIQUE constraint từ schema_adapter `PrepareForCDCInsertInSchema`).
- KHÔNG set `_raw_data`, `_hash` (delete event không có data); những column này sẽ giữ NULL nếu first-touch, hoặc giữ giá trị cũ nếu UPDATE branch.

### Verification

1. `go build ./...` PASS.
2. Existing test `event_handler_test.go::TestExtractSourceAndTable` (nếu có) vẫn pass.
3. Unit test mới `TestHandleDelete_FirstTouch_TombstoneInsert`:
   - Setup mock route + mock DB.
   - Gọi `handleDelete` với event chỉ có `before.id=999` (không có shadow row trước).
   - Assert SQL contains `INSERT INTO ... ON CONFLICT`.
4. Smoke live:
   - DELETE row id=64 trên Postgres source (vốn vừa được B11 verify INSERT).
   - Trong 5s, shadow row id=64 phải có `_deleted=true`, `_gpay_source_id='64'`, `_updated_at` mới.
   - Master next cron tick (≤60s) phải reflect tombstone (assume master query `WHERE _deleted=FALSE`).

---

## P0.1 — Reader manager với NATS-driven refresh (G1)

### Critical files

- `internal/handler/kafka_consumer.go` — toàn bộ struct `KafkaConsumer`. Refactor:
  - `kc.readers []*kafka.Reader` (line 153) → giữ nguyên struct slice, nhưng thêm mutex + `topics []string` cached.
  - `Start(ctx)` (line 99-242) — tách phần "build reader" thành method `buildReader(ctx, topics)` để reuse.
  - Thêm method `RefreshTopics(ctx) error` để re-discover, so sánh, recreate nếu khác.
- `internal/server/worker_server.go:248-261` — thêm subscribe `cdc.cmd.kafka.refresh-topics`.

### Approach

#### Bước A — refactor `KafkaConsumer` struct

Thêm fields:
```go
type KafkaConsumer struct {
    // existing...
    mu              sync.Mutex
    currentTopics   []string  // cached topic set của reader hiện tại
    readerCtx       context.Context
    readerCancel    context.CancelFunc
}
```

#### Bước B — tách `buildReader`

```go
func (kc *KafkaConsumer) buildReader(topics []string) *kafka.Reader {
    return kafka.NewReader(kafka.ReaderConfig{
        Brokers:          kc.config.Brokers,
        GroupID:          kc.config.GroupID,
        GroupTopics:      topics,
        // ... existing options
    })
}
```

#### Bước C — `RefreshTopics` method

```go
// RefreshTopics re-discovers Kafka topics and recreates the reader if
// the topic set has changed. Idempotent — no-op if topics unchanged.
// Caller must hold no lock; method takes kc.mu internally.
func (kc *KafkaConsumer) RefreshTopics(ctx context.Context) error {
    newTopics, err := kc.discoverTopics(ctx)
    if err != nil {
        return fmt.Errorf("discover topics: %w", err)
    }

    kc.mu.Lock()
    defer kc.mu.Unlock()

    if topicSetEqual(kc.currentTopics, newTopics) {
        kc.logger.Debug("topic set unchanged, skipping refresh",
            zap.Int("count", len(newTopics)))
        return nil
    }

    kc.logger.Info("topic set changed, recreating reader",
        zap.Strings("old", kc.currentTopics),
        zap.Strings("new", newTopics))

    // Flush in-flight batches BEFORE closing reader (avoid drop)
    kc.flushAllBatches()

    // Close old reader (commits last offsets via kafka-go internal)
    for _, r := range kc.readers {
        if err := r.Close(); err != nil {
            kc.logger.Warn("close old reader", zap.Error(err))
        }
    }
    kc.readers = nil

    // Build new reader with new topic set
    newReader := kc.buildReader(newTopics)
    kc.readers = append(kc.readers, newReader)
    kc.currentTopics = newTopics

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

#### Bước D — Update `Start` để consume loop dùng dynamic reader

Đổi line 174 `reader.FetchMessage(ctx)` thành snapshot reader hiện tại từ kc:

```go
for {
    select {
    case <-ctx.Done():
        kc.flushAllBatches()
        kc.Stop()
        return
    case <-flushTicker.C:
        kc.flushAllBatches()
    case <-refreshTicker.C:
        // Safety net — auto re-discover every 60s.
        if err := kc.RefreshTopics(ctx); err != nil {
            kc.logger.Warn("auto refresh failed", zap.Error(err))
        }
    default:
        kc.mu.Lock()
        if len(kc.readers) == 0 {
            kc.mu.Unlock()
            time.Sleep(100 * time.Millisecond)
            continue
        }
        reader := kc.readers[0]
        kc.mu.Unlock()

        msg, err := reader.FetchMessage(ctx)
        // ... existing processing
    }
}
```

Thêm `refreshTicker := time.NewTicker(60 * time.Second)` + defer Stop. Init `kc.currentTopics = topics` sau initial build.

#### Bước E — NATS subscribe

`worker_server.go` add (sau line 261):
```go
// Core-Flow Hardening P0.1 (G1) — kafka topic dynamic refresh.
natsClient.Conn.Subscribe("cdc.cmd.kafka.refresh-topics", func(msg *nats.Msg) {
    if err := kafkaConsumer.RefreshTopics(context.Background()); err != nil {
        logger.Warn("nats-triggered topic refresh failed", zap.Error(err))
        return
    }
    logger.Info("nats-triggered topic refresh ok")
})
```

(Cần expose `kafkaConsumer` reference ở worker_server boot — verify line where created.)

### Verification

1. `go build ./...` PASS.
2. Unit test `TestRefreshTopics_NoChange` + `TestRefreshTopics_AddTopic`.
3. Smoke live (full E2E):
   - Start worker fresh.
   - Add Mongo collection mới (pre-staged dữ liệu test).
   - PUT include list extend.
   - `nats pub cdc.cmd.kafka.refresh-topics ""`.
   - INSERT 1 doc → shadow landed trong 10s.
   - Worker logs có dòng "topic set changed, recreating reader".

### Edge cases

- **EC-1**: Refresh đang chạy (lock acquired) khi consume loop đang `FetchMessage` → consume loop sẽ Block ở `kc.mu.Lock()` ở next iteration sau khi msg hiện tại xử lý xong. Acceptable vì chỉ 1 reader, không có race.
- **EC-2**: Discover trả 0 topic mới sau đã có topic cũ → `topicSetEqual` return false → recreate với 0 topic → reader sẽ không fetch gì. Đúng behavior.
- **EC-3**: Discover lỗi (Kafka mất kết nối) → return error, không recreate. Reader cũ tiếp tục chạy. Acceptable.

---

## P0.2 — cdc-admin-api transactional registration (G6)

### New files

- `cmd/admin-api/main.go` — HTTP server entry point.
- `internal/admin/server.go` — Gin/echo router setup.
- `internal/admin/source_register.go` — `POST /v2/sources/register` handler.
- `internal/admin/types.go` — request/response DTOs.
- `internal/admin/server_test.go` — handler tests with mock DB / mock HTTP clients.

### Approach

#### File `cmd/admin-api/main.go`

```go
package main

import (
    "context"
    "log"
    "os"

    "centralized-data-service/internal/admin"
    "centralized-data-service/internal/config"
    // ...
)

func main() {
    cfg := config.MustLoad()
    db := mustOpenDB(cfg.MasterDB.Default)
    natsConn := mustConnectNATS(cfg.NATS)
    logger := mustBuildLogger()

    srv := admin.NewServer(admin.Deps{
        DB:               db,
        NATS:             natsConn,
        DebeziumBaseURL:  cfg.Debezium.URL,
        SchemaRegistryURL: cfg.SchemaRegistry.URL,
        AuthToken:        os.Getenv("ADMIN_API_TOKEN"),
        Logger:           logger,
    })

    addr := cfg.AdminAPI.ListenAddr // default :8090
    log.Printf("admin-api listening on %s", addr)
    if err := srv.Run(context.Background(), addr); err != nil {
        log.Fatalf("admin-api crashed: %v", err)
    }
}
```

#### File `internal/admin/types.go`

```go
package admin

type RegisterSourceRequest struct {
    ObjectCode        string          `json:"object_code"          binding:"required"`
    SourceEngineType  string          `json:"source_engine_type"   binding:"required"` // postgres|mongodb|mariadb
    SyncEngine        string          `json:"sync_engine"          binding:"required"` // debezium
    SourceObjectName  string          `json:"source_object_name"   binding:"required"`
    SourceLocator     map[string]any  `json:"source_locator"       binding:"required"`
    TargetMasterTable string          `json:"target_master_table"  binding:"required"`
    Notes             string          `json:"notes"`
}

type RegisterSourceResponse struct {
    SourceObjectID    int64    `json:"source_object_id"`
    ProvisioningState string   `json:"provisioning_state"`
    StepsCompleted    []string `json:"steps_completed"`
    LastStepError     string   `json:"last_step_error,omitempty"`
}
```

#### File `internal/admin/source_register.go`

```go
func (s *Server) handleRegisterSource(c *gin.Context) {
    var req RegisterSourceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // === Step 1: DB transaction (idempotent insert) ===
    var sourceID int64
    err := s.db.Transaction(func(tx *gorm.DB) error {
        // Resolve master_binding_id (must exist)
        var masterID int64
        if err := tx.Raw(`SELECT id FROM cdc_system.master_binding
                          WHERE master_table = ? AND is_active = true
                          LIMIT 1`, req.TargetMasterTable).Scan(&masterID).Error; err != nil {
            return err
        }
        if masterID == 0 {
            return fmt.Errorf("master_table %q not found in master_binding", req.TargetMasterTable)
        }

        // Insert source_object_registry (idempotent on object_code)
        locatorJSON, _ := json.Marshal(req.SourceLocator)
        if err := tx.Exec(`
            INSERT INTO cdc_system.source_object_registry
                (object_code, source_engine_type, sync_engine, source_object_name,
                 source_locator_json, is_active, provisioning_state, notes,
                 created_at, updated_at)
            VALUES (?, ?, ?, ?, ?::jsonb, true, 'pending', ?, NOW(), NOW())
            ON CONFLICT (object_code) DO UPDATE SET
                source_locator_json = EXCLUDED.source_locator_json,
                provisioning_state = 'pending',
                last_step_error = NULL,
                updated_at = NOW()
            RETURNING id`,
            req.ObjectCode, req.SourceEngineType, req.SyncEngine,
            req.SourceObjectName, string(locatorJSON), req.Notes).
            Scan(&sourceID).Error; err != nil {
            return err
        }

        // Insert shadow_binding (FK to source_object_registry)
        // Schema name convention: shadow_<source_db>
        // Table name: req.SourceObjectName
        // ... (similar idempotent INSERT … ON CONFLICT)
        return nil
    })
    if err != nil {
        c.JSON(500, gin.H{"error": "step 1 (registry insert) failed: " + err.Error()})
        return
    }

    stepsCompleted := []string{"registry_insert"}

    // === Step 2: Debezium include list extend ===
    if err := s.extendDebeziumInclude(c.Request.Context(), req); err != nil {
        s.markProvisioningFailed(sourceID, "step2_debezium", err)
        c.JSON(207, RegisterSourceResponse{
            SourceObjectID:    sourceID,
            ProvisioningState: "step2_failed",
            StepsCompleted:    stepsCompleted,
            LastStepError:     err.Error(),
        })
        return
    }
    stepsCompleted = append(stepsCompleted, "debezium_include_extend")

    // === Step 3: Schema Registry compat=NONE preempt ===
    if err := s.preemptSchemaRegistry(c.Request.Context(), req); err != nil {
        s.markProvisioningFailed(sourceID, "step3_schema_registry", err)
        c.JSON(207, RegisterSourceResponse{
            SourceObjectID:    sourceID,
            ProvisioningState: "step3_failed",
            StepsCompleted:    stepsCompleted,
            LastStepError:     err.Error(),
        })
        return
    }
    stepsCompleted = append(stepsCompleted, "schema_registry_preempt")

    // === Step 4: Trigger worker reload via NATS ===
    if err := s.nats.Publish("cdc.cmd.kafka.refresh-topics", []byte("{}")); err != nil {
        s.logger.Warn("nats publish refresh-topics failed", zap.Error(err))
        // Non-fatal: worker safety-net ticker sẽ pick up trong 60s.
    }
    stepsCompleted = append(stepsCompleted, "worker_signal")

    // === Step 5: Mark provisioning_state = active ===
    s.db.Exec(`UPDATE cdc_system.source_object_registry
               SET provisioning_state = 'active', last_step_error = NULL,
                   updated_at = NOW()
               WHERE id = ?`, sourceID)

    c.JSON(200, RegisterSourceResponse{
        SourceObjectID:    sourceID,
        ProvisioningState: "active",
        StepsCompleted:    stepsCompleted,
    })
}
```

#### Helper `extendDebeziumInclude`

```go
func (s *Server) extendDebeziumInclude(ctx context.Context, req RegisterSourceRequest) error {
    // 1. GET current connector config
    connector := connectorNameFor(req.SourceEngineType, req.SourceLocator)
    url := fmt.Sprintf("%s/connectors/%s/config", s.debeziumBaseURL, connector)
    resp, err := http.Get(url)
    if err != nil { return err }
    var cfg map[string]string
    json.NewDecoder(resp.Body).Decode(&cfg)
    resp.Body.Close()

    // 2. Compute new include list = old + req.SourceObjectName (deduplicated)
    includeKey := includeKeyFor(req.SourceEngineType) // table.include.list, collection.include.list, ...
    current := cfg[includeKey]
    qualified := qualifiedSourceObjectName(req) // e.g. "public.orders" or "goopay.payment_bills"
    if strings.Contains(current, qualified) {
        return nil // already in list — idempotent
    }
    cfg[includeKey] = current + "," + qualified

    // 3. PUT updated config
    body, _ := json.Marshal(cfg)
    putReq, _ := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
    putReq.Header.Set("Content-Type", "application/json")
    putResp, err := http.DefaultClient.Do(putReq)
    if err != nil { return err }
    defer putResp.Body.Close()
    if putResp.StatusCode >= 300 {
        b, _ := io.ReadAll(putResp.Body)
        return fmt.Errorf("debezium PUT returned %d: %s", putResp.StatusCode, b)
    }
    return nil
}
```

#### Helper `preemptSchemaRegistry`

```go
func (s *Server) preemptSchemaRegistry(ctx context.Context, req RegisterSourceRequest) error {
    topic := topicNameFor(req)
    subject := topic + "-value"
    url := fmt.Sprintf("%s/config/%s", s.schemaRegistryURL, subject)
    body := []byte(`{"compatibility":"NONE"}`)
    putReq, _ := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
    putReq.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
    resp, err := http.DefaultClient.Do(putReq)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("schema registry PUT returned %d: %s", resp.StatusCode, b)
    }
    return nil
}
```

### Verification

1. `go build ./cmd/admin-api ./...` PASS.
2. Unit tests:
   - Mock DB → `handleRegisterSource` step 1 idempotent.
   - Mock HTTP server cho Debezium + Schema Registry → step 2/3 happy path + 207 partial.
3. Smoke E2E:
   - Boot admin-api side-by-side với worker.
   - `curl -X POST :8090/v2/sources/register -H 'Content-Type: application/json' -d '{...}'` (collection mới `payment_bills_addtest_v3`).
   - Assert response 200, provisioning_state=active.
   - INSERT 1 doc nguồn → shadow row landed trong 30s.
   - Master cron tick (≤60s) reflect.

### Edge cases

- **EC-1**: Connector chưa tồn tại → step 2 GET trả 404 → fail rõ ràng. Không tự auto-create connector (out-of-scope).
- **EC-2**: Schema Registry chưa có subject (topic chưa có message) → PUT compat=NONE trả 404. Acceptable: skip với log INFO, vì khi message đầu xuất hiện connector sẽ register subject với default global compatibility (cần ensure global = NONE pre-set, hoặc step 3 fail soft).
- **EC-3**: Worker đang restart khi NATS publish → message lost. Mitigation: safety-net ticker 60s ở P0.1 sẽ pick up.
- **EC-4**: Duplicate object_code → ON CONFLICT DO UPDATE → idempotent.

### Security gate

- Auth: `Authorization: Bearer <ADMIN_API_TOKEN>` (env var). Reject 401 nếu token sai.
- Listen address default `127.0.0.1:8090` (loopback only). Public expose phải qua reverse proxy.
- KHÔNG log payload chứa connection credentials.

---

## End-to-end smoke after all 3 land

```bash
# 1. Boot worker + admin-api
docker-compose up -d cdc-worker cdc-admin-api

# 2. Register a brand new Mongo collection
curl -X POST http://localhost:8090/v2/sources/register \
  -H "Authorization: Bearer $ADMIN_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "object_code": "mongo_payment_bills_v3_smoke",
    "source_engine_type": "mongodb",
    "sync_engine": "debezium",
    "source_object_name": "payment_bills_smoke_v3",
    "source_locator": {"database":"goopay","collection":"payment_bills_smoke_v3"},
    "target_master_table": "payment_bills_addtest",
    "notes": "P0.2 smoke"
  }'

# Expect: 200, provisioning_state=active, steps_completed=[registry_insert, debezium_include_extend, schema_registry_preempt, worker_signal]

# 3. INSERT 1 doc nguồn
docker exec gpay-mongo-source mongosh goopay --eval \
  'db.payment_bills_smoke_v3.insertOne({_id:ObjectId(),amount:1000,note:"P0.2-smoke"})'

# 4. Verify shadow trong 30s
sleep 30
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  -c "SELECT count(*) FROM shadow_goopay_mongo.payment_bills_smoke_v3;"
# Expect: 1

# 5. Verify master sau cron tick
sleep 60
docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest \
  -c "SELECT count(*) FROM dw_payments.payment_bills_addtest WHERE _gpay_source_id LIKE 'P0.2%' OR ...;"

# 6. Test P1.1 delete branch
docker exec gpay-mongo-source mongosh goopay --eval \
  'db.payment_bills_smoke_v3.deleteOne({note:"P0.2-smoke"})'
sleep 10
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  -c "SELECT id, _deleted, _gpay_source_id FROM shadow_goopay_mongo.payment_bills_smoke_v3;"
# Expect: _deleted=true, _gpay_source_id NOT NULL

# 7. Test P0.1 worker stays up — không cần restart
docker logs cdc-worker --tail 50 | grep -i "topic set changed"
# Expect: at least 1 line "topic set changed, recreating reader"
```

---

## Files to be modified or created (full list)

| Path | Phase | Action | Owner |
|------|-------|--------|-------|
| `internal/handler/event_handler.go` | P1.1 | Edit `handleDelete` UPSERT | Muscle |
| `internal/handler/event_handler_test.go` | P1.1 | New `TestHandleDelete_FirstTouch_TombstoneInsert` | Muscle |
| `internal/handler/kafka_consumer.go` | P0.1 | Refactor `Start`, add `RefreshTopics`, `buildReader`, mutex, ticker | Muscle |
| `internal/handler/kafka_consumer_test.go` | P0.1 | New `TestRefreshTopics_NoChange`, `TestRefreshTopics_AddTopic` | Muscle |
| `internal/server/worker_server.go` | P0.1 | Add NATS subscribe `cdc.cmd.kafka.refresh-topics` | Muscle |
| `cmd/admin-api/main.go` | P0.2 | NEW — entry point | Muscle |
| `internal/admin/server.go` | P0.2 | NEW — router setup | Muscle |
| `internal/admin/types.go` | P0.2 | NEW — DTOs | Muscle |
| `internal/admin/source_register.go` | P0.2 | NEW — handler + helpers | Muscle |
| `internal/admin/server_test.go` | P0.2 | NEW — handler tests | Muscle |
| `internal/config/config.go` | P0.2 | Add `AdminAPI.ListenAddr` + `Debezium.URL` + `SchemaRegistry.URL` if absent | Muscle |
| `agent/memory/workspaces/feature-cdc-integration/05_progress.md` | All | APPEND closure entries | Brain |
| `agent/memory/global/lessons.md` | All | APPEND Global Pattern lessons | Brain |

---

## Pre-flight check (CLAUDE.md §14)

- [x] Plan file vật lý đã ghi.
- [x] Requirements file vật lý đã ghi.
- [ ] Tasks file (08_tasks_*) — sẽ ghi tiếp ngay.
- [ ] Solution file (09_tasks_solution_*) — sẽ ghi tiếp ngay.
- [ ] User approve trước khi delegate Muscle.
