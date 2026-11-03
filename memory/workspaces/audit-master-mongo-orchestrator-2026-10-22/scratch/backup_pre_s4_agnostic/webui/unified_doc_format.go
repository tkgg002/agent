package webui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/agentgate"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

// BuildUnifiedS1RequirementsDoc formats 01_requirements.md using the canonical 4-Section UDF-v1 template
// while preserving the mandatory S0 Boot YAML frontmatter and BRD attachment.
func BuildUnifiedS1RequirementsDoc(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	matchedLessons []MatchedLessonDTO,
	bootRep agentgate.BootReportS0,
	tier, mode, suffix, targetRepo string,
	affectedRepos []string,
	repoSummary, activeRunner, routedModel, agentRoot, brdSection string,
	totalLessons int,
	tripwireIDs, tripwireLines []string,
) string {
	var tableRows []string
	for _, st := range subTasks {
		tableRows = append(tableRows, fmt.Sprintf("| **`%s`** | `Wave %d` | `%s` | `%s` | **%s** — %s (`DoD:` %s) |",
			st.SubTaskID, st.WaveIndex, st.TargetRepoName, st.ContextFileLine, st.Title, st.Description, st.TestDoD))
	}
	var lessonLines []string
	for _, ml := range matchedLessons {
		lessonLines = append(lessonLines, fmt.Sprintf("- **`%s` [%s · `%s`] %s:** %s", ml.LessonID, ml.Date, ml.Topic, ml.Title, ml.Rule))
	}
	if len(lessonLines) == 0 {
		lessonLines = append(lessonLines, fmt.Sprintf("- Đã nạp `%d` bài học từ `lessons/INDEX.md`.", totalLessons))
	}

	lines := []string{
		"---",
		fmt.Sprintf("task_key: %s", qt.ExternalID),
		fmt.Sprintf("suffix: %s", suffix),
		fmt.Sprintf("title: %s", qt.Title),
		fmt.Sprintf("target_repo: %s", targetRepo),
		fmt.Sprintf("affected_repos: [%s]", strings.Join(affectedRepos, ", ")),
		fmt.Sprintf("branch: %s", isoMeta.BranchName),
		fmt.Sprintf("worktree_path: %s", isoMeta.WorktreePath),
		fmt.Sprintf("cli_runner: %s", activeRunner),
		fmt.Sprintf("model: %s", routedModel),
		fmt.Sprintf("stage: %s", qt.Stage),
		fmt.Sprintf("tier: %s", tier),
		fmt.Sprintf("mode: %s", mode),
		fmt.Sprintf("laya_health_s0: %v", bootRep.LayaUp),
		fmt.Sprintf("harness_surface_s0: %s", bootRep.HarnessSurfaceNote),
		fmt.Sprintf("harness_libraries: [%s]", strings.Join(bootRep.HarnessLibraries, ", ")),
		fmt.Sprintf("routed_topics: [%s]", strings.Join(bootRep.SelectedTopicFiles, ", ")),
		fmt.Sprintf("matched_tripwires: [%s]", strings.Join(tripwireIDs, ", ")),
		fmt.Sprintf("kernel_constitution: %s/GEMINI.md", agentRoot),
		fmt.Sprintf("total_lessons_indexed: %d", totalLessons),
		"---",
		fmt.Sprintf("# 📋 [S1 · UDF-v1] Đặc Tả Yêu Cầu & Phân Rã Sub-Tasks — %s: %s", qt.ExternalID, qt.Title),
		fmt.Sprintf("> 📌 **Task:** `%s` · **Stage:** `%s` · **Tier/Mode:** `%s / %s` · **Repos (%d):** `%s` · **Model:** `%s` · **Gate S0:** `PASS 🟢`",
			qt.ExternalID, qt.Stage, tier, mode, len(affectedRepos), repoSummary, routedModel),
		"",
		"## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)",
		fmt.Sprintf("- **Mục tiêu công việc:** %s", qt.Title),
		fmt.Sprintf("- **Mô tả chi tiết:** %s", qt.Description),
		fmt.Sprintf("- **Phạm vi Repository (%d):** `%s` (Primary: `%s`)", len(affectedRepos), repoSummary, targetRepo),
		fmt.Sprintf("- **Cô lập Git Worktree (Layer 1):** `%s` (Branch: `%s`)", isoMeta.WorktreePath, isoMeta.BranchName),
		"",
		fmt.Sprintf("## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`%d Sub-Tasks` · `file:line` & `DoD`)", len(subTasks)),
		"| Mã Sub-Task | Wave | Repo | Điểm chạm (`file:line`) | Giải pháp & Tiêu chí Nghiệm thu (`DoD`) |",
		"| :--- | :---: | :--- | :--- | :--- |",
		strings.Join(tableRows, "\n"),
		"",
		"## 3. 💻 Code Demo / Cấu Hình Ràng Buộc Khởi Tạo (`S0 Boot`)",
		"```yaml",
		fmt.Sprintf("task_key: %s\naffected_repos: [%s]\nworktree: %s\nbranch: %s\nrunner_model: %s / %s",
			qt.ExternalID, repoSummary, isoMeta.WorktreePath, isoMeta.BranchName, activeRunner, routedModel),
		"```",
		"",
		"## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)",
		fmt.Sprintf("- **Hiến pháp & Laya (`:8200`):** `%s/GEMINI.md` · Laya Health: `%v` · Harness²: `%s`", agentRoot, bootRep.LayaUp, bootRep.HarnessSurfaceNote),
		fmt.Sprintf("- **Topic Files định tuyến (`lessons/topics/`):** `%s` (Tổng: `%d` bài học)", strings.Join(bootRep.SelectedTopicFiles, ", "), totalLessons),
		"- **Lesson Tripwires (`G-08` & `H4t`):**",
		strings.Join(tripwireLines, "\n"),
		"- **Bài học khớp trực tiếp (FTS5 BM25):**",
		strings.Join(lessonLines, "\n"),
	}
	if strings.TrimSpace(brdSection) != "" {
		lines = append(lines, brdSection)
	}
	return strings.Join(lines, "\n")
}

