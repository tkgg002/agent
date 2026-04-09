# Kế hoạch nâng cấp hạ tầng (Core: agent/)

Kế hoạch này thực hiện đồng bộ hóa bộ công cụ kỹ thuật từ `everything-claude-code-main` (v1.10.0) nhưng lấy hệ điều hành `@[agent](file:///Users/trainguyen/Documents/work/agent)` làm trung tâm chỉ huy.

## User Review Required

> [!IMPORTANT]
> - **Cấu trúc Core**: Mọi thay đổi sẽ phải tuân thủ quy trình Brain/Muscle định nghĩa trong `agent/GEMINI.md`.
> - **Bản sắc Agentic**: Chúng ta không chỉ copy file, mà sẽ "hấp thụ" tính năng mới của v1.10.0 vào trong Core.
> - **Muscle Upgrade**: Các skills/rules kỹ thuật sẽ được đẩy vào hạ tầng kỹ thuật (`.agent/` và global skills) để Muscle sử dụng.

## Proposed Changes

### [Phase 1: Muscle Infrastructure Upgrade]
Mục tiêu: Cập nhật "Cơ bắp" cho Agent.

#### [MODIFY] [Hạ tầng Rules kỹ thuật](file:///Users/trainguyen/Documents/work/.agent/rules)
- Cập nhật các quy tắc ngôn ngữ mới (Go, TS, Python v.v.) từ bản v1.10.0.

#### [MODIFY] [Hạ tầng Kỹ năng (Global Skills)](file:///Users/trainguyen/.gemini/antigravity/skills)
- Cập nhật/Thêm mới các skills kỹ thuật (181 skills). Đây là kho vũ khí để Muscle triệu hồi.

### [Phase 2: Agentic Core Enhancement]
Mục tiêu: Tích hợp các bộ óc vận hành mới của v1.10.0 vào Brain.

#### [MODIFY] [agent/workflows/](file:///Users/trainguyen/Documents/work/agent/workflows)
- Xem xét tích hợp các "Operator Workflows" mới (như `project-flow-ops`, `workspace-surface-audit`) vào trong quy trình điều phối của Brain.

#### [MODIFY] [agent/GEMINI.md](file:///Users/trainguyen/Documents/work/agent/GEMINI.md)
- Củng cố Quy tắc #10 về việc `agent/` là Core ghi đè lên toàn bộ hệ thống.

## Verification Plan

### Automated/Manual Verification
1. **Kiểm tra vai trò**: Chạy lệnh `/brain-delegate` để xác nhận Brain vẫn nắm quyền chỉ huy sau khi Muscle được nâng cấp vũ khí.
2. **Kiểm tra công cụ**: Muscle thực hiện thử một task kỹ thuật sử dụng Skill mới (ví dụ: `brand-voice` hoặc `social-graph-ranker`) để verify hạ tầng Muscle đã sẵn sàng.
3. **Audit Trạng thái**: Đảm bảo `agent/memory/workspaces` vẫn ghi nhận chính xác tiến độ mà không bị reset bởi quá trình cập nhật.
