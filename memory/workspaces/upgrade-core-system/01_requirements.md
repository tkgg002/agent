# 01_requirements.md - Requirements

## Trigger
> User: "Upgrade core hoàn chỉnh nhất trước khi vào làm GooPay Refactor"

## Functional Requirements
1. **Memory System**: lessons.md có format chuẩn, searchable bằng tags
2. **Session Protocol**: Brain có checklist rõ ràng đầu/cuối phiên — không được skip
3. **Workflow Automation**: session-start + session-wrapup workflows, mọi workflow có DoD
4. **Workspace Standard**: V3 hoàn chỉnh, conventions rõ ràng
5. **Decision Log**: Trace lịch sử quyết định quan trọng của Brain

## Constraints
- Không thay đổi cấu trúc folder hiện tại
- GEMINI.md thay đổi tối thiểu — chỉ bổ sung, không xóa rule cũ
- Phải hoàn thành trước khi bắt đầu workspace `feature-refactor-2026`
