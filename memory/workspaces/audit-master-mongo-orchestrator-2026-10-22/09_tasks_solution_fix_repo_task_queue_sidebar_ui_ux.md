# 09. Technical Solution & Code Demo — Luồng Thực Thi Fractal `N` Sub-Tasks (`Approve All` vs `Approve Từng Sub-Task` qua Telegram/WebUI) & Khắc phục UI/UX `Hàng Đợi Task (Theo Repo)` (`fix_repo_task_queue_sidebar_ui_ux`)

## 1. Phương án Kỹ thuật Tối ưu Duy nhất (Single Best Solution)

### 1.1. Kiến trúc Vận hành Fractal `1 Task -> 10 Sub-Tasks` theo Wave (`Approve All` vs `Approve Từng Sub-Task`)
Khi một Task cha (`Parent Task`, ví dụ `tích hợp Master MongoDB Destination` có `10 Sub-Tasks` chia theo `Wave 0`, `Wave 1`, `Wave 2`) hoàn tất `S3 Plan` và bước vào giai đoạn **Implement (`S4`)**, hệ thống vận hành theo đúng nguyên lý **Fractal Task & Wave Barrier**:

1. **Mỗi Sub-Task (`1/10 .. 10/10`) là một Task độc lập thu nhỏ:**
   - Có hồ sơ riêng tại `.orchestrator/tasks/<TaskKey>/subtasks/<SubTaskID>/` (`01_requirements.md`, `10_preplan_critique.md`, `09_tasks_solution.md`, `diff.patch`, `audit_report.md`).
   - Có chỉ số `WaveIndex` (`Wave 0`, `Wave 1`, `Wave 2`) và phụ thuộc `DependsOn` (Wave sau chỉ mở khi Wave trước đã `DONE`).
   - Đi qua đúng các chặng: `Plan (S3) -> Human Review/Approve -> Implement (S4) -> 5A/5B QA & Laya DG3 (S5) -> DONE`.

