---
description: Quản lý bộ nhớ dự án v2.1 (Role-aware + Feature Context)
---

# Context Manager Workflow v2.1

> Quy tắc #6 (Knowledge Retention) + Quy tắc #1 (Separation of Concerns)
> **Brain** (Chairman) chịu trách nhiệm *duy trì* context.
> **Muscle** (Chief Engineer) chịu trách nhiệm *cập nhật* context thực tế.

## Khi nào dùng (Triggers)

- **READ**:
    - Đầu phiên làm việc (`/restore`).
    - Ngay khi **Brain** bắt đầu workflow `/brain-delegate`.
- **WRITE**:
    - Sau khi **Brain** chốt Plan.
    - Ngay khi **Muscle** hoàn thành workflow `/muscle-execute` (exit code 0).


## Ai làm gì?

| Role | Action | Khi nào? | Mục đích |
|------|--------|----------|----------|
| **Brain** (Chairman) | **READ** | Bắt đầu session, trước `/brain-delegate` | Lấy bối cảnh, luật chơi, tech stack để ra quyết định đúng. |
| **Brain** (Chairman) | **WRITE** | Sau khi Plan, chốt Solution | Ghi lại active plan, ADR mới, update Global Context. |
| **Muscle** (Builder) | **READ** | Khi nhận lệnh từ Brain | Hiểu rõ task, coding convention, DB schema. |
| **Muscle** (Builder) | **WRITE** | Sau khi code/test (`/muscle-execute`) | Ghi log vào `progress.md`, update task status. |

## Workflow Steps

### 1. Context Restore (Read Mode)

User/Brain chạy lệnh này để load context vào bộ nhớ của session.

// turbo
```bash
# 1. Global Context (Luật chơi & Quyết định cốt lõi)
cat agent/memory/global/project_context.md
cat agent/memory/global/tech_stack.md
cat agent/memory/global/architectural_decisions.md

# 2. Feature Context (Cụ thể theo task)
# (Thay 'feature-goopay-backend' bằng feature folder tương ứng)
TARGET_FEATURE="feature-goopay-backend" 
cat agent/memory/workspaces/$TARGET_FEATURE/active_plan.md
cat agent/memory/workspaces/$TARGET_FEATURE/progress.md
# Nếu cần DB/Docs specific:
# cat agent/memory/workspaces/$TARGET_FEATURE/db_context.md
```

### 2. Context Save (Write Mode)

#### A. Brain Update (Planning & Architecture)
- **`active_plan.md`**: Brain cập nhật danh sách task, đánh dấu completed `[x]`, thêm next steps.
- **`architectural_decisions.md`**: Brain ghi lại các quyết định thay đổi hệ thống (nếu có).

```bash
# Template update plan
cat > agent/memory/workspaces/$TARGET_FEATURE/active_plan.md <<EOF
... nội dung latest plan ...
EOF
```

#### B. Muscle Update (Execution Log)
- **`progress.md`**: Muscle ghi lại work log ngắn gọn sau khi execute thành công.
- **`db_context.md`**: Muscle update nếu có migration mới chạy.

```bash
# Template log progress
cat >> agent/memory/workspaces/$TARGET_FEATURE/progress.md <<EOF
- $(date +%Y-%m-%d): [Completed] Refactor module X. Verified on Staging.
EOF
```

## Workspaces Structure
- `global/`: Quy tắc chung (Brain quản lý chính) — `lessons.md`, `active_plans.md`, `conventions.md`
- `workspaces/[name]/`: Mỗi workspace là 1 project độc lập — V3 Standard (00→06)

## Definition of Done
- [ ] Đầu phiên: `active_plans.md` đã được đọc, workspace active đã được xác định
- [ ] Cuối phiên: `05_progress.md` đã được cập nhật với hành động trong phiên
- [ ] Mọi thay đổi kiến trúc đã được ghi vào `04_decisions.md`
- [ ] `active_plans.md` phản ánh đúng status hiện tại của các workspaces
