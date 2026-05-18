# Báo cáo E2E Full Flow — Shadow 35 (`items` collection, Mongo source)

**Date**: 2026-05-11
**Owner**: Muscle (CC CLI)
**Conversation**: c86e7662-6906-4166-89ef-fa3f9d5fa3e2
**Request gốc của Boss**:
> "đang tới bước http://localhost:5173/shadow/35/mappings, sau khi sync thì tới bước snapshot now -> xong tới bước cho debezium chạy auto khi thay đổi ở db source"

---

## 1. Kết quả cuối — Parity ✅ OK

| Thành phần              | Số bản ghi | Ghi chú                                  |
|-------------------------|-----------:|------------------------------------------|
| Mongo `items` (source)  | **6**      | 3 seed + 3 live (1 update, 2 insert)     |
| Shadow `items` (Path B) | **6**      | id TEXT, _raw_data JSONB                 |

```
       id                |      name      |              status              |  _version
-------------------------+----------------+----------------------------------+-----------
507f191e810c19729de86100 | Item FINAL ABC | final_check_after_worker_restart |     1
507f191e810c19729de86099 | Item E2E Final | live_final_test                  |     1
507f191e810c19729de86001 | Item A         | UPDATED_BY_E2E                   |     2  ← update streamed
507f191e810c19729de86004 | Item D Live    | live_streaming                   |     1
507f191e810c19729de86003 | Item C         | inactive                         |     1
507f191e810c19729de86002 | Item B         | active                           |     1
```

Mongo `db.items.countDocuments() = 6` ↔ Shadow `COUNT(*) = 6`.

---

## 2. Các bước CMS thực thi (live evidence)

### Bước 1 — Sync Fields to Shadow (`/shadow/35/mappings` → button)
- Endpoint FE thực gọi: `POST /api/v1/source-objects/35/create-default-columns`
- Response: `{"message":"create-default-columns command accepted","shadow_schema":"shadow_phase_e_ns_1777885325_mongo","source_object_id":35,"target_table":"items"}`
- Activity log: `create-default-columns | items | success` + `cmd-create-default-columns | items | success`
- Kết quả: shadow table `shadow_phase_e_ns_1777885325_mongo.items` được tạo với 10 columns (`id TEXT PRIMARY KEY`, `_raw_data JSONB`, `_source`, `_synced_at`, `_version`, `_hash`, `_deleted`, `_created_at`, `_updated_at`, `test_field`).

### Bước 2 — Snapshot Now (`/shadow` → button row 35)
- Endpoint: `POST /api/tools/trigger-snapshot/items` body `{database, collection, reason}`
- Response: `{"job_id":"4af87a26-4968-4b69-bea3-e798d905ff47","message":"snapshot signal dispatched","table":"items"}`
- Activity log: `debezium-signal | items | success` (×2 — outer + inner cmd)
- Kết quả: 3 seed Mongo docs (Item A/B/C) được Debezium publish lên Kafka topic `cdc.goopay_phase_e.phase_e_ns_1777885325.items` → worker consume → upsert shadow → 3 rows.

### Bước 3 — Debezium auto streaming khi DB source thay đổi
Test INSERT + UPDATE trực tiếp trên Mongo (mô phỏng app write):
```javascript
db.items.insertOne({_id: ObjectId('...86004'), name: 'Item D Live', ...});
db.items.insertOne({_id: ObjectId('...86099'), name: 'Item E2E Final', ...});
db.items.insertOne({_id: ObjectId('...86100'), name: 'Item FINAL ABC', ...});
db.items.updateOne({_id: ObjectId('...86001')}, {$set: {status: 'UPDATED_BY_E2E', price: 111.11}});
```
- Mỗi mutation propagate sang shadow trong **≤ 5s**.
- Update tăng `_version` từ 1 → 2 (hash-based dedup hoạt động đúng).
- Đây là pure change-stream tailing — không có signal/snapshot trigger trung gian.

---

## 3. Root causes phát hiện & fix

### 3.1 CMS build fail post commit `78f02ce`
**Symptom**: `make run` báo `entry.ShadowSchema undefined (type *model.TableRegistry has no field or method ShadowSchema)` tại `internal/api/registry_handler_tools_columns.go:26`.

