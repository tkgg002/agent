# 09. Technical Solution — Khắc phục lỗi `SQLSTATE 21000` và lỗi không kích hoạt `transmute` trong `bridge-oplog`

## 1. Bức tranh Tổng quan Vận hành Vật lý (Macro Runtime Topology)
- Khi người dùng kích hoạt **Bridge Oplog** trên CMS (`POST /api/v1/system/connectors/bridge-oplog`), `cdc-cms-service` phát bản tin NATS `cdc.cmd.bridge-oplog` sang `centralized-data-service` (`BridgeHandler.HandleBridgeOplog`).
- `BridgeHandler.processCollection` (`centralized-data-service/internal/handler/source/bridge_handler.go:367-475`) mở `MongoBridgeStrategy.ReadChanges` đọc luồng sự kiện thay đổi từ MongoDB Oplog (Change Stream) trong khoảng `[start_time_sec, end_time_sec]`, gom thành từng lô `batch` (`BatchSize = 500`) và gọi `h.batchUpsert(ctx, coll, batch)` để ghi multi-row `INSERT INTO <schema>.<table> (...) VALUES ... ON CONFLICT (...) DO UPDATE SET ...` xuống PostgreSQL Shadow DB.
- Sau khi ghi xong vào Shadow DB, `BridgeHandler.triggerTransmute` (`bridge_handler.go:472, 574-597`) bắn bản tin NATS `cdc.cmd.transmute-shadow` để `TransmuteHandler.handleTransmuteShadowWithOutcome` (`transmute_handler.go:136-231`) đưa vào `TableDebouncer` (`runDebouncedGroup` tại `transmute_handler.go:518-680`) và gọi `TransmuterModule` đồng bộ dữ liệu sang bảng Master.

## 2. Phân tích Nguyên nhân Gốc rễ (`F1` – `F5`)

### Phát hiện `F1` (Primary Root Cause gây `SQLSTATE 21000` — `VERIFIED`): Thiếu khử trùng lặp theo khóa xung đột (`DedupByConflictKey`) trong `BridgeHandler.batchUpsert`
- **Bằng chứng `file:line`:**
  - Tại `centralized-data-service/internal/handler/source/bridge_handler.go:428-435` và `:523-544`:
    `processCollection` đọc tuần tự mọi sự kiện từ MongoDB Change Stream (`op = "c" | "u" | "d"`) trong cửa sổ thời gian `[start_time_sec, end_time_sec]` và `append` trực tiếp vào slice `batch` (mặc định 500 sự kiện). Trong thực tế vận hành, một document MongoDB (cùng `_id` / `SourceID`) thường phát sinh nhiều sự kiện liên tiếp trong cùng cửa sổ oplog (ví dụ: vừa `insert` `"c"` xong liền `update` `"u"` trạng thái, hoặc nhiều lần `update` `"u"` trong cùng 1 mẻ 500 sự kiện).
  - Trong khi `BatchBuffer.batchUpsert` (`centralized-data-service/internal/handler/shadow/batch_buffer.go:634` gọi `dedupByConflictKey(sc.Records)` tại `batch_buffer_utils.go:90-153`) và `TransmuterModule` (`centralized-data-service/internal/service/master/transmuter.go:1238`) đều đã được vá bước khử trùng lặp trước khi chạy `ON CONFLICT DO UPDATE`, thì hàm `BridgeHandler.batchUpsert` (`bridge_handler.go:532-534`) lại bị bỏ sót — truyền thẳng slice `records` chưa khử trùng lặp vào `h.schemaAdapter.BuildBatchUpsertSQLsInSchema(schema, coll.SchemaName, coll.TableName, coll.pgPKField, records)`.
  - Khi câu lệnh `INSERT INTO ... VALUES (...), (...) ON CONFLICT (...) DO UPDATE` chứa $\ge 2$ dòng có cùng `PrimaryKeyValue` (`_source_id` hoặc `pgPKField`), PostgreSQL lập tức từ chối toàn bộ câu lệnh với lỗi:
    `ERROR: ON CONFLICT DO UPDATE command cannot affect row a second time (SQLSTATE 21000)`.

### Phát hiện `F2` (Secondary Root Cause — `VERIFIED`): Gộp chung sự kiện xóa (`IsDelete = true`) vào `BuildBatchUpsertSQLsInSchema` và thiếu thực thi Soft-Delete chuẩn trên Partial Unique Index `(_source_id) WHERE NOT _deleted`
- **Bằng chứng `file:line`:**
  - Tại `centralized-data-service/internal/handler/source/bridge_handler.go:503-519`: khi gặp sự kiện xóa (`event.Op == "d"`), `eventToUpsertRecord` gán `mappedColumns["_deleted"] = true` và `IsDelete: true`, nhưng `batchUpsert` (`bridge_handler.go:532-560`) vẫn nhét chung bản ghi xóa vào `BuildBatchUpsertSQLsInSchema` (và `BuildUpsertSQLInSchema` ở nhánh fallback).
  - Trong khi đó, bảng Shadow V2 dùng Partial Unique Index `(_source_id) WHERE NOT _deleted` (`schema_adapter.go:336-343`, `:981-986`). Nếu dùng `INSERT ... (_deleted = true) ON CONFLICT (_source_id) WHERE NOT _deleted DO UPDATE` cho bản ghi xóa, hoặc trong cùng batch có `insert/update` rồi `delete` trên cùng `SourceID`, nó vừa gây xung đột `SQLSTATE 21000` vừa không khớp chuẩn Soft-Delete (`BuildSoftDeleteUpdateSQL` + `BuildSoftDeleteInsertSQL` như `BatchBuffer.executeSoftDelete` tại `batch_buffer.go:615-673, 786-809`).

### Phát hiện `F3` (Edge-Case Guard — `VERIFIED`): Chưa đồng bộ lại `effectivePK = "_source_id"` trong `batchUpsert` sau khi `PrepareForCDCInsertInSchema` khởi tạo bảng mới
- **Bằng chứng `file:line`:**
  - Tại `centralized-data-service/internal/handler/source/bridge_handler.go:201, 334-355`: `resolveCollection` kiểm tra `schema.Columns["_source_id"]` **trước khi** `processCollection` gọi `h.schemaAdapter.PrepareForCDCInsertInSchema` (`bridge_handler.go:405`). Nếu bảng Shadow chưa tồn tại hoặc vừa được chuẩn bị lần đầu, tại thời điểm `resolveCollection` chạy thì `schema` là `nil` nên `coll.pgPKField` chưa được nâng lên `"_source_id"`. Khi vào `batchUpsert` (`bridge_handler.go:529-550`), cần kiểm tra lại `effectivePK` từ `schema.Columns["_source_id"]` (giống hệt `BatchBuffer.batchUpsert` tại `batch_buffer.go:589-592`).

### Phát hiện `F4` (Primary Root Cause khiến `bridge-oplog` KHÔNG chạy `transmute` — `VERIFIED`): `BridgeHandler.triggerTransmute` gửi bản tin rỗng thiếu `_source_ids`, `events` và `shadow_binding_code`, dẫn tới `TransmuterModule.Run` bỏ qua toàn bộ các dòng UPDATE (do `_gpay_id <= last_gpay_id`)
- **Bằng chứng `file:line`:**
  - Tại `centralized-data-service/internal/handler/source/bridge_handler.go:472` và `:574-597`:
    Khác hoàn toàn với `BatchBuffer.publishTransmuteTrigger` (`centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:32-98` gửi đầy đủ `"shadow_binding_code"`, `"_source_ids"`, `"events"` kèm `SourceID`, `Op`, `SourceTs`, `EventDate`, `RawData`, `IsDelete`), hàm `BridgeHandler.triggerTransmute` chỉ bắn 1 bản tin ở cuối `processCollection` mà **không truyền `"_source_ids"`, không truyền `"events"`, và không truyền `"shadow_binding_code"`**:
    ```go
    // bridge_handler.go:575-582
    payload, _ := json.Marshal(map[string]interface{}{
        "shadow_table":          coll.TableName,
        "shadow_schema":         coll.SchemaName,
        "shadow_connection_key": coll.shadowConnectionKey,
        "triggered_by":          "bridge-oplog",
        "correlation_id":        fmt.Sprintf("bridge-%s-%d", coll.TableName, time.Now().UnixNano()),
        "trace_id":              traceID,
    })
    ```
  - Khi bản tin này đi vào `TransmuteHandler.handleTransmuteShadowWithOutcome` (`centralized-data-service/internal/handler/master/transmute_handler.go:194-230`) và `TableDebouncer.runDebouncedGroup` (`transmute_handler.go:536-600`):
    1. Vì `len(allEvents) == 0` và `len(uniqueIDs) == 0`, `runDebouncedGroup` không chạy được `h.execRunEvents(runCtx, masterTable, allEvents, masterBindingID)` (vốn xử lý trực tiếp từng event từ memory mà không phụ thuộc con trỏ `_gpay_id`), mà rơi sang `h.execRun(runCtx, masterTable, uniqueIDs=nil, runType="", "", masterBindingID)` (`transmute_handler.go:599`).
    2. Trong `TransmuterModule.Run` (`centralized-data-service/internal/service/master/transmuter.go:366-418, 1003-1023`), khi `len(onlySourceIDs) == 0` và `runType == ""`, nó lấy `last_gpay_id` từ checkpoint (`state.LastCursorJSON`) rồi chạy câu lệnh:
       `SELECT ... FROM shadow WHERE (_gpay_id)::bigint > last_gpay_id`.
    3. Thế nhưng khi `bridge-oplog` thực hiện `ON CONFLICT (_source_id) WHERE NOT _deleted DO UPDATE` trên các dòng đã tồn tại trong bảng Shadow (`schema_adapter.go:626`), cột `_gpay_id` của dòng Shadow **giữ nguyên giá trị cũ** (trigger `tg_sonyflake_fallback` chỉ cấp `_gpay_id` khi `BEFORE INSERT`, không cấp lại khi `UPDATE`). Vì `_gpay_id` của các bản ghi được cập nhật qua `bridge-oplog` đều $\le$ `last_gpay_id` hiện tại của Master, câu truy vấn `WHERE (_gpay_id)::bigint > last_gpay_id` trả về **0 dòng (`Scanned: 0, Written: 0`)** — làm toàn bộ dữ liệu update/delete vừa bridge không được đồng bộ sang Master!
    4. Hơn nữa, vì `resolvedCollection` (`bridge_handler.go:77-83, 305-319`) chưa lưu `bindingCode = route.ShadowBinding.BindingCode` để gửi `"shadow_binding_code"`, truy vấn `MasterBindingRepo.ListMasterTargetsByShadowIdentity` (`centralized-data-service/internal/repository/master/master_binding_repo.go:124-130`) không đi vào nhánh chính xác `(@binding_code <> '' AND sb.binding_code = @binding_code)` mà rơi vào nhánh fallback so khớp `scr.connection_code = @shadow_connection_key`.

