# Report — Audit Shadow Create Bugs (2026-05-27)

**Phase**: AUDIT-ONLY  
**Date**: 2026-05-27 ICT  
**Operator**: Muscle (Claude-Opus-4.7)  
**Scope**: 2 bug user phát hiện khi tạo `sd_export_jobs_1` qua FE `http://localhost:5173/shadow`.

---

## TL;DR

| Bug | Root cause | File:Line | Severity |
|---|---|---|---|
| **B1 — Cross-entity field auto-leak** | `GetActiveRulesBySourceTable` JOIN bằng `source_object_name` (trùng nhau hợp lệ) thay vì `source_object_id` | `centralized-data-service/internal/repository/mapping_rule_v2_repo.go:54-61` + caller `command_handler.go:649` | **HIGH** — vi phạm isolation, leak business cols giữa các shadow |
| **B2 — Shadow CREATE thiếu `_source_ts` (+ `_gpay_source_id`, `_gpay_deleted`)** | DDL builder ở FE-trigger path drift so với spec spec runtime (sinkworker) | `centralized-data-service/internal/handler/command_handler.go:586-602` (CREATE) + `:163-172` (ensureCDCColumnsInSchema) | **CRITICAL** — `_source_ts` là OCC anchor; thiếu → OCC guard crash, V2 master upsert hỏng |

**Fix size**: ~24 LOC trong 1 file (`command_handler.go`) — minimal impact.  
**Files thay đổi trong audit phase**: **0 source files**. Chỉ tạo workspace docs.

---

## 1. Reproduction context

User thao tác: vào `http://localhost:5173/shadow` → tạo shadow mới với:
- `target_table = sd_export_jobs_1`
- `source_table = export_jobs` (cùng tên với entity `dbsource → export_jobs` đã đăng ký từ trước)

**Observed**:
- Shadow mới `sd_export_jobs_1` ngay sau khi tạo đã có nhiều business column (móc từ entity cũ).
- Cột `_source_ts` không xuất hiện trong DDL của shadow mới.

---

## 2. Trace — luồng tạo shadow

| Layer | File:Line | Action |
|---|---|---|
| FE | `cdc-cms-web/src/pages/TableRegistry.tsx` (1127 LOC) | Form antd → `POST /api/v1/source-objects/register` với raw values; **không leak field** |
| API | `cdc-cms-service/internal/api/registry_handler_register.go:17` | Parse body → exec `RegisterRegistryCommand` → `ResolveShadowSchema` → publish NATS `cdc.cmd.create-default-columns` với payload `{registry_id, source_object_id, shadow_schema, target_table, source_table, primary_key_field, primary_key_type}` |
| Command | `cdc-cms-service/internal/app/commands/register_registry.go:99-128` | TX: INSERT V1 + V2 sync → `EnsureShadowTable` (DDL initial) → reload event |
| Worker | `centralized-data-service/internal/handler/command_handler.go HandleCreateDefaultColumns` | Consume `cdc.cmd.create-default-columns`: CREATE TABLE (line 586) → `ensureCDCColumnsInSchema` (line 610) → auto-discovery `scanFieldsDebezium` (line 630) → `GetActiveRulesBySourceTable` + loop ALTER ADD COLUMN (line 649-720) |

---

## 3. Bug 1 — Cross-entity field auto-leak (root cause)

**Defect site**: `centralized-data-service/internal/repository/mapping_rule_v2_repo.go:54-61`
```go
func (r *MappingRuleV2Repo) GetActiveRulesBySourceTable(ctx context.Context, sourceTable string) ([]model.MappingRuleV2, error) {
    var items []model.MappingRuleV2
    err := r.db.WithContext(ctx).
        Joins("JOIN cdc_system.source_object_registry so ON cdc_system.mapping_rule_v2.source_object_id = so.id").
        Where("so.source_object_name = ? AND cdc_system.mapping_rule_v2.is_active = ? AND cdc_system.mapping_rule_v2.status = ?", sourceTable, true, "approved").
        Find(&items).Error
    return items, err
}
```

