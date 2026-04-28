# Gap Analysis — Systematic Connect→Master Flow (Automation-First)

> Date: 2026-04-24 · Stage: 2 (Codebase Exploration) · Muscle: claude-opus-4-7-1m
> Input: Boss directive 2026-04-24 — 3 Tasks + DoD "add 1 source mới → Master có data"
> Phase suffix: `systematic_flow`

---

## 0. Phạm vi

Tách ra khỏi "gap 1-6" của Registry/Masters trước đó. Đây là **luồng vận hành** (operational flow), không phải cleanup UI. Mục tiêu: biến Connect→Master từ chuỗi UI rời rạc → State Machine tự động.

---

## 1. Tóm tắt hạ tầng hiện có (đã verify)

### 1.1. Schema & DB
- **Database dùng chung**: `goopay_dw` (cdc-cms-service config-local.yml:10 = centralized-data-service). Hai service cùng connect, nhưng migrations numbering tách rời:
  - cdc-cms-service: `003_add_mapping_rule_status`, `004_bridge_columns`, `005_admin_actions`, `013_alerts`.
  - centralized-data-service: `001..026` (canonical).
- **Registry v1**: `public.cdc_table_registry` (001_init_schema.sql:12) — nơi Register Table hiện tại.
- **Registry v2** (Sprint 5): `cdc_internal.table_registry` (018_*.sql) — shadow layer registry, `is_active` gate.
- **Master Registry**: `cdc_internal.master_table_registry` (023) — master layer, `schema_status` + `is_active`.
- **Sonyflake foundation**: `cdc_internal.machine_id_seq`, `fencing_token_seq`, `worker_registry`, `claim_machine_id()`, `tg_fencing_guard()` (018).
- **Shadow DDL helper**: SQL function `create_cdc_table(target_table, pk_field, pk_type)` — tạo `public.<target>` hoặc `cdc_internal.<target>` với CDC system cols (001:148, sửa lại bởi 002:67 và 003:15).
- **Schema auto-ALTER**: Worker-side SchemaManager (SinkWorker) — auto ALTER TABLE khi field mới.
- **Master DDL**: MasterDDLGenerator (Sprint 5) — CREATE TABLE `public.<master>` từ master_table_registry.spec khi admin approve.

### 1.2. Backend APIs (cdc-cms-service)
| Endpoint | Status | File |
|:-|:-|:-|
| `POST /api/v1/system/connectors` (Create) | ✅ LIVE | `internal/api/system_connectors_handler.go:175` |
| `GET /api/v1/system/connectors` (List) | ✅ LIVE | `:44` |
| `GET /api/v1/system/connector-plugins` | ✅ LIVE | `:127` |
| `POST /api/registry` (Register) | ✅ LIVE | `internal/api/registry_handler.go:114` |
| `POST /api/registry/:id/create-default-columns` | ✅ LIVE | `:727` |
| `POST /api/tools/trigger-snapshot/:table` | ✅ LIVE | `reconciliation_handler.go:368` |
| `POST /api/v1/schedules/:id/run-now` | ✅ LIVE | Sprint 5 |
| `POST /api/v1/masters/:name/approve` | ✅ LIVE | master handler |
| **Schema Discovery** (list collections/fields từ Debezium topic) | ❌ **THIẾU** | — |
| **Wizard Draft persist** (POST/GET wizard state) | ❌ **THIẾU** | — |

### 1.3. Frontend (cdc-cms-web)
- **SourceToMasterWizard.tsx** (pages/) — 11 steps, nhưng **STATELESS**: `useState(current)`, F5 là mất. Mỗi step chỉ là Link điều hướng. Không fetch status, không persist draft.
- **TableRegistry.tsx** — form Register nhập TAY `source_db`, `source_table`, `target_table`, `primary_key_field`, `primary_key_type`. Không có dropdown source từ connector đã tạo.
- **SourceConnectors.tsx** — danh sách connector + lifecycle ops, **không link sang Register bước tiếp**.

