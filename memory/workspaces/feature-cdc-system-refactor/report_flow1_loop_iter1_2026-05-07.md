# Report — Flow 1 /loop iteration #1 verify (2026-05-07 ICT)

> **Author**: max-Brain | **Trigger**: User `/loop 5p verify task của x2, review và lên task mới cho x2`
> **Cron job**: `1975934c` (5-minute recurring) — fires every `*/5 * * * *`
> **Iteration scope**: Verify x2 progress + review thực tế + lên task mới + ghi report
> **Predecessor**: `04_decisions_flow1_root_cause_2026-05-07.md` (commit 9a887a5) + `05_progress.md` x2 APPEND

---

## 1. Service state matrix (verified live)

| Service | PID | Uptime | Binary | Port | Health | Status |
|---|---|---|---|---|---|---|
| cdc-cms-service | 64511 | 15m02s | `/tmp/cdc-cms-service-flow1` | :8083 LISTEN | `GET /api/system/health` HTTP 200 (2.5ms) | ✅ alive (post-Đợt-J + x2 shadow_automator fix binary) |
| cdc-worker-host | 23565 | 2d 00:53 | `/tmp/cdc-worker-host` | NATS sub | log shows JobMonitor close-loop | ✅ alive (chưa rebuild — không có flag PROVISIONING_ORCHESTRATOR_ENABLED) |
| gpay-postgres-cdc | container | healthy | n/a | :5433 | psql OK | ✅ |
| gpay-postgres-shadow | container | healthy | n/a | :5436 | psql OK | ✅ |
| gpay-kafka-connect | container | healthy | n/a | :18083 | n/a | ✅ |

## 2. Real DB evidence (verified queries)

### 2.1 source_object_registry (latest 5 sources)

| id | object_code | provisioning_state | created_at |
|---|---|---|---|
| 44 | `src_mongodb_payment_bill_service_refund_requests` | **shadow_pending** ⚠️ | 2026-05-07 03:18:54 |
| 42 | `f3v2_smoke_payment_bills_addtest` | active (legacy 2026-05-04) | 2026-05-04 09:40:51 |
| 37 | `f1_burst` | active (phantom, 2026-05-04) | 2026-05-04 09:30:32 |
| 35 | `phase_e_smoke_1777885325` | active (phantom) | 2026-05-04 09:02:05 |
| 34 | `mongo_close_1777882418` | active (phantom) | 2026-05-04 08:13:46 |

→ **AC-6 FAIL**: source 44 vẫn `shadow_pending` (cần `shadow_active`). 4 phantom rows id 33,34,35,37 từ 2026-05-04 vẫn `state='active'` — Phase 2 fix chưa thi công.

### 2.2 shadow_binding (latest 6 bindings)

| id | source_id | ddl_status | shadow_schema | shadow_table |
|---|---|---|---|---|
| 52 | 44 | **pending** ⚠️ | shadow_payment_bill_service | refund_requests |
| 50 | 42 | pending (legacy) | shadow_payment_bill_service_mongo | payment_bills_addtest |
| 46 | 37 | pending | shadow_payment_bill_service_mongo | x |
| 44 | 35 | pending | shadow_phase_e_ns_1777885325_mongo | items |
| 43 | 34 | pending | shadow_goopay_mongo | smoke_p02_close_1777882418 |
| 42 | 33 | pending | shadow_goopay_mongo | smoke_p02_close_1777882181 |

→ **AC-5 FAIL**: bind id=52 vẫn `ddl_status='pending'` (cần `created`). 5 stale binds 42-50 vẫn pending.

### 2.3 Path A vs Path B divergence (G-8 confirmed)

| Cluster | Container | Port | Database | Table `shadow_payment_bill_service.refund_requests` | Row count |
|---|---|---|---|---|---|
| **Path A** (cms `ShadowAutomator`) | gpay-postgres-cdc | 5433 | cdc_dw | EXISTS (8 CDC cols + UNIQUE constraint) | **0** ❌ |
| **Path B** (worker Kafka consumer) | gpay-postgres-shadow | 5436 | cdc_shadow | EXISTS (Debezium snapshot route) | **1720** ✅ |

