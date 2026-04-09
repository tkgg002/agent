---
description: Chief Engineer workflow v2 — Full-loop execution với dependency check và rollback protocol
---

# Muscle Execute Workflow v2

> Quy tắc #1 + #2 + #4 (Newline Rule)
> Muscle = Chief Engineer. Code, test, deploy. Không dừng cho đến khi verify xong.

## Khi nào dùng
Trigger: `/muscle-execute`, hoặc khi nhận delegation prompt từ Brain.

## Workflow Steps

### 0. Load Model Configuration (Rule #3)

// turbo
```bash
# Ưu tiên load model rẻ cho execution và nạp Key Rotation
source agent/models.env || echo "Using system defaults"
# Kiểm tra quota trước khi chạy lệnh chính
./agent/scripts/quota_check.sh $MUSCLE_MODEL
```

### 1. Parse & Validate Delegation

- [ ] Hiểu rõ task description
- [ ] Biết Phase hiện tại (GĐ0-4)
- [ ] Biết Service Group + risk level
- [ ] Definition of Done rõ ràng
- [ ] Dependencies đã được list

> Thiếu thông tin → DỪNG, yêu cầu Brain bổ sung.

### 2. Dependency Impact Check

// turbo
```bash
# Xem dependency graph
cat agent/memory/project_context.md | grep -A 20 "Key Dependencies"

# Tìm services phụ thuộc vào service đang sửa
grep -rn "<service_name>" /Users/trainguyen/Documents/work/*/src/ --include="*.ts" -l | head -20
```

**Câu hỏi BẮT BUỘC**:
- Service nào GỌI service đang sửa?
- Service đang sửa GỌI service nào?
- Nếu thay đổi API contract → services nào bị break?

### 3. Root Cause Analysis (nếu bug fix)

Gọi `/debug-agent` nếu cần. Hoặc tự investigate:
// turbo
```bash
grep -rn "error_keyword" <service>/src/ --include="*.ts"
cat <log_file> | grep -A5 "ERROR"
```

### 4. Implement Changes

- Tuân thủ patterns trong `agent/memory/tech_stack.md`
- CQRS structure nếu GĐ3+
- KHÔNG over-engineer — fix đúng scope

### 5. Self-verify

// turbo
```bash
npm run lint
npm run build       # hoặc npx tsc --noEmit
npm test
```

> **NEWLINE RULE**: Mọi `send_command_input` PHẢI kèm `\n`.

### 6. Dependency Regression Check

// turbo
```bash
# Nếu thay đổi API contract, verify services phụ thuộc
# Chạy test ở service gọi tới service vừa sửa
cd /Users/trainguyen/Documents/work/<dependent-service>
npm test
```

Nếu break → **ROLLBACK** changes hoặc update dependent services.

### 7. Security Check

Gọi `/security-agent`:
- Input validation
- Secrets check
- SQL injection (đặc biệt với raw queries trong Go services)

### 8. QA Gate

Gọi `/qa-agent`:
- All existing tests pass
- New tests cho new code
- Integration test nếu cross-service changes

### 9. K8s Deploy Alignment (GĐ1+)

Nếu thay đổi ảnh hưởng deployment:
// turbo
```bash
# Verify K8s config match với code changes
cat /Users/trainguyen/Documents/work/workload-sre-*/deployments/<service>.yaml | grep -A5 "terminationGracePeriodSeconds\|preStop\|readinessProbe"
```

### 10. Commit & Push

// turbo
```bash
git add -A
git commit -m "<type>(<service>): mô tả ngắn gọn

- Chi tiết 1
- Chi tiết 2

Phase: GĐ<0-4>
Closes #issue"

git push origin <branch>
```

### 11. Update Memory (Rule #6)

- [ ] Update `agent/memory/active_plans.md` (đánh dấu task done)
- [ ] Update `agent/memory/tech_stack.md` (nếu pattern/lib mới)
- [ ] Ghi ADR mới nếu có quyết định quan trọng

### 12. Report cho Brain

```markdown
## Execution Report
### Task: <tên>
### Phase: GĐ<X> | Service Group: <group>
### Root Cause: <nếu bug fix>
### Changes
| File | Change |
|------|--------|
| path | mô tả |
### Dependency Impact: <None / Updated X services>
### Verification
- [x] Lint/Build: PASS
- [x] Unit Tests: PASS
- [x] Regression: PASS
- [x] Security: PASS
- [x] K8s Align: PASS
### Memory Updated: ✅
### Model Used: <model_name>
```

> [!IMPORTANT]
> **Tracking Rule**: Mọi hành động thực thi phải được ghi vào `05_progress.md` theo định dạng:
> `[Provider:Model-Name] [Time] Action description` (Ví dụ: `[Antigravity:gemini-3-flash] Brain cập nhật code`)

## Critical Rules

- ⚠️ NEWLINE RULE: `send_command_input` + `\n`
- ⚠️ Dependency check TRƯỚC implement, KHÔNG SAU
- ⚠️ Financial Core changes: PHẢI có rollback plan
- ⚠️ Update memory SAU MỖI task

## Definition of Done
- [ ] Lint/Build pass (không có lỗi mới)
- [ ] Unit tests pass
- [ ] Security scan pass (không có Critical/High)
- [ ] K8s config align với code changes
- [ ] Execution Report đã gửi cho Brain
- [ ] `05_progress.md` và `active_plans.md` đã được cập nhật