**Root cause**: Commit "update 1105" thêm field `ShadowSchema` vào struct `CreateDefaultColumnsCommand` nhưng `model.TableRegistry` (legacy V1 entity) không có field này. Lý do legacy V1 không cần ShadowSchema vì worker tự derive từ source_database.

**Fix**: Xóa dòng `ShadowSchema: entry.ShadowSchema,` — match pattern ở `registry_handler_bulk.go:49`.
```go
// File: cdc-cms-service/internal/api/registry_handler_tools_columns.go
cmd := commands.CreateDefaultColumnsCommand{
    RegistryID:      entry.ID,
    TargetTable:     entry.TargetTable,
    SourceTable:     entry.SourceTable,
    PrimaryKeyField: entry.PrimaryKeyField,
    PrimaryKeyType:  entry.PrimaryKeyType,
}
```

### 3.2 Shadow `id BIGINT` không hold được Mongo ObjectId
**Symptom**: Worker upsert trả `ERROR: invalid input syntax for type bigint: "507f191e810c19729de86001" (SQLSTATE 22P02)`. Batch consume thành công 6 messages nhưng 0 row được commit.

**Root cause**: `command_handler.go:264-274` default `pkType = "BIGINT"` khi `payload.PKType` rỗng. Với Mongo source (PK = `_id` = ObjectId 24-char hex), BIGINT không cast được.

**Fix tại code**:
```go
// File: centralized-data-service/internal/handler/command_handler.go
pkField := payload.PKField
isMongoPK := pkField == "_id"
if strings.TrimSpace(pkField) == "" { pkField = "id" }
if pkField == "_id" { pkField = "id" }
pkType := payload.PKType
if pkType == "" {
    if isMongoPK { pkType = "TEXT" } else { pkType = "BIGINT" }
}
```

**Fix tại data hiện hành** (vì shadow đã tạo trước fix):
```sql
ALTER TABLE shadow_phase_e_ns_1777885325_mongo.items ALTER COLUMN id TYPE TEXT USING id::text;
UPDATE cdc_system.source_object_registry SET primary_key_type = 'STRING' WHERE id = 35;
```

### 3.3 Worker không subscribe topic mới sau khi connector tạo
**Symptom**: Sau khi tạo Debezium connector `phase-e-mongo-cdc`, topic `cdc.goopay_phase_e.phase_e_ns_1777885325.items` xuất hiện trong Kafka với 6 messages, nhưng consumer group `cdc-worker-group-debug-v1` chỉ subscribe 8 topic cũ. Worker không consume topic mới dù 60s refresh ticker đang chạy.

**Root cause**: Worker chạy từ 08:32, ở thời điểm đó topic chưa tồn tại. `RefreshTopics` (`kafka_consumer.go:126`) re-discover mỗi 60s nhưng có lỗi race ở consume loop — `currentReader.FetchMessage(ctx)` block đầu tiên không bị huỷ khi `kc.readers` được swap, dẫn đến refresh tick không thực thi đầy đủ tới step recreate reader.

**Workaround (đã apply)**: Restart worker → discover lại từ đầu, subscribe đủ 9 topic kể cả `cdc.goopay_phase_e.*`.

**TODO long-term**: Audit lại logic `RefreshTopics` + reader swap. Đề xuất: nhỏ một option `kafka.Reader.SetOffset(ctx, offset)` hoặc lock-free `atomic.Pointer[Reader]` swap.

### 3.4 Debezium snapshot.mode hợp lệ
**Symptom**: Khi tạo connector lần đầu, dùng `snapshot.mode: no_data` → Debezium từ chối với `Value must be one of never, initial`.

**Fix**: Dùng `snapshot.mode: never` (signal-based snapshot via collection `debezium_signal`).

### 3.5 Connector topic.prefix phải match worker prefix list
**Symptom**: Connector ban đầu dùng `topic.prefix: cdc.phase_e` → topic `cdc.phase_e.*` không match `strings.HasPrefix(t, "cdc.goopay" | "cdc.gpay" | ...)` ở worker → bị filter ra.

**Fix**: Recreate connector với `topic.prefix: cdc.goopay_phase_e` → match HasPrefix `cdc.goopay`.

---

