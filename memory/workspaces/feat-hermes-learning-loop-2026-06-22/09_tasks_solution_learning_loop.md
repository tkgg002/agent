# Technical Solution: Hermes Closed Learning Loop

This document outlines the detailed implementation specifications for the three core learning loop scripts: `sop_patcher.py`, `skill_miner.py`, and `rule_promoter.py`.

---

## 1. SOP Patcher (`sop_patcher.py`)

### Goal
Parse progress files or git diffs to locate and apply workflow fixes.

### Algorithm
1. Locate the active workspace's `05_progress.md` or look at git diff.
2. Read the file line-by-line and search for the pattern:
   `\[SOP_PATCH\]\s*([a-zA-Z0-9_\-\.]+)\s*\|\s*(.*?)\s*\|\s*(.*)`
   - Group 1: Target workflow filename (e.g., `bug-handling-sop.md`).
   - Group 2: Target content to replace.
   - Group 3: Replacement content.
3. Open `agent/workflows/<filename>`.
4. Validate that the target content exists uniquely. If not unique or not found, log a warning and skip.
5. Perform string replacement (preserving indentation and newlines).
6. Write back to the target file.
7. Log the successful patch operation.

---

## 2. Skill Miner (`skill_miner.py`)

### Goal
Automatically extract new agent skills from completed workspace tasks.

### Algorithm
1. Read the active workspace's `05_progress.md` and count the completed steps. If steps < 5, skip unless forced via `--force`.
2. Extract the workspace name (e.g., `feat-decouple-handlers-db-2026-06-22`).
3. Clean the name to create a human-readable title (e.g., "Decouple Handlers DB").
4. Determine the category by checking keywords in the workspace name and logs:
   - Keywords `db`, `sql`, `postgres`, `mongo` -> `database`
   - Keywords `go`, `golang` -> `golang`
   - Keywords `react`, `typescript`, `ts`, `css`, `html` -> `web`
   - Keywords `python` -> `python`
   - Default -> `common`
5. Construct a `SKILL.md` template conforming to *agentskills.io*:
   ```markdown
   ---
   name: <Title>
   description: Automated skill extracted from workspace <Workspace Name>
   tags: [<category>, learning-loop]
   version: 1.0.0
   ---
   # <Title>
   
   ## Context & Goal
   Automatically mined from task logs.
   
   ## Key Instructions
   - Step 1: ...
   - Step 2: ...
   
   ## Pitfalls & Solutions
   - Pitfall: ...
   - Solution: ...
   ```
6. Parse logs in `05_progress.md` to populate steps.
7. Save the skill to `agent/skills/<category>/<skill-name-kebab>/SKILL.md`.

---

## 3. Rule Promoter (`rule_promoter.py`)

### Goal
Promote repeating lesson tags into hard-coded constitution rules in `GEMINI.md` and sync with `CLAUDE.md`.

### Algorithm
1. Read `agent/memory/global/lessons.md`.
2. Parse lessons. Each lesson starts with `### [YYYY-MM-DD]`.
3. Extract `Tags` or `Pattern` from lessons. Maintain a count of occurrences.
4. If any tag or pattern error occurs >= 3 times:
   - Identify the tag (e.g., `governance`).
   - Draft a rule proposition.
5. Open `agent/GEMINI.md`.
6. Parse existing numbered rules (e.g., `1. `, `2. `, ..., `20. `).
7. Append the new rule as the next number (e.g., `21. `) under a "Supplemental Rules" section or directly after the last rule.
8. Create a backup `agent/GEMINI.bak` before saving.
9. Open `agent/CLAUDE.md` and sync the new rule with a backup `agent/CLAUDE.bak`.
10. Print the promoted rules summary.
