# 05_progress — Cleanup `_gpay_source_id` + `_gpay_deleted`

> APPEND-ONLY. KHÔNG xóa/sửa entries cũ (§11 GEMINI).

---

## Entry 1 — 2026-05-28 — Workspace bootstrap + lesson read
- Action: Đọc `agent/memory/global/lessons.md` (focus: 2026-05-20 anti-pattern over-correct, 2026-05-20 verify ở destination, 2026-05-26 DoD at destination). Đọc `project_context.md`, `active_plans.md`, `tech_stack.md`.
- Action: Khởi tạo workspace `cleanup-gpay-cols-2026-05-28` per §7 Full Doc Set.
- Output: `00_context.md` + `01_requirements.md` + `02_plan.md`.
- Note: User trigger phrase "rác kỹ thuật" — cần audit chi tiết trước khi cắt; KHÔNG over-correct.

## Entry 2 — 2026-05-28 — Inventory 104 references
- Action: grep `_gpay_source_id\|_gpay_deleted` toàn bộ codebase.
- Result: 104 hits, phân loại 4 path:
  - Path A FE Shadow (cdc-cms-service): 3 file (shadow_automator.go drift-free, mapping_preview_handler.go DRIFT, e2e test).
  - Path B FE Shadow (centralized-data-service handler): 2 file (command_handler.go yesterday's Bug #2, event_handler.go tombstone INSERT).
  - Path C Master + Sinkworker V2: 8 file (sinkworker/*, service/transmuter.go, service/master_ddl_generator.go, service/schema_adapter.go).
  - Path D Test + UI: 3 file (test fixtures + MasterRegistry.tsx placeholder).
- Output: `03_implementation_audit.md`.

## Entry 3 — 2026-05-28 — Drift detection
- Action: Verify chain `shadow_automator.go` (CREATE) vs `mapping_preview_handler.go` (SELECT).
- Finding: Path A drift hiện hữu — preview handler đọc `_gpay_source_id`/`_gpay_id` từ shadow nhưng `shadow_automator.go` chỉ tạo `id` + `source_id`. Bug #2 fix yesterday che giấu drift này vì Path B (handler create) thêm `_gpay_source_id` vào shadow → preview "work" trên Path B nhưng broken trên Path A.
- Implication: Cleanup option phải address drift hoặc note GAP.

## Entry 4 — 2026-05-28 — Decision 3 OPTIONS + recommend A
- Action: Draft 3 cleanup options trong `04_decisions.md` + `09_tasks_solution_cleanup.md`.
  - **Option A (Conservative)**: Rollback Path B (Bug #2 portion). Minimal.
  - **Option B (Mid-Scope)**: Unify FE shadow — Path A + Path B đều dùng `source_id`/`_deleted` + fix preview handler.
  - **Option C (Full)**: Refactor master + sinkworker + transmuter → REJECTED phase này.
- Recommend: **Option A** dựa trên lesson "anti over-correct".
- Status: Brain phase DONE. Chờ user verb pick option.

## Entry 5 — 2026-05-28 — Doc set hoàn tất
- Action: Tạo đủ `00..10` + `report_audit_*.md`.
- Files: 00_context, 01_requirements, 02_plan, 03_implementation_audit, 04_decisions, 05_progress (file này), 06_validation, 07_status, 08_tasks_audit, 09_tasks_solution_cleanup, 10_gap_analysis, report_audit_cleanup_gpay_cols_2026-05-28.md.
- Output: 12 file vật lý ở `agent/memory/workspaces/cleanup-gpay-cols-2026-05-28/`.

---

## Entry 7 — 2026-05-28 — User verb "làm đi" → Muscle apply RENAME
- Action: Apply 16 file code rename + tạo migration SQL.
- Order: read each file → Edit → build → vet → test → grep zero-residue → smoke verify.
- Status: STARTED.

## Entry 6 — 2026-05-28 — User re-direct: RENAME refactor, KHÔNG remove
- User clarify: "_gpay_source_id đã có source_id, _gpay_deleted đã có _deleted ... chuyển sang những cái này thôi". Tức là RENAME `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted` ở mọi nơi. KHÔNG bỏ logic semantic.
- Self-critique: Lesson 2026-05-20 "anti over-correct" tôi áp dụng sai chiều — over-defer thay vì over-correct. User intent đơn giản: cùng concept = 1 tên = 1 cột. V2 master partial UNIQUE INDEX không bị bỏ, chỉ đổi cột target.
- Lesson mới (sẽ promote): "Cleanup ≠ Remove. Khi user nói 2 field 'rác trùng nhau' thì semantic là RENAME/MERGE chứ không phải DELETE. Verify intent trước khi pick option scope."
- Action: Reframe `04_decisions.md` D-8 + REWRITE `09_tasks_solution_cleanup.md` thành single unified RENAME plan + DB migration SQL.

## Pending entries (sẽ APPEND sau khi user verb)
- Entry 7+: Muscle apply RENAME — file changes + LOC delta + DB migration script.
- Entry 8+: Build verify 3 service.
- Entry 9+: Test verify (handler + service + sinkworker package).
- Entry 10+: Destination verify (`\d shadow.<t>` + grep `_gpay_*` = 0).
- Entry 11+: Security gate `/security-agent` (optional).

---

## Entry 8 — 2026-05-28 — Muscle apply DONE (16 file source + 1 ops + 1 migration)
- Action: Apply rename ở 10 file `centralized-data-service` (sinkworker + service + handler) + 2 file `cdc-cms-service` (mapping preview + integration test) + 1 file FE (`MasterRegistry.tsx`) + 3 file test fixtures Go + 1 ops script (`smoke_failover.sh`) + tạo migration SQL (`migration_rename_gpay_cols.sql`).
- Tools used: Read → Edit (replace_all cho test fixtures), Bash (grep + build + test).
- File count NET: 18 file vật lý touched. LOC delta NET: ~0 (mechanical rename) + 134 LOC migration mới.
- Method: 1 column = 1 concept, ALTER RENAME COLUMN idempotent (skip nếu đích đã tồn tại → DROP duplicate cho Path B Bug #2 hôm qua).

## Entry 9 — 2026-05-28 — Build verify PASS
- centralized-data-service: `go build ./...` silent success.
- cdc-cms-service: `go build ./...` silent success.
- cdc-cms-web: `npm run build` → `built in 477ms`.

## Entry 10 — 2026-05-28 — Test verify PASS
- `centralized-data-service/internal/service` `ok 0.947s`.
- `centralized-data-service/internal/handler` `ok 0.481s`.
- `centralized-data-service/test/internal/handler` `ok 3.502s`.
- `centralized-data-service/test/internal/service` `ok 3.326s`.
- `centralized-data-service/test/internal/sinkworker` `ok 0.219s`.
- `centralized-data-service/test/internal/activity` `ok 0.892s`.
- `centralized-data-service/test/internal/admin` `ok 2.153s`.
- `cdc-cms-service/test/internal/api` `ok 0.646s`.
- `cdc-cms-service/test/internal/app/commands` `ok 2.019s`.
- `cdc-cms-service/test/internal/app/queries` `ok 3.847s`.
- Zero test fail. Integration test postgres testcontainers KHÔNG chạy ở smoke; setup CREATE TABLE đã update PK `source_id`.

## Entry 11 — 2026-05-28 — Zero-residue grep PASS
- Command: `grep -rn "_gpay_source_id\|_gpay_deleted\|GpayDeleted" centralized-data-service/ cdc-cms-service/ cdc-cms-web/src/`.
- Result: 0 hits (sau khi fix `smoke_failover.sh` line 54 DUP-check SQL).
- Note: Còn 1 file `ops` ban đầu lọt qua audit vì path không match grep root khi initial inventory — fix kèm.

## Entry 12 — 2026-05-28 — Report + status update
- Action: Tạo `report_fix_cleanup_gpay_cols_2026-05-28.md` (file thứ 14 workspace).
- Action: Update `07_status.md` → FIX APPLIED.
- Action: Mark task G-5 (#30) completed.
- Lesson promotion candidate: "Cleanup ≠ Remove. Khi user nói 'rác trùng nhau' thì semantic là RENAME/MERGE, KHÔNG phải DELETE." — sẽ APPEND `agent/memory/global/lessons.md` ở turn sau (Brain duty).
- Remaining: `/security-agent` gate (optional, §8 GEMINI) + production deploy coordination (ops).

## Entry 13 — 2026-05-28 — Lesson promoted vào global memory
- Action: APPEND `L-2026-05-28-cleanup-is-not-remove` vào `agent/memory/global/lessons.md` per §7 + §13.
- Format: Global Pattern dùng biến A/B/X/S → kiểm tra "áp dụng được cho 3 dự án khác nhau không" PASS (DB column dedup, API field rename, config key consolidation).
- Liên kết với 2 lesson cũ: `L-2026-05-20-anti-over-correct` (cùng họ, ngược chiều) + `L-2026-05-20-verify-at-destination` (zero-residue verify).

## Entry 14 — 2026-05-28 — PROD ERROR: SQLSTATE 42701 "source_id specified more than once" (rename-blind tạo duplicate)
- Trigger: User báo log production sau khi tao báo "done" — `shadow_goopay_local_ws_wallet_service.events` upsert fail liên tục với `column "source_id" specified more than once (SQLSTATE 42701)`. User feedback (lần 3): "rename thì mày cũng phải xem logic của flow đó đã có chưa, có thì ko cần rename mà BỎ".
- §5 mid-session correction: Dừng lại, APPEND lesson `L-2026-05-28-rename-blind-creates-duplicate` vào `agent/memory/global/lessons.md` TRƯỚC khi sửa code.
- Root cause: `internal/service/schema_adapter.go` 2 helper `getMetadataInsertCols` + `getMetadataInsertPlaceholdersAndValues`. Pre-rename check `schema.Columns["_gpay_source_id"]` (V2 master Path C — cột riêng biệt, không trùng PK). Tao rename blanket thành `schema.Columns["source_id"]`. Nhưng Path A shadow (`shadow_automator.go` line 78-90) đã có `source_id VARCHAR(200) UNIQUE` riêng, và `batch_buffer.go:251-256` remap `effectivePK = "source_id"`. Sau remap, pkField + metadata branch cùng emit `"source_id"` → INSERT INTO X (`source_id`, ..., `source_id`, ...) → SQL runtime 42701.
- Fix file 1: `internal/service/schema_adapter.go` — thêm `pkField` param cho 2 helper, skip `source_id` nếu `pkField == "source_id"`. Update 3 caller site (BuildUpsertSQLInSchema + BuildBatchUpsertSQLsInSchema chunk size estimate + per-row).
- Fix file 2 (defensive): `internal/handler/event_handler.go` tombstone INSERT — conditional 2 nhánh: `pgPKField == "source_id"` → bỏ extra `source_id` column từ INSERT; ngược lại giữ pattern cũ.
- Fix file 3 (defensive): `internal/handler/command_handler.go` Path B inline CREATE TABLE — conditional 2 nhánh: `pkField == "source_id"` → promote PK đến cột `source_id` luôn (bỏ `source_id TEXT UNIQUE` riêng); ngược lại giữ pattern cũ.
- Regression test: `test/internal/service/schema_adapter_test.go` — thêm `TestBuildUpsertSQL_PKIsSourceID_NoDuplicateColumn` đếm `"source_id"` xuất hiện trong SQL, assert ≤3 (col list + ON CONFLICT + value-ref tối đa, KHÔNG có metadata 4th).
- Build + test: `go build ./...` PASS x2 service. `go test ./test/...` PASS toàn bộ. Regression test PASS.

## Entry 15 — 2026-05-28 — Lesson `L-2026-05-28-rename-blind-creates-duplicate` promoted
- Action: APPEND vào `agent/memory/global/lessons.md` per §5+§13.
- Core: "Cleanup = mixture của RENAME (target chưa có) + REMOVE (target đã có), KHÔNG phải pure RENAME. Per-site case analysis bắt buộc trước Edit. Blind `replace_all` cross-path → duplicate SQL column ở runtime."
- Liên kết: L-2026-05-28-cleanup-is-not-remove (sister lesson — phải đọc cặp).
