---
description: Chairman workflow v2 — Phase-aware delegation với service group routing cho GooPay Refactor
---

# Brain Delegate Workflow v2

> Quy tắc #1 (Separation of Concerns) + #2 (Full-Stack Prompting) + #6 (Knowledge Retention)
> Brain = Chairman. KHÔNG nhúng tay vào code. Chỉ delegate với full context.

## Khi nào dùng
Trigger: `/brain-delegate`, hoặc khi bắt đầu bất kỳ task nào.

## Workflow Steps

### 0. Load Context (Rule #6)

// turbo
```bash
# Load Global Context
cat agent/memory/global/project_context.md
cat agent/memory/global/tech_stack.md
cat agent/memory/global/architectural_decisions.md

# Load Feature Context (Default: feature-goopay-backend)
cat agent/memory/workspaces/feature-goopay-backend/active_plan.md

# Load AI Models Config (Rule #3)
cat agent/models.env || echo "Using default models"
```

### 0. Load Model Configuration (Rule #3)

// turbo
```bash
# Nạp danh sách keys và cấu hình tracking
source agent/models.env
echo "Current Brain Model: $BRAIN_MODEL (Provider: $BRAIN_PROVIDER)"
```

### 1. Phase Check — Đang ở giai đoạn nào?

| Phase | Focus | Sub-agents ưu tiên |
|-------|-------|---------------------|
| **GĐ0** Rà soát | DB audit, config, shared lib | `/debug-agent` → `/infra-validator` |
| **GĐ1** Hạ tầng | Graceful shutdown, K8s | `/infra-validator` → `/qa-agent` |
| **GĐ2** Safety net | Retry, sweeper, admin tools | `/debug-agent` → `/qa-agent` → `/security-agent` |
| **GĐ3** Architecture | NATS JetStream, Saga, CQRS | `/service-migration` → `/qa-agent` → `/security-agent` |
| **GĐ4** Observability | Tracing, logging, metrics | `/infra-validator` → `/qa-agent` |

### 2. Service Group Router

Xác định task thuộc nhóm service nào → đánh giá rủi ro:

| Group | Risk Level | Blast Radius | Special Rules |
|-------|-----------|-------------|---------------|
| **Financial Core** | 🔴 Critical | Toàn bộ flow tiền | PHẢI có rollback plan, test trên staging TRƯỚC |
| **Banking Connectors** | 🟡 High | Flow nạp/rút cụ thể | Timeout 60s+, circuit breaker check |
| **Gateways** | 🟢 Medium | Traffic entry | Rate limiting verify, zero-downtime deploy |
| **Business** | 🔵 Low | Feature cụ thể | Integration test đủ |
| **Utilities** | ⚪ Lowest | Support chức năng | Basic tests đủ |

### 3. Lập Delegation Prompt (BẮT BUỘC ghi vào `08_tasks.md` hoặc `08_tasks_[issue].md`)

> **QUY TẮC MỚI**: Tuyệt đối KHÔNG ĐƯỢC tạo các file `delegate_*.md` rác rưởi. Mọi task cho Muscle phải được ghi trực tiếp vào `08_tasks.md` hoặc file task theo issue (VD: `08_tasks_schema_detection.md`).

Format khi thêm task vào `08_tasks*.md`:
```markdown
## Task: [Tên Task]
- **Phase**: GĐ<0-4>
- **Service Group**: <Financial Core / Banking / Gateway / Business / Utilities>
- **Service(s)**: <tên service cụ thể>
- **Mô tả**: <cụ thể, không mơ hồ>
- **Trạng thái**: [ ] TODO (chưa thực hiện) / [x] DONE (đã thực hiện)

### [Context]
- Current state: <trích từ active_plans.md>
- Dependencies: <services bị ảnh hưởng>
- ADR liên quan: <ADR-00X nếu có>
- Logs/Error: <paste thực tế>

### [Definition of Done]
- [ ] Điều kiện 1: <đo lường được>
- [ ] Điều kiện 2: <đo lường được>
- [ ] **[QA Gate]**: workflow `/qa-agent` check coverage/tests
- [ ] **[Security Gate]**: workflow `/security-agent` check vuln
- [ ] Blast radius verified
- [ ] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.
```

> **CẢNH BÁO**: KHÔNG delegate mơ hồ. KHÔNG skip Phase Check.

### 3b. Cập nhật `05_progress.md` theo format Kanban

File `05_progress.md` không chỉ là text log. Mọi entry CẦN có trạng thái và thời gian. Format bắt buộc:
`| [Timestamp] | [Agent/Model] | [Trạng thái: TODO/DOING/DONE] | [Thời gian thực hiện] | [Hành động] |`

### 4. Chọn Sub-agents (Phase-Aware)

Dựa vào Phase + Service Group ở step 1-2, chọn pipeline:

**GĐ0-1 (Infrastructure)**:
```
/debug-agent → /infra-validator → /muscle-execute → /qa-agent
```

**GĐ2 (Resilience)**:
```
/debug-agent → /muscle-execute → /security-agent → /qa-agent
```

**GĐ3 (Architecture)**:
```
/service-migration → /muscle-execute → /security-agent → /qa-agent
```

### 5. Dispatch & Monitor

Sau khi dispatch, chuyển sang `/monitor-agent`:
- KHÔNG can thiệp logic implementation
- CHỈ can thiệp khi sai hướng chiến lược
- **Quota Watch**: Nếu Muscle báo lỗi 429, hướng dẫn xoay vòng key hoặc switch model.
- Verify blast radius match với Step 2 assessment

## Anti-patterns

- ❌ Brain tự sửa code
- ❌ Delegate Financial Core service KHÔNG có rollback plan
- ❌ Bỏ qua Phase Check → chọn sai sub-agents
- ❌ Không update memory sau khi task xong

## Definition of Done
- [ ] Đã đọc context workspace trước khi delegate
- [ ] Delegation Prompt có đủ 3 phần: Task Description + Context + DoD
- [ ] Sub-agent pipeline đã được chọn phù hợp với Phase + Risk Level
- [ ] Sau khi Muscle xong: `active_plans.md` và `05_progress.md` đã được cập nhật
