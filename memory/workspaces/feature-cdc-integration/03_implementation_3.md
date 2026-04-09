# Phase 3: Tối ưu ELT — Hoàn thiện Airbyte ↔ CMS Sync + Kích hoạt Worker Transform

> **Date**: 2026-04-08  
> **Decision**: Tối ưu Phase 1, đóng kín vòng lặp đồng bộ Airbyte ↔ CMS.  
> **Debezium**: KHÔNG bỏ — defer sang Phase 2 (`03_implementation_phase_2.md`). CDC trong Airbyte không đạt realtime, Debezium standalone sẽ kích hoạt riêng khi cần true realtime.  
> **dbt**: KHÔNG thêm — CDC Worker **đã là** transformation layer.  
> **Plan thực thi**: Xem `02_plan_v1.11.md`

---

## 1. Pipeline ELT hiện tại — Đã hoạt động

```
┌─────────────┐         ┌──────────────┐         ┌──────────────────────────┐
│  Source DBs  │──CDC───►│   Airbyte    │──Load──►│     PostgreSQL DW        │
│  MongoDB     │ Oplog   │  (E + L)     │  raw    │  _raw_data JSONB         │
│  MySQL       │ Binlog  │              │         │  + _airbyte_* metadata   │
└─────────────┘         └──────────────┘         └────────────┬─────────────┘
                                                               │
                                                    ┌──────────▼──────────┐
                                                    │    CDC Worker       │
                                                    │    (Transform)      │
                                                    │                     │
                                                    │  mapping_rules →    │
                                                    │  extract from       │
                                                    │  _raw_data JSONB →  │
                                                    │  typed columns      │
                                                    │  + hash dedup       │
                                                    │  + schema drift     │
                                                    └─────────────────────┘
```

### Tại sao CDC Worker đã là Transform layer

| Chức năng dbt | CDC Worker đã làm | File |
|---------------|-------------------|------|
| Extract từ JSON | `event_handler.go:80-88` — đọc `_raw_data[field]` theo mapping_rules | event_handler.go |
| Type casting | `command_handler.go:buildCastExpr()` — `::INTEGER`, `::NUMERIC`, `::TIMESTAMP` | command_handler.go |
| Field renaming | `cdc_mapping_rules` table: `source_field` → `target_column` | mapping_rule_repo.go |
| Incremental load | Hash-based change detection: `WHERE _hash IS DISTINCT FROM EXCLUDED._hash` | batch_buffer.go |
| Backfill missing data | `HandleBackfill` — populate NULL columns từ `_raw_data` | command_handler.go |
| Schema discovery | `HandleDiscover` — scan `information_schema` → auto-create rules | command_handler.go |
| Data quality | `_raw_data` JSONB preserved → luôn có thể verify/reprocess | batch_buffer.go |

**Thêm dbt = thêm 1 hop thừa:**
```
❌  Airbyte → PG → dbt transforms → PG (2 lần write)
✅  Airbyte → PG(_raw_data) → CDC Worker transforms in-place (1 lần write)
```

---

## 2. Hạ tầng hiện tại — 4 Services

### 2.1 Service Map

| # | Service | Project | Port | Role | Tech |
|---|---------|---------|------|------|------|
| 1 | Auth Service | `cdc-auth-service` | :8081 | Login, JWT, RBAC | Go |
| 2 | CDC Worker | `centralized-data-service` | :8082 | **Transform layer** — consume events, mapping, upsert, schema inspect | Go |
| 3 | CMS API | `cdc-cms-service` | :8083 | Config management, Airbyte proxy, approval workflow | Go/Fiber |
| 4 | CMS Frontend | `cdc-cms-web` | :5173 | UI quản trị | React + Ant Design |

### 2.2 External Dependencies

| Service | Port | Role |
|---------|------|------|
| PostgreSQL | :5432 | DW (goopay_dw) — cả metadata tables + CDC data tables |
| NATS JetStream | :4222 | Async commands, config reload, CDC events |
| Redis | :6379 | Schema cache (TTL 5m) |
| Airbyte | :18000 | Extract + Load engine |

### 2.3 Database Schema

