#!/usr/bin/env python3
import os
import sys
import re
import shutil

def log(msg):
    print(f"[RULE_PROMOTER] {msg}")

def extract_tags_from_lessons(lessons_path):
    if not os.path.exists(lessons_path):
        log(f"Lessons file not found at {lessons_path}")
        return {}

    tag_counts = {}
    # Match tags line: - **Tags**: ... or - Tags: ...
    tag_pattern = re.compile(r'-\s*\*\*Tags\*\*:\s*(.*)', re.IGNORECASE)
    
    with open(lessons_path, "r", encoding="utf-8") as f:
        for line in f:
            match = tag_pattern.search(line)
            if match:
                tags_str = match.group(1).strip()
                # Split by commas or spaces, remove #, strip whitespace
                # e.g., "#process-governance #workspace" or "recidivism, await-approval"
                cleaned_tags = []
                if ',' in tags_str:
                    cleaned_tags = [t.strip().replace('#', '').lower() for t in tags_str.split(',')]
                else:
                    cleaned_tags = [t.strip().replace('#', '').lower() for t in tags_str.split()]
                
                for tag in cleaned_tags:
                    if tag:
                        tag_counts[tag] = tag_counts.get(tag, 0) + 1
                        
    return tag_counts

def get_next_rule_number(gemini_content):
    # Find all numbered rules in GEMINI.md like "20. Quy tắc..."
    # Match lines starting with numbers followed by dot and space
    rule_numbers = re.findall(r'^(\d+)\.\s+', gemini_content, re.MULTILINE)
    if rule_numbers:
        return max(int(num) for num in rule_numbers) + 1
    return 21 # Default fallback if not found

def promote_tag_to_rules(tag, count, gemini_path, claude_path):
    log(f"Promoting tag '{tag}' (violating {count} times) to rules...")
    
    if not os.path.exists(gemini_path):
        log(f"Error: GEMINI.md not found at {gemini_path}")
        return False
        
    with open(gemini_path, "r", encoding="utf-8") as f:
        gemini_content = f.read()

    # Avoid duplicate rules for the same tag
    rule_check_str = f"Quy tắc về {tag.upper()}"
    if rule_check_str.lower() in gemini_content.lower() or f"#{tag}" in gemini_content.lower():
        log(f"Rule for tag '{tag}' already exists or is referenced in GEMINI.md. Skipping.")
        return False

    next_num = get_next_rule_number(gemini_content)
    
    # Draft new rule
    new_rule_gemini = f"""
{next_num}. Quy tắc về {tag.capitalize()} (Promoted from Lessons)
- **Bối cảnh**: Tag '{tag}' vi phạm lặp lại {count} lần trong lessons.md.
- **Quy định**: Phải nghiêm túc tuân thủ quy trình liên quan đến {tag}, tránh tái phạm các lỗi quy trình đã được cảnh báo trong lessons.md.
"""
    
    new_rule_claude = f"""
## {next_num}. Quy tắc về {tag.capitalize()}
- Phải tuân thủ quy trình liên quan đến {tag}. Tránh lỗi tái phạm (recidivism) đã có trong lessons.md.
"""

    # Backup GEMINI.md
    gemini_bak = f"{gemini_path}.bak"
    shutil.copyfile(gemini_path, gemini_bak)
    log(f"Created backup of GEMINI.md at {gemini_bak}")

    # Insert into GEMINI.md before "## Workflows Reference"
    workflow_ref_pattern = r'(## Workflows Reference)'
    if re.search(workflow_ref_pattern, gemini_content):
        new_gemini_content = re.sub(workflow_ref_pattern, new_rule_gemini + "\n\\1", gemini_content)
    else:
        # Fallback to append at the end
        new_gemini_content = gemini_content + "\n" + new_rule_gemini

    with open(gemini_path, "w", encoding="utf-8") as f:
        f.write(new_gemini_content)
    log(f"Updated GEMINI.md with new rule {next_num}.")

    # Backup and Update CLAUDE.md
    if os.path.exists(claude_path):
        claude_bak = f"{claude_path}.bak"
        shutil.copyfile(claude_path, claude_bak)
        log(f"Created backup of CLAUDE.md at {claude_bak}")
        
        with open(claude_path, "r", encoding="utf-8") as f:
            claude_content = f.read()

        if re.search(workflow_ref_pattern, claude_content):
            new_claude_content = re.sub(workflow_ref_pattern, new_rule_claude + "\n\\1", claude_content)
        else:
            new_claude_content = claude_content + "\n" + new_rule_claude

        with open(claude_path, "w", encoding="utf-8") as f:
            f.write(new_claude_content)
        log(f"Updated CLAUDE.md and synchronized rule {next_num}.")
    else:
        log(f"CLAUDE.md not found at {claude_path}. Sync skipped.")

    return True

