# 08 Tasks: Kiến Trúc Fractal Sub-Task Lifecycle (`Sub-Task Là Task Độc Lập`: Đầu Vào -> Plan Riêng -> `Start -> Isolated Branch` -> Đầu Ra)

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `subtask_fractal_lifecycle`
- **Tier:** `T2`

---

## Danh mục Task

- [x] **TASK-SUB-01 (S1/S2 Investigation — Phân tích hiện trạng `SubTaskDTO` và luồng thực thi Sub-Task trong `ai-orchestrator`):**
  - **Context (`file:line`):** `internal/webui/task_inspector.go:34-54, 1082-1925`, `internal/webui/server.go:180-258, 1033-1686`, `internal/webui/static/index.html:180-465`, `internal/webui/static/app.js:900-1420`.
  - **Mô tả:** Ghi nhận bài học Rule #5 vào `lessons.md` (`2198 -> 2205` dòng, `285/285` entries), khảo sát chi tiết cấu trúc `SubTaskDTO`, `generateDeepTaskArtifacts`, `runTaskLifecycleAction` và giao diện UI để xác định toàn bộ điểm cần nâng cấp nhằm biến mỗi `Sub-Task` thành một `Task` độc lập có `InputContract`, `OutputContract`, `Dedicated Plan`, và luồng `Start -> Isolated Branch` riêng.
  - **Logs/Test:** Chỉ đọc (`view_file`), kiểm chứng `REF: path:start-end`.
  - **Definition of Done:** Hoàn tất `13_analysis_subtask_fractal_lifecycle.md`.

- [x] **TASK-SUB-02 (S3 Solution & Code Demo + Kiểm định `/laya_router` DG1/DG2):**
  - **Context (`file:line`):** `09_tasks_solution_subtask_fractal_lifecycle.md`, `13_analysis_subtask_fractal_lifecycle.md`.
  - **Mô tả:** Thiết kế phương án kỹ thuật duy nhất & tối ưu nhất cho **Fractal Sub-Task Execution Architecture** (Backend + Frontend + Wave I/O Chaining), chạy `laya_gate.py run --stage dg1` đạt `PASS`, giữ nguyên **Propose-Only** trình anh duyệt.
  - **Logs/Test:** `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1` (`gate: "PASS"`).
  - **Definition of Done:** `DG1/DG2 PASS`, `verify_governance.py --stage S3` `PASSED 🟢`.

- [x] **TASK-SUB-03 (S4/S5 Execution — Thực thi Fractal Sub-Task Lifecycle trên `ai-orchestrator` SAU KHI có lệnh `APPROVE` của anh):**
  - **Context (`file:line`):** `internal/webui/task_inspector.go`, `internal/webui/server.go`, `internal/webui/static/index.html`, `internal/webui/static/app.js`, `internal/webui/static/app.css`, `internal/webui/server_test.go`.
  - **Mô tả:** Sau khi anh `APPROVE`, thực thi:
    1. Mở rộng `SubTaskDTO` bổ sung `Stage` (`READY|PLANNING|NEEDS_YOU|APPROVED|RUNNING|IN_REVIEW|DONE`), `BranchName` (`task/<parentKey>/<subTaskID>`), `ArtifactDir` (`.orchestrator/tasks/<parentKey>/subtasks/<subTaskID>`), `InputContract` (`SourceContext`, `UpstreamSubTasks`, `UpstreamOutputs`, `InputSchema`), `OutputContract` (`Deliverables`, `DownstreamSubTasks`, `OutputSchema`, `EvidenceReceiptID`), `PlanMarkdown`, `CodeDemo`, `DiffPreview`, `AuditSummary`, và `PipelineStages` (6 bước `Start -> Plan -> You -> Implement -> Local QA & DG3 -> Isolated Branch` riêng của Sub-Task).
    2. Xây dựng bộ sinh hồ sơ vật lý riêng cho từng Sub-Task (`ensureSubTaskArtifactsAndContracts`) và bộ thực thi vòng đời Sub-Task (`POST /api/subtasks/action` hoặc `sub_task_id` trong `/api/tasks/action`: hỗ trợ chạy `select`, `plan`, `approve`, `approve_and_run`, `implement`, `qa`, `autopilot` cho từng Sub-Task độc lập, kèm cơ chế **Wave I/O Chaining**: khi Sub-Task ở Wave $k$ đạt `DONE`, `OutputContract` của nó tự động được bơm vào `InputContract.UpstreamOutputs` của các Sub-Task phụ thuộc ở Wave $k+1$).
    3. Nâng cấp giao diện Web UI (`index.html`, `app.js`, `app.css`): khi bấm chọn bất kỳ Sub-Task nào (ở bảng Sub-Tasks hoặc ở tầng 3 `Sub-Tasks Dependency DAG`), hiển thị thanh **Sub-Task Execution Inspector (`Start -> Plan -> You -> Implement -> Local QA -> Isolated Branch`)** kèm 3 thẻ hợp đồng **📥 Đầu vào (`Input Contract`)**, **💡 Plan & Code Demo Riêng (`09_tasks_solution_<subTaskID>.md`)**, **📤 Đầu ra (`Output Contract` + `Branch` + `EV-...`)** và các nút chạy từng bước cho riêng Sub-Task đó!
  - **Logs/Test:** `evidence_run.py` (`go vet ./...`, `go test -v -race ./internal/webui/...`, `go run ./cmd/orchestrator daemon --once`, live `:8090` runtime check) + `laya_gate.py --stage dg3` + `verify_governance.py --stage S6`.
  - **Definition of Done:** Toàn bộ 10+ Sub-Tasks của một Task đều có Đầu vào, Đầu ra, Plan riêng, thư mục hồ sơ riêng và chạy được luồng `Start -> Isolated Branch` độc lập hoặc theo chuỗi Wave DAG.

