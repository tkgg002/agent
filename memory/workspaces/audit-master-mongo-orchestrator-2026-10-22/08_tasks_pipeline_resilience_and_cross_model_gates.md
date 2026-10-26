---
tier: T2
suffix: pipeline_resilience_and_cross_model_gates
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T10:07:35+07:00
---

# 08. Tasks — Nâng cấp Multi-Agent Pipeline Resilience & Cross-Model Gates (`A..E`)

## Danh sách Task Thực thi (Sau khi có `APPROVE`)

- [x] `TASK-PIPE-01` — Đa dạng hóa Model theo Slot (`Cross-Model Auditing` — Khắc phục `Monoculture Bias` [Mục A]) (`EV-20261023T110647-c555`, `EV-20261023T110833-ea95`)
- [x] `TASK-PIPE-02` — Chốt chặn `Circuit Breaker (Max 3 Retries)` & Phân tách 2 tầng `Self-Healing` (`S5->S4->S5` vs `S5->S3->Human Gate` [Mục B]) (`EV-20261023T110647-c555`)
- [x] `TASK-PIPE-03` — Đưa `Worktree Provisioning (Slot C)` về trước `Implement (S4)` & Đổi Stage 6 thành `Finalize & Diff Review` [Mục C] (`EV-20261023T110633-d688`, `EV-20261023T110647-c555`)
- [x] `TASK-PIPE-04` — Cơ chế `Contract/Integration Sync Gate` & `Atomic Wave Rollback` cho `Multi-Repo Wave` [Mục D] (`EV-20261023T110647-c555`)
- [x] `TASK-PIPE-05` — Tách bạch Stage 5 thành `5A: Hard Deterministic Gate (Zero-LLM)` & `5B: Semantic Audit Gate (Cross-Model QA + DG3 + G1–G8)` [Mục E] (`EV-20261023T110647-c555`, `EV-20261023T110730-64f4`)

### `TASK-PIPE-01` — Đa dạng hóa Model theo Slot (`Cross-Model Auditing` — Khắc phục `Monoculture Bias` [Mục A])
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:758-941` (`buildInspectorPipelineStages`) & `internal/webui/task_inspector.go:1758-1860` (`buildSubTaskPipelineStages`): Hiện tại tất cả các Slot (`Slot A/G` Architect, `Slot B/D/E` Muscle Coder, `Slot H` Laya Gate, `Slot I` Security Guard, `Slot K` QA Auditor) đều đang fallback về cùng một model `gemini-4-argon (antigravity)` khi cấu hình slot chưa tách họ model.
  - `internal/webui/server.go:620-710` (khởi tạo `defaultCLISlots` / `loadCLISlotsConfig`): Đang gán mặc định tất cả các slot về cùng `gemini-4-argon (antigravity)`.
  - Cập nhật cấu hình mặc định và bộ kiểm tra `CrossModelAuditStatus`:
    - `Slot A / Slot G` (`BRAIN Architect`): `gemini-4-argon (antigravity)`
    - `Slot B / Slot D / Slot E` (`MUSCLE Coder`): `gemini-4-argon (antigravity)`
    - `Slot H` (`Laya Gate DG1/DG2/DG3`): `claude-sonnet-4-6 (antigravity)`
    - `Slot K` (`QA Auditor S5B`): `claude-opus-4-6 (antigravity)`
    - `Slot I` (`Security Guard S5B`): `codex-5.3 (antigravity)`
  - Bổ sung trường `CrossModelVerified bool` và `ModelDiversitySummary string` vào `TaskInspectorDTO` / `InspectorFlowStageDTO` để hiển thị rõ huy hiệu **`Cross-Model Audit: gemini-4-argon (Coder) ⇄ claude-opus-4-6 (QA) ⇄ claude-sonnet-4-6 (DG3)`** trên UI.
- **Logs / Test:**
  - `go test -v -race ./internal/webui/... -run TestPipelineResilienceAndCrossModelGates`
- **Definition of Done (DoD):**
  - Model của `QA Auditor (Slot K)` và `Laya Gate (Slot H)` khác họ model với `MUSCLE (Slot B/D/E)`; UI hiển thị rõ model family của từng Slot trên từng Stage card.

### `TASK-PIPE-02` — Chốt chặn `Circuit Breaker (Max 3 Retries)` & Phân tách 2 tầng `Self-Healing` (`S5->S4->S5` vs `S5->S3->Human Gate` [Mục B])
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:64-136` (`TaskInspectorDTO`, `InspectorFlowStageDTO`, `SubTaskDTO`) & `internal/webui/task_inspector.go:880-915`: Hiện tại nhánh `Self-Healing` chỉ có `LoopRoute: "S5 -> S3 (Re-plan/Demo) -> S4 (Re-code) -> S5 (Re-QA)"` mà thiếu giới hạn `MaxRetries` và chưa tách 2 tầng lùi.
  - `internal/webui/server.go:1470-1620` & `internal/webui/task_inspector.go:2280-2410`: Bổ sung bộ đếm `SelfHealRetryCount int`, `SelfHealMaxRetries int` (`= 3`), `CircuitBreakerTripped bool`, và `SelfHealTier string`:
    - **Tier-1 Implementation Self-Heal (`S5 -> S4 -> S5`):** Khi lỗi ở `5A Hard Gate` hoặc lỗi code ở `5B` (không đổi thiết kế `S3`), chỉ cho phép lặp `S5 -> S4 -> S5` tối đa `3` lần (`Retry 1/3 .. 3/3`).
    - **Tier-2 Architectural Re-Plan (`S5 -> S3 -> Human Gate NEEDS_YOU`):** Nếu phát hiện lỗi thiết kế đòi hỏi sửa `09_tasks_solution.md` (`S3`), bắt buộc ngắt về trạng thái `NEEDS_YOU` (`Human Gate Re-Approve Required`) để chống **Scope Drift**.
    - **Circuit Breaker Trip (`Retry > 3`):** Khi `SelfHealRetryCount >= 3` mà vẫn fail `5A` hoặc `5B (DG3)`, tự động mở cầu dao (`CircuitBreakerTripped = true`), dừng vòng lặp và chuyển về `NEEDS_YOU` kèm log `[Circuit Breaker OPEN: 3/3 Retries Exhausted -> Escalated to NEEDS_YOU]`.