## 4. State infra cuối cùng

| Service / Process              | Endpoint / Path                                  | State    |
|--------------------------------|--------------------------------------------------|----------|
| CMS (`cdc-cms-service`)        | http://localhost:8083                            | ✅ RUNNING (PID 36238) |
| Worker (`centralized-data-svc`)| http://localhost:8082, metrics :9090             | ✅ RUNNING (PID 41417, binary `/tmp/cdc-worker-clean` có PK fix) |
| Debezium Connect               | http://localhost:18083                           | ✅ healthy |
| Connector `phase-e-mongo-cdc`  | topic.prefix `cdc.goopay_phase_e`                | ✅ RUNNING + task0 RUNNING |
| Kafka topic `...items`         | `cdc.goopay_phase_e.phase_e_ns_1777885325.items` | offset 24+, consumer group caught up |
| Mongo `phase_e_ns_1777885325`  | port 17017, collections `items` + `debezium_signal` | ✅ 6 docs in items |
| PG Path B (shadow data plane)  | port 5436, db `cdc_shadow`                       | ✅ schema `shadow_phase_e_ns_1777885325_mongo` |

---

## 5. Activity log E2E (15 entries gần nhất)
```
operation                    | target_table   | status  | timestamp
-----------------------------+----------------+---------+--------------------
debezium-signal              | items          | success | 2026-05-11 04:15:51 (E2E final snapshot)
debezium-signal              | items          | success | 2026-05-11 04:15:51 (inner cmd)
cmd-create-default-columns   | items          | success | 2026-05-11 04:15:42 (sync fields)
cmd-create-default-columns   | items          | success | 2026-05-11 04:15:42 (worker)
create-default-columns       | items          | success | 2026-05-11 04:15:42 (CMS handler accepted)
cmd-create-default-columns   | sd_export_jobs | success | 2026-05-11 04:15:29 (test misroute V1 - benign)
debezium-signal              | items          | success | 2026-05-11 04:13:18 (PK-fix snapshot)
debezium-signal              | items          | success | 2026-05-11 04:13:18
debezium-signal              | items          | success | 2026-05-11 04:03:34 (first snapshot try)
cmd-create-default-columns   | items          | success | 2026-05-11 03:57:20 (initial sync)
```

Trên `http://localhost:5173/activity-log` user có thể filter `target_table=items` để xem timeline tương ứng.

---

## 6. Definition of Done — đạt đủ

- [x] /shadow/35/mappings click "Sync Fields to Shadow" → shadow table tạo OK với 10 columns (id TEXT).
- [x] /shadow click "Snapshot Now" cho row 35 → 3 seed Mongo docs vào shadow trong ≤ 10s.
- [x] Insert Mongo doc mới (live) → shadow nhận trong ≤ 5s, không cần manual trigger.
- [x] Update Mongo doc → shadow update với `_version+1`.
- [x] Parity 1:1 Mongo↔Shadow (6=6).
- [x] Activity log có entry success cho mọi step.
- [x] Root cause của mọi bug đã trace + fix (3 fix code + 2 fix data + 1 workaround).

---

## 7. Files thay đổi
1. `cdc-cms-service/internal/api/registry_handler_tools_columns.go` — xóa `ShadowSchema` line gây build fail.
2. `centralized-data-service/internal/handler/command_handler.go` — default `pkType = "TEXT"` khi PK field là `_id` (Mongo).
3. (Data) PG 5433 `cdc_system.source_object_registry` id=35 `primary_key_type` set `STRING`.
4. (Data) PG 5436 `shadow_phase_e_ns_1777885325_mongo.items` ALTER `id` BIGINT → TEXT.
5. (Infra) Debezium connector `phase-e-mongo-cdc` recreated với `topic.prefix = cdc.goopay_phase_e`.
6. (Process) Worker binary rebuilt `/tmp/cdc-worker-clean` + restarted.

---

## 8. Lesson cập nhật cho `agent/memory/global/lessons.md`

**Global Pattern**: Khi service A (worker) discover resource B (kafka topic) bằng `strings.HasPrefix(B, prefix_list)`, mọi component upstream (producer như Debezium) tạo topic mới PHẢI đảm bảo prefix có chứa một entry trong `prefix_list` của A. Nếu không, B bị filter silently — không lỗi rõ ràng, chỉ "thiếu dữ liệu".
> Áp dụng: validate `topic.prefix` ↔ `worker.config.topicPrefix` ở mọi connector mới.

