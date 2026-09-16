# Fix Transmute Pipeline Reliability

## Scope
Sửa 6 drop points trong pipeline CDC → NATS → Debouncer → Transmute gây rớt message, dẫn đến phải chạy recon liên tục.

## Components
- `centralized-data-service/internal/handler/master/transmute_handler.go` — NATS handler
- `centralized-data-service/internal/handler/master/debounce.go` — TableDebouncer  
- `centralized-data-service/internal/service/master/transmute_scheduler.go` — Cron scheduler
- `centralized-data-service/internal/service/recon/recon_smoke.go` — Smoke recon CountRows

## Root Cause
Pipeline dùng NATS Core (at-most-once) + silent drop + no retry = message mất ở nhiều điểm.
