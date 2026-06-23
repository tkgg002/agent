#!/usr/bin/env python3
import os
import sys
import re

def log(msg):
    print(f"[SKILL_MINER] {msg}")

def clean_name(name):
    # Convert feat-decouple-handlers-db-2026-06-22 -> Decouple Handlers DB
    # Remove prefix like feat-, bug-
    cleaned = re.sub(r'^(feat|bug|hotfix|chore|refactor)-', '', name, flags=re.IGNORECASE)
    # Remove suffix like -YYYY-MM-DD or -2026-06-22
    cleaned = re.sub(r'-\d{4}-\d{2}-\d{2}$', '', cleaned)
    # Replace dashes/underscores with spaces and capitalize
    words = cleaned.replace('-', ' ').replace('_', ' ').split()
    return ' '.join(word.capitalize() for word in words)

def clean_kebab(name):
    cleaned = re.sub(r'^(feat|bug|hotfix|chore|refactor)-', '', name, flags=re.IGNORECASE)
    cleaned = re.sub(r'-\d{4}-\d{2}-\d{2}$', '', cleaned)
    return cleaned.lower().replace('_', '-')

def determine_category(title, logs_text):
    text = (title + " " + logs_text).lower()
    if any(k in text for k in ['db', 'sql', 'postgres', 'mongo', 'database', 'repository']):
        return 'database'
    if any(k in text for k in ['go', 'golang']):
        return 'golang'
    if any(k in text for k in ['react', 'typescript', 'ts', 'css', 'html', 'js', 'javascript', 'frontend']):
        return 'web'
    if any(k in text for k in ['python', 'py']):
        return 'python'
    return 'common'

def generate_skill(workspace_name, progress_path):
    if not os.path.exists(progress_path):
        log(f"Error: Progress file {progress_path} does not exist.")
        return False

    with open(progress_path, "r", encoding="utf-8") as f:
        lines = f.readlines()

    logs_text = "".join(lines)
    
    # Extract steps from lines starting with - `[Timestamp] [Agent:Model]` or similar
    # or just parse action descriptions
    steps = []
    # Match: - `[2026-06-22T15:20:00+07:00] [Brain:Antigravity]` Hoàn tất Phase 1...
    step_pattern = re.compile(r'-\s*`\[[^\]]+\]\s*\[[^\]]+\]`?\s*(.*)')
    
    for line in lines:
        match = step_pattern.search(line)
        if match:
            action_desc = match.group(1).strip()
            # Avoid logging planning or mock logs if possible, but keep actionable steps
            if action_desc and not any(k in action_desc.lower() for k in ['bắt đầu', 'lập kế hoạch', 'chờ phê duyệt']):
                steps.append(action_desc)

    # Fallback to general lines if no structured logs found
    if not steps:
        for line in lines:
            if line.strip().startswith("- ") and not line.strip().startswith("- **"):
                steps.append(line.strip()[2:])

    title = clean_name(workspace_name)
    kebab_name = clean_kebab(workspace_name)
    category = determine_category(title, logs_text)

    # Format steps
    formatted_steps = ""
    for i, step in enumerate(steps[:10], 1): # Max 10 steps
        formatted_steps += f"{i}. **{step}**\n"

    if not formatted_steps:
        formatted_steps = "1. **Perform task requirements.**\n2. **Verify implementation correctness.**\n"

    # Construct skill markdown content
    skill_content = f"""---
name: {title}
description: Automated skill mined from workspace {workspace_name}
tags: [{category}, learning-loop]
version: 1.0.0
---

# {title}

## Context & Goal / Bối cảnh & Mục tiêu
This skill captures the process and steps implemented during the {title} workspace.
Kỹ năng này ghi nhận quy trình và các bước thực hiện trong workspace {title}.

## Key Instructions / Hướng dẫn chi tiết
{formatted_steps}
## Common Pitfalls & Solutions / Cạm bẫy & Giải pháp
- **Pitfall**: Incomplete testing or missing environment variable configurations.
- **Solution**: Always run local mock tests and verify build success.

- **Sai sót**: Kiểm thử thiếu sót hoặc thiếu cấu hình biến môi trường.
- **Giải pháp**: Luôn chạy thử nghiệm giả lập cục bộ và xác minh build thành công.

## Verification & QA / Xác minh & Đảm bảo chất lượng
- Verify that the generated services compile successfully.
- Check logs for any syntax or runtime errors.
"""

    dest_dir = f"agent/skills/{category}/{kebab_name}"
    os.makedirs(dest_dir, exist_ok=True)
    dest_path = f"{dest_dir}/SKILL.md"

    with open(dest_path, "w", encoding="utf-8") as f:
        f.write(skill_content)

    log(f"Successfully mined and generated skill at {dest_path}")
    return True

