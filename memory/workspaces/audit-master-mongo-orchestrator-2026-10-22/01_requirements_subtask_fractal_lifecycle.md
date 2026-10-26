---
workspace: audit-master-mongo-orchestrator-2026-10-22
suffix: subtask_fractal_lifecycle
tier: T2
mode: standard
target_repo: /Users/trainguyen/Documents/work/ai-orchestrator
---
# 01 Requirements: Kiến Trúc Fractal Sub-Task Lifecycle (`Sub-Task Là Task Độc Lập`: Đầu Vào -> Plan Riêng -> `Start -> Isolated Branch` -> Đầu Ra)

## 1. Bối Cảnh & Mệnh Lệnh Của Anh
- **Nguyên văn yêu cầu:** *"folow cốt lõi, ví dụ: đưa task vào, phân rã thành 10 sub task đi, thì bản chất subtask nó là task và nó sẽ có cái quy trình thực hiện task của riêng nó, nó có đầu vào, đầu ra, plan để giải quyết riêng cho nó. luồng làm task từ Start -> Isolated Branch"*
- **Vấn đề hiện tại trong `ai-orchestrator` (`VERIFIED`):**
  1. Tại [`internal/webui/task_inspector.go:34-54`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go#L34-L54), `SubTaskDTO` hiện chỉ là một phần tử checklist thụ động (`Status: "pending" | "done"`) nằm trong mảng `subtasks.json` của Task cha (`TASK-11`).
  2. Từng `Sub-Task` (`TASK-M01 .. TASK-M10` hoặc `TASK-A04 .. TASK-A07`) **chưa có**:
     - **Đầu vào (`InputContract`):** Nhận dữ kiện gì từ bài toán gốc, từ `ContextFileLine`, và nhận kết quả đầu ra (`UpstreamOutputs`) gì từ các `Sub-Task` phụ thuộc ở Wave trước (`DependsOn`).
     - **Đầu ra (`OutputContract`):** Sản phẩm bàn giao cụ thể của Sub-Task đó là gì (file/hàm/endpoint/schema tạo mới hoặc cập nhật, đường dẫn `diff.patch` riêng, mã biên lai `EV-...` riêng, trạng thái bàn giao cho Wave kế tiếp).
     - **Plan & Hồ sơ giải quyết riêng (`SubTaskArtifactDir`):** Chưa có thư mục hồ sơ riêng `.orchestrator/tasks/<parentKey>/subtasks/<subTaskID>/` chứa `01_requirements_<subTaskID>.md` (Đầu vào & DoD riêng), `13_analysis_<subTaskID>.md` (Phân tích `file:line` riêng), `09_tasks_solution_<subTaskID>.md` (Plan & Code Demo riêng của Sub-Task đó), `diff.patch` riêng, và `audit_report_<subTaskID>.md` riêng.
     - **Quy trình thực thi `Start -> Plan (S1–S3) -> You (Approve) -> Implement (S4) -> Local QA & DG3 (S5) -> Isolated Branch (S6)` riêng cho từng Sub-Task:** Chưa có trường `Stage` (`READY -> PLANNING -> NEEDS_YOU -> APPROVED -> RUNNING -> IN_REVIEW -> DONE`), `BranchName` (`task/<parentKey>/<subTaskID>`), `WorktreePath` riêng, và các nút điều khiển/xem chi tiết luồng `Start -> Isolated Branch` của từng Sub-Task trên giao diện Web UI (`1. Autopilot & Report` & `2. Wave 0..N Dependency DAG`).

## 2. Yêu Cầu Kiến Trúc Cần Đạt
1. **Mô hình Hóa `Sub-Task` Là Một `Task` Độc Lập (`First-Class Fractal Task`):**
   - Mỗi `Sub-Task` (`TASK-M01 .. TASK-M10`) có đầy đủ:
     - **Đầu vào (`InputContract`):** `SourceContext` (`file:line`), `UpstreamDeps` (danh sách `DependsOn` Sub-Task IDs + trạng thái đã nhận `OutputContract` từ Wave trước hay chưa), `InputSpec` (dữ liệu/struct/API đầu vào).
     - **Plan giải quyết riêng (`Dedicated Plan & Code Demo`):** Hồ sơ vật lý riêng tại `.orchestrator/tasks/<parentKey>/subtasks/<subTaskID>/` gồm `01_requirements.md`, `13_analysis.md`, `09_tasks_solution.md` (Plan + Code Demo riêng cho đúng Sub-Task đó), `diff.patch` riêng, `audit_report.md` riêng, `05_progress.md` riêng.
     - **Đầu ra (`OutputContract`):** `Deliverables` (symbols/files/endpoints tạo ra), `DownstreamConsumers` (các Sub-Task ở Wave sau sẽ dùng đầu ra này), `DiffFile`, `AuditFile`, `EvidenceReceiptID` (`EV-...`).
     - **Vòng đời `Start -> Isolated Branch` riêng (`Stage` & `PipelineStages` 6 bước):**
       1. `Start · Intake & Input Contract` (`S0/S1`)
       2. `Plan · S2 Scan & S3 Sub-Task Plan + DG1/DG2` (`PLANNING`)
       3. `You · Sub-Task Plan Review / Approve` (`NEEDS_YOU` -> `APPROVED`)
       4. `Implement · S4 Isolated Sub-Task Worktree` (`RUNNING`)
       5. `Local QA & DG3 · S5 Sub-Task Audit & Output Contract Verification` (`IN_REVIEW`)
       6. `Isolated Branch · task/<parentKey>/<subTaskID>` (`DONE` -> Mở khóa đầu vào cho các Sub-Task phụ thuộc ở Wave tiếp theo!).
2. **Chế Độ Điều Phối Kép (Cả Từng `Sub-Task` Lẫn Theo `Wave DAG` Của `Task Cha`):**
   - Người dùng có thể **bấm chọn bất kỳ `Sub-Task` nào** (trên bảng Sub-Tasks hoặc trên Sơ đồ `Tier 3: Sub-Tasks Dependency DAG Wave 0..N`) để:
     - Xem ngay **luồng `Start -> Plan -> You -> Implement -> Local QA -> Isolated Branch` riêng của Sub-Task đó**,
     - Xem **Đầu vào (`Input Contract`)**, **Plan & Code Demo riêng (`09_tasks_solution_<subTaskID>.md`)**, **Unified Diff riêng**, **Báo cáo Audit S5 riêng**, và **Đầu ra (`Output Contract`)** của Sub-Task đó,
     - Bấm chạy từng bước (`1. Plan Sub-Task`, `✓ APPROVE Sub-Task`, `3. Implement Sub-Task`, `4. QA Sub-Task`, hoặc `▶ Autopilot Sub-Task`) cho riêng Sub-Task đó, HOẶC bấm `✓ APPROVE & Run All Waves` ở Task cha để hệ thống tự động chạy lần lượt từng Sub-Task theo thứ tự `Wave 0 -> Wave 1 -> ... -> Wave N` (trong đó Đầu ra của `Sub-Task` Wave trước tự động truyền vào Đầu vào của `Sub-Task` Wave sau).
3. **Ranh Giới Bất Biến (`L002`/`L003`):**
   - Tuyệt đối không sửa bất kỳ file nào trong `/Users/trainguyen/Documents/work/data-hub/*`.