### Phát hiện `F5` (Secondary Root Cause khiến mất dấu `transmute` của `bridge-oplog` — `VERIFIED`): `TransmuteHandler.handleTransmuteShadowWithOutcome` cưỡng bức ghi đè `triggered_by = "bridge-oplog"` thành `"kafka-consumer-hook"`
- **Bằng chứng `file:line`:**
  - Tại `centralized-data-service/internal/handler/master/transmute_handler.go:189-192`:
    ```go
    hookTrigger := "kafka-consumer-hook"
    if req.TriggeredBy == "snapshot" {
        hookTrigger = "snapshot-hook"
    }
    ```
    Mặc dù `BridgeHandler.triggerTransmute` (`bridge_handler.go:579`) gửi `"triggered_by": "bridge-oplog"`, `handleTransmuteShadowWithOutcome` vẫn ghi đè cứng `hookTrigger = "kafka-consumer-hook"`! Hệ quả là trong `TableDebouncer` (`groupDebouncedTasksAt` tại `transmute_event_date_split.go:126-130`) và trên bảng `cdc_activity_log` (UI Activity Log), mọi lần transmute từ `bridge-oplog` đều bị biến thành `"kafka-consumer-hook"` thay vì `"bridge-oplog"`, khiến người dùng kiểm tra log không thấy bất kỳ lệnh transmute nào được kích hoạt bởi `bridge-oplog`.

---

## 3. Phương án Kỹ thuật Duy nhất (Ánh xạ 1-1 với `F1` – `F5`)

### Giải pháp cho `F1`: Trích xuất hàm dùng chung `shadow.DedupByConflictKey` tại tầng `internal/model/shadow` và gọi trước khi sinh SQL Batch Upsert trong `BridgeHandler.batchUpsert`
1. Chuyển logic khử trùng lặp và gộp cột theo thứ tự `SourceTsMs` từ `internal/handler/shadow/batch_buffer_utils.go:90-153` sang hàm dùng chung `shadow.DedupByConflictKey(records []*UpsertRecord) []*UpsertRecord` tại file đơn trách nhiệm `centralized-data-service/internal/model/shadow/upsert_dedup.go` (giữ `dedupByConflictKey` trong `batch_buffer_utils.go` gọi uỷ quyền sang `shadow.DedupByConflictKey` để tương thích 100% với các test hiện có).
2. Trong `BridgeHandler.batchUpsert` (`centralized-data-service/internal/handler/source/bridge_handler.go:523`), gọi `shadow.DedupByConflictKey(records)` ngay đầu hàm để khử trùng lặp toàn bộ các sự kiện cùng `PrimaryKeyValue` trong batch trước khi chia `subChunks` và gọi `BuildBatchUpsertSQLsInSchema`.

### Giải pháp cho `F2`: Tách nhánh xử lý `IsDelete` trong `BridgeHandler.batchUpsert` bằng `BuildSoftDeleteUpdateSQL` + `BuildSoftDeleteInsertSQL`
1. Với các sự kiện `r.IsDelete == true` trong sub-chunk (đã qua `shadow.DedupByConflictKey`), thực thi `BuildSoftDeleteUpdateSQL` trước; nếu `RowsAffected == 0` thì gọi `BuildSoftDeleteInsertSQL` (đúng chuẩn `BatchBuffer.executeSoftDelete` tại `batch_buffer.go:786-809`).

### Giải pháp cho `F3`: Chuẩn hóa `effectivePK` sau khi lấy `schema` trong `BridgeHandler.batchUpsert`
1. Ngay đầu `BridgeHandler.batchUpsert` (`bridge_handler.go:529`), sau khi lấy `schema := h.schemaAdapter.GetSchemaInSchema(coll.SchemaName, coll.TableName)`, xác định `effectivePK := coll.pgPKField`; nếu `schema != nil` và `schema.Columns["_source_id"]` tồn tại thì gán `effectivePK = "_source_id"`. Dùng `effectivePK` nhất quán cho cả `BuildBatchUpsertSQLsInSchema`, `BuildUpsertSQLInSchema` (fallback) và `executeSoftDelete`.

### Giải pháp cho `F4`: Truyền đầy đủ `shadow_binding_code`, `_source_ids`, và `events` (`[]shadow.TransmuteEventItem`) trong `BridgeHandler.triggerTransmute` theo từng mẻ đã ghi xuống Shadow
1. Bổ sung trường `bindingCode string` vào `resolvedCollection` (`bridge_handler.go:77-83`) và gán `resolved.bindingCode = strings.TrimSpace(route.ShadowBinding.BindingCode)` trong `resolveCollection` (`bridge_handler.go:316-319`).
2. Cập nhật `BridgeHandler.batchUpsert` trả về `([]*shadow.UpsertRecord, int, error)` (trả về danh sách `deduped` đã ghi thành công). Trong `flushBatch()` của `processCollection` (`bridge_handler.go:419-426`), mỗi khi ghi thành công một mẻ (`n > 0` và `len(persisted) > 0`), gọi `h.triggerTransmute(ctx, coll, persisted, traceID)`.
3. Trong `BridgeHandler.triggerTransmute` (`bridge_handler.go:574-597`), xây dựng mảng `sourceIDs` (`[]string`) và `events` (`[]shadow.TransmuteEventItem` kèm `SourceID`, `Op`, `SourceTs`, `EventDate` qua `shadow.ResolveEventDateHCM(r.SourceTsMs, now)`, `RawData`, `IsDelete`) và đính kèm `"shadow_binding_code": coll.bindingCode`, `"_source_ids": sourceIDs`, `"events": events` vào payload `cdc.cmd.transmute-shadow` (đồng nhất 100% với `BatchBuffer.publishTransmuteTrigger` tại `batch_buffer_fanout.go:32-98`). Nhờ đó `TableDebouncer.runDebouncedGroup` gọi thẳng `h.execRunEvents(runCtx, masterTable, allEvents, masterBindingID)` và đồng bộ tức thì cả dòng INSERT, UPDATE lẫn DELETE sang Master bất kể `_gpay_id` cũ hay mới.

### Giải pháp cho `F5`: Giữ nguyên `triggered_by = "bridge-oplog"` trong `TransmuteHandler.handleTransmuteShadowWithOutcome`
1. Tại `centralized-data-service/internal/handler/master/transmute_handler.go:189-193`, cập nhật logic xác định `hookTrigger`: nếu `req.TriggeredBy == "snapshot"` thì gán `"snapshot-hook"`, nếu `req.TriggeredBy == "bridge-oplog"` thì giữ nguyên `"bridge-oplog"`, còn mặc định là `"kafka-consumer-hook"`. Nhờ đó `cdc_activity_log` và `TableDebouncer` ghi nhận chính xác các lượt transmute được kích hoạt bởi `bridge-oplog`.

---

## 4. Điểm Gọi Thực thi & Thời điểm Kích hoạt (Runtime Call-Site Wiring — L017)

| Hàm / Module mới hoặc cập nhật | Caller `file:line` | Lifecycle Stage / Event Trigger |
| :--- | :--- | :--- |
| `shadow.DedupByConflictKey` (`internal/model/shadow/upsert_dedup.go`) | 1. `BridgeHandler.batchUpsert` (`internal/handler/source/bridge_handler.go:523`)<br>2. `dedupByConflictKey` (`internal/handler/shadow/batch_buffer_utils.go:90`) | Kích hoạt mỗi khi `flushBatch()` trong `BridgeHandler.processCollection` (mỗi 500 events hoặc cuối stream) và mỗi nhịp flush của `BatchBuffer`. |
| `BridgeHandler.executeSoftDelete`, `BridgeHandler.batchUpsert` & `BridgeHandler.triggerTransmute` (`internal/handler/source/bridge_handler.go:419-475, 523-625`) | `BridgeHandler.processCollection` (`internal/handler/source/bridge_handler.go:423, 434, 454`) | Khi nhận lệnh NATS `cdc.cmd.bridge-oplog` từ CMS API `POST /api/v1/system/connectors/bridge-oplog`, mỗi mẻ ghi Shadow thành công sẽ bắn `cdc.cmd.transmute-shadow` kèm `_source_ids` + `events` + `shadow_binding_code`. |
| `TransmuteHandler.handleTransmuteShadowWithOutcome` (`internal/handler/master/transmute_handler.go:189-195`) | `TransmuteHandler.HandleTransmuteShadowTask` & `HandleTransmuteShadow` (`internal/handler/master/transmute_handler.go:130, 721`) | Khi nhận bản tin `cdc.cmd.transmute-shadow` từ `BridgeHandler.triggerTransmute`. |

---

## 5. Code Demo (Trích dẫn nguyên văn từ HEAD)

### 5.1. File mới: `centralized-data-service/internal/model/shadow/upsert_dedup.go`
```go
package shadow

// DedupByConflictKey removes duplicate UpsertRecords within a batch by their
// PrimaryKeyValue, keeping only the latest occurrence by SourceTsMs (tie-breaker: later index)
// while merging non-conflicting MappedData columns from earlier events.
//
// This prevents PostgreSQL SQLSTATE 21000 — "ON CONFLICT DO UPDATE command
// cannot affect row a second time".
func DedupByConflictKey(records []*UpsertRecord) []*UpsertRecord {
	if len(records) <= 1 {
		return records
	}
	lastIdx := make(map[string]int, len(records))
	for i, r := range records {
		if r == nil {
			continue
		}
		if prevIdx, exists := lastIdx[r.PrimaryKeyValue]; !exists || r.SourceTsMs >= records[prevIdx].SourceTsMs {
			lastIdx[r.PrimaryKeyValue] = i
		}
	}
	if len(lastIdx) == len(records) {
		return records
	}
	mergedData := make(map[string]map[string]interface{}, len(lastIdx))
	maxTs := make(map[string]int64, len(lastIdx))
	for _, r := range records {
		if r == nil {
			continue
		}
		pk := r.PrimaryKeyValue
		m, exists := mergedData[pk]
		if !exists {
			m = make(map[string]interface{}, len(r.MappedData))
			mergedData[pk] = m
			maxTs[pk] = r.SourceTsMs
			for k, v := range r.MappedData {
				m[k] = v
			}
			continue
		}
		if r.SourceTsMs >= maxTs[pk] {
			maxTs[pk] = r.SourceTsMs
			for k, v := range r.MappedData {
				m[k] = v
			}
		} else {
			for k, v := range r.MappedData {
				if _, hasKey := m[k]; !hasKey {
					m[k] = v
				}
			}
		}
	}
	out := make([]*UpsertRecord, 0, len(lastIdx))
	for i, r := range records {
		if r == nil {
			continue
		}
		if lastIdx[r.PrimaryKeyValue] == i {
			cloned := *r
			cloned.MappedData = mergedData[r.PrimaryKeyValue]
			out = append(out, &cloned)
		}
	}
	return out
}
```

