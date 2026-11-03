---
tier: T2
suffix: audit_core_workflow_ex002_and_activity_log
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-27T14:23:15+07:00
status: PROPOSED
---

# 09. Technical Solution & Code Demo (`S3`) — Triệt Tiêu Toàn Bộ Điểm "Làm Láo" Trong Core Workflow (`S0 -> S6`) & Khôi Phục `Activity-Log Progress` (WebUI + Telegram)

> **Nguyên tắc Thiết kế:** Chỉ trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT**, đặt đúng layer kiến trúc gốc (`internal/agentgate`, `internal/webui`, `internal/notify`), tách file theo đúng trách nhiệm đơn (`Single-Responsibility`, giữ `NOW <= HEAD` trên mọi file lớn), và giữ kỷ luật **Propose-Only** chờ User `APPROVE`.

---

## 1. Kiến Trúc Giải Pháp Duy Nhất Cho 4 Nhóm Lỗi Cốt Lõi (`TASK-CW-01 .. TASK-CW-04`)

### 1.1. `TASK-CW-01` — Xóa sổ 100% bẫy từ khóa `V1/V2 -> PostgresDB` & Đấu nối `invokeArchitectLLM` thật vào Bộ Sinh Plan Tổng (`S1/S2/S3`)
- **Điểm chạm cần sửa:**
  - `internal/webui/task_revise_workflow.go:53-81`: Xóa bỏ `isDualVersionV1V2Directive` và xóa nhánh ép cứng `PostgreSQL V2` trong `buildSubTaskModuleTitleAndDesc`.
  - `internal/webui/task_llm_bridge.go:16-122, 144-210`:
    1. Mở rộng `ArchitectLLMAnalysis` thêm trường `SubTasks []ArchitectSubTaskSpec` (`json:"sub_tasks"`).
    2. Xóa bỏ hoàn toàn nhánh `if isDualVersionV1V2Directive(combined) || isPostgresOrDualSourceMigrationTask(combined)` trong `synthesizeDirectAnswerSummary` (`L27-29`) và `synthesizeSymbolGroundedArchitectAnalysis` (`L164-210`).
    3. Xóa câu chỉ thị cứng nhắc tới `TASK-13 (centralized-export-service) ... PostgresDB` khỏi `sysPrompt` (`L84`), thay bằng quy tắc tổng quát: *"Bám sát 100% công nghệ đích mà User yêu cầu trong Task Description / User Prompt (ví dụ: nếu User yêu cầu thêm V3 ClickHouse song song V1 Mongo và V2 Postgres thì giữ nguyên V1/V2 và thiết kế mới adapter + router cho V3 ClickHouse, tuyệt đối không lấy V2 Postgres làm mục tiêu triển khai mới)."*
  - `internal/webui/telegram_message_format.go:135-161`: Xóa `isPostgresOrDualSourceMigrationTask` và đoạn code cứng `export class PostgresDB` trong `buildCleanKeyDiffPreview`.
  - `internal/webui/task_inspector.go:1089-1198, 2984-2989, 3078-3294`:
    1. Trong `inspectTargetRepoSymbols` (`L2984-2989`), gỡ bỏ điểm cộng thiên vị `postgres` tĩnh; thay vào đó trích xuất các thực thể công nghệ mục tiêu từ `qt.Title + " " + qt.Description` (`extractPromptTargetTechKeywords`, ưu tiên thực thể đi sau cụm *"làm thêm"*, *"triển khai"*, *"v3"*, *"clickhouse"*). Khi file adapter của công nghệ mới (VD: `src/common/adapters/database/dbHandler/clickhouse.ts`) chưa tồn tại trên đĩa, tự động đề xuất đường dẫn tạo mới chuẩn theo cấu trúc thư mục adapter hiện có (`src/common/adapters/database/dbHandler/<tech>.ts`)!
    2. Trong `generateDeepTaskArtifacts` (`L3078-3258`), gọi trực tiếp `s.invokeArchitectLLM(ctx, qt, subTasks, qt.Description, selectedModel, "plan")` để lấy `DirectAnswer`, `RootCauses`, `Solutions`, `CodeDemo` và `SubTasks` thật từ Model (`gemini-4-argon`) ghi vào `01_requirements.md`, `08_tasks.md`, `09_tasks_solution.md` và `subtasks.json`.

