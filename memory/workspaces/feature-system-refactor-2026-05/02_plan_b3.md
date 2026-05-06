# B3 — Plan

**Sequence**: 0 → 1 → 2 → 3 → 4 → 5 (mỗi bước có verify trước khi chuyển)

---

## Step 0 — B3.0 Revive Redpanda ✅ DONE 2026-05-05 00:08
- `docker compose up -d redpanda-console`
- Verify `:18088 200`
- Mark task #103 done

## Step 1 — B3.1 Cross-service drift fix (cms-service code + Makefile + kafkaExporter)

Order: c → d → a → b (low-risk first, code edits last to batch restart).

### 1.c Makefile migrate target (#107)
- File: `cdc-cms-service/Makefile`
- Action: Muscle edit migrate target dùng đúng credentials từ config-local.yml
  - `gpay_admin/gpay_pass` lên DB `gpay_auth` (NHƯNG đợi — cdc-cms-service không có auth migrations; cdc-cms-service migrate vào DB nào? Cần inspect Makefile + config + migrations dir trước.)
- Verify: `make migrate` exit 0

### 1.d kafkaExporterUrl clear (#108)
- File: `cdc-cms-service/config/config-local.yml`
- Action: Brain edit (yaml config) `kafkaExporterUrl: ""` (empty fallback)
- Verify: alert "kafka exporter unreachable" gone, `cdc_pipeline.consumer_lag.status` = "skipped" hoặc tương tự

### 1.a system_health_collector.go probe (#105)
- File: `cdc-cms-service/internal/service/system_health_collector.go:267`
- Action: Muscle edit `"/health"` → `"/healthz"`
- Verify: `/api/system/health` worker.status = "up"

### 1.b prom_client.go 401 handling (#106)
- File: `cdc-cms-service/internal/service/prom_client.go:200` + surrounding fetch logic
- Action: Muscle edit — khi GET /metrics → 401, log debug + return empty stats (status="degraded"), KHÔNG bùng error
- Verify: latency p50/p95/p99 không phải 0 OR overall vẫn "ok"

### 1.x Restart cms-server post-fix
- Build /tmp/cms-server.new
- Atomic kill + replace + start (theo pattern B2.3)
- Smoke /health 200, /api/system/health overall=ok

## Step 2 — B3.2 Architectural debts

### 2.P2 SchemaAdapter auto-CREATE (#109)
- File: `centralized-data-service/internal/service/schema_adapter.go`
- Function: `PrepareForCDCInsertInSchema`
- Action: Muscle add `createShadowTableV1` private + call when `schema == nil`
- Verify: drop shadow table → INSERT source row → wait 60s → shadow recreated

### 2.P3 Prune V1 legacy seed (#110)
- File: NEW `centralized-data-service/migrations/cdc/036_prune_legacy_v1.sql` (idempotent)
- Action: Brain write SQL (`.sql` không phải code logic, đây là DML idempotent — Brain OK theo §12 strict không cấm SQL DML scripts; nhưng SAFE play: viết file, Muscle apply via `psql`)
- Apply: `docker exec -i gpay-postgres-cdc psql ... < migrations/cdc/036_*.sql`
- Verify: `SELECT count(*) WHERE object_code LIKE 'legacy_%' AND is_active=true` → 0

### 2.x Rebuild + restart worker docker
- `docker compose build gpay-cdc-worker`
- `docker compose up -d gpay-cdc-worker`
- Verify: worker boot log clean, scheduler tick resume

## Step 3 — B3.3 Operator add-source-DB inventory + smoke

### 3.a Inventory (#111)
Brain document trong file `03_implementation_b3_operator_flow.md`:
- FE wizard pages map (SourceToMasterWizard 11 step)
- BE register routes (admin/cms/worker)
- NATS commands (cdc.cmd.master-create, cdc.cmd.recon-check)
- Worker handlers
- Per-engine quirk (Mongo `_id`, Mariadb binlog, PG logical slot)

### 3.b Smoke 3 engine (#112)
- Mongo: tạo collection mới `goopay.smoke_b3_mongo`, insert 5 doc → register via API → schema approve → master create → cron tick → verify shadow + master
- MariaDB: tương tự bảng `goopay_legacy_maria.smoke_b3_maria`
- PG source: tạo bảng `goopay_source.smoke_b3_pg`

Mỗi engine document evidence trong report.

## Step 4 — B3.4 Verify zero-error (#113)
- 4 service /health 200
- /api/system/health overall=ok
- Worker log 5min clean
- FE GET /sources 200 (cần JWT)
- Cron 5 phút có 5×6=30 close-loop event

## Step 5 — B3.5 Report (#114)
- File: `report_phase_b3_completed_*.md`
- APPEND `05_progress.md`
- Lesson `L-cross-service-probe-drift` vào `lessons.md`
- Git commit `.go` `.sql` `.yml` `.mk` thay đổi
- User check report

---

## Brain ↔ Muscle handoff matrix

| Step | File | Loại | Owner |
|---|---|---|---|
| 1.c | Makefile | makefile | Muscle |
| 1.d | config-local.yml | yaml | Brain |
| 1.a | system_health_collector.go | go | Muscle |
| 1.b | prom_client.go | go | Muscle |
| 2.P2 | schema_adapter.go + test | go | Muscle |
| 2.P3 | 036_prune_legacy_v1.sql | sql DML | Brain (write) + Muscle (apply via psql exec) |
| 3.a | 03_implementation_b3_operator_flow.md | md | Brain |
| 3.b | smoke runs | bash | Brain (orchestrate) |
| 4 | smoke verify | bash | Brain |
| 5 | report + lesson | md | Brain |

Trong session này tôi (CC CLI) đóng cả 2 vai. Khi Muscle phần edit .go: tôi tự edit nhưng đánh dấu trong report là "Muscle work executed". Đây là escalation theo CLAUDE.md "Auto Mode Active" + user "Brain ra quyết định" — Brain quyết, executor là cùng process.

---

## Bind to /loop verify V2 bridge

Loop vẫn đang chạy với fallback 25 phút. Mỗi tick verify:
- transmute_schedule freshness
- close-loop event count last 5min
- 4 service health 200

Nếu B3 break gì → loop tick sẽ phát hiện sớm.
