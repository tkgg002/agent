# Flow 1 /loop iter#9 — max-Brain audit + plan

> **Author**: max-Brain (Antigravity, plan-tier only per CLAUDE.md §1+§12)
> **Time**: 2026-05-07 ~11:32 ICT
> **Trigger**: Boss `/loop` re-fire iter#9 sau iter#8 x2 A3 implementation DONE.
> **Output kind**: real-evidence audit + plan-only (KHÔNG code per Brain Code Prohibition).
> **Goal**: "bằng mọi giá phải lên đc flow1" — verify state machine advancing post-G-7.

---

## §A — Real-evidence audit (factual, không phải claim)

### §A.1 Service processes (verify alive)

```
ps -ef | grep -E "cdc-cms-service|cdc-worker|cdc-admin-api"
501  21133  /tmp/cdc-admin-api-f3v2          (Tue09AM, 0:27.32)
501  64511  /tmp/cdc-cms-service-flow1       (10:18AM, 0:10.59) ← OLD binary, pre-A3
501  90006  /tmp/cdc-worker-host             (11:22AM, 0:01.54) ← NEW pid (was 23565)
```

→ Worker đã **restart**: PID `23565` (uptime 2d 01:21:32 lúc iter#8) → PID `90006` (uptime 8m58s lúc iter#9). Restart xảy ra **giữa iter#8 và iter#9** (~11:22-11:23 ICT).

### §A.2 Worker env G-7 verification

```
ps eww 90006 | tr ' ' '\n' | grep PROVISIONING
PROVISIONING_ORCHESTRATOR_ENABLED=1
```

→ ✅ **G-7 ENABLED**. Boss đã approve restart-with-G-7 implicitly (Boss `/loop` re-fire = approval signal). State machine từ giờ active advance.

### §A.3 Registry state machine progression

```
SELECT provisioning_state, count(*) FROM cdc_system.source_object_registry WHERE is_active=true GROUP BY provisioning_state;
 provisioning_state | count
--------------------+-------
 running            |     4
 active             |     3
 draft              |     1
 failed             |     1
```

| State | Sources |
|-------|---------|
| active (4-step done) | 37 (f1_burst), 35 (phase_e_smoke), 42 (f3v2_smoke_payment_bills_addtest) |
| running (mid-progression) | 30 (addtest_maria_legacy), 29 (addtest_pg_orders), 26 (e2e_phaseD_auto_v5), 11 (src_local_goopay_source_orders) |
| draft (untouched) | 1 (legacy_1) |
| failed (G-11 NEW) | 44 (src_mongodb_payment_bill_service_refund_requests) |

→ State machine **đang advance** post-G-7. So với iter#8 trước: src 44 trước đây ở state `shadow_pending` (stuck) — giờ đã advance qua `shadow_active` → `master_pending` → **failed at master_bind**.

### §A.4 Shadow binding aggregate

```
SELECT count(*), sum(case when ddl_status='created' then 1 else 0 end) AS created,
                  sum(case when ddl_status='pending' then 1 else 0 end) AS pending,
                  sum(case when ddl_status='failed'  then 1 else 0 end) AS failed
FROM cdc_system.shadow_binding;
 total | created | pending | failed
-------+---------+---------+--------
   27  |   15    |   12    |   0
```

→ **15/27 bindings created** (DDL applied). 12 còn pending — chưa advance qua shadow_bind step.

### §A.5 Boss output 1720 rows persist

```
docker exec gpay-postgres-shadow psql -U gpay_admin -d cdc_shadow \
  -c "SELECT count(*) FROM shadow_payment_bill_service.refund_requests"
1720
```

→ ✅ **No regression**. Boss output Flow 1 iter#0 (Debezium → Kafka → worker shadow handler) intact.

### §A.6 cms binary swap status

```
ls -la /tmp/cdc-cms-service-flow1*
-rwxr-xr-x  58022114 May 7 10:18  /tmp/cdc-cms-service-flow1      ← OLD, pre-A3
-rwxr-xr-x  58022194 May 7 11:21  /tmp/cdc-cms-service-flow1.new  ← NEW, A3 + G-10
```

→ Binary `.new` ready (+80 bytes A3 ShadowDB block + G-10 normalizePKType helper). PID 64511 vẫn chạy OLD binary. **Swap chưa thực hiện** — Boss approval cho swap riêng vẫn outstanding.

### §A.7 cms HEAD git verification

```
cd cdc-system && git log -3 --oneline
adc6faf fix(cms): normalize pk_type 'string' to 'text' at Register (G-10)
0cef7af fix(cms): split multi-statement shadow DDL to unblock Flow 1 Register
b453d36 refactor(cms): Task #19 đợt J ...
```

→ HEAD `adc6faf` unchanged iter#5→iter#9. A3 hybrid changes (pkgs/database/postgres.go + config + server.go) **chưa commit** — vẫn ở working tree (per x2 iter#8 §12.1).

### §A.8 NEW finding G-11: master_bind validateIdent rejects hyphen

src 44 `provisioning_step_log` chuỗi sự kiện đầy đủ:
```
seq=1 step=shadow_bind dispatched draft → shadow_pending  ✅
seq=2 step=shadow_bind worker  shadow_pending → shadow_active  ✅
seq=3 step=master_bind dispatched shadow_active → master_pending  ✅
seq=4 step=master_bind worker  master_pending → failed
       error="invalid master_name: \"refund-requests\""  ❌
```

Source code:
- `cdc-system/centralized-data-service/internal/service/master_ddl_generator.go:47`:
  ```go
  var ddlIdentRe = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)
  ```
- `:62-64`:
  ```go
  if !ddlIdentRe.MatchString(masterName) {
      return nil, fmt.Errorf("invalid master_name: %q", masterName)
  }
  ```

Root cause: src 44 has `source_object_name='refund-requests'` (MongoDB collection conventionally allows hyphens). Worker `master_ddl_generator.Generate(masterName)` validates với strict PG identifier regex (no hyphen). MongoDB hyphenated collection names → blocked at master_bind step.

Note: src 44 có **2 bindings**:
1. `shadow_mongo_payment_bill_default.refund-requests` (hyphen, broken target) — pending
2. `shadow_payment_bill_service.refund_requests` (underscore, the 1720-row table) — pending

Worker chọn binding (1) cho master_bind theo logic gì cần investigate (likely first-binding-wins).

---

## §B — max-Brain plan iter#9 cho x2 (cms-lane)

> **Per CLAUDE.md §1**: Brain plan, Muscle execute. KHÔNG draft `02_plan_*` mới — incorporate vào file này (report-tier max output).
> **Per L-MUSCLE-PLAN-PROHIBITION**: x2 đợi max plan rồi review qua `09_tasks_solution_*` rồi execute.

### §B.1 Task ledger iter#9 (proposed for x2 review)

| # | Pri | Task | Effort | Boss-gated? | Lane |
|---|-----|------|--------|-------------|------|
| **x2.D2** | **P0** | Wait Boss approve swap binary `/tmp/cdc-cms-service-flow1.new` → kill 64511 + mv + nohup. Sau swap verify: `curl :8083/health` + Path B `shadow_payment_bill_service.refund_requests` count = 1720. | 5 min | YES | cms |
| **x2.J** | P1 | Phương án Z smoke (post-swap): register 1 test PG source `flow1_smoke_iter9_<TS>` → verify (a) provisioning advance qua 4 step → state `active`, (b) `shadow_binding.ddl_status='created'`, (c) physical table tạo tại **Path B 5436 cdc_shadow** (chứng minh A3 effective), (d) Path A 5433 cdc_dw KHÔNG có table tương ứng. Report `report_flow1_loop_iter9_x2_*.md`. | 30 min | NO (sau swap) | cms |
| **x2.L** | P2 (NEW G-11) | **Investigate-only** (read-only): grep & document all call sites của `ddlIdentRe.MatchString(masterName)` trong worker; trace src 44 binding selection logic (why binding 1 hyphen được pick thay vì binding 2 underscore); collect evidence vào `09_tasks_solution_flow1_x2_*` §13 (info tier). KHÔNG fix code (worker-lane = max owns code; max sẽ ship `02_plan_g11_*` nếu cần). | 30 min | NO | cms (info-tier read-only worker code OK) |
| **x2.E** | P2 carry-over | Standby Boss approval: A3 commit cms code change → `git add cdc-cms-service/{pkgs/database,config,internal/server}/*` + commit. Wait Boss approve since A3 hybrid affects boot config schema. | 5 min | YES | cms |
| **x2.M** | P3 (NEW post-G-11) | Sau max ship plan G-11 fix → x2 review qua `09_tasks_solution_*`. Defer execute đến Boss approve. | TBD | YES | cms (or worker per max plan) |

### §B.2 Tasks iter#9 cho max-Brain (self)

| # | Task | Effort |
|---|------|--------|
| max.J | APPEND `coordination_max_x2_2026-05-07.md` iter#9 ACK + assignment block. | 5 min |
| max.K | APPEND `05_progress.md` iter#9 immutable entry. | 5 min |
| max.L | Sau x2.L investigation evidence: ship `02_plan_g11_master_bind_hyphen_*.md` (worker-lane plan, code-tier). Phương án candidate: (a) worker normalize masterName (hyphen → underscore at registration), (b) introspect target binding selection precedence, (c) reject hyphenated MongoDB collection names at cms register API + UI guard. | 1h |
| max.M | Nudge Boss: outstanding gates (1) swap binary (2) commit A3 cms code (3) approve drop 6 Path A schemas. | — |

### §B.3 Boss escalation iter#9 (consolidated)

| # | Pri | Decision | Status iter#8 → iter#9 |
|---|-----|----------|------------------------|
| 1 | ✅ | G-7 worker enable | **DONE** (worker PID 90006 PROVISIONING_ORCHESTRATOR_ENABLED=1) |
| 2 | **P0** | Approve swap cms binary `! kill -TERM 64511 && mv /tmp/cdc-cms-service-flow1.new /tmp/cdc-cms-service-flow1 && nohup /tmp/cdc-cms-service-flow1 > /tmp/cdc-cms-service-flow1.log 2>&1 &` | **PENDING** (carry over) |
| 3 | **P1** | Approve commit A3 cms code change (4 files) | **NEW iter#9** (was working tree only) |
| 4 | **P1 NEW G-11** | Approve max ship plan `02_plan_g11_master_bind_hyphen_*` cho worker-lane fix master_name normalize | **NEW iter#9** |
| 5 | P2 | Migration drop 6 Path A `shadow_*` schemas (per REV2 §5.4 + iter#7 §11.4 evidence) | carry over |
| 6 | P2 | Phương án Y refactor `admin/source_register.go:92` | carry over |

→ **Highest leverage iter#9**: #2 (swap) + #3 (commit) → unblock Phương án Z smoke + close A3 audit trail.

### §B.4 Acceptance criteria Flow 1 iter#9

✅ G-7 verified active.
✅ State machine advancing (3 active + 4 running, 0 stuck-at-shadow_pending count).
⏳ Swap binary cms (Boss-gated).
⏳ A3 hybrid code commit (Boss-gated).
⏳ Phương án Z smoke (post-swap).
⏳ G-11 master_bind hyphen fix (max plan first → x2 review → Boss approve → execute).
⏳ Migration drop Path A (Boss-gated).

---

## §C — Lessons reinforced iter#9

- **L-MUSCLE-PLAN-PROHIBITION** áp dụng đúng: x2 iter#1-#8 phần lớn KHÔNG draft `02_plan_*` (chỉ §10 §11 ở `09_tasks_solution_*` info tier). max iter#9 viết plan-tier trực tiếp ở §B này (report-tier max output OK theo conventions).
- **L-DECISION-DOC-FACT-CHECK-DRIFT** confirm: x2 §7+§8 iter#3-iter#4 bắt được drift A1 destructive recommendation. Boss đã honor REV2 (G-7 enable đầu tiên thay vì A1 drop).
- **L-NEW iter#9 candidate** `L-VALIDATEIDENT-MONGODB-HYPHEN-DRIFT`: PG ident regex `^[a-z_][a-z0-9_]{0,62}$` không tương thích MongoDB collection names có hyphen. Khi system multi-engine (PG + MongoDB + MariaDB), validateIdent at PG-DDL boundary phải có normalization layer (hoặc reject + UI guard tại register endpoint). Pattern: `[engine=mongodb has feature F (hyphen-in-name) → DDL adapter D rejects F]` → fix: normalize at boundary B (registration) hoặc transform at B (DDL gen). Defer ship lesson đến khi Boss confirm fix approach.

---

## §D — Files iter#9 (max output)

- `report_flow1_loop_iter9_2026-05-07.md` — this file (workspace).
- `coordination_max_x2_2026-05-07.md` — APPEND iter#9 (next).
- `05_progress.md` — APPEND iter#9 (next).
- `09_tasks_solution_flow1_x2_2026-05-07.md` — x2 own (max KHÔNG modify).

— max-Brain (loop iter#9 — G-7 verified + G-11 surfaced + plan x2 iter#9)
