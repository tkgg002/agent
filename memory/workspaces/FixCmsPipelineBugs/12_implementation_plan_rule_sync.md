# Kế hoạch triển khai - Đồng bộ Mapping Rules khi Approve Master

Kế hoạch này giải quyết lỗi mapping rules bị clone sang `mapping_rule_master` dưới dạng status `'pending'` thay vì thừa hưởng status của `mapping_rule_v2` (thường là `'approved'`), dẫn đến việc transmuter không tìm thấy approved rules và ném lỗi `no approved mapping rules found`.

## Thay đổi kỹ thuật đề xuất

### 1. `cdc-cms-service` (Persistence Layer)
- **Tệp tin**: `internal/infra/persistence/master/master_repo_gorm.go`
- **Sửa đổi**:
  - Trong phương thức `ApproveSchemaTx`: Thay thế `'pending'` gán cứng trong câu lệnh `INSERT INTO cdc_system.mapping_rule_master` thành `v2.status` để thừa kế trạng thái từ shadow rule.
  - Trong phương thức `CloneMappingRules`: Tương tự, đổi `'pending'` thành `v2.status`.

### 2. Integration Tests
- **Tệp tin**: `test/internal/app/commands/approve_master_test.go`
  - Sửa lỗi compilation: import `persisMaster "cdc-cms-service/internal/infra/persistence/master"`.
  - Khởi tạo repo `persisMaster.NewMasterRepo(db)` truyền vào handler thay vì truyền raw `db` (`*gorm.DB`).
  - Sửa đổi assertion kết quả trả về của handler thành `approved` thay vì `approved_but_dispatch_failed` (do publisher nil không còn trả về lỗi trong thiết kế mới).
- **Tệp tin**: `test/internal/app/commands/approve_schema_proposal_integration_test.go`
  - Sửa lỗi compilation tương tự: import `persisGov "cdc-cms-service/internal/infra/persistence/governance"`.
  - Khởi tạo repo `persisGov.NewSchemaProposalRepo(db)` và truyền vào handler.
