---
tier: T2
mode: propose-only
target_project: ai-orchestrator
suffix: transparency_ui_declutter_2026_11_04
created_at: 2026-11-04T19:32:00+07:00
---

# 09. Technical Solution & Code Demo — Minh Bạch Thực Thi Model Theo Phiên/Task, Khởi Tạo Memory Định Nghĩa Repo & Tinh Gọn UI/UX (`ai-orchestrator`)

## 1. Bảng Tổng Hợp Issues & Điểm Nghẽn (`F1` – `F4`)

| Mã | Mức độ | Vị trí (`file:line`) | Nguyên nhân gốc rễ (Root Cause) | Trạng thái & Hành động |
| :--- | :---: | :--- | :--- | :--- |
| **`F1`** | **CRITICAL** | `internal/webui/task_llm_bridge.go:241-298`, `internal/webui/subtask_codegen_muscle.go:250-272`, `internal/webui/subtask_edit_service.go:311-333` | **Thiếu sổ cái minh bạch từng phiên gọi Model trên FE:** Khi gọi `agRunner.Complete(llmCtx, runner.CompletionRequest{...})`, hệ thống không lưu lại `SystemPrompt`, `UserPrompt` thực tế gửi đi, `RawResponse` nguyên bản model trả về, `DurationMs`, `SessionID` (`convID`), hay trạng thái thật (`REAL_LLM_OK` vs `LLM_ERROR` vs `STATIC_FALLBACK`) theo từng Task. Người dùng trên FE không biết mỗi phiên gọi model nào, prompt ra sao, trả về kết quả gì. | **Pending (`APPROVE`):** Tạo `internal/webui/task_llm_trace.go` lưu `.orchestrator/tasks/<taskKey>/llm_call_ledger.json` & hiển thị trực tiếp Tab/Bảng **Minh Bạch Phiên Gọi Model** trên FE |
| **`F2`** | **CRITICAL** | `internal/webui/task_inspector.go:2794-2843`, `internal/webui/task_llm_bridge.go:191-211`, `internal/webui/server.go:778,2124` | **Không kiểm tra/khởi tạo Memory Define của Repo & đi lòng vòng qua các task khác:** Mỗi lần chạy Task trên 1 repo, hệ thống không kiểm tra xem trong Memory (`agent/memory/global/project/<repo>/repo_definition.json` / `.orchestrator/memory/repos/<repo>/repo_definition.json`) đã có bản định nghĩa (`define`) của repo đó chưa mà luôn quét mò lại từ đầu, đồng thời gọi `sorter.TriageAll` tìm kiếm FTS5 qua lịch sử các task khác và tự suy diễn liên kết chéo. | **Pending (`APPROVE`):** Tạo `internal/webui/repo_memory_store.go` (`LoadOrInitRepoMemoryDefinition`) đọc/init `repo_definition.json` cho từng repo, cắt bỏ liên kết/suy đoán chéo task |
| **`F3`** | **HIGH** | `internal/webui/static/index.html:13-307`, `internal/webui/static/app.js:561-1023,1652-1778` | **UI/UX quá rối, lặp đi lặp lại cùng 1 thông tin 6–7 lần trên 1 màn hình:** Cùng 1 thông tin (`Task ID`, `Stage`, `Target Repo`, `Branch`, `Worktree`, `Model`, nút `Chạy Autopilot`, nút `APPROVE`) bị nhồi lặp lại ở 6 khối liên tiếp: (1) Topbar, (2) Context Bar, (3) `autopilot-header-row`, (4) `live-telemetry-hud` 4 thẻ to, (5) `agent-kernel-card`, (6) `isolation-inspector-card`, (7) `autopilot-flow`, (8) Sidebar `17 bước` & `11 Agent Slots`. | **Pending (`APPROVE`):** Rà soát từng label text, ẩn/cắt bỏ toàn bộ khối trang trí lặp lại, gom về 1 thanh Context + 1 khung chính trực quan |
| **`F4`** | **MEDIUM** | `internal/webui/static/app.js:1111-1280,1505-1558` | **Thẻ Sub-Task & Fractal Inspector quá nặng nề:** Mỗi thẻ Sub-Task gánh 6 badge ở dòng trên + 4 pill ở dòng dưới + 2 dropdown chọn lại Role/Model + 4 nút bấm; khung `subtask-fractal-inspector` luôn bung sẵn 3 cột hợp đồng dài dòng đè lên danh sách Sub-Tasks. | **Pending (`APPROVE`):** Tinh gọn thẻ Sub-Task chỉ giữ thông tin cốt lõi (`ID`, `Title`, `Repo`, `File:Line`, `Model`, `Stage`, nút `Chạy/Xem`); thu gọn `subtask-fractal-inspector` chỉ mở khi bấm xem chi tiết |

---

## 2. Bảng Kiểm Toán Từng Label Text & Khối UI Trên Giao Diện (`Label-by-Label UI Audit`)

