# 13 Analysis: TASK-4-M01 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql` (agent:memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql (verified file on disk)) |
| Hợp đồng Đầu vào (`Wave 0`) | `VERIFIED` | `UpstreamReady = true` (DependsOn: ``) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-4/task-4-m01` (`.orchestrator/worktrees/TASK-4/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql`)
```go
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
