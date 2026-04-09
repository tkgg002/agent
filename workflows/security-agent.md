---
description: Security sub-agent - Soát xét lỗ hổng bảo mật trước khi Push code
---

# Security Agent Workflow

> Quy tắc #3 (Deep Execution - Agent-within-Agent)
> Sub-agent chuyên trách: Security Review. Output = vulnerability report + remediation.

## Khi nào dùng

Dùng trước khi commit/push code changes.
Trigger: `/security-agent`, hoặc được gọi bởi `/muscle-execute` step 5.

## Input Requirements

Security Agent CẦN nhận:
- **Changed files**: Danh sách files đã thay đổi
- **Change type**: API endpoint / Database query / Auth logic / Config / Dependencies
- **Service name**: Service nào bị ảnh hưởng

## Workflow Steps

### 1. Code Review - Input Validation

// turbo
```bash
# Tìm user input không validate
grep -rn "req\.body\|req\.query\|req\.params" <service>/src/ --include="*.ts"

# Tìm raw SQL queries (SQL injection risk)
grep -rn "raw\|query(" <service>/src/ --include="*.ts"
```

Checklist:
- [ ] Tất cả user input được validate (Joi/class-validator/Zod)
- [ ] Không dùng raw SQL với user input
- [ ] Query parameters được sanitize
- [ ] File uploads được giới hạn size & type

### 2. Secrets Check

// turbo
```bash
# Tìm hardcoded credentials
grep -rn "password\|secret\|apiKey\|token\|api_key" <service>/src/ --include="*.ts" | grep -v "node_modules" | grep -v ".test."

# Kiểm tra .env không bị commit
cat .gitignore | grep ".env"

# Kiểm tra git history cho leaked secrets
git log --diff-filter=A --summary | grep ".env"
```

Checklist:
- [ ] Không hardcode passwords/secrets/API keys
- [ ] Secrets đều từ env variables
- [ ] `.env` files trong `.gitignore`
- [ ] Không có secrets trong git history

### 3. Dependency Audit

// turbo
```bash
# npm audit
npm audit --production

# Kiểm tra outdated packages
npm outdated
```

Checklist:
- [ ] Không có critical vulnerabilities
- [ ] High vulnerabilities có plan remediation
- [ ] Dependencies up-to-date

### 4. API Security

Checklist:
- [ ] Endpoints có authentication middleware
- [ ] Authorization checks đúng role/permission
- [ ] Rate limiting được implement
- [ ] CORS configured properly
- [ ] Response không leak sensitive data (passwords, tokens)
- [ ] Error messages không expose internal details

### 5. Report

Output format (BẮT BUỘC):
```markdown
## Security Report

### Scan Summary
| Category | Issues Found | Severity |
|----------|-------------|----------|
| Input Validation | X | Critical/High/Medium/Low |
| Secrets | X | Critical/High/Medium/Low |
| Dependencies | X | Critical/High/Medium/Low |
| API Security | X | Critical/High/Medium/Low |

### Vulnerabilities Found
1. **[CRITICAL]** <mô tả>
   - File: `path/to/file` line X
   - Remediation: <cách fix>

2. **[HIGH]** <mô tả>
   - File: `path/to/file` line X
   - Remediation: <cách fix>

### Verdict
✅ PASS / ❌ FAIL (có Critical/High issues)
⚠️ PASS WITH WARNINGS (chỉ Medium/Low issues)
```

## Blocking Rules

- ❌ **BLOCK PUSH** nếu: Critical vulnerability found
- ❌ **BLOCK PUSH** nếu: Hardcoded secrets detected
- ⚠️ **WARNING** nếu: High vulnerability nhưng có plan fix
- ✅ **ALLOW PUSH** nếu: Chỉ Medium/Low issues

## Definition of Done
- [ ] Tất cả 4 categories đã được scan (Input / Secrets / Deps / API)
- [ ] Security Report đã được output với Verdict rõ ràng
- [ ] Không có Critical/High issues (hoặc đã có plan remediation)
- [ ] Verdict đã báo cáo về Brain trước khi Muscle push