### 1.2. `TASK-CW-02` — Chấm dứt "Làm Láo" ở `S2.5 (DG0)` & `BuildAndRunCalibratedLayaGate (DG0/DG1/DG2/DG3)` theo chuẩn 100% Agnostic Core (Không dùng danh sách từ khóa cheat)
- **Điểm chạm cần sửa:**
  - `internal/agentgate/kernel_bridge.go:637-923` (`BuildAndRunCalibratedLayaGate`):
    1. **Cấm tự lấy `pickLine(0)[:80]` của chính file vừa sinh làm `Claim == Evidence == MustContain`!**
    2. **Cấm dùng mảng từ khóa công nghệ hardcode (`"clickhouse"`, `"postgres"`, `"mongodb"`, v.v.)!**
    3. Thay bằng cơ chế **100% Agnostic Cross-Document Ground-Truth Verification**:
       - Đọc yêu cầu gốc của User từ `01_requirements.md` (`extractUserRequirementCoreLines`) làm `Requirement Ground Truth`.
       - Đọc nội dung giải pháp/phản biện từ `refFilePath` (`10_preplan_critique.md`, `09_tasks_solution.md`, `audit_report.md`) và mã nguồn thật tại các đường dẫn `REF: <path>:<start>-<end>` được trích dẫn trong tài liệu.
       - Gửi cặp `(Claim = Giải pháp/Nhận định trong Plan S3)` vs `(Evidence = Yêu cầu gốc trong 01_requirements.md + Code Excerpt thật trên đĩa)` trực tiếp sang Neural Engine `convaiinnovations/laya (:8200 /v1/predict)` qua `laya_gate.py` (3 vòng `r1/r2/r3` kèm `control_true` và `control_false`).
       - Cắt chuỗi an toàn theo `[]rune` (`truncateUTF8Runes(s, 160)`) thay vì cắt byte `trimmed[:80]` để không bao giờ làm hỏng UTF-8 tiếng Việt.
    4. Tại `S0` (`kernel_bridge.go:150-175`), nếu `checkLayaHealthHTTP` trả về `false`, tự động gọi `/Users/trainguyen/Documents/work/agent/hooks/agy/ensure_laya_up.sh` để khởi động Laya `:8200`.

### 1.3. `TASK-CW-03` — Khôi phục `Activity-Log Progress` Thời gian thực trên cả WebUI lẫn Telegram Forum Sub-Topic
- **Điểm chạm cần sửa:**
  - `internal/webui/server.go:154-166, 932-968, 1122-1167` & `internal/webui/telegram_bridge.go:2674-2771`:
    1. Giữ lịch sử các bước thực thi trong `s.liveStepProgress` thay vì xóa trắng ngay lập tức bằng `clearPacedWorkflowStep`.
    2. Đưa khối **📟 Live Activity-Log & Step Progress Stream** lên đầu màn hình `screen-autopilot` (`internal/webui/static/index.html`) và đồng bộ live-update vào Forum Sub-Topic trên Telegram.