### 5.2. File cập nhật: `centralized-data-service/internal/handler/shadow/batch_buffer_utils.go:90-153`
```go
// Trước (HEAD batch_buffer_utils.go:90-153):
func dedupByConflictKey(records []*shadow.UpsertRecord) []*shadow.UpsertRecord {
	if len(records) <= 1 {
		return records
	}
	// Map PrimaryKeyValue → index of the latest record by SourceTsMs (tie-breaker: later index)
	lastIdx := make(map[string]int, len(records))
...
	return out
}

// Sau:
func dedupByConflictKey(records []*shadow.UpsertRecord) []*shadow.UpsertRecord {
	return shadow.DedupByConflictKey(records)
}
```

### 5.3. File cập nhật: `centralized-data-service/internal/handler/source/bridge_handler.go` (`:77-83`, `:315-319`, `:419-475`, `:522-597`)
```go
// Trước (HEAD bridge_handler.go:77-83):
type resolvedCollection struct {
	BridgeCollection
	connString          string
	shadowConnectionKey string
	bindingID           int64
	pgPKField           string
}

// Sau:
type resolvedCollection struct {
	BridgeCollection
	connString          string
	shadowConnectionKey string
	bindingID           int64
	bindingCode         string
	pgPKField           string
}
```

```go
// Trước (HEAD bridge_handler.go:315-319):
	resolved.shadowConnectionKey = route.ShadowConnectionKey
	if route.ShadowBinding != nil {
		resolved.bindingID = route.ShadowBinding.ID
	}

// Sau:
	resolved.shadowConnectionKey = route.ShadowConnectionKey
	if route.ShadowBinding != nil {
		resolved.bindingID = route.ShadowBinding.ID
		resolved.bindingCode = strings.TrimSpace(route.ShadowBinding.BindingCode)
	}
```

```go
// Trước (HEAD bridge_handler.go:419-475):
	flushBatch := func() (int, error) {
		if len(batch) == 0 {
			return 0, nil
		}
		n, flushErr := h.batchUpsert(ctx, coll, batch)
		batch = batch[:0]
		return n, flushErr
	}

	for event := range eventCh {
		eventCount++
		record := h.eventToUpsertRecord(ctx, event, coll)
		batch = append(batch, record)

		if len(batch) >= payload.BatchSize {
			n, flushErr := flushBatch()
			writtenCount += n
			if flushErr != nil {
				observability.Ctx(ctx, h.logger).Error("bridge-oplog: batch upsert failed",
					zap.String("table", coll.TableName),
					zap.String("trace_id", traceID),
					zap.Error(flushErr))
			}
		}

		// Progress log every 1000 events
		if eventCount%1000 == 0 {
			observability.Ctx(ctx, h.logger).Info("bridge-oplog: progress",
				zap.String("table", coll.TableName),
				zap.Int("events", eventCount),
				zap.Int("written", writtenCount))
		}
	}

	// Flush remaining
	n, flushErr := flushBatch()
	writtenCount += n
	if flushErr != nil {
		err = flushErr
	}

	// Wait for strategy to finish
	if stratErr := <-errCh; stratErr != nil && err == nil {
		err = stratErr
	}

	observability.Ctx(ctx, h.logger).Info("bridge-oplog: collection done",
		zap.String("source_db", coll.SourceDB),
		zap.String("source_table", coll.SourceTable),
		zap.Int("events", eventCount),
		zap.Int("written", writtenCount))

	// Trigger transmuter for this shadow table
	h.triggerTransmute(ctx, coll, traceID)

	return eventCount, writtenCount, err

// Sau:
	flushBatch := func() (int, error) {
		if len(batch) == 0 {
			return 0, nil
		}
		persisted, n, flushErr := h.batchUpsert(ctx, coll, batch)
		batch = batch[:0]
		if n > 0 && len(persisted) > 0 {
			h.triggerTransmute(ctx, coll, persisted, traceID)
		}
		return n, flushErr
	}

	for event := range eventCh {
		eventCount++
		record := h.eventToUpsertRecord(ctx, event, coll)
		batch = append(batch, record)

		if len(batch) >= payload.BatchSize {
			n, flushErr := flushBatch()
			writtenCount += n
			if flushErr != nil {
				observability.Ctx(ctx, h.logger).Error("bridge-oplog: batch upsert failed",
					zap.String("table", coll.TableName),
					zap.String("trace_id", traceID),
					zap.Error(flushErr))
			}
		}

		// Progress log every 1000 events
		if eventCount%1000 == 0 {
			observability.Ctx(ctx, h.logger).Info("bridge-oplog: progress",
				zap.String("table", coll.TableName),
				zap.Int("events", eventCount),
				zap.Int("written", writtenCount))
		}
	}

	// Flush remaining
	n, flushErr := flushBatch()
	writtenCount += n
	if flushErr != nil {
		err = flushErr
	}

	// Wait for strategy to finish
	if stratErr := <-errCh; stratErr != nil && err == nil {
		err = stratErr
	}

	observability.Ctx(ctx, h.logger).Info("bridge-oplog: collection done",
		zap.String("source_db", coll.SourceDB),
		zap.String("source_table", coll.SourceTable),
		zap.Int("events", eventCount),
		zap.Int("written", writtenCount))

	return eventCount, writtenCount, err
```

```go
// Trước (HEAD bridge_handler.go:522-597):
// batchUpsert writes a batch of UpsertRecords to the shadow table using SchemaAdapter batch SQL.
func (h *BridgeHandler) batchUpsert(ctx context.Context, coll resolvedCollection, records []*shadow.UpsertRecord) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	// Get cached schema for batch SQL generation
	schema := h.schemaAdapter.GetSchemaInSchema(coll.SchemaName, coll.TableName)

	// Use batch upsert SQL (multi-row INSERT ... ON CONFLICT) for performance
	queries, argSlices := h.schemaAdapter.BuildBatchUpsertSQLsInSchema(
		schema, coll.SchemaName, coll.TableName, coll.pgPKField, records,
	)

	written := 0
	var lastErr error
	for i, query := range queries {
		result := h.db.WithContext(ctx).Exec(query, argSlices[i]...)
		if result.Error != nil {
			observability.Ctx(ctx, h.logger).Warn("bridge-oplog: batch chunk failed, falling back to single-row",
				zap.Int("chunk", i),
				zap.Error(result.Error))
			lastErr = result.Error

			// Fallback: single-row upsert outside of failed transaction block
			for _, r := range records {
				q, a := h.schemaAdapter.BuildUpsertSQLInSchema(
					schema, r.SchemaName, r.TableName, r.PrimaryKeyField,
					r.PrimaryKeyValue, r.MappedData, r.RawData, r.Source, r.Hash, r.SourceTsMs,
				)
				if res := h.db.WithContext(ctx).Exec(q, a...); res.Error != nil {
					observability.Ctx(ctx, h.logger).Warn("bridge-oplog: single upsert failed, skipping",
						zap.String("source_id", r.PrimaryKeyValue),
						zap.Error(res.Error))
					lastErr = res.Error
					continue
				}
				written++
			}
			continue
		}
		written += int(result.RowsAffected)
	}

	if written > 0 {
		return written, nil
	}
	return written, lastErr
}

// triggerTransmute publishes NATS command to run transmuter for a shadow table.
// Payload matches HandleTransmuteShadow expected format.
func (h *BridgeHandler) triggerTransmute(ctx context.Context, coll resolvedCollection, traceID string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"shadow_table":          coll.TableName,
		"shadow_schema":         coll.SchemaName,
		"shadow_connection_key": coll.shadowConnectionKey,
		"triggered_by":          "bridge-oplog",
		"correlation_id":        fmt.Sprintf("bridge-%s-%d", coll.TableName, time.Now().UnixNano()),
		"trace_id":              traceID,
	})
	natsMsg := &nats.Msg{
		Subject: "cdc.cmd.transmute-shadow",
		Data:    payload,
		Header:  make(nats.Header),
	}
	if traceID != "" {
		natsMsg.Header.Set("Cdc-Correlation-Id", traceID)
	}
	if err := h.natsConn.PublishMsg(natsMsg); err != nil {
		observability.Ctx(ctx, h.logger).Warn("bridge-oplog: transmute trigger failed",
			zap.String("table", coll.TableName),
			zap.String("trace_id", traceID),
			zap.Error(err))
	}
}

// Sau:
// batchUpsert writes a batch of UpsertRecords to the shadow table using SchemaAdapter batch SQL.
// Deduplicates records by PrimaryKeyValue before building multi-row INSERT ... ON CONFLICT
// to prevent PostgreSQL SQLSTATE 21000 ("ON CONFLICT DO UPDATE command cannot affect row a second time").
func (h *BridgeHandler) batchUpsert(ctx context.Context, coll resolvedCollection, records []*shadow.UpsertRecord) ([]*shadow.UpsertRecord, int, error) {
	if len(records) == 0 {
		return nil, 0, nil
	}

	// Get cached schema for batch SQL generation
	schema := h.schemaAdapter.GetSchemaInSchema(coll.SchemaName, coll.TableName)
	effectivePK := coll.pgPKField
	if schema != nil {
		if _, hasSourceID := schema.Columns["_source_id"]; hasSourceID {
			effectivePK = "_source_id"
		}
	}

	// Deduplicate across the entire batch first so multiple events for the same
	// document (including insert/update/delete transitions) collapse to the latest state
	// by SourceTsMs while preserving non-conflicting columns from earlier events.
	deduped := shadow.DedupByConflictKey(records)

	type subChunkGroup struct {
		IsDelete bool
		Records  []*shadow.UpsertRecord
	}
	var subChunks []subChunkGroup
	for _, r := range deduped {
		if r == nil {
			continue
		}
		n := len(subChunks)
		if n == 0 || subChunks[n-1].IsDelete != r.IsDelete {
			subChunks = append(subChunks, subChunkGroup{
				IsDelete: r.IsDelete,
				Records:  []*shadow.UpsertRecord{r},
			})
		} else {
			subChunks[n-1].Records = append(subChunks[n-1].Records, r)
		}
	}

	written := 0
	var lastErr error
	for _, sc := range subChunks {
		if sc.IsDelete {
			for _, r := range sc.Records {
				rows, err := h.executeSoftDelete(ctx, schema, coll.SchemaName, coll.TableName, effectivePK, r)
				if err != nil {
					observability.Ctx(ctx, h.logger).Warn("bridge-oplog: soft delete failed, skipping",
						zap.String("source_id", r.PrimaryKeyValue),
						zap.Error(err))
					lastErr = err
					continue
				}
				written += int(rows)
			}
			continue
		}

		// Use batch upsert SQL (multi-row INSERT ... ON CONFLICT) for performance
		queries, argSlices := h.schemaAdapter.BuildBatchUpsertSQLsInSchema(
			schema, coll.SchemaName, coll.TableName, effectivePK, sc.Records,
		)

		for i, query := range queries {
			result := h.db.WithContext(ctx).Exec(query, argSlices[i]...)
			if result.Error != nil {
				observability.Ctx(ctx, h.logger).Warn("bridge-oplog: batch chunk failed, falling back to single-row",
					zap.Int("chunk", i),
					zap.Error(result.Error))
				lastErr = result.Error

				// Fallback: single-row upsert outside of failed transaction block
				for _, r := range sc.Records {
					q, a := h.schemaAdapter.BuildUpsertSQLInSchema(
						schema, r.SchemaName, r.TableName, effectivePK,
						r.PrimaryKeyValue, r.MappedData, r.RawData, r.Source, r.Hash, r.SourceTsMs,
					)
					if res := h.db.WithContext(ctx).Exec(q, a...); res.Error != nil {
						observability.Ctx(ctx, h.logger).Warn("bridge-oplog: single upsert failed, skipping",
							zap.String("source_id", r.PrimaryKeyValue),
							zap.Error(res.Error))
						lastErr = res.Error
						continue
					}
					written++
				}
				continue
			}
			written += int(result.RowsAffected)
		}
	}

	if written > 0 {
		return deduped, written, nil
	}
	return nil, written, lastErr
}

func (h *BridgeHandler) executeSoftDelete(
	ctx context.Context,
	schema *serviceshadow.TableSchema,
	schemaName, tableName, effectivePK string,
	r *shadow.UpsertRecord,
) (int64, error) {
	queryUpdate, argsUpdate := h.schemaAdapter.BuildSoftDeleteUpdateSQL(schema, schemaName, tableName, effectivePK, r)
	resUpdate := h.db.WithContext(ctx).Exec(queryUpdate, argsUpdate...)
	if resUpdate.Error != nil {
		return 0, resUpdate.Error
	}
	if resUpdate.RowsAffected > 0 {
		return resUpdate.RowsAffected, nil
	}
	queryInsert, argsInsert := h.schemaAdapter.BuildSoftDeleteInsertSQL(schema, schemaName, tableName, effectivePK, r)
	resInsert := h.db.WithContext(ctx).Exec(queryInsert, argsInsert...)
	if resInsert.Error != nil {
		return 0, resInsert.Error
	}
	return resInsert.RowsAffected, nil
}

// triggerTransmute publishes NATS command to run transmuter for a persisted shadow batch.
// Includes _source_ids, events, and shadow_binding_code so TransmuterModule.RunEvents
// materialises both inserted and updated/deleted records immediately.
func (h *BridgeHandler) triggerTransmute(ctx context.Context, coll resolvedCollection, records []*shadow.UpsertRecord, traceID string) {
	if h.natsConn == nil || coll.TableName == "" || len(records) == 0 {
		return
	}
	sourceIDs := make([]string, 0, len(records))
	events := make([]shadow.TransmuteEventItem, 0, len(records))
	now := time.Now()
	for _, r := range records {
		if r == nil || r.PrimaryKeyValue == "" {
			continue
		}
		evDate := shadow.ResolveEventDateHCM(r.SourceTsMs, now)
		sourceIDs = append(sourceIDs, r.PrimaryKeyValue)
		events = append(events, shadow.TransmuteEventItem{
			SourceID:  r.PrimaryKeyValue,
			Op:        r.Op,
			SourceTs:  r.SourceTsMs,
			EventDate: evDate,
			RawData:   r.RawData,
			IsDelete:  r.IsDelete,
		})
	}
	if len(sourceIDs) == 0 {
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"shadow_binding_code":   coll.bindingCode,
		"shadow_table":          coll.TableName,
		"shadow_schema":         coll.SchemaName,
		"shadow_connection_key": coll.shadowConnectionKey,
		"_source_ids":           sourceIDs,
		"events":                events,
		"triggered_by":          "bridge-oplog",
		"correlation_id":        fmt.Sprintf("bridge-%s-%d", coll.TableName, time.Now().UnixNano()),
		"trace_id":              traceID,
	})
	natsMsg := &nats.Msg{
		Subject: "cdc.cmd.transmute-shadow",
		Data:    payload,
		Header:  make(nats.Header),
	}
	observability.InjectNATSHeader(ctx, natsMsg.Header)
	if traceID != "" {
		natsMsg.Header.Set("Cdc-Correlation-Id", traceID)
	}
	natsMsg.Header.Set("Nats-Msg-Id", fmt.Sprintf("ts-%s-bridge-%d-%d", coll.TableName, time.Now().UnixNano(), len(sourceIDs)))
	if err := h.natsConn.PublishMsg(natsMsg); err != nil {
		observability.Ctx(ctx, h.logger).Warn("bridge-oplog: transmute trigger failed",
			zap.String("table", coll.TableName),
			zap.String("trace_id", traceID),
			zap.Error(err))
	}
}
```

