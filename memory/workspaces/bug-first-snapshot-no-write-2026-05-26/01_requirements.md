# 01_requirements — Bug snapshot.v2 first-run no write

## Functional
1. Khi `ResolveSourceRoutes(srcDB, srcColl)` trả empty (route cache
   stale / source_object inactive / shadow_binding inactive):
   - Snapshot **KHÔNG được phép báo success** với rows > 0.
   - Snapshot phải FAIL FAST với error message chỉ rõ root cause để
     operator biết phải bật `is_active` hoặc đợi cache reload.
2. Lần đầu trigger snapshot ngay sau khi register source_object phải
   work — không phụ thuộc race với `schema.config.reload` NATS signal.
3. `activity_log.rows_affected` phải phản ánh số doc thực sự được
   routing thành công, không phải số doc đã đọc từ Mongo Find.

## Non-functional
- Minimal impact: chỉ sửa snapshot_runner + processEvent log level.
  KHÔNG đổi schema, KHÔNG đổi config, KHÔNG đổi NATS subject.
- Circuit breaker hiện tại (consecutive 100 / batch ratio 50%) phải
  tiếp tục hoạt động — không regress.
- Strict mode + non-strict mode đều phải work.
- Resume snapshot (replay với last_seen_id) phải hoạt động: nếu run đầu
  fail-fast vì route empty, lần sau resume phải đi tiếp được khi route
  đã được load đúng.

## Definition of Done
- [ ] `go build ./...` pass ở `centralized-data-service`.
- [ ] `processEvent` route-empty log nhảy từ Debug → Warn.
- [ ] `runSnapshot` pre-flight reload + assert route exists; trả error
      rõ ràng khi route empty.
- [ ] `rowsTotal` accounting dùng số `written` thực sự (return value
      của HandleRaw), không dùng len(batch).
- [ ] `written == 0` được treat như doc error (đi qua `recordDocError`
      → CB sẽ trip nếu deterministic route-miss).
- [ ] `report_first_snapshot_no_write_2026-05-26.md` ghi đủ file thay đổi
      + reasoning + verify steps.
