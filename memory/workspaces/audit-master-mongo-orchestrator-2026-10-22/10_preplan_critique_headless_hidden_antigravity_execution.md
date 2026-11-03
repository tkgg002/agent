---
workspace: audit-master-mongo-orchestrator-2026-10-22
suffix: headless_hidden_antigravity_execution
tier: T2
---
# S2.5 Pre-Plan Adversarial Critique (DG0) — Chạy Ẩn 100% (`Headless Ephemeral Hidden Execution`)

## 1. Phản biện Giả định (Adversarial Critique vs Ground Truth)

### Giả định 1: "Gom các bước Wave vào 1 Conversation dùng chung (`send-message`) là đủ để đỡ rác IDE"
- **Bác bỏ bằng Ground Truth (`internal/runner/antigravity.go:446-486`):**
  - Dù chỉ tạo 1 Conversation cho mỗi Task (`TASK-2`), nhưng vì không gắn `hideFromConversationList: true` và liên tục bắn `send-message` cho mọi bước Wave (`S1/S2.5 Critique`, `S3 Plan`, `Sub-Task Plan`, `Rep Update Plan / Chat`, `S4 Muscle CodeGen`, `5B Semantic Audit`, `S6 Finalize`), User vẫn nhìn thấy phiên Conversation đó nhảy lên đầu danh sách Antigravity IDE và bị nhồi hàng chục message JSON/code hỗn tạp.
  - Hơn nữa, việc nhồi chung tất cả các bước Wave (vốn có System Prompt và vai trò khác nhau: Brain Architect vs Muscle Coder vs QA Auditor) vào chung 1 lịch sử chat (`send-message`) còn làm ô nhiễm ngữ cảnh (context pollution) giữa các bước và làm chậm tốc độ inference ở các bước cuối.

### Giả định 2: "`agentapi` bắt buộc phải hiển thị Conversation trên UI thì mới chạy được model Antigravity 4.0"
- **Bác bỏ bằng Ground Truth (`/Applications/Antigravity.app/Contents/Resources/bin/language_server`):**
  - Kiểm tra thực tế binary `language_server` cho thấy `agentapi new-conversation` hỗ trợ chính thức biến môi trường `ANTIGRAVITY_SOURCE_METADATA='{"hideFromConversationList":true}'` ánh xạ trực tiếp vào protobuf `exa.cortex_pb.SourceMetadata.HideFromConversationList` (field 3).
  - Khi bật `hideFromConversationList: true`, phiên thực thi hoàn toàn vô hình trên danh sách Conversation của Antigravity IDE nhưng vẫn chạy model Antigravity 4.0 thật và ghi kết quả vào `transcript_full.jsonl`.
  - Ngay sau khi đọc xong `PLANNER_RESPONSE` (`status == "DONE"`), gọi thêm Connect-RPC `POST http://${ANTIGRAVITY_LS_ADDRESS}/exa.language_server_pb.LanguageServerService/DeleteCascadeTrajectory` với `{"cascadeId":"<convID>"}` và `os.RemoveAll(brainDir)` sẽ xoá sạch hoàn toàn phiên tạm đó khỏi cả `language_server` lẫn đĩa cứng.

## 2. Kết luận Tiền Kế Hoạch (DG0)
- Giải pháp chạy ẩn 100% (**Headless Ephemeral Hidden Execution**) trên model Antigravity 4.0 là **hoàn toàn khả thi và đã được kiểm chứng vật lý 100% trên máy thật**:
  1. Mỗi lần gọi `completeWithAntigravityAgentAPI` sẽ chạy một phiên ẩn độc lập (`ANTIGRAVITY_SOURCE_METADATA={"hideFromConversationList":true}`), **không dùng `send-message` nối đuôi** làm rác luồng chat và làm bẩn context giữa các Wave.
  2. Ngay khi nhận xong phản hồi (`defer`), tự động gọi `DeleteCascadeTrajectory` qua Connect-RPC và xoá thư mục `brain/<convID>` (đồng thời dọn luôn các `agentapi_conv_id.txt` cũ nếu có).
  3. Tự động dò `ANTIGRAVITY_LS_ADDRESS` và `ANTIGRAVITY_CSRF_TOKEN` từ tiến trình `language_server` đang chạy nếu biến môi trường chưa có.