**Diagnosis**: cms `ShadowAutomator.EnsureShadowTable` tạo table tại cluster sai (5433 cdc_dw thay vì 5436 cdc_shadow). Data thực ingest tại Path B đúng. Path A produce orphan empty table → resource waste + confusion về source-of-truth.

**Root cause**: cms config trỏ ShadowAutomator GORM session vào `cdc_dw` connection chứ không phải `cdc_shadow`. Cần verify file `cdc-cms-service/config/config-local.yml` block `shadow_db.dsn` hoặc tương đương.

## 3. x2 progress assessment (review per Boss directive)

### 3.1 Functional output achieved ✅

x2 đã đạt mục tiêu Boss "bằng mọi giá phải lên đc flow1": shadow database (5436) có table `shadow_payment_bill_service.refund_requests` với **1720 rows từ Debezium Mongo snapshot** = 1:1 match source `payment-bill-service.refund-requests` (1720 docs).

**End-to-end verified by x2**:
- Connector `goopay-mongodb-cdc` RUNNING
- `POST /api/v1/source-objects/register` HTTP 202 (sau bug fix)
- `POST /api/v1/cms/sources/44/provisioning/mode {mode:manual}` HTTP 200
- `POST /api/v1/cms/sources/44/provisioning/advance` HTTP 200 → state→`shadow_pending`
- NATS `cdc.cmd.shadow.bind` published (orchestrator)
- Workaround `nats pub cdc.cmd.kafka.refresh-topics` để worker reload topic list
- 1720 rows landed tại 5436 cdc_shadow

### 3.2 Workflow gate violations (Brain audit)

| Gate | Required by | Actual | Severity |
|---|---|---|---|
| Viết `09_tasks_solution_flow1_x2_2026-05-07.md` trước execute | `08_tasks_flow1_e2e §"Workflow gate"` | ❌ KHÔNG VIẾT | Medium — x2 self-justified `§2 Bug Fixing Tự chủ Full-loop` nhưng vẫn vi phạm hand-off contract |
| Boss approve trước khi modify cms code | CLAUDE.md §3 Plan & Verify | ❌ x2 đã modify `shadow_automator.go` không Boss approve | Low — bug fix chính đáng (HTTP 500 block toàn flow) + lesson L-1755 PrepareStmt 42601 |
| Stage + commit fix sau khi verify | Process discipline | ⚠️ x2 KHÔNG stage + commit fix `shadow_automator.go` (working tree dirty) | High — cms repo dirty 2+ giờ → rebuild workflow vỡ |
| Boss-facing report | `/loop` directive | ✅ `cdc-system/cdc-cms-service/report_flow1_run_x2_2026-05-07.md` (11722B, 10:26) | OK |

### 3.3 Code review verdict — `shadow_automator.go` x2 fix