### 5.4. File cập nhật: `centralized-data-service/internal/handler/master/transmute_handler.go:189-193`
```go
// Trước (HEAD transmute_handler.go:189-193):
	hookTrigger := "kafka-consumer-hook"
	if req.TriggeredBy == "snapshot" {
		hookTrigger = "snapshot-hook"
	}

// Sau:
	hookTrigger := "kafka-consumer-hook"
	switch req.TriggeredBy {
	case "snapshot":
		hookTrigger = "snapshot-hook"
	case "bridge-oplog":
		hookTrigger = "bridge-oplog"
	}
```

---

## 6. Bổ sung Vòng Lặp Phản Tỉnh & Khắc Phục Tự Động (`S5 -> S3 -> S4 -> S5` Self-Healing Loop)

Qua quá trình phản biện dòng-theo-dòng ở chặng `S5`, phát hiện 3 điểm kỹ thuật & kiến trúc cần chuẩn hóa triệt để trước khi đóng phiên:
1. **`AF-1` (Tách Module Đơn Trách Nhiệm & Chống Phình God-File — `L003`/`L022`):**
   - Tại HEAD, `internal/handler/source/bridge_handler.go` đã dài **628 dòng** (`> 500` dòng) và `internal/handler/master/transmute_debounce_group_test.go` dài **1,059 dòng**. Việc nhồi thêm `batchUpsert` mới, `executeSoftDelete` và `triggerTransmute` trực tiếp vào `bridge_handler.go` làm file phình lên 733 dòng (`+105` dòng), vi phạm kỷ luật Single-Responsibility và quy tắc `NOW <= HEAD` với file `> 500` dòng.
   - **Khắc phục:** Tách toàn bộ trách nhiệm ghi lô Shadow (`batchUpsert`, `recordSignature`, `executeSoftDelete`) và phát sự kiện fanout NATS (`triggerTransmute`) ra file riêng `internal/handler/source/bridge_upsert_fanout.go` (đồng nhất 100% với Design Pattern của `internal/handler/shadow/batch_buffer.go` + `batch_buffer_fanout.go` + `batch_buffer_utils.go`). Nhờ đó `bridge_handler.go` giảm từ **628 dòng (HEAD)** xuống còn **602 dòng (`<= HEAD`)**. Đồng thời tách test tích hợp `bridge-oplog` sang `internal/handler/master/transmute_bridge_oplog_test.go` để trả `transmute_debounce_group_test.go` về đúng **1,059 dòng (`== HEAD`)**.
2. **`AF-2` (Bảo toàn Dữ liệu khi Schema Dị biệt & Khi Lỗi Cục bộ trong `batchUpsert`):**
   - **Gom nhóm theo `recordSignature` (Signature Grouping):** Tương tự `BatchBuffer.batchUpsert` (`batch_buffer.go:636-663`), khi MongoDB Change Stream phát các bản ghi có tập cột `MappedData` khác nhau (spec drift hoặc partial update), nếu gộp chung vào một lệnh `BuildBatchUpsertSQLsInSchema` thì các cột vắng mặt ở bản ghi B sẽ bị điền `NULL` vào `VALUES` và `ON CONFLICT DO UPDATE SET col = EXCLUDED.col` sẽ ghi đè `NULL` lên dữ liệu cũ trong PostgreSQL. Do đó `BridgeHandler.batchUpsert` phải gom nhóm theo `recordSignature(r)` trước khi gọi `BuildBatchUpsertSQLsInSchema`.
   - **Theo dõi chính xác `persisted []*shadow.UpsertRecord`:** Nếu một câu lệnh soft-delete hoặc single-row fallback bị lỗi và bị skip, bản ghi lỗi **tuyệt đối không được đưa vào slice `persisted`** trả về cho `triggerTransmute` (tránh việc Shadow DB ghi lỗi nhưng Master DB vẫn nhận `events` và upsert dữ liệu ma). Đồng thời khi một chunk `queries[i]` lỗi và chuyển sang fallback từng dòng, chỉ fallback đúng `chunkRecords` của câu lệnh `i` đó để không cộng trùng `written` hoặc chạy lại các chunk trước đã thành công.

---

## 7. Đề xuất Khắc phục 3 Điểm Edge-Case Phát hiện trong Kiểm toán Sâu (`OPEN-1`..`OPEN-3` — Chế độ Propose-Only Chờ `APPROVE`)

Qua kiểm toán phản biện từng dòng ở Lượt 5 (giữ nguyên chế độ **100% Read-Only đối với source code** theo `L001` & `L019`), phát hiện thêm **3 điểm edge-case (`OPEN-1`, `OPEN-2`, `OPEN-3`)** trên code hiện tại cần trình anh xem xét và phê duyệt (`APPROVE`) trước khi sửa:

### 7.1. Giải pháp cho `OPEN-1` (Khớp đúng `route` theo `coll.TableName` & `coll.SchemaName` khi 1 bảng nguồn có nhiều Shadow Binding / Logical Clone — `bridge_handler.go:293-304`)
- **Bằng chứng:** Trong DB thật (`EV-20261106T135930-e492`), một collection nguồn như `payment_bills` có 2 bảng Shadow đích (`shadow_traitestpbs.payment_bills` và `shadow_traitestpbs.payment_bills_1`). Hiện tại `resolveCollection` (`bridge_handler.go:293`) lấy cứng `route := routes[0]`, nên nếu `BridgeCollection` truyền `TableName: "payment_bills_1"` thì vẫn bị ghi đè thành `"payment_bills"`.
- **Code Demo nguyên văn từ file hiện tại (`bridge_handler.go:292-305`):**
```go
// Trước (bridge_handler.go:292-305 hiện tại):
	route := routes[0]
	tc := route.TableConfig

	// Resolve DSN from registry (secure — no credential in NATS payload)
	if tc.SourceURL != "" {
		resolved.connString = tc.SourceURL
	}

	// Override from binding config
	resolved.TableName = tc.TargetTable
	resolved.PKField = tc.PrimaryKeyField

// Sau (Đề xuất chờ APPROVE):
	route := routes[0]
	for _, candidate := range routes {
		if candidate == nil || candidate.TableConfig == nil {
			continue
		}
		if coll.TableName != "" && strings.EqualFold(candidate.TableConfig.TargetTable, coll.TableName) {
			if coll.SchemaName == "" || (candidate.ShadowBinding != nil && strings.EqualFold(strings.TrimSpace(candidate.ShadowBinding.ShadowSchema), coll.SchemaName)) {
				route = candidate
				break
			}
		}
	}
	tc := route.TableConfig

	// Resolve DSN from registry (secure — no credential in NATS payload)
	if tc.SourceURL != "" {
		resolved.connString = tc.SourceURL
	}

	// Override from binding config
	resolved.TableName = tc.TargetTable
	resolved.PKField = tc.PrimaryKeyField
```

