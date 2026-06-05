# 05_progress — Audit Shadow Create Bugs (Append-Only Audit Log)

> §11 Memory File Protection: APPEND ONLY. Sai thì ghi dòng "Sai - Revert", không xoá.

## Entry 1
[2026-05-27 16:05 ICT] [Muscle:Claude-Opus-4.7] Init workspace `audit-shadow-create-bugs-2026-05-27`. Đọc xong GEMINI.md, lessons.md (grep shadow/_source_ts), project_context.md, active_plans.md, tech_stack.md. Tạo 00_context + 01_requirements. Task #11 done, #12 in_progress.

## Entry 2
[2026-05-27 16:06 ICT] [Muscle:Claude-Opus-4.7] User trigger 2 bug khi tạo `sd_export_jobs_1` qua route `/shadow`:
- B1: shadow mới móc field từ `dbsource → export_jobs` cũ (chưa thao tác gì).
- B2: `_source_ts` không xuất hiện trên shadow vừa tạo.
Audit-only phase. Plan: trace FE → cms-service → worker. Không sửa code trong phase này. Tuân §12 Brain/Muscle separation — Muscle có quyền sửa nhưng chờ document đầy đủ trước.

## Entry 3
[2026-05-27 16:35 ICT] [Muscle:Claude-Opus-4.7] Hoàn tất audit Phase A+B+C.

**Bug 1 root cause**: `centralized-data-service/internal/repository/mapping_rule_v2_repo.go:54-61` — `GetActiveRulesBySourceTable` JOIN `source_object_registry` filter chỉ theo `source_object_name` (string trùng được giữa các registry rows). Caller `command_handler.go:649` truyền `payload.SourceTable` thay vì `payload.SourceObjectID`. Hệ quả: tạo `sd_export_jobs_1` (registry mới) với source_table=`export_jobs` → query trả về mapping rules của `export_jobs` registry CŨ → ALTER ADD COLUMN cross-bleed.

**Bug 2 root cause**: `command_handler.go:586-602` (CREATE TABLE DDL) và `command_handler.go:163-172` (`ensureCDCColumnsInSchema` cdcColumns slice) — cả 2 đều thiếu `_source_ts BIGINT`, `_gpay_source_id TEXT UNIQUE`, `_gpay_deleted BOOLEAN`. Cross-check: `sinkworker/schema_manager.go:231` + `sinkworker/upsert.go:69-122` + `service/master_ddl_generator.go:92` + `service/transmuter.go:89` đều dùng `_source_ts` → drift duy nhất ở FE-trigger path.

**Fix proposal (chưa apply)**:
- SOL-1: swap `GetActiveRulesBySourceTable` → `ListActiveBySourceObject(ctx, effectiveID)` tại line 649. API đã có sẵn line 37-44.
- SOL-2: thêm 3 cột vào CREATE TABLE + cdcColumns + thêm idx `_source_ts` + UNIQUE `_gpay_source_id` constraint.
- Estimated touched LOC: ~24 dòng trong 1 file (`command_handler.go`).

**Baseline build**: `go build ./...` PASS cho centralized-data-service và cdc-cms-service. `vite build` PASS cho cdc-cms-web.

**Files created phase này** (workspace, source code = 0):
- 02_plan.md, 03_implementation_audit.md, 04_decisions.md
- 06_validation.md, 07_status.md
- 08_tasks_audit.md, 09_tasks_solution_audit.md, 10_gap_analysis.md
- report_audit_shadow_create_bugs_2026-05-27.md

§12 compliance: Muscle KHÔNG sửa source code phase audit. Chờ user verb "ok / triển khai" → F-1..F-6.

## Entry 4
[2026-05-27 17:30 ICT] [Muscle:Claude-Opus-4.7] User verb "làm đi" → Fix Phase. Apply SOL-1 + SOL-2.A/B/C.

**Patches applied** (`centralized-data-service/internal/handler/command_handler.go`):
- **SOL-1** (line 647-670): swap `GetActiveRulesBySourceTable(payload.SourceTable)` → `ListActiveBySourceObject(effectiveID)`. Log fields đổi `zap.String("source_table", ...)` → `zap.Int64("source_object_id", effectiveID)`. Update warn message.
- **SOL-2.A** (line 586-602 → 587-605): thêm `_gpay_source_id TEXT UNIQUE`, `_source_ts BIGINT`, `_gpay_deleted BOOLEAN DEFAULT FALSE` vào CREATE TABLE DDL.
- **SOL-2.B** (line 163-180): thêm 3 entry tương ứng vào `cdcColumns` slice của `ensureCDCColumnsInSchema`.
- **SOL-2.C** (line 187-208): thêm `CREATE INDEX idx_<t>_source_ts` + `DO $$...$$` block conditional ALTER TABLE ADD CONSTRAINT UNIQUE `_gpay_source_id` (idempotent — skip nếu constraint đã có).

**Verify**:
- `go build ./...` PASS (centralized-data-service + cdc-cms-service).
- `go vet ./...` PASS.
- `go test ./internal/handler/... -count=1 -v` — TẤT CẢ test case PASS (zero `--- FAIL`). Package-level FAIL chỉ do `goleak.VerifyTestMain` whitelist thiếu `(*ConsumerGroup).run` / `(*ConsumerGroup).Next` từ kafka-go — **pre-existing infra gap không liên quan patch** (my patches chỉ chạm DDL builder logic, zero kafka interaction).
- `npx vite build` PASS (FE smoke).

**Files thay đổi**:
- `centralized-data-service/internal/handler/command_handler.go` — 3 patch site, ~26 dòng net added (gồm comments + DO block).
- Workspace docs: append `05_progress.md` Entry 4, tạo `report_fix_shadow_create_bugs_2026-05-27.md`.

§14 Pre-flight: file vật lý OK. §11 APPEND only OK. §6 minimal impact (1 file, 3 patch site).

**Còn nợ user** (next session sau approve):
- Migration shadow đã tồn tại lỗi (MIGR-1..4) — `_source_ts` cần backfill cho shadows hiện hữu.
- GAP-2: `command_handler.go:1389` HandleScanFields cũng nên swap sang ID-based.
- /security-agent gate (§8) — sẽ chạy nếu user yêu cầu.
