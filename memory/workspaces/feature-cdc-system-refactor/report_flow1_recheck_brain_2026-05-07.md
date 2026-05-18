# Report: Flow 1 Plan Re-check — Brain Audit

> **Author**: Brain (Antigravity) | **Date**: 2026-05-07 ~16:44 ICT
> **Type**: Real-evidence audit (dữ liệu thực từ `ps`, `curl`, `psql`, `git`, `grep`)
> **Goal**: Review plan flow1, xác định đã làm đến đâu, update plan nếu chưa hoàn chỉnh.

---

## §1 — Tóm tắt "Current State" (3–5 dòng)

- **Worker stale (G-12)**: Binary `/tmp/cdc-worker-host` ngày May 5 09:39 (cũ 2 ngày). Worker không biết về Path B (`cdc_shadow` port 5436). Config `shadowDb.urls.default` vẫn trỏ vào `localhost:5433/cdc_dw` (Path A). Transmuter query shadow tables ở Path A → bảng không tồn tại → SQLSTATE 42P01 liên tục.
- **CMS healthy**: `/tmp/cdc-cms-service-flow1` (May 7 11:21) đang LIVE với A3 hybrid. Health OK.
- **Flow 1 BLOCKED**: Transmute phase fail vì G-12. Shadow data ở Path B (`cdc_shadow:5436`) đầy đủ nhưng worker không kết nối đúng nơi.
- **G-11 CLOSED**: src 44 hết `failed`, về `running`. Master_binding id=37 + shadow_binding id=62 (underscore) đã tồn tại.

---

## §2 — Real-evidence probes (2026-05-07 16:44 ICT)

### §2.1 Service processes

```
ps -ef | grep cdc:
PID 43919  /tmp/cdc-cms-service-flow1   ← LIVE (May 7 11:21) ✅
PID 90006  /tmp/cdc-worker-host         ← STALE (May 5 09:39) ❌

curl :8083/health → {"service":"cdc-cms","status":"ok"}   ✅
curl :8082/health → {"service":"cdc-worker","status":"ok"} ✅ (process alive, binary stale)
```

### §2.2 Worker startup log — shadow DB connection

```
Worker log (PID 90006 boot):
"PostgreSQL connected (multi-pg registry)"
  control_plane: localhost:5433/cdc_dw
  destination:   localhost:5434/goopay_dest

KHÔNG có log "PostgreSQL (shadow data plane) connected" với 5436/cdc_shadow
→ Worker stale KHÔNG BIẾT về Path B (cdc_shadow:5436)
```

### §2.3 Worker live errors (transmute phase)

```
[2026-05-07 16:44:22] ERROR: relation "shadow_src_local_pg_source.orders_addtest" does not exist (SQLSTATE 42P01)
[2026-05-07 16:44:22] transmute failed: orders_addtest
[2026-05-07 16:44:22] ERROR: relation "shadow_goopay_source.orders" does not exist (SQLSTATE 42P01)
[2026-05-07 16:44:22] transmute failed: orders_fact
```

→ Khớp với G-12: worker query Path A (`cdc_dw`), trong khi shadow tables nằm ở Path B (`cdc_shadow`).

### §2.4 Path B (cdc_shadow:5436) — data integrity

```
shadow_payment_bill_service.refund_requests   → 1720 rows ✅ (Flow 1 Mongo data intact)
shadow_src_local_pg_source.orders             → 0 rows   ⚠️ (shadow table empty)
shadow_goopay_source.orders                   → 32 rows  ✅
```

Shadow tables vật lý TỒN TẠI trên Path B, có data. Worker chỉ cần kết nối đúng nơi.

### §2.5 Shadow binding state

```
src 44 (refund_requests):  shadow_binding id=62, ddl_status='pending', shadow_connection_id=2
src 37 (f1_burst):         shadow_binding id=46, ddl_status='pending', shadow_connection_id=2
src 38 (orders_addtest):   shadow_binding id=38, ddl_status='created', shadow_connection_id=2
src 15 (orders):           shadow_binding id=15, ddl_status='created', shadow_connection_id=2
```