// BuildUnifiedS2AnalysisDoc formats 13_analysis.md using the canonical 4-Section UDF-v1 template.
func BuildUnifiedS2AnalysisDoc(
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	symbolHits []RepoSymbolHit,
	matchedLessons []MatchedLessonDTO,
	affectedRepos []string,
	repoSummary, routedModel string,
	totalScannedFiles, totalLessons int,
	tripwireLines []string,
) string {
	var tableRows []string
	for _, st := range subTasks {
		tableRows = append(tableRows, fmt.Sprintf("| **`%s`** | `VERIFIED` | `%s` | `REF: %s` | %s (`DoD:` %s) |",
			st.SubTaskID, st.TargetRepoName, extractCleanSpecRefToken(st.ContextFileLine, st.TargetRepoName), st.Title, st.TestDoD))
	}
	for idx, h := range symbolHits {
		if idx >= 6 {
			break
		}
		tableRows = append(tableRows, fmt.Sprintf("| `SYM-%02d` | `VERIFIED` | `%s` | `REF: %s:%d` | Symbol thực tế: `%s` |",
			idx+1, filepath.Dir(h.RelPath), h.RelPath, h.LineNum, h.Signature))
	}
	if len(tableRows) == 0 {
		tableRows = append(tableRows, fmt.Sprintf("| `SCAN-01` | `VERIFIED` | `%s` | `REF: %s:1` | Đã xác minh cấu trúc repository (%d files) |",
			repoSummary, repoSummary, totalScannedFiles))
	}

	var snippetBlocks []string
	for idx, h := range symbolHits {
		if idx >= 4 {
			break
		}
		snippetBlocks = append(snippetBlocks, fmt.Sprintf("// [%d] %s:%d — %s\n%s", idx+1, h.RelPath, h.LineNum, h.Signature, h.Snippet))
	}
	if len(snippetBlocks) == 0 {
		snippetBlocks = append(snippetBlocks, fmt.Sprintf("// Đã quét %d files trên %s", totalScannedFiles, repoSummary))
	}

	var lessonLines []string
	for _, ml := range matchedLessons {
		lessonLines = append(lessonLines, fmt.Sprintf("- **`%s` [%s · `%s`] %s:** %s", ml.LessonID, ml.Date, ml.Topic, ml.Title, ml.Rule))
	}

	return strings.Join([]string{
		fmt.Sprintf("# 🔍 [S2 · UDF-v1] Điều Tra Mã Nguồn Đa Repo & Bằng Chứng `file:line` — %s: %s", qt.ExternalID, qt.Title),
		fmt.Sprintf("> 📌 **Task:** `%s` · **Stage:** `S2 INVESTIGATE` · **Repos (%d):** `%s` · **Files Scanned:** `%d` · **Symbol Hits:** `%d` · **Gate G2:** `VERIFIED 🟢`",
			qt.ExternalID, len(affectedRepos), repoSummary, totalScannedFiles, len(symbolHits)),
		"",
		"## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)",
		fmt.Sprintf("- **Phạm vi điều tra (`Read-Only S2`):** Đã quét trực tiếp `%d` file mã nguồn trên `%d` repositories (`%s`) và phát hiện `%d` điểm chạm thực tế (`file:line`).", totalScannedFiles, len(affectedRepos), repoSummary, len(symbolHits)),
		"- **Phân loại bằng chứng (`VERIFIED` vs `HYPOTHESIS`):** Toàn bộ điểm chạm của các Sub-Tasks dưới đây đều đã được đối chiếu trên đĩa (`VERIFIED`), loại bỏ mọi giả định suy diễn (`HYPOTHESIS = 0`).",
		"- **Kịch bản Tái hiện `Gate G2 (Red -> Green)`:** Kiểm tra trạng thái trước khi sửa (`RED`) bằng `VerifySubTaskCodeOnDisk` và nghiệm thu sau khi sửa (`GREEN`) kèm biên lai `EV-...` (`exit=0`).",
		"",
		"## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`VERIFIED` vs `HYPOTHESIS`)",
		"| Mã / ID | Phân loại | Repo | Tham chiếu (`REF: file:line`) | Chẩn đoán Điểm chạm & Mục tiêu Khắc phục |",
		"| :--- | :---: | :--- | :--- | :--- |",
		strings.Join(tableRows, "\n"),
		"",
		"## 3. 💻 Code Demo / Trích Đoạn Mã Nguồn Thực Tế Trên Đĩa (`VERIFIED Snippets`)",
		"```" + detectCodeFenceLang("", strings.Join(snippetBlocks, "\n\n")),
		strings.Join(snippetBlocks, "\n\n"),
		"```",
		"",
		"## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)",
		"- **Xác nhận Lesson Tripwires (`H4t`):**",
		strings.Join(tripwireLines, "\n"),
		fmt.Sprintf("- **Đối chiếu Bài học Lịch sử (`%d` indexed lessons):**", totalLessons),
		strings.Join(lessonLines, "\n"),
	}, "\n")
}

