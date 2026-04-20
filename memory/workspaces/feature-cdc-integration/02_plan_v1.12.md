# Plan v1.12: High-Performance CDC at Scale (500GB+)

> **Date**: 2026-04-13
> **Version**: 1.12
> **Base**: Kế thừa toàn bộ v1.11 (5 Tracks A-E) + bổ sung tối ưu hiệu năng cho quy mô 100M+ records
> **Quyết định chiến lược**: Bỏ Snowflake, tối ưu thuần trên PostgreSQL + Go Worker
> **Prerequisite**: `02_plan_v1.11.md`, `00_current.md`

---

## 1. Bối cảnh mới (so với v1.11)

### 1.1 Quy mô thực tế
- **Target**: 500GB data, 100M+ records
- **Throughput**: 5K events/sec/pod, scale 25K-50K với multi-pod
- **60 microservices** → cần ID generation phân tán, không trùng

### 1.2 Quyết định loại bỏ Snowflake
- **Lý do**: Tránh đốt credit cho Cloud DW khi PostgreSQL + Go đủ sức gánh
- **Hệ quả**: PostgreSQL phải tự gánh storage + query performance → cần Partitioning + Data Flattening

### 1.3 Bổ sung kỹ thuật mới

| Hạng mục | v1.11 | v1.12 |
|:---|:---|:---|
| ID Generator | Serial / auto | **Sonyflake** (64-bit BigInt, K-ordered) |
| JSON Parser | encoding/json | **Gjson** (truy xuất nhanh gấp ~10x) |
| Batch Write | Single-row loop | **pgx.Batch / CopyFrom** |
| Storage | JSONB giữ nguyên | **Data Flattening** (JSONB → Typed Columns, tiết kiệm ~150-200GB) |
| Table Scaling | Single table | **Declarative Partitioning** theo tháng |
| Mapping Cache | DB read | **In-memory** (sync.Map / RWMutex) + Hot-reload |
| Bridge Read | SELECT * | **Keyset Pagination** (theo `_airbyte_emitted_at`) |

---

## 2. Architecture Overview v1.12

```
Airbyte (batch 5-15 min)
    ↓ sync
_airbyte_raw_{stream} (_airbyte_data JSONB)
    ↓ E0: Bridge (Keyset Pagination, Sonyflake ID)
cdc_{stream} (id BIGINT Sonyflake, _raw_data JSONB, _source, _hash, _version)
    ↓ E1: Transform (Gjson parse, pgx.Batch write)
cdc_{stream} (id, _raw_data, typed_col_1, typed_col_2, ..., _updated_at)
    ↓ ADR-013: Partitioning (monthly)
cdc_{stream}_2026_04, cdc_{stream}_2026_05, ...
```

### Hot-Reload Flow
```
CMS: Approve mapping rule
    ↓ NATS publish
schema.config.reload
    ↓ Worker subscriber
In-memory Mapping Cache (sync.Map) → update ngay, không restart Pod
```

---

## 3. Ưu tiên P0 — ~~Fix ngay (14/04)~~ ✅ ĐÃ CÓ SẴN

> **Verified 2026-04-13**: Cả 2 item P0 đã được implement trước đó.
> - `schema.config.reload` subscriber: `worker_server.go:94-115`
> - In-memory cache (RWMutex): `registry_service.go` — `registryCache`, `sourceCache`, `mappingCache`
> - ReloadAll() gọi khi nhận NATS message + startup
> → **SKIP, chuyển thẳng sang Sonyflake.**

### ~~P0-1: NATS Reload Subscriber (Hot-reload)~~ ✅ DONE

**Vấn đề**: ~~Worker không listen `schema.config.reload`~~ Đã có.

**Implementation**:
- File: `internal/server/worker_server.go`
- Subscribe NATS subject `schema.config.reload`
- Khi nhận signal: reload mapping rules từ DB → update in-memory cache
- Cache dùng `sync.RWMutex` (hoặc `sync.Map`) để thread-safe với 10 goroutines

```go
// worker_server.go
natsClient.Conn.Subscribe("schema.config.reload", func(msg *nats.Msg) {
    log.Info("Reloading mapping cache...")
    rules, err := mappingRuleRepo.GetAllActive(ctx)
    if err != nil {
        log.Errorf("Failed to reload mapping: %v", err)
        return
    }
    mappingCache.Reload(rules) // sync.RWMutex protected
    log.Infof("Mapping cache reloaded: %d rules", len(rules))
})
```

