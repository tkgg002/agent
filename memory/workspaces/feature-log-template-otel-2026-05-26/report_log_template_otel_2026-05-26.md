# Report — Log Template OTel (Phase 1)
**Date**: 2026-05-26
**Workspace**: `agent/memory/workspaces/feature-log-template-otel-2026-05-26/`
**Trigger**: User báo SigNoz UI chỉ hiển thị `timestamp | message`, không có trace_id / error block / business attributes → khó debug.

---

## §1 Executive Summary
Viết bộ helper trong `pkgs/observability/log_template.go` để chuẩn hóa log thành format OTel structured (component, trace_id, span_id, error.{kind,message,stack}, attributes.*). KHÔNG migrate hàng loạt — phase này chỉ demo 3 call site đại diện 3 pattern (Error có exception, Info milestone có business attrs, Info drift có strings array). Migrate hàng loạt sẽ làm dần khi touch code.

Đây là PHASE 1 (log). PHASE 2 (trace span attributes) chờ user verb tiếp theo.

---

## §2 Files Changed (Exhaustive)

| File | LOC | Loại | Mô tả |
|------|----:|------|-------|
| `centralized-data-service/pkgs/observability/log_template.go` | **+102** | NEW | 4 helpers: `ComponentLogger`, `Ctx`, `ErrorField`, `Attrs` + encoder structs |
| `centralized-data-service/pkgs/observability/log_template_test.go` | **+125** | NEW | 6 unit test PASS |
| `centralized-data-service/internal/handler/command_handler.go` | +1 / -0 import; +8 / -2 body | Edit | Migrate 2 call site (`introspect.mongo.databases.failed` Error + `introspect.mongo.collections.ok` Info) |
| `centralized-data-service/internal/service/schema_inspector.go` | +1 / -0 import; +3 / -1 body | Edit | Migrate 1 call site (`schema drift detected (batch summary)`) |
| `agent/memory/workspaces/feature-log-template-otel-2026-05-26/00_context.md` | +47 | NEW | Bối cảnh + audit |
| `agent/memory/workspaces/feature-log-template-otel-2026-05-26/01_requirements.md` | +25 | NEW | R1-R5, N1-N4, DoD A1-A5 |
| `agent/memory/workspaces/feature-log-template-otel-2026-05-26/02_plan.md` | +118 | NEW | Roadmap + code demo + ADR |
| `agent/memory/workspaces/feature-log-template-otel-2026-05-26/05_progress.md` | append | NEW | Audit log APPEND-ONLY |
| `agent/memory/workspaces/feature-log-template-otel-2026-05-26/report_log_template_otel_2026-05-26.md` | (this file) | NEW | Báo cáo |

**Không đụng**: config files, BE, FE, DB, secrets, NATS/Kafka infra.

---

## §3 Helper API

```go
// pkgs/observability/log_template.go
func ComponentLogger(base *zap.Logger, component string) *zap.Logger
func Ctx(ctx context.Context, base *zap.Logger) *zap.Logger
func ErrorField(err error) zap.Field
func Attrs(fields ...zap.Field) zap.Field
```

### Cách dùng
```go
// 1. Subsystem-tagged logger (mỗi service/component khởi tạo 1 lần)
log := observability.ComponentLogger(h.logger, "mongo-introspect")

// 2. Per-request log với trace context
log := observability.Ctx(ctx, h.logger)

// 3. Error log với exception block + business attrs
log.Error("op.failed",
    observability.ErrorField(err),
    observability.Attrs(
        zap.String("source_table", table),
        zap.Int("batch_size", n),
    ),
)
```

### Output JSON (verified qua unit test)
```json
{
  "level": "error",
  "ts": 1716738000.123,
  "msg": "op.failed",
  "component": "mongo-introspect",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "error": {
    "kind": "*errors.errorString",
    "message": "db down",
    "stack": "goroutine 17 [running]:\n..."
  },
  "attributes": {
    "source_table": "public.orders",
    "batch_size": 500
  }
}
```

---

## §4 BEFORE / AFTER 3 Call Site

### Site 1 — `command_handler.go:1199` Error
**BEFORE**
```go
h.logger.Error("introspect.mongo.databases.failed",
    zap.String("sanitized_dsn", sanitized), zap.Error(err))
```
**AFTER**
```go
observability.Ctx(context.Background(), h.logger).Error("introspect.mongo.databases.failed",
    observability.ErrorField(err),
    observability.Attrs(
        zap.String("sanitized_dsn", sanitized),
    ),
)
```
Note: NATS handler không nhận `context.Context` → dùng `Background()`. Khi handler có ctx thật (HTTP/Kafka), pass ctx vào — Ctx() sẽ tự inject trace_id/span_id.

### Site 2 — `command_handler.go:1321` Info milestone
**BEFORE**
```go
h.logger.Info("introspect.mongo.collections.ok",
    zap.String("sanitized_dsn", sanitized),
    zap.String("database", database),
    zap.Int("count", len(cols)),
    zap.Bool("audit", true))
```
**AFTER**
```go
h.logger.Info("introspect.mongo.collections.ok",
    zap.Bool("audit", true),
    observability.Attrs(
        zap.String("sanitized_dsn", sanitized),
        zap.String("database", database),
        zap.Int("count", len(cols)),
    ),
)
```
Note: `audit` GIỮ Ở ROOT vì `severityAwareCore.Write` iterate top-level fields để detect bypass (xem ADR-01 trong 02_plan).

