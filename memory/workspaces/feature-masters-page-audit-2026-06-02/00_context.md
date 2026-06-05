# 00_context.md — Masters Page Audit & Fix

## Workspace
- **Feature**: Masters Page (http://localhost:5173/masters) — Audit, Gap Analysis & Fix Plan
- **Created**: 2026-06-02
- **Status**: 🟡 Active (Plan phase)

## Scope
Audit toàn bộ chức năng trang `/masters` (MasterRegistry.tsx), bao gồm:
1. UI/UX chức năng hiện tại
2. API backend tương ứng
3. Luồng Sync Shadow → Master (3 loại: run_now, schedule/cron, realtime via post_ingest)
4. Gap analysis so với yêu cầu

## Components liên quan
- **FE**: `cdc-cms-web/src/pages/MasterRegistry.tsx` (503 lines)
- **FE**: `cdc-cms-web/src/pages/TransmuteSchedules.tsx` (schedule management)
- **CMS API** (cdc-cms-service):
  - `internal/api/master_registry_handler*.go` (6 files)
  - `internal/api/transmute_schedule_handler.go`
  - `internal/app/commands/create_master.go`
  - `internal/app/commands/approve_master.go`
  - `internal/app/commands/reject_master.go`
  - `internal/app/commands/toggle_master_active.go`
  - `internal/app/commands/master_swap.go`
  - `internal/app/commands/create_schedule.go`
  - `internal/app/queries/list_masters.go`
  - `internal/router/router.go` (routes)
- **Worker** (centralized-data-service):
  - `internal/handler/transmute_handler.go` (HandleTransmute, HandleTransmuteShadow)
  - `internal/service/transmute_scheduler.go` (cron tick)
  - `internal/sinkworker/sinkworker.go` (post_ingest hook)
  - `internal/model/transmute_schedule.go`

## Sync Strategy đã có trong hệ thống
- **Mode: `cron`** — Scheduler tick 60s, chọn hàng `mode='cron' AND next_run_at <= NOW()`, publish `cdc.cmd.transmute`
- **Mode: `immediate`** — Lưu vào DB nhưng KHÔNG tự trigger; cần gọi `/run-now` thủ công
- **Mode: `post_ingest`** — SinkWorker sau khi write shadow gọi `publishTransmuteTrigger` → publish `cdc.cmd.transmute-shadow` → TransmuteHandler fan-out → `cdc.cmd.transmute`

**Vấn đề yêu cầu**: User muốn trong trang `/masters` có 3 loại sync rõ ràng:
1. Chạy ngay (run_now / immediate dispatch)
2. Hẹn giờ (cron schedule)
3. Realtime theo oplog → shadow → master (post_ingest)
