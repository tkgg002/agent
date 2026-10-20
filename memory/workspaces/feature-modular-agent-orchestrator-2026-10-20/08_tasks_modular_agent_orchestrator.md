# 08 — Tasks & Execution Checklist: `modular_agent_orchestrator`

> **Trạng thái:** `PROPOSED` — Chờ User `APPROVE` trước khi thực thi (Rule #1, Rule #13).

---

## Task 1 (Phase 1): Xây dựng Core Queue Engine, Wave DAG State Store & OpenClaw Memory (`orchestrator_core`)
- **[Mô tả & Phạm vi]:**
  - Khởi tạo package điều phối trung tâm (`tooling/orchestrator/store.py` & `schema.sql` hoặc Go package tương đương) sử dụng **SQLite WAL + FTS5** lưu trữ:
    1. Bảng `task_queue` (hàng đợi thô từ Scout + hàng đợi ưu tiên từ Sorting + hàng đợi Coding/Review).
    2. Bảng `task_dag_edges` (quan hệ phụ thuộc cha-con giữa các sub-task theo `Wave 0..N`, tự động chuyển trạng thái từ `BLOCKED` sang `READY` khi toàn bộ node cha `DONE`).
    3. Bảng `autopilot_rounds` (ghi nhận `cycle`, `round_num`, `stage`, `verdict`, `blocking_count`, `advisory_count`, `turns`, `duration_sec`, `cost_usd`).
    4. Bảng `memory_chunks_fts` (SQLite FTS5 BM25 index cho `.orchestrator/memory/MEMORY.md`, `daily/`, `tasks/`).
- **[Context & Bằng chứng `file:line`]:**
  - Kế thừa cơ chế kiểm định `/laya_router` tại `tooling/laya_gate.py:108-141` và quản trị workspace tại `tooling/verify_governance.py:102-147`.
- **[Definition of Done (DoD)]:**
  - Unit tests & integration tests trên SQLite thật kiểm chứng: (1) Enqueue/Dequeue atomic theo `priority_score DESC`; (2) Tự động mở khóa node `Wave k+1` khi node `Wave k` chuyển sang `DONE`; (3) Không bị race condition khi nhiều worker cùng bốc queue (`BEGIN IMMEDIATE`).

---

## Task 2 (Phase 2): Xây dựng Non-LLM Scout Worker & Priority Sorting Worker (`scout_and_sorting`)
- **[Mô tả & Phạm vi]:**
  - **Scout Worker (`tooling/orchestrator/workers/scout.py`):** Script/Worker thuần (0 LLM token) polling Jira REST API (`/rest/api/3/search`) và GitHub Issues API (`/repos/{owner}/{repo}/issues`), lọc theo JQL/Label, chống trùng lặp bằng `external_id` (`JIRA-123`, `GH-456`) và đẩy vào `Task Queue` (`status = 'RAW'`).
  - **Sorting Worker (`tooling/orchestrator/workers/sorter.py`):** Bốc các bản ghi `RAW`, tính `priority_score` (dựa trên mức độ nghiêm trọng `Blocker/Critical`, nhãn `security/production`, tuổi ticket, và đánh giá nhanh từ model nhẹ nếu cần), chuyển trạng thái sang `PRIORITIZED` trong `Priority Task Queue`.
- **[Context & Bằng chứng `file:line`]:**
  - Tuân thủ kỷ luật bảo mật secret/token tại `hooks/agy/agy_guard.py:41-46` & `hooks/agy/agy_guard.py:370-376`.
- **[Definition of Done (DoD)]:**
  - Chạy test thực tế với payload Jira/GitHub thật; đảm bảo Scout không gọi bất kỳ LLM API nào (0$ cost) và Sorting xếp đúng thứ tự ưu tiên trên `Priority Task Queue`.

---

## Task 3 (Phase 3): Xây dựng Planner Worker (Opus), Coder Worker (Dynamic Model Router + Git Worktree) & Reviewer/QA Autopilot Loop (`agents_pipeline`)
- **[Mô tả & Phạm vi]:**
  - **Planner Worker (`workers/planner.py`):** Cố định model `claude-opus` (vai trò Brain theo `GEMINI.md:19`), bốc task cao nhất từ `Priority Task Queue`, truy xuất `Project Memory`, sinh `TaskSpec` + `Wave DAG` sub-tasks (mỗi sub-task được gắn `complexity: low|medium|high` và `recommended_model: sonnet|opus`). Chạy vòng `Plan Review` (QA + Security + `laya_gate.py --stage dg1/dg2`) và dừng chờ `Human Approval` (`NEEDS_YOU`).
  - **Coder Worker (`workers/coder.py`):** Khi có sub-task `READY`, khởi tạo `git worktree` cô lập (`.orchestrator/worktrees/<task_id>`), chọn động Runner/Model (`opus` cho `high`, `sonnet` cho `low/medium`), chạy TDD + Implement + Build/Test, rồi đẩy sang `Review Queue` (`IN_REVIEW`).
  - **Reviewer & Local QA Worker (`workers/reviewer.py`):** Chạy `Code Review` + `Security Audit` + `Local QA` + `laya_gate.py --stage dg3` (`review`). Ghi nhận `turns`, `duration_sec`, `cost_usd` vào `autopilot_rounds`. Nếu `blocking > 0` (tối đa 3 rounds) $\rightarrow$ trả về `Coding Queue`; nếu `PASS` $\rightarrow$ merge/tạo Draft PR và mở khóa các task con ở Wave tiếp theo.
- **[Context & Bằng chứng `file:line`]:**
  - Tích hợp trực tiếp với `tooling/laya_gate.py:1-24` và các cổng `G1–G8` trong `GEMINI.md:98-99`.
- **[Definition of Done (DoD)]:**
  - Chạy trọn vẹn 1 luồng end-to-end từ `Priority Queue -> Plan (2 rounds) -> Approve -> Implement (Worktree) -> Code Review & Local QA -> Done`, ghi đầy đủ log vào `05_progress` và `audit_report_<suffix>.md`.

---

## Task 4 (Phase 4): Xây dựng Giao diện Điều khiển Hợp nhất (Wave DAG + Autopilot Cost Tracker + Multi-Terminal Grid) (`orchestrator_ui`)
- **[Mô tả & Phạm vi]:**
  - Xây dựng Dashboard điều khiển (TUI bằng Go Bubble Tea hoặc Web Dashboard kết nối qua WebSocket/HTTP tới Orchestrator Daemon) hiển thị 3 màn hình:
    1. **Wave Dependency DAG View (như Ảnh 1):** Hiển thị các cột `Wave 0..N`, đường nối phụ thuộc, đường găng màu vàng (`critical`), bộ lọc `needs you`, `running`, `in review`, `ready`, `blocked`, `done`.
    2. **Autopilot Pipeline & Cost Ledger (như Ảnh 2):** Hiển thị tiến trình `Start -> Plan / Plan review -> You -> Implement / Code review -> Local QA / Fix -> Draft PR` kèm bảng chi tiết từng Round (`turns`, `min`, `$ cost`).
    3. **Multi-Agent Terminal Grid (như Ảnh 3):** Lưới điều khiển song song các worker (`A..K`) đang chạy `Claude`, `Codex`, `Gemini`, `OpenCode`, `Terminal` theo 17 bước quy trình.
- **[Context & Bằng chứng `file:line`]:**
  - Kết nối trực tiếp vào database trạng thái `orchestrator.db` và PTY/tmux session của từng worker.
- **[Definition of Done (DoD)]:**
  - Hiển thị realtime trạng thái các Wave, chi phí từng Round và stream terminal của từng Agent mà không bị treo hay rò rỉ goroutine/process.