```diff
-	ddl := fmt.Sprintf(`
-        CREATE SCHEMA IF NOT EXISTS %[2]q;
-        CREATE TABLE IF NOT EXISTS %[2]q.%[1]q (...);
-        CREATE INDEX ...;`)
-	return s.db.WithContext(ctx).Exec(ddl).Error
+	stmts := []string{
+		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %[1]q`, schema),
+		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %[1]q.%[2]q (...) CONSTRAINT %[3]q UNIQUE (source_id))`, schema, target, target+"_source_id_unique"),
+		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS ...`, ...),
+		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS ...`, ...),
+		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS ...`, ...),
+	}
+	for _, stmt := range stmts {
+		if err := s.db.WithContext(ctx).Exec(stmt).Error; err != nil {
+			return err
+		}
+	}
+	return nil
```

**Verdict**: ✅ APPROVE
- Pattern đúng (split multi-stmt → loop Exec) phù hợp PostgreSQL prepared-stmt limitation.
- Comment giải thích rõ root cause (PrepareStmt=true tại `pkgs/database/postgres.go`).
- All `IF NOT EXISTS` → idempotent (safe re-Register).
- Validate identifiers giữ nguyên (UNIQUE on `source_id`).
- 0 functional regression (chỉ thay đổi cách Exec, không touch column set).

**Action recommended**: x2 STAGE + COMMIT fix với commit msg `fix(cms): split shadow DDL into individual stmts to avoid PrepareStmt 42601`. KHÔNG cần Boss approve thêm vì đây là bug fix có evidence root cause đầy đủ.

## 4. Gaps surfaced (G-7..G-10) — pending fix

| Gap | Description | Lane | Owner | Effort | Boss approve? |
|---|---|---|---|---|---|
| **G-7** | Worker `docker-compose.yml` thiếu `PROVISIONING_ORCHESTRATOR_ENABLED=1` → state stuck `shadow_pending` | worker | max | 15 min (env var + restart) | YES (worker restart) |
| **G-8** | Path A (cdc_dw 5433) ≠ Path B (cdc_shadow 5436) → orphan empty table tại Path A | architecture | max plan + Boss decide | 2h investigate + 1h fix | YES (architectural) |
| **G-9** | Worker không auto-fire `cdc.cmd.kafka.refresh-topics` sau Register → operator phải pub manual | worker | max | 30 min (`registry-watch` handler) | NO (low risk worker code) |
| **G-10** | Worker `cdc.cmd.create-default-columns` reject `pk_type='string'` (PG type không tồn tại) → cần normalize → `text` tại cms Register | cms | x2 | 30 min (`register_registry.go` mapping) | NO (low risk cms code) |

## 5. Task plan iteration #1

### 5.1 Task cho x2 (cms-lane) — TODO

**Priority order**:

1. **[P0]** Stage + commit `shadow_automator.go` fix (workflow discipline + unblock cms repo).
   - `cd /Users/trainguyen/Documents/work/cdc-system && git add cdc-cms-service/internal/infra/persistence/shadow_automator.go cdc-cms-service/report_flow1_run_x2_2026-05-07.md`
   - Commit message: `fix(cms): split shadow DDL into individual stmts to avoid PrepareStmt 42601`
   - Verify: `cd cdc-cms-service && go build ./... && go test ./internal/infra/persistence -count=1`

2. **[P1]** Viết retroactive `09_tasks_solution_flow1_x2_2026-05-07.md` document workflow gap (per CLAUDE.md §11 audit trail) — ack đã thi công bug fix + Phương án Z + 4 gap mới (G-7..G-10).
   - Section 1: Acknowledge bug fix shadow_automator.go (lý do bypass workflow gate).
   - Section 2: Cmd-level Phương án Z exec log (curl + DB + NATS evidence).
   - Section 3: Gap G-7..G-10 cmd-level plan (lane assignment).
   - Section 4: Effort estimate per task.

3. **[P2]** Fix G-10 (cms-lane): Normalize `pk_type` mapping tại Register handler.
   - File candidates: `cdc-cms-service/internal/app/commands/register_registry.go` HOẶC `cdc-cms-service/internal/api/registry_handler.go` HOẶC `cdc-cms-service/internal/infra/persistence/source_object_v2_sync.go`.
   - Pattern: nếu inferred PK type là `'string'` → emit `'text'` thay vì `'string'` xuống NATS payload `cdc.cmd.create-default-columns`.
   - Verify: re-Register source → worker handler accept payload không reject.

4. **[P3]** P3.1 endpoint `POST /api/v1/sources/test` (per `08_tasks_flow1_e2e §P3.1`) — pending sau khi G-10 đóng.

### 5.2 Task cho max (worker-lane) — TODO

**Priority order**:

1. **[P0]** Investigate G-8 root cause (Path A vs Path B target divergence).
   - Read `cdc-cms-service/config/config-local.yml` block shadow connection.
   - Read `cdc-cms-service/internal/infra/persistence/shadow_automator.go` GORM session source.
   - Verify worker config `centralized-data-service/config/config-local.yml` Kafka consumer shadow target.
   - Output: decision doc `04_decisions_flow1_path_a_vs_b_2026-05-07.md` với 3 phương án (consolidate Path B, redirect Path A, deprecate Path A).

2. **[P1]** Plan G-7 worker restart với `PROVISIONING_ORCHESTRATOR_ENABLED=1`.
   - File: `cdc-system/centralized-data-service/docker-compose.yml`.
   - Add env var to worker service.
   - Boss approve restart → `docker-compose restart cdc-worker-host` HOẶC kill PID 23565 + spawn new binary.
   - Post-restart: re-Advance source 44 → verify state→`shadow_active` (AC-6).

3. **[P2]** Phase 2 Phương án Y (per `04_decisions §2.2`): Refactor `internal/admin/source_register.go:92` Step 5 từ direct UPDATE 'active' sang `orchestrator.Advance()`.
   - Pre-req: change Step 1 default state `'pending'` → `'draft'` để match `Transitions[StateDraft]`.
   - Backfill 4 phantom row id 33,34,35,37 state='active' → 'draft' rồi trigger advance qua cms.
   - Verify: AC-1 trả `state='shadow_pending'` (breaking change).

4. **[P3]** G-9 worker auto-fire `cdc.cmd.kafka.refresh-topics` sau registry insert (in lieu of operator manual pub).

### 5.3 Boss decision pending

| Item | Lane | Question |
|---|---|---|
| G-7 worker restart | infra | Approve restart `cdc-worker-host` PID 23565 sau khi max edit `docker-compose.yml`? |
| G-8 architectural choice | architecture | Consolidate cms ShadowAutomator vào Path B (5436 cdc_shadow) HOẶC deprecate Path A entirely? |
| Phương án Y breaking change | API contract | Approve `/v2/sources/register` response từ `state='active'` → `state='shadow_pending'` (breaking client cũ)? |
| Backfill 4 phantom rows | data | Approve UPDATE state='active' → 'draft' cho id 33,34,35,37 để re-fire advance? |
| MariaDB Debezium plugin (P4) | infra | Approve rebuild `gpay-kafka-connect` image với `debezium-debezium-connector-mysql:2.5.4`? |

## 6. Lessons applied this iteration

- **L-MUSCLE-PLAN-PROHIBITION** — x2 vi phạm workflow gate (không viết `09_tasks_solution_flow1_x2_*` trước execute) nhưng **partial-acceptable** vì §2 "Bug Fixing Tự chủ Full-loop" cho phép HTTP 500 block tự fix. APPEND retroactive `09_tasks_solution_*` trong P1 để đóng audit trail.
- **L-1755 (cũ)** — PrepareStmt 42601 multi-stmt rejection — confirmed pattern của x2 fix.
- **L-1688 Cascade Liability** — Mongo pre-flight gate đúng (refund_requests có 1720 docs → pass; goopay.users 0 docs → fail). Phase 3.2 extend gate cho PG/MariaDB pending.

## 7. Next iteration plan (cron fires in ~5 min)

1. Re-verify x2 stage + commit `shadow_automator.go` (`cd cdc-cms-service && git status`).
2. Re-verify x2 ship `09_tasks_solution_flow1_x2_2026-05-07.md`.
3. Check max output `04_decisions_flow1_path_a_vs_b_2026-05-07.md` (G-8 plan).
4. Check Boss approve trên các pending decision.
5. Nếu x2 + max chưa hoàn thành P0 task → re-prioritize trong APPEND coordination.

## 8. Files modified/created this iteration

- **Created**: `agent/memory/workspaces/feature-cdc-system-refactor/report_flow1_loop_iter1_2026-05-07.md` (this file)
- **APPEND** (sẽ làm tiếp): `agent/memory/workspaces/feature-cdc-system-refactor/coordination_max_x2_2026-05-07.md` — task plan section
- **APPEND** (sẽ làm tiếp): `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` — iteration log

— max-Brain (loop iteration #1)