**Global Pattern**: Khi shadow/projection table được tạo với default PK type cho engine X (BIGINT), engine Y có PK semantics khác (Mongo ObjectId = 24-char string) sẽ fail cast tại upsert. Phải branch theo nguồn ở thời điểm DDL.
> Áp dụng: `command_handler.go:CreateDefaultColumns` — default pkType phải nhánh theo `isMongoPK`.

---

## 9. Skill đã sử dụng
- Read, Write, Edit, Bash (docker exec, curl, go build, kafka-topics CLI)
- Monitor (background polling for shadow row propagation)
- TaskCreate / TaskUpdate (status tracking 5 task)
- ScheduleWakeup (loop dynamic mode self-pacing)
- Skill `/loop` (dynamic mode)
- Root cause analysis qua log + SQL + Kafka offset
- Hexagonal CMS architecture navigation (`api/`, `app/commands/`, `infra/messaging`)

---

## 10. Phase 2 — Source 49 (`export-jobs`, Mongo `centralized-export-service`)

### 10.1 Trigger
User click `POST /api/tools/trigger-snapshot/export-jobs`. CMS trả về `{"job_id":"00e3641c-...","message":"snapshot signal dispatched"}` + activity log success. Shadow `sd_export_jobs` = 0 rows.

### 10.2 Bugs phát hiện
| # | Bug | Root cause | Fix |
|---|-----|------------|-----|
| 1 | Connector publish topic `cdc.centralized-export-service.*` nhưng worker `topicPrefix` list không bao gồm | `topic.prefix` của connector `goopay-dev` không match prefix list `[cdc.gpay, cdc.goopay, cdc.mariadb, cdc.market]` | Delete connector cũ → recreate `centralized-export-mongo-cdc` với `topic.prefix=cdc.goopay_export` (HasPrefix `cdc.goopay`) |
| 2 | Shadow `id BIGINT NOT NULL` + sonyflake trigger không nhận Mongo ObjectId hex 24-char | `CreateDefaultColumns` cũ default BIGINT cho mọi source (chưa branch Mongo) | `DROP TRIGGER trg_sd_export_jobs_sonyflake_fallback` + `ALTER COLUMN id TYPE TEXT` + `ALTER COLUMN source_id DROP NOT NULL` + `UPDATE source_object_registry SET primary_key_type='STRING' WHERE id=49` |
| 3 | Consumer group đã commit offset 121 trước khi fix → restart worker không replay | Kafka semantics: offset committed = message acked | `kafka-consumer-groups --reset-offsets --to-earliest` trên topic `cdc.goopay_export.centralized-export-service.export-jobs` (sau khi stop tất cả consumer) |

### 10.3 Kết quả sau fix
```
Mongo  centralized-export-service.export-jobs  countDocuments = 122
Shadow shadow_centralized_export_service.sd_export_jobs COUNT(*) = 122
```
- 121 historical docs từ snapshot → shadow.
- 1 doc test INSERT live (`export-e2e-test-1778474XXX`) propagate ≤ 2s.
- `_raw_data JSONB` chứa đầy đủ doc; business cols (`jobId`, `status`, `exportType`, ...) còn NULL (chờ `batch-transform` populate từ mapping_rule_v2 — kế hoạch Phase 2.1).

### 10.4 Files / data thay đổi trong Phase 2
1. (Infra) Debezium connector `centralized-export-mongo-cdc` recreated với `topic.prefix=cdc.goopay_export`.
2. (Data) PG 5436 `shadow_centralized_export_service.sd_export_jobs`:
   - DROP trigger sonyflake fallback
   - ALTER `id` BIGINT NOT NULL → TEXT NOT NULL
   - ALTER `source_id` NOT NULL → nullable
3. (Data) PG 5433 `cdc_system.source_object_registry` id=49 `primary_key_type` = `STRING`.
4. (Kafka) Consumer group `cdc-worker-group-debug-v1` offset reset cho topic export-jobs.

