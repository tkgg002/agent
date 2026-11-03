# 09. Tasks Solution — Giải Pháp Kỹ Thuật Duy Nhất & Code Demo: Khắc Phục Lỗi `EX-002` Báo `DONE` Nhưng Không Có Code Diff Thật & Sub-Tasks Không Báo `DONE` (`fix_s4_real_llm_diff_and_subtask_done_workflow`)

## I. Kiến trúc Giải Pháp Duy Nhất (Single Optimal Solution — Rule #9.5)

Để chấm dứt triệt để việc `S4` chạy giả (sinh comment `"// Isolated Remediation Hunk"`) và bỏ qua luồng Sub-Task Wave khi User bấm `APPROVE` (`approve_and_run`) hoặc `RUN_TASK`, toàn bộ luồng thực thi từ `Step 8 (Human Gate APPROVED)` đến `Step 17 (DONE)` được chuẩn hóa khép kín qua 4 mắt xích:

1. **Mắt xích 1 — Hợp nhất `approve_and_run` & `approve_all` để luôn chạy `executeAllSubTasksWaveByWave` (`internal/webui/server.go:1199-1283`):**
   - Sửa điều kiện tại `internal/webui/server.go:1209` từ `if len(subTasks) > 0 && action == "approve_all"` thành `if len(subTasks) > 0 && (action == "approve_all" || action == "approve_and_run")`.
   - Khi từng Sub-Task (`TASK-TRAN01`, `TASK-TRAN02`) chạy qua `executeAllSubTasksWaveByWave` (`internal/webui/subtask_wave_engine.go:390-405`) và khi Task cha hoàn tất (`internal/webui/server.go:1260, 1506, 1847` & `internal/webui/task_inspector.go:1994`), **bắt buộc cập nhật đồng bộ**:
     - `st.Stage = "DONE"`
     - `st.Status = "done"`
     - `st.WorkflowStep = fmt.Sprintf("Wave %d · S6 Hoàn Tất (DONE)", st.WaveIndex)`
   - Cập nhật bảng Sub-Tasks trong `09_tasks_solution.md` của Task cha (`Stage: DONE`) và gửi `sendTelegramWaveExecutionLedger` + danh sách Sub-Tasks `✅ DONE` trong `notifyTelegramOnTaskDone` (`internal/webui/telegram_bridge.go:2266-2295`) vào đúng Topic Telegram của Task.

2. **Mắt xích 2 — Xóa sổ Hunk comment giả `"// Isolated Remediation Hunk"`, chặn nhiễm bẩn diff chéo (`includeAllWhenAnyMatch = false`) và gắn LLM Coder (`invokeMuscleSubTaskCodeGen`) tại `S4 Implement` (`internal/webui/telegram_bridge.go:2813-2962` & `internal/webui/task_llm_bridge.go`):**
   - **Xóa sạch 100%** khối lệnh sinh 4 dòng comment giả `"// Isolated Remediation Hunk for ... External repo untouched per L003"` tại `internal/webui/telegram_bridge.go:2918-2932`.
   - Đặt `includeAllWhenAnyMatch = false` tại `internal/webui/telegram_bridge.go:2850, 2878` để `extractMatchingCandidateDiffSections` **chỉ lấy đúng các file thuộc danh sách `candidates` của Task/Sub-Task đang chạy**, tuyệt đối không quét nhầm các file uncommitted của Task khác (như 6 file PostgreSQL của `TASK-13`).
   - Tại `S4 Implement` (`executeAllSubTasksWaveByWave` tại `subtask_wave_engine.go:245-276` & `synthesizeIsolatedRemediationDiffIfApproved` tại `telegram_bridge.go:2882`), khi phát hiện file mục tiêu của Sub-Task chưa có code mới thực sự trên `TargetRepo`, gọi trực tiếp **`s.invokeMuscleSubTaskCodeGen(ctx, qt, st)`** (`AntigravityRunner` — MUSCLE Slot B) để:
     - Đọc nội dung hiện tại của các file ứng viên trên `st.TargetRepo` (hoặc tạo mới nếu là file mới như `src/infrastructure/database/dbHandler/clickhouse.ts`) cùng với đặc tả `09_tasks_solution.md` và `subtasks/<SubID>/08_tasks.md`.
     - Sinh code thực thi thật và ghi xuống các đường dẫn tương ứng trong `st.TargetRepo`.
     - Thu thập `git diff` thật (gồm cả untracked file mới qua `collectRepoUnifiedDiffWithUntracked`) và ghi vào `.orchestrator/tasks/<Key>/subtasks/<SubID>/diff.patch` + `.orchestrator/tasks/<Key>/diff.patch`.