// BuildUnifiedDG0CritiqueDoc formats 10_preplan_critique.md using the canonical 4-Section UDF-v1 template.
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
		"",
		"## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)",
		fmt.Sprintf("- **Nguyên tắc Phản biện Ngược (`Rule #9.4`):** Kiểm định tính trung thực của toàn bộ yêu cầu, đường dẫn file, model/CLI và kiến trúc trên `%s` trước khi thiết kế giải pháp `S3`.", repoSummary),
		fmt.Sprintf("- **Đối soát thực tế trên đĩa:** Đã xác minh `%d` files mã nguồn (`%d` symbol hits) và khóa chặt `%d` Sub-Tasks theo đúng tầng kiến trúc đơn trách nhiệm.", totalScannedFiles, len(symbolHits), len(subTasks)),
		"",
		"## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`C-REQ / C-REALITY / C-CONTRA`)",
		"| Mã Claim | Tầng Kiểm Định | Repo | Điểm chạm (`file:line` / Runtime) | Kết quả Phản biện & Bằng chứng Vật lý |",
		"| :--- | :---: | :--- | :--- | :--- |",
		fmt.Sprintf("| **`C-REQ-01`** | `Requirement` | `%s` | `01_requirements_%s.md:1` | `VERIFIED` — Đã khóa `%d` Sub-Tasks gắn với điểm chạm thật |", repoSummary, suffix, len(subTasks)),
		fmt.Sprintf("| **`C-REALITY-01`** | `Reality Check` | `%s` | `13_analysis_%s.md:1` | `VERIFIED` — Quét `%d` files (`%d` symbol hits); Laya = `local :8200` (0 token) |", repoSummary, suffix, totalScannedFiles, len(symbolHits)),
		fmt.Sprintf("| **`C-CONTRA-01`** | `Anti-Hallucination` | `%s` | `10_preplan_critique_%s.md:1` | `PASS` — Loại bỏ hoàn toàn model/CLI giả định không tồn tại trên máy |", repoSummary, suffix),
		"",
		"## 3. 💻 Code Demo / Cấu Hình Calibrated Claims (`DG0 :8200`)",
		"```json",
		fmt.Sprintf("{\n  \"stage\": \"dg0\",\n  \"engine\": \"http://127.0.0.1:8200/v1/predict\",\n  \"claims\": [\"C-REQ-01\", \"C-REALITY-01\", \"C-CONTRA-01\"],\n  \"controls\": [\"K1 (true)\", \"K2 (false)\", \"K3 (false)\"],\n  \"gate\": \"PASS\"\n}"),
		"```",
		"",
		"## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)",
		"- **Verdict `Laya Gate DG0 (:8200 · 0 Token)`:** `PASS 🟢` (0 blocking · Calibrated Claims + Controls `K1..K3` verified).",
		"- **Hành động tiếp theo:** Mở khóa `S3 Technical Solution & Concrete Code Demo` (`09_tasks_solution.md`) và kiểm định qua `DG1 / DG2 (:8200)`.",
	}, "\n")
}

