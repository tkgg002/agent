# 08 — Tasks & Execution Checklist: `modular_agent_orchestrator` (Fork & Extend `Open-Makers` Go Engine + Queue-Driven + `/agent` Kernel)

> **Trạng thái:** `DONE` — Đã hoàn tất thực thi S4, kiểm định S5 (`DG3 PASS`) và đóng hồ sơ S6 (`verify_governance.py PASSED`).
> **Chiến lược cốt lõi (`Fork & Extend` — Tái sử dụng 80% Engine Go sẵn có):**
> Tuyệt đối **KHÔNG viết lại từ đầu bằng Python**. Chúng ta kế thừa trực tiếp codebase Go của `Open-Makers/ai-coding-agents-orchestrator` (đã có sẵn `internal/runner`, `internal/agent`, `internal/beads`, `internal/memory` SQLite FTS5, `internal/bus`, `internal/multiplexer/tmux.go`, `internal/tui`) và chỉ code bổ sung **5 Nâng Cấp Delta (`20%` phần còn thiếu)** trực tiếp vào các package Go của repo đó để đạt đúng mô hình `Scout -> Sorting -> Priority Queue -> Opus Planner Wave DAG -> Dynamic Opus/Sonnet Worktree Coders -> Reviewer + /agent Gatekeeper`.

---

## [x] Task 1 (Upgrade 1 — Đóng `F1`, `F2`, `F6`): Xây dựng `internal/queue`, `internal/scout` (Non-LLM `$0`) & `internal/sorter` trên nền `modernc.org/sqlite` có sẵn của Repo Go
- **[Mô tả & Phạm vi]:**
  - **Tái sử dụng 100%:** Driver pure-Go `modernc.org/sqlite` (`go.mod:15`) và bộ đánh chỉ mục `internal/memory/store.go:58-136` (`chunks_fts USING fts5` + WAL mode). Mở rộng `store.go` để index thêm `agent/memory/global/lessons/topics/*.md` và `agent/memory/workspaces/*/`.
  - **Code bổ sung (`internal/queue/store.go`, `internal/scout/scout.go`, `internal/sorter/sorter.go`):**
    1. `internal/queue/store.go`: Thêm bảng `task_queue` (`RAW -> PRIORITIZED -> PLANNING -> NEEDS_YOU -> READY -> RUNNING -> IN_REVIEW -> DONE`) và `autopilot_rounds` (`round_num`, `stage`, `blocking_count`, `advisory_count`, `turns`, `duration_sec`, `cost_usd`).
    2. `internal/scout/scout.go` (**Non-LLM Go Worker — `$0` token**): Polling GitHub Issues REST API / Jira REST API / Local Backlog JSON, lọc secret bằng regex `SECRET` của Hook `H3` (`hooks/agy/agy_guard.py:41-46`) rồi đẩy vào `Task Queue` (`RAW`).
    3. `internal/sorter/sorter.go`: Bốc các dòng `RAW`, chấm điểm `priority_score` (0–100) theo Severity/SLA/Labels (`blocker`, `security`, `production`) và đẩy sang `Priority Task Queue` (`PRIORITIZED`).
    4. Thêm lệnh CLI `orchestrator daemon` trong `cmd/orchestrator/main.go:40-68` để chạy vòng lặp tự động `Scout -> Sorter -> TaskRunner`.
- **[Context & Bằng chứng `file:line`]:**
  - Tái sử dụng `internal/memory/store.go:58-136` và `cmd/orchestrator/main.go:40-68`.
- **[Definition of Done (DoD)]:**
  - [x] `go test ./internal/queue/... ./internal/scout/... ./internal/sorter/...` PASS 100%; chứng minh `Scout` không gọi LLM (`$0`) và `Sorter` xếp đúng thứ tự `priority_score DESC`.

---

