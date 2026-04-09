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

## Anti-patterns

- ❌ Đoán root cause không có evidence
- ❌ Fix ngay mà chưa hiểu rõ nguyên nhân
- ❌ Bỏ qua bước Reproduce
- ❌ Report không có logs/evidence

## Definition of Done
- [ ] Root cause được xác định với evidence cụ thể (file + line)
- [ ] Bug được reproduce thành công
- [ ] Debug Report đã output đầy đủ (Root Cause / Evidence / Category / Fix / Impact)
- [ ] Severity được đánh giá — nếu Critical → escalate lên Brain ngay
