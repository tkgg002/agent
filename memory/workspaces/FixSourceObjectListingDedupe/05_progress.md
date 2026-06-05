# 05_progress — FixSourceObjectListingDedupe (IMMUTABLE APPEND-ONLY)

## 2026-05-21 — Session Start (Muscle)

- T1 ✅ Workspace dir created.
- T1 ✅ 00_context, 01_requirements, 02_plan, 09_tasks_solution, 08_tasks initialized.
- T2 ✅ Located: `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go:40-65` (listBaseFromWhere const).
- T3 ✅ Root cause confirmed: JOIN missing `source_connection_id` predicate post-migration-054/055/056.
- Decision: dùng LATERAL + connection_id scope thay vì GROUP BY array_agg (user picked Option A; LATERAL subsumes & more elegant — bảo toàn wire shape).

## 2026-05-21 — Implementation + Verify (Muscle)

- T4 ✅ Edited `source_object_read_repo_gorm.go` listBaseFromWhere constant:
  - Replaced `LEFT JOIN cdc_table_registry tr ON ...` with `LEFT JOIN LATERAL (SELECT ... LIMIT 1) tr ON TRUE`.
  - Added predicate `tr.source_connection_id = so.source_connection_id OR tr.source_connection_id IS NULL`.
  - ORDER `(tr.source_connection_id IS NULL) ASC, tr.id ASC` — exact match preferred, deterministic tiebreaker.
- T5 ✅ Build + vet + test PASS. Binary `/tmp/cdc-cms-service-fixdedupe` (58 MB).
- T6 ✅ SQL verified directly against gpay-postgres-cdc/cdc_dw: 4 rows (was 6); cross-connection bleed eliminated (so.id=1↔tr.id=1 conn2; so.id=36↔tr.id=4 conn42).
- T7 ✅ Security self-review: no SQL injection (parameterized), no auth bypass, NULL fallback risk accepted (admin-only registration path).
- T8 ✅ This APPEND.
- T9 ⏳ APPEND lesson to global lessons.md (next).

## Files Touched

- `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go` (1 const refactored, no API surface change)
- `agent/memory/workspaces/FixSourceObjectListingDedupe/*` (new workspace, 7 docs)

