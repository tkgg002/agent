# Active Plan: Refactoring Master Mapping Rule Handler

## Kế hoạch thực hiện (High-Level Steps)
1. **Bước 1**: Tạo các hàm validators dùng chung `pkgs/utils/pg_validator.go` (`IsValidDataType`, `IsValidColumnName`) và cập nhật `internal/naming/naming.go` (`IsSystemColumn`).
2. **Bước 2**: Khai báo Interface Port `internal/app/ports/master_ddl_publisher.go` và bổ sung các methods mới cho `mapping.MasterRuleRepository` trong `internal/domain/mapping/master_rule.go`.
3. **Bước 3**: Triển khai tầng Infrastructure:
   - Viết các methods mới cho `master_rule_repo_gorm.go` trong `internal/infra/persistence/`.
   - Tạo `nats_master_ddl_publisher.go` trong `internal/infra/messaging/` triển khai `MasterDDLPublisher`.
4. **Bước 4**: Triển khai các Use Cases tầng Application:
   - Read Model: `internal/app/queries/mapping/list_master_rules.go`
   - Write Model: `save_master_rule.go`, `update_master_rule_staging.go`, `approve_master_ddl.go`, `drop_master_column.go`, `sync_master_rules_from_shadow.go`, `batch_update_master_rules.go`
5. **Bước 5**: Tái cấu trúc file API Handler `internal/api/master_mapping_rule_handler.go` chỉ còn giữ nhiệm vụ chuyển tiếp HTTP Fiber.
6. **Bước 6**: Wire dependencies trong `internal/server/server.go` và kiểm tra biên dịch (`go build ./...`) cùng unit tests.