### Site 3 — `schema_inspector.go:162` Info drift
**BEFORE**
```go
si.logger.Info("schema drift detected (batch summary)",
    zap.String("source_db", sourceDB),
    zap.String("table", tableName),
    zap.Int("unique_new_fields", len(fields)),
    zap.Int64("events_with_drift", count),
    zap.Strings("fields", fieldNames),
    zap.Bool("audit", true),
)
```
**AFTER**
```go
si.logger.Info("schema drift detected (batch summary)",
    zap.Bool("audit", true),
    observability.Attrs(
        zap.String("source_db", sourceDB),
        zap.String("table", tableName),
        zap.Int("unique_new_fields", len(fields)),
        zap.Int64("events_with_drift", count),
        zap.Strings("fields", fieldNames),
    ),
)
```

---

## §5 Verification Evidence (KHÔNG báo láo)

| Gate | Command | Exit | Log path |
|------|---------|-----:|----------|
| Build full | `go build ./...` | **0** | /tmp/log_template_build.log (0 bytes — clean) |
| Vet full | `go vet ./...` | **0** | /tmp/log_template_vet.log (0 bytes — clean) |
| Unit test observability | `go test ./pkgs/observability/... -v -count=1` | **0** | 6/6 PASS (TestComponentLogger_TagsField, TestCtx_AttachesTraceAndSpan, TestCtx_NoSpan_ReturnsBaseUnchanged, TestErrorField_NilError_SkipsField, TestErrorField_NestedKindMessageStack, TestAttrs_NestedNamespace) |
| Regression test handler+service | `go test ./internal/handler/... ./internal/service/... -count=1 -timeout 90s` | **0** | handler 3.75s OK, service 1.43s OK |

KHÔNG có flake. KHÔNG skip test. KHÔNG dùng --no-verify.

---

## §6 SigNoz UI cần config (phía user)

Backend đã emit đúng template, nhưng SigNoz UI mặc định chỉ hiện 2 cột `timestamp + body`. User cần:

1. **Logs page → "+ Add column"** → chọn:
   - `severity_text` (INFO/WARN/ERROR)
   - `attributes.component`
   - `attributes.attributes.source_table` (vì attrs nested 1 cấp dưới `attributes` key)
   - `trace_id`
2. **Log Pipelines (nếu cần)**: nếu một subsystem khác emit text thuần (Debezium, Kafka image), dùng SigNoz Pipelines JSON Parser/Grok để parse trước khi index.
3. **Filter mẫu** trên search bar:
   ```
   service.name = "cdc-worker" AND severity_text = "ERROR"
   service.name = "cdc-worker" AND attributes.component = "mongo-introspect"
   ```
4. **Correlate log ↔ trace**: click một log có `trace_id` → SigNoz hiện nút "View Trace" → nhảy sang trace view.

---

## §7 Behavior Changes
- Log emitted to SigNoz GIỜ có nested objects: `error.{kind,message,stack}`, `attributes.{...}`. Old flat-field consumers (nếu có dashboard query `attributes.sanitized_dsn`) phải update query thành `attributes.attributes.sanitized_dsn` HOẶC tự re-query lại.
- Stdout (console branch) hiển thị JSON với nested objects — vẫn human-readable.
- Audit field `audit=true` VẪN ở root level → severityAwareCore bypass sampling đúng.

## §8 Backward Compat
- 100+ call site chưa migrate VẪN dùng flat fields như trước — không break.
- Helper là pure function, không global state, không thread-issue.
- Nếu rollback: chỉ cần git revert `log_template.go` + 3 file migrate.

---

## §9 Rollback Plan
```bash
cd centralized-data-service
git checkout HEAD -- \
  pkgs/observability/log_template.go \
  pkgs/observability/log_template_test.go \
  internal/handler/command_handler.go \
  internal/service/schema_inspector.go
```
Hoặc trong PR review: revert 1 commit duy nhất.

---

## §10 Lessons Learned Candidate
Sẽ append vào `agent/memory/global/lessons.md`:

**Global Pattern [A emits log to backend X via bridge B → observability platform Y chỉ hiển thị title]**:
- Nguyên nhân: bridge serialize flat zap.Field thành flat OTel attribute. Backend không tự gom thành nested object cho UI render.
- Đúng: dùng helper structured (Object/Inline namespace) ngay tại call site → bridge map đúng thành nested attribute group.
- Anti-pattern: dùng `zap.Error(err)` cho error log → OTel chỉ thấy 1 string field `error="..."`, không có Exception tab.

---

## §11 Open Items / Defer (KHÔNG báo done những thứ này)
- Migrate ~100+ call site khác sang template mới — defer, làm dần khi touch code.
- Phase 2: trace span attributes (`kafka.consume` thêm topic/partition/offset/batch_size + propagate trace context xuống child operations) — chờ user verb tiếp theo.
- SigNoz UI selected columns — user tự config (cần access SigNoz UI).
- Unit test cho stack truncation > 8KB — defer.

---

## §12 Next Verb chờ user
- `trace` / `phase 2` → tôi làm trace span attributes cho `kafka.consume` + các op span khác.
- `migrate <package>` → tôi migrate toàn bộ call site trong 1 package cụ thể.
- `commit` → stage + commit 4 file (1 repo).
- `done` → đóng workspace, append lesson global.

---

## §13 Pre-flight Checklist (CLAUDE.md §14)
- ✅ Workspace docs đầy đủ (00, 01, 02, 05, report — 5/5).
- ✅ Build EXIT=0 (verified, log path provided).
- ✅ Vet EXIT=0 (verified).
- ✅ Test EXIT=0 (6 helper test + 2 regression suite, đếm thực: handler 3.75s + service 1.43s).
- ✅ Không cheat config / DB.
- ✅ Có file report_*.md (file này).
- ✅ KHÔNG báo done những việc chưa làm (phase 2 trace + mass migration noted as defer).
- ✅ Brain → Muscle pipeline tuân thủ: plan → ADR → execute → verify → report.
