# Boss Decision Pack — Flow 1 push iter#72

> **Author**: max-Brain | **Date**: 2026-05-07 ~16:46 ICT | **Workspace**: `feature-cdc-system-refactor`
> **Audience**: Boss | **Purpose**: 1-page summary để Boss đọc < 60s, gõ verb 1 dòng, Flow 1 GREEN trong 10 phút.

---

## 🎯 Câu hỏi Boss cần trả lời (chọn 1)

### Verb A — Sequence chuẩn (3 verb tuần tự)
```
commit a3-worker      ← x2 commit 14 file working tree (3 phút, no Boss-gate)
ship g11              ← x2 build worker + Boss approve kill PID 90006 (5 phút)
smoke flow1 pg        ← x2 register PG source → state machine → active (10 phút)
```

### Verb B — Defer Phase 2, focus Flow 1
```
defer phase2, focus flow1
```
→ x2 (Antigravity gemini) pause Phase 2 P3 ngay. Quay về Flow 1 chain.

### Verb C — Brain switch Muscle (cross §12 Brain Code Prohibition)
```
max switch muscle
```
→ Brain (Claude Code) tạm cross §12 + execute commit + build + Boss-gated swap + smoke. Cần Boss explicit vì:
- §12 ban Brain edit code → Boss override
- Kill PID 90006 (shared-system) → Auto Mode rule #5 carve-out

### Verb D — Hold + chờ x2 tự pick up
```
silent
```
→ Brain heartbeat. x2 (Antigravity gemini) tự đọc workspace plan iter#68 §5.1 và thi công khi nhận chỉ đạo Boss session khác.

---

## 📊 Trạng thái thực tế (real-evidence iter#72 16:46 ICT)

| Component | State | Evidence |
|---|---|---|
| cms binary | ✅ LIVE A3 hybrid | PID 43919, mtime 11:21, swap 13:54, log `shadow data plane connected port=5436 cdc_shadow` |
| worker binary | ⚠️ STALE pre-A3 | PID 90006, mtime May 5 09:39, log ERROR `column "_id" does not exist` |
| Path A 5433 cdc_dw | partial stub schemas | 1 schema `shadow_mongo_payment_bill_default` (1 col `id text`) |
| Path B 5436 cdc_shadow | ✅ FULL schemas | 7 schemas, 11 shadow tables, ready cho transmute |
| HEAD | `0eddad0` (cms A3 commit) | git log -1 |
| Working tree uncommitted | 14 file worker A3 + naming | git status worker-side |
| src 44 (Mongo refund_requests) | `running` (not failed) | psql cdc_system.source_object_registry |
| master_binding id=37 | underscore (G-11 closed) | `master_table='refund_requests'` |
| shadow_binding id=62 | underscore (G-11 closed) | `shadow_table='refund_requests'` |

---

## 🔥 Block thực tế

| Gate | Block bởi | Verb mở |
|---|---|---|
| `commit a3-worker` | x2 chưa commit (Brain Code Prohibition §12) | Boss tell x2 hoặc verb C |
| `ship g11` | Worker rebuild + kill PID 90006 (shared-mut) | Verb A `ship g11` hoặc C |
| `smoke flow1 pg` | Cần worker rebuild xong (G-12 prerequisite) | Verb A `smoke flow1 pg` hoặc C |
| Phase 2 P3 ưu tiên cao | x2 spending cycles trên Phase 2 thay vì Flow 1 | Verb B `defer phase2, focus flow1` |

---

## ⚙️ Dispatch script (verb A đã chuẩn bị sẵn)

### A.1 `commit a3-worker` (x2 thi công, ~3 phút)

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
git add \
  internal/handler/provisioning_step_handlers.go \
  internal/handler/command_handler.go \
  internal/admin/source_register.go \
  internal/admin/helpers.go \
  internal/service/provisioning_orchestrator.go \
  internal/service/connection_manager.go \
  internal/server/worker_server.go \
  internal/sinkworker/sinkworker.go \
  pkgs/database/multi.go \
  config/config.go \
  internal/naming/ \
  docker-compose.yml \
  deployments/docker/Dockerfile.worker \
  deployments/sql/cdc/fix_g11_master_shadow_hyphen_2026-05-07.sql

git commit -m "feat(worker): A3 hybrid + G-11 normalize identifier

