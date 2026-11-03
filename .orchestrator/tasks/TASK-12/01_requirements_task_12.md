---
task_key: TASK-12
suffix: task_12
title: //127.0.0.1:8090: LIVE-PROCESS-HUD-01, ROLE-MODEL-ATTRIBUTION-01, CO-ARCHITECT-CHAT-01, TASK-INTAKE-DUAL-MODE-01
target_repo: /Users/trainguyen/Documents/work/agent
affected_repos: [/Users/trainguyen/Documents/work/agent]
branch: task/TASK-12
worktree_path: /Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-12
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
# 📋 [S1 · UDF-v1] Đặc Tả Yêu Cầu & Phân Rã Sub-Tasks — TASK-12: //127.0.0.1:8090: LIVE-PROCESS-HUD-01, ROLE-MODEL-ATTRIBUTION-01, CO-ARCHITECT-CHAT-01, TASK-INTAKE-DUAL-MODE-01
> 📌 **Task:** `TASK-12` · **Stage:** `DONE` · **Tier/Mode:** `T2 / standard` · **Repos (1):** `agent` · **Model:** `opus` · **Gate S0:** `PASS 🟢`

## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)
- **Mục tiêu công việc:** //127.0.0.1:8090: LIVE-PROCESS-HUD-01, ROLE-MODEL-ATTRIBUTION-01, CO-ARCHITECT-CHAT-01, TASK-INTAKE-DUAL-MODE-01
- **Mô tả chi tiết:** //127.0.0.1:8090: LIVE-PROCESS-HUD-01, ROLE-MODEL-ATTRIBUTION-01, CO-ARCHITECT-CHAT-01, TASK-INTAKE-DUAL-MODE-01
- **Phạm vi Repository (1):** `agent` (Primary: `/Users/trainguyen/Documents/work/agent`)
- **Cô lập Git Worktree (Layer 1):** `/Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-12` (Branch: `task/TASK-12`)

## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`4 Sub-Tasks` · `file:line` & `DoD`)
| Mã Sub-Task | Wave | Repo | Điểm chạm (`file:line`) | Giải pháp & Tiêu chí Nghiệm thu (`DoD`) |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-12-M01`** | `Wave 0` | `agent` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`. (`DoD:` go test -race ./...) |
| **`TASK-12-M02`** | `Wave 0` | `agent` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` go test -race ./...) |
| **`TASK-12-M03`** | `Wave 1` | `agent` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) trong worktree cô lập của `agent`. (`DoD:` go test -race ./...) |
| **`TASK-12-M04`** | `Wave 1` | `agent` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (`def norm_ws(text: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` go test -race ./...) |

## 3. 💻 Code Demo / Cấu Hình Ràng Buộc Khởi Tạo (`S0 Boot`)
```yaml
task_key: TASK-12
affected_repos: [agent]
worktree: /Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-12
branch: task/TASK-12
runner_model: antigravity / opus
```

## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)
- **Hiến pháp & Laya (`:8200`):** `/Users/trainguyen/Documents/work/agent/GEMINI.md` · Laya Health: `false` · Harness²: `c1..c8 PASS (always-on=22661B <= 24576B)`
- **Topic Files định tuyến (`lessons/topics/`):** `01_governance_and_workflow.md` (Tổng: `350` bài học)
- **Lesson Tripwires (`G-08` & `H4t`):**
- Không có tripwire rủi ro cao bị kích hoạt (`tripwires.json` = CLEAN).
- **Bài học khớp trực tiếp (FTS5 BM25):**
- **`L040` [2026-10-26 · `T1:L278`] Omitt…:** (1) Dừng lại ngay lập tức (Rule #5), ghi bài học vào `lessons.md` và chạy `build_lessons_index.py`; (2) Mở vòng Re-Plan `Task 12` (`S1 -> S2 -> S3`) trong workspace `feature-modula...
- **`L011` [2026-10-27 · `T1:L74`] Bypas…:** Khi User [A] đối thoại hoặc chỉ đạo thay đổi kiến trúc trong khung **Co-Architect Studio (`/brain-delegate` · `Slot A` · `gemini-4-argon (antigravity)`)** trên `ai-orchestrator` [B...
- **`L027` [2026-10-27 · `T2:L187`] Stuff…:** Khi mở rộng một hệ thống điều phối đa tác nhân (`Multi-Agent Orchestrator` [A]) để bổ sung định tuyến model động (`Dynamic Model Routing`), chạy song song theo tầng `Wave DAG` bằng...
- **`L007` [2026-10-27 · `T2:L46`] -> Hậu quả [X]: Vi phạm kỷ luật Single-Responsibility & `NOW <= HEAD`, tạo rác trùng lặp khó bảo trì, để lọt nội dung mâu thuẫn (`Knex/PG Pool` & đè V1) ngay trong file vật lý `.orchestrator/tasks/TASK-13/09_tasks_solution.md`, và báo cáo hoàn thành khi tiến trình `:8090` vẫn chạy binary cũ [Y]. **Đúng:** (1) Mọi thay đổi bắt buộc đặt đúng layer và tách thành các module đơn trách nhiệm gọn nhẹ (`repo_tech_profile.go` chuyên trách normalize path & nhận diện Tech Stack/DoD/Fence dùng chung toàn package `webui`, `task_revise_workflow.go` chuyên trách luồng `/revise` & Clean-Room Spec, `subtask_git_apply.go <= HEAD 312 dòng`); (2) Xóa bỏ 100% câu chữ mẫu nhắc tới `Knex` hoặc sửa đè `trans-his.model.ts` trong `BuildUnifiedS3SolutionDoc`, normalize relative path ở mọi cột bảng Sub-Tasks, và chuẩn hóa `TestDoD` chính xác tuyệt đối (`npm run build && npm test` cho `package.json`, `go test -race ./...` cho `go.mod`) kèm test assertion nghiêm ngặt; (3) Kiểm chứng trực tiếp trên repo thật (`/Users/trainguyen/Documents/work/data-hub/centralized-export-service`), hồ sơ `.orchestrator/tasks/TASK-13` thật trên đĩa và trace giá trị JSON runtime thật trên tiến trình `:8090` đã nạp binary mới.:** (1) Dừng ngay theo Rule #5, ghi bài học vào `lessons.md`, kiểm tra `wc -l` (`2536 -> 2543`) và chạy `build_lessons_index.py`; (2) Thực hiện vòng lặp tự chữa lành `S5 -> S3 -> S4 ->...
- **`L029` [2026-10-26 · `T1:L201`] Arbit…:** Khi xây dựng giao diện điều phối tác vụ đa tác tử (`[A]` — `ai-orchestrator` Web UI trên `:8090`), nếu Agent chỉ tập trung thêm các nút kích hoạt chạy (`Start`, `Plan`, `Approve`, ...