# Plan Hoàn Thành Flow 1 — v1.0

> **Author**: Brain (Antigravity) | **Date**: 2026-05-07 16:54 ICT
> **References**: `report_flow1_recheck_brain_2026-05-07.md`, `02_plan_flow1_e2e_2026-05-07.md`, `01_requirements_flow1_e2e_2026-05-07.md`
> **Goal**: "bằng mọi giá phải lên đc flow1" — PG source smoke E2E pass 8/8 AC.
> **Lane**: x2 (Muscle) thực thi toàn bộ theo plan này (Brain Code Prohibition §12).

---

## §0 — Real-evidence Summary (2026-05-07 16:54 ICT)

### Hiện trạng thực tế

| Component | Giá trị | Status |
|---|---|---|
| CMS binary | May 7 11:21, PID 43919, port 8083 | ✅ LIVE A3 |
| Worker binary | **May 5 09:39**, PID 90006, port 8082 | ❌ STALE pre-A3 |
| Worker `shadowDb.default` | `localhost:5433/cdc_dw` (Path A) | ❌ SAI — phải là `5436/cdc_shadow` |
| Shadow DB Path B | `localhost:5436/cdc_shadow` — 11 tables, data intact | ✅ READY |
| Worker A3 code | `git diff HEAD`: 5 files, 172 insertions — UNCOMMITTED | ❌ Chưa commit |
| Transmute errors | `SQLSTATE 42P01` relation không tồn tại ở Path A | ❌ FAIL mỗi cycle |
| G-7 | `PROVISIONING_ORCHESTRATOR_ENABLED=1` active | ✅ OK |
| G-11 | master_binding id=37, shadow_binding id=62 (underscore) | ✅ CLOSED |

### Root cause duy nhất của Flow 1 failing

```
Worker binary May 5 → config shadowDb.default = Path A (5433)
→ transmuter.go:316 GetShadowDB() → return 5433/cdc_dw
→ query shadow_goopay_source.orders tại 5433 → table không tồn tại
→ SQLSTATE 42P01 → transmute FAIL mọi schedule
```

**Fix**: Commit A3 code + update config-local.yml shadowDb URL → `5436/cdc_shadow` + rebuild + restart worker.

---

## §1 — PHASE 0: Pre-flight (5 phút, không cần Boss)

**Mục tiêu**: Chuẩn bị môi trường trước khi chạy.

### Task 0.1 — Backup worker log (x2)

```bash
cp /tmp/cdc-worker-host.log /tmp/cdc-worker-host.log.preFlow1.bak
```

### Task 0.2 — Verify docker services healthy (x2)

```bash
# Tất cả container phải HEALTHY trước khi proceed
docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "gpay-postgres-(cdc|shadow|source|dest)|gpay-kafka"
```

Expected: tất cả `Up X hours` / `healthy`.

### Task 0.3 — Verify Kafka Connect PG connector (x2)

```bash
curl -s http://localhost:18083/connectors/cdc-pg-source/status | python3 -m json.tool | grep -E "state|type"
```

Expected: `connector.state = "RUNNING"`, `tasks[0].state = "RUNNING"`.

**DoD Phase 0**: Docker healthy + PG connector RUNNING.

---

## §2 — PHASE 1: Fix G-12 — Worker A3 Hybrid (15 phút, Boss-gated swap)

**Mục tiêu**: Worker binary mới kết nối Path B (`cdc_shadow:5436`). Đây là blocker duy nhất của Flow 1.

### Task 1.1 — Update config-local.yml shadowDb URL (x2)

```yaml
# File: centralized-data-service/config/config-local.yml
# Line: shadowDb: → urls: → default:
# OLD:
shadowDb:
  defaultKey: default
  urls:
    default: postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable

# NEW:
shadowDb:
  defaultKey: default
  urls:
    default: postgres://gpay_admin:gpay_pass@localhost:5436/cdc_shadow?sslmode=disable
```

> ⚠️ config-local.yml là file config, KHÔNG phải Go source code — x2 edit trực tiếp OK.

### Task 1.2 — Commit worker A3 hybrid code (x2)

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service

# Verify staging set
git status -s | grep -v "^\?\?" | head -20

# Stage 5 worker A3 files đã diff
git add \
  config/config.go \
  internal/handler/provisioning_step_handlers.go \
  internal/server/worker_server.go \
  internal/service/connection_manager.go \
  pkgs/database/multi.go

# Naming package (G-11 normalize)
git add internal/naming/ || true

# SQL backfill (G-11)
git add deployments/sql/cdc/fix_g11_master_shadow_hyphen_2026-05-07.sql || true

# Commit
git commit -m "feat(worker): A3 hybrid shadow DB routing + G-11 identifier normalize