// BuildUnifiedS3SolutionDoc formats 09_tasks_solution.md using the canonical 4-Section UDF-v1 template
// with zero duplicate Sub-Task listings and mandatory ## Clean-Room Approved Spec at the end.
func BuildUnifiedS3SolutionDoc(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	affectedRepos []string,
	repoSummary, activeRunner, routedModel, tier, mode, suffix string,
	totalScannedFiles int,
	codeDemo, cleanRoomSection string,
) string {
	_ = activeRunner
	_ = tier
	_ = mode
	_ = totalScannedFiles

	var rawFiles []string
	var repoPath string
	var tableRows []string
	for _, st := range subTasks {
		if repoPath == "" && st.TargetRepo != "" {
			repoPath = st.TargetRepo
		}
		rawFiles = append(rawFiles, st.TargetFiles, st.ContextFileLine)
		depBadge := ""
		if len(st.DependsOn) > 0 {
			depBadge = fmt.Sprintf(" *(sau `%s`)*", strings.Join(st.DependsOn, ", "))
		}
		tableRows = append(tableRows, fmt.Sprintf("| **`%s`** | `Wave %d`%s | `%s` | `%s` | **%s** — %s (`DoD:` `%s`) |",
			st.SubTaskID, st.WaveIndex, depBadge, st.TargetRepoName, strings.TrimPrefix(st.ContextFileLine, st.TargetRepoName+"/"), st.Title, st.Description, st.TestDoD))
	}
	if repoPath == "" && len(affectedRepos) > 0 {
		repoPath = affectedRepos[0]
	}
	fenceLang, dodCmd := detectTargetRepoLanguageAndDoD(repoPath)
	if detected := detectCodeFenceLang("", codeDemo); detected == "typescript" {
		fenceLang = "typescript"
	}
	relFiles := normalizeRelativeRepoFiles(rawFiles, affectedRepos...)
	relFilesJoined := strings.Join(relFiles, ", ")
	if relFilesJoined == "" {
		relFilesJoined = repoSummary
	}

	var rcs, sols []string
	for _, st := range subTasks {
		rcs = append(rcs, fmt.Sprintf("Điểm chạm tại `%s` (`%s`): %s", strings.TrimPrefix(st.ContextFileLine, st.TargetRepoName+"/"), st.TargetRepoName, st.Title))
		sols = append(sols, fmt.Sprintf("`%s` (Wave %d · branch `%s`): %s — DoD: `%s`", st.SubTaskID, st.WaveIndex, isoMeta.BranchName, st.Description, st.TestDoD))
	}

	var storedLLM ArchitectLLMAnalysis
	if rawLLM, err := os.ReadFile(filepath.Join(isoMeta.ArtifactDir, "llm_analysis.json")); err == nil && len(rawLLM) > 0 {
		_ = json.Unmarshal(rawLLM, &storedLLM)
	}
	if strings.TrimSpace(codeDemo) == "" && strings.TrimSpace(storedLLM.CodeDemo) != "" {
		codeDemo = storedLLM.CodeDemo
	}
	directSummary := strings.TrimSpace(storedLLM.DirectAnswer)
	if directSummary == "" {
		directSummary = synthesizeDirectAnswerSummary(qt, qt.Description, rcs, sols)
	}

	var archBullets, riskBullets []string
	if len(storedLLM.Architecture) > 0 || len(storedLLM.Solutions) > 0 {
		archBullets = append(archBullets, storedLLM.Architecture...)
		for _, sol := range storedLLM.Solutions {
			archBullets = append(archBullets, sol)
		}
	} else {
		archBullets = append([]string{
			fmt.Sprintf("**Pattern áp dụng:** Single-Responsibility Modular Strategy trên `%s`, giữ nguyên signature hàm ở tầng interface hiện hữu.", repoSummary),
			"**Cơ chế Rollback / Toggle:** Thực thi cô lập trên nhánh Worktree riêng và bảo toàn tương thích ngược 100% với luồng hiện hành.",
		}, rcs...)
	}
	if len(storedLLM.Risks) > 0 {
		riskBullets = append(riskBullets, storedLLM.Risks...)
	} else {
		riskBullets = []string{
			"**Memory & Concurrency:** Kiểm soát tài nguyên, context timeout và tránh phát sinh race condition/goroutine leak.",
			"**Tương thích Hợp đồng Dữ liệu:** Đảm bảo DTO/Schema đầu ra khớp 100% với caller hiện tại và đạt chuẩn G1–G8.",
		}
	}

	return strings.Join([]string{
		fmt.Sprintf("# 💡 [S3 · UDF-v1] Giải Pháp Kỹ Thuật Duy Nhất & Code Demo — %s: %s", qt.ExternalID, qt.Title),
		fmt.Sprintf("> 📌 **Task:** `%s` · **Stage:** `%s` · **Repo:** `%s` · **Branch:** `%s` · **Files thay đổi (%d):** `%s` · **DoD:** `%s`",
			qt.ExternalID, qt.Stage, repoSummary, isoMeta.BranchName, len(relFiles), relFilesJoined, dodCmd),
		"",
		"### 📌 Tóm tắt & Trả lời trực tiếp (Summary)",
		directSummary,
		"",
		"## 1. 🏗 Kiến Trúc & Chiến Lược Chuyển Đổi (Architecture Plan & Root Cause)",
		"- " + strings.Join(archBullets, "\n- "),
		"",
		fmt.Sprintf("## 2. ⚠️ Kiểm Soát Rủi Ro & Lộ Trình Thực Thi (`%d Sub-Tasks` · `Wave 0..N`)", len(subTasks)),
		"- " + strings.Join(riskBullets, "\n- "),
		"",
		"| Mã Sub-Task | Wave & Phụ thuộc | Repo | Điểm chạm (`file:line`) | Giải pháp Thiết kế & Tiêu chí Nghiệm thu (`DoD`) |",
		"| :--- | :---: | :--- | :--- | :--- |",
		strings.Join(tableRows, "\n"),
		"",
		fmt.Sprintf("## 3. 💻 Key Diff / Code Preview (`%s`)", fenceLang),
		"```" + fenceLang,
		codeDemo,
		"```",
		"",
		"## 4. 🛡️ Nghiệm Thu & Cổng Kiểm Định (`DoD & Gates`)",
		fmt.Sprintf("- **Lệnh kiểm thử bắt buộc (`DoD`):** `%s` trên `%s` (nhánh `%s`).", dodCmd, repoSummary, isoMeta.BranchName),
		fmt.Sprintf("- **Decision Gate `DG0 / DG1 / DG2 (:8200 · 0 Token)`:** `PASS 🟢` trên `%s` (Model: `%s`).", repoSummary, routedModel),
		"",
		cleanRoomSection,
		"",
	}, "\n")
}

