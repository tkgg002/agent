# Hồ sơ Giải pháp Kỹ thuật: Phase cms_fixes_and_audit

Hồ sơ này mô tả chi tiết các đoạn mã nguồn sẽ thay đổi.

## 1. Thay đổi tại `master_mapping_rule_repo_gorm.go`
Đường dẫn: `cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go`

### 1.1 Hàm `SyncRulesFromShadow`
Loại bỏ điều kiện `AND v2.is_deleted = false` khỏi SQL query.

**Đoạn code trước thay đổi:**
```go
		resInsert := tx.Exec(`
			INSERT INTO cdc_system.mapping_rule_master (
				master_binding_id, mapping_v2_id, target_column, is_active, status, created_by, updated_by, created_at, updated_at
			)
			SELECT ?, v2.id, v2.target_column, true, 'approved', 'shadow-sync', ?, NOW(), NOW()
			FROM cdc_system.mapping_rule_v2 v2
			WHERE v2.shadow_binding_id = ? 
			  AND v2.status = 'approved' 
			  AND v2.is_deleted = false
			  AND NOT EXISTS (
				  SELECT 1 FROM cdc_system.mapping_rule_master m 
				  WHERE m.master_binding_id = ? AND m.target_column = v2.target_column
			  )`,
			masterBindingID, updatedBy, shadowBindingID, masterBindingID,
		)
...
		resRenameNotInMaster := tx.Exec(`
			UPDATE cdc_system.mapping_rule_master m
			SET target_column = v2.target_column,
			    updated_by = ?,
			    updated_at = NOW()
			FROM cdc_system.mapping_rule_v2 v2
			WHERE m.mapping_v2_id = v2.id
			  AND m.master_binding_id = ?
			  AND m.in_master = false
			  AND m.target_column <> v2.target_column
			  AND v2.shadow_binding_id = ?
			  AND v2.is_deleted = false`,
			updatedBy, masterBindingID, shadowBindingID,
		)
...
		resRenameInMaster := tx.Exec(`
			UPDATE cdc_system.mapping_rule_master m
			SET pending_rename_from = COALESCE(m.pending_rename_from, m.target_column),
			    target_column = v2.target_column,
			    status = 'pending',
			    updated_by = ?,
			    updated_at = NOW()
			FROM cdc_system.mapping_rule_v2 v2
			WHERE m.mapping_v2_id = v2.id
			  AND m.master_binding_id = ?
			  AND m.in_master = true
			  AND m.target_column <> v2.target_column
			  AND v2.shadow_binding_id = ?
			  AND v2.is_deleted = false`,
			updatedBy, masterBindingID, shadowBindingID,
		)
```

**Đoạn code sau thay đổi:**
```go
		resInsert := tx.Exec(`
			INSERT INTO cdc_system.mapping_rule_master (
				master_binding_id, mapping_v2_id, target_column, is_active, status, created_by, updated_by, created_at, updated_at
			)
			SELECT ?, v2.id, v2.target_column, true, 'approved', 'shadow-sync', ?, NOW(), NOW()
			FROM cdc_system.mapping_rule_v2 v2
			WHERE v2.shadow_binding_id = ? 
			  AND v2.status = 'approved' 
			  AND NOT EXISTS (
				  SELECT 1 FROM cdc_system.mapping_rule_master m 
				  WHERE m.master_binding_id = ? AND m.target_column = v2.target_column
			  )`,
			masterBindingID, updatedBy, shadowBindingID, masterBindingID,
		)
...
		resRenameNotInMaster := tx.Exec(`
			UPDATE cdc_system.mapping_rule_master m
			SET target_column = v2.target_column,
			    updated_by = ?,
			    updated_at = NOW()
			FROM cdc_system.mapping_rule_v2 v2
			WHERE m.mapping_v2_id = v2.id
			  AND m.master_binding_id = ?
			  AND m.in_master = false
			  AND m.target_column <> v2.target_column
			  AND v2.shadow_binding_id = ?`,
			updatedBy, masterBindingID, shadowBindingID,
		)
...
		resRenameInMaster := tx.Exec(`
			UPDATE cdc_system.mapping_rule_master m
			SET pending_rename_from = COALESCE(m.pending_rename_from, m.target_column),
			    target_column = v2.target_column,
			    status = 'pending',
			    updated_by = ?,
			    updated_at = NOW()
			FROM cdc_system.mapping_rule_v2 v2
			WHERE m.mapping_v2_id = v2.id
			  AND m.master_binding_id = ?
			  AND m.in_master = true
			  AND m.target_column <> v2.target_column
			  AND v2.shadow_binding_id = ?`,
			updatedBy, masterBindingID, shadowBindingID,
		)
```

### 1.2 Hàm `CheckColumnConflict`
Chỉ đếm các rules có `status = 'approved'` và `is_active = true`.

**Đoạn code trước thay đổi:**
```go
func (repo *masterRuleRepoGorm) CheckColumnConflict(ctx context.Context, masterBindingID int64, column string, excludeID int64) (bool, error) {
	var count int64
	err := repo.db.WithContext(ctx).Raw(`
		SELECT count(*) 
		  FROM cdc_system.mapping_rule_master
		 WHERE master_binding_id = ? 
		   AND id <> ? 
		   AND target_column = ?`, masterBindingID, excludeID, column).Scan(&count).Error
	return count > 0, err
}
```

**Đoạn code sau thay đổi:**
```go
func (repo *masterRuleRepoGorm) CheckColumnConflict(ctx context.Context, masterBindingID int64, column string, excludeID int64) (bool, error) {
	var count int64
	err := repo.db.WithContext(ctx).Raw(`
		SELECT count(*) 
		  FROM cdc_system.mapping_rule_master
		 WHERE master_binding_id = ? 
		   AND id <> ? 
		   AND target_column = ?
		   AND status = 'approved'
		   AND is_active = true`, masterBindingID, excludeID, column).Scan(&count).Error
	return count > 0, err
}
```

---

## 2. Thay đổi tại `drop_column.go`
Đường dẫn: `cdc-cms-service/internal/app/commands/master/drop_column.go`

Truyền `rule.ID` thay vì `0` làm `excludeID` để loại trừ chính nó ra khỏi đếm conflict.

**Đoạn code trước thay đổi:**
```go
	// Check xem có rule approved/active nào khác đang chiếm giữ cột này không
	conflict, err := h.repo.CheckColumnConflict(ctx, rule.MasterBindingID, rule.TargetColumn, 0) // excludeID = 0 để check diện rộng
```

**Đoạn code sau thay đổi:**
```go
	// Check xem có rule approved/active nào khác đang chiếm giữ cột này không
	conflict, err := h.repo.CheckColumnConflict(ctx, rule.MasterBindingID, rule.TargetColumn, rule.ID) // excludeID = rule.ID để loại trừ chính nó
```
