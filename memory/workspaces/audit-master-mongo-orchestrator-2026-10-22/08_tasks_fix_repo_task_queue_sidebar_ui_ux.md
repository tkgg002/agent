# 08. Tasks — Luồng Thực Thi Fractal `N` Sub-Tasks (`Approve All` vs `Approve Từng Sub-Task`) & Khắc phục UI/UX `Hàng Đợi Task (Theo Repo)` (`fix_repo_task_queue_sidebar_ui_ux`)

## Danh sách Hạng mục Thực thi (Sau khi User APPROVE)

### `TASK-UIQ-01` — Rào chắn Cứng `Parent S4 -> S5 Wave Barrier` & Thực thi Từng Sub-Task (`step_subtasks` / `approve_subtask`) trong `server.go` & `task_inspector.go`
- **Mô tả & Context `file:line`:**
  - `internal/webui/server.go:1184-1255, 1378-1505, 1647-1690`:
    - Tách bạch 2 chế độ duyệt ở cấp Parent Task:
      - `approve_all` / `approve_and_run`: Duyệt Task cha, thực thi tuần tự toàn bộ `1..N` Sub-Tasks theo thứ tự Wave (`Wave 0 -> Wave 1 -> Wave 2`), và sau khi đủ `N/N` Sub-Tasks đạt `DONE` thì chạy tiếp các bước Wave sau của Task cha (`Parent S5 QA -> Laya DG3 -> S6 DONE`).
      - `step_subtasks`: Duyệt thiết kế S3 của Task cha để bước vào pha **Thực thi Từng Sub-Task (`S4_SUBTASKS`)** (ghi marker `.step-by-step-subtasks`, KHÔNG tự động đóng dấu `.user-approved` lên toàn bộ 10 Sub-Tasks), chuẩn bị Sub-Task đầu tiên chưa xong (`[1/N]`, `Stage = NEEDS_YOU`) và đẩy Thẻ Review Sub-Task `[1/N]` lên Telegram chờ User duyệt.
    - Bổ sung hàm `executeSingleSubTaskAndAdvanceOrFinishParent(ctx, qt, subTaskID, fromUser, stepDelay)`:
      - Khi User duyệt Sub-Task `k/N` (`APPROVE_SUB #<ID> <SubTaskID>`), hệ thống tự động đảm bảo các Sub-Task cùng hoặc trước đó nếu đã được duyệt thì hoàn tất, thực thi `subtask_approve -> subtask_implement -> subtask_qa` cho đúng Sub-Task `k/N`, hợp nhất diff của Sub-Task `k/N` vào `diff.patch`.
      - **Rào chắn `Parent S4 -> S5 Barrier`:** Kiểm tra số lượng Sub-Tasks `DONE` (`doneCnt / totalSubTasks`):
        - Nếu `doneCnt < totalSubTasks`: Giữ Task cha ở bước Implement (`S4_SUBTASKS`, chưa chạy `Parent S5 QA`), chuyển Sub-Task kế tiếp `(k+1)/N` sang `NEEDS_YOU` và bắn Thẻ Review Sub-Task `(k+1)/N` lên Telegram.
        - Nếu `doneCnt == totalSubTasks` (`10/10 DONE`): Xóa marker `.step-by-step-subtasks` và **tự động chạy tiếp các bước Wave sau của Task cha (`Parent S4 Finalize -> Parent S5 Adversarial Audit -> Laya DG3 -> S6 DONE`)** rồi gửi thông báo hoàn tất toàn trình lên Telegram!
- **Logs / Test:**
  - Unit & Integration test kiểm chứng Task 10 Sub-Tasks chạy ở cả 2 chế độ (`STEP_SUBTASKS -> APPROVE_SUB 1..10 -> Parent S5 DONE` và `APPROVE_ALL -> 10/10 Wave 0..2 -> Parent S5 DONE`).
- **Definition of Done (DoD):**
  - Tuyệt đối không chạy `Parent S5 -> DONE` khi chưa xong đủ `N/N` (`10/10`) Sub-Tasks.