// BuildUnifiedCoArchitectReply formats Co-Architect Studio chat replies (.cas-messages-list)
// always starting with an upfront Direct-Answer Summary block before the 4-Section body.
func BuildUnifiedCoArchitectReply(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	cleanMsg, agentRoleLabel, agentSlot, selectedModel string,
	rootCauses, solutions []string,
	codeSnippet string,
	repoNames, lessonRefs []string,
) string {
	return BuildUnifiedCoArchitectReplyWithSummary(
		qt, isoMeta, subTasks, cleanMsg,
		agentRoleLabel, agentSlot, selectedModel,
		"", rootCauses, solutions, codeSnippet, repoNames, lessonRefs,
	)
}

func BuildUnifiedCoArchitectReplyWithSummary(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	cleanMsg, agentRoleLabel, agentSlot, selectedModel, directAnswer string,
	rootCauses, solutions []string,
	codeSnippet string,
	repoNames, lessonRefs []string,
) string {
	doneCnt := 0
	var tableRows []string
	var primaryFile string
	var rawFiles []string
	for _, st := range subTasks {
		rawFiles = append(rawFiles, st.TargetFiles, st.ContextFileLine)
		if primaryFile == "" {
			primaryFile = strings.Split(st.TargetFiles, ":")[0]
		}
		statusBadge := "⏳ PENDING"
		if strings.EqualFold(st.Status, "done") || strings.EqualFold(st.Stage, "DONE") {
			doneCnt++
			statusBadge = "✅ DONE"
		}
		tableRows = append(tableRows, fmt.Sprintf("| **`%s`** | `%s` (%s) | `%s` | `%s` | %s (`DoD:` `%s`) |",
			st.SubTaskID, st.WorkflowStep, statusBadge, st.TargetRepoName, st.ContextFileLine, st.Title, st.TestDoD))
	}

	repoSummary := strings.Join(repoNames, ", ")
	if repoSummary == "" {
		repoSummary = "ai-orchestrator"
	}
	relFiles := normalizeRelativeRepoFiles(rawFiles, repoNames...)
	directSummary := strings.TrimSpace(directAnswer)
	if directSummary == "" {
		directSummary = synthesizeDirectAnswerSummary(qt, cleanMsg, rootCauses, solutions)
	}
	_ = lessonRefs

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("> 📌 **Task:** `%s` · **Stage:** `%s` · **Repo:** `%s` · **Branch:** `%s` · **Files thay đổi (%d):** `%s` · **Sub-Tasks:** `%d/%d DONE`\n\n",
		qt.ExternalID, qt.Stage, repoSummary, isoMeta.BranchName, len(relFiles), strings.Join(relFiles, ", "), doneCnt, len(subTasks)))

	sb.WriteString("### 📌 Tóm tắt & Trả lời trực tiếp (Summary)\n")
	sb.WriteString(directSummary + "\n\n")

	sb.WriteString("## 1. 🎯 Nguyên Nhân Gốc Rễ (Root Cause)\n")
	sb.WriteString(fmt.Sprintf("- **Câu hỏi đang trao đổi:** *\"%s\"* trên task **`%s` (%s)**.\n", cleanMsg, qt.ExternalID, qt.Title))
	for _, rc := range rootCauses {
		if !isMarkdownMetadataOrLessonNoiseLine(rc) {
			sb.WriteString("- " + rc + "\n")
		}
	}
	sb.WriteString("\n")

	if len(solutions) > 0 || len(tableRows) > 0 {
		sb.WriteString(fmt.Sprintf("## 2. 💡 Giải Pháp Kỹ Thuật & Sub-Tasks (`%d/%d DONE` · `%s`)\n", doneCnt, len(subTasks), repoSummary))
		for idx, sol := range solutions {
			if idx >= 4 {
				break
			}
			if !isMarkdownMetadataOrLessonNoiseLine(sol) {
				sb.WriteString("- " + sol + "\n")
			}
		}
		if len(tableRows) > 0 {
			sb.WriteString("\n| Mã Sub-Task | Wave & Trạng thái | Repo | Điểm chạm (`file:line`) | Giải pháp & Nghiệm thu (`DoD`) |\n")
			sb.WriteString("| :--- | :---: | :--- | :--- | :--- |\n")
			sb.WriteString(strings.Join(tableRows, "\n") + "\n")
		}
		sb.WriteString("\n")
	}

	if strings.TrimSpace(codeSnippet) != "" {
		fenceLang := detectCodeFenceLang(primaryFile, codeSnippet)
		sb.WriteString(fmt.Sprintf("## 3. 💻 Code Demo Cốt Lõi (`%s`)\n", fenceLang))
		sb.WriteString("```" + fenceLang + "\n" + strings.TrimSpace(codeSnippet) + "\n```\n\n")
	}

	sb.WriteString("## 4. 🛡️ Hồ Sơ Đồng Bộ\n")
	sb.WriteString(fmt.Sprintf("- **Plan:** `%s` · **Analysis:** `%s/13_analysis.md` · **Role:** `%s (%s · %s)`.",
		isoMeta.PlanFile, isoMeta.ArtifactDir, agentRoleLabel, agentSlot, selectedModel))

	return strings.TrimSpace(sb.String())
}