| Table | Owner | Mục đích |
|-------|-------|----------|
| `cdc_table_registry` | CMS API | Registry: source → target table mapping |
| `cdc_mapping_rules` | CMS API | Field mapping: source_field → target_column + data_type |
| `pending_fields` | Worker → CMS | Schema drift: fields phát hiện trong source chưa có mapping |
| `schema_changes_log` | CMS API | Audit: ALTER TABLE history |
| `cdc_*` (dynamic) | Worker | CDC data tables: `_raw_data` JSONB + mapped typed columns |

---

## 3. Luồng dữ liệu chi tiết

### 3.1 Extract + Load (Airbyte)

```
MongoDB (goopay_wallet) ──Oplog CDC──► Airbyte Source Connector
                                           │
                                           ▼
                                    Airbyte Connection
                                    (SyncCatalog: streams + config)
                                           │
                                           ▼
                                    Airbyte Destination (PostgreSQL)
                                           │
                                           ▼
                                    PostgreSQL goopay_dw
                                    Table: cdc_merchants
                                    ├─ _raw_data JSONB  ← full document
                                    ├─ _airbyte_ab_id
                                    └─ _airbyte_emitted_at
```

**Airbyte ghi trực tiếp vào PostgreSQL** — không qua NATS, không qua Worker.

### 3.2 Transform (CDC Worker)

```
NATS "cdc.goopay.{db}.{table}" ──► event_handler.go
                                        │
                                        ├─ 1. Lookup table config (sourceCache → TargetTable)
                                        ├─ 2. Schema inspect (detect new fields → pending_fields)
                                        ├─ 3. Apply mapping_rules (source_field → typed target_column)
                                        ├─ 4. Calculate hash (dedup)
                                        └─ 5. Add to batch_buffer
                                              │
                                              ▼
                                        batch_buffer.go (flush mỗi 500 records hoặc 2s)
                                              │
                                              ▼
                                        PostgreSQL UPSERT
                                        INSERT INTO cdc_merchants (id, name, email, _raw_data, _hash, ...)
                                        ON CONFLICT (id) DO UPDATE SET ...
                                        WHERE _hash IS DISTINCT FROM EXCLUDED._hash
```

### 3.3 On-demand Transform (NATS Commands)

| Command | Trigger | What it does |
|---------|---------|-------------|
| `cdc.cmd.standardize` | CMS API → NATS | Thêm metadata columns (_raw_data, _source, ...) vào CDC table |
| `cdc.cmd.discover` | CMS API → NATS | Scan `information_schema.columns` → auto-create mapping_rules |
| `cdc.cmd.backfill` | CMS API → NATS | `UPDATE table SET col = (_raw_data->>'field')::TYPE WHERE col IS NULL` |
| `cdc.cmd.scan-raw-data` | CMS API → NATS (req/reply) | `SELECT DISTINCT jsonb_object_keys(_raw_data)` → so sánh mapping_rules |
| `cdc.cmd.introspect` | CMS API → NATS (req/reply) | Airbyte DiscoverSchema → so sánh mapping_rules |
| `schema.config.reload` | CMS API → NATS | Worker reload mapping cache từ DB |

---

## 4. API Endpoints — Chi tiết từng endpoint

### 4.1 Airbyte Integration

#### `GET /api/airbyte/sources` — Liệt kê sources
- **Auth**: Không cần JWT
- **Airbyte API**: `POST /api/v1/sources/list` (workspaceId)
- **Response**:
  ```json
  [
    {
      "sourceId": "uuid",
      "name": "goopay-wallet-mongo",
      "sourceName": "MongoDB V2",
      "workspaceId": "uuid",
      "database": "goopay_wallet"
    }
  ]
  ```
- **Lưu vào DB**: Không. Read-only.

#### `GET /api/airbyte/jobs` — Jobs sync gần nhất
- **Auth**: Không cần JWT
- **Airbyte API**: `POST /api/v1/jobs/list` per connection (sequential)
- **Response**: `[{jobId, status, createdAt, updatedAt, connectionName}]`
- **Hạn chế**: Sequential, hardcoded top 20

#### `GET /api/airbyte/sync-audit` — So sánh CMS vs Airbyte
- **Auth**: Không cần JWT
- **Airbyte API**: `GetConnection()` per registry entry
- **Response**: `[{connectionId, tableName, cmsActive, airbyteSelected, mismatch}]`

#### `GET /api/airbyte/import/list` — Streams chưa import
- **Auth**: Không cần JWT
- **Airbyte API**: `ListConnections()` → filter streams chưa có trong registry
- **Response**: `[{connectionId, name, stream, namespace, isRegistered}]`

