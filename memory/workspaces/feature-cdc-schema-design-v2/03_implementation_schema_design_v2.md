# Implementation Notes — Schema Design V2

## A. Current-State Findings From The Code

### 1. Current identity is table-centric, not resource-centric

- `internal/model/table_registry.go`
  - Một row hiện chứa lẫn:
    - source endpoint metadata
    - shadow target identity
    - sync policy
    - runtime status
    - recon metadata
  - Thiếu identity chuẩn cho `database/schema/collection`.
- `internal/model/mapping_rule.go`
  - Mapping rule vẫn neo vào `source_table`.
  - Điều này không đủ khi một `source_table` có:
    - nhiều master projections
    - nhiều destinations
    - namespace trùng tên ở source khác

### 2. Current runtime is single-sink by construction

- `config/config.go`
  - Chỉ có một `DBConfig`.
- `pkgs/database/postgres.go`
  - Chỉ mở một primary pool + một read replica pool.
- `internal/service/master_ddl_generator.go`
  - Tạo master vào `public.<master_name>`.
- `internal/service/transmuter.go`
  - Đọc từ `cdc_internal.<shadow>`, ghi vào `public.<master>`.

### 3. Current routing is built around `target_table`

- `internal/repository/registry_repo.go`
  - lookup theo `target_table`, filter theo `source_db`.
- `internal/service/registry_service.go`
  - cache:
    - `target_table -> registry`
    - `source_table -> registry`
    - `target_table -> mapping_rules`
- `internal/handler/event_handler.go`
  - resolve `source_table -> target_table`, sau đó insert vào một shadow table.

### 4. Current migrations already show design drift

- `001_init_schema.sql` tạo `cdc_table_registry`, `cdc_mapping_rules`, ... theo mindset 1 registry chính.
- `019_system_registry.sql` thêm `cdc_internal.table_registry`.
- `023_master_table_registry.sql` thêm `cdc_internal.master_table_registry`.
- `027_systematic_sources.sql` thêm `cdc_internal.sources`.

Kết luận: dự án đã bắt đầu tự cảm nhận được việc V1 không đủ, nhưng hiện đang tồn tại nhiều "nửa bước" chồng nhau mà chưa có model thống nhất.

## B. Design Principles For V2

1. `system` là control plane, không phải nơi chứa shadow/master payload.
2. Mọi object phải có identity đầy đủ theo engine.
3. Logical object và physical destination phải tách thành 2 lớp.
4. Routing phải đi qua binding, không hardcode trong registry gốc.
5. Runtime state phải tách khỏi static metadata để tránh một bảng gánh quá nhiều trách nhiệm.

## C. Proposed V2 Domain Model

### 1. Physical connection

Thực thể mô tả nơi kết nối vật lý:
- engine
- host/port
- database mặc định
- secret ref
- capability

### 2. Source object

Thực thể mô tả object logical ở source:
- source connection nào
- catalog/database nào
- schema/namespace nào
- object name nào
- object type gì
- primary key/timestamp/cdc mode

### 3. Shadow binding

Thực thể mô tả source object này được materialize về đâu ở layer shadow:
- source object nào
- destination connection nào
- destination catalog/schema/table
- write mode
- active state

### 4. Master binding

Thực thể mô tả projection/master nào được sinh ra từ source/shadow:
- source object hoặc shadow binding nào
- master connection nào
- destination catalog/schema/table
- transform spec
- lifecycle state

### 5. Runtime state

Tách state runtime ra riêng:
- sync watermark
- last success
- last error
- drift state
- DDL state
- recon state

## D. Proposed Metadata Schemas

### D.1. Schema separation

- `cdc_system`
  - control plane metadata
  - registry
  - bindings
  - rules
  - runtime states
  - logs
- `cdc_shadow_*` hoặc schema đích tùy binding
  - payload shadow tables
- `dw_*` hoặc schema đích tùy binding
  - payload master tables

Lưu ý: không dùng `cdc_internal` làm nghĩa mặc định cho shadow nữa. Nếu vẫn giữ `cdc_internal` trong giai đoạn chuyển tiếp, nó chỉ là một physical schema cụ thể của một shadow destination, không phải semantic layer name.

### D.2. Core tables

#### 1. `cdc_system.connection_registry`

Mục tiêu:
- quản lý endpoint vật lý
- tách secret và capability
- dùng chung cho source, shadow, master, system

