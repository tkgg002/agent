# 08. Tasks — Khắc phục Lỗi Không Trả Lời Câu Hỏi Kỹ Thuật (`Vì Sao Chạy Không Đúng, Giải Pháp Là Gì`) trong `💬 THẢO LUẬN KIẾN TRÚC` (`fix_coarchitect_chat_answering_root_cause_and_solution`)

## Danh sách Hạng mục Thực thi (Sau khi User APPROVE)

### `TASK-CHAT-01` — Nâng cấp `buildContextualDiscussReply` & Bộ bóc tách Hồ sơ Kỹ thuật Đa tầng trong `internal/webui/task_chat.go`
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_chat.go:1325-1399`:
    - Loại bỏ hoàn toàn mẫu trả lời khuôn mẫu nhại lại câu hỏi ở cuối `buildContextualDiscussReply`.
    - Bổ sung hàm `extractDeepRootCauseAndSolutionForChat(qt, isoMeta, subTasks)`:
      - Đọc kết hợp `09_tasks_solution.md`, `13_analysis.md`, `audit_report.md` tại `.orchestrator/tasks/<TaskKey>/` **VÀ** hồ sơ gốc trong `/Users/trainguyen/Documents/work/agent/memory/workspaces/` (qua `FindMatchingAgentWorkspace` hoặc tìm kiếm theo từ khóa đề bài như `shadow_master_update_drop` cho `CDC-RECON-01`).
      - Trích xuất đầy đủ các mục **Nguyên nhân gốc rễ (`F1..F8` / Root Cause kèm `file:line`)**, **Giải pháp kỹ thuật (`Solution` cho từng sơ hở)**, **Khối `Code Demo`**, và **Danh mục Sub-Tasks / Điểm chạm trên từng Repo**.
    - Phân tích ý định câu hỏi của User trong `buildContextualDiscussReply` (`vì sao`, `tại sao`, `nguyên nhân`, `ko đúng`, `sai`, `giải pháp`, `sửa gì`, `làm sao`, `code demo`, `file nào`, `subtask`, hoặc câu hỏi bất kỳ) để trả lời trực tiếp, mạch lạc, đúng trọng tâm câu hỏi của User.
- **Logs / Test:**
  - Unit & Integration test `TestCoArchitectChat_AnswersWhyReconShadowToMasterWrongAndSolution` trong `internal/webui/server_test.go`.
- **Definition of Done (DoD):**
  - Khi gửi `"rồi vì sao chạy ko đúng, giải pháp là gì"` vào `POST /api/tasks/chat` cho `CDC-RECON-01` (hoặc bất kỳ task nào), phản hồi trả về đầy đủ các nguyên nhân gốc rễ (`F1..F8` kèm `file:line`), giải pháp kỹ thuật cho từng nguyên nhân và Code Demo thực tế; tuyệt đối không còn câu trả lời khuôn mẫu nhại lại câu hỏi.

### `TASK-CHAT-02` — Đồng bộ Hồ sơ `13_analysis.md` & Cập nhật Câu trả lời Thực chất vào `chat_history.json` của `CDC-RECON-01` + Hỗ trợ Hỏi-Đáp trên `telegram_bridge.go`
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:1540-1620` & `.orchestrator/tasks/CDC-RECON-01/13_analysis.md`, `chat_history.json`:
    - Khi task liên kết với workspace `/agent` có sẵn `13_analysis_*.md` hoặc `09_tasks_solution_*.md` giàu nội dung thực chứng (như `audit-shadow-master-update-drop-2026-10-20` cho `CDC-RECON-01`), bảo toàn nội dung phân tích gốc (`F1..F8`) thay vì ghi đè bằng khung scanner mẫu.
    - Cập nhật lại tin nhắn trả lời lúc `15:21:12` trong `.orchestrator/tasks/CDC-RECON-01/chat_history.json` bằng bản giải trình đầy đủ Nguyên nhân (`F1..F8`) & Giải pháp kỹ thuật.
  - `internal/webui/telegram_bridge.go:495-535`:
    - Khi User nhắn câu hỏi (`vì sao`, `tại sao`, `nguyên nhân`, `giải pháp`, `?`) trên Telegram, tự động định tuyến sang `buildContextualDiscussReply` để trả lời trực tiếp trên Telegram.
- **Logs / Test:**
  - Kiểm chứng trực tiếp qua `GET /api/tasks/chat?task_key=CDC-RECON-01` và `POST /api/tasks/chat` trên `:8090` (`evidence_run.py`).
- **Definition of Done (DoD):**
  - Giao diện `💬 THẢO LUẬN KIẾN TRÚC` trên `:8090` và Telegram trả lời thông minh, chi tiết, đúng bản chất kỹ thuật kèm `file:line`.
