# 09. TASKS SOLUTION (GIẢI PHÁP CỐT LÕI DUY NHẤT & CODE DEMO ĐẦY ĐỦ) — `daily_flow_audit`

## 1. Tóm tắt Nguyên lý Cốt lõi
1. **Neo mốc ngày sự kiện gốc (`event_date` theo `SourceTsMs` múi giờ `Asia/Ho_Chi_Minh`) xuyên suốt `Kafka -> Shadow -> Master`:**
   - Mọi bản ghi CDC đều mang `SourceTsMs` từ Debezium (`cdc_event.go:51`).
   - Tại `BatchBuffer` (`Shadow`) và `TransmuteHandler` (`Master`), tách mẻ ở giao thời `00:00:00` theo `event_date` của sự kiện và ghi `"event_date": "YYYY-MM-DD"` vào `cdc_activity_log.details`.
   - Tại `GetDailyFlow` (`cdc-cms-service`), gom nhóm theo `COALESCE(NULLIF(e.details->>'event_date', ''), TO_CHAR(e.started_at ...))`.
2. **Gộp (Dedup/Coalesce) khi Upsert DB Master trong `RunEvents` nhưng giữ nguyên bộ đếm `Inserted(c)` / `Updated(u)` 1:1 theo Event:**
   - Trong `RunEvents` (`transmuter.go:705`), gom các sự kiện upsert liên tiếp trong mẻ, dedup theo `conflictTarget` (giữ bản mới nhất theo `_source_ts`), gọi `bulkUpsertMaster` 1 lần cho cả sub-batch, ghi tổng số dòng DB vật lý thực tế (`ins + upd`) vào `res.Written` (`details["written"]`), trong khi vẫn đếm `res.Inserted` (`c`/`r`) và `res.Updated` (`u`) theo từng `ev.Op` của luồng sự kiện.

---

## 2. Code Demo Đầy Đủ Cho Từng File

### File 1: `centralized-data-service/internal/model/shadow/cdc_event.go` & `cdc_event_date.go` (Tách file đơn trách nhiệm)
```go
// Trong cdc_event.go (giữ thuần DTO struct):
type TransmuteEventItem struct {
	SourceID  string `json:"source_id"`
	Op        string `json:"op"` // "c" create/insert, "u" update, "d" delete, "r" snapshot read
	SourceTs  int64  `json:"source_ts"`
	EventDate string `json:"event_date,omitempty"`
	RawData   string `json:"raw_data,omitempty"`
	IsDelete  bool   `json:"is_delete"`
}

// Trong cdc_event_date.go (file mới chuyên trách chuẩn hóa ngày sự kiện GMT+7):
package shadow

import "time"

var locHCM = time.FixedZone("GMT+7", 7*3600)

func ResolveEventDateHCM(tsMs int64, fallback time.Time) string {
	if tsMs > 0 {
		return time.UnixMilli(tsMs).In(locHCM).Format("2006-01-02")
	}
	if fallback.IsZero() {
		fallback = time.Now()
	}
	return fallback.In(locHCM).Format("2006-01-02")
}

// ResolveBatchEventDateHCM duyệt tìm bản ghi dữ liệu thật đầu tiên (bỏ qua IsMarker hoặc SourceTsMs <= 0)
// để lấy event_date chính xác cho sub-batch.
func ResolveBatchEventDateHCM(records []*UpsertRecord, fallback time.Time) string {
	for _, r := range records {
		if r == nil || r.IsMarker || r.SourceTsMs <= 0 {
			continue
		}
		return ResolveEventDateHCM(r.SourceTsMs, fallback)
	}
	return ResolveEventDateHCM(0, fallback)
}
```

