---
tier: T2
mode: standard
suffix: headless_hidden_antigravity_execution
---

# 01 Requirements — Giải Pháp Chạy Ẩn (Headless / Hidden Conversation) Cho Model `gemini-4-argon (antigravity)` Không Làm Rác Antigravity IDE

## 1. Bối Cảnh & Yêu Cầu Cốt Lõi Của User
- **Nguyên văn yêu cầu User:**
  > *"rồi chạy các bước Wave (S1/S2.5 Critique, S3 Plan, Sub-Task Plan, Rep Update Plan / Chat, S4 Muscle CodeGen, 5B Semantic Audit & S6 Finalize) vào AI Model thật trên cùng phiên Conversation của Task có bắn send-message ko. vậy rác cái ide của tao lắm. mày xem lại giải pháp chạy ẩn đi, xem đc không"*
- **Vấn đề thực tế:**
  - Nếu dùng `agentapi new-conversation` hoặc `agentapi send-message <convID>` trực tiếp mà không ẩn hội thoại (`hideFromConversationList: false`) hoặc bắn tin nhắn vào conversation hiển thị trên sidebar của Antigravity IDE, mỗi bước Wave (`S1/S2.5 Critique`, `S3 Plan`, `Sub-Task Plan`, `Rep Update Plan / Chat`, `S4 Muscle CodeGen`, `5B Semantic Audit`, `S6 Finalize`) sẽ liên tục đẩy prompt JSON hệ thống dài hàng chục ngàn ký tự vào giao diện IDE của User, gây rác danh sách hội thoại và làm phiền cửa sổ làm việc trên Antigravity IDE.

## 2. Ràng Buộc Kỹ Thuật Bất Biến (Lesson-as-Enforced-Constraint)
1. **Bắt buộc dùng đúng Model 4.0 của Antigravity (`gemini-4-argon (antigravity)`)**: Tuyệt đối cấm quay lại quét npm `@google/gemini-cli` (`L002`) hoặc hạ cấp xuống `gemini-2.5-flash` (`L003`, `L004`).
2. **Không làm rác Antigravity IDE của User**: Kiểm tra cơ chế thực tế của `/Applications/Antigravity.app/Contents/Resources/bin/language_server` (gRPC/Connect-RPC local port, CSRF token, `StartCascade` / `SendUserCascadeMessage`, cờ `hideFromConversationList: true`, `subagentSpec`, hoặc cơ chế xóa/ẩn conversation khỏi danh sách hiển thị IDE).
3. **Kỷ luật Propose-Only (Rule #13)**: Hiện tại chỉ điều tra thực chứng (`S2`), phản biện `DG0` (`S2.5`), thiết kế 1 phương án tối ưu nhất kèm Code Demo (`S3`), kiểm định `DG1/DG2` và trình User xem xét `APPROVE` — **tuyệt đối không sửa source code trước khi User `APPROVE`**.
