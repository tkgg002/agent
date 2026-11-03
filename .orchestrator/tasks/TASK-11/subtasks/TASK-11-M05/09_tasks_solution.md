---
parent_task: TASK-11
sub_task_id: TASK-11-M05
wave_index: 2
target_repo: /Users/trainguyen/Documents/work/data-hub/centralized-data-service
worktree_path: .orchestrator/worktrees/TASK-11/centralized-data-service
isolated_branch: task/TASK-11/task-11-m05
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-11-M05 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {)` (`centralized-data-service`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-11-M03 · cdc-cms-web · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`) (cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [))
  - [TASK-11-M04 · cdc-cms-web · PENDING] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`) (cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `internal/service/source/metadata_registry_test_helpers.go` (`type MetadataRegistryServiceForTestOpts struct {`) trong worktree cô lập của `centralized-data-service`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {)` (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_test_helpers.go`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-11-M05` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-11-M05 (centralized-data-service · Wave 2): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-11-M03, TASK-11-M04), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-11/centralized-data-service` (nhánh `task/TASK-11/task-11-m05`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 2`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-11-M05` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-11-M05` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-11/task-11-m05` (`.orchestrator/worktrees/TASK-11/centralized-data-service`) trước S4 + kiểm tra `Wave 2 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-11-M05-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-11/task-11-m05` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-11)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-11-M05`
```go
// [TASK-11-M05 · centralized-data-service] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`)
// Đầu vào (InputContract): centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {) (Upstream DependsOn: [TASK-11-M03 TASK-11-M04])
// Đầu ra (OutputContract): Hoàn tất centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {) trên nhánh cô lập task/TASK-11/task-11-m05 (/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_test_helpers.go)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_test_helpers.go (lines 13-37)
  13 | type MetadataRegistryServiceForTestOpts struct {
  14 | 	SourceCache    map[string]*source.TableRegistry
  15 | 	RouteCache     map[string][]*metadata.ResolvedSourceRoute
  16 | 	TargetRouteMap map[string]*metadata.ResolvedSourceRoute
  17 | 	MappingCache   map[int64][]mastermodel.MappingRule
  18 | 	TargetCache    map[string]*source.TableRegistry
  19 | 	CloneRoutes    map[int64][]*metadata.ResolvedSourceRoute
  20 | }
  21 | 
  22 | // NewMetadataRegistryServiceForTest preloads a MetadataRegistryService with the supplied cache snapshots.
  23 | func NewMetadataRegistryServiceForTest(opts MetadataRegistryServiceForTestOpts) *MetadataRegistryService {
  24 | 	rs := &MetadataRegistryService{
  25 | 		logger:         zap.NewNop(),
  26 | 		sourceCache:    opts.SourceCache,
  27 | 		routeCache:     opts.RouteCache,
  28 | 		targetRouteMap: opts.TargetRouteMap,
  29 | 		mappingCache:   opts.MappingCache,
  30 | 		targetCache:    opts.TargetCache,
  31 | 		cloneRoutes:    opts.CloneRoutes,
  32 | 	}
  33 | 	if rs.mappingCache == nil {
  34 | 		rs.mappingCache = make(map[int64][]mastermodel.MappingRule)
  35 | 	}
  36 | 	return rs
  37 | }
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [centralized-data-service] Cập nhật & kiểm chứng centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {) — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-11-M05-PENDING · sha256=9b10f8b0143f)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-11)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M05/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M05/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-11-M05-PENDING` (`HandoffReady = false`)