| Khu vực | Label / Khối UI hiện tại (`index.html` / `app.js`) | Vấn đề hiện tại | Quyết định (`GIỮ` / `THU GỌN` / `LOẠI BỎ`) | Cách hiển thị mới trực quan |
| :--- | :--- | :--- | :---: | :--- |
| **1. Topbar (`index.html:13-54`)** | `AI-ORCHESTRATOR` + 3 Tab màn hình + `CLI Runner:` + `⚙ Setup CLI & Model` + `✈️ Setup Telegram` + `gate-badges-strip` + `▶ Chạy Autopilot` + `+ Giao việc mới` | Nút `▶ Chạy Autopilot` và `gate-badges-strip` trên Topbar trùng lặp với thanh điều khiển Task ngay bên dưới. | **THU GỌN** | Topbar chỉ giữ: Tên App, 3 Tab chuyển màn hình, chọn `CLI Runner`, nút `⚙ CLI/Model`, `✈️ Telegram`, và nút `+ Giao việc mới`. |
| **2. Thanh Task Đang Chọn (`index.html:59-89`)** | `🎯 Task đang chọn:` + Select + `Stage` + `Repos · Worktree · Branch` + **9 nút bấm dài dòng** (`1. Plan S1–S3 (Stop @ NEEDS_YOU)`, `2. ✓ APPROVE & Chạy S4–S5`, `3. Implement S4`, `4. QA & DG3 (S5)`, `⚡ Autopilot (2-Phase Gate)`, `⏹ Dừng`, `✏️ Sửa Task`, `🗑 Xóa Task`, `✕ Reset`) | 9 nút bấm dàn hàng ngang với chữ quá dài (`(Stop @ NEEDS_YOU)`, `(2-Phase Gate)`), vừa trùng nhau vừa làm vỡ dòng trên màn hình hẹp. | **THU GỌN** | Giữ: Chọn Task + Badge `Stage` + Badge `📦 Repo` + Badge **`🧠 Repo Memory: ĐÃ NẠP / VỪA INIT`** + 4 nút hành động rõ nghĩa theo ngữ cảnh: **`▶ Chạy S1–S3`**, **`✓ APPROVE & Chạy S4–S5`**, **`⏹ Dừng`** (chỉ hiện khi đang chạy), **`✏️ Sửa`**, **`🗑 Xóa`**. |
| **3. Sidebar Trái (`index.html:123-160`)** | 3 Section chồng lên nhau: (1) `Hàng Đợi Task (Theo Repo)`, (2) `Quy trình /agent · 17 Bước`, (3) `Agent Slots (A..K) · Task` | Section 2 (17 bước tĩnh) và Section 3 (11 slot tĩnh) chiếm mất 60% chiều cao Sidebar, ép danh sách Task còn một khe hẹp; nút `+ Task` trùng với `+ Giao việc mới`. | **LOẠI BỎ Section 2 & 3 khỏi Sidebar** | Dành **100% chiều cao Sidebar trái** cho **`Hàng Đợi Task (Theo Repo)`** + 4 tab lọc nhanh (`Tất cả`, `⚡ Chạy`, `⏸ Duyệt`, `✓ Xong`). |
| **4. Thẻ Task ở Sidebar (`app.js:891-908`)** | `#1 TASK-100` + 4 nút (`⚡ All`, `🧩 Từng Sub`, `✏️`, `🗑`) + `Title` + `⏸ DUYỆT S3` + `🧩 0/3 Sub` + `📝 0f` | Quá nhiều nút nhỏ trên thẻ hẹp (`310px`) làm cắt chữ tiêu đề Task. | **THU GỌN** | Dòng 1: `TASK-100` + Badge `Stage` + Nút chạy nhanh (`▶` / `✓`). Dòng 2: Tiêu đề Task rõ ràng. Dòng 3: `🧩 0/3 Sub` · `🔍 N phiên gọi Model`. |
| **5. Khối Tiêu Đề Phụ (`index.html:169-181`)** | `AUTOPILOT & PER-TASK ISOLATION ENGINE` + `Planner: gemini-4-argon · Coder: gemini-4-argon · Scout: $0.00` + `💬 Đối thoại Giải pháp` + `⚙ Đổi CLI / Model` | Khẩu hiệu vô nghĩa, lặp lại tên Model và lặp lại 2 nút bấm đã có ở trên/dưới. | **LOẠI BỎ 100%** | Ẩn hoàn toàn khối `autopilot-header-row`. |
| **6. Khối Telemetry HUD (`index.html:187-227`)** | `⚡ LIVE TELEMETRY HUD` + 4 hộp chữ in hoa khổng lồ (`1. ĐANG CHẠY CÁI GÌ (WHAT IS RUNNING)`, `2. AI / ROLE / MODEL NÀO LÀM`, `3. ĐANG CHẠY Ở ĐÂU`, `4. CHẠY NHƯ THẾ NÀO & TIẾN ĐỘ TỪNG REPO`) | Khi Task đang đứng yên (`READY` / `NEEDS_YOU` / `DONE`), 4 hộp này vẫn chiếm 180px chiều dọc chỉ để lặp lại tên Repo, tên Branch và tên Model! | **THU GỌN MẠNH** | Thu gọn thành **1 thanh tóm tắt 1 dòng**; 4 thẻ chi tiết chỉ hiện khi tiến trình thực sự đang chạy (`isRunning == true`). |
| **7. Khối `/agent Kernel Card` (`index.html:230-251`)** | `🛡️ /agent KERNEL ACTIVE` + `BRAIN — Chairman & Architect...` + `📚 339 Lessons (INDEX.md)` + `Skills & Lifecycle Hooks (H1..H7)` + `Top Bài Học /agent Khớp Với Task Này` | Lặp lại ở mọi Task y hệt nhau (`H1..H7`, `339 Lessons`), làm người dùng phải cuộn chuột rất xa mới thấy nội dung Task. | **LOẠI BỎ KHỎI MÀN HÌNH CHÍNH** | Ẩn khỏi màn hình chính (nội dung bài học đã nằm sẵn trong báo cáo Tab `S2`). |
| **8. Khối `Isolation Inspector Card` (`index.html:254-286`)** | `🔒 CÔ LẬP VẬT LÝ 4 LỚP (PER-TASK ISOLATION)` + 4 ô in đường dẫn tĩnh `.orchestrator/tasks/TASK-1/...` | Vô bổ vì đường dẫn file đã hiện ngay trên thanh tiêu đề của từng Tab báo cáo `S1..S5`. | **LOẠI BỎ 100%** | Ẩn hoàn toàn khối `isolation-inspector-card`. |
| **9. Sơ đồ 6 hộp `autopilot-flow` (`index.html:289`)** | 6 cột hộp xanh lá (`Intake -> Plan -> Needs You -> Implement -> QA -> Done`) | Trùng lặp 100% với dãy Tab báo cáo `S1 -> S2 -> S2.5 -> S3 -> S4 -> S5` nằm ngay sát bên dưới. | **LOẠI BỎ 100%** | Ẩn hoàn toàn `#autopilot-flow`. |
| **10. Khung Chính `step-report-inspector` (`index.html:294-365`)** | 6 Tab (`S1`, `S2`, `DG0`, `S3`, `S4`, `S5`) + 2 cột (`Markdown` + `Code Demo & Sub-Tasks`) | Thiếu nơi xem **Minh bạch từng phiên gọi Model (`Model + Prompt + Raw Response`)** và thông tin **Repo Memory Define**. | **GIỮ & NÂNG CẤP TRỌNG TÂM** | Đưa lên **ngay đầu màn hình chính**! Bổ sung Tab số 1 nổi bật: **`🔍 Minh Bạch Model & Repo Memory (Prompt / Output Thực Tế)`** bên cạnh `S1`, `S2`, `S2.5 (DG0)`, `S3 (Giải pháp & Demo)`, `S4 (Diff)`, `S5 (Audit)`. |
| **11. Thẻ Sub-Task & `subtask-fractal-inspector` (`app.js:1111-1558`)** | Mỗi Sub-Task hiện 10 badge/pill + 2 select + 4 nút; khung `subtask-fractal-inspector` bung sẵn 3 cột hợp đồng dài dòng. | Làm rối mắt và choáng hết cột phải. | **THU GỌN** | Thẻ Sub-Task chỉ hiện 1 dòng sạch sẽ: `[x] ID — Tên Sub-Task` · `📦 Repo` · `📄 file:line` · `🤖 Model` · `Stage` + nút `▶ Chạy` / `📐 Chi tiết`. Khung `subtask-fractal-inspector` mặc định ẩn, chỉ bung ra khi bấm `📐 Chi tiết`. |
| **12. Khung Đáy `autopilot-split-bottom` (`index.html:443-469`)** | Cột trái: `autopilot-cycles` (SQLite rounds) + 4 chấm chú thích màu (`done`, `running now`, `waiting on you`, `pending`). Cột phải: `Live Task Execution Log` | Cột trái (`SQLite autopilot_rounds`) chỉ hiện số phút/tiền giả lập (`$0.00`), không có thông tin Prompt/Response thật. | **THAY THẾ CỘT TRÁI** | Biến cột trái thành **Sổ Cái Các Phiên Gọi Model Của Task (`Per-Task Model Call Sessions`)** bấm vào từng phiên xem ngay `Model`, `Prompt`, `Response`; cột phải giữ `Terminal Log` thật của Task. |

---

## 3. Thiết Kế Kỹ Thuật Ánh Xạ 1-1 (`F1` → `F4`) & Bảng Đấu Nối Runtime (`Explicit Call-Site Wiring` — `L014`)

### 3.1. Bảng Đấu Nối Điểm Gọi Thực Thi (`Caller file:line` & `Lifecycle Trigger`)

| Module / Hàm mới | File chứa (`Single-Responsibility`) | Điểm gọi thực tế (`Caller file:line`) | Thời điểm kích hoạt (`Lifecycle Stage / Trigger`) |
| :--- | :--- | :--- | :--- |
| `LoadOrInitRepoMemoryDefinition(repoPath)` | `internal/webui/repo_memory_store.go` (Mới) | 1. `internal/webui/task_inspector.go:2800` (`generateDeepTaskArtifacts`)<br>2. `internal/webui/task_llm_bridge.go:198` (`invokeArchitectLLM`)<br>3. `internal/webui/server.go:1645` (`BuildDashboardResponse`) | Kích hoạt ngay đầu pha **S1/S2** khi xử lý `targetRepo`: kiểm tra `agent/memory/global/project/<repo>/repo_definition.json` (và `.orchestrator/memory/repos/<repo>/repo_definition.json`). Nếu có -> đọc ngay (`LOADED_FROM_MEMORY`). Nếu chưa có -> quét cấu trúc gốc 1 lần, lưu file (`INITIALIZED_NEW`) cho các lượt sau. |
| `appendTaskModelCallTrace(taskKey, entry)` & `loadTaskModelCallLedger(taskKey)` | `internal/webui/task_llm_trace.go` (Mới) | 1. `internal/webui/task_llm_bridge.go:245` (`invokeArchitectLLM` — `BRAIN`)<br>2. `internal/webui/subtask_codegen_muscle.go:255` (`executeRealMuscleCodeGenForSubTask` — `MUSCLE`)<br>3. `internal/webui/subtask_edit_service.go:315` (`invokeQAAuditorLLM` — `QA`)<br>4. `internal/webui/server.go:1648` (`BuildDashboardResponse`) | Kích hoạt **ngay sau mỗi phiên gọi Model (`Complete`)** ở `S1-S3 Plan`, `Co-Architect Chat`, `S4 Muscle CodeGen`, `S5 QA Audit`. Ghi nguyên bản `Model`, `SessionID`, `SystemPrompt`, `UserPrompt`, `RawResponse`, `DurationMs`, `Status` vào `.orchestrator/tasks/<taskKey>/llm_call_ledger.json` (cô lập tuyệt đối theo từng Task). |
| `renderModelTransparencyAndRepoMemory()` | `internal/webui/static/app.js` & `index.html` | `internal/webui/static/app.js:1886` (`renderAll` & `renderStepReportInspector`) | Hiển thị trực tiếp trên FE ở thanh Context Bar (`🧠 Repo Memory Badge`), Tab `🔍 Minh Bạch Model & Repo Memory` và bảng `Sổ Cái Phiên Gọi Model` cho đúng Task đang chọn. |