#### `POST /api/airbyte/import/execute` — Import streams vào registry
- **Auth**: Không cần JWT
- **Body**: `{connection_id: "uuid", stream_names: ["merchants", "orders"]}`
- **Airbyte API**: `GetConnection()`
- **Side effects**: Tạo `cdc_table_registry` + default `cdc_mapping_rules` (field "id") + CDC table + NATS reload
- **Response**: `{message, imported_count}` (201)

### 4.2 Registry Management

#### `GET /api/registry` — Danh sách tables
- **Auth**: admin + operator
- **Query**: `source_db`, `sync_engine`, `priority`, `is_active`, `page`, `page_size`
- **Response**: `{data: TableRegistry[], total, page}`

#### `GET /api/registry/stats` — Thống kê
- **Auth**: admin + operator
- **Response**: `{total, by_source_db: {}, by_sync_engine: {}, by_priority: {}, tables_created}`

#### `GET /api/registry/:id/status` — Trạng thái Airbyte sync
- **Auth**: admin + operator
- **Airbyte API**: `GetConnection()`
- **Response**: `{connection_id, status, name, stream_enabled}`

#### `POST /api/registry` — Đăng ký table mới
- **Auth**: admin only
- **Body**:
  ```json
  {
    "source_db": "goopay_wallet",
    "source_type": "mongodb",
    "source_table": "merchants",
    "target_table": "cdc_merchants",
    "sync_engine": "airbyte",
    "priority": "normal",
    "primary_key_field": "_id",
    "primary_key_type": "VARCHAR(36)"
  }
  ```
- **Airbyte API**: `ListSources()` → match by DB name → `DiscoverSchema()` → `ListConnections()` → `GetConnection()` → `UpdateConnection()` (enable stream)
- **Side effects**: PL/pgSQL `create_cdc_table()`, NATS reload
- **Response**: `{message, data: TableRegistry}` (201)
- **Default**: `is_active = false`

#### `PATCH /api/registry/:id` — Update table config
- **Auth**: admin only
- **Body**: `{sync_engine?, sync_interval?, priority?, is_active?, notes?}`
- **Airbyte API**: Nếu `is_active` thay đổi → `GetConnection()` → `DiscoverSchema()` (fallback) → `UpdateConnection()` (toggle stream selection)
- **NATS**: `schema.config.reload`

#### `POST /api/registry/batch` — Bulk register
- **Auth**: admin only
- **Body**: `[]TableRegistry`
- **Side effects**: `create_all_pending_cdc_tables()`, NATS reload "*"

#### `POST /api/registry/:id/sync` — Manual trigger sync
- **Auth**: admin only
- **Airbyte API**: `TriggerSync(connectionId)`
- **Response**: `{message, job_id}`

#### `GET /api/registry/:id/jobs` — Job history cho 1 table
- **Auth**: admin only
- **Airbyte API**: `ListJobs(connectionId, limit=10)`

#### `POST /api/registry/:id/standardize` — Chuẩn hóa CDC table
- **Auth**: admin only
- **NATS**: `cdc.cmd.standardize` → Worker gọi `standardize_cdc_table()` PL/pgSQL
- **Response**: 202 Accepted (async)

#### `POST /api/registry/:id/discover` — Auto-discover mapping rules
- **Auth**: admin only
- **NATS**: `cdc.cmd.discover` → Worker scan `information_schema.columns`
- **Response**: 202 Accepted (async)

#### `POST /api/registry/:id/refresh-catalog` — Refresh Airbyte catalog
- **Auth**: admin only
- **Airbyte API**: `DiscoverSchema(sourceId)`

#### `POST /api/registry/:id/scan-fields` — Scan source fields từ Airbyte
- **Auth**: admin only
- **Airbyte API**: `DiscoverSchema(sourceId)` → extract fields từ JSONSchema → tạo pending mapping_rules
- **Response**: `{message, added, total}`
- **Lưu ý**: Vi phạm service boundary — gọi Airbyte trực tiếp thay vì delegate Worker

#### `POST /api/registry/scan-source` — Discover all streams trong 1 source
- **Auth**: admin only
- **Query**: `source_id`
- **Airbyte API**: `DiscoverSchema(sourceId)` → bulk register streams

### 4.3 Schema Changes / Approval

