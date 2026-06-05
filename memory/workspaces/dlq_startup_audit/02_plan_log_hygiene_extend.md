# 02 — Plan Extend Log Hygiene Pattern Beyond DLQ State Machine

**Date**: 2026-05-29
**Trigger**: User "làm đi" sau khi đã apply Option 1 cho `dlq_state_machine.go` + SigNoz body=msg inline pattern.

## Mục tiêu
Áp dụng cùng pattern (msg self-descriptive với fmt.Sprintf inline key context) cho các file hot path còn lại trong `centralized-data-service`, để SigNoz body column hiển thị đủ info, không bắt operator click detail.

## Pattern cần apply

**Anti-pattern**:
```go
logger.Info("processing failed", zap.Uint64("id", x), zap.String("subject", s), zap.Error(err))
```

**Pattern đúng**:
```go
logger.Info(fmt.Sprintf("processing failed id=%d subject=%s err=%s", x, s, err.Error()),
    zap.Uint64("id", x), zap.String("subject", s), zap.Error(err))
```

**Rules**:
1. Chỉ inline key context (id, name, count, error, status). Không inline payload lớn.
2. Giữ NGUYÊN zap.Field để query attribute trên SigNoz.
3. Nếu msg đã self-descriptive (vd "server listening on :8080") → skip.
4. Per-message log trong batch loop → ép xuống `Debug`, cycle summary mới là `Info`.
5. Không touch `Fatal` — đã có context đầy đủ.

## Scope quyết định
Top 3 hot path file theo log count + visibility:
1. **`internal/handler/kafka_consumer.go`** (30 logs) — consumer loop per-msg, cao volume.
2. **`internal/server/worker_server.go`** (45 logs) — startup visibility, lifecycle.
3. **`internal/handler/dlq_handler.go`** (12 logs) — sibling DLQ state machine.

**Out-of-scope** (để Phase sau nếu user cần):
- `command_handler.go` (108) — command-driven, lower visibility loop.
- `recon_handler.go` (38) — recon cycle, sẽ apply nếu user yêu cầu.
- `snapshot_runner_handler.go` (28) — snapshot batch.
- Còn lại (< 20 logs/file) → cá nhân.

## Verify Plan
- Sau mỗi file: `go build ./internal/handler/` hoặc `./internal/server/`.
- Sau tất cả: `go test -count=1 -short ./test/internal/handler/... ./test/internal/server/...`.
- Không touch test files trừ khi test assert log msg literal (sẽ refactor assertion).

## Risk
- Diff ~150 LOC churn, có thể vỡ test assertion nếu test khớp literal log msg.
- Mitigation: grep test files xem có assert nào không trước khi edit.