### 1.4. Worker flow (centralized-data-service)
- `HandleCreateDefaultColumns` (command_handler.go:156) — subscribe NATS `cdc.cmd.create-default-columns`, check `tableExists`, gọi `create_cdc_table()`, ALTER thêm approved mapping rules, update `is_table_created=true`. **Async qua NATS — CMS không biết thành công/thất bại ngay.**
- SinkWorker (v1.25) — consume Kafka topic `cdc.goopay.<db>.<table>` → upsert shadow + auto-ALTER.

---

## 2. Gap Map — 3 Tasks Boss yêu cầu

### Task 1 — Connection Fingerprint persist

**Current**:
```go
// system_connectors_handler.go:175-199 Create()
// 1. Validate name/config/connector.class
// 2. Forward POST → kafka-connect /connectors
// 3. Return 201 với kafka-connect response
// → KHÔNG ghi gì vào local DB
```

**Gap**: Không có bảng `sources` local. Registry (TableRegistry.tsx) vì thế phải nhập tay `source_db` + `source_table` + topic prefix. User sai typo = silent failure khi Worker consume topic sai.

**Boss target**:
- Tạo bảng `public.sources` (hoặc `cdc_internal.sources`) lưu "Connection Fingerprint": `connector_name`, `source_type` (mongodb/mysql/postgres), `topic_prefix`, `server_address`, `database_include_list`, `collection_include_list`, `status`, `created_by`, timestamps.
- Registry form → dropdown select source → tự fill `source_db` + available collections.

### Task 2 — Shadow Automator (EnsureShadowTable)

**Current**:
```go
// registry_handler.go:114 Register()
// 1. DB insert vào cdc_table_registry
// 2. Publish NATS cdc.cmd.create-default-columns (async)
// 3. Return 202 ngay → client không biết DDL done chưa
```

```go
// command_handler.go:156 HandleCreateDefaultColumns
// Worker nhận NATS:
//   - check tableExists
//   - SELECT create_cdc_table() // 8 system cols
//   - ALTER thêm approved mapping rules
//   - UPDATE is_table_created=true
// → FAIL = log warning, không retry, không publish error back, client không biết
```

**Gap**:
- DDL là **async via NATS** → nếu Worker offline, registry mồ côi.
- Template shadow hiện tại (`create_cdc_table()`) đã có `_gpay_id BIGINT PK` + `_gpay_source_id VARCHAR UNIQUE` tương thích Boss SQL. Nhưng **không có** Postgres trigger tự sinh Sonyflake — ID do Go Worker generate rồi insert.
- Không có cơ chế **atomic rollback**: nếu ALTER fail giữa chừng, table mồ côi.

**Boss target**:
- `EnsureShadowTable(ctx, registry) error` — **synchronous** trong Register flow (hoặc có retry + status tracking).
- Template SQL idempotent (`IF NOT EXISTS` + Unique Constraint).
- Postgres Trigger sinh Sonyflake ID tự động (BEFORE INSERT on shadow table).

### Task 3 — Wizard State Machine

**Current**:
- SourceToMasterWizard.tsx dùng `useState(current)` — mất khi F5.
- Step 11 ("Activate + Schedule") chỉ link sang `/schedules` — không có "Atomic Swap".
- Không có thanh Progress thật (chỉ badge vẽ tay).
- Không có validation "step N chỉ chạy được nếu step N-1 done".

**Boss target**:
- Bảng `wizard_sessions` lưu draft (session_id, source_name, current_step, step_payload JSONB, status, created_by).
- Active Step: output step N → input step N+1 (connector_id → source_db → collections → registry_id → target_table → shadow_ready → …).
- Atomic Swap (step 11): `BEGIN; RENAME TABLE public.<master> TO <master>_old; RENAME TABLE <master>_new TO public.<master>; COMMIT;` (hoặc VIEW swap).

---

## 3. Ưu tiên & Order

Dựa DoD "add 1 Source mới → Master có data":

