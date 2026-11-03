---
tier: T2
suffix: audit_core_workflow_ex002_and_activity_log
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-27T14:23:05+07:00
status: PASS
laya_summary: 06_laya_summary_dg0_20261027T142253.json
---

# 10. Pre-Plan Adversarial Critique (`S2.5 DG0`) — Kiểm Định Hiện Trạng Core Workflow Của `ai-orchestrator`

| Mã Claim | Nội dung Phản biện & Đối chiếu Thực tế (`file:line`) | Bằng chứng Vật lý | Kết luận |
| :--- | :--- | :--- | :--- |
| **`C-REQ`** | Phản biện vì sao `EX-002` (ClickHouse V3) bị sinh Plan PostgreSQL V2: `isDualVersionV1V2Directive` và `buildSubTaskModuleTitleAndDesc` tại `internal/webui/task_revise_workflow.go:53-74` cùng `synthesizeDirectAnswerSummary` tại `internal/webui/task_llm_bridge.go:24-29` thấy chữ `V1` và `V2` trong mô tả của `EX-002` nên ép toàn bộ Sub-Tasks, Summary và Code Demo về `PostgresDB`. | `REF: internal/webui/task_revise_workflow.go:53-73`, `REF: internal/webui/task_llm_bridge.go:24-29` | `VERIFIED` |
| **`C-REALITY`** | Phản biện vì sao `DG0` và `Laya Gate` không chặn được Plan sai: `BuildUnifiedDG0CritiqueDoc` (`internal/webui/unified_doc_format.go:169-201`) chỉ in bảng tĩnh `VERIFIED / PASS`, còn `BuildAndRunCalibratedLayaGate` (`internal/agentgate/kernel_bridge.go:671-703`) đọc lại chính dòng đầu của file vừa ghi qua `pickLine(0)` để gán `Claim = q1`, `Evidence = ev1`, `MustContain = [q1]`. | `REF: internal/agentgate/kernel_bridge.go:671-703`, `.orchestrator/tasks/EX-002/06_laya_claims_dg0_ex_002.json` | `VERIFIED` |
| **`C-CONTRA`** | Phản biện vì sao không thấy `activity-log progress` và các chỗ làm giả ở `S4/S5`: `runTaskLifecycleAction` (`internal/webui/server.go:1122-1167`) chạy với `stepDelay = 0` và `defer clearPacedWorkflowStep(qt.ExternalID)` xóa ngay tiến trình trong `< 1ms`; `advancePacedWorkflowStep` (`internal/webui/telegram_bridge.go:2674-2721`) không đẩy tin nhắn vào Topic Telegram; `S4` không gọi LLM coder; `S5` (`internal/webui/server.go:1826, 1939`) chạy `test -f 01_requirements.md` để đúc biên lai `EV-...` giả và hardcode `masterMongoPool`. | `REF: internal/webui/server.go:1122-1167, 1826, 1939`, `REF: internal/webui/telegram_bridge.go:2674-2721` | `VERIFIED` |
| **`C-CORE003`** | Phản biện vì sao Task `core-003` (`audit lại`) bị trộn 4 Sub-Tasks hardcode của `Core-01` (`S03..S06`) và 2 Sub-Tasks ảo giác `gate.go` (`S01..S02`): (1) `isSelfUpgradeOrchestratorTask` (`internal/webui/task_inspector.go:980-995`) khớp chữ `"ui"` nằm trong tên hàm `BuildAndRunCalibratedLayaGate` và gọi `decomposeSelfUpgradeOrchestratorSubTasks` (`internal/webui/task_planner_multilayer.go:12-150`) trả về 6 Sub-Tasks cứng của `Core-01`; (2) `invokeArchitectLLM` (`internal/webui/task_llm_bridge.go:197, 202`) ép trả về đúng 2 Sub-Tasks và bị đoạn code mẫu trong đề bài đánh lừa trỏ vào `internal/agentgate/gate.go` (nơi hàm `extractMandatoryTechEntitiesFromRequirements` không hề tồn tại); (3) `applyLLMSubTaskSpecs` (`internal/webui/task_revise_workflow.go:75-107`) chỉ ghi đè 2 phần tử đầu (`S01, S02`, giữ nguyên `TestDoD` của `Core-01`) và bỏ lại 4 phần tử `S03..S06` của `Core-01`. | `REF: internal/webui/task_planner_multilayer.go:12-150`, `REF: internal/webui/task_inspector.go:980-1012`, `REF: internal/webui/task_revise_workflow.go:75-107`, `REF: internal/webui/task_llm_bridge.go:197-205` | `VERIFIED` |