- pkgs/database/multi.go: GetDB/GetShadowDB route to Path B (cdc_shadow:5436) when configured
- config/config.go: ShadowDB multi-target config support
- internal/server/worker_server.go: inject shadowDB to ProvisioningStepHandler
- internal/service/connection_manager.go: hybrid shadow resolution
- internal/handler/provisioning_step_handlers.go: swap schemaAdapter to shadowDB
- internal/naming/: NormalizeIdentifier helper (G-11 root cause fix)"

git log --oneline -3
```

### Task 1.3 — Build worker (x2)

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
go build -o /tmp/cdc-worker-host.new ./cmd/worker

# Verify binary fresh
ls -la /tmp/cdc-worker-host.new
# Expected: timestamp > May 7 16:54
```

Build expected time: ~60s.

### Task 1.4 — [BOSS-GATED] Swap worker binary (x2, Boss approve)

> **Boss approve trước khi chạy lệnh này** — kill shared service PID 90006.

```bash
# Backup
cp /tmp/cdc-worker-host /tmp/cdc-worker-host.preG12.bak

# Swap
kill -TERM 90006 && sleep 3

# Verify PID gone
ps -p 90006 && echo "STILL RUNNING — abort" || echo "PID gone OK"

# Move new binary
mv /tmp/cdc-worker-host.new /tmp/cdc-worker-host

# Start với A3 env
PROVISIONING_ORCHESTRATOR_ENABLED=1 \
  nohup /tmp/cdc-worker-host \
  > /tmp/cdc-worker-host.log 2>&1 &

# Save new PID
NEW_PID=$!
echo "New worker PID: $NEW_PID"
sleep 5
```

### Task 1.5 — Verify G-12 closed (x2)

```bash
# 1. Health OK
curl -s http://127.0.0.1:8082/health
# Expected: {"service":"cdc-worker","status":"ok"}

# 2. Startup log có shadow data plane
grep -E "shadow.*5436|cdc_shadow|shadow data plane|PostgreSQL.*shadow" /tmp/cdc-worker-host.log | head -5
# Expected: có ít nhất 1 dòng với 5436 hoặc cdc_shadow

# 3. Không còn SQLSTATE 42P01 sau 30s
sleep 30
grep "SQLSTATE 42P01" /tmp/cdc-worker-host.log | wc -l
# Expected: 0 (hoặc chỉ từ trước khi restart — check timestamp)

# 4. Transmute succeeding
grep "transmute complete" /tmp/cdc-worker-host.log | tail -5
# Expected: scanned > 0 cho các table có data
```

**DoD Phase 1 (G-12 closed)**:
- ✅ Worker health OK
- ✅ Log có `5436` hoặc `cdc_shadow` connection
- ✅ KHÔNG còn `SQLSTATE 42P01` sau restart
- ✅ `git log --oneline -1` hiện commit A3 worker

---

## §3 — PHASE 2: Smoke Flow 1 PG Happy-path (10–15 phút)

**Mục tiêu**: Chứng minh Flow 1 HOẠT ĐỘNG end-to-end với 1 PG source mới.

### Task 2.1 — Tạo source table (x2)

```bash
TS=$(date +%s)
TABLE="flow1_smoke_$TS"
echo "Smoke target: public.$TABLE"

docker exec gpay-postgres-source psql -U src_user -d goopay_source -c "
CREATE TABLE IF NOT EXISTS public.$TABLE (
  id BIGSERIAL PRIMARY KEY,
  user_id INT NOT NULL,
  amount NUMERIC(10,2) NOT NULL,
  status TEXT DEFAULT 'pending',
  notes TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
INSERT INTO public.$TABLE (user_id, amount, status, notes)
SELECT 1000+i, (100+i*10)::NUMERIC(10,2), 'pending', 'flow1-smoke-'||i::text
FROM generate_series(1,5) i;
SELECT count(*) FROM public.$TABLE;"

# Expected: count = 5
```

### Task 2.2 — Register source qua Admin API (x2)

```bash
# Admin API port — check
curl -s http://127.0.0.1:8083/health | head -1
# Nếu 8083 = CMS → dùng 8083 (theo plan 02_plan_flow1_e2e)
# Nếu Admin API khác port → check config

REGISTER_RESPONSE=$(curl -sS -X POST http://localhost:8083/api/sources/register \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(curl -sS -X POST http://localhost:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"admin@goopay.vn","password":"admin123"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])' 2>/dev/null)" \
  -d "{
    \"object_code\":\"flow1_smoke_pg_$TS\",
    \"source_connection_code\":\"postgres_primary\",
    \"source_object_name\":\"$TABLE\",
    \"source_object_type\":\"table\",
    \"primary_key_field\":\"id\",
    \"primary_key_type\":\"bigint\",
    \"timestamp_field\":\"updated_at\",
    \"cdc_mode\":\"cdc\",
    \"sync_engine\":\"debezium\"
  }" 2>&1)

echo "$REGISTER_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$REGISTER_RESPONSE"
echo "$REGISTER_RESPONSE" > /tmp/flow1_smoke_register_$TS.json
```

