# Gap Analysis — /registry + /masters vs architecture.md

> Date: 2026-04-24 06:30 ICT
> Reviewer: Muscle
> Scope: Boss yêu cầu review 2 FE routes `/masters` + `/registry`, mô tả full luồng từ Connect Source tới Master có data.
> Source docs: `cdc-system/architecture.md` (740 LOC), `cdc-cms-web/src/pages/TableRegistry.tsx` (462 LOC), `cdc-cms-web/src/pages/MasterRegistry.tsx` (323 LOC), `cdc-cms-service/internal/router/router.go`.

---

## 1. Tóm tắt trong 3 câu

1. Architecture.md mô tả pipeline **1 tầng** (Mongo → Debezium/Airbyte → Kafka/NATS → Worker → PG target + metadata), còn code Sprint 5 đã tiến hoá thành **2 tầng** — Shadow (`cdc_internal.*`) + Master (`public.*_master`) với Transmuter Module giữa 2 tầng. Doc drift.
2. `/registry` (Table Registry) quản trị **Shadow layer** — nguồn ingestion CDC. `/masters` (Master Registry) quản trị **Master layer** — nơi business data typed + governed + RLS. 2 page bổ sung cho nhau theo thứ tự: Register Shadow → Create Master.
3. Operator workflow end-to-end cần **11 bước** qua 5 page (`/sources`, `/registry`, `/registry/:id/mappings`, `/schema-proposals`, `/masters`, `/schedules`) để "connect source" → "data chảy vào Master". Hiện thiếu 2 button UI (Step 1 add connector + Step 4 activate stream) → operator phải curl thủ công.

---

## 2. Kiến trúc 2 tầng (reality vs architecture.md)

### Architecture.md — 1 tầng (section 5 Critical Ingestion Path)
```
Mongo → Debezium → Kafka → Worker[SchemaInspector/Mapper/Adapter] → PostgreSQL
```
Target = 1 tầng PG, schema evolve inline qua SchemaInspector + ALTER TABLE.

### Sprint 5 reality — 2 tầng
```
Mongo ──→ Debezium (kafka-connect) ──→ Kafka (cdc.goopay.*)
                                            ↓
                               SinkWorker (consumer, machine_id fencing)
                                            ↓
                     cdc_internal.<table> (SHADOW: raw + auto-ALTER JSONB/TEXT)
                                            ├─ SchemaManager → schema_proposal (drift flagging)
                                            └─ publishTransmuteTrigger → NATS cdc.cmd.transmute-shadow
                                                                              ↓
                     TransmuteModule (worker) ←──────────────────────────────┘
                                            ↓
                           [rule check + jsonpath + transform_fn]
                                            ↓
                     public.<table>_master (MASTER: typed cols + RLS + indexes)
                                            ↑
                     TransmuteScheduler (cron 60s tick + FOR UPDATE SKIP LOCKED)
```

Master layer là **mới** so với architecture.md, xuất hiện từ Sprint 5 §R8 (Master DDL Generator) + §R9 (Schema Proposal Workflow).

---

## 3. Vai trò 2 page

### `/registry` — TableRegistry (Shadow layer metadata)
- **Backing table**: `cdc_table_registry` (CMS-owned).
- **Role**: Register 1 Mongo collection → Debezium source topic → 1 shadow table `cdc_internal.<target_table>`.
- **Columns hiển thị**: source_db, source_table, target_table, sync_engine, priority, pk_field, is_active, sync_status, is_table_created, created_at.
- **Actions/row**:
  - "Tạo Table" → tạo shadow DDL + system cols.
  - "Tạo Field MĐ" → add system default cols vào table có sẵn.
  - "Đồng bộ" (Bridge) + "Batch" + "Chuyển đổi" → legacy Bridge pipeline (Sprint 4 đã retire implementation).
  - "Quét field" (async scan-fields) — đọc `_raw_data` phát hiện field mới.
  - Click row → `/registry/:id/mappings` (mapping rules CRUD).
- **Register Modal**: source_db, source_table, target_table, sync_engine, priority, pk, timestamp_field.

### `/masters` — MasterRegistry (Master layer metadata)
- **Backing table**: `cdc_internal.master_table_registry` (Sprint 5 migration).
- **Role**: Declare 1 Master table materialisation spec → CREATE TABLE `public.<master_name>` via DDL Generator khi approve.
- **Columns hiển thị**: master_name, source_shadow, transform_type, schema_status (pending_review/approved/rejected/failed), is_active, schema_reviewed_by, schema_reviewed_at.
- **Actions/row**:
  - "Create Master" wizard → insert row với schema_status='pending_review'.
  - "Approve" → CMS publish NATS `cdc.cmd.master-create` → Worker MasterDDLGenerator.Apply → CREATE TABLE + indexes + RLS.
  - "Reject" → schema_status='rejected'.
  - "Toggle Active" switch → flip `is_active` (chỉ bật khi schema_status='approved' — CHECK constraint L2 gate).