### 7.2. Giải pháp cho `OPEN-2` (Bảo toàn `RawData` đầy đủ gần nhất khi bản ghi `INSERT/UPDATE` bị `DELETE` trong cùng một batch — `upsert_dedup.go:25-65`)
- **Bằng chứng:** Trong `MongoBridgeStrategy.ReadChanges`, sự kiện `DELETE` (`op = "d"`) chỉ trích được `{"_id": docID}` từ `documentKey`. Nếu trong cùng một batch có sự kiện `c`/`u` ở $t_1$ (mang `RawData` đầy đủ các cột nghiệp vụ) và sự kiện `d` ở $t_2 > t_1$, `DedupByConflictKey` hiện giữ `MappedData` đã gộp nhưng giữ `cloned.RawData` của sự kiện `d` (`{"_id":"..."}`), làm `TransmuterModule.RunEvents` thiếu trường JSON khi ánh xạ dòng xóa mềm sang bảng Master.
- **Code Demo nguyên văn từ file hiện tại (`upsert_dedup.go:25-66`):**
```go
// Trước (upsert_dedup.go:25-66 hiện tại):
	mergedData := make(map[string]map[string]interface{}, len(lastIdx))
	maxTs := make(map[string]int64, len(lastIdx))
	for _, r := range records {
		if r == nil {
			continue
		}
		pk := r.PrimaryKeyValue
		m, exists := mergedData[pk]
		if !exists {
			m = make(map[string]interface{}, len(r.MappedData))
			mergedData[pk] = m
			maxTs[pk] = r.SourceTsMs
			for k, v := range r.MappedData {
				m[k] = v
			}
			continue
		}
		if r.SourceTsMs >= maxTs[pk] {
			maxTs[pk] = r.SourceTsMs
			for k, v := range r.MappedData {
				m[k] = v
			}
		} else {
			for k, v := range r.MappedData {
				if _, hasKey := m[k]; !hasKey {
					m[k] = v
				}
			}
		}
	}
	out := make([]*UpsertRecord, 0, len(lastIdx))
	for i, r := range records {
		if r == nil {
			continue
		}
		if lastIdx[r.PrimaryKeyValue] == i {
			cloned := *r
			cloned.MappedData = mergedData[r.PrimaryKeyValue]
			out = append(out, &cloned)
		}
	}
	return out

// Sau (Đề xuất chờ APPROVE):
	mergedData := make(map[string]map[string]interface{}, len(lastIdx))
	maxTs := make(map[string]int64, len(lastIdx))
	bestNonDeleteRaw := make(map[string]string, len(lastIdx))
	bestNonDeleteTs := make(map[string]int64, len(lastIdx))
	for _, r := range records {
		if r == nil {
			continue
		}
		pk := r.PrimaryKeyValue
		if !r.IsDelete && r.RawData != "" && r.SourceTsMs >= bestNonDeleteTs[pk] {
			bestNonDeleteRaw[pk] = r.RawData
			bestNonDeleteTs[pk] = r.SourceTsMs
		}
		m, exists := mergedData[pk]
		if !exists {
			m = make(map[string]interface{}, len(r.MappedData))
			mergedData[pk] = m
			maxTs[pk] = r.SourceTsMs
			for k, v := range r.MappedData {
				m[k] = v
			}
			continue
		}
		if r.SourceTsMs >= maxTs[pk] {
			maxTs[pk] = r.SourceTsMs
			for k, v := range r.MappedData {
				m[k] = v
			}
		} else {
			for k, v := range r.MappedData {
				if _, hasKey := m[k]; !hasKey {
					m[k] = v
				}
			}
		}
	}
	out := make([]*UpsertRecord, 0, len(lastIdx))
	for i, r := range records {
		if r == nil {
			continue
		}
		if lastIdx[r.PrimaryKeyValue] == i {
			cloned := *r
			cloned.MappedData = mergedData[r.PrimaryKeyValue]
			if cloned.IsDelete && bestNonDeleteRaw[r.PrimaryKeyValue] != "" {
				cloned.RawData = bestNonDeleteRaw[r.PrimaryKeyValue]
			}
			out = append(out, &cloned)
		}
	}
	return out
```

### 7.3. Giải pháp cho `OPEN-3` (Đồng bộ cơ chế publish JetStream + Fast-Fallback sang `cdc.cmd.transmute-shadow.core-fallback` trong `BridgeHandler.triggerTransmute` — `bridge_upsert_fanout.go:219-235`)
- **Bằng chứng:** `BatchBuffer.publishTransmuteTrigger` (`batch_buffer_fanout.go:121-151`) ưu tiên publish qua `js.PublishMsg` (với fast-break khi gặp `ErrNoStreamResponse` / `ErrNoResponders` / `ErrStreamNotFound`) và fallback sang subject `cdc.cmd.transmute-shadow.core-fallback` (luôn có `QueueSubscribe` lắng nghe tại `server_setup.go:338`). Việc bổ sung `SetJetStream(js nats.JetStreamContext)` và nhánh fallback `cdc.cmd.transmute-shadow.core-fallback` cho `BridgeHandler.triggerTransmute` đảm bảo tính nhất quán 100% giữa luồng Kafka CDC và luồng `bridge-oplog`.
- **Code Demo nguyên văn từ file hiện tại (`bridge_upsert_fanout.go:219-235`):**
```go
// Trước (bridge_upsert_fanout.go:219-235 hiện tại):
	natsMsg := &nats.Msg{
		Subject: "cdc.cmd.transmute-shadow",
		Data:    payload,
		Header:  make(nats.Header),
	}
	observability.InjectNATSHeader(ctx, natsMsg.Header)
	if traceID != "" {
		natsMsg.Header.Set("Cdc-Correlation-Id", traceID)
	}
	natsMsg.Header.Set("Nats-Msg-Id", fmt.Sprintf("ts-%s-bridge-%d-%d", coll.TableName, time.Now().UnixNano(), len(sourceIDs)))
	if err := h.natsConn.PublishMsg(natsMsg); err != nil {
		observability.Ctx(ctx, h.logger).Warn("bridge-oplog: transmute trigger failed",
			zap.String("table", coll.TableName),
			zap.String("trace_id", traceID),
			zap.Error(err))
	}
}

// Sau (Đã được User APPROVE):
	natsMsg := &nats.Msg{
		Subject: "cdc.cmd.transmute-shadow",
		Data:    payload,
		Header:  make(nats.Header),
	}
	observability.InjectNATSHeader(ctx, natsMsg.Header)
	if traceID != "" {
		natsMsg.Header.Set("Cdc-Correlation-Id", traceID)
	}
	natsMsg.Header.Set("Nats-Msg-Id", fmt.Sprintf("ts-%s-bridge-%d-%d", coll.TableName, time.Now().UnixNano(), len(sourceIDs)))

	if h.js != nil {
		var pubErr error
		for attempt := 1; attempt <= 3; attempt++ {
			_, pubErr = h.js.PublishMsg(natsMsg, nats.RetryAttempts(0))
			if pubErr == nil {
				return
			}
			if errors.Is(pubErr, nats.ErrNoStreamResponse) ||
				errors.Is(pubErr, nats.ErrNoResponders) ||
				errors.Is(pubErr, nats.ErrStreamNotFound) {
				break
			}
			time.Sleep(time.Duration(attempt*50) * time.Millisecond)
		}
		observability.Ctx(ctx, h.logger).Warn("bridge-oplog: failed to publish transmute trigger to JetStream, falling back to core NATS",
			zap.String("table", coll.TableName),
			zap.String("trace_id", traceID),
			zap.Error(pubErr))
		fallbackMsg := &nats.Msg{
			Subject: "cdc.cmd.transmute-shadow.core-fallback",
			Data:    payload,
			Header:  natsMsg.Header,
		}
		if err := h.natsConn.PublishMsg(fallbackMsg); err != nil {
			observability.Ctx(ctx, h.logger).Warn("bridge-oplog: fallback transmute trigger failed",
				zap.String("table", coll.TableName),
				zap.String("trace_id", traceID),
				zap.Error(err))
		}
		return
	}

	if err := h.natsConn.PublishMsg(natsMsg); err != nil {
		observability.Ctx(ctx, h.logger).Warn("bridge-oplog: transmute trigger failed",
			zap.String("table", coll.TableName),
			zap.String("trace_id", traceID),
			zap.Error(err))
	}
}
```

---

## 8. Giải pháp Khắc phục 4 Điểm Lỗ Hổng Phát hiện ở Lượt 7 & Đã Hiệu chỉnh Theo Đánh giá Kỹ thuật Lượt 8 (`OPEN-4`..`OPEN-7` — Đã Được User `APPROVE` & Đang Thực Thi S4)

Qua kiểm toán phản biện ở Lượt 7 và tiếp thu 100% đánh giá kỹ thuật của anh ở Lượt 8 (giữ nguyên chế độ **100% Read-Only đối với source code** theo `LS-GLOBAL-001` & `L001`), em đã hiệu chỉnh triệt để 3 điểm hổng kỹ thuật trong đề xuất `OPEN-4`, `OPEN-6`, `OPEN-7` và trình anh phê duyệt (`APPROVE`):

### 8.1. Giải pháp cho `OPEN-4` ([HIGH] Nối dây `SetJetStream(natsClient.JS)` An toàn Nil-Pointer tại `internal/server/server_setup.go:298` & `internal/server/server_setup_buffers.go:58-76` — `L019` & `L001`)
- **Bằng chứng `file:line` & Hiệu chỉnh Lượt 8:**
  - Tại `internal/server/server_setup.go:298`, `BridgeHandler` đang được khởi tạo mà chưa gọi `SetJetStream(natsClient.JS)` (khác với `BatchBuffer` tại `internal/server/server_setup_buffers.go:33-36`).
  - Trong đề xuất sơ bộ ở Lượt 7, lệnh `NewBridgeHandler(..., natsClient.Conn, logger)` nằm trên dòng `if natsClient != nil`, dẫn tới nguy cơ panic nil-pointer dereference nếu `natsClient == nil`.
  - **Giải pháp chuẩn:** Tách biến `var nc *nats.Conn` và kiểm tra `if natsClient != nil` **trước** khi truy cập `natsClient.Conn` và `natsClient.JS` (tương đồng 100% với `newConfiguredBatchBuffer` tại `internal/server/server_setup_buffers.go:33-36`).