### 1.4. `TASK-CW-04` — Xóa Sổ Toàn Bộ 10 Ổ "Cheat Code" Khỏi Source Code `ai-orchestrator`
1. `internal/webui/task_llm_bridge.go`: Xóa sạch `isDualVersionV1V2Directive`, `isPostgresOrDualSourceMigrationTask`, `"TASK-13"`, `"PostgresDB"`, `"TransHisModel"`, `"pgPool"`, `"knexPg"` tại dòng `27-29`, `84`, `150-210`.
2. `internal/webui/task_revise_workflow.go`: Xóa sạch `isMasterMongoDestinationTask` (`L47-51`), `isDualVersionV1V2Directive` (`L53-59`), và nhánh hardcode `PostgreSQL V2` trong `buildSubTaskModuleTitleAndDesc` (`L66-73`).
3. `internal/webui/telegram_message_format.go` & `telegram_subtask_card.go`: Xóa sạch `isPostgresOrDualSourceMigrationTask` (`L13-20`), đoạn code cứng `export class PostgresDB` trong `buildCleanKeyDiffPreview` (`L22-40`), và mọi nhánh `if isPostgresOrDualSourceMigrationTask` (`L106-186`, `telegram_subtask_card.go:337`).
4. `internal/webui/unified_doc_format.go`: Xóa sạch nhánh `if isPostgresOrDualSourceMigrationTask` (`L256-265`) và check `isMasterMongoDestinationTask` (`L453`).
5. `internal/webui/task_planner_multilayer.go`: Xóa bỏ hoàn toàn 6 Sub-Tasks hardcode `CORE-01-S01..S06` và `buildSelfUpgradeOrchestratorCodeDemo` (`L12-177`), chuyển về bộ phân rã động qua `invokeArchitectLLM` & `inspectTargetRepoSymbols`.
6. `internal/webui/task_inspector.go`:
   - Xóa sạch `buildDedicatedSubTaskCodeDemo` switch `case "TASK-M01" .. "TASK-M10"` (`L2599-2707`).
   - Xóa sạch các nhánh cheat `if isSelfUpgradeOrchestratorTask`, `if isPostgresOrDualSourceMigrationTask`, `if isMasterMongoDestinationTask` trong `buildConcreteCodeDemoForTask` (`L3283-3346`).
   - Xóa sạch điểm cộng thiên vị từ khóa `"trans-his"`, `"postgres"`, `"recon"`, `"shadow"`, `"master"` trong `inspectTargetRepoSymbols` (`L2978-2995`).
   - Thay cơ chế chọn repo hardcode tên `cdc-cms-service`, `centralized-data-service`, `centralized-export-service` (`L512-655`) bằng cơ chế khớp động theo danh sách git repositories thực tế được phát hiện trong workspace (`DiscoverWorkspaceRepos`).
   - Đấu nối `invokeArchitectLLM` trực tiếp vào `DecomposeMultiRepoCodebaseSubTasks` và `generateDeepTaskArtifacts` để Model sinh Sub-Tasks, Root Causes, Solutions và Code Demo thực chất cho mọi task.
7. `internal/webui/task_chat.go` & `task_chat_knowledge.go`: Xóa sạch các nhánh `case strings.Contains(..., "centralized-data-service")` (`task_chat.go:182-186, 1019-1035`), danh sách issue giả của `cdc-cms-service`/`centralized-data-service` (`task_chat.go:1495-1580`), và văn mẫu tĩnh trong `renderContextualDiscussReply` (`task_chat_knowledge.go:40-68`).
8. `internal/webui/telegram_bridge.go` & `server.go`:
   - Xóa bỏ đoạn sinh hunk giả `// Isolated Remediation Hunk for ... CoerceToBSONValue, Ordered BulkWrite OCC (_source_ts <= incomingTs), masterMongoPool` trong `synthesizeIsolatedRemediationDiffIfApproved` (`telegram_bridge.go:2895-2932`).
   - Xóa bỏ lệnh giả `test -f 01_requirements.md` đúc biên lai `EV-...` tại `server.go:1825-1830` (thay bằng chạy lệnh build/test thực tế của repo theo `detectRepoTechStackProfile`).
   - Xóa bỏ dòng hardcode `masterMongoPool / _source_ts` trong `audit_report.md` tại `server.go:1939`.

---

