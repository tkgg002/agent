# 08_tasks.md — Task Breakdown chi tiết (Muscle execution checklist)

> Mỗi task có ID stable, DoD rõ, dependency map, estimate. Muscle dùng file này để self-track.

---

## TASK STATUS LEGEND
- ⬜ TODO (chưa start)
- 🟡 IN-PROGRESS
- ✅ DONE
- ❌ BLOCKED (đợi user / dep khác)
- 🔴 FAIL (cần escalate)

> Muscle update status trong `05_progress.md` (append-only), KHÔNG sửa file này.

---

## PHASE 0 — Coverage Baseline (3-5 ngày)

### T0.1 — Đo coverage baseline
- **ID:** `T0.1`
- **Owner:** Muscle
- **Estimate:** 30m
- **Depends:** Không
- **Steps:**
  1. `cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service`
  2. `make test-cover` → output `coverage.out`
  3. `go tool cover -func=coverage.out > coverage_baseline_2026-06-01.txt`
  4. `mv coverage_baseline_2026-06-01.txt /Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-cms-hexagonal-refactor-2026-06-01/`
- **DoD:**
  - File `coverage_baseline_2026-06-01.txt` tồn tại trong workspace
  - File ghi rõ % cho mỗi package
- **Verify:** `cat coverage_baseline_2026-06-01.txt | tail -1`

### T0.2 — Phân tích coverage report
- **ID:** `T0.2`
- **Estimate:** 1h
- **Depends:** T0.1
- **Steps:**
  1. Đọc `coverage_baseline_2026-06-01.txt`
  2. List package `internal/app/` < 60%
  3. List package `internal/server/` < 60%
  4. List package `internal/bootstrap/` < 70%
  5. Tạo `coverage_gap_2026-06-01.md` trong workspace (list package + % hiện tại + LOC test cần viết)
- **DoD:**
  - File `coverage_gap_2026-06-01.md` liệt kê đủ package thiếu coverage
- **Verify:** Mỗi package phân loại Low/Med/High effort

### T0.3 — Viết test bổ sung commands
- **ID:** `T0.3`
- **Estimate:** 1-2d
- **Depends:** T0.2
- **Steps:**
  1. Với mỗi command < 60% coverage:
     - Viết test happy path + error path
     - Mock dependencies (sẽ refactor sang port hẹp ở Phase 1)
  2. Re-run `make test-cover`
- **DoD:**
  - `internal/app/commands/` ≥ 60%
- **Verify:** `go tool cover -func=coverage.out | grep "internal/app/commands"`

### T0.4 — Viết test composition root
- **ID:** `T0.4`
- **Estimate:** 0.5d
- **Depends:** T0.2
- **Steps:**
  1. Viết smoke test `test/server/server_test.go`:
     - `TestNew_Success`: tạo Server với mock infra → no error
     - `TestNew_DBFail`: mock DB fail → error wrap đúng
  2. Coverage `internal/server/` ≥ 60%
- **DoD:**
  - `internal/server/` ≥ 60%
- **Verify:** `go tool cover -func=coverage.out | grep "internal/server"`

### T0.5 — Viết test bootstrap (high-risk)
- **ID:** `T0.5`
- **Estimate:** 1d
- **Depends:** T0.2
- **Steps:**
  1. Viết test cho `internal/bootstrap/registry_mirror.go`:
     - Mock DB + assert SQL emit
     - Test idempotent (chạy 2 lần kết quả như nhau)
  2. Viết test cho `internal/bootstrap/shadow_connection.go`:
     - Test connect success + retry
     - Test fallback khi shadow DB down
- **DoD:**
  - `internal/bootstrap/` ≥ 70%
- **Verify:** `go tool cover -func=coverage.out | grep "internal/bootstrap"`

### T0.6 — Re-run coverage + confirm
- **ID:** `T0.6`
- **Estimate:** 30m
- **Depends:** T0.3, T0.4, T0.5
- **DoD:**
  - `app ≥ 60%`, `server ≥ 60%`, `bootstrap ≥ 70%`
  - Append `coverage_after_T0_2026-06-01.txt` vào workspace
- **Verify:** Diff vs baseline > 0