- **Code Demo nguyên văn từ file hiện tại (`internal/server/server_setup.go:296-300` & `internal/server/server_setup_buffers.go:53-58`):**
```go
// Trước (internal/server/server_setup.go:296-300 hiện tại — 574 dòng == HEAD):
	natsClient.Conn.QueueSubscribe("cdc.cmd.system-connector-recover", workerQueueGroup, syncHandler.HandleRecoverConnector)
	natsClient.Conn.QueueSubscribe("cdc.cmd.system-connector.recover", workerQueueGroup, syncHandler.HandleRecoverConnector)
	bridgeHandler := handlersource.NewBridgeHandler(shadowDB, db, schemaAdapter, registrySvc, dynamicMapper, natsClient.Conn, logger)
	natsClient.Conn.QueueSubscribe("cdc.cmd.bridge-oplog", workerQueueGroup, bridgeHandler.HandleBridgeOplog)
	natsClient.Conn.QueueSubscribe("cdc.cmd.alter-column", workerQueueGroup, schemaDDLHandler.HandleAlterColumn)

// Sau (Đề xuất chờ APPROVE — giữ server_setup.go đúng 574 dòng == HEAD):
	natsClient.Conn.QueueSubscribe("cdc.cmd.system-connector-recover", workerQueueGroup, syncHandler.HandleRecoverConnector)
	natsClient.Conn.QueueSubscribe("cdc.cmd.system-connector.recover", workerQueueGroup, syncHandler.HandleRecoverConnector)
	bridgeHandler := newConfiguredBridgeHandler(shadowDB, db, schemaAdapter, registrySvc, dynamicMapper, natsClient, logger)
	natsClient.Conn.QueueSubscribe("cdc.cmd.bridge-oplog", workerQueueGroup, bridgeHandler.HandleBridgeOplog)
	natsClient.Conn.QueueSubscribe("cdc.cmd.alter-column", workerQueueGroup, schemaDDLHandler.HandleAlterColumn)
```
```go
// Bổ sung vào internal/server/server_setup_buffers.go (hiện chỉ 65 dòng):
func newConfiguredBridgeHandler(
	shadowDB *gorm.DB,
	db *gorm.DB,
	schemaAdapter *shadow.SchemaAdapter,
	registrySvc *servicesource.MetadataRegistryService,
	dynamicMapper *shadow.DynamicMapper,
	natsClient *natsconn.NatsClient,
	logger *zap.Logger,
) *handlersource.BridgeHandler {
	var nc *nats.Conn
	if natsClient != nil {
		nc = natsClient.Conn
	}
	bh := handlersource.NewBridgeHandler(shadowDB, db, schemaAdapter, registrySvc, dynamicMapper, nc, logger)
	if natsClient != nil && natsClient.JS != nil {
		bh.SetJetStream(natsClient.JS)
	}
	return bh
}
```

### 8.2. Giải pháp cho `OPEN-6` ([MEDIUM] Triển khai `resolveCollections` Khử Trùng theo `SchemaName.TableName` + Đọc Oplog Stream 1 Lần Duy Nhất và Fanout ở Tầng `flushBatch` — `bridge_handler.go:206-221` & `bridge_upsert_fanout.go:23-95`)
- **Giải trình Kỹ thuật về Cơ chế Re-read Oplog Stream (Trả lời trực diện câu hỏi của anh):**
  1. **Tại sao giao diện truyền `coll.TableName == ""`:** Trên giao diện `cdc-cms-web/src/pages/SourceConnectors.tsx:1978-1983`, khi bấm nút *"Bắt đầu Bridge"*, frontend gửi mảng `collections` với `{ source_db: db, source_table: table, schema_name: '', table_name: '', pk_field: '' }`.
  2. **Hậu quả nếu gọi lặp `processCollection` $N$ lần cho cùng 1 `SourceDB.SourceTable`:** Trong `bridge_handler.go:406-410`, mỗi lần gọi `processCollection` sẽ khởi chạy một goroutine `strategy.ReadChanges(ctx, cfg, eventCh)` mới (`bridge_mongo.go:53-160`), mở lại kết nối MongoDB Change Stream (hoặc chạy lại 3 câu query `coll.Find`) **gấp $N$ lần** trên cùng một collection nguồn, tốn gấp $N$ lần I/O mạng/DB lên MongoDB, và làm bộ đếm `totalEvents` (`oplog_fetched` trong `cdc_activity_log`, dòng `211`) bị nhân sai gấp $N$ lần!
  3. **Thiết kế Tối ưu Nhất (1-Pass Oplog Stream Read + Multi-Target Fanout tại `flushBatch` + Deduplicate theo `SchemaName.TableName`):**
     - **Hàm `selectSourceRoutes(routes, coll)` (`bridge_upsert_fanout.go`):**
       - Khi `coll.TableName != ""`: chỉ trả về **đúng 1 route** khớp bảng đó (`[]*metadata.ResolvedSourceRoute{selectSourceRoute(routes, coll)}`).
       - Khi `coll.TableName == ""`: duyệt danh sách `routes` và **khử trùng theo khóa chuẩn hóa `strings.ToLower(schemaName + "." + targetTable)`** để trả về danh sách các active bindings phân biệt theo bảng Shadow đích (ví dụ: `shadow_traitestpbs.payment_bills` có `bindingID=46` và `shadow_traitestpbs.payment_bills_1` có `bindingID=47`), tuyệt đối tránh ghi trùng vào cùng 1 bảng Shadow 2 lần.
     - **Hàm `resolveCollections` & `processCollectionTargets` (`bridge_handler.go` & `bridge_upsert_fanout.go`):**
       - Giữ nguyên `resolveCollection` (`bridge_handler.go:269`) làm hàm tương thích ngược cho 1 route.
       - Khi `len(targets) == 1`: gọi thẳng `h.processCollection(ctx, strategy, payload, targets[0], traceID)`.
       - Khi `len(targets) > 1`: `processCollectionTargets` chỉ gọi `strategy.ReadChanges(ctx, cfg, eventCh)` **ĐÚNG 1 LẦN** cho `(targets[0].SourceDB, targets[0].SourceTable)`. Mỗi `event` đọc từ `eventCh` chỉ cộng `eventCount++` **1 lần**, sau đó fanout qua `h.eventToUpsertRecord(ctx, event, target)` vào từng buffer `batches[i]` của từng bảng đích (đồng nhất 100% với B3 fan-out trong `EventHandler.processEvent` tại `internal/handler/shadow/event_handler.go:369-446`), và tại `flushBatch()` thực thi `batchUpsert` + `triggerTransmute` cho từng bảng đích.