2. **Hai Chức năng Duyệt & Thực thi tại Bước Implement (trên Telegram & WebUI):**
   - **Chức năng 1 — `⚡ Approve All (Chạy hết 10/10 Sub-Tasks theo Wave -> S5)` (`APPROVE_ALL #<ID>`):**
     - Duyệt giải pháp S3 của Task cha và duyệt tự động toàn bộ `10/10` Sub-Tasks.
     - Hệ thống chạy tuần tự từng Sub-Task theo đúng thứ tự Wave (`Wave 0 -> Wave 1 -> Wave 2`), gửi bảng tổng hợp tiến độ thực thi từng Wave/Sub-Task rất tinh tế qua Telegram (`⚡ [TIẾN ĐỘ THỰC THI 10/10 SUB-TASKS THEO WAVE]`), và **sau khi xong đủ `10/10` Sub-Tasks** thì tự động chạy tiếp các bước Wave sau của Task cha (`Parent S5 Adversarial Audit -> Laya DG3 -> S6 DONE`).
   - **Chức năng 2 — `🧩 Approve Từng Sub-Task (Step-by-Step 1/10 -> 10/10)` (`STEP_SUBTASKS #<ID>` -> `APPROVE_SUB #<ID> <SubTaskID>`):**
     - Duyệt kiến trúc S3 của Task cha để bước vào pha **`S4_SUBTASKS` (Duyệt & Chạy Từng Sub-Task)** (ghi marker `.step-by-step-subtasks` trong thư mục Task cha, không đóng dấu `.user-approved` hàng loạt lên cả 10 Sub-Tasks).
     - Hệ thống lấy Sub-Task đầu tiên chưa hoàn thành (`[1/10]` thuộc `Wave 0`, ví dụ `TASK-MONG01`), chuyển trạng thái sang `NEEDS_YOU`, và gửi ngay **Thẻ Review Sub-Task Tinh Tế (`🧩 [SUB-TASK REVIEW · BƯỚC 1/10 · WAVE 0]`)** lên Telegram gồm:
       - Thanh tiến độ trực quan: `[■□□□□□□□□□] 0/10 DONE (Đang duyệt bước 1/10 · Wave 0)`
       - Thông tin định danh: `SubTaskID`, `Wave`, `TargetRepo`, `Role & Model`, `Điểm chạm file:line`, đường dẫn `subtasks/<SubTaskID>/09_tasks_solution.md`
       - **Nguyên nhân & Điểm chạm của Sub-Task `k/10`**
       - **Giải pháp kỹ thuật & Code Demo của riêng Sub-Task `k/10`**
       - **Tiêu chuẩn Nghiệm thu (`TestDoD`)**
       - **Bàn phím Inline Keyboard tinh tế riêng cho Sub-Task `k/10`:**
         1. `[ ✅ Approve Sub-Task 1/10 (TASK-MONG01) & Chạy ]` (`APPROVE_SUB #1 TASK-MONG01`)
         2. `[ ✏️ Phản hồi / Sửa Plan Sub-Task 1/10 ]` (`REVISE_SUB_PROMPT #1 TASK-MONG01`)
         3. `[ ⚡ Approve All Còn Lại (Chạy hết -> S5) ]` (`APPROVE_ALL #1`)
         4. `[ 📋 Xem Bảng 10 Sub-Tasks (#1) ]` (`LIST_SUBTASKS #1`)
     - Khi User bấm **`[ ✅ Approve Sub-Task k/10 ]`**:
       - Hệ thống thực thi `subtask_approve -> subtask_implement -> subtask_qa` cho đúng Sub-Task `k/10`, hợp nhất diff hunk của Sub-Task `k/10` vào `diff.patch`, và đánh dấu Sub-Task `k/10 = DONE`.
       - **Nếu `doneCount < 10` (ví dụ mới xong `1/10`):** Giữ Task cha ở bước Implement (`S4_SUBTASKS`), gửi thông báo xác nhận `✅ Sub-Task [1/10] TASK-MONG01 hoàn tất (5A/5B QA PASS)` kèm **Thẻ Review của Sub-Task kế tiếp `[2/10] TASK-MONG02`** ngay lập tức để User bấm duyệt tiếp!
       - **Khi Sub-Task thứ 10 (`10/10`) hoàn tất (`doneCount == 10/10`):** Xóa marker `.step-by-step-subtasks`, mở khóa **Rào chắn `Parent S4 -> S5 Wave Barrier`** và tự động chạy tiếp các bước Wave sau của Task cha (`Parent S4 Final Diff Merge -> Parent S5 Adversarial Audit -> Laya DG3 -> S6 DONE`), sau đó bắn bản tin `✅ [S4–S5 COMPLETE · 10/10 SUB-TASKS & PARENT AUDIT PASS]` lên Telegram!

### 1.2. Khắc phục triệt để UI/UX `Hàng Đợi Task (Theo Repo)` và nút `✈️ /task` trên Sidebar WebUI (`:8090`)
- Mở rộng cột Sidebar trái từ `248px` lên `310px` (`grid-template-columns: 310px 1fr;`), tăng `.task-queue-section` lên `flex: 1.55`.
- Tách tiêu đề `.vibe-col-title` và `.vibe-col-actions` (`mini-tg-btn` màu xanh Telegram cho `✈️ /task`, có hiệu ứng phản hồi trực tiếp `⏳ Đang gửi... -> ✓ Đã đẩy Tele` ngay trên nút).
- Thêm `flex-shrink: 0` (và bỏ `overflow: hidden` trên `.sb-repo-group`) để chấm dứt lỗi CSS Flexbox bóp chiều cao các nhóm Repo.
- Hợp nhất thanh Filter + Counter, chống tràn ngang trên `.sb-task-key`, và hiển thị trực tiếp 2 nút **`⚡ All`** (`approve_and_run`) & **`🧩 Từng Sub`** (`step_subtasks`) trên thẻ Task đang chờ duyệt.