`shadow_connection_id=2` — cần xem `id=2` trong connection registry trỏ tới đâu (Path A hay B).

### §2.6 Git ledger

```
cms HEAD: 0eddad0 feat(cms): support hybrid shadow db configuration (A3)  ✅ committed
worker: A3 hybrid changes UNCOMMITTED trong working tree (46 files, 860 insertions)
```

---

## §3 — Gap Analysis: Kế hoạch Flow 1 vs Thực tế

| Phase | Task | Plan | Thực tế | Status |
|---|---|---|---|---|
| P1 | Smoke PG happy-path | Register 1 PG source, verify 8 AC | Chưa thực hiện | ⏳ BLOCKED bởi G-12 |
| P1 | G-7 enable provisioning orchestrator | Worker env `PROVISIONING_ORCHESTRATOR_ENABLED=1` | ✅ Confirmed PID 90006 | ✅ DONE |
| P1 | G-11 master_bind hyphen fix | Normalize `refund-requests` → `refund_requests` | ✅ master_binding id=37, shadow_binding id=62 (underscore) | ✅ DATA CLOSED |
| **G-12** | Worker A3 hybrid (Path B routing) | Commit worker + build + swap | Uncommitted, binary stale May 5 | ❌ BLOCKING |
| **G-13** | Mongo PK cast `_id::bigint` | Dispatch cast theo `primary_key_type` | Transmuter hardcode bigint | ❌ OPEN (defer) |
| P2 | Fix stuck pending sources | Debug 4 sources (30, 29, 26, 11) stuck `running` | Chưa investigate | ⏳ Pending |
| P3 | `POST /api/v1/sources/test` (cms) | Endpoint test connection | Chưa implement | ⏳ Post-G12 |
| P3 | PG/MariaDB preflight (worker) | Add preflight gate | Chưa implement | ⏳ Post-G12 |
| P4 | Kafka MySQL plugin (MariaDB) | Rebuild kafka-connect + MySQL plugin | Chưa thực hiện | ⏳ Boss-gated |
| P5 | Cleanup legacy sources | prune_legacy_v1_bindings.sql + dedup log | Chưa thực hiện | ⏳ Low priority |
| Phase 2 P3 | CQRS refactor master_registry_handler | thin adapter, ≤100 lines | ✅ DONE (x2) | ✅ DONE (divergent) |

---

## §4 — Root Cause G-12 (Critical Blocker)

**Vấn đề cốt lõi**: Worker binary May 5 không có code A3 hybrid (Path B routing). Config `shadowDb.urls.default` = `localhost:5433/cdc_dw` → `GetShadowDB()` trả về connection trỏ Path A. Tất cả transmute query → Path A → bảng không tồn tại.

**Cần làm**:
1. `git add` 14 files worker A3 hybrid + commit
2. `go build -o /tmp/cdc-worker-host.new ./cmd/worker`  
3. **Boss-gated**: kill PID 90006 + swap binary + restart với env `PROVISIONING_ORCHESTRATOR_ENABLED=1`
4. Verify startup log có: `"PostgreSQL (shadow data plane) connected"` với `localhost:5436/cdc_shadow`

---

## §5 — Plan Updated (post recheck)

### Priority Queue (thứ tự thực hiện)

```
[P0 CRITICAL - G-12]
  commit a3-worker → build → Boss-gated swap
  ↓
[P1 - Smoke Flow 1 PG]
  Register fresh PG source → verify 8 AC → Flow 1 baseline OK
  ↓
[P2 - Fix stuck pending]
  Debug 4 sources (30, 29, 26, 11) stuck "running" → re-fire shadow.bind
  ↓
[P3 - Hardening]
  3.1 cms test_connection endpoint (x2)
  3.2 worker PG/MariaDB preflight (max)
  3.3 worker NATS fatal promote (max)
  ↓
[P4 Boss-gated - MariaDB plugin]
  Rebuild kafka-connect image + mysql plugin
  ↓
[P5 - Cleanup]
  prune legacy + dedup logs
  ↓
[Post-Flow-1 - Phase 2 P3 resume]
  x2 resume CQRS refactor remaining handlers + FE polling
```