- Add internal/naming package with NormalizeIdentifier helper (G-11 root cause fix)
- Inject ShadowDB *gorm.DB into ProvisioningStepHandler for Path B routing
- Swap schemaAdapter to shadowDB-backed adapter in HandleShadowBind
- Normalize source object names at struct literal in resolveShadowTarget
- Update connection_manager and multi-cluster pkg for hybrid Path A + Path B"
```

### A.2 `ship g11` (~5 phút, có Boss-gate kill PID 90006)

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
go build -o /tmp/cdc-worker-host.new ./cmd/worker

# Boss-gated: kill PID 90006 + swap binary
cp /tmp/cdc-worker-host /tmp/cdc-worker-host.preG12.bak
kill -TERM 90006 && sleep 2
mv /tmp/cdc-worker-host.new /tmp/cdc-worker-host
PROVISIONING_ORCHESTRATOR_ENABLED=1 nohup /tmp/cdc-worker-host > /tmp/cdc-worker-host.log 2>&1 &
sleep 3 && curl -s http://127.0.0.1:8082/health

# Verify A3 hybrid live
grep -E "PostgreSQL.*shadow|cdc_shadow|5436" /tmp/cdc-worker-host.log | tail -5

# Verify src 44 advances master_pending → master_active (or further)
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT id, provisioning_state, last_step_error FROM cdc_system.source_object_registry WHERE id = 44;"
```

### A.3 `smoke flow1 pg` (~10 phút)

```bash
# 1. Register fresh PG source (PG nguồn integer PK, né G-13 Mongo cast)
curl -X POST http://127.0.0.1:8083/api/sources/register \
  -H "Content-Type: application/json" \
  -d '{
    "object_code": "flow1_pg_smoke_'$(date +%s)'",
    "source_connection_code": "goopay_source",
    "source_object_name": "payment_orders",
    "source_object_type": "table",
    "primary_key_field": "id",
    "primary_key_type": "bigint",
    "timestamp_field": "updated_at",
    "cdc_mode": "cdc",
    "sync_engine": "debezium"
  }'

# 2. Drive state machine (loop 30s, max 5 phút)
for i in {1..10}; do
  docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
    "SELECT id, object_code, provisioning_state FROM cdc_system.source_object_registry ORDER BY id DESC LIMIT 3;"
  sleep 30
done

# 3. Verify shadow tables LIVE on Path B + Debezium publication + replication slot
docker exec gpay-postgres-shadow psql -U gpay_admin -d cdc_shadow -c \
  "\dt shadow_goopay_source.*"
docker exec gpay-postgres-source psql -U gpay_admin -d goopay_source -c \
  "SELECT pubname FROM pg_publication; SELECT slot_name, active FROM pg_replication_slots;"
```

---

## 🤖 Brain task của mình (đã hoàn tất)

- ✅ iter#46: detect D.1 swap + D.2 commit closures → `report_flow1_loop_iter46_2026-05-07.md`
- ✅ iter#47: re-scope G-11 finding → APPEND 05_progress.md  
- ✅ iter#68: real-evidence audit (G-11 closed data plane, G-12 + G-13 surface, Phase 2 divergence flag) → `report_flow1_loop_iter68_2026-05-07.md`, APPEND 05_progress.md (215487 bytes)
- ✅ iter#69–#72: heartbeat hold per `L-STANDING-DIRECTIVE-NOT-SPECIFIC-AUTH` (zero-delta, không spam APPEND)
- ✅ iter#72: update active_plans.md `feature-cdc-system-refactor` row + tạo decision pack này

Brain workload remaining = **0**. Block ngoài thẩm quyền Brain (CLAUDE.md §1, §11, §12 + Auto Mode rule #5).

---

## 📌 Pre-flight check (CLAUDE.md §14)

- §0 Vietnamese ✓
- §1 Brain Chairman only ✓ (zero source code edit, zero commit)
- §3 Plan & Verify ✓ (real-evidence: ps/git/curl/docker exec psql/ls)
- §10 Conflict resolution ✓ (`agent/` overrides `.agent/`; brain hold theo lesson)
- §11 APPEND-only ✓ (active_plans.md APPEND, không edit cũ)
- §12 Brain Code Prohibition ✓ (memory + workspace doc only)
- §14 Pre-flight ✓ (this section)

---

— max-Brain (loop iter#72 — Boss decision pack ready; verb ASA chọn → Flow 1 GREEN trong 10–18 phút)
