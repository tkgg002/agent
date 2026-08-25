# 🟢 AUDIT REPORT ROUND 2 — Fix ReconB Master Binding (Post-Fix)
**Phiên:** 2026-08-25 09:57–10:12 UTC+7  
**Agent:** Gemini  
**Phương pháp:** Đọc 100% code thật, query DB thật, trace E2E 7 hops, không suy diễn

---

## I. TỔNG KẾT

| Tiêu chí | Kết quả |
|----------|---------|
| Critical bugs | **0** |
| Lỗi logic | **0** |
| Cảnh báo nhỏ | **2** (dead code DTO, TS naming) |
| E2E trace | **✅ 7/7 hops verified** |
| Build | **✅ Go CMS + CDS + TypeScript** |
| Tests | **✅ 9/9 pass** |
| DB schema match | **✅ SQL khớp 100% DDL** |
| Binding uniqueness | **✅ Không có duplicate** |

---

## II. AUDIT TỪNG FILE — TỪNG DÒNG

### File 1: [`recon_check.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_check.go) (CMS)

| Dòng | Thay đổi | Đánh giá |
|------|----------|----------|
| 35 | `MasterSchema string \`json:"master_schema,omitempty"\`` | ✅ PascalCase Go + snake_case JSON + omitempty (optional) |
| 36 | `MasterTable  string \`json:"master_table,omitempty"\`` | ✅ Đồng bộ pattern với StartTime/EndTime |

`Validate()` chỉ bắt buộc `Table` + `TypeRecon` → **đúng** vì master fields chỉ cần cho segment B.

---

### File 2: [`recon_dto.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/dto/recon_dto.go) (CMS)

| Dòng | Thay đổi | Đánh giá |
|------|----------|----------|
| 12 | `MasterSchema string \`json:"master_schema"\`` | ⚠️ **DEAD CODE** |
| 13 | `MasterTable  string \`json:"master_table"\`` | ⚠️ **DEAD CODE** |

> [!WARNING]
> `dto.ReconScopeRequest` **KHÔNG ĐƯỢC DÙNG** ở bất kỳ đâu. CMS handler dùng local struct `reconScopeRequest` (file `reconciliation_handler.go`). Thêm field vào DTO này vô hại nhưng gây duplicate code. Nên dọn dẹp sau.

---

### File 3: [`reconciliation_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler.go) (CMS)

| Dòng | Thay đổi | Đánh giá |
|------|----------|----------|
| 66 | `MasterSchema string \`json:"master_schema"\`` | ✅ Parse từ body khi `TriggerCheckAll` dùng embedded struct |
| 67 | `MasterTable  string \`json:"master_table"\`` | ✅ |

`resolveTargetTable` không dùng MasterSchema/MasterTable → **đúng** vì nó resolve target cho Tier 1 (Source→Shadow).

---

### File 4: [`reconciliation_handler_commands.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go) (CMS)

**TriggerCheck (dòng 37-56):**

| Dòng | Code | Đánh giá |
|------|------|----------|
| 42 | `MasterSchema string \`json:"master_schema"\`` | ✅ Parse từ body |
| 43 | `MasterTable  string \`json:"master_table"\`` | ✅ |
| 54 | `MasterSchema: req.MasterSchema,` | ✅ Truyền vào command |
| 55 | `MasterTable:  req.MasterTable,` | ✅ |

**TriggerCheckAll (dòng 96-128):**

| Dòng | Code | Đánh giá |
|------|------|----------|
| 105 | `MasterSchema: scope.MasterSchema,` | ✅ Nhánh single table |
| 106 | `MasterTable:  scope.MasterTable,` | ✅ |
| 127 | `MasterSchema: scope.MasterSchema,` | ✅ Nhánh wildcard `*` |
| 128 | `MasterTable:  scope.MasterTable,` | ✅ |

> [!NOTE]
> **Routing quan trọng:** Frontend gọi `POST /api/reconciliation/check` (không có `:table`) → đi vào `TriggerCheckAll` (không phải `TriggerCheck`). Nhưng vì body có `table: "payment_bills"`, `resolveTargetTable` trả về đúng → nhánh single table (dòng 97-118) chạy với đủ master fields. ✅

---

### File 5: [`recon_check_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go) (CDS)

| Dòng | Code | Đánh giá |
|------|------|----------|
| 28 | `MasterSchema string \`json:"master_schema"\`` | ✅ Nhận từ CMS wire |
| 29 | `MasterTable  string \`json:"master_table"\`` | ✅ |
| 215 | `MasterSchema: payload.MasterSchema,` | ✅ Truyền vào event |
| 216 | `MasterTable:  payload.MasterTable,` | ✅ |

---