### P0-2: In-memory Mapping Cache

**Implementation**:
- File: `internal/cache/mapping_cache.go` (mới)
- Struct `MappingCache` với `sync.RWMutex`
- Key = TargetTable (theo ADR-012)
- Methods: `Get(tableName)`, `Reload(rules)`, `Invalidate(tableName)`
- EventHandler + BatchBuffer đọc từ cache thay vì query DB mỗi lần

---

## 4. Sonyflake Integration (15-16/04)

### 4.1 Tại sao Sonyflake?
- **64-bit BigInt**: Index B-Tree hiệu quả hơn NanoID/UUID (string) rất nhiều ở 100M+ rows
- **K-ordered (tăng dần theo thời gian)**: Insert luôn ở cuối B-Tree → không fragmentation
- **Machine ID (16 bits)**: Hỗ trợ 65K unique machines → đủ cho 60 microservices + multi-pod K8s
- **Không cần server quản lý tập trung**: Machine ID = 16-bit cuối của Pod IP

### 4.2 Implementation

- File: `pkgs/idgen/sonyflake.go` (mới)
- Dependency: `github.com/sony/sonyflake`

```go
package idgen

import (
    "net"
    "github.com/sony/sonyflake"
)

var sf *sonyflake.Sonyflake

func Init() {
    sf = sonyflake.NewSonyflake(sonyflake.Settings{
        MachineID: func() (uint16, error) {
            ip := getOutboundIP()
            return uint16(ip[2])<<8 + uint16(ip[3]), nil
        },
    })
}

func NextID() (uint64, error) {
    return sf.NextID()
}
```

### 4.3 Migration — Dual-PK Strategy
CDC tables cần giữ source PK (MongoDB _id) cho dedup, nhưng dùng Sonyflake cho internal PK.

**Schema mới cho CDC tables**:
```sql
CREATE TABLE cdc_{table} (
    id BIGINT PRIMARY KEY,                    -- Sonyflake ID (internal PK, B-Tree friendly)
    source_id VARCHAR(200) NOT NULL,          -- Original source PK (_id, etc.) cho dedup
    _raw_data JSONB NOT NULL,
    _source VARCHAR(20) DEFAULT 'airbyte',
    _synced_at TIMESTAMP DEFAULT NOW(),
    _version BIGINT DEFAULT 1,
    _hash VARCHAR(64),
    _deleted BOOLEAN DEFAULT FALSE,
    _created_at TIMESTAMP DEFAULT NOW(),
    _updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_id)                         -- Dedup on source PK
);
```

**Bridge thay đổi**: ON CONFLICT (source_id) thay vì ON CONFLICT (id)

**Bảng hiện tại**: Giữ nguyên schema cũ. Chỉ bảng mới dùng Sonyflake.

### 4.4 High-throughput Bridge (pgx.CopyFrom)
Go-based bridge thay thế SQL-only bridge cho v1.12 tables:
1. Read Airbyte rows via Keyset Pagination (`_airbyte_extracted_at > last_bridge_at`)
2. Parse `_raw_data` với **gjson** để extract source ID
3. Generate **Sonyflake ID** per row
4. Batch write via **pgx.CopyFrom** (COPY protocol, fastest Postgres insert)
5. NATS subject: `cdc.cmd.bridge-airbyte-batch`

**Benchmark target**: 5K rows/sec sustained (Sonyflake: 25K IDs/sec, 0 allocs)

### 4.4 Lab Test
- Gán Sonyflake ID cho 1M dòng demo
- Đo: Index size, Insert throughput, Query latency
- So sánh với NanoID/UUID

---

## 5. Track E: Bridge & High-Performance Transform (17-21/04+)

> Kế thừa v1.11 Track E, bổ sung Sonyflake + Gjson + pgx.Batch

### E0: Airbyte → CDC Bridge (2.5 ngày)

**Thay đổi so với v1.11**:
- Dùng **Keyset Pagination** thay `OFFSET/LIMIT`:
  ```sql
  SELECT _airbyte_data, _airbyte_emitted_at
  FROM _airbyte_raw_{stream}
  WHERE _airbyte_emitted_at > $1  -- last_bridge_at
  ORDER BY _airbyte_emitted_at ASC
  LIMIT 1000
  ```
- Gán **Sonyflake ID** cho mỗi row mới (thay vì dùng `_airbyte_ab_id` hoặc `_id` từ source)
- Dùng **Gjson** để extract ID từ `_airbyte_data`:
  ```go
  rawID := gjson.GetBytes(airbyteData, "_id.$oid").String()
  if rawID == "" {
      rawID = gjson.GetBytes(airbyteData, "id").String()
  }
  ```
