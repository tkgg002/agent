# Report Flow 1 LOOP iter#46 — D.1 + D.2 closed externally, G-11 partial

> **Author**: max-Brain | **Date**: 2026-05-07 ~14:39 ICT | **Workspace**: `feature-cdc-system-refactor`
> **Type**: Brain-tier real-evidence state delta (zero source mutation, zero commit, zero shared kill).

---

## §1 TL;DR

Giữa iter#41 và iter#46, Boss đã thực thi **D.1 swap** + **D.2 commit A3** ngoài /loop. Brain ledger update:

- ✅ D.1 swap cms binary — CLOSED (PID 43919 13:54 chạy A3, log có `PostgreSQL (shadow data plane) connected`)
- ✅ D.2 commit A3 — CLOSED (HEAD `0eddad0 feat(cms): support hybrid shadow db configuration (A3)`)
- ⏳ D.3 ship G-11 — partial: x2 đã ship naming package + backfill SQL file, nhưng `provisioning_step_handlers.go:295` vẫn leak; worker binary `/tmp/cdc-worker-host` stale May 5

Còn 1 line code + 1 build + 1 swap worker + 1 SQL run = G-11 đóng. Verb cần: `ship g11`.

---

## §2 Real-evidence probes (chạy iter#46)

### §2.1 Service state

```
$ ps -ef | grep -E "cdc-(cms-service-flow1|worker-host)" | grep -v grep
501 43919 1  1:54PM /tmp/cdc-cms-service-flow1   ← A3 binary, started 13:54 (post-swap)
501 90006 1 11:22AM /tmp/cdc-worker-host         ← worker, started 11:22

$ curl :8083/health → {"service":"cdc-cms","status":"ok"}
$ curl :8082/health → {"service":"cdc-worker","status":"ok"}
```

### §2.2 cms log evidence — A3 hybrid LIVE

```
{"level":"info","msg":"PostgreSQL (control plane) connected"}
{"level":"info","msg":"PostgreSQL (shadow data plane) connected","host":"localhost","port":5436,"database":"cdc_shadow"}
```

→ A3 hybrid (Path A 5433 cdc_dw + Path B 5436 cdc_shadow đồng thời mở 2 GORM session) đã LIVE. Đây là gate critical đã đợi từ iter#15.

### §2.3 Git ledger — A3 đã commit

```
$ git log --oneline -1
0eddad0 feat(cms): support hybrid shadow db configuration (A3)
```

Boss đã commit A3 hybrid (4 file: `config/config-local.yml`, `config/config.go`, `internal/server/server.go`, `pkgs/database/postgres.go`). D.2 đóng.

### §2.4 Worker binary — STALE (G-11 chưa fix)

```
$ ls -la /tmp/cdc-worker-host
-rwxr-xr-x 50556514 May 5 09:39 /tmp/cdc-worker-host   ← 2 ngày cũ
```

Naming package `internal/naming/naming.go` modified 13:55 May 7. PID 90006 chạy binary May 5 → **không có** `naming.NormalizeIdentifier`. Lý do shadow_binding id=53 vẫn lưu hyphen.

### §2.5 x2 đã prepare SQL backfill

```
$ ls deployments/sql/cdc/
fix_g11_master_shadow_hyphen_2026-05-07.sql   ← x2 created (untracked)
```

Nội dung: `REPLACE(shadow_table,'-','_')` + `REPLACE(master_table,'-','_')` + reset src 44 `failed → master_pending`. Idempotent (WHERE LIKE '%-%').

### §2.6 Source code leak `provisioning_step_handlers.go:295`

```go
if out.TableName == "" {
    out.TableName = row.SourceObjectName   // ← leaks "refund-requests"
}
```

**Fix cần** (1 line — x2 scope):
```go
out.TableName = naming.NormalizeIdentifier(row.SourceObjectName)
```

---

## §3 Ledger update

