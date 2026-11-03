# 13. Deep Root-Cause Analysis — Kiểm tra Toàn diện Hệ thống `ai-orchestrator` (5 Lỗi Chí mạng khiến `/brain-delegate` Trả lời Rác Ghép Chuỗi Tĩnh)

## 1. Tổng quan Kiểm định Hệ thống `ai-orchestrator`
Thực hiện chỉ thị của anh (*"ko đào sâu vào export, cái mày cần làm là kiểm tra lại hệ thống ai-orchestrator"*), em đã rà soát toàn bộ luồng xử lý từ WebUI Chat (`internal/webui/task_chat.go`, `internal/webui/task_chat_knowledge.go`), Telegram Topic Chat (`internal/webui/telegram_forum_topic.go`), Codebase Scanner & Sub-Task Planner (`internal/webui/task_inspector.go`, `internal/webui/task_planner_multilayer.go`) cho tới tầng thực thi LLM (`internal/runner/antigravity.go`, `internal/runner/factory.go`).

Kết quả đối chiếu từng `file:line` xác nhận 100% nhận định của anh: **Khi anh chat vào khung `💬 THẢO LUẬN KIẾN TRÚC` (`Slot A · /brain-delegate · gemini-4-argon (antigravity)`), hệ thống `ai-orchestrator` hoàn toàn KHÔNG gọi tới model `gemini-4-argon` (hay bất kỳ LLM nào), mà đang chạy một hàm ghép chuỗi tĩnh (`renderContextualDiscussReply`) cắt dán bừa bãi các dòng từ file cũ và thư mục build `dist/`.**

---

## 2. Truy vết `file:line` của 5 Lỗi Chí mạng trong `ai-orchestrator`

### Lỗi #1: Hoàn toàn KHÔNG nối qua `gemini-4-argon` (`AntigravityRunner`) trong `internal/webui/` + Lỗi CLI `exit code 55`
1. **Không có bất kỳ lời gọi LLM nào trong `internal/webui/` (`VERIFIED`):**
   - Tìm kiếm toàn bộ repo `ai-orchestrator` cho thấy `runner.NewAntigravityRunner` và `.Complete(ctx, ...)` chỉ xuất hiện trong `internal/agent/{pm,coder,qa,security,ux_reviewer}.go` và `internal/tui/chat.go`.
   - Trong toàn bộ package `internal/webui/`, số lượng lời gọi tới `AntigravityRunner` là **`0`**.
   - Cụ thể tại `internal/webui/task_chat.go:974` (`handleTaskPlanningChat`) và `internal/webui/telegram_forum_topic.go:205` (`handleTelegramTechnicalQuestion`), khi nhận câu hỏi/chỉ đạo của anh, code gọi thẳng vào:
     - `s.buildContextualDiscussReply(qt, msg, roleBadge, slotLabel, selectedModel)` (`internal/webui/task_chat_knowledge.go:16-24`)
     - `-> s.renderContextualDiscussReply(...)` (`internal/webui/task_chat_knowledge.go:26-118`)
   - Tại `internal/webui/task_chat_knowledge.go:82-83`, hàm này chỉ ghép chuỗi giả danh `gemini-4-argon`:
     ```go
     b.WriteString(fmt.Sprintf("**[%s · `%s` · `%s`]** Dạ anh, em xin trả lời trực tiếp câu hỏi của anh (*\"%s\"*) trên **`%s` (%s)**:\n\n",
         roleBadge, slotLabel, selectedModel, qText, qt.ID, strings.TrimSpace(qt.Title)))
     ```
     Sau đó nó đổ nguyên các mảng chuỗi tĩnh `rootCauses`, `solutions`, `codeBlock`, `subTaskLines` đã cắt từ đĩa ra mà không hề đọc hay suy luận câu lệnh mới của anh!
2. **Lỗi tầng `internal/runner/antigravity.go` khi chạy headless CLI (`VERIFIED`):**
   - Tại `internal/runner/antigravity.go:115-116`, `resolveCommand` chỉ `TrimPrefix(lowerModel, "antigravity/")` nhưng không strip hậu tố `" (antigravity)"` từ `selectedModel` của UI (`"gemini-4-argon (antigravity)"`).
   - Tại `internal/runner/antigravity.go:151-154`, nhánh `gemini` chỉ truyền `[]string{"-p", prompt}` mà **thiếu cờ `--skip-trust`** và biến môi trường `GEMINI_CLI_TRUST_WORKSPACE=true`. Thực nghiệm runtime (`task-79`) chứng minh nếu thiếu `--skip-trust`, binary `/Users/trainguyen/.nvm/versions/node/v22.18.0/bin/gemini` thoát ngay lập tức với **`exit code 55`** (`Gemini CLI is not running in a trusted directory`).
   - Đồng thời, truyền `-m gemini-4-argon` trực tiếp xuống `@google/gemini-cli` sẽ bị Google API trả về `429 limit: 0 for model: gemini-4-argon` (vì `gemini-4-argon` là định danh model trên Antigravity IDE, trong khi `gemini -p` không truyền `-m` sẽ dùng model mặc định hợp lệ của Gemini CLI và trả lời tiếng Việt chính xác).

