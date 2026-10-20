---
description: Bug Handling SOP - Standard operating procedure cho mọi bug fix từ intake đến close
---

# Bug Handling SOP

> Governance reference: CLAUDE.md Rule 3 (Plan & Verify), Rule 6 (Simplicity), Rule 7 (Workspace-first), Rule 8 (Security gate), Rule 13 (Lesson writing), Rule 14 (Pre-flight)

Mục đích: Thống nhất quy trình xử lý bug để không sót documentation + lesson. Mọi agent (Brain/Muscle/Sub-agent) phải tuân theo.

---

## Flow tổng quan

```
INTAKE → PLAN → EXECUTE → VERIFY → DOCUMENT → LESSON → CLOSE
  ↑         ↓         ↓         ↓          ↓         ↓
 bug      5-whys    Muscle    runtime    workspace  global
 log              code fix   evidence    doc        lessons.md
```

---

## Stage 1 — INTAKE (Brain)

Khi user báo bug HOẶC agent tự phát hiện:

1. **Quote nguyên văn error** (stack trace, log message, UI screenshot).
2. **Xác định scope**: service nào, file:line nào, tần suất, severity.
3. **Note vào workspace**:
   - Nếu bug thuộc feature đang active (có workspace) → APPEND `05_progress.md` với dòng "[TIMESTAMP] User/Agent — BUG: <quote>"
   - Nếu bug mới hoàn toàn → tạo workspace mới hoặc dùng workspace `bug-triage/` chung.
4. **KHÔNG** fix ngay. Plan trước.

---

## Stage 2 — PLAN (Brain)

**5-whys root cause analysis**:
- Why 1: Tại sao symptom xuất hiện?
- Why 2: Tại sao why 1 xảy ra?
- Why 3-5: tiếp tục tới architectural/design level root cause.

**Spec vs Impl gap check**:
- Đọc plan/spec gốc (02_plan_*.md trong workspace).
- Compare với code hiện tại.
- Gap nào cause bug?

**Cross-service pattern search**:
```bash
rg "<error pattern|function|config>" --type <lang> -l
```
- Output list callsite. Verify nếu pattern xuất hiện ở service khác (monorepo).
- Nếu có → plan fix cross-service, không 1 chỗ.

**Plan output**: propose approach A/B/C với tradeoff. Chọn approach simplest (Rule 6).

**Anti-pattern** (Rule 13 violation):
- ❌ Band-aid fix symptom khi chưa xác định root cause
- ❌ Cap log/widening threshold/disable alert (hide symptom)

---

## Stage 3 — EXECUTE (Muscle, qua Brain delegate)

Brain KHÔNG code (Rule 12). Delegate Muscle với brief rõ:
- File paths cụ thể
- Expected behavior (AC)
- Anti-cases (đừng làm gì)
- Build + test commands

Muscle tuân Rule 6: minimal-impact fix, không refactor beyond scope.

---

## Stage 4 — VERIFY (Muscle + Brain cross-check)

**Runtime verification** (Rule 3):
- Build pass (`go build` / `npm run build` / etc.)
- Lint/vet/tsc pass
- Unit test pass (nếu liên quan)
- **Startup log clean**: `grep -iE "error|fail|panic|sqlstate|warning" <log>` = 0 match
- **Functional runtime test**: reproduce bug original → confirm fixed (before/after evidence)