### File 6: [`recon_job_worker.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_job_worker.go) (CDS)

| Dòng | Code | Đánh giá |
|------|------|----------|
| 49 | `MasterSchema string \`json:"master_schema,omitempty"\`` | ✅ Event struct |
| 50 | `MasterTable  string \`json:"master_table,omitempty"\`` | ✅ |
| 79 | `ExecuteSegment(... masterSchema, masterTable string)` | ✅ Interface |
| 256 | `event.MasterSchema, event.MasterTable` | ✅ Caller |
| 502 | `masterSchema, masterTable string` | ✅ Adapter (backward compat, không dùng) |

---

### File 7: [`recon_stream_bucket_engine.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go) (CDS)

| Dòng | Code | Đánh giá |
|------|------|----------|
| 409 | `masterSchema, masterTable string` | ✅ ExecuteSegment params |
| 412 | `e.executeSegmentB(ctx, entry, startTime, endTime, masterSchema, masterTable)` | ✅ |
| 419 | `e.executeSegmentB(ctx, entry, startTime, endTime, masterSchema, masterTable)` | ✅ "both" case |
| 544 | `masterSchema, masterTable string` | ✅ executeSegmentB params |
| 553 | `if masterSchema != "" && masterTable != ""` | ✅ Guard |
| 554 | `ref = e.lookupMasterRefExact(ctx, masterSchema, masterTable)` | ✅ |
| 558 | `ref = e.lookupMasterRef(ctx, entry.TargetTable)` | ✅ Fallback |

**`lookupMasterRefExact` SQL audit:**

```sql
SELECT mb.id, mb.master_schema, mb.master_table, sb.shadow_schema, sb.shadow_table
  FROM cdc_system.master_binding mb
  JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
 WHERE mb.is_active = true AND mb.schema_status = 'approved'
   AND mb.master_schema = ? AND mb.master_table = ?
```

| SQL element | DB DDL (Migration 031/032) | Khớp? |
|-------------|---------------------------|-------|
| `cdc_system.master_binding` | Table exists | ✅ |
| `mb.id` | `id BIGSERIAL PRIMARY KEY` | ✅ |
| `mb.master_schema` | `master_schema TEXT NOT NULL` | ✅ |
| `mb.master_table` | `master_table TEXT NOT NULL` | ✅ |
| `mb.shadow_binding_id` | `shadow_binding_id BIGINT REFERENCES` | ✅ |
| `mb.is_active` | `is_active BOOLEAN DEFAULT TRUE` | ✅ |
| `mb.schema_status` | `schema_status TEXT DEFAULT 'pending'` | ✅ |
| `sb.shadow_schema` | `shadow_schema TEXT NOT NULL` | ✅ |
| `sb.shadow_table` | `shadow_table TEXT NOT NULL` | ✅ |
| `MasterBindingRef` struct fields | `gorm:"column:..."` tags khớp | ✅ |

**Uniqueness check:**

```sql
-- Query: SELECT count(*) FROM master_binding WHERE is_active=true AND schema_status='approved' 
--        GROUP BY master_schema, master_table HAVING count(*) > 1
-- Result: 0 rows → NO DUPLICATES ✅
```

---

### File 8: [`recon_job_worker_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_job_worker_test.go) (CDS)

| Dòng | Code | Đánh giá |
|------|------|----------|
| 125 | `func (m *mockChunkStreamEngine) ExecuteSegment(... masterSchema, masterTable string)` | ✅ Khớp interface |

---

### File 9: [`DataIntegrity.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx) (Frontend)

| Dòng | Code | Đánh giá |
|------|------|----------|
| 320 | `masterSchema: row?.master_schema \|\| undefined,` | ✅ |
| 321 | `masterTable: row?.target_table \|\| undefined,` | ✅ **ĐÚNG** (xem chứng minh dưới) |

**Chứng minh `row?.target_table` là đúng (không phải `row?.master_table`):**
- SQL report: `CASE WHEN segment='shadow_master' THEN master_table ELSE shadow_table END AS target_table`
- Khi segment B: `target_table` = `master_table` từ DB ✅
- `ReconRow` interface KHÔNG có field `master_table`, CHỈ có `target_table: string` ✅
- DB evidence: `payment_bills` segment B → `master_table = payment_bills` → `target_table = payment_bills` ✅

**Edge case — segment A row:**
- DB evidence: segment A row → `master_schema = NULL`, `master_table = NULL`
- Frontend: `row?.master_schema` = `null` → `|| undefined` → `undefined` → không gửi
- CDS engine: `masterSchema = ""` → guard `if "" != ""` = false → fallback ✅

---

### File 10: [`useReconStatus.ts`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts) (Frontend)