- **Code Demo nguyên văn từ file hiện tại (`internal/handler/source/bridge_handler.go:206-221, 269-305` & `internal/handler/source/bridge_upsert_fanout.go:23-36`):**
```go
// Trước (internal/handler/source/bridge_handler.go:206-221 hiện tại):
	for _, coll := range payload.Collections {
		// Resolve shadow binding config for this collection
		resolved := h.resolveCollection(ctx, coll, payload.ConnString, payload.ConnectorName)

		events, written, err := h.processCollection(ctx, strategy, payload, resolved, traceID)
		atomic.AddInt64(&totalEvents, int64(events))
		atomic.AddInt64(&totalWritten, int64(written))
		if err != nil {
			lastErr = err
			observability.Ctx(ctx, h.logger).Error("bridge-oplog: collection failed",
				zap.String("source_db", coll.SourceDB),
				zap.String("source_table", coll.SourceTable),
				zap.String("trace_id", traceID),
				zap.Error(err))
		}
	}

// Sau (Đề xuất chờ APPROVE — Đọc Oplog 1 lần duy nhất cho mỗi SourceDB.SourceTable và fanout sang các bảng đích đã deduplicate):
	for _, coll := range payload.Collections {
		targets := h.resolveCollections(ctx, coll, payload.ConnString, payload.ConnectorName)
		events, written, err := h.processCollectionTargets(ctx, strategy, payload, targets, traceID)
		atomic.AddInt64(&totalEvents, int64(events))
		atomic.AddInt64(&totalWritten, int64(written))
		if err != nil {
			lastErr = err
			observability.Ctx(ctx, h.logger).Error("bridge-oplog: collection failed",
				zap.String("source_db", coll.SourceDB),
				zap.String("source_table", coll.SourceTable),
				zap.String("trace_id", traceID),
				zap.Error(err))
		}
	}
```
```go
// Trước (internal/handler/source/bridge_upsert_fanout.go:23-36 hiện tại):
func selectSourceRoute(routes []*metadata.ResolvedSourceRoute, coll BridgeCollection) *metadata.ResolvedSourceRoute {
	route := routes[0]
	for _, candidate := range routes {
		if candidate == nil || candidate.TableConfig == nil {
			continue
		}
		if coll.TableName != "" && strings.EqualFold(candidate.TableConfig.TargetTable, coll.TableName) {
			if coll.SchemaName == "" || (candidate.ShadowBinding != nil && strings.EqualFold(strings.TrimSpace(candidate.ShadowBinding.ShadowSchema), coll.SchemaName)) {
				return candidate
			}
		}
	}
	return route
}

// Sau (Đề xuất chờ APPROVE — selectSourceRoutes + processCollectionTargets trong bridge_upsert_fanout.go):
func selectSourceRoute(routes []*metadata.ResolvedSourceRoute, coll BridgeCollection) *metadata.ResolvedSourceRoute {
	route := routes[0]
	for _, candidate := range routes {
		if candidate == nil || candidate.TableConfig == nil {
			continue
		}
		if coll.TableName != "" && strings.EqualFold(candidate.TableConfig.TargetTable, coll.TableName) {
			if coll.SchemaName == "" || (candidate.ShadowBinding != nil && strings.EqualFold(strings.TrimSpace(candidate.ShadowBinding.ShadowSchema), coll.SchemaName)) {
				return candidate
			}
		}
	}
	return route
}

func selectSourceRoutes(routes []*metadata.ResolvedSourceRoute, coll BridgeCollection) []*metadata.ResolvedSourceRoute {
	if len(routes) == 0 {
		return nil
	}
	if coll.TableName != "" {
		return []*metadata.ResolvedSourceRoute{selectSourceRoute(routes, coll)}
	}
	seen := make(map[string]struct{}, len(routes))
	out := make([]*metadata.ResolvedSourceRoute, 0, len(routes))
	for _, candidate := range routes {
		if candidate == nil || candidate.TableConfig == nil {
			continue
		}
		targetTable := strings.TrimSpace(candidate.TableConfig.TargetTable)
		if targetTable == "" {
			continue
		}
		schemaName := "public"
		if candidate.ShadowBinding != nil {
			if s := strings.TrimSpace(candidate.ShadowBinding.ShadowSchema); s != "" {
				schemaName = s
			}
		}
		key := strings.ToLower(schemaName + "." + targetTable)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, candidate)
	}
	if len(out) == 0 {
		return []*metadata.ResolvedSourceRoute{routes[0]}
	}
	return out
}

func (h *BridgeHandler) resolveCollections(ctx context.Context, coll BridgeCollection, payloadConnString string, connectorName ...string) []resolvedCollection {
	var conn string
	if len(connectorName) > 0 {
		conn = strings.TrimSpace(connectorName[0])
	}
	routes := h.registrySvc.ResolveSourceRoutes(coll.SourceDB, coll.SourceTable, conn)
	selected := selectSourceRoutes(routes, coll)
	if len(selected) <= 1 {
		return []resolvedCollection{h.resolveCollection(ctx, coll, payloadConnString, connectorName...)}
	}
	out := make([]resolvedCollection, 0, len(selected))
	for _, route := range selected {
		scoped := coll
		scoped.TableName = route.TableConfig.TargetTable
		if route.ShadowBinding != nil && strings.TrimSpace(route.ShadowBinding.ShadowSchema) != "" {
			scoped.SchemaName = strings.TrimSpace(route.ShadowBinding.ShadowSchema)
		}
		out = append(out, h.resolveCollection(ctx, scoped, payloadConnString, connectorName...))
	}
	return out
}

func (h *BridgeHandler) processCollectionTargets(
	ctx context.Context,
	strategy BridgeStrategy,
	payload BridgeOplogPayload,
	targets []resolvedCollection,
	traceID string,
) (int, int, error) {
	if len(targets) == 0 {
		return 0, 0, nil
	}
	if len(targets) == 1 {
		return h.processCollection(ctx, strategy, payload, targets[0], traceID)
	}
	primary := targets[0]
	for _, t := range targets {
		if prepErr := h.schemaAdapter.PrepareForCDCInsertInSchema(t.SchemaName, t.TableName, t.pgPKField); prepErr != nil {
			return 0, 0, prepErr
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	cfg := BridgeStrategyConfig{
		ConnString:   primary.connString,
		SourceDB:     primary.SourceDB,
		SourceTable:  primary.SourceTable,
		StartTimeSec: payload.StartTimeSec,
		EndTimeSec:   payload.EndTimeSec,
	}
	eventCh := make(chan BridgeEvent, 1000)
	errCh := make(chan error, 1)
	go func() {
		errCh <- strategy.ReadChanges(ctx, cfg, eventCh)
		close(eventCh)
	}()
	batches := make([][]*shadow.UpsertRecord, len(targets))
	for i := range batches {
		batches[i] = make([]*shadow.UpsertRecord, 0, payload.BatchSize)
	}
	var eventCount, writtenCount int
	var lastErr error
	flushAll := func() {
		for i, t := range targets {
			if len(batches[i]) == 0 {
				continue
			}
			persisted, n, flushErr := h.batchUpsert(ctx, t, batches[i])
			batches[i] = batches[i][:0]
			writtenCount += n
			if n > 0 && len(persisted) > 0 {
				h.triggerTransmute(ctx, t, persisted, traceID)
			}
			if flushErr != nil {
				lastErr = flushErr
			}
		}
	}
	for event := range eventCh {
		eventCount++
		for i, t := range targets {
			batches[i] = append(batches[i], h.eventToUpsertRecord(ctx, event, t))
		}
		if len(batches[0]) >= payload.BatchSize {
			flushAll()
		}
	}
	flushAll()
	if stratErr := <-errCh; stratErr != nil && lastErr == nil {
		lastErr = stratErr
	}
	return eventCount, writtenCount, lastErr
}
```

### 8.3. Giải pháp cho `OPEN-7` ([LOW] Thay thế Heuristic So sánh Độ dài Chuỗi `len(bestRaw) > len(item.RawData)` bằng Kiểm tra `isMissingBeforeImage(item.RawData)` — `upsert_dedup.go:70-87`)
- **Bằng chứng `file:line` & Hiệu chỉnh Lượt 8:**
  - Trong `bridge_mongo.go:222-227`, sự kiện `DELETE` (`op == "d"`) không có `fullDocument` nên `event.Data == nil`, khi qua `bridge_handler.go:489` (`json.Marshal(event.Data)`) tạo ra `RawData == "null"`, hoặc trong `event_handler.go:398` (khi `event.Data.Before == nil`) tạo ra JSON stub chỉ chứa khóa chính và `_deleted` (`{"_id": "...", "_deleted": true}`).
  - Ngược lại, nếu nguồn có cấu hình `fullDocumentBeforeChange` (MongoDB) hoặc `REPLICA IDENTITY FULL` (PostgreSQL), sự kiện `DELETE` đã mang sẵn before-image thực sự chứa các trường nghiệp vụ (ví dụ `status`, `amount`), trong đó một trường string có thể ngắn hơn bản ghi cũ. Do đó dùng `len(bestRaw) > len(item.RawData)` là sai ngữ nghĩa!
  - **Giải pháp chuẩn:** Dùng hàm `isMissingBeforeImage(item.RawData)` chỉ trả về `true` khi `item.RawData` rỗng (`""`, `"{}"`, `"null"`) hoặc là JSON stub chỉ chứa các khóa định danh/metadata (`_id`, `id`, `_source_id`, `_deleted`) mà không có bất kỳ trường nghiệp vụ nào. Chỉ khi `isMissingBeforeImage(item.RawData)` bằng `true`, mới áp dụng fallback `bestRaw` từ sự kiện non-delete có `SourceTsMs` lớn nhất trong cùng batch.
- **Code Demo nguyên văn từ file hiện tại (`internal/model/shadow/upsert_dedup.go:70-87`):**
```go
// Trước (internal/model/shadow/upsert_dedup.go:70-87 hiện tại):
func applyNonDeleteRawFallback(records []*UpsertRecord, out []*UpsertRecord) {
	for _, item := range out {
		if item == nil || !item.IsDelete {
			continue
		}
		bestTs := int64(-1)
		bestRaw := ""
		for _, r := range records {
			if r != nil && !r.IsDelete && r.PrimaryKeyValue == item.PrimaryKeyValue && r.RawData != "" && r.SourceTsMs >= bestTs {
				bestTs = r.SourceTsMs
				bestRaw = r.RawData
			}
		}
		if bestRaw != "" {
			item.RawData = bestRaw
		}
	}
}

// Sau (Đề xuất chờ APPROVE):
func applyNonDeleteRawFallback(records []*UpsertRecord, out []*UpsertRecord) {
	for _, item := range out {
		if item == nil || !item.IsDelete || !isMissingBeforeImage(item.RawData) {
			continue
		}
		bestTs := int64(-1)
		bestRaw := ""
		for _, r := range records {
			if r != nil && !r.IsDelete && r.PrimaryKeyValue == item.PrimaryKeyValue && r.RawData != "" && r.SourceTsMs >= bestTs {
				bestTs = r.SourceTsMs
				bestRaw = r.RawData
			}
		}
		if bestRaw != "" {
			item.RawData = bestRaw
		}
	}
}

func isMissingBeforeImage(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return true
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return false
	}
	for k := range obj {
		switch k {
		case "_id", "id", "_source_id", "_deleted":
		default:
			return false
		}
	}
	return true
}
```

### 8.4. Kế hoạch Đóng Cổng `/laya_router` Mode B cho `OPEN-5` sau khi `APPROVE`
- Ngay sau khi anh gõ lệnh `APPROVE` và em áp dụng 3 bản vá `OPEN-4`, `OPEN-6`, `OPEN-7` ở trên, em sẽ chạy lại full test suite qua `evidence_run.py`, xuất **100% full `git diff` thật** và chạy `laya_gate.py run --stage dg3` + `laya_gate.py review --stage dg3review` kèm `laya_gate.py resolve` hợp lệ để mở khóa hoàn toàn cổng `DG3` trước khi đóng task.

---

## 9. Giải pháp Gia cố Phòng vệ Phát hiện trong Kiểm toán Sâu Lượt 10 & Tinh chỉnh Kỹ thuật Lượt 11 (`OPEN-8`..`OPEN-10` — Đã Được User `APPROVE` & Đang Thực Thi S4)

Qua kiểm toán phản biện từng dòng ở Lượt 10 và tiếp thu 100% đánh giá kỹ thuật cùng 2 tinh chỉnh của anh ở Lượt 11, cả 3 hạng mục `OPEN-8` (nâng mức độ lên **`[MEDIUM]`**), `OPEN-9` (`[LOW]`), `OPEN-10` (`[LOW]`) đã được phê duyệt (`APPROVE`) để triển khai:

### 9.1. Giải pháp cho `OPEN-8` (`[MEDIUM]` — Bảo toàn `err = flushErr` ở các mẻ trung gian trong `processCollection` mà không bị mẻ cuối ghi đè, và chuẩn hóa `BatchSize <= 0` + log lỗi trong `processCollectionTargets` tại `bridge_handler.go:422-468` & `bridge_upsert_fanout.go:130-165`)
- **Bằng chứng `file:line` & Tinh chỉnh Lượt 11:**
  - Tại `internal/handler/source/bridge_handler.go:445-450` (hàm `processCollection`), khi một mẻ trung gian (`len(batch) >= batchSize`) trả về `flushErr != nil`, hàm chỉ ghi log `Error("bridge-oplog: batch upsert failed")` mà chưa gán `err = flushErr`. Khi bổ sung `err = flushErr` ở mẻ trung gian, tại đoạn xử lý mẻ cuối (`bridge_handler.go:463-467`) giữ nguyên kiểm tra `if flushErr != nil { err = flushErr }` để khi mẻ cuối thành công (`flushErr == nil`) tuyệt đối không ghi đè làm mất lỗi `err` của mẻ trung gian trước đó.
  - Tại `internal/handler/source/bridge_upsert_fanout.go:130-158` (hàm `processCollectionTargets`) và `bridge_handler.go:423, 442` (hàm `processCollection`), nếu caller truyền `payload.BatchSize == 0`, điều kiện `len(batches[0]) >= payload.BatchSize` luôn đúng ngay từ bản ghi đầu tiên (`1 >= 0`), làm mất hoàn toàn cơ chế gom mẻ; nếu `payload.BatchSize < 0` thì `make(..., 0, payload.BatchSize)` sẽ gây panic. Do đó chuẩn hóa `batchSize := payload.BatchSize; if batchSize <= 0 { batchSize = maxBridgeBatchChunkRows }`, đồng thời bổ sung `observability.Ctx(ctx, h.logger).Error("bridge-oplog: batch upsert failed", ...)` khi `flushErr != nil` trong `processCollectionTargets`.