> **Note**: Nếu cần token auth khác, x2 check `config-local.yml` của CMS/Admin để tìm đúng endpoint và auth scheme.

### Task 2.3 — Monitor state machine (x2)

```bash
# Poll mỗi 15s trong 3 phút
for i in {1..12}; do
  STATE=$(docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -t -c \
    "SELECT provisioning_state FROM cdc_system.source_object_registry WHERE object_code='flow1_smoke_pg_$TS';" | tr -d ' ')
  echo "[$(date +%H:%M:%S)] State: $STATE"
  [ "$STATE" = "shadow_active" ] && echo "✅ SHADOW_ACTIVE REACHED — proceed" && break
  [ "$STATE" = "failed" ] && echo "❌ FAILED — check worker log" && break
  sleep 15
done
```

### Task 2.4 — Verify 8 Acceptance Criteria (x2)

```bash
# AC-1: Register response
cat /tmp/flow1_smoke_register_$TS.json | python3 -m json.tool | grep -E "provisioning_state|steps_completed"

# AC-2: Connector RUNNING
curl -s http://localhost:18083/connectors/cdc-pg-source/status | python3 -m json.tool | grep -E "state"

# AC-3: Shadow schema exists (Path B)
docker exec gpay-postgres-shadow psql -U gpay_admin -d cdc_shadow -c \
  "SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'shadow%';"

# AC-4: Shadow table có CDC + business cols
SRC_ID=$(docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -t -c \
  "SELECT id FROM cdc_system.source_object_registry WHERE object_code='flow1_smoke_pg_$TS';" | tr -d ' ')
SHADOW_SCHEMA=$(docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -t -c \
  "SELECT shadow_schema FROM cdc_system.shadow_binding WHERE source_object_id=$SRC_ID;" | tr -d ' ' | head -1)
docker exec gpay-postgres-shadow psql -U gpay_admin -d cdc_shadow -c "\d $SHADOW_SCHEMA.$TABLE"

# AC-5: ddl_status = created
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT ddl_status FROM cdc_system.shadow_binding WHERE source_object_id=$SRC_ID;"

# AC-6: provisioning_state = shadow_active
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT provisioning_state FROM cdc_system.source_object_registry WHERE object_code='flow1_smoke_pg_$TS';"

# AC-7: Kafka topic có data (chờ tối đa 30s)
TOPIC="cdc.goopay_source.public.$TABLE"
timeout 35 docker exec gpay-kafka kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic $TOPIC \
  --from-beginning \
  --max-messages 1 \
  --timeout-ms 30000 2>&1 | head -5

# AC-8: Shadow row count >= 5
docker exec gpay-postgres-shadow psql -U gpay_admin -d cdc_shadow -c \
  "SELECT count(*) FROM $SHADOW_SCHEMA.\"$TABLE\";"
```

**DoD Phase 2 (Flow 1 GREEN)**:
- ✅ 8/8 AC pass
- ✅ provisioning_state = `shadow_active`
- ✅ ddl_status = `created`
- ✅ Shadow row count >= 5

---

## §4 — PHASE 3: Verify Sources "running" bị stuck (10 phút, không Boss-gated)

**Mục tiêu**: Sau Flow 1 smoke pass, investigate 4 sources còn `running` để hiểu có cần fix không.

### Sources cần check

| ID | object_code | State | Shadow table exists Path B? |
|---|---|---|---|
| 30 | addtest_maria_legacy | running | ✅ `legacy_orders_addtest` |
| 29 | addtest_pg_orders | running | ✅ `orders_addtest` |
| 26 | e2e_phaseD_auto_v5 | running | ✅ `orders_e2e_d_v5` |
| 11 | src_local_goopay_source_orders | — | ✅ `orders` |

### Task 3.1 — Check shadow_binding ddl_status cho 4 sources (x2)

```bash
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "
SELECT sor.id, sor.object_code, sor.provisioning_state, sb.ddl_status, sb.shadow_schema, sb.shadow_table
FROM cdc_system.source_object_registry sor
LEFT JOIN cdc_system.shadow_binding sb ON sb.source_object_id = sor.id
WHERE sor.id IN (30, 29, 26, 11)
ORDER BY sor.id;"
```

**Nếu `ddl_status = 'created'` nhưng state = 'running'**: State machine không advance → cần check worker log sau G-12 fix để xem orchestrator có tự advance không.

**Nếu `ddl_status = 'pending'`**: Shadow bind chưa chạy → cần re-fire `cdc.cmd.shadow.bind`.

