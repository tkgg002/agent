# Agent Conventions (Luật Bất Thành Văn)

## 1. Preservation of Context (Bảo toàn ngữ cảnh)
> **Rule**: Không bao giờ được xóa logic, thông tin, hoặc liên kết với task/source cũ một cách tùy tiện.

- **Khi Refactor**: Phải đảm bảo tính năng tương đương (Parity). Nếu workflow A gọi workflow B, phiên bản mới của A vẫn phải gọi B (hoặc thay thế tương đương).
- **Khi Update Docs**: Không xóa thông tin lịch sử quan trọng. Dùng `Deprecated` thay vì xóa hẳn nếu cần.
- **Lý do**: Project lớn có sự phụ thuộc chằng chịt. Xóa 1 dòng có thể làm gãy quy trình của người khác hoặc Agent khác.

## 2. Explicit Linking (Liên kết rõ ràng)
- Workflow phải định nghĩa rõ: Trigger từ đâu? Output đi đâu?
- Không dùng từ chung chung ("Task xong"). Phải cụ thể ("Sau khi `/muscle-execute` trả về `exit code 0`").

## 3. Role-Based Actions
- Luôn chỉ định rõ **Ai** làm gì (**Brain** vs **Muscle**).
- Tránh passive voice ("File được cập nhật"). Dùng active voice ("Brain cập nhật file").

## 4. V3 Workspace Standard (Cấu trúc Thư mục Chuẩn)
Mọi Workspace (Task/Feature) mới PHẢI tuân thủ cấu trúc 7 file:
1. `00_context.md`: Context & Scope.
2. `01_requirements.md`: Yêu cầu & Feedback của User.
3. `02_plan.md`: Chiến lược thực thi (Active Plan).
4. `03_implementation.md`: Chi tiết kỹ thuật (Tech Specs).
5. `04_decisions.md`: Quyết định quan trọng (ADRs).
6. `05_progress.md`: Nhật ký công việc (Chronological Log).
7. `06_validation.md`: Kết quả nghiệm thu (QA/Tests).

## 5. Protocol Signature (Chữ ký Giao thức)
Mọi phản hồi cuối cùng của Agent PHẢI có chữ ký:
```markdown
**Role**: [Brain/Muscle]
**Task**: [Tên Task hiện tại]
**Exec**: [Hành động tóm tắt]
**Skills**: `skill_1`, `skill_2`...
```