---

### 3.2. Code Demo 1 (`F1` & `F2`): Ghi Nhận Sổ Cái Phiên Gọi Model (`task_llm_trace.go`) & Khởi Tạo Memory Định Nghĩa Repo (`repo_memory_store.go`) + Đấu Nối Vào `task_llm_bridge.go`

```go
// File mới 1: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/repo_memory_store.go
package webui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RepoMemoryDefinitionDTO struct {
	RepoName       string   `json:"repo_name"`
	RepoPath       string   `json:"repo_path"`
	CurrentBranch  string   `json:"current_branch"`
	Language       string   `json:"language"`
	CodeFenceLang  string   `json:"code_fence_lang"`
	BuildCommand   string   `json:"build_command"`
	TestCommand    string   `json:"test_command"`
	EntryPoints    []string `json:"entry_points"`
	KeyDirectories []string `json:"key_directories"`
	ExistingDocs   []string `json:"existing_docs"`
	MemoryStatus   string   `json:"memory_status"` // "LOADED_FROM_MEMORY" | "INITIALIZED_NEW"
	MemoryFilePath string   `json:"memory_file_path"`
	InitializedAt  string   `json:"initialized_at"`
	LastVerifiedAt string   `json:"last_verified_at"`
}

func (s *Server) LoadOrInitRepoMemoryDefinition(repoPath string) RepoMemoryDefinitionDTO {
	cleanRepo := strings.TrimSpace(repoPath)
	if cleanRepo == "" {
		cleanRepo = s.Root
	}
	repoName := filepath.Base(cleanRepo)
	branch := detectRepoBranchFast(cleanRepo)
	nowStr := time.Now().Format(time.RFC3339)

	// Ưu tiên lưu trong .orchestrator/memory/repos/<repoName>/repo_definition.json (và đọc kết hợp /agent/memory/global/project/<repoName> nếu có)
	localMemDir := filepath.Join(s.Root, ".orchestrator", "memory", "repos", sanitizeTaskKeyPath(repoName))
	_ = os.MkdirAll(localMemDir, 0o755)
	defPath := filepath.Join(localMemDir, "repo_definition.json")

	if raw, err := os.ReadFile(defPath); err == nil && len(raw) > 0 {
		var existing RepoMemoryDefinitionDTO
		if json.Unmarshal(raw, &existing) == nil && existing.RepoName != "" {
			existing.CurrentBranch = branch
			existing.MemoryStatus = "LOADED_FROM_MEMORY"
			existing.MemoryFilePath = defPath
			existing.LastVerifiedAt = nowStr
			return existing
		}
	}

	fenceLang, dodCmd := detectTargetRepoLanguageAndDoD(cleanRepo)
	stack := detectRepoTechStackProfile(cleanRepo, nil)
	var keyDirs, entryPoints, existingDocs []string

	if entries, err := os.ReadDir(cleanRepo); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || isSkippedBuildOrVendorDir(name) {
				continue
			}
			if e.IsDir() {
				keyDirs = append(keyDirs, name)
				for _, subEntry := range []string{"main.go", "index.ts", "server.ts", "app.ts", "main.py"} {
					cand := filepath.Join(name, subEntry)
					if _, stErr := os.Stat(filepath.Join(cleanRepo, cand)); stErr == nil {
						entryPoints = append(entryPoints, cand)
					}
				}
			} else if name == "main.go" || name == "package.json" || name == "go.mod" || name == "Cargo.toml" || name == "pyproject.toml" {
				entryPoints = append(entryPoints, name)
			}
		}
	}

	agentProjDir := filepath.Join("/Users/trainguyen/Documents/work/agent/memory/global/project", repoName)
	if docs, err := os.ReadDir(agentProjDir); err == nil {
		for _, d := range docs {
			if !d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
				existingDocs = append(existingDocs, d.Name())
			}
		}
	}

	def := RepoMemoryDefinitionDTO{
		RepoName:       repoName,
		RepoPath:       cleanRepo,
		CurrentBranch:  branch,
		Language:       stack.LanguageLabel,
		CodeFenceLang:  fenceLang,
		BuildCommand:   stack.BuildAndDoD,
		TestCommand:    dodCmd,
		EntryPoints:    entryPoints,
		KeyDirectories: keyDirs,
		ExistingDocs:   existingDocs,
		MemoryStatus:   "INITIALIZED_NEW",
		MemoryFilePath: defPath,
		InitializedAt:  nowStr,
		LastVerifiedAt: nowStr,
	}
	if encoded, err := json.MarshalIndent(def, "", "  "); err == nil {
		_ = os.WriteFile(defPath, encoded, 0o644)
	}
	return def
}
```

```go
// File mới 2: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_trace.go
package webui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/scout"
)

type TaskModelCallTraceDTO struct {
	CallID       string `json:"call_id"`
	TaskKey      string `json:"task_key"`
	SubTaskID    string `json:"sub_task_id,omitempty"`
	Stage        string `json:"stage"`
	Role         string `json:"role"`
	Runner       string `json:"runner"`
	Model        string `json:"model"`
	SessionID    string `json:"session_id,omitempty"`
	TargetRepo   string `json:"target_repo"`
	TargetBranch string `json:"target_branch"`
	RepoMemory   string `json:"repo_memory_status"`
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
	RawResponse  string `json:"raw_response"`
	Status       string `json:"status"` // "REAL_LLM_OK" | "LLM_ERROR" | "STATIC_FALLBACK"
	DurationMs   int64  `json:"duration_ms"`
	Timestamp    string `json:"timestamp"`
}

func (s *Server) taskModelCallLedgerPath(taskKey string) string {
	safeKey := sanitizeTaskKeyPath(taskKey)
	return filepath.Join(s.Root, ".orchestrator", "tasks", safeKey, "llm_call_ledger.json")
}

func (s *Server) loadTaskModelCallLedger(taskKey string) []TaskModelCallTraceDTO {
	if s == nil || strings.TrimSpace(taskKey) == "" {
		return nil
	}
	raw, err := os.ReadFile(s.taskModelCallLedgerPath(taskKey))
	if err != nil || len(raw) == 0 {
		return nil
	}
	var list []TaskModelCallTraceDTO
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}
	return list
}

func (s *Server) appendTaskModelCallTrace(taskKey string, entry TaskModelCallTraceDTO) {
	if s == nil || strings.TrimSpace(taskKey) == "" {
		return
	}
	list := s.loadTaskModelCallLedger(taskKey)
	entry.CallID = fmt.Sprintf("CALL-%02d", len(list)+1)
	entry.TaskKey = taskKey
	entry.SystemPrompt = scout.MaskSecrets(entry.SystemPrompt)
	entry.UserPrompt = scout.MaskSecrets(entry.UserPrompt)
	entry.RawResponse = scout.MaskSecrets(entry.RawResponse)
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format("2006-01-02T15:04:05-07:00")
	}
	list = append(list, entry)
	p := s.taskModelCallLedgerPath(taskKey)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if raw, err := json.MarshalIndent(list, "", "  "); err == nil {
		_ = os.WriteFile(p, raw, 0o644)
	}
}
```

