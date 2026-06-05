# 07_status — Audit Shadow Create Bugs

## Current state: AUDIT + FIX COMPLETE — chờ user verb migration / security-gate

> Update 2026-05-27 17:30 ICT: User verb "làm đi" → Fix Phase applied. Build + test verify PASS. Xem `report_fix_shadow_create_bugs_2026-05-27.md`.

### Done
- ✅ Workspace bootstrapped per §7 GEMINI.
- ✅ Bug 1 root cause identified: `mapping_rule_v2_repo.go:54-61` + caller `command_handler.go:649`.
- ✅ Bug 2 root cause identified: `command_handler.go:586-602` + `command_handler.go:163-172`.
- ✅ Code demo cho 2 fix viết đầy đủ trong `09_tasks_solution_audit.md` (3 patch site, ~24 LOC).
- ✅ Baseline build verify PASS cho 3 service (cdc-cms-web, cdc-cms-service, centralized-data-service).
- ✅ Workspace doc set đầy đủ prefix 00..10 + report.

### Done (Fix Phase, 2026-05-27 17:30 ICT)
- ✅ F-1 SOL-1 applied: `command_handler.go` line 647-670 swap to `ListActiveBySourceObject(effectiveID)`.
- ✅ F-2 SOL-2.A applied: CREATE TABLE +3 cột.
- ✅ F-2 SOL-2.B applied: cdcColumns +3 entries.
- ✅ F-2 SOL-2.C applied: idx_source_ts + UNIQUE constraint (DO block idempotent).
- ✅ F-3 build + vet PASS 3 service. Test case PASS (zero `--- FAIL`); goleak pre-existing not related.
- ✅ F-4 `report_fix_shadow_create_bugs_2026-05-27.md` viết.

### Blocked (cần user verb)
- ⏳ MIGR-1..4 (migration shadow đã tạo lỗi).
- ⏳ GAP-2 fix `HandleScanFields` line 1389 cùng pattern.
- ⏳ /security-agent gate (§8) — chạy khi user yêu cầu.

### Future (out-of-scope hôm nay)
- MIGR-1..4: Migration shadow đã tồn tại lỗi.
- GAP-1..6: refactor + integration test + lint rule.

## Sign-off checklist (§14 Pre-flight)
- [x] §11 Memory Protection: `05_progress.md` APPEND only — verified.
- [x] §12 Brain Code Prohibition: KHÔNG sửa source code phase này — verified (workspace docs only).
- [x] §7 Full Doc Set: 00..10 + report — file exists check trong report.
- [x] §6 Simplicity First / Demand Elegance: SOL-1 dùng API có sẵn, SOL-2 minimal patch — không over-engineer.
- [x] User constraint "không cheat DB / không đổi config" — respected (fix tại core flow `HandleCreateDefaultColumns`, không ALTER thủ công).
- [x] User constraint "report dựa trên kết quả tính toán thực tế" — file/line cụ thể, có cross-check evidence.
- [x] Build verify 3 service — PASS.
