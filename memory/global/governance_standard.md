# Project Brain Governance Standard (Rule 7)

Tài liệu hướng dẫn thực thi (Operational Guide) cho Quy tắc số 7, đảm bảo tính nhất quán, trực quan và khả năng truy vết tuyệt đối của dự án GooPay.

## 1. Nguyên tắc Cốt lõi (Core Principles)

- **Workspace-First**: Cấm thảo luận hoặc thảo giải pháp mà không có file vật lý trong Workspace folder (`agent/memory/workspaces/[FeatureName]`).
- **Immutable Log (05_progress.md)**: Nhật ký tiến độ là lịch sử Audit không thể bị xóa bỏ. Chỉ cho phép APPEND. Quy tắc: "Sai thì ghi đè bằng dòng Log sau là Phản hồi/Revert, tuyệt đối không sửa dòng cũ".
- **Metadata Integrity**: Mọi dòng trạng thái PHẢI đi kèm: `[YYYY-MM-DD HH:mm] [Agent:Model ID] Action`.
- **Knowledge Loop**: Mọi "vấp ngã" (User sửa lưng) đều phải được chuyển hóa thành **Global Pattern** trong `lessons.md`.

## 2. Hệ thống Định danh Prefix (Mandatory Doc Registry)

Mọi tệp tin trong Workspace folder PHẢI bắt đầu bằng số thứ tự sau:

| Prefix | Loại tài liệu | Nội dung chi tiết |
| :--- | :--- | :--- |
| `00` | Context & Scope | Phạm vi, kiến trúc tổng quát, các thành phần liên quan. |
| `01` | Requirements | Specs chi tiết, User Stories, Luồng nghiệp vụ. |
| `02` | High-level Plan | Roadmap cao tầng của toàn bộ Phase. |
| `03` | Tech Design | Thiết kế kỹ thuật chi tiết (Implementation Plan). |
| `04` | Decision Log | Các quyết định kiến trúc (ADRs). |
| `05` | Progress Log | **Audit Log tối cao (Append ONLY)**. |
| `06` | Validation Plan | Kế hoạch kiểm thử (Test Cases, QA workflow). |
| `07` | Status Report | Báo cáo hiện trạng session/phase. |
| `08` | Tasks Breakdown | Checklist Task chi tiết (TODO list). |
| `09` | Tech Solution | Hồ sơ giải pháp cụ thể (Code snippets, logic mapping). |
| `10` | Gap Analysis | Phân tích lỗ hổng kiến trúc và giải pháp bù đắp. |

## 3. Metadata Format Standard

Mẫu định dạng chuẩn cho `05_progress.md`:
```markdown
| Timestamp | Agent | Model | Action |
| :--- | :--- | :--- | :--- |
| [2026-04-06 15:38] | Brain | gemini-3-flash | Created 08_tasks_rule_7_refactoring.md |
| [2026-04-06 15:39] | Brain | gemini-3-flash | Created governance_standard.md (Global Memory) |
```

> [!IMPORTANT]
> **No Overwrite Policy**: TUYỆT ĐỐI CẤM dùng `Overwrite: true` trên bất kỳ Memory file nào. Mọi hành vi vi phạm sẽ bị coi là Critical Failure và phải thực hiện RCA ngay lập tức.
>
> **RCA Responsibility**: Brain có trách nhiệm tự phát hiện vi phạm Governance và ghi dòng log RCA vào `05_progress.md` TRƯỚC khi thực hiện tiếp task.