#### `GET /api/schema-changes/pending` — Pending schema drift
- **Auth**: admin + operator
- **Query**: `status`, `source_db`, `table`, `page`, `page_size`
- **Response**: `{data: PendingField[], total, page}`

#### `GET /api/schema-changes/history` — Audit log
- **Auth**: admin + operator
- **Response**: `{data: SchemaChangeLog[], count}`

#### `POST /api/schema-changes/:id/approve` — Duyệt pending field
- **Auth**: admin only
- **Body**: `{target_column_name: "merchant_name", final_type: "TEXT", approval_notes?: ""}`
- **Side effects**: `ALTER TABLE ADD COLUMN` → tạo mapping_rule → NATS reload → Airbyte refresh (nếu có source_id)
- **Transaction**: Atomic

#### `POST /api/schema-changes/:id/reject` — Từ chối
- **Auth**: admin only
- **Body**: `{rejection_reason: "Not needed"}`

### 4.4 Mapping Rules

#### `GET /api/mapping-rules` — Danh sách rules
- **Auth**: admin + operator
- **Query**: `table` | `source_table` | `table_name`, `status`, `rule_type`, `page`, `page_size`
- **Response**: `{data: MappingRule[], count, total, page, page_size}`

#### `POST /api/mapping-rules` — Tạo rule mới (custom mapping)
- **Auth**: admin only
- **Body**:
  ```json
  {
    "source_table": "merchants",
    "source_field": "business_name",
    "target_column": "business_name",
    "data_type": "TEXT",
    "is_active": true,
    "is_enriched": false
  }
  ```

#### `PATCH /api/mapping-rules/:id` — Update status
- **Auth**: admin only
- **Body**: `{status: "approved"}`
- **Side effects**: NATS reload

#### `POST /api/mapping-rules/reload` — Signal workers reload
- **Auth**: admin only
- **Query**: `table?`
- **NATS**: `schema.config.reload`

#### `POST /api/mapping-rules/:id/backfill` — Backfill data
- **Auth**: admin only
- **NATS**: `cdc.cmd.backfill` → Worker `UPDATE table SET col = (_raw_data->>'field')::TYPE WHERE col IS NULL`
- **Response**: 202 Accepted (async)

### 4.5 Introspection

#### `GET /api/introspection/scan/:table` — Scan unmapped fields
- **Auth**: admin + operator
- **Flow**: Primary: NATS `cdc.cmd.scan-raw-data` (JSONB scan) → Fallback: NATS `cdc.cmd.introspect` (Airbyte)
- **Response**: `{status, table, total_raw_keys, mapped_count, new_fields: ["field1", "field2"]}`

#### `GET /api/introspection/scan-raw/:table` — Scan _raw_data trực tiếp
- **Auth**: admin + operator
- **NATS**: `cdc.cmd.scan-raw-data`
- **Response**: `{status, table, source_table, total_raw_keys, mapped_count, new_fields[]}`

---

## 5. Đồng bộ Airbyte ↔ Hệ thống — Hiện trạng & Gaps

### 5.1 Ma trận đồng bộ

| Entity | Airbyte→CMS (Import) | CMS→Airbyte (Push) | Detect thay đổi | Lưu vào DB? |
|--------|---------------------|--------------------|--------------------|-------------|
| **Sources** | `GET /api/airbyte/sources` — read-only | Không | Không cần | Không |
| **Destinations** | **Chưa có API** | Không | Không cần | Không |
| **Connections** | Read qua `GetConnection()` | Không | `sync-audit` | Không (chỉ lưu ID) |
| **Streams** | `import/execute` → registry | `is_active` toggle → `UpdateConnection()` | `import/list` (manual) | Có (`cdc_table_registry`) |
| **Field Mapping** | `scan-fields`, `scan-raw-data` → mapping_rules | **Chưa push** | `scan-raw-data` | Có (`cdc_mapping_rules`) |

### 5.2 Gaps cần xử lý

#### Gap 1: Destinations API (LOW)
- **Hiện trạng**: Không có endpoint nào cho destinations
- **Cần**: `GET /api/airbyte/destinations` — read-only, hiển thị trên UI
- **Effort**: 0.5 ngày

#### Gap 2: Connections detail (LOW)
- **Hiện trạng**: Chỉ lưu `airbyte_connection_id`, không show detail
- **Cần**: `GET /api/airbyte/connections/:id` — show schedule, sync mode, destination
- **Effort**: 0.5 ngày

