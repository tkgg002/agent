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

---

## [x] Task 5 (`S5 -> S3 Re-Plan` — Khắc phục triệt để 6 Sơ hở `AF-1` $\rightarrow$ `AF-6` phát hiện qua Adversarial Re-Audit Commit `54e36e4`, thực thi tại Commit `fa6b410`)
- **[Mô tả & Ánh xạ 1:1 theo từng Sơ hở `AF-1..AF-6`]:**
  1. **Khắc phục `AF-1` (`P0` — [`internal/runner/claude.go:38-78`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/claude.go#L38-L78) & [`internal/orchestrator/wave_parallel_test.go:192-360`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/wave_parallel_test.go#L192-L360)):**
     - Truyền `req.Model` vào `ClaudeRunner.startStreamingProcess` để `--model <req.Model>` (`opus` / `sonnet`) thực sự được truyền xuống CLI `claude`.
     - Bổ sung bài test tích hợp đầu-cuối `TestExecuteSubTasksInOrder_RealCoderAgentAndClaudeRunnerProcessArgsAndCrossWaveSeed` đi xuyên qua `CoderAgent.Run -> ClaudeRunner.Complete` gọi binary script thật ghi nhận `os.Args` (`--model opus` cho `T1`/`T4`, `--model sonnet` cho `T2`/`T3`) và biên dịch Go thật (`go build ./...`).
  2. **Khắc phục `AF-2` (`P0` — [`internal/gitclient/gitclient.go:126-260`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/gitclient/gitclient.go#L126-L260), [`internal/orchestrator/task_runner.go:1059-1343`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go#L1059-L1343) & [`internal/agent/coder.go:193-210`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agent/coder.go#L193-L210)):**
     - Thêm `SeedWorktreeFromRoot` để đồng bộ toàn bộ file uncommitted (TDD tests từ Phase A + file mới của `Wave 0..W-1`) từ `tr.root` sang worktree của `Wave W`.
     - Phát hiện xung đột tập file sửa giữa các sub-task cùng Wave trong `SyncWorktreeFilesWithMerge` (3-way merge qua `git merge-file -p` hoặc báo lỗi tường minh thay vì ghi đè Last-Writer-Wins).
     - Tách riêng file `coder_output_<subTaskID>.md` và `changes_<subTaskID>.md` trong `.orchestrator/` khi chạy song song, bảo vệ bằng `coderArtifactMu`.
  3. **Khắc phục `AF-3` (`P0` — [`cmd/orchestrator/daemon_cmd.go:89-156`](file:///Users/trainguyen/Documents/work/ai-orchestrator/cmd/orchestrator/daemon_cmd.go#L89-L156)):**
     - Bổ sung luồng `--execute-next` gọi `qStore.PopNextPrioritized -> qStore.UpdateStage -> qStore.RecordRound` trong `daemon_cmd.go` và thay thế hoàn toàn mảng `Gates` gán cứng bằng `gk.EvaluateLiveGateBadges` đọc trạng thái kiểm định thật.
  4. **Khắc phục `AF-4` (`P1` — [`internal/agentgate/gate.go:175-355`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/gate.go#L175-L355) & [`internal/orchestrator/task_runner.go:518-576, 838-845`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go#L518-L576)):**
     - Nối dây 4 hàm `LoadCoreConstitutionPrompt`, `RunLayaGate`, `RunMutationGate`, `RunGovernanceLinter` vào `task_runner.go` (`runDecomposeGate`, `implementAndReview`) kèm `context.WithTimeout` 60s.
  5. **Khắc phục `AF-5` (`P1` — [`internal/queue/store.go:335-360`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/queue/store.go#L335-L360), [`internal/orchestrator/task_runner.go:1755-1815`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go#L1755-L1815) & [`internal/orchestrator/helpers.go:226-238`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/helpers.go#L226-L238)):**
     - Nối `tr.recordAutopilotRound` (`qStore.RecordRound`) vào `TaskRunner` và sửa `emitTokenStats` + `startUsageAccumulator` tính đúng đơn giá `opus` (`$15/$75`) vs `sonnet` (`$3/$15`) theo model thực tế của `CoderAgent`.
  6. **Khắc phục `AF-6` (`P2` — [`internal/tui/model.go:179-180, 713-714, 945-947, 1613-1635`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/tui/model.go#L1613-L1635)):**
     - Gắn `RenderWaveDAGDashboard` vào Bubble Tea `internal/tui/model.go` (phím tắt `Ctrl+W`) và hiển thị danh sách `Workers` động từ `tr.SnapshotWaveProgress()`.
- **[Definition of Done (DoD)]:**
  - [x] Toàn bộ 6 sơ hở `AF-1..AF-6` có bài test Red-to-Green không dùng mockup Agent (`TestClaudeRunner_Complete_ForwardsReqModelOverrideToProcessArgs`, `TestWorktreeAdd_SeedsDirtyRootFiles`, `TestSyncWorktreeFilesWithMerge_NonOverlappingAndConflict`, `TestExecuteSubTasksInOrder_RealCoderAgentAndClaudeRunnerProcessArgsAndCrossWaveSeed`), `go vet ./...` & `go test -race ./...` PASS 100%, và đã chạy kiểm chứng runtime thật (`commit fa6b410`).

---

## [x] Task 6 (`S3 Re-Plan` — Xây dựng Giao diện Đồ họa Web UI/UX `internal/webui` + Lệnh `orchestrator ui` / `daemon --web` tái hiện chuẩn 1:1 cả 3 tấm hình mẫu của User)
- **[Mô tả & Phạm vi]:**
  - Xây dựng gói [`internal/webui`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui) (nhúng trực tiếp tài nguyên HTML5/CSS3/SVG/JS qua `embed.FS` vào binary Go `orchestrator`, không phụ thuộc `node_modules` bên ngoài) và mở rộng [`cmd/orchestrator/main.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/cmd/orchestrator/main.go) + [`cmd/orchestrator/daemon_cmd.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/cmd/orchestrator/daemon_cmd.go) với lệnh `orchestrator ui [--addr :8090]` và cờ `orchestrator daemon --web :8090`.
  - **Ánh xạ 1:1 với 3 tấm hình mẫu của User:**
    1. **Màn hình 1 — `Wave 0..N Dependency Graph Board` (Chuẩn Hình 1):**
       - Thanh công cụ lọc trên cùng: nút chuyển `Items | Epics`, dropdown `Anyone ▾`, `Any milestone ▾`, `Any epic ▾`, ô tìm kiếm `Find...`, bộ đếm `N of M`.
       - Thanh trạng thái & chú giải: `● needs you`, `● running`, `● in review`, `● under way`, `● ready`, `◌ blocked`, `● done`, bộ lọc hiển thị `Show | Collapse | Hide`, và đường chỉ báo `— critical`.
       - Bố cục cột ngang `Wave 0`, `Wave 1`, `Wave 2`, `Wave 3`, `Wave 4`, `Wave 5...` kèm tiến độ `<done>/<total>` trên đầu mỗi cột.
       - Thẻ công việc (Task Card) hiển thị mã task (`PM-02`, `SRV-01`, `SRV-02`, `SRV-03`, `SRV-04`, `CLI-02`...), milestone (`M0`, `M1`, `M2`), badge trạng thái (`● READY`, `● NEEDS YOU`, `◌ BLOCKED`, `● RUNNING`, `● DONE`), badge model (`opus` / `sonnet`), và hiệu ứng làm mờ (`dim`) các thẻ ngoài luồng khi người dùng chọn/di chuột vào một chuỗi phụ thuộc.
       - Lớp đồ họa vector `<svg>` tự động tính tọa độ các thẻ và vẽ **đường cong Cubic Bezier (`M x1 y1 C cx1 y1, cx2 y2, x2 y2`)** nối xuyên qua các cột Wave: tô sáng màu xanh cyan (`#38bdf8`) cho chuỗi phụ thuộc đang chọn và màu vàng (`#eab308`) cho đường găng `critical`.
    2. **Màn hình 2 — `AUTOPILOT Pipeline & Cycle/Round Cost ($) Ledger` (Chuẩn Hình 2):**
       - Sơ đồ ngang 6 cụm thẻ bo góc màu xanh lá (`#166534`/`#4ade80`) nối bằng mũi tên `→`:
         `✓ Start (approved)` $\rightarrow$ `[✓ Plan / ✓ Plan review] (⟲ round X of 3)` $\rightarrow$ `✓ You (approved)` $\rightarrow$ `[✓ Implement / ✓ Code review] (⟲ round X of 3)` $\rightarrow$ `[✓ Local QA / ✓ Fix] (⟲ round X of 3)` $\rightarrow$ `✓ Draft PR`.
       - Bảng chi tiết từng chu kỳ & vòng lặp bên dưới:
         - Nhóm `Cycle 1 · first plan` (`you approved`): hiển thị từng dòng `Round 1 · plan`, `Round 1 · review: revise`, `Round 2 · replan`, `Round 2 · review: approve` kèm thông số `blocking · advisory · turns · min · $cost`.
         - Nhóm `Build · after your approval`: hiển thị `Round 1 · implement`, `Round 1 · review: pass`, `QA · specs written`, `QA round 1 · run ×2: retry`, `QA round 2 · run ×2: pass`.
         - Dòng tổng cộng `Total so far` bên trái và `<N> claude runs · <M> min · $<Total>` bên phải, cùng thanh chú giải 6 trạng thái ở đáy (`done`, `running now`, `waiting on you`, `stopped`, `not started`, `not automated yet`).
    3. **Màn hình 3 — `Vibe System Agent Studio` 3 Cột (Chuẩn Hình 3):**
       - Thanh tiêu đề trên cùng: `Vibe System Agent / <tên-dự-án>`, nhánh Git hiện tại, nút `Mở bằng Zed ▾`, các tab chuyển đổi nhanh giữa 3 màn hình (`1. Xây dựng (Multi-Agent Grid)` | `2. Wave 0..N DAG Board` | `3. Autopilot Cost ($) Ledger`), và nút hồng `☼ Giao việc` (mở modal thêm task mới vào `task_queue` hoặc chạy chu trình `Scout -> Sorter -> Execute Next`).
       - **Cột trái (`Dự án & Agents`):** Danh sách dự án và danh sách Agent `A · Yêu cầu`, `B · T11`, `C · Terminal`, `D · T12`, `E · T30`, `F · Claude`, `G · T40` kèm trạng thái hoạt động thật và nút `+ Agent mới`.
       - **Cột giữa (`Quy trình · Mới` 17 Bước):** Hiển thị đầy đủ 17 bước quy trình (`1. Tổng quan` $\rightarrow$ `11. Xây dựng` đang active hồng $\rightarrow$ `17. Vận hành & cải tiến`) ánh xạ trực tiếp với trạng thái của pipeline và cổng `/agent` (`S0 -> S6`, `DG1–DG3`), cùng 4 mục dưới đáy (`Thành phần & CSDL`, `Ảnh chụp`, `Git & phiên bản`, `Chi phí & Tài khoản`).
       - **Cột phải (`Xây dựng` — Lưới đa cửa sổ Agent/Terminal `A..K`):** Thanh lọc runner (`Terminal`, `Claude`, `Codex`, `Gemini`, `OpenCode`, `Cursor`), lưới các ô cửa sổ `C`, `A`, `I`, `G`, `E`, `K`, `B`, `H`, `D`, `F`, `J` hiển thị log/terminal thời gian thực, trạng thái (`Rảnh`, `Chờ T99`, `Đã gửi · T12`, `Bỏ qua quyền`), model (`opus`/`sonnet`), ô nhập lệnh `Nhắn cho agent... Enter để gửi` + `/voice` + nút `Gửi` (gửi lệnh về API backend), và hộp thông báo Git ở góc dưới bên phải.
  - **Backend API & Thời gian thực (`internal/webui/server.go`):**
    - `GET /api/dashboard`: Trả về JSON tổng hợp từ SQLite `.orchestrator/queue.db` (`task_queue`, `autopilot_rounds`), `agent.ComputeSubTaskWaves`, `agentgate.Gatekeeper.EvaluateLiveGateBadges`, cấu hình model `opus`/`sonnet` và trạng thái các Agent `A..K`.
    - `POST /api/tasks`: Nhận task mới từ nút `☼ Giao việc` đẩy vào `Scout`/`task_queue` và tự động chạy `Sorter.TriageAll`.
    - `POST /api/pipeline/trigger`: Kích hoạt chu trình `Scout -> Sorter -> PopNextPrioritized -> Wave Execution` ngay từ giao diện Web.
    - `POST /api/agents/message`: Gửi tin nhắn/lệnh từ ô input của từng cửa sổ Agent `A..K` trong lưới `Vibe System Agent` và ghi nhận phản hồi/vòng lặp mới.
    - `GET /api/stream` (SSE): Đẩy sự kiện thời gian thực xuống trình duyệt mỗi khi trạng thái hàng đợi, Wave DAG hoặc Agent thay đổi.
- **[Context & Bằng chứng `file:line`]:**
  - Kết nối trực tiếp với [`internal/queue/store.go:83-379`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/queue/store.go#L83-L379), [`internal/agent/pm.go:1103-1171`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agent/pm.go#L1103-L1171) (`ComputeSubTaskWaves`), [`internal/agentgate/gate.go:175-231`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/gate.go#L175-L231) (`EvaluateLiveGateBadges`), và [`cmd/orchestrator/main.go:40-87`](file:///Users/trainguyen/Documents/work/ai-orchestrator/cmd/orchestrator/main.go#L40-L87).
  - Bao gồm 4 hạng mục tự chữa lành `UI-AF1..UI-AF4` phát hiện qua `S5 Line-by-Line Diff Critique`: chuẩn hóa `Title` của 17 bước quy trình không lặp số thứ tự (`UI-AF1`), hỗ trợ chế độ gom nhóm `Epics` + đồng bộ dropdown `filter-assignee` (`UI-AF2`), cộng dồn động 100% số lượt chạy/phút/chi phí `$ USD` trên sơ đồ `AUTOPILOT` khi chạy thêm vòng mới (`UI-AF3`), và gắn listener `scroll` vẽ lại đường cong `<svg>` Bezier (`UI-AF4`).
- **[Definition of Done (DoD)]:**
  - [x] `go vet ./...` và `go test -race ./...` (bao gồm `internal/webui/server_test.go` kiểm tra toàn bộ REST/SSE endpoints, kiểm chứng không lặp số bước `1. 1.`, và kiểm chứng chi phí `$ USD` tăng động sau khi trigger pipeline) PASS 100%.
  - [x] Khởi chạy server thật `./bin/orchestrator ui --addr 127.0.0.1:8090`, kiểm chứng bằng HTTP request thật tới `/`, `/api/dashboard`, `/api/tasks`, `/api/pipeline/trigger`, `/api/agents/message` và cập nhật `README_START.md`.



