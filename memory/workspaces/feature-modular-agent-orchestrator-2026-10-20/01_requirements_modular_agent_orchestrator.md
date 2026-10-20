# 01 — Requirements: Hệ thống Điều phối Đa Tác nhân Modular Queue-Driven (`modular_agent_orchestrator`)

tier: T2
suffix: modular_agent_orchestrator
workspace: feature-modular-agent-orchestrator-2026-10-20
status: PROPOSED (Waiting for User APPROVE)

---

## 1. Bối cảnh & Mục tiêu Tổng hợp (Phiên 1 + Phiên 2)

Tổng hợp tinh hoa từ 2 phiên thảo luận và nền tảng `/agent` hiện có để thiết kế & xây dựng **Hệ thống Điều phối Đa Tác nhân theo Hàng đợi (Modular Queue-Driven AI Coding Agents Orchestrator)**:

1. **Từ Phiên 1 (`Open-Makers/ai-coding-agents-orchestrator`):**
   - Pipeline phát triển phần mềm tất định, có thể khôi phục trạng thái (resumable): `Collect Context → Negotiate (Human Gate) → Decompose (Human Gate) → Implement (TDD) → Quality Review → Fix → Done`.
   - Đội ngũ Agent chuyên biệt (`PM/Planner`, `Coder`, `QA`, `UX Reviewer`, `Security`) với System Prompt và Skills riêng.
   - Bộ nhớ dự án kiểu OpenClaw (`.orchestrator/memory/`: `MEMORY.md`, `daily/YYYY-MM-DD.md`, `tasks/<id>.md`) đánh chỉ mục bằng **SQLite FTS5 (BM25 + Hybrid Embedding)**.
   - Đa Runner CLI (`Claude CLI`, `Codex CLI`, `Gemini`, `OpenCode`) và Local LLM (`Ollama`, `LM Studio`, `MLX`).

2. **Từ Phiên 2 (Kiến trúc Modular Queue-Driven + 3 Giao diện Thực chiến):**
   - **Separation of Concerns qua Hàng đợi (`Task Queue` & `Priority Task Queue`):**
     - **Scout Agent (Non-LLM Script/Worker):** Gọi trực tiếp REST/GraphQL API của Jira, GitHub Issues... gom việc đẩy vào `Task Queue` mà không tốn LLM token.
     - **Sorting Agent (Triage & Priority Scorer):** Bốc việc từ `Task Queue`, chấm điểm độ ưu tiên và sắp xếp vào `Priority Task Queue`.
     - **Planner Agent (Khóa cứng `Claude Opus`):** Bốc việc từ `Priority Task Queue`, lập kế hoạch và phân rã thành **Đồ thị phụ thuộc theo đợt (`Wave 0 → Wave N` Dependency DAG — Ảnh 1)** với các trạng thái `READY`, `BLOCKED`, `NEEDS YOU`, `DONE`, `critical`, rồi đẩy các task `READY` sang `Coding Queue`.
     - **Coder Agent (Dynamic Model Routing):** Bốc plan từ `Coding Queue` và viết code trên nhánh/worktree cô lập; tự động chọn **Opus** (task khó/core) hoặc **Sonnet** (task tiêu chuẩn) tùy theo độ phức tạp được gắn trong Plan.
     - **Reviewer & Local QA Agent (`AUTOPILOT` Loop — Ảnh 2):** Bốc code từ `Review Queue`, chạy vòng lặp kiểm định đa tầng (`Plan review`, `Code review`, `Local QA`, `Security`, `/laya_router` `DG1–DG3`), thống kê chi tiết **Turns / Thời gian / Chi phí USD (`$`)** của từng round trước khi xuất `Draft PR`.
   - **Giao diện Điều khiển Đa luồng (`Multi-Terminal Agent Grid` — Ảnh 3):**
     - Quản lý đa dự án, theo dõi 17 bước quy trình chuẩn và giám sát lưới nhiều Worker/Terminal chạy song song (`A..K`).

---

## 2. Danh sách Yêu cầu Chức năng (Functional Requirements — Đóng kín `F1–F6`)

- **R0 (Nguyên lý Dung Hợp: "Vỏ Queue-Driven" + "Lõi `/agent` Kernel"):**
  - **Tầng Vỏ (`Outer Event/Queue-Driven Orchestration Plane`):** Đảm nhiệm gom việc tự động (`Scout`), xếp hạng ưu tiên (`Sorting`), lập đồ thị phụ thuộc (`Wave 0..N` DAG), định tuyến model động (`Opus 5.5` vs `Sonnet 4.5`) và chạy song song nhiều Worker trên `git worktree` độc lập.
  - **Tầng Lõi (`Inner /agent Governance Kernel`):** Giữ nguyên 100% hiến pháp `GEMINI.md` (`S0 → S6`), 7 Lifecycle Hooks `H1–H7` (`hooks/agy/agy_guard.py`), cổng toán học `/laya_router` `DG1–DG3` (`tooling/laya_gate.py`), cách ly phòng sạch (`Clean-Room Isolation`), cổng đột biến `3-Mutant Gate` (`tooling/mutation_gate.py`), Process Linter (`tooling/verify_governance.py`) và bộ nhớ 2 tầng (`memory/global/lessons/INDEX.md`). Mọi Worker ở Tầng Vỏ khi thực thi đều bắt buộc đi qua Tầng Lõi `/agent`.
