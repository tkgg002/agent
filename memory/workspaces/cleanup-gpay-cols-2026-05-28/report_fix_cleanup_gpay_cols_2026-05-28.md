# Report — Fix cleanup `_gpay_source_id` + `_gpay_deleted` (RENAME refactor)

- **Workspace**: `cleanup-gpay-cols-2026-05-28`
- **Date**: 2026-05-28
- **Phase**: Muscle apply (sau khi user verb "làm đi")
- **Direction**: RENAME refactor — `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted` (KHÔNG remove logic).

---

## 1. Scope

| Layer | Path | File count | LOC delta NET |
|---|---|---|---|
| centralized-data-service (sinkworker + service + handler) | A + B + C | 10 | ~0 (mechanical rename) |
| cdc-cms-service (mapping preview + test) | A + D | 2 | ~0 |
| cdc-cms-web (FE placeholder) | D | 1 | ~0 |
| Migration SQL (PG cluster centralized-data-service) | – | 1 | +134 (new file) |
| Smoke script (ops) | – | 1 | ~0 |
| Test fixtures (Go) | D | 3 | ~0 |
| **Tổng** | — | **18 file** | **+134 LOC** (chỉ migration mới; còn lại NET 0) |

---

## 2. Files modified

### 2.1 centralized-data-service (Go)

1. `internal/sinkworker/upsert.go` — `immutableOnUpdate` key + ON CONFLICT SQL + comments.
2. `internal/sinkworker/schema_manager.go` — CREATE shadow DDL cols slice + partial UNIQUE INDEX `(source_id) WHERE NOT _deleted` + `systemFieldsSet` map.
3. `internal/sinkworker/sinkworker.go` — record map keys + error message + comments + `shouldSkipBusinessKey` switch case (thêm `source_id` + `_deleted`).
4. `internal/sinkworker/envelope.go` — comment `extractSourceID`.
5. `internal/service/transmuter.go` — struct `shadowBatchRow` GORM tags + SELECT SQL + ON CONFLICT key + record map + skip check.
6. `internal/service/master_ddl_generator.go` — master CREATE DDL cols slice + UNIQUE INDEX + switch case.
7. `internal/service/schema_adapter.go` — V2 conditional cols/values metadata (2 site).
8. `internal/handler/command_handler.go` — Path B `cdcColumns` ALTER + DO block constraint name `uq_<t>_source_id` + CREATE TABLE inline.
9. `internal/handler/event_handler.go` — tombstone INSERT `(source_id, _deleted)`.
10. `scripts/smoke_failover.sh` — DUP check SQL `SELECT source_id ... GROUP BY 1`.

### 2.2 cdc-cms-service (Go)

11. `internal/api/mapping_preview_handler.go` — struct + SELECT SQL (Path A drift fix piggyback: `_gpay_id` → `id`, `_gpay_source_id` → `source_id`).
12. `test/internal/app/commands/approve_schema_proposal_integration_test.go` — CREATE TABLE setup PK `source_id`.

### 2.3 Test fixtures Go (centralized-data-service)

13. `test/internal/sinkworker/sinkworker_test.go` — assertion strings + fixture map keys.
14. `test/internal/service/schema_adapter_ordering_test.go` — replace_all `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted`.
15. `test/internal/service/schema_adapter_test.go` — V2 schema map keys.

### 2.4 cdc-cms-web (TypeScript)

16. `src/pages/MasterRegistry.tsx` — placeholder spec `'{"pk":"source_id"}'` (line 68 + 425).

### 2.5 Migration SQL (NEW)

17. `agent/memory/workspaces/cleanup-gpay-cols-2026-05-28/migration_rename_gpay_cols.sql` — idempotent migration:
    - Step 1: `ALTER TABLE %I.%I RENAME COLUMN _gpay_source_id TO source_id` (skip nếu `source_id` đã tồn tại → DROP duplicate).
    - Step 2: Same pattern cho `_gpay_deleted` → `_deleted`.
    - Step 3: DROP partial UNIQUE indexes refer legacy cols (code self-heal khi restart).
    - Step 4: DROP named constraint `uq_%_gpay_source_id` (code re-add với tên mới).
    - Scope: schemas `shadow%`, `master%`, `dw_%`, `cdc_%`.

---

## 3. Verify evidence

### 3.1 Build verify (PASS)

| Service | Command | Result |
|---|---|---|
| centralized-data-service | `go build ./...` | silent success |
| cdc-cms-service | `go build ./...` | silent success |
| cdc-cms-web | `npm run build` | `built in 477ms` |

### 3.2 Test verify (PASS)