def run_mock_test():
    log("Running mock test for Skill Miner...")
    mock_ws = "feat-mock-test-db-2026-06-22"
    mock_progress_dir = "agent/memory/workspaces/feat-mock-test-db-2026-06-22"
    mock_progress_path = f"{mock_progress_dir}/05_progress.md"

    os.makedirs(mock_progress_dir, exist_ok=True)
    
    mock_progress_content = """# Progress: Mock Test
## Audit Trail
- `[2026-06-22T15:00:00+07:00] [Brain:Antigravity]` Bắt đầu lập kế hoạch.
- `[2026-06-22T15:10:00+07:00] [Brain:Antigravity]` Thiết lập cấu trúc cơ sở dữ liệu Postgres.
- `[2026-06-22T15:20:00+07:00] [Brain:Antigravity]` Cấu hình các API routes và handlers kết nối DB.
- `[2026-06-22T15:30:00+07:00] [Brain:Antigravity]` Viết các kịch bản kiểm thử tích hợp DB.
- `[2026-06-22T15:40:00+07:00] [Brain:Antigravity]` Chạy kiểm thử thành công và bàn giao tính năng.
"""
    with open(mock_progress_path, "w", encoding="utf-8") as f:
        f.write(mock_progress_content)

    success = generate_skill(mock_ws, mock_progress_path)

    # Clean up mock progress
    if os.path.exists(mock_progress_path):
        os.remove(mock_progress_path)
    if os.path.exists(mock_progress_dir):
        os.rmdir(mock_progress_dir)

    if success:
        generated_skill_path = "agent/skills/database/mock-test-db/SKILL.md"
        if os.path.exists(generated_skill_path):
            with open(generated_skill_path, "r", encoding="utf-8") as f:
                content = f.read()
            if "Mock Test" in content and "Postgres" in content:
                log("Mock test PASSED successfully.")
                # Clean up generated skill
                os.remove(generated_skill_path)
                os.rmdir(os.path.dirname(generated_skill_path))
                return True
            else:
                log("Mock test FAILED: Skill content verification failed.")
        else:
            log(f"Mock test FAILED: Mined skill file not found at {generated_skill_path}.")
    else:
        log("Mock test FAILED: Skill generation returned False.")

    return False

def main():
    if "--mock" in sys.argv:
        success = run_mock_test()
        sys.exit(0 if success else 1)

    # Default behaviour: mine from active workspaces
    active_plans_path = "agent/memory/global/active_plans.md"
    if not os.path.exists(active_plans_path):
        log(f"active_plans.md not found at {active_plans_path}")
        sys.exit(1)

    with open(active_plans_path, "r", encoding="utf-8") as f:
        plans_content = f.read()

    active_workspaces = re.findall(r'agent/memory/workspaces/([a-zA-Z0-9_\-]+)', plans_content)
    
    if not active_workspaces:
        log("No active workspaces found in active_plans.md.")
        sys.exit(0)

    total_mined = 0
    for ws in active_workspaces:
        progress_path = f"agent/memory/workspaces/{ws}/05_progress.md"
        if os.path.exists(progress_path):
            # Check step count in progress file
            with open(progress_path, "r", encoding="utf-8") as f:
                content = f.read()
            # Count the occurrences of log formats or lines
            log_entries = re.findall(r'-\s*`\[[^\]]+\]', content)
            
            # Allow force mining via argv
            if len(log_entries) >= 5 or "--force" in sys.argv:
                log(f"Mining workspace {ws} with {len(log_entries)} steps...")
                if generate_skill(ws, progress_path):
                    total_mined += 1
            else:
                log(f"Skipping workspace {ws} - only has {len(log_entries)} steps (requires >= 5 steps).")

    log(f"Skill mining completed. Total skills mined: {total_mined}")

if __name__ == "__main__":
    main()