---

## 2. Code Demo Chi Tiết (Exact Changes by `file:line`)

### 2.1. `internal/webui/telegram_bridge.go` — Dual Approval Buttons (`Approve All` vs `Approve Từng Sub-Task`) & Thẻ Review Từng Sub-Task Tinh Tế (`1/10 -> 10/10`)

```go
// Trong buildTelegramApprovalRequiredMessage (internal/webui/telegram_bridge.go:1138-1167):
	sb.WriteString("\n👉 *CHỌN CHẾ ĐỘ THỰC THI KHI BƯỚC VÀO S4 IMPLEMENT:*\n")
	sb.WriteString(fmt.Sprintf("1️⃣ *⚡ Approve All (%d/%d Sub-Tasks):* Bấm nút *⚡ Approve All & Chạy 1..%d -> S5 (#%d)* để duyệt và chạy tự động toàn bộ %d Sub-Tasks tuần tự theo Wave (`Wave 0 -> Wave 1 -> Wave 2`), xong đủ %d/%d mới qua bước S5 Audit của Task cha.\n", len(subTasks), len(subTasks), len(subTasks), d.TaskID, len(subTasks), len(subTasks), len(subTasks)))
	sb.WriteString(fmt.Sprintf("2️⃣ *🧩 Approve Từng Sub-Task (1/%d -> %d/%d):* Bấm nút *🧩 Duyệt Từng Sub-Task (1..%d) (#%d)* để kiểm tra & Approve từng Sub-Task một qua Telegram; xong hết %d/%d Sub-Tasks hệ thống mới chạy tiếp các Wave sau của Task cha.\n", len(subTasks), len(subTasks), len(subTasks), len(subTasks), d.TaskID, len(subTasks), len(subTasks)))
	sb.WriteString(fmt.Sprintf("3️⃣ *✏️ Sửa Giải Pháp:* Bấm nút *✏️ Phản hồi / Sửa Giải Pháp (#%d)* (hoặc Reply tin nhắn này) để yêu cầu Brain sửa lại Plan.", d.TaskID))

	refToken := fmt.Sprintf("#%d", d.TaskID)
	if d.TaskID <= 0 {
		refToken = d.TaskKey
	}
	rows := [][]map[string]string{
		{
			{
				"text":          fmt.Sprintf("⚡ Approve All (%d/%d Sub-Tasks -> S5) (%s)", len(subTasks), len(subTasks), refToken),
				"callback_data": fmt.Sprintf("APPROVE_ALL %s", refToken),
			},
		},
		{
			{
				"text":          fmt.Sprintf("🧩 Duyệt Từng Sub-Task (1/%d -> %d/%d) (%s)", len(subTasks), len(subTasks), len(subTasks), refToken),
				"callback_data": fmt.Sprintf("STEP_SUBTASKS %s", refToken),
			},
		},
		{
			{
				"text":          fmt.Sprintf("✏️ Phản hồi / Sửa Giải Pháp (%s)", refToken),
				"callback_data": fmt.Sprintf("REVISE_PROMPT %s", refToken),
			},
			{
				"text":          "📋 Hàng Đợi (/task)",
				"callback_data": "/task",
			},
		},
	}
```