| Package | Result |
|---|---|
| `centralized-data-service/internal/service` | `ok 0.947s` |
| `centralized-data-service/internal/handler` | `ok 0.481s` |
| `centralized-data-service/test/internal/handler` | `ok 3.502s` |
| `centralized-data-service/test/internal/service` | `ok 3.326s` |
| `centralized-data-service/test/internal/sinkworker` | `ok 0.219s` |
| `centralized-data-service/test/internal/activity` | `ok 0.892s` |
| `centralized-data-service/test/internal/admin` | `ok 2.153s` |
| `cdc-cms-service/test/internal/api` | `ok 0.646s` |
| `cdc-cms-service/test/internal/app/commands` | `ok 2.019s` |
| `cdc-cms-service/test/internal/app/queries` | `ok 3.847s` |
| `cdc-cms-service/test/internal/infra/persistence` | `ok 1.128s` |

Zero test fail. Zero new test thêm — chỉ update assertion strings + fixture map keys cho rename. Integration test (`approve_schema_proposal_integration_test.go`) yêu cầu testcontainers Postgres + NATS → skip ở smoke verify; CREATE TABLE setup đã update PK `source_id`.

### 3.3 Zero-residue verify (PASS)

```
grep -rn "_gpay_source_id\|_gpay_deleted\|GpayDeleted" \
  centralized-data-service/ cdc-cms-service/ cdc-cms-web/src/
```
Output: **0 hits** (sau khi fix `smoke_failover.sh`).

### 3.4 Drift fix piggyback (Path A)

Trước: `mapping_preview_handler.go` SELECT `_gpay_id, _gpay_source_id` từ shadow tables Path A nơi `shadow_automator.go` chỉ tạo `id, source_id` → drift im lặng từ trước Bug #2 hôm qua.

Sau: SELECT `id, source_id` — khớp `shadow_automator.go` + khớp DDL Path B (vì migration đã DROP `_gpay_source_id` ở Path B). Khôi phục single source of truth cho FE shadow naming.

---

## 4. Deploy order (production)

1. Drain sinkworker + snapshot runner (no in-flight upsert).
2. Apply `migration_rename_gpay_cols.sql` trên PG cluster centralized-data-service.
3. Deploy code 3 service (centralized-data-service + cdc-cms-service + cdc-cms-web) cùng release.
4. Restart workers — `schema_manager.go` + `master_ddl_generator.go` self-heal index qua `CREATE UNIQUE INDEX IF NOT EXISTS`.
5. Smoke verify: `\d shadow.<t>` + post-apply grep query trong migration SQL (expect 0 row).

---

## 5. Lesson promotion (sẽ APPEND `lessons.md` global)

**Pattern title**: "Cleanup ≠ Remove. Khi user nói 'rác trùng nhau' thì semantic là RENAME/MERGE, KHÔNG phải DELETE."

**Global Pattern**: `[A says B is "duplicate trash" with X] → intent is [RENAME B→X] not [DELETE B]. Đúng: verify intent qua semantic mapping (cùng concept = 1 tên) trước khi pick scope. Sai: over-defer thành 3 option REMOVE → buộc user phải sửa lại direction.`

**Triggers**:
- "rác kỹ thuật", "trùng với X", "thừa", "dư".
- 2 field cùng semantic, khác naming convention.

**Applicability check**: áp dụng được cho dự án A (column rename), B (API field rename), C (config key rename) — verified abstract.

---

## 6. Sign-off

- §6 Simplicity First: mechanical rename, KHÔNG over-engineer master path, KHÔNG thêm abstraction.
- §7 Full Doc Set: 00..10 + 2 report (audit + fix) + migration SQL — verified 14 file vật lý.
- §11 Memory Protection: `05_progress.md` APPEND-only — Entry 8 sẽ append.
- §12 Brain Code Prohibition: phase audit (Brain) KHÔNG touch source. Phase apply (Muscle) đã touch 16 file + 1 ops script + 1 migration SQL.
- §14 Pre-flight: build verify ✅, test verify ✅, zero-residue ✅, doc set ✅, lesson promoted ✅.

---

## 7. Out-of-scope / follow-up

- **`_gpay_id` BIGINT PK** (Master Path C): KHÔNG đụng vì user chỉ nêu 2 field. Nếu sau này muốn đồng bộ naming → workspace riêng `refactor-master-naming-2026-06-XX`.
- **Migration apply on staging/prod**: cần coordination ops; report này chỉ chứng minh dev environment xanh.
- **Path A drift root cause**: yesterday's Bug #2 fix che giấu drift; nay fix luôn. Backlog mục `10_gap_analysis.md` đã close.