| Dòng | Code | Đánh giá |
|------|------|----------|
| 201 | `masterSchema?: string;` | ✅ Type param |
| 202 | `masterTable?: string;` | ✅ |
| 217 | `masterSchema,` | ✅ Destructure |
| 218 | `masterTable,` | ✅ |
| 233 | `master_schema: masterSchema \|\| undefined,` | ✅ camelCase → snake_case |
| 234 | `master_table: masterTable \|\| undefined,` | ✅ |

> [!NOTE]
> **Lưu ý nhỏ (không ảnh hưởng runtime):** `start_time`/`end_time` trong params type dùng snake_case trong khi các field khác dùng camelCase. Đây là code cũ, không thuộc scope fix.

---

## III. E2E TRACE VERIFICATION (DB EVIDENCE)

```
HOP 1: DataIntegrity.tsx → row?.master_schema = "master_payment_bill_service", row?.target_table = "payment_bills"
        ✅ DB: SELECT master_schema, CASE WHEN 'shadow_master' THEN master_table END AS target_table
            → "master_payment_bill_service", "payment_bills"

HOP 2: useReconStatus.ts → body { master_schema: "master_payment_bill_service", master_table: "payment_bills" }
        ✅ camelCase → snake_case mapping correct

HOP 3: CMS TriggerCheckAll → req.reconScopeRequest has MasterSchema, MasterTable (via embedded struct parse)
        → scope.MasterSchema, scope.MasterTable → cmd.MasterSchema, cmd.MasterTable
        ✅ Verified code lines 42-43, 54-55, 105-106

HOP 4: CMS nats_command_bus.go → json.Marshal(cmd) → NATS wire
        ✅ json tags "master_schema,omitempty" + "master_table,omitempty"

HOP 5: CDS recon_check_handler.go → payload.MasterSchema → event.MasterSchema
        ✅ Verified lines 28-29, 215-216

HOP 6: CDS recon_job_worker.go → event.MasterSchema → ExecuteSegment(..., masterSchema, masterTable)
        ✅ Verified line 256

HOP 7: CDS recon_stream_bucket_engine.go → lookupMasterRefExact("master_payment_bill_service", "payment_bills")
        ✅ SQL WHERE master_schema=? AND master_table=? → binding #4 (payment_bills, copy_1_to_1)
        ✅ DB: 0 duplicate bindings for same (master_schema, master_table)
```

---

## IV. SỤY DIỄN & BÁO CÁO LÁO CHECK

| # | Kiểm tra | Kết quả |
|---|----------|---------|
| 1 | `row?.target_table` = master table name khi segment B? | ✅ Chứng minh bằng SQL + DB query |
| 2 | Segment A row gửi master_schema? | ✅ DB: NULL → frontend gửi undefined → engine guard → fallback |
| 3 | SQL `lookupMasterRefExact` khớp DDL? | ✅ So sánh từng column với Migration 031/032 |
| 4 | `(master_schema, master_table)` unique? | ✅ DB query: 0 duplicates |
| 5 | Build pass thật? | ✅ `go build` exit 0 (CMS + CDS), `npx tsc --noEmit` exit 0 |
| 6 | Tests pass thật? | ✅ 9/9 PASS (terminal output verified) |
| 7 | Frontend route → đúng handler? | ✅ `POST /check` → `TriggerCheckAll` (router.go:174) |
| 8 | `TriggerCheckAll` truyền master fields ở CẢ 2 nhánh? | ✅ Lines 105-106 + 127-128 |

**Kết luận:** Không phát hiện suy diễn hoặc báo cáo láo trong round 2.

---

## V. CÁC CẢNH BÁO NHỎ (KHÔNG ẢNH HƯỞNG RUNTIME)

| # | Mô tả | Mức độ | Action |
|---|-------|--------|--------|
| W1 | `dto.ReconScopeRequest` (recon_dto.go) là dead code — không được import ở đâu | Low | Dọn dẹp sau |
| W2 | TS params `start_time`/`end_time` dùng snake_case, khác với `masterSchema` camelCase | Low | Code cũ, ngoài scope |
| W3 | `TriggerCheck` gọi `BodyParser` 2 lần (scope + req) — thừa parse | Low | Refactor sau |

---

## VI. PHẢN TỈNH — LESSONS VIOLATED IN THIS ROUND

| Lesson | Vi phạm? |
|--------|----------|
| Báo cáo láo "Done" khi chỉ thêm struct field | ❌ Không — đã trace E2E 7 hops |
| Fix bug lookup sai bằng đổi kiến trúc | ❌ Không — dùng explicit params |
| Bỏ quên bộ ba định danh Metadata | ❌ Không — cặp (schema, table) đầy đủ |

**Kết luận round 2: 0 vi phạm lesson.**