```go
// Hàm mới trong internal/webui/telegram_bridge.go: Tạo Thẻ Review Tinh Tế cho từng Sub-Task [k/N] và điều phối Step-by-Step -> Parent S5
func renderSubTaskProgressBar(doneCount, totalCount int) string {
	if totalCount <= 0 {
		return "[■■■■■■■■■■] 0/0"
	}
	slots := 10
	filled := (doneCount * slots) / totalCount
	if doneCount > 0 && filled == 0 {
		filled = 1
	}
	if filled > slots {
		filled = slots
	}
	return "[" + strings.Repeat("■", filled) + strings.Repeat("□", slots-filled) + fmt.Sprintf("] %d/%d DONE", doneCount, totalCount)
}

func (s *Server) buildTelegramSubTaskReviewMessage(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO, subTasks []SubTaskDTO, targetIdx int, headerNote string) (string, [][]map[string]string) {
	total := len(subTasks)
	if targetIdx < 0 || targetIdx >= total {
		targetIdx = 0
	}
	st := subTasks[targetIdx]
	doneCnt := 0
	for _, item := range subTasks {
		if strings.EqualFold(item.Status, "done") {
			doneCnt++
		}
	}

	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	safeSub := sanitizeTaskKeyPath(st.SubTaskID)
	subDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSub)
	subPlanFile := filepath.Join(subDir, "09_tasks_solution.md")
	rawSubPlan, _ := os.ReadFile(subPlanFile)
	rcLines, solLines, revLines, codeDemo := extractMarkdownRootCauseAndSolution(string(rawSubPlan), "")

	var sb strings.Builder
	if strings.TrimSpace(headerNote) != "" {
		sb.WriteString(headerNote + "\n\n")
	}
	sb.WriteString(fmt.Sprintf("🧩 *[SUB-TASK REVIEW REQUIRED · BƯỚC %d/%d · WAVE %d (#%d)]*\n", targetIdx+1, total, st.WaveIndex, qt.ID))
	sb.WriteString(fmt.Sprintf("• *Task cha:* `%s` (#%d) — %s\n", qt.ExternalID, qt.ID, truncateRunes(qt.Title, 70)))
	sb.WriteString(fmt.Sprintf("• *Tiến độ Sub-Tasks:* `%s` (Chỉ khi xong đủ `%d/%d` mới chạy tiếp Wave S5 của Task cha)\n", renderSubTaskProgressBar(doneCnt, total), total, total))
	sb.WriteString(fmt.Sprintf("• *Sub-Task [%d/%d]:* `%s` — *%s*\n", targetIdx+1, total, st.SubTaskID, st.Title))
	sb.WriteString(fmt.Sprintf("• *Wave & Phụ thuộc:* `Wave %d` (DependsOn: `%v`)\n", st.WaveIndex, st.DependsOn))
	sb.WriteString(fmt.Sprintf("• *Repo & Điểm chạm:* `📦 %s` · 📄 `%s`\n", st.TargetRepoName, st.ContextFileLine))
	sb.WriteString(fmt.Sprintf("• *Role & Model:* `%s (%s)` · `%s`\n", st.AgentRole, st.AgentSlot, st.Model))
	sb.WriteString(fmt.Sprintf("• *File Plan Sub-Task:* `%s`\n", subPlanFile))

	if len(revLines) > 0 {
		sb.WriteString("\n✏️ *Phản hồi Sub-Task đã cập nhật:*\n")
		for _, r := range revLines {
			sb.WriteString("  " + r + "\n")
		}
	}

	sb.WriteString("\n🔍 *1. MỤC TIÊU & PHẠM VI SUB-TASK:*\n")
	sb.WriteString(fmt.Sprintf("  - %s\n", st.Description))
	for i, rc := range rcLines {
		if i >= 2 {
			break
		}
		sb.WriteString("  " + rc + "\n")
	}

	sb.WriteString("\n💡 *2. GIẢI PHÁP & CODE DEMO SUB-TASK:*\n")
	for i, sol := range solLines {
		if i >= 3 {
			break
		}
		sb.WriteString("  " + sol + "\n")
	}
	if strings.TrimSpace(codeDemo) != "" {
		sb.WriteString("```go\n" + truncateRunes(codeDemo, 380) + "\n```\n")
	}

	sb.WriteString(fmt.Sprintf("\n🎯 *3. TIÊU CHUẨN NGHIỆM THU (DoD):*\n  - `%s`\n", st.TestDoD))

	refToken := fmt.Sprintf("#%d", qt.ID)
	if qt.ID <= 0 {
		refToken = qt.ExternalID
	}
	rows := [][]map[string]string{
		{
			{
				"text":          fmt.Sprintf("✅ Approve Sub-Task %d/%d (%s) & Chạy", targetIdx+1, total, st.SubTaskID),
				"callback_data": fmt.Sprintf("APPROVE_SUB %s %s", refToken, st.SubTaskID),
			},
		},
		{
			{
				"text":          fmt.Sprintf("✏️ Sửa Plan Sub-Task %d/%d (%s)", targetIdx+1, total, st.SubTaskID),
				"callback_data": fmt.Sprintf("REVISE_SUB_PROMPT %s %s", refToken, st.SubTaskID),
			},
			{
				"text":          fmt.Sprintf("📋 Bảng %d Sub-Tasks", total),
				"callback_data": fmt.Sprintf("LIST_SUBTASKS %s", refToken),
			},
		},
		{
			{
				"text":          fmt.Sprintf("⚡ Approve All Còn Lại (Chạy hết %d/%d -> S5)", total-doneCnt, total),
				"callback_data": fmt.Sprintf("APPROVE_ALL %s", refToken),
			},
		},
	}
	return sb.String(), rows
}
```