**Semantic validation** (lesson #2):
- Nếu fix metric/aggregation → cross-validate source-of-truth độc lập
- Nếu fix cross-store consistency → property test (equal input → equal output)
- Nếu fix scale concern → test với dataset ≥ 10× expected production

**Cross-service verify**:
- Nếu cross-service pattern → start ALL affected services → verify clean startup.

---

## Stage 5 — DOCUMENT (Brain)

Mọi bug fix PHẢI có physical file trong workspace (Rule 7, No Shadow Files):

### File: `03_implementation_<short-name>_fix.md`
Trong workspace `agent/memory/workspaces/<feature>/`:

```markdown
# <Bug Title> Fix

> Date: YYYY-MM-DD
> Trigger: <error nguyên văn + quote user>
> Status: ✅ RESOLVED | ⚠️ PARTIAL | ❌ BLOCKED

## 1. Symptom
<error + tần suất + scope>

## 2. Iteration Timeline
- HH:MM Discovery
- HH:MM Hypothesis 1 + test + result
- HH:MM Hypothesis N + test + result
- HH:MM Root cause identified (evidence)
- HH:MM Fix applied (file:line)
- HH:MM Verified (numbers)

## 3. Root Cause
<paragraph>

## 4. Fix
<files + diff summary>

## 5. Verify
- Before: <number/state>
- After: <number/state>
- Reduction: <%>
- EXPLAIN/runtime log evidence

## 6. Related lessons
<reference lessons.md entries>

## 7. Follow-ups
<non-blocking tactical items>
```

### APPEND `05_progress.md` (Rule 11 immutable):
```
| YYYY-MM-DD HH:MM | Agent | Model | **BUG/FIX title**: <1-2 sentence summary + evidence>. |
```

---

## Stage 6 — LESSON (Brain, BẮT BUỘC nếu sơ sót)

**Sơ sót thường gặp** (session history lessons reference):

| # | Sơ sót | Lesson title gợi ý |
|:--|:-------|:-------------------|
| 1 | Scale calc missing → plan fail ở production size | "Scale calculation mandatory" |
| 2 | Runtime "chạy ra số" ≠ semantic correct | "Runtime verified ≠ semantic correct" |
| 3 | Hỏi user assumption thay vì đọc workspace | "Brain archaeology workspace-first" |
| 4 | Over-engineer role (DevOps) ở local dev | "Environment-match ceremony" |
| 5 | Service listening ≠ healthy (startup log bypass) | "Service listening discipline" |
| 6 | Ghi lesson sai chỗ (auto-memory vs workspace) | "Lesson location discipline" |
| 7 | Chôn critical limitation trong doc dài | "Surface NOT_DELIVERED at top" |
| 8 | Fix 1 service, miss cross-service pattern | "Cross-service pattern search" |
| 9 | Band-aid symptom thay vì root cause | "Symptom vs cause separation" |
| 10 | Upgrade vendor version giả định "stable" | "Version regression possible" |
| 11 | Partition table thiếu parent index | "Partitioned index propagation" |
| 12 | Log-per-item scale unbounded | "Audit sampling at scale" |
| 13 | Hash algorithm khác cross-store | "Unified hash layer" |

**Format lesson** (Rule 13):
```markdown
## [YYYY-MM-DD] <Title>

- **Trigger**: <context + user quote>
- **Root Cause (meta)**: <why this class of error>
- **Global Pattern [A does B to X] → Result Y**: <abstract, applicable 3+ projects>
- **Correct Pattern**:
  1. ...
  2. ...
- **Anti-pattern**: ...
- **Tags**: #category #keywords
```

APPEND vào `/Users/trainguyen/Documents/work/agent/memory/global/lessons.md` — KHÔNG auto-memory riêng. KHÔNG overwrite.

---

## Stage 7 — CLOSE (Brain)

**Pre-flight checklist** (Rule 14):
- [ ] Root cause xác định (không guess)
- [ ] Fix applied + build pass
- [ ] Runtime verified với số trước/sau
- [ ] Startup log clean
- [ ] `03_implementation_*_fix.md` tạo trong workspace
- [ ] `05_progress.md` append (Rule 11)
- [ ] Lesson append nếu có sơ sót
- [ ] Cross-service scope verified
- [ ] Security gate (Rule 8): không leak secret, không bypass auth, không SQL injection
- [ ] Report user: quote error → fix → evidence → skills used

**Response user format**:
```
## <Bug title> — ✅ RESOLVED

### Root cause
<1-2 sentence>

### Fix
<file + diff summary>

### Evidence
Before: <X>
After: <Y>
Reduction: <Z>%

### Files
- <impl doc>
- <migration if any>

### Skills/tools
- <list>
```

---

## Lesson pointer — khi nào user sẽ nhắc

User thường nhắc khi phát hiện:
- Brain "im re" → Stage 2 lấy quá lâu hoặc skip → không response updates
- Fix không kỹ → Stage 4 verify yếu
- Lặp cùng loại lỗi → Stage 6 lesson không được áp dụng
- Doc sai chỗ → Stage 5/6 violation Rule 7

→ Đọc `lessons.md` đầu mỗi session. Áp dụng pattern.

---

## Quick reference card

```
BUG received
  ↓
INTAKE — quote + scope + progress log
  ↓
PLAN — 5-whys + spec-impl gap + cross-service grep
  ↓
EXECUTE — delegate Muscle, minimal-impact
  ↓
VERIFY — build + startup log clean + semantic validation + numerical before/after
  ↓
DOCUMENT — 03_implementation_*_fix.md + progress append
  ↓
LESSON — if sơ sót → global lessons.md A/B/X/Y pattern
  ↓
CLOSE — pre-flight Rule 14 + user response
```

---

## Related workflows
- `/debug-agent` — detailed root cause analysis sub-workflow
- `/muscle-execute` — Muscle implementation workflow
- `/security-agent` — Rule 8 security gate
- `/checkpoint` — periodic state snapshot
