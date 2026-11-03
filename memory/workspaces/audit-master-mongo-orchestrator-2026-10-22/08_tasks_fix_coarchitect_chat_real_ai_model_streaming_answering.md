# 08. Tasks — Đấu Nối 100% Model AI Thật Vào Co-Architect Chat (`fix_coarchitect_chat_real_ai_model_streaming_answering`)

### [ ] `TASK-AI-01`: Xóa Bỏ Điều Kiện Chặn `if actionMode != "discuss"` & Khắc Phục Lỗi `streamAntigravityOutput` Nuốt Mất Output Model
- **Mô tả & Context (`file:line`):**
  - `internal/runner/antigravity.go:232-242`:
    - Sửa `streamAntigravityOutput`: khi gặp dòng bắt đầu bằng `{`, nếu `extractStreamJSONText(ev)` trả về chuỗi rỗng và dòng đó không phải là event streaming của Claude (không có key `type`), **không được `continue` nuốt mất dòng đó**, mà phải chuyển toàn bộ dòng đó vào `Token{Text: line + "\n"}` để ứng dụng nhận được đầy đủ chuỗi JSON của Model!
    - Đồng thời trong `buildGeminiCLIArgs` (`antigravity.go:174-184`), luôn truyền `-m <model>` cho Gemini CLI (mặc định `-m gemini-2.5-flash` khi model là `gemini-4-argon` / `antigravity`) để phản hồi cực nhanh (`~3–5s`).
  - `internal/webui/task_llm_bridge.go:170-245`:
    - Xóa bỏ hoàn toàn điều kiện `if actionMode != "discuss"` để 100% tin nhắn chat của User đều được gửi tới Model AI thật sự.
- **Logs / Test:**
  - Chạy `gemini --skip-trust -e none -m gemini-2.5-flash` và unit test `TestAntigravityRunner` qua `evidence_run.py`.
- **Definition of Done (DoD):**
  - Model AI thật sự nhận prompt và trả về output trực tiếp, không bị nuốt mất token hay rơi vào fallback ghép chuỗi.

### [ ] `TASK-AI-02`: Giữ Nguyên 100% Câu Trả Lời Của Model AI, Tuyệt Đối Cấm Ghi Đè Bằng Template Chuỗi Tĩnh
- **Mô tả & Context (`file:line`):**
  - `internal/webui/task_chat_knowledge.go:30-65`:
    - Xóa bỏ đoạn `if !isProgressOrStatusQuestion(cleanMsg)` ghi đè `rootCauses`, `solutions`, `directAnswer` bằng `fbRC`, `fbSol` và `synthesizeProgressAwareDirectAnswer`.
    - Khi Model trả về `llmRes`, ưu tiên giữ nguyên 100% câu trả lời thông minh của Model (`llmRes.DirectAnswer`). Chỉ bổ sung thông tin từ đĩa khi Model yêu cầu hoặc thiếu.
  - `internal/webui/task_llm_bridge.go:92-160`:
    - Prompt gửi Model AI: Cung cấp đầy đủ bối cảnh Task (`Task Key`, `Title`, `Description`, `Stage`, các Sub-Tasks hiện có trên đĩa) và yêu cầu Model trả lời trực tiếp, tự nhiên bằng tiếng Việt vào câu hỏi của User (ví dụ khi User hỏi *"mày trả lời nhanh không"*, Model trả lời thẳng thắn và lịch sự về tốc độ và khả năng của mình, không ghép nối văn mẫu đề bài).
- **Logs / Test:**
  - Bắn request `POST /api/tasks/chat` với câu hỏi `"mày trả lời nhanh không"` trên `core-003` và xác nhận câu trả lời là văn phong AI tự nhiên từ Model.
- **Definition of Done (DoD):**
  - Co-Architect Studio trả lời bằng trí tuệ nhân tạo thực sự của Model, thời gian phản hồi `< 5s`, tuyệt đối không còn câu ghép chuỗi vô hồn.