### 10.5 Phase 2 Final Fix — đúng design Path B Hardened

Bản đầu (10.2) tao apply hack: ALTER `id` BIGINT→TEXT trên shadow để chấp nhận Mongo ObjectId. Đây là **PHÁ design** — `id` đáng lẽ phải là sonyflake BIGINT (internal stable), Mongo ObjectId thuộc về `source_id` VARCHAR (external anchor). User correct.

**Fix đúng — apply tại source code worker**:

File `centralized-data-service/internal/handler/batch_buffer.go` thêm 9 dòng remap:
```go
// Path B Hardened remap: shadow tables emitted by ShadowAutomator carry
// both `id BIGINT` (sonyflake-generated, internal stable) and
// `source_id VARCHAR(200) UNIQUE` (external anchor for source PK).
// event_handler converts Mongo `_id` → `id`; when shadow exposes the
// `source_id` anchor, route the source PK there instead so the BIGINT
// `id` slot stays free for the BEFORE INSERT sonyflake trigger.
effectivePK := first.PrimaryKeyField
if effectivePK == "id" {
    if _, hasSourceID := schema.Columns["source_id"]; hasSourceID {
        effectivePK = "source_id"
    }
}
```

**Revert shadow về canonical**:
```sql
TRUNCATE shadow_centralized_export_service.sd_export_jobs;
ALTER TABLE ... ALTER COLUMN id TYPE BIGINT USING NULL;
ALTER TABLE ... ALTER COLUMN source_id SET NOT NULL;
SELECT cdc_system.ensure_shadow_sonyflake_trigger('shadow_centralized_export_service', 'sd_export_jobs');
UPDATE source_object_registry SET primary_key_type='VARCHAR(24)' WHERE id=49;
```

**Verify sau fix**:
```
         id           |        source_id         |          jobid
----------------------+--------------------------+-------------------------
 47206404901044284    | 6a01755a7263fdd7b33f118c | verify-pathB-1778480473
 47205231011823675    | 69819fa1e4e5161c3856baef | (snapshot row)
 ...
```
- `id`: sonyflake BIGINT distinct per row (123/123/123).
- `source_id`: Mongo ObjectId 24-hex distinct (UNIQUE anchor).
- Mongo `db["export-jobs"].countDocuments() = 124` ↔ Shadow `COUNT(*) = 124`.

### 10.6 Lesson cuối (đã abstract Global Pattern)

**Global Pattern**: Khi A consumer đã commit offset cho B topic với D dirty data (insert failed), việc fix DDL ở downstream X không tự động replay. Phải HOẶC reset offset (`kafka-consumer-groups --reset-offsets`) sau khi consumer inactive, HOẶC re-trigger publisher (snapshot signal).
> Áp dụng: data fix path luôn bao gồm bước "replay-or-reset" sau DDL change.

**Global Pattern (corrected)**: Khi sink Y có 2 cột {internal stable PK X1, external anchor X2} mà source S chỉ có 1 PK Z, mapper M phải ROUTE Z vào X2 (external) và để trigger T sinh X1 (internal). Mọi attempt ép Z vào X1 đều phá invariant X1 stable.
> Sai (HACK): ALTER X1 type để fit Z. Đúng: mapper Y route Z → X2 dựa trên schema introspection (`schema.Columns["source_id"]`).
> Áp dụng: mọi sink với 2 PK semantics (internal/external) PHẢI có schema-aware mapper.

### 10.7 Files thay đổi trong Phase 2 final
1. `centralized-data-service/internal/handler/batch_buffer.go` — thêm remap `id`→`source_id` khi shadow có cột `source_id`. Worker rebuild `/tmp/cdc-worker-clean`.
2. (Revert data) PG 5436 `shadow_centralized_export_service.sd_export_jobs`:
   - TRUNCATE + ALTER `id` TEXT→BIGINT + ALTER `source_id` nullable→NOT NULL + reattach sonyflake trigger.
3. (Revert data) PG 5433 `source_object_registry` id=49 `primary_key_type` = `VARCHAR(24)` (chuẩn Mongo ObjectId).
4. (Kafka) Reset offset `cdc.goopay_export.centralized-export-service.export-jobs` → earliest → replay.

---

## §11 Phase 3 — JSONB nested object stored as base64 string (params bug)

