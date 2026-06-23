import os

def patch_master_repo(filepath):
    if not os.path.exists(filepath):
        print(f"File not found: {filepath}")
        return False
        
    with open(filepath, "r", encoding="utf-8") as f:
        lines = f.readlines()
        
    new_lines = []
    i = 0
    modified = False
    
    while i < len(lines):
        line = lines[i]
        
        # 1. Check for SyncRulesFromShadow queries containing v2.is_deleted = false
        if "v2.is_deleted = false" in line:
            # Case 1: end of query with `,
            # e.g., "  AND v2.is_deleted = false`,"
            if "`," in line:
                # We need to append the closing backtick and comma to the previous non-empty line
                # and skip the current line
                j = len(new_lines) - 1
                while j >= 0 and not new_lines[j].strip():
                    j -= 1
                if j >= 0:
                    # Strip any trailing whitespace/newlines from the target line first
                    prev_line = new_lines[j].rstrip()
                    # Append `,\n
                    new_lines[j] = prev_line + "`,\n"
                    modified = True
                    print(f"Removed is_deleted from query end at line {i+1}, appended `, to line {j+1}")
                else:
                    new_lines.append(line)
            else:
                # Case 2: middle of query, just skip this line
                print(f"Removed is_deleted from query middle at line {i+1}")
                modified = True
            i += 1
            continue
            
        # 2. Check for CheckColumnConflict SQL query
        if "SELECT count(*)" in line and i + 4 < len(lines):
            # Check if this is the target block in CheckColumnConflict
            block = "".join(lines[i:i+6])
            if "cdc_system.mapping_rule_master" in block and "target_column = ?" in block and "status = 'approved'" not in block:
                # Insert the status and is_active filters before the scan/params
                # Let's find target_column = ? line
                k = i
                while k < i + 6:
                    if "target_column = ?" in lines[k]:
                        # Append the new filters to target_column line (or insert after it)
                        # We preserve indentation
                        indent = " " * (len(lines[k]) - len(lines[k].lstrip()))
                        # Check if it has comma/backtick at the end
                        content = lines[k].rstrip()
                        # If it ends with `, we need to place the filters before `,
                        if content.endswith("`"):
                            lines[k] = content[:-1] + "\n" + indent + "  AND status = 'approved'\n" + indent + "  AND is_active = true`" + "\n"
                        elif content.endswith("`,"):
                            lines[k] = content[:-2] + "\n" + indent + "  AND status = 'approved'\n" + indent + "  AND is_active = true`,\n"
                        else:
                            lines[k] = content + "\n" + indent + "  AND status = 'approved'\n" + indent + "  AND is_active = true\n"
                        modified = True
                        print(f"Patched CheckColumnConflict query at line {k+1}")
                        break
                    k += 1
                    
        new_lines.append(line)
        i += 1
        
    if modified:
        with open(filepath, "w", encoding="utf-8") as f:
            f.writelines(new_lines)
        print(f"Successfully patched {filepath}")
        return True
    else:
        print(f"No changes made to {filepath}")
        return False

# Execute patching
file1 = "/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go"
success = patch_master_repo(file1)
if success:
    print("Patch applied successfully via smart patcher!")
else:
    print("Failed to apply patch via smart patcher.")
