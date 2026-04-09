# Plan v1.11: Hoàn thiện Airbyte ↔ CMS + Kích hoạt Worker Transform

> **Date**: 2026-04-08  
> **Version**: 1.11  
> **Scope**: (1) Đóng kín vòng lặp đồng bộ Airbyte ↔ CMS, (2) Kích hoạt Worker transform layer thực sự hoạt động với data từ Airbyte.  
> **Prerequisite**: `03_implementation_3.md` — hiện trạng hệ thống  
> **Debezium**: Deferred → `03_implementation_phase_2.md` (standalone, true realtime)

---

## 1. Bối cảnh & Vấn đề

### 1.1 Airbyte CDC không đạt realtime
Debezium tích hợp trong Airbyte chạy **batch scheduling** (5-15 phút interval), không phải true realtime. Chấp nhận near-realtime cho v1.11. Debezium standalone sẽ kích hoạt riêng trong Phase 2 khi cần true realtime (<1s latency).

### 1.2 Worker Transform — Code đầy đủ nhưng data không chảy đến

**Audit thực tế cho thấy:**

| Component | Code | Wired up | Data flowing |
|-----------|------|----------|-------------|
| EventHandler (parse + map + upsert) | ✅ | ✅ | ❌ **Không có event** |
| BatchBuffer (UPSERT SQL with typed columns) | ✅ | ✅ | ❌ Chờ EventHandler |
| ConsumerPool (NATS pull `cdc.goopay.>`) | ✅ | ✅ | ❌ Không có publisher |
| SchemaInspector (drift detection) | ✅ | ✅ | ❌ Chờ EventHandler |
| CommandHandlers (discover/backfill/scan) | ✅ | ✅ | ✅ **Hoạt động** (manual trigger) |

**Root cause**: Worker chờ CDC events trên NATS (`cdc.goopay.>`) từ Debezium — nhưng Debezium chưa deploy.

### 1.3 CRITICAL: Airbyte tables ≠ CDC tables

**Phát hiện nghiêm trọng**: Airbyte và CDC Worker sử dụng **2 hệ thống tables hoàn toàn tách rời**.

```
Airbyte output:     _airbyte_raw_merchants  (_airbyte_data JSONB, _airbyte_emitted_at, ...)
CDC Worker output:  cdc_merchants           (_raw_data JSONB, _source, _hash, _version, ...)
                    ↑ TRỐNG — không có data vì chưa có Debezium
```

- Airbyte ghi vào `_airbyte_raw_{stream}` tables — format riêng của Airbyte
- CDC Worker tạo `cdc_*` tables bằng `create_cdc_table()` — format riêng (column `_raw_data`)
- **KHÔNG CÓ BRIDGE** giữa 2 bên: không có code nào đọc từ `_airbyte_raw_*` → ghi vào `cdc_*`
- CMS Import chỉ tạo registry entry + empty CDC table — **KHÔNG copy data**

**Hệ quả**: 
- `_raw_data` trong CDC tables = NULL/TRỐNG
- HandleBatchTransform sẽ **KHÔNG hoạt động** vì không có data để transform
- Backfill cũng fail vì `_raw_data` trống

### 1.4 Giải pháp — Cần quyết định 1 trong 2 hướng

**Hướng A: Bridge — Worker đọc `_airbyte_raw_*` → populate CDC `_raw_data` → transform**
```
_airbyte_raw_merchants (_airbyte_data JSONB)
        ↓ (Worker Bridge: periodic read + copy)
cdc_merchants (_raw_data = _airbyte_data, typed columns from mapping)
```
- Ưu: Giữ CDC table format hiện tại, tách biệt Airbyte internal format
- Ưu: Debezium Phase 2 ghi vào cùng CDC tables → không cần thay đổi transform logic
- Nhược: Duplicate data (Airbyte tables + CDC tables)

**Hướng B: Đọc trực tiếp — Worker transform trên `_airbyte_raw_*` tables**
```
_airbyte_raw_merchants (_airbyte_data JSONB)
        ↓ (Worker: transform in-place, thêm typed columns)
_airbyte_raw_merchants (_airbyte_data + typed columns + metadata)
```
- Ưu: Không duplicate data
- Nhược: Coupling với Airbyte internal format (có thể thay đổi khi upgrade Airbyte)
- Nhược: Khi thêm Debezium Phase 2, cần refactor lại

**Khuyến nghị**: Hướng A (Bridge) — vì khi Debezium Phase 2 kích hoạt, cả 2 source (Airbyte batch + Debezium realtime) đều ghi vào cùng CDC tables thống nhất.

