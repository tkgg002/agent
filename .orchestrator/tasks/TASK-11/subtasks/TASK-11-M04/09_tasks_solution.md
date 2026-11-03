---
parent_task: TASK-11
sub_task_id: TASK-11-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web
worktree_path: .orchestrator/worktrees/TASK-11/cdc-cms-web
isolated_branch: task/TASK-11/task-11-m04
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-11-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {)` (`cdc-cms-web`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-11-M01 · cdc-cms-service · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`) (cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{}))
  - [TASK-11-M02 · cdc-cms-service · PENDING] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`) (cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `src/hooks/useAuditStatus.ts` (`export function useAuditStatus() {`) và viết integration test xác minh đầu-cuối trên `cdc-cms-web`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {)` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useAuditStatus.ts`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-11-M04` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-11-M04 (cdc-cms-web · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-11-M01, TASK-11-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-11/cdc-cms-web` (nhánh `task/TASK-11/task-11-m04`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (npm run build && npm test) và bàn giao đầu ra cho 2 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-11-M04` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-11-M04` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-11/task-11-m04` (`.orchestrator/worktrees/TASK-11/cdc-cms-web`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`npm run build && npm test`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-11-M04-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-11/task-11-m04` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`TASK-11-M05, TASK-11-M06`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-11-M04`
```typescript
// [TASK-11-M04 · cdc-cms-web] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`)
// Đầu vào (InputContract): cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {) (Upstream DependsOn: [TASK-11-M01 TASK-11-M02])
// Đầu ra (OutputContract): Hoàn tất cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {) trên nhánh cô lập task/TASK-11/task-11-m04 (/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useAuditStatus.ts)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useAuditStatus.ts (lines 5-29)
   5 | export function useAuditStatus() {
   6 |     const [summary, setSummary] = useState<QASummaryResponse | null>(null);
   7 |     const [metrics, setMetrics] = useState<MetricHealth[]>([]);
   8 |     const [loading, setLoading] = useState(true);
   9 |     const [error, setError] = useState<string | null>(null);
  10 | 
  11 |     const fetchData = async () => {
  12 |         try {
  13 |             setLoading(true);
  14 |             const [sumRes, metricRes] = await Promise.all([
  15 |                 cmsApi.get('/api/v1/audit/qa-summary').then(r => r.data),
  16 |                 cmsApi.get('/api/v1/audit/metric-health').then(r => r.data.metrics || [])
  17 |             ]);
  18 |             setSummary(sumRes);
  19 |             setMetrics(metricRes);
  20 |             setError(null);
  21 |         } catch (err: any) {
  22 |             setError(err.message || 'Failed to fetch audit status');
  23 |         } finally {
  24 |             setLoading(false);
  25 |         }
  26 |     };
  27 | 
  28 |     useEffect(() => {
  29 |         fetchData();
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [cdc-cms-web] Cập nhật & kiểm chứng cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`)
  - [DoD Gate] npm run build && npm test (Biên lai vật lý: EV-SUB-TASK-11-M04-PENDING · sha256=6666607f90c9)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `TASK-11-M05, TASK-11-M06`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M04/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M04/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-11-M04-PENDING` (`HandoffReady = false`)
