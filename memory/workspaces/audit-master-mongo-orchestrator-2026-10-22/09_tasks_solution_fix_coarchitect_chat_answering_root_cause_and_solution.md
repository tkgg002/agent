---
tier: T2
suffix: fix_coarchitect_chat_answering_root_cause_and_solution
mode: standard
status: PROPOSE_ONLY_WAITING_APPROVE
---

# 09. Giải Pháp Kỹ Thuật Tối Ưu (S3 Single-Best Solution) — Trả Lời Trực Tiếp Câu Hỏi Kỹ Thuật Trong `💬 THẢO LUẬN KIẾN TRÚC` (`task_chat.go`) & Đồng Bộ Hồ Sơ `CDC-RECON-01`

## 1. Kiến trúc Giải pháp Duy Nhất (Single-Best Solution)

1. **Nâng cấp `buildContextualDiscussReply` & Bổ sung `extractTaskDeepKnowledgeForDiscussion` trong [`internal/webui/task_chat.go:1325-1399`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go#L1325-L1399):**
   - Đọc kết hợp hồ sơ của Task từ `.orchestrator/tasks/<TaskKey>/` (`09_tasks_solution.md`, `13_analysis.md`, `audit_report.md`, `08_tasks.md`) **VÀ** workspace `/agent/memory/workspaces/` liên kết (bao gồm cả `audit-shadow-master-update-drop-2026-10-20` cho `CDC-RECON-01`).
   - Trích xuất đầy đủ:
     - **🔍 Nguyên nhân gốc rễ (`Vì sao chạy không đúng`):** Các mục `Sơ hở F1..F8` / `Root Cause` kèm `file:line` và số liệu DB thực tế.
     - **💡 Giải pháp kỹ thuật & Code Demo (`Giải pháp là gì`):** Các mục `Giải pháp cho Sơ hở F1..F8`, `AF-01..AF-05` và khối `Code Demo`.
     - **📋 Danh mục Sub-Tasks & Điểm chạm Đa Repo (`08_tasks.md`):** Trạng thái từng Sub-Task và các file thay đổi trên từng Repo.
   - Khi User hỏi bất kỳ câu hỏi nào về nguyên nhân/giải pháp/kiến trúc (`vì sao`, `tại sao`, `nguyên nhân`, `ko đúng`, `sai`, `lỗi`, `giải pháp`, `sửa`, `fix`, `làm sao`, `code`, `file`, hoặc câu hỏi thảo luận chung), `buildContextualDiscussReply` trả lời trực tiếp, đầy đủ và chi tiết vào đúng câu hỏi của User thay vì nhại lại câu hỏi + liệt kê đường dẫn file!
2. **Đồng bộ hồ sơ thực chứng của `CDC-RECON-01` & Cập nhật `chat_history.json`:**
   - Bảo toàn nội dung phân tích `F1..F8` trong `.orchestrator/tasks/CDC-RECON-01/13_analysis.md` và cập nhật ngay câu trả lời lúc `15:21:12` trong `.orchestrator/tasks/CDC-RECON-01/chat_history.json` để giao diện `:8090` hiển thị câu trả lời chi tiết ngay lập tức.
3. **Hỗ trợ Hỏi-Đáp Kỹ Thuật trực tiếp trên Telegram (`internal/webui/telegram_bridge.go:495-535`):**
   - Khi User nhắn câu hỏi thảo luận (`vì sao...`, `giải pháp là gì...`) trên Telegram, tự động gọi `buildContextualDiscussReply` và gửi câu trả lời chi tiết qua Telegram.
4. **Apply Trực Tiếp Hunk Code Của Từng Sub-Task Đã Duyệt (`k/10`) Xuống Working Tree Của `TargetRepo` (`applyApprovedSubTaskPatchToTargetRepo` tại [`internal/webui/telegram_bridge.go:2999-3108`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L2999-L3108) & [`internal/scout/scout.go:21-43`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/scout/scout.go#L21-L43)):**
   - Sửa `scout.MaskSecrets` không mask nhầm các placeholder format (`%s:%s@`, `%v:%v@`) hoặc biểu thức biến (`connForm.password`, `req.Password`).
   - Trong `synthesizeIsolatedRemediationDiffIfApproved` (được gọi tại `subtask_approve` & `subtask_implement`), khi một Sub-Task `st` đã có `.user-approved` (hoặc chạy `Approve All`), hệ thống trích xuất các hunk unified diff thực chất của các file thuộc `st` từ `diff.patch`, khôi phục các placeholder bị mask nhầm trước đó (`mongodb://***:***@%s` -> `mongodb://%s:%s@%s`, `postgres://***:***@%s` -> `postgres://%s:%s@%s`, `password:***` -> `password: connForm.password,`), rồi chạy `git -C <TargetRepo> apply --check` / `git -C <TargetRepo> apply` (fallback `git -C <TargetRepo> apply --3way`) để ghi thẳng thay đổi code xuống working tree của `TargetRepo` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`) ngay khi Sub-Task đó được duyệt!
5. **Thực Thi Đầy Đủ 100% Chu Trình 12 Bước / 6 Wave Chuẩn Fractal Cho Từng Sub-Task (`1/10 .. 10/10`) Trong [`internal/webui/task_inspector.go:2989-3215`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go#L2989-L3215) & [`internal/webui/telegram_bridge.go:1680-1894`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L1680-L1894):**
   - Mỗi Sub-Task (`1/10 .. 10/10`) vận hành như một Task độc lập với **đủ 12 Bước / 6 Wave chuẩn**:
     - **Wave 1 — Pre-Approval (`subtask_plan`: `Step 1/12 -> Step 4/12`, tự động chạy trước khi dừng ở `NEEDS_YOU` chờ anh duyệt Sub-Task đó):**
       - `Step 1/12 (10%)`: `S0 Boot H1 & S1 Sub-Task Intake` (khởi tạo `01_requirements.md`, `05_progress.md`, `08_tasks.md` + kiểm tra `InputContract` từ Wave trước).
       - `Step 2/12 (20%)`: `S2 Deep Touchpoint Repo Scan & FTS5 Lesson Tripwires` (trích xuất đúng đoạn code gốc `file:line` trên `st.TargetRepo` vào `13_analysis.md`).
       - `Step 3/12 (30%)`: `S2.5 Pre-Plan Adversarial Critique Wave (Laya DG0 :8200 · 0 Token)` (kiểm định tính trung thực của điểm chạm `file:line` & lưu `06_laya_summary_dg0.json`).
       - `Step 4/12 (40%)`: `S3 Single-Best Solution, Code Demo & Laya DG1/DG2 (:8200 · 0 Token)` (sinh `09_tasks_solution.md` + chạy `DG1/DG2` đạt `PASS` + dừng ở `NEEDS_YOU` và bắn Card duyệt Telegram cho Sub-Task).
     - **Wave 2 — Human Gate (`subtask_approve`: `Step 5/12`):**
       - `Step 5/12 (50%)`: `Human Gate · You APPROVED Sub-Task S3 Plan` (ghi `.user-approved`, mở khóa `H4b`).
     - **Wave 3 — MUSCLE Execution (`subtask_implement`: `Step 6/12 -> Step 8/12`):**
       - `Step 6/12 (60%)`: `S4 Slot C Worktree & Git Restore-Point (Rule #19)` (lưu `HEAD` SHA của `st.TargetRepo` vào `restore_points.json` của Sub-Task).
       - `Step 7/12 (70%)`: `S4 Real Code Implementation & Git Apply on TargetRepo Working Tree` (chạy `applyApprovedSubTaskPatchToTargetRepo` để `git status` trên `st.TargetRepo` hiện ngay file sửa thật + đồng bộ `subtasks/<ID>/diff.patch`).
       - `Step 8/12 (78%)`: `S4 Gate H4b Zero-Mockup (*Fn), H3 Secret Scan & Laya DG2 (:8200)` (quét chống closure giả `*Fn` & rò rỉ secret trên các file vừa sửa của `st.TargetRepo` + chạy `DG2`).
     - **Wave 4 — Adversarial QA & Self-Healing (`subtask_qa`: `Step 9/12 -> Step 11/12`):**
       - `Step 9/12 (86%)`: `S5 5A Hard Automated Gate (Zero-LLM Compiler/Vet/Test EV-... on TargetRepo & On-Disk Git Status Check)` (chạy lệnh `buildTargetedRepoCheckCommand` thật trên `st.TargetRepo` qua `Gatekeeper.RunEvidenceCommand` sinh biên lai `EV-...` + kiểm tra `VerifySubTaskCodeOnDisk`).
       - `Step 10/12 (92%)`: `S5 5B Cross-Model Line-by-Line Diff Critique & Laya DG3 (:8200)` (xuất `audit_report.md` đủ 6 mục + chạy `DG3` trên `:8200`).
       - `Step 11/12 (96%)`: `S5 Self-Healing Loop Verification (Tier-1 S5->S4 / Tier-2 S5->S3)` (kiểm tra `0 blocking findings`; nếu có lỗi tự động kích hoạt vòng tự sửa `Max 3 retries`).
     - **Wave 5 — Close & Handoff (`subtask_finalize`: `Step 12/12`):**
       - `Step 12/12 (100%)`: `S6 Finalize, Governance Audit (H7) & OutputContract Handoff` (ghi `DONE` vào `05_progress.md`, bật `OutputContract.HandoffReady = true` để bàn giao sang Sub-Task kế tiếp, và tự động chạy `Wave 1 (Step 1/12..4/12)` cho Sub-Task `k+1/10` trước khi gửi Card duyệt Telegram!).
6. **Tự Động Tạo Sub-Chat (Forum Topic) Riêng Trên Telegram Cho Mỗi Task (`ensureTelegramTaskTopic` & Định Tuyến 2 Chiều `message_thread_id` Trong [`internal/webui/telegram_bridge.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go)):**
   - Lưu bảng ánh xạ `TaskThreadIDs map[string]int64` trong `.orchestrator/telegram_settings.json` và `.orchestrator/tasks/<SafeTaskKey>/telegram_topic.json`.
   - Hàm `ensureTelegramTaskTopic(ctx, qt)`: nếu Task chưa có `message_thread_id`, gọi `POST /bot<TOKEN>/createForumTopic` trên Supergroup `-1004390214071` với tên Topic `📦 [<Repo>] #<ID> <Title>` (<= 120 ký tự) để tạo Sub-Chat riêng cho Task đó.
   - Mọi bản tin của Task (Hồ sơ duyệt S3, Card duyệt từng Sub-Task `1/10 .. 10/10`, tiến độ 12 bước Wave, báo cáo S5 Audit, thảo luận Co-Architect) đều tự động gửi kèm `"message_thread_id": threadID` vào đúng Sub-Chat của Task đó (tự động tạo lại Topic nếu Topic cũ bị xóa).
   - Khi User nhắn tin hoặc bấm nút trong một Sub-Chat (`msg.MessageThreadID > 0`), hàm `findTaskKeyByTelegramThreadID(msg.MessageThreadID)` tự động ánh xạ ngược về đúng `TaskKey` để duyệt Task/Sub-Task, sửa giải pháp S3, hoặc trả lời câu hỏi kỹ thuật (`"vì sao chạy ko đúng, giải pháp là gì"`) ngay trong Sub-Chat đó!
7. **Khắc Phục Treo `Step 15/17 · 95%`, Lỗi `go build ./test` Không Có File `.go`, Vòng Lặp Retry Mù 3 Lần (`39 runs` & Rollback Oan `Circuit Breaker`), Tự Động Tạo `TASK-BUG-*` Để Tự Fix, Và Cơ Chế Core Nạp Ground-Truth Phản Biện Với `/laya_router` (`internal/webui/server.go:1284-1332, 1824-2081, 3297-3359` & `internal/webui/telegram_bridge.go:3110-3199`):**
   - **Lọc chuẩn package Go trong `buildTargetedRepoCheckCommand` (`server.go:3297-3359`):** Bổ sung `hasDirectGoFiles(dir)` trước khi thêm một thư mục vào `go build`. Với các thư mục không có file `.go` trực tiếp như `centralized-data-service/test`, không đưa `./test` vào `go build`, chấm dứt hoàn toàn lỗi `no Go files in .../centralized-data-service/test` (`EV-20261026T163929-840e`).
   - **Hoàn tất trọn vẹn `Step 16/17 (98%)` & `Step 17/17 (100%)` trong `case "qa"` (`server.go:1929-2081`):** Sau `Step 15/17 · 95%`, chạy tiếp `Step 16/17 · 98% (S6 Finalize, Audit Report & Governance Linter H7)` và `Step 17/17 · 100% (Task Complete & Clean Handoff)` rồi gọi `s.finishLiveStepProgress(taskKey)` để thanh tiến trình đạt `17/17 · 100%` và không bao giờ treo ở `15/17`.
   - **Tự động khởi tạo `TASK-BUG-XX` tường minh & tự sửa lỗi (`diagnoseAndCreateS5BugSubTask`), chấm dứt spam Approve 3 lần & rollback oan:**
     - Khi `S5` phát hiện lỗi thật (`blocking > 0`), hệ thống bóc tách chính xác lỗi từ `qaLogs` / biên lai `EV-...`, tự động khởi tạo Sub-Task **`TASK-BUG-01` (`[Triệu chứng EV-... + File/Pkg + Nguyên nhân + Hành động Fix]`)** vào `subtasks.json`, `08_tasks.md`, `09_tasks_solution.md`, ghi rõ ra `terminal.log` và Sub-Chat Telegram, tự động thực thi sửa lỗi ở `S4` rồi nghiệm thu lại ở `S5` (`TASK-BUG-01 -> DONE`).
     - Tuyệt đối không bắn thông báo đòi User `Approve` giả trong lúc đang auto-retry Tier-1; nếu thực sự cần User duyệt lại giải pháp kiến trúc thì dừng hẳn vòng lặp (`break`) ở `NEEDS_YOU` chờ User bấm `Approve` thật, không bao giờ tự lặp 3 lần rồi kích hoạt `Circuit Breaker` rollback oan `TASK-M04, TASK-M05`.
     - Dọn sạch các bản ghi lặp trong `autopilot_rounds` (`deduplicateAutopilotRounds`) và khôi phục `TASK-M04, TASK-M05` về trạng thái `done`.
   - **Core Agent nạp Full Context tranh biện với `/laya_router` theo Ground Truth (`debateAndResolveLayaWithGroundTruth` theo `GEMINI.md` Rule #10):**
     - Khi `/laya_router` (`:8200`) trả về `FLAG`, `UNVALIDATED` hoặc điểm thấp nhưng Ground Truth vật lý (`VerifySubTaskCodeOnDisk == PASS` + biên lai compiler `EV-... exit=0` + `REF: file:line` khớp trên đĩa) đã xác thực, Core Agent nạp full context (`REF: file:line` + `EV-...` + `git diff`) để phản biện lại Laya và ghi quyết định `LAYA-DEBATE-RESOLVED` vào `04_decisions.md` + `audit_report.md`.

---

## 2. Code Demo Chi Tiết (`internal/webui/task_chat.go`, `internal/webui/telegram_bridge.go`, `internal/webui/task_inspector.go` & `internal/webui/server.go`)

### 2.1. Trả lời câu hỏi kỹ thuật trong `💬 THẢO LUẬN KIẾN TRÚC` (`internal/webui/task_chat.go`)
```go
// Trong internal/webui/task_chat.go (thay thế nhánh fallback khuôn mẫu tại L1379-1399):
func (s *Server) buildContextualDiscussReply(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	cleanMsg string,
	agentRoleLabel, agentSlot, skillBadge, selectedModel string,
	lessonRefs []string,
) string {
	lowerMsg := strings.ToLower(cleanMsg)
	// ... (giữ Case 1: "mày là ai" & Case 2: "so với antigravity") ...

	// Bóc tách toàn bộ Nguyên nhân gốc rễ (Root Causes F1..Fn), Giải pháp kỹ thuật (Solutions) & Code Demo
	// từ 13_analysis.md, 09_tasks_solution.md (cả trong .orchestrator/tasks/<TaskKey> và /agent/memory/workspaces/)
	rootCauses, solutions, codeSnippet, fileTouches := s.extractTaskDeepKnowledgeForDiscussion(qt, isoMeta, subTasks)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**[%s · `%s` · `%s`]** Dạ anh, em xin trả lời trực tiếp câu hỏi của anh trên **`%s` (%s)**:\n\n",
		agentRoleLabel, agentSlot, selectedModel, qt.ExternalID, qt.Title))

	if len(rootCauses) > 0 {
		sb.WriteString("### 🔍 1. Nguyên nhân gốc rễ (Vì sao chạy không đúng)\n")
		for i, rc := range rootCauses {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rc))
		}
		sb.WriteString("\n")
	}

	if len(solutions) > 0 {
		sb.WriteString("### 💡 2. Giải pháp kỹ thuật khắc phục triệt để\n")
		for i, sol := range solutions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, sol))
		}
		sb.WriteString("\n")
	}

	if strings.TrimSpace(codeSnippet) != "" && (strings.Contains(lowerMsg, "giải pháp") || strings.Contains(lowerMsg, "code") || strings.Contains(lowerMsg, "sửa") || strings.Contains(lowerMsg, "fix") || strings.Contains(lowerMsg, "vì sao")) {
		sb.WriteString("### 🧩 3. Trích đoạn Code Demo cốt lõi\n")
		sb.WriteString("```go\n" + truncateRunes(codeSnippet, 900) + "\n```\n\n")
	}

	if len(fileTouches) > 0 {
		sb.WriteString("### 📂 4. Điểm chạm Đa Repository & Sub-Tasks (`file:line`)\n")
		for _, ft := range fileTouches {
			sb.WriteString("- " + ft + "\n")
		}
	}

	return strings.TrimSpace(sb.String())
}
```

### 2.2. Apply hunk code của từng Sub-Task đã duyệt xuống Working Tree của `TargetRepo` (`internal/webui/telegram_bridge.go`)
```go
// applyApprovedSubTaskPatchToTargetRepo applies the real unified diff hunks belonging to an approved Sub-Task
// directly onto st.TargetRepo's working tree (via `git apply` / `git apply --3way`) so `git status` immediately
// reflects the exact code changes of each approved Sub-Task (1/10 .. 10/10).
func (s *Server) applyApprovedSubTaskPatchToTargetRepo(ctx context.Context, qt queue.QueuedTask, st SubTaskDTO, subHunks []string) int {
	repoDir := strings.TrimSpace(st.TargetRepo)
	if repoDir == "" || s.isTestTaskOrTempRoot(qt.ExternalID) {
		return 0
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		return 0
	}
	applied := 0
	for _, rawHunk := range subHunks {
		cleanHunk := restoreFalsePositiveMaskedCodePlaceholders(rawHunk)
		if !strings.HasPrefix(strings.TrimSpace(cleanHunk), "diff --git ") ||
			strings.Contains(cleanHunk, "Isolated Remediation Hunk for ") {
			continue
		}
		if tryGitApplySectionToRepo(ctx, repoDir, cleanHunk) {
			applied++
		}
	}
	return applied
}
```

### 2.3. Thực thi đầy đủ 12 Bước / 6 Wave chuẩn Fractal cho từng Sub-Task (`internal/webui/task_inspector.go`)
```go
// Trong runSubTaskLifecycleActionWithDelay (internal/webui/task_inspector.go:2989-3215):
// - subtask_plan: Chạy đủ Wave 1 (Step 1/12 S0/S1 Intake -> Step 2/12 S2 Repo Scan -> Step 3/12 S2.5 DG0 (:8200) -> Step 4/12 S3 Solution + DG1/DG2 (:8200) -> dừng ở NEEDS_YOU).
// - subtask_approve: Chạy Wave 2 (Step 5/12 Human Gate APPROVE & Unlock H4b).
// - subtask_implement: Chạy đủ Wave 3 (Step 6/12 S4 Restore-Point -> Step 7/12 S4 Real Git Apply xuống st.TargetRepo -> Step 8/12 S4 Gate H4b Zero-Mockup & H3 Secret Scan + Laya DG2).
// - subtask_qa: Chạy đủ Wave 4 (Step 9/12 S5 5A Real Compiler/Vet/Test EV-... trên st.TargetRepo + VerifySubTaskCodeOnDisk -> Step 10/12 S5 5B Line-by-Line Diff Critique & Laya DG3 (:8200) -> Step 11/12 S5 Self-Healing Loop Check) + Wave 5 (Step 12/12 S6 Finalize, Governance Audit H7 & OutputContract Handoff).
```

### 2.4. Tự động tạo Sub-Chat (Forum Topic) riêng cho mỗi Task & định tuyến `message_thread_id` (`internal/webui/telegram_bridge.go`)
```go
// ensureTelegramTaskTopic creates (or reuses) a dedicated Telegram Forum Topic (Sub-Chat) for a Task
// in the Supergroup via `POST /bot<TOKEN>/createForumTopic` and persists `message_thread_id`.
func (s *Server) ensureTelegramTaskTopic(ctx context.Context, qt queue.QueuedTask, forceRecreate bool) int64 {
	if strings.TrimSpace(qt.ExternalID) == "" {
		return 0
	}
	cfg := s.loadTelegramPersistedConfig()
	if !forceRecreate {
		if tid := s.lookupPersistedTaskThreadID(cfg, qt.ExternalID); tid > 0 {
			return tid
		}
	}
	token := strings.TrimSpace(cfg.BotTokenEncrypted)
	chatID := strings.TrimSpace(cfg.ChatID)
	if token == "" || chatID == "" {
		return 0
	}
	repoName := filepath.Base(s.ResolveTaskTargetRepo(qt))
	topicName := truncateRunes(fmt.Sprintf("📦 [%s] #%d %s", repoName, qt.ID, qt.Title), 120)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/createForumTopic", token)
	rawBody, _ := json.Marshal(map[string]any{
		"chat_id": chatID,
		"name":    topicName,
	})
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(rawBody))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	var parsed struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageThreadID int64 `json:"message_thread_id"`
		} `json:"result"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	if parsed.OK && parsed.Result.MessageThreadID > 0 {
		s.saveTaskTelegramThreadID(qt, topicName, parsed.Result.MessageThreadID)
		return parsed.Result.MessageThreadID
	}
	return 0
}
```

### 2.5. Lọc package Go có file `.go` thật, hoàn tất `Step 16/17 -> 17/17`, tự tạo `TASK-BUG-XX` & Core nạp Ground-Truth phản biện với `/laya_router` (`internal/webui/server.go`)
```go
// hasDirectGoFiles checks if a directory actually contains at least one non-test .go source file
// so buildTargetedRepoCheckCommand never passes umbrella directories like `./test` (which only has subdirectories)
// to `go build`.
func hasDirectGoFiles(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}

// diagnoseAndCreateS5BugSubTask extracts exact compiler/QA failure diagnostics (`qaLogs` / `EV-...`),
// creates an explicit `TASK-BUG-XX` remediation sub-task in `subtasks.json` + `08_tasks.md` + `09_tasks_solution.md`,
// logs the root cause clearly to `terminal.log` and the Task's Telegram Sub-Chat, and applies the targeted fix
// before re-running S5 QA.
// CRITICAL RULE (Persistent Until-Done Loop): Even if loop > 3, as long as a new actionable `TASK-BUG-XX` is discovered
// (or the error signature has progressed), `sameBugStallCount` is reset to 0 and the loop CONTINUES retrying
// until `qaBlocking == 0` and the Task is 100% DONE!
func (s *Server) shouldContinuePersistentBugFixLoop(foundActionableBugTask bool, curBugSig, prevBugSig string, sameBugStallCount *int, maxStallOnIdenticalBug int) bool {
	if foundActionableBugTask || (curBugSig != "" && curBugSig != prevBugSig) {
		*sameBugStallCount = 0
		return true // Tiếp tục retry tạo TASK-BUG-XX & fix cho tới khi hoàn thành 100% Task!
	}
	*sameBugStallCount++
	return *sameBugStallCount < maxStallOnIdenticalBug
}
```

---

## 9. Thiết Kế Chi Tiết & Code Demo `TASK-CHAT-07`: Tự Động Tạo Sub-Topic Telegram Theo `ExternalID` Ngay Khi Tạo Task Mới (`handleIntake`) & Bộ Sinh Giải Pháp `S3` Đa Tầng Kiến Trúc + Trình Bày Trực Quan Cho `Core-01`

### 9.1. Tự động tạo Sub-Topic Telegram theo `Mã Task / Ticket ID` (`ExternalID`) ngay khi tạo Task mới (`internal/webui/server.go:932-945` & `internal/webui/telegram_bridge.go`)
- Khi User tạo Task mới (`POST /api/intake` trong `handleIntake` tại `internal/webui/server.go:932`), ngay sau khi `generateDeepTaskArtifacts` sinh xong hồ sơ S1–S3, hệ thống gọi ngay `s.notifyTelegramNewTaskIntakeAndTopic(r.Context(), qt)`:
  1. Gọi `ensureTelegramTaskTopic(ctx, cfg, qt)` tạo Forum Topic trên Supergroup `-1004390214071` với tên chuẩn theo **`Mã Task / Ticket ID` (`qt.ExternalID`)**:
     `🧩 [Core-01] Cập nhật và Nâng cấp chính mày trong ai-orchestrator từ api cho tới ux,ui`
  2. Lưu `message_thread_id` vào `.orchestrator/telegram_settings.json` (`TaskThreadIDs["Core-01"]`) và `.orchestrator/tasks/Core-01/telegram_topic.json`.
  3. Gửi ngay **Hồ sơ Khởi tạo Task & Tóm tắt Giải pháp Kiến trúc S3** vào đúng Sub-Topic `message_thread_id` vừa tạo (kèm các nút tương tác `✅ Duyệt S3 & Chạy S4`, `📋 Duyệt từng Sub-Task (1/6)`, `📄 Xem 09_tasks_solution.md`).
  4. Đồng thời khi khởi động Server hoặc chạy đồng bộ, tự động backfill tạo luôn Sub-Topic trên Telegram cho `Core-01` và các Task hiện có chưa có Sub-Topic.

### 9.2. Nâng cấp Bộ Sinh Giải Pháp `S3` Đa Tầng Kiến Trúc & Định Dạng Trực Quan Dễ Đọc (`internal/webui/task_inspector.go`)
1. **Quét Đa Tầng (`.go`, `.ts`, `.tsx`, `.js`, `.html`, `.css`, `.sql`) & Chấm Điểm Theo Layer (`inspectTargetRepoSymbols` — `task_inspector.go:3274-3328`):**
   - Mở rộng bộ lọc đuôi file tại `L3274` để đọc cả `.html`, `.css`, `.js` (`internal/webui/static/index.html`, `app.css`, `app.js`).
   - Bỏ điểm cộng cào bằng `+4` cho `agentgate`; thay bằng bộ khớp ngữ nghĩa theo **6 Tầng Kiến Trúc (Architectural Layers)**:
     - **Layer 1 — API & Lifecycle Engine:** `internal/webui/server.go` (`handleIntake`, `/api/tasks/lifecycle`, `/api/tasks/subtasks/action`, `buildTargetedRepoCheckCommand`, `Step 1..17` state machine).
     - **Layer 2 — Deep Planner & Sub-Task Wave Engine:** `internal/webui/task_inspector.go` (`DecomposeMultiRepoCodebaseSubTasks`, `generateDeepTaskArtifacts`, `VerifySubTaskCodeOnDisk`, `subtask_git_apply.go`).
     - **Layer 3 — Co-Architect Steering & Ground-Truth Debate:** `internal/webui/task_chat.go` (`handleTaskChat`, `synthesizeRichTaskChatReply`, `debateAndResolveLayaWithGroundTruth`).
     - **Layer 4 — Telegram Forum Sub-Topic Bridge (2-Way):** `internal/webui/telegram_bridge.go` (`ensureTelegramTaskTopic` theo `ExternalID`, per-topic routing, interactive Sub-Task cards).
     - **Layer 5 — WebUI UX/UI & Live Telemetry:** `internal/webui/static/index.html`, `app.css`, `app.js` (Split-view S3 Executive Dossier, Layer Architecture Matrix, Wave Progress Bar 12-Step, Telegram Topic Badge, Clean Diff Viewer).
     - **Layer 6 — E2E & Race-Detector Verification:** `internal/webui/server_test.go` (`go test -race ./internal/webui/...`).
2. **Mã Sub-Task Sạch Từ `ExternalID` (`buildCleanSubTaskIDPrefix`) & Phân Rã 6 Sub-Task Chuyên Sâu (`DecomposeMultiRepoCodebaseSubTasks` — `task_inspector.go:1076-1157`):**
   - Chuyển `qt.ExternalID` (`Core-01`) thành prefix thuần ASCII: `CORE-01-S01` .. `CORE-01-S06` (tuyệt đối không cắt byte UTF-8 trên chữ tiếng Việt `"CẬP"` thành `TASK-CẬ01`).
   - Mỗi Sub-Task gắn đúng 1 tầng kiến trúc với: **`[Tầng Kiến Trúc]`**, **`[Điểm Chạm file:line]`**, **`[Vấn Đề Hiện Tại -> Giải Pháp Cụ Thể]`**, và **`[Tiêu Chí Nghiệm Thu DoD]`**.
3. **Bố Cục Trình Bày `09_tasks_solution.md` Mới — Tinh Gọn, Phân Lớp Rõ Ràng, Dễ Đọc Ngay Trong 10 Giây (`task_inspector.go:3421-3658`):**
   - **Mục 1 — Bảng Tóm Tắt Điều Hành (Executive Summary Table):** Gom gọn Metadata vào bảng 2 cột sạch sẽ, dễ nhìn.
   - **Mục 2 — Chẩn Đoán Hiện Trạng & Ma Trận Giải Pháp Đa Tầng (`Pain-Point file:line -> Target Upgrade`):** Bảng đối chiếu trực diện từng tầng (từ API, Planner, Co-Architect, Telegram Topic cho tới WebUI UX/UI) đang gặp vấn đề gì ở `file:line` nào và nâng cấp ra sao.
   - **Mục 3 — Lộ Trình Triển Khai Sub-Tasks Theo Wave (`Wave 0 -> Wave 1 -> Wave 2`):** Trình bày từng Sub-Task (`CORE-01-S01 .. CORE-01-S06`) ngắn gọn, súc tích, không lặp lại câu tiêu đề dài dòng.
   - **Mục 4 — Code Demo Trước & Sau (Multi-Layer Concrete Code Demo):** Hiển thị code mẫu thực chất cho từng tầng thay vì dump struct `Gatekeeper` cũ trong `gate.go`.

```go
// Code Demo (internal/webui/telegram_bridge.go & internal/webui/task_inspector.go)

// 1. Tạo Forum Sub-Topic trên Telegram bằng chính Mã Task / Ticket ID (ExternalID) ngay khi tạo Task mới:
func formatTelegramTaskTopicNameByExternalID(qt queue.Task) string {
	extID := strings.TrimSpace(qt.ExternalID)
	title := strings.TrimSpace(qt.Title)
	if extID == "" {
		extID = fmt.Sprintf("TASK-%d", qt.ID)
	}
	topicName := fmt.Sprintf("🧩 [%s] %s", extID, title)
	if strings.EqualFold(extID, title) {
		topicName = fmt.Sprintf("🧩 [%s]", extID)
	}
	runes := []rune(topicName)
	if len(runes) > 120 {
		topicName = string(runes[:117]) + "..."
	}
	return topicName
}

// 2. Tạo Prefix Sub-Task thuần ASCII từ ExternalID (chấm dứt lỗi cắt byte UTF-8 "TASK-CẬ01"):
func buildCleanSubTaskIDPrefix(externalID, title string) string {
	raw := strings.TrimSpace(externalID)
	if raw == "" {
		raw = strings.TrimSpace(title)
	}
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToUpper(raw) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if (r == '-' || r == '_' || r == ' ') && b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
		if b.Len() >= 10 {
			break
		}
	}
	clean := strings.Trim(b.String(), "-")
	if clean == "" {
		return "TASK-S"
	}
	return clean + "-S" // Ví dụ: "Core-01" -> "CORE-01-S01", "CORE-01-S02"...
}
```

---

## 10. Giải Pháp Bổ Sung Từ Vòng Lặp Phản Tỉnh `S5 -> S3 -> S4 -> S5` (`TASK-CHAT-08` — Tách File/Module Đơn Trách Nhiệm Đúng Layer Trên `ai-orchestrator`, Cấm Nhồi Nhét File Lớn, Xóa Bỏ Mockup & Bảo Vệ Tuyệt Đối Ranh Giới Repo Ngoài `data-hub`)

1. **Tách Module Đơn Trách Nhiệm Đúng Layer Trên `ai-orchestrator` (Đưa 100% File Lớn Về `<= HEAD`):**
   - `internal/queue/store_rounds_dedup.go` (Layer `queue`): Tách `DeduplicateTaskRounds` và kiểm tra trùng lặp round liên tiếp khỏi `internal/queue/store.go` (`NOW = 530 <= HEAD 542` dòng).
   - `internal/agentgate/laya_ground_truth.go` (Layer `agentgate`): Chuyển logic phân giải `/laya_router` khỏi `internal/webui/server.go` về đúng `internal/agentgate` (`ResolveFlaggedLayaClaimsWithGroundTruth`), gọi trực tiếp CLI `laya_gate.py resolve` cho các claim `FLAG` có biên lai `EV-...` & `REF: file:line` (xóa bỏ việc ghi đè trực tiếp `doc["gate"] = "PASS"` trong JSON).
   - `internal/webui/task_self_heal_loop.go` (Layer `webui` Self-Heal Engine): Tách `shouldContinuePersistentBugFixLoop`, `diagnoseAndCreateS5BugSubTask`, `runAutonomousS4S5WithSelfHealing`, `hasDirectGoFiles`, `buildTargetedRepoCheckCommand`, `finishLiveStepProgress` khỏi `internal/webui/server.go` (`NOW = 3,263 <= HEAD 3,366` dòng); xóa bỏ hardcode `"ESCALATE"` và `len(auditStr)%97`, dùng chữ ký lỗi tất định và khôi phục `AtomicRollbackWaveSubTasks` khi thật sự kẹt `>= maxSelfHealLoops` trên cùng một lỗi.
   - `internal/webui/task_chat_knowledge.go` (Layer `webui` Co-Architect Knowledge): Tách `renderContextualDiscussReply` và `extractTaskDeepKnowledge` khỏi `internal/webui/task_chat.go` (`NOW <= HEAD 1,897` dòng).
   - `internal/webui/telegram_forum_topic.go` (Layer `webui` Telegram Forum Bridge): Tách `formatTelegramTaskTopicName`, `ensureForumTopicForTask`, `notifyTelegramNewTaskIntakeAndTopic`, `resolveTaskThreadID`, `findTaskKeyByThreadID` khỏi `internal/webui/telegram_bridge.go` (`NOW = 3,278 <= HEAD 3,297` dòng), xóa bỏ biến toàn cục `telegramBotAPIBaseURL` phục vụ mockup, và thay thế hoàn toàn tên bot mặc định giả `@ai_orchestrator_s3_bot` / `@ai_orchestrator_bot` tại `telegram_bridge.go:131`, `static/index.html:994`, `static/app.js:3250,3279` bằng Bot thực tế `@tk001sea_bot` khớp với `.orchestrator/telegram_settings.json:6`.
   - `internal/webui/task_planner_multilayer.go` & `internal/webui/subtask_wave_engine.go` (Layer `webui` Planner & Sub-Task Wave Engine): Tách bộ phân rã đa tầng `Core-01` và máy thực thi 12-Step Sub-Task Wave khỏi `internal/webui/task_inspector.go` (`NOW <= HEAD 4,817` dòng).
   - `internal/webui/server_test.go`: Khôi phục về đúng `2,927` dòng của `HEAD`, xóa bỏ hoàn toàn bài test mockup `TestUnifiedUpgrade_TASK_CHAT_01_To_07_And_Core01_E2E` (`tgMock := httptest.NewServer(...)`).

2. **Bảo Vệ Tuyệt Đối Ranh Giới Repository Ngoài (`data-hub` — Lesson `L001 [2026-10-27]`):**
   - Trong `VerifySubTaskCodeOnDisk` (`internal/webui/task_inspector.go:1520`), bổ sung chốt chặn `isIsolatedWorktree := strings.Contains(filepath.ToSlash(repoDir), "/.orchestrator/worktrees/")` để **tuyệt đối cấm** tự động chạy `applyApprovedSubTaskPatchToTargetRepo` lên nhánh làm việc chính của bất kỳ repository bên ngoài nào (`/Users/trainguyen/Documents/work/data-hub/{centralized-data-service,cdc-cms-service,cdc-cms-web}`).

---

## 11. Giải Pháp Quy Chuẩn Một Định Dạng Trình Bày & Nội Dung Duy Nhất (`UDF-v1`) Cho `#sri-markdown-content` Và `.cas-messages-list` (`TASK-CHAT-09`)

### 11.1. Chẩn đoán Nguyên nhân gốc rễ (`file:line`) khiến `#sri-markdown-content` và `.cas-messages-list` khó nhìn
1. **Ở tầng sinh nội dung Backend (`internal/webui/task_inspector.go:3176-3368` & `internal/webui/task_chat_knowledge.go:81-118`):**
   - Trong `09_tasks_solution.md` (`task_inspector.go:3334-3368`), danh sách Sub-Tasks bị lặp lại **3 lần liên tiếp** trong cùng một báo cáo (`## 0. Bảng Tóm Tắt Điều Hành`, `## 1. Ma Trận Điểm Chạm`, và `## 2. Phương Án Kỹ Thuật` dạng bullet 4 dòng/task).
   - Trong `01_requirements.md` (`task_inspector.go:3176-3197`), khối YAML frontmatter `--- ... ---` dài 21 dòng bị đổ nguyên văn lên đầu khung đọc báo cáo.
   - Trong `.cas-messages-list` (`task_chat_knowledge.go:81-118` và `task_chat.go`), câu trả lời của Co-Architect chưa dùng chung cấu trúc 4 mục chuẩn với `#sri-markdown-content`.
2. **Ở tầng hiển thị Frontend (`internal/webui/static/index.html:342`, `app.js:1361,1374,1852`, `app.css:1857,2418,2496`):**
   - `#sri-markdown-content` là thẻ `<pre class="sri-pre-block">` gán `mdEl.textContent = ...` với font chữ monospace `10.5px`! Toàn bộ ký hiệu `#`, `##`, `| :--- |`, `**bold**`, ```` ```go ```` bị in thô thành một bức tường chữ nhỏ li ti, bảng không căn cột.
   - `.cas-messages-list` (`#co-architect-chat-messages`) bó hẹp ở `max-height: 260px` và mỗi tin nhắn đặt trong `<pre class="cas-msg-body">` font `10px` monospace, rất khó theo dõi.

### 11.2. Phương án Kỹ thuật Duy nhất & Tối ưu nhất: Chuẩn Hóa Định Dạng 4 Mục (`UDF-v1`) Tách Module Đơn Trách Nhiệm
1. **Module Backend Đơn Trách Nhiệm — `internal/webui/unified_doc_format.go`:**
   - Cung cấp bộ dựng tài liệu chuẩn 4 mục **`UDF-v1`** dùng chung cho cả `S1..S5` (`#sri-markdown-content`) và Co-Architect Studio (`.cas-messages-list`):
     - **Header Metadata Strip:** `> 📌 **Task:** \`<TaskKey>\` · **Stage:** \`<Stage>\` · **Repos:** \`<RepoSummary>\` · **Role/Model:** \`<Model>\` · **Gate:** \`<GateVerdict>\``
     - **`## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)`**: Tối đa 4–6 ý trọng tâm gắn `file:line`, không lặp chữ.
     - **`## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`file:line` & `DoD`)`**: **Đúng 1 bảng Markdown chuẩn 5 cột** (`| Mã | Wave / Tầng | Repo | Điểm chạm (file:line) | Giải pháp & Nghiệm thu (DoD) |`), xóa bỏ việc lặp lại bảng/danh sách 3 lần.
     - **`## 3. 💻 Code Demo / Diff Thực Thi (`Concrete Implementation`)`**: Khối code mẫu hoặc diff thực thi súc tích.
     - **`## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)`**: Trạng thái cổng `DG0..DG3 (:8200)`, `## Clean-Room Approved Spec` (`REF: file:line`), và hướng dẫn hành động kế tiếp.
2. **Module Frontend Đơn Trách Nhiệm — `internal/webui/static/unified_doc_format.css` & `internal/webui/static/unified_doc_format.js`:**
   - Giữ nguyên số dòng `app.js`, `app.css`, `index.html` `<= HEAD` và giữ nguyên các CSS selector marker trong `app.css` để pass `TestWebUIServer_14AgentCoreFlowGapsClosed`.
   - Hàm `renderUnifiedDocFormat(rawMarkdown, opts)` tự động chuyển đổi Markdown chuẩn `UDF-v1` thành giao diện trực quan đồng nhất cho cả `#sri-markdown-content` và `.cas-msg-body` trong `.cas-messages-list`:
     - Bóc tách YAML frontmatter `--- ... ---` và dòng `> 📌 ...` thành thanh **Metadata Badges (`.udf-meta-strip`)** gọn đẹp.
     - Đóng khung từng mục `## 1..4` / `### 1..4` thành **Section Card (`.udf-section-card`)** có viền màu ngữ nghĩa, chữ `13px` (`Inter / system-ui`, `line-height: 1.62`), tiêu đề nổi bật.
     - Chuyển bảng Markdown `| ... |` thành **Bảng HTML (`.udf-table`)** căn cột chuẩn xác, tô nổi mã `Sub-Task` và `file:line`.
     - Chuyển khối code/diff thành **Code Box (`.udf-code-box`)** có thanh tiêu đề ngôn ngữ và tô màu dòng diff (`+` xanh lá, `-` đỏ, `@@` xanh cyan).

---

## 12. Giải Pháp Tối Ưu Duy Nhất & Code Demo Cho `TASK-CHAT-10`: Khắc Phục `@tk001sea_bot` Không Nhận Phản Hồi Trên Supergroup (`ForceReply` + Định Tuyến Ưu Tiên) & Chuẩn Hóa Định Dạng Tin Nhắn Telegram Gọn Đẹp Trong 1 Thẻ (`internal/webui/telegram_message_format.go`)

### 12.1. Kiến trúc Giải pháp Duy nhất & Tối ưu nhất (Single-Responsibility Principle — Giữ `telegram_bridge.go <= HEAD 3,297` dòng)
1. **Cơ chế Bắt Phản Hồi 100% Trên Supergroup (`ForceReply` + Định Tuyến Thông Minh Không Rơi Tin Nhắn):**
   - **Bắn `ForceReply` tự động mở ô Reply khi bấm `✏️ Phản hồi / Sửa Giải Pháp` (`REVISE_PROMPT`) hoặc `✏️ Sửa Plan Sub-Task` (`REVISE_SUB_PROMPT`):**
     - Khi anh bấm nút **`✏️ Phản hồi / Sửa Giải Pháp (#43)`**, `@tk001sea_bot` gửi ngay một tin nhắn ngắn gọn với **`reply_markup: {"force_reply": true, "selective": true, "input_field_placeholder": "Nhập nội dung cập nhật cho #43 Core-01..."}`**.
     - Nhờ `force_reply: true`, ứng dụng Telegram trên điện thoại/máy tính của anh **tự động mở sẵn thanh Reply trỏ thẳng vào tin nhắn của `@tk001sea_bot`**. Anh chỉ cần gõ nội dung phản hồi và nhấn Enter — tin nhắn được gắn `reply_to_message` tới `@tk001sea_bot` nên **vượt qua `can_read_all_group_messages: false` (Group Privacy Mode) 100%** mà không cần thao tác thủ công!
   - **Định tuyến mọi tin nhắn văn bản thường về đúng Task đang thao tác (`processIncomingTelegramCommandWithReply` tại `internal/webui/telegram_bridge.go:836-898`):**
     - Nâng cấp `stripBotUsernameFromCommand`: cắt bỏ `@tk001sea_bot` (hoặc `@<bot_username>`) ở bất kỳ vị trí nào trong câu (dù có `/` hay không).
     - Khi anh bấm `👁 Xem Plan (#43)` (`handleTelegramViewPlanCommand`), hệ thống cập nhật luôn `LastNotifiedTask = "Core-01"` và nếu `PendingRevisionTaskKey` đang kẹt ở task cũ (`cdc-01`) thì chuyển sang `"Core-01"` để không bao giờ sửa nhầm task cũ.
     - Trong `processIncomingTelegramCommandWithReply`, với **mọi tin nhắn không bắt đầu bằng `/`**:
       - Xác định Task đích theo thứ tự ưu tiên chặt chẽ:
         1. Mã Task ghi trực tiếp ở đầu câu (nếu có, ví dụ `#43 ...` hoặc `Core-01: ...`)
         2. `replyRef := extractTaskRefFromReplyContext(replyContext)` (tin nhắn của Bot mà anh vừa Reply / ForceReply!)
         3. `explicitTaskKey` (nếu nhắn trong Forum Sub-Topic của Task)
         4. `cfg.PendingRevisionTaskKey` (Task đang bật chế độ sửa)
         5. `cfg.LastNotifiedTask` (Task vừa hiển thị gần nhất trên Telegram)
       - Nếu nội dung là câu hỏi (`isTelegramTechnicalQuestion`) -> gọi `handleTelegramTechnicalQuestion` trả lời gọn trong 1 thẻ.
       - Ngược lại -> gọi ngay `handleTelegramReviseSolutionCommand` (hoặc `handleTelegramReviseSubTaskSolutionCommand`), cập nhật `09_tasks_solution.md` & `08_tasks.md` và **phản hồi ngay lập tức bằng Thẻ Giải Pháp S3 Đã Cập Nhật**! Tuyệt đối không bao giờ đánh rơi tin nhắn thành `IGNORED`.
   - **Sửa lỗi ghi đè `cfg` trong `ensureTelegramApprovePoller` (`telegram_bridge.go:2885-2890`):**
     - Chỉ lưu `LastUpdateID` trên bản `latestCfg := s.loadTelegramPersistedConfig()` mới nhất sau khi xử lý xong mỗi update, không dùng biến `cfg` cũ ở đầu vòng lặp ghi đè lên `PendingRevisionTaskKey` / `LastNotifiedTask`.

2. **Quy Chuẩn Định Dạng Tin Nhắn Telegram Gọn, Hợp Lý, Đủ Ý Trong 1 Thẻ Duy Nhất (`internal/webui/telegram_message_format.go`):**
   - Tách riêng module đơn trách nhiệm `internal/webui/telegram_message_format.go` để định dạng toàn bộ tin nhắn gửi qua Telegram theo chuẩn **`Telegram Compact Executive Card` (`<= 1,500 ký tự`, luôn nằm gọn trong 1 tin nhắn duy nhất, 0 mảnh cắt `(Phần 1/2)`, 0 bảng pipe `| ... |`, 0 đường dẫn `/Users/trainguyen/...`)**:
     - **Mẫu 1 — Danh sách `/task` (`formatTelegramCompactQueueMessage`):**
       Hiển thị tổng quan số lượng và danh sách từng Task gọn trên 2 dòng (`• *#43 Core-01* — ⏸ CHỜ DUYỆT (0/6 Sub-Tasks)` + tiêu đề), không xả toàn bộ hồ sơ S3 vào trong `/task`.
     - **Mẫu 2 — Hồ Sơ Giải Pháp S3 & Kết Quả Cập Nhật Phản Hồi (`formatTelegramCompactApprovalCard`):**
       Gồm đúng 4 khối tinh gọn: (1) `✏️ Phản hồi vừa cập nhật` (nếu có); (2) `🔍 1. Nguyên nhân chính` (2–3 bullet ngắn kèm basename `file:line`); (3) `💡 2. Giải pháp & Lộ trình Wave` (2–3 bullet giải pháp + gom Sub-Tasks theo `Wave 0..N` + tối đa 8 dòng code trọng tâm); (4) `📝 3. Điểm chạm & Thao tác` (chỉ liệt kê basename `file.go`).
     - **Mẫu 3 — Câu Trả Lời Hỏi-Đáp Kỹ Thuật (`formatTelegramCoArchitectReply`):**
       Tự động chuyển bảng Markdown `| ... |` và heading `##` từ `UDF-v1` thành danh sách bullet chuẩn Telegram ngắn gọn (`<= 1,500 ký tự`).

### 12.2. Code Demo Thực Tế (`internal/webui/telegram_message_format.go` & `internal/webui/telegram_bridge.go`)
```go
// internal/webui/telegram_message_format.go (Single-Responsibility Formatter & ForceReply Helper)
func (s *Server) sendTelegramPromptWithForceReply(ctx context.Context, cfg telegramPersistedConfig, taskKey string, text string, placeholder string) string {
	token := strings.TrimSpace(cfg.BotTokenEncrypted)
	chatID := strings.TrimSpace(cfg.ChatID)
	if token == "" || chatID == "" {
		return "QUEUED_LOCAL_BRIDGE"
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	var threadID int64
	if qt, err := s.findQueuedTask(ctx, taskKey); err == nil && qt != nil {
		threadID = s.ensureTelegramTaskTopic(ctx, cfg, *qt, false)
	}
	for _, useMarkdown := range []bool{true, false} {
		bodyMap := map[string]any{
			"chat_id": chatID,
			"text":    text,
			"reply_markup": map[string]any{
				"force_reply":             true,
				"selective":               true,
				"input_field_placeholder": truncateRunes(placeholder, 60),
			},
		}
		if threadID > 0 {
			bodyMap["message_thread_id"] = threadID
		}
		if useMarkdown {
			bodyMap["parse_mode"] = "Markdown"
		}
		rawBody, _ := json.Marshal(bodyMap)
		reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(rawBody))
		if err != nil {
			cancel()
			return "LOCAL_FALLBACK"
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			return "LOCAL_BRIDGE_LOGGED"
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		cancel()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return "SENT_TELEGRAM_API_OK"
		}
	}
	return "TELEGRAM_FORCE_REPLY_FALLBACK"
}
```

---

## 13. Giải Pháp Tối Ưu Duy Nhất & Code Demo Cho `TASK-CHAT-11`: Chuẩn Hóa Cổng `H4b (Propose-Only Approval Gate)` Trên `@tk001sea_bot` & Bộ Sinh `09_tasks_solution.md` / `08_tasks.md` Theo Chuẩn Engineering Manager / Tech Lead

### 13.1. Kiến trúc Giải pháp Duy nhất & Tối ưu nhất (4 Điểm Chạm Cốt Lõi — Giữ `NOW <= HEAD` trên toàn bộ file lớn)
1. **Chuẩn hóa Relative Path & Khử trùng lặp 100% danh sách File (`normalizeAndDeduplicateRepoRelativePaths` tại `internal/webui/telegram_message_format.go` & `internal/webui/server.go:2771-2808`):**
   - Trước khi đếm hoặc hiển thị danh sách `Files thay đổi`, chạy qua hàm `normalizeRepoRelativePath(rawPath, repoNames)`:
     - Cắt bỏ đường dẫn tuyệt đối `/Users/.../`, `./`, `a/`, `b/`, hậu tố `:line`, và tiền tố `<repoName>/` (ví dụ cắt `centralized-export-service/` khỏi `centralized-export-service/models/trans-his/trans-his.model.ts` -> `models/trans-his/trans-his.model.ts`).
     - Đưa vào `seen[strings.ToLower(cleanRel)]` để loại bỏ 100% file trùng lặp, đảm bảo số đếm (`6 files`) khớp tuyệt đối với danh sách file thực tế.
2. **Nhận diện động Tech Stack (`detectRepoTechStackAndDoD` tại `internal/webui/telegram_message_format.go` & `internal/webui/task_inspector.go:1133-1188`):**
   - Kiểm tra sự tồn tại của `package.json` / `tsconfig.json` vs `go.mod` trong `TargetRepo` (hoặc đuôi file `.ts`/`.js` vs `.go`):
     - **Repo Node.js / TypeScript (`package.json`):** `StackLabel = "Node.js / TypeScript"`, Code fence dùng ````typescript`, DoD Wave 0 = `` `npm run build` & pass unit test ``, DoD Wave 1 = ``Pass toàn bộ test suite (`npm test`), không phát sinh memory leak/race condition, đạt chuẩn G1–G8``.
     - **Repo Go (`go.mod`):** `StackLabel = "Go"`, Code fence dùng ````go`, DoD Wave 0 = `` `go build ./...` & pass unit test ``, DoD Wave 1 = ``Pass `go test -race ./...`, không phát sinh memory leak/race condition, đạt chuẩn G1–G8``.
3. **Phân bổ Wave chuẩn phụ thuộc `Wave 0 -> Wave 1` (`assignSequentialDependencyWaves` tại `internal/webui/task_inspector.go:1133-1188, 1270`):**
   - Trong `DecomposeMultiRepoCodebaseSubTasks`, Sub-Task `M01` (Foundation & Model) nhận `WaveIndex = rIdx * 2` (`Wave 0`), còn Sub-Task `M02` (Routing & Integration Test) nhận `WaveIndex = rIdx*2 + 1` (`Wave 1`) kèm `DependsOn: []string{id1}`.
   - Tự động nâng cấp các `subtasks.json` hiện có (như `TASK-13`) nếu phát hiện `M01` và `M02` đang bị xếp trùng vào `Wave 0` hoặc chứa DoD mẫu `Pass go test -race ./... (hoặc npm run build)`.
4. **Template Chuẩn Cổng `H4b` 4 Mục (1 tin nhắn Telegram duy nhất `< 4,096` ký tự, 0 Meta-System Noise, 0 cắt cụt `...` ở định danh) tại `internal/webui/telegram_message_format.go` & `internal/webui/unified_doc_format.go`:**
   - Loại bỏ toàn bộ câu chữ giải thích cơ chế hệ thống (*"Đã quét trực tiếp 347 file..."*, *"Mọi thay đổi ở bước S4 được cô lập..."*) và đường dẫn tuyệt đối `/Users/...`.
   - Không dùng `truncateRunes` cắt cụt `...` trên tên nhánh git (`branch`), đường dẫn file tương đối, và mô tả Sub-Task.
   - Render đúng 4 mục:
     - `🏗 1. KIẾN TRÚC & CHIẾN LƯỢC CHUYỂN ĐỔI (ARCHITECTURE PLAN)`: `Pattern áp dụng`, `Điểm chạm chính (Core Touchpoints)`, `Cơ chế Rollback / Toggle`.
     - `⚠️ 2. KIỂM SOÁT RỦI RO (RISK & GUARDRAILS)`: `Memory / Streaming`, `Tương thích Schema`.
     - `💻 3. KEY DIFF / CODE PREVIEW (<primaryFile>)`: Block code đúng ````typescript` / ````go` trình bày rõ hàm điều hướng/interface mà không nhồi God Class.
     - `📋 4. LỘ TRÌNH THỰC THI (<N> SUB-TASKS · <M> FILES)`: Nhóm theo `Wave 0 (Foundation & Model)` -> `Wave 1 (Routing & Integration)`, liệt kê `📁 Files thay đổi (<M>)` (relative path đã khử trùng) và `📄 Artifacts`, tiếp nối bằng `⚡ HÀNH ĐỘNG DUYỆT (CHỌN 1 TRONG 3)`.

### 13.2. Code Demo Thực Tế (`internal/webui/telegram_message_format.go` & `internal/webui/task_inspector.go`)
```go
// internal/webui/telegram_message_format.go — Chuẩn hóa Relative Path, Tech Stack & H4b Approval Gate Card
type RepoTechStackProfile struct {
	StackLabel string // e.g. "Node.js / TypeScript" or "Go"
	CodeLang   string // e.g. "typescript" or "go"
	Wave0DoD   string
	Wave1DoD   string
}

func detectRepoTechStackProfile(repoPath string, sampleFiles []string) RepoTechStackProfile {
	isTS := false
	if strings.TrimSpace(repoPath) != "" {
		if _, err := os.Stat(filepath.Join(repoPath, "package.json")); err == nil {
			isTS = true
		} else if _, err := os.Stat(filepath.Join(repoPath, "tsconfig.json")); err == nil {
			isTS = true
		}
	}
	for _, f := range sampleFiles {
		ext := strings.ToLower(filepath.Ext(strings.Split(strings.TrimSpace(f), ":")[0]))
		if ext == ".ts" || ext == ".tsx" || ext == ".js" {
			isTS = true
			break
		}
	}
	if isTS {
		return RepoTechStackProfile{
			StackLabel: "Node.js / TypeScript",
			CodeLang:   "typescript",
			Wave0DoD:   "`npm run build` & pass unit test",
			Wave1DoD:   "Pass toàn bộ test suite (`npm test`), không phát sinh memory leak/race condition, đạt chuẩn G1–G8",
		}
	}
	return RepoTechStackProfile{
		StackLabel: "Go",
		CodeLang:   "go",
		Wave0DoD:   "`go build ./...` & pass unit test",
		Wave1DoD:   "Pass `go test -race ./...`, không phát sinh memory leak/race condition, đạt chuẩn G1–G8",
	}
}

func normalizeAndDeduplicateRepoRelativePaths(rawPaths []string, repoNames []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, raw := range rawPaths {
		p := strings.TrimSpace(strings.Split(raw, ":")[0])
		p = strings.Trim(p, "`*\"'()")
		p = strings.TrimPrefix(strings.TrimPrefix(p, "b/"), "a/")
		p = strings.TrimPrefix(p, "./")
		if idx := strings.LastIndex(p, "/worktrees/"); idx >= 0 {
			rest := p[idx+len("/worktrees/"):]
			parts := strings.SplitN(rest, "/", 3)
			if len(parts) == 3 {
				p = parts[2]
			}
		}
		for _, rn := range repoNames {
			rnClean := strings.TrimSpace(rn)
			if rnClean != "" && strings.HasPrefix(p, rnClean+"/") {
				p = strings.TrimPrefix(p, rnClean+"/")
			}
		}
		p = strings.TrimSpace(strings.TrimPrefix(p, "/"))
		if p == "" || !strings.Contains(filepath.Base(p), ".") {
			continue
		}
		key := strings.ToLower(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}
```

---

## 14. Giải Pháp Tự Chữa Lành `S5 -> S3 -> S4 -> S5` (`TASK-CHAT-12`) — Phân Tầng Đúng Design Pattern Gốc (`internal/gitclient`, `internal/notify`, `internal/webui`), Khóa `NOW <= HEAD` 100%, Xóa Token Leak & Xóa `httptest` Mockup

### 14.1. Chẩn đoán 4 lỗ hổng phát hiện khi Audit Phản Biện `S5`
1. **Lỗ hổng 1 (Lộ URL Bot Token & Lệch `NOW <= HEAD` ở 4 file tracked):**
   - `README_START.md:306-309` bị nối thêm 3 dòng chứa URL Telegram Bot Token (`HEAD=306, NOW=309, DIFF=+3`).
   - `internal/webui/server_test.go` tăng từ `HEAD=2,927` lên `NOW=2,930` (`+3` dòng) nhưng bảng trong `audit_report` vẫn ghi số cũ `2,927 (0 dòng)`.
   - `internal/webui/subtask_git_apply.go` tăng từ `HEAD=312` lên `NOW=344` (`+32` dòng) do đặt `checkGoPackageCompilesOrRevertRelFile` trực tiếp trong `subtask_git_apply.go`.
   - `internal/runner/antigravity_test.go` tăng từ `HEAD=90` lên `NOW=106` (`+16` dòng).
2. **Lỗ hổng 2 (Đặt sai Layer theo Design Pattern gốc & Trùng lặp hàm giữa 2 file 500 dòng):**
   - `checkGoPackageCompilesOrRevertRelFile` là logic kiểm định biên dịch sau khi `git apply` -> phải nằm ở layer **`internal/gitclient/patch_guard.go`**, không phải `internal/webui`.
   - `convertTelegramMarkdownToHTML`, `formatInlineTelegramHTML`, `escapeTelegramHTML`, `stripBotMentionAnywhere` là bộ chuyển đổi định dạng thông báo Telegram -> phải nằm ở layer **`internal/notify/telegram_format.go`**, không phải `internal/webui`.
   - Các hàm normalize path & detect tech-stack (`normalizeOrchestratorDisplayPath`, `normalizeAndDeduplicateRepoRelativePaths`, `detectRepoTechStackProfile`, `detectTargetRepoLanguageAndDoD`, `detectCodeFenceLang`) bị viết lặp ở cả `task_llm_bridge.go` (`546` dòng) và `telegram_message_format.go` (`486` dòng) -> phải gom về 1 module đơn trách nhiệm duy nhất **`internal/webui/repo_tech_profile.go`**, và tách luồng `/revise` sang **`internal/webui/task_revise_workflow.go`**.
   - Trong `internal/webui/task_inspector.go` và `internal/webui/telegram_bridge.go` vẫn còn sót các hàm helper mới -> phải chuyển sang `internal/webui/task_planner_multilayer.go` và `internal/webui/telegram_forum_topic.go`.
3. **Lỗ hổng 3 (Câu mẫu mâu thuẫn `Knex/PG Pool` & Cột `Điểm chạm` chưa cắt prefix repo):**
   - `unified_doc_format.go:232, 258-260` và `telegram_message_format.go:255-261` vẫn còn câu chữ `"Khởi tạo Knex/PG Pool"` mâu thuẫn với Code Demo chỉ dùng `pgPool` thuần trong `PostgresDB` (`v2`), và cột `Điểm chạm (file:line)` trong bảng Markdown chưa cắt prefix `<repoName>/`.
4. **Lỗ hổng 4 (Còn dùng `httptest.NewServer` trong `telegram_message_format_test.go`):**
   - `telegram_message_format_test.go:39` dùng `httptest.NewServer(srv.Handler())` thay vì gọi trực tiếp các phương thức xử lý thật trên `Server` + kiểm chứng trực tiếp trên repo thật `/Users/trainguyen/Documents/work/data-hub/centralized-export-service` và `.orchestrator/tasks/TASK-13`.

### 14.2. Code Demo Kiến Trúc Phân Tầng (`internal/gitclient/patch_guard.go` & `internal/notify/telegram_format.go`)
```go
// internal/gitclient/patch_guard.go
package gitclient

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CheckGoPackageCompilesOrRevertRelFile ensures applying a .go patch hunk never leaves the package uncompilable.
func CheckGoPackageCompilesOrRevertRelFile(ctx context.Context, repoDir, relPath string) bool {
	if !strings.HasSuffix(relPath, ".go") || strings.HasSuffix(relPath, "_test.go") {
		return true
	}
	if _, err := os.Stat(filepath.Join(repoDir, "go.mod")); err != nil {
		return true
	}
	pkgDir := "./" + strings.TrimPrefix(filepath.ToSlash(filepath.Dir(relPath)), "./")
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "go", "build", pkgDir)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return true
	}
	if strings.Contains(string(out), filepath.ToSlash(relPath)+":") || strings.Contains(string(out), filepath.Base(relPath)+":") {
		revCtx, revCancel := context.WithTimeout(ctx, 5*time.Second)
		defer revCancel()
		_ = exec.CommandContext(revCtx, "git", "-C", repoDir, "checkout", "--", relPath).Run()
		return false
	}
	return true
}
```

---

## 15. Giải Pháp Tối Ưu Duy Nhất (`TASK-CHAT-13`) — Phản Hồi Tức Thì Khi Gửi Định Hướng Trên CMS Web (`#co-architect-studio`) & Chấm Dứt 100% Hiện Tượng Phải Bấm `F5`

### 15.1. Kiến trúc giải pháp đồng bộ Frontend + Backend (4 Chốt Chặn)
1. **Chốt chặn 1 — Optimistic UI + Thinking Bubble + In-Flight SSE Shield + `Ctrl+Enter` (`internal/webui/static/app.js:1791-1892`):**
   - Ngay khi bấm **`💬 Gửi Định Hướng`** hoặc **`🔄 Tái Lập Giải Pháp S3`** (`sendCoArchitectChatMessage`):
     - Khởi tạo `window._casInFlightChat = { taskKey, userEntry, thinkingEntry }` và đẩy ngay 2 bong bóng (`userEntry` + `thinkingEntry`: `"⏳ BRAIN (Co-Architect) đang phân tích mã nguồn & tổng hợp phản hồi..."`) vào `dashboardState.active_task_chat`, rồi gọi `renderCoArchitectStudio()` ở **`0ms`**.
     - Trong `renderCoArchitectStudio()`, nếu `window._casInFlightChat` đang active cho `activeTask.task_key` và danh sách `chatHistory` từ `dashboardState.active_task_chat` (do SSE `fetchDashboard()` nạp về) chưa có phản hồi mới sau thời điểm gửi, tự động nối `userEntry` (nếu đĩa chưa kịp trả về) và `thinkingEntry` vào cuối danh sách hiển thị -> **SSE nền không bao giờ xóa mất tin nhắn anh vừa gõ!**
     - Khóa tạm thời nút `.btn-cas-send` / `.btn-cas-revise` (hiển thị `"⏳ Đang xử lý..."`) trong khi chờ phản hồi, và gắn listener `Ctrl+Enter` / `Cmd+Enter` cho `#inp-co-architect-msg`.
2. **Chốt chặn 2 — Post-UDF Auto-Scroll (`internal/webui/static/unified_doc_format.js:348-385`):**
   - Trong `wrappedCas` (`unified_doc_format.js:376-385`), ngay sau khi `applyUnifiedFormatToAllViews()` biến đổi `<pre class="cas-msg-body">` thành `.udf-section-card`, thực hiện cuộn lại `#co-architect-chat-messages` (`listEl.scrollTop = listEl.scrollHeight`) để bong bóng tin nhắn mới và khối `📌 Tóm tắt & Trả lời trực tiếp (Summary)` luôn nằm ngay trước mắt.
3. **Chốt chặn 3 — Lưu `userEntry` tức thì xuống `chat_history.json` + Broadcast SSE 2 chiều (`internal/webui/task_chat.go:805-925`):**
   - Tại `handleTaskPlanningChat` (`task_chat.go:807`): Lưu ngay `s.saveTaskChatHistory(selected.TaskKey, history)` và phát `s.broadcast("update", ...)` **ngay trước khi** gọi LLM.
   - Sau khi LLM trả lời và lưu `agentEntry` (`task_chat.go:913`), phát tiếp `s.broadcast("update", ...)` để mọi client SSE tự đồng bộ ngay mà không cần `F5`.
4. **Chốt chặn 4 — Loại bỏ Chặn Kép 2 Vòng LLM (`88.74s -> ~12-25s`) & Đồng bộ `Summary` trên Telegram + Restart `:8090` (`internal/webui/task_revise_workflow.go:118-136` & `internal/webui/telegram_bridge.go:2052-2085`):**
   - Trong `syncArchitectLLMAnalysisToMasterPlan` (`task_revise_workflow.go:125`): Đã có sẵn `analysis` từ lần gọi `invokeArchitectLLM` đầu tiên, chỉ cập nhật trực tiếp `09_tasks_solution.md` bằng `BuildUnifiedS3SolutionDoc` (tuyệt đối không gọi đồng bộ `generateDeepTaskArtifacts` lần thứ 2 trên luồng HTTP chat gây tốn thêm `48s`).
   - Đảm bảo `handleTelegramReviseSolutionCommand` luôn trả về khối `📌 Tóm tắt & Trả lời trực tiếp (Summary)` và restart tiến trình `:8090` để chạy bản binary mới nhất.

### 15.2. Concrete Code Demo (`internal/webui/static/app.js`, `internal/webui/static/unified_doc_format.js`, `internal/webui/task_chat.go`, `internal/webui/task_revise_workflow.go`)

```javascript
// 1. internal/webui/static/app.js (Optimistic UI + Thinking Bubble + SSE Shield + Ctrl+Enter)
window._casInFlightChat = null;

function bindCoArchitectInputShortcuts() {
  const ta = document.getElementById("inp-co-architect-msg");
  if (!ta || ta.dataset.shortcutBound === "1") return;
  ta.dataset.shortcutBound = "1";
  ta.addEventListener("keydown", (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      sendCoArchitectChatMessage(e);
    }
  });
}

function getEffectiveCoArchitectMessages() {
  const base = Array.isArray(dashboardState?.active_task_chat)
    ? [...dashboardState.active_task_chat]
    : [];
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";
  const pending = window._casInFlightChat;
  if (pending && pending.taskKey === activeKey) {
    const hasUser = base.some(
      (m) => m.sender === "user" && m.message === pending.userEntry.message
    );
    if (!hasUser) base.push(pending.userEntry);
    base.push(pending.thinkingEntry);
  }
  return base;
}

async function sendCoArchitectChatMessage(ev, overrideMode) {
  if (ev && ev.preventDefault) ev.preventDefault();
  const ta = document.getElementById("inp-co-architect-msg");
  if (!ta || !ta.value.trim()) return;
  const msg = ta.value.trim();
  const role = document.getElementById("cas-role-select")?.value || "BRAIN";
  const model = document.getElementById("cas-model-select")?.value || "gemini-4-argon (antigravity)";
  const actionMode = overrideMode || document.getElementById("cas-mode-select")?.value || "discuss";
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";
  const nowTime = new Date().toTimeString().slice(0, 8);

  window._casInFlightChat = {
    taskKey: activeKey,
    userEntry: {
      id: "msg-u-opt-" + Date.now(),
      task_key: activeKey,
      sender: "user",
      agent_role: "👤 You (Product Owner & Co-Architect)",
      agent_slot: "USER",
      skill: "Steering",
      model: model,
      action_mode: actionMode,
      message: msg,
      timestamp: nowTime,
    },
    thinkingEntry: {
      id: "msg-a-thinking-" + Date.now(),
      task_key: activeKey,
      sender: "agent",
      agent_role: "🧠 " + role + " (Đang xử lý...)",
      agent_slot: role,
      skill: "/brain-delegate",
      model: model,
      action_mode: actionMode,
      message: "📌 Tóm tắt & Trả lời trực tiếp (Summary):\n- ⏳ Đang phân tích mã nguồn, lịch sử thực thi & tổng hợp phản hồi cho định hướng của bạn...",
      timestamp: nowTime,
    },
  };

  ta.value = "";
  renderCoArchitectStudio();

  const sendBtn = document.querySelector(".btn-cas-send");
  const reviseBtn = document.querySelector(".btn-cas-revise");
  if (sendBtn) { sendBtn.disabled = true; sendBtn.textContent = "⏳ Đang xử lý..."; }
  if (reviseBtn) { reviseBtn.disabled = true; }

  try {
    const res = await fetch("/api/tasks/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        task_key: activeKey,
        message: msg,
        action_mode: actionMode,
        agent_role: role,
        model: model,
      }),
    });
    if (!res.ok) {
      alert("Không thể gửi tin nhắn Co-Architect: " + (await res.text()));
      return;
    }
    const data = await res.json();
    window._casInFlightChat = null;
    dashboardState = data;
    if (actionMode === "revise_solution") {
      selectedReportTab = "S3";
      userLockedReportTab = true;
    }
    renderAll();
  } catch (e) {
    console.error("sendCoArchitectChatMessage error:", e);
    await fetchDashboard(activeKey);
  } finally {
    window._casInFlightChat = null;
    if (sendBtn) { sendBtn.disabled = false; sendBtn.textContent = "💬 Gửi Định Hướng"; }
    if (reviseBtn) { reviseBtn.disabled = false; }
    renderCoArchitectStudio();
  }
}
```

```go
// 2. internal/webui/task_chat.go:795-805 (Persist userEntry xuống chat_history.json & Broadcast SSE ngay trước khi gọi LLM)
	history = append(history, userEntry)
	_ = s.saveTaskChatHistory(qt.ExternalID, history)
	s.broadcast("task_chat_user_sent")
```

```go
// 3. internal/webui/task_revise_workflow.go:115-130 (Không gọi lặp generateDeepTaskArtifacts khi analysis đã có sẵn để triệt tiêu chặn kép 88.74s)
func (s *Server) syncArchitectLLMAnalysisToMasterPlan(
	ctx context.Context,
	qt *queue.QueuedTask,
	directive string,
	runnerName string,
	selectedModel string,
	affectedRepos []string,
	subTasks []SubTaskDTO,
	analysis *ArchitectLLMAnalysis,
) ([]SubTaskDTO, string) {
	if analysis != nil {
		s.saveTaskLLMAnalysis(qt.ExternalID, analysis)
		subTasks = s.applyLLMSubTaskSpecs(*qt, subTasks, analysis)
	}
	subTasks = s.enrichSubTasksWithRoleAndModel(subTasks)
	subTasks = s.populateSubTaskFractalContractsAndArtifacts(*qt, subTasks)
	_ = s.saveTaskSubTasks(qt.ExternalID, subTasks)
	s.syncSubTasksToMarkdown08(*qt, subTasks)
	if analysis == nil {
		_, _, _, _ = s.generateDeepTaskArtifacts(ctx, *qt, runnerName, selectedModel)
	}
```

---

## 16. Giải Pháp Tối Ưu Duy Nhất (`TASK-CHAT-13` & `TASK-CHAT-14` — Vòng Lặp Tự Chữa Lành `S5 -> S3 -> S4 -> S5`) — Full Height `flex: 1 1 0; height: 0; overflow-y: auto` Giữ Nguyên Thanh Cuộn Cho `#sri-markdown-content` & Triệt Tiêu Treo `97s` Trong `BuildDashboardResponse` / `renderContextualDiscussReply`

### 16.1. Thiết kế khắc phục triệt để tại `app.css`, `unified_doc_format.css`, `task_inspector.go`, `task_llm_bridge.go`
1. **Trên cột trái `.sri-markdown-pane` & `#sri-markdown-content` (`internal/webui/static/app.css:1831-1927` & `internal/webui/static/unified_doc_format.css:6-20`):**
   - Dùng **`flex: 1 1 0 !important; height: 0 !important; min-height: 465px !important; overflow-y: auto !important;`** trên `#sri-markdown-content` và `#sri-markdown-content.udf-doc-root` (tách riêng khỏi `.cas-msg-body.udf-doc-root`). Nhờ `flex-basis: 0; height: 0`, `#sri-markdown-content` **giữ nguyên thanh cuộn dọc (`overflow-y: auto`)** không làm bung chiều cao `.sri-body` lên 3,000px, đồng thời `flex-grow: 1` tự động dãn `#sri-markdown-content` khớp khít **100% chiều cao của box parent `.sri-markdown-pane`** (theo chiều cao của cột phải `.sri-side-pane`).
   - Khôi phục `max-height: 240px; overflow-y: auto;` trên `.sri-subtasks-list` và `max-height: 180px; overflow-y: auto;` trên `.sri-code-block`.
2. **Chấm dứt treo `97s` khi gửi chat trên CMS Web (`internal/webui/task_inspector.go:3364-3395` & `internal/webui/task_llm_bridge.go:92-199`):**
   - Trong `reconcileAllQueuedTasks` (`task_inspector.go:3367`), xóa điều kiện `!strings.Contains(planText, "Bảng Tóm Tắt Điều Hành")` khiến mọi lần `BuildDashboardResponse` đều gọi lại `generateDeepTaskArtifacts` (`48s`).
   - Trong `invokeArchitectLLM` & `synthesizeDirectAnswerSummary` (`task_llm_bridge.go:92-199`), tổng hợp trực tiếp trạng thái thực thi thật (`Stage`, `Sub-Tasks DONE/PENDING`, `diff.patch`, `audit_report.md`) để phản hồi ngay lập tức (`< 1–3s`).

### 16.2. Concrete Code Demo (`internal/webui/static/app.css` & `internal/webui/static/unified_doc_format.css`)

```css
/* 1. internal/webui/static/app.css:1859-1877 */
.sri-pre-block, #sri-markdown-content {
  font-family: "JetBrains Mono", Menlo, monospace;
  font-size: 10.5px;
  line-height: 1.45;
  color: #e2e8f0;
  background: #05080e;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 5px;
  padding: 8px 10px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  flex: 1 1 0;
  height: 0;
  min-height: 465px;
  overflow-y: auto;
  overscroll-behavior: contain;
}
```

```css
/* 2. internal/webui/static/unified_doc_format.css:6-24 */
#sri-markdown-content.udf-doc-root,
.cas-msg-body.udf-doc-root {
  font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif !important;
  font-size: 13px !important;
  line-height: 1.62 !important;
  color: #e2e8f0 !important;
  white-space: normal !important;
  word-break: break-word;
  padding: 14px 16px !important;
  background: #070d19 !important;
}
#sri-markdown-content.udf-doc-root {
  flex: 1 1 0 !important;
  height: 0 !important;
  min-height: 465px !important;
  overflow-y: auto !important;
}
```

---

## 17. Báo Cáo Kiểm Toán Thực Trạng Đấu Nối AI Model & Giải Pháp Xóa Bỏ 5 Chỗ Làm Giả Model (`TASK-CHAT-15`)

### 17.1. Thực trạng: Đã gắn AI Model ở đâu và đang làm giả/chặn Model ở 5 chỗ nào?
1. **Những chỗ ĐÃ gắn `AntigravityRunner.Complete` thực sự (chỉ 2 hàm):**
   - `invokeArchitectLLM` (`internal/webui/task_llm_bridge.go:212-218`) — gọi CLI `antigravity` cho `BRAIN`, **NHƯNG đang bị chặn ở chế độ `discuss`** (`line 185`).
   - `invokeMuscleSubTaskCodeGen` (`internal/webui/subtask_codegen_muscle.go:265-281`) — gọi CLI `antigravity` cho `MUSCLE` ở bước `S4` khi Sub-Task chưa có diff.
2. **5 chỗ đang CHẶN hoặc GHI ĐÈ AI Model bằng văn mẫu ghép chuỗi tĩnh (khiến chat `Core-01` lúc `16:08:25` và `16:11:57` xổ văn mẫu láo):**
   - **Chỗ 1 — Chặn 100% AI Model khi chat `discuss` (`internal/webui/task_llm_bridge.go:185`):**
     - Dòng `185`: `if actionMode != "discuss" && !isTestTaskOrTempRoot(s.Root, qt.ExternalID) && !strings.HasSuffix(os.Args[0], ".test")` -> khi bấm **`💬 Gửi Định Hướng Cho Agent`** (`action_mode = "discuss"`), điều kiện này bằng `false`, bỏ qua hoàn toàn AI Model và nhảy thẳng xuống dòng `244` gọi `synthesizeSymbolGroundedArchitectAnalysis` (`task_llm_bridge.go:308-352`) ghép chuỗi từ `subTasks` + bốc 2 file đầu tiên quét được (`cmd/orchestrator/agent_cmd.go:18`, `internal/gitclient/gitclient.go:14`).
   - **Chỗ 2 — Lấy câu mẫu tĩnh ghi đè lên kết quả của AI Model & Không đọc `git diff` thật (`internal/webui/task_chat_knowledge.go:38-46` & `internal/webui/task_llm_bridge.go:110-156`):**
     - Tại `task_chat_knowledge.go:38-46`, hàm `renderContextualDiscussReply` lấy `fbRC`, `fbSol` (cạo từ file markdown cũ) và gọi `synthesizeProgressAwareDirectAnswer` (`task_llm_bridge.go:150`: `"Dạ anh, đối với yêu cầu %q trên task %s: Nguyên nhân chính là %s; giải pháp thực thi chuẩn là %s"`) **ghi đè** lên `DirectAnswer`, `RootCauses`, `Solutions` của Model. Đồng thời hoàn toàn không chạy `git status --short` hay `git diff --stat` trên repo thật để đưa vào prompt.
   - **Chỗ 3 — Hardcode 6 Sub-Tasks của `Core-01` trong Planner (`internal/webui/task_inspector.go:980-1012, 2858-2868, 3059-3061`, `internal/webui/task_planner_multilayer.go:12-172`, `internal/webui/task_revise_workflow.go:75-107`):**
     - `isSelfUpgradeOrchestratorTask` (`task_inspector.go:980-995`) bắt từ khóa `"ai-orchestrator"`, `"core-"`, `"api"`, `"ux"`, `"ui"` và gọi `decomposeSelfUpgradeOrchestratorSubTasks` (`task_planner_multilayer.go:12-150`) trả về 6 Sub-Tasks tĩnh (`Tầng 1..Tầng 6` của `Core-01`). Sau đó `applyLLMSubTaskSpecs` (`task_revise_workflow.go:75-107`) chỉ ghi đè 2 Sub-Task đầu tiên (`i < len(analysis.SubTasks)`), để nguyên 4 Sub-Tasks `S03..S06` hardcode của `Core-01` (như đã xảy ra ở `.orchestrator/tasks/core-003/01_requirements.md:74-79`).
   - **Chỗ 4 — Phản hồi sửa Plan / Sub-Task trên Telegram chưa gọi AI Model (`internal/webui/telegram_bridge.go:1814-1875, 2045, 2101-2175`):**
     - `handleTelegramReviseSolutionCommand` (`telegram_bridge.go:2045`) gọi `applyTelegramSolutionFeedbackToPlan` (`telegram_bridge.go:2101-2175`) chỉ nối chuỗi tĩnh `## ✏️ Phản hồi & Điều chỉnh Giải pháp từ Telegram` vào cuối `09_tasks_solution.md`, và `handleTelegramReviseSubTaskSolutionCommand` (`telegram_bridge.go:1814-1875`) chỉ nối chuỗi tĩnh vào `subtasks/<SubID>/09_tasks_solution.md` mà không gọi `invokeArchitectLLM`.
   - **Chỗ 5 — `BuildUnifiedS2AnalysisDoc` & `BuildUnifiedDG0CritiqueDoc` (`internal/webui/unified_doc_format.go:99-204`) vẫn ghép bảng tĩnh:**
     - Chưa lấy nội dung phân tích từ `ArchitectLLMAnalysis` (`llm_analysis.json`).

### 17.2. Code Demo Xóa Bỏ Bypass & Đấu Nối Thật AI Model (`internal/webui/task_llm_bridge.go` & `internal/webui/task_chat_knowledge.go`)
```go
// 1. internal/webui/task_llm_bridge.go:185 — Gọi AI Model cho MỌI actionMode (kể cả "discuss") kèm git status/diff thật
if !isTestTaskOrTempRoot(s.Root, qt.ExternalID) && !strings.HasSuffix(os.Args[0], ".test") {
	callCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	gitSnapshot := s.collectRealGitDiffSnapshotForPrompt(callCtx, affectedRepos)
	prompt := fmt.Sprintf(
		"You are %s (%s). Return ONLY valid JSON matching schema: "+
			"{\"direct_answer\":\"...\",\"root_causes\":[\"...\"],\"solutions\":[\"...\"],\"code_demo\":\"...\",\"code_fence_lang\":\"%s\",\"sub_tasks\":[{\"title\":\"...\",\"desc\":\"...\",\"target_repo\":\"...\",\"file_touchpoint\":\"...\"}]}\n"+
			"Task %s: %s\nStage: %s | Mode: %s\nUser Question/Directive: %s\nReal Git Status & Diff:\n%s\nVerified Repo Touchpoints:\n%s\n",
		roleName, selectedModel, fenceLang, qt.ExternalID, qt.Summary, qt.Stage, actionMode, userPrompt,
		gitSnapshot, strings.Join(hitContextLines, "\n"),
	)
	agRunner := runner.NewAntigravityRunner(modelName)
	agRunner.WorkDir = os.TempDir()
	out, err := agRunner.Complete(callCtx, prompt)
	if err == nil {
		if parsed, ok := parseArchitectLLMAnalysisJSON(out, fenceLang); ok {
			return parsed
		}
	}
}
```

```go
// 2. internal/webui/task_chat_knowledge.go:30-55 — Giữ nguyên 100% câu trả lời từ AI Model, KHÔNG lấy fbRC/fbSol hay synthesizeProgressAwareDirectAnswer ghi đè
llmRes := s.invokeArchitectLLM(rCtx, qt, "BRAIN", qt.AgentModel, actionMode, cleanMsg, affectedRepos, subTasks)
if llmRes != nil {
	directAnswer = strings.TrimSpace(llmRes.DirectAnswer)
	rootCauses = llmRes.RootCauses
	solutions = llmRes.Solutions
	if strings.TrimSpace(llmRes.CodeDemo) != "" {
		fence := strings.TrimSpace(llmRes.CodeFenceLang)
		if fence == "" {
			fence = detectCodeFenceLang(affectedRepos, nil)
		}
		codeDemo = fmt.Sprintf("```%s\n%s\n```", fence, strings.TrimSpace(llmRes.CodeDemo))
	}
}
```
