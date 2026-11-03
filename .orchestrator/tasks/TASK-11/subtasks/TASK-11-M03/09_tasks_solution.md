---
parent_task: TASK-11
sub_task_id: TASK-11-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web
worktree_path: .orchestrator/worktrees/TASK-11/cdc-cms-web
isolated_branch: task/TASK-11/task-11-m03
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-11-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [)` (`cdc-cms-web`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-11-M01 · cdc-cms-service · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`) (cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{}))
  - [TASK-11-M02 · cdc-cms-service · PENDING] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`) (cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `src/constants/postgresDataTypes.ts` (`export const POSTGRES_DATA_TYPES = [`) trong worktree cô lập của `cdc-cms-web`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [)` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/postgresDataTypes.ts`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-11-M03` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-11-M03 (cdc-cms-web · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-11-M01, TASK-11-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-11/cdc-cms-web` (nhánh `task/TASK-11/task-11-m03`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (npm run build && npm test) và bàn giao đầu ra cho 2 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-11-M03` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-11-M03` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-11/task-11-m03` (`.orchestrator/worktrees/TASK-11/cdc-cms-web`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`npm run build && npm test`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-11-M03-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-11/task-11-m03` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`TASK-11-M05, TASK-11-M06`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-11-M03`
```typescript
// [TASK-11-M03 · cdc-cms-web] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`)
// Đầu vào (InputContract): cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [) (Upstream DependsOn: [TASK-11-M01 TASK-11-M02])
// Đầu ra (OutputContract): Hoàn tất cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [) trên nhánh cô lập task/TASK-11/task-11-m03 (/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/postgresDataTypes.ts)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/postgresDataTypes.ts (lines 1-25)
   1 | export const POSTGRES_DATA_TYPES = [
   2 |   'VARCHAR',
   3 |   'VARCHAR(50)',
   4 |   'VARCHAR(100)',
   5 |   'VARCHAR(255)',
   6 |   'CHAR',
   7 |   'TEXT',
   8 |   'INTEGER',
   9 |   'BIGINT',
  10 |   'SMALLINT',
  11 |   'NUMERIC',
  12 |   'DECIMAL(18,6)',
  13 |   'DOUBLE PRECISION',
  14 |   'REAL',
  15 |   'BOOLEAN',
  16 |   'TIMESTAMP',
  17 |   'TIMESTAMPTZ',
  18 |   'DATE',
  19 |   'JSONB',
  20 |   'JSON',
  21 |   'BYTEA',
  22 | ] as const;
  23 | 
  24 | export type PostgresDataType = typeof POSTGRES_DATA_TYPES[number];
  25 | 
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [cdc-cms-web] Cập nhật & kiểm chứng cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [) — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`)
  - [DoD Gate] npm run build && npm test (Biên lai vật lý: EV-SUB-TASK-11-M03-PENDING · sha256=18aa63b91dee)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `TASK-11-M05, TASK-11-M06`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M03/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-11/subtasks/TASK-11-M03/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-11-M03-PENDING` (`HandoffReady = false`)