```sql
CREATE TABLE cdc_system.connection_registry (
  id                   BIGSERIAL PRIMARY KEY,
  connection_code      VARCHAR(100) NOT NULL UNIQUE,
  display_name         VARCHAR(200) NOT NULL,
  role_type            VARCHAR(32)  NOT NULL
    CHECK (role_type IN ('source','shadow','master','system','mixed')),
  engine_type          VARCHAR(32)  NOT NULL
    CHECK (engine_type IN ('postgresql','mariadb','mysql','mongodb','clickhouse')),
  host                 VARCHAR(255),
  port                 INTEGER,
  default_database     VARCHAR(255),
  default_schema       VARCHAR(255),
  secret_ref           VARCHAR(255) NOT NULL,
  options_json         JSONB NOT NULL DEFAULT '{}'::jsonb,
  capabilities_json    JSONB NOT NULL DEFAULT '{}'::jsonb,
  status               VARCHAR(32) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','paused','failed','retired')),
  created_by           VARCHAR(100),
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

`capabilities_json` gợi ý:
- `{"supports_schema": true, "supports_upsert": true, "supports_jsonb": true}`

#### 2. `cdc_system.source_object_registry`

Mục tiêu:
- chuẩn hóa identity của source object
- thay thế vai trò trung tâm của `cdc_table_registry`

```sql
CREATE TABLE cdc_system.source_object_registry (
  id                        BIGSERIAL PRIMARY KEY,
  object_code               VARCHAR(150) NOT NULL UNIQUE,
  source_connection_id      BIGINT NOT NULL REFERENCES cdc_system.connection_registry(id),
  source_engine_type        VARCHAR(32)  NOT NULL,
  source_database           VARCHAR(255),
  source_schema             VARCHAR(255),
  source_namespace          VARCHAR(255),
  source_object_name        VARCHAR(255) NOT NULL,
  source_object_type        VARCHAR(32)  NOT NULL
    CHECK (source_object_type IN ('table','collection','view')),
  source_locator_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
  normalized_source_key     VARCHAR(500) NOT NULL UNIQUE,
  primary_key_field         VARCHAR(255) NOT NULL DEFAULT 'id',
  primary_key_type          VARCHAR(100),
  timestamp_field           VARCHAR(255),
  timestamp_candidates_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  cdc_mode                  VARCHAR(32) NOT NULL DEFAULT 'incremental'
    CHECK (cdc_mode IN ('snapshot','incremental','full_refresh','hybrid')),
  sync_engine               VARCHAR(32) NOT NULL DEFAULT 'debezium'
    CHECK (sync_engine IN ('debezium','airbyte','both','custom')),
  is_active                 BOOLEAN NOT NULL DEFAULT TRUE,
  profile_status            VARCHAR(32) NOT NULL DEFAULT 'draft'
    CHECK (profile_status IN ('draft','pending_data','syncing','active','failed','paused')),
  notes                     TEXT,
  created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Quy ước:
- PostgreSQL/MariaDB:
  - `source_database`: database/catalog
  - `source_schema`: schema
  - `source_namespace`: có thể copy từ schema để normalize
- MongoDB:
  - `source_database`: database
  - `source_schema`: NULL
  - `source_namespace`: database logical namespace
  - `source_object_name`: collection

#### 3. `cdc_system.shadow_binding`

Mục tiêu:
- route từng source object tới shadow destination riêng
- preserve namespace cha ở destination

```sql
CREATE TABLE cdc_system.shadow_binding (
  id                        BIGSERIAL PRIMARY KEY,
  binding_code              VARCHAR(150) NOT NULL UNIQUE,
  source_object_id          BIGINT NOT NULL REFERENCES cdc_system.source_object_registry(id) ON DELETE CASCADE,
  shadow_connection_id      BIGINT NOT NULL REFERENCES cdc_system.connection_registry(id),
  shadow_database           VARCHAR(255),
  shadow_schema             VARCHAR(255) NOT NULL,
  shadow_table              VARCHAR(255) NOT NULL,
  physical_table_fqn        VARCHAR(600) NOT NULL,
  namespace_strategy        VARCHAR(32) NOT NULL DEFAULT 'preserve'
    CHECK (namespace_strategy IN ('preserve','prefix','flatten','custom')),
  write_mode                VARCHAR(32) NOT NULL DEFAULT 'upsert'
    CHECK (write_mode IN ('upsert','append','replace')),
  ddl_status                VARCHAR(32) NOT NULL DEFAULT 'pending'
    CHECK (ddl_status IN ('pending','created','failed','drifted')),
  is_active                 BOOLEAN NOT NULL DEFAULT TRUE,
  created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (source_object_id, shadow_connection_id, shadow_schema, shadow_table)
);
```

Ví dụ:
- source: `billing.public.invoices`
- shadow binding:
  - connection: `postgres_shadow_a`
  - schema: `shadow_billing_public`
  - table: `invoices`

Hoặc Mongo:
- source: `wallet.transactions`
- shadow binding:
  - connection: `postgres_shadow_b`
  - schema: `shadow_wallet`
  - table: `transactions`

#### 4. `cdc_system.master_binding`

Mục tiêu:
- 1 source/shadow -> N master projections
- mỗi master có destination riêng

```sql
CREATE TABLE cdc_system.master_binding (
  id                        BIGSERIAL PRIMARY KEY,
  binding_code              VARCHAR(150) NOT NULL UNIQUE,
  source_object_id          BIGINT NOT NULL REFERENCES cdc_system.source_object_registry(id) ON DELETE CASCADE,
  shadow_binding_id         BIGINT REFERENCES cdc_system.shadow_binding(id) ON DELETE SET NULL,
  master_connection_id      BIGINT NOT NULL REFERENCES cdc_system.connection_registry(id),
  master_database           VARCHAR(255),
  master_schema             VARCHAR(255) NOT NULL,
  master_table              VARCHAR(255) NOT NULL,
  physical_table_fqn        VARCHAR(600) NOT NULL,
  transform_type            VARCHAR(32) NOT NULL
    CHECK (transform_type IN ('copy_1_to_1','filter','aggregate','group_by','join','custom_sql')),
  transform_spec            JSONB NOT NULL DEFAULT '{}'::jsonb,
  schema_status             VARCHAR(32) NOT NULL DEFAULT 'pending_review'
    CHECK (schema_status IN ('pending_review','approved','rejected','failed','drifted')),
  is_active                 BOOLEAN NOT NULL DEFAULT FALSE,
  schema_reviewed_by        VARCHAR(100),
  schema_reviewed_at        TIMESTAMPTZ,
  rejection_reason          TEXT,
  created_by                VARCHAR(100),
  created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (master_connection_id, master_schema, master_table)
);
```

#### 5. `cdc_system.mapping_rule_v2`

Mục tiêu:
- bind rule vào source object + master binding
- không nhầm giữa nhiều master khác nhau của cùng một source

```sql
CREATE TABLE cdc_system.mapping_rule_v2 (
  id                        BIGSERIAL PRIMARY KEY,
  source_object_id          BIGINT NOT NULL REFERENCES cdc_system.source_object_registry(id) ON DELETE CASCADE,
  master_binding_id         BIGINT REFERENCES cdc_system.master_binding(id) ON DELETE CASCADE,
  source_field              VARCHAR(255) NOT NULL,
  source_path               VARCHAR(500),
  target_column             VARCHAR(255) NOT NULL,
  data_type                 VARCHAR(100) NOT NULL,
  source_format             VARCHAR(32) NOT NULL DEFAULT 'raw'
    CHECK (source_format IN ('raw','jsonpath','expression')),
  transform_fn              VARCHAR(100),
  is_nullable               BOOLEAN NOT NULL DEFAULT TRUE,
  default_value             TEXT,
  is_active                 BOOLEAN NOT NULL DEFAULT TRUE,
  status                    VARCHAR(32) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','approved','rejected')),
  created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (source_object_id, COALESCE(master_binding_id, 0), target_column)
);
```

Ghi chú:
- PostgreSQL không cho `COALESCE(master_binding_id, 0)` trong `UNIQUE` constraint trực tiếp; khi implement thật nên dùng `UNIQUE INDEX`.

#### 6. `cdc_system.sync_runtime_state`

Mục tiêu:
- tách runtime state khỏi registry metadata

```sql
CREATE TABLE cdc_system.sync_runtime_state (
  id                        BIGSERIAL PRIMARY KEY,
  source_object_id          BIGINT NOT NULL REFERENCES cdc_system.source_object_registry(id) ON DELETE CASCADE,
  shadow_binding_id         BIGINT REFERENCES cdc_system.shadow_binding(id) ON DELETE CASCADE,
  master_binding_id         BIGINT REFERENCES cdc_system.master_binding(id) ON DELETE CASCADE,
  runtime_scope             VARCHAR(32) NOT NULL
    CHECK (runtime_scope IN ('source','shadow','master')),
  last_success_at           TIMESTAMPTZ,
  last_error_at             TIMESTAMPTZ,
  last_error_message        TEXT,
  last_cursor_json          JSONB,
  last_source_ts            BIGINT,
  last_recon_at             TIMESTAMPTZ,
  recon_drift_count         BIGINT NOT NULL DEFAULT 0,
  ddl_status                VARCHAR(32),
  stats_json                JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### 7. `cdc_system.schema_drift_log_v2`

Tách drift log theo object/binding identity mới, thay vì chỉ theo `table_name`.

## E. Naming Strategy

### E.1. Không dùng `target_table` như canonical key

Thay bằng:
- `object_code` cho source logical object
- `binding_code` cho shadow/master route
- `physical_table_fqn` cho nơi lưu thật

### E.2. Gợi ý `normalized_source_key`

- PostgreSQL/MariaDB:
  - `<engine>:<connection_code>:<database>:<schema>:<table>`
- MongoDB:
  - `mongodb:<connection_code>:<database>:<collection>`

Ví dụ:
- `postgresql:payment_prod:billing:public:invoices`
- `mongodb:wallet_cluster:wallet:transactions`

## F. Backward-Compatible Migration Mapping

### F.1. Từ `cdc_table_registry` sang `source_object_registry` + `shadow_binding`

| V1 field | V2 target | Ghi chú |
|---|---|---|
| `source_db` | `source_database` hoặc `source_namespace` | cần chuẩn hóa theo engine |
| `source_type` | `source_engine_type` | đổi tên cho rõ |
| `source_table` | `source_object_name` | chỉ là phần object name, không đủ làm key |
| `target_table` | `shadow_table` | chỉ đúng với route shadow mặc định cũ |
| `primary_key_field` | `primary_key_field` | giữ |
| `primary_key_type` | `primary_key_type` | giữ |
| `timestamp_field*` | `timestamp_field*` | giữ, chuyển sang source object |
| `sync_engine` | `sync_engine` | giữ |
| `is_active` | `source_object.is_active` + `shadow_binding.is_active` | cần tách |
| `is_table_created` | `shadow_binding.ddl_status` | đổi sang trạng thái rõ nghĩa |

### F.2. Từ `cdc_internal.master_table_registry` sang `master_binding`

| V1 field | V2 target |
|---|---|
| `master_name` | `master_table` hoặc `binding_code` |
| `source_shadow` | `shadow_binding_id` hoặc `source_object_id` |
| `transform_type` | `transform_type` |
| `spec` | `transform_spec` |
| `schema_status` | `schema_status` |
| `is_active` | `is_active` |

### F.3. Từ `cdc_mapping_rules` sang `mapping_rule_v2`

| V1 field | V2 target |
|---|---|
| `source_table` | `source_object_id` |
| `master_table` | `master_binding_id` |
| `source_field` | `source_field` |
| `jsonpath` | `source_path` |
| `target_column` | `target_column` |
| `data_type` | `data_type` |
| `transform_fn` | `transform_fn` |
| `status` | `status` |

## G. Project-Level Solution Proposal

### G.1. New migrations to add

Đề xuất chuỗi migration mới:

1. `029_v2_connection_registry.sql`
2. `030_v2_source_object_registry.sql`
3. `031_v2_shadow_binding.sql`
4. `032_v2_master_binding.sql`
5. `033_v2_mapping_rule.sql`
6. `034_v2_runtime_state.sql`
7. `035_v2_backfill_from_legacy_registry.sql`

Không xóa migration cũ ở phase đầu.

### G.2. New model files

Thêm tại `internal/model/`:
- `connection_registry.go`
- `source_object_registry.go`
- `shadow_binding.go`
- `master_binding_v2.go`
- `mapping_rule_v2.go`
- `sync_runtime_state.go`

### G.3. Repository refactor

Tách `RegistryRepo` hiện tại thành:
- `source_object_repo.go`
- `shadow_binding_repo.go`
- `master_binding_repo.go`
- `connection_repo.go`

Lý do:
- Repo hiện tại đang ôm quá nhiều semantics vào một bảng.

### G.4. Registry service refactor

`internal/service/registry_service.go` cần đổi từ:
- cache theo `target_table`
- reverse lookup theo `source_table`

Sang:
- cache `sourceObjectID -> source object`
- cache `normalizedSourceKey -> source object`
- cache `sourceObjectID -> active shadow binding`
- cache `sourceObjectID -> active master bindings[]`
- cache `masterBindingID -> mapping rules[]`

### G.5. Connection manager

Tạo mới `internal/service/connection_manager.go`

Trách nhiệm:
- load `connection_registry`
- resolve secret từ `secret_ref`
- mở pool theo engine + connection id
- cache/reuse pool
- expose:
  - `GetSystemDB()`
  - `GetShadowDB(bindingID)`
  - `GetMasterDB(bindingID)`
  - `GetSourceReader(sourceObjectID)` nếu cần backfill/recon

### G.6. Event ingest refactor

`internal/handler/event_handler.go`

Luồng mới:
1. Parse event -> resolve source identity đầy đủ.
2. Map sang `source_object_registry` bằng `normalizedSourceKey`, không chỉ `source_table`.
3. Load active `shadow_binding`.
4. Lấy đúng destination DB/schema/table từ binding.
5. Upsert shadow row bằng connection manager.

### G.7. Master DDL generator refactor

`internal/service/master_ddl_generator.go`

Hiện tại:
- hardcode `public.<master>`

V2:
- load `master_binding`
- validate `master_schema`, `master_table`
- tạo `CREATE SCHEMA IF NOT EXISTS <schema>`
- apply DDL trên đúng `master_connection_id`

### G.8. Transmuter refactor

`internal/service/transmuter.go`

Hiện tại:
- đọc từ `cdc_internal.<shadow>`
- ghi vào `public.<master>`

V2:
- đọc `shadow_binding`
- mở đúng shadow DB/schema/table
- load `master_binding`
- mở đúng master DB/schema/table
- mapping rules load theo `master_binding_id`

### G.9. Recon / DLQ / schema inspector

Các module:
- `recon_handler.go`
- `recon_heal.go`
- `schema_inspector.go`
- `backfill_source_ts.go`

đều phải đổi key từ `target_table` sang:
- `source_object_id`
- `shadow_binding_id`
- `master_binding_id`

Ít nhất ở phase chuyển tiếp, các payload NATS cần support cả 2:
- legacy: `target_table`
- v2: `binding_code` / `source_object_id`

## H. Rollout Plan

### Phase 1 — Add V2 metadata side-by-side

- thêm các bảng `cdc_system.*`
- backfill từ `cdc_table_registry`, `cdc_internal.master_table_registry`, `cdc_mapping_rules`
- chưa đổi luồng runtime chính

### Phase 2 — Read V2 metadata, write legacy payload

- `RegistryService` đọc song song V1 + V2
- build compatibility adapters:
  - V2 source object -> legacy target_table

### Phase 3 — Move shadow writes to binding-based routing

- `EventHandler`
- `BatchBuffer`
- `Upsert` path

### Phase 4 — Move master DDL and transmute

- `MasterDDLGenerator`
- `TransmuterModule`
- `TransmuteScheduler`

### Phase 5 — Migrate recon and ops commands

- `ReconHandler`
- `CommandHandler`
- `SchemaInspector`
- `BackfillSourceTS`

### Phase 6 — Deprecate legacy tables

- khóa tạo mới từ `cdc_table_registry`
- chuyển UI/API sang V2
- chỉ giữ compatibility view nếu cần

## I. Risks And Guards

1. Risk: trùng logical name giữa nhiều source
   - Guard: `normalized_source_key` unique
2. Risk: runtime vẫn gửi payload chỉ có `target_table`
   - Guard: compatibility adapter trong phase 2-5
3. Risk: nhiều connection làm pool nở quá lớn
   - Guard: LRU/TTL pool manager + max pools cấu hình
4. Risk: hardcoded SQL schema name ở nhiều nơi
   - Guard: grep/refactor checklist trước rollout
5. Risk: mapping rule migrate sai master
   - Guard: migration backfill cần join theo `master_table` cũ nếu có
