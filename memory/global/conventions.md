# Agent Conventions (Luật Bất Thành Văn)

## 1. Preservation of Context (Bảo toàn ngữ cảnh)
> **Rule**: Không bao giờ được xóa logic, thông tin, hoặc liên kết với task/source cũ một cách tùy tiện.

- **Khi Refactor**: Phải đảm bảo tính năng tương đương (Parity). Nếu workflow A gọi workflow B, phiên bản mới của A vẫn phải gọi B (hoặc thay thế tương đương).
- **Khi Update Docs**: Không xóa thông tin lịch sử quan trọng. Dùng `Deprecated` thay vì xóa hẳn nếu cần.
- **Lý do**: Project lớn có sự phụ thuộc chằng chịt. Xóa 1 dòng có thể làm gãy quy trình của người khác hoặc Agent khác.

## 2. Explicit Linking (Liên kết rõ ràng)
- Workflow phải định nghĩa rõ: Trigger từ đâu? Output đi đâu?
- Không dùng từ chung chung ("Task xong"). Phải cụ thể ("Sau khi `/muscle-execute` trả về `exit code 0`").

## 3. Role-Based Actions & Audit Log Format
- Luôn chỉ định rõ **Ai** làm gì (**Brain** vs **Muscle**).
- **Brain**: Phân loại Tier (`T0–T3`), thiết kế `09_tasks_solution_*.md`, chạy cổng quyết định `laya_gate.py` (`DG1`, `DG2`), trình User chờ `APPROVE`.
- **Muscle**: Chỉ sửa source code sau khi có `APPROVE` (cưỡng chế bởi Hook `H4`); chạy `checks` và `runtime_check` theo `.agent-project.json`.
- **Progress Tracking**: Mọi dòng trong `05_progress_*.md` PHẢI tuân thủ định dạng: `- [YYYY-MM-DDTHH:MM:SS+07:00] [Agent:<Model>] <Action>`.

## 4. Phân tầng Tài liệu Workspace (Tiers `T0–T3` — ADR-03)
Mọi Workspace tuân thủ Mandatory Doc Registry (`00–14`, chi tiết tại `skills/workspace-docs/SKILL.md`):
- **Tier `T0` (Hỏi đáp/Tư vấn, không sửa file):** Không tạo workspace, không chạy linter.
- **Tier `T1` (Hotfix/Micro-task):** Tối thiểu 3 file: `01_requirements_*.md` (`tier: T1`), `05_progress_*.md`, `08_tasks_*.md`.
- **Tier `T2` (Feature/Bugfix):** `T1` + `09_tasks_solution_*.md`, `06_validation_*.md`, `11_report_*.md` (`tier: T2`).
- **Tier `T3` (Epic/Phase lớn):** Bộ đầy đủ `00, 01, 02, 04, 05, 06, 08, 09, 11, 12, 13` (`tier: T3`).

## 5. Cuối Phiên & Chữ ký Phản hồi (Rule #0 & S6)
Cuối mỗi câu trả lời, Agent bắt buộc:
1. Liệt kê các bài học (`lessons.md`) đã tuân thủ hoặc mắc phải trong phiên.
2. Liệt kê danh sách `Skills` và `Tools` đã sử dụng.

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

## 9. Runtime Verification (Quy tắc Verify trước khi Done) — Added 2026-04-13
> **Rule**: Build pass ≠ Done. Brain PHẢI verify runtime trước khi báo hoàn thành.

- **Bắt buộc trước khi báo "done"**:
    1. Build pass ✅
    2. Migration: auto-migrate trong code startup HOẶC ghi rõ "cần chạy migration X"
    3. API: handle empty data + missing table gracefully (không crash/500)
    4. Logic check: so sánh output với TỪNG mong muốn trong plan
    5. Workspace docs: update 00_context/02_plan nếu có thay đổi code
    6. Lesson: ghi ngay nếu user yêu cầu
- **Nếu không thể verify runtime**: Nói thẳng "Chưa verify runtime, cần user test X, Y, Z"
- **TUYỆT ĐỐI KHÔNG** báo "done" chỉ vì build pass

## 10. Brain Self-Review (Quy tắc Tự kiểm tra Logic) — Added 2026-04-13
> **Rule**: Sau mỗi block code, Brain PHẢI review logic trước khi tiếp tục.

- **Checklist**:
    1. Code này có thực sự giải quyết đúng yêu cầu user không?
    2. Edge case: table rỗng? Data chưa có? API chưa chạy?
    3. So sánh từng mong muốn trong plan → code đã cover hết chưa?
    4. Nếu có N mong muốn, phải verify N kết quả — không skip cái nào
- **Khi user báo lỗi**: ĐIỀU TRA root cause trước, không sửa bề mặt