def run_mock_test():
    log("Running mock test for Rule Promoter...")
    mock_lessons_path = "agent/memory/global/lessons_mock.md.tmp"
    mock_gemini_path = "agent/GEMINI_mock.md.tmp"
    mock_claude_path = "CLAUDE_mock.md.tmp"

    # 1. Create mock lessons with tag repeating 3 times
    mock_lessons_content = """# Mock Lessons
### [2026-06-22] Lesson 1
- **Pattern**: A does B
- **Tags**: #mock-recidivism-tag #other-tag
- **Nguồn**: source

### [2026-06-21] Lesson 2
- **Pattern**: C does D
- **Tags**: mock-recidivism-tag, governance
- **Nguồn**: source

### [2026-06-20] Lesson 3
- **Pattern**: E does F
- **Tags**: mock-recidivism-tag
- **Nguồn**: source
"""
    os.makedirs(os.path.dirname(mock_lessons_path), exist_ok=True)
    with open(mock_lessons_path, "w", encoding="utf-8") as f:
        f.write(mock_lessons_content)

    # 2. Create mock GEMINI.md
    mock_gemini_content = """# Mock GEMINI Constitution
20. Quy tắc Vệ sinh Context
- Chi tiết quy tắc 20.

## Workflows Reference
- Workflow detail
"""
    with open(mock_gemini_path, "w", encoding="utf-8") as f:
        f.write(mock_gemini_content)

    # 3. Create mock CLAUDE.md
    mock_claude_content = """# Mock CLAUDE Constitution
## 20. Quy tắc Vệ sinh Context
- Chi tiết 20.

## Workflows Reference
- Workflow detail
"""
    with open(mock_claude_path, "w", encoding="utf-8") as f:
        f.write(mock_claude_content)

    # Run promoter on mock files
    tag_counts = extract_tags_from_lessons(mock_lessons_path)
    success = False
    
    for tag, count in tag_counts.items():
        if count >= 3 and tag == "mock-recidivism-tag":
            success = promote_tag_to_rules(tag, count, mock_gemini_path, mock_claude_path)

    # Verify updates
    if success:
        with open(mock_gemini_path, "r", encoding="utf-8") as f:
            gemini_new = f.read()
        with open(mock_claude_path, "r", encoding="utf-8") as f:
            claude_new = f.read()

        if "21. Quy tắc về Mock-recidivism-tag" in gemini_new and "21. Quy tắc về Mock-recidivism-tag" in claude_new:
            log("Mock test PASSED successfully.")
            # Clean up all mock files and backups
            for path in [mock_lessons_path, mock_gemini_path, mock_claude_path, 
                         mock_gemini_path + ".bak", mock_claude_path + ".bak"]:
                if os.path.exists(path):
                    os.remove(path)
            return True
        else:
            log("Mock test FAILED: Rule was not correctly inserted or synced.")
    else:
        log("Mock test FAILED: Tag promotion returned False or was not executed.")

    # Clean up mock files in case of failure
    for path in [mock_lessons_path, mock_gemini_path, mock_claude_path]:
        if os.path.exists(path):
            os.remove(path)
            
    return False

def main():
    if "--mock" in sys.argv:
        success = run_mock_test()
        sys.exit(0 if success else 1)

    # Default behaviour: check actual lessons.md
    lessons_path = "agent/memory/global/lessons.md"
    gemini_path = "agent/GEMINI.md"
    claude_path = "CLAUDE.md"

    log("Scanning lessons.md for repeating tags...")
    tag_counts = extract_tags_from_lessons(lessons_path)
    
    promoted_any = False
    for tag, count in tag_counts.items():
        # Exclude very generic tags that are already covered or too general
        if tag in ["governance", "process-governance", "recidivism", "workspace"]:
            continue
            
        if count >= 3:
            log(f"Tag '{tag}' found with count {count}.")
            if promote_tag_to_rules(tag, count, gemini_path, claude_path):
                promoted_any = True

    if not promoted_any:
        log("No new rules need promotion (no tag met threshold >= 3 or rules already exist).")

if __name__ == "__main__":
    main()