```go
// File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_bridge.go:241-298
// --- TRƯỚC (HEAD hiện tại) ---
		agRunner := runner.NewAntigravityRunner(modelName)
		agRunner.WorkDir = os.TempDir()
		agRunner.SessionFile = s.taskAgentAPISessionFile(qt.ExternalID)
		agRunner.SessionTitle = fmt.Sprintf("[%s] %s", qt.ExternalID, qt.Title)
		stream, err := agRunner.Complete(llmCtx, runner.CompletionRequest{
			Model:        modelName,
			SystemPrompt: sysPrompt,
			Messages:     []runner.ConvMessage{{Role: "user", Content: userMsg}},
			SessionFile:  agRunner.SessionFile,
			SessionTitle: agRunner.SessionTitle,
		})
		if err == nil {
			var raw strings.Builder
			var streamErr error
			for tok := range stream {
				if tok.Error != nil {
					streamErr = tok.Error
					break
				}
				raw.WriteString(tok.Text)
			}
			rawStr := strings.TrimSpace(raw.String())
			if streamErr != nil {
// ...
			if rawStr != "" {
				if parsed := parseArchitectLLMResponse(rawStr, hits); parsed != nil && (strings.TrimSpace(parsed.DirectAnswer) != "" || len(parsed.Solutions) > 0) {
					if strings.TrimSpace(parsed.DirectAnswer) == "" {
						parsed.DirectAnswer = synthesizeProgressAwareDirectAnswer(qt, userPrompt, subTasks, parsed.RootCauses, parsed.Solutions)
					}
					if len(parsed.Architecture) == 0 {
						parsed.Architecture = parsed.RootCauses
					}
					s.saveTaskLLMAnalysis(qt.ExternalID, parsed)
					return parsed, nil
				}
			}
		} else {
			fmt.Printf("[CoArchitect-LLM] completeErr=%v\n", err)
			return &ArchitectLLMAnalysis{
				DirectAnswer: fmt.Sprintf("⚠️ **Không thể khởi tạo Model Runner (%s):** %v", modelName, err),
			}, nil
		}
	}

	fallback := synthesizeSymbolGroundedArchitectAnalysis(qt, subTasks, hits, userPrompt, fenceLang)
	s.saveTaskLLMAnalysis(qt.ExternalID, fallback)
	return fallback, nil
}

// --- SAU KHI SỬA ---
		callStart := time.Now()
		repoMem := s.LoadOrInitRepoMemoryDefinition(targetRepo)
		agRunner := runner.NewAntigravityRunner(modelName)
		agRunner.WorkDir = os.TempDir()
		agRunner.SessionFile = s.taskAgentAPISessionFile(qt.ExternalID)
		agRunner.SessionTitle = fmt.Sprintf("[%s] %s", qt.ExternalID, qt.Title)
		stream, err := agRunner.Complete(llmCtx, runner.CompletionRequest{
			Model:        modelName,
			SystemPrompt: sysPrompt,
			Messages:     []runner.ConvMessage{{Role: "user", Content: userMsg}},
			SessionFile:  agRunner.SessionFile,
			SessionTitle: agRunner.SessionTitle,
		})
		if err == nil {
			var raw strings.Builder
			var streamErr error
			for tok := range stream {
				if tok.Error != nil {
					streamErr = tok.Error
					break
				}
				raw.WriteString(tok.Text)
			}
			rawStr := strings.TrimSpace(raw.String())
			durMs := time.Since(callStart).Milliseconds()
			if streamErr != nil {
				s.appendTaskModelCallTrace(qt.ExternalID, TaskModelCallTraceDTO{
					Stage:        "S1_S3_" + strings.ToUpper(actionMode),
					Role:         "BRAIN",
					Runner:       s.activeRunner,
					Model:        modelName,
					SessionID:    s.readTaskAgentAPIConvID(qt.ExternalID),
					TargetRepo:   repoMem.RepoName,
					TargetBranch: repoMem.CurrentBranch,
					RepoMemory:   repoMem.MemoryStatus,
					SystemPrompt: sysPrompt,
					UserPrompt:   userMsg,
					RawResponse:  rawStr + "\n[ERROR]: " + streamErr.Error(),
					Status:       "LLM_ERROR",
					DurationMs:   durMs,
				})
				cleanErr := streamErr.Error()
// ...
			if rawStr != "" {
				s.appendTaskModelCallTrace(qt.ExternalID, TaskModelCallTraceDTO{
					Stage:        "S1_S3_" + strings.ToUpper(actionMode),
					Role:         "BRAIN",
					Runner:       s.activeRunner,
					Model:        modelName,
					SessionID:    s.readTaskAgentAPIConvID(qt.ExternalID),
					TargetRepo:   repoMem.RepoName,
					TargetBranch: repoMem.CurrentBranch,
					RepoMemory:   repoMem.MemoryStatus,
					SystemPrompt: sysPrompt,
					UserPrompt:   userMsg,
					RawResponse:  rawStr,
					Status:       "REAL_LLM_OK",
					DurationMs:   durMs,
				})
				if parsed := parseArchitectLLMResponse(rawStr, hits); parsed != nil && (strings.TrimSpace(parsed.DirectAnswer) != "" || len(parsed.Solutions) > 0) {
					if strings.TrimSpace(parsed.DirectAnswer) == "" {
						parsed.DirectAnswer = synthesizeProgressAwareDirectAnswer(qt, userPrompt, subTasks, parsed.RootCauses, parsed.Solutions)
					}
					if len(parsed.Architecture) == 0 {
						parsed.Architecture = parsed.RootCauses
					}
					s.saveTaskLLMAnalysis(qt.ExternalID, parsed)
					return parsed, nil
				}
			}
```

---

### 3.3. Code Demo 2 (`F3` & `F4`): Tinh Gọn Giao Diện & Hiển Thị Sổ Cái Minh Bạch Phiên Gọi Model + Repo Memory (`internal/webui/static/index.html` & `internal/webui/static/app.js`)