**Caller**: `command_handler.go:649`
```go
rules, err := h.mappingV2Repo.GetActiveRulesBySourceTable(context.Background(), payload.SourceTable)
```

**Hệ quả**: 2 registry rows hợp lệ cùng `source_object_name = "export_jobs"` (khác `target_table`) → query trả về union mapping rules → loop ALTER ADD COLUMN ở line 690 thêm field của registry cũ vào shadow mới.

**Fix proposal** (`09_tasks_solution_audit.md` SOL-1): Swap caller sang `ListActiveBySourceObject(ctx, effectiveID)` — API có sẵn line 37-44 cùng repo, filter bằng `source_object_id`. `effectiveID` đã resolve ở line 620-623.

---

## 4. Bug 2 — Shadow CREATE thiếu `_source_ts` (root cause)

**Defect site A**: `centralized-data-service/internal/handler/command_handler.go:586-602` (CREATE TABLE)
```go
createSQL := fmt.Sprintf(
    `CREATE TABLE IF NOT EXISTS %s.%s (
        %s %s PRIMARY KEY,
        _raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
        _source VARCHAR(20) NOT NULL DEFAULT 'debezium',
        _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
        _version BIGINT NOT NULL DEFAULT 1,
        _hash VARCHAR(64),
        _deleted BOOLEAN DEFAULT FALSE,
        _created_at TIMESTAMP DEFAULT NOW(),
        _updated_at TIMESTAMP DEFAULT NOW()
    )`, ...)
```

**Defect site B**: `command_handler.go:163-172` (`ensureCDCColumnsInSchema.cdcColumns`)
```go
cdcColumns := []struct{ name, def string }{
    {"_raw_data", "JSONB"},
    {"_source", "VARCHAR(20) DEFAULT 'debezium'"},
    {"_synced_at", "TIMESTAMP DEFAULT NOW()"},
    {"_version", "BIGINT DEFAULT 1"},
    {"_hash", "VARCHAR(64)"},
    {"_deleted", "BOOLEAN DEFAULT FALSE"},
    {"_created_at", "TIMESTAMP DEFAULT NOW()"},
    {"_updated_at", "TIMESTAMP DEFAULT NOW()"},
}
```

**Cột thiếu**:
| Cột | Vai trò | Reference |
|---|---|---|
| `_source_ts BIGINT` | **OCC older-wins anchor** — sinkworker upsert guard `EXCLUDED._source_ts > shadow._source_ts` | `sinkworker/upsert.go:69-122` |
| `_gpay_source_id TEXT UNIQUE` | V2 master anchor, ON CONFLICT key cho upsert master | `service/master_ddl_generator.go:92` |
| `_gpay_deleted BOOLEAN` | Tombstone soft-delete | `project_context.md §Shadow Layer required cols` |

**Cross-check (path khác đúng spec)**:
- `centralized-data-service/internal/sinkworker/schema_manager.go:231` — `"_source_ts" BIGINT` ✓
- `centralized-data-service/internal/sinkworker/upsert.go:69-122` — OCC reference ✓
- `centralized-data-service/internal/service/master_ddl_generator.go:92` — master DDL ✓
- `centralized-data-service/internal/service/transmuter.go:89` — `SourceTs int64 gorm:"column:_source_ts"` ✓
- `centralized-data-service/internal/recon/recon_handler.go:263` — recon hash ✓

→ **Drift duy nhất ở FE-trigger path** (`HandleCreateDefaultColumns`). Mọi path runtime đều phụ thuộc `_source_ts` → silent crash khi ingest vào shadow này.

**Fix proposal** (`09_tasks_solution_audit.md` SOL-2): Thêm 3 cột vào cả 2 vị trí + thêm index `idx_<t>_source_ts` + UNIQUE constraint `uq_<t>_gpay_source_id`. Match contract của `sinkworker/schema_manager.go`.

---

## 5. Files thay đổi trong AUDIT phase

