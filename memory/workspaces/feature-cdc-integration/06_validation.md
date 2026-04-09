# Validation & Testing Plan: CDC Integration

> **Workspace**: feature-cdc-integration
> **Purpose**: Define comprehensive testing strategy for CDC implementation

---

## 1. Unit Testing

### 1.1 CDC Worker Components

**Test Coverage Target**: > 80%

#### Event Parser Tests (`internal/application/handlers/*_test.go`)
```go
func TestWalletTxHandler_MapToEntity(t *testing.T) {
    tests := []struct {
        name    string
        event   *dto.CDCEvent
        want    *entities.WalletTransaction
        wantErr bool
    }{
        {
            name: "Valid INSERT event",
            event: &dto.CDCEvent{
                Payload: dto.Payload{
                    Op: "c",
                    After: map[string]interface{}{
                        "id": "tx-123",
                        "user_id": "user-456",
                        "amount": 100000.0,
                        // ...
                    },
                },
            },
            want: &entities.WalletTransaction{
                ID: "tx-123",
                UserID: "user-456",
                Amount: 100000.0,
            },
            wantErr: false,
        },
        {
            name: "Invalid event - missing required field",
            event: &dto.CDCEvent{
                Payload: dto.Payload{
                    Op: "c",
                    After: map[string]interface{}{
                        "id": "tx-123",
                        // Missing user_id
                    },
                },
            },
            want: nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := NewWalletTxHandler(...)
            got, err := handler.mapToEntity(tt.event)

            if (err != nil) != tt.wantErr {
                t.Errorf("mapToEntity() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("mapToEntity() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### Repository Tests (với Testcontainers)
```go
func TestWalletTxRepository_Upsert(t *testing.T) {
    // Setup: Spin up PostgreSQL container
    ctx := context.Background()
    postgresC, _ := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
    )
    defer postgresC.Terminate(ctx)

    // Run migrations
    connString, _ := postgresC.ConnectionString(ctx)
    db, _ := sql.Open("postgres", connString)
    runMigrations(db)

    // Test upsert logic
    repo := NewWalletTxRepository(db)
    tx := &entities.WalletTransaction{
        ID: "tx-test-1",
        UserID: "user-1",
        Amount: 50000.0,
        Status: "SUCCESS",
    }

    // First insert
    err := repo.Upsert(ctx, tx)
    assert.NoError(t, err)

    // Verify inserted
    fetched, _ := repo.GetByID(ctx, "tx-test-1")
    assert.Equal(t, tx.Amount, fetched.Amount)

    // Update same record
    tx.Amount = 75000.0
    err = repo.Upsert(ctx, tx)
    assert.NoError(t, err)

    // Verify updated
    fetched, _ = repo.GetByID(ctx, "tx-test-1")
    assert.Equal(t, 75000.0, fetched.Amount)
    assert.Equal(t, int64(2), fetched.Version) // Version incremented
}
```

---

## 2. Integration Testing

### 2.1 CDC Worker End-to-End Test

**Setup**:
- Docker Compose với: NATS, PostgreSQL, MongoDB (for source simulation)
- Debezium không cần thiết (mock CDC events trực tiếp vào NATS)

**Test Flow**:
```
1. Publish test CDC events vào NATS
2. CDC Worker consumes và processes
3. Verify records trong PostgreSQL
4. Verify metrics được ghi nhận
```

**Implementation** (`tests/integration/cdc_worker_test.go`):
```go
func TestCDCWorker_Integration(t *testing.T) {
    // Start dependencies (docker-compose up)
    compose := setupDockerCompose(t)
    defer compose.Down()

    // Wait for services ready
    waitForNATS(t)
    waitForPostgres(t)

    // Publish test events
    nc, _ := nats.Connect("nats://localhost:4222")
    js, _ := nc.JetStream()

    testEvent := createTestCDCEvent("tx-integration-1")
    js.Publish("cdc.goopay.wallet_transactions", testEvent)

    // Wait for processing (max 5s)
    time.Sleep(2 * time.Second)

    // Verify in PostgreSQL
    db := connectPostgres(t)
    var count int
    db.QueryRow("SELECT COUNT(*) FROM wallet_transactions WHERE id = $1",
        "tx-integration-1").Scan(&count)

    assert.Equal(t, 1, count, "Record should exist in Postgres")
}
```

---

### 2.2 Event Bridge Integration Test

**Test Cases**:
- **Trigger-based**: Insert record vào Postgres → verify NATS event published
- **Polling-based**: Insert vào changelog → verify poller publishes NATS event

```go
func TestEventBridge_TriggerBased(t *testing.T) {
    // Setup: NATS subscriber
    nc, _ := nats.Connect("nats://localhost:4222")
    eventReceived := make(chan bool, 1)

    nc.Subscribe("goopay.wallet_transactions.INSERT", func(msg *nats.Msg) {
        eventReceived <- true
    })

    // Insert record vào Postgres (trigger sẽ fire)
    db := connectPostgres(t)
    db.Exec(`
        INSERT INTO wallet_transactions (id, user_id, amount, _source)
        VALUES ('bridge-test-1', 'user-1', 100000, 'airbyte')
    `)

    // Wait for event
    select {
    case <-eventReceived:
        t.Log("Event received successfully")
    case <-time.After(5 * time.Second):
        t.Fatal("Timeout waiting for NATS event")
    }
}
```

---

## 3. Load Testing

### 3.1 Throughput Test (k6 Script)

**Goal**: Verify CDC Worker can handle 50K events/sec with 5 pods

```javascript
// load_test.js
import { check } from 'k6';
import nats from 'k6/x/nats';

