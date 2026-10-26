# 11 — Implementation & Delivery Report: `modular_agent_orchestrator`

## 1. Tóm tắt Kết quả Thực thi (Executive Summary)

Đã hoàn tất việc kế thừa và mở rộng codebase Go của `Open-Makers/ai-coding-agents-orchestrator` tại `/Users/trainguyen/Documents/work/ai-orchestrator` (nhánh `feat/modular-queue-agent-kernel`, commit `54e36e4`), dung hợp kiến trúc **"Vỏ Queue-Driven (Outer Orchestration Plane)"** với **"Lõi `/agent` Kernel (Inner Governance Kernel)"**:

1. **Tận dụng 80% Engine Go sẵn có:** Giữ nguyên và tích hợp trực tiếp với `internal/runner` (`claude.go`, `codex.go`, `opencode.go`, `ollama.go`), `internal/beads`, `internal/memory` (SQLite FTS5 BM25), `internal/bus`, `internal/multiplexer/tmux.go` và `internal/tui`.
2. **Hiện thực hóa 20% Go Delta (4 Tasks trong `08_tasks_modular_agent_orchestrator.md`):**
   - **Task 1 (`internal/queue`, `internal/scout`, `internal/sorter`, `internal/memory`):** Bổ sung hàng đợi bền vững SQLite WAL (`task_queue`, `autopilot_rounds`), `Scout Worker` Non-LLM (`$0` token, có `MaskSecrets` chuẩn Hook `H3`), `Sorting Worker` chấm điểm `0..100` có cộng điểm rủi ro từ chỉ mục FTS5 BM25 của `/agent` (`IndexExternalDir`).
   - **Task 2 (`internal/agent`, `internal/config`, `internal/gitclient`, `internal/orchestrator`):** Khóa mặc định `PMAgent` vào `opus`, tính toán đồ thị phụ thuộc `Wave 0..N` (`ComputeSubTaskWaves`), định tuyến model động (`ResolveModelForComplexity`: `high -> opus`, `medium/low -> sonnet`), và thực thi song song các sub-task cùng `Wave` trên các `git worktree` cô lập (`WorktreeAdd`, `SyncWorktreeFiles`, `WorktreeRemove`).
   - **Task 3 (`internal/agentgate`, `internal/executil`, `internal/runner/budget.go`):** Xây dựng trạm kiểm soát `/agent` (`Gatekeeper`: nạp hiến pháp `GEMINI.md` + `lessons/INDEX.md`, chặn lệnh cấm `H5`, quét file production chặn mock closure `H4b` & lộ secret `H3`, tích hợp `laya_gate.py`, `mutation_gate.py`, `verify_governance.py`) và tính toán chi phí USD (`CalculateCostUSD`).
   - **Task 4 (`internal/tui/wave_dag_view.go`, `cmd/orchestrator/daemon_cmd.go`):** Bổ sung giao diện 3 tầng (`Wave 0..N DAG`, `Multi-Worker Grid A..K`, `Autopilot Cost ($) Ledger & Gate Badges`) và lệnh `orchestrator daemon [--once] [--backlog <file>]`.

---

## 2. Thống kê Thay đổi Mã nguồn (`git diff --stat 5e871e5..54e36e4`)

- **Repository:** `/Users/trainguyen/Documents/work/ai-orchestrator`
- **Branch:** `feat/modular-queue-agent-kernel`
- **Restore-point Baseline Commit:** `5e871e5` (`chore(restore-point): pre-modular_agent_orchestrator baseline`)
- **Implementation Commit:** `54e36e4` (`feat(orchestrator): modular queue-driven waves, non-llm scout, fts5 sorter, dynamic model routing & /agent gatekeeper`)
- **Tổng số file thay đổi:** `24 files changed, 2452 insertions(+), 106 deletions(-)`

