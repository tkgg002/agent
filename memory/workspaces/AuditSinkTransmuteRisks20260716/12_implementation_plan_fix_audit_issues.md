# Plan Fix Issues Audit Sink & Transmute

> **Nguồn:** [Báo cáo audit](file:///Users/trainguyen/.gemini/antigravity/brain/14e12fe7-f980-4c06-bf4d-85ed1b1d4d4d/audit_sink_transmute_risks.md) — 40 rủi ro, 7 Critical
> **Nguyên tắc:** Stop the bleeding first — chặn data loss production trước, rồi mới cải thiện

---

## Phase 0 — Chặn data loss production (tuần này)

> [!CAUTION]
> 5 tasks dưới đây fix các lỗi **ĐANG gây mất data ở production mỗi lần deploy/restart**. Ưu tiên tuyệt đối.

---

### P0-1: Fix `CommitInterval: 0` — Chặn auto-commit trước xử lý

| | |
|---|---|
| **Risk** | SINK-C1 (Critical) |
| **File** | [kafka_consumer.go:151](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L151) |
| **Effort** | 1 dòng code |
| **Impact** | Chặn auto-commit offset trước khi data xử lý xong |

**Hiện tại:**
```go
// kafka_consumer.go:151
CommitInterval: time.Second,  // ← auto-commit mỗi 1s bất kể xử lý xong chưa
```

**Fix:**
```diff
- CommitInterval:   time.Second,
+ CommitInterval:   0, // Manual commit only — commit sau khi processMessage thành công
```

**Giải thích:** `CommitInterval: 0` buộc `segmentio/kafka-go` chỉ commit khi code gọi `CommitMessages()` tường minh (đã có ở line 474). Khi crash giữa chừng, offset chưa commit → restart sẽ re-process messages → **at-least-once** thay vì at-most-once.

**Lưu ý:** Fix này CẦN đi kèm P0-2 để đảm bảo commit chỉ xảy ra SAU khi data đã flush vào DB.

---

### P0-2: Flush BatchBuffer TRƯỚC commit offset — Đảo thứ tự commit

| | |
|---|---|
| **Risk** | SINK-C2 (Critical) |
| **File** | [kafka_consumer.go:345-489](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L345) (consume loop) |
| **Effort** | ~50 LOC refactor |
| **Impact** | Đóng cửa sổ data loss 5s giữa commit và flush |

**Vấn đề hiện tại:**
```
Loop: FetchMessage → processMessage (add to buffer) → CommitMessages
                                                          ↑ commit NGAY
Buffer: ─────────── chờ 5s hoặc đầy ────────────► Flush to DB
                                                      ↑ data ghi DB SAU
```

**Fix — Chiến lược: Accumulate offsets, commit SAU flush:**

```go
// Thêm struct track pending offsets
type pendingCommit struct {
    reader  *kafka.Reader
    offsets []kafka.Message
}

// Trong consume loop — KHÔNG commit ngay sau processMessage
// Thay vào đó, collect offsets và commit sau khi batch flush
```

**Flow mới:**
```
Loop: FetchMessage → processMessage (add to buffer) → accumulate offset
                                                          ↓
Buffer đầy hoặc timer 5s → Flush to DB → thành công → CommitMessages(accumulated)
                                        → thất bại → KHÔNG commit → restart sẽ re-process
```

**Cách implement cụ thể:**

1. Trong consume loop (line 345-490): Bỏ block `CommitMessages` ở line 473-489
2. Thêm callback vào `BatchBuffer.Flush()` — sau khi flush thành công, gọi `commitPendingOffsets()`
3. Track `map[topicPartition]kafka.Message` — chỉ giữ highest offset per partition
4. Khi flush fail → không commit → messages sẽ được re-deliver

**Verification:** Unit test simulate crash giữa processMessage và flush → assert messages re-delivered.

---

### P0-3: Log + Metrics cho 4 silent drop points — Biết data mất ở đâu

| | |
|---|---|
| **Risk** | SINK-H1, SINK-H2, SINK-H3, SINK-H4 |
| **Files** | [kafka_consumer.go:502](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L502), [kafka_consumer.go:569](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L569), [event_handler.go:164](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L164), [event_handler.go:232](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L232) |
| **Effort** | ~30 LOC |
| **Impact** | Visibility — biết chính xác bao nhiêu data bị drop và tại đâu |

**4 điểm cần thêm Warn log + metrics counter:**

**Drop 1 — Empty value (line 502):**
```diff
  if len(value) == 0 {
+     kc.logger.Warn("kafka message empty value, dropping",
+         zap.String("topic", msg.Topic),
+         zap.Int("partition", msg.Partition),
+         zap.Int64("offset", msg.Offset))
+     metrics.EventsDropped.WithLabelValues("empty_value", msg.Topic).Inc()
      return 0, nil
  }
```

**Drop 2 — Nil afterData non-delete (line 569):**
```diff
  if afterData == nil && opStr != "d" {
-     kc.logger.Debug("kafka message has no 'after' data, skipping",
+     kc.logger.Warn("kafka message has no 'after' data, dropping (non-delete)",
          zap.String("topic", msg.Topic),
          zap.String("op", opStr),
+         zap.Int("partition", msg.Partition),
+         zap.Int64("offset", msg.Offset),
      )
+     metrics.EventsDropped.WithLabelValues("nil_after_data", msg.Topic).Inc()
      return 0, nil
  }
```

**Drop 3 — Source not registered (line 164-177):** Đã có Warn log, chỉ cần thêm metrics:
```diff
  // event_handler.go:170 — sau Warn log hiện có
+ metrics.EventsDropped.WithLabelValues("source_not_registered", sourceTable).Inc()
  return 0, nil
```

**Drop 4 — Missing PK (line 232-237):** Đã có Warn log, chỉ cần thêm metrics:
```diff
  // event_handler.go:233 — sau Warn log hiện có
+ metrics.EventsDropped.WithLabelValues("missing_pk", sourceTable).Inc()
  return 0, nil
```

**Metrics mới cần khai báo:**
```go
// metrics/sink_metrics.go
var EventsDropped = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "cdc_sink_events_dropped_total",
        Help: "Total CDC events dropped by reason",
    },
    []string{"reason", "topic"},
)
```

---

### P0-4: Thêm `recover()` trong transmute goroutine — Chặn panic crash

| | |
|---|---|
| **Risk** | TX-H2 (High) |
| **File** | [transmute_handler.go:204](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L204) |
| **Effort** | ~15 LOC |
| **Impact** | Panic không làm schedule stuck vĩnh viễn |

**Hiện tại — goroutine không có recover:**
```go
// transmute_handler.go:204
go func() {
    defer span.End()
    // ... nếu panic ở đây → goroutine chết, schedule row stuck "running" vĩnh viễn
}()
```

**Fix:**
```diff
  go func() {
+     defer func() {
+         if r := recover(); r != nil {
+             stack := string(debug.Stack())
+             h.logger.Error("transmute goroutine panic recovered",
+                 zap.Any("panic", r),
+                 zap.String("master", req.MasterTable),
+                 zap.String("stack", stack))
+             if logEntry != nil {
+                 h.activity.Fail(logEntry, fmt.Sprintf("PANIC: %v", r))
+             }
+             metrics.TransmuteErrorTotal.WithLabelValues(req.MasterTable).Inc()
+         }
+     }()
      defer span.End()
```

---

### P0-5: Fix bare type assertion — Chặn panic cascade từ dedup

| | |
|---|---|
| **Risk** | TX-H5 (High) |
| **File** | [transmuter.go:655, 663-664](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L655) |
| **Effort** | ~10 LOC |
| **Impact** | Dedup không panic khi `_gpay_id` hoặc `_source_ts` type không đúng |

**Hiện tại — bare assertion, panic nếu type sai:**
```go
// transmuter.go:655
gpayID := rec["_gpay_id"].(int64)     // ← PANIC nếu nil hoặc không phải int64
// transmuter.go:663-664
currentTs := rec["_source_ts"].(int64)  // ← PANIC
storedTs := allRecords[storedIdx]["_source_ts"].(int64) // ← PANIC
```

**Fix — comma-ok pattern:**
```diff
- gpayID := rec["_gpay_id"].(int64)
+ gpayID, ok := rec["_gpay_id"].(int64)
+ if !ok {
+     t.logger.Warn("dedup: _gpay_id type assertion failed, skipping",
+         zap.Any("_gpay_id", rec["_gpay_id"]))
+     continue
+ }

- currentTs := rec["_source_ts"].(int64)
- storedTs := allRecords[storedIdx]["_source_ts"].(int64)
+ currentTs, ok1 := rec["_source_ts"].(int64)
+ storedTs, ok2 := allRecords[storedIdx]["_source_ts"].(int64)
+ if !ok1 || !ok2 {
+     bestIndices[gpayID] = idx // fallback: lấy bản mới nhất
+     continue
+ }
```

---

## Phase 1 — Sprint tiếp theo

---

### P1-1: Retry logic cho `bulkUpsertMaster` — Không skip batch khi transient error

| | |
|---|---|
| **Risk** | TX-C1 (Critical) |
| **File** | [transmuter.go:700-704](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L700) |
| **Effort** | ~50 LOC |

**Fix:** Exponential backoff, max 3 retries. Chỉ skip nếu non-retryable error (constraint violation).

```go
// transmuter.go:700 — thay khối if err != nil
maxRetries := 3
for attempt := 0; attempt <= maxRetries; attempt++ {
    ins, upd, occSkip, err := t.bulkUpsertMaster(ctx, binding, allRecords[i:end])
    if err == nil {
        out.inserted += ins
        out.updated += upd
        out.occSkipped += occSkip
        out.skipped += occSkip
        break
    }
    if !isRetryableDBError(err) || attempt == maxRetries {
        t.logger.Error("bulk upsert failed permanently",
            zap.String("master", binding.MasterTable),
            zap.Int("attempt", attempt+1),
            zap.Error(err))
        out.skipped += int64(end - i)
        break
    }
    backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
    time.Sleep(backoff)
}
```

---

### P1-2: NATS Subscribe → QueueSubscribe — Chặn duplicate processing

| | |
|---|---|
| **Risk** | TX-H1 (High) |
| **File** | [server_setup.go:282-283](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L282) |
| **Effort** | 1 dòng code |

```diff
- nc.Subscribe("cdc.cmd.transmute", handler.HandleNATS)
+ nc.QueueSubscribe("cdc.cmd.transmute", "transmute-workers", handler.HandleNATS)
```

---

### P1-3: Log chi tiết khi transmute rules bị filter

| | |
|---|---|
| **Risk** | TX-C3 (Critical) |
| **File** | [transmuter.go:425-438](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L425) |
| **Effort** | ~10 LOC |

Thêm Warn log + metrics cho 2 case đang silent skip (transform_fn không whitelist, data_type invalid).

---

### P1-4: Default value cho non-nullable rules khi field miss

| | |
|---|---|
| **Risk** | TX-C4 (Critical) |
| **File** | [transmuter.go:791-793](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L791) |
| **Effort** | ~20 LOC |

Khi field miss + `is_nullable=false` + `default_value != nil` → dùng default thay vì drop record. Cần review business logic với User.

---

### P1-5: Fix DLQ write error swallow

| | |
|---|---|
| **Risk** | SINK-H6 (High) |
| **File** | [dlq_helper.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/dlq_helper.go) |
| **Effort** | ~10 LOC |

Thay `_` ignore bằng log Error + metrics `cdc_dlq_write_fail_total`.

---

## Phase 2 — Cải thiện kiến trúc (sprint +2)

| # | Action | Risk | Effort |
|---|--------|------|--------|
| P2-1 | Implement Concurrency Optimization (thiết kế 34KB đã có) | Sequential flush, Connection Storm | ~500 LOC |
| P2-2 | Flatten orphan cleanup — soft-delete master rows khi array shrink | TX-C2 | ~100 LOC |
| P2-3 | Reconciliation tự động Kafka offset vs Shadow DB row count | Detection gap | ~200 LOC |
| P2-4 | Scheduler stuck cleanup — timeout `last_status='running'` sau 2x interval | TX-H2 complement | ~30 LOC |

---

## Verification Plan

### Automated Tests
- Unit test P0-1 + P0-2: simulate crash giữa processMessage và flush → assert re-delivery
- Unit test P0-5: dedup với `_gpay_id` type `float64`, `string`, `nil` → assert no panic
- Unit test P1-1: mock DB deadlock → assert retry 3 lần → assert thành công sau retry

### Manual Verification
- Deploy P0-1 + P0-2 lên staging → restart worker giữa chừng → kiểm tra data không mất
- Monitor metrics `cdc_sink_events_dropped_total` sau deploy P0-3 → đánh giá mức drop thực tế
- Trigger transmute trên staging → kill goroutine → verify schedule row không stuck

### Thứ tự deploy
```
P0-1 + P0-2 (PHẢI đi cùng nhau) → P0-3 → P0-4 + P0-5 → P1-*
```

> [!WARNING]
> **P0-1 không được deploy riêng lẻ.** Đổi `CommitInterval: 0` mà không fix thứ tự flush/commit (P0-2) sẽ gây commit quá trễ hoặc không commit — dẫn đến re-process lặp lại. Hai task phải đi cùng 1 PR.