- Dùng **pgx.Batch** để gom 500-1000 rows/batch trước khi flush

**NATS subject**: `cdc.cmd.bridge-airbyte`
**CMS endpoint**: `POST /api/registry/:id/bridge`
**Registry columns mới**: `airbyte_raw_table`, `last_bridge_at`

### E1: Post-sync Batch Transform (2 ngày)

**Thay đổi so với v1.11**:
- Đọc mapping rules từ **in-memory cache** (không query DB mỗi lần)
- Dùng **Gjson** parse `_raw_data`:
  ```go
  for _, rule := range mappingCache.Get(tableName) {
      val := gjson.GetBytes(rawData, rule.SourceField)
      // convert val theo rule.DataType
  }
  ```
- Batch UPDATE dùng **pgx.Batch**:
  ```go
  batch := &pgx.Batch{}
  for _, row := range rows {
      batch.Queue(updateSQL, extractedValues...)
  }
  results := conn.SendBatch(ctx, batch)
  ```

**NATS subject**: `cdc.cmd.batch-transform`
**CMS endpoint**: `POST /api/registry/:id/transform`

### E2: Periodic Scheduler (1 ngày)

Giữ nguyên v1.11:
- Ticker goroutine trong `worker_server.go`
- Config: `worker.transformInterval: 5m`
- Flow: Bridge → Transform liên hoàn mỗi 5 phút

### E3: Transform Status Tracking (0.5 ngày)

Giữ nguyên v1.11:
- `GET /api/registry/:id/transform-status`
- Response: `{total_rows, transformed_rows, pending_rows}`
- FE: progress bar

---

## 6. Track A-D: Giữ nguyên v1.11

### Track A: Airbyte Read APIs (2 ngày)
- A1: `GET /api/airbyte/destinations`
- A2: `GET /api/airbyte/connections` (enriched)
- A3: `GET /api/airbyte/connections/:id/streams`

### Track B: Stream Sync (3 ngày)
- B1: Full Stream Sync (auto-detect PK, create registry entries)
- B2: Stream config sync (airbyte_sync_mode, cursor_field, namespace)
- B3: Bidirectional active/inactive toggle

### Track C: Field Mapping Sync (3 ngày)
- C1: Auto-detect fields khi import (parse Airbyte JSONSchema)
- C2: Periodic field scan từ `_raw_data` (interval 1h)
- C3: Batch approve/reject + auto-backfill

### Track D: Monitoring & Consistency (2 ngày)
- D1: ✅ Sync health dashboard (GET /sync/health)
- D2: Reconciliation report — `GET /api/sync/reconciliation`
  - So sánh row count: Airbyte raw table vs CDC table
  - So sánh field coverage: mapped fields vs total fields in _raw_data
  - Per-table report: {table, airbyte_rows, cdc_rows, diff, mapped_fields, total_fields, coverage_pct}

---

## 7. PostgreSQL Partitioning — ADR-013 (Proposed)

### 7.1 Tại sao cần?
- 500GB trên single table → query báo cáo scan toàn bộ → treo DB
- Partitioning giữ Index gọn, query chỉ scan partition liên quan
- Xóa data cũ = DROP PARTITION (instant, không vacuum)

### 7.2 Strategy: Declarative Partitioning theo tháng

```sql
-- Parent table
CREATE TABLE cdc_merchants (
    id BIGINT NOT NULL,
    _raw_data JSONB,
    _source VARCHAR DEFAULT 'airbyte',
    _synced_at TIMESTAMP NOT NULL,
    _hash VARCHAR,
    _version INT DEFAULT 1,
    -- typed columns from mapping rules...
    PRIMARY KEY (id, _synced_at)
) PARTITION BY RANGE (_synced_at);

-- Monthly partitions
CREATE TABLE cdc_merchants_2026_04 PARTITION OF cdc_merchants
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE cdc_merchants_2026_05 PARTITION OF cdc_merchants
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
```

### 7.3 Auto-create partition
- Worker hoặc cron job tạo partition cho tháng tiếp theo trước khi hết tháng
- Hàm `create_cdc_table()` cập nhật: tạo parent + partition tháng hiện tại

### 7.4 Áp dụng cho bảng nào?
- **Bắt buộc**: Bảng có volume lớn (transactions, trans-his, logs)
- **Không cần**: Bảng nhỏ (merchants, users) — giữ single table

