#!/usr/bin/env python3
import os
import sys
import re

def log(msg):
    print(f"[SOP_PATCHER] {msg}")

def apply_patch(workflow_path, target_content, replacement_content):
    if not os.path.exists(workflow_path):
        log(f"Error: Target workflow file {workflow_path} does not exist.")
        return False

    with open(workflow_path, "r", encoding="utf-8") as f:
        content = f.read()

    # Normalize target content to avoid spacing/newline mismatches
    # Let's try direct string replace first
    if target_content not in content:
        log(f"Warning: Target content not found in {workflow_path}.")
        # Let's try a backup search with spacing normalization
        normalized_target = re.sub(r'\s+', ' ', target_content.strip())
        normalized_content = re.sub(r'\s+', ' ', content)
        if normalized_target not in normalized_content:
            log(f"Error: Target content could not be matched even with normalization in {workflow_path}.")
            return False
        else:
            log(f"Warning: Match found only via spacing normalization. Exact patch might fail.")

    occurrences = content.count(target_content)
    if occurrences > 1:
        log(f"Error: Target content is not unique in {workflow_path} ({occurrences} matches). Refusing to patch.")
        return False

    new_content = content.replace(target_content, replacement_content)
    with open(workflow_path, "w", encoding="utf-8") as f:
        f.write(new_content)

    log(f"Successfully patched {workflow_path}.")
    return True

def parse_progress_file(progress_path):
    if not os.path.exists(progress_path):
        log(f"Progress file {progress_path} not found.")
        return []

    patches = []
    # Pattern: [SOP_PATCH] workflow_name.md | TargetContent | ReplacementContent
    patch_pattern = re.compile(r'\[SOP_PATCH\]\s*([a-zA-Z0-9_\-\.]+)\s*\|\s*(.*?)\s*\|\s*(.*)')

    with open(progress_path, "r", encoding="utf-8") as f:
        for line in f:
            match = patch_pattern.search(line)
            if match:
                workflow_file = match.group(1).strip()
                target = match.group(2).strip()
                replacement = match.group(3).strip()
                patches.append((workflow_file, target, replacement))

    return patches

def run_mock_test():
    log("Running mock test...")
    mock_workflow_path = "agent/workflows/mock_sop.md.tmp"
    mock_content = """# Mock Workflow
This is a test workflow file.
- Step 1: Initial step
- Step 2: Error-prone step to be optimized
- Step 3: Final step
"""
    # Create mock workflow file
    os.makedirs(os.path.dirname(mock_workflow_path), exist_ok=True)
    with open(mock_workflow_path, "w", encoding="utf-8") as f:
        f.write(mock_content)

    # Perform mock patch
    target = "- Step 2: Error-prone step to be optimized"
    replacement = "- Step 2: Optimized and verified step (New SOP)"
    
    success = apply_patch(mock_workflow_path, target, replacement)
    
    if success:
        # Verify
        with open(mock_workflow_path, "r", encoding="utf-8") as f:
            patched_content = f.read()
        
        if replacement in patched_content and target not in patched_content:
            log("Mock test PASSED successfully.")
            # Clean up
            if os.path.exists(mock_workflow_path):
                os.remove(mock_workflow_path)
            return True
        else:
            log("Mock test FAILED: replacement text not found or target still present.")
    else:
        log("Mock test FAILED: patch execution failed.")
    
    if os.path.exists(mock_workflow_path):
        os.remove(mock_workflow_path)
    return False

def main():
    if "--mock" in sys.argv:
        success = run_mock_test()
        sys.exit(0 if success else 1)

    # Default behaviour: scan current active workspaces' progress log
    # Find active workspaces from active_plans.md or scan agent/memory/workspaces/*/05_progress.md
    log("Scanning active workspaces for SOP patches...")
    
    # We will search the active workspace progress file
    active_plans_path = "agent/memory/global/active_plans.md"
    if not os.path.exists(active_plans_path):
        log(f"active_plans.md not found at {active_plans_path}")
        sys.exit(1)

    with open(active_plans_path, "r", encoding="utf-8") as f:
        plans_content = f.read()

    # Parse active workspaces from active_plans.md
    # Look for paths like agent/memory/workspaces/<workspace_name> marked as active
    active_workspaces = re.findall(r'agent/memory/workspaces/([a-zA-Z0-9_\-]+)', plans_content)
    
    if not active_workspaces:
        log("No active workspaces found in active_plans.md.")
        sys.exit(0)

    total_applied = 0
    for ws in active_workspaces:
        progress_path = f"agent/memory/workspaces/{ws}/05_progress.md"
        if os.path.exists(progress_path):
            log(f"Parsing progress file for workspace {ws}...")
            patches = parse_progress_file(progress_path)
            for workflow_file, target, replacement in patches:
                workflow_path = f"agent/workflows/{workflow_file}"
                log(f"Applying patch to {workflow_file}...")
                if apply_patch(workflow_path, target, replacement):
                    total_applied += 1

    log(f"SOP patching completed. Total patches applied: {total_applied}")

if __name__ == "__main__":
    main()