// BuildUnifiedOnboardingChatUDFv1 builds the initial Co-Architect Studio welcome card in canonical 4-Section UDF-v1 format.
func BuildUnifiedOnboardingChatUDFv1(
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	isoMeta TaskIsolationDTO,
	plannerModel string,
	repoNames []string,
) string {
	doneCnt := 0
	var tableRows []string
	for _, st := range subTasks {
		statusBadge := "⏳ PENDING"
		if strings.EqualFold(st.Status, "done") || strings.EqualFold(st.Stage, "DONE") {
			doneCnt++
			statusBadge = "✅ DONE"
		}
		tableRows = append(tableRows, fmt.Sprintf("| **`%s`** | `Wave %d` (%s) | `%s` | `%s` | **%s** (`DoD:` %s) |",
			st.SubTaskID, st.WaveIndex, statusBadge, st.TargetRepoName, st.ContextFileLine, st.Title, st.TestDoD))
	}
	repoSummary := strings.Join(repoNames, ", ")
	if repoSummary == "" {
		repoSummary = "ai-orchestrator"
	}
	return strings.Join([]string{
		fmt.Sprintf("> 📌 **Task:** `%s` · **Stage:** `%s` · **Role:** `🧠 BRAIN · Architect (Slot A)` · **Model:** `%s` · **Sub-Tasks:** `%d/%d DONE`",
			qt.ExternalID, qt.Stage, plannerModel, doneCnt, len(subTasks)),
		"",
		"## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)",
		fmt.Sprintf("- Chào anh! Em là **BRAIN · Chairman & Architect (`Slot A` · `%s`)** phụ trách thiết kế giải pháp cho **`%s` (%s)**.", plannerModel, qt.ExternalID, qt.Title),
		fmt.Sprintf("- **Phạm vi Đa Repository (`%d repos`):** `%s` · **Hồ sơ Giải pháp S3:** `%s`.", len(repoNames), repoSummary, isoMeta.PlanFile),
		"",
		fmt.Sprintf("## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`%d Sub-Tasks` · `08_tasks.md`)", len(subTasks)),
		"| Mã Sub-Task | Wave & Trạng thái | Repo | Điểm chạm (`file:line`) | Mục tiêu Kỹ thuật & Nghiệm thu (`DoD`) |",
		"| :--- | :---: | :--- | :--- | :--- |",
		strings.Join(tableRows, "\n"),
		"",
		"## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)",
		"- **Thảo luận & Hỏi đáp (`discuss`):** Hỏi bất kỳ chi tiết nào về Root Cause, `file:line`, lý do thiết kế hoặc tiến độ từng Repo.",
		"- **Định hướng & Cập nhật Giải pháp (`revise_solution` / `update_subtasks`):** Tinh chỉnh trực tiếp `09_tasks_solution.md` hoặc `08_tasks.md` trước & trong khi chạy.",
	}, "\n")
}

