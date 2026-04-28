# Giải pháp chi tiết 6 gap — Registry + Masters

> Date: 2026-04-24 06:50 ICT
> Reviewer: Muscle
> Input: `10_gap_analysis_registry_masters.md` (6 gap)
> Output: solution spec + API availability matrix (Boss không thực thi, chỉ ra phương án)

---

## 0. API availability matrix (Stage 3 live test)

| Gap | Endpoint cần | Trạng thái | Test evidence |
|:-|:-|:-|:-|
| 2 | — (pure FE) | N/A | — |
| 3 | `GET /api/v1/system/connectors` | ✅ CÓ | `curl` trả 401 (auth-gated, route wired) |
| 4a Bridge | `POST /api/registry/:id/bridge` | ⚰️ 410 Gone | `registry_handler.go:647-651` stub |
| 4b Transform | `POST /api/registry/:id/transform` + NATS `cdc.cmd.batch-transform` | ✅ CÓ (legacy) | `registry_handler.go:654`; subscribe `worker_server.go:227` |
| 4c Transmute | `POST /api/v1/schedules/:id/run-now` + NATS `cdc.cmd.transmute` | ✅ CÓ (Sprint 5) | `worker_server.go:244` subscribe |
| 5a Add connector | CMS proxy POST `/api/v1/system/connectors` | ❌ THIẾU | CMS router chỉ có List/Get/Pause/Resume/Restart — không có POST |
| 5a kafka-connect direct | `POST http://kafka-connect:18083/connectors` | ✅ live (kafka-connect) | `curl POST` empty body trả 500 — endpoint accepts |
| 5b Activate snapshot | `POST /api/tools/trigger-snapshot/:table` | ✅ CÓ | `router.go:140` `registerDestructive` + handler `reconciliation_handler.go:368` publish NATS `cdc.cmd.debezium-signal` |
| 6 Cockpit | Chỉ là FE wrapper các route đã có | N/A | Không cần BE mới |

**Kết luận nhanh**: 4/6 gap dùng API đã có. 1 gap (5a Add Connector) cần BE mới. 1 gap (1 Arch doc) không cần code.

---

## Gap 1 — Architecture.md drift: append Section 5.5 Shadow→Master

**API**: không cần.
**Effort**: ~30 phút doc writing.
**Thực hiện** (khi Boss duyệt):
- Append vào `cdc-system/architecture.md` sau Section 5 (Critical Ingestion Path):

```markdown
## 5.5 Shadow → Master Materialisation Path (since Sprint 5)

Bổ sung so với section 5: ingestion từ Kafka không ghi thẳng vào PG target business.
Thay vào đó đi qua 2 tầng:

1. **Shadow Layer** (`cdc_internal.<table>`):
   - SinkWorker consume Kafka topic `cdc.goopay.<db>.<table>`.
   - Upsert raw event + system cols. SchemaManager auto-ALTER cho field mới (JSONB/TEXT).
   - Emit drift detection qua `cdc_internal.schema_proposal` table (admin approval needed).
   - Post-ingest: publish NATS `cdc.cmd.transmute-shadow` với source_id list.

2. **Transmuter Module**:
   - Subscribe `cdc.cmd.transmute-shadow` (per-row, real-time) và `cdc.cmd.transmute` (per-master, batch).
   - Check gate: `master.is_active=true AND schema_status='approved'` + `shadow.is_active=true`.
   - Load `cdc_mapping_rules WHERE is_active AND status='approved'`.
   - Apply gjson JsonPath + transform_fn per rule → typed cols.
   - Upsert `public.<master_name>` với OCC (`_source_ts`) + fencing.

3. **Master Layer** (`public.<master_name>`):
   - Tạo qua `MasterDDLGenerator.Apply` khi admin approve trong `/masters`.
   - Business-typed columns + system cols + RLS policy `rls_master_default_permissive`.
   - Governed bởi `cdc_internal.master_table_registry` (schema_status + is_active gate L2).

4. **TransmuteScheduler**:
   - Cron poll 60s + `FOR UPDATE SKIP LOCKED` + fencing token.
   - 3 mode: `cron` (scheduled), `immediate` (manual only), `post_ingest` (fan-out mỗi upsert).

Xem thêm: `internal/service/transmuter.go`, `internal/service/master_ddl_generator.go`, `internal/sinkworker/schema_manager.go`.
```

