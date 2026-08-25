# 00_context.md — FixDeadlockAndProgressNotification20260824

## Phạm vi (Scope)
Fix 4 vấn đề liên quan đến CDC Worker và CMS UI:
1. **Deadlock SQLSTATE 40P01** trên `master_payment_service.payments` trong `master.bulk_upsert`
2. **Thông báo "đang chạy"** từ pipeline (cdc-worker) khi transmute đang chạy — thêm badge/indicator trên TransmuteSchedules
3. **Bug không hiện progress** khi đang chạy Sync Now & Transform trong TableRegistry / MasterRegistry
4. **Thiếu TraceID** trên trang Transmute Schedules để user trace lại từng lần run

## Thành phần liên quan

### Backend — `centralized-data-service`
- `internal/handler/master/transmute_handler.go` — `getOrCreateDebouncer()` line 377 (concurrencyLimit=10)
- `internal/handler/master/debounce.go` — `TableDebouncer` struct
- `internal/model/master/transmute_schedule.go` — thiếu field `last_trace_id`
- `internal/repository/master/transmute_schedule_repo.go`
- DB migration — thêm cột `last_trace_id` vào `cdc_system.transmute_schedule`

### Frontend — `cdc-cms-web`
- `src/pages/TransmuteSchedules.tsx` — thiếu TraceID column, thiếu running badge
- `src/pages/MasterRegistry.tsx` — `TransmuteJobStatus` progress bug
- `src/pages/TableRegistry.tsx` — `TransformJobStatus` progress bug
- `src/utils/actionToast.tsx` — OK (dùng lại)

## Môi trường
- Go 1.22+, PostgreSQL, NATS, Ant Design 5.x
- Workspace: `/Users/trainguyen/Documents/work/data-hub`
