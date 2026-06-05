# 09 — Proposed Solutions (chờ user approve)

Cả 4 option là **không bắt buộc**. Burst log không phải bug — đây là log hygiene / hardening đề xuất.

## Option 1 — Log hygiene (RECOMMENDED, risk thấp)
**Vấn đề**: 1 INFO/message → spam khi backlog lớn, giảm signal-to-noise của startup log.

**Giải pháp**:
- Downgrade per-message log `dlq state machine replayed message` từ INFO → Debug.
- Thêm 1 INFO "dlq state machine cycle finished" ở cuối `RunOnce` với aggregate counters: `replayed`, `exhausted`, `scheduled`, `errors`.

**Demo patch** (chưa apply, đợi approve):
```go
// dlq_state_machine.go: RunOnce
func (sm *DLQStateMachine) RunOnce(ctx context.Context) {
    // ... existing guard + query ...

    var replayed, exhausted, scheduled int
    for _, row := range rows {
        if ctx.Err() != nil { return }
        status := sm.retryOne(ctx, row) // refactor return enum
        switch status {
        case replayStatusOK:        replayed++
        case replayStatusExhausted: exhausted++
        case replayStatusScheduled: scheduled++
        }
    }
    sm.logInfo("dlq state machine cycle finished",
        zap.Int("polled", len(rows)),
        zap.Int("replayed", replayed),
        zap.Int("exhausted", exhausted),
        zap.Int("scheduled", scheduled),
    )
}

// retryOne return enum thay vì void
func (sm *DLQStateMachine) retryOne(ctx context.Context, row model.FailedSyncLog) replayStatus {
    // ... existing flow ...
    if publishErr == nil {
        // UPDATE resolved
        sm.logger.Debug("dlq state machine replayed message", // ← INFO → Debug
            zap.Uint64("id", row.ID), ...)
        return replayStatusOK
    }
    // ...
}
```

**Files thay đổi**: `dlq_state_machine.go` (~ +15 / -5 LOC).
**Tests cần update**: `test/internal/handler/dlq_*_test.go` nếu assert log message.
**Risk**: Thấp. Per-message vẫn xem được khi bật log level Debug.

---

## Option 2 — Startup delay (KHÔNG khuyến nghị standalone)
**Vấn đề**: Burst hiện ngay lúc start làm operator confuse với init error.

**Giải pháp**: Thay `Start()` chạy `RunOnce` ngay → đợi N giây (ví dụ 30s, theo pattern `worker_server.go:707` đã có `time.Sleep(30 * time.Second)` cho schedule executor).

**Demo patch**:
```go
func (sm *DLQStateMachine) Start(ctx context.Context) {
    sm.logInfo("dlq state machine started", ...)
    select {
    case <-ctx.Done(): return
    case <-time.After(sm.cfg.StartupDelay): // mặc định 30s
    }
    sm.RunOnce(ctx)
    // ticker loop ...
}
```

**Risk**: Trì hoãn replay của backlog → tăng latency phục hồi DLQ. KHÔNG khuyến nghị nếu không kèm Option 1.

---

## Option 3 — Concurrency safety (RECOMMENDED nếu chạy multi-instance)
**Vấn đề** (B1 trong `02_plan_audit.md`): Query không SKIP LOCKED → 2 instance cùng pick 1 row → double publish.

**Giải pháp**: Wrap SELECT trong txn + `FOR UPDATE SKIP LOCKED`.
```sql
BEGIN;
SELECT * FROM cdc_system.failed_sync_logs
 WHERE status IN ('pending','failed','retrying')
   AND (next_retry_at IS NULL OR next_retry_at <= NOW())
   AND retry_count < $1
 ORDER BY next_retry_at NULLS FIRST, id
 LIMIT $2
 FOR UPDATE SKIP LOCKED;
-- per row: UPDATE ... WHERE id = ?
COMMIT;
```

**Risk**: Trung bình. Cần test migration index `idx_fsl_retry_poll` còn hit. Tăng lock window cycle.

---

## Option 4 — Publish acknowledgement (B2+B3 fix)
**Vấn đề** (B2, B3): `nats.Conn.Publish` fire-and-forget, không Flush → có thể loss msg + race với UPDATE `resolved`.

**Giải pháp**: 
- Dùng `nats.Conn.PublishMsg` + check `Conn.Flush` sau publish trước UPDATE resolved.
- Hoặc dùng JetStream `js.Publish` (acked) thay vì core NATS.

**Risk**: Cao. Đụng tới messaging contract — cần regression test full DLQ→consumer chain.

---

## Đề xuất combo
- **Apply ngay**: Option 1 (log hygiene). Vô hại, giảm noise rõ rệt.
- **Cân nhắc nếu multi-instance**: Option 3.
- **Để Phase sau**: Option 2, Option 4 (đụng latency / messaging contract).

User quyết approve option nào → tôi mở workspace mới `dlq_log_hygiene_fix` (hoặc tương tự), tạo `01_requirements`, `02_plan`, ... thực thi đúng pattern GEMINI.md §7.
