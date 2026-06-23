import os

def patch_file(filepath, replacements):
    if not os.path.exists(filepath):
        print(f"File not found: {filepath}")
        return False
        
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()
        
    original = content
    for old, new in replacements:
        if old not in content:
            print(f"Target content not found in {filepath} for replacement.")
            # print(f"Looking for:\n{old}\n")
            return False
        content = content.replace(old, new)
        
    if content == original:
        print(f"No changes made to {filepath}")
        return False
        
    with open(filepath, "w", encoding="utf-8") as f:
        f.write(content)
    print(f"Successfully patched {filepath}")
    return True

# 1. Patch master_mapping_rule_repo_gorm.go
file1 = "/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go"
replacements1 = [
    (
        # Replacement 1: SyncRulesFromShadow - Insert
        "v2.shadow_binding_id = ? \n\t\t\t  AND v2.status = 'approved' \n\t\t\t  AND v2.is_deleted = false",
        "v2.shadow_binding_id = ? \n\t\t\t  AND v2.status = 'approved'"
    ),
    (
        # Replacement 2: SyncRulesFromShadow - RenameNotInMaster
        "AND v2.shadow_binding_id = ?\n\t\t\t  AND v2.is_deleted = false",
        "AND v2.shadow_binding_id = ?"
    ),
    (
        # Replacement 3: SyncRulesFromShadow - RenameInMaster
        "AND v2.shadow_binding_id = ?\n\t\t\t  AND v2.is_deleted = false`",
        "AND v2.shadow_binding_id = ?`"
    ),
    (
        # Replacement 4: CheckColumnConflict SQL
        """		SELECT count(*) 
		  FROM cdc_system.mapping_rule_master
		 WHERE master_binding_id = ? 
		   AND id <> ? 
		   AND target_column = ?""",
        """		SELECT count(*) 
		  FROM cdc_system.mapping_rule_master
		 WHERE master_binding_id = ? 
		   AND id <> ? 
		   AND target_column = ?
		   AND status = 'approved'
		   AND is_active = true"""
    )
]

# 2. Patch drop_column.go
file2 = "/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/drop_column.go"
replacements2 = [
    (
        # Replacement 1: excludeID
        "CheckColumnConflict(ctx, rule.MasterBindingID, rule.TargetColumn, 0)",
        "CheckColumnConflict(ctx, rule.MasterBindingID, rule.TargetColumn, rule.ID)"
    )
]

success1 = patch_file(file1, replacements1)
success2 = patch_file(file2, replacements2)

if success1 and success2:
    print("All patches applied successfully!")
else:
    print("Failed to apply patches. Please check the replacements and file paths.")