| P | Task | Effort | Blocker cho |
|:-|:-|:-|:-|
| P0 | **Migration** bảng `sources` + trigger Sonyflake template | 1-2h | Task 1 + 2 |
| P0 | **Task 1 BE**: `Create` persist sources + return fingerprint | 1h | Wizard step 1 |
| P0 | **Task 2 BE**: `EnsureShadowTable` service + gọi đồng bộ từ Register | 2-3h | Wizard step 3-4 |
| P1 | **Schema Discovery** endpoint (liệt kê collections từ connector config) | 2h | Registry dropdown |
| P1 | **Task 3a**: `wizard_sessions` migration + CRUD endpoints | 2-3h | FE state machine |
| P2 | **Task 3b FE**: Wizard tái cấu trúc — active step + API-driven | 3-4h | Final UX |
| P2 | **Atomic Swap** cho step 11 | 2h | Production-safe swap |

**P0-P1 MVP**: ~8-10h. Full P0-P2: ~14-16h.

---

## 4. Rủi ro & Giả định cần Boss confirm

### 4.1. Schema template conflict
Boss SQL template:
```sql
CREATE TABLE IF NOT EXISTS %s_shadow (
    _gpay_id BIGINT PRIMARY KEY,
    _gpay_source_id VARCHAR(255) UNIQUE,
    data JSONB,
    _gpay_sync_ts TIMESTAMP DEFAULT NOW()
);
```

Hiện tại `create_cdc_table()` tạo NHIỀU cột hơn: `_raw_data JSONB`, `_source TEXT`, `_created_at/_updated_at/_deleted`, `_hash`, `_airbyte_raw_id`, `_airbyte_extracted_at`. Đây là schema CDC chuẩn đã chạy 1M rows thực tế.

→ **Câu hỏi Q1**: Boss muốn **thay thế** template hay **giữ nguyên** (chỉ thêm trigger Sonyflake)?

### 4.2. Table naming
Boss template: `<table>_shadow`. Code hiện tại: `cdc_internal.<table>` (schema-based separation, Sprint 5).

→ **Câu hỏi Q2**: Đi theo naming `cdc_internal.<table>` (đang chạy) hay đổi sang `<table>_shadow`?

### 4.3. Trigger Sonyflake
Hiện tại Go Worker generate ID (pkgs/idgen/sonyflake.go) rồi insert. Boss muốn thêm Postgres Trigger BEFORE INSERT.

→ **Câu hỏi Q3**: Trigger là **fallback** (chỉ gen khi client không cung cấp) hay **authoritative** (luôn overwrite)? Nếu authoritative thì Machine ID từ đâu (đang được Go worker claim qua `claim_machine_id()`)?

### 4.4. sources table location
Bảng `sources` nên ở `public.sources` hay `cdc_internal.sources`? DB chung `goopay_dw`.

→ **Câu hỏi Q4**: Boss chọn schema nào?

### 4.5. Wizard migration strategy
SourceToMasterWizard.tsx hiện tại là read-only (điều hướng). Biến thành stateful → FE phải refactor lớn.

→ **Câu hỏi Q5**: Tôi:
   - (A) Rewrite hoàn toàn → 1 component mới, dùng `wizard_sessions` state machine;
   - (B) Tạo component song song `SystematicWizard.tsx`, giữ `SourceToMasterWizard.tsx` làm read-only reference.
   Boss prefer A hay B?

---

## 5. Definition of Done (proposed)

Khi P0-P1 xong:
1. Admin click "New Connector" trên `/sources` → modal điền config → submit.
2. BE: validate → POST Kafka Connect → **INSERT INTO sources** → return `{connector_id, source_id, topic_prefix, collections_discovered}`.
3. UI auto-navigate sang `/registry?source_id=X` → form **pre-filled** (source_db, dropdown collections).
4. Admin chọn collection + PK → Submit.
5. BE: insert `cdc_table_registry` → **gọi `EnsureShadowTable` synchronous** → return `{registry_id, target_table, is_table_created: true}`.
6. Admin click "Snapshot Now" → publish NATS → SinkWorker consume → rows đổ về shadow.
7. SinkWorker auto-flag field mới vào `schema_proposal` → admin approve trên `/schema-proposals`.
8. Admin define master spec trên `/masters` → approve → MasterDDLGenerator chạy.
9. Transmute Scheduler bơm từ shadow → master.
10. `public.<master>` có rows → DoD ✅.

Thời gian end-to-end (MVP): < 5 phút cho source mới, zero CLI.
