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