### T0.7 — Setup `go-arch-lint` skeleton
- **ID:** `T0.7`
- **Estimate:** 1h
- **Depends:** Không
- **Steps:**
  1. `go install github.com/fe3dback/go-arch-lint@latest`
  2. Tạo `tools/lint/arch-lint.yml` theo template `03_implementation.md` Demo 4.1
  3. Thêm rule **CHỈ** `internal/infra/persistence/` được import `gorm.io/gorm`
  4. Thêm target `arch-lint` vào `Makefile`
  5. Chạy `make arch-lint` → expect FAIL (vì 18 cmd còn dùng gorm)
- **DoD:**
  - File `tools/lint/arch-lint.yml` tồn tại
  - `make arch-lint` chạy được (không crash, có thể fail)
- **Verify:** `cat tools/lint/arch-lint.yml | head`

### T0.8 — Smoke test 53 routes baseline
- **ID:** `T0.8`
- **Estimate:** 2h
- **Depends:** Không
- **Steps:**
  1. Tạo `scripts/smoke_routes.sh` theo template `03_implementation.md` Demo 5
  2. Extract 53 routes từ `internal/router/router.go` vào array
  3. Run service local: `make run`
  4. Run smoke: `bash scripts/smoke_routes.sh > smoke_baseline_2026-06-01.txt`
- **DoD:**
  - 53/53 routes PASS với 2xx hoặc 4xx-expected
  - File `smoke_baseline_2026-06-01.txt` lưu workspace
- **Verify:** `grep -c "PASS" smoke_baseline_2026-06-01.txt` = 53

### ✅ Gate G0 — Phase 0 complete
- T0.1 → T0.8 ALL DONE
- User approve trước khi sang Phase 1

---

## PHASE 1 — Port Split (2-3 ngày)

### T1.1 — Tạo 9 file port mới
- **ID:** `T1.1`
- **Estimate:** 0.5d
- **Depends:** Gate G0
- **Steps:**
  1. Tạo các file (KHÔNG xoá `repository.go` chưa):
     - `internal/app/ports/master_port.go`
     - `internal/app/ports/mapping_port.go`
     - `internal/app/ports/source_port.go`
     - `internal/app/ports/job_port.go`
     - `internal/app/ports/recon_port.go`
     - `internal/app/ports/wizard_port.go`
     - `internal/app/ports/system_connector_port.go`
     - `internal/app/ports/registry_port.go`
     - `internal/app/ports/transform_port.go`
  2. Mỗi file: 2-4 interface hẹp ngữ nghĩa, ≤ 80 LOC
  3. `go build ./...` PASS
- **DoD:**
  - 9 file tạo xong
  - Mỗi file ≤ 80 LOC
  - `wc -l internal/app/ports/*_port.go` confirm
- **Verify:** `ls internal/app/ports/*_port.go | wc -l` = 9

### T1.2 — Migrate caller (commands)
- **ID:** `T1.2`
- **Estimate:** 0.5d
- **Depends:** T1.1
- **Steps:**
  1. `grep -rln "ports.MasterRepo" internal/app/commands/` → list cmd file
  2. Với mỗi file, đổi field type:
     - `ports.MasterRepo` → `ports.MasterReader / Writer / Approver / Swapper` (chọn cái dùng đến)
  3. Sửa constructor param
  4. `go build ./...` PASS
  5. Repeat cho mapping, source, job, recon, wizard, system_connector, registry, transform
- **DoD:**
  - 0 file commands import `ports.MasterRepo / MappingRuleRepo / SourceRepo / ...`
- **Verify:** `grep -rn "ports.MasterRepo\|ports.MappingRuleRepo\|ports.SourceRepo" internal/app/commands/` → empty

### T1.3 — Migrate caller (queries)
- **ID:** `T1.3`
- **Estimate:** 0.5d
- **Depends:** T1.1
- **Steps:** Tương tự T1.2 nhưng cho `internal/app/queries/`
- **DoD:** 0 query file import legacy repo interface
- **Verify:** `grep -rn "ports.MasterRepo\|..." internal/app/queries/` → empty

### T1.4 — Migrate caller (handlers/api)
- **ID:** `T1.4`
- **Estimate:** 0.5d
- **Depends:** T1.2, T1.3
- **Steps:**
  1. Tìm handler trong `internal/api/` import legacy interface
  2. Đổi sang port hẹp