- **Code Demo nguyên văn từ file hiện tại (`bridge_handler.go:422-468` & `bridge_upsert_fanout.go:130-160`):**
```go
// Trước (internal/handler/source/bridge_handler.go:422-468 hiện tại):
	// Consumer: batch events and upsert to shadow
	batch := make([]*shadow.UpsertRecord, 0, payload.BatchSize)

	flushBatch := func() (int, error) {
		if len(batch) == 0 {
			return 0, nil
		}
		persisted, n, flushErr := h.batchUpsert(ctx, coll, batch)
		batch = batch[:0]
		if n > 0 && len(persisted) > 0 {
			h.triggerTransmute(ctx, coll, persisted, traceID)
		}
		return n, flushErr
	}

	for event := range eventCh {
		eventCount++
		record := h.eventToUpsertRecord(ctx, event, coll)
		batch = append(batch, record)

		if len(batch) >= payload.BatchSize {
			n, flushErr := flushBatch()
			writtenCount += n
			if flushErr != nil {
				observability.Ctx(ctx, h.logger).Error("bridge-oplog: batch upsert failed",
					zap.String("table", coll.TableName),
					zap.String("trace_id", traceID),
					zap.Error(flushErr))
			}
		}

		// Progress log every 1000 events
		if eventCount%1000 == 0 {
			observability.Ctx(ctx, h.logger).Info("bridge-oplog: progress",
				zap.String("table", coll.TableName),
				zap.Int("events", eventCount),
				zap.Int("written", writtenCount))
		}
	}

	// Flush remaining
	n, flushErr := flushBatch()
	writtenCount += n
	if flushErr != nil {
		err = flushErr
	}

// Sau (Đã được User APPROVE):
	// Consumer: batch events and upsert to shadow
	batchSize := payload.BatchSize
	if batchSize <= 0 {
		batchSize = maxBridgeBatchChunkRows
	}
	batch := make([]*shadow.UpsertRecord, 0, batchSize)

	flushBatch := func() (int, error) {
		if len(batch) == 0 {
			return 0, nil
		}
		persisted, n, flushErr := h.batchUpsert(ctx, coll, batch)
		batch = batch[:0]
		if n > 0 && len(persisted) > 0 {
			h.triggerTransmute(ctx, coll, persisted, traceID)
		}
		return n, flushErr
	}

	for event := range eventCh {
		eventCount++
		record := h.eventToUpsertRecord(ctx, event, coll)
		batch = append(batch, record)

		if len(batch) >= batchSize {
			n, flushErr := flushBatch()
			writtenCount += n
			if flushErr != nil {
				err = flushErr
				observability.Ctx(ctx, h.logger).Error("bridge-oplog: batch upsert failed",
					zap.String("table", coll.TableName),
					zap.String("trace_id", traceID),
					zap.Error(flushErr))
			}
		}

		// Progress log every 1000 events
		if eventCount%1000 == 0 {
			observability.Ctx(ctx, h.logger).Info("bridge-oplog: progress",
				zap.String("table", coll.TableName),
				zap.Int("events", eventCount),
				zap.Int("written", writtenCount))
		}
	}

	// Flush remaining
	n, flushErr := flushBatch()
	writtenCount += n
	if flushErr != nil {
		err = flushErr
	}
```
```go
// Trước (internal/handler/source/bridge_upsert_fanout.go:130-160 hiện tại):
	batches := make([][]*shadow.UpsertRecord, len(targets))
	for i := range batches {
		batches[i] = make([]*shadow.UpsertRecord, 0, payload.BatchSize)
	}
	var eventCount, writtenCount int
	var lastErr error
	flushAll := func() {
		for i, t := range targets {
			if len(batches[i]) == 0 {
				continue
			}
			persisted, n, flushErr := h.batchUpsert(ctx, t, batches[i])
			batches[i] = batches[i][:0]
			writtenCount += n
			if n > 0 && len(persisted) > 0 {
				h.triggerTransmute(ctx, t, persisted, traceID)
			}
			if flushErr != nil {
				lastErr = flushErr
			}
		}
	}
	for event := range eventCh {
		eventCount++
		for i, t := range targets {
			batches[i] = append(batches[i], h.eventToUpsertRecord(ctx, event, t))
		}
		if len(batches[0]) >= payload.BatchSize {
			flushAll()
		}
	}

// Sau (Đã được User APPROVE):
	batchSize := payload.BatchSize
	if batchSize <= 0 {
		batchSize = maxBridgeBatchChunkRows
	}
	batches := make([][]*shadow.UpsertRecord, len(targets))
	for i := range batches {
		batches[i] = make([]*shadow.UpsertRecord, 0, batchSize)
	}
	var eventCount, writtenCount int
	var lastErr error
	flushAll := func() {
		for i, t := range targets {
			if len(batches[i]) == 0 {
				continue
			}
			persisted, n, flushErr := h.batchUpsert(ctx, t, batches[i])
			batches[i] = batches[i][:0]
			writtenCount += n
			if n > 0 && len(persisted) > 0 {
				h.triggerTransmute(ctx, t, persisted, traceID)
			}
			if flushErr != nil {
				lastErr = flushErr
				observability.Ctx(ctx, h.logger).Error("bridge-oplog: batch upsert failed",
					zap.String("table", t.TableName),
					zap.String("trace_id", traceID),
					zap.Error(flushErr))
			}
		}
	}
	for event := range eventCh {
		eventCount++
		for i, t := range targets {
			batches[i] = append(batches[i], h.eventToUpsertRecord(ctx, event, t))
		}
		if len(batches[0]) >= batchSize {
			flushAll()
		}
	}
```

### 9.2. Giải pháp cho `OPEN-9` (`[LOW]` — Hỗ trợ tên cột khóa chính tùy chỉnh `item.PrimaryKeyField` trong `isMissingBeforeImage` và đặt `r.SourceTsMs >= bestTs` trước `!isMissingBeforeImage(r.RawData, r.PrimaryKeyField)` tại `upsert_dedup.go:75-111`)
- **Bằng chứng `file:line` & Tinh chỉnh Lượt 11:**
  - Tại `internal/model/shadow/upsert_dedup.go:94-111`, truyền thêm `item.PrimaryKeyField` vào `isMissingBeforeImage(raw string, pkField ...string)` giúp nhận diện đúng bản ghi before-image stub trên các bảng dùng khóa chính tùy chỉnh như `bill_code`.
  - Tại vòng lặp dòng `83`, đặt `r.SourceTsMs >= bestTs` **trước** `!isMissingBeforeImage(r.RawData, r.PrimaryKeyField)` để chỉ gọi `json.Unmarshal` khi bản ghi ứng viên thực sự có mốc thời gian `SourceTsMs >= bestTs`.
- **Code Demo nguyên văn từ file hiện tại (`internal/model/shadow/upsert_dedup.go:75-111`):**
```go
// Trước (internal/model/shadow/upsert_dedup.go:75-111 hiện tại):
func applyNonDeleteRawFallback(records []*UpsertRecord, out []*UpsertRecord) {
	for _, item := range out {
		if item == nil || !item.IsDelete || !isMissingBeforeImage(item.RawData) {
			continue
		}
		bestTs := int64(-1)
		bestRaw := ""
		for _, r := range records {
			if r != nil && !r.IsDelete && r.PrimaryKeyValue == item.PrimaryKeyValue && r.RawData != "" && r.SourceTsMs >= bestTs {
				bestTs = r.SourceTsMs
				bestRaw = r.RawData
			}
		}
		if bestRaw != "" {
			item.RawData = bestRaw
		}
	}
}

func isMissingBeforeImage(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return true
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return false
	}
	for k := range obj {
		switch k {
		case "_id", "id", "_source_id", "_deleted":
		default:
			return false
		}
	}
	return true
}

// Sau (Đã được User APPROVE):
func applyNonDeleteRawFallback(records []*UpsertRecord, out []*UpsertRecord) {
	for _, item := range out {
		if item == nil || !item.IsDelete || !isMissingBeforeImage(item.RawData, item.PrimaryKeyField) {
			continue
		}
		bestTs := int64(-1)
		bestRaw := ""
		for _, r := range records {
			if r != nil && !r.IsDelete && r.PrimaryKeyValue == item.PrimaryKeyValue && r.SourceTsMs >= bestTs && !isMissingBeforeImage(r.RawData, r.PrimaryKeyField) {
				bestTs = r.SourceTsMs
				bestRaw = r.RawData
			}
		}
		if bestRaw != "" {
			item.RawData = bestRaw
		}
	}
}

func isMissingBeforeImage(raw string, pkField ...string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return true
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return false
	}
	customPK := ""
	if len(pkField) > 0 {
		customPK = strings.TrimSpace(pkField[0])
	}
	for k := range obj {
		if customPK != "" && strings.EqualFold(k, customPK) {
			continue
		}
		switch k {
		case "_id", "id", "_source_id", "_deleted":
		default:
			return false
		}
	}
	return true
}
```

### 9.3. Giải pháp cho `OPEN-10` (`[LOW]` — Tôn trọng bộ lọc `coll.SchemaName` khi `coll.TableName == ""` nhưng `coll.SchemaName != ""` trong `selectSourceRoutes` tại `bridge_upsert_fanout.go:38-72`)
- **Bằng chứng `file:line`:**
  - Tại `internal/handler/source/bridge_upsert_fanout.go:38-72`, khi `coll.TableName == ""` nhưng caller truyền `coll.SchemaName != ""`, lọc theo `coll.SchemaName` giúp tránh fanout nhầm sang các schema khác.
- **Code Demo nguyên văn từ file hiện tại (`internal/handler/source/bridge_upsert_fanout.go:55-68`):**
```go
// Trước (internal/handler/source/bridge_upsert_fanout.go:55-68 hiện tại):
		schemaName := "public"
		if candidate.ShadowBinding != nil {
			if s := strings.TrimSpace(candidate.ShadowBinding.ShadowSchema); s != "" {
				schemaName = s
			}
		}
		key := strings.ToLower(schemaName + "." + targetTable)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, candidate)

// Sau (Đã được User APPROVE):
		schemaName := "public"
		if candidate.ShadowBinding != nil {
			if s := strings.TrimSpace(candidate.ShadowBinding.ShadowSchema); s != "" {
				schemaName = s
			}
		}
		if coll.SchemaName != "" && !strings.EqualFold(schemaName, strings.TrimSpace(coll.SchemaName)) {
			continue
		}
		key := strings.ToLower(schemaName + "." + targetTable)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, candidate)
```