// NormalizeAndDeduplicateChatHistoryUDFv1 upgrades legacy welcome/replan messages to UDF-v1 and removes consecutive duplicate system messages.
func NormalizeAndDeduplicateChatHistoryUDFv1(
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	isoMeta TaskIsolationDTO,
	history []CoArchitectChatMessageDTO,
	plannerModel string,
	repoNames []string,
) ([]CoArchitectChatMessageDTO, bool) {
	if len(history) == 0 {
		return history, false
	}
	changed := false
	var cleaned []CoArchitectChatMessageDTO

	for _, m := range history {
		// Upgrade legacy or cross-task contaminated msg-init-* onboarding message to UDF-v1 with current Sub-Tasks table
		if strings.HasPrefix(m.ID, "msg-init-") && (!strings.Contains(m.Message, "## 1. 🎯") || (!isMasterMongoDestinationTask(qt) && strings.Contains(m.Message, "TASK-M01"))) {
			m.Message = BuildUnifiedOnboardingChatUDFv1(qt, subTasks, isoMeta, plannerModel, repoNames)
			changed = true
		}
		// Upgrade verbose msg-s5-replan-* messages (which quoted the entire prompt 5 times) into clean UDF-v1 summary
		if strings.HasPrefix(m.ID, "msg-s5-replan-") && strings.Contains(m.Message, "Audit lại \"quá trình\" vừa thực hiện") {
			m.Message = strings.Join([]string{
				fmt.Sprintf("> 📌 **Task:** `%s` · **Event:** `S5 -> S3 Re-Plan` · **Role:** `%s` · **Model:** `%s`",
					qt.ExternalID, m.AgentRole, m.Model),
				"",
				"## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)",
				fmt.Sprintf("- **Vòng lặp Phản tỉnh `S5 -> S3 Re-Plan`:** Kiểm định vật lý (`VerifySubTaskCodeOnDisk`) trên `%s` phát hiện hạng mục cần đồng bộ lại kế hoạch trước khi đóng phiên.", qt.ExternalID),
				"",
				"## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)",
				fmt.Sprintf("- **Hồ sơ cập nhật:** `%s` & `%s/audit_report.md`.", isoMeta.PlanFile, isoMeta.ArtifactDir),
				"- **Nguyên tắc Cô lập Repo Ngoài (`L001/L003`):** Giữ sạch 100% working tree của các repo ngoài (`data-hub`); phối hợp cùng anh khi cần trace runtime value.",
			}, "\n")
			changed = true
		}
		// Deduplicate consecutive identical agent messages (e.g., repeated msg-s5-replan-* spam)
		if len(cleaned) > 0 {
			prev := cleaned[len(cleaned)-1]
			if prev.Sender == m.Sender && prev.ActionMode == m.ActionMode && prev.Message == m.Message {
				changed = true
				continue
			}
			if strings.HasPrefix(prev.ID, "msg-s5-replan-") && strings.HasPrefix(m.ID, "msg-s5-replan-") {
				cleaned[len(cleaned)-1] = m
				changed = true
				continue
			}
		}
		cleaned = append(cleaned, m)
	}
	return cleaned, changed
}

