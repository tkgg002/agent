# Report — Flow 1 /loop aggregate iterations (2026-05-07 ICT)

> **Author**: max-Brain
> **Cron job**: `1975934c` recurring `*/5 * * * *`
> **User directive**: `/loop 5p verify task của x2, review và lên task mới cho x2 ... bằng mọi giá phải lên đc flow1`
> **Per-iteration detail**: `report_flow1_loop_iter1_*` đã commit; iteration sau APPEND vào file này.

---

## Iteration #2 — 2026-05-07 ICT (re-verify x2 progress)

### A. Lessons re-read (grep `flow.?1|brain-delegate|x2|muscle-plan|role-swap`)

| Lesson | Line | Áp dụng iter này |
|---|---|---|
| L-ROLE-SWAP-MID-TRANSFORMATION | 2446 | Đã apply iter#1 — không trigger lần này |
| L-MUSCLE-PLAN-PROHIBITION | 2492 | x2 vẫn chưa viết `09_tasks_solution_flow1_x2_*` (P1 iter#1) → soft-reminder qua coordination |
| Lesson "Open questions cần stakeholder rule" | 1445 | Boss còn 5 decision pending → KHÔNG advance Phương án Y/backfill cho đến khi approve |

### B. Service state (real verified)

| Service | PID | Uptime | Health | Δ vs iter#1 |
|---|---|---|---|---|
| cdc-cms-service | 64511 | 21m22s | `GET /api/system/health` HTTP 200 (2.5ms), `GET /health` HTTP 200 | +6m20s, không restart |
| cdc-worker-host | 23565 | 2d 00:59:57 | log shows JobMonitor close-loop active (vẫn duplicate emit `schedule_id` 2x trong 1s) | +5m, không restart |
| gpay-postgres-cdc 5433 | container | healthy | psql OK | unchanged |
| gpay-postgres-shadow 5436 | container | healthy | psql OK + 1720 rows | unchanged |

### C. DB state (delta vs iter#1)

| Object | iter#1 | iter#2 | Δ |
|---|---|---|---|
| `shadow_binding.id=52` ddl_status | pending | **pending** | NO_CHANGE |
| `source_object_registry.id=44` state | shadow_pending | **shadow_pending** | NO_CHANGE |
| Path A `cdc_dw.shadow_payment_bill_service.refund_requests` count | 0 | (skip — Path A orphan, no trigger to populate) | — |
| Path B `cdc_shadow.shadow_payment_bill_service.refund_requests` count | 1720 | **1720** | NO_CHANGE (snapshot stable) |
| Phantom rows id 33,34,35,37 state='active' | active | active | NO_CHANGE (Phase 2 Y chưa Boss approve) |

→ **Verdict**: System ở trạng thái stable. Không có forward progress vì Boss decision pending + x2 chưa commit + max không được implement worker code.

### D. x2 progress assessment (delta vs iter#1)

| Task plan iter#1 | Status iter#2 | Evidence |
|---|---|---|
| **x2.1 P0**: Stage + commit `shadow_automator.go` fix | ⚠️ **PARTIAL** — đã `git add` (M staged) nhưng CHƯA `git commit` | `git status` show `M cdc-cms-service/internal/infra/persistence/shadow_automator.go` + `A cdc-cms-service/report_flow1_run_x2_2026-05-07.md` (cả 2 staged, HEAD vẫn `b453d36`) |
| **x2.2 P1**: Viết retroactive `09_tasks_solution_flow1_x2_2026-05-07.md` | ❌ **NOT STARTED** | `ls .../09_tasks_solution_flow1*.md` → no matches |
| **x2.3 P2**: Fix G-10 normalize `pk_type='string'→'text'` | ❌ **NOT STARTED** | cms repo không có file modify cho task này |
| **x2.4 P3**: P3.1 endpoint `POST /api/v1/sources/test` | ❌ **NOT STARTED** | (chờ G-10 đóng) |

**Improvement**: x2 đã `git add` (chuyển từ "working tree dirty" iter#1 → "staged but uncommitted" iter#2). 1 step closer to commit.

**Concern**: 21+ phút từ iter#1 mà x2 vẫn chưa `git commit` lẫn ship `09_tasks_solution_*`. Có thể x2 đang chờ Boss approve hoặc đang làm task khác.

### E. max-Brain progress (lane = worker)

| Task plan iter#1 | Status iter#2 | Note |
|---|---|---|
| **max.1 P0**: Decision doc `04_decisions_flow1_path_a_vs_b_2026-05-07.md` (G-8) | ❌ NOT STARTED | Tier upgrade — sẽ làm trong iter#3 hoặc khi user nudge |
| **max.2 P1**: Plan G-7 worker restart | ❌ NOT STARTED | Boss approve gate |
| **max.3 P2**: Phương án Y refactor + backfill | ❌ NOT STARTED | Boss approve gate (breaking response) |
| **max.4 P3**: G-9 worker auto-refresh | ❌ NOT STARTED | Boss approve gate trên worker code change |

**Self-honest**: iter#1 đã spend ~30 phút trên discovery + report; iter#2 spend ~5 phút verify. max-Brain CHƯA produce decision doc G-8 — đó là task P0 tier doc (Brain plan only, không phải code) → có thể làm ngay không cần Boss approve. **Action iter#2**: produce G-8 decision doc trong iteration kế tiếp (iter#3) sau khi user/Boss respond.

### F. Worker observation (passive — không touch code)

- Log shows `schedule_id 13` close 2 lần trong 1 giây (`1778125144.16514` + `1778125144.2341251`):
  ```
  job monitor: schedule closed schedule_id=13 status=success master=orders_addtest
  job monitor: schedule closed schedule_id=13 status=success master=orders_addtest  ← duplicate
  ```
- Confirms P5.2 dedup issue persistent (đã document trong `08_tasks_flow1_e2e §P5.2`).
- **No regression** but cosmetic noise — defer.

### G. Soft nudge cho x2 (qua coordination, không gọi trực tiếp)

x2 nên hoàn thành theo thứ tự:
1. **Commit** (1 phút): `cd /Users/trainguyen/Documents/work/cdc-system && git commit -m "fix(cms): split shadow DDL into individual stmts to avoid PrepareStmt 42601"` — mở khóa cms repo cho rebuild + downstream task.
2. **Ship `09_tasks_solution_flow1_x2_2026-05-07.md`** (30 phút): retroactive document workflow để Boss audit trail intact (per CLAUDE.md §11).
3. (Sau commit + ship) Resume P2 G-10 fix.

### H. Phương án Z (cms 2-step) — STATE FOR BOSS

x2 đã thi công Phương án Z manual qua curl:
- Step 1: `POST /api/v1/source-objects/register` HTTP 202 ✓
- Step 2: `POST /api/v1/cms/sources/44/provisioning/mode {mode:manual}` HTTP 200 ✓
- Step 3: `POST /api/v1/cms/sources/44/provisioning/advance` HTTP 200 → state→`shadow_pending` ✓
- Step 4 (poll state=shadow_active): ❌ FAIL vì G-7 (worker subscriber `cdc.cmd.shadow.bind` tắt do `PROVISIONING_ORCHESTRATOR_ENABLED` chưa set)

→ Phương án Z block bởi G-7. **Cần Boss approve enable env var + restart worker** mới đạt AC-6.

### I. Boss decision matrix (5 items pending — không đổi từ iter#1)

| # | Decision | Pri | Block | Lane |
|---|---|---|---|---|
| 1 | G-7 worker enable `PROVISIONING_ORCHESTRATOR_ENABLED=1` + restart | **P0** | Phương án Z AC-6 | worker (max) |
| 2 | G-8 architecture: consolidate Path B (5436) HOẶC redirect Path A | P1 | Path A orphan empty table | architecture (max plan) |
| 3 | Phương án Y breaking change `/v2/sources/register` response | P2 | Phase 2 fix legacy | worker (max) |
| 4 | Backfill 4 phantom rows id 33,34,35,37 state='active'→'draft' | P2 | Legacy data drift | data (max) |
| 5 | P4 MariaDB Debezium plugin rebuild kafka-connect image | P3 | MariaDB connector RUNNING | infra (Boss) |

**Recommendation cho user/Boss**: approve **#1 ngay** (lowest risk + unblocks AC-6 + tăng confidence Phương án Z working end-to-end).

### J. Files modified iter#2

- **Created**: `report_flow1_loop_2026-05-07.md` (this aggregate file)
- **APPEND** (sẽ làm tiếp): `coordination_max_x2_2026-05-07.md` — iter#2 nudge
- **APPEND** (sẽ làm tiếp): `05_progress.md` — iter#2 entry

### K. Self-pacing

Cron `1975934c` recurring 5m → next fire ~5 min. Iteration #3 sẽ:
1. Re-verify x2 commit + ship `09_tasks_solution_*`.
2. Nếu x2 commit → max-Brain produce `04_decisions_flow1_path_a_vs_b_2026-05-07.md` (G-8 decision doc P0).
3. Nếu Boss approve #1 (G-7) → escalate to user trong report.

— max-Brain (loop iteration #2)

---

## Iteration #3 — 2026-05-07 ICT (x2 commit detected + G-10 in progress + G-8 decision doc shipped)

### A. Lessons re-read (grep `flow.?1|brain-delegate|x2|muscle-plan|role-swap`)

Cùng 2 lessons relevant như iter#1+#2: L-ROLE-SWAP-MID-TRANSFORMATION + L-MUSCLE-PLAN-PROHIBITION. Không có lesson mới apply.

### B. Service state (verified)

| Service | PID | Uptime | Health | Δ vs iter#2 |
|---|---|---|---|---|
| cdc-cms-service | 64511 | 24m57s | HTTP 200 (2.6ms) | +3m35s, không restart |
| cdc-worker-host | 23565 | 2d 01:03:32 | log JobMonitor close-loop active | +4m, không restart |
| `cdc-admin-api-f3v2` | 21133 | 2d 01:10:21 | n/a | +5m (passive process discovered iter#3) |
| gpay-postgres-cdc 5433 | container | healthy | psql OK | unchanged |
| gpay-postgres-shadow 5436 | container | healthy | psql OK + 1720 rows stable | unchanged |

### C. x2 PROGRESS DELTA — significant improvement

| Task plan iter#1 | iter#2 | iter#3 | Δ |
|---|---|---|---|
| **x2.1 P0** Stage + commit `shadow_automator.go` | STAGED, no commit | ✅ **COMMITTED** `0cef7af` | DONE |
| **x2.2 P1** `09_tasks_solution_flow1_x2_2026-05-07.md` | NOT STARTED | NOT STARTED | NO_CHANGE |
| **x2.3 P2** G-10 normalize `pk_type='string'→'text'` | NOT STARTED | 🟡 **IN PROGRESS** (working tree, not staged/committed) | +2 steps (designed + unit-tested) |
| **x2.4 P3** P3.1 endpoint test | NOT STARTED | NOT STARTED | NO_CHANGE |

#### x2 commit `0cef7af` review (max-Brain APPROVE)

```
Date: Thu May 7 10:40:09 2026 +0700
fix(cms): split multi-statement shadow DDL to unblock Flow 1 Register

ShadowAutomator.createShadowDDL build 1 multi-statement string
(CREATE SCHEMA + CREATE TABLE + 3x CREATE INDEX) rồi db.Exec(ddl).
Global GORM session set PrepareStmt=true (pkgs/database/postgres.go:24)
nên PostgreSQL reject với SQLSTATE 42601 ...
```

→ Commit message rõ root cause + fix pattern. ✅ APPROVE.

#### x2 working-tree G-10 fix review (max-Brain APPROVE)

`internal/app/commands/register_registry.go` diff:
```go
+	"strings"
...
+	entry.PrimaryKeyType = normalizePKType(entry.PrimaryKeyType)
...
+func normalizePKType(t string) string {
+	if strings.EqualFold(strings.TrimSpace(t), "string") {
+		return "text"
+	}
+	return t
+}
```

`internal/app/commands/commands_test.go` diff: thêm `TestNormalizePKType` 7 cases (string/STRING/whitespace/text/BIGINT/empty/objectid).

**Verdict**: ✅ APPROVE
- Pattern đúng (single normalizer, narrow scope only "string" → "text", others pass-through).
- Unit test cover edge cases (case-insensitive, whitespace, empty string, pass-through).
- Comment giải thích root cause (worker `command_handler` propagate verbatim → SQLSTATE 42704).
- Không over-engineer.

**Action recommended**: x2 stage + commit khi build/test PASS.

### D. G-8 architectural drift — decision doc shipped (Brain commitment iter#3 delivered)

**File mới**: `04_decisions_flow1_path_a_vs_b_2026-05-07.md` (workspace).

**Evidence iter#3 deeper investigation**:
- 2 PG instance độc lập (5433 cdc_dw vs 5436 cdc_shadow), schema/cột khác (10 cols vs 9 cols).
- Worker config (line 9-36) + docker-compose:67 + cms config (line 8-11): cả 3 đều shadowDb=5433 cdc_dw.
- 3 Kafka Connect connector đều là Debezium SOURCE (không có sink).
- cms PID 64511 + admin-api-f3v2 PID 21133 lsof: chỉ connect 5433, KHÔNG tới 5436.

→ **No active writer to Path B 5436 trong iter#3**. 1720 rows = test artifact từ session trước (có thể env override `CDC_SHADOW_DB_URL=...:5436/cdc_shadow` hoặc x2 manual COPY).

**4 phương án + recommendation**:
- A1: Deprecate Path B (drop gpay-postgres-shadow container) — 30 min, low risk.
- A2: Adopt Path B (deprecate Path A) — 4h, medium risk, breaks config intent.
- A3: Hybrid 2-cluster (control plane vs shadow data) — 6h, medium-high risk.
- **A4: Status quo (Path A only) + A1 cleanup** ← **max-Brain RECOMMEND**.

**Lý do A4**: config + comment intent rõ shadow tại 5433 cdc_dw; Path B = test artifact; A4 không block Phương án Y; nếu Boss muốn A3 dual-cluster sau, có thể migrate sau Phase 2 stable.

**Open questions cho Boss**:
- Q-1: Path B tồn tại có chủ đích?
- Q-2: Approve A4 + A1 cleanup?
- Q-3: A3 effort 6h sau Phase 2 Y?

### E. Boss decision matrix update (5 → 6 items)

| # | Decision | Pri | Block | Lane | iter#3 status |
|---|---|---|---|---|---|
| 1 | G-7 worker enable `PROVISIONING_ORCHESTRATOR_ENABLED=1` + restart | **P0** | Phương án Z AC-6 | worker (max) | pending |
| 2 | G-8 phương án A4 (status quo Path A) + A1 cleanup | P1 | Path A orphan empty | architecture | **decision doc shipped iter#3** |
| 3 | Phương án Y breaking change `/v2/sources/register` | P2 | Phase 2 fix legacy | worker (max) | pending |
| 4 | Backfill 4 phantom rows id 33,34,35,37 | P2 | Legacy data drift | data (max) | pending |
| 5 | P4 MariaDB Debezium plugin rebuild | P3 | MariaDB connector RUNNING | infra (Boss) | pending |
| 6 | A3 dual-cluster effort 6h sau Phase 2 Y | P3 | Long-term architecture | architecture | new — pending |

**Highest-leverage action cho Boss**: Approve #1 (G-7 worker restart) — unblocks Phương án Z AC-6.

### F. Updated task plan iter#3

#### x2 (cms-lane) — TODO

| # | Pri | Task | Effort | Boss approve? |
|---|---|---|---|---|
| **x2.A** | **P0** | Stage + commit G-10 fix (`register_registry.go` + `commands_test.go`) sau khi `go build && go test ./internal/app/commands -count=1` PASS | 5 min | NO |
| **x2.B** | **P1** | Ship retroactive `09_tasks_solution_flow1_x2_2026-05-07.md` (workflow audit trail per CLAUDE.md §11) | 30 min | NO |
| **x2.C** | P3 | P3.1 endpoint `POST /api/v1/sources/test` | 2h | NO (sau khi G-10 commit) |

#### max-Brain — TODO

| # | Pri | Task | Effort | Boss approve? |
|---|---|---|---|---|
| **max.A** | iter#3 commit | ✅ DONE iter#3: `04_decisions_flow1_path_a_vs_b_*` (G-8 plan) | — | — |
| **max.B** | iter#4 plan | Re-verify x2 G-10 commit + audit DB state delta | 5 min | — |
| **max.C** | Boss approve gate | Phương án Y refactor `admin/source_register.go:92` + backfill SQL | 2h | YES |

### G. Cron + self-pacing

Cron `1975934c` recurring 5m → next fire ~5 min. Iter#4 sẽ:
1. Re-verify x2.A commit G-10.
2. Re-verify x2.B 09_tasks_solution shipped.
3. Re-verify Boss approve trên item #1 hoặc #2.
4. Nếu Boss approve #1: max.C escalate Phương án Y cmd-level plan.

### H. Files iter#3

- **Created**: `04_decisions_flow1_path_a_vs_b_2026-05-07.md` (workspace, 4-option G-8 decision doc)
- **APPEND** (this file): iter#3 section
- **APPEND** (sẽ làm tiếp): coordination + 05_progress

— max-Brain (loop iter#3)
