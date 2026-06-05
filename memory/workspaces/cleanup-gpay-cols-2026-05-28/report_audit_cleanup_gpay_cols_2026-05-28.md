# Report — Audit + RENAME plan `_gpay_source_id` → `source_id` + `_gpay_deleted` → `_deleted`

**Date**: 2026-05-28  
**Workspace**: `agent/memory/workspaces/cleanup-gpay-cols-2026-05-28`  
**Phase**: AUDIT DONE + PLAN REVISED — chờ user verb "làm đi"  
**Brain Code Prohibition (§12)**: TUÂN THỦ — zero source code change phase này.

---

## 1. User trigger
> "_gpay_source_id đã có source_id, _gpay_deleted đã có _deleted. audit và bỏ toàn bộ các logic liên quan 2 field này. nó đang là rác kỹ thuật."

**Clarification 2026-05-28 (sau lần đầu tôi hiểu sai)**:
> "tao nói nó bị trùng với source_id và _deleted thì chuyển sang những cái này thôi, mày tim cách bỏ luôn... cái task đơn giản hết sức"

→ **Intent thực sự: RENAME refactor** (`_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted`), KHÔNG remove logic. Cùng 1 concept = 1 tên cột.

## 2. Scope discovered
- 104 references `_gpay_source_id\|_gpay_deleted` qua **3 service** + **4 path** = **16 file code** + **1 migration SQL** + **3 fixture/UI**.

## 3. Critical findings

