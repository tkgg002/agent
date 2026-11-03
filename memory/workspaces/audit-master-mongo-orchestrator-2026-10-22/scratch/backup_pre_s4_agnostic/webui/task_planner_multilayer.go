package webui

import (
	"fmt"
	"path/filepath"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

// decomposeSelfUpgradeOrchestratorSubTasks builds the 6-Layer Full-Stack Sub-Tasks (CORE-01-S01..S06)
// for self-upgrading ai-orchestrator from HTTP API to WebUI UX/UI.
func (s *Server) decomposeSelfUpgradeOrchestratorSubTasks(qt queue.QueuedTask, affectedRepos []string, prefix, safeParent, defaultModel string) []SubTaskDTO {
	orchRepo := s.Root
	for _, rp := range affectedRepos {
		if filepath.Base(rp) == "ai-orchestrator" {
			orchRepo = rp
			break
		}
	}
	repoName := filepath.Base(orchRepo)
	if repoName == "" || repoName == "." {
		repoName = "ai-orchestrator"
	}
	wtPath := filepath.Join(".orchestrator", "worktrees", safeParent, repoName)
	grpWave0 := fmt.Sprintf("Nhóm 1 (Wave 0): Tầng 1 & 2 — HTTP API Endpoints, Targeted Builder & Deep Multi-Layer S3 Planner (`%s`)", repoName)
	grpWave1 := fmt.Sprintf("Nhóm 2 (Wave 1): Tầng 3 & 4 — Co-Architect Studio Direct Q&A & Telegram Forum Sub-Topic Bridge (`%s`)", repoName)
	grpWave2 := fmt.Sprintf("Nhóm 3 (Wave 2): Tầng 5 & 6 — WebUI UX/UI Inspector (`index.html`/`app.css`/`app.js`) & E2E Race Verification (`%s`)", repoName)

	s01 := fmt.Sprintf("%s01", prefix)
	s02 := fmt.Sprintf("%s02", prefix)
	s03 := fmt.Sprintf("%s03", prefix)
	s04 := fmt.Sprintf("%s04", prefix)
	s05 := fmt.Sprintf("%s05", prefix)
	s06 := fmt.Sprintf("%s06", prefix)

	return []SubTaskDTO{
		{
			SubTaskID:       s01,
			ParentKey:       qt.ExternalID,
			GroupTitle:      grpWave0,
			WaveIndex:       0,
			WorkflowStep:    "Wave 0",
			Title:           "Tầng 1 (HTTP API & Runtime Engine): Nâng cấp handleIntake, Live Progress 17/17, Targeted Repo Check & Persistent TASK-BUG-* Self-Healing trong `server.go` / `task_self_heal_loop.go`",
			TargetRepo:      orchRepo,
			TargetRepoName:  repoName,
			WorktreePath:    wtPath,
			TargetFiles:     "ai-orchestrator/internal/webui/task_self_heal_loop.go:1",
			ContextFileLine: "ai-orchestrator/internal/webui/task_self_heal_loop.go:1 (func (s *Server) runAutonomousS4S5WithSelfHealing(ctx context.Context, taskKey, activeRunner string, stepDelay time.Duration) error)",
			Description:     "Tự động gọi notifyTelegramNewTaskIntakeAndTopic ngay khi tạo task mới; bổ sung hasDirectGoFiles trong buildTargetedRepoCheckCommand tránh lỗi 'no Go files in ./test'; hoàn tất Step 16/17..17/17 (100% DONE) trong case 'qa'; và thay vòng lặp retry mù bằng Persistent TASK-BUG-* Self-Healing tới khi DONE.",
			TestDoD:         "Pass `go vet ./...` và `go test -race ./internal/webui/...`.",
			Status:          "pending",
			Assignee:        "Coder",
			Model:           defaultModel,
		},
		{
			SubTaskID:       s02,
			ParentKey:       qt.ExternalID,
			GroupTitle:      grpWave0,
			WaveIndex:       0,
			WorkflowStep:    "Wave 0",
			Title:           "Tầng 2 (Deep Planner & 12-Step Fractal Wave Engine): Nâng cấp bộ lập kế hoạch S3 đa tầng & vòng đời 12 bước cho từng Sub-Task trong `task_planner_multilayer.go` & `subtask_wave_engine.go`",
			TargetRepo:      orchRepo,
			TargetRepoName:  repoName,
			WorktreePath:    wtPath,
			TargetFiles:     "ai-orchestrator/internal/webui/task_planner_multilayer.go:1",
			ContextFileLine: "ai-orchestrator/internal/webui/task_planner_multilayer.go:1 (func (s *Server) decomposeSelfUpgradeOrchestratorSubTasks)",
			Description:     "Chuẩn hóa mã Sub-Task ASCII từ ExternalID (buildCleanSubTaskIDPrefix), mở rộng inspectTargetRepoSymbols quét cả .go/.ts/.tsx/.py/.sql/.html/.css/.js, xuất 09_tasks_solution.md có Bảng Executive Summary + Ma trận Điểm Đau -> Điểm Chạm, và thực thi đủ 12 bước / 6 Wave (Step 1/12..12/12) cho mỗi Sub-Task.",
			TestDoD:         "Pass `go vet ./...` và kiểm chứng 6 Sub-Tasks CORE-01-S01..S06.",
			Status:          "pending",
			Assignee:        "Coder",
			Model:           defaultModel,
		},
		{
			SubTaskID:       s03,
			ParentKey:       qt.ExternalID,
			GroupTitle:      grpWave1,
			WaveIndex:       1,
			WorkflowStep:    "Wave 1",
			DependsOn:       []string{s01, s02},
			Title:           "Tầng 3 (Co-Architect Studio Direct Q&A): Trả lời trực tiếp Root Cause (`F1..Fn` + `file:line`), Giải pháp kỹ thuật & Code Demo trong `task_chat_knowledge.go`",
			TargetRepo:      orchRepo,
			TargetRepoName:  repoName,
			WorktreePath:    wtPath,
			TargetFiles:     "ai-orchestrator/internal/webui/task_chat_knowledge.go:1",
			ContextFileLine: "ai-orchestrator/internal/webui/task_chat_knowledge.go:1 (func (s *Server) renderContextualDiscussReply)",
			Description:     "Bổ sung extractTaskDeepKnowledge bóc tách trực tiếp Root Cause (F1..Fn kèm file:line), Giải pháp kỹ thuật và Code Demo từ 13_analysis.md & 09_tasks_solution.md để trả lời thẳng câu hỏi kỹ thuật của User trên WebUI & Telegram, đồng thời tự động nâng cấp lịch sử chat_history.json cũ.",
			TestDoD:         "Pass `go test -race ./internal/webui/...`.",
			Status:          "pending",
			Assignee:        "Coder",
			Model:           defaultModel,
		},
		{
			SubTaskID:       s04,
			ParentKey:       qt.ExternalID,
			GroupTitle:      grpWave1,
			WaveIndex:       1,
			WorkflowStep:    "Wave 1",
			DependsOn:       []string{s01, s02},
			Title:           "Tầng 4 (Telegram Forum Sub-Topic Bridge): Tự động tạo Sub-Topic theo `ExternalID` (`createForumTopic`) & định tuyến 2 chiều `message_thread_id` trong `telegram_forum_topic.go`",
			TargetRepo:      orchRepo,
			TargetRepoName:  repoName,
			WorktreePath:    wtPath,
			TargetFiles:     "ai-orchestrator/internal/webui/telegram_forum_topic.go:1",
			ContextFileLine: "ai-orchestrator/internal/webui/telegram_forum_topic.go:1 (func (s *Server) ensureForumTopicForTask)",
			Description:     "Tự động tạo Forum Sub-Topic riêng trên Telegram Supergroup đặt tên theo Mã Task / ExternalID (🧩 [Core-01] ...), lưu TaskThreadIDs, gửi thông báo & thẻ duyệt Sub-Task đúng vào Sub-Topic và nhận lệnh/câu hỏi 2 chiều qua message_thread_id.",
			TestDoD:         "Pass `go vet ./...` và kiểm chứng API Telegram Forum Topic.",
			Status:          "pending",
			Assignee:        "Coder",
			Model:           defaultModel,
		},
		{
			SubTaskID:       s05,
			ParentKey:       qt.ExternalID,
			GroupTitle:      grpWave2,
			WaveIndex:       2,
			WorkflowStep:    "Wave 2",
			DependsOn:       []string{s03, s04},
			Title:           "Tầng 5 (WebUI UX/UI Inspector & Live Wave Controls): Đồng bộ giao diện `index.html`, `app.css`, `app.js` hiển thị 12-Step Sub-Task Wave, Forum Topic & Executive S3 Plan",
			TargetRepo:      orchRepo,
			TargetRepoName:  repoName,
			WorktreePath:    wtPath,
			TargetFiles:     "ai-orchestrator/internal/webui/static/app.js:1",
			ContextFileLine: "ai-orchestrator/internal/webui/static/app.js:1 (WebUI Task Inspector, 12-Step Sub-Task Wave Progress & Co-Architect Studio UI)",
			Description:     "Đồng bộ hóa payload JSON từ /api/tasks/detail, /api/tasks/subtasks, /api/tasks/chat và /api/settings/telegram lên giao diện WebUI để hiển thị tiến độ 12-Step / 6-Wave của từng Sub-Task, bảng Executive Summary trong S3 Plan và liên kết Telegram Thread ID.",
			TestDoD:         "Kiểm chứng render dữ liệu thật qua HTTP API /api/dashboard?task_key=Core-01 sạch lỗi.",
			Status:          "pending",
			Assignee:        "Coder",
			Model:           defaultModel,
		},
		{
			SubTaskID:       s06,
			ParentKey:       qt.ExternalID,
			GroupTitle:      grpWave2,
			WaveIndex:       2,
			WorkflowStep:    "Wave 2",
			DependsOn:       []string{s03, s04},
			Title:           "Tầng 6 (E2E Race Test & Governance Gate G1–G8): Kiểm thử tích hợp đầu-cuối (`-race`) & Laya Gate `DG0/DG1/DG2/DG3 (:8200)` trong `server_test.go`",
			TargetRepo:      orchRepo,
			TargetRepoName:  repoName,
			WorktreePath:    wtPath,
			TargetFiles:     "ai-orchestrator/internal/webui/server_test.go:1",
			ContextFileLine: "ai-orchestrator/internal/webui/server_test.go:1 (End-to-End Integration & Concurrency Race Detector Suite)",
			Description:     "Thực thi bộ test tích hợp đầu-cuối kiểm chứng toàn bộ 6 tầng từ API, Deep Planner, 12-Step Sub-Task Wave, Co-Architect Direct Q&A, Telegram Sub-Topic theo ExternalID tới WebUI payload với cờ -race và xuất biên lai vật lý EV-...",
			TestDoD:         "Pass `go test -v -race ./internal/webui/...` và `go build -o bin/orchestrator ./cmd/orchestrator`, đạt chuẩn G1–G8.",
			Status:          "pending",
			Assignee:        "Tester",
			Model:           defaultModel,
		},
	}
}

func buildSelfUpgradeOrchestratorCodeDemo() string {
	return `// === [Tầng 1: HTTP API, Live Progress 17/17 & Targeted Repo Builder] internal/webui/task_self_heal_loop.go:1 ===
// CORE-01-S01: Tự động tạo Forum Sub-Topic Telegram theo ExternalID ngay khi tạo task mới & lọc thư mục Go hợp lệ
isoMeta, subTasks, _, _ := s.generateDeepTaskArtifacts(r.Context(), createdTask, activeRunner, routedModel)
go s.notifyTelegramNewTaskIntakeAndTopic(context.Background(), createdTask, isoMeta, subTasks)

// Trong buildTargetedRepoCheckCommand(repoPath, subTasks): chỉ đưa thư mục có file .go trực tiếp vào go build
if hasDirectGoFiles(filepath.Join(repoPath, pkgDir)) {
	pkgs = append(pkgs, targetPkg)
}

// === [Tầng 2: Deep Multi-Layer Planner & 12-Step Fractal Sub-Task Wave Engine] internal/webui/subtask_wave_engine.go:1 ===
// CORE-01-S02: Chuẩn hóa mã ASCII (CORE-01-S01..S06) & chạy đủ 12 bước / 6 Wave cho từng Sub-Task (Step 1/12..12/12)
s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 1, 12, 10, "Step 1/12 · S0 Boot H1 (:8200) & S1 Sub-Task Intake", ...)
s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 4, 12, 35, "Step 4/12 · S3 Plan, Code Demo & Laya DG1/DG2 (:8200)", ...)
// Sau khi User APPROVE (Step 5/12) -> Step 6/12..8/12 (S4 Restore-Point + Real Code Apply + DG2) -> Step 9/12..12/12 (5A EV-... + 5B DG3 + S6 DONE)

// === [Tầng 3: Co-Architect Studio Direct Root Cause & Solution Q&A] internal/webui/task_chat_knowledge.go:1 ===
// CORE-01-S03: Trích xuất trực tiếp Root Cause (F1..Fn + file:line), Giải pháp & Code Demo trả lời thẳng câu hỏi của User
rootCauses, solutions, codeSnippet, fileTouches := s.extractTaskDeepKnowledge(qt, isoMeta, subTasks)

// === [Tầng 4: Telegram Forum Sub-Topic theo ExternalID & 2-Way Thread Routing] internal/webui/telegram_forum_topic.go:1 ===
// CORE-01-S04: Tạo Sub-Topic đặt tên theo Mã Task / ExternalID ("🧩 [Core-01] ...") & định tuyến 2 chiều message_thread_id
topicName := formatTelegramTaskTopicName(qt)
threadID := s.ensureForumTopicForTask(ctx, cfg, qt, false) // POST /bot<token>/createForumTopic -> lưu TaskThreadIDs[qt.ExternalID]`
}
