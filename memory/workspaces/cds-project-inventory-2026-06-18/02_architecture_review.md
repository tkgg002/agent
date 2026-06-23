# Đánh giá Kiến trúc — `centralized-data-service`

> **Ngày đánh giá**: 2026-06-18 | **Đánh giá bởi**: Brain (Antigravity)
> **Nguồn**: Code review trực tiếp, không dựa trên giả định.

---

## Tổng quan

`centralized-data-service` là một **CDC Worker Service** thuần Go, đảm nhận vai trò trung tâm trong pipeline dữ liệu của hệ thống GooPay. Dự án có **191 file Go**, 3 entrypoints độc lập, và ~30 NATS subjects subscribed.

---

## 1. Điểm Mạnh ✅

### 1.1 Interface-based Dependency Injection

`MetadataRegistry` được định nghĩa là **interface** rõ ràng — các caller không phụ thuộc vào implementation cụ thể. Điều này cho phép test dễ dàng:

```go
type MetadataRegistry interface {
    ReloadAll(ctx context.Context) error
    GetTableConfig(targetTable string) *model.TableRegistry
    GetSourceDSN(ctx context.Context, connectionCode string) (string, error)
    // ...
}
```

`WorkerServer` inject `service.MetadataRegistry` (interface), không phải `*MetadataRegistryService` (concrete) → **Đây là pattern đúng**.

### 1.2 Test suite phân tách (`test/`)

**50 test files** được tổ chức riêng trong `test/` — tách hoàn toàn khỏi production code. Có cả:
- Unit tests (pure logic)
- Integration tests (`_integration_test.go`)
- Benchmark tests (`benchmark_test.go`)

### 1.3 Observability: OTel + zap bridge

`cmd/worker/main.go` wires OTel → SigNoz với **severity-aware sampling**, nghĩa là `debug/info` logs không flood collector. Pattern sử dụng `zapcore.NewTee(logger.Core(), bridge)` là đúng.

### 1.4 Transmute — Strategy Pattern đúng

`internal/service/transmute/` có Strategy interface riêng:
```go
type Strategy interface { ... }
func Register(s Strategy) {}
func Get(transformType string) (Strategy, bool) {}
```
Tách pure functions ra sub-package — **code có thể test độc lập, không phụ thuộc DB**.

### 1.5 Graceful shutdown & Fencing token

`cmd/sinkworker/main.go` implement **fencing token** qua Postgres function `heartbeat_machine_id(machineID, fencingToken)`:
```go
if !alive {
    logger.Error("FENCING: token reclaimed by another pod — self-terminating")
    cancel()
}
```
→ Chống split-brain khi multi-instance deployment. Pattern tốt.

### 1.6 Multi-DB plane separation

`worker_server.go` phân tách rõ:
- `db` = Control Plane (cdc_system metadata)
- `shadowDB` = Shadow Plane (cdc_shadow, raw events)
- `masterDB` = Destination Plane (cdc_dw, final data)
- `dbReplica` = Read Replica

Đây là architectural boundary đúng đắn cho CDC pipeline.

### 1.7 Connection override pattern

`CONNECTION_OVERRIDE_<CODE>=<uri>` env vars → `applyConnectionOverrides()` → cho phép redirect source connections mà không cần sửa DB. Hữu ích cho dev/staging.

---

## 2. Vấn đề Nghiêm trọng 🔴

### 2.1 God Object: `command_handler.go` (3,437 dòng)

**File lớn nhất** trong project — một struct `CommandHandler` đảm nhận **quá nhiều trách nhiệm**:
- Schema discovery (Mongo + Debezium)
- DDL execution (`CREATE TABLE`, `ALTER COLUMN`)
- Mapping rule management
- Connector management
- Backfill coordination
- Scan operations (raw data, array fields, periodic)

