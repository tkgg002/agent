# Final Adversarial Audit Report: `modular_agent_orchestrator` (S5 Self-Audit)

## 1. Line-by-Line Diff & Task Critique (Phản biện từng dòng update / `git diff` & từng task)

| Task / File:Line | Thay đổi thực tế (`git diff` commit `54e36e4`) | Phản biện (Adversarial Question: Cách phá / Edge case) | Bằng chứng Code / Runtime thật | Kết luận |
| :--- | :--- | :--- | :--- | :--- |
| **Task 1 — `internal/memory/store.go:249-346` & `memory.go:73-75`** | Thêm `IndexAbsoluteFile` và `IndexExternalDir` để index các file Markdown ngoài `.orchestrator/memory` (`agent-lessons:*`, `agent-ws:*`); giữ lại các key chứa `:` khi `Reindex`. | Nếu gặp file `lessons.md` nguyên khối `>256KB` (Rule #21) hoặc chạy `Reindex` nhiều lần liên tiếp thì sao? | `store.go:331-333` chặn cứng `if fi.Size() > 256*1024 { return nil }`; `store.go:274-277` kiểm tra `sha256` để skip file không đổi; `memory_test.go` (`TestIndexExternalDir_AndPreservedAcrossReindex`) PASS. | ✅ **PASS** |
| **Task 1 — `internal/queue/store.go:83-379`** | Khởi tạo SQLite WAL (`busy_timeout=5000`, `journal_mode=WAL`) với 2 bảng `task_queue` & `autopilot_rounds`; cài đặt `EnqueueRaw`, `UpdatePriority`, `PopNextPrioritized`, `RecordRound`, `TotalCostUSD`. | Nếu 2 goroutine cùng gọi `EnqueueRaw` với cùng `external_id` hoặc `PopNextPrioritized` cùng lúc thì có bị duplicate hay race condition không? | Bảng `task_queue` có constraint `external_id TEXT NOT NULL UNIQUE` (`store.go:113`), `Store` khóa `s.mu.Lock()` kết hợp transaction `BeginTx` (`store.go:254-295`). `go test -race ./internal/queue/...` PASS. | ✅ **PASS** |
| **Task 1 — `internal/scout/scout.go:27-192`** | Cài đặt `Worker` Non-LLM (`LLMTokensUsed() == 0`) hỗ trợ `PollBacklogFile`, `PollGitHubIssues`, `PollJiraIssues` kèm `MaskSecrets` (Rule #20 / Hook `H3`). | Nếu mô tả ticket từ Jira/GitHub chứa URI DB có mật khẩu hoặc token thật thì có bị lưu thô vào SQLite không? | `MaskSecrets` (`scout.go:27-43`) quét và che `postgres://***:***@`, `ghp_***`, `Bearer ***` trước khi gọi `EnqueueRaw`. Kiểm chứng tại `scout_test.go:13-62` (`PASS`). | ✅ **PASS** |
| **Task 1 — `internal/sorter/sorter.go:28-115`** | Chấm điểm `ComputeScore` trong đoạn `[0, 100]` từ `Severity`, `Labels`, `SLAHours` và cộng điểm thưởng hotspot khi `memStore.Search` (FTS5 BM25) tìm thấy bài học liên quan. | Nếu `memStore == nil` hoặc điểm cộng vượt quá `100` / thấp hơn `0` thì sao? | `sorter.go:77-96` kiểm tra `if w.memStore != nil`, giới hạn `math.Min(10.0, ...)` và kẹp biên `[0.0, 100.0]`; nếu `score >= 70.0` tự động nâng `complexity = "high"`. Kiểm chứng tại `sorter_test.go` (`PASS`). | ✅ **PASS** |
| **Task 2 — `internal/agent/pm.go:35-46, 643-644, 1103-1171`** | Khóa mặc định `PMAgent` (`cfg.Model = "opus"`), thêm `Complexity` & `Wave` vào `SubTask`, cài đặt `ComputeSubTaskWaves` (Kahn/DFS topological level + phát hiện chu trình). | Nếu LLM trả về `DependsOn` tạo vòng lặp (cyclic dependency `A -> B -> A`) thì `ComputeSubTaskWaves` có bị đệ quy vô tận (stack overflow) không? | `pm.go:1121-1129` dùng mảng trạng thái `visiting[key]` (3-color DFS): khi gặp đỉnh đang `visiting`, ngắt chu trình về `0` ngay lập tức. Kiểm chứng tại `pm_test.go` (`PASS`). | ✅ **PASS** |
| **Task 2 — `internal/config/config.go:145-178` & `internal/agent/coder.go:29-30, 104-118`** | Thêm `ResolveModelForComplexity` (`high -> opus`, `medium/low -> sonnet`) và `CoderAgent.WithRootAndModel(worktreeRoot, modelOverride)` trả về shallow copy cô lập cho từng goroutine. | Nếu 2 goroutine trong cùng Wave chạy song song với 2 model khác nhau (`opus` và `sonnet`) trên 2 worktree khác nhau, chúng có ghi đè `CoderAgent.root` hay `cfg.Model` của nhau không? | `WithRootAndModel` (`coder.go:104-118`) tạo bản sao giá trị (`clone := *a`) với `root` và `cfg.Model` riêng biệt trước khi gọi `implement`, không mutate struct dùng chung. Kiểm chứng với `go test -race ./internal/orchestrator/...` (`PASS`). | ✅ **PASS** |
| **Task 2 — `internal/gitclient/gitclient.go:123-167` & `internal/orchestrator/task_runner.go:894-1205`** | Thêm `WorktreeAdd`, `WorktreeRemove`, `SyncWorktreeFiles` và nâng cấp `executeBeads` + `executeSubTasksInOrder` chạy song song toàn bộ sub-tasks trong cùng Wave bằng `sync.WaitGroup`. | Khi nhiều goroutine cùng cập nhật `tr.state`, `tr.codingStarted`, `tr.agentDurations` và đồng bộ file từ worktree về `tr.root` thì có xảy ra data race không? | Mọi truy cập `state`, `codingStarted`, `agentDurations`, `loadedModel` và `gc.SyncWorktreeFiles` đều được bọc trong `tr.statsMu.Lock()` (`task_runner.go:142-165, 1152-1199, 1441-1522`). Kiểm chứng thực tế bằng `go test -race ./internal/orchestrator/...` (`PASS`). | ✅ **PASS** |
| **Task 3 — `internal/agentgate/gate.go:1-224` & `internal/executil/executil.go:142-151`** | Cài đặt `Gatekeeper` (`LoadCoreConstitutionPrompt`, `ValidateCommandSafety`, `ScanProductionFiles`, `RunLayaGate`, `RunMutationGate`, `RunGovernanceLinter`) và chặn lệnh cấm `H5` trong `executil.Sanitize`. | Nếu Coder tạo file production `.go` chứa closure giả hoặc lộ credential thô thì `TaskRunner` xử lý thế nào? | `implementAndReview` (`task_runner.go:473-485`) gọi `tr.gatekeeper.ScanProductionFiles(tr.root, files)` và nối lỗi vào `secFeedback`, buộc PM/Coder bước vào vòng `fixRounds` tiếp theo. Kiểm chứng tại `gate_test.go` (`PASS`). | ✅ **PASS** |
| **Task 4 — `internal/tui/wave_dag_view.go:1-155` & `cmd/orchestrator/daemon_cmd.go:1-168`** | Hiển thị bảng điều khiển 3 tầng (Wave 0..N DAG, Multi-Worker Grid A..K, Autopilot Cost `$` Ledger & `/agent` Gate Badges) và lệnh CLI `orchestrator daemon --once`. | Lệnh `go run ./cmd/orchestrator daemon --once` có chạy thực tế từ đầu đến cuối với SQLite WAL + FTS5 BM25 + `/agent` lessons không? | Đã chạy thực tế `go run ./cmd/orchestrator daemon --once` (và kèm `--backlog`), index 5 file chủ đề bài học `/agent` vào FTS5, chấm điểm 2 task mẫu (`GH-201` -> `opus`, `JIRA-202` -> `sonnet`) và in bảng Wave DAG (exit code `0`). | ✅ **PASS** |

---

## 2. Anti-Smoke-Speculation & Real Runtime Trace Verification (Chống suy diễn Smoke Test, Mockup, Hallucination & Trace Runtime thật)

- **Kiểm tra Production Mock Pollution (`Hook H4b`):**
  - Đã quét toàn bộ các file production (`.go` không phải `*_test.go`) bằng `Gatekeeper.ScanProductionFiles` và `go vet ./...`: xác nhận **0 hàm mock/closure giả** hay `sqlmock`/`fakemodel` trong mã nguồn production.
- **Bằng chứng Test & Runtime thật (`Gate G3`):**
  - Toàn bộ 18 package có test trong `/Users/trainguyen/Documents/work/ai-orchestrator` đều đạt `PASS` dưới cờ phát hiện tranh chấp bộ nhớ `go test -race ./...`.
  - Đã thực thi tiến trình runtime thật theo đúng khai báo `runtime_check` trong `/Users/trainguyen/Documents/work/ai-orchestrator/.agent-project.json`:
    - Lệnh 1: `go run ./cmd/orchestrator daemon --once` -> khởi tạo `.orchestrator/memory.db` (FTS5 BM25) + `.orchestrator/queue.db` (SQLite WAL), nạp `5` file bài học từ `/Users/trainguyen/Documents/work/agent/memory/global/lessons/topics/*.md`, kiểm tra các cổng `[S0-Constitution: PASS] [DG1+DG2: PASS] [H4b-ZeroMock: PASS] [H5-NoGitPush: PASS] [FTS5-BM25: PASS(5 new)]`.
    - Lệnh 2: `go run ./cmd/orchestrator daemon --once --backlog /tmp/sample_backlog.json` -> `Scout` đọc 2 ticket (`GH-201`, `JIRA-202`) với `Scout LLM tokens=0`, `Sorter` chấm điểm và đẩy vào `PRIORITIZED=2`, phân rã vào `Wave 0 [PARALLEL x2 WORKTREES]` với `GH-201 (complexity=high -> model=opus)` và `JIRA-202 (complexity=low -> model=sonnet)`.
- **Điểm cần phối hợp User trace runtime value trên môi trường thật:**
  - Khi kết nối với tài khoản GitHub/Jira thật hoặc chạy CLI `claude` (`Opus` / `Sonnet`) thực tế trên repo sản phẩm của User, User chỉ cần cung cấp `GITHUB_TOKEN` / `JIRA_API_TOKEN` qua biến môi trường và cấu hình `.orchestrator/project.yaml`. Không có bất kỳ kết quả nào bị giả lập hay hallucinate.

---

## 3. Concurrency, Timeout/Heartbeat & Data Continuity Audit (Kiểm tra Đa luồng, Multi-Worker, Timeout & Liên tục dữ liệu)

- **Concurrency / Multi-worker / Parallel Wave Execution (`WorkerCount > 1`):**
  - Trong `internal/orchestrator/task_runner.go:935-975` (`executeBeads`) và `1065-1114` (`executeSubTasksInOrder`), các sub-task cùng `Wave` được thực thi song song qua `sync.WaitGroup`.
  - Mỗi worker khi `len(batch) > 1` được cấp một thư mục `git worktree` độc lập (`orch-wt-<runID>-<key>-<nano>`) thông qua `gc.WorktreeAdd(wtDir, "")` (`gitclient.go:124-136`) và bản sao `CoderAgent.WithRootAndModel` (`coder.go:104-118`).
  - Sau khi goroutine hoàn thành, `gc.SyncWorktreeFiles` được khóa đồng bộ bằng `tr.statsMu.Lock()` (`task_runner.go:1180-1185`) để gom file an toàn về repo gốc trước khi `defer gc.WorktreeRemove(wtDir)` dọn sạch worktree.
  - Đã kiểm chứng bằng test đồng thời `TestExecuteSubTasksInOrder_ParallelWavesAndDynamicModelRouting` (`internal/orchestrator/wave_parallel_test.go:76-189`) dùng channel barrier xác nhận 2 tác vụ `Wave 0` thực sự chạy song song trước khi mở khóa `Wave 1`, chạy sạch dưới `go test -race`.
- **Timeout / Busy-Wait / Resource Cleanup:**
  - `internal/scout/scout.go:52-55` áp đặt `http.Client{Timeout: 15 * time.Second}` và `io.LimitReader(resp.Body, 5<<20)` để chống treo kết nối hoặc tràn bộ nhớ khi gọi API ngoài.
  - `internal/queue/store.go:88` và `internal/memory/store.go:58` cấu hình `_pragma=busy_timeout=5000&_pragma=journal_mode=WAL` giúp chống lỗi `SQLITE_BUSY` khi nhiều worker đọc/ghi trạng thái hàng đợi và bộ nhớ FTS5.
- **Data Continuity / Idempotency / Watermark:**
  - `queue.Store.EnqueueRaw` (`store.go:156-201`) đảm bảo tính idempotent tuyệt đối theo `external_id UNIQUE`: nếu `Scout` quét lại cùng một ticket ở chu kỳ sau, hàm trả về `(existingID, false, nil)` mà không tạo bản ghi trùng hoặc ghi đè `stage` đang chạy.
  - `memory.Store.IndexAbsoluteFile` (`store.go:249-313`) đối chiếu `sha256` của nội dung file để chỉ đánh chỉ mục lại khi nội dung thực sự thay đổi, và `memory.Reindex` (`memory.go:73-75`) bảo toàn 100% các mục chỉ mục ngoại vi có tiền tố `:` (`agent-lessons:*`, `agent-ws:*`).

---

## 4. Anti-Speculation & Honesty Check (Kiểm tra suy diễn & báo cáo láo)

- Mọi khẳng định trong báo cáo này và trong `06_validation_modular_agent_orchestrator.md` đều gắn trực tiếp với `file:line` thực tế trong `/Users/trainguyen/Documents/work/ai-orchestrator` và output thật từ terminal.
- Không sử dụng bất kỳ mock closure nào trong code production; giữ nguyên tính tương thích ngược với toàn bộ test suite gốc của `Open-Makers/ai-coding-agents-orchestrator` (`TestEmitSummary_IncludesTimeAndStats`, `TestPickInProgress`, v.v. đều `PASS`).

---

## 5. Self-Improvement Loop (Vòng lặp Phản tỉnh & Khắc phục — Lessons Audit)

- Đã đối chiếu với `memory/global/lessons/INDEX.md` (`253` bài học):
  - Tuân thủ bài học `L001 [2026-10-20] Reading Only README Without Cloning Full Source Code`: đã clone toàn bộ repo Go gốc, tận dụng 80% các package có sẵn (`internal/runner`, `internal/agent`, `internal/beads`, `internal/memory`, `internal/bus`, `internal/tui`) và chỉ mở rộng 20% Go Delta.
  - Tuân thủ bài học `L002 [2026-10-20] Missing Macro Runtime & Integration Topology`: đã hiện thực hóa đầy đủ lệnh `orchestrator daemon --once` và cơ chế tiêm hiến pháp `/agent` (`internal/agentgate/gate.go`).
  - Trong chặng S4–S5 này không phát sinh vi phạm quy trình mới.

---

## 6. Kết quả QC & Nhận xét từ `/laya_router` (`DG3`)

- **Summary file:** `06_laya_summary_dg3_20261020T160259.json`
- **Gate Verdict:** `PASS` (`trusted_rounds: ["r3"]`)
- **Chi tiết các Claim đã kiểm định:**
  - `DG3-R1-QUEUE-FTS5` (`target`): `ACCEPT` (`r3: YES, conf=0.8340`) — Xác nhận `queue.Store.Open` khởi tạo SQLite WAL (`busy_timeout=5000`) và `memory.Store.IndexExternalDir` đánh chỉ mục FTS5 có chặn file `>256KB`.
  - `DG3-R2-SCOUT-SORTER` (`target`): `ACCEPT` (`r3: YES, conf=0.6468`) — Xác nhận `scout.Worker` dùng `0` LLM token kèm `MaskSecrets` (`H3`), và `sorter.Worker` tính điểm ưu tiên kết hợp FTS5 BM25.
  - `DG3-R3-R4-WAVE-MODEL` (`target`): `FLAG -> RESOLVED` (`r3: YES, conf=0.4964`, giải trình kèm `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go:1152-1196` trong `04_decisions_feature-modular-agent-orchestrator-2026-10-20.md`).
  - `DG3-R5-AGENTGATE` (`target`): `FLAG -> RESOLVED` (giải trình kèm `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go:473-485` trong `04_decisions_feature-modular-agent-orchestrator-2026-10-20.md`).
  - Controls: `DG3-CTRL-TRUE-1` (`control_true`), `DG3-CTRL-FALSE-1` (`control_false`), `DG3-CTRL-FALSE-2` (`control_false`) đều vượt qua kiểm định đối chứng.