## [x] Task 2 (Upgrade 2 & 3 — Đóng `F3`, `F4`): Nâng cấp `internal/agent/pm.go`, `internal/gitclient` & `internal/orchestrator/task_runner.go` để hỗ trợ Dynamic Model Routing (`Opus` vs `Sonnet`) và Chạy Song Song Đa Coder theo `Wave 0..N` trên `git worktree`
- **[Mô tả & Phạm vi]:**
  - **Tái sử dụng 100%:**
    - `internal/runner/claude.go:64-100` (đã hỗ trợ truyền `--model` động theo `CompletionRequest.Model` tại `internal/runner/runner.go:37-42`).
    - `internal/beads/beads.go:103-145` (`AddDependency` và `Ready` đã trả về danh sách toàn bộ các bead không còn bị chặn ở Wave hiện tại!).
    - `internal/orchestrator/task_runner.go:823-865` (`createSubTaskBeads` đã nối dây phụ thuộc `DependsOn` giữa các sub-task).
  - **Code bổ sung (Độ trực tiếp vào package có sẵn):**
    1. **Dynamic Model Routing (`internal/agent/pm.go` & `task_runner.go:1070-1088`):** Thêm trường `Complexity string` (`"high" | "medium" | "low"`) và `Wave int` vào `agent.SubTask`. Khóa cứng `PMAgent` luôn dùng `model: "opus"`; khi `TaskRunner.implementBead` gọi `CoderAgent`, nếu `subTask.Complexity == "high"` thì truyền `Model: "opus"`, nếu `"medium" | "low"` thì truyền `Model: "sonnet"`.
    2. **Git Worktree Isolation (`internal/gitclient/gitclient.go:26-132`):** Thêm 2 hàm `WorktreeAdd(worktreePath, branch string)` và `WorktreeRemove(worktreePath string)` vào `GitClient`.
    3. **Parallel Wave Execution (`internal/orchestrator/task_runner.go:913-941`):** Sửa vòng lặp trong `executeBeads`: thay vì `nextBead` chỉ lấy `ready[0]` chạy tuần tự 1 con (`task_runner.go:1001`), lấy **toàn bộ slice `ready` của Wave hiện tại** từ `beads.Ready`, tạo `git worktree` riêng cho từng bead (`.orchestrator/worktrees/<bead_id>`), và bật goroutines (`errgroup`) cho nhiều `Coder` (`A..K`) chạy song song, sau đó merge về nhánh task khi xong Wave!
- **[Context & Bằng chứng `file:line`]:**
  - Độ trực tiếp trên `internal/orchestrator/task_runner.go:823-1006`, `internal/beads/beads.go:103-145`, `internal/gitclient/gitclient.go:26-132`, và `internal/runner/claude.go:64-100`.
- **[Definition of Done (DoD)]:**
  - [x] `go test ./internal/orchestrator/... ./internal/gitclient/...` PASS; kiểm chứng Wave 0 có 2 sub-task độc lập được chạy song song trên 2 `git worktree` (1 `opus`, 1 `sonnet`) và tự động mở khóa (`unblock`) sub-task ở Wave 1 khi cả 2 `Close`.

---

