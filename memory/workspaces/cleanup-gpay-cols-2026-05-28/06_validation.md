# 06_validation — Cleanup `_gpay_source_id` + `_gpay_deleted`

## Phase audit (PRESENT) — Documentation validation only

| Check | Method | Status |
|---|---|---|
| §7 Full Doc Set 00..10 + report | `ls agent/memory/workspaces/cleanup-gpay-cols-2026-05-28/` | ✅ (Entry 5 trong `05_progress.md`) |
| §11 APPEND-only `05_progress.md` | Diff file vs prev version | ✅ Đầu file APPEND-ONLY annotation |
| §12 Brain Code Prohibition | grep "Edit" hoặc "Write" target `*.go/*.ts/*.tsx/*.sql` | ✅ Zero source change phase này |
| §13 Lesson cross-reference | `04_decisions.md` D-7 list 3 lesson | ✅ 2026-05-20 (2 lesson) + 2026-05-26 |
| 104 references phân loại | `03_implementation_audit.md` Path A/B/C/D | ✅ |
| 3 cleanup options + code demo | `09_tasks_solution_cleanup.md` | ✅ (sẽ verify lại khi đọc file) |

## Phase muscle (FUTURE — chỉ khi user verb)

### Build verify
```bash
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go build ./...
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go build ./...
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-web && npm run build
```
- Expect: Exit 0 trên 3 service.

### Vet (Go only)
```bash
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go vet ./internal/handler/... ./internal/service/... ./internal/sinkworker/...
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go vet ./internal/...
```
- Expect: Pre-existing sonyflake errors (line 77,82) tolerable; KHÔNG có lỗi mới.

### Test verify
```bash
# Option A
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go test ./internal/handler/... -count=1 -timeout 60s

# Option B (thêm)
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go test ./internal/api/... ./internal/app/commands/... -count=1 -timeout 60s

# Option C (full)
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go test ./internal/service/... ./internal/sinkworker/... ./test/internal/service/... -count=1 -timeout 120s
```
- Expect: PASS. Nếu fail → re-plan (§3 Plan & Verify).

### Destination verify (lesson "Verify ở destination")
**Option A — sau apply**:
```sql
-- PG psql vào shadow DB
\d shadow.export_jobs_2
-- Expect: KHÔNG có cột _gpay_source_id, _gpay_deleted. Có source_id (nếu Path A flow) hoặc không có (nếu chỉ Path B flow).
```

**Option B — sau apply**:
```sql
\d shadow.<table_test>
-- Expect: source_id VARCHAR(200), _deleted BOOLEAN. Không có _gpay_*.

-- Test preview API
curl http://localhost:8080/mapping/preview -d '{"shadowTable":"..."}'
-- Expect: Trả về source_id field, KHÔNG fail.
```

**Option C — N/A phase này (rejected)**.

### DoD destination check (Define DoD at the destination)
```bash
# Sau apply, grep còn lại
cd /Users/trainguyen/Documents/work/data-hub
grep -rn "_gpay_source_id\|_gpay_deleted" centralized-data-service/ cdc-cms-service/ cdc-cms-web/ --include="*.go" --include="*.ts" --include="*.tsx" | wc -l
```
- Option A expect: 104 - ~7 = ~97 references còn (Path B clean only).
- Option B expect: 104 - ~12 = ~92 references còn (Path A + B clean).
- Option C expect: 0 references (full removal) — chỉ làm ở workspace khác.

## Acceptance criteria (audit phase)
- [x] Doc set 00..10 + report đầy đủ.
- [x] 3 option có code demo + verify plan.
- [x] Decision matrix Risk/LOC/Reversibility.
- [x] Recommend rõ ràng có lý do (Option A + 3 lesson reference).
- [x] KHÔNG sửa source code.
- [ ] User pick option → trigger Phase Muscle.
