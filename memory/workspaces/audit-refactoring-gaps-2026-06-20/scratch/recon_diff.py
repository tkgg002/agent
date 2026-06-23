import re
import os
import difflib

def extract_function_bodies(filepath):
    if not os.path.exists(filepath):
        return {}
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
        
    # We want to find func declarations and capture their body by matching braces { }
    # Let's find all occurrences of func
    matches = list(re.finditer(r'func\s+(?:\([^)]+\)\s+)?([A-Z][a-zA-Z0-9_]*)\s*\(', content))
    
    funcs = {}
    for i, match in enumerate(matches):
        func_name = match.group(1)
        start_pos = match.start()
        
        # Find the opening brace of the function body
        brace_pos = content.find('{', match.end())
        if brace_pos == -1:
            continue
            
        # Match braces to find the end of the function body
        brace_count = 1
        curr_pos = brace_pos + 1
        while brace_count > 0 and curr_pos < len(content):
            char = content[curr_pos]
            if char == '{':
                brace_count += 1
            elif char == '}':
                brace_count -= 1
            curr_pos += 1
            
        body = content[start_pos:curr_pos]
        funcs[func_name] = body
        
    return funcs

old_file = "/Users/trainguyen/Documents/work/data-hub-bf/centralized-data-service/internal/service/recon_core.go"
new_files = {
    "engine": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go",
    "tier_a": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go",
    "tier_b": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go"
}

old_funcs = extract_function_bodies(old_file)
new_funcs = {}
for name, path in new_files.items():
    new_funcs.update(extract_function_bodies(path))

diffs_found = 0
for func_name, old_body in old_funcs.items():
    if func_name not in new_funcs:
        print(f"Function {func_name} is missing in new files.")
        continue
    new_body = new_funcs[func_name]
    
    # Normalize whitespaces for a quick comparison
    old_norm = re.sub(r'\s+', ' ', old_body).strip()
    new_norm = re.sub(r'\s+', ' ', new_body).strip()
    
    if old_norm != new_norm:
        diffs_found += 1
        print(f"\n==================================================")
        print(f"DIFF FOUND IN FUNCTION: {func_name}")
        print(f"==================================================")
        
        # Generate line-by-line diff
        old_lines = old_body.splitlines(keepends=True)
        new_lines = new_body.splitlines(keepends=True)
        
        diff = difflib.unified_diff(old_lines, new_lines, fromfile='old_recon_core.go', tofile='new_recon_files')
        print(''.join(diff))

print(f"\nAudit complete. Total functions with diffs: {diffs_found}")