| Loại | File | LOC |
|---|---|---|
| Source code | (none) | **0** |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/00_context.md` | 36 |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/01_requirements.md` | 28 |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/02_plan.md` | 70+ |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/03_implementation_audit.md` | 35+ |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/04_decisions.md` | 30+ |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/05_progress.md` | 3 entries (APPEND) |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/06_validation.md` | 50+ |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/07_status.md` | 25+ |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/08_tasks_audit.md` | 40+ |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/09_tasks_solution_audit.md` | 150+ |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/10_gap_analysis.md` | 45+ |
| Workspace doc | `agent/memory/workspaces/audit-shadow-create-bugs-2026-05-27/report_audit_shadow_create_bugs_2026-05-27.md` | (this file) |

**Tổng source code change**: **0 dòng**.  
**Tổng workspace doc**: 12 file, ~500+ dòng tài liệu.

---

## 6. Estimated fix cost (Fix Phase — pending approval)

| Patch | File | LOC | Risk |
|---|---|---|---|
| SOL-1 (Bug 1) | `command_handler.go:649` + log fields | ~4 LOC | Low — API có sẵn |
| SOL-2.A (Bug 2 CREATE) | `command_handler.go:586-602` add 3 cols | ~3 LOC | Low — DDL append-only |
| SOL-2.B (Bug 2 cdcColumns) | `command_handler.go:163-172` add 3 entries | ~3 LOC | Low |
| SOL-2.C (Bug 2 idx/UNIQUE) | `command_handler.go` cuối `ensureCDCColumnsInSchema` | ~14 LOC | Medium — DO block PL/pgSQL, cần test idempotent |
| **Total** | **1 file** | **~24 LOC** | — |

---

## 7. Build verify (baseline trước fix)

| Service | Command | Result |
|---|---|---|
| `centralized-data-service` | `go build ./...` | **PASS** |
| `cdc-cms-service` | `go build ./...` | **PASS** |
| `cdc-cms-web` | `npx vite build` | **PASS** (797ms, 9 chunks) |

---

## 8. Rule compliance (§14 Pre-flight check)

- [x] §1 Brain/Muscle: Muscle thực hiện audit, KHÔNG chạm code phase này.
- [x] §3 Plan & Verify: Plan rõ ràng có code demo, baseline build PASS.
- [x] §6 Simplicity First / Demand Elegance: SOL-1 dùng API có sẵn, SOL-2 minimal patch.
- [x] §7 Knowledge Retention + Full Doc Set: 12 file doc trong workspace.
- [x] §11 Memory Protection: `05_progress.md` chỉ APPEND, không overwrite.
- [x] §12 Brain Code Prohibition: KHÔNG sửa source code; documented vào `09_tasks_solution_audit.md` chờ approve.
- [x] §14 Pre-flight: Files đã tạo vật lý (verify bằng `ls workspace/`).

User constraints:
- [x] "Đọc lesson + GEMINI.md trước" — done từ Entry 1.
- [x] "Chỉ làm đúng những gì đc yêu cầu" — audit-only, không sửa code.
- [x] "Không cheat DB / không đổi config" — fix tại core flow, không ALTER thủ công, không touch env.
- [x] "Plan rõ ràng, code demo tới từng chi tiết" — `09_tasks_solution_audit.md` có patch trước/sau từng dòng.
- [x] "Report dựa trên kết quả tính toán thực tế, ko báo láo" — mọi file:line đều có grep evidence.
- [x] "Kiểm tra service work mới báo done" — build PASS cả 3 service.
- [x] "Note lại file thay đổi + LOC" — bảng tại §5 (source = 0, workspace = 12 doc).

---

## 9. Next step (chờ user verb)

User cần OK 1 trong các option:
- **A. "OK fix"** → Muscle vào Fix Phase: apply SOL-1 + SOL-2.A/B/C, build verify, /security-agent gate, write follow-up report.
- **B. "Audit sâu hơn"** → tiếp tục audit GAP-1..6 trước khi fix.
- **C. "Hold"** → đóng audit, để feature reque sau.