- **R1 (Đóng `F1` & `F6` — Modular Queue Store & Hybrid FTS5 Memory Indexer):**
  - Lưu trữ bền vững `Task Queue` (`RAW`), `Priority Task Queue` (`PRIORITIZED`), `Wave DAG` (`PLANNING | NEEDS_YOU | READY | BLOCKED | RUNNING | IN_REVIEW | DONE`) trong SQLite WAL (`.orchestrator/orchestrator.db`).
  - Xây dựng bộ đánh chỉ mục tìm kiếm toàn văn **SQLite FTS5 BM25** cho cả `.orchestrator/memory/` lẫn **hơn 70 workspaces lịch sử** trong `agent/memory/workspaces/*/` (`01_requirements`, `09_tasks_solution`, `13_analysis`, `audit_report`) và `agent/memory/global/lessons/topics/*.md`.
- **R2 (Đóng `F2` — Non-LLM Scout Worker `$0` & Priority Sorting Worker):**
  - `Scout Worker` chạy bằng script Python thuần (0 token LLM, `$0` chi phí) kết nối Jira/GitHub Issues/Local Backlog API, lọc theo JQL/Label, chống trùng lặp bằng `external_id`, tuân thủ mask secret `H3` (`agy_guard.py:284-309`) và đẩy vào `Task Queue` (`RAW`).
  - `Sorting Worker` bốc task `RAW`, chấm điểm `priority_score` (0–100) theo Severity/SLA/Labels và đẩy sang `Priority Task Queue` (`PRIORITIZED`).
- **R3 (Đóng `F3` — Opus 5.5 Planner + Wave 0..N Dependency DAG + Inner Kernel `S1–S3` & `DG1/DG2`):**
  - `Planner Worker` luôn khóa cứng model `claude-opus-5.5` (vai trò Brain), bốc task đầu bảng `Priority Task Queue`, khởi tạo hồ sơ workspace chuẩn `/agent` (`01_requirements`, `05_progress` append `START`, `08_tasks`, `09_tasks_solution`), phân rã thành **Đồ thị phụ thuộc `Wave 0..N` DAG** (gắn `complexity`, `critical_path`, `clean_room` cho từng node), bắt buộc vượt qua `laya_gate.py run --stage dg1` & `--stage dg2` (`PASS`) và dừng chờ `Human Approval (NEEDS_YOU)` trước khi mở khóa `Wave 0` sang `READY`.
- **R4 (Đóng `F4` — Dynamic Model Router + Isolated Git Worktree + Inner Kernel `H2–H6` & `Clean-Room`):**
  - `Coder Worker` bốc các node `READY` từ `Coding Queue`, tự động định tuyến model (`claude-opus-5.5` cho `complexity == "high"`, `claude-sonnet-4.5` cho `complexity in ("medium", "low")`), thực thi trên `git worktree` riêng biệt (`.orchestrator/worktrees/<task_id>`).
  - Trong suốt quá trình viết code, Coder bị giám sát cứng bởi **Inner Kernel Hooks `H2–H6`** (chặn cài mock `*Fn`/`sqlmock` vào code production, chặn `Overwrite` memory, chặn `git push`/`git reset --hard`, tự động chạy `checks`/`test`/`runtime_check` từ `.agent-project.json`) và tuân thủ **Clean-Room Isolation** (nếu node có `clean_room=True`, Coder chỉ nhận `## Clean-Room Approved Spec`, tuyệt đối không đọc file gốc cấm).
- **R5 (Đóng `F5` — Autopilot Review/QA Loop + Inner Kernel `S5–S6`, `3-Mutant Gate`, `DG3` & Cost/Turn Ledger):**
  - `Reviewer/QA Worker` bốc từ `Review Queue` (`IN_REVIEW`), thực thi đầy đủ **6 bước Mandatory Final Adversarial Audit (`S5`) của `/agent`**: (a) Line-by-Line Diff Critique, (b) Anti-Smoke-Test Speculation, (c) Zero-Mockup & Runtime Trace, (d) `3-Mutant Gate` (`tooling/mutation_gate.py` $\ge 3$ mutants `KILLED`), (e) `/laya_router` `DG3` (`tooling/laya_gate.py run --stage dg3`), (f) xuất `audit_report_<suffix>.md` & chạy `tooling/verify_governance.py`.
  - Ghi nhận định lượng `round_num (1..3)`, `blocking_count`, `advisory_count`, `turns`, `duration_sec`, `cost_usd ($)` vào bảng `autopilot_rounds` và append vào `05_progress_<suffix>.md`. Nếu `blocking > 0` $\rightarrow$ trả về `READY` để Coder sửa; nếu `0 blocking (PASS)` $\rightarrow$ chuyển `DONE`, tự động mở khóa (`unblock`) các node con ở `Wave k+1` và tạo `Draft PR`.
- **R6 (Observability Control Plane — Wave DAG, Autopilot Ledger & Multi-Worker Grid + `/agent` Gate Badges):**
  - Cung cấp CLI & Dashboard hiển thị trực quan: (1) Bản đồ `Wave 0..N` Dependency DAG, (2) Bảng `AUTOPILOT` theo dõi từng Round, Chi phí `$` & trạng thái cổng `DG1/DG2/DG3`/`Mutation Gate`, và (3) Lưới `Multi-Terminal Agent Grid (Workers A..K)`.