- [x] **TASK-SUB-04 (S5 -> S3 -> S4 -> S5 Adversarial Self-Healing Loop — Khắc phục 5 lỗ hổng phản biện `GAP-FRAC-AUDIT-01 .. GAP-FRAC-AUDIT-05`):**
  - **Context (`file:line`):** `internal/webui/task_inspector.go:1614-2470`, `internal/webui/server.go:1544-1565`, `internal/webui/server_test.go:1380-1630`.
  - **Mô tả:**
    1. **`GAP-FRAC-AUDIT-01` (`internal/webui/task_inspector.go:2330-2470`):** Trong `runSubTaskLifecycleAction`, cưỡng chế 3 cổng cứng trước khi cho phép chuyển sang `IN_REVIEW` hoặc `DONE`: (a) **Upstream Wave Dependency Gate** (`st.InputContract.UpstreamReady == false` với `WaveIndex >= 1` -> chặn thực thi `subtask_implement`/`subtask_qa`/`subtask_autopilot` Phase 2, giữ ở `NEEDS_YOU` kèm log `[Wave Dependency Gate BLOCK]`); (b) **H4b Approve Gate** (`.user-approved` của Sub-Task hoặc Task cha bắt buộc tồn tại trước khi `subtask_implement`/`subtask_qa`); (c) **Real On-Disk Verification Gate** (`VerifySubTaskCodeOnDisk(ctx, *st)` bắt buộc chạy và ghi biên lai vật lý `06_evidence/EV-SUB-<safeSubID>.json` trước khi đóng `DONE`).
    2. **`GAP-FRAC-AUDIT-02` (`internal/webui/task_inspector.go:1707-1715` & `:1925-1980` & `:2060-2085`):** Xóa bỏ việc gán chuỗi tĩnh `"EV-SUB-" + st.SubTaskID + "-PASS"` không có file thật; tạo hàm `writeSubTaskPhysicalEvidenceReceipt` ghi biên lai JSON thật vào `.orchestrator/tasks/<safeParent>/subtasks/<safeSubID>/06_evidence/EV-SUB-<safeSubID>.json` kèm `src_fingerprint` SHA-256 tính từ file thật trong `st.TargetRepo`. Xóa bỏ việc sinh 4 dòng comment `diff --git` giả trong `extractSubTaskDiffFromParentPatch`; thay bằng kiểm tra `git diff` thật hoặc trích xuất đoạn code thật đã verify trên đĩa kèm header trung thực.
    3. **`GAP-FRAC-AUDIT-03` (`internal/webui/task_inspector.go:1982-2213`):** Xóa bỏ hàm stub rỗng `func ExecuteSubTask_...(ctx context.Context) error { return nil }` ở nhánh `default:` của `buildDedicatedSubTaskCodeDemo`; viết hàm `readSubTaskRealSourceSnippet(st SubTaskDTO)` đọc trực tiếp dòng code thật trên đĩa từ `st.TargetRepo` + `st.ContextFileLine` (`file:line`) để mọi Sub-Task (cả trong và ngoài `TASK-M01..M10`) đều có Code Demo và `13_analysis.md` trích xuất từ source code thật 100%.
    4. **`GAP-FRAC-AUDIT-04` (`internal/webui/task_inspector.go:1835-1910`):** Bổ sung đầy đủ bộ hồ sơ vật lý chuẩn `/agent` Tier T2 cho từng Sub-Task tại `.orchestrator/tasks/<safeParent>/subtasks/<safeSubID>/`: `01_requirements.md` (có YAML frontmatter `tier: T2`), `08_tasks.md` (checklist & DoD riêng), `13_analysis.md` (bảng `VERIFIED` vs `HYPOTHESIS` + snippet code thật từ `TargetRepo`), `09_tasks_solution.md`, `audit_report.md` (đủ 6 mục S5 Adversarial Audit), và `06_evidence/EV-SUB-<safeSubID>.json`.
    5. **`GAP-FRAC-AUDIT-05` (`internal/webui/server.go:1544-1565`):** Tại nhánh `case "qa"` của Task cha, cập nhật đồng bộ `subTasks[i].Stage = "DONE"`, ghi `.subtask-stage`, và gọi `subTasks = s.populateSubTaskFractalContractsAndArtifacts(*qt, subTasks)` trước khi `saveTaskSubTasks`.
  - **Logs/Test:** `evidence_run.py` (`EV-20261023T095428-9c30`, `EV-20261023T095441-a34f`, `EV-20261023T095550-3ea0`, `EV-20261023T095618-46d6`, `EV-20261023T095631-254d`) + `laya_gate.py run --stage dg1` (`06_laya_summary_dg1_20261023T094028.json` = `PASS`) & `--stage dg3` (`06_laya_summary_dg3_20261023T095823.json` = `PASS`) + `verify_governance.py --stage S6` (`PASSED 🟢`).
  - **Definition of Done:** 5/5 lỗ hổng phản biện được đóng kín 100%, test âm tính chặn `Wave 1` khi `Wave 0` chưa `DONE` và chặn `H4b` khi thiếu `.user-approved` PASS, biên lai vật lý `EV-SUB-*.json` tồn tại trên đĩa và trên live `:8090`.
