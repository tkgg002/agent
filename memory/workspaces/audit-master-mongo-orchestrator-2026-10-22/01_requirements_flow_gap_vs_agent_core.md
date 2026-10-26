---
project: ai-orchestrator
tier: T2
stage: S3
suffix: flow_gap_vs_agent_core
---

# 01 Requirements: Đối chiếu Chuyên sâu Toàn bộ Flow Thực thi Task (`Start -> Isolated Branch -> S0..S6`) của `ai-orchestrator` với Hệ thống Lõi `/agent`

## 1. Yêu cầu của User
1. `"mày xem lại cái flow làm tast từ start -> isolated branch, đối chiếu nó vào /agent của tao. còn thiếu bước nào không"`
2. **Chỉ đạo chấn chỉnh (Mid-Session Fix):** `"audit lại đi, đọc thật kỹ /agent "`

## 2. Phạm vi Đọc & Đối chiếu 100% Hệ Thống Lõi `/agent`
1. **Hệ quy chiếu Chuẩn (`/Users/trainguyen/Documents/work/agent/` — Đọc 100% các thành phần vật lý):**
   - `GEMINI.md` (`1–130`), `rules/GEMINI_FULL_ARCHIVE_2026-10.md` (`1–255`), `OPERATOR_MAP.md` (`1–34`).
   - `hooks/agy/agy_guard.py` (`1–1243` — đầy đủ 7 Lifecycle Hooks `H1..H7` + các chốt chặn nội bộ `H4a`, `H4b`, `H4c`, `H4t`, `G-01..G-08`, `Clean-Room Subagent Leak Guard`, `Claim-Evidence Source Fingerprint`, `No Shadow Files`, `Escalation Loop >= 3`).
   - `tooling/orchestrator/harness2_engine.py` (`1–1406` — `AgentSurfaceValidator c1..c8 <= 24KB`, `Orthogonal Proposer Axes INV vs DOM`, `Evidence ContextTree & assert_pre_flight_clean_room`, `Blind Anonymized Selection & Line-Jaccard Diversity`, `Two-Level Propose-Probe-Compose Recursion`).
   - `memory/global/lessons/INDEX.md` (`1–305`), `memory/global/lessons/tripwires.json` (`1–15`), `memory/global/harness_library/*/INDEX.md`.
   - `tooling/verify_governance.py` (`1–407`), `tooling/evidence_run.py` (`1–119`), `tooling/mutation_gate.py` (`1–400`), `tooling/laya_gate.py` (`1–687`).
2. **Hệ thống Đối chiếu (`/Users/trainguyen/Documents/work/ai-orchestrator/` — Cả 2 luồng Core & WebUI):**
   - **Luồng 1 — Core `TaskRunner`:** `internal/orchestrator/task_runner.go` (`1–1941`).
   - **Luồng 2 — WebUI `Server`:** `internal/webui/server.go` (`1–2482`), `internal/webui/task_inspector.go` (`1–2501`), `internal/webui/task_chat.go`, `internal/webui/telegram_bridge.go`.
   - **Cầu nối `/agent`:** `internal/agentgate/gate.go` (`1–374`).

## 3. Ràng buộc Kỷ luật (Propose-Only — Rule #1 & #13)
- Giữ nguyên **Propose-Only** (CẤM sửa source code trong `ai-orchestrator` hoặc `data-hub` khi chưa có `APPROVE` của anh).
- Mọi khẳng định bắt buộc gắn `file:line` thật của cả `/agent` và `ai-orchestrator`.
- Thiết kế **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** tại `09_tasks_solution_flow_gap_vs_agent_core.md` (kèm báo cáo phân tích đầy đủ tại `10_gap_analysis_flow_gap_vs_agent_core.md` và `13_analysis_flow_gap_vs_agent_core.md`), kiểm định qua `/laya_router` (`DG1/DG2` & `DG3`).
