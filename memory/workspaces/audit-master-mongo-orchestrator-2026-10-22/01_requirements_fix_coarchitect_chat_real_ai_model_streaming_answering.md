# 01. Requirements — Đấu Nối 100% Model AI Thật Vào Co-Architect Chat (Chấm Dứt Hoàn Toàn Việc Ghép Chuỗi Tĩnh & Nuốt JSON Model) (`fix_coarchitect_chat_real_ai_model_streaming_answering`)

- **Tier:** T2
- **Suffix:** fix_coarchitect_chat_real_ai_model_streaming_answering
- **Mục tiêu:**
  - Chấm dứt 100% việc ghép chuỗi văn mẫu tĩnh (`"Dạ anh, đối với yêu cầu 'mày trả lời nhanh không' trên task ...: Nguyên nhân chính là..."`) trong Co-Architect Studio (`#co-architect-studio`) trên CMS WebUI (`:8090`).
  - Đấu nối Model AI thật sự (`AntigravityRunner` với model `gemini-2.5-flash` hoặc model User chọn) vào toàn bộ các câu hỏi đối thoại trong chế độ `"discuss"`.
  - Khắc phục lỗi `streamAntigravityOutput` trong `internal/runner/antigravity.go` nuốt mất dòng JSON của model khi model trả về JSON object 1 dòng.
  - Xóa bỏ việc `renderContextualDiscussReply` ghi đè (`overwrite`) câu trả lời của Model bằng hàm `synthesizeProgressAwareDirectAnswer`.
  - Phản hồi nhanh (`2–5s`), thông minh, đúng chất Architect / AI Agent đối với mọi câu hỏi của User (câu hỏi chào hỏi, hỏi tiến độ, hỏi kỹ thuật, hỏi kiến trúc).