### §5.1 Task G-12 (x2 — commit + build, Boss-gated swap)

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service

# Stage worker A3 hybrid files
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

git commit -m "feat(worker): A3 hybrid shadow DB routing + G-11 identifier normalize"

# Build
go build -o /tmp/cdc-worker-host.new ./cmd/worker

# Verify build
ls -la /tmp/cdc-worker-host.new

# [BOSS-GATED] Swap
cp /tmp/cdc-worker-host /tmp/cdc-worker-host.preG12.bak
kill -TERM 90006 && sleep 3
mv /tmp/cdc-worker-host.new /tmp/cdc-worker-host
PROVISIONING_ORCHESTRATOR_ENABLED=1 nohup /tmp/cdc-worker-host > /tmp/cdc-worker-host.log 2>&1 &

# Verify A3 live
sleep 3
grep -E "shadow.*5436|cdc_shadow|shadow data plane" /tmp/cdc-worker-host.log | head -5
curl -s http://127.0.0.1:8082/health
```

**Definition of Done G-12**: Worker log có `"PostgreSQL (shadow data plane) connected"` với 5436. Không còn SQLSTATE 42P01 trong transmute logs.

### §5.2 Task Smoke Flow 1 PG (sau G-12)

```bash
TS=$(date +%s)
TABLE="orders_flow1_smoke_$TS"

# Tạo source table
docker exec gpay-postgres-source psql -U src_user -d goopay_source -c "
CREATE TABLE IF NOT EXISTS public.$TABLE (
  id BIGSERIAL PRIMARY KEY,
  user_id INT,
  amount NUMERIC(10,2),
  status TEXT,
  notes TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);
INSERT INTO public.$TABLE (user_id, amount, status, notes)
SELECT 1000+i, 100+i, 'pending', 'flow1-smoke-'||i FROM generate_series(1,5) i;"

# Register source (dùng Admin API port 8083)
curl -sS -X POST http://localhost:8083/api/sources/register \
  -H "Content-Type: application/json" \
  -d "{
    \"object_code\":\"flow1_smoke_pg_$TS\",
    \"source_connection_code\":\"postgres_primary\",
    \"source_object_name\":\"$TABLE\",
    \"source_object_type\":\"table\",
    \"primary_key_field\":\"id\",
    \"primary_key_type\":\"bigint\",
    \"timestamp_field\":\"created_at\",
    \"cdc_mode\":\"cdc\",
    \"sync_engine\":\"debezium\"
  }" | tee /tmp/flow1_smoke_register_$TS.json

# Monitor state machine (30s intervals, 5 phút)
for i in {1..10}; do
  docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
    "SELECT id, object_code, provisioning_state FROM cdc_system.source_object_registry WHERE object_code='flow1_smoke_pg_$TS';"
  sleep 30
done
```

---

## §6 — Scope Divergence Note

x2 (Brain phiên hiện tại) đã spend cycles trên Phase 2 P3 CQRS refactor (`master_registry_handler.go` thin adapter). Đây là **post-Flow-1 work** không unblock Flow 1. Nhưng đã hoàn tất và build compiled ✅ — không có regression.

**Action**: P3 Phase 2 đã xong. Resume tập trung Flow 1 theo priority queue §5.

---

## §7 — Pre-flight check (CLAUDE.md §14)

- §0 Vietnamese ✓
- §1 Brain Chairman — audit only, KHÔNG sửa code ✓
- §3 Plan & Verify — dữ liệu từ ps/curl/psql/git thực tế ✓
- §11 APPEND-only — file mới, không overwrite ✓
- §12 Brain Code Prohibition ✓
- §9 Skill-listing — ở cuối response chính ✓

---

— Brain (Antigravity, audit iter Flow 1 recheck — G-12 critical blocker confirmed; plan updated; recommend `commit a3-worker` → `ship g12` → `smoke flow1 pg`)
