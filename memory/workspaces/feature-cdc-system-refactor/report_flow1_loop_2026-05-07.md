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

---

## Iteration #4 — 2026-05-07 ICT (re-verify x2.A + x2.B commitments)

### A. Lessons re-applied

| Lesson | Trigger iter#4? | Action |
|---|---|---|
| L-MUSCLE-PLAN-PROHIBITION | NO | x2 đã ship `09_tasks_solution_flow1_x2_*` (7906B, May 7 10:43) — workflow gate satisfied |
| L-ROLE-SWAP-MID-TRANSFORMATION | NO | Lane lock vẫn giữ — max chỉ APPEND doc, x2 own cms code |
| L-FLOW1-LEGACY-ADMIN-BYPASS (pending) | NO (chờ Boss approve Phương án Y) | — |

### B. Service state (real verified)

| Service | PID | Uptime | Health | Δ vs iter#3 |
|---|---|---|---|---|
| cdc-cms-service | 64511 | 33m38s | `GET /api/system/health` HTTP 200 (4.0ms) | +6m, không restart |
| cdc-worker-host | 23565 | 2d 01:12:13 | (no health endpoint exposed; process alive) | +5m, không restart |
| cdc-admin-api-f3v2 | 21133 | 2d 01:16:53 | (legacy admin) | +5m, không restart |
| gpay-postgres-cdc 5433 | container | healthy | psql OK | unchanged |