- **DoD:** 0 api file import legacy
- **Verify:** `grep -rn "ports.MasterRepo\|..." internal/api/` → empty

### T1.5 — Update composition root `server.go` tạm thời
- **ID:** `T1.5`
- **Estimate:** 0.5d
- **Depends:** T1.2, T1.3, T1.4
- **Steps:**
  1. Wire repo concrete (chưa tách `repos.go`) — inject vào constructor port hẹp
  2. `go build ./...` PASS
- **DoD:** Build PASS, smoke test PASS
- **Verify:** `make test` + `bash scripts/smoke_routes.sh` PASS

### T1.6 — XÓA `repository.go`
- **ID:** `T1.6`
- **Estimate:** 5m
- **Depends:** T1.5
- **Steps:**
  1. `rm internal/app/ports/repository.go`
  2. `go build ./...` PASS
- **DoD:**
  - File deleted
  - Build PASS
- **Verify:** `test ! -f internal/app/ports/repository.go && go build ./...`

### T1.7 — Test + smoke
- **ID:** `T1.7`
- **Estimate:** 30m
- **Depends:** T1.6
- **DoD:**
  - `make test` PASS
  - `make test-integration` PASS
  - Coverage ≥ baseline (NFR-1)
  - 53 routes smoke PASS
- **Verify:** Compare `coverage_after_T1.txt` vs `coverage_baseline.txt`

### ✅ Gate G1 — Phase 1 complete
- AC-2 + AC-3 PASS
- User approve

---

## PHASE 2 — Composition Root Split (2 ngày)

### T2.1 — Tạo `infra.go`
- **ID:** `T2.1`
- **Estimate:** 2h
- **Depends:** Gate G1
- **Steps:**
  1. Copy DB/NATS/Redis/OTel init từ `server.go` sang `infra.go`
  2. Convert thành pure function `buildInfra(cfg, logger) (*Infra, error)`
  3. Define `Infra` struct (xem `03_implementation.md` Demo 2.2.1)
  4. Thêm method `(i *Infra) Close() error`
- **DoD:** `infra.go` tồn tại + pure func + `go build` PASS
- **Verify:** `grep -q "func buildInfra" internal/server/infra.go`

### T2.2 — Tạo `repos.go`
- **ID:** `T2.2`
- **Estimate:** 2h
- **Depends:** T2.1
- **Steps:**
  1. Define `Repos` struct typed (KHÔNG `map[string]any`)
  2. `buildRepos(infra *Infra) Repos` — pure func
- **DoD:** `repos.go` tồn tại, build PASS, `Repos` struct có ≥ 20 field port hẹp
- **Verify:** `grep -c "ports\." internal/server/repos.go` ≥ 20

### T2.3 — Tạo `bus.go`
- **ID:** `T2.3`
- **Estimate:** 4h
- **Depends:** T2.2
- **Steps:**
  1. Move khối `cmdBus.RegisterSubject` + `RegisterSync` (~100 dòng) từ `server.go` sang `bus.go`
  2. Tách thành: `buildCommandBus()`, `registerSyncHandlers()`, `registerAsyncSubjects()`
  3. NFR-3: KHÔNG đổi subject name nào — diff list trước/sau
- **DoD:**
  - `bus.go` tồn tại
  - 26 NATS subject preserve (không thêm, không bớt, không đổi tên)
- **Verify:** `grep "RegisterSubject" internal/server/bus.go | wc -l` so với baseline

### T2.4 — Tạo `routes.go`
- **ID:** `T2.4`
- **Estimate:** 2h
- **Depends:** T2.2
- **Steps:**
  1. Move router building từ `server.go` sang `routes.go`
  2. `buildRouter(handlers, mw) *fiber.App` — pure func
  3. NFR-2: KHÔNG đổi 53 route path
- **DoD:** 53 route preserve
- **Verify:** `bash scripts/smoke_routes.sh` PASS

### T2.5 — Tạo `workers.go`
- **ID:** `T2.5`
- **Estimate:** 2h
- **Depends:** T2.1, T2.3
- **Steps:**
  1. Move worker init từ `server.go` sang `workers.go`
  2. `buildWorkers(infra, bus) []Worker`
  3. Define `Worker` interface (Start/Stop/Name)
