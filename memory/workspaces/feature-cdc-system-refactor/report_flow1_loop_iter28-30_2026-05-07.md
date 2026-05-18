# Report Flow 1 LOOP iter#28–#30 — max-Brain Coordination Update

> **Author**: max-Brain | **Date**: 2026-05-07 ICT | **Workspace**: `feature-cdc-system-refactor`
> **Phạm vi**: 3 iter Brain-tier (read-only state verification + x2 progress sync). KHÔNG mutation source code, KHÔNG kill service, KHÔNG commit. Hoàn toàn tuân `L-STANDING-DIRECTIVE-NOT-SPECIFIC-AUTH` + Auto Mode rule #5.

---

## §1 TL;DR

- **Iter#28**: Fresh real-evidence snapshot — services alive, A3 binary `.new` ready, 4 cms file dirty, gate ledger unchanged.
- **Iter#29**: Đọc x2 iter#8 report (last 11:22 ICT) — x2 đã ship A3 hybrid code-side, build PASS, đang chờ Boss `swap`.
- **Iter#30**: File report này (Brain-tier coordination doc).

State Flow 1 không đổi từ iter#15 → iter#30 (12+ iter idle). 1 verb duy nhất unblock = `swap`.

---

## §2 Iter#28 — Real-evidence state probe

```
$ ps -ef | grep cdc-cms-service-flow1
501  64511  1  0  10:18AM  /tmp/cdc-cms-service-flow1     ← pre-A3, 5+hr uptime, NO shadow DB

$ ps -ef | grep cdc-worker-host
501  90006  1  0  11:22AM  /tmp/cdc-worker-host           ← PROVISIONING_ORCHESTRATOR_ENABLED=1

$ curl :8083/health   → {"service":"cdc-cms","status":"ok"}
$ curl :8082/health   → {"service":"cdc-worker","status":"ok"}
$ curl :18083/connectors → ["cdc-pg-source","cdc-mariadb-source","goopay-mongodb-cdc"]
$ curl :18083/connectors/cdc-pg-source/status → "state":"RUNNING" (connector + task)

$ ls -la /tmp/cdc-cms-service-flow1*
-rwxr-xr-x  58022114  May 7 10:18  /tmp/cdc-cms-service-flow1       ← pre-A3 currently running
-rwxr-xr-x  58022194  May 7 11:21  /tmp/cdc-cms-service-flow1.new   ← A3 ready

$ git log --oneline -1 (cms)
adc6faf fix(cms): normalize pk_type 'string' to 'text' at Register (G-10)

$ git status --short (cms scope)
M config/config-local.yml
M config/config.go
M internal/server/server.go
M pkgs/database/postgres.go
```

**Note**: psql probe src 44 G-11 không chạy được (role mismatch + introspection denied) — verification gap. Gate G-11 vẫn assume unchanged từ iter#15.

---

## §3 Iter#29 — x2 progress sync

Đọc `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/report_flow1_loop_iter8_x2_2026-05-07.md` (11:22 ICT, 9962B).

x2 đã hoàn thành **A3 hybrid cms-side**:

| x2 deliverable | Status |
|---|---|
| `pkgs/database/postgres.go` accept `DBConfig` | ✅ refactored |
| `config/config.go` add `ShadowDB` + 9 env binds `shadowDb.*` → `CMS_SHADOW_DB_*` | ✅ |
| `config/config-local.yml` `shadowDb:` block (5436 cdc_shadow, gpay_admin) | ✅ |
| `internal/server/server.go` open 2nd gorm session + ShadowAutomator inject | ✅ graceful fallback `if cfg.ShadowDB.Host != ""` |
| `go build ./...` | ✅ exit 0 |
| `go vet ./...` | ✅ exit 0 |
| `go test ./... -count=1` | ✅ pass (1 flake pre-existing TestNewProvisioningCorrelationID, không liên quan A3) |
| Re-run flake `-count=3` isolated | ✅ pass |
| Boss output 1720 rows persist | ✅ |
| Binary `/tmp/cdc-cms-service-flow1.new` rebuild A3 | ✅ 58022194B 11:21 |
| Smoke run binary mới port 18099 | ❌ Auto-mode safety denied (acceptable) |

x2 iter#8 §7 escalate 3 Boss decisions:
1. **P1 Swap binary** → vẫn pending (= D.1 max ledger)
2. **P0 G-7 worker enable** → x2 nhầm — thực tế **đã enabled** từ iter#9 (worker PID 90006 chạy với `PROVISIONING_ORCHESTRATOR_ENABLED=1`). **Gate này CLOSED.**
3. **P2 Drop 6 Path A schemas** → defer-able (= D.4 max)

