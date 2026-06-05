# 05_progress — APPEND-ONLY

---

## [2026-05-26 16:30] [Agent:claude-opus-4-7] Brain plan complete

**Actor**: Brain.
**Trigger**: User verb `plan trace`.

**Pre-work**:
- ✅ Audit `internal/handler/kafka_consumer.go` (line 1-457): chỉ có 1 span `kafka.consume` (line 382), KHÔNG có child span trong toàn `internal/`.
- ✅ Identify gaps: business attrs thiếu (cdc.source_table, cdc.engine, cdc.operation, cdc.rows_affected); log call site không inject trace_id; error path dùng attribute.String thay vì RecordError; flushBatch ctx tách spanCtx.
- ✅ ADR-01..10 quyết định.
- ✅ Effort estimate 3h25m / M1..M8.

**Files created (workspace, KHÔNG đụng source code)**:
- `00_context.md` — bối cảnh, audit, scope, risks
- `01_requirements.md` — R1-R8 + N1-N6 + DoD A1-A6 + inverse req
- `02_plan.md` — M0..M8 roadmap với code demo chi tiết
- `04_decisions.md` — ADR-01..10
- `08_tasks.md` — T1.1..T8.6
- `09_tasks_solution.md` — reference snippet per task

**Status**: Plan-only. Chờ user verb để Muscle thực thi.

**Next verb chờ user**:
- `execute` / `thực thi` / `go` → Muscle chạy M1 → M8.
- `revise` → re-plan section cụ thể.
- `defer` → archive plan.
- `subset M1+M2` → chỉ làm 1 phần.

---

## [2026-05-26 17:45] [Agent:claude-opus-4-7] Muscle execute M1→M8 hoàn tất

**Actor**: Muscle.
**Trigger**: User verb `thực hiện`.

**Done**:
- ✅ M1 — `pkgs/observability/trace_helpers.go` (77 LOC: EndSpan, ParseDebeziumTopic, SourceTableAttr, ChildSpan). Build EXIT=0.
- ✅ M2 — `kafka_consumer.go`: thêm 3 attrs business (`cdc.source_table`, `cdc.engine`, `cdc.source_db`), RecordError + SetStatus thay cho `attribute.String("error",...)`, `cdc.rows_affected` success path, child span `cdc.process_message`, 3 log site migrate. Build EXIT=0.
- ✅ M3 — `event_handler.go`: HandleRaw child span `cdc.event_handle` + named returns, 2 log site migrate. Build EXIT=0.
- ✅ M4 — `batch_buffer.go`: batchUpsert root span `cdc.batch_upsert` (qua `context.Background()`, ADR-02), 2 log site migrate. Build EXIT=0.
- ✅ M5 — `schema_inspector.go`: InspectEvent child span `cdc.schema_inspect` + `cdc.new_field_count` attr, 1 log site migrate. Build EXIT=0.
- ✅ M6 — 9 log site critical đã dùng `observability.Ctx(ctx, logger)`.
- ✅ M7 — `pkgs/observability/trace_helpers_test.go` (4 test PASS: TestParseDebeziumTopic 5 subtest, TestChildSpan_ParentChild, TestEndSpan_RecordsError, TestEndSpan_NilSafe).
- ✅ M8 — Full repo: `go build ./...` EXIT=0, `go vet ./...` EXIT=0, `go test -short ./internal/...` EXIT=0, `go test ./pkgs/observability/...` EXIT=0. Report `report_trace_span_attrs_2026-05-26.md` đã tạo (13 section + pre-flight §14).

**Files thay đổi**:
- New: `pkgs/observability/trace_helpers.go`, `pkgs/observability/trace_helpers_test.go`.
- Sửa: `internal/handler/kafka_consumer.go`, `internal/handler/event_handler.go`, `internal/handler/batch_buffer.go`, `internal/service/schema_inspector.go`.

**Logs**: `/tmp/cdc-trace-verify/{build,vet,test_observability,test_internal}.log` — all EXIT=0.

**Status**: ✅ DONE. Phase 2 trace + log correlation hoàn tất. Lesson sẽ append `agent/memory/global/lessons.md`.

**Next verb chờ user**:
- `deploy` → user tự deploy + verify SigNoz UI.
- `mass migrate logs` → phase 3.
- `trace gorm` → bổ sung gorm callback span.
- `tune sample` → sau khi có baseline storage.

---
