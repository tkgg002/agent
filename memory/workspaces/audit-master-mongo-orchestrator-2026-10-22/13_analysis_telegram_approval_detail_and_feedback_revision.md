# 13. Analysis & Root Cause Investigation: Vì Sao Thông Báo `⏸ CHỜ APPROVE` Trên Telegram Bị Cụt Lủn 1 Dòng & Thiếu Luồng Phản Hồi Sửa Giải Pháp (`telegram_approval_detail_and_feedback_revision`)

> **Workspace**: `audit-master-mongo-orchestrator-2026-10-22`
> **Tier**: `T2`
> **Target Repository**: `/Users/trainguyen/Documents/work/ai-orchestrator`

---

## 1. Phân Tích Root Cause Thực Chứng (`VERIFIED file:line`)

### 1.1. `F-TGAP-01`: `handleTelegramTaskListCommand` dùng chung template 1 dòng cụt lủn cho mọi trạng thái và gắn tất cả nút bấm vào `RUN_TASK` (`internal/webui/telegram_bridge.go:734-825`)
- **Bằng chứng `VERIFIED` (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:748-824`):**
  - Tại dòng `813-821`, hàm `handleTelegramTaskListCommand` in mọi Task (cả `DONE`, `READY`, lẫn `NEEDS_YOU`) bằng đúng 1 dòng:
    ```go
    sb.WriteString(fmt.Sprintf("• *%s*%s [%s] — %s (`%d/%d Sub-Tasks`%s)\n",
        qt.ExternalID, activeMark, statusBadge, truncateRunes(qt.Title, 52), doneSubs, len(subTasks), diffTag,
    ))
    ```
  - Tại dòng `748-756`, toàn bộ nút bấm trong Inline Keyboard đều gán cứng `cbData := fmt.Sprintf("RUN_TASK %s", qt.ExternalID)`, đi kèm câu nhắc ở dòng `824`: `"👇 *Bấm trực tiếp vào nút Task bên dưới để kích hoạt chạy ngay trên Orchestrator:*"`.
  - **Hậu quả:** Với task `tích hợp Master MongoDB Destination` đang ở trạng thái `⏸ CHỜ APPROVE (#1)`, User chỉ thấy `0/10 Sub-Tasks · 📝 19 files diff` mà không hề thấy **Nguyên nhân (`Root Cause`)**, **Giải pháp (`Solution`)**, hay **Những thay đổi (`10 Sub-Tasks` làm gì, `19 files` là những file nào, đường dẫn file Plan ở đâu)**.

### 1.2. `F-TGAP-02`: `notifyTelegramOnS3PlanReadyForce` chỉ gửi 6 dòng metadata rỗng, không trích xuất nội dung từ `09_tasks_solution.md`, `13_analysis.md`, `08_tasks.md` (`internal/webui/telegram_bridge.go:922-963`)
- **Bằng chứng `VERIFIED` (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:938-957`):**
  - Khi Task chuyển sang `NEEDS_YOU` (`S3 Solution Ready` hoặc `S5 -> S3 Re-Plan`), hàm `notifyTelegramOnS3PlanReadyForce` chỉ gửi 6 dòng thông tin bề mặt (`Trạng thái`, `Task`, `Affected Repos`, `Sub-Tasks: 0/10 done`, `File Giải pháp`) mà hoàn toàn không đọc và hiển thị:
    1. **Nguyên nhân gốc rễ (`Root Cause` & `file:line`)** đã có sẵn trong `.orchestrator/tasks/<safeKey>/09_tasks_solution.md` (`L10-26`) và `13_analysis.md`.
    2. **Giải pháp kỹ thuật & Code Demo (`Solution & Code Preview`)** đã có sẵn trong `.orchestrator/tasks/<safeKey>/09_tasks_solution.md` (`L29-120`).
    3. **Những thay đổi (`Changes`)**: Danh sách chi tiết `1..N` Sub-Tasks từ `08_tasks.md` (`L8-78`) và danh sách tên các files bị ảnh hưởng/thay đổi.

### 1.3. `F-TGAP-03`: Thiếu hoàn toàn luồng Phản hồi / Sửa Giải Pháp (`Revise Solution`) trên Telegram và lỗi định danh `TaskKey` có khoảng trắng / nhiều dấu gạch ngang (`internal/webui/telegram_bridge.go:564, 589-678, 832-847`)
- **Bằng chứng `VERIFIED` (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:564-615`):**
  - Tại dòng `600-615`, `processIncomingTelegramCommand` chỉ kiểm tra `/task`, `RUN_TASK` hoặc `isApprove`. Nếu User không đồng ý Approve và nhắn tin phản hồi để yêu cầu sửa giải pháp, tin nhắn bị đánh dấu `"IGNORED — not a /task, RUN_TASK, or APPROVE command"` (`L613-614`) và trả lỗi HTTP 400!
  - Trong khi đó, bộ máy Co-Architect `revise_solution` và `update_subtasks` đã tồn tại hoàn chỉnh trong `internal/webui/task_chat.go:843-972`, nhưng chưa được kết nối với `telegram_bridge.go`.
  - Ngoài ra, regex `telegramTaskKeyRe` tại dòng `564`:
    ```go
    var telegramTaskKeyRe = regexp.MustCompile(`(?i)\b(TASK-[A-Z0-9_-]+|[A-Z]{2,10}-\d+)\b`)
    ```
    chỉ bắt được `TASK-...` hoặc `ABC-123` (1 dấu gạch ngang), nên:
    - Với `CDC-RECON-01` (2 dấu gạch ngang), regex chỉ bắt được `RECON-01` (mất tiền tố `CDC-`).
    - Với `tích hợp Master MongoDB Destination` (tiếng Việt có khoảng trắng), regex không khớp được gì cả, khiến nút bấm `RUN_TASK` hoặc lệnh `APPROVE` cho task này bị rỗng `targetKey`!

### 1.4. `F-TGAP-04`: `inspectTaskRepoDiffSummary` chỉ trả về số lượng `19 files diff` mà đánh rơi danh sách tên file cụ thể (`internal/webui/server.go:2684-2725`)
- **Bằng chứng `VERIFIED` (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:2684-2725`):**
  - Hàm `inspectTaskRepoDiffSummary` thu thập map `seenFiles` từ `diff.patch` và `git status --short`, nhưng chỉ trả về `(cnt int, summary string)` mà không trả về danh sách các tên file cụ thể (`[]string`), đồng thời chưa gộp các file mục tiêu (`CandidateFiles` trong `08_tasks.md` / `subTasks`) để hiển thị cho User biết chính xác những file nào sẽ bị thay đổi.

---

## 2. Bảng Phân Loại Bằng Chứng (`VERIFIED` vs `HYPOTHESIS`)

| Mã | Phân loại | Tham chiếu (`REF:`) | Nội dung |
| :--- | :---: | :--- | :--- |
| `E1` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:734-825` | `handleTelegramTaskListCommand` in 1 dòng rút gọn cho cả Task `⏸ CHỜ APPROVE` và gán mọi nút bấm vào `RUN_TASK`. |
| `E2` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:922-963` | `notifyTelegramOnS3PlanReadyForce` không trích xuất Nguyên nhân, Giải pháp, Code Demo, danh sách Sub-Tasks và danh sách Files bị ảnh hưởng. |
| `E3` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:564-615` | `processIncomingTelegramCommand` từ chối mọi tin nhắn phản hồi sửa giải pháp; `telegramTaskKeyRe` lỗi với `CDC-RECON-01` và `tích hợp Master MongoDB Destination`. |
| `E4` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:843-972` | Logic Co-Architect `revise_solution` & `update_subtasks` đã có sẵn trong `task_chat.go`, sẵn sàng tái sử dụng cho Telegram. |
| `E5` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:2684-2725` | `inspectTaskRepoDiffSummary` thu thập `seenFiles` nhưng chưa trả về danh sách tên file cụ thể. |
