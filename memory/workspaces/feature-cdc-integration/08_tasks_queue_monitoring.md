# Task: CDC Worker Queue Monitoring Execution

Thực hiện giải pháp giám sát tiến trình xử lý và buffer trong CDC Worker.

- `[x]` 1. Backend Instrumentation:
    - `[x]` Update `internal/handler/consumer_pool.go`: Add atomic counters (processed, failed, active) + `GetStats()`
    - `[x]` Update `internal/handler/batch_buffer.go`: Add `GetStatus()` (size, last flush)
- `[x]` 2. API Implementation:
    - `[x]` Update `internal/server/worker_server.go`: Expose `/api/v1/internal/stats` endpoint
- `[x]` 3. Frontend Implementation:
    - `[x]` Create `src/pages/QueueMonitoring.tsx` in `cdc-cms-web`
    - `[x]` Update `src/App.tsx`: Add "Queue Monitor" to Sidebar
- `[x]` 4. Verification:
    - `[x]` Unit test stats gathering (if applicable)
    - `[x]` Manual check via Browser
- `[x]` 5. Governance:
    - `[x]` Update `05_progress.md` after each step
    - `[x]` Update `04_decisions.md` (ADR for observability)