| Gate | iter#41 | iter#46 | Action remaining |
|---|---|---|---|
| D.1 swap cms binary | PENDING | ✅ **CLOSED** | — |
| D.2 commit A3 (4 file) | PENDING | ✅ **CLOSED** | — |
| D.3 ship G-11 | scoped | ⏳ partial | 1 line edit + worker rebuild + SQL run |
| D.4 P2 hardening / Phương án Y | defer | defer | — |
| **NEW** P1 smoke happy-path | not opened | **OPENABLE** | Register fresh PG source → drive state machine → `active` |

---

## §4 Plan x2 dispatch (chuẩn bị sẵn cho verb tương lai)

### §4.1 `ship g11` dispatch

```bash
# x2 step 1: edit handler:295
# File: internal/handler/provisioning_step_handlers.go
# Change line 295 from:
#   out.TableName = row.SourceObjectName
# to:
#   out.TableName = naming.NormalizeIdentifier(row.SourceObjectName)

# x2 step 2: rebuild worker
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
go build -o /tmp/cdc-worker-host.new ./cmd/worker

# Boss-gated step 3: swap worker (5s downtime)
cp /tmp/cdc-worker-host /tmp/cdc-worker-host.preG11.bak
kill -TERM 90006 && sleep 2
mv /tmp/cdc-worker-host.new /tmp/cdc-worker-host
PROVISIONING_ORCHESTRATOR_ENABLED=1 nohup /tmp/cdc-worker-host > /tmp/cdc-worker-host.log 2>&1 &
sleep 3 && curl -s http://127.0.0.1:8082/health

# Boss-gated step 4: backfill SQL
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -f /path/to/fix_g11_master_shadow_hyphen_2026-05-07.sql

# x2 step 5: verify
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "
  SELECT id, provisioning_state, last_step_error
    FROM cdc_system.source_object_registry WHERE id = 44;
  SELECT id, master_table FROM cdc_system.master_binding WHERE source_object_id = 44;
  SELECT id, shadow_table FROM cdc_system.shadow_binding WHERE source_object_id = 44;
"
```

Total: 5 phút (1 phút edit + 30s build + 5s swap + 1s SQL + verify).

### §4.2 `smoke flow1` dispatch (post-G-11 hoặc parallel cho PG source)

Tham khảo `08_tasks_flow1_e2e_2026-05-07.md` P1.1-P1.13. Tóm tắt:

1. POST `/api/sources/register` cho PG source mới (ví dụ `goopay_source.payment_orders`).
2. Wait state machine: `draft → shadow_pending → shadow_active → master_pending → master_active → discover_pending → discover_active → schedule_enable → active`.
3. Verify shadow tables row count > 0 trong `cdc_shadow.shadow_<conn>`.
4. Verify Debezium publication `pub_cdc_<conn>` + replication slot `rs_cdc_<conn>` ACTIVE.

Mongo `refund-requests` (src 44) chỉ chạy được sau `ship g11`.

---

## §5 Verb dictionary iter#46

| Verb | Triggers |
|---|---|
| `ship g11` | x2 edit handler:295 + worker rebuild + worker restart + SQL backfill |
| `smoke flow1` | x2 dispatch P1.1-P1.13 (register fresh PG source E2E) |
| `defer g11, smoke flow1` | PG happy-path only, skip Mongo |
| `defer flow1, làm <X>` | Park, kickoff alternate |

---

## §6 Pre-flight check (CLAUDE.md §14)

- §0 Vietnamese ✓
- §1 Brain Chairman only ✓ (zero code edit)
- §3 Plan & Verify ✓ (real-evidence: ps, curl, grep, ls, git log)
- §11 APPEND-only ✓ (file mới, không edit cũ)
- §12 Brain Code Prohibition ✓ (memory only)
- §14 Pre-flight ✓ (this section)

---

— max-Brain (loop iter#46 — Boss đã đóng D.1+D.2 ngoài loop; G-11 còn 1 line away; halt cho `ship g11` hoặc `smoke flow1`)
