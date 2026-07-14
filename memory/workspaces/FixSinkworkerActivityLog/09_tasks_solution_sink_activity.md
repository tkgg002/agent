# Hồ sơ Giải pháp: Bổ sung Ghi nhận Activity Log 'sink-upsert' cho CDC Pipeline

Hồ sơ giải pháp chi tiết cho việc sửa đổi mã nguồn.

## Các file cần chỉnh sửa

### 1. File `internal/handler/shadow/batch_buffer.go`

#### A. Import `centralized-data-service/internal/model/system`
Thêm package model system vào khối import:
```go
import (
	"centralized-data-service/internal/model/shadow"
	"centralized-data-service/internal/model/system" // <-- Thêm
	"context"
...
```

#### B. Thêm ghi nhận activity log trong `batchUpsert`
Chèn logic khởi tạo và defer complete/fail cho `ActivityLogger` ngay sau bước kiểm tra nil `bb.db == nil || bb.schemaAdapter == nil`:
```go
func (bb *BatchBuffer) batchUpsert(ctx context.Context, records []*shadow.UpsertRecord) (written int, err error) {
	if len(records) == 0 {
		return 0, nil
	}
	// Safety guard for unit tests where db or schemaAdapter are mocked as nil
	if bb.db == nil || bb.schemaAdapter == nil {
		return len(records), nil
	}
	first := records[0]
	tableName := first.TableName
	schemaName := bb.recordSchema(first)

	// --- Bắt đầu tích hợp Activity Log ---
	var logEntry *system.ActivityLog
	var act *governance.ActivityLogger
	if bb.db != nil {
		act = governance.NewActivityLogger(bb.db, bb.logger)
		targetFQN := schemaName + "." + tableName
		logEntry = act.Start("sink-upsert", targetFQN, "kafka-consumer")
	}

	defer func() {
		if act != nil && logEntry != nil {
			if err != nil {
				act.Fail(logEntry, err.Error())
			} else {
				details := map[string]any{
					"batch_size": len(records),
					"written":    written,
				}
				act.Complete(logEntry, int64(written), details)
			}
		}
	}()
	// --------------------------------------

	// Root span: batch is async and aggregates multiple kafka messages — see ADR-02.
	ctx, span := observability.ChildSpan(ctx, "cdc.batchbuffer.upsert",
...
```
