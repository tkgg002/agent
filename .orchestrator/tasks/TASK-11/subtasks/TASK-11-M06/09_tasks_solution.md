---
parent_task: TASK-11
sub_task_id: TASK-11-M06
wave_index: 2
target_repo: /Users/trainguyen/Documents/work/data-hub/centralized-data-service
worktree_path: .orchestrator/worktrees/TASK-11/centralized-data-service
isolated_branch: task/TASK-11/task-11-m06
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-11-M06 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `centralized-data-service/internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {)` (`centralized-data-service`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-11-M03 · cdc-cms-web · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`) (cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [))
  - [TASK-11-M04 · cdc-cms-web · PENDING] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`) (cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `internal/model/recon/reconciliation_report.go` (`type ReconciliationReport struct {`) và viết integration test xác minh đầu-cuối trên `centralized-data-service`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `centralized-data-service/internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {)` (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-11-M06` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-11-M06 (centralized-data-service · Wave 2): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-11-M03, TASK-11-M04), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-11/centralized-data-service` (nhánh `task/TASK-11/task-11-m06`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 2`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `centralized-data-service/internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-11-M06` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-11-M06` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-11/task-11-m06` (`.orchestrator/worktrees/TASK-11/centralized-data-service`) trước S4 + kiểm tra `Wave 2 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-11-M06-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-11/task-11-m06` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-11)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-11-M06`
```go
// [TASK-11-M06 · centralized-data-service] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`)
// Đầu vào (InputContract): centralized-data-service/internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {) (Upstream DependsOn: [TASK-11-M03 TASK-11-M04])
// Đầu ra (OutputContract): Hoàn tất centralized-data-service/internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {) trên nhánh cô lập task/TASK-11/task-11-m06 (/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go (lines 8-32)
   8 | type ReconciliationReport struct {
   9 | 	ID          uint64 `gorm:"primaryKey" json:"id"`
  10 | 	TargetTable string `gorm:"-" json:"target_table"`
  11 | 	SourceDB    string `gorm:"column:source_db" json:"source_db"`
  12 | 	SourceType  string `gorm:"column:source_type" json:"source_type"`
  13 | 	SourceHost  string `gorm:"column:source_host" json:"source_host"`
  14 | 	SourceTable string `gorm:"column:source_table" json:"source_table"`
  15 | 	// Migration 017 — SourceCount is pointer/nullable so we can distinguish
  16 | 	// query failure (NULL) from actual empty window (0). KHÔNG tag default:0:
  17 | 	// tag đó làm gorm ghi 0 khi nil → error-row thành source_count=0 → FE
  18 | 	// hiện 0 GIẢ + "ingest +N thừa" rác suốt lúc nguồn unreachable (đã xảy ra).
  19 | 	SourceCount  *int64          `gorm:"column:source_count" json:"source_count"`
  20 | 	DestCount    int64           `gorm:"column:dest_count;default:0" json:"dest_count"`
  21 | 	Diff         int64           `gorm:"column:diff;default:0" json:"diff"`
  22 | 	MissingCount int             `gorm:"column:missing_count;default:0" json:"missing_count"`
  23 | 	MissingIDs   json.RawMessage `gorm:"column:missing_ids;type:jsonb" json:"missing_ids"`
  24 | 	StaleCount   int             `gorm:"column:stale_count;default:0" json:"stale_count"`
  25 | 	StaleIDs     json.RawMessage `gorm:"column:stale_ids;type:jsonb" json:"stale_ids"`
  26 | 	CheckType    string          `gorm:"column:check_type;not null" json:"check_type"`
  27 | 	Status       string          `gorm:"column:status;not null" json:"status"`
  28 | 	// Migration 081 (Recon V4) — segment định vị tầng đối soát:
  29 | 	// 'source_shadow' (A — ingest path) | 'shadow_master' (B — transmute path).
  30 | 	Segment string `gorm:"column:segment;default:source_shadow" json:"segment"`
  31 | 	// Migration 085 — KHÓA PIPELINE ổn định. target_table (bare) KHÔNG unique:
  32 | 	// trùng tên across-schema + segment A ghi tên shadow / B ghi tên master. Cặp
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [centralized-data-service] Cập nhật & kiểm chứng centralized-data-service/internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-11-M06-PENDING · sha256=802e5fe9523d)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-11)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M06/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M06/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-11-M06-PENDING` (`HandoffReady = false`)