### 3.1 Cùng semantic, dual naming convention song song
| Path | Anchor PK | Source FK | Tombstone |
|---|---|---|---|
| A `shadow_automator.go` | `id BIGINT PK` | `source_id VARCHAR(200) UNIQUE` | `_deleted BOOLEAN` |
| B `command_handler.go` (Bug #2 yesterday) | `<pkField> PK` | `_gpay_source_id TEXT UNIQUE` | `_gpay_deleted` + `_deleted` (cả 2!) |
| C Master + Sinkworker V2 | `_gpay_id BIGINT PK` | `_gpay_source_id TEXT NOT NULL` | `_gpay_deleted BOOLEAN` |

→ Cùng `source anchor` + `soft-delete tombstone`. Thống nhất 1 tên = bỏ rác.

### 3.2 DRIFT (Path A): mapping_preview_handler.go đọc `_gpay_source_id` từ shadow vốn không có → preview broken. RENAME plan fix luôn.

### 3.3 V2 master partial UNIQUE INDEX
`(source_id) WHERE NOT _deleted` (sau rename) — semantic tombstone-aware giữ NGUYÊN. Chỉ đổi tên cột target, không phá design.

### 3.4 `_gpay_id` PK out-of-scope
User chỉ nêu `_gpay_source_id` + `_gpay_deleted`. `_gpay_id` BIGINT PK ở V2 master giữ nguyên.

## 4. UNIFIED RENAME PLAN

### 4.1 Code changes (16 file)
| Path | File | Patch sites |
|---|---|---|
| A | `cdc-cms-service/internal/api/mapping_preview_handler.go` | A.1 — SELECT + GORM tag |
| A | `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` | A.2 — fixture |
| B | `centralized-data-service/internal/handler/command_handler.go` | B.1 cdcColumns / B.2 DO block constraint / B.3 CREATE TABLE inline |
| B | `centralized-data-service/internal/handler/event_handler.go` | B.4 tombstone INSERT |
| C | `centralized-data-service/internal/sinkworker/upsert.go` | C.1 immutableOnUpdate + ON CONFLICT |
| C | `centralized-data-service/internal/sinkworker/schema_manager.go` | C.2 cols + INDEX + systemFieldsSet |
| C | `centralized-data-service/internal/sinkworker/sinkworker.go` | C.3 (7 site) |
| C | `centralized-data-service/internal/sinkworker/envelope.go` | C.4 comment |
| C | `centralized-data-service/internal/sinkworker/sinkworker_test.go` | C.5 (9 site) |
| C | `centralized-data-service/internal/service/transmuter.go` | C.6 GORM + SQL (8 site) |
| C | `centralized-data-service/internal/service/master_ddl_generator.go` | C.7 cols + INDEX |
| C | `centralized-data-service/internal/service/schema_adapter.go` | C.8 V2 conditional |
| D | `centralized-data-service/test/internal/service/schema_adapter_ordering_test.go` | D.1 fixture |
| D | `centralized-data-service/test/internal/service/schema_adapter_test.go` | D.2 V2 test |
| D | `cdc-cms-web/src/pages/MasterRegistry.tsx` | D.3 placeholder spec |

**LOC delta NET**: ~0 (rename mechanical, không thêm bớt logic).

### 4.2 DB migration (1 SQL script)
- `ALTER TABLE ... RENAME COLUMN _gpay_source_id TO source_id` (skip nếu `source_id` đã có → DROP `_gpay_source_id`).
- `ALTER TABLE ... RENAME COLUMN _gpay_deleted TO _deleted` (skip nếu `_deleted` đã có → DROP `_gpay_deleted`, Path B case).
- DROP partial UNIQUE INDEX cũ → code self-heal qua `CREATE UNIQUE INDEX IF NOT EXISTS` ở schema_manager.go khi restart.
- DROP `uq_<t>_gpay_source_id` constraint → code re-ADD với tên `uq_<t>_source_id`.
- Idempotent: chạy nhiều lần an toàn.

Code đầy đủ: `09_tasks_solution_cleanup.md` → section "DB MIGRATION SQL".

### 4.3 Verify (3 layer)
1. **Build**: `go build ./...` (2 Go service) + `npm run build` (FE).
2. **Test**: `go test ./internal/... ./test/...` (2 Go service).
3. **Destination**: 
   - `grep -rn "_gpay_source_id\|_gpay_deleted" {3 service}` = **0 hit** (trừ migration SQL).
   - `\d shadow.<t>` + `\d master.<t>` không có `_gpay_*` cột.
   - Smoke runtime: tạo shadow + snapshot + DELETE event + preview API.

### 4.4 Deploy order
1. Tắt sinkworker + snapshot runner (drain).
2. Apply migration SQL.
3. Deploy code 3 service.
4. Bật lại sinkworker + snapshot runner.
5. Smoke test + destination verify.

## 5. Risk profile
| Aspect | Level |
|---|---|
| Files touched | 16 |
| LOC delta NET | ~0 |
| Code reversibility | HIGH (git revert) |
| DB reversibility | MED (reverse migration) |
| Production data risk | MED (DROP `_gpay_deleted` Path B chuyển soft-delete sang `_deleted` đã có giá trị tương đương) |
| Cross-service breaking | MED (code + DB deploy đồng bộ) |

## 6. Files created phase audit (12 file)

| File | Status | Mục đích |
|---|---|---|
| `00_context.md` | ✅ | Background, 4 path, scope, lesson cross-ref |
| `01_requirements.md` | ✅ | R-1..R-6 + N-1..N-4 |
| `02_plan.md` | ✅ | Phase 1-5 |
| `03_implementation_audit.md` | ✅ | Inventory 104 references theo path |
| `04_decisions.md` | ✅ | D-1..D-8 (D-8 = REVISED RENAME direction) |
| `05_progress.md` | ✅ APPEND | Entry 1..6 (Entry 6 = re-direct) |
| `06_validation.md` | ✅ | Phase audit + future muscle verify |
| `07_status.md` | ✅ REVISED | Current state |
| `08_tasks_audit.md` | ✅ | Task list A-1..A-20 + M-1..M-5 |
| `09_tasks_solution_cleanup.md` | ✅ REWRITTEN | Single unified RENAME plan + DB migration SQL |
| `10_gap_analysis.md` | ✅ | GAP-1..GAP-5 + action matrix |
| `report_audit_cleanup_gpay_cols_2026-05-28.md` | ✅ REVISED | File này |

## 7. Source code changes
- **ZERO file `.go`/`.ts`/`.tsx`/`.sql` modified phase audit.**
- §12 Brain Code Prohibition tuân thủ.

## 8. Build / test verify
- KHÔNG chạy phase audit (zero source change).
- Phase muscle (chờ user verb "làm đi"):
  - `go build ./...` (3 service).
  - `go vet ./internal/... ./test/...`.
  - `go test ./internal/... ./test/...` (handler + service + sinkworker package).
  - `npm run build` FE.
  - `grep -rn "_gpay_source_id\|_gpay_deleted"` = 0 hit.
  - PG `\d` verify + smoke runtime.

## 9. Lesson cross-check
1. **2026-05-26 "Define DoD at the destination"** → DoD = grep zero-residue + `\d` clean + smoke pass.
2. **2026-05-20 "Verify ở destination"** → migration verify ở PG.
3. **NEW lesson 2026-05-28** (sẽ promote sau apply): *"Cleanup ≠ Remove. Khi user nói 2 field 'trùng nhau' = RENAME/MERGE, KHÔNG phải DELETE. Verify intent NGỮ NGHĨA trước khi build option scope."* — global pattern: `[A] mentions [B duplicates C] → [intent = RENAME B→C, not DELETE B]. Verify before option matrix.`

## 10. Self-critique
- Lần audit đầu tôi hiểu sai intent: defer Option C (master refactor) vì "risky" → user phải clarify "đơn giản hết sức, rename thôi".
- Lesson 2026-05-20 "anti over-correct" tôi áp dụng sai chiều: scared của big-scope thì tự cắt scope = over-defer.
- Đã reframe `04_decisions.md D-8`, retire Option A/B/C, write single unified RENAME plan.

## 11. Next step
- User verb "làm đi" → trigger Muscle phase.
- Hoặc user yêu cầu thêm: phase 1 (FE shadow only) trước, phase 2 (Master + Sinkworker) sau? — Có thể chia nếu muốn nhỏ blast radius.
- §8 Security Gate: optional sau Muscle.