Plus mermaid diagram dặn dò 2-tier. Không code change. Không build impact.

---

## Gap 2 — TableRegistry Register Modal: xoá "airbyte" + "both"

**API**: không cần.
**Effort**: 5 phút.
**File**: `cdc-cms-web/src/pages/TableRegistry.tsx:442-444`.
**Current**:
```tsx
<Form.Item name="sync_engine" label="Sync Engine">
  <Select>
    <Select.Option value="airbyte">Airbyte</Select.Option>
    <Select.Option value="debezium">Debezium</Select.Option>
    <Select.Option value="both">Both</Select.Option>
  </Select>
</Form.Item>
```

**Fix**: giữ nguyên `sync_engine: 'debezium'` mặc định (line 429), xoá 2 option + disable select → hoặc replace bằng read-only Input.

**Fix spec** (Muscle sẽ thực thi):
```tsx
<Form.Item name="sync_engine" label="Sync Engine" initialValue="debezium">
  <Input disabled />
</Form.Item>
```

Hoặc xoá hẳn Form.Item (FE types/index.ts:24 đã `sync_engine: 'debezium'` literal type — backend chấp nhận).

**Verify**: `npx tsc --noEmit` EXIT=0 + register table mới → row insert với sync_engine='debezium'.

---

## Gap 3 — SyncStatusIndicator xoá hoặc repurpose

**API**: `GET /api/v1/system/connectors` ✅ CÓ (401 verified).
**Effort**: 15 phút (option A) hoặc 45 phút (option B).

**Option A — Xoá column + component** (simpler):
- File: `TableRegistry.tsx:15-51` → DELETE `SyncStatusIndicator` component.
- Column "Sync Engine" (line 306-313) → thay bằng:
  ```tsx
  {
    title: 'Sync Engine', dataIndex: 'sync_engine', width: 120,
    render: (v: string) => <Tag color="blue">{v || 'debezium'}</Tag>,
  }
  ```
- Thêm help text: "Connector status: xem trang Debezium Command Center" với link `/sources`.

**Option B — Fetch real status** (preferred nếu Boss muốn observability):
- Replace `fetchStatus` logic:
  ```tsx
  const fetchStatus = useCallback(async () => {
    try {
      const { data } = await cmsApi.get('/api/v1/system/connectors');
      const connector = data.data?.find((c) =>
        c.config?.['collection.include.list']?.includes(sourceTable)
      );
      setStatus(connector ? connector.status?.connector?.state : 'not_configured');
    } catch { setStatus('error'); }
  }, [sourceTable]);
  ```
- Match logic: compare `source_db.source_table` với `connector.config.collection.include.list`.

**Khuyến nghị**: Option A cho Sprint gấp. Option B nếu Boss muốn cockpit view.

---

## Gap 4 — Bridge/Batch/Transform buttons

**API**:
- `POST /api/registry/:id/bridge` → 410 Gone ⚰️.
- `POST /api/registry/:id/transform` → 202 live (dispatch NATS `cdc.cmd.batch-transform`). Legacy flow.
- `POST /api/v1/schedules/:id/run-now` → live (Sprint 5) — trigger Transmute cho 1 master.

**Effort**: 15 phút.
**File**: `TableRegistry.tsx:362-373`.

**Current**:
```tsx
<Button onClick={(e) => handleBridge(e, record.id)}>Đồng bộ</Button>
<Button onClick={(e) => handleBridge(e, record.id, true)}>Batch</Button>
<Button onClick={(e) => handleTransform(e, record.id)}>Chuyển đổi</Button>
```

**Fix spec**:
1. XOÁ 2 button Đồng bộ + Batch (410 Gone — không nên trong UI).
2. XOÁ `handleBridge` function (line 232-243).
3. Button "Chuyển đổi" → relabel "Transmute Masters" → mở modal list master cho source_shadow này (query `/api/v1/masters?source_shadow=<target>`), cho admin click "Run Now" per master.

Hoặc đơn giản hơn: xoá "Chuyển đổi" + direct admin sang `/schedules` hoặc `/masters` để Run Now.