**x2 silent post-11:22** — không có x2 report mới trong 1+ giờ. Boss `/loop` re-fires đến iter#30 chỉ cập nhật max-Brain session.

---

## §4 Đối chiếu max ledger ↔ x2 ledger

| Gate | max ledger (iter#15 §D) | x2 iter#8 §7 | Status thực tế |
|---|---|---|---|
| Swap cms binary | D.1 P0 🟥 | #1 P1 | **PENDING — same gate, đợi verb `swap`** |
| Commit A3 (4 file) | D.2 P1 🟧 | implicit (post-build) | PENDING — đợi `commit a3` |
| Ship G-11 (Mongo refund-requests) | D.3 P1 🟧 | n/a | PENDING — defer-able cho P1 happy-path |
| Drop Path A schemas | D.4 P2 | #3 P2 | DEFER-ABLE post-Flow 1 |
| G-7 worker enable | n/a (đã done iter#9) | #2 P0 (NHẦM) | **CLOSED** — worker enabled |

**Net**: 2 ledger agree — `swap` là gate critical duy nhất unblock Flow 1 P1 smoke.

---

## §5 Plan x2 post-swap (max-Brain prepare per directive "plan cho x2")

Khi Boss approve `swap`:

**Pre-swap** (max-Brain dispatch):
```bash
# Backup pre-A3
cp /tmp/cdc-cms-service-flow1 /tmp/cdc-cms-service-flow1.preA3.bak
```

**Swap** (Auto Mode rule #5 gate — Boss verb `swap`):
```bash
kill -TERM 64511 && sleep 2
mv /tmp/cdc-cms-service-flow1.new /tmp/cdc-cms-service-flow1
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service \
  && nohup /tmp/cdc-cms-service-flow1 > /tmp/cdc-cms-service-flow1.log 2>&1 &
sleep 3 && curl -s http://127.0.0.1:8083/health
```

**Post-swap x2 verify** (per x2 iter#8 §7 commitment):
1. `/health=ok` — `{"service":"cdc-cms","status":"ok"}`
2. `grep "PostgreSQL (shadow data plane) connected" /tmp/cdc-cms-service-flow1.log`
3. Smoke G-10 fix `pk_type=string → text` register flow (10 phút)

**Post-swap max dispatch** (P1 Flow 1 happy-path):
- Refer `08_tasks_flow1_e2e_2026-05-07.md` P1.1–P1.13
- Register fresh PG source → shadow_pending → shadow_active (Path B 5436)
- master_pending → master_active
- discover_pending → discover_active
- schedule_enable → state machine end-state `active`
- Verify shadow tables row count > 0 trong cdc_shadow
- Verify Debezium publication + replication slot lit up

---

## §6 Files iter#28–#30 (memory only, KHÔNG source)

```
agent/memory/workspaces/feature-cdc-system-refactor/
└── report_flow1_loop_iter28-30_2026-05-07.md   ← THIS FILE (NEW)
```

**Zero source code touched** (CLAUDE.md §12 respected).
**Zero memory APPEND violation** (NEW file, không edit cũ — §11 respected).
**Zero shared-system mutation** (kill/swap/commit không thực hiện — Auto Mode rule #5 + L-STANDING-DIRECTIVE respected).

---

## §7 Verb dictionary (re-surface)

| Verb | Triggers |
|---|---|
| **`swap`** / `swap đi` | Execute D.1 + dispatch P1.1–P1.13 |
| **`commit a3`** | Execute D.2 git commit 4 cms file |
| **`ship g11`** | Execute D.3 Phương án X (`02_plan_g11_master_bind_hyphen_2026-05-07.md`) |
| **`defer g11`** | Mark D.3 carry, focus D.1/D.2 |
| **`defer flow1, làm <X>`** | Park gates, kickoff alternate |

Generic signals (`/loop`, `tiếp`, `bằng mọi giá`, silence) **KHÔNG** trigger gated action — đã capture lesson `L-STANDING-DIRECTIVE-NOT-SPECIFIC-AUTH`.

---

## §8 Pre-flight check (CLAUDE.md §14)

- §0 Vietnamese ✓ (file mostly tiếng Việt)
- §1 Brain Chairman only ✓ (no code touch)
- §3 Plan & Verify ✓ (real-evidence probes Bash output)
- §11 APPEND-only ✓ (NEW file, not overwrite)
- §12 Brain Code Prohibition ✓ (memory file only)
- §14 Pre-flight ✓ (this section)

---

— max-Brain (iter#30 — coordination consolidation, hard-halt loop continues)
