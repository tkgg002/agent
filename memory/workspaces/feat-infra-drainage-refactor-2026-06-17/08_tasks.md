## Task: Refactor and Drainage of DB & NATS from API and App Layers
- **Phase**: GĐ0
- **Service Group**: Utilities
- **Service(s)**: cdc-cms-service
- **Mô tả**: Decouple API and App layers from direct GORM/NATS dependencies.
- **Trạng thái**: [x] DONE

### [Context]
- Current state: Refactoring and draining GORM/NATS dependencies in `cdc-cms-service`.
- Dependencies: `internal/api`, `internal/app`, `internal/infra`, `internal/server`.
- ADR liên quan: standard Hexagonal Architecture rules.

### [Definition of Done]
- [x] Interface `ReloadPublisher` added in `internal/app/ports/publisher.go`.
- [x] Constructor `NewReloadPublisher` and interface implementation added in `internal/infra/messaging/nats_publisher.go`.
- [x] `gorm.ErrRecordNotFound` mapped to `ports.ErrRecordNotFound` in `bridge_status_repo_gorm.go` and `job_repo_gorm.go`.
- [x] All `gorm.ErrRecordNotFound` references in `internal/api/` changed to `ports.ErrRecordNotFound`.
- [x] All `*nats.Conn` and `*natsconn.NatsClient` in `internal/api/` and `internal/app/` changed to `ports.Publisher` or `ports.ReloadPublisher`.
- [x] Unused `nats` removed from `MasterRegistryHandler`.
- [x] `internal/server/server.go` updated to wire the new interfaces.
- [x] Project compiles cleanly: `go build ./...` (inside `cdc-cms-service`).
- [x] Tests pass: `go test ./...` (inside `cdc-cms-service`).
- [x] **[QA Gate]**: Tests verified.
- [x] **[Security Gate]**: Security agent verified.
- [x] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.

### [Remediation & Final Integrity Audit]
- [x] Audit for leftover `natsconn.NatsClient` in `internal/app/commands/` -> Completed, 4 remaining files identified.
- [x] Decoupled NATS client from remaining App Command Handlers:
  - [x] Refactored `internal/app/commands/source/register_registry.go`
  - [x] Refactored `internal/app/commands/source/bulk_register_registry.go`
  - [x] Refactored `internal/app/commands/source/update_registry.go`
  - [x] Refactored `internal/app/commands/master/update_mapping_rule.go`
- [x] Wiring updated in `internal/server/server.go` to use `reloadPublisher`.
- [x] Verify compile `go build ./...` passes.
- [x] Verify test suite `go test ./...` passes.
