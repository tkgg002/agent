# 03_implementation_phase_p1 — Chi tiết kỹ thuật

## G-5 — Restart smoke test failover

### File NEW: `centralized-data-service/scripts/smoke_failover.sh`
```bash
#!/usr/bin/env bash
set -euo pipefail

# Pre-condition checks
PG="psql -h ${PG_HOST:-localhost} -p ${PG_PORT:-5433} -U ${PG_USER:-cdc} -d ${PG_DB:-cdc_metadata}"
MONGO_URI="${MONGO_URI:-mongodb://localhost:17017/test}"
TABLE="${SHADOW_TABLE:-cdc_internal.shadow_test_users}"
COUNT=${COUNT:-10000}

echo "=== Smoke Failover Test ==="
INITIAL=$($PG -At -c "SELECT COUNT(*) FROM $TABLE;")
echo "Initial rows: $INITIAL"

# Burst insert at source
echo "Inserting $COUNT docs into Mongo..."
mongosh --quiet "$MONGO_URI" --eval "
  const batch = Array.from({length: $COUNT}, (_, i) => ({_id: 'test_' + (Date.now() + i), v: Math.random()}));
  db.users.insertMany(batch);
"

# Wait for first 30% to propagate
sleep 5
PARTIAL=$($PG -At -c "SELECT COUNT(*) FROM $TABLE;")
echo "After 5s wait: $PARTIAL"

# KILL worker
WORKER_PID=$(pgrep -f cdc-worker || echo "")
[ -z "$WORKER_PID" ] && { echo "Worker not running"; exit 1; }
echo "Killing worker PID $WORKER_PID"
kill -9 $WORKER_PID
sleep 10

# Restart
echo "Restarting worker..."
nohup ./bin/worker > /tmp/worker_smoke.log 2>&1 &
NEW_PID=$!
sleep 30

# Verify
AFTER=$($PG -At -c "SELECT COUNT(*) FROM $TABLE;")
EXPECTED=$((INITIAL + COUNT))
echo "After restart: $AFTER (expected $EXPECTED)"

# Check duplicates
DUP=$($PG -At -c "SELECT COUNT(*) FROM (SELECT _gpay_source_id, COUNT(*) FROM $TABLE GROUP BY 1 HAVING COUNT(*) > 1) t;")
echo "Duplicates: $DUP"

[ "$AFTER" -eq "$EXPECTED" ] || { echo "FAIL: data loss"; exit 1; }
[ "$DUP" -eq 0 ] || { echo "FAIL: duplicates"; exit 1; }
echo "PASS: zero loss + zero duplicate"
```

### CI integration (`.github/workflows/smoke-failover.yml` NEW)
```yaml
name: smoke-failover
on: [pull_request, workflow_dispatch]
jobs:
  failover:
    runs-on: ubuntu-latest
    services:
      postgres: { image: postgres:15, ports: [5433:5432], env: { POSTGRES_PASSWORD: pass } }
      mongo:    { image: mongo:7,    ports: [17017:27017] }
      kafka:    { image: confluentinc/cp-kafka:7.5.0, ports: [9092:9092] }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26.1' }
      - run: make build
      - run: ./scripts/smoke_failover.sh
```

---

## G-6 — WAL slot expire alert

### File NEW: `deployments/prometheus/alerts/wal_slot.yml`
```yaml
groups:
- name: pg-replication
  rules:
  - alert: ReplicationSlotLagHigh
    expr: pg_replication_slot_pg_xlog_location_diff > 1073741824  # 1GB
    for: 10m
    labels: { severity: warning, team: cdc }
    annotations:
      summary: "Slot {{ $labels.slot_name }} lag > 1GB"
      runbook: "docs/runbooks/wal-slot-expire.md"
      description: "WAL slot accumulating. CDC consumer may be slow or stopped."

  - alert: ReplicationSlotInactive
    expr: pg_replication_slot_active == 0
    for: 5m
    labels: { severity: critical, team: cdc }
    annotations:
      summary: "Slot {{ $labels.slot_name }} INACTIVE — slot may be dropped if WAL exhausted"
      runbook: "docs/runbooks/wal-slot-expire.md"
```

