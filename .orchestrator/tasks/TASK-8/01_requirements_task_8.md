---
task_key: TASK-8
suffix: task_8
title: S3 Re-Plan — Hợp nhất 1 Khung Ứng Dụng Duy Nhất Single Unified IDE Shell, Xây dựng Bộ Máy Thực Thi Tự Động Per-Task Multi-Stage Autopilot Core Engine & Cô lập Vật lý 4 Lớp Git Worktree + Task Dir + Terminal + Scoped Autopilot
target_repo: /Users/trainguyen/Documents/work/agent
affected_repos: [/Users/trainguyen/Documents/work/agent]
branch: task/TASK-8
worktree_path: /Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-8
cli_runner: antigravity
model: opus
stage: DONE
tier: T2
mode: standard
laya_health_s0: false
harness_surface_s0: c1..c8 PASS (always-on=22661B <= 24576B)
harness_libraries: [agent_gov, agent_governance]
routed_topics: [01_governance_and_workflow.md]
matched_tripwires: []
kernel_constitution: /Users/trainguyen/Documents/work/agent/GEMINI.md
total_lessons_indexed: 350
---
# 📋 [S1 · UDF-v1] Đặc Tả Yêu Cầu & Phân Rã Sub-Tasks — TASK-8: S3 Re-Plan — Hợp nhất 1 Khung Ứng Dụng Duy Nhất Single Unified IDE Shell, Xây dựng Bộ Máy Thực Thi Tự Động Per-Task Multi-Stage Autopilot Core Engine & Cô lập Vật lý 4 Lớp Git Worktree + Task Dir + Terminal + Scoped Autopilot
> 📌 **Task:** `TASK-8` · **Stage:** `DONE` · **Tier/Mode:** `T2 / standard` · **Repos (1):** `agent` · **Model:** `opus` · **Gate S0:** `PASS 🟢`

## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)
- **Mục tiêu công việc:** S3 Re-Plan — Hợp nhất 1 Khung Ứng Dụng Duy Nhất Single Unified IDE Shell, Xây dựng Bộ Máy Thực Thi Tự Động Per-Task Multi-Stage Autopilot Core Engine & Cô lập Vật lý 4 Lớp Git Worktree + Task Dir + Terminal + Scoped Autopilot
- **Mô tả chi tiết:** S3 Re-Plan — Hợp nhất 1 Khung Ứng Dụng Duy Nhất Single Unified IDE Shell, Xây dựng Bộ Máy Thực Thi Tự Động Per-Task Multi-Stage Autopilot Core Engine & Cô lập Vật lý 4 Lớp Git Worktree + Task Dir + Terminal + Scoped Autopilot
- **Phạm vi Repository (1):** `agent` (Primary: `/Users/trainguyen/Documents/work/agent`)
- **Cô lập Git Worktree (Layer 1):** `/Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-8` (Branch: `task/TASK-8`)

## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`4 Sub-Tasks` · `file:line` & `DoD`)
| Mã Sub-Task | Wave | Repo | Điểm chạm (`file:line`) | Giải pháp & Tiêu chí Nghiệm thu (`DoD`) |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-8-M01`** | `Wave 0` | `agent` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _extract_yaml_frontmatter(src: str) -> str:`) trong worktree cô lập của `agent`. (`DoD:` go test -race ./...) |
| **`TASK-8-M02`** | `Wave 0` | `agent` | `agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`analyze_dependencies.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (`import os`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` go test -race ./...) |
| **`TASK-8-M03`** | `Wave 1` | `agent` | `agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`analyze_dependencies.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (`import os`) trong worktree cô lập của `agent`. (`DoD:` go test -race ./...) |
| **`TASK-8-M04`** | `Wave 1` | `agent` | `agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)** — Kết nối dây dẫn nghiệp vụ tại `tooling/evidence_run.py` (`def _git(repo: Path, *args: str) -> Optional[str]:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` go test -race ./...) |

## 3. 💻 Code Demo / Cấu Hình Ràng Buộc Khởi Tạo (`S0 Boot`)
```yaml
task_key: TASK-8
affected_repos: [agent]
worktree: /Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-8
branch: task/TASK-8
runner_model: antigravity / opus
```

## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)
- **Hiến pháp & Laya (`:8200`):** `/Users/trainguyen/Documents/work/agent/GEMINI.md` · Laya Health: `false` · Harness²: `c1..c8 PASS (always-on=22661B <= 24576B)`
- **Topic Files định tuyến (`lessons/topics/`):** `01_governance_and_workflow.md` (Tổng: `350` bài học)
- **Lesson Tripwires (`G-08` & `H4t`):**
- Không có tripwire rủi ro cao bị kích hoạt (`tripwires.json` = CLEAN).
- **Bài học khớp trực tiếp (FTS5 BM25):**
- **`L053` [2026-10-23 · `T1:L369`] Viết…:** Khi thiết kế và trình bày một hệ thống nền tảng điều phối đa tác nhân chạy ngầm theo hàng đợi (`Event/Queue-Driven Multi-Agent Orchestrator` [A]) có tích hợp bộ quy tắc quản trị hi...
- **`L020` [2026-10-27 · `T1:L137`] S5:** Khi xây dựng luồng thực thi tự động (`S1 -> S2 -> DG0 -> S3 -> APPROVE -> S4 -> S5 -> DG3 -> DONE`) cho hệ thống Orchestrator [A] quản lý nhiều loại Task (Single-Repo, Multi-Repo, ...
- **`L027` [2026-10-27 · `T2:L187`] Stuff…:** Khi mở rộng một hệ thống điều phối đa tác nhân (`Multi-Agent Orchestrator` [A]) để bổ sung định tuyến model động (`Dynamic Model Routing`), chạy song song theo tầng `Wave DAG` bằng...
- **`L029` [2026-10-26 · `T1:L201`] Arbit…:** Khi xây dựng giao diện điều phối tác vụ đa tác tử (`[A]` — `ai-orchestrator` Web UI trên `:8090`), nếu Agent chỉ tập trung thêm các nút kích hoạt chạy (`Start`, `Plan`, `Approve`, ...
- **`L044` [2026-10-26 · `T1:L306`] Stuff…:** Khi hiện thực hóa quy trình xây dựng lại phòng sạch (`Clean-Room Rebuild Protocol` kiểu REA — *"Model thấy code gốc thì không bao giờ được viết code, và model viết code thì không b...