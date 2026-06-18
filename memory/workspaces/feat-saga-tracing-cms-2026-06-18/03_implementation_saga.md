# 03 — Implementation Design: Saga Pattern

## 1. Package Structure

```
internal/
  app/
    saga/
      saga.go          ← Core: Step + Runner
      saga_test.go     ← Unit tests
```

## 2. Core Types — `internal/app/saga/saga.go`

```go
package saga

import (
    "context"
    "fmt"
    "go.uber.org/zap"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    otelTrace "go.opentelemetry.io/otel/trace"
    "cdc-cms-service/pkgs/observability"
)

// Step định nghĩa 1 bước trong saga.
// Compensate là best-effort (nil = no-op).
type Step struct {
    Name       string
    Execute    func(ctx context.Context) error
    Compensate func(ctx context.Context) error
}

// Runner thực thi steps tuần tự với compensation stack.
// Thread-safe: không có shared mutable state — executed slice là local.
type Runner struct {
    name   string        // saga name for tracing (vd: "registry.register")
    steps  []Step
    logger *zap.Logger
}

// New tạo Runner. name dùng cho OTel span "saga.{name}".
func New(name string, logger *zap.Logger, steps ...Step) *Runner {
    if logger == nil {
        logger = zap.NewNop()
    }
    return &Runner{name: name, steps: steps, logger: logger}
}

// Run thực thi tất cả steps theo thứ tự.
// Nếu step[i] fail → compensate step[i-1]..step[0] (reverse, best-effort).
// Returns wrapped error với step name.
func (r *Runner) Run(ctx context.Context) error {
    // Parent span bao toàn bộ saga
    ctx, sagaSpan := observability.StartSpan(ctx, "saga."+r.name)
    defer sagaSpan.End()

    executed := make([]Step, 0, len(r.steps))

    for _, step := range r.steps {
        // Child span cho từng step
        stepCtx, stepSpan := observability.StartSpan(ctx, "saga.step",
            attribute.String("saga.name", r.name),
            attribute.String("saga.step", step.Name),
        )

        r.logger.Debug("saga.step.execute",
            zap.String("saga", r.name),
            zap.String("step", step.Name))

        err := step.Execute(stepCtx)
        if err != nil {
            stepSpan.RecordError(err)
            stepSpan.SetStatus(codes.Error, err.Error())
            stepSpan.End()

            r.logger.Error("saga.step.failed",
                zap.String("saga", r.name),
                zap.String("step", step.Name),
                zap.Int("compensate_count", len(executed)),
                zap.Error(err))

            sagaSpan.RecordError(err)
            sagaSpan.SetStatus(codes.Error, fmt.Sprintf("step %q failed", step.Name))

            r.compensate(ctx, executed)
            return fmt.Errorf("saga %q step %q: %w", r.name, step.Name, err)
        }

        stepSpan.End()
        executed = append(executed, step)
    }
    return nil
}

// compensate chạy compensation ngược từ cuối về đầu (best-effort).
// Không return error: compensation failure chỉ log, không panic.
func (r *Runner) compensate(ctx context.Context, executed []Step) {
    for i := len(executed) - 1; i >= 0; i-- {
        step := executed[i]
        if step.Compensate == nil {
            continue
        }
        r.logger.Warn("saga.compensate",
            zap.String("saga", r.name),
            zap.String("step", step.Name))

        if cErr := step.Compensate(ctx); cErr != nil {
            // CRITICAL: compensation failure cần manual intervention
            r.logger.Error("saga.compensate.failed — MANUAL ACTION REQUIRED",
                zap.String("saga", r.name),
                zap.String("step", step.Name),
                zap.Error(cErr))
        }
    }
}
```

## 3. Saga Implementations theo từng luồng

### S1: `registry.register` (Source/register_registry.go)

**Steps**:
1. `register-db` — `sourceRepo.Register()` → Compensate: `sourceRepo.DeleteRegistry()`
2. `ensure-shadow-ddl` — `automator.EnsureShadowTable()` → Compensate: `automator.DropShadowTable()` *(nếu có)*
3. `nats-reload` — `nats.PublishReload()` → Compensate: nil (fire-and-forget)

**Trước**: Manual rollback inline (không nhất quán, không log chuẩn)  
**Sau**: `saga.New("registry.register", h.log, step1, step2, step3).Run(ctx)`

---

### S2: `approve_schema_proposal` (Governance)

**Steps**:
1. `validate-state` — kiểm tra proposal status = pending → Compensate: nil
2. `shadow-alter-column` — `repo.AlterShadowColumn()` → Compensate: `repo.RevertShadowColumn()`
3. `master-alter-column` — `repo.AlterMasterColumn()` → Compensate: `repo.RevertMasterColumn()`
4. `insert-mapping-rules` — `repo.InsertMappingRules()` → Compensate: `repo.DeleteMappingRules()`
5. `mark-proposal-approved` — `repo.MarkApproved()` → Compensate: `repo.MarkPending()`

> **⚠️ Open Question Q1**: Nếu RevertShadowColumn gặp conflict data → strategy? (mark `compensation_failed` vs hard DROP COLUMN)

---

### S3: `master.create` (Master)

**Steps**:
1. `resolve-shadow-binding` — `masterRepo.ResolveShadowBinding()` → Compensate: nil (read-only)
2. `create-master-binding` — `masterRepo.CreateMasterBinding()` → Compensate: `masterRepo.DeleteMasterBinding()`
3. `clone-mapping-rules` — `masterRepo.CloneMappingRules()` → Compensate: `masterRepo.DeleteClonedRules()`

---

### S4: `master.approve` (Governance/approve_master.go)

**Steps**:
1. `approve-schema-tx` — `repo.ApproveSchemaTx()` → Compensate: `repo.RevertSchemaTx()`
2. `publish-master-create` — `publisher.Publish("cdc.cmd.master-create")` → Compensate: nil (NATS fire-and-forget)

---

### S5: `connector.create` (Source/debezium_connector.go)

**Steps**:
1. `save-fingerprint-db` — `repo.SaveFingerprint()` → Compensate: `repo.DeleteFingerprint()`
2. `create-kafka-connector` — `w.CreateConnector()` → Compensate: `w.DeleteConnector()`

> **⚠️ Open Question Q2**: Thứ tự hiện tại — DB trước hay KafkaConnect API trước? Cần verify để set đúng compensation.

## 4. Port Interface Bổ Sung

Thêm vào `internal/app/ports/repository.go` — interface `MasterRepo`:

```go
// RevertSchemaTx reverts schema_status từ 'approved' về 'pending_review'.
// Dùng cho saga compensation bước S4.
RevertSchemaTx(ctx context.Context, masterName, updatedBy string) error

// DeleteMasterBinding xóa binding theo ID.
// Dùng cho saga compensation bước S3.
DeleteMasterBinding(ctx context.Context, bindingID int64) error

// DeleteClonedRules xóa toàn bộ rules được clone từ shadow vào master binding.
// Dùng cho saga compensation bước S3.
DeleteClonedRules(ctx context.Context, masterBindingID int64) error
```