### Postgres exporter deploy (file NEW `deployments/postgres-exporter.yml`)
```yaml
version: '3.8'
services:
  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:v0.15.0
    environment:
      DATA_SOURCE_NAME: "postgresql://exporter:${PG_EXPORTER_PASS}@postgres-source:5432/postgres?sslmode=disable"
      PG_EXPORTER_INCLUDE_DATABASES: "postgres"
    ports: ["9187:9187"]
```

### File NEW: `docs/runbooks/wal-slot-expire.md`
```markdown
# Runbook: WAL Slot Expire

## Symptom
Alert `ReplicationSlotLagHigh` hoặc `ReplicationSlotInactive`.

## Diagnose
1. `psql -c "SELECT slot_name, active, restart_lsn, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS lag FROM pg_replication_slots;"`
2. Identify slot: `cdc_gpay_pg_source`.

## Resolve
- Nếu slot active nhưng lag tăng → check connector status `curl http://kafka-connect:8083/connectors/pg-source/status`.
- Nếu connector RUNNING nhưng slow → scale up worker (HPA).
- Nếu connector FAILED → restart `curl -X POST http://kafka-connect:8083/connectors/pg-source/restart`.
- Nếu slot inactive + lag > 10GB → drop slot + re-snapshot:
  ```sql
  SELECT pg_drop_replication_slot('cdc_gpay_pg_source');
  ```
  Sau đó re-register connector với `snapshot.mode: initial`.
```

---

## G-7 — pprof endpoint + goleak verify

### File: `centralized-data-service/cmd/worker/main.go`
```go
import (
    _ "net/http/pprof" // pprof handlers register vào http.DefaultServeMux
    "net/http"
)

// THÊM trong main() sau khi config loaded:
if cfg.Debug.PprofEnabled {
    go func() {
        addr := cfg.Debug.PprofAddr
        if addr == "" { addr = "localhost:6060" }
        logger.Info("pprof endpoint", zap.String("addr", addr))
        if err := http.ListenAndServe(addr, nil); err != nil {
            logger.Error("pprof ListenAndServe", zap.Error(err))
        }
    }()
}
```

### Config knob `config.go`
```go
type DebugConfig struct {
    PprofEnabled bool   `mapstructure:"pprofEnabled"`
    PprofAddr    string `mapstructure:"pprofAddr"`
}
```

### `config-local.yml` thêm
```yaml
debug:
  pprofEnabled: true
  pprofAddr: localhost:6060
```

### goleak in tests — `internal/handler/main_test.go` NEW
```go
package handler

import (
    "testing"
    "go.uber.org/goleak"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m,
        goleak.IgnoreTopFunction("github.com/segmentio/kafka-go.(*Reader).run"),
        goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
    )
}
```

Tương tự thêm `TestMain` vào:
- `internal/service/main_test.go`
- `internal/sinkworker/main_test.go`

### Verify
- `curl localhost:6060/debug/pprof/heap` → binary profile download.
- `go test ./internal/handler/...` → exit 0 + log "no leak found".

---

## G-8 — Event Ordering test

### File NEW: `internal/service/schema_adapter_ordering_test.go`
```go
package service_test

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/require"
    "gorm.io/gorm"
    // ...
)

// Scenario: Insert→Update1→Update2 với Update1 đến SAU Update2 (out-of-order)
// Expected: row giữ value của Update2 (newer ts), Update1 bị OCC reject.
func TestEventOrdering_OlderTsIgnored(t *testing.T) {
    db := setupTestDB(t)
    adapter := service.NewSchemaAdapter(db, ...)

    sourceID := "test_user_1"
    // Step 1: Insert ts=1000
    insertEvent := makeShadowEvent(sourceID, map[string]any{"name": "alice"}, 1000)
    _, err := adapter.UpsertShadow(context.Background(), "test_users", insertEvent)
    require.NoError(t, err)
    require.Equal(t, "alice", readShadowValue(t, db, sourceID, "name"))

    // Step 2: Update ts=3000 (newer)
    update2 := makeShadowEvent(sourceID, map[string]any{"name": "carol"}, 3000)
    rows2, err := adapter.UpsertShadow(context.Background(), "test_users", update2)
    require.NoError(t, err)
    require.EqualValues(t, 1, rows2)
    require.Equal(t, "carol", readShadowValue(t, db, sourceID, "name"))

    // Step 3: Update ts=2000 (OLDER than step 2 — out-of-order replay)
    update1 := makeShadowEvent(sourceID, map[string]any{"name": "bob"}, 2000)
    rows1, err := adapter.UpsertShadow(context.Background(), "test_users", update1)
    require.NoError(t, err)
    require.EqualValues(t, 0, rows1, "OCC should reject older ts")
    require.Equal(t, "carol", readShadowValue(t, db, sourceID, "name"), "value preserved (carol)")

    // Step 4: Delete ts=4000
    delete := makeShadowDelete(sourceID, 4000)
    rowsD, err := adapter.UpsertShadow(context.Background(), "test_users", delete)
    require.NoError(t, err)
    require.EqualValues(t, 1, rowsD)
    require.True(t, readShadowDeleted(t, db, sourceID))
}

