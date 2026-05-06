# Phase 2 — cdc-cms-service Refactor — Tasks Solution Sketch

> **Date**: 2026-05-05 | **Tasks ref**: `08_tasks_phase2_cms_refactor.md`

Solution sketch chi tiết cho từng task. **Code chưa chạm** — sketch để Muscle pre-flight đúng pattern.

---

## T1 — P0: Dead-code prune

### File diff sketch

`internal/service/reconciliation_service.go` — DELETE toàn bộ file (line 1-50).

`internal/server/server.go`:
```diff
@@ struct
- reconSvc *service.ReconciliationService
- mappingRuleRepo *repository.MappingRuleRepo  // chỉ dùng bởi reconSvc — verify

@@ New()
- reconSvc := service.NewReconciliationService(db, registryRepo, mappingRuleRepo, logger)

@@ Start()
- go func() { srv.reconSvc.Start(ctx) }()
```

`internal/model/cdc_event.go` — verify trước:
```bash
grep -rn "model.CDCEvent\|model.CDCEventData\|model.UpsertRecord" \
  /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service \
  --include='*.go'
```
- 0 hit → DELETE.
- ≥1 hit → giữ + ghi note "kept due to <callsite>".

### Verify
```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service
go build ./...
# expect: PASS, no error
go vet ./...
# expect: clean
```

8 smoke endpoint via curl với JWT (preset từ login `auth-service`).

---

## T2 — P1.0: Repo skeleton (MappingRuleV2Repo demo)

### New file `internal/repository/v2_mapping_rule_repo.go`

```go
package repository

import (
    "context"
    "errors"
    "fmt"
    "strings"

    "gorm.io/gorm"
)

type MappingRuleV2DTO struct {
    ID            int64  `gorm:"column:id"`
    SourceObjectID int64 `gorm:"column:source_object_id"`
    SourceField   string `gorm:"column:source_field"`
    TargetColumn  string `gorm:"column:target_column"`
    DataType      string `gorm:"column:data_type"`
    JSONPath      string `gorm:"column:jsonpath"`
    TransformFn   string `gorm:"column:transform_fn"`
    Status        string `gorm:"column:status"`
    CreatedAt     string `gorm:"column:created_at"`
    UpdatedAt     string `gorm:"column:updated_at"`
}

type MappingRuleV2Filter struct {
    SourceObjectID *int64
    Scope          string
    Status         string
    Limit          int
    Offset         int
}

type MappingRuleV2Repo interface {
    List(ctx context.Context, f MappingRuleV2Filter) ([]MappingRuleV2DTO, int64, error)
    GetByID(ctx context.Context, id int64) (*MappingRuleV2DTO, error)
    Create(ctx context.Context, rule *MappingRuleV2DTO) error
    UpdateStatus(ctx context.Context, id int64, status string) error
    BatchUpdateStatus(ctx context.Context, ids []int64, status string) (int64, error)
}

type mappingRuleV2GormRepo struct {
    db *gorm.DB
}

func NewMappingRuleV2Repo(db *gorm.DB) MappingRuleV2Repo {
    return &mappingRuleV2GormRepo{db: db}
}

func (r *mappingRuleV2GormRepo) List(ctx context.Context, f MappingRuleV2Filter) ([]MappingRuleV2DTO, int64, error) {
    // Move query từ mapping_rule_handler.go:244-357 vào đây.
    // Schema-qualified per Lesson #1240.
    var dst []MappingRuleV2DTO
    var total int64

    base := r.db.WithContext(ctx).Table("cdc_system.mapping_rule_v2")
    if f.SourceObjectID != nil {
        base = base.Where("source_object_id = ?", *f.SourceObjectID)
    }
    if f.Status != "" {
        base = base.Where("status = ?", f.Status)
    }
    if err := base.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("count: %w", err)
    }
    q := base.Order("id DESC")
    if f.Limit > 0 { q = q.Limit(f.Limit) }
    if f.Offset > 0 { q = q.Offset(f.Offset) }
    if err := q.Find(&dst).Error; err != nil {
        return nil, 0, fmt.Errorf("find: %w", err)
    }
    return dst, total, nil
}

// ... GetByID, Create, UpdateStatus, BatchUpdateStatus tương tự ...
```

### Refactor `internal/api/mapping_rule_handler.go:List`