---

### Lỗi #2: Râu ông nọ cắm cằm bà kia (`FindMatchingAgentWorkspace` lấy nhầm Workspace khác & `extractTaskDeepKnowledge` cắt nhầm dòng)
1. **Lấy nhầm hồ sơ của Workspace khác (`internal/webui/task_inspector.go:868-935` — `VERIFIED`):**
   - Trong `FindMatchingAgentWorkspace(qt)` (`task_inspector.go:892-915`), thuật toán tách các từ `>= 4` ký tự từ `qt.Title` (`"con centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn chuyển qua posgret."` -> từ khóa `mongo`, `trans`, `his`) rồi quét các thư mục trong `/Users/trainguyen/Documents/work/agent/memory/workspaces/` **mà không kiểm tra `qt.TargetRepo` (`centralized-export-service`) có khớp với workspace đó không**.
   - Hậu quả: Nó khớp nhầm vào workspace `migration-mongo-to-postgres-trans-proxy-history` (vốn của dự án `core-trans-proxy-history-service` có CDC Worker NATS/Kafka DLQ) và bốc câu:
     `Nếu gặp permanent error (BSON document size limit > 16MB, type mismatch): Đẩy vào NATS/Kafka Dead Letter Queue (DLQ) để CMS báo động và operator can thiệp manual.`
     nhét thẳng vào câu trả lời của `TASK-13`!
2. **Gán nhầm biến trả về thứ 4 & Regex bắt nhầm chữ `"lỗi"` (`internal/webui/task_chat_knowledge.go:160-226` — `VERIFIED`):**
   - Tại `task_chat_knowledge.go:160`:
     ```go
     _, _, ws09, ws13 := s.FindMatchingAgentWorkspace(qt)
     ```
     Trong khi chữ ký của `FindMatchingAgentWorkspace` (`task_inspector.go:868`) trả về `(wsDir, plan12File, sol09File, req01File string)` — tức biến thứ 4 là `01_requirements_*.md` chứ không phải `13_analysis_*.md`!
   - Tại `task_chat_knowledge.go:208, 224`, vòng lặp quét dòng chỉ kiểm tra:
     ```go
     if strings.Contains(lowerLn, "nguyên nhân") || strings.Contains(lowerLn, "root cause") || strings.Contains(lowerLn, "vì sao") || strings.Contains(lowerLn, "lỗi")
     ```
     Vì trong file `13_analysis.md` sinh tự động (`task_inspector.go:3262-3264`) có dòng tiêu đề:
     `## 3. Kịch Bản Tái Hiện Lỗi / Kiểm Chứng Trước Khi Sửa (Gate G2: Red -> Green Reproduction)`
     chứa chữ `"lỗi"`, nên `inCauseSec` bật thành `true` và bốc nguyên 3 dòng quy trình nội bộ của Orchestrator (`Gate G2`, ` Bước RED: VerifySubTaskCodeOnDisk`, `Bước GREEN: evidence_run.py`) nhét vào mục `### 🔍 1. Nguyên nhân gốc rễ`!

---

### Lỗi #3: Scanner quét lọt vào `dist/` (`trans-his.model.js:3`), trích xuất hàm polyfill `function adopt(value)` và bọc cứng trong ````go`
1. **Thiếu `dist/` và `build/` trong bộ lọc của `inspectTargetRepoSymbols` (`internal/webui/task_inspector.go:2955-2960` — `VERIFIED`):**
   - Tại `task_inspector.go:2957`:
     ```go
     if name == ".git" || name == ".orchestrator" || name == "vendor" || name == "node_modules" || name == "scratch" || strings.HasPrefix(name, "backups_") {
         return filepath.SkipDir
     }
     ```
     Hàm này **không hề skip `dist`, `build`, `out`, `.next`, `coverage`, `target`**.
   - Khi quét repo `centralized-export-service`, các ứng viên cùng số điểm keyword được sắp xếp theo alphabet đường dẫn (`task_inspector.go:3017`: `return candidates[i].RelPath < candidates[j].RelPath`). Vì `"dist/models/..."` đứng trước `"models/..."` theo bảng chữ cái (`'d' < 'm'`), file build `dist/models/trans-his/trans-his.model.js` bị xếp làm ứng viên số 1 (`TASK-13-M01`)!
