---
description: Khởi động phiên làm việc mới — Brain load context và summarize current state trước khi nhận lệnh
---

## Governance Requirements
1. **Restate Requirements** - Clarify what needs to be built
2. **Identify Risks** - Surface potential issues and blockers
3. **Governance Check (Rule 7)** - Ensure Project Brain documents (04, 05, lessons) are maintained
4. **Create Step Plan** - Break down implementation into phases
5. **Wait for Confirmation** - MUST receive user approval before proceeding

## Trigger Conditions
- Bắt đầu bất kỳ phiên làm việc mới nào (conversation mới)
- Brain chưa biết current state của dự án

## Inputs Required
- Không cần input từ User — Brain tự chạy

## Steps

### Bước 1: Load Lessons
- Đọc `agent/memory/global/lessons.md`
- Filter các lessons có tag liên quan đến domain sắp làm (ví dụ: nếu sắp làm database → filter `#database`)

### Bước 1.5: Governance Audit (Rule 7)
- Thực hiện workflow `/governance-audit` để rà soát việc tuân thủ quy trình quản trị.
- Nếu phát hiện vi phạm (ví dụ: thiếu progress log phiên trước), PHẢI thực hiện RCA ngay lập tức.

### Bước 2: Xác định Active Workspace
- Đọc `agent/memory/global/active_plans.md`
- Tìm workspace có Status = 🟡 Active

### Bước 3: Load Workspace Context
Đọc theo thứ tự (workspace đang active):
1. `agent/memory/workspaces/[active]/00_context.md` — scope
2. `agent/memory/workspaces/[active]/02_plan.md` — kế hoạch
3. `agent/memory/workspaces/[active]/05_progress.md` — tiến độ gần nhất

### Bước 4 (nếu dự án là GooPay): Load Project-specific Memory
- `agent/memory/global-goopay/project_context.md`
- `agent/memory/global-goopay/tech_stack.md`

### Bước 5: Summarize Current State
Brain output 3-5 dòng tóm tắt:
```
Current State (YYYY-MM-DD):
- Workspace: [tên workspace]
- Phase: [đang ở giai đoạn nào]
- Last Action: [hành động cuối cùng]
- Next Step: [bước tiếp theo cần làm]
- Blockers: [nếu có]
```

## Outputs / Artifacts
- Không tạo file mới — chỉ đọc và summarize
- Brain nắm đủ context để nhận lệnh tiếp theo

## Definition of Done
- [ ] Đã đọc lessons.md và không bỏ sót lesson liên quan
- [ ] Đã xác định đúng workspace đang active
- [ ] Đã output "Current State" summary trước khi nhận lệnh mới
- [ ] Nếu không có workspace active → hỏi User cần làm gì