## [x] Task 3 (Upgrade 4 & 5 — Đóng `F5` & Gắn Lõi `/agent`): Xây dựng `internal/agentgate` (Trạm Kiểm Soát `/agent`) & Sổ Cái Chi Phí `Autopilot Cost ($) Ledger`
- **[Mô tả & Phạm vi]:**
  - **Tái sử dụng 100%:**
    - `internal/runner/runner.go:52-76` (`ResolveSkills` tự động nối `<skills>...</skills>` vào `SystemPrompt`).
    - `internal/runner/claude.go:146-152` & `internal/orchestrator/task_runner.go:75-80` (đã thu thập sẵn `InputTokens`, `OutputTokens`, `agentDurations`, `fixRounds`).
  - **Code bổ sung (`internal/agentgate/gate.go` & cập nhật `task_runner.go`):**
    1. **Điểm gắn 1 — System Prompt Injection:** Cấu hình `skills.Loader` nạp trực tiếp `/Users/trainguyen/Documents/work/agent/GEMINI.md`, `memory/global/lessons/INDEX.md` và `skills/*/SKILL.md` vào `SystemPrompt` của `PM`, `Coder`, `QA`, `Security`.
    2. **Điểm gắn 2 — External Gatekeeper (`internal/agentgate/gate.go`):**
       - Tại cuối `runDecomposeGate` (`task_runner.go:702` — trước khi mở cổng `waitArtifact` chờ User `Approve`): tự động gọi `laya_gate.py run --stage dg1` & `dg2` và `verify_governance.py --stage S3`.
       - Tại `implementAndReview` (`task_runner.go:455` — sau khi `Coder` viết code và `QA/Security` review): tự động quét `git diff` bằng regex `PROD_MOCK_POLLUTION` (`hooks/agy/agy_guard.py:51-53` — chặn mock `*Fn`/`sqlmock`), chạy `mutation_gate.py` (`3-Mutant Gate >= 3 KILLED`), chạy `laya_gate.py run --stage dg3` và `verify_governance.py`. Nếu bất kỳ cổng nào `FAIL` $\rightarrow$ đẩy vào vòng `fixRounds` tiếp theo bắt `Coder` sửa!
    3. **Sổ cái Chi phí (`Cost $ USD`):** Tính `cost_usd` từ `InputTokens`/`OutputTokens` theo đơn giá `Opus` / `Sonnet` cho từng Round (`Plan review`, `Code review`, `Local QA`) và lưu vào `autopilot_rounds` + `05_progress`.
- **[Context & Bằng chứng `file:line`]:**
  - Kết nối `internal/orchestrator/task_runner.go:437-500` & `674-717` với `/Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:10-24`, `tooling/mutation_gate.py:8-11`, `hooks/agy/agy_guard.py:51-53`.
- **[Definition of Done (DoD)]:**
  - [x] Test tích hợp chứng minh: nếu `Coder` lén nhét `*Fn` mock hoặc rớt `mutation_gate.py` / `DG3`, `TaskRunner` chặn không cho qua `PipelineDone` và tự động kích hoạt vòng Fix.

---

## [x] Task 4 (Upgrade UI — Mở rộng `internal/tui` & `internal/multiplexer`): Bổ sung 3 Tab Hiển Thị (`Wave 0..N DAG`, `Autopilot Cost & Gate Ledger`, `Multi-Worker Grid`) vào Bubble Tea TUI có sẵn
- **[Mô tả & Phạm vi]:**
  - **Tái sử dụng 100%:** Bộ khung Bubble Tea TUI sẵn có trong `internal/tui/` (`model.go`, `agent_panel.go`, `control.go`, `statusbar.go`, `sysmon.go`) và `internal/multiplexer/tmux.go` (`CreateSession`, `NewPane`, `SpawnAgent`).
  - **Code bổ sung (`internal/tui/wave_dag_view.go`, `internal/tui/autopilot_view.go`):**
    1. Thêm chế độ xem **Wave 0..N DAG** (hiển thị các cột Wave, trạng thái `NEEDS YOU`, `READY`, `BLOCKED`, `RUNNING`, `DONE`, model `Opus`/`Sonnet` của từng bead).
    2. Thêm bảng **Autopilot Rounds & Cost (`$`) + `/agent` Gates (`DG1–DG3`, `3-Mutant Gate`)** ngay trong `control.go` / `statusbar.go`.
- **[Context & Bằng chứng `file:line`]:**
  - Mở rộng trực tiếp trên `internal/tui/model.go` và `internal/multiplexer/tmux.go:11-96`.
- **[Definition of Done (DoD)]:**
  - [x] Build binary `go build ./cmd/orchestrator` và chạy `go test ./...` toàn repo PASS 100%.


