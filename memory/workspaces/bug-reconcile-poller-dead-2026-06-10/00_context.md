# 00_context — Reconcile bật nhưng không tự chạy (poller goroutine chết)

- **Ngày**: 2026-06-10 · **Vai trò**: Muscle
- **Triệu chứng (user)**: bật reconcile ở /activity-manager nhưng (1) không tự chạy, (2) không show auto ở /activity-log.

## Root cause (đã loại trừ từng tầng bằng kiểm tra thật)
- `cdc_worker_schedule` id=5 reconcile: is_enabled=t, interval 1m, nhưng **last_run kẹt 2026-06-02, run_count=3** (kể cả sau khi enable lại 10:19 hôm nay).
- Loại trừ: code path có `case "reconcile"` ✓ · reconCore wired (không nil) ✓ · redis gpay-redis:16379 reachable + SetNX OK ✓ · lock TTL 50s không kẹt ✓ · manual recon-check chạy + CÓ trong cdc_activity_log ✓.
- **Kết luận**: poller goroutine (`worker_server.go` vòng `for range ticker.C`) **KHÔNG có recover()** → 1 panic trong cycle (rất có thể `runReconcileCycle`→`reconCore.CheckAll`) ~06-02 đã **giết goroutine poller vĩnh viễn**. Worker vẫn sống (transmute/NATS consumer ở goroutine khác) nhưng mọi auto-op của cdc_worker_schedule (reconcile) đóng băng.

## Fix (phương án b user chọn)
- Thêm `defer recover()` per-operation bao quanh `switch sched.Operation` trong poller → 1 cycle panic không giết scheduler; log `schedule_exec_panic` + `zap.Stack` để lộ root; các op khác vẫn chạy.
- File: `centralized-data-service/internal/server/worker_server.go`. go build PASS.

## Còn lại
- **PHẢI rebuild + restart worker** để áp build mới (binary đang chạy là cũ → poller vẫn chết).
- Sau restart: nếu reconcile vẫn panic → log sẽ in `schedule cycle PANIC recovered ... stacktrace` mỗi phút → dùng stack đó fix root thật của reconcile. Nếu không panic → reconcile chạy lại bình thường + sinh activity `operation="reconcile"`.
- "Không show activity-log": bản scheduled (operation="reconcile") sẽ xuất hiện sau khi poller sống lại; recon-check thủ công vốn đã có trong DB (nếu FE không hiện thì là filter FE — thứ yếu).