### 7.5 Implementation
- Migration `004_partitioning.sql` (Worker) — function `create_cdc_table_partitioned()`
- Registry flag `is_partitioned` + `partition_key` columns
- Auto-create partition cho tháng hiện tại + tháng sau
- Cron/periodic: Worker tạo partitions trước khi hết tháng

---

## 8. Data Flattening Strategy

### 8.1 Mục tiêu
Chuyển từ JSONB → Typed Columns để:
- Giảm ~150-200GB storage (Typed Columns nén tốt hơn JSONB)
- Query trực tiếp trên column (không cần `->>'field'`)
- Index trên typed columns (B-Tree, không cần GIN)

### 8.2 Flow
```
Phase 1: _raw_data JSONB giữ nguyên (zero data loss)
Phase 2: Transform → ghi typed columns
Phase 3: Khi typed columns đầy đủ → DROP GIN Index trên _raw_data
Phase 4: (Tùy chọn) Set _raw_data = NULL cho rows đã transform xong → tiết kiệm storage
```

### 8.3 Index Optimization
- Bỏ GIN Index trên `_raw_data` sau khi transform xong
- Chỉ tạo B-Tree Index trên typed columns thực sự cần query
- Partial Index nếu cần: `WHERE status = 'active'`

---

## 9. Migration SQL — v1.12

```sql
-- v1.12: Kế thừa v1.11 migrations + Sonyflake + Partitioning prep

-- 1. Registry columns (from v1.11)
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_sync_mode VARCHAR DEFAULT 'incremental';
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_destination_sync_mode VARCHAR DEFAULT 'append';
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_cursor_field VARCHAR;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_namespace VARCHAR;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_raw_table VARCHAR;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS last_bridge_at TIMESTAMP;

-- 2. Partitioning flag
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS is_partitioned BOOLEAN DEFAULT false;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS partition_key VARCHAR DEFAULT '_synced_at';

-- 3. Transform tracking
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS last_transform_at TIMESTAMP;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS total_rows BIGINT DEFAULT 0;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS transformed_rows BIGINT DEFAULT 0;

-- 4. Update create_cdc_table() function: id BIGINT thay vì TEXT
-- (Chi tiết trong 09_tasks_solution)
```

---

## 10. Dependencies mới (Go modules)

```go
// go.mod additions
require (
    github.com/sony/sonyflake v1.2.0    // Sonyflake ID generator
    github.com/tidwall/gjson v1.18.0     // Fast JSON parser
    github.com/jackc/pgx/v5 v5.7.0      // pgx for Batch/CopyFrom (nếu chưa có)
)
```

---

## 11. Timeline tổng hợp v1.12

```
Week 1 (14-20/04) — Foundation + Bridge:
  ├─ 14/04: P0 Fix — Reload Subscriber + In-memory Cache
  ├─ 15-16/04: Sonyflake Lab — integration + benchmark 1M rows
  └─ 17-20/04: Track E0 — Bridge (Keyset Pagination + Sonyflake + Gjson)

Week 2 (21-27/04) — Transform + Airbyte Sync:
  ├─ 21-22/04: Track E1 — Batch Transform (pgx.Batch + Gjson)
  ├─ 23/04: Track E2 — Periodic Scheduler
  ├─ 23/04: Track E3 — Transform Status
  ├─ 24/04: Track A (A1-A3) — Airbyte Read APIs
  └─ 25-27/04: Track B (B1-B3) — Stream Sync

Week 3 (28/04-04/05) — Field Mapping + Monitoring + Partitioning:
  ├─ 28-30/04: Track C (C1-C3) — Field Mapping Sync
  ├─ 01-02/05: Track D (D1-D2) — Monitoring & Reconciliation
  └─ 03-04/05: ADR-013 — Partitioning cho bảng lớn

Week 4 (05-09/05) — Performance Tuning + Testing:
  ├─ Index Optimization (drop GIN, add B-Tree on typed columns)
  ├─ Load test: 5K events/sec sustained
  ├─ Unit + Integration tests
  └─ Documentation + Walkthrough
```

**Critical path**: P0 (Reload) → Sonyflake → E0 (Bridge) → E1 (Transform) → E2 (Scheduler)

---

## 12. API tổng hợp (kế thừa v1.11 + mới)