```
HandleStandardize → HandleDiscover → HandleBackfill → HandleMasterSwap
HandleScanRawData → HandleScanArrayFields → HandleBatchTransform → HandlePeriodicScan
HandleScanFields → HandleSyncRegister → HandleSyncState → HandleRestartDebezium
HandleAlterColumn → HandleDropGINIndex → HandleDiscoverMongoDatabases → ...
```

**18 public NATS handlers** trong 1 file = vi phạm Single Responsibility Principle.

**Giải pháp đề xuất**: Tách theo domain:
```
handler/
├── discovery_handler.go    (scan, discover, introspect)
├── schema_handler.go       (standardize, alter, create-default)
├── connector_handler.go    (sync-register, sync-state, restart)
├── backfill_handler.go     (backfill, batch-transform, master-swap)
└── command_handler.go      (shared types + base struct)
```

### 2.2 Thiếu schema quản lý tập trung

Chỉ có **1 migration file** (`migrations/dest/001_dest_init.sql`). Comments trong `worker_server.go` cho thấy có **ít nhất 17 migration files** (001→017) nhưng chúng không được commit vào repo:

```go
// Tables & owning migrations:
//   - cdc_table_registry      -> 001_init_schema.sql
//   - cdc_activity_log        -> 006_activity_log.sql, 010_partitioning.sql
//   - cdc_failed_sync_logs    -> 008_reconciliation.sql, 012_dlq_state_machine.sql
```

→ **Không thể onboard developer mới** mà không có các migrations này. Schema drift risk cao.

### 2.3 `worker_server.go` — Constructor 700+ dòng

`NewWorkerServer()` là một **mega-constructor** ~700 dòng thực hiện:
- Init 4 DB connections
- Init NATS + ensure streams
- Init Redis
- Init 10+ services
- Init 10+ handlers
- Subscribe 30+ NATS subjects
- Register cron jobs

Không có **phân tầng dependency** (Wire/fx). Khi thêm dependency mới → phải sửa file này.

**Đây là anti-pattern**: mọi thứ được wired thủ công, thứ tự init không tường minh, khó test riêng từng thành phần.

---

## 3. Vấn đề Trung bình 🟡

### 3.1 V1/V2 Dual Model coexist

`MappingRule` (V1) và `MappingRuleV2` song song tồn tại. V1 được convert sang V2 qua `convertV2ToLegacyRule()`:
```go
// V1 table `cdc_mapping_rules` is deprecated — keep ONE source of truth
// at `cdc_system.mapping_rule_v2`.
```

V1 vẫn còn trong code (`MappingRuleRepo`, `MappingRule` model). Gây confusion cho người đọc code. **Cần deprecation plan rõ ràng.**

### 3.2 `SetX()` methods thay vì constructor injection

`CommandHandler` sử dụng setter-based injection:
```go
cmdHandler.SetMetadataRegistry(registrySvc)
cmdHandler.SetKafkaConnectURL(cfg.Debezium.KafkaConnectURL)
cmdHandler.SetNATSConn(natsClient.Conn)
cmdHandler.SetMongoService(mongoIntrospectSvc)
```

Điều này có nghĩa là `CommandHandler` có thể được sử dụng ở trạng thái **chưa được wired đầy đủ** — nil pointer panic tiềm ẩn khi deploy không đủ config. Constructor nên enforce required deps.

### 3.3 `scratch/` và binaries checked-in vào repo

```
admin-api     ← compiled binary tại root
worker        ← compiled binary tại root
scratch/      ← debug scripts
test_output.log ← log file
```

Không có `.gitignore` cho các artifacts này → Git history bẩn, repo size tăng theo thời gian.

### 3.4 `Provisioning` gated bởi env var, không phải config

```go
if os.Getenv("PROVISIONING_ORCHESTRATOR_ENABLED") == "1" {
```

Feature flag dùng `os.Getenv` raw thay vì đi qua `AppConfig` → không được document trong `config.go`, không có validation → dễ misconfigure silently.

### 3.5 Không có error budget / rate limiting cho NATS handlers