- **Create Modal**: master_name, source_shadow (dropdown shadow tables từ table_registry), transform_type (copy_1_to_1/filter/aggregate/group_by/join), spec JSON, reason ≥10 chars.

---

## 4. Luồng FULL từ "Connect Source" → Master có data (11 bước)

| # | Bước | Actor | UI | API | Verify |
|:-|:-|:-|:-|:-|:-|
| 1 | Create Debezium connector cho Mongo collection | DevOps/Boss | **🚨 Không có UI** — phải curl | `POST http://kafka-connect:8083/connectors` | `GET /api/v1/system/connectors` trên `/sources` |
| 2 | Register shadow table | Admin | `/registry` "Register Table" modal | `POST /api/registry` | Row xuất hiện trong `/registry` group theo source_db |
| 3 | Tạo shadow DDL | Admin | `/registry` click "Tạo Table" | `POST /api/registry/:id/create-default-columns` | `\d cdc_internal.<target>` hiện system cols |
| 4 | Activate Debezium stream (snapshot signal) | Admin | **🚨 Không có UI** — phải gọi NATS hoặc `/sources` restart | NATS `cdc.cmd.debezium-signal` | Kafka topic `cdc.goopay.<db>.<table>` có message |
| 5 | SinkWorker ingest → Shadow | Auto | — | Kafka → SinkWorker → `cdc_internal.<table>` upsert | `SELECT COUNT(*) FROM cdc_internal.<table>` > 0; SinkWorker log "upserted" |
| 6 | SchemaManager detect drift + proposal | Auto | `/schema-proposals` badge pending | `GET /api/v1/schema-proposals?status=pending` | Proposal rows với table_layer='shadow' hoặc 'master' |
| 7 | Approve proposal (optional override) | Admin | `/schema-proposals` Approve modal | `POST /api/v1/schema-proposals/:id/approve` | ALTER TABLE log + `cdc_mapping_rules` row mới |
| 8 | Add custom mapping rules | Admin | `/registry/:id/mappings` → Add Mapping + Preview | `POST /api/mapping-rules` + `POST /api/v1/mapping-rules/preview` | Rule xuất hiện, Preview render 3 sample extracted values |
| 9 | Create Master table | Admin | `/masters` "Create Master" wizard | `POST /api/v1/masters` | Row schema_status='pending_review' |
| 10 | Approve Master → materialise DDL | Admin | `/masters` Approve modal | `POST /api/v1/masters/:name/approve` | `\d public.<master>` exist, log "master DDL applied rls_applied:true" |
| 11 | Activate + schedule Transmute | Admin | `/masters` toggle is_active + `/schedules` cron | `PATCH /api/v1/masters/:name/toggle-active` + `POST /api/v1/schedules` | `transmute complete scanned:N inserted:N` worker log |

---

## 5. Gap — stale / broken / missing

### Gap 1 (UI stale) — /registry Register Modal có airbyte + both
```tsx
<Form.Item name="sync_engine" label="Sync Engine">
  <Select>
    <Select.Option value="airbyte">Airbyte</Select.Option>     ← ❌ retired Sprint 3
    <Select.Option value="debezium">Debezium</Select.Option>
    <Select.Option value="both">Both</Select.Option>            ← ❌ retired
  </Select>
</Form.Item>
```
**Impact**: Admin chọn airbyte → row insert nhưng không có pipeline nào consume → data không chảy. Confusing.

### Gap 2 (UI stale) — SyncStatusIndicator hardcode 'n/a'
`TableRegistry.tsx:18-24`:
```tsx
const fetchStatus = useCallback(async () => {
  // Legacy per-entry status endpoint retired in Sprint 4.
  setStatus('n/a');
}, [id, engine]);
```
**Impact**: Column "Sync Engine" luôn render badge "N/A" — dead UI. Operator không biết connector status; phải sang `/sources` Debezium Command Center.

### Gap 3 (UI stale) — 2 button Bridge trả 410 Gone, 1 button Transform còn chạy
```tsx
<Button onClick={(e) => handleBridge(e, record.id)}>Đồng bộ</Button>       // 410 Gone
<Button onClick={(e) => handleBridge(e, record.id, true)}>Batch</Button>    // 410 Gone
<Button onClick={(e) => handleTransform(e, record.id)}>Chuyển đổi</Button> // live
```
Verified `registry_handler.go:647-651`:
```go
func (h *RegistryHandler) Bridge(c *fiber.Ctx) error {
  return c.Status(410).JSON(fiber.Map{
    "error": "bridge endpoint retired — use POST /api/v1/tables/:name/transmute",
  })
}
```
`Transform()` (line 654) vẫn publish NATS `cdc.cmd.batch-transform` — dead behaviour vì Sprint 5 đã chuyển sang TransmuteModule (shadow → master) thay vì batch-transform (raw→cols cũ).

