-- Migration: fix Bug B — TZ freeze schedule
-- Workspace: InvestigateTransformSchedulerStuck
-- Date: 2026-05-14
-- Target: gpay-postgres-cdc:5433 database cdc_dw schema cdc_system
--
-- Cause: Cột `timestamp without time zone`. Go ghi `time.Now()`
-- (Location=Asia/Ho_Chi_Minh +07). Driver strip TZ → PG store wall-clock
-- local. GORM read back as no-TZ → Location=UTC → lệch +7h vào tương lai.
-- Gating `now.Sub(lastRunAt) < intervalDur` cho duration âm → SKIP.
--
-- Fix: convert sang timestamptz, INTERPRET data hiện tại như local
-- wall-clock (Asia/Ho_Chi_Minh) để dịch về đúng instant UTC.

BEGIN;

ALTER TABLE cdc_system.cdc_worker_schedule
    ALTER COLUMN last_run_at TYPE timestamptz
        USING last_run_at AT TIME ZONE 'Asia/Ho_Chi_Minh',
    ALTER COLUMN next_run_at TYPE timestamptz
        USING next_run_at AT TIME ZONE 'Asia/Ho_Chi_Minh',
    ALTER COLUMN created_at  TYPE timestamptz
        USING created_at  AT TIME ZONE 'Asia/Ho_Chi_Minh',
    ALTER COLUMN updated_at  TYPE timestamptz
        USING updated_at  AT TIME ZONE 'Asia/Ho_Chi_Minh';

-- Sanity: confirm các cột đã đổi sang timestamptz
DO $$
DECLARE
    v_count int;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM information_schema.columns
    WHERE table_schema = 'cdc_system'
      AND table_name = 'cdc_worker_schedule'
      AND column_name IN ('last_run_at','next_run_at','created_at','updated_at')
      AND data_type = 'timestamp with time zone';
    IF v_count <> 4 THEN
        RAISE EXCEPTION 'Migration sanity check failed: expected 4 timestamptz cols, got %', v_count;
    END IF;
END $$;

COMMIT;