### 2.2. `internal/webui/server.go` — Rào chắn `Parent S4 -> S5 Wave Barrier` & Hai Chế độ `step_subtasks` / `approve_all`

```go
// Trong runTaskLifecycleActionWithDelay (internal/webui/server.go:1184-1196):
	if action == "step_subtasks" {
		_, err := s.handleTelegramStepSubTasksCommand(execCtx, qt.ExternalID, "", "", "webui", stepDelay)
		return err
	}

	if action == "approve_and_run" || action == "approve_all" {
		// Xóa marker step-by-step nếu chuyển sang chế độ Approve All
		isoMeta := s.buildTaskIsolationMeta(qt.ExternalID)
		_ = os.Remove(filepath.Join(isoMeta.ArtifactDir, ".step-by-step-subtasks"))
		if err := s.runSingleStageWithDelay(execCtx, qt.ExternalID, "approve", activeRunner, stepDelay); err != nil {
			return err
		}
		if execCtx.Err() != nil {
			return nil
		}
		// Thực thi tuần tự từng Sub-Task 1..N theo đúng thứ tự Wave (Wave 0 -> Wave 1 -> Wave 2) trước khi chạy Parent S4/S5
		subTasks := s.loadOrGenerateSubTasks(*qt)
		if len(subTasks) > 0 {
			subTasks = s.executeAllSubTasksWaveByWave(execCtx, *qt, subTasks, stepDelay)
			_ = s.saveTaskSubTasks(qt.ExternalID, subTasks)
		}
		return s.runAutonomousS4S5WithSelfHealing(execCtx, qt.ExternalID, activeRunner, stepDelay)
	}
```

### 2.3. `internal/webui/static/index.html`, `app.css`, `app.js` — Sửa lỗi UI/UX Sidebar `Hàng Đợi Task (Theo Repo)`, nút `✈️ /task` và tích hợp 2 nút `⚡ Duyệt All` / `🧩 Từng Sub`

