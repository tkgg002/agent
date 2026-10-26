# 13. Root-Cause Analysis — Lỗi Luồng Thực Thi Task 10 Sub-Tasks (Thiếu Duyệt Từng Sub-Task vs Approve All) & Lỗi UI/UX `Hàng Đợi Task (Theo Repo)` (`fix_repo_task_queue_sidebar_ui_ux`)

## 1. Truy vết Nguyên nhân Gốc rễ (`VERIFIED` kèm `file:line`)

### 1.1. RC1 (`VERIFIED`) — Chạy gộp mù quáng toàn bộ 10 Sub-Tasks khi bấm `APPROVE` ở Task cha, thiếu 2 chế độ (`Approve All` vs `Approve Từng Sub-Task`) và thiếu thông báo Review từng Sub-Task qua Telegram
- **Bằng chứng `file:line`:**
  - `internal/webui/server.go:1393-1408` (`case "approve":`):
    ```go
    subTasks := s.loadOrGenerateSubTasks(*qt)
    safeParent := sanitizeTaskKeyPath(qt.ExternalID)
    for i := range subTasks {
        safeSub := sanitizeTaskKeyPath(subTasks[i].SubTaskID)
        subDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSub)
        _ = os.MkdirAll(subDir, 0o755)
        _ = os.WriteFile(filepath.Join(subDir, ".user-approved"), nowStamp, 0o644)
    ```
  - `internal/webui/server.go:1459-1495` (`case "implement":`):
    Khi bước vào `implement`, hệ thống gọi `s.synthesizeIsolatedRemediationDiffIfApproved(ctx, *qt, isoMeta, subTasks)` cho **toàn bộ `subTasks` cùng lúc**, đánh dấu cả 10 Sub-Tasks thành `DONE` trong 1 lượt rồi chạy thẳng sang `case "qa":` (`Parent S5 -> DONE`).
  - `internal/webui/telegram_bridge.go:1146-1166`:
    Trong `buildTelegramApprovalRequiredMessage`, bàn phím Telegram chỉ có duy nhất 1 nút `✅ Approve & Chạy S4–S5` (`callback_data: "APPROVE #<ID>"`), hoàn toàn **không cho User lựa chọn giữa 2 chức năng**:
    1. **`⚡ Approve All (Chạy tự động hết 10/10 Sub-Tasks theo Wave -> S5)`**
    2. **`🧩 Approve Từng Sub-Task (Review & Approve từng Sub-Task 1/10 -> 10/10 theo Wave)`**
  - Hậu quả: Với một Task lớn có 10 Sub-Tasks (như `tích hợp Master MongoDB Destination` có `TASK-MONG01..10` chia làm 3 Wave), các Sub-Tasks không hề vận hành như những Task thực thụ có cổng Review/Approve riêng trên Telegram, và Task cha nhảy thẳng qua các bước sau mà không đợi duyệt từng Sub-Task.

### 1.2. RC2 (`VERIFIED`) — Cột Sidebar quá hẹp (`248px`) + Nhồi tiêu đề dài và 3 nút thiếu `white-space: nowrap` vào `.vibe-col-hdr`
- **Bằng chứng `file:line`:**
  - `internal/webui/static/app.css:340`: `.ide-workspace-body` đặt `grid-template-columns: 248px 1fr;`.
  - `internal/webui/static/index.html:131-138`: Nhét `<span>Hàng Đợi Task (Theo Repo)</span>` + 3 nút (`✈️ /task`, `🧹 Dọn DONE`, `+ Task`) vào `.vibe-col-hdr` không có `white-space: nowrap`, khiến chữ `Hàng Đợi Task (Theo Repo)` và `✈️ /task` bị gãy thành nhiều dòng chồng chéo.

### 1.3. RC3 (`VERIFIED`) — `.sb-repo-group` có `overflow: hidden` nhưng thiếu `flex-shrink: 0` trong `.sidebar-task-list` (`flex-direction: column; overflow-y: auto`)
- **Bằng chứng `file:line`:**
  - `internal/webui/static/app.css:395-402` và `app.css:3598-3604`: `.sb-repo-group` đặt `overflow: hidden` mà không có `flex-shrink: 0`, khiến CSS Flexbox co rút chiều cao của từng nhóm Repo thay vì cuộn dọc, cắt cụt các thẻ Task bên trong.

### 1.4. RC4 (`VERIFIED`) — `.sb-task-top` nhồi chung `n.key` dài + `stageBadgeHtml` + `actionButtonsHtml`, in lặp `n.title === n.key`, và `sendTelegramTaskMenuNow()` thiếu phản hồi trên nút `✈️ /task`
- **Bằng chứng `file:line`:**
  - `internal/webui/static/app.js:890-896` & `app.js:3369-3392`.
