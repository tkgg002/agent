---
parent_task: TASK-11
sub_task_id: TASK-11-M01
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service
worktree_path: .orchestrator/worktrees/TASK-11/cdc-cms-service
isolated_branch: task/TASK-11/task-11-m01
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-11-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})` (`cdc-cms-service`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `true`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [Wave 0 Entry · Parent TASK-11] Nhận trực tiếp đặc tả gốc 01_requirements.md & điểm chạm cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{}) trên cdc-cms-service
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `internal/app/queries/recon/list_latest_reports.go` (`type ListLatestReportsQuery struct{}`) trong worktree cô lập của `cdc-cms-service`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/list_latest_reports.go`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-11-M01` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-11-M01 (cdc-cms-service · Wave 0): Nhận đầu vào từ 0 Upstream Sub-Tasks (Wave 0 Entry), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-11/cdc-cms-service` (nhánh `task/TASK-11/task-11-m01`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 2 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 0`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-11-M01` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-11-M01` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-11/task-11-m01` (`.orchestrator/worktrees/TASK-11/cdc-cms-service`) trước S4 + kiểm tra `Wave 0 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-11-M01-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-11/task-11-m01` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`TASK-11-M03, TASK-11-M04`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-11-M01`
```go
// [TASK-11-M01 · cdc-cms-service] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`)
// Đầu vào (InputContract): cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{}) (Upstream DependsOn: [])
// Đầu ra (OutputContract): Hoàn tất cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{}) trên nhánh cô lập task/TASK-11/task-11-m01 (/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/list_latest_reports.go)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/list_latest_reports.go (lines 9-33)
   9 | type ListLatestReportsQuery struct{}
  10 | 
  11 | func (q ListLatestReportsQuery) Type() string { return "recon.list_latest" }
  12 | 
  13 | // ListLatestReportsResult returns the raw rows. The HTTP handler
  14 | // fills the four `gorm:"-"` enrichment fields (DriftPct,
  15 | // ComputedStatus, ErrorMessageVI, SourceQueryMethod) from
  16 | // ComputeDriftStatus + ErrorMessagesVI + deriveSourceQueryMethod —
  17 | // those helpers stay in `internal/api/` because they own a 16-case
  18 | // unit test suite (ComputeDriftStatus_test.go) we don't want to
  19 | // move yet.
  20 | type ListLatestReportsResult struct {
  21 | 	Data  []LatestReportRow
  22 | 	Count int
  23 | }
  24 | 
  25 | // ListLatestReportsHandler resolves the query against an injected
  26 | // ReconReader.
  27 | type ListLatestReportsHandler struct {
  28 | 	reader ReconReader
  29 | }
  30 | 
  31 | func NewListLatestReportsHandler(r ReconReader) *ListLatestReportsHandler {
  32 | 	return &ListLatestReportsHandler{reader: r}
  33 | }
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [cdc-cms-service] Cập nhật & kiểm chứng cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{}) — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-11-M01-PENDING · sha256=179c01860198)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `TASK-11-M03, TASK-11-M04`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M01/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M01/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-11-M01-PENDING` (`HandoffReady = false`)