| Method | Path | Track | Mô tả |
|:---|:---|:---|:---|
| `POST` | `/api/registry/:id/bridge` | E0 | Trigger Airbyte→CDC bridge |
| `POST` | `/api/registry/:id/transform` | E1 | Trigger batch transform |
| `GET` | `/api/registry/:id/transform-status` | E3 | Transform progress |
| `GET` | `/api/airbyte/destinations` | A1 | List destinations |
| `GET` | `/api/airbyte/connections` | A2 | List connections enriched |
| `GET` | `/api/airbyte/connections/:id/streams` | A3 | Streams + registry comparison |
| `POST` | `/api/registry/sync-from-airbyte` | B1 | Full sync streams → registry |
| `PATCH` | `/api/mapping-rules/batch` | C3 | Batch approve/reject |
| `GET` | `/api/sync/health` | D1 | Sync health summary |

---

## 13. Definition of Done — v1.12

### Kế thừa v1.11
- [ ] Worker tự bridge + transform mỗi 5 phút
- [ ] Transform status visible trên UI
- [ ] Tất cả Airbyte streams có trong CMS registry
- [ ] Mỗi stream có mapping rules cho tất cả fields
- [ ] Active/inactive mismatch = 0
- [ ] Periodic scan phát hiện fields mới
- [ ] Approve mapping → auto-backfill
- [ ] Sync health dashboard

### Mới v1.12
- [ ] Hot-reload mapping cache qua NATS (không restart Pod)
- [ ] Sonyflake ID cho tất cả CDC tables mới
- [ ] Bridge dùng Keyset Pagination (không OFFSET)
- [ ] Transform dùng Gjson + pgx.Batch
- [ ] Throughput >= 5K events/sec sustained (benchmark)
- [ ] Partitioning cho bảng > 10M rows
- [ ] GIN Index trên `_raw_data` được drop sau transform hoàn tất
- [ ] All builds OK
- [ ] `05_progress.md` updated per change

---

## 14. Không làm trong v1.12

| Hạng mục | Lý do | Khi nào |
|:---|:---|:---|
| Debezium standalone | Cần true realtime, deploy riêng | Phase 2 |
| Event Bridge (PG Triggers → NATS) | Cần core sync ổn trước | v1.13+ |
| dbt | Go Worker là transform layer | Không cần |
| Snowflake / Cloud DW | Chi phí cao, PostgreSQL đủ sức | Không dùng |
| Connection/Source CRUD trên CMS | Airbyte UI là master | Không cần |

---

## 15. Files thay đổi tổng hợp

### centralized-data-service (Worker)
| File | Thay đổi | Track |
|:---|:---|:---|
| `pkgs/idgen/sonyflake.go` | **Mới** — Sonyflake ID generator | Sonyflake |
| `internal/cache/mapping_cache.go` | **Mới** — In-memory cache với RWMutex | P0 |
| `internal/handler/command_handler.go` | Bridge, Transform, periodic scan | E0, E1, C2 |
| `internal/handler/event_handler.go` | Đọc mapping từ cache thay vì DB | P0 |
| `internal/server/worker_server.go` | Reload subscriber, transform ticker | P0, E2 |
| `config/config.go` | TransformInterval, ScanInterval | E2, C2 |
| `config/config-local.yml` | Config values | E2, C2 |
| `go.mod` | sonyflake, gjson, pgx | All |
| `migrations/` | Sonyflake ID, partitioning | Migration |

### cdc-cms-service (API)
| File | Thay đổi | Track |
|:---|:---|:---|
| `pkgs/airbyte/client.go` | ListDestinations | A1 |
| `internal/api/airbyte_handler.go` | Destinations, connections, streams | A1-A3 |
| `internal/api/registry_handler.go` | SyncFromAirbyte, Transform, TransformStatus, Bridge | B1, E0-E3 |
| `internal/api/mapping_rule_handler.go` | BatchUpdate | C3 |
| `internal/service/approval_service.go` | Auto-backfill after approve | C3 |
| `internal/router/router.go` | Register new routes | All |
| `internal/model/table_registry.go` | New columns | B2, v1.12 |
| `migrations/` | ALTER TABLE | B2, v1.12 |

### cdc-cms-web (Frontend)
| File | Thay đổi | Track |
|:---|:---|:---|
| `src/pages/TableRegistry.tsx` | Transform progress bar | E3 |
| `src/pages/MappingFieldsPage.tsx` | Batch approve/reject | C3 |
| `src/pages/Dashboard.tsx` | Sync health widget | D1 |
| `src/pages/SourceConnectors.tsx` | Destinations display | A1 |
