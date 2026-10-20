---
description: Kết thúc phiên làm việc — Brain cập nhật memory và ghi lại những gì đã xảy ra
---

## Trigger Conditions
- Cuối mỗi phiên làm việc (trước khi conversation kết thúc)
- Sau khi hoàn thành 1 milestone quan trọng trong workspace

## Inputs Required
- Danh sách hành động đã làm trong phiên (Brain tự tổng hợp từ conversation)

## Steps

### Bước 1: Cập nhật Progress Log
- Mở `agent/memory/workspaces/[active]/05_progress.md`
- Append entry mới với format:
```markdown
## [YYYY-MM-DD HH:MM]
- [Hành động 1]
- [Hành động 2]
- Trạng thái: [Phase X.Y — % hoàn thành]
```

### Bước 2: Kiểm tra Architectural Decision
- Trong phiên này có quyết định quan trọng nào không? (công nghệ, pattern, structure...)
- Nếu có → ghi vào `agent/memory/workspaces/[active]/04_decisions.md`

### Bước 3: Kiểm tra Lesson
- User có phải sửa lưng Brain trong phiên này không?
- Nếu có → ghi lesson vào `agent/memory/global/lessons.md` với format chuẩn (Trigger / Root Cause / Correct Pattern / Tags)

### Bước 4: Cập nhật Active Plans Registry
- Mở `agent/memory/global/active_plans.md`
- Cập nhật Status và Last Active của workspace hiện tại
- Nếu workspace Done → đổi status thành ✅ Done

### Bước 5: Update Checklist
- Mở `agent/memory/workspaces/[active]/02_plan.md`
- Mark các task đã hoàn thành `[x]`

## Outputs / Artifacts
- `05_progress.md` — updated
- `04_decisions.md` — updated (nếu có decision mới)
- `global/lessons.md` — updated (nếu có lesson mới)
- `global/active_plans.md` — updated status
- `02_plan.md` — checklist updated

## Definition of Done
- [ ] `05_progress.md` có entry mới cho phiên này
- [ ] Mọi architectural decision đã được capture
- [ ] Mọi lesson từ sai lầm trong phiên đã được ghi
- [ ] `active_plans.md` phản ánh đúng status hiện tại
