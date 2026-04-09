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
- **Brain**: Phải xác định model/provider phù hợp trước khi delegate.
- **Muscle**: Phải nạp `models.env` để sử dụng model tối ưu chi phí (Flash/Haiku) và hỗ trợ retry/rotate key khi đạt quota.
- **Identity Verification**: Trước mỗi Task/Phase, phải chạy lệnh `env | grep MODEL` để xác thực model đang chạy khớp với cấu hình.
- **Progress Tracking**: Mọi thay đổi trong `05_progress.md` PHẢI bắt đầu bằng tag model: `[Model-X] Task Description`.
- Tránh passive voice ("File được cập nhật"). Dùng active voice ("Brain cập nhật file").

## 4. V3 Workspace Standard (Cấu trúc Thư mục Chuẩn)
Mọi Workspace (Task/Feature) mới PHẢI tuân thủ cấu trúc 8 file:
1. `00_context.md`: Context & Scope.
2. `01_requirements.md`: Yêu cầu & Feedback của User.
3. `02_plan.md`: Chiến lược thực thi (Active Plan + Checklist).
4. `03_implementation.md`: Chi tiết kỹ thuật (Tech Specs).
5. `04_decisions.md`: Quyết định quan trọng (ADRs local của workspace).
6. `05_progress.md`: Nhật ký công việc (Chronological Log).
7. `06_validation.md`: Kết quả nghiệm thu (QA/Tests).
8. `07_lessons.md`: Lessons learned riêng của workspace này (filter từ lessons.md global).

## 5. Protocol Signature (Chữ ký Giao thức)
Mọi phản hồi cuối cùng của Agent PHẢI có chữ ký:
```markdown
**Role**: [Brain/Muscle]
**Model**: [e.g. gemini-3-flash | gemini-3-pro-high]
**Provider**: [Antigravity | OpenAI | Anthropic]
**Task**: [Tên Task hiện tại]
**Exec**: [Hành động tóm tắt]
**Skills**: `skill_1`, `skill_2`...
```

## 6. Workspace Isolation (Cô lập Workspace)
> **Rule**: Mỗi workspace là 1 project/feature LỚN **hoàn toàn độc lập**. Chúng KHÔNG liên lạc nhau.

- Workspace = 1 domain riêng biệt (ví dụ: GooPay Refactor, Upgrade Core System...)
- Brain chỉ load context của workspace **đang làm việc** trong phiên hiện tại
- `active_plans.md` chỉ là **registry** (danh bạ) — không phải cơ chế inter-agent communication
- Không bao giờ tạo dependency chéo giữa các workspace

## 7. New Task = New Workspace (Quy tắc Khởi tạo)
> **Rule**: Mọi task lớn (từ 3+ bước, có scope rõ ràng) PHẢI được khởi tạo workspace ngay khi bắt đầu.

- **Đúng**: User giao task → Brain tạo workspace → Brain lập plan → Bắt đầu làm
- **Sai**: Brain làm việc, lên plan, rồi mới nhớ ra tạo workspace
- Nếu Brain quên tạo workspace → đây là lesson phải ghi vào `lessons.md`

## 8. Final Compliance & Rule Check (Quy tắc Tự kiểm tra)
> **Rule**: Trước khi kết thúc bất kỳ task nào, Brain BẮT BUỘC phải thực hiện bước kiểm tra cuối cùng về tính tuân thủ.

- **Mandatory Sub-task**: Sau khi hoàn thành các bước thực thi chính, Brain phải tự tạo (hoặc cập nhật) một task nội bộ mang tên "Final Compliance & Rule Check".
- **Checklist**:
    1. Đã cập nhật `task.md`, `implementation_plan.md`, `walkthrough.md` chưa?
    2. Các file artifacts đã được đồng bộ với thay đổi thực tế chưa?
    3. Metadata trong progress log và signature đã đúng format chưa?
    4. Mọi quyết định quan trọng đã được ghi vào `04_decisions.md` (nếu có)?
- **Lý do**: Đảm bảo Brain không bị cuốn vào code mà bỏ quên nhiệm vụ "Chairman" (Quản lý và Lưu trữ tri thức).
