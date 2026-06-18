# 08 — Tasks: Phase 1 — Core Infrastructure

## Phase 1: Saga Core + OTel Helpers

### T1.1 — Tạo `internal/app/saga/saga.go`
- **File**: `internal/app/saga/saga.go` [NEW]
- **LOC**: ~90
- **Content**: Step struct, Runner struct, New(), Run(), compensate()
- **Notes**: Bao gồm OTel span per-step (import observability package)
- **DoD**: compile, không có unused imports

### T1.2 — Tạo `internal/app/saga/saga_test.go`
- **File**: `internal/app/saga/saga_test.go` [NEW]
- **LOC**: ~150
- **Content**: TC-S1 → TC-S5 (xem 06_test_cases.md)
- **DoD**: `go test ./internal/app/saga/... -v` → 5 cases PASS

### T1.3 — Thêm `EndSpan` + `Ctx` vào `pkgs/observability/otel.go`
- **File**: `pkgs/observability/otel.go` [MODIFY]
- **LOC delta**: +30
- **Content**: hàm EndSpan(span, *error), hàm Ctx(ctx, *zap.Logger)
- **Import thêm**: `"go.opentelemetry.io/otel/codes"`
- **DoD**: `go build ./pkgs/observability/...` pass

### T1.4 — Set W3C TextMapPropagator trong `InitOtel()`
- **File**: `pkgs/observability/otel.go` [MODIFY]
- **LOC delta**: +8
- **Content**: `otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(...))`
- **DoD**: propagator không nil khi gọi `otel.GetTextMapPropagator()`

### T1.5 — Tạo `internal/middleware/otel_propagator.go`
- **File**: `internal/middleware/otel_propagator.go` [NEW]
- **LOC**: ~35
- **Content**: `OtelPropagator() fiber.Handler` — W3C extract + inject ctx
- **DoD**: compile, fiber handler signature đúng

### T1.6 — Register middleware trong `internal/server/server.go`
- **File**: `internal/server/server.go` [MODIFY]
- **LOC delta**: +3 (import + 1 dòng `app.Use(...)`)
- **Content**: `app.Use(middleware.OtelPropagator())` trước routes
- **DoD**: compile, middleware được gọi trước handlers

### T1.7 — Instrument `CommandBus.Execute` + `Dispatch`
- **File**: `internal/infra/messaging/nats_command_bus.go` [MODIFY]
- **LOC delta**: +15
- **Content**: named return, StartSpan, defer EndSpan trong Execute và Dispatch
- **DoD**: `go test ./internal/infra/messaging/... -v` không regress

---

## Phase 2: Source Group Saga

### T2.1 — Refactor `register_registry.go`
- **File**: `internal/app/commands/source/register_registry.go` [MODIFY]
- **LOC delta**: +35
- **Content**: Thay manual rollback → `saga.New("registry.register", ...)` 3 steps
- **Step 1**: register-db (Register + DeleteRegistry)
- **Step 2**: ensure-shadow-ddl (EnsureShadowTable + DropShadowTable nếu có)
- **Step 3**: nats-reload (PublishReload + nil compensate)
- **DoD**: compile, behavior giữ nguyên khi các bước đều pass

### T2.2 — Refactor `debezium_connector.go` (Create + Delete)
- **File**: `internal/app/commands/source/debezium_connector.go` [MODIFY]
- **LOC delta**: +40
- **Content**: saga 2 steps cho Create và Delete connector
- **Lưu ý**: Cần confirm Q2 trước (thứ tự DB vs KafkaConnect)
- **DoD**: compile, behavior giữ nguyên khi happy path

---

## Phase 3: Governance + Master Saga

### T3.1 — Thêm methods vào `ports.MasterRepo` interface
- **File**: `internal/app/ports/repository.go` [MODIFY]
- **LOC delta**: +8
- **Methods**: RevertSchemaTx, DeleteMasterBinding, DeleteClonedRules

### T3.2 — Implement 3 methods mới trong master_repo_gorm.go
- **File**: `internal/infra/persistence/master/master_repo_gorm.go` [MODIFY]
- **LOC delta**: +60
- **DoD**: compile, method signatures khớp interface

### T3.3 — Refactor `approve_master.go`
- **File**: `internal/app/commands/governance/approve_master.go` [MODIFY]
- **LOC delta**: +30
- **Content**: saga.New("master.approve") 2 steps

### T3.4 — Refactor `approve_schema_proposal.go`
- **File**: `internal/app/commands/governance/approve_schema_proposal.go` [MODIFY]
- **LOC delta**: +60
- **Content**: saga.New("schema-proposal.approve") 4 steps
- **Lưu ý**: Cần confirm Q1 trước (revert strategy)

### T3.5 — Refactor `create_master.go`
- **File**: `internal/app/commands/master/create_master.go` [MODIFY]
- **LOC delta**: +35
- **Content**: saga.New("master.create") 3 steps

---

## Phase 4: Verification & Report

### T4.1 — Full build + vet
```bash
go build ./... && go vet ./...
```

### T4.2 — Full test
```bash
go test ./... -count=1
```

### T4.3 — Tạo `report_saga_tracing_2026-06-18.md`
Dựa trên kết quả thực tế sau khi Muscle execute xong:
- Danh sách files thay đổi
- LOC delta thực tế (diff)
- Test results summary