### File 2: `centralized-data-service/internal/handler/shadow/batch_buffer_event_date.go`, `batch_buffer.go` & `batch_buffer_fanout.go`
```go
// Trong batch_buffer_event_date.go (file mới tách riêng trách nhiệm chia sub-batch theo ngày sự kiện):
func splitRecordsByEventDate(records []*shadow.UpsertRecord, fallback time.Time) [][]*shadow.UpsertRecord {
	if len(records) <= 1 {
		return [][]*shadow.UpsertRecord{records}
	}
	defaultDate := shadow.ResolveBatchEventDateHCM(records, fallback)
	var out [][]*shadow.UpsertRecord
	idx := make(map[string]int, 2)
	currentDate := defaultDate
	for _, r := range records {
		d := currentDate
		if r != nil && !r.IsMarker && r.SourceTsMs > 0 {
			d = shadow.ResolveEventDateHCM(r.SourceTsMs, fallback)
			currentDate = d
		}
		i, ok := idx[d]
		if !ok {
			i = len(out)
			idx[d] = i
			out = append(out, make([]*shadow.UpsertRecord, 0, len(records)))
		}
		out[i] = append(out[i], r)
	}
	return out
}

// Trong batch_buffer.go (batchUpsert - lúc ghi details vào cdc_activity_log):
				eventDate := shadow.ResolveBatchEventDateHCM(records, logEntry.StartedAt)
				details := map[string]any{
					"batch_size":    len(records),
					"written":       written,
					"c":             cntC,
					"u":             cntU,
					"d":             cntD,
					"event_date":    eventDate,
					"buffer_type":   bufferType,
					"is_snapshot":   bb.isSnapshotBuffer,
					"trace_id":      traceID,
					"shadow_schema": schemaName,
				}
				act.Complete(logEntry, int64(written), details)

// Trong batch_buffer_fanout.go (publishTransmuteTrigger):
	now := time.Now()
	for _, r := range records {
		if r.IsMarker {
			continue
		}
		if r.PrimaryKeyValue != "" {
			sourceIDs = append(sourceIDs, r.PrimaryKeyValue)
			events = append(events, shadow.TransmuteEventItem{
				SourceID:  r.PrimaryKeyValue,
				Op:        r.Op,
				SourceTs:  r.SourceTsMs,
				EventDate: shadow.ResolveEventDateHCM(r.SourceTsMs, now),
				RawData:   r.RawData,
				IsDelete:  r.IsDelete,
			})
		}
	}
```

### File 3: `centralized-data-service/internal/handler/master/transmute_event_date_split.go`, `transmute_handler.go` & `transmute_event_date_split_test.go`
```go
// Trong transmute_event_date_split.go (file mới tách riêng trách nhiệm tách NatsMsgTask theo event_date và giữ chốt chặn ACK):
func taskEventDate(t NatsMsgTask, fallback time.Time) string {
	for _, ev := range t.Req.Events {
		if ev.EventDate != "" {
			return ev.EventDate
		}
		if ev.SourceTs > 0 {
			return shadowmodel.ResolveEventDateHCM(ev.SourceTs, fallback)
		}
	}
	return shadowmodel.ResolveEventDateHCM(0, fallback)
}
// splitTaskByEventDate tách 1 NatsMsgTask chứa sự kiện vắt qua nửa đêm thành các sub-task theo ngày,
// dùng atomic.Int32 + sync.Once để chỉ ACK NATS message gốc khi TẤT CẢ sub-task đều thành công.
```

### File 4: `centralized-data-service/internal/service/master/transmuter_events_batch.go`, `transmuter.go` & `transmuter_events_batch_test.go`
```go
// Trong transmuter_events_batch.go (file mới tách toàn bộ vòng lặp batching RunEvents, dedupMasterEventRecords, lookupShadowGpayIDs khỏi transmuter.go):
// - Khi xóa (ev.IsDelete || ev.Op == "d"): chỉ cộng res.Written += delCnt (không cộng khống khi delCnt == 0).
// - Khi dedup (dedupMasterEventRecords): nếu conflictTarget == "_gpay_id" thì dedup theo item.sourceID + "|" + item.keySuffix
//   để các event cùng SourceID được gộp chính xác dù mỗi event copy_1_to_1 sinh _gpay_id Sonyflake mới.
```

### File 5: `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go` & `activity_log_read_repo_gorm_test.go`
```go
// Trong CTE act_filtered: dùng al.created_at >= NOW() - ((? + 1) * INTERVAL '1 day') để không cắt cụt đầu ngày xa nhất.
// Trong CTE act_daily:
	act_daily AS (
		SELECT
			COALESCE(
				NULLIF(e.details->>'event_date', ''),
				TO_CHAR(e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD')
			) AS day_str,
			...
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by <> 'snapshot-hook'
			         THEN COALESCE(
			             (e.details->>'written')::bigint,
			             NULLIF(e.rows_affected, 0),
			             COALESCE((e.details->>'inserted')::bigint, 0) + COALESCE((e.details->>'updated')::bigint, 0) + COALESCE((e.details->>'deleted')::bigint, 0)
			         ) ELSE 0 END) AS master_written,
```

