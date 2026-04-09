# 01_requirements.md - Requirements & Voice of Customer

## Trigger
> "tôi muốn tách ra chỗ memory này. 1 là memory nó sẽ giữ toàn bô info dự án tổng thể. còn wokspace... là thông tin của feature"

## Functional Requirements
1.  **Global Memory**: Store project-wide info (Context, Tech Stack, Decisions).
2.  **Workspace Memory**: Store feature-specific info (Plans, Progress, DB Docs).
3.  **Consolidation**: Merge `work-db` and `work-desc` context into the feature memory.
4.  **Workflow Support**: `context-manager` must handle "Restore" (Read) and "Save" (Write) for both scopes.

## Constraints
- Must not lose existing information during migration (Rule #1: Preservation).