| File | Loại thay đổi | Số dòng (`+/-`) | Mô tả |
| :--- | :---: | :---: | :--- |
| `.agent-project.json` | New | `+13` | Khai báo manifest dự án cho `/agent` (`checks`, `test`, `runtime_check`) |
| `.gitignore` | Modified | `+4 / -7` | Bổ sung ignore cho `.orchestrator/` runtime và các file `*.db-wal`, `*.db-shm` |
| `cmd/orchestrator/main.go` | Modified | `+3` | Nối lệnh `daemon` vào CLI dispatcher |
| `cmd/orchestrator/daemon_cmd.go` | New | `+168` | Chu trình tự động `FTS5 Index -> Scout ($0) -> Sorter -> Wave DAG Dashboard` |
| `internal/agent/pm.go` | Modified | `+79 / -1` | Mặc định `opus` cho Planner, thêm `Complexity` & `Wave`, hàm `ComputeSubTaskWaves` |
| `internal/agent/pm_test.go` | Modified | `+62` | Unit test kiểm chứng `ComputeSubTaskWaves`, chống chu trình và mặc định `opus` |
| `internal/agent/coder.go` | Modified | `+26 / -2` | Thêm `ModelOverride`, `WorktreeRoot` và `WithRootAndModel` thread-safe |
| `internal/agentgate/gate.go` | New | `+224` | Cầu nối cưỡng chế `/agent` Kernel (`H3`, `H4b`, `H5`, `DG1–DG3`, `3-Mutant`, `VerifyGov`) |
| `internal/agentgate/gate_test.go` | New | `+89` | Unit test kiểm chứng `Gatekeeper` |
| `internal/config/config.go` | Modified | `+37` | Thêm `HighComplexityModel`, `StandardModel` và `ResolveModelForComplexity` |
| `internal/executil/executil.go` | Modified | `+10` | Cưỡng chế Hook `H5` trong `Sanitize` |
| `internal/executil/executil_test.go` | Modified | `+23` | Unit test kiểm chứng chặn lệnh cấm `H5` |
| `internal/gitclient/gitclient.go` | Modified | `+46` | Thêm `WorktreeAdd`, `WorktreeRemove`, `SyncWorktreeFiles` |
| `internal/gitclient/gitclient_test.go` | Modified | `+44` | Test tạo, đồng bộ file và xóa `git worktree` thật |
| `internal/memory/memory.go` | Modified | `+3` | Bảo toàn các khóa ngoại vi chứa `:` (`agent-lessons:*`) khi `Reindex` |
| `internal/memory/store.go` | Modified | `+102` | Thêm `IndexAbsoluteFile` và `IndexExternalDir` (chặn file `>256KB` theo Rule #21) |
| `internal/memory/memory_test.go` | Modified | `+45` | Test đánh chỉ mục thư mục ngoài và bảo toàn qua `Reindex` |
| `internal/orchestrator/helpers.go` | Modified | `+15 / -3` | Hiển thị chi phí `$ USD` theo từng role và tổng cộng trong `emitTokenStats` |
| `internal/orchestrator/task_runner.go` | Modified | `+191 / -93` | Thực thi song song theo `Wave 0..N` trên `git worktree`, định tuyến model động, gắn `Gatekeeper` |
| `internal/orchestrator/wave_parallel_test.go` | New | `+189` | Test chạy song song `Wave 0` (`opus` + `sonnet`) trên `git worktree` trước khi mở khóa `Wave 1` |
| `internal/queue/store.go` | New | `+380` | Hàng đợi SQLite WAL `task_queue` & sổ cái `autopilot_rounds` |
| `internal/queue/store_test.go` | New | `+97` | Unit test kiểm chứng vòng đời hàng đợi và sổ cái chi phí |
| `internal/runner/budget.go` | Modified | `+31` | Thêm `CalculateCostUSD` theo bảng giá token của `opus`, `sonnet`, `haiku` |
| `internal/scout/scout.go` | New | `+192` | `Scout Worker` Non-LLM (`$0` token) + `MaskSecrets` (`H3`) |
| `internal/scout/scout_test.go` | New | `+88` | Test `MaskSecrets`, `PollBacklogFile`, `PollGitHubIssues` và `LLMTokensUsed() == 0` |
| `internal/sorter/sorter.go` | New | `+116` | `Sorting Worker` chấm điểm `0..100` + tăng điểm theo FTS5 BM25 hotspot |
| `internal/sorter/sorter_test.go` | New | `+95` | Test xếp hạng ưu tiên và tích hợp FTS5 BM25 |
| `internal/tui/wave_dag_view.go` | New | `+155` | Giao diện 3 tầng `RenderWaveDAGDashboard` |
| `internal/tui/wave_dag_view_test.go` | New | `+62` | Test kết xuất `RenderWaveDAGDashboard` |

---

## 3. Kết quả Thực thi Task 5 (`AF-1` $\rightarrow$ `AF-6` Remediation — Commit `fa6b410`)

Sau khi nhận lệnh `APPROVE` của User cho kế hoạch khắc phục `S5 -> S3 Re-Plan`, toàn bộ **6 phát hiện phản biện (`AF-1` đến `AF-6`)** đã được sửa triệt để trong mã nguồn Go (commit `fa6b410`, `16 files changed, 796 insertions(+), 56 deletions(-)`), kiểm thử Red-to-Green không dùng mockup Agent và vượt qua hội đồng `/laya_router` (`DG3 PASS` tại `06_laya_summary_dg3_20261020T224237.json`):

1. **`AF-1` (`internal/runner/claude.go:38-78`, `claude_test.go:92-150`, `wave_parallel_test.go:192-360`):** Truyền `req.Model` xuống `startStreamingProcess` và ưu tiên `reqModel` trước `r.Model` khi nạp cờ `--model <effectiveModel>` cho tiến trình `claude`. Kiểm chứng bằng `agent.NewCoder` + `runner.NewClaude` thật ghi lại `os.Args` thực tế.
2. **`AF-2` (`internal/gitclient/gitclient.go:126-260`, `internal/agent/coder.go:193-210`, `internal/orchestrator/task_runner.go:1185-1197, 1374-1389`):** Bổ sung `SeedWorktreeFromRoot` (tự động chép các file mới/đang sửa từ `r.root` sang worktree của Wave kế tiếp), `SyncWorktreeFilesWithMerge` (hợp nhất 3 chiều bằng `git merge-file -p` và báo lỗi khi trùng dòng thay vì ghi đè âm thầm), cô lập `coder_output_<SubTaskID>.md` / `changes_<SubTaskID>.md` bằng `coderArtifactMu`, và sửa vòng lặp vô hạn trong `globalBuildFix` khi `hasQA == false`.
3. **`AF-3` (`internal/agentgate/gate.go:175-231`, `cmd/orchestrator/daemon_cmd.go:89-156`, `internal/queue/store.go:335-360`):** Thay thế mảng tĩnh `GateBadges` bằng `gk.EvaluateLiveGateBadges(workspaceName, indexedCount)` kiểm tra trực tiếp file vật lý & `ScanProductionFiles`, đồng thời bổ sung cờ `--execute-next` rút tự động task ưu tiên cao nhất từ `task_queue`.
4. **`AF-4` (`internal/orchestrator/task_runner.go:518-576, 838-845`, `internal/agentgate/gate.go:240-265`):** Nối trực tiếp `LoadCoreConstitutionPrompt`, `RunLayaGate`, `RunMutationGate`, `RunGovernanceLinter` vào luồng chạy của `TaskRunner` kèm `context.WithTimeout` 60s trong `runPythonTool`.
5. **`AF-5` (`internal/bus/types.go:38-45`, `internal/agent/base.go:52-60`, `internal/orchestrator/helpers.go:226-238`, `internal/orchestrator/task_runner.go:1762-1815`):** Gắn `Model` thực tế vào `TokenUpdatePayload`, tính giá riêng cho từng model (`opus` `$15/$75` vs `sonnet` `$3/$15`) trong `emitTokenStats`, và ghi nhận `AutopilotRound` vào bảng SQLite `autopilot_rounds` qua `recordAutopilotRound`.
6. **`AF-6` (`internal/tui/model.go:179-180, 713-714, 945-947, 1613-1635`):** Tích hợp phím tắt `Ctrl+W` và hàm `renderLiveWaveDashboard()` để hiển thị trực tiếp `RenderWaveDAGDashboard` ngay trong giao diện TUI chính.
7. **Tài liệu Hướng dẫn Khởi động (`README_START.md`):** Đã biên soạn hướng dẫn cài đặt, cấu hình định tuyến model động (`opus`/`sonnet`) và vận hành 5 chế độ (`ui`, `daemon`, `TUI`, `task`, `memory`) tại `/Users/trainguyen/Documents/work/ai-orchestrator/README_START.md` kèm binary `./bin/orchestrator`.

---

## 4. Kết quả Thực thi Task 6 & Tự Chữa Lành `UI-AF1..UI-AF4` (`3-Screen Graphical Web UI` — Commits `2ed2e89` & `d9c42aa`)

- **Commits:**
  - `2ed2e89` (`feat(webui): add 3-screen graphical web studio matching wave dag, autopilot ledger, and vibe studio screenshots` — `10 files changed, 3918 insertions(+), 28 deletions(-)`)
  - `d9c42aa` (`fix(webui): resolve UI-AF1..UI-AF4 step title prefix, epics toggle, and dynamic autopilot totals` — `5 files changed, 84 insertions(+), 36 deletions(-)`)

| File | Loại thay đổi | Số dòng (`+/-`) | Mô tả |
| :--- | :---: | :---: | :--- |
| `internal/webui/server.go` | New | `+999` | HTTP/SSE Server phục vụ `/`, `/api/dashboard`, `/api/tasks`, `/api/pipeline/trigger`, `/api/agents/message`, `/api/stream` |
| `internal/webui/seed_data.go` | New | `+288` | Dữ liệu đồ thị phụ thuộc `Wave 0..5` (`PM-02 -> SRV-01 -> SRV-02 -> SRV-03 -> SRV-04 -> CLI-02`), 9 vòng lặp Autopilot và 11 cửa sổ Vibe Studio `A..K` |
| `internal/webui/static/index.html` | New | `+315` | Giao diện HTML5 của cả 3 màn hình đồ họa + modal `☼ Giao việc` |
| `internal/webui/static/app.css` | New | `+1244` | Hệ thống CSS Dark Cyber (`#0d121b`), thẻ xanh lá Autopilot, và bố cục 3 cột Vibe System Agent |
| `internal/webui/static/app.js` | New | `+600` | Bộ vẽ đường cong `<svg>` Cubic Bezier động (`drawWaveBezierEdges`), lọc chuỗi phụ thuộc, đồng bộ SSE thời gian thực |
| `internal/webui/server_test.go` | New | `+282` | Bộ test tự động kiểm chứng cả 3 màn hình, tạo task mới, kích hoạt Wave kế tiếp, chặn `git push` (`H5`) và luồng SSE |
| `internal/queue/store.go` | Modified | `+45` | Bổ sung `UpdateWaveAndStage` và `ListAll` |
| `cmd/orchestrator/daemon_cmd.go` | Modified | `+142 / -26` | Bổ sung lệnh `uiCmd` (`orchestrator ui --addr :8090`), cờ `--web :8090` cho `daemonCmd`, và đồng bộ `DependsOn` từ nhãn `depends:` |
| `cmd/orchestrator/main.go` | Modified | `+5 / -1` | Đăng ký lệnh `orchestrator ui` / `orchestrator web` |
| `README_START.md` | Modified | `+26 / -1` | Bổ sung hướng dẫn khởi chạy Chế độ 0 (`./bin/orchestrator ui --addr 127.0.0.1:8090`) |

---

## 5. Kết quả Thực thi Task 7 (`Zero-Mockup & Unified Cross-Screen Entity Graph` — Commit `ae7ffdc`)

- **Commit:** `ae7ffdc` (`feat(webui): purge 100% mockup data and wire cross-screen entity graph linkage (Task 7)` — `10 files changed, 2209 insertions(+), 796 deletions(-)`)
- **Biên lai Kiểm định (`06_evidence/`):** `EV-20261021T114444-7c71` (`unit`, `exit=0`), `EV-20261021T114502-b220` (`integration`, `exit=0`), `EV-20261021T114549-0bb7` (`runtime`, `exit=0`).
- **Hội đồng `/laya_router` (`DG3`):** `06_laya_summary_dg3_20261021T114858.json` (`gate: PASS`).
- **Tóm tắt 2 Trụ cột Hoàn tất:**
  1. **Xóa bỏ 100% dữ liệu mockup & kẹp sàn giả (`MF-01..MF-08`):** Dọn sạch DB qua `PurgeMockSeedData`, đồng bộ trực tiếp `TASK-1..TASK-7` từ `08_tasks_*.md` và 27 vòng kiểm định thật từ `06_laya_summary_*.json`, suy diễn động 17 bước quy trình từ file thật trên đĩa, thực thi lệnh shell thật trên ô `Slot C` (`exec.CommandContext` + `ValidateCommandSafety` + `MaskSecrets`) và truy vấn FTS5 BM25 thật trên các ô Agent.
  2. **Đồ thị Thực thể Thống nhất & Liên kết Điều hướng 2 Chiều Xuyên 3 Màn hình (`MF-09`):** Hợp nhất khóa ngoại xuyên suốt `Project -> WorkflowStep 1..17 -> TaskKey TASK-1..7 -> AgentSlot A..K -> AutopilotRound` kèm thanh điều hướng `#cross-screen-context-bar` cho phép lọc và nhảy tức thì qua lại giữa Màn hình 1 (`Wave 0..N`), Màn hình 2 (`Autopilot`) và Màn hình 3 (`Vibe System Agent Studio`).

---

## 6. Kết quả Thực thi Task 8 (`Unified 100vh Single-App IDE Shell + 4-Layer Per-Task Isolation + Live Autopilot Engine + Antigravity Runner` — Commit `2d9a0a2`)

- **Commit:** `2d9a0a2` (`feat(webui,runner): unify 100vh IDE shell, 4-layer per-task isolation, live autopilot engine, and Antigravity runner` — `9 files changed, 3257 insertions(+), 1609 deletions(-)`)
- **Biên lai Kiểm định (`06_evidence/`):** `EV-20261021T144834-d8fa` (`unit`, `exit=0`), `EV-20261021T144849-9628` (`integration`, `exit=0`), `EV-20261021T144945-7d7f` (`runtime`, `exit=0`), `EV-20261021T145011-5f4a` (`runtime`, `exit=0`), `EV-20261021T145047-5245` (`runtime`, `exit=0`).
- **Hội đồng `/laya_router` (`DG3`):** `06_laya_summary_dg3_20261021T145755.json` (`gate: PASS`).
- **Tóm tắt 4 Hạng mục Cốt lõi Hoàn tất:**
  1. **Hợp nhất 1 Khung Ứng Dụng IDE `100vh` Duy Nhất (`SHELL-01`, `UX-01..UX-04`):** Xóa bỏ hoàn toàn thanh `.vibe-topbar` trùng lặp (`Vibe System Agent / ai-orchestrator`), gom về 1 Top Header + 1 Cột trái Quản lý Hàng đợi Task & 17 Bước quy trình + 3 Tab góc nhìn (`1. ⚡ Autopilot & Task Runner`, `2. ⛁ Wave 0..N Dependency DAG`, `3. ⌨ Multi-Agent Terminal Grid A..K`), bảng Wave ngang `grid-auto-flow: column` không rớt dòng và lưới 4 dải dọc thu gọn (`C, A, I, G`) + Bento (`E, K, B, H, D, F`) + thanh trạng thái `J`.
  2. **Bộ Máy Thực Thi Tự Động `Per-Task Multi-Stage Autopilot Core Engine` (`CORE-01..02`):** Hiện thực `POST /api/tasks/action` (`select`, `plan`, `approve`, `implement`, `qa`, `autopilot`) và phạm vi hóa sơ đồ 6 hộp Autopilot theo đúng Task đang chọn (`active_task_key`).
  3. **Cô lập Vật lý 4 Lớp cho từng Task (`ISO-01..02`):** Mỗi Task tự động được cấp nhánh Git riêng `task/<safeKey>`, thư mục worktree riêng `.orchestrator/worktrees/<safeKey>`, thư mục hồ sơ riêng `.orchestrator/tasks/<safeKey>/{01_requirements.md,09_tasks_solution.md,audit_report.md,terminal.log,diff.patch}`, thực thi lệnh trong `cmd.Dir = WorktreePath`, và sổ cái `autopilot_rounds` riêng.
  4. **Tích hợp `AntigravityRunner` & Áp dụng thật `/agent` Governance Kernel (`AGY-01`, `KERNEL-01`):** Bổ sung [`internal/runner/antigravity.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity.go) kết nối 3 tầng (`agy` CLI / `google.antigravity` Python SDK / `gemini` CLI / `claude` CLI) và nối `Gatekeeper` (`LoadCoreConstitutionPrompt`, `ValidateCommandSafety H5`, `ScanProductionFiles H4b+H3`, `EvaluateLiveGateBadges`) vào từng chặng Autopilot.

---

## 7. Kết quả Thực thi Task 9 (`2-Phase Gate-Stop Autopilot WF-01 + Target Repo & CLI/Model Setup + Sub-Task Wave Builder REPO-SUB-01 + 5-Tab Step Report & Code Demo Inspector REPORT-01 + 100% Strict Right-Window Scoping SCOPE-01 + Live /agent Kernel Active Inspector AGENT-VIS-01` — Commit `617e961`)

- **Commit:** `617e961` (`feat(webui): enforce 2-phase gate-stop workflow, repo/CLI/subtask builder, 5-tab step report inspector, strict right-window scoping, and live /agent kernel binding` — `7 files changed, 3261 insertions(+), 341 deletions(-)`)
- **Biên lai Kiểm định (`06_evidence/`):** `EV-20261021T170007-5ee9` (`unit`, `exit=0`), `EV-20261021T170010-bdb0` (`integration`, `exit=0`), `EV-20261021T170216-45ac` (`runtime`, `exit=0`), `EV-20261021T171044-fea1` (`runtime` `go run ./cmd/orchestrator daemon --once`, `exit=0`).
- **Hội đồng `/laya_router` (`DG3`):** `06_laya_summary_dg3_20261021T170437.json` (`gate: PASS`).
- **Tóm tắt 5 Hạng mục Hoàn tất:**
  1. **`WF-01` (2-Phase Gate-Stop Autopilot — Chấm dứt chạy lướt Smoke-Test qua `APPROVE`):** Khi Task ở `READY`/`PLANNING` và bấm `▶ Run / Autopilot`, hệ thống chỉ chạy **Pha 1 (`S0 -> S1 -> S2 Quét sâu TargetRepo & FTS5 Lesson Match -> S3 Giải pháp + Code Demo + Chia Sub-Tasks -> DG1/DG2`)** rồi **DỪNG CỨNG TẠI `NEEDS_YOU`** chờ User đọc Report & Code Demo và bấm **`✓ APPROVE & Chạy S4–S5`** mới mở khóa **Pha 2 (`S4 Worktree Implementation -> S5 QA & DG3 -> DONE`)**. Đã khôi phục Task `"tích hợp Master MongoDB Destination"` về trạng thái `NEEDS_YOU` trên repo `/Users/trainguyen/Documents/work/data-hub/centralized-data-service` kèm 4 Sub-Tasks và Code Demo `MasterMongoDestination.UpsertMasterBatch`.
  2. **`REPO-SUB-01` (Chọn `Target Repository` + `⚙ Setup CLI & Model` + Tạo/Chia nhỏ `Sub-Tasks`):** Tự động quét 24 repo trong `/Users/trainguyen/Documents/work` (`data-hub/centralized-data-service`, `data-hub/cdc-cms-service`, `data-hub/cdc-cms-web`, `ai-orchestrator`, `agent`...); thêm modal `⚙ Setup CLI & Model` (`POST /api/settings/cli`) cấu hình CLI (`antigravity`, `gemini`, `claude`, `codex`, `opencode`) và Model theo từng Role (`Planner S1-S3`, `Coder High S4`, `Coder Std S4`, `QA S5`, `Security`); thêm cơ chế tạo & tự động chia nhỏ Sub-Tasks (`POST /api/tasks/subtasks`).
  3. **`REPORT-01` (Khung `5-Tab Step Report, Solution & Code Demo Inspector`):** Hiển thị trực tiếp 5 tab báo cáo (`📋 S1 Requirements & Sub-Tasks`, `🔍 S2 Repo Investigation (file:line)`, `💡 S3 Giải Pháp & Code Demo`, `🔧 S4 Worktree & Git Diff`, `🛡️ S5 Audit Report & DG3`) ngay trên cửa sổ phải, tự động chuyển tab tương ứng khi bấm vào bất kỳ bước nào trong 17 bước `S0..S6` hoặc 6 hộp Autopilot.
  4. **`SCOPE-01` (Đồng bộ Tuyệt đối 100% Cửa Sổ Bên Phải Theo Task Đang Chọn):** Khi chọn `TASK-1`, `TASK-2` hoặc `"tích hợp Master MongoDB Destination"`, toàn bộ cửa sổ bên phải (6 hộp Autopilot, bảng Rounds, đồ thị Sub-Tasks Wave DAG, 17 bước quy trình, 5 tab Report/Code Demo, thẻ `/agent` Kernel và 11 ô Terminal `A..K`) đều lọc 100% theo đúng Task đó.
  5. **`AGENT-VIS-01` (Hiển thị Trực quan Bộ Não `/agent` Kernel & `268` Bài học FTS5):** Đọc trực tiếp tổng số `268` bài học từ `/Users/trainguyen/Documents/work/agent/memory/global/lessons/INDEX.md` và hiển thị thẻ **`🛡️ /agent Kernel Active`** (`Active Role`, `Active Skills`, `Hooks H1..H7`, và danh sách bài học `Lxxx` khớp với Task đang chọn).

---

## 9. Báo cáo Hoàn thành Task 10 (`AGY-ARGON-01`, `DECOMP-DOMAIN-01`, `MULTI-REPO-SUBTASK-01` — Commit `196f8f7`)
- **Commit Local Restore-Point:** `196f8f7` (`feat(webui,runner): add gemini-4-argon model & 2-tier multi-repo domain sub-task decomposer (TASK-M01..M10)` — `7 files changed, 1445 insertions(+), 459 deletions(-)`).
- **Biên lai Kiểm định (`06_evidence/`):** `EV-20261021T221154-3e45` (`unit`, `exit=0`), `EV-20261021T221209-7341` (`integration`, `exit=0`), `EV-20261021T221221-20b6` (`runtime` `go run ./cmd/orchestrator daemon --once`, `exit=0`), `EV-20261021T221324-8111` (`runtime` `verify_live_8090_task10.py`, `exit=0`).
- **Hội đồng `/laya_router` (`DG3`):** `06_laya_summary_dg3_20261021T221738.json` (`gate: PASS`).
- **Tóm tắt 3 Hạng mục Hoàn tất:**
  1. **`AGY-ARGON-01` (Model `gemini-4-argon` & `gemini-4-argon-thinking` trong Antigravity):** Đưa `antigravity/gemini-4-argon` (`gemini-4-argon (antigravity)`) và `antigravity/gemini-4-argon-thinking` lên vị trí mặc định số 1 trong `AntigravityRunner` (`internal/runner/antigravity.go`), `loadCLISettings` (`internal/webui/task_inspector.go`), Topbar Runner Selector, Modal `+ Giao việc mới`, và Modal `⚙ Setup CLI & Model`.
  2. **`DECOMP-DOMAIN-01` (Phân rã Sub-Tasks 2 Tầng theo Nhóm Kiến Trúc & `Context file:line + Mô tả + Test/DoD`):** Xóa bỏ kiểu chia 4 bước quy trình `S1..S5` giả hiệu; xây dựng `FindMatchingAgentWorkspace` + `ParseMarkdownSubTasksFrom08` (Tầng 1: đồng bộ trực tiếp `feat-master-mongo-destination-2026-10-20/08_tasks_master_mongo.md` & `09_tasks_solution_master_mongo.md` thành **5 Nhóm & 10 Sub-Tasks `TASK-M01..TASK-M10`** kèm 3 khối Code Demo thực tế) và `DecomposeMultiRepoCodebaseSubTasks` (Tầng 2: quét đa repo sinh `TASK-<PREFIX>01..N` cho mọi Task lớn mới).
  3. **`MULTI-REPO-SUBTASK-01` (Hỗ trợ Epic Đa-Repo & Cô lập `TargetRepo` Riêng Cho Từng Sub-Task):** Cho phép chọn nhiều `AffectedRepos` cho Task cha (`cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`), gắn từng Sub-Task với đúng `TargetRepo` & `TargetRepoName` riêng (`TASK-M01, M02 -> cdc-cms-service`, `TASK-M03 -> cdc-cms-web`, `TASK-M04..M10 -> centralized-data-service`), hiển thị thanh lọc theo Repo (`#sri-repo-filter-bar`), thẻ Sub-Task theo Nhóm 1..5 và đồ thị **Wave 0..4 Dependency DAG** xuyên 3 repositories.

---

## 10. Báo cáo Hoàn thành Task 11 (`REAL-S4-EXEC-01`, `REAL-S5-QA-01`, `REAL-MONGO-M01-M10`)
- **Biên lai Kiểm định (`06_evidence/`):** `EV-20261022T010017-5dee` (`unit`, `exit=0`), `EV-20261022T010321-738f` (`integration`, `exit=0`), `EV-20261022T010407-c3d1` (`runtime`, `exit=0`).
- **Hội đồng `/laya_router` (`DG3`):** [`06_laya_summary_dg3_20261022T010557.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-modular-agent-orchestrator-2026-10-20/06_laya_summary_dg3_20261022T010557.json) (`trusted_rounds: ["r2"]`, `100% ACCEPT`, `gate: PASS`).
- **Tóm tắt Kết quả Thực thi Xuyên 4 Repositories:**
  1. **Trên `data-hub/cdc-cms-service` (`TASK-M01`, `TASK-M02`):** Mở rộng DTO, Handler (`CreateConnection`, `UpdateConnection`, `TestConnection`) và GORM Repository hỗ trợ `engine_type = "mongodb"`, default port `27017`, `mongodb://` URI (`authSource`, `replicaSet`) và ping MongoDB + lấy `buildInfo` version; pass unit tests trong `master_registry_handler_connections_test.go`.
  2. **Trên `data-hub/cdc-cms-web` (`TASK-M03`):** Cập nhật `src/pages/MasterRegistry.tsx` thêm cột `Engine` (`MONGODB` / `POSTGRESQL`), lựa chọn `Engine Type` trong Modal Quản lý Kết nối Đích, tự động chuyển đổi port `5432` $\leftrightarrow$ `27017` và trường `Auth Source` / `Replica Set`; `npm run build` pass sạch lỗi (`493ms`).
  3. **Trên `data-hub/centralized-data-service` (`TASK-M04` $\rightarrow$ `TASK-M10`):**
     - `TASK-M04`: Bổ sung `masterMongoPool` và `GetMasterMongoClient` vào `ConnectionManager` (`connection_manager.go` + `connection_manager_mongo_test.go`).
     - `TASK-M05`: Bổ sung `applyMongoCollectionAndIndexes` (`CreateCollection` + `Indexes().CreateMany` cho `_gpay_id` unique, `_source_id`, `_source_ts`) trong `MasterDDLGenerator`.
     - `TASK-M06` & `TASK-M07`: Xây dựng `mongo_type_converter.go` (`CoerceToBSONValue`, `CoerceRecordToBSONDoc`) và `transmuter_mongo.go` (`BuildMongoOCCBulkModels`, `bulkUpsertMongoMaster`, `deleteFromMongoMasterBySourceAndGpayIDs`) tích hợp vào `TransmuterModule`.
     - `TASK-M08` & `TASK-M09`: Xây dựng `recon_dest_mongo.go` tích hợp vào `ReconDestAgent` & `GetMasterAgent`, và mở rộng `recon_execute_heal_handler.go` hỗ trợ Self-Healing / Prune trên Master MongoDB.
     - `TASK-M10`: Kiểm thử tích hợp & đa worker concurrency (`transmuter_mongo_test.go`, `recon_dest_mongo_test.go`).
  4. **Trên `ai-orchestrator` (`REAL-S4-EXEC-01` & `REAL-S5-QA-01`):** Xóa bỏ gán giả `StageDone -> done`, bổ sung `VerifySubTaskCodeOnDisk`, gom `2288` dòng `git diff` thật từ 3 repo vào `diff.patch`, chạy test/build thật trên từng repo đích và xuất báo cáo `audit_report.md` 6 mục thật trên `http://127.0.0.1:8090`.

---

## 11. Báo cáo Hoàn thành Task 12 (`LIVE-PROCESS-HUD-01`, `ROLE-MODEL-ATTRIBUTION-01`, `CO-ARCHITECT-CHAT-01`, `TASK-INTAKE-DUAL-MODE-01`)
- **Biên lai Kiểm định (`06_evidence/`):** `EV-20261022T100330-8ddd` (`unit`, `exit=0`), `EV-20261022T100421-1a06` (`integration`, `exit=0`), `EV-20261022T100437-6ff9` (`integration`, `exit=0`), `EV-20261022T100516-ee5e` (`runtime`, `exit=0`), `EV-20261022T100545-d1ef` (`runtime`, `exit=0`), `EV-20261022T100556-312d` (`runtime`, `exit=0`).
- **Hội đồng `/laya_router` (`DG3`):** [`06_laya_summary_dg3_20261022T100829.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-modular-agent-orchestrator-2026-10-20/06_laya_summary_dg3_20261022T100829.json) (`gate: PASS`).
- **Tóm tắt 4 Nâng cấp Lõi trên `http://127.0.0.1:8090`:**
  1. **`LIVE-PROCESS-HUD-01` (`#live-telemetry-hud`):** Bổ sung bảng điều khiển Telemetry 4 ô thời gian thực ngay đầu màn hình Autopilot hiển thị: **(1) Đang chạy cái gì** (`Stage`, `Step M/17`, `Sub-Task TASK-M01..M10`), **(2) Ai & Model nào đang làm** (`RoleLabel`, `AgentSlot`, `SkillBadge`, `Model`), **(3) Đang chạy ở đâu** (`ActiveRepoName`, `Worktree/Branch`, `Context file:line`), và **(4) Chạy như thế nào & Tiến độ %** (`ActiveCommand`, `OverallProgressPct%` + thanh tiến độ riêng cho từng repo `cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`).
  2. **`ROLE-MODEL-ATTRIBUTION-01`:** Gắn huy hiệu `Role + Slot + Skill + Model` lên toàn bộ 17 bước workflow bên trái, 6 ô Autopilot stage cards, và 10 thẻ Sub-Tasks (`TASK-M01..TASK-M10`), kèm dropdown cho phép đổi trực tiếp `Role/Assignee` và `Model` của từng Sub-Task (`POST /api/tasks/subtasks` với `action: "update_role_model"`, lưu xuống `subtasks.json` và `08_tasks.md`).
  3. **`CO-ARCHITECT-CHAT-01` (`#co-architect-studio` — `POST /api/tasks/chat`):** Tích hợp khung đối thoại đồng kiến trúc ngay trong `Step Report, Solution & Code Demo Inspector`, cho phép chọn `Role` (`BRAIN / MUSCLE / QA`) và `Model` để thảo luận (`discuss`), yêu cầu sửa trực tiếp giải pháp `09_tasks_solution.md` (`revise_solution`) trước khi bấm `✓ APPROVE`, hoặc thêm/cập nhật Sub-Tasks (`update_subtasks`).
  4. **`TASK-INTAKE-DUAL-MODE-01` (Modal `+ Giao việc mới` 2 Chế độ):**
     - **Option 1 (`🔗 Đồng bộ từ Jira / Linear / GitHub / GitLab`):** Quản lý và xác thực nhiều tài khoản quản lý công việc (`/api/trackers`, `/api/trackers/test`, mask token theo Rule #20), duyệt/tìm kiếm ticket (`/api/trackers/issues`) và chọn ticket để tự động nạp `ExternalID`, `Title`, `Description`, `BRD Markdown` và `AffectedRepos`.
     - **Option 2 (`📝 Nhập Thủ Công: Title + Detailed Description + Copy/Upload Tài Liệu BRD`):** Tách bạch 3 ô nhập liệu gồm `Tiêu đề công việc`, `Mô tả chi tiết & Tiêu chí nghiệm thu`, và `Tài liệu BRD / PRD / Spec` (hỗ trợ copy-paste, upload file `.md/.txt/.json`, hoặc nạp mẫu BRD), lưu vào `brd_source.md` và `01_requirements.md`.