- **Logs / Test:**
  - Unit & integration test kiểm chứng: (1) lỗi code lặp `S5 -> S4 -> S5` tăng `SelfHealRetryCount` từ `1/3 -> 3/3` và tự ngắt về `NEEDS_YOU` ở lần thứ 4; (2) lỗi sửa `S3 Plan` lập tức yêu cầu `Human Gate (NEEDS_YOU)`.
- **Definition of Done (DoD):**
  - Không còn vòng lặp `S5 -> S3 -> S4 -> S5` tự chạy ngầm qua mặt `Human Gate`; bộ đếm `0/3 Retries (Max 3 · Circuit Breaker)` hiển thị trực tiếp trên Stage 5 card của cả Task và Sub-Task.

### `TASK-PIPE-03` — Đưa `Worktree Provisioning (Slot C)` về trước `Implement (S4)` & Đổi Stage 6 thành `Finalize & Diff Review` [Mục C]
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:845-941` (`buildInspectorPipelineStages`) & `internal/webui/task_inspector.go:1805-1860` (`buildSubTaskPipelineStages`):
    - Tích hợp `Slot C (Worktree Provisioner)` vào ngay đầu bước **`4. Implement (S4 · Worktree Slot C)`** (hiển thị rõ `Worktree Attached: task/<id>@<sha>` ngay ở Stage 4 và trên thanh Header của Task/Sub-Task).
    - Đổi tên Stage 6 từ `Isolated Branch` thành **`Finalize & Diff Review`** (Subtitle: `Deliverable Ready · Diff Viewer & Merge Gate · No-Push (H5)`), phản ánh đúng bản chất bước 6 là nghiệm thu bản `git diff` cuối cùng tại worktree đã tạo từ bước 4.
  - `internal/webui/static/app.js:940-1035` (`renderFlowStageGrid`): Hiển thị huy hiệu `🌿 Worktree Attached` ngay trên Header và Stage 4 (`Implement S4`), đồng thời cập nhật nút/nhãn ở Stage 6 thành `Finalize & Diff Review`.
- **Logs / Test:**
  - Kiểm tra JSON `/api/tasks/<key>/inspector` và DOM render trên `:8090`.
- **Definition of Done (DoD):**
  - `Slot C (Worktree Provisioner)` gắn đúng vị trí vật lý tại đầu `Implement (S4)` và Header; Stage 6 mang tên `Finalize & Diff Review`.

### `TASK-PIPE-04` — Cơ chế `Contract/Integration Sync Gate` & `Atomic Wave Rollback` cho `Multi-Repo Wave` [Mục D]
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:1645-1755` (`populateSubTaskFractalContractsAndArtifacts`) & `internal/webui/task_inspector.go:2280-2410` (`runSubTaskLifecycleAction`):
    - Bổ sung **`VerifyWaveContractSync(subTasks []SubTaskDTO, wave int)`**: Kiểm tra mọi Sub-Task trong `Wave` có `OutputContract.ExportedSymbols` và `TargetRepo` khớp với `InputContract.RequiredFrom` của các Sub-Task phụ thuộc trước khi kết thúc Wave.
    - Bổ sung **`RollbackWaveWorktrees(taskDir string, subTasks []SubTaskDTO, wave int, reason string)`**: Ghi nhận `WaveRestorePointSHA` cho tất cả Sub-Tasks thuộc cùng `Wave`; nếu bất kỳ Sub-Task nào trong `Wave` bị `CircuitBreakerTripped` (thất bại quá `3/3` lần tự chữa ở `5A/5B`), tự động rollback đồng loạt trạng thái và worktree của toàn bộ Sub-Tasks trong `Wave` đó về `WaveRestorePointSHA` (`Stage = "NEEDS_YOU"`, `WaveRollbackTriggered = true`).
