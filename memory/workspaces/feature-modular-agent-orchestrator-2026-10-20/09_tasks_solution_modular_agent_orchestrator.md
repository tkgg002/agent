# 09 — Technical Solution: Hệ thống Điều phối Đa Tác nhân Modular Queue-Driven (`modular_agent_orchestrator`)

> **Trạng thái:** `PROPOSED` — Chờ User `APPROVE` trước khi thực thi (Rule #1, Rule #13).
> **Workspace:** `feature-modular-agent-orchestrator-2026-10-20` | **Tier:** `T2` | **Stage:** `S3`

---

## 1. Root Cause / Bài toán cốt lõi (Tổng hợp Phiên 1 + Phiên 2 + Hiện trạng `/agent`)

### 1.1. Đối chiếu 3 mảnh ghép kiến trúc (`VERIFIED`)
1. **Phiên 1 (`Open-Makers/ai-coding-agents-orchestrator`):**
   - **Điểm mạnh:** Cung cấp một CLI/TUI Engine viết bằng Go điều phối 5 vai trò (`PM`, `Coder`, `QA`, `UX Reviewer`, `Security`) chạy theo luồng `Collect Context → Negotiate → Decompose → Implement (TDD) → Quality Review → Fix → Done`; có bộ nhớ dự án kiểu OpenClaw (`.orchestrator/memory/MEMORY.md`, `daily/`, `tasks/`) đánh chỉ mục bằng **SQLite FTS5 (BM25 + Hybrid Embedding)** và theo dõi sub-task bền vững qua `beads`.
   - **Hạn chế:** Chạy đơn luồng theo lệnh gõ tay (`orchestrator task --description ...`); chưa có cơ chế tự động đi gom việc từ Jira/GitHub; chưa có hàng đợi ưu tiên (`Priority Queue`); chưa phân tách Worker không dùng LLM (`Non-LLM Script`) với Worker dùng LLM; chưa có đồ thị phụ thuộc nhiều đợt (`Wave 0..N` DAG) để chạy song song nhiều Coder.
2. **Phiên 2 (Mô hình Modular Queue-Driven Multi-Agent + 3 Giao diện Thực chiến):**
   - Giải quyết triệt để hạn chế của Phiên 1 bằng tư duy **Modular Design + Separation of Concerns qua Hàng đợi (`Queue-Driven`)**:
     - **Scout Agent (Non-LLM Script — 0 Token):** Chỉ dùng REST/GraphQL API của Jira, GitHub Issues để tìm ticket/issue mới và đẩy vào **`Task Queue`**.
     - **Sorting Agent:** Bốc từ `Task Queue`, đánh giá mức độ quan trọng và sắp xếp vào **`Priority Task Queue`**.
     - **Planner Agent (Cố định `Claude Opus`):** Bốc việc từ đầu `Priority Task Queue`, tạo kế hoạch hoặc đẻ thêm sub-tasks theo **Đồ thị phụ thuộc theo đợt (`Wave 0 → Wave N` Dependency DAG — Ảnh 1)** với các trạng thái `NEEDS YOU`, `READY`, `BLOCKED`, `RUNNING`, `IN REVIEW`, `DONE` và nhánh `critical`.
     - **Coder Agent (Dynamic Model Routing — `Opus` vs `Sonnet`):** Cứ có plan ở trạng thái `READY` là bốc ra viết code trên nhánh/worktree cô lập; tự động chọn **Opus** hay **Sonnet** tùy theo độ khó của sub-task trong plan.
     - **Reviewer & Local QA Agent (`AUTOPILOT` Loop — Ảnh 2):** Cứ có code là bốc ra review & chạy Local QA theo vòng lặp tự trị (`Round 1..3`), thống kê chi tiết **Số turns, Thời gian chạy, và Chi phí `$ USD`** của từng bước trước khi tạo `Draft PR`.
     - **Bảng điều khiển Đa luồng (`Multi-Terminal Agent Grid` — Ảnh 3):** Điều phối song song nhiều worker (`A..K`) trên nhiều CLI/Model (`Claude`, `Codex`, `Gemini`, `OpenCode`, `Terminal`) bám sát quy trình chuẩn hóa.
3. **Hiện trạng Hệ thống `/agent` của chúng ta (`REF: /Users/trainguyen/Documents/work/agent/GEMINI.md:18-30`):**
   - Đã có sẵn bộ **Phân quyền Brain / Muscle / QA / Security** (`GEMINI.md:18-21`), nguyên tắc **Agnostic Core** (`GEMINI.md:28-30`), cổng thẩm định toán học **`/laya_router` (`DG1–DG3`)** (`tooling/laya_gate.py:10-24`), bộ **7 Lifecycle Hooks (`H1–H7`)** (`hooks/agy/agy_guard.py:7-21`) và **Process Linter** (`tooling/verify_governance.py:25-54`).
   - Tuy nhiên, hiện tại `/agent` đang vận hành theo mô hình **Interactive Single-Session** (1 phiên chat IDE tại 1 thời điểm), chưa có lớp **Queue Daemon (`Task Queue` / `Priority Queue` / `Wave DAG`)** để tự động hóa dây chuyền từ Jira/GitHub tới Draft PR.

---

## 2. Giải pháp Kỹ thuật Duy nhất (Unified Modular Queue-Driven Architecture)

Thiết kế hệ thống **Modular AI Coding Agents Orchestrator** thành **4 tầng độc lập** giao tiếp hoàn toàn qua **SQLite WAL Queue + Event Bus**, kế thừa 100% kỷ luật của `/agent`:

```mermaid
flowchart TB
    subgraph T1["Tầng 1: Ingestion & Triage (Tiết kiệm Token)"]
        EXT["Jira API / GitHub Issues"] --> SCOUT["1. Scout Worker (Non-LLM Python/Go Script - $0)"]
        SCOUT -->|INSERT status=RAW| TQ[("Task Queue (SQLite WAL)")]
        TQ --> SORT["2. Sorting Worker (Rule + Fast Triage)"]
        SORT -->|UPDATE priority_score DESC| PTQ[("Priority Task Queue")]
    end

    subgraph T2["Tầng 2: Brain Planning & Wave DAG (Khóa cứng Opus)"]
        PTQ -->|Dequeue Top Priority| PLAN["3. Planner Agent (Claude Opus 5.5)"]
        MEM[("OpenClaw Memory (MEMORY.md + SQLite FTS5 BM25)")] <--> PLAN
        PLAN --> PREV["Plan Review Loop (QA + Security + /laya_router DG1/DG2)"]
        PREV -->|Round 1..3 Revise| PLAN
        PREV -->|PASS| GATE["Human Approval Gate (NEEDS YOU -> APPROVED)"]
        GATE -->|Unlock Wave 0| DAG[("Wave 0..N Dependency DAG (READY / BLOCKED)")]
    end

    subgraph T3["Tầng 3: Execution & Autopilot Quality Loop (Dynamic Routing)"]
        DAG -->|Pop status=READY| ROUTER{"Dynamic Model Router (Theo độ khó Plan)"}
        ROUTER -->|High Complexity / Core| CODER_O["4a. Coder Worker (Opus + Git Worktree)"]
        ROUTER -->|Standard / CRUD / UI| CODER_S["4b. Coder Worker (Sonnet + Git Worktree)"]
        CODER_O & CODER_S -->|Push branch/worktree| RQ[("Review & QA Queue")]
        RQ --> REV["5. Reviewer & Local QA Agent (QA + Security + /laya_router DG3)"]
        REV -->|Blocking > 0 (Round 1..3)| DAG
        REV -->|0 Blocking (PASS)| UNLOCK["Unblock Wave k+1 & Create Draft PR"]
        UNLOCK --> MEM
    end

    subgraph T4["Tầng 4: Control Plane UI (3 Chế độ)"]
        UI1["View 1: Wave 0..N DAG Board"]
        UI2["View 2: Autopilot Rounds & Cost ($) Ledger"]
        UI3["View 3: Multi-Terminal Agent Grid (Workers A..K)"]
    end
```

### 2.1. Chi tiết 5 Module Agent & Quy tắc Định tuyến Model
| Thứ tự | Tên Agent | Bản chất thực thi | Model sử dụng | Hàng đợi Đọc → Ghi | Vai trò & Quy tắc cốt lõi |
| :---: | :--- | :--- | :--- | :--- | :--- |
| **1** | **Scout Agent** | **Non-LLM Script** (`workers/scout.py`) | **Không dùng LLM (`$0`)** | Jira/GitHub API → `Task Queue` (`RAW`) | Polling định kỳ REST/GraphQL API, lọc ticket theo JQL/label, chống trùng bằng `external_id`. |
| **2** | **Sorting Agent** | **Rule Scorer + Light Classifier** (`workers/sorter.py`) | Rule-based (hoặc Flash/Haiku khi cần phân loại ngữ nghĩa) | `Task Queue` (`RAW`) → `Priority Task Queue` (`PRIORITIZED`) | Tính `priority_score` (0–100) dựa trên Severity, SLA, Label (`blocker`, `security`), đẩy việc gấp lên đầu hàng đợi. |
| **3** | **Planner Agent** | **LLM Architect Worker** (`workers/planner.py`) | **Luôn cố định `Claude Opus`** | `Priority Task Queue` → `Wave DAG` & `Coding Queue` | Recall `MEMORY.md` (SQLite FTS5), lập `09_tasks_solution`, tách sub-tasks thành **Wave 0..N DAG**, gán `complexity` cho từng sub-task, qua `Plan Review` (`DG1/DG2`) & chờ `You (Approved)`. |
| **4** | **Coder Agent** | **LLM Engineer Worker** (`workers/coder.py`) | **Dynamic: `Opus` (khó) hoặc `Sonnet` (vừa/dễ)** | `Coding Queue` (`READY`) → `Review Queue` (`IN_REVIEW`) | Chạy trong **Git Worktree riêng** (`.orchestrator/worktrees/<task_id>`), viết test TDD + code, chạy `checks`/`test` từ `.agent-project.json`. |
| **5** | **Reviewer / QA Agent** | **LLM Auditor + `/laya_router`** (`workers/reviewer.py`) | **`Opus` / `Sonnet` + `Laya Server (:8200)`** | `Review Queue` (`IN_REVIEW`) → `DONE` (hoặc trả về `READY`) | Chạy `Code Review` + `Local QA` + `Security Audit` + `laya_gate.py review` (`DG3`), ghi `turns / min / $ cost` cho từng Round. Khi `PASS` sẽ tự động mở khóa (`unblock`) các node con ở Wave kế tiếp. |

---

## 3. Blast Radius & Rollback Plan

- **Blast Radius (Phạm vi ảnh hưởng):**
  - Toàn bộ mã nguồn mới của Orchestrator được đóng gói độc lập trong thư mục `tooling/orchestrator/` và lưu dữ liệu runtime tại `.orchestrator/` (gitignored cho file `.db` và `worktrees/`, chỉ giữ Markdown memory).
  - **Không sửa đổi phá vỡ** các hook hiện hành trong `hooks/agy/agy_guard.py` hay linter `tooling/verify_governance.py`; trái lại, các Worker của Orchestrator tái sử dụng trực tiếp `laya_gate.py`, `verify_governance.py` và tuân thủ chuẩn file `01`, `05`, `08`, `09`, `audit_report` của `/agent`.
- **Rollback Plan:**
  - Nếu cần quay lui bất kỳ Phase nào, chỉ cần dừng tiến trình daemon của Orchestrator, dọn dẹp các `git worktree` tạm thời bằng `git worktree prune` và revert commit trên nhánh tính năng tương ứng.

---

## 4. Code Demo: Lõi Queue Engine, Wave DAG Unblocker & Dynamic Model Router

Đoạn code minh họa dưới đây thể hiện chính xác cơ chế:
1. **Non-LLM Scout** đẩy việc vào `Task Queue`.
2. **Sorting Agent** chấm điểm đưa vào `Priority Task Queue`.
3. **Planner Agent (khóa cứng `claude-opus-5.5`)** phân rã thành **Wave DAG (`Wave 0 → Wave 1`)** và kiểm chứng qua `verify_governance.py` (`STAGE_S3_PREFIXES`).
4. **Coder Agent** tự động chọn `claude-opus-5.5` hoặc `claude-sonnet-4.5` tùy theo `complexity` của từng node `READY`, và **Reviewer Agent** gọi `laya_gate.py` (`DECISIVE_CONF = 0.5`, `BLOCK_CONF = 0.8`, `DEBATE_APPROVE_CONF = 0.95`) kèm ghi nhận chi phí từng Round:

```python
#!/usr/bin/env python3
"""Code Demo: Modular Queue-Driven Orchestrator (Scout -> Sorting -> Planner -> Coder -> Reviewer)."""
from __future__ import annotations

import sqlite3
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional

from hooks.agy.agy_guard import MANIFEST, component_for
from tooling.laya_gate import BLOCK_CONF, DEBATE_APPROVE_CONF, DECISIVE_CONF, DEFAULT_URL
from tooling.verify_governance import STAGE_S3_PREFIXES


@dataclass(frozen=True)
class SubTaskSpec:
    task_id: str
    title: str
    wave_index: int
    complexity: str  # "high" | "medium" | "low"
    depends_on: List[str]


def select_agent_model(role: str, complexity: str = "medium") -> str:
    """Dynamic Model Routing (Lưu ý 2):
    - Planner luôn cố định Opus 5.5.
    - Coder chọn Opus 5.5 cho task 'high', Sonnet cho 'medium'/'low'.
    """
    if role == "planner":
        return "claude-opus-5.5"
    if role == "coder":
        return "claude-opus-5.5" if complexity == "high" else "claude-sonnet-4.5"
    if role == "reviewer":
        return "claude-opus-5.5" if complexity == "high" else "claude-sonnet-4.5"
    return "non-llm-script"


class OrchestratorQueueDB:
    """SQLite WAL Queue & Wave DAG State Store."""

    def __init__(self, db_path: Path) -> None:
        self.conn = sqlite3.connect(str(db_path))
        self.conn.row_factory = sqlite3.Row
        self._init_schema()

    def _init_schema(self) -> None:
        with self.conn:
            self.conn.executescript(
                """
                PRAGMA journal_mode=WAL;
                CREATE TABLE IF NOT EXISTS tasks (
                    task_id TEXT PRIMARY KEY,
                    parent_id TEXT,
                    title TEXT NOT NULL,
                    queue_stage TEXT NOT NULL, -- RAW | PRIORITIZED | PLANNING | NEEDS_YOU | READY | BLOCKED | RUNNING | IN_REVIEW | DONE
                    priority_score INTEGER NOT NULL DEFAULT 0,
                    wave_index INTEGER NOT NULL DEFAULT 0,
                    complexity TEXT NOT NULL DEFAULT 'medium',
                    assigned_model TEXT NOT NULL DEFAULT ''
                );
                CREATE TABLE IF NOT EXISTS task_dependencies (
                    task_id TEXT NOT NULL,
                    depends_on_id TEXT NOT NULL,
                    PRIMARY KEY (task_id, depends_on_id)
                );
                CREATE TABLE IF NOT EXISTS autopilot_rounds (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    task_id TEXT NOT NULL,
                    stage TEXT NOT NULL,       -- plan_review | code_review | local_qa
                    round_num INTEGER NOT NULL,
                    verdict TEXT NOT NULL,     -- revise | approve | pass | retry
                    blocking_count INTEGER NOT NULL DEFAULT 0,
                    advisory_count INTEGER NOT NULL DEFAULT 0,
                    turns INTEGER NOT NULL DEFAULT 0,
                    duration_sec INTEGER NOT NULL DEFAULT 0,
                    cost_usd REAL NOT NULL DEFAULT 0.0
                );
                """
            )

    def scout_enqueue_raw(self, external_id: str, title: str) -> None:
        """1. Scout Agent (Non-LLM Script): đẩy ticket mới vào Task Queue."""
        with self.conn:
            self.conn.execute(
                """
                INSERT OR IGNORE INTO tasks (task_id, title, queue_stage, assigned_model)
                VALUES (?, ?, 'RAW', 'non-llm-script')
                """,
                (external_id, title),
            )

    def sorting_prioritize(self, task_id: str, priority_score: int) -> None:
        """2. Sorting Agent: chấm điểm và đẩy vào Priority Task Queue."""
        with self.conn:
            self.conn.execute(
                """
                UPDATE tasks
                   SET queue_stage = 'PRIORITIZED',
                       priority_score = ?
                 WHERE task_id = ? AND queue_stage = 'RAW'
                """,
                (priority_score, task_id),
            )

    def planner_register_wave_dag(self, parent_id: str, specs: List[SubTaskSpec], approved: bool) -> None:
        """3. Planner Agent (Opus 5.5): tạo đồ thị Wave 0..N; mở khóa Wave 0 khi User Approved."""
        with self.conn:
            for sp in specs:
                initial_stage = (
                    ("READY" if approved else "NEEDS_YOU")
                    if not sp.depends_on
                    else "BLOCKED"
                )
                model = select_agent_model("coder", sp.complexity)
                self.conn.execute(
                    """
                    INSERT OR REPLACE INTO tasks
                        (task_id, parent_id, title, queue_stage, wave_index, complexity, assigned_model)
                    VALUES (?, ?, ?, ?, ?, ?, ?)
                    """,
                    (sp.task_id, parent_id, sp.title, initial_stage, sp.wave_index, sp.complexity, model),
                )
                for dep_id in sp.depends_on:
                    self.conn.execute(
                        "INSERT OR IGNORE INTO task_dependencies (task_id, depends_on_id) VALUES (?, ?)",
                        (sp.task_id, dep_id),
                    )

    def coder_pop_next_ready(self) -> Optional[Dict[str, object]]:
        """4. Coder Agent: bốc 1 task READY có độ ưu tiên/wave sớm nhất."""
        with self.conn:
            row = self.conn.execute(
                """
                SELECT task_id, title, complexity, assigned_model, wave_index
                  FROM tasks
                 WHERE queue_stage = 'READY'
                 ORDER BY wave_index ASC, priority_score DESC
                 LIMIT 1
                """
            ).fetchone()
            if not row:
                return None
            self.conn.execute(
                "UPDATE tasks SET queue_stage = 'RUNNING' WHERE task_id = ?",
                (row["task_id"],),
            )
            return dict(row)

    def reviewer_complete_and_unblock_next_wave(
        self,
        task_id: str,
        round_num: int,
        blocking_count: int,
        advisory_count: int,
        turns: int,
        duration_sec: int,
        cost_usd: float,
    ) -> List[str]:
        """5. Reviewer/QA Agent: ghi nhận kết quả Autopilot Round.
        Nếu 0 blocking -> chuyển DONE và tự động mở khóa các node con đã đủ điều kiện sang READY.
        """
        verdict = "pass" if blocking_count == 0 else "revise"
        unblocked: List[str] = []
        with self.conn:
            self.conn.execute(
                """
                INSERT INTO autopilot_rounds
                    (task_id, stage, round_num, verdict, blocking_count, advisory_count, turns, duration_sec, cost_usd)
                VALUES (?, 'code_review', ?, ?, ?, ?, ?, ?, ?)
                """,
                (task_id, round_num, verdict, blocking_count, advisory_count, turns, duration_sec, cost_usd),
            )
            if blocking_count > 0:
                # Đẩy ngược lại Coding Queue để Coder sửa
                self.conn.execute("UPDATE tasks SET queue_stage = 'READY' WHERE task_id = ?", (task_id,))
                return unblocked

            self.conn.execute("UPDATE tasks SET queue_stage = 'DONE' WHERE task_id = ?", (task_id,))
            # Tìm các task BLOCKED mà tất cả depends_on_id đều đã DONE
            rows = self.conn.execute(
                """
                SELECT t.task_id
                  FROM tasks t
                 WHERE t.queue_stage = 'BLOCKED'
                   AND NOT EXISTS (
                       SELECT 1
                         FROM task_dependencies d
                         JOIN tasks dep ON dep.task_id = d.depends_on_id
                        WHERE d.task_id = t.task_id
                          AND dep.queue_stage != 'DONE'
                   )
                """
            ).fetchall()
            for r in rows:
                cid = str(r["task_id"])
                self.conn.execute("UPDATE tasks SET queue_stage = 'READY' WHERE task_id = ?", (cid,))
                unblocked.append(cid)
        return unblocked
```

---

## 5. Bảng Kết quả Decision Gate (`DG1` & `DG2`)

Đã thực thi kiểm định qua `/laya_router` (`tooling/laya_gate.py`):
- **File Summary DG1:** `06_laya_summary_dg1_20261020T120129.json` (`gate: PASS`, `evidence_error: ""`). *(Lưu ý theo Rule #10: ở batch DG1, `trusted_rounds` rỗng nên các claim được ghi nhận `UNVALIDATED` và được xác thực trực tiếp bằng dòng code thật tại `REF:` trên đĩa).*
- **File Summary DG2:** `06_laya_summary_dg2_20261020T120130.json` (`gate: PASS`, `trusted_rounds: ["r2", "r3"]`, 100% claim đạt `ACCEPT`).

| Gate | Claim ID | Nội dung Claim | REF Kiểm chứng | Quyết định | Kết quả Gate |
| :---: | :---: | :--- | :--- | :---: | :---: |
| **DG1** | `S-RC` | `GEMINI.md` định nghĩa 3 nhóm vai trò Brain, Muscle và Sub-Agents/Skills. | `GEMINI.md:18-21` | `UNVALIDATED` (Verified by `REF:`) | **PASS** 🟢 |
| **DG1** | `S-FIX` | `laya_gate.py` quy định exit code `0` cho `PASS` và `2` cho `BLOCK or NEEDS_RESOLUTION`. | `tooling/laya_gate.py:10-14` | `UNVALIDATED` (Verified by `REF:`) | **PASS** 🟢 |
| **DG1** | `S-IMPACT` | `APPEND_ONLY` trong `agy_guard.py` bảo vệ `05_progress`, `04_decisions`, `lessons`, `governance_metrics`. | `hooks/agy/agy_guard.py:39-40` | `UNVALIDATED` (Verified by `REF:`) | **PASS** 🟢 |
| **DG1** | `S-PATTERN` | Quy tắc Agnostic Core (ADR-07) cấm hardcode tên service/port dự án vào core. | `GEMINI.md:28-30` | `UNVALIDATED` (Verified by `REF:`) | **PASS** 🟢 |
| **DG2** | `D-BEHAV` | `STAGE_S3_PREFIXES` trong `verify_governance.py` gồm 4 tiền tố `01`, `05`, `08`, `09`. | `tooling/verify_governance.py:56-57` | **`ACCEPT`** (`r2: 0.35`, `r3: 0.96`) | **PASS** 🟢 |
| **DG2** | `D-EDGE` | Các ngưỡng trong `laya_gate.py` là `DECISIVE_CONF=0.5`, `BLOCK_CONF=0.8`, `DEBATE_APPROVE_CONF=0.95`. | `tooling/laya_gate.py:44-46` | **`ACCEPT`** (`r2: 0.74`, `r3: 0.98`) | **PASS** 🟢 |
| **DG2** | `D-API` | `component_for` trong `agy_guard.py` đọc `.agent-project.json` và lấy component có prefix khớp dài nhất. | `hooks/agy/agy_guard.py:286-302` | **`ACCEPT`** (`r2: 0.47`, `r3: 0.94`) | **PASS** 🟢 |
