---
parent_task: TASK-4
sub_task_id: TASK-4-M01
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-4/agent
isolated_branch: task/TASK-4/task-4-m01
stage: DONE
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-4-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`migration_fix_tz_worker_schedule.sql`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `true`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [Wave 0 Entry · Parent TASK-4] Nhận trực tiếp đặc tả gốc 01_requirements.md & điểm chạm agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule) trên agent
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql` (`-- Migration: fix Bug B — TZ freeze schedule`) trong worktree cô lập của `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule)` (`/Users/trainguyen/Documents/work/agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-4-M01` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-4-M01 (agent · Wave 0): Nhận đầu vào từ 0 Upstream Sub-Tasks (Wave 0 Entry), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-4/agent` (nhánh `task/TASK-4/task-4-m01`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 2 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 0`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-4-M01` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-4-M01` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-4/task-4-m01` (`.orchestrator/worktrees/TASK-4/agent`) trước S4 + kiểm tra `Wave 0 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-4-M01-PASS`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-4/task-4-m01` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`TASK-4-M03, TASK-4-M04`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-4-M01`
```go
// [TASK-4-M01 · agent] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`migration_fix_tz_worker_schedule.sql`)
// Đầu vào (InputContract): agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule) (Upstream DependsOn: [])
// Đầu ra (OutputContract): Hoàn tất agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule) trên nhánh cô lập task/TASK-4/task-4-m01 (/Users/trainguyen/Documents/work/agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql (lines 1-25)
   1 | -- Migration: fix Bug B — TZ freeze schedule
   2 | -- Workspace: InvestigateTransformSchedulerStuck
   3 | -- Date: 2026-05-14
   4 | -- Target: gpay-postgres-cdc:5433 database cdc_dw schema cdc_system
   5 | --
   6 | -- Cause: Cột `timestamp without time zone`. Go ghi `time.Now()`
   7 | -- (Location=Asia/Ho_Chi_Minh +07). Driver strip TZ → PG store wall-clock
   8 | -- local. GORM read back as no-TZ → Location=UTC → lệch +7h vào tương lai.
   9 | -- Gating `now.Sub(lastRunAt) < intervalDur` cho duration âm → SKIP.
  10 | --
  11 | -- Fix: convert sang timestamptz, INTERPRET data hiện tại như local
  12 | -- wall-clock (Asia/Ho_Chi_Minh) để dịch về đúng instant UTC.
  13 | 
  14 | BEGIN;
  15 | 
  16 | ALTER TABLE cdc_system.cdc_worker_schedule
  17 |     ALTER COLUMN last_run_at TYPE timestamptz
  18 |         USING last_run_at AT TIME ZONE 'Asia/Ho_Chi_Minh',
  19 |     ALTER COLUMN next_run_at TYPE timestamptz
  20 |         USING next_run_at AT TIME ZONE 'Asia/Ho_Chi_Minh',
  21 |     ALTER COLUMN created_at  TYPE timestamptz
  22 |         USING created_at  AT TIME ZONE 'Asia/Ho_Chi_Minh',
  23 |     ALTER COLUMN updated_at  TYPE timestamptz
  24 |         USING updated_at  AT TIME ZONE 'Asia/Ho_Chi_Minh';
  25 | 
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule) — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`migration_fix_tz_worker_schedule.sql`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-4-M01-PASS · sha256=326c4f873bc7)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `TASK-4-M03, TASK-4-M04`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-4/subtasks/TASK-4-M01/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-4/subtasks/TASK-4-M01/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-4-M01-PASS` (`HandoffReady = true`)
