---
tier: T2
mode: propose-only
target_project: ai-orchestrator
suffix: task_100_2026_11_02
created_at: 2026-11-02T16:20:40+07:00
---

# 09. Technical Solution & Code Demo — Khắc Phục Lỗi `S1 / S2 / S2.5 / S3` & Nhiễm Bẩn CDC Repo Trên `ai-orchestrator` (`TASK-100`)

## 1. Bảng Tổng Hợp Issues (`F1`–`F5`)
| Mã | Mức độ | Vị trí (`file:line`) | Nguyên nhân gốc rễ (Root Cause) | Trạng thái & Hành động |
| :--- | :---: | :--- | :--- | :--- |
| **`F1`** | **CRITICAL** | `internal/webui/static/app.js:2724-2748`, `internal/webui/task_inspector.go:423-425`, `488-559` | **(1)** Khi mở modal `+ Giao Task Mới`, `openDispatchModal()` lấy `dashboardState.active_target_repo` của task CDC trước đó (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service`) gán sẵn làm `target_repo` mặc định; **(2)** `ResolveTaskTargetRepo` & `ResolveTaskAffectedRepos` chỉ khớp tên thư mục (`baseName`) chứ **không khớp tên nhánh Git (`CurrentBranch`)** trong câu lệnh (`branch feat/auto-recon-export` thuộc về `/Users/trainguyen/Documents/work/centralized-export-service`), đồng thời return ngay `repo:` bị leak từ modal UI dù repo đó đang ở nhánh `recon-heal`; **(3)** `DiscoverWorkspaceRepos` chặn cứng ở `len(repos) >= 32` làm cắt mất hơn 50 repo trong `/Users/trainguyen/Documents/work/`. | **Pending (`APPROVE`):** Sửa `ResolveTaskTargetRepo`, `ResolveTaskAffectedRepos`, `DiscoverWorkspaceRepos` & `app.js` |
| **`F2`** | **HIGH** | `internal/webui/task_inspector.go:2819-2901`, `internal/webui/unified_doc_format.go:16-96` | Ở bước `S1`, `generateDeepTaskArtifacts` gọi `loadOrGenerateSubTasks` + `invokeArchitectLLM` trên repo sai (`centralized-data-service`), sinh ra 3 Sub-Tasks sửa code Go bịa đặt (`TASK-100-M01..M03`: `ReconciliationReportRepo`, `SourceCount *int64 Migration 017`, `recon_smoke.go`) rồi **ghi đè ngược lại vào Mục 2 của `01_requirements_task_100.md` và `08_tasks_task_100.md`**. | **Pending (`APPROVE`):** Cố định đúng `targetRepo` theo nhánh trước khi phân rã `S1` & nạp danh sách file diff thực tế của nhánh |
| **`F3`** | **HIGH** | `internal/webui/task_inspector.go:2682-2790`, `internal/webui/unified_doc_format.go:99-168` | Bước `S2` (`inspectTargetRepoSymbols` & `BuildUnifiedS2AnalysisDoc`) không kiểm tra `git diff <base>...HEAD` của nhánh cần audit, chỉ tách từ khóa `"recon"` rồi bốc bừa 12 dòng đầu của 3 file Go trong CDC repo (`reconciliation_report.go`, `transmute_schedule_repo.go`, `recon_smoke_repo.go`) và đóng dấu `VERIFIED 🟢` cứng bằng văn mẫu tĩnh. | **Pending (`APPROVE`):** Bổ sung quét ưu tiên các file thực sự thay đổi trong `git diff` của branch tại `inspectTargetRepoSymbols` |
| **`F4`** | **CRITICAL** | `internal/webui/unified_doc_format.go:171-204`, `internal/agentgate/kernel_bridge.go:632-661`, `bin/orchestrator` (`PID 7644`) | **(1)** `BuildUnifiedDG0CritiqueDoc` (`S2.5`) là hàm `fmt.Sprintf` tĩnh 100%, tự in cứng `VERIFIED` và `"gate": "PASS"` mà không kiểm tra xem `targetRepo` có đang đứng ở branch `feat/auto-recon-export` như đề bài yêu cầu hay không (vi phạm Rule #9.4 & `L018` — không "la làng lên" khi lệch branch/repo); **(2)** `BuildAndRunCalibratedLayaGate` dùng chính dòng văn bản của file vừa sinh làm cả `Claim`, `Evidence`, `MustContain` ($A \equiv A$, vi phạm `L019`); **(3)** Tiến trình daemon `PID 7644` đang chạy bản binary build từ `28/10 16:34` khiến `laya_health_s0: false` và rơi vào nhánh mock JSON. | **Pending (`APPROVE`):** Kiểm tra thực tế branch mismatch tại `S2.5 (DG0)` + rebuild/restart `./bin/orchestrator` |
| **`F5`** | **HIGH** | `internal/webui/task_llm_bridge.go:107-131`, `185-236` | `inspectRealGitStatusAndDiff` chỉ lấy `git status --short` (trả về 55 files modified của nhánh `recon-heal` trên CDC repo) mà không lấy tên branch hiện tại (`git rev-parse --abbrev-ref HEAD`) và không lấy `git diff develop...HEAD` cho feature branch đã commit; `invokeArchitectLLM` cấm LLM gọi tool (`KHÔNG gọi tool read_file hay run_command`), khiến LLM ghép nhầm 55 files của CDC repo + `Migration 017 SourceCount *int64` vào branch `feat/auto-recon-export` và bịa ra Code Demo `FindReportsForExport`. | **Pending (`APPROVE`):** Bổ sung `currentBranch` + `git diff <base>...HEAD` vào `inspectRealGitStatusAndDiff` và `invokeArchitectLLM` |

---

## 2. Thiết Kế Giải Pháp Duy Nhất Ánh Xạ 1-1 (`F1` → `F5`) & Code Demo

### 2.1. Giải pháp cho `F1`, `F2`, `F3`: Định tuyến Repository theo Branch thực tế trên đĩa & Ưu tiên File trong `git diff` của Branch (`internal/webui/task_inspector.go` & `internal/webui/static/app.js`)
- **Tại `internal/webui/task_inspector.go`:**
  1. Bỏ giới hạn `if len(repos) >= 32 { break }` tại `DiscoverWorkspaceRepos` (`task_inspector.go:423-425`) để nạp đầy đủ 100% repositories trong `/Users/trainguyen/Documents/work/`.
  2. Bổ sung helper `extractExplicitBranchFromPrompt(text string) string` nhận diện mẫu `branch <name>` / `nhánh <name>` (hoặc token dạng `feat/...`, `fix/...`, `hotfix/...`, `release/...`) từ `Title`/`Description`.
  3. Trong `ResolveTaskTargetRepo(qt)` (`task_inspector.go:488-511`) và `ResolveTaskAffectedRepos(qt)` (`task_inspector.go:514-559`):
     - Trước khi tin vào label `repo:` (vốn dễ bị leak từ `active_target_repo` của task cũ trên modal UI), kiểm tra xem đề bài có chỉ định `wantBranch := extractExplicitBranchFromPrompt(qt.Title + " " + qt.Description)` hay không.
     - Nếu `wantBranch != ""` mà repo trong label `repo:` **không** đang đứng ở `wantBranch` (ví dụ `centralized-data-service` đang ở `recon-heal`), tự động quét `DiscoverWorkspaceRepos` để tìm repo có ` strings.EqualFold(r.CurrentBranch, wantBranch)` (khớp chính xác `/Users/trainguyen/Documents/work/centralized-export-service` đang ở `feat/auto-recon-export`!).
  4. Trong `inspectTargetRepoSymbols` (`task_inspector.go:2682-2790`), thu thập danh sách file thay đổi thực tế trên branch (`git status --porcelain` + `git diff --name-only develop...HEAD` / `main...HEAD`) và cộng `+100` điểm cho các file nằm trong diff của branch để `S1`, `S2`, `S3` luôn lấy đúng các file đang có thay đổi trên branch.

```go
// File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:416-430 & 488-559
// --- TRƯỚC (HEAD hiện tại) ---
	if entries, err := os.ReadDir(workRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			full := filepath.Join(workRoot, e.Name())
			addRepo(e.Name(), full)
			if len(repos) >= 32 {
				break
			}
		}
	}

	return repos
}
// ...
func (s *Server) ResolveTaskTargetRepo(qt queue.QueuedTask) string {
	if explicit := extractSingleLabel(qt.Labels, "repo:", ""); explicit != "" {
		if st, err := os.Stat(explicit); err == nil && st.IsDir() {
			return explicit
		}
	}
	lower := strings.ToLower(qt.ExternalID + " " + qt.Title + " " + qt.Description)
	for _, r := range s.DiscoverWorkspaceRepos(context.Background()) {
		baseName := strings.ToLower(filepath.Base(r.Path))
		if baseName == "" || baseName == "data-hub" || baseName == "agent" {
			continue
		}
		if baseName == "ai-orchestrator" && !isExplicitOrchestratorTarget(qt) {
			continue
		}
		if strings.Contains(lower, baseName) {
			return r.Path
		}
	}
	if strings.TrimSpace(s.Root) != "" {
		return s.Root
	}
	return "."
}

