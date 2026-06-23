# Thiết kế kỹ thuật: Phase cms_fixes_and_audit

## 1. Sửa lỗi Sync Shadow
- **File**: `internal/infra/persistence/master/master_mapping_rule_repo_gorm.go`
- **Hàm**: `SyncRulesFromShadow`
- **Sửa đổi**: Xoá bỏ dòng `AND v2.is_deleted = false` khỏi 3 câu query:
  1. Query Insert (dòng ~250)
  2. Query RenameNotInMaster (dòng ~272)
  3. Query RenameInMaster (dòng ~292)

## 2. Sửa lỗi Drop Column
- **File**: `internal/infra/persistence/master/master_mapping_rule_repo_gorm.go`
- **Hàm**: `CheckColumnConflict`
- **Sửa đổi**: Thêm điều kiện `status = 'approved'` và `is_active = true` vào SQL query.
  ```sql
  SELECT count(*) 
    FROM cdc_system.mapping_rule_master
   WHERE master_binding_id = ? 
     AND id <> ? 
     AND target_column = ?
     AND status = 'approved'
     AND is_active = true
  ```
- **File**: `internal/app/commands/master/drop_column.go`
- **Hàm**: `Handle`
- **Sửa đổi**: Thay thế đối số `0` bằng `rule.ID` trong lệnh gọi `CheckColumnConflict`.
  ```go
  conflict, err := h.repo.CheckColumnConflict(ctx, rule.MasterBindingID, rule.TargetColumn, rule.ID)
  ```

## 3. Quy trình Triển khai sửa code (Muscle Execution)
- Vì Brain không trực tiếp chạm tay vào source code, Brain sẽ tạo một scratch script Python `/Users/trainguyen/Documents/work/agent/memory/workspaces/audit-refactoring-gaps-2026-06-20/scratch/apply_patches.py` để áp dụng các thay đổi này một cách an toàn.
- Sau đó Brain ra lệnh cho Muscle chạy script này qua `run_command`.
