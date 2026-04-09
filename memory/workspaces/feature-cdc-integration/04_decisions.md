# Architecture Decision Records (ADRs): CDC Integration

> **Workspace**: feature-cdc-integration
> **Purpose**: Document key technical decisions made during implementation

---

## ADR-001: Event Bridge Architecture (Postgres → NATS)

**Date**: 2026-03-16
**Status**: Proposed
**Deciders**: Brain (Antigravity), Muscle (Dev Team), DevOps

### Context
Airbyte ghi dữ liệu trực tiếp vào PostgreSQL. Các Moleculer services cần nhận events khi có thay đổi trong Postgres để xử lý business logic real-time.

**Options**:

#### Option A: PostgreSQL Triggers + NOTIFY/LISTEN
**Pros**:
- Real-time (< 10ms latency)
- Native PostgreSQL feature, không cần thêm dependencies
- Simple setup cho critical tables

**Cons**:
- Tight coupling với database
- Cần Go Listener service chạy 24/7 (single point of failure)
- Khó scale horizontally (LISTEN chỉ 1 connection)
- Performance impact nếu quá nhiều triggers

#### Option B: Polling + Changelog Table
**Pros**:
- Loosely coupled
- Dễ scale (multiple poller instances)
- Có thể batch events để giảm NATS traffic
- Fault-tolerant (poller crash không ảnh hưởng data)

**Cons**:
- Higher latency (1-5s polling interval)
- Extra storage cho changelog table
- Cần cleanup job để archive old changelogs

### Decision

**Hybrid Approach**:
- **Option A (Trigger-based)** cho **Critical Tables**: wallet_transactions, payments, orders
  - Rationale: Cần real-time updates cho financial data
- **Option B (Polling-based)** cho **Non-Critical Tables**: logs, reports, analytics tables
  - Rationale: Có thể chấp nhận latency cao hơn, ưu tiên reliability

### Consequences

**Positive**:
- Critical data có real-time updates
- Non-critical data có reliability cao hơn
- Giảm load trên PostgreSQL (không phải trigger mọi table)

**Negative**:
- Phải maintain 2 cơ chế song song
- Complexity tăng lên

**Mitigation**:
- Chuẩn hóa configuration (YAML config file chỉ định table nào dùng approach nào)
- Shared codebase cho publishing NATS events

---

## ADR-002: CDC Event Format Standardization

**Date**: 2026-03-16
**Status**: Proposed
**Deciders**: Brain, Muscle

### Context
Debezium sử dụng CloudEvents hoặc Avro format. Cần quyết định format cho NATS events từ Event Bridge.

### Decision

**Use Debezium CloudEvents JSON format** cho consistency:

```json
{
  "specversion": "1.0",
  "id": "event-uuid",
  "source": "/airbyte/postgres/goopay/{table}",
  "type": "io.goopay.datachangeevent",
  "datacontenttype": "application/json",
  "time": "2026-03-16T10:30:00.123Z",
  "data": {
    "op": "c|u|d",
    "before": {...},
    "after": {...}
  }
}
```

**Rationale**:
- Moleculer services đã quen thuộc với JSON format
- CloudEvents là industry standard (CNCF)
- Dễ debug và monitoring
- Schema evolution không cần recompile (so với Avro)

### Consequences
- Events có size lớn hơn Avro (~30-40% overhead)
- Chấp nhận trade-off vì ease of use
- Nếu bandwidth trở thành vấn đề, có thể migrate sang Avro sau

---

## ADR-003: Conflict Resolution Strategy

**Date**: 2026-03-16
**Status**: Proposed
**Deciders**: Brain, Database Engineer

### Context
Edge case: Cả Debezium CDC Worker và Airbyte cùng write vào 1 table (do misconfiguration hoặc table migration period).

### Decision

**Timestamp-based Last-Write-Wins với Version Tracking**:

```sql
ON CONFLICT (id) DO UPDATE SET
    ...
    _synced_at = NOW(),
    _version = table._version + 1
WHERE table._synced_at < NOW() - INTERVAL '1 second'
   OR table._hash != EXCLUDED._hash;
```

**Rules**:
1. Nếu `_synced_at` của record mới > record cũ → Update
2. Nếu `_hash` khác nhau → Update (data đã thay đổi)
3. Increment `_version` mỗi lần update
4. Log conflict events vào `cdc_conflicts` table cho audit

### Consequences
- Có thể mất updates nếu 2 writes xảy ra trong cùng 1 giây (acceptable risk)
- Version tracking giúp detect concurrent updates
- Audit trail cho troubleshooting

---

## ADR-004: Data Reconciliation Frequency

**Date**: 2026-03-16
**Status**: Proposed
**Deciders**: Brain, DevOps

