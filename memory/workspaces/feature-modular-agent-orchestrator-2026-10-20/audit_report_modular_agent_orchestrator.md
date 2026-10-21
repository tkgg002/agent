# Final Adversarial Audit Report: `modular_agent_orchestrator` (S5 Deep Re-Audit & Closed-Loop `S5 -> S3` Re-Plan)

> **Trạng thái Kiểm định:** Đã thực hiện **Adversarial Re-Audit (Phản biện toàn diện cấp độ từng dòng `git diff` trên commit `54e36e4`)**.
> Phát hiện **6 lỗi thực thi & suy diễn Smoke-Test (`AF-1` đến `AF-6`, gồm `3 P0 Critical`, `2 P1 High`, `1 P2 Medium`)** từng bị bỏ lọt ở lần tự kiểm trước.
> Tuân thủ nghiêm ngặt **Rule #9.7(d) & Hook `H4b` (Closed-Loop `S5 -> S3` Re-Plan)**: **TUYỆT ĐỐI KHÔNG tự ý sửa thẳng source code Go khi đang Audit** — đã ghi bài học mới vào `lessons.md` (`256` bài học), chạy lại hội đồng `/laya_router` (`DG3` & `DG1/DG2`), cập nhật đặc tả khắc phục 1:1 theo từng sơ hở (`AF-1..AF-6`) vào `09_tasks_solution_modular_agent_orchestrator.md` + `08_tasks_modular_agent_orchestrator.md`, và **DỪNG CHỜ lệnh `APPROVE` của User**.

---

## 1. Line-by-Line Diff & Task Critique (Phản biện từng dòng update / `git diff` & từng task trên commit `54e36e4`)

### 1.1. Bảng Tổng hợp 6 Sơ hở Phát hiện qua Phản biện Từng Dòng Code (`AF-1` $\rightarrow$ `AF-6`)

