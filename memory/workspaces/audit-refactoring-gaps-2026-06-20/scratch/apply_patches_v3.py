import os

def patch_master_repo(filepath):
    if not os.path.exists(filepath):
        print(f"File not found: {filepath}")
        return False
        
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()
        
    original = content
    
    # 1. Patch SyncRulesFromShadow
    # Query 1
    old1 = """			FROM cdc_system.mapping_rule_v2 v2
			WHERE v2.shadow_binding_id = ? 
			  AND v2.status = 'approved' 
			  AND v2.is_deleted = false
			  AND NOT EXISTS ("""
    new1 = """			FROM cdc_system.mapping_rule_v2 v2
			WHERE v2.shadow_binding_id = ? 
			  AND v2.status = 'approved' 
			  AND NOT EXISTS ("""
              
    # Query 2
    old2 = """			  AND m.target_column <> v2.target_column
			  AND v2.shadow_binding_id = ?
			  AND v2.is_deleted = false`,"""
    new2 = """			  AND m.target_column <> v2.target_column
			  AND v2.shadow_binding_id = ?`,"""
              
    # Query 3
    old3 = """			  AND m.target_column <> v2.target_column
			  AND v2.shadow_binding_id = ?
			  AND v2.is_deleted = false`,"""
    # (Lưu ý: old2 và old3 giống nhau, nhưng ta replace từng cái hoặc replace_all nếu dùng replace)
    
    # 2. Patch CheckColumnConflict
    old_conflict = """		SELECT count(*) 
		  FROM cdc_system.mapping_rule_master
		 WHERE master_binding_id = ? 
		   AND id <> ? 
		   AND target_column = ?`, masterBindingID, excludeID, column).Scan(&count).Error"""
           
    new_conflict = """		SELECT count(*) 
		  FROM cdc_system.mapping_rule_master
		 WHERE master_binding_id = ? 
		   AND id <> ? 
		   AND target_column = ?
		   AND status = 'approved'
		   AND is_active = true`, masterBindingID, excludeID, column).Scan(&count).Error"""

    # Áp dụng thay thế
    # Vì old2 và old3 giống nhau nên replace(old2, new2) sẽ xử lý cả hai.
    # Nhưng ta cần kiểm tra xem các chuỗi này có tồn tại trong file hay không.
    
    if old1 not in content:
        print("Error: old1 pattern not found in file.")
        # Hãy in ra một đoạn xung quanh shadow_binding_id để debug nếu cần
        return False
        
    if old2 not in content:
        print("Error: old2 pattern not found in file.")
        return False
        
    if old_conflict not in content:
        print("Error: old_conflict pattern not found in file.")
        return False
        
    content = content.replace(old1, new1)
    content = content.replace(old2, new2)
    content = content.replace(old_conflict, new_conflict)
    
    with open(filepath, "w", encoding="utf-8") as f:
        f.write(content)
        
    print(f"Successfully patched {filepath} using precise string replacement!")
    return True

file1 = "/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go"
patch_master_repo(file1)