### Context
Cần balance giữa data accuracy và resource consumption.

### Decision

**Tiered Reconciliation Schedule**:

| Tier | Tables | Frequency | Method |
|------|--------|-----------|--------|
| Critical | wallet_transactions, payments, orders | Every 15 minutes | Full checksum validation |
| High | users, merchants, wallets | Every 1 hour | Count + Sampling (10% random) |
| Medium | logs, reports | Every 4 hours | Count only |
| Low | analytics tables | Daily | Count only |

**Rationale**:
- Critical tables cần nhanh chóng detect drift
- Non-critical tables có thể chậm hơn để tiết kiệm resources

### Consequences
- Critical data có SLA cao (detect drift trong 15 phút)
- Resource usage tối ưu (không full scan mọi table mỗi 5 phút)

---

## ADR-005: Go CDC Worker Concurrency Model

**Date**: 2026-03-16
**Status**: Proposed
**Deciders**: Muscle (Go Developer)

### Context
Cần xác định concurrency model cho CDC Worker để đạt throughput target 50K events/sec.

### Decision

**Worker Pool Pattern với Batch Processing**:

```
NATS Fetch Loop (goroutine 1)
    ↓ (channel)
Worker Pool (10 goroutines per pod)
    ↓
Batch Buffer (500 records)
    ↓
PostgreSQL Batch Upsert
```

**Configuration**:
- **Workers per pod**: 10 concurrent goroutines
- **Batch size**: 500 records
- **Batch timeout**: 2 seconds (flush nếu không đủ 500 records)
- **NATS fetch size**: 1000 messages/pull

**Rationale**:
- Worker pool giảm overhead tạo goroutines liên tục
- Batch upserts giảm database round-trips (10x faster than individual upserts)
- Timeout đảm bảo low latency cho low-traffic periods

### Consequences
- Throughput cao: 5K events/sec per pod → 25K với 5 pods → có thể scale lên 50K
- Latency tăng nhẹ (max 2s do batching) → acceptable cho use case này
- Memory usage: ~100MB per pod (500 records * 200KB/record)

---

## ADR-006: Schema Migration Strategy

**Date**: 2026-03-16
**Status**: Proposed
**Deciders**: Brain, Database Engineer

### Context
Source schemas (MongoDB/MySQL) có thể thay đổi. Cần strategy để handle schema evolution.

### Decision

**Dual-Phase Migration với Backward Compatibility**:

**Phase 1: Additive Changes (No Breaking)**:
- Thêm columns mới vào PostgreSQL
- CDC Worker check field existence trước khi map
- Default values cho fields mới

**Phase 2: Breaking Changes (Coordinated)**:
- Pause CDC Worker
- Run migration script
- Update CDC Worker code
- Resume

**Schema Registry**:
- Maintain schema versions trong `cdc_schema_versions` table
- CDC Worker validate event schema version trước khi process
- Reject events với schema version không tương thích

### Consequences
- Zero-downtime cho additive changes
- Controlled downtime (< 5 min) cho breaking changes
- Schema mismatches dễ detect và debug

---

## ADR-007: Error Handling & Dead Letter Queue (DLQ)

**Date**: 2026-03-16
**Status**: Proposed
**Deciders**: Muscle (Go Developer)

### Context
Một số CDC events có thể fail do data corruption, schema mismatch, hoặc business validation errors.

### Decision

**Multi-tier Error Handling**:

```
Event Processing
    ↓ (fail)
Retry 3 times (exponential backoff)
    ↓ (still fail)
Send to DLQ (NATS stream: cdc.dlq)
    ↓
Alert DevOps (Slack/PagerDuty)
    ↓
Manual Investigation & Replay
```

**DLQ Event Format**:
```json
{
  "original_event": {...},
  "error_message": "...",
  "error_stack": "...",
  "retry_count": 3,
  "failed_at": "2026-03-16T10:30:00Z",
  "worker_id": "cdc-worker-pod-1"
}
```

**Replay Mechanism**:
- Admin CLI tool: `cdc-admin replay --dlq-id=<id>`
- Sau khi fix root cause (schema/code), replay events từ DLQ

### Consequences
- No message loss (tất cả failed events đều trong DLQ)
- Giảm noise (không retry vô hạn)
- Traceability cao (có đầy đủ context để debug)

---

---

## ADR-008: JSONB Landing Zone Strategy (NEW v2.0)

**Date**: 2026-03-16
**Status**: Approved
**Deciders**: Brain, Muscle, Database Engineer

### Context
Khi source schema thay đổi (thêm field mới) mà PostgreSQL chưa kịp ALTER TABLE, data có thể bị mất nếu CDC Worker reject events.

### Decision

**Use JSONB Landing Zone** - Column `_raw_data JSONB` trong mọi CDC table:

```sql
CREATE TABLE {table_name} (
    -- Business columns
    id VARCHAR(36) PRIMARY KEY,
    {mapped_columns},

    -- Landing Zone
    _raw_data JSONB,  -- Lưu toàn bộ JSON thô

    -- Metadata
    _source VARCHAR(20),
    _synced_at TIMESTAMP
);
```

**Workflow**:
1. CDC Worker luôn lưu toàn bộ JSON vào `_raw_data`
2. Nếu field đã có mapping rule → extract ra column riêng
3. Nếu field chưa có mapping → chỉ lưu trong `_raw_data`
4. Sau khi approve schema change → extract từ `_raw_data` ra column mới (backfill)

**Rationale**:
- **Zero Data Loss**: Guaranteed, ngay cả khi schema chưa ready
- **Flexible Queries**: Có thể query JSONB với PostgreSQL operators (`->`, `->>`, `@>`)
- **Audit Trail**: Raw data luôn available cho troubleshooting
- **Gradual Migration**: Không cần rush ALTER TABLE

### Consequences

**Positive**:
- Hoàn toàn loại bỏ risk mất data trong schema migration
- Dev có thời gian review schema changes cẩn thận
- Có thể revert/reprocess data nếu mapping sai

**Negative**:
- Storage tăng ~30-40% (duplicate data in columns + JSONB)
- Query performance có thể chậm hơn nếu query trực tiếp JSONB (mitigate bằng GIN index)

---

## ADR-009: Dynamic Mapping Engine Architecture (NEW v2.0)

**Date**: 2026-03-16
**Status**: Approved
**Deciders**: Brain, Muscle (Go Developer)

### Context
Hard-coding struct trong Go CDC Worker khiến mỗi lần thêm field phải push code, rebuild Docker, restart pods → downtime và slow iteration.

### Decision

**Implement Generic Processor với Rule-based Mapping**:

**Architecture**:
```
[Go CDC Worker]
    ↓
[Load Mapping Rules from DB] → Cache in Redis
    ↓
[Dynamic Query Builder] → Build INSERT/UPDATE dựa trên rules
    ↓
[PostgreSQL Upsert]
```

**Mapping Rules Table**:
```sql
CREATE TABLE cdc_mapping_rules (
    source_table VARCHAR(100),
    source_field VARCHAR(100),
    target_column VARCHAR(100),
    data_type VARCHAR(50),
    is_active BOOLEAN,
    is_enriched BOOLEAN
);
```

**Hot Reload**: NATS event `schema.config.reload` → reload rules without restart

**Rationale**:
- **Config-Driven**: Thêm field = configuration change, not code change
- **Zero Downtime**: Reload rules in < 5 seconds
- **Self-Service**: DevOps có thể add mappings qua CMS UI
- **Maintainability**: Single codebase handles all tables

### Consequences
- Phải maintain mapping rules carefully (wrong mapping = data corruption)
- Slightly slower than hard-coded (query builder overhead ~5-10ms)
- Requires Redis for caching rules

---

## ADR-010: CMS Approval Workflow for Schema Changes (NEW v2.0)

**Date**: 2026-03-16
**Status**: Approved
**Deciders**: Brain, DevOps, Security Team

### Context
Fintech systems require strict change management. Automatic ALTER TABLE (ADR-003 option) quá risky cho production.

### Decision

**Implement CMS-based Approval Workflow**:

**Workflow**:
1. **Detect**: Schema Inspector phát hiện field mới → lưu `pending_fields`
2. **Alert**: Publish NATS event → CMS Service nhận notification
3. **Review**: DevOps/Dev vào CMS UI, xem field mới, suggest type
4. **Approve**: Click "Approve" → CMS Backend execute ALTER TABLE
5. **Reload**: Publish `schema.config.reload` → CDC Worker reload mapping
6. **Backfill** (optional): Extract field từ `_raw_data` ra column mới

**Technology Stack**:
- **Backend**: Go (Gin framework) hoặc Node.js (Express)
- **Frontend**: React + Ant Design/Material-UI
- **Auth**: JWT-based với role: `admin`, `developer`, `viewer`

**Rationale**:
- **Change Control**: Mọi schema change đều có approval trail
- **Risk Mitigation**: Prevent accidental AUTO DDL breaking production
- **Audit Compliance**: Meet fintech regulatory requirements
- **Human-in-the-loop**: Critical decisions need human review

### Consequences
- Thêm manual step (approval) → latency từ detect đến live ~2-30 phút (tùy response time)
- Cần maintain thêm 1 service (CMS)
- Cần training DevOps/Dev sử dụng CMS

---

## ADR-011: Schema Drift Detection Strategy (NEW v2.0)

**Date**: 2026-03-16
**Status**: Approved
**Deciders**: Brain, Muscle

