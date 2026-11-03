---
tier: T2
suffix: add_edit_task_and_subtask_crud
---

# 💡 09 Tasks Solution & Code Demo — Bổ sung Tính năng Chỉnh sửa Task (`Edit Task` — `✏️ Sửa Task`) & Sub-Task (`add_edit_task_and_subtask_crud`)

## 1. Nguyên Nhân Gốc Rễ (`VERIFIED` kèm `file:line`)
1. **Thiếu điểm chạm UI `✏️ Sửa Task` & Chế độ Edit trên `#dispatch-modal`:**
   - Thanh ngữ cảnh `#cross-screen-context-bar` (`internal/webui/static/index.html:68-91`), danh sách Hàng đợi Sidebar (`internal/webui/static/app.js:869-911`), và thẻ Task trên Bảng Wave (`internal/webui/static/app.js:2239-2248`) chỉ có nút Chạy (`▶`), Dừng (`⏹`) và Xóa (`🗑`).
   - Hàm `openDispatchModal()` và `submitNewTask(ev)` (`internal/webui/static/app.js:2744-2786, 3072-3153`) chỉ phục vụ tạo mới (`POST /api/tasks`), chưa có trạng thái `editingTaskKey` để nạp dữ liệu Task hiện hữu lên form và gửi yêu cầu cập nhật.
2. **Thiếu API `GET /api/tasks/detail` & `POST /api/tasks/update` và hàm cập nhật SQLite `UpdateTaskDetails`:**
   - `Handler()` (`internal/webui/server.go:639-663`) chưa có route lấy chi tiết đầy đủ của Task (tách riêng `clean_description` và `brd_content` từ `brd_source.md`) và route cập nhật Task.
   - `EnqueueRaw` (`internal/queue/store.go:183-193`) không cập nhật cột `assigned_model`, và `handleCreateTask` (`internal/webui/server.go:851-860`) luôn reset `stage` về `READY` đồng thời nối lặp `### Tài Liệu BRD / PRD Đính Kèm:` nếu gọi lại.

---

## 2. Phương Án Thiết Kế Duy Nhất Tối Ưu (Single Best Architecture — Đúng Layer & Single-Responsibility)

### 2.1. Layer `internal/queue`: Tách module `internal/queue/store_task_update.go`
- Bổ sung 2 hàm chuẩn mực trên `queue.Store`:
  - `GetByExternalID(ctx context.Context, externalID string) (QueuedTask, error)`: Truy vấn trực tiếp 1 bản ghi từ `task_queue` theo `external_id`.
  - `UpdateTaskDetails(ctx context.Context, externalID, title, description, complexity, assignedModel string, labels []string) (QueuedTask, error)`: Cập nhật nguyên tử `title`, `description`, `complexity`, `assigned_model`, `labels_json`, `updated_at` trong `task_queue` dưới `s.mu.Lock()`, giữ nguyên `stage` hiện tại của Task (trừ khi User chủ động tích chọn chạy lại Phase 1 `S1–S3`).

### 2.2. Layer `internal/webui`: Tách module `internal/webui/task_edit_handler.go`
- Đăng ký 2 route mới trong `Server.Handler()` (`internal/webui/server.go:646-648`):
  - `mux.HandleFunc("/api/tasks/detail", s.handleTaskDetail)` (`GET /api/tasks/detail?task_key=...`)
  - `mux.HandleFunc("/api/tasks/update", s.handleUpdateTask)` (`POST /api/tasks/update`)