```html
<!-- File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:294-338 -->
<!-- --- TRƯỚC (HEAD hiện tại) --- -->
            <div class="step-report-inspector" id="step-report-inspector-card">
              <div class="sri-header">
                <div class="sri-title-wrap">
                  <span class="sri-badge">📑 BÁO CÁO WORKFLOW, GIẢI PHÁP &amp; CODE DEMO</span>
                  <span id="sri-active-file" class="sri-file-pill">.orchestrator/tasks/TASK-1/09_tasks_solution.md</span>
                </div>
                <div class="sri-actions-wrap" id="sri-gate-actions">
                  <button type="button" class="btn-mini-cli" onclick="scrollToCoArchitectStudio()">💬 Thảo luận &amp; Sửa Giải Pháp cùng Agent</button>
                  <button id="sri-approve-btn" class="btn-approve-prominent hidden" onclick="runTaskAction('', 'approve_and_run')">
                    ✓ APPROVE Giải Pháp S3 &amp; Chạy S4–S5 Ngay
                  </button>
                </div>
              </div>

              <!-- 6 Step Report Tabs (including S2.5 Pre-Plan Critique DG0 :8200) -->
              <div class="sri-tabs" id="sri-tabs-bar">
                <button type="button" class="sri-tab" data-step="S1" onclick="selectReportTab('S1')">
                  📋 S1 Requirements &amp; Sub-Tasks
                </button>
                <button type="button" class="sri-tab" data-step="S2" onclick="selectReportTab('S2')">
                  🔍 S2 Repo Investigation (file:line)
                </button>
                <button type="button" class="sri-tab" data-step="DG0" onclick="selectReportTab('DG0')">
                  ⚡ S2.5 Phản Biện Trước Plan (DG0 :8200)
                </button>
                <button type="button" class="sri-tab active" data-step="S3" onclick="selectReportTab('S3')">
                  💡 S3 Giải Pháp &amp; Code Demo
                </button>
                <button type="button" class="sri-tab" data-step="S4" onclick="selectReportTab('S4')">
                  🔧 S4 Worktree &amp; Git Diff
                </button>
                <button type="button" class="sri-tab" data-step="S5" onclick="selectReportTab('S5')">
                  🛡️ S5 Audit Report &amp; DG3
                </button>
              </div>

<!-- --- SAU KHI SỬA (Thêm Tab Minh Bạch Model & Repo Memory + Tinh Gọn Header) --- -->
            <div class="step-report-inspector" id="step-report-inspector-card">
              <div class="sri-header">
                <div class="sri-title-wrap">
                  <span class="sri-badge">📑 HỒ SƠ TASK &amp; MINH BẠCH THỰC THI</span>
                  <span id="sri-repo-memory-pill" class="sri-file-pill">🧠 Repo Memory: Đang kiểm tra...</span>
                  <span id="sri-active-file" class="sri-file-pill">.orchestrator/tasks/TASK-1/09_tasks_solution.md</span>
                </div>
                <div class="sri-actions-wrap" id="sri-gate-actions">
                  <button id="sri-approve-btn" class="btn-approve-prominent hidden" onclick="runTaskAction('', 'approve_and_run')">
                    ✓ APPROVE &amp; Chạy S4–S5
                  </button>
                </div>
              </div>

              <!-- 7 Step Report Tabs (Bổ sung Tab TRACE: Minh bạch Model, Prompt & Raw Output từng phiên) -->
              <div class="sri-tabs" id="sri-tabs-bar">
                <button type="button" class="sri-tab" data-step="TRACE" onclick="selectReportTab('TRACE')">
                  🔎 Minh Bạch Model &amp; Prompt (<b id="sri-trace-count">0</b> phiên)
                </button>
                <button type="button" class="sri-tab" data-step="S1" onclick="selectReportTab('S1')">
                  📋 S1 Yêu Cầu
                </button>
                <button type="button" class="sri-tab" data-step="S2" onclick="selectReportTab('S2')">
                  🔍 S2 Điều Tra Repo
                </button>
                <button type="button" class="sri-tab" data-step="DG0" onclick="selectReportTab('DG0')">
                  ⚡ S2.5 Phản Biện (DG0)
                </button>
                <button type="button" class="sri-tab active" data-step="S3" onclick="selectReportTab('S3')">
                  💡 S3 Giải Pháp &amp; Demo
                </button>
                <button type="button" class="sri-tab" data-step="S4" onclick="selectReportTab('S4')">
                  🔧 S4 Git Diff
                </button>
                <button type="button" class="sri-tab" data-step="S5" onclick="selectReportTab('S5')">
                  🛡️ S5 Kiểm Định
                </button>
              </div>
```

---

## 4. Vòng Lặp Phản Tỉnh & Tự Khắc Phục `S5 → S3 → S4 → S5` (`AF-1` → `AF-5`)

Qua kiểm định phản biện thực tế sau lần chạy đầu, phát hiện 5 lỗ hổng kỹ thuật cần đóng kín ngay trong lượt:
1. **`AF-1` (`internal/webui/task_llm_trace.go`, `internal/webui/task_llm_bridge.go`, `internal/runner/antigravity_agentapi_hidden.go`):**
   - Bổ sung hàm `isLLMErrorText(raw string) bool` trong `task_llm_trace.go` để bắt chính xác các chuỗi lỗi xác thực / hạn mức trên stdout (`"Failed to authenticate"`, `"API Error: 401"`, `"OAuth access token is invalid"`, `"TerminalQuotaError"`, `"Quota exceeded"`).
   - Chặn đứng việc gắn nhãn láo `"status": "REAL_LLM_OK"` hoặc lưu chuỗi lỗi 401 vào `DirectAnswer` của `09_tasks_solution.md`; bắt buộc gắn `"status": "LLM_ERROR"` và điều hướng `s.activeRunner == "antigravity"` qua `agRunner.Binary = "agentapi"`, đồng thời ghi `convID` thực tế vào `sessionFile + ".last_conv_id"`.
2. **`AF-2` (`internal/webui/repo_memory_store.go`, `internal/webui/task_inspector.go` — Định tuyến kho mã nguồn theo nhánh `feat/auto-recon-export` & Cắt đứt quét tràn `ResolveTaskAffectedRepos`):**
   - Trong `ResolveTaskAffectedRepos`: khi `qt.Labels` đã có `repo:` / `repos:` tường minh, trả về ngay danh sách đó và **cấm** quét substring `strings.Contains(lower, baseName)` lôi thêm các repo khác (như `package` hoặc `cdc-*`).
   - Bổ sung `resolveWorkspaceRepoByBranchHint(lowerText)` trong `repo_memory_store.go`: khi đề bài chỉ định tên nhánh (ví dụ `branch feat/auto-recon-export`), ưu tiên khớp repo có nhánh đó trong Git (`centralized-export-service`).
3. **`AF-3` (Single-Responsibility Module Split & Line-Budget Guard `NOW <= HEAD` — `L018`):**
   - Tách toàn bộ hàm JS/CSS mới sang `internal/webui/static/task_transparency.js` và `internal/webui/static/task_transparency.css`; tách test mới sang `internal/webui/repo_memory_trace_test.go`.
   - Rút gọn toàn bộ các file cũ để đạt trần `NOW <= HEAD` so với mốc trước S4: `task_inspector.go <= 4024`, `server.go <= 3231`, `task_llm_bridge.go <= 487`, `subtask_codegen_muscle.go <= 532`, `subtask_edit_service.go <= 352`, `index.html <= 1027`, `app.js <= 3456`, `app.css <= 3741`.
4. **`AF-4` (`internal/webui/repo_memory_trace_test.go` — Chống suy diễn Smoke Test):**
   - Kiểm thử thông qua HTTP handler thật (`httptest.NewRequest` tới `/api/dashboard` và `invokeArchitectLLM` với lỗi 401 & định tuyến nhánh `feat/auto-recon-export`).
5. **`AF-5` (Biên dịch lại `./bin/orchestrator` có `//go:embed static/*` & Khởi động lại tiến trình sống `:8090`):**
   - Tăng cache-busting `?v=20261105b` trong `index.html`, chạy `go build -o ./bin/orchestrator ./cmd/orchestrator`, khởi động lại server `:8090` và kiểm chứng trực tiếp `/api/dashboard` trên cổng `:8090`.

---

## 5. Thiết Kế Bổ Sung (`Task 6` — Khôi Phục & Tinh Gọn Trực Quan Sơ Đồ Workflow `S1→S6` + Dải Wave `0..N` Sub-Tasks Của Task Trên Màn Hình Chính)

### 5.1. Nguyên nhân gốc rễ (Root Cause — `F6`)
- Khi thực hiện cắt giảm nhãn chữ lặp lại ở lượt trước, việc gắn class `hidden` vào `<div id="autopilot-flow" class="autopilot-flow hidden"></div>` tại `internal/webui/static/index.html:290` đã vô tình **ẩn mất toàn bộ thanh sơ đồ quy trình thực thi của Task** trên màn hình chính (`#screen-autopilot`).
- Hơn nữa, trước đây `#autopilot-flow` (`internal/webui/static/app.js:1958-2004`) vừa hiển thị 4 dòng chữ lặp lại rối mắt trong mỗi hộp bước (`title`, `subtitle`, `role_label + agent_slot + model`, `meta`), vừa **thiếu hẳn dải Wave `0..N` Sub-Tasks** của Task đó ngay trên màn hình chính (bắt người dùng phải bấm chuyển sang màn hình `2. Sơ Đồ Wave DAG` mới thấy Task có mấy Wave và Sub-Task nào đang chạy).

### 5.2. Giải pháp duy nhất tối ưu (`Single-Best Solution`)
1. **Khôi phục hiển thị `#autopilot-flow` (`internal/webui/static/index.html:290`):**
   - Bỏ class `hidden` trên `<div id="autopilot-flow" class="autopilot-flow"></div>` để hiển thị ngay phía trên khung `HỒ SƠ TASK & MINH BẠCH MODEL` (`#step-report-inspector-card`).