```diff
- // raw SQL inline (line 244-357)
- rows := []struct{...}{}
- if err := h.db.Raw(`SELECT ...`, args...).Scan(&rows).Error; err != nil {
-     return ... 500
- }
+ filter := repository.MappingRuleV2Filter{
+     SourceObjectID: parseInt64Param(c, "source_object_id"),
+     Scope:          c.Query("scope"),
+     Status:         c.Query("status"),
+     Limit:          parseLimit(c, 50),
+     Offset:         parseOffset(c),
+ }
+ rows, total, err := h.mappingRepo.List(c.Context(), filter)
+ if err != nil { return ... 500 }
  return c.JSON(fiber.Map{"data": rows, "total": total})
```

### Wire `internal/server/server.go`

```diff
+ mappingRepo := repository.NewMappingRuleV2Repo(db)
- mappingHandler := api.NewMappingRuleHandler(db, natsClient, ...)
+ mappingHandler := api.NewMappingRuleHandler(db, natsClient, mappingRepo, ...)
```

### Unit test `v2_mapping_rule_repo_test.go`

```go
func TestList_FilterBySourceObjectID(t *testing.T) {
    db, mock := newSqlMock(t)
    repo := NewMappingRuleV2Repo(db)
    mock.ExpectQuery(`SELECT count\(\*\) FROM "cdc_system"\."mapping_rule_v2" WHERE source_object_id = `).
        WithArgs(int64(42)).
        WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
    mock.ExpectQuery(`SELECT \* FROM "cdc_system"\."mapping_rule_v2"`).
        WithArgs(int64(42)).
        WillReturnRows(sqlmock.NewRows([]string{"id","source_object_id","status"}).
            AddRow(1, 42, "approved").AddRow(2, 42, "pending"))

    soid := int64(42)
    rows, total, err := repo.List(context.Background(), MappingRuleV2Filter{SourceObjectID: &soid})
    require.NoError(t, err)
    require.Equal(t, int64(2), total)
    require.Len(t, rows, 2)
}
```

### Verify
```bash
go test ./internal/repository/ -run TestList -v
curl -sH "Authorization: Bearer $JWT" "http://localhost:8083/api/mapping-rules?source_object_id=42" | jq .
```

---

## T3-T8 — P1.1-P1.6: Fan-out 6 repo

Same pattern T2. Cho mỗi repo:
- Define DTO struct (flat, column tags)
- Define Filter struct
- Define Repo interface (List/Get/Create/Update verbs theo handler hiện tại)
- Implement với GORM
- Refactor 1-2 handler call site
- Unit test golden path

Effort split:
- T3 (SourceObject) 1d — biggest schema, 2 table join (registry + binding)
- T4 (MasterBinding) 6h
- T5 (ConnectionRegistry) 4h — đơn giản, lookup-only
- T6 (WizardSession) 6h
- T7 (Alert) 6h — `alert_manager.go` đã layered, mostly extract DB queries
- T8 (AdminAction) 4h — đơn giản, append-only INSERT

---

## T9 — P2.1: Split `reconciliation_handler.go`

### File mới `internal/service/reconciliation/drift_calculator.go`

```go
package reconciliation

type DriftStatus string
const (
    DriftOK       DriftStatus = "ok"
    DriftMinor    DriftStatus = "minor"
    DriftMajor    DriftStatus = "major"
    DriftCritical DriftStatus = "critical"
)

type DriftInput struct {
    SourceCount int64
    DestCount   int64
    Threshold   float64  // default 0.05 (5%)
}

func ComputeDriftStatus(in DriftInput) (DriftStatus, float64) {
    // Move logic từ reconciliation_handler.go:34-73
    if in.SourceCount == 0 {
        if in.DestCount == 0 { return DriftOK, 0 }
        return DriftCritical, 1.0
    }
    diff := float64(in.DestCount-in.SourceCount) / float64(in.SourceCount)
    abs := diff
    if abs < 0 { abs = -abs }
    switch {
    case abs <= in.Threshold: return DriftOK, abs
    case abs <= 0.10: return DriftMinor, abs
    case abs <= 0.30: return DriftMajor, abs
    default: return DriftCritical, abs
    }
}
```