## 2. 💻 Code Demo Cụ Thể (100% Pure Agnostic Core — Không Từ Khóa Cheat)

### 2.1. `BuildAndRunCalibratedLayaGate` 100% Agnostic (`internal/agentgate/kernel_bridge.go`) — Đối chiếu trực tiếp `01_requirements.md` vs `09_tasks_solution.md` qua Neural Engine `:8200`
```go
func truncateUTF8Runes(s string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes])
}

// extractAgnosticDocumentClaims extracts meaningful non-header statements from a markdown document
// without any hardcoded technology or domain keywords.
func extractAgnosticDocumentClaims(rawDoc string, maxCount int) [] lineWithNum {
	lines := strings.Split(rawDoc, "\n")
	var out []lineWithNum
	for idx, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if len(trimmed) < 20 || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") ||
			strings.HasPrefix(trimmed, "| :---") || strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "> 📌") {
			continue
		}
		out = append(out, lineWithNum{
			LineNum: idx + 1,
			Text:    truncateUTF8Runes(trimmed, 160),
		})
		if len(out) >= maxCount {
			break
		}
	}
	return out
}
```

### 2.2. `invokeArchitectLLM` & `synthesizeDirectAnswerSummary` 100% Agnostic (`internal/webui/task_llm_bridge.go`)
```go
type ArchitectSubTaskSpec struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
	Wave        int      `json:"wave"`
}

type ArchitectLLMAnalysis struct {
	DirectAnswer string                 `json:"direct_answer"`
	RootCauses   []string               `json:"root_causes"`
	Solutions    []string               `json:"solutions"`
	CodeDemo     string                 `json:"code_demo"`
	CodeLang     string                 `json:"code_lang"`
	SubTasks     []ArchitectSubTaskSpec `json:"sub_tasks,omitempty"`
}

// synthesizeDirectAnswerSummary is 100% domain-agnostic: it summarizes directly from the LLM's analysis
// and the task's actual solutions, with ZERO hardcoded task IDs or technology templates.
func synthesizeDirectAnswerSummary(qt queue.QueuedTask, userPrompt string, rootCauses, solutions []string) string {
	cleanQ := strings.TrimSpace(userPrompt)
	var rcPart, solPart string
	if len(rootCauses) > 0 {
		rcPart = strings.TrimSpace(strings.TrimLeft(rootCauses[0], "-•* "))
	}
	if len(solutions) > 0 {
		solPart = strings.TrimSpace(strings.TrimLeft(solutions[0], "-•* "))
	}
	if cleanQ != "" && cleanQ != strings.TrimSpace(qt.Description) && rcPart != "" && solPart != "" {
		return fmt.Sprintf("Dạ anh, trả lời trực tiếp cho yêu cầu %q trên task `%s`: Nguyên nhân chính là %s; giải pháp thực thi là %s.", cleanQ, qt.ExternalID, rcPart, solPart)
	}
	if solPart != "" {
		return fmt.Sprintf("Dạ anh, đối với task `%s` (%s): %s", qt.ExternalID, qt.Title, solPart)
	}
	return fmt.Sprintf("Dạ anh, đối với task `%s` (%s): Hệ thống đã phân tích các điểm chạm mã nguồn thực tế trên repository đích và lập phương án thực thi cô lập.", qt.ExternalID, qt.Title)
}
```

### 2.3. Chốt chặn `applyApprovedSubTaskPatchToTargetRepo` không tự ghi đè cây nguồn `s.Root` (`internal/webui/subtask_git_apply.go:205-208`)
```go
	repoDir := strings.TrimSpace(st.TargetRepo)
	if repoDir == "" || (s.Root != "" && filepath.Clean(repoDir) == filepath.Clean(s.Root)) {
		return 0
	}
```

