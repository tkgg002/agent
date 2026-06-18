# Todo List: Refactoring Master Mapping Rule Handler

- [x] Di chuyển utility validators ra `pkgs/utils/pg_validator.go` và `internal/naming/naming.go`
- [x] Định nghĩa Ports & interfaces (`MasterDDLPublisher` và bổ sung methods vào `MasterRuleRepository`)
- [x] Triển khai `internal/infra/persistence/master_rule_repo_gorm.go`
- [x] Triển khai `internal/infra/messaging/nats_master_ddl_publisher.go`
- [x] Triển khai `internal/app/queries/mapping/list_master_rules.go`
- [x] Triển khai `internal/app/commands/mapping/` use cases:
  - [x] `save_master_rule.go`
  - [x] `update_master_rule_staging.go`
  - [x] `approve_master_ddl.go`
  - [x] `drop_master_column.go`
  - [x] `sync_master_rules_from_shadow.go`
  - [x] `batch_update_master_rules.go`
- [x] Clean up và làm mỏng `internal/api/master_mapping_rule_handler.go`
- [x] Wire server.go dependencies và chạy build/tests