#### Gap 3: Stream auto-sync (HIGH)
- **Hiện trạng**: Khi Airbyte thêm stream mới, CMS không biết. Phải manual `import/execute`.
- **Cần**: Periodic scan hoặc webhook → auto-create registry entry (is_active=false)
- **Effort**: 2 ngày

#### Gap 4: Stream config sync (MEDIUM)
- **Hiện trạng**: Registry chỉ lưu `is_active`. Không lưu sync_mode, cursor_field.
- **Cần**: Thêm fields vào `cdc_table_registry`: `sync_mode`, `destination_sync_mode`, `cursor_field`
- **Effort**: 1 ngày

#### Gap 5: Field mapping auto-detect (HIGH — CORE)
- **Hiện trạng**: Scan chỉ chạy khi user bấm nút. Không tự động detect fields mới.
- **Cần**: Periodic scan `_raw_data` → auto-create `pending_fields` hoặc `mapping_rules` (status=pending)
- **Effort**: 2 ngày

#### Gap 6: Field mapping push to Airbyte (MEDIUM)
- **Hiện trạng**: Khi toggle `is_active` trên mapping rule → chỉ update DB. Không push lên Airbyte.
- **Cần**: Sau khi update mapping rule → sync field selection vào Airbyte Connection SyncCatalog
- **Effort**: 2 ngày
- **Lưu ý**: Airbyte field selection chỉ ở mức stream (selected/deselected). Không có per-field toggle trong Airbyte API v1.

---

## 6. Data Models — Đầy đủ

### 6.1 `cdc_table_registry`

| Column | Type | Default | Mô tả |
|--------|------|---------|-------|
| `id` | SERIAL | PK | |
| `source_db` | VARCHAR | NOT NULL | Database nguồn (goopay_wallet) |
| `source_type` | VARCHAR | NOT NULL | mongodb / mysql / postgresql |
| `source_table` | VARCHAR | NOT NULL | Table/collection nguồn |
| `target_table` | VARCHAR | NOT NULL | Table đích trong DW |
| `sync_engine` | VARCHAR | 'airbyte' | airbyte / debezium / both |
| `sync_interval` | VARCHAR | '1h' | Tần suất |
| `priority` | VARCHAR | 'normal' | critical / high / normal / low |
| `primary_key_field` | VARCHAR | 'id' | PK field |
| `primary_key_type` | VARCHAR | 'VARCHAR(36)' | PK SQL type |
| `is_active` | BOOLEAN | **false** | Đang sync |
| `is_table_created` | BOOLEAN | false | CDC table đã tạo |
| `airbyte_connection_id` | VARCHAR | NULL | Airbyte connection UUID |
| `airbyte_source_id` | VARCHAR | NULL | Airbyte source UUID |
| `notes` | TEXT | NULL | |
| `created_at` | TIMESTAMP | NOW() | |
| `updated_at` | TIMESTAMP | NOW() | |

### 6.2 `cdc_mapping_rules`

| Column | Type | Default | Mô tả |
|--------|------|---------|-------|
| `id` | SERIAL | PK | |
| `source_table` | VARCHAR | NOT NULL | Table nguồn |
| `source_field` | VARCHAR | NOT NULL | Field nguồn (key trong _raw_data) |
| `target_column` | VARCHAR | NOT NULL | Column đích trong DW |
| `data_type` | VARCHAR | NOT NULL | SQL type: TEXT, BIGINT, NUMERIC, BOOLEAN, TIMESTAMP, JSONB |
| `is_active` | BOOLEAN | true | Mapping đang active |
| `is_enriched` | BOOLEAN | false | Cần enrichment |
| `is_nullable` | BOOLEAN | true | Cho phép NULL |
| `default_value` | VARCHAR | NULL | Default SQL value |
| `enrichment_function` | VARCHAR | NULL | |
| `status` | VARCHAR | 'pending' | pending / approved / rejected |
| `rule_type` | VARCHAR | 'mapping' | system / discovered / mapping |
| `created_by` | VARCHAR | NULL | Username |
| `updated_by` | VARCHAR | NULL | Username |
| `notes` | TEXT | NULL | |
| `created_at` | TIMESTAMP | NOW() | |
| `updated_at` | TIMESTAMP | NOW() | |

**UNIQUE**: `(source_table, source_field)`

### 6.3 `pending_fields`

