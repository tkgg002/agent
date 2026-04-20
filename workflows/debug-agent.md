---
description: Debugger sub-agent - Tìm root cause của bug bằng phương pháp hệ thống
---

# Debug Agent Workflow

> Quy tắc #3 (Deep Execution - Agent-within-Agent)
> Sub-agent chuyên trách: Root Cause Analysis. Output = evidence-based root cause.

## Khi nào dùng

Dùng khi Muscle cần tìm nguyên nhân gốc rễ của bug.
Trigger: `/debug-agent`, hoặc được gọi bởi `/muscle-execute` step 2.

## Input Requirements

Debugger Agent CẦN nhận:
- **Error message / Stack trace**: Copy/paste nguyên văn
- **Service name**: Service nào bị lỗi
- **Reproduction steps**: Cách tái tạo lỗi (nếu biết)
- **Environment**: dev / staging / production

## Workflow Steps

### 1. Reproduce (Tái tạo)

// turbo
```bash
# Đọc logs gần nhất
tail -100 <log_file>

# Hoặc search error pattern
grep -rn "<error_keyword>" <service>/src/ --include="*.ts"
```

Mục tiêu: Xác nhận lỗi tồn tại và có thể tái tạo.

### 2. Isolate (Thu hẹp phạm vi)

// turbo
```bash
# Tìm file chứa logic liên quan
grep -rn "<function_name>" <service>/src/ --include="*.ts"

# Xem git blame để tìm commit gây lỗi
git log --oneline -20 -- <suspected_file>

# Dùng git bisect nếu cần
git bisect start
git bisect bad HEAD
git bisect good <known_good_commit>
```

### 3. Trace (Theo dõi data flow)

- Trace từ entry point (API route / event handler)
- Theo dõi data transformation qua từng layer
- Đánh dấu nơi data bị sai

```
Entry → Controller → Service → Repository → Database
                          ↑
                     BUG Ở ĐÂY
```

### 4. Identify (Xác định root cause)

Phân loại root cause:
| Category | Ví dụ |
|----------|-------|
| Logic Error | Điều kiện if/else sai |
| Data Error | Input không validate |
| Race Condition | Async timing issue |
| Config Error | Env variable sai |
| Dependency | Thư viện bị bug |
| Integration | API contract thay đổi |

### 5. Report

Output format (BẮT BUỘC):
```markdown
## Debug Report

### Root Cause
<mô tả ngắn gọn, 1-2 câu>

### Evidence
- File: `<path/to/file>` line <N>
- Error: `<exact error message>`
- Logs: 
```
<relevant log lines>
```

### Category
<Logic Error / Data Error / Race Condition / Config Error / Dependency / Integration>

### Suggested Fix
<mô tả cách fix, không cần code cụ thể>

### Impact Assessment
- Scope: <file/service/system>
- Severity: <Critical/High/Medium/Low>
- Regression risk: <High/Medium/Low>
```

### 6. Document (BẮT BUỘC sau khi fix apply)

Tạo file `03_implementation_<short-name>_fix.md` trong workspace feature liên quan. Format:
```markdown
# <Bug short title> Fix

> Date: YYYY-MM-DD
> Trigger: <error user báo, quote nguyên văn>
> Status: ✅ RESOLVED | ⚠️ PARTIAL | ❌ BLOCKED

## 1. Symptom
<error message + nơi xuất hiện + tần suất>

## 2. Timeline / Iteration
- HH:MM — Phát hiện: <cách discovery>
- HH:MM — Hypothesis 1: <giả thuyết>. Test: <cách>. Result: confirm/reject.
- HH:MM — Hypothesis N: ...
- HH:MM — Root cause identified: <evidence>
- HH:MM — Fix applied: <file:line diff>
- HH:MM — Verified: <numbers/EXPLAIN/runtime log>

## 3. Root Cause
<1 đoạn văn explain >

## 4. Fix
<files changed + diff summary>

## 5. Verify
- Command: `<exact command>`
- Before: <number/state>
- After: <number/state>
- Reduction: <%>

## 6. Related lessons
- Reference `agent/memory/global/lessons.md` entry: <date + title>
```

APPEND `05_progress.md` với 1 dòng summary timestamp + agent + short description.

### 7. Lesson Capture (BẮT BUỘC nếu có sơ sót)

Nếu trong quá trình debug có 1 trong các sơ sót sau → **MUST** append `agent/memory/global/lessons.md`:

| Sơ sót | Phải ghi lesson |
|:-------|:----------------|
| Band-aid fix symptom trước khi hiểu root cause | ✅ |
| Miss cross-service pattern (fix 1 chỗ, bỏ sót chỗ khác) | ✅ |
| Báo done mà chưa verify runtime end-to-end | ✅ |
| Sai chỗ lưu lesson/doc (auto-memory vs workspace) | ✅ |
| Chôn critical limitation trong doc dài, user miss | ✅ |
| Upgrade version giả định "mới = stable" → regression | ✅ |
| Skip property test → semantic bug escape | ✅ |
| Log spam per-item khi N unbounded | ✅ |
| Hash function khác cross-store (app vs DB) | ✅ |
| Service "listening" báo done mà startup log có error | ✅ |
| Không đọc workspace đủ → hỏi user redundant | ✅ |

Format lesson (Rule 13 - Global Pattern A/B/X/Y):
```markdown
## [YYYY-MM-DD] <One-line title>
- **Trigger**: <context + user quote nếu có>
- **Root Cause**: <meta-level, why this class of error>
- **Global Pattern [A does B to X] → Result Y**: <abstract form applicable 3+ projects>
- **Correct Pattern**: <bullet list 3-5 items>
- **Anti-pattern**: <what to avoid>
- **Tags**: #category #keywords
```

## Anti-patterns

- ❌ Đoán root cause không có evidence
- ❌ Fix ngay mà chưa hiểu rõ nguyên nhân
- ❌ Bỏ qua bước Reproduce
- ❌ Report không có logs/evidence
- ❌ Fix xong không ghi documentation trong workspace (Rule 7 violation)
- ❌ Sơ sót mà không ghi lesson (Rule 13 violation)
- ❌ Band-aid symptom khi chưa xác định root cause
- ❌ Ghi lesson vào auto-memory riêng thay vì `agent/memory/global/lessons.md`
- ❌ Symptom-first fix policy không explicit "band-aid tạm thời, root cause X cần fix sau"

## Definition of Done
- [ ] Root cause được xác định với evidence cụ thể (file + line)
- [ ] Bug được reproduce thành công
- [ ] Debug Report đã output đầy đủ (Root Cause / Evidence / Category / Fix / Impact)
- [ ] Severity được đánh giá — nếu Critical → escalate lên Brain ngay
- [ ] **Fix applied + runtime verified với numerical evidence (before/after)**
- [ ] **Document file tạo trong workspace (`03_implementation_*_fix.md`)**
- [ ] **Progress log APPEND (`05_progress.md`)**
- [ ] **Nếu có sơ sót → lesson APPEND `agent/memory/global/lessons.md`**
- [ ] **Cross-service pattern search: `rg "<pattern>" --type <lang> -l` → verify fix hết scope**
- [ ] **Startup log clean post-fix (grep `error|fail|panic|sqlstate|warn` = 0)**
