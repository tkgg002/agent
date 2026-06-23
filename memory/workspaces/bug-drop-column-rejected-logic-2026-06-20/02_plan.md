# Plan: Sửa lỗi logic Drop Column đối với Rule bị Rejected

## Kế hoạch thực hiện chi tiết

### Bước 1: Khai báo cổng nghiệp vụ mới trong Domain Repository
- **File**: `internal/domain/master/repository.go`
- **Thay đổi**: Thêm phương thức `CheckColumnInUseByActiveRules` vào interface `MasterRuleRepository`:
  ```go
  CheckColumnInUseByActiveRules(ctx context.Context, bindingID int64, columnName string, excludeID int64) (bool, error)
  ```

### Bước 2: Triển khai phương thức trong persistence adapter (GORM)
- **File**: `internal/infra/persistence/master/master_mapping_rule_repo_gorm.go`
- **Thay đổi**: Triển khai `CheckColumnInUseByActiveRules`. Câu SQL query:
  ```sql
  SELECT count(*) 
    FROM cdc_system.mapping_rule_master
   WHERE master_binding_id = ? 
     AND id <> ? 
     AND target_column = ?
     AND status = 'approved'
     AND is_active = true
  ```
  Nếu count > 0, trả về true.

### Bước 3: Sửa logic drop column trong handler
- **File**: `internal/app/commands/master/drop_column.go`
- **Thay đổi**: Thay thế lời gọi `CheckColumnConflict` thành `CheckColumnInUseByActiveRules` và truyền `rule.ID` thay vì `0` để loại trừ chính rule hiện tại ra khỏi kiểm tra xung đột:
  ```diff
  - conflict, err := h.repo.CheckColumnConflict(ctx, rule.MasterBindingID, rule.TargetColumn, 0)
  + conflict, err := h.repo.CheckColumnInUseByActiveRules(ctx, rule.MasterBindingID, rule.TargetColumn, rule.ID)
  ```

### Bước 4: Viết Unit Test cho DropColumnHandler
- **File**: `internal/app/commands/master/drop_column_test.go` [NEW]
- **Nội dung**: Viết các ca kiểm thử bằng mock repository:
  1. `TestDropColumn_Success`: Rule bị reject, không có rule approved+active khác trùng cột. Phải drop thành công.
  2. `TestDropColumn_Conflict`: Rule bị reject, nhưng có rule approved+active khác trùng cột. Phải trả về lỗi `"cột đang được rule approved+active khác dùng — không drop"`.
  3. `TestDropColumn_NotRejected`: Rule không ở trạng thái rejected. Phải trả về lỗi `ErrRuleNotRejected`.
  4. `TestDropColumn_SystemColumn`: Cột hệ thống `_*`. Phải trả về lỗi `ErrSystemColumnDrop`.

### Bước 5: Kiểm tra và xác nhận
- Biên dịch dự án: `go build ./...`
- Chạy toàn bộ unit test: `go test ./...`
- Chạy trực tiếp test của commands/master: `go test -v ./internal/app/commands/master/...`

### Bước 6: Cập nhật tài liệu quản trị dự án
- Tạo `03_implementation_drop_column_fix.md`
- Tạo `report_drop_column_logic_fix.md`
- Cập nhật `05_progress.md`