2. **Thiết kế Khung Kép `Workflow S1→S6 + Wave 0..N` Gọn Sạch (`renderTaskWorkflowAndWaveStrip()` trong `internal/webui/static/task_transparency.js` & gọi từ `renderAutopilot()` trong `internal/webui/static/app.js:1958-2005`):**
   - **Tầng 1 — Luồng Bước Quy Trình Của Task (`S1 Intake → S2.5 DG0 → S3 Plan & Demo → Human Gate → S4 Implement → S5 QA & DG3`):**
     - Mỗi bước hiển thị gọn trên **1 pill/thẻ ngang trực quan**: `[Icon Trạng Thái] Tên Bước Ngắn Gọn` + `1 nhãn trạng thái/round` (toàn bộ thông tin chi tiết `Role · Slot · Model · Meta` được đưa vào `title` tooltip khi di chuột và hiển thị ở Inspector khi click `selectAutopilotStageBox(idx)`).
     - Khi Task ở trạng thái `NEEDS_YOU`, bước `Human Gate` hiển thị trực tiếp nút `✓ APPROVE & Run S4`.
   - **Tầng 2 — Dải Wave Thực Thi Sub-Tasks Của Task (`Wave 0 → Wave 1 → Wave 2...`):**
     - Gom nhóm trực tiếp `dashboardState.active_sub_tasks` (hoặc `activeNode.sub_tasks`) của Task đang chọn theo `wave_index` / `wave` (`Wave 0`, `Wave 1`, `Wave 2`...).
     - Hiển thị các cột Wave nối tiếp nhau (`Wave 0 (done/total) → Wave 1 (done/total) → ...`), bên trong mỗi Wave là các chip Sub-Task (`[●/✓/⟳] ST-1: Tên ngắn · 📦 repo`), bấm vào chip Sub-Task sẽ mở/nhảy trực tiếp tới chi tiết Sub-Task đó (`focusSubTaskFromWorkflowStrip(subTaskId)`), kèm nút `🔭 Mở Sơ Đồ DAG` chuyển nhanh sang màn hình Wave đầy đủ nếu muốn xem đường cong Bezier.
   - **Bảo toàn trần số dòng (`NOW <= HEAD` — `L018`):**
     - Đặt hàm dựng DOM `renderTaskWorkflowAndWaveStrip()` trong module đơn trách nhiệm `internal/webui/static/task_transparency.js` và CSS trong `internal/webui/static/task_transparency.css`, đồng thời thay đoạn vòng lặp cũ dài 46 dòng trong `internal/webui/static/app.js:1958-2004` bằng lời gọi `renderTaskWorkflowAndWaveStrip()`, giúp `app.js` giảm thêm ~38 dòng (`3367 <= 3409`).

### 5.3. Code Demo 3 (`Task 6` — `index.html`, `app.js`, `task_transparency.js`, `task_transparency.css`)

```html
<!-- File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:289-292 -->
<!-- --- TRƯỚC (Hiện tại đang bị gắn class hidden) --- -->
            <!-- 6-Stage Graphical Pipeline Diagram (Scoped 100% to Selected Task!) -->
            <div id="autopilot-flow" class="autopilot-flow hidden"></div>

<!-- --- SAU KHI SỬA (Hiển thị trực quan Sơ đồ Workflow S1->S6 & Dải Wave 0..N của Task) --- -->
            <!-- 6-Stage Workflow Stepper + Per-Task Wave 0..N Sub-Task Strip (Scoped 100% to Selected Task!) -->
            <div id="autopilot-flow" class="autopilot-flow"></div>
```

```javascript
// File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:1958-2006
// --- TRƯỚC (Vòng lặp 46 dòng render 4 dòng text lặp lại trong mỗi thẻ bước) ---
  const flowEl = document.getElementById("autopilot-flow");
  if (flowEl) {
    const groups = dashboardState.autopilot_stages || [];
    // ... (40 dòng render c.title, c.subtitle, c.role_label, c.agent_slot, c.model, c.meta)
    flowEl.innerHTML = htmlParts.join("");
  }
  if (typeof renderPerTaskModelCallLedger === "function") renderPerTaskModelCallLedger();

// --- SAU KHI SỬA (Ủy quyền sang module đơn trách nhiệm task_transparency.js) ---
  if (typeof renderTaskWorkflowAndWaveStrip === "function") {
    renderTaskWorkflowAndWaveStrip();
  }
  if (typeof renderPerTaskModelCallLedger === "function") renderPerTaskModelCallLedger();
```

```javascript
// File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/task_transparency.js
function renderTaskWorkflowAndWaveStrip() {
  if (!dashboardState) return;
  const flowEl = document.getElementById("autopilot-flow");
  if (!flowEl) return;

  const activeKey = dashboardState.active_task_key || selectedChainKey || "TASK-1";
  const activeStage = dashboardState.active_task_stage || "READY";
  const groups = dashboardState.autopilot_stages || [];

  // 1. Tầng 1: 6-Stage Workflow Stepper gọn sạch (Bấm vào từng bước để mở Báo cáo/Code Demo bước đó)
  const stageStepsHtml = groups
    .map((grp, idx) => {
      const card = (grp.cards || [])[0] || {};
      const st = String(card.status || "pending").toLowerCase();
      const icon = st === "running" ? `<span class="spin-inline">⟳</span>` : st === "done" ? "✓" : st === "waiting" ? "●" : "◌";
      const isApproveGate = String(card.title || "").startsWith("You") && activeStage === "NEEDS_YOU";
      const tooltip = [card.title, card.subtitle, card.role_label ? `${card.role_label} (${card.agent_slot || ""} · ${card.model || ""})` : "", card.meta]
        .filter(Boolean)
        .join(" · ");
      const tabCode = mapAutopilotBoxIndexToReportTab(idx);
      const isFocusedTab = selectedReportTab === tabCode;

      return `
        <div class="twf-stage-pill status-${escapeAttr(st)} ${isFocusedTab ? "active-stage-pill" : ""}"
             onclick="selectAutopilotStageBox(${idx})"
             title="${escapeAttr(tooltip || "Click để mở báo cáo bước này")}">
          <div class="twf-stage-top">
            <span class="twf-stage-icon">${icon}</span>
            <span class="twf-stage-name">${escapeHtml(card.title || `Stage ${idx + 1}`)}</span>
            ${grp.round_badge ? `<span class="twf-stage-badge">${escapeHtml(grp.round_badge)}</span>` : ""}
          </div>
          ${card.subtitle ? `<div class="twf-stage-sub">${escapeHtml(card.subtitle)}</div>` : ""}
          ${
            isApproveGate
              ? `<button type="button" class="mini-run-btn" style="margin-top:4px;width:100%;" onclick="runTaskAction('', 'approve_and_run', event)">✓ APPROVE &amp; Run S4</button>`
              : ""
          }
        </div>
      `;
    })
    .join(`<span class="twf-arrow">→</span>`);

  // 2. Tầng 2: Dải Wave 0..N Sub-Tasks của riêng Task đang chọn
  const activeNode = (typeof findWaveNodeByKey === "function" ? findWaveNodeByKey(activeKey) : null) || {};
  const subTasks =
    Array.isArray(dashboardState.active_sub_tasks) && dashboardState.active_sub_tasks.length > 0
      ? dashboardState.active_sub_tasks
      : activeNode.sub_tasks || [];

  const waveMap = new Map();
  for (const st of subTasks) {
    const w = Number(st.wave_index ?? st.wave ?? 0);
    if (!waveMap.has(w)) waveMap.set(w, []);
    waveMap.get(w).push(st);
  }
  const sortedWaves = Array.from(waveMap.entries()).sort((a, b) => a[0] - b[0]);

  const waveStripHtml =
    sortedWaves.length > 0
      ? sortedWaves
          .map(([wNum, list]) => {
            const doneCnt = list.filter((x) => {
              const s = String(x.stage || x.status || "").toUpperCase();
              return s === "DONE" || s === "PASSED";
            }).length;
            const chipsHtml = list
              .map((st) => {
                const stId = st.sub_task_id || st.id || "";
                const stStage = String(st.stage || st.status || "PENDING").toUpperCase();
                const tone = stStage === "DONE" || stStage === "PASSED" ? "done" : stStage === "RUNNING" ? "running" : stStage === "NEEDS_YOU" ? "waiting" : "pending";
                const repoShort = st.target_repo_name || formatRepoBasename(st.target_repo || dashboardState.active_target_repo);
                return `
                  <button type="button" class="twf-subtask-chip status-${tone}"
                          onclick="focusSubTaskFromWorkflowStrip('${escapeAttr(stId)}')"
                          title="${escapeAttr(`${stId}: ${st.title || ""} · Repo: ${repoShort} · Status: ${stStage}`)}">
                    <span class="twf-st-id">${escapeHtml(stId)}</span>
                    <span class="twf-st-title">${escapeHtml(st.title || stId)}</span>
                    <span class="twf-st-repo">📦 ${escapeHtml(repoShort)}</span>
                  </button>
                `;
              })
              .join("");

            return `
              <div class="twf-wave-box">
                <div class="twf-wave-hdr">
                  <span>🌊 Wave ${wNum}</span>
                  <b>${doneCnt}/${list.length}</b>
                </div>
                <div class="twf-wave-chips">${chipsHtml}</div>
              </div>
            `;
          })
          .join(`<span class="twf-arrow">→</span>`)
      : `<span class="mono-dim">Chưa có Sub-Task phân rã Wave cho ${escapeHtml(activeKey)} (Bấm 1. Plan S1–S3 để phân rã).</span>`;

  flowEl.innerHTML = `
    <div class="twf-container">
      <div class="twf-row">
        <div class="twf-row-label">🔄 WORKFLOW S1→S6</div>
        <div class="twf-stages-track">${stageStepsHtml}</div>
      </div>
      <div class="twf-row twf-wave-row">
        <div class="twf-row-label">
          <span>🌊 WAVE SUB-TASKS (${subTasks.length})</span>
          <button type="button" class="twf-open-dag-btn" onclick="switchMainView('waves')" title="Mở Sơ Đồ Wave DAG toàn màn hình">🔭 Sơ đồ DAG</button>
        </div>
        <div class="twf-waves-track">${waveStripHtml}</div>
      </div>
    </div>
  `;
}

function focusSubTaskFromWorkflowStrip(subTaskId) {
  if (!subTaskId) return;
  selectedSubTaskID = subTaskId;
  window._userExpandedSubTaskFractal = true;
  selectReportTab("SUBTASKS");
}
```