- **Logs / Test:**
  - Test kịch bản `Wave 1` có 2 Sub-Tasks (`TASK-M02`, `TASK-M03`): `TASK-M02` đã `DONE`, nhưng `TASK-M03` chạm `Circuit Breaker (3/3 Retries)` -> `RollbackWaveWorktrees` tự động đưa cả `Wave 1` (`TASK-M02` và `TASK-M03`) về `WaveRestorePointSHA` để không lệch trạng thái giữa các worktree.
- **Definition of Done (DoD):**
  - `Contract Sync Gate` và `Atomic Wave Rollback` hoạt động xác định, có biên bản log và hiển thị trạng thái trên Stage 4 (`Implement S4`).

### `TASK-PIPE-05` — Tách bạch Stage 5 thành `5A: Hard Deterministic Gate (Zero-LLM)` & `5B: Semantic Audit Gate (Cross-Model QA + DG3 + G1–G8)` [Mục E]
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:880-915` (`buildInspectorPipelineStages`), `internal/webui/task_inspector.go:1825-1845` (`buildSubTaskPipelineStages`), và `internal/webui/static/app.js:975-1025`:
    - Cấu trúc hóa Stage 5 (`Local QA & DG3`) thành 2 cổng tuần tự tách biệt rõ ràng trong `InspectorFlowStageDTO` (`HardGateStatus`, `HardGateCmds`, `HardGateZeroLLM: true`, `SemanticGateStatus`, `SemanticGateModel`, `SemanticGateSkippedReason`):
      1. **`5A. Hard Deterministic Gate (Zero-LLM)`**: Chạy `go vet ./...`, `golangci-lint`, `go test -race`, `go build`. Nếu `5A` fail -> trả thẳng `stdout/stderr` về `MUSCLE (S4)` (`Tier-1 Self-Heal S5A -> S4`), **`SemanticGateStatus = "SKIPPED_BY_HARD_GATE"` (tốn `0` LLM call)**.
      2. **`5B. Semantic Audit Gate (Cross-Model QA + DG3 + G1–G8 + Security Guard)`**: Chỉ kích hoạt khi `5A == PASS`. Sử dụng `claude-opus-4-6` (`QA Auditor`) + `claude-sonnet-4-6` (`Laya DG3`) + `codex-5.3` (`Security Guard`).
- **Logs / Test:**
  - Unit test & live HTTP trace xác nhận khi `5A Hard Gate` fail thì `5B Semantic Audit Gate` không bị gọi (`0 LLM Auditor calls`) và lập tức kích hoạt `Tier-1 Self-Heal (S5A -> S4)`.
- **Definition of Done (DoD):**
  - Stage 5 hiển thị rõ 2 tầng `5A Hard Gate (Zero-LLM)` và `5B Semantic Audit Gate (Cross-Model DG3 + G1–G8)` trên cả Backend DTO và Frontend UI.