export let options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 50000, // 50K events/sec
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 100,
      maxVUs: 200,
    },
  },
};

export default function () {
  const nc = nats.connect('nats://nats-cluster:4222');

  const event = {
    specversion: '1.0',
    id: `load-test-${__VU}-${__ITER}`,
    source: '/debezium/test',
    type: 'io.debezium.mongodb.datachangeevent',
    data: {
      op: 'c',
      after: {
        id: `tx-${__VU}-${__ITER}`,
        user_id: `user-${__VU}`,
        amount: Math.random() * 1000000,
        status: 'SUCCESS',
      },
    },
  };

  nc.publish('cdc.goopay.wallet_transactions', JSON.stringify(event));
  nc.close();
}
```

**Metrics to Monitor**:
- NATS publish rate (target: 50K/sec)
- CDC Worker processing latency (target p99 < 100ms)
- PostgreSQL write throughput
- CPU/Memory usage per pod

---

### 3.2 Latency Test

**Goal**: Measure end-to-end latency (NATS publish → Postgres write)

```go
func BenchmarkE2ELatency(b *testing.B) {
    nc, _ := nats.Connect("nats://localhost:4222")
    db := connectPostgres()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        eventID := fmt.Sprintf("latency-test-%d", i)
        startTime := time.Now()

        // Publish event
        event := createTestCDCEvent(eventID)
        nc.Publish("cdc.goopay.wallet_transactions", event)

        // Poll Postgres until record appears (max 1s)
        timeout := time.After(1 * time.Second)
        ticker := time.NewTicker(10 * time.Millisecond)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                var exists bool
                db.QueryRow("SELECT EXISTS(SELECT 1 FROM wallet_transactions WHERE id = $1)",
                    eventID).Scan(&exists)

                if exists {
                    latency := time.Since(startTime)
                    b.ReportMetric(latency.Seconds()*1000, "latency_ms")
                    goto NextIteration
                }
            case <-timeout:
                b.Fatalf("Timeout waiting for record %s", eventID)
            }
        }

    NextIteration:
    }
}
```

---

## 4. Data Reconciliation Testing

### 4.1 Consistency Test

**Scenario**: Insert 10K records vào MongoDB → Verify all appear in Postgres

```go
func TestReconciliation_Consistency(t *testing.T) {
    // Insert 10K records into MongoDB
    mongoClient := connectMongo()
    collection := mongoClient.Database("goopay").Collection("wallet_transactions")

    for i := 0; i < 10000; i++ {
        doc := bson.M{
            "_id": primitive.NewObjectID(),
            "user_id": fmt.Sprintf("user-%d", i),
            "amount": 100000.0,
            "status": "SUCCESS",
        }
        collection.InsertOne(context.Background(), doc)
    }

    // Wait for CDC to process (adjust based on throughput)
    time.Sleep(30 * time.Second)

    // Run reconciliation
    report := runReconciliation("wallet_transactions")

    assert.True(t, report.CountMatch, "Source and target counts should match")
    assert.True(t, report.ChecksumMatch, "Checksums should match")
    assert.Empty(t, report.MissingInTarget, "No missing records in target")
}
```

---

### 4.2 Drift Detection Test

**Scenario**: Modify record trực tiếp trong Postgres → Reconciliation phát hiện

```go
func TestReconciliation_DetectDrift(t *testing.T) {
    // Insert record via CDC
    insertViaCDC(t, "drift-test-1", 100000.0)

    // Wait for sync
    time.Sleep(2 * time.Second)

    // Manually modify in Postgres (simulate drift)
    db := connectPostgres(t)
    db.Exec("UPDATE wallet_transactions SET amount = 999999 WHERE id = $1", "drift-test-1")

    // Run reconciliation
    report := runReconciliation("wallet_transactions")

    assert.False(t, report.ChecksumMatch, "Should detect checksum mismatch")
    assert.Contains(t, report.MismatchedData, "drift-test-1")
}
```

---

## 5. Failure Scenario Testing

### 5.1 Pod Crash Recovery

**Scenario**: Kill CDC Worker pod mid-processing → Verify no data loss

```bash
# Test script
kubectl delete pod cdc-worker-0 &

