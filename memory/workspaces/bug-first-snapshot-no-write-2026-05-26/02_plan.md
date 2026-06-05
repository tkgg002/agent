# 02_plan — Fix snapshot.v2 first-run no write

## Phase 1 — Tăng visibility (event_handler.go)

### Change 1: log skip ở mức WARN có context đầy đủ
File: `centralized-data-service/internal/handler/event_handler.go` L86-89

Trước:
```go
routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable)
if len(routes) == 0 {
    h.logger.Debug("table not in registry, skipping", zap.String("source_table", sourceTable))
    return 0, nil
}
```

Sau:
```go
routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable)
if len(routes) == 0 {
    h.logger.Warn("event skipped: source not in registry cache — check is_active flags or trigger schema.config.reload",
        zap.String("subject", subject),
        zap.String("source_db", sourceDB),
        zap.String("source_table", sourceTable),
    )
    return 0, nil
}
```

Lý do: log Debug bị tắt mặc định ở mọi env, operator không bao giờ thấy.
Bug này TỒN TẠI ở Kafka consumer path cũ (lesson L-route-zero-rows) nhưng
chưa fix log layer ở event_handler — fix ở đây cho cả Kafka + snapshot.v2.

## Phase 2 — Fail-fast cho snapshot.v2 (snapshot_runner_handler.go)

### Change 2: Pre-flight reload + assert route exists
File: `centralized-data-service/internal/handler/snapshot_runner_handler.go`
Vị trí: ngay sau line 279 (`coll := client.Database(srcDB).Collection(srcColl)`)
và trước line 281 (`consecutiveErrors := 0`)

Thêm:
```go
// Pre-flight: registry cache may be stale (registered just-now, reload
// NATS signal is fire-and-forget). Trigger an in-process ReloadAll so we
// don't burn 6M docs through processEvent's silent-skip branch.
if reloader, ok := r.registrySvc.(interface{ ReloadAll(context.Context) error }); ok {
    if err := reloader.ReloadAll(ctx); err != nil {
        r.logger.Warn("snapshot.v2 pre-flight registry reload failed (proceeding)",
            zap.Int64("source_object_id", so.ID),
            zap.Error(err))
    }
}
if routes := r.registrySvc.ResolveSourceRoutes(srcDB, srcColl); len(routes) == 0 {
    msg := fmt.Sprintf("no active route for source_db=%q source_collection=%q — verify "+
        "source_object_registry.is_active=true AND shadow_binding.is_active=true "+
        "for source_object_id=%d", srcDB, srcColl, so.ID)
    r.markProgressError(ctx, progressID, msg)
    return errors.New(msg)
}
```

### Change 3: dùng written count thực sự từ HandleRaw, treat 0 như error
File: `centralized-data-service/internal/handler/snapshot_runner_handler.go`
Vị trí: L442 (`if _, err := r.eventHandler.HandleRaw(...)`)

Trước:
```go
if _, err := r.eventHandler.HandleRaw(ctx, subject, envelope); err != nil {
    if rerr := recordDocError("handle event", idStr, afterJSON, err); rerr != nil {
        return rerr
    }
    continue
}
consecutiveErrors = 0
if idStr != "" {
    batchTail = idStr
}
```

Sau:
```go
written, err := r.eventHandler.HandleRaw(ctx, subject, envelope)
if err != nil {
    if rerr := recordDocError("handle event", idStr, afterJSON, err); rerr != nil {
        return rerr
    }
    continue
}
if written == 0 {
    // processEvent skipped this doc — route cache miss (already
    // logged WARN inside processEvent). Treat as a per-doc failure so
    // the circuit breaker stops a snapshot that's silently writing nothing.
    if rerr := recordDocError("route empty", idStr, afterJSON,
        errors.New("processEvent returned 0 — source not in active registry")); rerr != nil {
        return rerr
    }
    continue
}
batchWritten += int64(written)
consecutiveErrors = 0
if idStr != "" {
    batchTail = idStr
}
```

### Change 4: rowsTotal đếm written thực sự, không phải len(batch)
Vị trí: L362-363 (`batchErrors := 0`) và L475 (`rowsTotal += int64(len(batch))`)

Thêm khai báo `batchWritten := int64(0)` cùng `batchErrors := 0`.

Đổi L475:
```go
rowsTotal += int64(len(batch))
```
thành:
```go
rowsTotal += batchWritten
```

## Phase 3 — Verify

- `go build ./...` ở `centralized-data-service` phải pass.
- Đọc lại 2 file đã sửa, đảm bảo import errors đã có sẵn (snapshot_runner
  đã import `errors`; event_handler không cần thêm gì).
- Đối chiếu workspace `bug-snapshot-v2-host-uri-2026-05-21` để không
  break circuit breaker hiện tại.

## Risk & Mitigation
- **Risk**: pre-flight ReloadAll thêm 1 round-trip DB query mỗi lần
  snapshot dispatch. Acceptable — snapshot.v2 dispatch là sự kiện hiếm
  (operator-driven), không phải hot-path streaming.
- **Risk**: `written == 0` treat như doc error có thể trip CB ngay batch 1.
  Đó CHÍNH LÀ behavior mong muốn — operator được thông báo ngay thay vì
  burn 6M docs.
- **Risk**: legacy snapshot có thể đang dùng pattern `written=0` cho
  trường hợp hợp lệ nào đó. Kiểm tra processEvent — chỉ có 2 đường ra
  written=0: (a) routes empty (L86-89), (b) handleDelete path không phù
  hợp ở đây vì envelope op='c'. → an toàn.