3. **Mắt xích 3 — Siết chặt `VerifySubTaskCodeOnDisk` & Hỗ trợ đa ngôn ngữ (`package.json` + `go.mod`) ở `5A Hard Gate` (`internal/webui/task_inspector.go:1420-1473` & `internal/webui/server.go:1788-1815`):**
   - Trong `VerifySubTaskCodeOnDisk` (`task_inspector.go:1423-1436`), loại bỏ mọi section chứa `"Isolated Remediation Hunk for "` và **bắt buộc kiểm tra `os.Stat(absTarget)` tồn tại thật trên đĩa** kết hợp có diff thật (`git status --porcelain` hoặc `diff.patch` có hunk code thật). Nếu file chưa tồn tại trên đĩa (như `dbHandler/clickhouse.ts` trước đây), trả về `ok = false` và chặn không cho báo `DONE` giả!
   - Trong `server.go:1788-1815`, ngoài `go.mod`, bổ sung kiểm tra `package.json` để chạy kiểm định đối với các repo TypeScript/Node.js.

4. **Mắt xích 4 — Hiển thị trực tiếp `diff.patch` của `S4` và trạng thái `DONE` của Sub-Tasks trên WebUI (`internal/webui/static/app.js:1366-1378`):**
   - Trong `renderStageReportInspector` (`app.js:1372-1377`), khi tab đang chọn là `S4` (hoặc khi Task đã ở `S4`/`S5`/`DONE` và có `s4Report.diff_preview`), khung `#sri-code-demo-content` hiển thị trực tiếp `Unified Git Diff (S4 diff.patch)` thay vì vẫn giữ `s3Report.code_demo`.
   - Sau khi `approveAndRunActiveTask` (`app.js:1389-1405`) hoàn tất, tự động chuyển `activeStageReportTab = "S4"` để User nhìn thấy ngay `diff.patch` thật và danh sách Sub-Tasks `Wave 0/1 · S6 Hoàn Tất (DONE)`.

---

## II. Code Demo Cụ Thể Từng File

### 1. `internal/webui/server.go` (Hợp nhất `approve_and_run` & `approve_all` chạy `executeAllSubTasksWaveByWave` + cập nhật `WorkflowStep`)

```go
// internal/webui/server.go:1208-1266
if len(subTasks) > 0 && (action == "approve_all" || action == "approve_and_run") {
	s.appendTaskTerminalLog(taskKey, fmt.Sprintf("[⚡ %s · Wave Engine] Kích hoạt tuần tự %d Sub-Tasks qua 6 Wave (S1→S6) cho %s", action, len(subTasks), taskKey))
	waveErr := s.executeAllSubTasksWaveByWave(r.Context(), &targetTask, subTasks)
	if waveErr != nil {
		s.writeJSON(w, http.StatusConflict, map[string]any{
			"error":     waveErr.Error(),
			"sub_tasks": subTasks,
		})
		return
	}
	s.sendTelegramWaveExecutionLedger(&targetTask, subTasks)
}
for i := range subTasks {
	subTasks[i].Approved = true
	if subTasks[i].Stage == "DONE" || subTasks[i].Status == "done" {
		subTasks[i].WorkflowStep = fmt.Sprintf("Wave %d · S6 Hoàn Tất (DONE)", subTasks[i].WaveIndex)
	}
}
s.savePersistedSubTasks(taskKey, subTasks)
```

Và tại `internal/webui/server.go:1506-1510, 1847-1852`:
```go
for i := range subTasks {
	subTasks[i].Stage = "DONE"
	subTasks[i].Status = "done"
	subTasks[i].Approved = true
	subTasks[i].WorkflowStep = fmt.Sprintf("Wave %d · S6 Hoàn Tất (DONE)", subTasks[i].WaveIndex)
}
s.savePersistedSubTasks(taskKey, subTasks)
s.syncMasterSolutionSubTaskStatusOnDisk(&targetTask, subTasks)
```

### 2. `internal/webui/telegram_bridge.go` & `internal/webui/task_llm_bridge.go` (Xóa Hunk comment giả, tắt `includeAllWhenAnyMatch`, và gọi `invokeMuscleSubTaskCodeGen` sinh code thật ở `S4`)

```go
// internal/webui/telegram_bridge.go:2848-2895
for _, repo := range targetRepos {
	if raw := s.collectRepoUnifiedDiffWithUntracked(repo, candidates); strings.TrimSpace(raw) != "" {
		// TUYỆT ĐỐI KHÔNG dùng includeAllWhenAnyMatch=true để tránh kéo nhầm diff của task khác (VD: TASK-13 sang EX-002)
		matched := extractMatchingCandidateDiffSections(raw, candidates, false)
		if strings.TrimSpace(matched) != "" {
			realFilteredBlocks = append(realFilteredBlocks, strings.TrimSpace(matched))
		}
	}
}

combinedReal := strings.TrimSpace(strings.Join(realFilteredBlocks, "\n\n"))
// Nếu các file ứng viênของ Sub-Task chưa có diff thật trên đĩa, gọi MUSCLE (AntigravityRunner) sinh & ghi code thật vào TargetRepo
if missingCandidates := findMissingCandidateFilesInDiff(combinedReal, candidates, targetRepos); len(missingCandidates) > 0 {
	if s.invokeMuscleSubTaskCodeGen(qt, subTasks, missingCandidates) {
		realFilteredBlocks = nil
		for _, repo := range targetRepos {
			if raw := s.collectRepoUnifiedDiffWithUntracked(repo, candidates); strings.TrimSpace(raw) != "" {
				if matched := extractMatchingCandidateDiffSections(raw, candidates, false); strings.TrimSpace(matched) != "" {
					realFilteredBlocks = append(realFilteredBlocks, strings.TrimSpace(matched))
				}
			}
		}
		combinedReal = strings.TrimSpace(strings.Join(realFilteredBlocks, "\n\n"))
	}
}
```

