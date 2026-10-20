---
description: QA sub-agent - Kiểm thử tự động bao gồm unit tests, integration tests, và E2E (Playwright)
---

# QA Agent Workflow

> Quy tắc #3 (Deep Execution - Agent-within-Agent)
> Sub-agent chuyên trách: Quality Assurance. Output = test results + coverage report.

## Khi nào dùng

Dùng sau khi code changes được implement.
Trigger: `/qa-agent`, hoặc được gọi bởi `/muscle-execute` step 6.

## Input Requirements

QA Agent CẦN nhận:
- **Changed files**: Danh sách files đã thay đổi
- **Change type**: Bug fix / Feature / Refactor
- **Test scope**: Unit / Integration / E2E / All

## Workflow Steps

### 1. Identify Test Scope

// turbo
```bash
# Liệt kê files đã thay đổi
git diff --name-only HEAD~1

# Tìm test files tương ứng
find <service>/src -name "*.test.ts" -o -name "*.spec.ts"
```

Map changed files → test files:
- `src/services/payment.ts` → `src/services/__tests__/payment.test.ts`
- `src/controllers/order.ts` → `src/controllers/__tests__/order.test.ts`

### 2. Run Existing Tests

// turbo
```bash
# Chạy toàn bộ tests
npm test

# Hoặc chạy tests liên quan
npx jest --testPathPattern="<pattern>" --verbose

# Với coverage
npx jest --coverage --testPathPattern="<pattern>"
```

### 3. Evaluate Coverage

Kiểm tra:
- [ ] Line coverage ≥ 80%
- [ ] Branch coverage ≥ 70%
- [ ] Uncovered lines có nằm trong changed files không?
- [ ] Edge cases được cover chưa?

### 4. Write New Tests (nếu cần)

Viết tests cho:
- Happy path chưa được cover
- Edge cases: null, undefined, empty, boundary values
- Error handling paths
- Async/timeout scenarios

### 5. Run E2E Tests (nếu có UI changes)

// turbo
```bash
# Playwright
npx playwright test --project=chromium

# Hoặc specific test file
npx playwright test <test-file> --headed
```

### 6. Report

Output format (BẮT BUỘC):
```markdown
## QA Report

### Test Results
| Suite | Total | Pass | Fail | Skip |
|-------|-------|------|------|------|
| Unit | XX | XX | XX | XX |
| Integration | XX | XX | XX | XX |
| E2E | XX | XX | XX | XX |

### Coverage
| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Lines | XX% | XX% | +X% |
| Branches | XX% | XX% | +X% |

### Failing Tests (nếu có)
1. `test-name`: <error message>
2. `test-name`: <error message>

### New Tests Added
- `test-file.test.ts`: <mô tả>

### Verdict
✅ PASS / ❌ FAIL
```

## Quality Gates

- ❌ FAIL nếu: Bất kỳ existing test nào fail
- ❌ FAIL nếu: Coverage giảm so với trước
- ⚠️ WARNING nếu: Không có test cho new code
- ✅ PASS nếu: Tất cả tests pass + coverage không giảm

## Definition of Done
- [ ] Tất cả existing tests pass
- [ ] Coverage không giảm so với baseline
- [ ] QA Report đã output đầy đủ với Verdict
- [ ] Nếu FAIL → Muscle KHÔNG được push, phải fix trước
