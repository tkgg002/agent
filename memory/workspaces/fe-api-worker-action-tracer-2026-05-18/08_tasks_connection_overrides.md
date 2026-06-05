# Tasks — connection_overrides

- [ ] T1: `config/config.go` — `AppConfig.ConnectionOverrides map[string]string` + env scanner `CONNECTION_OVERRIDE_<CODE>`.
- [ ] T2: `config/config-local.yml` — example `connectionOverrides:` block (commented).
- [ ] T3: `internal/service/connection_overrides.go` (NEW) — `ApplyConnectionOverride`, `NormalizeConnectionOverrides`.
- [ ] T4: `internal/service/metadata_registry_service.go` — field `connectionOverrides` + logger; ctor nhận overrides + logger; `GetSourceDSN` check overlay đầu tiên.
- [ ] T5: `internal/handler/command_handler.go` — field + `SetConnectionOverrides`; `scanFieldsMongoSource` check overlay sau `db.First`.
- [ ] T6: `internal/handler/recon_handler.go` — field + `WithConnectionOverrides`; `resolveSourceMongoDSN` check overlay sau `db.First`.
- [ ] T7: `internal/server/worker_server.go` — wire `cfg.ConnectionOverrides` → ctor + setters.
- [ ] T8: `go build ./...` PASS.
- [ ] T9: `go vet ./...` PASS.
- [ ] T10: `go test -count=1 ./internal/handler/... ./internal/service/...` PASS.
- [ ] T11: APPEND `05_progress.md` mỗi file thay đổi.
- [ ] T12: Tạo `09_tasks_solution_connection_overrides.md`.
- [ ] T13: Tạo `report_connection_overrides.md`.
- [ ] T14: APPEND `agent/memory/global/lessons.md`.