# Continue publishing events during crash
for i in {1..1000}; do
    nats pub cdc.goopay.wallet_transactions "$(generate_event $i)"
done

# Wait for pod restart
kubectl wait --for=condition=Ready pod/cdc-worker-0 --timeout=60s

# Verify all 1000 events processed (check Postgres count)
```

---

### 5.2 Database Connection Loss

**Test**: Simulate Postgres downtime → Verify retry mechanism

```go
func TestCDCWorker_PostgresDowntime(t *testing.T) {
    // Start processing
    publishTestEvents(100)

    // Kill Postgres
    stopPostgres()

    // Wait 10s (retries should happen)
    time.Sleep(10 * time.Second)

    // Restart Postgres
    startPostgres()

    // Wait for recovery
    time.Sleep(5 * time.Second)

    // Verify all 100 events eventually processed
    count := getPostgresCount("wallet_transactions")
    assert.Equal(t, 100, count)
}
```

---

## 6. Security Testing

### 6.1 SQL Injection Test (via /security-agent)

Verify upsert functions không vulnerable:

```go
func TestRepository_SQLInjection(t *testing.T) {
    repo := NewWalletTxRepository(db)

    maliciousID := "'; DROP TABLE wallet_transactions; --"
    tx := &entities.WalletTransaction{
        ID: maliciousID,
        UserID: "user-1",
        Amount: 100,
    }

    // Should handle safely via prepared statements
    err := repo.Upsert(context.Background(), tx)
    assert.NoError(t, err)

    // Verify table still exists
    var count int
    db.QueryRow("SELECT COUNT(*) FROM wallet_transactions").Scan(&count)
    assert.NotPanics(t, func() { _ = count })
}
```

---

## 7. Test Environments

| Environment | Purpose | Data | Duration |
|-------------|---------|------|----------|
| **Local (Docker Compose)** | Unit + Integration tests | Mocked/Test data | On every commit |
| **Staging** | Load testing, E2E validation | Production-like data (anonymized) | Before deployment |
| **Production (Canary)** | Real traffic testing | Real data | 10% traffic for 24h |

---

## 8. Acceptance Criteria

### Must Pass Before Production:

- [ ] Unit test coverage > 80%
- [ ] All integration tests pass
- [ ] Load test: 50K events/sec sustained for 10 minutes
- [ ] Latency p99 < 100ms
- [ ] Reconciliation detects 100% of injected drifts
- [ ] Zero data loss in pod crash scenario
- [ ] Security scan (SAST/DAST) pass
- [ ] Performance benchmarks meet targets

---

## 9. Continuous Testing (CI/CD Pipeline)

```yaml
# .github/workflows/cdc-worker-test.yml
name: CDC Worker Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - run: go test -v -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
      nats:
        image: nats:latest
    steps:
      - uses: actions/checkout@v3
      - run: go test -tags=integration -v ./tests/integration/...

  load-tests:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      - uses: grafana/k6-action@v0.3.0
        with:
          filename: tests/load_test.js
```

---

**End of Validation Plan**