### 2.4. `TASK-CW-05` — Xóa Sổ 100% Chốt Chặn Giấu AI Model, Xóa Template 6-Tầng `Core-01` & Thay Thế Trọn Vẹn Sub-Tasks Từ AI Model (`internal/webui/task_planner_multilayer.go`, `internal/webui/task_revise_workflow.go`, `internal/webui/task_llm_bridge.go`, `internal/webui/task_chat_knowledge.go`, `internal/webui/telegram_bridge.go`)
- **Tham chiếu `REF:`:**
  - `REF: internal/webui/task_planner_multilayer.go:12-172`
  - `REF: internal/webui/task_inspector.go:980-1012`
  - `REF: internal/webui/task_revise_workflow.go:75-110`
  - `REF: internal/webui/task_llm_bridge.go:185-298`
  - `REF: internal/webui/task_chat_knowledge.go:16-112`
  - `REF: internal/webui/telegram_bridge.go:1814-1875`
  - `REF: internal/webui/telegram_bridge.go:2020-2175`
```go
// internal/webui/task_revise_workflow.go
func (s *Server) applyLLMSubTaskSpecs(qt queue.QueuedTask, subTasks []SubTaskDTO, analysis *ArchitectLLMAnalysis) []SubTaskDTO {
	if analysis == nil || len(analysis.SubTasks) == 0 {
		return subTasks
	}
	targetRepo := s.ResolveTaskTargetRepo(qt)
	if strings.TrimSpace(targetRepo) == "" && len(subTasks) > 0 {
		targetRepo = subTasks[0].TargetRepo
	}
	if strings.TrimSpace(targetRepo) == "" {
		targetRepo = s.Root
	}
	repoName := filepath.Base(targetRepo)
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	prefix := buildCleanSubTaskIDPrefix(qt.ExternalID, qt.Title)
	defaultModel := strings.TrimSpace(qt.AssignedModel)
	if defaultModel == "" {
		defaultModel = "gemini-4-argon (antigravity)"
	}

	out := make([]SubTaskDTO, 0, len(analysis.SubTasks))
	for i, spec := range analysis.SubTasks {
		ctxLine := strings.TrimSpace(spec.ContextFileLine)
		rawCands := extractCandidateRelPaths(ctxLine, repoName)
		var normFiles []string
		for _, rc := range rawCands {
			if norm := resolveActualRepoRelPath(targetRepo, rc); norm != "" {
				normFiles = append(normFiles, norm)
			}
		}
		targetFiles := ctxLine
		if len(normFiles) > 0 {
			targetFiles = strings.Join(normFiles, ", ")
			ctxLine = targetFiles
		}
		stack := detectRepoTechStackProfile(targetRepo, []string{ctxLine})
		waveIdx := i / 2
		role := "Coder"
		if i > 0 && i == len(analysis.SubTasks)-1 && strings.Contains(strings.ToLower(spec.Title+" "+ctxLine), "test") {
			role = "Tester"
		}
		out = append(out, SubTaskDTO{
			SubTaskID:       fmt.Sprintf("%s%02d", prefix, i+1),
			ParentKey:       qt.ExternalID,
			GroupTitle:      fmt.Sprintf("Nhóm %d (Wave %d): Thực thi & Kiểm thử trên `%s`", waveIdx+1, waveIdx, repoName),
			WaveIndex:       waveIdx,
			WorkflowStep:    fmt.Sprintf("Wave %d", waveIdx),
			Title:           strings.TrimSpace(spec.Title),
			TargetRepo:      targetRepo,
			TargetRepoName:  repoName,
			WorktreePath:    filepath.Join(".orchestrator", "worktrees", safeParent, repoName),
			TargetFiles:     targetFiles,
			ContextFileLine: ctxLine,
			Description:     strings.TrimSpace(spec.Description),
			TestDoD:         stack.BuildAndDoD,
			Status:          "pending",
			Stage:           "NEEDS_YOU",
			Assignee:        role,
			Model:           defaultModel,
		})
	}
	return out
}
```

