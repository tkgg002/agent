# 09_tasks_solution — Snapshot.v2 first-run no write

## Solution summary
Bug có 4 lớp chồng nhau, tất cả ở phía worker pipeline:

1. **L4 Cache stale**: ReloadAll registry chỉ chạy ở (a) startup, (b)
   khi nhận NATS signal `schema.config.reload`. Signal là fire-and-forget,
   không ack. Khi user register source rồi click Snapshot Now → race.
2. **L3 Silent skip**: `event_handler.processEvent` trả `(0, nil)` khi
   `ResolveSourceRoutes` rỗng, kèm log Debug — operator không thấy.
3. **L2 Caller discard**: `snapshot_runner` vứt giá trị written
   (`_, err := HandleRaw(...)`) → mất tín hiệu route-miss.
4. **L1 Misleading metric**: `rowsTotal += int64(len(batch))` đếm số doc
   Find quét, không phải số doc routed. Activity log báo success với
   N rows trong khi shadow 0 rows.

## Fix layered
- **L4 → Pre-flight reload**: snapshot_runner force `ReloadAll(ctx)` ngay
  sau khi mở mongo client, trước cursor loop.
- **L4 → Hard-assert**: nếu sau reload route vẫn rỗng → fail fast với
  message chỉ rõ `is_active` cần check, KHÔNG đi vào cursor loop.
- **L3 → Warn log**: chuyển Debug → Warn với (subject, source_db,
  source_table).
- **L2 → Use return**: lấy `written` từ HandleRaw; treat 0 như doc error
  → CB trip nhanh nếu route deterministic empty.
- **L1 → Accurate count**: `rowsTotal += batchWritten` (sum of
  per-doc written returns).

## Why this layering
- 4 lớp fix tương ứng 4 failure mode độc lập. Nếu chỉ fix L4 (pre-flight),
  vẫn còn risk khi route bị xóa giữa snapshot. L3 + L2 + L1 đảm bảo
  defense in depth: dù route miss vì lý do gì, snapshot sẽ fail fast và
  metric trung thực.
- Tuân thủ §7 GEMINI.md "Demand Elegance" — không cải tổ registry signal
  layer (vốn là pattern fire-and-forget cố ý). Chỉ thêm minimal hooks.

## File touched
- `centralized-data-service/internal/handler/event_handler.go` (L84-99)
- `centralized-data-service/internal/handler/snapshot_runner_handler.go`
  (L279-307 pre-flight; L390-392 batchWritten decl; L461-495 inner loop;
   L518-521 rowsTotal accumulation)

## Verification
- `go build ./...` PASS.
- `go vet ./internal/handler/...` PASS.
- `go test ./internal/handler/` PASS (0.9s, no regression).

## Open follow-up (defer)
- BatchBuffer.Flush() vẫn nuốt lỗi UPSERT — không propagate về snapshot
  caller. Hiện tại forensics ở `failed_sync_logs` đầy đủ; nếu cần kill
  switch khi tất cả flush fail thì thêm `FlushStats() (ok, fail int)`
  vào BatchBuffer + snapshot_runner check delta sau mỗi `FlushBatchBuffer()`.
- Cân nhắc thêm metric Prometheus `snapshot_route_miss_total` để alert
  từ Grafana.
