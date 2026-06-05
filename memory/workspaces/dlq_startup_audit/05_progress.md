# 05 — Progress Log (APPEND ONLY — không sửa/xoá)

## 2026-05-28T17:30+07 [Muscle:claude-opus-4-7] Init workspace
- User cung cấp log 33 dòng `dlq state machine replayed message` trong 103ms khi startup.
- Yêu cầu: audit, KHÔNG fix.
- Action: đọc `lessons.md` (#820, #866, #989), `GEMINI.md`, code `dlq_state_machine.go`, `dlq_handler.go`, `worker_server.go`.

## 2026-05-28T17:32+07 [Muscle:claude-opus-4-7] Evidence collected
- File `00_context.md` created (scope + evidence).
- File `01_requirements_audit.md` created (acceptance criteria).
- File `02_plan_audit.md` created (6 phase plan).
- File `03_implementation_audit.md` created (architecture map, 5-whys, kết luận).
- File `09_tasks_solution_audit.md` created (4 option đề xuất, kèm demo patch).

## 2026-05-28T17:34+07 [Muscle:claude-opus-4-7] Conclusion
- Burst log = expected catch-up behavior, KHÔNG phải bug.
- Identify 4 bug surface phụ (B1-B4) ghi nhận trong `02_plan_audit.md` để user quyết riêng.
- KHÔNG sửa source code (đúng yêu cầu user "audit").
- Report ra `/Users/trainguyen/Documents/work/data-hub/report_dlq_startup_log_spam.md`.

## 2026-05-28T17:45+07 [Muscle:claude-opus-4-7] USER CORRECTION — apply Option 1
- User feedback: "log bắn tùm lum mà ko mang lại giá trị nó là bug của log. cãi cãi cái gì".
- Em chấp nhận framing này: log spam không value = bug, độc lập với severity.
- **Applied Option 1** (log hygiene) tại `centralized-data-service/internal/handler/dlq_state_machine.go`:
  - Add `logDebug` helper, add `replayStatus` enum (skipped/replayed/exhausted/scheduled).
  - `retryOne` return `replayStatus` thay vì void.
  - `RunOnce` aggregate counters + 1 INFO `dlq cycle finished polled=N replayed=X exhausted=Y scheduled=Z skipped=W` ở cuối.
  - `polled=0` → silent (không log INFO).
  - Per-message `dlq state machine replayed message` INFO → Debug `dlq replayed id=… subject=… retry=…`.
- Verify: `go build ./internal/handler/` PASS. `go test -count=1 -short -run "TestDLQ" ./test/internal/handler/...` → ok (0.834s).

## 2026-05-28T17:48+07 [Muscle:claude-opus-4-7] SigNoz body=msg inline pattern
- User ask thêm: SigNoz UI chỉ hiện "title" (= body), fields phải click detail mới thấy.
- Root cause: `cmd/worker/main.go:42-91` zap.NewProduction + otelzap bridge → msg → OTel body, fields → attributes; SigNoz UI default chỉ render body column.
- Fix: inline key context trong `msg` (kèm zap.Field giữ nguyên để query attribute).
- Applied khắp dlq_state_machine.go: tất cả log msg dùng `fmt.Sprintf` với id/subject/retry/err inline.
- Lesson ghi vào `agent/memory/global/lessons.md` (2 phần: log spam = bug + SigNoz body=msg pattern).

## 2026-05-29T10:30+07 [Muscle:claude-opus-4-7] Extend pattern → 3 hot path files
- User: "làm đi" → scope rộng ra ngoài dlq_state_machine.
- Plan doc: `02_plan_log_hygiene_extend.md` — chọn top 3 file high-impact.
- **Applied pattern** (msg dùng `fmt.Sprintf` inline + giữ zap.Field):
  - `internal/handler/dlq_handler.go` (12 logs) — 6 calls transformed (HandleWithRetryContext, sendToDLQ, markPublishFailure, ReplayDLQ).
  - `internal/handler/kafka_consumer.go` (30 logs) — 12 calls transformed (Start, RefreshTopics, fetch error path, discoverTopics, getAvroCodec, runPostConsumeAction).
  - `internal/server/worker_server.go` (45 logs) — 27 calls transformed via subagent (PostgreSQL connected, NATS streams, MongoDB client, Reconciliation Core, consumer pools, schedule poller, CDC Worker started, reconcile cycle...).
- **Verify**:
  - `go build ./internal/handler/ ./internal/server/ ./cmd/worker/` → PASS.
  - `go test -count=1 -short ./test/internal/handler/...` → ok 4.088s.
  - Không có test file đụng log msg literal (đã grep trước khi edit).
- **Skip cố ý**: `command_handler.go` (108 logs), `recon_handler.go` (38 logs), `snapshot_runner_handler.go` (28 logs) — command-driven hoặc cycle-driven, để Phase sau nếu user yêu cầu.

## 2026-05-29T15:10+07 [Muscle:claude-opus-4-7] Phase 5 — Tech depth pattern (component/op/duration_ms/err_type)
- User feedback: "log nó phải có hướng tech chứ. để còn biết mà debug. kiểu thông báo thôi vậy" → inline msg chưa đủ, cần technical anchors cho debugging.
- **Pattern added to msg + zap.Field**:
  - `component=<service_module>` (worker_server, kafka_consumer, dlq_state_machine, dlq_handler).
  - `op=<operation_name>` (pg_init, mongo_init, fetch_message, commit, reconcile_cycle, schedule_exec, retry, send_to_dlq...).
  - `phase=<lifecycle>` (started/completed/skipped/transient/fatal/paused/fallback/idle).
  - `duration_ms` cho mọi operation timed (init_duration_ms, fetch_duration_ms, cycle_duration_ms, commit_duration_ms, refresh_duration_ms, action_duration_ms, db_duration_ms, publish_duration_ms).
  - `err_type` taxonomy (classifyDLQErr / classifyKafkaErr): ctx_deadline_exceeded, ctx_canceled, nats_timeout, nats_conn_closed, pg_sqlstate, net_conn_refused, kafka_not_leader, kafka_request_timeout, kafka_rebalance, schema_not_found, timeout, io_eof, unknown.
  - Resource counters: payload_bytes, topic_count, broker_count, partition, offset, reader_lag, batch_size, schema_bytes, run_count, throughput_msg_per_sec.
- **Files updated tech depth**:
  - `dlq_state_machine.go` — Start, RunOnce, retryOne với cycle/poll/db/publish duration + err_type + subject_source + payload_bytes + target_table + backoff.
  - `dlq_handler.go` — HandleWithRetryContext, sendToDLQ, markPublishFailure, ReplayDLQ với attempt/phase/source_table/payload_bytes/db_duration_ms/err_type.
  - `kafka_consumer.go` — `classifyKafkaErr` helper added (lines 1280-1320). Start (component/broker_count/schema_registry/adaptive_enabled), discoverTopics (raw_topic_count/topic_count/broker_count), fetch loop (partition/offset/reader_lag/fetch_duration_ms/err_type, phase=transient|fatal|reader_closed), commit (commit_duration_ms/err_type), processing failed (process_duration_ms/payload_bytes), DLQ write failed (payload_bytes/err_type), RefreshTopics (refresh_duration_ms/delta/old_count/new_count/phase), getAvroCodec (fetch_duration_ms/registry_url/schema_bytes/cache_size), runPostConsumeAction (batch_duration_ms/action_duration_ms/throughput_msg_per_sec).
  - `worker_server.go` — PostgreSQL connected (init_duration_ms), read-replica init, NATS streams (subject_count), MongoDB client (init_duration_ms/mode/features_disabled), kafka consumer started (broker_count/prefix_count/adaptive_enabled/batch_flush_size), nats-triggered refresh (refresh_duration_ms), schedule poller started (enabled_count/lock_backend), schedule_exec (operation/interval_min/first_run/target_table/run_count), CDC Worker started (port/consumer_pools/feature flags), reconcile_cycle (phase=started|completed|skipped, tables_checked/drift_detected/error_count/cycle_duration_ms, err_type=wiring_regression), Shutdown (consumer_pools/shutdown_duration_ms/phase).
- **Verify**:
  - `go build ./internal/handler/ ./internal/server/ ./cmd/worker/` → PASS.
  - `go test -count=1 -short ./test/internal/handler/...` → ok 3.806s.
- **Why**: SigNoz body column giờ chứa technical anchors. Operator có thể grep `err_type=nats_timeout`, filter `component=kafka_consumer op=fetch_message`, sort by `duration_ms`, đọc `phase=fatal` mà không cần click detail panel.