**Khuyến nghị**:
- Xoá 3 button hoàn toàn (legacy).
- Thêm link "Manage Masters →" điều hướng `/masters?source_shadow=${record.target_table}`.

---

## Gap 5a — Add Debezium connector UI (BE + FE)

**API**: ❌ **THIẾU** CMS endpoint. Kafka-connect REST live nhưng không expose ra FE (security + audit).
**Effort**: ~2-3h (BE 1h + FE 1.5h).

### Phần BE (cdc-cms-service)

**New handler**: `SystemConnectorsHandler.Create` trong `system_connectors_handler.go`.
```go
// POST /api/v1/system/connectors
// Body: {name: string, config: map[string]any}
// Forward → kafka-connect POST /connectors
func (h *SystemConnectorsHandler) Create(c *fiber.Ctx) error {
    var req struct {
        Name   string         `json:"name"`
        Config map[string]any `json:"config"`
    }
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "bad_json"})
    }
    if !connectorNameRe.MatchString(req.Name) {
        return c.Status(400).JSON(fiber.Map{"error": "invalid_connector_name"})
    }
    payload := map[string]any{"name": req.Name, "config": req.Config}
    body, _ := json.Marshal(payload)
    resp, err := http.Post(h.kafkaConnectURL+"/connectors", "application/json", bytes.NewReader(body))
    if err != nil {
        return c.Status(502).JSON(fiber.Map{"error": "kafka_connect_unreachable", "detail": err.Error()})
    }
    defer resp.Body.Close()
    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 300 {
        return c.Status(resp.StatusCode).Send(respBody)
    }
    return c.Status(201).Send(respBody)
}
```

**Route** (`router.go`):
```go
registerDestructive("/v1/system/connectors", systemConnectorsHandler.Create)  // POST
```

### Phần FE (cdc-cms-web)

**File mới** hoặc edit `SourceConnectors.tsx`:
- Thêm button "New Connector" ở header.
- Modal với 2 tab:
  - Tab "Template" — dropdown (MongoDB / MySQL / Postgres) → pre-fill connector.class + common fields.
  - Tab "JSON" — raw config editor.
- Required fields (MongoDB template):
  ```json
  {
    "name": "goopay-mongodb-cdc-<service>",
    "config": {
      "connector.class": "io.debezium.connector.mongodb.MongoDbConnector",
      "mongodb.connection.string": "mongodb://gpay-mongo:27017/?replicaSet=rs0",
      "database.include.list": "...",
      "collection.include.list": "...",
      "topic.prefix": "cdc.goopay",
      "signal.data.collection": "<db>.debezium_signal",
      "capture.mode": "change_streams_update_full",
      "snapshot.mode": "initial",
      "key.converter": "io.confluent.connect.avro.AvroConverter",
      "value.converter": "io.confluent.connect.avro.AvroConverter"
    }
  }
  ```
- Mutation: `cmsApi.post('/api/v1/system/connectors', payload)` với Idempotency-Key + reason ≥10 chars.
- Validation: name regex `^[a-z0-9-]+$`, 1-63 chars.

### Verify
- `go build ./...` xanh.
- `npx tsc --noEmit` xanh.
- Live test: click "New Connector" → dummy config → kafka-connect trả 201.

---

## Gap 5b — Activate stream (Snapshot Now) button

**API**: `POST /api/tools/trigger-snapshot/:table` ✅ **CÓ** (`router.go:140`, `reconciliation_handler.TriggerSnapshot`).
**Effort**: 45 phút.

### Handler hiện tại (đã verified Probe H)
```go
// reconciliation_handler.go:368
payload, _ := json.Marshal(map[string]string{
  "type":       "signal-snapshot",
  "database":   body.Database,
  "collection": body.Collection,
})
h.nats.Conn.Publish("cdc.cmd.debezium-signal", payload)
```
Worker `HandleDebeziumSignal` (recon_handler.go:264) ghi vào Mongo `<db>.debezium_signal` collection → Debezium consume + trigger incremental snapshot.

### FE fix spec
**File**: `TableRegistry.tsx`.
Thêm button "Snapshot Now" ở cột "Thao tác" (sau "Tạo Table"):
```tsx
<Tooltip title="Trigger Debezium incremental snapshot cho collection này">
  <Button
    size="small"
    icon={<ThunderboltOutlined />}
    loading={actionLoadingId === record.id}
    onClick={(e) => { e.stopPropagation(); openSnapshotConfirm(record); }}
  >
    Snapshot Now
  </Button>
</Tooltip>
```