Test:
```go
func TestComputeDriftStatus(t *testing.T) {
    cases := []struct{ src, dst int64; want DriftStatus }{
        {100, 100, DriftOK},
        {100, 103, DriftOK},     // 3% < threshold
        {100, 108, DriftMinor},  // 8%
        {100, 130, DriftMajor},  // 30%
        {100, 200, DriftCritical}, // 100%
        {0, 0, DriftOK},
        {0, 10, DriftCritical},
    }
    // ...
}
```

### File mới `internal/service/reconciliation/recon_dispatcher.go`

```go
type Dispatcher struct {
    nats   *nats.Conn
    logger *zap.Logger
}

func (d *Dispatcher) ReconCheck(ctx context.Context, tier int, table string) error {
    payload := map[string]any{"tier": tier, "table": table}
    return d.publish(ctx, "cdc.cmd.recon-check", payload)
}
func (d *Dispatcher) Heal(ctx, table) error { ... }
func (d *Dispatcher) Retry(ctx, failedLogID, scope) error { ... }
func (d *Dispatcher) DebeziumSignal(ctx, db, coll) error { ... }
func (d *Dispatcher) BackfillSourceTS(ctx, table, runID, batchSize) error { ... }
func (d *Dispatcher) Snapshot(ctx, table) error { ... }

func (d *Dispatcher) publish(ctx, subject, payload) error {
    body, _ := json.Marshal(payload)
    if err := d.nats.Publish(subject, body); err != nil {
        return fmt.Errorf("nats publish %s: %w", subject, err)
    }
    d.logger.Info("dispatched", zap.String("subject", subject))
    return nil
}
```

### File mới `internal/service/reconciliation/{report_query,failed_log_query}.go`

Move 2 SQL block (line 175-450, 538-624) → 2 repo method (depend trên `ReconciliationReportRepo` + `FailedSyncLogRepo` từ P1).

### Refactor `reconciliation_handler.go`

Sau split: handler ≤200 dòng, mỗi method 5-15 dòng:
```go
func (h *ReconciliationHandler) GetReport(c *fiber.Ctx) error {
    f := parseReportFilter(c)
    rows, err := h.svc.ReportQuery(c.Context(), f)
    if err != nil { return err500(c, err) }
    return c.JSON(fiber.Map{"data": rows})
}
```

### Move utility

`pgIdent` (line 873-886) → `pkgs/utils/pg_ident.go`. Lesson #1240 verifies usage qualified everywhere.

### Verify
- `wc -l internal/api/reconciliation_handler.go` ≤ 200
- 4 endpoint smoke (xem T9 DoD)
- 1 test cho `ComputeDriftStatus` (mỗi case)

---

## T10-T12 — Pattern tương tự T9

Sketch tóm tắt — full code sketch mỗi T sẽ ghi tiếp khi thực thi (avoid bloat plan doc).

T10 (`source_object_actions_handler.go` 693 dòng):
- Generic `Dispatch(ctx, DispatchSpec)`:
  ```go
  type DispatchSpec struct {
      Subject     string
      Scope       Scope
      ExtraFields map[string]any
  }
  func (s *DispatchService) Dispatch(ctx, spec) error
  ```
- 7 method (`Standardize`, `ScanFields`, ...) thành 7 wrapper 3-5 dòng.

T11 (`mapping_rule_handler.go` 689 dòng):
- `MappingRuleService.BatchUpdate(ctx, ids, status) (BatchResult, error)` — single SQL UPDATE thay N+1.
- Bulk publish 1 NATS message với array thay 1-per-rule.

T12 (`master_registry_handler.go` 667 dòng):
- `MasterService.Create/Approve/Reject/ToggleActive` 4 method.
- `MasterService.Swap(ctx, name)` reuse `MasterSwap` service đã có (line `internal/service/master_swap.go`).

---

## T13 — P3: ActivityLog helper

### File mới `internal/service/activity_logger.go`

