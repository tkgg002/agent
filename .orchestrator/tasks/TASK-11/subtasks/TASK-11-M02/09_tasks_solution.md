---
parent_task: TASK-11
sub_task_id: TASK-11-M02
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service
worktree_path: .orchestrator/worktrees/TASK-11/cdc-cms-service
isolated_branch: task/TASK-11/task-11-m02
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-11-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)` (`cdc-cms-service`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `true`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [Wave 0 Entry · Parent TASK-11] Nhận trực tiếp đặc tả gốc 01_requirements.md & điểm chạm cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {) trên cdc-cms-service
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `internal/api/dto/audit_dto.go` (`type QASummaryResponse struct {`) và viết integration test xác minh đầu-cuối trên `cdc-cms-service`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/dto/audit_dto.go`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-11-M02` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-11-M02 (cdc-cms-service · Wave 0): Nhận đầu vào từ 0 Upstream Sub-Tasks (Wave 0 Entry), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-11/cdc-cms-service` (nhánh `task/TASK-11/task-11-m02`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 2 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 0`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-11-M02` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-11-M02` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-11/task-11-m02` (`.orchestrator/worktrees/TASK-11/cdc-cms-service`) trước S4 + kiểm tra `Wave 0 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-11-M02-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-11/task-11-m02` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`TASK-11-M03, TASK-11-M04`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-11-M02`
```go
// [TASK-11-M02 · cdc-cms-service] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`)
// Đầu vào (InputContract): cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {) (Upstream DependsOn: [])
// Đầu ra (OutputContract): Hoàn tất cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {) trên nhánh cô lập task/TASK-11/task-11-m02 (/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/dto/audit_dto.go)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/dto/audit_dto.go (lines 3-27)
   3 | type QASummaryResponse struct {
   4 |     CompositeScore   float64             `json:"composite_score"`     // 0-100
   5 |     TotalCriteria    int                 `json:"total_criteria"`       // 16
   6 |     RatingBreakdown  map[string]int      `json:"rating_breakdown"`     // {"L4":1,"L3":6,...}
   7 |     GapBreakdown     map[string]GapCount `json:"gap_breakdown"`        // {"P0":{open:4,closed:0},...}
   8 |     LastAudited      string              `json:"last_audited"`         // RFC3339
   9 |     Criteria         []CriterionRow      `json:"criteria"`
  10 | }
  11 | 
  12 | type GapCount struct {
  13 |     Open       int `json:"open"`
  14 |     InProgress int `json:"in_progress"`
  15 |     Closed     int `json:"closed"`
  16 |     Wontfix    int `json:"wontfix"`
  17 | }
  18 | 
  19 | type CriterionRow struct {
  20 |     ID       string `json:"id"`
  21 |     Group    string `json:"group"`
  22 |     Title    string `json:"title"`
  23 |     Rating   string `json:"rating"`
  24 |     Evidence string `json:"evidence"`
  25 | }
  26 | 
  27 | type GapRow struct {
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [cdc-cms-service] Cập nhật & kiểm chứng cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-11-M02-PENDING · sha256=c592b9c13763)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `TASK-11-M03, TASK-11-M04`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M02/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M02/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-11-M02-PENDING` (`HandoffReady = false`)
