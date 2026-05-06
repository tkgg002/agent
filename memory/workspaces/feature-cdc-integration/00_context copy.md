Context                                                                                                                                                     
                                                                                                                                                             
 Track D demo (Source MongoDB/PG → Kafka → Shadow → Master across 4 isolated Postgres containers) is functional, architect-approved. P1 Config Consolidation
  đã thi công xong (single source of truth masterDb.default + block sources: thay block destination:/mongodb:). Còn lại 3 hardening tasks đã được architect
 chốt scope:

 - P2 (Bug #6) — SchemaAdapter.PrepareForCDCInsertInSchema chỉ ALTER, fail nếu shadow table chưa tồn tại. Operator phải CREATE TABLE thủ công — fragile cho
 automation.
 - P3 (Bug #2) — 10 hàng V1 legacy seed trong source_object_registry (object_code LIKE 'legacy_%') tồn tại, dù đã is_active=false từ Track D. Cần script
 idempotent để mọi đợt test sau không phải UPDATE thủ công.
 - P4 (D-39.A event-driven) — TransmuteScheduler set last_status='running' rồi fire NATS, không bao giờ close loop → schedule treo running. Architect yêu
 cầu KHÔNG sửa bằng UPDATE thuần trong handler mà phải tách logic: handler publish cdc.evt.transmute.completed, JobMonitor service mới subscribe → UPDATE.

 Out of scope: P5 Track E (MongoDB Debezium connector) — sẽ spawn workspace riêng feature-track-e-mongo-cdc/.

 ---
 P2 — SchemaAdapter CREATE TABLE IF NOT EXISTS

 Critical files

 - internal/service/schema_adapter.go:108–155 — PrepareForCDCInsertInSchema (return error nếu schema nil)
 - internal/service/schema_adapter.go:73 — loadSchemaInSchema (reuse để reload sau CREATE)
 - internal/service/schema_adapter.go:119–128 — V1 cdcCols map (giữ nguyên convention V1: _deleted, TIMESTAMP)
 - internal/handler/batch_buffer.go:131–177 — call site batchUpsert, đã có pk = first.PrimaryKeyField + first.MappedData
 - internal/handler/recon_handler.go:249 — call site PrepareForCDCInsert("public", target, pk)
 - internal/handler/recon_handler_integration_test.go:72 — pattern integration test reuse

 Approach (V1 columns only — không trộn V2 để khỏi đụng BuildUpsertSQLInSchema)

 Sửa PrepareForCDCInsertInSchema (line 108):

 func (sa *SchemaAdapter) PrepareForCDCInsertInSchema(schemaName, tableName, pkColumn string) error {
     schema, err := sa.GetSchemaInSchema(schemaName, tableName)
     if err != nil {
         return err
     }
     if schema == nil {
         // NEW: auto-create per architect P2 ruling. Idempotent — IF NOT EXISTS.
         if err := sa.createShadowTableV1(schemaName, tableName, pkColumn); err != nil {
             return fmt.Errorf("create shadow table %s.%s: %w", schemaName, tableName, err)
         }
         // Reload schema after CREATE so the ALTER pass below sees the column set.
         schema, err = sa.loadSchemaInSchema(schemaName, tableName)
         if err != nil {
             return err
         }
     }
     // … existing ALTER ADD COLUMN IF NOT EXISTS loop unchanged …
 }

 func (sa *SchemaAdapter) createShadowTableV1(schemaName, tableName, pkColumn string) error {
     // Validate identifiers (schemaName/tableName/pkColumn) via existing
     // IsSafeIdentifier helper if present; else fall back to pq.QuoteIdentifier.
     // 1. CREATE SCHEMA IF NOT EXISTS  (in case the schema is the first
     //    binding for that connection_code).
     // 2. CREATE TABLE IF NOT EXISTS with pk + V1 CDC cols inline.
     // 3. The UNIQUE-on-pk check stays in the existing tail of
     //    PrepareForCDCInsertInSchema (line 143–155) — no duplicate.
     ddl := []string{
         fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %q`, schemaName),
         fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %q.%q (
             %q TEXT,
             "_raw_data" JSONB,
             "_source" VARCHAR(20) DEFAULT 'airbyte',
             "_synced_at" TIMESTAMP DEFAULT NOW(),
             "_version" BIGINT DEFAULT 1,
             "_hash" VARCHAR(64),
             "_deleted" BOOLEAN DEFAULT FALSE,
             "_created_at" TIMESTAMP DEFAULT NOW(),
             "_updated_at" TIMESTAMP DEFAULT NOW()
         )`, schemaName, tableName, pkColumn),
     }
     for _, stmt := range ddl {
         if err := sa.db.Exec(stmt).Error; err != nil {
             return err
         }
     }
     return nil
 }

 Lý do PK type = TEXT (conservative): không suy đoán type từ MappedData để tránh schema drift; ALTER pass sau đó sẽ chỉ ADD CDC cols (không touch PK), và
 nếu user cần PK kiểu BIGINT, V2 path đã tách riêng (SchemaManager.createShadowTable). V1 path là legacy fallback — TEXT PK đủ dùng cho route ingest.

 Verification

 1. go build ./... PASS.
 2. New unit test internal/service/schema_adapter_test.go::TestPrepareForCDCInsertInSchema_AutoCreate (theo pattern recon_handler_integration_test.go): drop
  shadow_test_auto.t1, gọi PrepareForCDCInsertInSchema("shadow_test_auto", "t1", "id"), assert table exists + có đầy đủ V1 CDC cols + UNIQUE(id).
 3. Smoke test live: DROP TABLE shadow_goopay_source.orders CASCADE → restart worker → INSERT 1 row source → table tự tạo → row landed in shadow.

 ---
 P3 — Prune legacy V1 seeds (idempotent SQL)

 Critical files

 - migrations/cdc/035_v2_backfill_legacy_registry.sql — gốc của 10 hàng legacy_1..legacy_10 (object_code prefix)
 - cdc_system.source_object_registry — không có cột deactivated_at/deactivated_by (live-confirmed)
 - cdc_system.shadow_binding — không có cột deactivated; FK CASCADE → source_object_registry
 - cdc_system.master_binding — 0 hàng V1 (migration 035 không insert được vì không match)

 Approach

 File mới: deployments/sql/cdc/prune_legacy_v1_bindings.sql

 -- ----------------------------------------------------------------------
 -- Phase 01 split E2E / Track D Hardening (P3 / Bug #2)
 -- Target DB: gpay-postgres-cdc (cdc_dw)
 --
 -- Idempotent prune of V1 legacy seed rows in source_object_registry +
 -- shadow_binding. Eliminates first-write-wins routeCache collisions
 -- when V2 source_object_name == V1 source_object_name (e.g. "orders").
 --
 -- Discriminator: object_code LIKE 'legacy_%' (10 rows from migration
 -- 035_v2_backfill_legacy_registry.sql).
 --
 -- Re-runnable: only touches rows that are still active.
 -- ----------------------------------------------------------------------
 BEGIN;

 -- 1. Deactivate shadow_binding rows tied to legacy sources.
 WITH legacy_src AS (
     SELECT id FROM cdc_system.source_object_registry
      WHERE object_code LIKE 'legacy\_%' ESCAPE '\'
 )
 UPDATE cdc_system.shadow_binding sb
    SET is_active = false,
        updated_at = NOW()
   FROM legacy_src ls
  WHERE sb.source_object_id = ls.id
    AND sb.is_active = true;

 -- 2. Deactivate master_binding rows tied to legacy sources (defensive —
 -- migration 035 currently inserts 0 master rows but a future re-seed
 -- might).
 WITH legacy_src AS (
     SELECT id FROM cdc_system.source_object_registry
      WHERE object_code LIKE 'legacy\_%' ESCAPE '\'
 )
 UPDATE cdc_system.master_binding mb
    SET is_active = false,
        updated_at = NOW()
   FROM legacy_src ls
  WHERE mb.source_object_id = ls.id
    AND mb.is_active = true;

 -- 3. Deactivate the source rows themselves. Stamp `notes` so reviewers
 -- see why (no deactivated_at column to use).
 UPDATE cdc_system.source_object_registry
    SET is_active = false,
        notes = COALESCE(notes || E'\n', '') ||
                '[pruned by deployments/sql/cdc/prune_legacy_v1_bindings.sql at ' ||
                NOW()::text || ']',
        updated_at = NOW()
  WHERE object_code LIKE 'legacy\_%' ESCAPE '\'
    AND is_active = true;

 -- 4. Report.
 SELECT
     (SELECT count(*) FROM cdc_system.source_object_registry
       WHERE object_code LIKE 'legacy\_%' ESCAPE '\' AND is_active = false) AS pruned_sources,
     (SELECT count(*) FROM cdc_system.shadow_binding sb
       JOIN cdc_system.source_object_registry sor ON sor.id = sb.source_object_id
       WHERE sor.object_code LIKE 'legacy\_%' ESCAPE '\' AND sb.is_active = false) AS pruned_shadow_bindings,
     (SELECT count(*) FROM cdc_system.master_binding mb
       JOIN cdc_system.source_object_registry sor ON sor.id = mb.source_object_id
       WHERE sor.object_code LIKE 'legacy\_%' ESCAPE '\' AND mb.is_active = false) AS pruned_master_bindings;

 COMMIT;

 Verification

 1. Run: docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw < deployments/sql/cdc/prune_legacy_v1_bindings.sql — output: pruned_sources=10,
 pruned_shadow_bindings=10, pruned_master_bindings=0.
 2. Re-run cùng command — output không đổi (idempotent, 0 rows updated lần 2 vì WHERE is_active=true không match).
 3. Restart worker → log V2 metadata sources:1, shadow_bindings:1 (confirm không leak).

 ---
 P4 — D-39.A Event-driven scheduler close-loop

 Architecture (architect ruling)

 ┌──────────────────────┐ NATS cdc.cmd.transmute  ┌──────────────────────┐
 │ TransmuteScheduler   │────[+ schedule_id]─────▶│ TransmuteHandler     │
 │ (cron poll 60s)      │                          │ HandleTransmute()    │
 │ UPDATE last_status=  │                          │ → svc.Run()          │
 │        'running'     │                          └──────────┬───────────┘
 └──────────────────────┘                                     │
                                                              ▼
                             ┌────────[ NATS cdc.evt.transmute.completed ]
                             │             {schedule_id, status, stats, error}
                             ▼
               ┌──────────────────────────────┐
               │ JobMonitor (NEW)             │
               │ HandleCompleted(msg) →       │
               │ UPDATE cdc_system.transmute_ │
               │ schedule SET                 │
               │   last_status='success'/...  │
               │   last_stats=...             │
               │   last_error=...             │
               │ WHERE id=schedule_id         │
               └──────────────────────────────┘

 Critical files

 - internal/service/transmute_scheduler.go:26–35 — struct (nats *nats.Conn available)
 - internal/service/transmute_scheduler.go:146–149 — payload thiếu schedule_id
 - internal/handler/transmute_handler.go:29 — natsConn *nats.Conn available
 - internal/handler/transmute_handler.go:128 — TransmuteRequest struct (thiếu ScheduleID)
 - internal/handler/transmute_handler.go:144–189 — HandleTransmute body, đã publish cdc.result.transmute (line 189)
 - internal/service/transmuter.go:94–105 — TransmuteResult struct (đã có Inserted/Updated/Scanned/...)
 - internal/server/worker_server.go:269 — subscribe cdc.cmd.transmute
 - internal/server/worker_server.go:279–281 — start TransmuteScheduler

 Approach

 (a) Scheduler payload — transmute_scheduler.go:146

 payload, _ := json.Marshal(map[string]any{
     "schedule_id":    d.id,           // NEW
     "master_table":   d.master,
     "triggered_by":   "scheduler",
     "correlation_id": fmt.Sprintf("sched-%d-%d", d.id, now.UnixNano()),
 })

 (b) Handler request struct — transmute_handler.go:128

 type TransmuteRequest struct {
     ScheduleID    int64    `json:"schedule_id,omitempty"`  // NEW
     MasterTable   string   `json:"master_table"`
     SourceIDs     []string `json:"source_ids,omitempty"`
     CorrelationID string   `json:"correlation_id,omitempty"`
     TriggeredBy   string   `json:"triggered_by,omitempty"`
 }

 (c) Handler completion publish — transmute_handler.go:HandleTransmute (after h.svc.Run returns, before existing reply path)

 // Architect P4 (D-39.A) — close-loop event for JobMonitor.
 // Published independently from the existing `cdc.result.transmute`
 // reply so subscribers see it even when the caller didn't use Reply.
 status := "success"
 errStr := ""
 if err != nil {
     status = "failed"
     errStr = SanitizeFreeformText(err.Error(), 2000)
 }
 evt, _ := json.Marshal(map[string]any{
     "schedule_id":    req.ScheduleID,
     "correlation_id": req.CorrelationID,
     "master_table":   req.MasterTable,
     "status":         status,
     "stats": map[string]any{
         "scanned":     res.Scanned,
         "inserted":    res.Inserted,
         "updated":     res.Updated,
         "skipped":     res.Skipped,
         "rule_misses": res.RuleMisses,
         "type_errors": res.TypeErrors,
         "duration_ms": res.DurationMs,
     },
     "error":        errStr,
     "completed_at": time.Now().UTC().Format(time.RFC3339Nano),
 })
 if perr := h.natsConn.Publish("cdc.evt.transmute.completed", evt); perr != nil {
     h.logger.Warn("publish transmute.completed failed",
         zap.Int64("schedule_id", req.ScheduleID),
         zap.Error(perr))
 }

 (d) New file internal/service/job_monitor.go

 package service

 import (
     "context"
     "encoding/json"

     "github.com/nats-io/nats.go"
     "go.uber.org/zap"
     "gorm.io/gorm"
 )

 const SubjectTransmuteCompleted = "cdc.evt.transmute.completed"

 type JobMonitor struct {
     db     *gorm.DB
     logger *zap.Logger
 }

 func NewJobMonitor(db *gorm.DB, logger *zap.Logger) *JobMonitor {
     return &JobMonitor{db: db, logger: logger}
 }

 type transmuteCompletedEvent struct {
     ScheduleID    int64           `json:"schedule_id"`
     CorrelationID string          `json:"correlation_id"`
     MasterTable   string          `json:"master_table"`
     Status        string          `json:"status"`
     Stats         json.RawMessage `json:"stats"`
     Error         string          `json:"error"`
     CompletedAt   string          `json:"completed_at"`
 }

 // HandleCompleted subscribes to cdc.evt.transmute.completed and
 // closes the loop on cdc_system.transmute_schedule. Idempotent:
 // only updates rows where current last_status='running' (skips
 // ones the scheduler already failed at publish time).
 func (m *JobMonitor) HandleCompleted(msg *nats.Msg) {
     var ev transmuteCompletedEvent
     if err := json.Unmarshal(msg.Data, &ev); err != nil {
         m.logger.Warn("job monitor: bad payload", zap.Error(err))
         return
     }
     if ev.ScheduleID == 0 {
         // ad-hoc transmute (not from scheduler) — nothing to update.
         return
     }
     ctx := context.Background()
     err := m.db.WithContext(ctx).Exec(
         `UPDATE cdc_system.transmute_schedule
             SET last_status = ?,
                 last_stats  = ?::jsonb,
                 last_error  = NULLIF(?, ''),
                 updated_at  = NOW()
           WHERE id = ?
             AND last_status = 'running'`,
         ev.Status, string(ev.Stats), ev.Error, ev.ScheduleID).Error
     if err != nil {
         m.logger.Warn("job monitor: update failed",
             zap.Int64("schedule_id", ev.ScheduleID), zap.Error(err))
         return
     }
     m.logger.Info("job monitor: schedule closed",
         zap.Int64("schedule_id", ev.ScheduleID),
         zap.String("status", ev.Status))
 }

 (e) Wire at boot — worker_server.go (after line 281, sau scheduler start)

 // Architect P4 (D-39.A) — JobMonitor closes the loop on
 // cdc_system.transmute_schedule by subscribing to the handler's
 // completion event.
 jobMonitor := service.NewJobMonitor(db, logger)
 if _, err := natsClient.Conn.Subscribe(service.SubjectTransmuteCompleted, jobMonitor.HandleCompleted); err != nil {
     return fmt.Errorf("subscribe %s: %w", service.SubjectTransmuteCompleted, err)
 }
 logger.Info("job monitor registered", zap.String("subject", service.SubjectTransmuteCompleted))

 Verification

 1. go build ./... PASS.
 2. go test ./internal/service/ ./internal/handler/ -count=1 PASS.
 3. Smoke: rebuild worker, restart. Wait 60s, check:
   - cdc_system.transmute_schedule.last_status='success' (not 'running')
   - last_stats contains {"scanned":20,...} JSON
   - last_error IS NULL
 4. Failure path: temporarily break a mapping rule (UPDATE mapping_rule_v2 SET data_type='BOGUS_TYPE' to cause type validation drop), wait 60s →
 last_status='success' still (because handler doesn't return err for rule_misses). Then to genuinely test failed: stop master DB → next tick handler returns
  err → last_status='failed', last_error <> ''.
 5. Idempotency: publish a duplicate cdc.evt.transmute.completed for the same schedule_id → second UPDATE is a no-op (WHERE clause filters
 last_status='running').

 ---
 Execution order (architect ruling Q3=b)

 1. P2 — schema_adapter.go + tests (smallest blast radius, isolated to one function)
 2. P3 — pure SQL file under deployments/sql/cdc/ (zero code touch)
 3. P4 — scheduler payload + handler publish + new job_monitor.go + boot wire (largest blast radius — boot path)
 4. After all 3 land: end-to-end smoke (DROP shadow → re-INSERT source → wait 1 cron tick → verify last_status='success' on schedule + 20 rows in master)
 5. Update workspace: APPEND agent/memory/workspaces/feature-multi-pg-isolation-e2e/05_progress.md (rule #11 — no overwrite). Add Global Pattern lesson to
 agent/memory/global/lessons.md for event-driven close-loop pattern.

 End-to-end verification (after P2+P3+P4 complete)

 # 1. Apply prune (P3) — idempotent.
 docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   < deployments/sql/cdc/prune_legacy_v1_bindings.sql

 # 2. Drop shadow table to force P2 auto-create path.
 docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   -c "DROP TABLE IF EXISTS shadow_goopay_source.orders CASCADE;"

 # 3. Rebuild + restart worker.
 go build -o /tmp/cdc-worker ./cmd/worker
 # (kill existing, restart with env)

 # 4. Insert 5 new rows on source.
 docker exec -i gpay-postgres-source psql -U src_user -d goopay_source <<SQL
 INSERT INTO public.orders (user_id, amount, status, notes)
 SELECT 4000+i, 100+i, 'pending', 'p2-p3-p4-smoke-'||i FROM generate_series(1,5) i;
 SQL

 # 5. Verify auto-create (P2):
 docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   -c "\d shadow_goopay_source.orders" \
   # Expect: 9 V1 CDC cols + pk col, no manual bootstrap.

 # 6. Wait 60s for cron tick. Verify close-loop (P4):
 docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   -c "SELECT id, last_status, last_stats, last_error FROM cdc_system.transmute_schedule WHERE id=1;"
 # Expect: last_status='success', last_stats={"inserted":N,...}, last_error=NULL.

 # 7. Verify master count grew:
 docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest \
   -c "SELECT count(*) FROM dw_orders.orders_fact;"
 # Expect: previous count + 5.

 # 8. Verify P3 idempotency:
 docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   -c "SELECT count(*) FROM cdc_system.source_object_registry WHERE object_code LIKE 'legacy_%' AND is_active=true;"
 # Expect: 0.

 Files to be modified or created

 ┌───────────────────────────────────────────────────────────────────────┬───────────────────────────────────────────────────────────────────────────────┐
 │                                 Path                                  │                                    Action                                     │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/service/schema_adapter.go                                    │ Edit PrepareForCDCInsertInSchema + add private createShadowTableV1            │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/service/schema_adapter_test.go                               │ NEW — TestPrepareForCDCInsertInSchema_AutoCreate                              │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ deployments/sql/cdc/prune_legacy_v1_bindings.sql                      │ NEW                                                                           │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/service/transmute_scheduler.go                               │ Edit dispatch payload to include schedule_id                                  │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/handler/transmute_handler.go                                 │ Edit TransmuteRequest struct + HandleTransmute to publish                     │
 │                                                                       │ cdc.evt.transmute.completed                                                   │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/service/job_monitor.go                                       │ NEW — JobMonitor + HandleCompleted                                            │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/server/worker_server.go                                      │ Edit boot to register JobMonitor subscription                                 │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ agent/memory/workspaces/feature-multi-pg-isolation-e2e/05_progress.md │ APPEND P2/P3/P4 done log                                                      │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ agent/memory/global/lessons.md                                        │ APPEND Global Pattern: "Fire-and-forget command needs companion completion    │
 │                                                                       │ event for close-loop status"                                                  │
 └───────────────────────────────────────────────────────────────────────┴───────────────────────────────────────────────────────────────────────────────┘
╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌

 Claude has written up a plan and is ready to execute. Would you like to proceed?

 ❯ 1. Yes, and use auto mode
   2. Yes, manually approve edits
   3. No, refine with Ultraplan on Claude Code on the web
   4. Tell Claude what to change
      shift+tab to approve with this feedback

 ctrl-g to edit in Vim · ~/.claude/plans/curried-waddling-spindle.md
     "correlation_id": fmt.Sprintf("sched-%d-%d", d.id, now.UnixNano()),
 })

 (b) Handler request struct — transmute_handler.go:128

 type TransmuteRequest struct {
     ScheduleID    int64    `json:"schedule_id,omitempty"`  // NEW
     MasterTable   string   `json:"master_table"`
     SourceIDs     []string `json:"source_ids,omitempty"`
     CorrelationID string   `json:"correlation_id,omitempty"`
     TriggeredBy   string   `json:"triggered_by,omitempty"`
 }

 (c) Handler completion publish — transmute_handler.go:HandleTransmute (after h.svc.Run 
 returns, before existing reply path)

 // Architect P4 (D-39.A) — close-loop event for JobMonitor.
 // Published independently from the existing `cdc.result.transmute`
 // reply so subscribers see it even when the caller didn't use Reply.
 status := "success"
 errStr := ""
 if err != nil {
     status = "failed"
     errStr = SanitizeFreeformText(err.Error(), 2000)
 }
 evt, _ := json.Marshal(map[string]any{
     "schedule_id":    req.ScheduleID,
     "correlation_id": req.CorrelationID,
     "master_table":   req.MasterTable,
     "status":         status,
     "stats": map[string]any{
         "scanned":     res.Scanned,
         "inserted":    res.Inserted,
         "updated":     res.Updated,
         "skipped":     res.Skipped,
         "rule_misses": res.RuleMisses,
         "type_errors": res.TypeErrors,
         "duration_ms": res.DurationMs,
     },
     "error":        errStr,
     "completed_at": time.Now().UTC().Format(time.RFC3339Nano),
 })
 if perr := h.natsConn.Publish("cdc.evt.transmute.completed", evt); perr != nil {
     h.logger.Warn("publish transmute.completed failed",
         zap.Int64("schedule_id", req.ScheduleID),
         zap.Error(perr))
 }

 (d) New file internal/service/job_monitor.go

 package service

 import (
     "context"
     "encoding/json"

     "github.com/nats-io/nats.go"
     "go.uber.org/zap"
     "gorm.io/gorm"
 )

 const SubjectTransmuteCompleted = "cdc.evt.transmute.completed"

 type JobMonitor struct {
     db     *gorm.DB
     logger *zap.Logger
 }

 func NewJobMonitor(db *gorm.DB, logger *zap.Logger) *JobMonitor {
     return &JobMonitor{db: db, logger: logger}
 }

 type transmuteCompletedEvent struct {
     ScheduleID    int64           `json:"schedule_id"`
     CorrelationID string          `json:"correlation_id"`
     MasterTable   string          `json:"master_table"`
     Status        string          `json:"status"`
     Stats         json.RawMessage `json:"stats"`
     Error         string          `json:"error"`
     CompletedAt   string          `json:"completed_at"`
 }

 // HandleCompleted subscribes to cdc.evt.transmute.completed and
 // closes the loop on cdc_system.transmute_schedule. Idempotent:
 // only updates rows where current last_status='running' (skips
 // ones the scheduler already failed at publish time).
 func (m *JobMonitor) HandleCompleted(msg *nats.Msg) {
     var ev transmuteCompletedEvent
     if err := json.Unmarshal(msg.Data, &ev); err != nil {
         m.logger.Warn("job monitor: bad payload", zap.Error(err))
         return
     }
     if ev.ScheduleID == 0 {
         // ad-hoc transmute (not from scheduler) — nothing to update.
         return
     }
     ctx := context.Background()
     err := m.db.WithContext(ctx).Exec(
         `UPDATE cdc_system.transmute_schedule
             SET last_status = ?,
                 last_stats  = ?::jsonb,
                 last_error  = NULLIF(?, ''),
                 updated_at  = NOW()
           WHERE id = ?
             AND last_status = 'running'`,
         ev.Status, string(ev.Stats), ev.Error, ev.ScheduleID).Error
     if err != nil {
         m.logger.Warn("job monitor: update failed",
             zap.Int64("schedule_id", ev.ScheduleID), zap.Error(err))
         return
     }
     m.logger.Info("job monitor: schedule closed",
         zap.Int64("schedule_id", ev.ScheduleID),
         zap.String("status", ev.Status))
 }

 (e) Wire at boot — worker_server.go (after line 281, sau scheduler start)

 // Architect P4 (D-39.A) — JobMonitor closes the loop on
 // cdc_system.transmute_schedule by subscribing to the handler's
 // completion event.
 jobMonitor := service.NewJobMonitor(db, logger)
 if _, err := natsClient.Conn.Subscribe(service.SubjectTransmuteCompleted,
 jobMonitor.HandleCompleted); err != nil {
     return fmt.Errorf("subscribe %s: %w", service.SubjectTransmuteCompleted, err)
 }
 logger.Info("job monitor registered", zap.String("subject",
 service.SubjectTransmuteCompleted))

 Verification

 1. go build ./... PASS.
 2. go test ./internal/service/ ./internal/handler/ -count=1 PASS.
 3. Smoke: rebuild worker, restart. Wait 60s, check:
   - cdc_system.transmute_schedule.last_status='success' (not 'running')
   - last_stats contains {"scanned":20,...} JSON
   - last_error IS NULL
 4. Failure path: temporarily break a mapping rule (UPDATE mapping_rule_v2 SET
 data_type='BOGUS_TYPE' to cause type validation drop), wait 60s → last_status='success'
  still (because handler doesn't return err for rule_misses). Then to genuinely test
 failed: stop master DB → next tick handler returns err → last_status='failed',
 last_error <> ''.
 5. Idempotency: publish a duplicate cdc.evt.transmute.completed for the same
 schedule_id → second UPDATE is a no-op (WHERE clause filters last_status='running').

 ---
 Execution order (architect ruling Q3=b)

 1. P2 — schema_adapter.go + tests (smallest blast radius, isolated to one function)
 2. P3 — pure SQL file under deployments/sql/cdc/ (zero code touch)
 3. P4 — scheduler payload + handler publish + new job_monitor.go + boot wire (largest
 blast radius — boot path)
 4. After all 3 land: end-to-end smoke (DROP shadow → re-INSERT source → wait 1 cron
 tick → verify last_status='success' on schedule + 20 rows in master)
 5. Update workspace: APPEND
 agent/memory/workspaces/feature-multi-pg-isolation-e2e/05_progress.md (rule #11 — no
 overwrite). Add Global Pattern lesson to agent/memory/global/lessons.md for
 event-driven close-loop pattern.

 End-to-end verification (after P2+P3+P4 complete)

 # 1. Apply prune (P3) — idempotent.
 docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   < deployments/sql/cdc/prune_legacy_v1_bindings.sql

 # 2. Drop shadow table to force P2 auto-create path.
 docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   -c "DROP TABLE IF EXISTS shadow_goopay_source.orders CASCADE;"

 # 3. Rebuild + restart worker.
 go build -o /tmp/cdc-worker ./cmd/worker
 # (kill existing, restart with env)

 # 4. Insert 5 new rows on source.
 docker exec -i gpay-postgres-source psql -U src_user -d goopay_source <<SQL
 INSERT INTO public.orders (user_id, amount, status, notes)
 SELECT 4000+i, 100+i, 'pending', 'p2-p3-p4-smoke-'||i FROM generate_series(1,5) i;
 SQL

 # 5. Verify auto-create (P2):
 docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   -c "\d shadow_goopay_source.orders" \
   # Expect: 9 V1 CDC cols + pk col, no manual bootstrap.

 # 6. Wait 60s for cron tick. Verify close-loop (P4):
 docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   -c "SELECT id, last_status, last_stats, last_error FROM cdc_system.transmute_schedule
  WHERE id=1;"
 # Expect: last_status='success', last_stats={"inserted":N,...}, last_error=NULL.

 # 7. Verify master count grew:
 docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest \
   -c "SELECT count(*) FROM dw_orders.orders_fact;"
 # Expect: previous count + 5.

 # 8. Verify P3 idempotency:
 docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
   -c "SELECT count(*) FROM cdc_system.source_object_registry WHERE object_code LIKE
 'legacy_%' AND is_active=true;"
 # Expect: 0.

 Files to be modified or created

 ┌──────────────────────────────────────────────────┬───────────────────────────────┐
 │                       Path                       │            Action             │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │                                                  │ Edit                          │
 │ internal/service/schema_adapter.go               │ PrepareForCDCInsertInSchema + │
 │                                                  │  add private                  │
 │                                                  │ createShadowTableV1           │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │ internal/service/schema_adapter_test.go          │ NEW — TestPrepareForCDCInsert │
 │                                                  │ InSchema_AutoCreate           │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │ deployments/sql/cdc/prune_legacy_v1_bindings.sql │ NEW                           │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │ internal/service/transmute_scheduler.go          │ Edit dispatch payload to      │
 │                                                  │ include schedule_id           │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │                                                  │ Edit TransmuteRequest struct  │
 │ internal/handler/transmute_handler.go            │ + HandleTransmute to publish  │
 │                                                  │ cdc.evt.transmute.completed   │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │ internal/service/job_monitor.go                  │ NEW — JobMonitor +            │
 │                                                  │ HandleCompleted               │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │ internal/server/worker_server.go                 │ Edit boot to register         │
 │                                                  │ JobMonitor subscription       │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │ agent/memory/workspaces/feature-multi-pg-isolati │ APPEND P2/P3/P4 done log      │
 │ on-e2e/05_progress.md                            │                               │
 ├──────────────────────────────────────────────────┼───────────────────────────────┤
 │                                                  │ APPEND Global Pattern:        │
 │ agent/memory/global/lessons.md                   │ "Fire-and-forget command      │
 │                                                  │ needs companion completion    │
 │                                                  │ event for close-loop status"  │
 └──────────────────────────────────────────────────┴───────────────────────────────┘
╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌

 Claude has written up a plan and is ready to execute. Would you like to proceed?

 ❯ 1. Yes, and use auto mode
   2. Yes, manually approve edits
   3. No, refine with Ultraplan on Claude Code on the web
   4. Tell Claude what to change
      shift+tab to approve with this feedback

 ctrl-g to edit in Vim · ~/.claude/plans/curried-waddling-spindle.md
 │ internal/service/transmute_scheduler.go                               │ Edit dispatch payload to include schedule_id                                  │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/handler/transmute_handler.go                                 │ Edit TransmuteRequest struct + HandleTransmute to publish                     │
 │                                                                       │ cdc.evt.transmute.completed                                                   │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/service/job_monitor.go                                       │ NEW — JobMonitor + HandleCompleted                                            │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ internal/server/worker_server.go                                      │ Edit boot to register JobMonitor subscription                                 │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ agent/memory/workspaces/feature-multi-pg-isolation-e2e/05_progress.md │ APPEND P2/P3/P4 done log                                                      │
 ├───────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────┤
 │ agent/memory/global/lessons.md                                        │ APPEND Global Pattern: "Fire-and-forget command needs companion completion    │
 │                                                                       │ event for close-loop status"                                                  │
 └───────────────────────────────────────────────────────────────────────┴───────────────────────────────────────────────────────────────────────────────┘
╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌

 Claude has written up a plan and is ready to execute. Would you like to proceed?

 ❯ 1. Yes, and use auto mode
   2. Yes, manually approve edits
   3. No, refine with Ultraplan on Claude Code on the web
   4. Tell Claude what to change
      shift+tab to approve with this feedback

 ctrl-g to edit in Vim · ~/.claude/plans/curried-waddling-spindle.md