```go
type ActivityLogger struct {
    repo   repository.ActivityLogRepo  // hoặc *gorm.DB nếu chưa có repo cho legacy
    logger *zap.Logger
    queue  chan ActivityEvent  // async drain
}

type ActivityEvent struct {
    Operation   string
    TargetTable string
    Status      string
    Details     map[string]any
    TriggeredBy string
    SourceID    *int64        // V2 scope
    SourceDB    string
    SourceTable string
    SyncEngine  string
}

func NewActivityLogger(...) *ActivityLogger {
    a := &ActivityLogger{queue: make(chan ActivityEvent, 1000), ...}
    go a.drain()
    return a
}

func (a *ActivityLogger) Log(ctx context.Context, e ActivityEvent) error {
    rec := model.ActivityLog{
        Operation:   e.Operation,
        TargetTable: e.TargetTable,
        Status:      e.Status,
        Details:     marshalJSON(e.Details),
        TriggeredBy: e.TriggeredBy,
        StartedAt:   time.Now(),
    }
    return a.repo.Create(ctx, &rec)
}

func (a *ActivityLogger) LogAsync(e ActivityEvent) {
    select {
    case a.queue <- e:
    default:
        a.logger.Warn("activity log queue full, dropping")
    }
}

func (a *ActivityLogger) drain() {
    for e := range a.queue {
        if err := a.Log(context.Background(), e); err != nil {
            a.logger.Error("activity log drain failed", zap.Error(err))
        }
    }
}
```

### Replace 8+ inline call site

Pattern from `registry_handler.go:50-61`:
```diff
- entry := model.ActivityLog{
-     Operation:   "register",
-     TargetTable: req.TargetTable,
-     Status:      "success",
-     Details:     marshalJSON(req),
-     TriggeredBy: c.Locals("username").(string),
-     StartedAt:   time.Now(),
- }
- if err := h.db.Create(&entry).Error; err != nil { ... }
+ h.activityLogger.LogAsync(service.ActivityEvent{
+     Operation:   "register",
+     TargetTable: req.TargetTable,
+     Status:      "success",
+     Details:     map[string]any{"req": req},
+     TriggeredBy: c.Locals("username").(string),
+ })
```

`grep "model.ActivityLog{" internal/api/` = 0 sau khi xong.

---

## T14 — P4: V1↔V2 dedup

### Pre-check (BẮT BUỘC trước commit)

Đọc:
- `agent/memory/workspaces/feature-cms-fe-overhaul/03_implementation_phase27_v2_write_sync.md`
- `03_implementation_phase29_v2_direct_update.md`
- `03_implementation_phase30_v2_direct_redetect.md`
- `03_implementation_phase31_direct_scan_transform_status.md`
- `03_implementation_phase32_direct_standardize.md`

Để biết V2 đã direct hay vẫn delegate. Nếu phase 32 đã "direct standardize" → V2 không còn gọi V1; T14 scope = chỉ dedup helper.

### Refactor sketch (assume vẫn delegate)

```diff
@@ source_object_actions_handler.go
- func (h *SourceObjectActionsHandler) StandardizeV2(c) error {
-     // ...resolve scope by source_object_id...
-     return h.registry.Standardize(c)  // V1 delegate ← REMOVE
- }
+ func (h *SourceObjectActionsHandler) StandardizeV2(c) error {
+     scope, err := h.scopeResolver.ResolveBySourceObjectID(c.Context(), id)
+     if err != nil { ... }
+     return h.dispatchSvc.Standardize(c.Context(), scope)
+ }
```

`grep "h.registry\." source_object_actions_handler.go` = 0.

V1 RegistryHandler giữ nguyên (FE legacy có thể vẫn dùng) — gọi cùng `dispatchSvc`.

---

## T15 — P5: Health probe split

### Sketch parallel fan-out

```go
// internal/service/health/collector.go (≤300 line)
func (c *Collector) Snapshot(ctx context.Context) (Snapshot, error) {
    g, ctx := errgroup.WithContext(ctx)
    var (
        worker, kafkaConn, debezium, kafkaLag, nats_, pg, redis_ ProbeResult
    )
    g.Go(func() error { worker, _ = probes.ProbeWorker(ctx, c.cfg.WorkerURL, c.http); return nil })
    g.Go(func() error { kafkaConn, _ = probes.ProbeKafkaConnect(ctx, c.cfg.KafkaConnectURL, c.http); return nil })
    g.Go(func() error { debezium, _ = probes.ProbeDebezium(ctx, c.cfg, c.http); return nil })
    g.Go(func() error { kafkaLag, _ = probes.ProbeKafkaLag(ctx, c.cfg.KafkaExporterURL, c.http); return nil })
    g.Go(func() error { nats_, _ = probes.ProbeNATS(ctx, c.cfg.NatsMonitorURL, c.http); return nil })
    g.Go(func() error { pg, _ = probes.ProbePostgres(ctx, c.db); return nil })
    g.Go(func() error { redis_, _ = probes.ProbeRedis(ctx, c.redis); return nil })
    _ = g.Wait()  // intentionally ignore — partial snapshot OK
    return assemble(worker, kafkaConn, debezium, kafkaLag, nats_, pg, redis_), nil
}
```