func TestEventOrdering_HashTiebreaker(t *testing.T) {
    // Same ts, different hash → second event SHOULD update (idempotent replay)
    db := setupTestDB(t)
    adapter := service.NewSchemaAdapter(db, ...)
    e1 := makeShadowEventWithHash("id1", map[string]any{"v": 1}, 1000, "hash_a")
    _, err := adapter.UpsertShadow(context.Background(), "t", e1)
    require.NoError(t, err)
    e2 := makeShadowEventWithHash("id1", map[string]any{"v": 2}, 1000, "hash_b")
    rows, err := adapter.UpsertShadow(context.Background(), "t", e2)
    require.NoError(t, err)
    require.EqualValues(t, 1, rows, "hash tiebreaker should allow update")
}
```

### Verify
- `go test ./internal/service/ -run TestEventOrdering -v -count=1` → 2 test PASS.

---

## G-9 — Schema Drift approve E2E test

### File NEW: `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go`
```go
//go:build integration
// +build integration

package commands_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/nats"
    // ...
)

func TestApproveSchemaProposal_E2E(t *testing.T) {
    ctx := context.Background()

    // 1. Spin up PG + NATS testcontainers
    pgC, err := postgres.Run(ctx, "postgres:15", postgres.WithDatabase("cdc"), postgres.WithUsername("u"), postgres.WithPassword("p"))
    require.NoError(t, err)
    defer pgC.Terminate(ctx)

    natsC, err := nats.Run(ctx, "nats:2.10")
    require.NoError(t, err)
    defer natsC.Terminate(ctx)

    // 2. Migrate schema
    db := connectGORM(t, pgC)
    runMigrations(t, db)

    // 3. Insert pending schema proposal
    proposal := &model.SchemaProposal{
        SourceTable: "users", FieldName: "new_col", DataType: "TEXT",
        Status: "pending",
    }
    db.Create(proposal)

    // 4. Subscribe NATS `cdc.evt.schema.approved`
    nc, _ := nats.Connect(natsC.URI(ctx))
    msgs := make(chan *nats.Msg, 1)
    nc.ChanSubscribe("cdc.evt.schema.approved", msgs)

    // 5. Execute command
    cmd := commands.ApproveSchemaProposalCommand{ID: proposal.ID, ApprovedBy: "test-admin"}
    handler := commands.NewApproveSchemaProposalHandler(db, nc, logger)
    err = handler.Handle(ctx, cmd)
    require.NoError(t, err)

    // 6. Verify shadow table ALTERED
    var col string
    db.Raw("SELECT column_name FROM information_schema.columns WHERE table_name='shadow_users' AND column_name='new_col'").Scan(&col)
    require.Equal(t, "new_col", col)

    // 7. Verify mapping rule inserted
    var rule model.MappingRule
    db.Where("source_table = ? AND target_column = ?", "users", "new_col").First(&rule)
    require.Equal(t, "approved", rule.Status)

    // 8. Verify NATS event published
    select {
    case msg := <-msgs:
        require.Contains(t, string(msg.Data), `"field_name":"new_col"`)
    case <-time.After(5 * time.Second):
        t.Fatal("schema.approved event not published")
    }
}
```

### Build tag run
- `go test -tags integration ./cdc-cms-service/internal/app/commands/... -v -count=1`

---

## Composite score change (P1 done)
- G-5 → 2.1 Failover L2 → L3 (+1).
- G-6 → 2.3 LSN Expire L1 → L3 (+2).
- G-7 → 4.1 Memory Leak L1 → L3 (+2).
- G-8 → 1.3 Event Ordering L2 → L3 (+1).
- G-9 → 1.2 Schema Drift L3 → L4 (+1).

**Sau P0+P1**: 44 + 7 = 51/64 ≈ 79.7%.
