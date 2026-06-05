# 02 — Plan: DLQ Startup Audit

## Phase 1 — Evidence collection (đã làm)
1. Đọc `internal/handler/dlq_state_machine.go` toàn file (237 LOC).
2. Đọc `internal/server/worker_server.go:240-260,680-710` (init + go-routine start).
3. Đọc `internal/handler/dlq_handler.go` cho constants (`MaxRetries=3`, `DLQSubject="cdc.dlq"`).
4. Confirm log message exact format `dlq_state_machine.go:150`.

## Phase 2 — Root cause walk (5-whys, lesson #866)
| # | Why | Evidence |
|---|-----|----------|
| 1 | Tại sao thấy burst log? | 33 INFO logs `dlq state machine replayed message` trong 103ms. |
| 2 | Tại sao nhiều record cùng được replay? | `RunOnce()` query `LIMIT BatchSize=100`; mỗi row pass điều kiện thì publish + log INFO. |
| 3 | Tại sao tất cả 33 record đều "due" lúc startup? | Query filter `next_retry_at IS NULL OR next_retry_at <= NOW()`. Backlog tích lũy giữa 2 cycle hoặc downtime → tất cả thoả. |
| 4 | Tại sao chạy ngay khi service start? | `Start()` `dlq_state_machine.go:68` gọi `sm.RunOnce(ctx)` IMMEDIATELY trước khi vào ticker loop. Đây là "catch-up at boot" design. |
| 5 | Tại sao 1 INFO/record (không aggregate)? | Design choice — observability per-message để truy trace từng message replayed. Trong scenario backlog → log dày đặc. |

## Phase 3 — Bug surface scan (orthogonal nhưng cần ghi nhận)
Khi đọc code, ghi nhận các bug tiềm năng KHÔNG liên quan trực tiếp burst log:
- **B1**: Query `dlq_state_machine.go:95-102` không `FOR UPDATE SKIP LOCKED` → multi-instance race có thể double-publish.
- **B2**: Publish + UPDATE `resolved` không atomic (`dlq_state_machine.go:137-149`). Crash giữa → status `retrying`, vòng sau replay lại → at-least-once duplicate.
- **B3**: `nats.Conn.Publish` không Flush — nếu service shutdown ngay sau publish, msg có thể không tới server (silent loss).
- **B4**: `subject = DLQSubject` fallback khi `KafkaTopic == ""` (`dlq_state_machine.go:132-135`) → replay vào `cdc.dlq` rồi đọc lại → loop tiềm năng nếu chain consumer của `cdc.dlq` cũng đi qua DLQ.

→ B1-B4 KHÔNG là root cause của burst log user thấy. Ghi nhận trong report để user quyết riêng.

## Phase 4 — Options (đề xuất, KHÔNG tự apply)
Liệt kê chi tiết tại `09_tasks_solution_audit.md`.

## Phase 5 — Verify
Vì audit-only, "verify" = re-read code, confirm refs, không cần chạy service.
Nếu user approve fix sau này → workspace mới, plan riêng, chạy `go test ./test/...` trước khi báo done.

## Phase 6 — Report
- Tạo `report_dlq_startup_log_spam.md` ở root project (`/Users/trainguyen/Documents/work/data-hub/`).
- Files changed: 0 source code (audit-only). Workspace doc count + LOC ghi rõ trong report.