// --- SAU KHI SỬA ---
	if entries, err := os.ReadDir(workRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			full := filepath.Join(workRoot, e.Name())
			addRepo(e.Name(), full)
		}
	}

	return repos
}
// ...
var explicitBranchPromptRe = regexp.MustCompile(`(?i)(?:\bbranch\b|\bnhánh\b)\s+[` + "`" + `"'()]?([a-zA-Z0-9._/-]+/[a-zA-Z0-9._/-]+|[a-zA-Z0-9._-]+)[` + "`" + `"'(),.;:]?`)

func extractExplicitBranchFromPrompt(text string) string {
	if m := explicitBranchPromptRe.FindStringSubmatch(text); len(m) == 2 {
		cand := strings.Trim(m[1], "`\"'(),.;:")
		if cand != "" && !strings.EqualFold(cand, "main") && !strings.EqualFold(cand, "master") && !strings.EqualFold(cand, "develop") {
			return cand
		}
	}
	return ""
}

func (s *Server) ResolveTaskTargetRepo(qt queue.QueuedTask) string {
	lower := strings.ToLower(qt.ExternalID + " " + qt.Title + " " + qt.Description)
	wantBranch := extractExplicitBranchFromPrompt(qt.Title + " " + qt.Description)
	discovered := s.DiscoverWorkspaceRepos(context.Background())

	// 1. Nếu đề bài nêu đích danh tên repo trong văn bản -> ưu tiên repo đó
	for _, r := range discovered {
		baseName := strings.ToLower(filepath.Base(r.Path))
		if baseName == "" || baseName == "data-hub" || baseName == "agent" {
			continue
		}
		if baseName == "ai-orchestrator" && !isExplicitOrchestratorTarget(qt) {
			continue
		}
		if strings.Contains(lower, baseName) {
			return r.Path
		}
	}
	// 2. Nếu đề bài chỉ định tên nhánh (vd: branch feat/auto-recon-export) -> ưu tiên repo đang đứng ở nhánh đó
	if wantBranch != "" {
		for _, r := range discovered {
			baseName := strings.ToLower(filepath.Base(r.Path))
			if baseName == "" || baseName == "data-hub" || baseName == "agent" || (baseName == "ai-orchestrator" && !isExplicitOrchestratorTarget(qt)) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(r.CurrentBranch), wantBranch) {
				return r.Path
			}
		}
	}
	// 3. Nếu có label repo: từ UI và không mâu thuẫn với nhánh yêu cầu
	if explicit := extractSingleLabel(qt.Labels, "repo:", ""); explicit != "" {
		if st, err := os.Stat(explicit); err == nil && st.IsDir() {
			return explicit
		}
	}
	if strings.TrimSpace(s.Root) != "" {
		return s.Root
	}
	return "."
}
```

- **Tại `internal/webui/static/app.js:3017-3028` (`autoPreviewSubTasksOnTitle`):**
  Khi người dùng gõ tiêu đề/mô tả trong modal tạo Task có chứa tên repo hoặc tên nhánh (`r.branch`, ví dụ `feat/auto-recon-export`), tự động chuyển dropdown `inp-task-repo` và checkbox `affected_repo_chk` sang đúng repo đó và bỏ tick repo cũ bị kế thừa từ task trước.

```javascript
// File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:3017-3028
// --- TRƯỚC (HEAD hiện tại) ---
function autoPreviewSubTasksOnTitle() {
  const title = document.getElementById("inp-task-title")?.value.trim() || "";
  if (!title) return;
  const lower = title.toLowerCase();
  document.querySelectorAll('input[name="affected_repo_chk"]').forEach((chk) => {
    const repoBase = formatRepoBasename(chk.value).toLowerCase();
    if (repoBase && lower.includes(repoBase)) {
      chk.checked = true;
    }
  });
  fillDefaultSubTasksInModal();
}