### Context
Cần detect schema changes tự động để kích hoạt approval workflow.

### Decision

**Implement Schema Inspector Module** trong CDC Worker:

**Detection Logic**:
```go
func (si *SchemaInspector) InspectEvent(event *CDCEvent) (*SchemaDrift, error) {
    // 1. Extract fields from JSON
    eventFields := extractFieldNames(event.Payload.After)

    // 2. Get cached schema from Redis (or query PostgreSQL)
    tableSchema := si.getTableSchema(event.Table)

    // 3. Find new fields
    newFields := difference(eventFields, tableSchema)

    if len(newFields) > 0 {
        // 4. Infer data type from sample value
        // 5. Save to pending_fields
        // 6. Publish drift alert
        si.publishDriftAlert(event.Table, newFields)
    }
}
```

**Type Inference**:
- `float64` với fractional → `DECIMAL`
- `float64` without fractional → `INTEGER`
- `string` parsable as RFC3339 → `TIMESTAMP`
- `string` otherwise → `VARCHAR(255)` (adjustable in CMS)
- `bool` → `BOOLEAN`
- `map[string]interface{}` → `JSONB`

**Rationale**:
- **Proactive**: Không chờ lỗi mới phát hiện
- **Automated**: Reduce manual monitoring
- **Fast**: Detect trong < 1 phút (realtime CDC stream)

### Consequences
- Small performance overhead (~5-10ms per event for schema lookup)
- False positives nếu JSON có fields không cần persist → filter list

---

## Open Questions (Updated v2.0)

1. **Airbyte Table Classification**: DevOps cần cung cấp list bảng nào đi qua Airbyte, bảng nào qua Debezium ✅ (planned for Phase 1)
2. **PostgreSQL Cluster Sizing**: Cần benchmark để xác định resources (CPU/RAM/Storage) cho PostgreSQL cluster
3. **NATS JetStream Retention**: Bao lâu giữ messages trong JetStream? (Recommend: 7 days)
4. **Monitoring Thresholds**: Alert thresholds cho reconciliation drift? (Recommend: > 1% cho critical tables)
5. **CMS Authentication** (NEW): Integrate với existing SSO hoặc standalone? (Recommend: SSO với GooPay auth system)
6. **JSONB Backfill Strategy** (NEW): Auto-backfill từ `_raw_data` sau approve, hay manual trigger? (Recommend: Auto với async job)
7. **Mapping Rules Versioning** (NEW): Có cần version mapping rules không? (Recommend: Yes, để rollback nếu cần)

---

**Next Review Date**: After Phase 5 completion (CMS + Schema Drift Detection)

---

## ADR-012: Target-Table Based Indexing for Mapping Cache

**Date**: 2026-04-06
**Status**: Approved
**Deciders**: Brain, Muscle

### Context
Trong worker processing loop, `EventHandler` nhận `TargetTable` từ registry và cần tra cứu `MappingRule` để chuẩn hóa dữ liệu. Tuy nhiên, logic cũ trong `RegistryService.ReloadAll` lại index `mappingCache` theo `SourceTable` (tên bảng gốc tại MongoDB/MySQL), dẫn đến việc tra cứu theo `TargetTable` luôn trả về `nil`.

### Decision
Thống nhất sử dụng **TargetTable** làm khóa chính (Primary Index Key) cho toàn bộ in-memory cache trong `RegistryService`:
1. `registryCache`: `map[string]*model.TableRegistry` (Key = TargetTable)
2. `mappingCache`: `map[string][]model.MappingRule` (Key = TargetTable)

**Implementation Detail**:
- Khi `ReloadAll`, xây dựng một bảng tra cứu tạm thời (Intermediate lookup) `sourceToTarget` từ dữ liệu Registry.
- Ánh xạ các `MappingRule` (vốn chỉ chứa `source_table`) sang `TargetTable` tương ứng trước khi đưa vào cache.

### Rationale
- **Consistency**: Khớp với flow xử lý của `EventHandler`.
- **Performance**: Tra cứu O(1) thay vì phải scan toàn bộ rules hoặc mapping ngược lại mỗi khi có event.
- **Robustness**: `RegistryService` trở thành "Source of Truth" duy nhất cho việc định tuyến dữ liệu.

### Consequences
- **Positive**: Sửa lỗi triệt để việc mất mapping rules khi xử lý event.
- **Negative**: Tăng nhẹ thời gian khởi tạo cache tại bước `ReloadAll` (O(N) với N là số lượng rules), nhưng không đáng kể so với lợi ích khi runtime.
- **Constraint**: Mọi mapping rule bắt buộc phải có một Registry entry tương ứng, nếu không rule đó sẽ bị bỏ qua và log cảnh báo.