2. **Bắt nhầm hàm polyfill `__awaiter` của TypeScript compiler (`internal/webui/task_inspector.go:3044` — `VERIFIED`):**
   - Tại `task_inspector.go:3044`, bộ tìm symbol kiểm tra `strings.HasPrefix(trimmed, "function ")`.
   - Ở đầu file `dist/models/trans-his/trans-his.model.js` do TypeScript biên dịch ra có helper `__awaiter`:
     `Line 3: function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }`
   - Thế là dòng số 3 (`function adopt(value)`) bị lấy làm `PrimarySymbol` cho `TASK-13-M01` và trích đoạn 12 dòng đầu file `.js` build này bị lấy làm `Code Demo`!
3. **Hardcode tag ````go` cho mọi ngôn ngữ (`VERIFIED`):**
   - Tại `internal/webui/task_chat_knowledge.go:103`, `internal/webui/task_inspector.go:2635`, `3235`, `3364`, code viết cứng:
     ````go
     b.WriteString("```go\n" + codeBlock + "\n```\n\n")
     ````
     khiến đoạn JavaScript/TypeScript bị đóng khung trong ````go`.

---

### Lỗi #4: Ghép chuỗi tĩnh lặp từ máy móc (`String Interpolation` trong `DecomposeMultiRepoCodebaseSubTasks`)
- Tại `internal/webui/task_inspector.go:1172-1199`:
  ```go
  title := fmt.Sprintf("Triển khai kiến trúc & nghiệp vụ cốt lõi cho %s trên `%s` (`%s`)", qt.Title, repoName, filepath.Base(primaryHit.RelPath))
  desc := fmt.Sprintf("Triển khai nghiệp vụ '%s' tại %s:%d trong nhánh worktree cô lập của repo %s.", qt.Title, primaryHit.RelPath, primaryHit.Line, repoName)
  ```
- Vì `qt.Title` là câu chat của anh (*"con centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn chuyển qua posgret."*), phép `fmt.Sprintf` này đẻ ra câu văn lặp từ vô nghĩa:
  *"Triển khai kiến trúc & nghiệp vụ cốt lõi cho con centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn chuyển qua posgret. trên centralized-export-service (trans-his.model.js)"*.

---

### Lỗi #5: Không nhận diện Chỉ đạo Đổi hướng Kiến trúc (`classifyChatActionMode`) & Khai gian trạng thái `DONE` ảo
1. **Bỏ lọt câu chỉ đạo kiến trúc tự nhiên (`internal/webui/task_chat.go:1019-1041` — `VERIFIED`):**
   - Trong `classifyChatActionMode(message)`:
     - Nhánh `"revise_solution"` (`task_chat.go:1032-1037`) chỉ bắt vài cụm cứng: `"sửa lại giải pháp"`, `"cập nhật giải pháp"`, `"đổi giải pháp"`, `"sửa phương án"`, `"thay đổi thiết kế"`, `"bổ sung vào giải pháp"`, `"sửa 09_tasks_solution"`.
     - Khi anh viết tự nhiên: *"hiện tại mày đang làm là bỏ cái export type cũ đi, đè lên nó. giải pháp của anh là tạo 1 cái type v2 qua postgrest, còn v1 vẫn là cái cũ"*, hàm không thấy từ khóa nào khớp nên rơi xuống default `"discuss"` (`task_chat.go:1040`)!
2. **Không Re-Open Task & Báo cáo `2/2 DONE (100% DONE)` ảo (`internal/webui/task_chat_knowledge.go:70-79, 108-115` — `VERIFIED`):**
   - Vì rơi vào `"discuss"`, hệ thống không chuyển `TASK-13` từ `DONE` về `NEEDS_YOU`, không xóa `.user-approved`, không re-plan lại `subtasks.json`, và ở cuối hàm `renderContextualDiscussReply` (`task_chat_knowledge.go:114`) nó đọc trạng thái cũ trong DB rồi in ra:
     `- Trạng thái hiện tại của TASK-13: Stage DONE | Tất cả Sub-Tasks hiện tại đã hoàn tất (100% DONE).`
   - Ngay cả khi vào nhánh `"revise_solution"` (`task_chat.go:843-921`), code hiện tại cũng chỉ nối thêm vài dòng markdown vào cuối `09_tasks_solution.md` chứ **chưa hề re-open `qt.Status` về `NEEDS_YOU`, chưa reset `subtasks.json` và chưa gọi LLM (`gemini-4-argon`) để thiết kế lại kiến trúc và Sub-Tasks**.