- **DoD:** Workers tách xong, start/stop graceful
- **Verify:** Integration test cover worker

### T2.6 — Refactor `server.go` ≤ 80 LOC
- **ID:** `T2.6`
- **Estimate:** 2h
- **Depends:** T2.1 → T2.5
- **Steps:**
  1. Xóa code đã move
  2. Giữ lại: struct `Server` + `New()` orchestrate + `Run()` + `Close()`
  3. Đảm bảo ≤ 80 LOC
- **DoD:** `wc -l internal/server/server.go` ≤ 80
- **Verify:** `[ $(wc -l < internal/server/server.go) -le 80 ]`

### T2.7 — Integration test + start time
- **ID:** `T2.7`
- **Estimate:** 1h30
- **Depends:** T2.6
- **Steps:**
  1. `make test-integration` PASS
  2. Đo start time: `time ./bin/cms` 3 lần → trung bình
  3. So NFR-5: ≤ baseline + 100ms
- **DoD:**
  - All AC-4, AC-5 PASS
  - NFR-5 PASS
- **Verify:** Append `phase2_metrics_2026-06-01.txt` vào workspace

### ✅ Gate G2 — Phase 2 complete

---

## PHASE 3 — Refactor 18 Commands (5-7 ngày)

### T3.X — Template (lặp cho mỗi command, X = 1..18)
Reference `02_plan.md` §6.2 cho thứ tự command.

- **ID:** `T3.X` (vd: `T3.1` cho `mark_failed_log_retrying`)
- **Estimate:** 2h / command
- **Depends:** Gate G2 (hoặc T3.{X-1} nếu trong cùng batch)
- **Steps:**
  1. Đọc file command cũ → identify SQL operation
  2. Thêm method vào port hẹp tương ứng (vd: `JobLogWriter.MarkRetrying(...)`)
  3. Implement method trong `internal/infra/persistence/<aggregate>_repo_gorm.go`:
     - Wrap nguyên block SQL từ command cũ
     - KHÔNG đổi logic
  4. Đổi command:
     - Remove `db *gorm.DB` field
     - Add `<port> ports.<XYZ>` field
     - Đổi constructor param
     - Đổi `Handle()` body: thay raw query bằng method call
  5. Update test:
     - Tạo `fake<XYZ>` implement port hẹp
     - Test happy + error path
  6. `go build ./... && make test` PASS
  7. `git commit -m "refactor(cmd): <name> → port"`
- **DoD per command:**
  - Command file 0 import `gorm.io/gorm`
  - Test PASS
  - 1 commit độc lập
- **Verify per command:** `grep -L "gorm.io/gorm" internal/app/commands/<name>.go` = path

### Stop rule (R1)
Nếu 3 command liên tiếp fail test → STOP, escalate Brain re-plan.

### T3.batch1 — Mark batch 1 DONE
- Sau T3.1, T3.2, T3.3, T3.4: `make test-integration` PASS + smoke 53 routes PASS
- **Verify:** `grep -rln "gorm.io/gorm" internal/app/commands/` count giảm 4

### T3.batch2 — Mark batch 2 DONE
- Sau T3.5 → T3.12: integration + smoke PASS
- Coverage ≥ baseline

### T3.batch3 — Mark batch 3 DONE (saga complex)
- Sau T3.13 → T3.18: integration + smoke PASS
- **Final verify (AC-6):** `grep -rln "gorm.io/gorm" internal/app/commands/` → **empty**

### T3.final — Run linter
- **ID:** `T3.final`
- **Estimate:** 30m
- **Steps:**
  1. `make arch-lint` → PASS (vì rule `app cannot depend gorm` giờ enforce được)
- **DoD:** Linter PASS

### ✅ Gate G3 — Phase 3 complete
- AC-6 PASS
- 18 commits độc lập (revertable)
- User approve

---

## PHASE 4 — Vertical Slice (OPTIONAL, 5-8 ngày)