| Column | Type | Mô tả |
|--------|------|-------|
| `id` | SERIAL PK | |
| `tbl_name` | VARCHAR | Target table trong DW |
| `source_db` | VARCHAR | Database nguồn |
| `field_name` | VARCHAR | Field mới phát hiện |
| `sample_value` | TEXT | Giá trị mẫu |
| `suggested_type` | VARCHAR | Type inferred |
| `final_type` | VARCHAR | Admin-approved type |
| `status` | VARCHAR | pending / approved / rejected / applied |
| `detected_at` | TIMESTAMP | |
| `detection_count` | INT | Số lần detect |
| `reviewed_by` | VARCHAR | |
| `target_column_name` | VARCHAR | Column name khi approve |
| `approval_notes` | TEXT | |
| `rejection_reason` | TEXT | |

### 6.4 `schema_changes_log`

| Column | Type | Mô tả |
|--------|------|-------|
| `id` | SERIAL PK | |
| `tbl_name` | VARCHAR | Target table |
| `change_type` | VARCHAR | ADD COLUMN / DROP / MODIFY |
| `field_name` | VARCHAR | Column affected |
| `sql_executed` | TEXT | Actual SQL |
| `status` | VARCHAR | pending / success / failed / rolled_back |
| `executed_by` | VARCHAR | Admin username |
| `executed_at` | TIMESTAMP | |
| `rollback_sql` | TEXT | Undo SQL |
| `execution_duration_ms` | INT | |
| `error_message` | TEXT | |

### 6.5 CDC Table Template (dynamic)

| Column | Type | Mô tả |
|--------|------|-------|
| `id` | VARCHAR(36) PK | Normalized PK (MongoDB _id → id) |
| `_raw_data` | JSONB NOT NULL | Full raw document |
| `_source` | VARCHAR | airbyte / debezium |
| `_synced_at` | TIMESTAMP | Last sync |
| `_version` | BIGINT | Record version |
| `_hash` | VARCHAR | SHA256 dedup |
| `_deleted` | BOOLEAN | Soft delete |
| `_created_at` | TIMESTAMP | DW creation |
| `_updated_at` | TIMESTAMP | DW update |
| *(mapped columns)* | *(per mapping_rules)* | Dynamic typed columns |

**Indexes**: GIN on `_raw_data`, B-tree on `_synced_at`, `_source`

---

## 7. Quyết định kiến trúc — Phase 3

### ADR-012: Debezium standalone — Defer sang Phase 2
- **Decision**: Không deploy Debezium trong Phase 3. Giữ lại cho Phase 2 khi cần true realtime (<1s latency).
- **Rationale**: Airbyte CDC (batch 5-15 phút) đủ cho near-realtime. Debezium standalone cần deploy riêng, cấu hình phức tạp — defer khi core sync ổn.
- **Ref**: `03_implementation_phase_2.md`

### ADR-013: KHÔNG thêm dbt
- **Decision**: CDC Worker là transformation layer. dbt là redundancy.
- **Rationale**: Worker đã extract từ `_raw_data`, type casting, backfill, schema discovery. Thêm dbt = thêm 1 hop thừa.

### ADR-014: Sources/Destinations/Connections = read-only
- **Decision**: Chỉ GET hiển thị, không lưu vào DB, không modify từ CMS.
- **Rationale**: Airbyte UI là master cho infra config. CMS quản lý streams + field mappings.

### ADR-015: Focus Phase 3 = đồng bộ Streams + Field Mapping
- **Decision**: Đóng kín vòng lặp: Airbyte stream ↔ CMS registry, Airbyte fields ↔ CMS mapping_rules.
- **Rationale**: Đây là core value — biết data gì đang chảy, fields nào đang map, gaps ở đâu.

---

## 8. Known Issues

| # | Issue | Severity | File |
|---|-------|----------|------|
| 1 | `/refresh-catalog-unauth` — endpoint công khai không auth | HIGH | router.go:34 |
| 2 | Import default mapping chỉ tạo field "id" | MEDIUM | airbyte_handler.go:261 |
| 3 | Stream name normalization `_`/`-` không nhất quán | MEDIUM | registry_handler.go:482 |
| 4 | `scan-fields` gọi Airbyte trực tiếp (vi phạm service boundary) | MEDIUM | registry_handler.go:766 |
| 5 | Job fetching sequential, không parallel | LOW | airbyte_handler.go:77 |
| 6 | Async commands không có progress polling | LOW | — |
