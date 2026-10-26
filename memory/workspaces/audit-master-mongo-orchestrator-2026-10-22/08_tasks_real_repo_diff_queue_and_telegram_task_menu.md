# 08 Tasks — Thực thi Code Diff Thật trên Repo Đích (`centralized-export-service`), Quản lý "Hàng Đợi Task (Theo Repo)" & Lệnh `/task` Chọn Chạy Task trên Telegram

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `real_repo_diff_queue_and_telegram_task_menu`
- **Tier:** `T2`

## Danh sách Sub-Tasks (Autonomous Full-Stack Prompting — Rule #2)

### 1. [x] `TASK-RDQ-01`: Thực thi Code Diff Thật trên Repo Đích `/Users/trainguyen/Documents/work/centralized-export-service` (Chuyển `trans_his` từ MongoDB sang PostgreSQL cho `TASK-13`) & Đồng bộ `VerifySubTaskCodeOnDisk`
- **Mô tả & Context `file:line`:**
  - Khảo sát và cập nhật các file `trans_his` trong `/Users/trainguyen/Documents/work/centralized-export-service` (chuyển truy vấn `trans_his` từ MongoDB sang PostgreSQL) và cập nhật cơ chế thực thi `S4 Implement` trong [`internal/webui/telegram_bridge.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go) & [`internal/webui/server.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go) để áp dụng thay đổi thật trên `TargetRepo` (không dùng hunk comment giả khi chạy trên repo thật).
- **Logs / Test:**
  - `git -C /Users/trainguyen/Documents/work/centralized-export-service status --short && git -C /Users/trainguyen/Documents/work/centralized-export-service diff --stat` (`EV-20261023T225300-dc57`, `EV-20261023T225015-3d87`: 23/23 vitest tests passed)
- **Definition of Done (DoD):**
  - Repo `/Users/trainguyen/Documents/work/centralized-export-service` có `git diff` thật sự chuyển đổi `trans_his` từ Mongo sang Postgres (`6 files changed, 437 insertions(+), 78 deletions(-)`), khớp với `diff.patch` của `TASK-13` trên UI `:8090` (`EV-20261023T225516-e660`).

### 2. [x] `TASK-RDQ-02`: Nâng cấp Quản lý "Hàng Đợi Task (Theo Repo)" trên Giao diện WebUI (`:8090`)
- **Mô tả & Context `file:line`:**
  - [`internal/webui/server.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go), [`internal/webui/static/index.html`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html), [`internal/webui/static/app.js`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js), [`internal/webui/static/app.css`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css).
  - Bổ sung thông tin trạng thái thực thi hàng đợi theo từng Repo (`⚡ ĐANG CHẠY`, `⏸ CHỜ APPROVE`, `⏳ HÀNG ĐỢI #N`, `✓ ĐÃ XONG`), thống kê `git diff` thật trên repo đích (`RepoDiffSummary`), bộ lọc trạng thái hàng đợi, và nút bấm **▶ Chạy / ✓ Duyệt & Chạy / ⏹ Dừng** trực tiếp ngay trên từng item của **Hàng Đợi Task (Theo Repo)**.
- **Logs / Test:**
  - `go test -v -race ./internal/webui/...` (`EV-20261023T225035-4495`)
- **Definition of Done (DoD):**
  - Khung **Hàng Đợi Task (Theo Repo)** hiển thị rõ ràng task nào đang chạy, task nào đang chờ, vị trí trong hàng đợi, số file/dòng `git diff` thật trên repo và cho phép bấm Chạy/Dừng trực tiếp.

### 3. [x] `TASK-RDQ-03`: Bổ sung sự kiện `/task` trên Telegram liệt kê danh sách Task kèm Inline Keyboard & Click vào Task nào thì tự động chạy Task đó
- **Mô tả & Context `file:line`:**
  - [`internal/webui/telegram_bridge.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go) (`processIncomingTelegramCommand`, `handleTelegramTaskListCommand`, `handleTelegramRunTaskCommand`, `ensureTelegramApprovePoller`, `handleTelegramWebhook`).
  - Khi nhận lệnh `/task` (hoặc `/tasks`, `/list`) từ Telegram, gửi bản tin tổng hợp Hàng Đợi Task hiện tại (gom theo Repo, hiển thị trạng thái `⚡ ĐANG CHẠY`, `⏸ CHỜ APPROVE`, `⏳ SẴN SÀNG`, `✓ DONE`) kèm các nút bấm Inline Keyboard (`callback_data: "RUN_TASK <TaskKey>"`).
  - Khi User click vào nút của một Task trên Telegram, tự động phản hồi `answerCallbackQuery`, gửi thông báo `🚀 [AI-ORCHESTRATOR · KÍCH HOẠT TASK TỪ MENU /task]` và thực thi ngay task đó (`plan` hoặc `approve_and_run`).
- **Logs / Test:**
  - `go test -v -race ./internal/webui/...` (`EV-20261023T225035-4495`) & Live `:8090` (`EV-20261023T225516-e660`)
- **Definition of Done (DoD):**
  - Gõ `/task` trên Telegram trả về danh sách task kèm nút bấm; click vào task nào thì task đó tự động chạy và cập nhật trạng thái thực tế.