- Trong `internal/webui/task_edit_handler.go`:
  1. **`splitCleanDescriptionAndBRD(rawDesc, artifactDir string) (cleanDesc, brdContent string)`**:
     - Tách sạch phần sau `\n\n### Tài Liệu BRD / PRD Đính Kèm:\n` khỏi `rawDesc` để trả về `clean_description` nguyên bản; ưu tiên đọc `brd_source.md` trong `.orchestrator/tasks/<TaskKey>/` (bỏ dòng header `> **Intake Mode:** ...`) để trả về `brd_content` chính xác 100%, chống nhân đôi BRD khi lưu nhiều lần.
  2. **`handleTaskDetail` (`GET /api/tasks/detail`)**:
     - Trả về JSON `taskDetailResponse` gồm: `external_id`, `title`, `clean_description`, `brd_content`, `target_repo`, `affected_repos`, `runner`, `model`, `complexity`, `milestone`, `epic`, `assignee`, `depends_on`, `critical`, `stage`, `sub_tasks`, và `sub_tasks_text` (mỗi dòng 1 Sub-Task hoặc định dạng `# 08 Tasks` để User sửa nhanh trên Modal).
  3. **`handleUpdateTask` (`POST /api/tasks/update`)**:
     - Nhận `updateTaskRequest` (cùng cấu trúc với `createTaskRequest` + `original_task_key` và cờ `replan_s1_s3`).
     - Gọi `s.QueueStore.UpdateTaskDetails(...)` cập nhật SQLite.
     - Cập nhật `.orchestrator/tasks/<TaskKey>/brd_source.md` và `01_requirements.md`.
     - Nếu User chỉnh sửa danh sách Sub-Tasks trong `#inp-task-subtasks`, parse và cập nhật `subtasks.json` + `08_tasks.md` + đồng bộ `llm_analysis.json` qua `syncEditedSubTasksToLLMAnalysis` (chống `hasLLMSubTaskMismatch` tại `task_inspector.go:1176-1184` ghi đè ngược `subtasks.json` khi `replan_s1_s3 == false`).
     - Nếu `req.AutoAutopilot` (`replan_s1_s3 == true`), xóa `llm_analysis.json` cũ để AI Model (`invokeArchitectLLM`) phân tích lại yêu cầu mới, tái sinh bộ hồ sơ `S1–S3` (`01_requirements.md`, `08_tasks.md`, `09_tasks_solution.md`, `10_dg0_critique.md`, `subtasks.json`) và đẩy thông báo cập nhật lên Topic Telegram.
  4. **Hỗ trợ `action: "edit_subtask"` trong `handleTaskSubTasks` (`internal/webui/task_inspector.go`)**:
     - Cho phép sửa trực tiếp `Title`, `TargetRepo`, `ContextFileLine`, `Description`, `TestDoD`, `WaveIndex`, `Assignee`, `Model` của từng Sub-Task và đồng bộ `llm_analysis.json` qua `syncEditedSubTasksToLLMAnalysis`.

### 2.3. Layer Frontend (`internal/webui/static/index.html` & `internal/webui/static/app.js`)
- Bổ sung nút **`✏️ Sửa Task`** tại 3 vị trí:
  1. Thanh ngữ cảnh `#cross-screen-context-bar` (`index.html:87`): `<button id="ctx-btn-edit-task" class="stage-action-btn" onclick="openEditTaskModal('')" title="Chỉnh sửa thông tin Task, BRD, Repo, Model & Sub-Tasks">✏️ Sửa Task</button>`
  2. Thẻ Task trong Hàng đợi Sidebar (`renderSidebarTaskQueue` tại `app.js:876-909`): Nút `✏️` cạnh nút `🗑`.
  3. Thẻ Task trên Bảng Wave (`renderWaveBoard` tại `app.js:2241-2247`): Nút `✏️` cạnh nút `🗑`.
- Nâng cấp `#dispatch-modal` với biến `editingTaskKey`:
  - Khi gọi `openEditTaskModal(taskKey, ev)`: gọi `GET /api/tasks/detail?task_key=...`, điền đầy đủ dữ liệu vào các trường của `#dispatch-modal`, đổi tiêu đề Modal thành `✏️ Chỉnh sửa Task: <TaskKey>`, đổi nhãn checkbox cuối thành `⚡ Chạy lại Phase 1 (S1–S3) với AI Model để tái sinh Giải pháp & Sub-Tasks theo nội dung vừa sửa` (mặc định bỏ chọn nếu muốn giữ nguyên Stage hiện tại, hoặc tích chọn để tái sinh Plan S1–S3), và đổi nút Submit thành `💾 Lưu Thay Đổi Task (<TaskKey>)`.
  - Khi gọi `openDispatchModal()`: reset `editingTaskKey = ""`, khôi phục tiêu đề `+ Giao việc mới` và form trống.

---

## 3. Code Demo

### 3.1. `internal/queue/store_task_update.go` (Module mới)
```go
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GetByExternalID returns a single QueuedTask matching externalID.
func (s *Store) GetByExternalID(ctx context.Context, externalID string) (QueuedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return QueuedTask{}, errors.New("queue: external_id is required")
	}
	var t QueuedTask
	var labelsJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, external_id, source, title, description, severity, labels_json, sla_hours,
		        priority_score, stage, wave_num, complexity, assigned_model, worktree_path, created_at, updated_at
		 FROM task_queue WHERE external_id = ?`, externalID,
	).Scan(
		&t.ID, &t.ExternalID, &t.Source, &t.Title, &t.Description, &t.Severity,
		&labelsJSON, &t.SLAHours, &t.PriorityScore, &t.Stage, &t.WaveNum,
		&t.Complexity, &t.AssignedModel, &t.WorktreePath, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return QueuedTask{}, fmt.Errorf("queue: get by external_id %q: %w", externalID, err)
	}
	_ = json.Unmarshal([]byte(labelsJSON), &t.Labels)
	return t, nil
}

