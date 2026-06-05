# 03 — Implementation Detail: DLQ State Machine Behavior Analysis

## Architecture map
```
worker_server.go:688           dlq_state_machine.go:61
  go s.dlqWorker.Start(ctx) ─▶ Start(ctx)
                                ├─ logInfo("dlq state machine started")
                                ├─ RunOnce(ctx)        ◀── IMMEDIATE catch-up
                                └─ for { tick(PollInterval=5m) → RunOnce(ctx) }

RunOnce(ctx)  dlq_state_machine.go:84
  ├─ guard: db & nats.Conn != nil
  ├─ SELECT * FROM cdc_system.failed_sync_logs
  │     WHERE status IN ('pending','failed','retrying')
  │       AND (next_retry_at IS NULL OR next_retry_at <= NOW())
  │       AND retry_count < MaxRetries(=3)
  │     ORDER BY next_retry_at NULLS FIRST, id
  │     LIMIT BatchSize(=100)
  └─ for row in rows: retryOne(row)

retryOne(row)  dlq_state_machine.go:116
  ├─ UPDATE status='retrying', last_retry_at=NOW
  ├─ subject = row.KafkaTopic || DLQSubject("cdc.dlq")
  ├─ publish = nats.Conn.Publish(subject, row.RawJSON)   ← fire & forget
  ├─ if publish OK:
  │     UPDATE status='resolved', retry_count++, resolved_at=NOW, next_retry_at=NULL
  │     logInfo("dlq state machine replayed message")   ← BURST LOG SOURCE
  └─ if publish FAIL:
        if retry_count++ >= MaxRetries: UPDATE status='dead_letter' + logWarn("replay exhausted")
        else: UPDATE next_retry_at = NOW + backoff(retry_count) + logWarn("scheduled replay retry")
```

## Trigger sequence khi user thấy burst
1. Service start (t=0)
2. `worker_server.go:688` schedules `dlqWorker.Start` ngoài go-routine
3. `Start()` logs "dlq state machine started"
4. `Start()` gọi `RunOnce(ctx)` ngay
5. `RunOnce` lấy ≤100 rows due
6. Loop publish + UPDATE per row → mỗi success → 1 `logInfo("dlq state machine replayed message")`
7. User thấy 33 dòng trong ~103ms = throughput ~320 rows/s (publish + UPDATE đơn giản, latency ms-level)

## Vì sao có 33 rows ở `next_retry_at <= NOW()` ngay startup?
3 nguồn chính (lesson #989 đã cảnh báo về cron-driven replayer):
- **Source A — Backlog từ downtime trước**: Service down N phút, các entry với `next_retry_at` đã expire trong cửa sổ đó tích lũy. Start → tất cả pick up cùng cycle.
- **Source B — Initial seed**: Entry với `next_retry_at IS NULL` (pending lần đầu) — query có `NULLS FIRST` → tất cả new entries lên top.
- **Source C — Previous cycle dropped**: Nếu cycle trước fail (context timeout, db error) → rows vẫn ở status `retrying` với `next_retry_at` cũ → cycle mới sẽ pick lại.

Cả 3 đều là **expected catch-up**, không phải race condition.

## Phân loại theo lesson framework
- **Lesson #820 (silent degradation)**: Log INFO chứ không ERROR/WARN → KHÔNG vi phạm "startup log clean". Service vẫn healthy theo định nghĩa cũ.
- **Lesson #866 (symptom vs upstream)**: Symptom = log spam. Upstream = (a) catch-up design, (b) log-per-message thiết kế. Cả 2 đều là **design intent**, không phải bug.
- **Lesson #989 (cron-driven replayer)**: Cảnh báo về scheduled job depending on lazily-init dep — guard `db == nil || nats.Conn == nil` ở `dlq_state_machine.go:85-88` đã có, log WARN "skipped due to missing dependency". OK.

## Kết luận audit
| Câu hỏi | Trả lời |
|--------|---------|
| Burst log có phải bug? | **Không.** Đây là expected catch-up; log INFO không là dấu hiệu degraded. |
| Cơ chế? | `Start()` chạy `RunOnce()` ngay → query tất cả rows due → log INFO/row. |
| Cải thiện đề xuất? | Có 4 option (xem `09_tasks_solution_audit.md`); chỉ apply nếu user approve. |

## Code references
- `centralized-data-service/internal/handler/dlq_state_machine.go:61-82` — Start()
- `centralized-data-service/internal/handler/dlq_state_machine.go:84-114` — RunOnce()
- `centralized-data-service/internal/handler/dlq_state_machine.go:116-198` — retryOne()
- `centralized-data-service/internal/handler/dlq_state_machine.go:150-154` — log spam source
- `centralized-data-service/internal/handler/dlq_handler.go:19-20` — DLQSubject, MaxRetries
- `centralized-data-service/internal/server/worker_server.go:687-689` — go-routine startup