→ Tất cả service alive, cms uptime tăng tuyến tính → x2 thi công (`adc6faf`) qua hot path không restart server. cms binary `/tmp/cdc-cms-service-flow1` start lúc ~10:14 ICT chưa pickup `adc6faf` (commit 10:44) → **next register call sẽ vẫn dùng binary cũ chưa có G-10 fix**. x2 cần rebuild + restart cms để G-10 effective (sẽ note trong task plan iter#5).

### C. DB state (delta vs iter#3)

```sql
-- src 33,34,35,37 = active (legacy phantom — unchanged)
-- src 44 (refund-requests) = shadow_pending — UNCHANGED vì G-7 worker chưa enable
-- shadow_binding 50 (src 42) = pending — unchanged
-- shadow_binding 52 (src 44) = pending ddl_status, is_active=t — UNCHANGED
```

→ **Không có advance**. Nguyên nhân: worker `PROVISIONING_ORCHESTRATOR_ENABLED` chưa enable (G-7 P0), nên `cdc.cmd.shadow.bind` consumer không pickup → `shadow_binding.ddl_status` mãi `pending`. Boss approve G-7 là **bottleneck duy nhất** chặn AC-6 (`provisioning_state='shadow_active'`).

### D. x2 progress verification (real evidence)

| Iter#3 task | Status iter#4 | Evidence |
|---|---|---|
| **x2.A P0** stage + commit G-10 fix | ✅ **DONE** | cms HEAD `adc6faf` "fix(cms): normalize pk_type 'string' to 'text' at Register (G-10)" by TraiNguyen 10:44:23. Body: import strings + normalizePKType helper + apply trước `db.Create(&entry)` + TestNormalizePKType 7/7. |
| **x2.B P1** ship `09_tasks_solution_flow1_x2_*` | ✅ **DONE** | File 7906B (139 lines) created May 7 10:43. §0 workflow gate disclosure + retroactive justification (CLAUDE.md §2 Bug Fixing Tự chủ Full-loop) + §1 task table x2.1-x2.4 + §2 phương án matrix (3 phương án A/B/C cho G-10) + §4 commits ledger + §6 skills used (14 lines). |
| **x2.C P3** endpoint `POST /api/v1/sources/test` | ⏸ **DEFER** | x2 self-defer trong `09_tasks_solution §2.x2.4` — lý do: G-7+G-8 blocker cao hơn cho true `shadow_active`. Không block Boss output (Flow 1 Path B vẫn 1720 rows). 2h effort không justify cùng iter. |

→ x2 hoàn thành **3/3 task P0/P1** + 1 self-defer chính đáng. Workflow audit trail đầy đủ.

### E. Boss decision matrix iter#4 (UNCHANGED — không có Boss input giữa iter#3 và iter#4)

| # | Decision | Pri | Effort | Risk | Source |
|---|---|---|---|---|---|
| 1 | **G-7**: Approve worker `PROVISIONING_ORCHESTRATOR_ENABLED=1` + restart | **P0** | 30 min | Low | iter#1, **highest-leverage** |
| 2 | **G-8 A4 + A1 cleanup**: Approve status quo Path A + drop test cluster `gpay-postgres-shadow` 5436 | P1 | 30 min | Low | iter#3 |
| 3 | **Phương án Y**: Refactor `admin/source_register.go:92` + backfill 4 phantom rows | P2 | 2h + 15 min | Medium (breaking response change) | iter#1 |
| 4 | Backfill 4 phantom rows state='active' → 'draft' | P2 | 15 min | Low | iter#1 |
| 5 | MariaDB Debezium plugin rebuild | P3 | 2h | Low | iter#1 |
| 6 | A3 dual-cluster effort 6h sau Phase 2 Y | P3 | 6h | Medium-High | iter#3 |

→ **Highest-leverage iter#4 vẫn là #1**. Approve G-7 + restart worker → src44/bind52 sẽ tự advance qua state machine → AC-3..AC-7 PASS.

### F. Updated task plan iter#5

#### x2 (cms-lane) — TODO iter#5

| # | Pri | Task | Effort | Boss approve? |
|---|---|---|---|---|
| **x2.D** | **P1** | **Rebuild + restart cms binary** để pickup commit `adc6faf` (G-10 fix). Hiện binary `/tmp/cdc-cms-service-flow1` start lúc ~10:14 ICT trước commit 10:44 → next Register vẫn fail PG SQLSTATE 42704 nếu operator gửi `pk_type='string'`. | 5 min | NO (lane self-action) |
| **x2.E** | P2 | Standby chờ Boss approve G-7. Sau khi worker restart + state advance, x2 verify từ cms `GET /api/v1/cms/sources/44/provisioning` → `state=shadow_active`. | 10 min | YES (G-7) |
| **x2.F** | P3 | P3.1 endpoint `POST /api/v1/sources/test` (carry-over từ x2.C) | 2h | NO (sau x2.D + x2.E) |

#### max-Brain — TODO iter#5

| # | Pri | Task | Effort |
|---|---|---|---|
| **max.D** | iter#5 verify | Re-verify x2.D rebuild + restart, audit DB delta sau Boss G-7 approve | 5 min |
| **max.E** | escalation | Boss decision item #1 G-7 escalate (highest leverage) | 0 (nudge already in coordination) |
| **max.F** | doc | Append `08_tasks_flow1_e2e_*.md` rev2 reflecting Phương án Z chỉnh sau khi G-7 enable (chờ Boss approve gate) | 15 min |

### G. Cron + self-pacing

Cron `1975934c` recurring 5m → next fire ~5 min. Iter#5 sẽ:
1. Re-verify x2.D rebuild cms (binary timestamp + new register smoke nếu Boss approve G-7).
2. Re-verify Boss approve G-7 / G-8 / Phương án Y.
3. Nếu G-7 approve + worker restart: trigger `cdc.cmd.shadow.bind` flow → verify src44 → `shadow_active` + bind52 ddl_status='created'.
4. Nếu Boss approve Phương án Y: max plan refactor `admin/source_register.go:92` cho worker-lane (max own).

### H. Files iter#4

- **APPEND** (this file): iter#4 section
- **APPEND**: `coordination_max_x2_2026-05-07.md` — acknowledge x2.A + x2.B DONE, queue x2.D rebuild
- **APPEND**: `05_progress.md` — iter#4 entry

— max-Brain (loop iter#4)

---

## Iteration #4 — SUPPLEMENT: x2 fact-check forces G-8 revision (post-§H)

### I. CRITICAL — x2 pause request 11:05 ICT (per `09_tasks_solution §7` + `coordination ⚠️`)

x2 (Muscle) Double-Verification per CLAUDE.md §9 phát hiện max iter#3 G-8 decision doc §1.5 SAI fact:

| Probe iter#4 (max re-verify, real evidence) | Output | Implication |
|---|---|---|
| `docker inspect gpay-cdc-worker --format CDC_SHADOW_DB_URL` | `postgres://gpay_admin:gpay_pass@gpay-postgres-shadow:5432/cdc_shadow?sslmode=disable` | **Worker runtime targets Path B, NOT Path A** (override docker-compose default) |
| `psql 5436 cdc_shadow -c "SELECT count(*), min(_synced_at), max(_synced_at) FROM shadow_payment_bill_service.refund_requests"` | `1720 \| 2026-05-07 03:23:44.52735 \| 2026-05-07 03:23:45.031237` | **Path B 1720 rows _synced_at 1-second window matches iter#0 Flow 1 run** — không phải orphan/legacy |
| `psql 5433 cdc_dw -c "SELECT count(*) FROM shadow_payment_bill_service.refund_requests"` | `0` | **Path A 0 rows = orphan** (cms `ShadowAutomator` writes Path A, worker writes Path B — config drift) |

→ **max iter#3 decision doc §1.5 claim "Path B = test artifact" SAI**. Path B = production shadow data path. A1 cleanup (drop `gpay-postgres-shadow`) sẽ **DESTROY 1720 rows Boss output Flow 1**.

### J. G-8 revised recommendation (max-Brain re-evaluate iter#4)

| Phương án | Iter#3 verdict | Iter#4 revised | Lý do |
|---|---|---|---|
| **A1** drop Path B | Recommended cleanup | ❌ **REJECT** | Destroy 1720 rows Boss output. Mâu thuẫn worker runtime config. |
| **A2** adopt Path B (deprecate Path A) | "Medium effort 4h" | ⚠️ **Candidate** | Match worker runtime. Cms config-local.yml + ShadowAutomator phải align Path B. |
| **A3** hybrid dual-cluster (Path A control plane + Path B shadow data) | "Medium-High effort 6h" | ✅ **RECOMMENDED iter#4** | **Đây chính là runtime hiện tại** — không phải refactor mới, mà là document + align config. cms reads registry/bindings từ Path A cdc_dw (control plane), worker writes shadow tables xuống Path B cdc_shadow (data plane). Phải sửa cms `ShadowAutomator` để route shadow writes qua Path B (hoặc không write Path A nữa). |
| **A4** status quo Path A only | Recommended | ❌ **REJECT** | Khẳng định Path A là single source nhưng runtime evidence cho thấy Path B mới có data. |

→ **max revised recommendation iter#4**: **A3 hybrid dual-cluster** + viết `04_decisions_flow1_path_a_vs_b_REV2_2026-05-07.md` để Boss approve, đồng thời revoke A1 cleanup escalation.

### K. Boss decision matrix iter#4 REVISED (item #2 changed)

| # | Decision | Iter#3 | Iter#4 REVISED |
|---|---|---|---|
| 1 | G-7 worker enable PROVISIONING_ORCHESTRATOR_ENABLED | P0 unchanged | **P0 unchanged — still highest leverage** |
| 2 | **G-8** | "A4 + A1 cleanup" | ⚠️ **REVOKED**. Replace với **A3 hybrid dual-cluster**: align cms `ShadowAutomator` route shadow writes Path B 5436. Effort ~6h, max-lane (cms infra/persistence — actually x2-lane). **HOLD escalation pending REV2 doc.** |
| 3 | Phương án Y refactor admin endpoint | P2 | P2 unchanged (orthogonal G-8) |
| 4 | Backfill 4 phantom rows | P2 | P2 unchanged |
| 5 | MariaDB Debezium plugin | P3 | P3 unchanged |
| 6 | A3 dual-cluster | P3 | **PROMOTED to G-8 recommended** (was deferred phương án) |

### L. Updated task plan iter#5 (revised post-fact-check)

#### x2 (cms-lane) — TODO iter#5

| # | Pri | Task | Effort | Boss approve? |
|---|---|---|---|---|
| **x2.D** | **P1** | Rebuild + restart cms binary để pickup `adc6faf` (G-10) | 5 min | NO (lane self-action) |
| **x2.E** | P2 | Standby Boss approve G-7 | 10 min | YES |
| **x2.F** | P3 | P3.1 endpoint test (carry-over) | 2h | NO |
| **x2.G** *(new iter#5)* | P2 | **Investigate cms `ShadowAutomator` connection target** (currently Path A 5433). Trả lời: liệu A3 cần tách `shadowConnPool` riêng trỏ Path B 5436 hay align toàn bộ cms về Path B? **Read-only investigation, KHÔNG code change**. | 30 min | NO (investigation, không phải plan) |

#### max-Brain — TODO iter#5

| # | Pri | Task | Effort |
|---|---|---|---|
| **max.D** | iter#5 verify | Re-verify x2.D + x2.G investigation result | 5 min |
| **max.E** *(revised)* | doc | Ship `04_decisions_flow1_path_a_vs_b_REV2_2026-05-07.md` reflecting iter#4 fact-check (A3 hybrid recommended, A1 revoked) | 30 min |
| **max.F** | escalation | Boss decision item #1 G-7 (highest leverage) — escalate qua coordination | 0 |

→ **max iter#4 commitment**: ship REV2 decision doc trước khi cron iter#5 fire.

### M. Lesson candidate (CRITICAL — pattern global)

**L-DECISION-DOC-FACT-CHECK-DRIFT** (candidate, sẽ append `lessons.md` sau Boss confirm REV2):
- **Global Pattern**: `[Brain X] viết decision doc dùng [config evidence A] làm authoritative source. [Muscle Y] perform double-verification phát hiện [runtime evidence B] mâu thuẫn A. → Brain phải REVOKE recommendation cũ, ship REV2 doc, không cộc lộn config-comment với runtime-actual.`
- **Đúng**: Decision doc PHẢI cite `docker inspect ENV` + `netstat/lsof active conn` + `actual data row count` — KHÔNG chỉ trust static `*.yml` comment.
- **Áp dụng**: ≥3 dự án CDC/migration/multi-cluster có config drift giữa docker-compose default vs runtime override.

— max-Brain (loop iter#4 SUPPLEMENT — G-8 revision triggered by x2 fact-check)