Handler:
```tsx
const triggerSnapshot = async (record: TRegistry, reason: string) => {
  setActionLoadingId(record.id);
  try {
    await cmsApi.post(
      `/api/tools/trigger-snapshot/${record.source_table}`,
      { database: record.source_db, collection: record.source_table, reason },
      { headers: { 'Idempotency-Key': `snapshot-${record.id}-${Date.now()}` } },
    );
    message.success(`Snapshot dispatched for ${record.source_table}`);
  } catch (err: any) {
    message.error(err.response?.data?.error || 'Snapshot failed');
  } finally {
    setActionLoadingId(null);
  }
};
```

Reuse `ConfirmDestructiveModal` cho reason input (≥10 chars).

### Verify
- Click → CMS publish NATS → Worker ghi Mongo debezium_signal → Kafka topic có event mới.
- Live: before/after `SELECT COUNT(*) FROM cdc_internal.<target>` tăng.

---

## Gap 6 — Cockpit `/source-to-master` (optional wizard)

**API**: không cần mới. Wrapper các route đã có:
- `/api/v1/system/connectors` (Gap 5a)
- `/api/registry` + `/api/registry/:id/create-default-columns`
- `/api/tools/trigger-snapshot/:table` (Gap 5b)
- `/api/v1/schema-proposals`
- `/api/mapping-rules` + `/api/v1/mapping-rules/preview`
- `/api/v1/masters` + `/api/v1/masters/:name/approve`
- `/api/v1/schedules`

**Effort**: 3-4h (nice-to-have).

### Design spec
File mới: `cdc-cms-web/src/pages/SourceToMasterWizard.tsx`.

```tsx
<Steps current={step}>
  <Step title="1. Debezium Connector" />
  <Step title="2. Register Shadow" />
  <Step title="3. Create Shadow Table" />
  <Step title="4. Activate Stream" />
  <Step title="5. Wait for Ingest" />
  <Step title="6. Review Proposals" />
  <Step title="7. Approve Proposals" />
  <Step title="8. Mapping Rules" />
  <Step title="9. Create Master" />
  <Step title="10. Approve Master" />
  <Step title="11. Schedule Transmute" />
</Steps>
```

Mỗi step = 1 form hoặc link sang trang con. Progress state lưu localStorage `source-to-master-<source_table>`.

**KHÔNG BẮT BUỘC cho MVP**. Nếu Boss chưa vội → skip Gap 6.

---

## 1. Tổng kết effort + priority

| Priority | Gap | Effort | API status |
|:-|:-|:-|:-|
| P0 | 2. Airbyte option stale | 5ph | Không cần |
| P0 | 4. Bridge/Transform buttons | 15ph | Xoá call 410 Gone |
| P1 | 3. SyncStatus column (Option A xoá) | 15ph | Không cần |
| P1 | 5b. Snapshot Now button | 45ph | ✅ API sẵn |
| P1 | 1. Architecture.md section 5.5 | 30ph | Doc only |
| P2 | 3. SyncStatus (Option B fetch) | 45ph | ✅ API sẵn |
| P2 | 5a. Add connector UI | 2-3h | ❌ BE handler mới |
| P3 | 6. Cockpit wizard | 3-4h | Wrapper only |

**P0-P1 total**: ~1h50 (cleanup + Snapshot Now + arch doc).
**P0-P2 total**: ~5h.
**Full (P0-P3)**: ~9h.

## 2. Thực thi khi Boss OK

Nếu Boss duyệt: Muscle sẽ làm theo 7-stage SOP riêng cho mỗi gap, bắt đầu từ P0. Mỗi gap có:
- `02_plan_<gap>.md`
- Stage 3 EXECUTE (edit code theo spec trên).
- Stage 4 VERIFY: tsc + vite HMR + API curl evidence.
- Stage 5 DOCUMENT: `03_implementation_<gap>.md`.
- Stage 6 LESSON (nếu sơ sót).
- Stage 7 CLOSE: APPEND progress + report Boss.

**KHÔNG LÀM BÂY GIỜ** — chờ lệnh.