**Impact**:
- "Đồng bộ" + "Batch" click → message.error hiển thị "bridge endpoint retired..." — operator confusion.
- "Chuyển đổi" click → 202 accepted nhưng luồng batch-transform đã superseded bởi Transmuter qua `/masters` + `/schedules`.

### Gap 4 (UX gap) — Không có UI cho Step 1 + Step 4
- **Step 1 (add Debezium connector)**: Hiện `/sources` chỉ List/Restart/Pause connector đã exist. **Không có button "Add new connector"**. DevOps phải curl `POST /connectors` với JSON config.
- **Step 4 (Activate stream)**: Register shadow xong, phải trigger Debezium incremental snapshot. UI không có button. Phải publish NATS thủ công hoặc restart connector.

### Gap 5 (Doc drift) — architecture.md chưa update 2 tầng
Architecture.md section 4-5 vẽ pipeline 1 tầng. Sprint 5 đã có:
- Shadow layer (`cdc_internal.*`).
- Master layer (`public.*_master`) + Transmuter + master_table_registry + schema_proposal.
- Dashboard.1-3 pages (Masters, Proposals, Schedules).
**Impact**: New dev / outside reviewer nhìn arch → hiểu sai. Cần append section 5.5 "Transmuter + Master layer" vào architecture.md.

### Gap 6 (observability) — SyncStatus column bỏ trống
Architecture.md 9.3 "Operability" nói command plane phải observable. Column Sync Status trong `/registry` là entry point quan trọng nhưng luôn "N/A". Cần:
- Hoặc link column → `/sources` lọc sẵn connector match source_db.
- Hoặc thay thành "Shadow status" (row count + last _synced_at).

### Gap 7 (no wizard end-to-end) — 11 bước qua 5 page
Operator mới sẽ lạc. Không có:
- Progress indicator "bạn đang step 4/11".
- Wizard gộp Step 2+3 (Register + Create Table).
- Wizard gộp Step 9+10+11 (Create Master + Approve + Schedule).

---

## 6. Nhận định tổng thể

### Điểm mạnh hiện tại
- 2 page role rõ ràng: shadow vs master, separation of concerns đúng Sprint 5 spec.
- Master approval flow với audit trail (reason ≥10 chars, reviewed_by/at) + NATS command plane.
- Preview button trên MappingFieldsPage kiểm tra JsonPath trước khi save (Dashboard.1).
- Transmute schedule support 3 mode (cron/immediate/post_ingest) + Run Now.

### Điểm yếu
- 3 UI elements dead code (airbyte option, SyncStatusIndicator, Bridge/Batch/Transform buttons) → operator confusion.
- 2 step end-to-end không có UI (add connector, activate stream) → operator phải curl.
- Arch doc drift — 2 tầng chưa được document chính thức.
- Không có cockpit tổng hợp progress end-to-end.

### Đề xuất (không trong task, chỉ để boss tham khảo)
1. **FE cleanup (30 phút)**: Xoá airbyte/both options, xoá SyncStatusIndicator, disable/remove 3 Bridge buttons. Chuyển logic "Đồng bộ" thành link tới `/sources` restart connector.
2. **Add "New Connector" UI trong /sources (~2h)**: Form field: name, mongodb.hosts, collection, signal.data.collection → POST qua CMS `/api/v1/system/connectors` (cần handler mới).
3. **Add "Activate stream" button trong /registry (~1h)**: Sau khi Register + Create Table, hiện button → publish NATS debezium-signal với collection path.
4. **Update architecture.md (~1h)**: Append section 5.5 về Shadow→Master với Transmuter, Master DDL Generator, Schema Proposal workflow.
5. **Cockpit page /source-to-master (~3h)**: Wizard 11 bước, progress bar, link qua từng step.

---

## 7. Conclusion

`/masters` đúng spec Sprint 5 — layer governance hoàn chỉnh. `/registry` còn nợ cleanup Sprint 4 (Airbyte remnants). Luồng end-to-end **có thể chạy được** nhưng cần vận hành thủ công 2 step (Debezium connector add + stream activate). Kiến trúc thực tế tiến hoá hơn architecture.md 1 bậc — doc cần update.

Nếu Boss muốn "click 1 lần là data chảy tới Master" → cần 3 thứ: FE cleanup + 2 wizard mới + arch doc append. Không bắt buộc cho MVP, nhưng là công nợ kỹ thuật rõ ràng.
