# 08 Tasks: Đối chiếu & Đồng bộ 100% Flow Thực thi Task (`Start -> Isolated Branch -> S0..S6`) của `ai-orchestrator` với Hệ Thống Lõi `/agent`

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `flow_gap_vs_agent_core`
- **Tier:** `T2`

---

## Danh mục Task

- [x] **TASK-GAP-01 (S2 Deep Audit — Đọc 100% mã nguồn `/agent` và đối chiếu từng dòng với cả 2 luồng `TaskRunner` & `WebUI Server` của `ai-orchestrator`):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/agent/GEMINI.md:1-130`, `/Users/trainguyen/Documents/work/agent/rules/GEMINI_FULL_ARCHIVE_2026-10.md:1-255`, `/Users/trainguyen/Documents/work/agent/OPERATOR_MAP.md:1-34`, `/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:1-1243`, `/Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py:1-1406`, `/Users/trainguyen/Documents/work/agent/memory/global/lessons/tripwires.json:1-15`, `/Users/trainguyen/Documents/work/agent/tooling/{verify_governance.py:1-407,evidence_run.py:1-119,mutation_gate.py:1-400,laya_gate.py:1-687}`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go:1-1941`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/{server.go:1-2482,task_inspector.go:1-2501}`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/gate.go:1-374`.
  - **Mô tả:** Thực hiện Rule #5 Mid-Session Fix (ghi bài học vào `lessons.md` + chạy `build_lessons_index.py`), sau đó đọc 100% các thành phần vật lý của `/agent` và đối chiếu toàn diện 14 điểm đứt gãy (`GAP-FLOW-01` → `GAP-FLOW-14`) trên cả 2 luồng `TaskRunner` và `WebUI Server`.
  - **Logs/Test:** Chỉ đọc (`view_file`), kiểm chứng `REF: path:start-end` trên đĩa.
  - **Definition of Done:** Hoàn tất `10_gap_analysis_flow_gap_vs_agent_core.md` và `13_analysis_flow_gap_vs_agent_core.md` với đầy đủ 14 điểm đối chiếu kèm `file:line`.

- [x] **TASK-GAP-02 (S3 Solution & Code Demo + Kiểm định `/laya_router` DG1/DG2/DG3):**
  - **Context (`file:line`):** `09_tasks_solution_flow_gap_vs_agent_core.md`, `10_gap_analysis_flow_gap_vs_agent_core.md`, `13_analysis_flow_gap_vs_agent_core.md`.
  - **Mô tả:** Thiết kế phương án kỹ thuật duy nhất & tối ưu nhất đóng kín cả 14 lỗ hổng (`GAP-FLOW-01` → `GAP-FLOW-14`), chạy `laya_gate.py` đạt `PASS` cho `DG1`, `DG2` và `DG3`, giữ nguyên **Propose-Only** trình anh duyệt.
  - **Logs/Test:** `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1` & `--stage dg3` (`gate: "PASS"`).
  - **Definition of Done:** `DG1/DG2/DG3 PASS`, `verify_governance.py --stage S3` `PASSED 🟢`.

- [x] **TASK-GAP-03 (S4/S5 Execution — Thực thi đóng 14 Gap trong `ai-orchestrator` SAU KHI có lệnh `APPROVE` của anh):**
  - **Context (`file:line`):** `internal/agentgate/kernel_bridge.go:1-897`, `internal/agentgate/gate.go:237-303`, `internal/webui/server.go:441-579, 1080-1686`, `internal/webui/task_inspector.go:1718-1950`, `internal/webui/telegram_bridge.go:926-1010`, `internal/webui/server_test.go:1179-1380`.
  - **Mô tả:** Sau khi anh `APPROVE`, tích hợp đầy đủ: (1) `S0 Boot` (`Laya :8200` healthcheck, `Harness² validate-surface c1..c8`, `harness_library/*/INDEX.md`, `tripwires.json G-08`, `memory/global/project/<proj>`, `topics/0X_*.md`); (2) `Per-Repo Git Worktree & Restore-Point` (`restore_points.json`); (3) `S1` (`05_progress_<suffix>.md` Append-Only ISO-8601 `+07:00`, `04_decisions_<suffix>.md`, Tier `T0..T3` & `mode: clean-room`); (4) `S2` (`H4t Tripwire Evidence Check`, bảng `VERIFIED vs HYPOTHESIS`, `Gate G2 Red -> Green Reproduce`); (5) `S3` (`## Clean-Room Approved Spec` + `spec_rule_issues G-03`, sinh `06_laya_claims_dg1_<suffix>.json` có `K1,K2,K3` và chạy thật `laya_gate.py run --stage dg1`); (6) `S4` (`Clean-Room Pre-Flight & Subagent Leak Guard G-01..G-04`, đọc `<repo>/.agent-project.json`, chạy `evidence_run.py` lấy biên lai `EV-...json` kèm `src_fingerprint` + `H4c Linter`, chạy `mutation_gate.py --base <restore_ref>`); (7) `S5/S6` (vòng lặp tự chữa lành `S5 -> S3 -> S4 -> S5` tối đa 3 vòng không bắt User `APPROVE` lần 2 + `H6 Escalation >= 3 loops -> harness2_engine.py run-recursion`, chạy `laya_gate.py run --stage dg3` Mode A + `review` Mode B với `mtime >= last_src_edit`, đồng bộ sang `/agent/memory/workspaces/<ws>/`, append `DONE` vào `05_progress` và chạy `verify_governance.py --stage S6`).
  - **Logs/Test:** `evidence_run.py --kind unit/runtime` (`EV-20261023T040743-eb9a`, `EV-20261023T040805-fe56`, `EV-20261023T040840-1f36`, `EV-20261023T040842-3607`, `EV-20261023T040846-8f91`, `EV-20261023T040951-bd0a`) + `laya_gate.py --stage dg3` + `verify_governance.py`.
  - **Definition of Done:** Toàn bộ 14 Gap được đóng kín, 100% test & runtime PASS có biên lai `EV-...`.


