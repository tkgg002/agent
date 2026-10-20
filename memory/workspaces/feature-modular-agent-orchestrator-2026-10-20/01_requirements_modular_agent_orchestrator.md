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

## 2. Danh sách Yêu cầu Chức năng (Functional Requirements)

- **R1 (Modular Queue & DAG State Store — SQLite WAL):** Lưu trữ bền vững `Task Queue`, `Priority Task Queue`, `Coding Queue`, `Review Queue`, và đồ thị phụ thuộc `Wave DAG` (`parent_task_id`, `wave_index`, `status: NEEDS_YOU | READY | BLOCKED | RUNNING | IN_REVIEW | DONE`, `critical_path`) trong SQLite WAL (`.orchestrator/orchestrator.db`) để không phụ thuộc hạ tầng ngoài khi chạy CLI/Local.
- **R2 (Hybrid Non-LLM Scout & Triage Sorting):** `Scout Worker` chạy bằng script/code thuần (Python/Go) kết nối Jira/GitHub API (0 token LLM) đẩy vào `Task Queue`; `Sorting Worker` chấm điểm ưu tiên (`priority_score`) đẩy sang `Priority Task Queue`.
- **R3 (Opus Planner & Wave DAG Unblocker):** `Planner Worker` luôn cố định model `claude-opus` (hoặc tương đương cấp Brain), bốc từ `Priority Task Queue`, sinh `TaskSpec` + `Wave DAG` sub-tasks, chạy cổng `Plan Review` (QA + Security + `/laya_router` `DG1/DG2`) và dừng ở cổng `Human Approval (You)` trước khi mở khóa các node `Wave 0` sang `READY`.
- **R4 (Dynamic Model Router & Isolated Git Worktree Coder):** `Coder Worker` bốc các node `READY` từ `Coding Queue`, tự động chọn model (`opus` khi `complexity == "high"`, `sonnet` khi `complexity in ("medium", "low")`), thực thi trên `git worktree` riêng biệt để nhiều Coder (`A..K`) chạy song song không đụng độ file.
- **R5 (Autopilot Quality Gate, Cost/Turn Tracker & `/laya_router` Integration):** `Reviewer/QA Worker` bốc từ `Review Queue`, thực thi TDD/Local QA + Code Review + Security Audit + `/laya_router` (`DG3`), ghi nhận `turns`, `duration_sec`, `cost_usd` cho từng Round (`round 1..3`); nếu có `blocking/must-fix` thì đẩy ngược về `Coding Queue`, nếu `pass` thì mở khóa các node `BLOCKED` ở Wave kế tiếp và tạo `Draft PR`.
- **R6 (Observability UI — Wave DAG, Autopilot & Multi-Terminal Grid):** Cung cấp giao diện trực quan hóa đủ 3 chế độ trong ảnh: (1) Bản đồ `Wave 0..N` Dependency DAG, (2) Bảng `AUTOPILOT` theo dõi từng Round & Chi phí `$`, và (3) Lưới `Multi-Agent Terminal Grid` theo dõi trạng thái các worker.