### 1.3 Đồng bộ Airbyte ↔ CMS — False ở nhiều chỗ

| Entity | Airbyte→CMS | CMS→Airbyte | Status |
|--------|-------------|-------------|--------|
| Sources | GET read-only | Không | OK |
| Destinations | **Chưa có** | Không | Missing |
| Connections | Chỉ lưu ID | Không | Partial |
| Streams | Import manual | is_active toggle | **Gap** |
| Field Mapping | Scan manual | **Chưa push** | **Gap** |

---

## 2. Mục tiêu v1.11

1. **Worker transform hoạt động tự động** — sau khi Airbyte sync xong, Worker tự detect + transform data
2. **Airbyte ↔ CMS đồng bộ đầy đủ** — streams, field mappings
3. **Zero manual steps** — approve mapping → auto-backfill → data sẵn sàng

---

## 3. Danh sách Task — 5 Tracks

### Track E: Kích hoạt Worker Transform (CRITICAL — làm trước)

> Mục tiêu: Worker tự động transform data sau mỗi Airbyte sync, không cần Debezium, không cần manual trigger.

#### E0: Airbyte → CDC Bridge (PREREQUISITE — giải quyết data gap)
- **Effort**: 2.5 ngày
- **Mô tả**: Worker đọc data từ `_airbyte_raw_*` tables → populate `_raw_data` trong CDC tables. Đây là bridge giữa 2 hệ thống tables.
- **Cách hoạt động**:
  ```
  _airbyte_raw_{stream} (_airbyte_data JSONB, _airbyte_emitted_at)
          ↓ Worker Bridge (periodic hoặc manual trigger)
  cdc_{stream} (_raw_data = _airbyte_data, _source = 'airbyte', _synced_at, _hash)
  ```
- **Implementation**:
  - Worker: thêm `HandleAirbyteBridge(msg *nats.Msg)` trong `command_handler.go`
    ```sql
    -- Step 1: Detect Airbyte raw table name
    -- Convention: _airbyte_raw_{source_table} hoặc airbyte config
    
    -- Step 2: Upsert từ Airbyte raw → CDC table
    INSERT INTO {cdc_table} (id, _raw_data, _source, _synced_at, _hash, _version)
    SELECT 
      COALESCE(
        _airbyte_data->>'_id',
        _airbyte_data->>'id',
        _airbyte_ab_id::text
      ) as id,
      _airbyte_data as _raw_data,
      'airbyte' as _source,
      _airbyte_emitted_at as _synced_at,
      md5(_airbyte_data::text) as _hash,
      1 as _version
    FROM {_airbyte_raw_table}
    WHERE _airbyte_emitted_at > {last_bridge_at}  -- incremental
    ON CONFLICT (id) DO UPDATE SET
      _raw_data = EXCLUDED._raw_data,
      _synced_at = EXCLUDED._synced_at,
      _hash = EXCLUDED._hash,
      _version = {cdc_table}._version + 1,
      _updated_at = NOW()
    WHERE {cdc_table}._hash IS DISTINCT FROM EXCLUDED._hash;
    ```
  - Thêm column `last_bridge_at TIMESTAMP` vào `cdc_table_registry` — track incremental bridge
  - Thêm column `airbyte_raw_table VARCHAR` vào `cdc_table_registry` — Airbyte raw table name
  - NATS subject: `cdc.cmd.bridge-airbyte`
  - CMS API endpoint: `POST /api/registry/:id/bridge` (admin, 202 Accepted)
  - CMS route: `admin.Post("/registry/:id/bridge", registryHandler.Bridge)`
- **Auto-detect Airbyte raw table**: Worker query `information_schema.tables WHERE table_name LIKE '_airbyte_raw_%'`

#### E1: Post-sync Transform — Worker batch process `_raw_data`
- **Effort**: 2 ngày
- **Prerequisite**: E0 (CDC tables phải có data trong `_raw_data`)
- **Mô tả**: Worker transform `_raw_data` → typed columns theo mapping_rules.
- **Cách hoạt động**:
  ```
  Trigger (1 trong 3 cách):
    ├─ CMS API: POST /api/registry/:id/transform (manual)
    ├─ Periodic: Worker tự chạy mỗi X phút (configurable)
    └─ Sau bridge xong → auto-trigger transform
  
  Worker nhận lệnh → cho mỗi active table:
    1. Đọc rows có _raw_data nhưng typed columns = NULL (chưa transform)
    2. Hoặc rows có _hash thay đổi (data updated)
    3. Apply mapping_rules → UPDATE typed columns
    4. Ghi metrics: rows_transformed, duration
  ```