| Mã Sơ hở | Mức độ | File & Dòng Code Thực Tế (`commit 54e36e4`) | Bản chất Lỗi & Tại sao Báo cáo trước bị lọt (Root Cause) | Trạng thái |
| :---: | :---: | :--- | :--- | :---: |
| **`AF-1`** | 🔴 **P0 Critical** | [`internal/runner/claude.go:38-77`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/claude.go#L38-L77)<br>[`internal/agent/coder.go:164-176`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agent/coder.go#L164-L176)<br>[`internal/orchestrator/wave_parallel_test.go:22-54`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/wave_parallel_test.go#L22-L54) | **Đứt mạch truyền `ModelOverride` ở tầng `ClaudeRunner` + Ảo tưởng Mockup Test:**<br>Tại `coder.go:164-176`, `cp.ModelOverride` (`"opus"` hoặc `"sonnet"`) được truyền vào `r.Complete(ctx, runner.CompletionRequest{Model: mdl})`. Trong khi `CodexRunner` (`codex.go:110-115`) và `OpenCodeRunner` (`opencode.go:35-40`) có đọc `req.Model`, thì **`ClaudeRunner.Complete` (`claude.go:41`) lại gọi `r.startStreamingProcess(ctx, prompt, req.SystemPrompt)` mà KHÔNG truyền `req.Model`**, và tại `claude.go:71-73` chỉ kiểm tra trường tĩnh `if r.Model != ""`! Do đó, khi chạy với `ClaudeRunner`, mọi `ModelOverride` đều bị phớt lờ.<br>**Lý do báo cáo trước không phát hiện:** Bài test `wave_parallel_test.go:22-54` dùng một **mockup struct `recordingCoderAgent`** chặn ngay ở ranh giới `TaskRunner` để đọc `payload.ModelOverride` mà không hề gọi vào `CoderAgent.Run` hay `ClaudeRunner.Complete`! | ❌ **FAIL — Cần sửa ở S4 sau APPROVE** |
| **`AF-2`** | 🔴 **P0 Critical** | [`internal/gitclient/gitclient.go:124-167`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/gitclient/gitclient.go#L124-L167)<br>[`internal/orchestrator/task_runner.go:1053-1190`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go#L1053-L1190)<br>[`internal/agent/coder.go:188-200`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agent/coder.go#L188-L200) | **Cắt `git worktree` từ `HEAD` làm rơi file chưa commit giữa các Wave + Ghi đè Last-Writer-Wins + Xung đột file `.orchestrator/`:**<br>1. `WorktreeAdd` (`gitclient.go:134`) chạy `git worktree add --detach <wtDir> HEAD`. Toàn bộ các file TDD test sinh ở Phase A (`generateAllTests` tại `task_runner.go:1054`) và các file code do `Wave 0` vừa sửa được `SyncWorktreeFiles` copy về `tr.root` (`task_runner.go:1181`) **chưa được `git commit` vào `HEAD`**. Hệ quả: khi `Wave 1` tạo worktree mới từ `HEAD`, worktree của `Wave 1` bị thiếu sạch toàn bộ code của `Wave 0` và bộ TDD test!<br>2. `SyncWorktreeFiles` (`gitclient.go:162`) gọi `os.WriteFile(dst, data, 0o600)` đè thẳng lên `tr.root` theo kiểu *Last-Writer-Wins*, không phát hiện nếu 2 sub-task cùng Wave sửa chung 1 file.<br>3. `CoderAgent.WithRootAndModel` (`coder.go:107-121`) vẫn dùng chung `a.ws` (`.orchestrator/`), nên 2 goroutine chạy song song cùng ghi đè `.orchestrator/coder_output.md` và `.orchestrator/changes.md` (`coder.go:188-200`). | ❌ **FAIL — Cần sửa ở S4 sau APPROVE** |
| **`AF-3`** | 🔴 **P0 Critical** | [`cmd/orchestrator/daemon_cmd.go:75-153`](file:///Users/trainguyen/Documents/work/ai-orchestrator/cmd/orchestrator/daemon_cmd.go#L75-L153) | **Lệnh `orchestrator daemon` dừng ở `PRIORITIZED` không gọi `TaskRunner` + Gán cứng Badge `PASS` tĩnh:**<br>1. Trong `08_tasks_modular_agent_orchestrator.md:16`, yêu cầu đặt ra cho `orchestrator daemon` là chạy vòng lặp `Scout -> Sorter -> TaskRunner`. Thực tế `daemon_cmd.go:75-153` chỉ gọi `PollBacklogFile` (`RAW`) và `TriageAll` (`RAW -> PRIORITIZED`), hoàn toàn **không gọi `qStore.PopNextPrioritized` và không gọi `TaskRunner`**.<br>2. Nghiêm trọng hơn, tại `daemon_cmd.go:133-139`, mảng `snap.Gates` được **gán cứng bằng các chuỗi hằng số tĩnh**: `{Name: "S0-Constitution", Status: "PASS"}`, `{Name: "DG1+DG2", Status: "PASS"}`, `{Name: "H4b-ZeroMock", Status: "PASS"}`, `{Name: "H5-NoGitPush", Status: "PASS"}` mà không hề gọi `agentgate.Gatekeeper`! Việc dùng dòng in ra này ở báo cáo trước để làm bằng chứng runtime cho 4 cổng trên là **Suy diễn Smoke-Test / Báo cáo vượt quá thực tế code**. | ❌ **FAIL — Cần sửa ở S4 sau APPROVE** |
| **`AF-4`** | 🟠 **P1 High** | [`internal/agentgate/gate.go:53-92, 155-204`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/gate.go#L53-L204)<br>[`internal/orchestrator/task_runner.go:473-485, 692-735`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go#L473-L485) | **Hội chứng "Unwired Helper Facade" — 4/5 hàm của `agentgate.Gatekeeper` có `0` Production Callers:**<br>Mặc dù `internal/agentgate/gate.go` đã cài đặt `LoadCoreConstitutionPrompt` (`53-92`), `RunLayaGate` (`155-172`), `RunMutationGate` (`175-190`), `RunGovernanceLinter` (`193-204`) và có unit test trong `gate_test.go`, nhưng khi kiểm tra toàn bộ codebase production (`task_runner.go` và `cmd/orchestrator/*.go`), **chỉ duy nhất `ScanProductionFiles` được gọi tại `task_runner.go:475`**. Bốn hàm còn lại chưa được nối vào `runDecomposeGate` (`task_runner.go:692-735`), `implementAndReview` (`task_runner.go:455-510`) hay lúc khởi tạo System Prompt của Agent. | ❌ **FAIL — Cần sửa ở S4 sau APPROVE** |
| **`AF-5`** | 🟠 **P1 High** | [`internal/queue/store.go:311-332`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/queue/store.go#L311-L332)<br>[`internal/orchestrator/helpers.go:226-231`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/helpers.go#L226-L231) | **`queue.Store.RecordRound` không có Caller thật & `helpers.go` tính thiếu 5x chi phí USD khi Coder chạy `opus`:**<br>1. Hàm `qStore.RecordRound` (`store.go:311-332`) chỉ được gọi trong `store_test.go` và `wave_dag_view_test.go`, chưa được gọi trong `TaskRunner` hay `daemon_cmd.go`, nên bảng `autopilot_rounds` luôn trống (`0` dòng) khi chạy production.<br>2. Tại `helpers.go:226-230`, `emitTokenStats` gán cứng `modelTier := "sonnet"` và chỉ đổi sang `"opus"` khi `role == bus.RolePM`. Khi `CoderAgent` chạy sub-task có `complexity == "high"` bằng model `opus` (`$15/$75` mỗi 1M token), hàm này vẫn nhân theo bảng giá `sonnet` (`$3/$15`), làm lệch giảm 5 lần chi phí thực tế của Coder! | ❌ **FAIL — Cần sửa ở S4 sau APPROVE** |
| **`AF-6`** | 🟡 **P2 Medium** | [`internal/tui/wave_dag_view.go:44-144`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/tui/wave_dag_view.go#L44-L144)<br>[`internal/tui/model.go:1-120`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/tui/model.go#L1-L120) | **`RenderWaveDAGDashboard` chưa nối vào Bubble Tea `tui.Model` và luôn hiển thị `(all worker slots idle)`:**<br>`RenderWaveDAGDashboard` mới chỉ được gọi in tĩnh ra stdout trong `daemon_cmd.go:141` với `Workers: nil` (nên luôn in `(all worker slots idle)`), chưa được gắn thành tab/view tương tác trong `internal/tui/model.go` hoặc `internal/multiplexer/tmux.go`. | ❌ **FAIL — Cần sửa ở S4 sau APPROVE** |

---

### 1.2. Phản biện Chi tiết Từng File trong `git diff` (`24 files changed` trên commit `54e36e4`)

1. **[`internal/memory/store.go:249-346`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/memory/store.go#L249-L346) & [`internal/memory/memory.go:73-75`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/memory/memory.go#L73-L75):**
   - **Điểm đạt (`VERIFIED`):** `IndexExternalDir` chặn đúng file `> 256*1024` bytes (`store.go:331-333`, tuân thủ Rule #21), băm `sha256` chống index trùng (`store.go:270-277`), và `memory.Reindex` (`memory.go:73-75`) giữ lại các key ngoại vi có dấu `:` (`strings.Contains(path, ":")`).
   - **Điểm hạn chế nhỏ:** Nếu một file bài học ngoại vi (`agent-lessons:*`) bị xóa vật lý trên đĩa, vòng lặp `Reindex` tại `memory.go:73-75` hiện skip toàn bộ key có dấu `:` nên chưa dọn orphan entry của file ngoại vi đã xóa.
2. **[`internal/queue/store.go:1-380`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/queue/store.go#L1-L380):**
   - **Điểm đạt (`VERIFIED`):** SQLite WAL + `busy_timeout=5000` (`store.go:88`), `external_id TEXT NOT NULL UNIQUE` (`store.go:113`), `PopNextPrioritized` dùng transaction nguyên tử chuyển `PRIORITIZED -> RUNNING` (`store.go:254-295`).
   - **Điểm lỗi (`AF-5`):** `RecordRound` (`store.go:311-332`) chưa được gọi từ `TaskRunner`.
3. **[`internal/scout/scout.go:1-192`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/scout/scout.go#L1-L192):**
   - **Điểm đạt (`VERIFIED`):** Non-LLM thuần Go (`LLMTokensUsed() == 0`), `MaskSecrets` (`scout.go:27-43`) che URI credentials, `ghp_*`, `sk-*`, `Bearer` trước khi lưu SQLite; `PollGitHubIssues` và `PollJiraIssues` dùng `http.Client{Timeout: 15 * time.Second}` và `io.LimitReader(resp.Body, 5<<20)`.
4. **[`internal/sorter/sorter.go:1-115`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/sorter/sorter.go#L1-L115):**
   - **Điểm đạt (`VERIFIED`):** Chấm điểm `ComputeScore` trong `[0, 100]` kết hợp tìm kiếm FTS5 BM25 (`w.memStore.Search(query, 3)` tại `sorter.go:83`), tự động nâng `complexity = "high"` khi `score >= 70.0` (`sorter.go:100-102`).
5. **[`internal/config/config.go:145-178`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/config/config.go#L145-L178):**
   - **Điểm đạt (`VERIFIED`):** `ResolveModelForComplexity` định tuyến `"high"|"critical"|"opus" -> "opus"` và `"low"|"medium"|default -> "sonnet"`, cho phép override qua `HighComplexityModel` / `StandardModel`.
6. **[`internal/agent/pm.go:35-46, 643-644, 1103-1171`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agent/pm.go#L35-L46):**
   - **Điểm đạt (`VERIFIED`):** `NewPMAgent` mặc định `cfg.Model = "opus"`, `SubTask` có thêm `Complexity` và `Wave`, `ComputeSubTaskWaves` (`pm.go:1103-1171`) tính tầng DAG và chống đệ quy vô tận khi có chu trình bằng mảng 3 trạng thái `visiting[key]`.
7. **[`internal/agent/coder.go:29-30, 104-121, 164-200`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agent/coder.go#L104-L200) & [`internal/runner/claude.go:38-77`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/claude.go#L38-L77):**
   - **Điểm lỗi (`AF-1` & `AF-2.3`):** `CoderAgent.Run` truyền `req.Model = mdl`, nhưng `ClaudeRunner.Complete` (`claude.go:41, 71-73`) bỏ quên `req.Model` (chỉ đọc `r.Model`). Đồng thời `coder.go:188-200` ghi đè chung `.orchestrator/coder_output.md` và `.orchestrator/changes.md` khi nhiều goroutine chạy song song.
8. **[`internal/gitclient/gitclient.go:124-167`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/gitclient/gitclient.go#L124-L167) & [`internal/orchestrator/task_runner.go:894-1205`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go#L894-L1205):**
   - **Điểm đạt một phần:** Đã chia nhóm sub-task theo `Wave` và chạy song song qua `sync.WaitGroup` + khóa `tr.statsMu` chống data race trên memory state.
   - **Điểm lỗi (`AF-2.1` & `AF-2.2`):** `WorktreeAdd` checkout từ `HEAD` nên mất các file uncommitted của Phase A (`generateAllTests`) và của `Wave` trước; `SyncWorktreeFiles` ghi đè Last-Writer-Wins không kiểm tra xung đột file giữa các worker cùng Wave.
9. **[`internal/agentgate/gate.go:1-224`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/gate.go#L1-L224) & [`internal/executil/executil.go:142-151`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/executil/executil.go#L142-L151):**
   - **Điểm đạt (`VERIFIED`):** `executil.Sanitize` chặn các lệnh nguy hiểm (`H5`), `ScanProductionFiles` được gọi trong `implementAndReview` (`task_runner.go:475`).
   - **Điểm lỗi (`AF-4`):** 4 hàm `LoadCoreConstitutionPrompt`, `RunLayaGate`, `RunMutationGate`, `RunGovernanceLinter` chưa được gọi trong `task_runner.go`.
10. **[`internal/runner/cost.go:1-58`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/cost.go#L1-L58) & [`internal/orchestrator/helpers.go:200-245`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/helpers.go#L200-L245):**
    - **Điểm lỗi (`AF-5`):** `CalculateCostUSD` tính đúng công thức nhưng `helpers.go:226-230` luôn truyền `modelTier := "sonnet"` cho `bus.RoleCoder`, làm sai lệch chi phí khi Coder chạy `opus`.
11. **[`internal/tui/wave_dag_view.go:1-155`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/tui/wave_dag_view.go#L1-L155) & [`cmd/orchestrator/daemon_cmd.go:1-171`](file:///Users/trainguyen/Documents/work/ai-orchestrator/cmd/orchestrator/daemon_cmd.go#L1-L171):**
    - **Điểm lỗi (`AF-3` & `AF-6`):** `daemon_cmd.go` dừng ở `PRIORITIZED`, gán cứng mảng `Gates` `PASS`, và `RenderWaveDAGDashboard` chưa nối vào `internal/tui/model.go`.

---

## 2. Anti-Smoke-Speculation & Real Runtime Trace Verification (Chống suy diễn Smoke Test, Mockup, Hallucination & Phối hợp User Trace Runtime thật)

### 2.1. Tự thú & Bóc tách Rạch ròi: Đâu là Mockup/Smoke-Test Illusion vs. Đâu là Runtime Thật
1. **Phát hiện dùng Mockup trong Test (`wave_parallel_test.go:22-54`):**
   - Trong `TestExecuteSubTasksInOrder_ParallelWavesAndDynamicModelRouting`, chúng tôi đã tạo struct giả `recordingCoderAgent` (`type recordingCoderAgent struct`) cài đặt interface `agent.Agent` để bắt `payload.ModelOverride` ở ranh giới `TaskRunner`.
   - **Tác hại thực tế:** Chính vì dùng mockup `recordingCoderAgent` thay vì chạy xuyên qua `CoderAgent.Run -> ClaudeRunner.Complete`, bài test đã báo `PASS` giả tạo trong khi `ClaudeRunner.startStreamingProcess` (`claude.go:71-73`) thực tế **bỏ quên `req.Model`** (`AF-1`)!
2. **Phát hiện Hardcode Badge trong CLI Smoke Test (`daemon_cmd.go:133-139`):**
   - Khi chạy `go run ./cmd/orchestrator daemon --once`, dòng output:
     `▸ /agent Kernel Gates: [S0-Constitution: PASS] [DG1+DG2: PASS] [H4b-ZeroMock: PASS] [H5-NoGitPush: PASS] [FTS5-BM25: PASS(5 new)]`
     thực chất chỉ có `[FTS5-BM25: PASS(5 new)]` là kết quả runtime thật từ `memStore.IndexExternalDir`, còn 4 nhãn `S0-Constitution`, `DG1+DG2`, `H4b-ZeroMock`, `H5-NoGitPush` là **chuỗi hardcode tĩnh tại `daemon_cmd.go:133-138`** (`AF-3`). Ở báo cáo trước, việc trích dẫn dòng output này như bằng chứng runtime của 4 cổng đó là **vi phạm quy tắc Chống Suy diễn Smoke Test (Rule #9.7b)**.

### 2.2. Những gì ĐÃ chạy Runtime thật 100% (Verified Real Runtime)
- **SQLite WAL Queue & FTS5 BM25 Indexer:** Đã tạo file DB vật lý `.orchestrator/queue.db` và `.orchestrator/memory.db`, index thật 5 file bài học từ `/Users/trainguyen/Documents/work/agent/memory/global/lessons/topics/*.md` vào bảng `chunks_fts USING fts5`, truy vấn BM25 thật trong `sorter.Worker.ComputeScore`.
- **Scout Non-LLM (`$0` token) + Secret Masking (`H3`) + Sorter Priority Queue:** Đã đọc file JSON backlog thật, che mật khẩu/token thật, chấm điểm và ghi vào bảng `task_queue` (`RAW -> PRIORITIZED`).
- **Git Worktree CLI thật (`git worktree add` / `remove`):** Đã gọi binary `git` thật tạo và xóa worktree trên đĩa trong `gitclient_worktree_test.go` và `wave_parallel_test.go`.

### 2.3. Danh sách Điểm BẮT BUỘC Phối hợp với User để Trace Runtime Value Thật (Rule #9.7c)
Để không rơi vào Hallucination hoặc test bằng giả lập ở chặng thực thi tiếp theo, chúng tôi xin báo cáo rõ **4 điểm chạm môi trường thật cần phối hợp cùng anh (User) để trace runtime value**:
1. **Trace lệnh gọi CLI `claude` / `codex` / `opencode` thật với `--model opus` vs `--model sonnet`:**
   - Sau khi sửa `AF-1` (`claude.go:71-73`), cần phối hợp chạy 1 task thật qua binary `claude` CLI trên máy của anh (hoặc trace log `executil.Command` thực tế) để xác nhận tiến trình con `ps aux | grep claude` nhận đúng đối số `--model opus` cho Planner/High-Complexity Coder và `--model sonnet` cho Standard Coder, đồng thời đối chiếu `InputTokens`/`OutputTokens` thật trả về từ `stream-json`.
2. **Trace `PollGitHubIssues` và `PollJiraIssues` trên API thật:**
   - Hiện tại `PollGitHubIssues` và `PollJiraIssues` mới được test qua `httptest.NewServer` cục bộ. Để trace runtime value thật từ GitHub/Jira của anh, cần anh cung cấp `GITHUB_TOKEN` (hoặc repo public) / `JIRA_API_TOKEN` + URL project để chạy `orchestrator daemon --once` kéo issue thật về `.orchestrator/queue.db`.
3. **Trace `executeBeads` với CLI `bd` (Beads) thật:**
   - Nếu máy anh đã cài binary `bd` (`beads`), chúng ta sẽ trace luồng `createSubTaskBeads -> beads.Ready -> executeBeads` trên DAG của `bd` thật (bên cạnh luồng fallback `executeSubTasksInOrder`).
4. **Trace Multi-Terminal `tmux` Grid & Interactive Bubble Tea TUI (`internal/tui` & `internal/multiplexer/tmux.go`):**
   - Vì môi trường chạy lệnh tự động của Agent không có TTY tương tác cho `tea.Program.Run()`, sau khi nối `RenderWaveDAGDashboard` vào `internal/tui/model.go` (`AF-6`), cần phối hợp với anh bật trực tiếp `orchestrator` trên terminal thật / `tmux` session của anh để kiểm tra giao diện 3 tầng thời gian thực.

---

## 3. Concurrency, Timeout/Heartbeat & Data Continuity Audit (Kiểm tra Đa luồng, Multi-Worker, Timeout & Liên tục dữ liệu)

- **Phản biện Đa Worker (`WorkerCount > 1`) & Git Worktree Isolation (`AF-2`):**
  - Dù các biến bộ nhớ trong `TaskRunner` (`tr.state`, `tr.codingStarted`, `tr.agentDurations`, `tr.loadedModel`) đã được khóa bằng `tr.statsMu.Lock()` và vượt qua `go test -race`, nhưng **ở tầng hệ thống file vật lý trên đĩa**, việc chạy đa worker cùng Wave đang có **3 lỗ hổng nghiêm trọng (`AF-2`)**:
    1. **Đứt gãy tính liên tục dữ liệu giữa `Phase A (TDD) -> Wave 0 -> Wave 1`:** `WorktreeAdd` cắt từ `HEAD` nên `Wave 1` không nhìn thấy các file chưa commit của `Wave 0` và của `generateAllTests`.
    2. **Ghi đè Last-Writer-Wins trong cùng Wave:** `SyncWorktreeFiles` dùng `os.WriteFile` ghi đè thẳng file sau lên file trước nếu 2 sub-task trong cùng 1 Wave chạm vào cùng 1 file.
    3. **Tranh chấp file `.orchestrator/coder_output.md` và `.orchestrator/changes.md`:** Tất cả các `CoderAgent` clone trong cùng Wave vẫn trỏ về cùng `a.ws` (`.orchestrator/`), gây race condition trên đĩa khi 2 goroutine cùng gọi `a.ws.WriteFile("coder_output.md", ...)`.
- **Timeout & Heartbeat:**
  - `ClaudeRunner.Complete` (`claude.go:50`) bọc channel bằng `withHeartbeat(ch, defaultHeartbeatInterval)` (15s) và `Scout` giới hạn `http.Client{Timeout: 15 * time.Second}`.
  - Tuy nhiên, khi `Gatekeeper` gọi các subprocess kiểm định (`RunLayaGate`, `RunMutationGate`, `RunGovernanceLinter` — `AF-4`), cần bổ sung `context.WithTimeout` rõ ràng để tránh treo `TaskRunner` nếu Laya engine hoặc test mutant bị nghẽn.

---

## 4. Anti-Speculation & Honesty Check (Kiểm tra suy diễn & báo cáo láo)

- Chúng tôi đã rà soát đối chiếu 100% giữa tuyên bố trong báo cáo cũ với từng dòng code thật trong commit `54e36e4`:
  - **Nhận lỗi thẳng thắn:** Báo cáo trước đã **kết luận vội vàng (Over-claiming)** ở 3 điểm: (1) Khẳng định Dynamic Model Routing hoạt động cho mọi runner trong khi `ClaudeRunner` bỏ sót `req.Model` (`AF-1`); (2) Khẳng định `orchestrator daemon` đã nối xong toàn bộ vòng lặp và các cổng `[S0-Constitution: PASS] [DG1+DG2: PASS]...` trong khi `daemon_cmd.go:133-153` dừng trước `TaskRunner` và in mảng `GateStatusBadge` tĩnh (`AF-3`); (3) Khẳng định `internal/agentgate` và `autopilot_rounds` đã khép kín vòng lặp trong khi 4/5 hàm của `Gatekeeper` và `qStore.RecordRound` chưa có caller trong `task_runner.go` (`AF-4`, `AF-5`).
  - Toàn bộ 6 điểm này đã được phơi bày minh bạch kèm `file:line` chính xác tại Mục 1.1 của báo cáo này, không che giấu bất kỳ thiếu sót nào.

---

## 5. Self-Improvement Loop (Vòng lặp Phản tỉnh & Khắc phục — Lessons Audit & Closed-Loop `S5 -> S3` Re-Plan)

- **Ghi bài học mới vào `memory/global/lessons.md` (Rule #5 & Rule #6):**
  - Đã chèn bài học mới lên đầu `memory/global/lessons.md` (`wc -l` tăng từ `1986` $\rightarrow$ `1993` dòng) và chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py` đồng bộ `256/256` bài học (`L001 [2026-10-20] Bỏ quên truyền req.Model xuống CLI subprocess (ClaudeRunner), dùng Mockup Agent chặn ở ranh giới Orchestrator che khuất lỗi Runner, cắt git worktree từ HEAD làm rơi file chưa commit giữa các Wave, viết hàm Gatekeeper/Ledger không có Caller thật và in Badge PASS tĩnh trong CLI`).
  - Các bài học trước đó đã đối chiếu: `L052` / `L060` / `L069` (`Mock Testing Illusion`), `L104` / `L112` (`Premature Done Without Wiring All Callers`).
- **Kích hoạt Vòng lặp Khép kín `S5 -> S3 Re-Plan` (Rule #9.7d & Hook `H4b`):**
  - Giữ nguyên hiện trạng mã nguồn Go (`Propose-Only`), quay về `S3` cập nhật `09_tasks_solution_modular_agent_orchestrator.md` (ánh xạ 1:1 **Giải pháp cho từng Sơ hở `AF-1` $\rightarrow$ `AF-6`** theo đúng bài học `L005`) và `08_tasks_modular_agent_orchestrator.md`, chạy lại `DG1`/`DG2` và trình User `APPROVE`.

---

## 6. Kết quả QC & Nhận xét từ `/laya_router` (`DG3`)

- **Summary file (Adversarial Re-Audit):** `06_laya_summary_dg3_20261020T163241.json`
- **Raw file:** `06_laya_raw_dg3_20261020T163241.json`
- **Gate Verdict:** `PASS` (`trusted_rounds: ["r3"]`, kiểm định 100% các phát hiện `AF-1..AF-5` trên code thật kèm đối chứng `control_true` & `control_false`)
- **Chi tiết các Claim phản biện đã được `/laya_router` (`DG3`) xác nhận:**
  1. `DG3-AF1-CLAUDE-RUNNER-MODEL` (`target`): **`ACCEPT`** (`r1: CORRECT 0.6552`, `r2: CORRECT 0.3531`, `r3: YES 0.7807`) — Xác nhận `ClaudeRunner.Complete` (`claude.go:38-76`) không truyền `req.Model` vào `startStreamingProcess` và chỉ kiểm tra `r.Model != ""`.
  2. `DG3-AF2-WORKTREE-SYNC-AND-HEAD` (`target`): **`ACCEPT`** (`r1: CORRECT 0.5262`, `r2: CORRECT 0.2266`, `r3: YES 0.5531`) — Xác nhận `WorktreeAdd` (`gitclient.go:124-167`) chạy `git worktree add --detach worktreePath HEAD` và `SyncWorktreeFiles` ghi đè trực tiếp qua `os.WriteFile`.
  3. `DG3-AF3-DAEMON-STATIC-BADGES` (`target`): **`FLAG -> RESOLVED`** (`r1: CORRECT 0.3202`, `r2: CORRECT 0.1539`, `r3: YES 0.3713` — cả 3 vòng đều xác nhận `CORRECT/YES`, đã giải trình kèm `REF: /Users/trainguyen/Documents/work/ai-orchestrator/cmd/orchestrator/daemon_cmd.go:133-153` vào `04_decisions_feature-modular-agent-orchestrator-2026-10-20.md`) — Xác nhận `daemon_cmd.go` gán tĩnh mảng `GateStatusBadge` và dừng trước khi gọi `PopNextPrioritized` / `TaskRunner`.
  4. `DG3-AF4-HELPERS-SONNET-COST` (`target`): **`ACCEPT`** (`r1: CORRECT 0.8390`, `r2: CORRECT 0.5174`, `r3: YES 0.9105`) — Xác nhận `helpers.go:226-231` gán cứng `modelTier := "sonnet"` cho mọi role khác `bus.RolePM`, làm lệch giá khi `bus.RoleCoder` chạy `opus`.
  5. **Controls Calibration:** `DG3-CTRL-TRUE-1` (`control_true`), `DG3-CTRL-FALSE-1` (`control_false`), `DG3-CTRL-FALSE-2` (`control_false`) đạt chuẩn phân biệt đúng/sai tại `r3`.

---

## 7. Kiểm chứng Sau Khắc phục (Post-Remediation Line-by-Line Diff Critique & Final `DG3` Audit — Commit `fa6b410`)

Sau khi nhận lệnh `APPROVE` của User cho **Task 5**, chúng tôi đã thực thi khắc phục toàn bộ 6 sơ hở `AF-1` $\rightarrow$ `AF-6` tại commit `fa6b410` (`15 files changed, 980 insertions(+), 62 deletions(-)`) và rà soát phản biện lại từng dòng diff (`git diff HEAD~1..HEAD`):

1. **Khắc phục `AF-1` (`internal/runner/claude.go:38-78` & `internal/runner/claude_test.go:92-150`):**
   - `ClaudeRunner.Complete` truyền `req.Model` xuống `r.startStreamingProcess(ctx, prompt, req.SystemPrompt, req.Model)`.
   - `startStreamingProcess` ưu tiên `strings.TrimSpace(reqModel)` trước `strings.TrimSpace(r.Model)` khi nối `--model <model>` vào `args` của tiến trình OS `claude`.
   - Kiểm chứng bằng `TestClaudeRunner_Complete_ForwardsReqModelOverrideToProcessArgs` và `TestExecuteSubTasksInOrder_RealCoderAgentAndClaudeRunnerProcessArgsAndCrossWaveSeed`: 4 lệnh gọi `CoderAgent.Run -> ClaudeRunner.Complete` sinh ra đúng 4 tiến trình OS thật với `T1:--model opus`, `T2:--model sonnet`, `T3:--model sonnet`, `T4:--model opus`.
2. **Khắc phục `AF-2` (`internal/gitclient/gitclient.go:126-260`, `internal/orchestrator/task_runner.go:1059-1343`, `internal/agent/coder.go:193-210`):**
   - `WorktreeAdd` gọi `g.SeedWorktreeFromRoot(worktreePath)` (`gitclient.go:140`) để copy toàn bộ file uncommitted/untracked từ `g.Root` (bao gồm TDD test files và file từ các Wave trước) vào worktree mới tạo.
   - `SyncWorktreeFilesWithMerge` (`gitclient.go:205-260`) theo dõi `waveOwners map[string]string` trong cùng Wave; nếu 2 sub-task cùng Wave chạm vào cùng 1 file thì thực hiện 3-way merge qua `git merge-file -p` so với bản gốc `HEAD:rel`, trả lỗi tường minh nếu xung đột dòng (`TestSyncWorktreeFilesWithMerge_NonOverlappingAndConflict` PASS).
   - `CoderAgent.Run` tách riêng `coder_output_<SubTaskID>.md` và `changes_<SubTaskID>.md` có khóa `coderArtifactMu`.
   - Đồng thời phát hiện và sửa thêm lỗi vòng lặp vô tận trong `globalBuildFix` (`task_runner.go:1374-1389`) khi `hasQA == false`.
3. **Khắc phục `AF-3` (`internal/agentgate/gate.go:175-231` & `cmd/orchestrator/daemon_cmd.go:89-156`):**
   - Xóa bỏ hoàn toàn mảng `GateStatusBadge` tĩnh trong `daemon_cmd.go`, thay bằng `gk.EvaluateLiveGateBadges(root, *workspaceName, indexedLessons)` đọc trực tiếp trạng thái `GEMINI.md`/`INDEX.md`, file `06_laya_summary_dg1/dg2_*.json`, quét `ScanProductionFiles` trên toàn bộ `.go` files (đã tinh chỉnh tránh false-positive trên `var reviseFn` và `apiKey: mc.EmbedderAPIKey`), và kiểm tra `ValidateCommandSafety`.
   - Bổ sung cờ `--execute-next` gọi `qStore.PopNextPrioritized(ctx) -> qStore.UpdateStage -> qStore.RecordRound` và sửa `qStore.ListRounds(ctx, "")` (`internal/queue/store.go:341-346`) trả về toàn bộ các round khi `taskKey == ""`.
4. **Khắc phục `AF-4` (`internal/orchestrator/task_runner.go:518-576, 838-845`):**
   - Nối `LoadCoreConstitutionPrompt`, `RunLayaGate("dg1"/"dg3")`, `RunMutationGate`, và `RunGovernanceLinter` vào `TaskRunner` với timeout 60s trong `runPythonTool`.
5. **Khắc phục `AF-5` (`internal/bus/types.go:38-45`, `internal/agent/base.go:52-60`, `internal/orchestrator/helpers.go:226-238`, `internal/orchestrator/task_runner.go:1762-1815`):**
   - `AgentUsage` mang theo `Model` và `CostUSD` thực tế; `startUsageAccumulator` và `recordAutopilotRound` cộng dồn chính xác đơn giá Opus (`$15/$75`) vs Sonnet (`$3/$15`), được kiểm chứng trong `TestExecuteSubTasksInOrder_RealCoderAgentAndClaudeRunnerProcessArgsAndCrossWaveSeed` (`2 opus + 2 sonnet = $0.7200`, khớp 100%).
6. **Khắc phục `AF-6` (`internal/tui/model.go:179-180, 713-714, 945-947, 1613-1635`):**
   - Gắn `renderLiveWaveDashboard()` (`Ctrl+W`) vào Bubble Tea `Model` đọc dữ liệu thời gian thực từ `tr.SnapshotWaveProgress()`.
7. **Kết quả `/laya_router` (`DG3`) sau Commit cuối `fa6b410`:**
   - **Summary file:** `06_laya_summary_dg3_20261020T224237.json` | **Raw file:** `06_laya_raw_dg3_20261020T224237.json`
   - **Gate Verdict:** `PASS` (`trusted_rounds: ["r3"]`, `DG3-POSTFIX-AF1-MODEL-ROUTING`: `ACCEPT` `0.8408`; `DG3-POSTFIX-AF2-WORKTREE-SEED` & `DG3-POSTFIX-AF3-LIVE-BADGES`: `FLAG -> RESOLVED` kèm `REF:` trong `04_decisions_feature-modular-agent-orchestrator-2026-10-20.md`).