// --- SAU KHI SỬA ---
function autoPreviewSubTasksOnTitle() {
  const title = document.getElementById("inp-task-title")?.value.trim() || "";
  const desc = document.getElementById("inp-task-desc")?.value.trim() || "";
  const combined = `${title} ${desc}`.toLowerCase();
  if (!combined.trim()) return;
  const repos = dashboardState?.available_repos || [];
  let matchedRepoPath = "";
  for (const r of repos) {
    const repoBase = formatRepoBasename(r.path).toLowerCase();
    const branchLower = String(r.branch || "").toLowerCase();
    if ((repoBase && repoBase !== "data-hub" && repoBase !== "agent" && combined.includes(repoBase)) ||
        (branchLower && branchLower !== "main" && branchLower !== "master" && branchLower !== "develop" && combined.includes(branchLower))) {
      matchedRepoPath = r.path;
      break;
    }
  }
  if (matchedRepoPath) {
    const repoSel = document.getElementById("inp-task-repo");
    if (repoSel) repoSel.value = matchedRepoPath;
    document.querySelectorAll('input[name="affected_repo_chk"]').forEach((chk) => {
      chk.checked = chk.value === matchedRepoPath;
    });
  }
  fillDefaultSubTasksInModal();
}
```

---

### 2.2. Giải pháp cho `F4`: Phản biện Thực tế Nhánh/Repo tại `S2.5 (DG0)` (`internal/webui/unified_doc_format.go:171-204`)
- Thay vì in cứng `C-REALITY-01: VERIFIED` và `C-CONTRA-01: PASS`, kiểm tra trực tiếp `actualBranch := detectRepoBranchFast(primaryRepo)` đối chiếu với `wantBranch := extractExplicitBranchFromPrompt(qt.Title + " " + qt.Description)`:
  - Nếu `wantBranch != ""` và `!strings.EqualFold(actualBranch, wantBranch)` -> Ghi rõ cảnh báo **`BLOCK / BRANCH_MISMATCH`** (`Prompt yêu cầu nhánh wantBranch nhưng TargetRepo đang ở nhánh actualBranch`) để đúng tinh thần Rule #9.4 ("la làng lên" khi lệch thực tế).
  - Nếu khớp nhánh -> Ghi rõ bằng chứng vật lý `actualBranch == wantBranch` kèm số lượng file thay đổi thực tế trên nhánh.

```go
// File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/unified_doc_format.go:171-204
// --- TRƯỚC (HEAD hiện tại) ---
func BuildUnifiedDG0CritiqueDoc(
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	symbolHits []RepoSymbolHit,
	affectedRepos []string,
	repoSummary, tier, mode, suffix string,
	totalScannedFiles int,
) string {
	return strings.Join([]string{
		fmt.Sprintf("# ⚡ [S2.5 · UDF-v1] Wave Phản Biện Trước Khi Lên Plan (`DG0 :8200`) — %s: %s", qt.ExternalID, qt.Title),
		fmt.Sprintf("> 📌 **Task:** `%s` · **Stage:** `S2.5 PRE-PLAN CRITIQUE` · **Repos (%d):** `%s` · **Engine:** `convaiinnovations/laya (:8200 · 0 Token)` · **Gate DG0:** `PASS 🟢`",
			qt.ExternalID, len(affectedRepos), repoSummary),

// --- SAU KHI SỬA ---
func BuildUnifiedDG0CritiqueDoc(
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	symbolHits []RepoSymbolHit,
	affectedRepos []string,
	repoSummary, tier, mode, suffix string,
	totalScannedFiles int,
) string {
	primaryRepo := ""
	if len(affectedRepos) > 0 {
		primaryRepo = affectedRepos[0]
	}
	actualBranch := detectRepoBranchFast(primaryRepo)
	wantBranch := extractExplicitBranchFromPrompt(qt.Title + " " + qt.Description)
	gateBadge := "PASS 🟢"
	realityStatus := fmt.Sprintf("`VERIFIED` — Repo `%s` đang ở nhánh `%s`; quét `%d` files (`%d` symbol hits)", repoSummary, actualBranch, totalScannedFiles, len(symbolHits))
	contraStatus := "`PASS` — Không có giả định nhánh/model/CLI mâu thuẫn với thực tế trên đĩa"
	if wantBranch != "" && !strings.EqualFold(actualBranch, wantBranch) {
		gateBadge = "BLOCK 🔴 (BRANCH_MISMATCH)"
		realityStatus = fmt.Sprintf("`BLOCK` — Đề bài yêu cầu branch `%s` nhưng repo `%s` đang ở branch `%s`!", wantBranch, repoSummary, actualBranch)
		contraStatus = fmt.Sprintf("`BLOCK` — Phát hiện mâu thuẫn nhánh giữa Prompt (`%s`) và TargetRepo (`%s`)", wantBranch, actualBranch)
	}
```

---

### 2.3. Giải pháp cho `F5`: Bơm `CurrentBranch` & `git diff <base>...HEAD` thực tế vào `inspectRealGitStatusAndDiff` & `invokeArchitectLLM` (`internal/webui/task_llm_bridge.go:107-131` & `216-236`)
- Khi `git status --short` trống (vì các commit trên nhánh `feat/auto-recon-export` đã được commit), `inspectRealGitStatusAndDiff` tự động lấy thêm:
  - Tên nhánh hiện tại (`git -C <targetRepo> rev-parse --abbrev-ref HEAD`).
  - Thống kê và nội dung diff của nhánh so với `develop` / `main` (`git -C <targetRepo> diff --stat develop...HEAD` và `git -C <targetRepo> diff --unified=2 develop...HEAD`) để truyền vào `invokeArchitectLLM`.

```go
// File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_bridge.go:107-131
// --- TRƯỚC (HEAD hiện tại) ---
func (s *Server) inspectRealGitStatusAndDiff(targetRepo, taskKey string) (statusSummary string, diffStat string, patchBytes int64, patchExists bool) {
	if strings.TrimSpace(targetRepo) == "" {
		targetRepo = s.Root
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmdStatus := exec.CommandContext(ctx, "git", "-C", targetRepo, "status", "--short")
	if out, err := cmdStatus.Output(); err == nil {
		statusSummary = strings.TrimSpace(string(out))
	}

	cmdDiff := exec.CommandContext(ctx, "git", "-C", targetRepo, "diff", "--stat")
	if out, err := cmdDiff.Output(); err == nil {
		diffStat = strings.TrimSpace(string(out))
	}

// --- SAU KHI SỬA ---
func (s *Server) inspectRealGitStatusAndDiff(targetRepo, taskKey string) (statusSummary string, diffStat string, patchBytes int64, patchExists bool) {
	if strings.TrimSpace(targetRepo) == "" {
		targetRepo = s.Root
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	branchName := detectRepoBranchFast(targetRepo)
	cmdStatus := exec.CommandContext(ctx, "git", "-C", targetRepo, "status", "--short")
	if out, err := cmdStatus.Output(); err == nil {
		statusSummary = strings.TrimSpace(string(out))
	}
	if statusSummary == "" {
		statusSummary = fmt.Sprintf("(working tree clean on branch %s)", branchName)
	} else {
		statusSummary = fmt.Sprintf("[branch: %s]\n%s", branchName, statusSummary)
	}

	cmdDiff := exec.CommandContext(ctx, "git", "-C", targetRepo, "diff", "--stat")
	if out, err := cmdDiff.Output(); err == nil {
		diffStat = strings.TrimSpace(string(out))
	}
	if diffStat == "" && branchName != "" && branchName != "main" && branchName != "master" && branchName != "develop" {
		for _, baseRef := range []string{"develop...HEAD", "main...HEAD", "HEAD~1..HEAD"} {
			if out, err := exec.CommandContext(ctx, "git", "-C", targetRepo, "diff", "--stat", baseRef).Output(); err == nil && strings.TrimSpace(string(out)) != "" {
				diffStat = fmt.Sprintf("[branch diff %s on %s]\n%s", baseRef, branchName, strings.TrimSpace(string(out)))
				break
			}
		}
	}
```