- **Implementation**:
  - Worker: thêm `HandleBatchTransform(tableName)` trong `command_handler.go`
    ```sql
    -- Cho mỗi mapping rule (source_field → target_column, data_type):
    UPDATE {target_table} SET 
      {target_column} = (_raw_data->>'{source_field}')::{data_type}
    WHERE {target_column} IS NULL 
      AND _raw_data ? '{source_field}'
    ```
  - NATS subject: `cdc.cmd.batch-transform`
  - CMS API endpoint: `POST /api/registry/:id/transform` (admin, 202 Accepted)

#### E2: Periodic transform scheduler
- **Effort**: 1 ngày
- **Mô tả**: Worker tự chạy batch transform theo interval (không cần manual trigger)
- **Implementation**:
  - Worker `worker_server.go`: thêm `go ticker` goroutine
    ```go
    go func() {
        ticker := time.NewTicker(cfg.Worker.TransformInterval) // default 5m
        for range ticker.C {
            entries, _ := registryRepo.GetAllActive(ctx)
            for _, entry := range entries {
                // Publish transform command for each active table
                natsClient.Conn.Publish("cdc.cmd.batch-transform", []byte(entry.TargetTable))
            }
        }
    }()
    ```
  - Config: `worker.transformInterval: 5m` trong config-local.yml
  - Logging: ghi số rows transformed per table per cycle

#### E3: Transform status tracking
- **Effort**: 0.5 ngày
- **Mô tả**: Biết table nào đã transform, bao nhiêu rows chưa transform
- **Implementation**:
  - Endpoint: `GET /api/registry/:id/transform-status`
  - Worker query:
    ```sql
    SELECT 
      COUNT(*) as total_rows,
      COUNT(*) FILTER (WHERE {first_mapped_column} IS NOT NULL) as transformed_rows,
      COUNT(*) FILTER (WHERE {first_mapped_column} IS NULL AND _raw_data IS NOT NULL) as pending_rows
    FROM {target_table}
    ```
  - FE: hiển thị progress bar trong Registry page

---

### Track A: Airbyte Read APIs (bổ sung thiếu)

> Mục tiêu: CMS hiển thị đầy đủ thông tin Airbyte mà không cần mở Airbyte UI.

#### A1: `GET /api/airbyte/destinations` — Liệt kê destinations
- **Effort**: 0.5 ngày
- **Airbyte API**: `POST /api/v1/destinations/list`
- **Response**: `[{destinationId, name, destinationName, workspaceId}]`
- **Lưu vào DB**: Không

#### A2: `GET /api/airbyte/connections` — Liệt kê connections với detail
- **Effort**: 0.5 ngày
- **Response**: `[{connectionId, name, sourceId, destId, status, schedule, streamCount, enabledStreams}]`
- **Airbyte API**: `POST /api/v1/connections/list` (đã có) + enrich summary

#### A3: `GET /api/airbyte/connections/:id/streams` — Streams của 1 connection
- **Effort**: 1 ngày
- **Response**: stream name, namespace, syncMode, selected, cursorField, primaryKey + comparison với registry
- **Airbyte API**: `GetConnection(id)` → parse syncCatalog.streams

---

### Track B: Stream Sync (đóng gap Streams)

> Mục tiêu: Khi Airbyte có stream mới → CMS biết. Khi CMS toggle active → Airbyte cập nhật.

#### B1: Full Stream Sync — Import all missing streams
- **Effort**: 1.5 ngày
- **Scope**:
  - Cải thiện `POST /api/airbyte/import/execute`:
    - Auto-detect PK từ Airbyte catalog `sourceDefinedPrimaryKey` (thay vì hardcoded "id")
    - Auto-detect source_type từ source connector name
    - Tạo mapping rules từ Airbyte JSONSchema fields (tất cả fields, không chỉ "id")
  - Endpoint mới: `POST /api/registry/sync-from-airbyte`
    - Quét tất cả connections → so sánh registry → tạo entries thiếu (is_active=false)
    - Response: `{added, already_exists, total}`

#### B2: Stream config sync — Lưu sync metadata
- **Effort**: 1 ngày
- **Thêm columns** vào `cdc_table_registry`:
  - `airbyte_sync_mode` — incremental / full_refresh
  - `airbyte_destination_sync_mode` — append / overwrite / upsert
  - `airbyte_cursor_field` — cursor cho incremental
  - `airbyte_namespace` — schema/database
- **Migration SQL**: ALTER TABLE ADD COLUMN

#### B3: Bidirectional active/inactive
- **Effort**: 0.5 ngày
- Verify CMS→Airbyte toggle (đã có)
- Thêm detect Airbyte→CMS (so sánh `selected` vs `is_active` → flag mismatch)