```html
        <!-- internal/webui/static/index.html:130-151 -->
        <div class="sidebar-section task-queue-section">
          <div class="vibe-col-hdr">
            <span class="vibe-col-title" title="Hàng Đợi Task (Theo Repo)">Hàng Đợi Task (Theo Repo)</span>
            <div class="vibe-col-actions">
              <button id="btn-push-tg-task-menu" type="button" class="mini-tg-btn" onclick="sendTelegramTaskMenuNow(this)" title="Đẩy danh sách Hàng Đợi Task (/task) kèm nút Click-to-Run lên Telegram ngay">✈️ /task</button>
              <button id="btn-cleanup-done-tasks" type="button" class="mini-clean-btn" onclick="deleteDoneTasksBulk()" title="Xóa gọn toàn bộ các Task đã hoàn tất (DONE) cho bớt rối">🧹 Dọn</button>
              <button type="button" class="mini-add-btn" onclick="openDispatchModal()" title="Thêm Task mới (+ Chọn Repo & Chia Sub-Tasks)">+ Task</button>
            </div>
          </div>
          <div id="sidebar-queue-filter-bar" class="sb-queue-filter-bar">
            <button type="button" class="sb-qf-btn active" data-qfilter="all" onclick="setSidebarQueueFilter('all')" title="Hiển thị toàn bộ Task theo Repo">Tất cả <b id="sb-qs-all">0</b></button>
            <button type="button" class="sb-qf-btn running" data-qfilter="active" onclick="setSidebarQueueFilter('active')" title="Lọc Task đang chạy hoặc chờ chạy">⚡ Chạy <b id="sb-qs-running">0</b></button>
            <button type="button" class="sb-qf-btn needs-you" data-qfilter="needs_you" onclick="setSidebarQueueFilter('needs_you')" title="Lọc Task đang chờ APPROVE S3">⏸ Duyệt <b id="sb-qs-needs-you">0</b></button>
            <button type="button" class="sb-qf-btn done" data-qfilter="done" onclick="setSidebarQueueFilter('done')" title="Lọc Task đã hoàn tất">✓ Xong <b id="sb-qs-done">0</b></button>
          </div>
          <div id="sidebar-task-queue" class="sidebar-task-list"></div>
        </div>
```

```javascript
// internal/webui/static/app.js:852-877 — Hỗ trợ cả 2 nút ⚡ All và 🧩 Từng Sub ngay trên thẻ Task ở Sidebar
          const actionButtonsHtml = isTaskRunning
            ? `
              <button type="button" class="sb-stop-btn"
                      title="Dừng ngay tiến trình của ${escapeAttr(n.key)}"
                      onclick="stopRunningProcess('${escapeAttr(n.key)}', '', event)">
                ⏹ Dừng
              </button>
              <button type="button" class="sb-del-btn"
                      title="Xóa Task ${escapeAttr(n.key)}"
                      onclick="deleteSingleTask('${escapeAttr(n.key)}', event)">
                🗑
              </button>
            `
            : n.stage === "NEEDS_YOU"
            ? `
              <button type="button" class="sb-run-btn"
                      title="Approve All: Duyệt & chạy tự động hết ${subTotal} Sub-Tasks theo Wave -> S5"
                      onclick="runTaskAction('${escapeAttr(n.key)}', 'approve_and_run', event)">
                ⚡ All
              </button>
              <button type="button" class="sb-step-btn"
                      title="Approve Từng Sub-Task: Duyệt & chạy từng Sub-Task (1/${subTotal} -> ${subTotal}/${subTotal}) qua Telegram/WebUI"
                      onclick="runTaskAction('${escapeAttr(n.key)}', 'step_subtasks', event)">
                🧩 Từng Sub
              </button>
              <button type="button" class="sb-del-btn"
                      title="Xóa Task ${escapeAttr(n.key)} khỏi hàng đợi"
                      onclick="deleteSingleTask('${escapeAttr(n.key)}', event)">
                🗑
              </button>
            `
            : `
              <button type="button" class="sb-run-btn"
                      title="Kích hoạt chạy ngay ${escapeAttr(n.key)}"
                      onclick="runTaskAction('${escapeAttr(n.key)}', '${btnAction}', event)">
                ${btnLabel}
              </button>
              <button type="button" class="sb-del-btn"
                      title="Xóa Task ${escapeAttr(n.key)} khỏi hàng đợi"
                      onclick="deleteSingleTask('${escapeAttr(n.key)}', event)">
                🗑
              </button>
            `;
```