Handlers như `HandleBatchTransform`, `HandleScanRawData` không có timeout context cho từng invocation. NATS message có thể gây handler chạy vô thời hạn nếu query chậm.

---

## 4. Technical Debt 🔵

### 4.1 Log statements duplicate (fmt.Sprintf + zap fields)

Pattern log hiện tại:
```go
logger.Info(fmt.Sprintf("PostgreSQL connected component=worker_server op=pg_init init_duration_ms=%d ...", pgDurationMs, ...),
    zap.String("component", "worker_server"),
    zap.String("op", "pg_init"),
    zap.Int64("init_duration_ms", pgDurationMs),
)
```

Thông tin bị duplicate — cả trong message string **và** trong structured fields. Đây là anti-pattern với zap. Nên dùng thuần structured fields, không ghi vào message string.

### 4.2 `recon_core.go` 1,900 dòng — cần refactor

File lớn thứ 2. Chứa toàn bộ reconciliation engine — 3-tier logic, Segment A/B, leader election, etc. Nên tách thành:
- `recon_tier1.go` (fast hash)
- `recon_tier2.go` (deep compare)
- `recon_segment_b.go` (shadow↔master)

### 4.3 SQL hardcoded table names

`command_handler.go` và nhiều file khác có hardcoded table references:
```go
h.db.Table("cdc_system.mapping_rule_v2")
h.db.Raw(`SELECT * FROM cdc_system.source_object_registry WHERE ...`)
```

Nên định nghĩa constants hoặc dùng GORM model với `TableName()`.

### 4.4 `pgxPool` allocated nhưng không dùng

```go
_ = pgxPool  // Sprint 4 4A.1. pgxPool remains available for future...
```

Connection pool được tạo, giữ nhưng không dùng — tốn connection resource. Cần remove hoặc implement sử dụng thực sự.

---

## 5. Đánh giá kiến trúc tổng thể

| Khía cạnh | Điểm (1-5) | Nhận xét |
|---|---|---|
| **Separation of Concerns** | 3/5 | Tốt ở model/repo/service, nhưng handler/ quá lớn |
| **Testability** | 4/5 | Interface-based, test suite phân tách — tốt |
| **Observability** | 4/5 | OTel + zap + Prometheus — đầy đủ. Log format chưa nhất quán |
| **Resilience** | 4/5 | Fencing token, DLQ, circuit breaker, adaptive batching — tốt |
| **Maintainability** | 2/5 | God objects, mega-constructor, thiếu migration files |
| **Security** | 4/5 | AES masking, SQL injection prevention, token auth |
| **Scalability** | 4/5 | Multi-instance via NATS queue groups, adaptive batching |
| **Deployment** | 3/5 | K8s manifests có nhưng binaries checked-in |

**Điểm tổng: 3.5/5**

---

## 6. Khuyến nghị ưu tiên

| # | Action | Priority | Effort |
|---|---|---|---|
| 1 | Tách `command_handler.go` thành 4-5 handlers theo domain | 🔴 High | 2-3 ngày |
| 2 | Thêm tất cả migration files vào `migrations/` | 🔴 High | 1 ngày |
| 3 | Tách `NewWorkerServer()` thành các sub-builders (hoặc dùng Wire) | 🟡 Med | 2 ngày |
| 4 | Deprecate hoàn toàn V1 `MappingRule` + `MappingRuleRepo` | 🟡 Med | 1 ngày |
| 5 | Thêm `.gitignore` cho `admin-api`, `worker`, `scratch/`, `*.log` | 🟢 Low | 30 phút |
| 6 | Fix log format: bỏ `fmt.Sprintf` trong zap message | 🟢 Low | 1 ngày |
| 7 | Move `PROVISIONING_ORCHESTRATOR_ENABLED` vào `AppConfig` | 🟢 Low | 30 phút |
| 8 | Remove hoặc sử dụng `pgxPool` | 🟢 Low | 30 phút |