---

### Track C: Field Mapping Sync (CORE)

> Mục tiêu: Biết fields nào source có, fields nào đã map, fields nào thiếu.

#### C1: Auto-detect fields khi import
- **Effort**: 1.5 ngày
- Parse Airbyte JSONSchema → extract fields + types → tạo `cdc_mapping_rules` (status=pending)
- Infer: string→TEXT, integer→BIGINT, number→NUMERIC, boolean→BOOLEAN, object→JSONB

#### C2: Periodic field scan từ `_raw_data`
- **Effort**: 1.5 ngày
- Worker periodic task (default 1h): scan `jsonb_object_keys(_raw_data)` cho mỗi active table
- Auto-create mapping_rules (status=pending, rule_type=discovered)
- Config: `worker.scanInterval: 1h`

#### C3: Batch approve/reject + auto-backfill
- **Effort**: 1.5 ngày
- API: `PATCH /api/mapping-rules/batch` — `{ids: [1,2,3], status: "approved"}`
- Sau approve → auto-trigger `cdc.cmd.backfill` cho field vừa approve (zero manual steps)
- FE: checkbox select multiple → batch approve button

---

### Track D: Monitoring & Consistency

#### D1: Sync health dashboard
- **Effort**: 1 ngày
- `GET /api/sync/health` → `{total_streams, registered, unregistered, mismatched, pending_rules, last_scan, transform_pending_rows}`
- FE Dashboard widget

#### D2: Reconciliation report
- **Effort**: 1 ngày
- Mở rộng sync-audit: field-level + row count comparison

---

## 4. Thứ tự thực hiện

```
Week 1 — Airbyte Bridge + Worker Transform (CRITICAL PATH):
  ├─ E0: Airbyte → CDC Bridge ──────────── 2.5 ngày (PREREQUISITE)
  ├─ E1: Post-sync batch transform ──────── 2 ngày
  └─ E2: Periodic bridge+transform ──────── 1 ngày  

Week 2 — Airbyte Sync + Status:
  ├─ E3: Transform status tracking ──────── 0.5 ngày
  ├─ Track A (A1, A2, A3) ──────────── 2 ngày ─── Read APIs
  └─ Track B (B1, B2, B3) ──────────── 3 ngày ── Stream sync

Week 3 — Field Mapping + Monitoring:
  ├─ Track C (C1, C2) ──────────────── 3 ngày ──── Field auto-detect (CORE)
  ├─ Track C (C3) ──────────────────── 1.5 ngày ── Batch approve + auto-backfill
  └─ Track D (D1, D2) ──────────────── 2 ngày ──── Monitoring
```

**Critical path**: **E0 (Bridge)** → E1 (Transform) → E2 (Periodic) → C1 (Field detect)

---

## 5. Files sẽ thay đổi

### Backend — centralized-data-service (Worker)
| File | Thay đổi | Track |
|------|---------|-------|
| `internal/handler/command_handler.go` | `HandleBatchTransform()`, periodic scan | E1, C2 |
| `internal/server/worker_server.go` | Transform ticker goroutine, subscribe `cdc.cmd.batch-transform` | E2 |
| `config/config.go` | `TransformInterval`, `ScanInterval` | E2, C2 |
| `config/config-local.yml` | Thêm config values | E2, C2 |

### Backend — cdc-cms-service (API)
| File | Thay đổi | Track |
|------|---------|-------|
| `pkgs/airbyte/client.go` | `ListDestinations()` | A1 |
| `internal/api/airbyte_handler.go` | Destinations, connections detail, streams | A1-A3 |
| `internal/api/registry_handler.go` | `SyncFromAirbyte()`, `Transform()`, `TransformStatus()` | B1, E1, E3 |
| `internal/api/mapping_rule_handler.go` | `BatchUpdate()` | C3 |
| `internal/service/approval_service.go` | Auto-backfill after approve | C3 |
| `internal/router/router.go` | Register all new routes | All |
| `internal/model/table_registry.go` | 4 new columns | B2 |
| `migrations/` | ALTER TABLE | B2 |

### Frontend — cdc-cms-web
| File | Thay đổi | Track |
|------|---------|-------|
| `src/pages/TableRegistry.tsx` | Transform status progress bar | E3 |
| `src/pages/MappingFieldsPage.tsx` | Batch approve/reject | C3 |
| `src/pages/Dashboard.tsx` | Sync health widget | D1 |
| `src/pages/SourceConnectors.tsx` | Destinations display | A1 |

---

## 6. API mới