### Task 3.2 — Theo dõi state machine sau G-12 fix (x2)

```bash
# Sau worker restart, theo dõi 3 phút
sleep 60
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "
SELECT id, object_code, provisioning_state
FROM cdc_system.source_object_registry
WHERE id IN (30, 29, 26, 11)
ORDER BY id;"
```

Nếu orchestrator G-7 đang enable và state machine advance → sources sẽ tự di chuyển.

---

## §5 — PHASE 4: Report cuối (5 phút)

### Task 4.1 — Tạo report (x2)

Sau khi smoke pass, x2 tạo file:
`agent/memory/workspaces/feature-cdc-system-refactor/report_flow1_completed_2026-05-07.md`

Nội dung bắt buộc:
1. Kết quả 8 AC (pass/fail với evidence psql/curl output)
2. Worker log sample (10 dòng sau restart)
3. Shadow table schema (`\d`) của smoke source
4. Git commit hash A3 worker
5. Thời điểm hoàn thành

### Task 4.2 — Append 05_progress.md (x2)

```bash
echo "| $(date -u +%Y-%m-%dT%H:%M:%SZ) | x2:Antigravity | Flow 1 COMPLETED: G-12 fixed (worker A3 hybrid), smoke PG pass 8/8 AC, shadow_active confirmed |" \
  >> /Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md
```

---

## §6 — Execution Timeline

```
16:54 ICT — Boss approve plan này
           ↓
17:00 ICT — Phase 0: Pre-flight (5 phút)
           ↓
17:05 ICT — Phase 1: G-12 fix
             1.1 Update config-local.yml (2 phút)
             1.2 Commit A3 worker code (3 phút)
             1.3 Build worker (2 phút)
             ↓ [BOSS GATE — approve swap]
             1.4 Swap worker PID (2 phút)
             1.5 Verify G-12 closed (5 phút)
           ↓
17:20 ICT — Phase 2: Smoke Flow 1 PG
             2.1 Create source table (2 phút)
             2.2 Register source (2 phút)
             2.3 Monitor state machine (5–10 phút)
             2.4 Verify 8 AC (5 phút)
           ↓
17:35 ICT — Phase 3: Fix stuck sources (10 phút, optional)
           ↓
17:45 ICT — Phase 4: Report + 05_progress append
           ↓
17:50 ICT — Flow 1 GREEN ✅
```

---

## §7 — Risk & Mitigation

| Risk | Mitigation |
|---|---|
| Build fail (Go compile error) | x2 đọc error output + fix trước khi swap |
| Config sai port 5436 | Verify `docker port gpay-postgres-shadow` trước |
| Admin API auth không biết token | Check `cdc-auth-service` credentials từ `config-local.yml` auth service |
| Smoke object_code conflict | Dùng timestamp suffix `flow1_smoke_pg_$(date +%s)` — unique |
| Kafka topic chưa tạo (AC-7 timeout) | Cho phép AC-7 timeout 30s; nếu fail → kiểm tra Debezium log connector |
| Worker crash sau swap | Backup `/tmp/cdc-worker-host.preG12.bak` → restore + revert config |

---

## §8 — Definition of Done — Flow 1 COMPLETED

```
✅ Worker PID mới running (timestamp > May 7 16:54)
✅ Worker log: "PostgreSQL (shadow data plane)" kết nối 5436/cdc_shadow
✅ KHÔNG còn SQLSTATE 42P01 trong worker log (30s sau restart)
✅ git log --oneline -1: commit A3 worker present
✅ source_object_registry WHERE object_code='flow1_smoke_pg_*' → provisioning_state='shadow_active'
✅ shadow_binding WHERE source_object_id=N → ddl_status='created'
✅ shadow table tồn tại trên Path B (5436/cdc_shadow) với CDC cols
✅ shadow row count >= 5 (5 rows inserted)
✅ report_flow1_completed_2026-05-07.md created
✅ 05_progress.md appended
```

---

## §9 — Pre-flight check (CLAUDE.md §14)

- §0 Vietnamese — plan này viết bằng tiếng Việt ✓
- §1 Brain Chairman — plan-only, KHÔNG code, KHÔNG commit ✓
- §2 Autonomous — full-loop plan, no hand-holding ✓
- §3 Plan & Verify — DoD tại mỗi phase ✓
- §9 Governance — workspace file vật lý (file này) ✓
- §11 APPEND-only — file mới, không overwrite ✓
- §12 Brain Code Prohibition — x2 execute, Brain plan ✓
- §14 Pre-flight ✓

---

— Brain (Antigravity, Flow 1 Completion Plan v1.0 — G-12 root cause confirmed; 4 phases; Boss-gated tại swap worker; Flow 1 DoD 8 AC defined)
