import re
import os

def extract_functions(filepath):
    if not os.path.exists(filepath):
        return []
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Regex to find func declarations in Go: func (receiver) Name(args) (returns) { or func Name(args) (returns) {
    # We will look for func definitions
    pattern = r'func\s+(?:\([^)]+\)\s+)?([A-Z][a-zA-Z0-9_]*)\s*\('
    funcs = re.findall(pattern, content)
    return funcs

old_file = "/Users/trainguyen/Documents/work/data-hub-bf/centralized-data-service/internal/service/recon_core.go"
new_files = {
    "engine": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go",
    "tier_a": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go",
    "tier_b": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go"
}

old_funcs = set(extract_functions(old_file))
new_funcs = {}
all_new_funcs = set()
for name, path in new_files.items():
    funcs = set(extract_functions(path))
    new_funcs[name] = funcs
    all_new_funcs.update(funcs)

print(f"Total functions in old recon_core.go: {len(old_funcs)}")
print(f"Total functions in new files: {len(all_new_funcs)}")
print(f"  - Engine: {len(new_funcs['engine'])}")
print(f"  - Tier A: {len(new_funcs['tier_a'])}")
print(f"  - Tier B: {len(new_funcs['tier_b'])}")

missing_in_new = old_funcs - all_new_funcs
newly_added = all_new_funcs - old_funcs

print("\n--- Functions in old file but MISSING in new files ---")
for f in sorted(missing_in_new):
    print(f"- {f}")

print("\n--- Newly added functions in new files ---")
for f in sorted(newly_added):
    # Find which file it is in
    loc = []
    for name, funcs in new_funcs.items():
        if f in funcs:
            loc.append(name)
    print(f"- {f} (in {', '.join(loc)})")

print("\n--- Common functions mapped to new files ---")
common = old_funcs & all_new_funcs
for f in sorted(common):
    loc = []
    for name, funcs in new_funcs.items():
        if f in funcs:
            loc.append(name)
    print(f"- {f} -> {', '.join(loc)}")