### T4.1 — Move `source` BC
- **ID:** `T4.1`
- **Estimate:** 1d
- **Depends:** Gate G3 + user approve Phase 4
- **Steps:**
  1. Tạo cấu trúc `internal/bc/source/{commands,queries,api,domain,infra,ports}/`
  2. `git mv internal/app/commands/{register_source,provision_source,...}.go internal/bc/source/commands/`
  3. `git mv internal/app/queries/{get_source,list_source,...}.go internal/bc/source/queries/`
  4. `git mv internal/api/source_*.go internal/bc/source/api/`
  5. `git mv internal/domain/source/*.go internal/bc/source/domain/`
  6. `git mv internal/infra/persistence/source_repo_gorm.go internal/bc/source/infra/`
  7. `git mv internal/app/ports/source_port.go internal/bc/source/ports/`
  8. Update import path trong moved file: `cdc-cms-service/internal/app/commands` → `cdc-cms-service/internal/bc/source/commands`
  9. `go build ./...` PASS
  10. `make test` PASS
- **DoD:**
  - source BC self-contained trong `internal/bc/source/`
  - Git blame preserve
- **Verify:** `git log --follow internal/bc/source/commands/register_source.go | head` chứa commit cũ

### T4.2 → T4.8 — Move các BC khác
Tương tự T4.1, theo thứ tự: `master → mapping → transform → reconciliation → wizard → system_control → observability`.

### T4.9 — Tạo `internal/platform/` + `internal/shared/`
- **ID:** `T4.9`
- **Estimate:** 1d
- **Steps:**
  1. `git mv internal/platform/bus/* internal/platform/bus/` (nếu cần tổ chức lại)
  2. Move audit middleware, auth, ratelimit, deprecation, observability vào `internal/platform/`
  3. Tạo `internal/shared/{naming,pg,id}/` cho technical primitive
  4. KHÔNG move domain enum vào `internal/shared/`
- **DoD:** 2 folder tồn tại, build PASS

### T4.10 — Enforce linter strict
- **ID:** `T4.10`
- **Estimate:** 2h
- **Steps:**
  1. Update `tools/lint/arch-lint.yml` thêm rule BC isolation (xem `03_implementation.md` Demo 4.2)
  2. Wizard exception explicit
  3. `make arch-lint` PASS
- **DoD:** Linter rule strict PASS

### ✅ Gate G4 — Phase 4 complete
- AC-7 PASS
- User approve

---

## CROSS-CUTTING TASKS (mọi phase)

### TC.1 — Update `05_progress.md`
Mỗi task done → append entry vào `05_progress.md`:
```
[2026-06-XX HH:MM] [Muscle:claude-opus-4-7] T<X.Y> DONE — <ngắn gọn>
```

### TC.2 — Run `/security-agent` sau mỗi phase
- DoD: Security scan PASS, không có high/critical vuln

### TC.3 — Update `report_*.md`
Sau mỗi phase: append files changed + LOC delta vào `report_*.md`

### TC.4 — Pre-flight check trước commit cuối
- `make test` PASS
- `make test-integration` PASS
- `go vet ./...` PASS
- `golangci-lint run ./...` PASS
- `make arch-lint` PASS
- `make swagger` PASS, no drift
- 53 routes smoke PASS
- Coverage ≥ baseline

---

## TASK GRAPH (text format)

```
T0.1 ─┬─ T0.2 ─┬─ T0.3 ─┐
      │        ├─ T0.4 ─┤
      │        └─ T0.5 ─┴─ T0.6
T0.7 ─┘                     │
T0.8 ─────────────────────  │
                             │
                            G0
                             │
T1.1 → T1.2 ┐
            ├→ T1.4 → T1.5 → T1.6 → T1.7 → G1
       T1.3 ┘                                │
                                             │
T2.1 → T2.2 → T2.3 ┐
            │      ├→ T2.6 → T2.7 → G2
            ├ T2.4 ┤              │
            └ T2.5 ┘              │
                                  │
T3.1 → T3.2 → ... → T3.18 → T3.final → G3
                                        │
[optional] T4.1 → T4.2 → ... → T4.8 → T4.9 → T4.10 → G4
```

---

## TỔNG SỐ TASK

| Phase | Task count |
|---|---|
| Phase 0 | 8 |
| Phase 1 | 7 |
| Phase 2 | 7 |
| Phase 3 | 18 + 1 final = 19 |
| Phase 4 (optional) | 10 |
| Cross-cutting | 4 |
| **Total mandatory (P0-3)** | **45** |
| **Total with P4** | **55** |