### 5.4. Quy Hoạch Đồng Bộ Kiến Trúc Thông Tin 3 Phân Vùng (`3-Zone Holistic IDE Architecture` — Chấm Dứt Tình Trạng Chắp Vá 7 Khối Chồng Chéo Trên `#screen-autopilot`)

1. **Nguyên nhân vì sao giao diện cũ nhìn vẫn rối và chắp vá (`F7`):**
   - Trước đây chỉ gắn `.hidden` cục bộ lên 3 thẻ đầu trang nhưng trên cùng một màn hình `#screen-autopilot` vẫn bày ra **7 khối chồng lên nhau cùng lúc**:
     1. Thanh `.sri-header` lặp lại nguyên văn `🧠 Repo Memory` và nút `✓ APPROVE` vốn đã nằm ngay ở `#cross-screen-context-bar` cách đó 40px.
     2. Cột phải `#sri-side-pane` nhồi cả khung `Code Demo` lẫn toàn bộ `Sub-Tasks Manager` (`.sri-subtasks-box` + 5 ô input inline + `#subtask-fractal-inspector` 3 cột) vào trong 1 cột hẹp 50% chiều ngang.
     3. Khung `#co-architect-studio` (với 4 nút gợi ý dài, 3 dropdown, lịch sử chat, ô nhập và 3 nút gửi/duyệt) luôn mở cố định chiếm nửa dưới màn hình.
     4. Khung `.autopilot-split-bottom` ở đáy trang lặp lại bảng `#autopilot-cycles` (trùng với Tab `🔎 Minh Bạch Model`) và `#active-task-console`.
2. **Kiến trúc 3 Phân Vùng Đồng Bộ (`3-Zone Clean IDE Layout` trong `task_transparency.js` & `task_transparency.css`):**
   - **Phân vùng 1 — Thanh Ngữ Cảnh & Hành Động Duy Nhất (`#cross-screen-context-bar`):** Hiển thị `🎯 Task`, `Stage`, `🧠 Memory`, `📦 Repo · ⎇ Branch` và đúng 1 bộ nút điều khiển (`1. Plan S1–S3`, `2. ✓ APPROVE & S4–S5`, `⚡ Autopilot`, `✏️ Sửa`, `🗑 Xóa`). Ẩn `.sri-header` trùng lặp.
   - **Phân vùng 2 — Khung Kép `Workflow S1→S6 + Dải Wave 0..N Sub-Tasks` (`#autopilot-flow`):** Hiển thị trực quan tiến độ 6 bước quy trình (bấm chọn bước để đổi tab báo cáo) và các đợt `Wave 0 → Wave 1 → ...` của Task đang chọn (bấm chọn chip Sub-Task để mở ngay Tab `🧩 Sub-Tasks` kèm Fractal Inspector của Sub-Task đó).
   - **Phân vùng 3 — Không Gian Làm Việc Phân Tab Độc Lập (`#step-report-inspector-card` — Mỗi thời điểm chỉ hiển thị đúng 1 chế độ nhìn sạch sẽ, full-height):**
     - **Chế độ 1 (`S1`, `S2`, `DG0`, `S3`, `S4`, `S5`):** Chia đúng 2 cột 50/50 full chiều cao: Cột trái = Báo cáo Markdown (`#sri-markdown-content`), Cột phải = Code Demo / Git Diff (`#sri-code-demo-content`). Không bị `.sri-subtasks-box`, `#co-architect-studio` hay `.autopilot-split-bottom` chèn ép bên dưới.
     - **Chế độ 2 (`TRACE` — `🔎 Minh Bạch Model`):** Cột trái hiển thị **Danh sách Phiên Gọi Model có thể click chọn ngay trên đầu** + toàn văn `System Prompt` & `User Prompt` của phiên đang chọn; Cột phải hiển thị nguyên bản `Raw Model Response` (`#sri-code-demo-content`).
     - **Chế độ 3 (`SUBTASKS` — `🧩 Sub-Tasks & Wave`):** Hiển thị toàn chiều ngang `.sri-subtasks-box` (gồm thanh lọc Repo, danh sách thẻ Sub-Task đã cắt bỏ nhãn model/slot lặp lại, và `#subtask-fractal-inspector` 3 cột rộng rãi khi chọn Sub-Task).
     - **Chế độ 4 (`STUDIO` — `💬 Đối Thoại & Log`):** Hiển thị song song 2 cột: Cột trái = `#co-architect-studio` (đã ẩn dải 4 nút văn mẫu dài dòng và nút APPROVE lặp lần 3), Cột phải = `.autopilot-console-col` (`📟 Live Task Execution Log & Worktree Shell`).


---

## 6. Thiết Kế Bổ Sung (`Task 7` — Bản Mẫu Trực Quan UI/UX, Hệ Màu Ngữ Nghĩa 5 Trạng Thái & Hệ Thống Progress Bar 3 Tầng Trước Khi Sửa Code)

### 6.1. Nguyên nhân gốc rễ vì sao giao diện mất màu sắc & mất Progress Bar (`F8`)
1. **Ẩn mất Thanh Tiến Độ Tổng Thể (`#live-telemetry-hud` tại `internal/webui/static/app.js:1675` & `index.html:188`):**
   - Dòng `hudWrap.classList.toggle("hidden", !isRunning)` trong `app.js:1675` đã làm ẩn toàn bộ thanh tiến độ (`0% → 100%`) mỗi khi Task ở trạng thái `READY`, `NEEDS_YOU` hoặc `DONE`, khiến người dùng không biết Task đang đạt bao nhiêu `%`, đã xong mấy bước (`X/17 bước`) hay mấy Sub-Task (`X/Y`).