| Method | Path | Track | Mô tả |
|--------|------|-------|-------|
| `POST` | `/api/registry/:id/bridge` | E0 | Trigger Airbyte→CDC bridge cho 1 table |
| `POST` | `/api/registry/:id/transform` | E1 | Trigger batch transform cho 1 table |
| `GET` | `/api/registry/:id/transform-status` | E3 | Transform progress (total/transformed/pending rows) |
| `GET` | `/api/airbyte/destinations` | A1 | List destinations (read-only) |
| `GET` | `/api/airbyte/connections` | A2 | List connections with detail |
| `GET` | `/api/airbyte/connections/:id/streams` | A3 | Streams detail + registry comparison |
| `POST` | `/api/registry/sync-from-airbyte` | B1 | Full sync all streams → registry |
| `PATCH` | `/api/mapping-rules/batch` | C3 | Batch update status |
| `GET` | `/api/sync/health` | D1 | Sync health summary |

---

## 7. Migration SQL (Track B2)

```sql
-- v1.11: Add Airbyte sync metadata + bridge tracking to registry
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_sync_mode VARCHAR DEFAULT 'incremental';
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_destination_sync_mode VARCHAR DEFAULT 'append';
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_cursor_field VARCHAR;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_namespace VARCHAR;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS airbyte_raw_table VARCHAR;  -- e.g. '_airbyte_raw_merchants'
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS last_bridge_at TIMESTAMP;   -- incremental bridge tracking
```

---

## 8. Worker Transform Flow — Chi tiết

### Hiện tại (Broken — data không chảy):
```
Airbyte sync → _airbyte_raw_merchants (_airbyte_data JSONB)   ← DATA Ở ĐÂY
CDC Worker   → cdc_merchants (_raw_data JSONB = NULL)          ← TRỐNG
                     ↓ (user manual)
              CMS: bấm Discover → Worker tạo mapping rules (OK)
              CMS: bấm Backfill → FAIL vì _raw_data = NULL
```

### Target v1.11 (Bridge + Auto Transform):
```
Airbyte sync → _airbyte_raw_merchants (_airbyte_data JSONB)
                     ↓ (E0: Bridge, periodic mỗi 5 phút)
              Worker Bridge:
                1. SELECT _airbyte_data FROM _airbyte_raw_merchants WHERE emitted_at > last_bridge
                2. UPSERT INTO cdc_merchants (_raw_data = _airbyte_data, _source = 'airbyte')
                     ↓ (E1: Transform, auto sau bridge)
              Worker Transform:
                1. Đọc mapping_rules cho table
                2. UPDATE typed columns từ _raw_data
                     ↓
              cdc_merchants: _raw_data FILLED + typed columns FILLED
```

### SQL cụ thể cho batch transform:
```sql
-- Cho table cdc_merchants với mapping rules:
-- business_name TEXT, email TEXT, phone VARCHAR, created_at TIMESTAMP

UPDATE cdc_merchants SET
  business_name = (_raw_data->>'business_name')::TEXT,
  email = (_raw_data->>'email')::TEXT,
  phone = (_raw_data->>'phone')::VARCHAR,
  created_at = (_raw_data->>'created_at')::TIMESTAMP,
  _updated_at = NOW()
WHERE (
  business_name IS NULL OR email IS NULL OR phone IS NULL OR created_at IS NULL
)
AND _raw_data IS NOT NULL;
```

---

## 9. Definition of Done — v1.11

- [ ] Worker tự transform data mỗi 5 phút (không cần manual trigger)
- [ ] Transform status visible trên UI (total/transformed/pending rows)
- [ ] Tất cả Airbyte streams có trong CMS registry (auto-sync)
- [ ] Mỗi stream có mapping rules cho tất cả fields (auto-detect)
- [ ] Active/inactive mismatch = 0 (bidirectional sync)
- [ ] Periodic scan phát hiện fields mới tự động
- [ ] Approve mapping → auto-backfill (zero manual steps)
- [ ] Sync health dashboard đúng thực tế
- [ ] All builds OK
- [ ] `05_progress.md` updated per change

---

## 10. Không làm trong v1.11

| Hạng mục | Lý do | Khi nào |
|----------|-------|---------|
| Debezium standalone | Cần true realtime, deploy riêng | Phase 2 (`03_implementation_phase_2.md`) |
| dbt | CDC Worker đã là transform layer | Không cần |
| Connection/Source CRUD | Airbyte UI là master | Không cần |
| Event Bridge (PG Triggers → NATS) | Cần core sync ổn trước | v1.12+ |
| Realtime event streaming | Chờ Debezium standalone | Phase 2 |
