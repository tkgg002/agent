# 02_plan — Giải pháp (1 hướng tốt nhất)

## Root cause (đã verify qua code + DB)
- Realtime transmute = `transmute_schedule.mode IN (immediate, post_ingest)` + `is_enabled`. FE label "Sync: Realtime" = `some(mode==='post_ingest')` (ReconPipelineGrid.tsx:500-504). master_binding KHÔNG có cột realtime.
- TransmuteScheduler poll **CHỈ chạy `mode='cron'`** (transmute_scheduler.go:113-114). Hiện DB **0 row mode='cron'** ⇒ realtime 100% dựa vào trigger lúc ingest (`publishTransmuteTrigger`→`cdc.cmd.transmute-shadow`).
- Toggle realtime: `PATCH /v1/schedules/:id` → `TransmuteScheduleHandler.Toggle` → `ToggleTransmuteScheduleCommand` flip `is_enabled`. **Hiện chỉ flip cờ, KHÔNG catch-up.**
- ⇒ Tắt realtime: ingest vẫn ghi shadow nhưng transmute-trigger không materialise master. Bật lại: chỉ forward event mới ⇒ record trong cửa sổ off **thiếu vĩnh viễn** (đúng case user; = bug binding-11 frozen 06-05).

## Giải pháp: Catch-up tại transition off→on (reuse full-transmute path)
Hook vào `TransmuteScheduleHandler.Toggle`: khi `is_enabled` chuyển **false→true**, sau khi flip cờ, dispatch **`TransmuteRunCommand{MasterTable}`** (publish `cdc.cmd.transmute`) — y hệt `RunNow`/"đồng bộ thủ công". Worker `HandleTransmute` chạy **full Shadow→Master OCC upsert**: insert record thiếu (gap), bỏ qua record master mới hơn (LWW) ⇒ đóng gap idempotent.

**Vì sao đây là hướng tốt nhất** (Simplicity-First, core-systems):
- Đóng gap ĐÚNG thời điểm nó sắp thành vĩnh viễn (re-enable), tự động hoá thao tác "manual sync" mà user mô tả.
- Reuse 100% path transmute đã kiểm chứng (RunNow/cron/heal-B đều publish `cdc.cmd.transmute`) — KHÔNG cơ chế mới, KHÔNG đụng worker.
- OCC/LWW có sẵn ⇒ an toàn, không trùng, không ghi đè.
- Thay đổi ~1 file, ~25 dòng.

## Edge/negative (G4)
- on→off: không catch-up (đúng R1.4).
- enable khi đã enabled (true→true): bỏ qua catch-up (chỉ chạy khi prev=false) — tối ưu, không scan thừa.
- master_table rỗng (binding lỗi/chưa map): bỏ qua dispatch (guard `prev.MasterTable != ""`).
- dispatch fail: log Warn + **revert** `last_status` về 'failed' để KHÔNG kẹt 'running' (bài học stuck-running).
- Toggle nhiều schedule-row của 1 binding (immediate+post_ingest): mỗi lần dispatch 1 catch-up cùng master_table — idempotent, vô hại.

## File thay đổi (dự kiến)
- `cdc-cms-service/internal/api/transmute_schedule_handler.go` — method `Toggle` (+~25 dòng). Không file khác.

## Verify (R3)
1. `go build ./...` CMS = 0.
2. Reproduce: chọn 1 schedule realtime đang enabled → tắt (is_enabled=false) → INSERT 1 record vào shadow (mô phỏng source thêm lúc off) → xác nhận master KHÔNG có → bật lại (is_enabled=true) → xác nhận response `catchup_dispatched=true` + sau vài giây master CÓ record đó (count shadow=master).
3. Restart CMS, smoke endpoint.

## Khuyến nghị bổ sung (không implement task này)
- Bật **recon Segment B (shadow↔master) định kỳ + auto heal-B** làm safety-net cho mọi đường rò khác (snapshot path, missed event), không chỉ toggle. Đã có sẵn cơ chế; chỉ cần lịch.