### `TASK-UIQ-02` — Hệ thống Thông báo & Điều khiển Sub-Task Tinh Tế trên Telegram (`APPROVE_ALL`, `STEP_SUBTASKS`, `APPROVE_SUB`, `REVISE_SUB_PROMPT`, `LIST_SUBTASKS`) trong `telegram_bridge.go`
- **Mô tả & Context `file:line`:**
  - `internal/webui/telegram_bridge.go:592-867, 1082-1495`:
    - Nâng cấp `buildTelegramApprovalRequiredMessage` và `/task`: Hiển thị phân bổ Sub-Tasks theo từng Wave (`Wave 0`, `Wave 1`, `Wave 2`) và cung cấp **2 nút duyệt chế độ Implement**:
      1. `[ ⚡ Approve All (Chạy 10/10 Sub-Tasks -> S5) ]` (`APPROVE_ALL #<ID>`)
      2. `[ 🧩 Duyệt Từng Sub-Task (1/10 -> 10/10) ]` (`STEP_SUBTASKS #<ID>`)
      cùng với nút `[ ✏️ Phản hồi / Sửa Giải Pháp ]` (`REVISE_PROMPT #<ID>`).
    - Xây dựng hàm `buildTelegramSubTaskReviewMessage(ctx, qt, isoMeta, subTasks, currentIdx, reasonLabel)`:
      - Tạo bản tin Telegram cực kỳ tinh tế cho Sub-Task `[k/N]` (`Wave W`): thanh tiến độ `[■■■□□□□□□□] k/N`, thông tin Wave, Repo, Role & Model, Điểm chạm `file:line`, Nguyên nhân & Giải pháp của Sub-Task `k`, `Code Demo` của Sub-Task `k`, và `TestDoD`.
      - Gắn bàn phím Inline Keyboard tinh tế cho Sub-Task `k`:
        - `[ ✅ Approve Sub-Task k/N (<SubTaskID>) & Chạy ]` (`APPROVE_SUB #<ID> <SubTaskID>`)
        - `[ ✏️ Sửa Plan Sub-Task k/N ]` (`REVISE_SUB_PROMPT #<ID> <SubTaskID>`)
        - `[ ⚡ Approve All Còn Lại (Chạy hết -> S5) ]` (`APPROVE_ALL #<ID>`)
        - `[ 📋 Tiến độ 10 Sub-Tasks (#<ID>) ]` (`LIST_SUBTASKS #<ID>`)
    - Hỗ trợ chỉnh sửa riêng Plan của từng Sub-Task (`REVISE_SUB_PROMPT #<ID> <SubTaskID>` hoặc Reply vào tin nhắn Sub-Task) cập nhật trực tiếp vào `subtasks/<SubTaskID>/09_tasks_solution.md` và gửi lại Thẻ Review của Sub-Task đó.
- **Logs / Test:**
  - Test `TestTelegramFractalSubTaskFlow_StepByStepAndApproveAll` trong `internal/webui/server_test.go`.
- **Definition of Done (DoD):**
  - Mọi tương tác duyệt từng Sub-Task (`1/10 -> 10/10`), sửa Sub-Task, hoặc `Approve All` trên Telegram hoạt động mượt mà, thông báo tinh tế, rõ ràng.

### `TASK-UIQ-03` — Khắc phục lỗi UI/UX khu vực `Hàng Đợi Task (Theo Repo)`, nút `✈️ /task` và Bổ sung Nút `⚡ Approve All` / `🧩 Từng Sub-Task` trên WebUI (`index.html`, `app.css`, `app.js`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/static/index.html:130-151`, `internal/webui/static/app.css:336-468, 3494-3665`, `internal/webui/static/app.js:721-925, 3369-3392`:
    - Mở rộng Sidebar trái lên `310px`, chống gãy dòng tiêu đề `Hàng Đợi Task (Theo Repo)` và nút `✈️ /task` (`.mini-tg-btn` có phản hồi trạng thái `⏳ Đang gửi... -> ✓ Đã đẩy Tele`).
    - Thêm `flex-shrink: 0` cho `.sb-repo-group` và `.sb-task-item` (bỏ `overflow: hidden` trên `.sb-repo-group`) để chấm dứt lỗi Flexbox bóp chiều cao danh sách Task.
    - Hợp nhất thanh Filter + Counter (`Tất cả`, `⚡ Chạy`, `⏸ Duyệt`, `✓ Xong`).
    - Trên thẻ Task ở trạng thái chờ duyệt (`NEEDS_YOU`) hoặc đang duyệt từng Sub-Task (`S4_SUBTASKS`), hiển thị rõ tiến độ `🧩 done/total Sub-Tasks` và hỗ trợ cả 2 nút hành động: **`⚡ Duyệt All`** (`approve_and_run`) và **`🧩 Từng Sub`** (`step_subtasks`).
- **Logs / Test:**
  - Kiểm thử trực tiếp trên `:8090` và `go test -v -race ./internal/webui/...`.
- **Definition of Done (DoD):**
  - Giao diện `Hàng Đợi Task (Theo Repo)` và nút `✈️ /task` hiển thị chuẩn xác, không tràn, không gãy chữ, đồng bộ 100% với luồng 10 Sub-Tasks trên Telegram.

## Trạng Thái Hoàn Tất (S4–S6 Verification)
- `[x]` **`TASK-UIQ-01`: DONE** — Đã triển khai `step_subtasks`, `approve_all`, `executeAllSubTasksWaveByWave`, rào chắn `hasApproval` (`.step-by-step-subtasks`) và `beginTrackExecution` (`EV-20261026T150807-d898`, `EV-20261026T150810-1109`).
- `[x]` **`TASK-UIQ-02`: DONE** — Đã triển khai `buildTelegramSubTaskReviewMessage`, `handleTelegramStepSubTasksCommand`, `handleTelegramApproveSubTaskCommand`, `handleTelegramListSubTasksCommand`, `handleTelegramReviseSubPromptCommand`, `handleTelegramReviseSubTaskSolutionCommand`, và `sendTelegramWaveExecutionLedger` (`EV-20261026T150451-1517`, `EV-20261026T150736-83a8`).
- `[x]` **`TASK-UIQ-03`: DONE** — Đã mở rộng Sidebar `310px`, sửa `.vibe-col-hdr` & nút `✈️ /task` (`.mini-tg-btn`), `#sidebar-queue-filter-bar`, `.sb-repo-group` (`flex-shrink: 0`), và bổ sung nút `⚡ All` / `🧩 Từng Sub` (`EV-20261026T150736-83a8`).