// UpdateTaskDetails updates mutable metadata of an existing task without blindly resetting its stage.
func (s *Store) UpdateTaskDetails(ctx context.Context, externalID, title, description, complexity, assignedModel string, labels []string) (QueuedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return QueuedTask{}, errors.New("queue: external_id is required")
	}
	if strings.TrimSpace(title) == "" {
		return QueuedTask{}, errors.New("queue: title is required")
	}
	if complexity == "" {
		complexity = "high"
	}
	labelsBytes, err := json.Marshal(labels)
	if err != nil {
		labelsBytes = []byte("[]")
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx,
		`UPDATE task_queue
		 SET title = ?, description = ?, labels_json = ?, complexity = ?, assigned_model = ?, updated_at = ?
		 WHERE external_id = ?`,
		strings.TrimSpace(title), strings.TrimSpace(description), string(labelsBytes),
		complexity, strings.TrimSpace(assignedModel), now, externalID,
	)
	if err != nil {
		return QueuedTask{}, fmt.Errorf("queue: update task details: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return QueuedTask{}, fmt.Errorf("queue: task %q not found", externalID)
	}
	s.mu.Unlock()
	t, getErr := s.GetByExternalID(ctx, externalID)
	s.mu.Lock()
	return t, getErr
}
```

### 3.2. `internal/webui/task_edit_handler.go` (Module mới)
```go
package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/trainguyen/ai-orchestrator/internal/scout"
)

type taskDetailResponse struct {
	ExternalID       string       `json:"external_id"`
	Title            string       `json:"title"`
	CleanDescription string       `json:"clean_description"`
	BRDContent       string       `json:"brd_content"`
	TargetRepo       string       `json:"target_repo"`
	AffectedRepos    []string     `json:"affected_repos"`
	Runner           string       `json:"runner"`
	Model            string       `json:"model"`
	Complexity       string       `json:"complexity"`
	Milestone        string       `json:"milestone"`
	Epic             string       `json:"epic"`
	Assignee         string       `json:"assignee"`
	DependsOn        []string     `json:"depends_on"`
	Critical         bool         `json:"critical"`
	Stage            string       `json:"stage"`
	SubTasks         []SubTaskDTO `json:"sub_tasks"`
	SubTasksText     string       `json:"sub_tasks_text"`
}

func splitCleanDescriptionAndBRD(rawDesc, artifactDir string) (string, string) {
	const brdMarker = "### Tài Liệu BRD / PRD Đính Kèm:"
	cleanDesc := strings.TrimSpace(rawDesc)
	var embeddedBRD string
	if idx := strings.Index(cleanDesc, brdMarker); idx >= 0 {
		embeddedBRD = strings.TrimSpace(cleanDesc[idx+len(brdMarker):])
		cleanDesc = strings.TrimSpace(cleanDesc[:idx])
	}
	if artifactDir != "" {
		if rawBytes, err := os.ReadFile(filepath.Join(artifactDir, "brd_source.md")); err == nil {
			text := strings.TrimSpace(string(rawBytes))
			if strings.HasPrefix(text, "> **Intake Mode:**") {
				if nl := strings.Index(text, "\n"); nl >= 0 {
					text = strings.TrimSpace(text[nl+1:])
				}
			}
			if text != "" {
				return cleanDesc, text
			}
		}
	}
	return cleanDesc, embeddedBRD
}
```

### 3.3. `internal/webui/subtask_edit_service.go` (Tách Module Đơn Trách Nhiệm ở `TASK-EDIT-04`)
- Tách riêng toàn bộ nghiệp vụ Sub-Task CRUD & đồng bộ hồ sơ (`applyCustomSubTasksOnEdit`, `syncEditedSubTasksToLLMAnalysis`, `applySubTaskEditOrRoleUpdate`, `syncUpdatedTaskRequirementsDoc`) khỏi `internal/webui/task_edit_handler.go`.
- Khi `replan_s1_s3 == false`, `syncUpdatedTaskRequirementsDoc` cập nhật ngay `.orchestrator/tasks/<TaskKey>/01_requirements.md` bằng `BuildUnifiedS1RequirementsDoc` để Tab `S1 Requirements` trên WebUI phản ánh tức thì `Title`, `Description`, `BRDContent`, `TargetRepo`, `AffectedRepos`, `Model` và `SubTasks` mới.
- Trong `internal/webui/static/app.js`, `fillDefaultSubTasksInModal(forceClear)` chỉ xóa `#inp-task-subtasks` khi `!window.editingTaskKey || forceClear === true`, ngăn chặn sự kiện `oninput="autoPreviewSubTasksOnTitle()"` xóa mất danh sách Sub-Tasks khi User đang sửa Tiêu đề Task.