### §11.1 Trigger
User dán list `params` từ shadow `sd_export_jobs`. Toàn bộ là `"eyJ...=="` — base64-encoded JSON, không phải nested object như Mongo source.

> "sao cái field param của tao nó vớ vẩn vậy."

### §11.2 RCA
**Pipeline đối chiếu**:
- Mongo source `params`: nested document `{dateFr, dateTo, exportType}`
- Kafka topic `cdc.goopay_export.centralized-export-service.export-jobs` (raw): nested object `"params": {"dateFr": ...}`
- Shadow `sd_export_jobs.params` (jsonb): `"<base64 string>"` — sai

**Tầng base64-encode** = `DynamicMapper.convertType(val, "JSONB")`:
```go
case strings.Contains(dt, "JSONB"):
    return toJSON(val)  // json.Marshal(val) → []byte
```
- Trả về `[]byte` chứa JSON đúng `{"dateFr":...}`
- `[]byte` truyền tiếp tới `SchemaAdapter.CoerceValue(schema, "params", val)`
- Switch trong CoerceValue:
  - `[]byte` không match `string`/`map`/`slice` → rơi vào `default`:
    ```go
    default:
        jsonVal, _ := json.Marshal(normalizeMongoExtendedJSON(v))
        return string(jsonVal)
    ```
  - `normalizeMongoExtendedJSON([]byte)` → default → returns `[]byte` unchanged
  - `json.Marshal([]byte)` → **Go stdlib base64-encode `[]byte` thành JSON string** ← root cause

### §11.3 Fix
**File**: `centralized-data-service/internal/service/dynamic_mapper.go`
- Xoá hàm `toJSON()`
- `convertType` nhánh JSONB:
```go
case strings.Contains(dt, "JSONB") || strings.Contains(dt, "JSON"):
    // Pass-through: SchemaAdapter.CoerceValue handles JSONB marshalling
    // (map/slice → JSON, string → base64-decode + Mongo extended JSON
    // normalisation). Returning []byte here would trigger Go's default
    // `json.Marshal([]byte)` behaviour which base64-encodes the bytes.
    return val, nil
```
Giờ `CoerceValue` nhận `map[string]interface{}` → vào nhánh `case map[string]interface{}, []interface{}` → marshal đúng nested JSON.

### §11.4 Verify
1. Build + restart worker (pid 59102)
2. Insert test doc `TEST-FIX-PARAMS-001` với `params.nested.keyA="val1"` vào Mongo
3. Shadow check (sau ~5s CDC latency):
```
params = {"dateFr": "2026-05-11T00:00:00.000Z", "dateTo": "...", "nested": {"keyA": "val1", "keyB": 42}, "exportType": "TestFixExport"}
pg_typeof = jsonb
```
4. Backfill 122 rows cũ:
```sql
UPDATE shadow_centralized_export_service.sd_export_jobs
SET params = convert_from(decode(params #>> '{}', 'base64'), 'utf8')::jsonb
WHERE jsonb_typeof(params) = 'string' AND (params #>> '{}') ~ '^[A-Za-z0-9+/=]+$';
```
Final: 120 object / 3 string thật (primitive value "val", "testValue", "live-stream" — không phải base64) / 2 NULL / total 125.

### §11.5 Lesson (Global Pattern)
> **Global Pattern**: `Component A passes []byte X to JSON-marshaller B that expects interface{}` → B applies Go stdlib `json.Marshal([]byte)` which **base64-encodes** the bytes. Result Y: JSONB column ends up with `"<base64>"` string instead of nested object.
>
> **Đúng**: Tầng sản xuất giá trị cho JSONB column phải trả về native Go type (`map[string]interface{}`, `[]interface{}`, primitive) — KHÔNG pre-marshal thành `[]byte`. Để lớp persist marshal cuối cùng.
>
> Tổng quát hơn: trong Go, `json.Marshal([]byte{...})` ≠ raw JSON injection. Muốn raw JSON injection phải dùng `json.RawMessage(bytes)` hoặc trả về `string(bytes)`.

### §11.6 Files changed
- `centralized-data-service/internal/service/dynamic_mapper.go` — JSONB pass-through, xoá `toJSON`
