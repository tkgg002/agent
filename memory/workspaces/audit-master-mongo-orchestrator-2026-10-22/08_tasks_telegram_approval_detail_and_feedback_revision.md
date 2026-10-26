# 08. Tasks Breakdown: Hồ Sơ Duyệt Giải Pháp S3 Đầy Đủ & Luồng Phản Hồi Sửa Giải Pháp Trên Telegram (`telegram_approval_detail_and_feedback_revision`)

> **Workspace**: `audit-master-mongo-orchestrator-2026-10-22`
> **Tier**: `T2`
> **Target Repository**: `/Users/trainguyen/Documents/work/ai-orchestrator`

---

## Danh mục Task Thực thi (Sau khi User `APPROVE`)

- [x] **TASK-TGAP-01 — Bộ trích xuất Hồ sơ Duyệt Giải pháp S3 (`Nguyên nhân` + `Giải pháp & Code Demo` + `Những thay đổi: Sub-Tasks & Affected Files`) & Nâng cấp Bản tin `/task` + `⏸ [APPROVAL REQUIRED]`**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:680-829` (`handleTelegramTaskListCommand`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:917-963` (`notifyTelegramOnS3PlanReadyForce`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:2684-2725` (`inspectTaskRepoDiffSummary`)
  - **Mô tả**:
    1. Xây dựng hàm `extractTaskApprovalDossier(ctx, qt, isoMeta, subTasks)` trong `internal/webui/telegram_bridge.go` đọc thực tế từ `.orchestrator/tasks/<safeKey>/09_tasks_solution.md`, `13_analysis.md`, `08_tasks.md` và `diff.patch` / `git status` để trích xuất:
       - `RootCauseDigest` & `RootCauseFull` (từ mục `Root Cause` / `Vấn đề cốt lõi` / `Điểm Chạm VERIFIED file:line`).
       - `SolutionDigest`, `SolutionFull` & `CodeDemoPreview` (từ mục `Giải pháp Kỹ thuật` / `Code Demo` / `Co-Architect Steering & Solution Revision`).
       - `SubTaskLines` (đầy đủ `1..N` Sub-Tasks kèm `[SubTaskID] (Repo) Title — Context: file:line`).
       - `AffectedFileNames` (danh sách tên file cụ thể từ cả `CandidateFiles` của `subTasks`, `diff.patch` và `git status`, loại bỏ trùng lặp).
    2. Trong `handleTelegramTaskListCommand`:
       - Mở rộng trực tiếp dưới dòng `[⏸ CHỜ APPROVE (#N)]`: hiển thị đường dẫn file Plan, tóm tắt Nguyên nhân, tóm tắt Giải pháp, các Sub-Tasks chính và danh sách tên files bị ảnh hưởng.
       - Đổi nút bấm Inline Keyboard của Task `NEEDS_YOU` thành cặp nút **`[ 👁 Xem Plan & Duyệt: #ID ]`** (`VIEW_PLAN #<ID>`) và **`[ ✏️ Sửa Giải Pháp: #ID ]`** (`REVISE_PROMPT #<ID>`).
    3. Trong `notifyTelegramOnS3PlanReadyForce` và `handleTelegramViewPlanCommand` (`VIEW_PLAN` / `/plan`):
       - Gửi bản tin **`⏸ [APPROVAL REQUIRED — HỒ SƠ DUYỆT GIẢI PHÁP S3]`** chứa đầy đủ: (1) Nguyên nhân gốc rễ (`file:line`), (2) Giải pháp kỹ thuật & Code Demo, (3) Những thay đổi (`1..N` Sub-Tasks + danh sách đầy đủ các files bị ảnh hưởng + đường dẫn file vật lý), kèm tự động tách message nếu vượt 3,800 ký tự và gắn 3 nút Inline Keyboard: `[ ✅ Approve & Chạy S4–S5 Ngay ]`, `[ ✏️ Phản hồi / Sửa Giải Pháp ]`, `[ 📋 Mở Hàng Đợi Task (/task) ]`.
  - **Test / DoD**: Gọi `/task` và `VIEW_PLAN` cho task `tích hợp Master MongoDB Destination` (`10 Sub-Tasks`, `19 files`) trả về đầy đủ Nguyên nhân, Giải pháp, 10 Sub-Tasks và danh sách 19 files.

- [x] **TASK-TGAP-02 — Luồng Phản hồi / Chỉnh sửa Giải pháp S3 trực tiếp trên Telegram (`REVISE_PROMPT`, Reply tin nhắn, `/revise` -> Co-Architect `revise_solution` & `update_subtasks`) + Định danh mọi loại `TaskKey`**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:60-95` (`TelegramSettingsDTO`, `telegramPersistedConfig`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:439-678` (`extractTelegramMessageFields`, `handleTelegramWebhook`, `processIncomingTelegramCommand`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:843-972` (`revise_solution`, `update_subtasks`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1051-1087` (`findQueuedTask`)
  - **Mô tả**:
    1. Bổ sung trường `PendingRevisionTaskKey string` vào `telegramPersistedConfig` và `TelegramSettingsDTO`.
    2. Nâng cấp `findQueuedTask(ctx, taskKey)` và `resolveTelegramTaskFromText`: hỗ trợ tìm kiếm theo `#<ID>` (VD `#1`), `ExternalID` có nhiều dấu gạch ngang (`CDC-RECON-01`), `ExternalID` tiếng Việt có khoảng trắng (`tích hợp Master MongoDB Destination`), hoặc `sanitizeTaskKeyPath(ExternalID)`. Dùng mã ngắn `#<ID>` trong `callback_data` khi `ExternalID` dài hoặc có dấu tiếng Việt để không bao giờ vượt giới hạn 64 bytes của Telegram.
    3. Hỗ trợ 3 cách phản hồi sửa giải pháp trên Telegram:
       - **Cách 1 (Nút bấm `[ ✏️ Phản hồi / Sửa Giải Pháp ]` — `REVISE_PROMPT #<ID>`):** Lưu `cfg.PendingRevisionTaskKey = qt.ExternalID`, gửi tin nhắn hướng dẫn kèm `force_reply: true`. Tin nhắn văn bản tiếp theo của User tự động được định tuyến vào `handleTelegramReviseSolutionCommand`.
       - **Cách 2 (Reply trực tiếp vào tin nhắn Plan của Bot trên Telegram):** Khi `replyContext` chứa tên Task / `#<ID>` và nội dung tin nhắn của User không phải lệnh `APPROVE`, tự động định tuyến vào `handleTelegramReviseSolutionCommand`.
       - **Cách 3 (Lệnh `/revise <Task> <nội dung>`, `/feedback <Task> <nội dung>`, `SỬA <Task>: <nội dung>`):** Định tuyến trực tiếp vào `handleTelegramReviseSolutionCommand`.
    4. Trong `handleTelegramReviseSolutionCommand`:
       - Cập nhật trực tiếp chỉ đạo của User vào `.orchestrator/tasks/<safeKey>/09_tasks_solution.md` (và thêm/cập nhật Sub-Task trong `08_tasks.md` + `subtasks.json` nếu nội dung phản hồi yêu cầu thêm/tách Sub-Task).
       - Ghi lịch sử đối thoại vào `chat_history.json` của Co-Architect Studio trên WebUI, thu hồi `.user-approved` cũ, giữ Task ở `NEEDS_YOU`, xóa `PendingRevisionTaskKey`, và gửi ngay bản tin **`⏸ [APPROVAL REQUIRED — ĐÃ CẬP NHẬT THEO PHẢN HỒI CỦA BẠN]`** mới nhất lên Telegram kèm nút `[ ✅ Approve & Chạy S4–S5 Ngay ]`.
  - **Test / DoD**: Bấm `[ ✏️ Phản hồi / Sửa Giải Pháp ]` rồi nhắn nội dung sửa giải pháp -> `09_tasks_solution.md` và `chat_history.json` cập nhật ngay trên đĩa, Bot gửi lại bản Plan mới trên Telegram.

- [x] **TASK-TGAP-03 — Kiểm thử Tự động & Xác minh Runtime Thật trên `:8090` (`EV-...`) + Kiểm định `/laya_router` `DG3`**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server_test.go`
  - **Mô tả**:
    1. Viết test `TestTelegramApprovalDossier_FullRootCauseSolutionChanges_AndInteractiveFeedbackRevision` kiểm chứng toàn trình:
       - `/task` hiển thị đầy đủ Nguyên nhân, Giải pháp, Sub-Tasks, 19 files cho `tích hợp Master MongoDB Destination`.
       - `VIEW_PLAN` bắn đầy đủ hồ sơ duyệt S3.
       - `REVISE_PROMPT` + tin nhắn phản hồi tiếp theo cập nhật `09_tasks_solution.md` và gửi lại bản Plan mới.
       - `APPROVE` trên bản Plan mới kích hoạt chạy `S4 -> S5` thành công.
    2. Build lại `./bin/orchestrator`, khởi động lại `:8090`, kiểm chứng runtime thật qua `evidence_run.py` (`EV-...`) và chạy `/laya_router` `DG3`.
  - **Test / DoD**: Pass 100% `go vet ./...`, `go test -v -race ./internal/webui/...`, runtime `:8090` (`EV-...`), và `laya_gate.py --stage dg3` (`PASS`).