2. **Đánh mất Hệ Màu Ngữ Nghĩa Gốc Của Dự Án (`internal/webui/static/app.css:777-810` vs `task_transparency.css:69-94`):**
   - Trong `app.css:777-810`, các thẻ bước `.ap-subcard` vốn có nền màu trạng thái rất rõ ràng (`status-done` xanh ngọc `rgba(16, 52, 37, 0.55)`, `status-waiting` vàng cam `rgba(120, 53, 15, 0.35)`, `status-running` xanh lam phát sáng `rgba(30, 58, 138, 0.45)`, `status-pending` nét đứt `rgba(15, 23, 42, 0.5)`).
   - Khi chuyển sang `.twf-stage-pill` trong `task_transparency.css:69-94`, chúng ta lại đặt chung một nền xám tối `background: rgba(30, 41, 59, 0.78)` cho mọi trạng thái, làm mất sạch độ tương phản màu sắc giữa bước Đã xong / Đang chạy / Chờ duyệt / Chưa chạy.
3. **Thiếu Thanh Mini Progress Bar cho từng Wave & Sai hàm chuyển màn hình (`task_transparency.js:315`):**
   - Mỗi hộp `Wave 0`, `Wave 1` trong dải `#autopilot-flow` chưa có thanh progress bar (`0%..100%`) và các chip Sub-Task chưa được tô nền màu theo trạng thái. Nút `🔭 Sơ đồ DAG` gọi nhầm `switchMainView('waves')` thay vì `switchScreen('wave')` (`index.html:25`).

### 6.2. Đặc Tả Bảng Màu Ngữ Nghĩa 5 Trạng Thái (`5-State Semantic Color System`)

| Trạng thái | Mã màu Nền (`background`) | Mã màu Viền (`border`) | Mã màu Chữ / Icon | Ý nghĩa & Hiệu ứng |
| :--- | :--- | :--- | :--- | :--- |
| 🟢 **Hoàn tất (`status-done`)** | `rgba(16, 52, 37, 0.62)` (Xanh Ngọc Đậm) | `1px solid rgba(34, 197, 94, 0.65)` (`#22c55e`) | `#4ade80` / `#86efac` (`✓`) | Bước quy trình / Wave / Sub-Task / Phiên gọi Model hoàn thành. |
| 🔵 **Đang chạy (`status-running`)** | `rgba(30, 58, 138, 0.55)` (Xanh Lam Phát Sáng) | `1.5px solid #38bdf8` + `box-shadow: 0 0 12px rgba(56, 189, 248, 0.35)` | `#38bdf8` / `#bae6fd` (`⟳` xoay) | Bước hoặc Sub-Task đang thực thi trực tiếp + Thanh progress shimmer chuyển động. |
| 🟡 **Chờ duyệt (`status-waiting` / `NEEDS_YOU`)** | `rgba(120, 53, 15, 0.48)` (Vàng Cam Hổ Phách) | `1.5px solid rgba(245, 158, 11, 0.78)` (`#f59e0b`) | `#fbbf24` / `#fde68a` (`●`) | Dừng tại cổng duyệt `S3 Plan & Code Demo` chờ User bấm nút vàng cam `✓ APPROVE & Run S4`. |
| 🔴 **Lỗi / Chặn (`status-error` / `BLOCKED`)** | `rgba(127, 29, 29, 0.48)` (Đỏ Hồng Sẫm) | `1.5px solid rgba(244, 63, 94, 0.72)` (`#f43f5e`) | `#f87171` / `#fda4af` (`✕`) | Phiên gọi Model lỗi (`LLM_ERROR 401`) hoặc cổng kiểm định bị chặn. |
| ⚪ **Chờ tới lượt (`status-pending`)** | `rgba(15, 23, 42, 0.55)` (Xám Đen Slate) | `1px dashed rgba(148, 163, 184, 0.35)` | `#64748b` / `#94a3b8` (`◌`) | Bước hoặc Sub-Task thuộc Wave sau chưa tới lượt thực thi (`opacity: 0.78`). |

### 6.3. Đặc Tả Hệ Thống Tiến Độ 3 Tầng (`3-Level Visual Progress Architecture`)

1. **Tầng Tiến Độ 1 — Thanh Tiến Độ Tổng Thể Của Task (`Overall Task Progress Strip` — Luôn hiển thị ở dòng đầu `#autopilot-flow`):**
   - Hiển thị cố định cho mọi trạng thái (`READY 12%`, `RUNNING`, `NEEDS_YOU 55%`, `DONE 100%`):
     - Bên trái: Huy hiệu trạng thái có màu (`⏸ DỪNG CHỜ DUYỆT S3 · 55%` / `⟳ ĐANG CHẠY · 68%` / `✓ HOÀN TẤT · 100%`) + Bước hiện tại (`Bước 9/17 · S3 Single-Best Solution`).
     - Ở giữa: **Thanh Progress Bar rộng (`height: 8px`)** với dải màu chuyển sắc `linear-gradient(90deg, #0ea5e9 0%, #22d3ee 50%, #22c55e 100%)` (kèm hiệu ứng shimmer khi đang chạy và đổi sang màu vàng cam `#f59e0b` khi dừng chờ `NEEDS_YOU`).
     - Bên phải: 3 chỉ số đếm trực quan: `🧩 Sub-Tasks: X/Y` · `🔄 Workflow: X/17 bước` · `🔎 Model Calls: N phiên`.
2. **Tầng Tiến Độ 2 — Dải 6 Thẻ Bước Quy Trình (`S1 Intake → S2.5 DG0 → S3 Plan & Demo → Human Gate → S4 Implement → S5 QA & DG3`):**
   - Mỗi thẻ bước tô đúng **màu nền ngữ nghĩa** trong Bảng 6.2 (Xanh lá khi xong, Vàng cam khi chờ duyệt, Xanh dương phát sáng khi đang chạy, Nét đứt khi chờ) kèm **thanh mini progress bar `3px` dưới đáy mỗi thẻ bước**.
3. **Tầng Tiến Độ 3 — Dải Đợt Thực Thi `Wave 0 → Wave 1 → ...` (`Per-Wave Progress Cards`):**
   - Mỗi hộp `Wave k` có tiêu đề `🌊 Wave k`, bộ đếm `done/total (pct%)`, **1 thanh mini progress bar `4px` ngay dưới tiêu đề Wave**, và danh sách các chip Sub-Task (`ST-1`, `ST-2`...) mang đúng màu nền trạng thái.

### 6.4. Code Demo 4 (`Task 7` — Khôi phục Hệ Màu Ngữ Nghĩa & Thanh Tiến Độ 3 Tầng trong `task_transparency.js` & `task_transparency.css`)

```css
/* File: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/task_transparency.css */
/* Semantic Status Colors for Stage Cards & Sub-Task Chips */
.twf-stage-pill.status-done,
.twf-subtask-chip.status-done {
  background: rgba(16, 52, 37, 0.62) !important;
  border: 1px solid rgba(34, 197, 94, 0.65) !important;
  color: #ecfdf5 !important;
}
.twf-stage-pill.status-waiting,
.twf-subtask-chip.status-waiting {
  background: rgba(120, 53, 15, 0.48) !important;
  border: 1.5px solid rgba(245, 158, 11, 0.78) !important;
  color: #fef3c7 !important;
}
.twf-stage-pill.status-running,
.twf-subtask-chip.status-running {
  background: rgba(30, 58, 138, 0.55) !important;
  border: 1.5px solid #38bdf8 !important;
  box-shadow: 0 0 12px rgba(56, 189, 248, 0.35) !important;
  color: #f0f9ff !important;
}
.twf-stage-pill.status-pending,
.twf-subtask-chip.status-pending {
  background: rgba(15, 23, 42, 0.55) !important;
  border: 1px dashed rgba(148, 163, 184, 0.35) !important;
  opacity: 0.78;
}
```
