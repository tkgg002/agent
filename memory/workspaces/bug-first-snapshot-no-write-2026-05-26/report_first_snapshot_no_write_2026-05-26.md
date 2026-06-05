# Report — Snapshot.v2 first-run không ghi data vào shadow

**Date**: 2026-05-26
**Workspace**: `bug-first-snapshot-no-write-2026-05-26`
**Trigger user**: "kiêm tra sao lần snapshot đầu ko ghi data vào shadow. debug nó xem."

---

## TL;DR
Lần snapshot đầu báo `status=success` với rows = N (số doc Mongo Find
quét được) NHƯNG shadow table 0 row. Nguyên nhân chuỗi:

1. Registry cache stale vì NATS reload signal là fire-and-forget.
2. `event_handler.processEvent` silent-skip khi route rỗng, log Debug.
3. `snapshot_runner` vứt return value `written` của `HandleRaw`.
4. `rowsTotal += len(batch)` đếm sai — số doc Find, không phải số doc routed.

Đã fix 4 lớp ở 2 file: `event_handler.go` + `snapshot_runner_handler.go`.

---

## Root cause analysis

### Failure mode chain
```
User register source_object/shadow_binding qua UI
  ↓ CMS commit DB tx
  ↓ NATS Publish("schema.config.reload", ...) — fire & forget, KHÔNG ack
  ↓ User immediately click "Snapshot Now"
  ↓ CMS Publish("cdc.cmd.snapshot.v2", ...)
  ↓ Worker nhận snapshot.v2
  ↓ (race) Worker chưa kịp ReloadAll registry → cache thiếu route
  ↓ runSnapshot mở mongo cursor, push từng doc qua HandleRaw
  ↓ processEvent gọi ResolveSourceRoutes → []*Route{} (empty)
  ↓ processEvent return (0, nil) + log Debug (không ai thấy)
  ↓ snapshot_runner: _, err := HandleRaw → bỏ written, err=nil
  ↓ rowsTotal += int64(len(batch))  ← đếm doc Find, không phải doc routed
  ↓ Loop exhausted → markProgressDone + writeActivity status=success rows=6M
  ↓ Operator nhìn shadow table → 0 row → "báo cáo láo"
```

### Source of truth — code references
| Layer | File:line | Behavior |
|-------|-----------|----------|
| L4 cache | `metadata_registry_service.go:108-256` | ReloadAll chỉ chạy lúc startup + khi nhận `schema.config.reload` |
| L4 trigger | `worker_server.go:230-251` | `Subscribe("schema.config.reload", ReloadAll)` — không có ack |
| L3 skip | `event_handler.go:86-89` | `len(routes)==0 → Debug log + return (0, nil)` |
| L2 discard | `snapshot_runner_handler.go:442` | `if _, err := ... HandleRaw(...)` — vứt written |
| L1 metric | `snapshot_runner_handler.go:475` | `rowsTotal += int64(len(batch))` |

---

## Fix applied

### File 1: `centralized-data-service/internal/handler/event_handler.go`
**L84-99**: skip log Debug → Warn, thêm context (subject, source_db, source_table).

### File 2: `centralized-data-service/internal/handler/snapshot_runner_handler.go`
**(a) L279-307 — Pre-flight reload + hard-assert route**:
```go
if reloader, ok := r.registrySvc.(interface{ ReloadAll(context.Context) error }); ok {
    if err := reloader.ReloadAll(ctx); err != nil {
        r.logger.Warn("snapshot.v2 pre-flight registry reload failed ...", ...)
    }
}
if routes := r.registrySvc.ResolveSourceRoutes(srcDB, srcColl); len(routes) == 0 {
    msg := fmt.Sprintf("no active route for source_db=%q source_collection=%q ...", ...)
    r.markProgressError(ctx, progressID, msg)
    return errors.New(msg)
}
```

**(b) L390-392 — Khai báo `batchWritten`**:
```go
batchErrors := 0
batchWritten := int64(0)
var lastErr error
```

**(c) L461-495 — Sử dụng `written` từ HandleRaw, treat 0 như doc error**:
```go
written, err := r.eventHandler.HandleRaw(ctx, subject, envelope)
if err != nil {
    if rerr := recordDocError("handle event", idStr, afterJSON, err); rerr != nil {
        return rerr
    }
    continue
}
if written == 0 {
    if rerr := recordDocError("route empty", idStr, afterJSON,
        errors.New("processEvent returned 0 — source not in active registry")); rerr != nil {
        return rerr
    }
    continue
}
batchWritten += int64(written)
```

**(d) L518-521 — `rowsTotal += batchWritten` thay vì `len(batch)`**.

---

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` (centralized-data-service) | EXIT=0 |
| `go vet ./internal/handler/...` | EXIT=0 |
| `go test ./internal/handler/ -count=1` | PASS 0.9s |
| Existing circuit breaker (consecutive 100 / ratio 50%) | preserved |
| Strict mode fail-on-first-error | preserved |
| Resume snapshot via last_seen_id | preserved (chỉ fail-fast trước khi vào loop) |

### Manual smoke (cần operator test khi run live)
1. Tạo source_object mới qua UI với `is_active=false`. Click Snapshot.
   - Trước fix: progress=done, rows=N, shadow 0 row.
   - Sau fix: progress=error, error_msg chỉ rõ `is_active=false`, không
     mất chu kỳ chạy snapshot.
2. Tạo source_object mới với `is_active=true`. Click Snapshot ngay
   sau register.
   - Trước fix: rủi ro race, 0 row.
   - Sau fix: pre-flight reload ensure route loaded → snapshot work.
3. Bình thường: snapshot.v2 trên source đã active từ trước.
   - Trước fix: work.
   - Sau fix: work (pre-flight reload chỉ thêm ~1 query, không đổi logic).

---

## Files changed (audit)
```
data-hub/centralized-data-service/internal/handler/event_handler.go
data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go
```
Tổng diff: +42 / -5 dòng.

## Memory files appended
```
agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/00_context.md
agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/01_requirements.md
agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/02_plan.md
agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/03_implementation.md
agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/05_progress.md
agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/08_tasks.md
agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/09_tasks_solution.md
agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26/report_first_snapshot_no_write_2026-05-26.md (file này)
agent/memory/global/lessons.md (sẽ append)
```

## Skills sử dụng
- Read / Edit / Write / Bash
- Codebase tracing (event_handler → snapshot_runner → metadata_registry → batch_buffer → schema_adapter)
- Lessons retrieval (grep + read)
- TaskCreate / TaskUpdate (progress tracking)
- go build + go vet + go test (verify pipeline)
- Memory file APPEND-only (§11 GEMINI.md)
- Workspace prefix structure (§7 GEMINI.md)