Latency snapshot generation: 7 sequential HTTP calls (~5s p95) → parallel (~1s p95).

---

## T16 — P6: V2 sync atomicity

### Diff sketch

`internal/service/source_object_v2_sync.go`:
```diff
- func (s *Service) SyncFromLegacy(ctx, entry) error {
-     // 4 DB call qua s.db
- }
+ func (s *Service) SyncFromLegacyTx(tx *gorm.DB, entry *model.TableRegistry) error {
+     // 4 DB call qua tx (passed in)
+ }
+ // backward compat
+ func (s *Service) SyncFromLegacy(ctx, entry) error {
+     return s.db.Transaction(func(tx *gorm.DB) error {
+         return s.SyncFromLegacyTx(tx, entry)
+     })
+ }
```

`internal/api/registry_handler.go:Register` (line ~140):
```diff
- if err := h.repo.Create(&entry); err != nil { ... }
- if err := h.v2sync.SyncFromLegacy(c.Context(), &entry); err != nil {
-     h.logger.Error("post-register v2 sync failed", ...) // SILENT
- }
+ err := h.db.Transaction(func(tx *gorm.DB) error {
+     if err := tx.Create(&entry).Error; err != nil {
+         return fmt.Errorf("v1 create: %w", err)
+     }
+     if err := h.v2sync.SyncFromLegacyTx(tx, &entry); err != nil {
+         return fmt.Errorf("v2 sync: %w", err)
+     }
+     return nil
+ })
+ if err != nil { return err500(c, err) }
```

Test:
```go
func TestRegister_V2SyncFails_RollbackV1(t *testing.T) {
    db := newTestDB(t)  // dockertest postgres
    h := newHandler(db, mockV2Sync{forceErr: errors.New("simulated")})
    
    resp := h.Register(...)
    require.Equal(t, 500, resp.StatusCode)
    
    var count int64
    db.Table("cdc_table_registry").Count(&count)
    require.Equal(t, int64(0), count)  // V1 rollback verified
}
```

---

## T17 — P7: Test uplift

Plan tests theo phân lớp:
- Repository (`internal/repository/`): sqlmock ưu tiên golden path; live PG container cho complex JOIN.
- Service (`internal/service/`): mock repo + mock NATS + mock Redis; cover 1 golden + 2 error path mỗi method.
- Handler (`internal/api/`): mock service; verify HTTP code + response shape.

Không target 100% coverage — target ≥35% combined repo+service. Effort 2d phù hợp.

---

## Pre-commit checklist (BẮT BUỘC mọi pillar)

```bash
# 1. Build
go build ./... || exit 1

# 2. Test
go test ./... -count=1 -timeout 5m || exit 1

# 3. Vet
go vet ./... || exit 1

# 4. Lint (if golangci-lint installed)
golangci-lint run ./... 2>/dev/null || true

# 5. Smoke endpoint (8 paths)
JWT=$(curl -sX POST http://localhost:8081/auth/login \
  -d '{"username":"admin","password":"..."}' | jq -r .token)

for ep in /health /api/system/health /api/sync/health \
          /api/v1/source-objects /api/mapping-rules \
          /api/v1/system/connectors /api/reconciliation/report \
          /api/v1/masters; do
    code=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $JWT" \
        "http://localhost:8083$ep")
    case "$code" in
        2*|502) echo "PASS $ep $code" ;;
        *)      echo "FAIL $ep $code"; exit 1 ;;
    esac
done

# 6. Security agent (CLAUDE.md §8)
# /security-agent run
```

## Anti-patterns to avoid (đã ghi lessons)

- ❌ Brain trực tiếp sửa code (Lesson #382 #399 #433 — Code Prohibition).
- ❌ Báo Done dựa vào /health (Lesson #1264).
- ❌ Hardcode `"public.X"` SQL khi schema đã move (Lesson #1240).
- ❌ Patch handler quên gán field (Lesson #475).
- ❌ Refactor base stable code không cần thiết (Lesson #160).
- ❌ Argue user rule với "exception" (Lesson #1277).
- ❌ Cross-domain model access trong handler (Lesson #258).