### 3. `internal/webui/task_inspector.go` (Siết chặt `VerifySubTaskCodeOnDisk` bác bỏ Hunk giả & kiểm tra file tồn tại vật lý trên đĩa)

```go
// internal/webui/task_inspector.go:1420-1455
for _, rawRel := range candidates {
	cleanRel := filepath.ToSlash(strings.TrimSpace(rawRel))
	if cleanRel == "" {
		continue
	}
	absTarget := filepath.Join(cleanRepo, filepath.FromSlash(cleanRel))
	info, statErr := os.Stat(absTarget)
	fileExistsOnDisk := statErr == nil && !info.IsDir() && info.Size() > 0

	hasRealPatchSection := false
	for _, sec := range splitUnifiedDiffSections(combinedPatch) {
		if strings.Contains(sec, "Isolated Remediation Hunk for ") {
			continue
		}
		if strings.Contains(sec, " b/"+cleanRel) || strings.Contains(sec, "\t"+cleanRel) {
			hasRealPatchSection = true
			break
		}
	}
	if fileExistsOnDisk && (hasRealPatchSection || strings.Contains(porcelainOut, cleanRel)) {
		verified = append(verified, cleanRel+" (on-disk + diff.patch)")
		continue
	}
	missing = append(missing, cleanRel)
}
```

### 4. `internal/webui/static/app.js` (Hiển thị `diff.patch` thật ở tab `S4` & tự mở tab `S4` sau khi `approve_and_run`)

```javascript
// internal/webui/static/app.js:1366-1380
if (codePre) {
  const s3Report = reports.find((r) => r.stage === "S3");
  const s4Report = reports.find((r) => r.stage === "S4");
  if (activeStageReportTab === "S4" && s4Report && s4Report.diff_preview) {
    codePre.textContent = s4Report.diff_preview;
  } else if (activeReport && activeReport.diff_preview && (activeStageReportTab === "S5" || activeStageReportTab === "S6")) {
    codePre.textContent = activeReport.diff_preview;
  } else if (s3Report && s3Report.code_demo) {
    codePre.textContent = s3Report.code_demo;
	} else if (s4Report && s4Report.diff_preview) {
    codePre.textContent = s4Report.diff_preview;
  } else {
    codePre.textContent = "// Chưa có Code Demo / Diff cho bước này.";
  }
}
```

### 5. Bổ sung Self-Healing `S5 -> S3 -> S4`: Gom lô sinh code theo Sub-Task (`{"files":[{"file_path":"...","code":"..."}]}`) & Chống bỏ sót file khi Repo có sẵn thay đổi từ Task trước (`subtask_codegen_muscle.go` & `telegram_bridge.go`)
- Trong `internal/runner/antigravity.go:268`, `extractStreamJSONText` tự động bóc trường `"content"` nếu dòng JSON bắt đầu bằng `{`, làm mất cấu trúc JSON `{"file_path": "...", "content": "..."}`. Do đó `muscleFileGenResponse` sử dụng khóa `"code"` (và gom toàn bộ `candidates` của 1 Sub-Task vào 1 lần gọi `AntigravityRunner` với `{"files": [{"file_path": "...", "code": "..."}]}`), đồng thời hỗ trợ fallback trích xuất code block hoặc raw source code.
- Trong `synthesizeIsolatedRemediationDiffIfApproved` (`telegram_bridge.go:2960-3035`), kiểm tra `subtasks/<subID>/diff.patch`:
  - Nếu `subtasks/<subID>/diff.patch` **chưa tồn tại** (lần đầu Sub-Task chạy `S4`), luôn gọi `invokeMuscleSubTaskCodeGen` cho toàn bộ `candidates` của Sub-Task đó để tích hợp logic của Task hiện tại (`EX-002`) lên trên nội dung hiện có của file (kể cả khi file đó như `models/trans-his/trans-his.model.ts` đang có sẵn thay đổi chưa commit từ `TASK-13`).
  - Nếu `subtasks/<subID>/diff.patch` **đã tồn tại** hợp lệ (khi bước `implement` gọi lại sau `executeAllSubTasksWaveByWave`), tái sử dụng ngay `subtasks/<subID>/diff.patch` để không gọi LLM lần 2.

