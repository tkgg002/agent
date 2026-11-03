---
tier: T2
mode: standard
suffix: dlq_188_missing_oct30_20261030
---

# Giải pháp Kỹ thuật & Code Demo (S3): Điều tra & Khắc phục 188 lỗi DLQ ẩn và lệch cột Master Sync trên bảng Thống kê đối soát 3 trạm theo ngày (`2026-10-30`)

## 1. Bảng Ánh Xạ 1-1 Các Phát Hiện (`F1..F4`) & Bằng Chứng Vật Lý (`file:line` + Runtime)

| Mã | Hiện tượng trên bảng `2026-10-30` | Root Cause vật lý (`file:line` & Runtime Evidence) | Giải pháp duy nhất tối ưu (`S3`) |
|---|---|---|---|
| **F1** | Cột `5. ĐỐI SOÁT & DLQ -> DLQ / Lỗi` báo **`188`** (và tiếp tục nhảy lên `350`), nhưng mở tab **`Log Transmute`** trong Drawer của Pipeline (`shadow_testcoretrans.transaction_history_1 -> devtest.transaction_history_1`) và trang **`Activity Log`** lại **không thấy bóng dáng 188 dòng DLQ đó đâu**. | 1. Cột `DLQ / Lỗi` trong `GetDailyFlow` (`cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:464`) đếm `SUM(CASE WHEN e.status = 'error' AND (e.details->>'split_retry')::boolean IS NOT TRUE THEN 1 ELSE 0 END)` trên bảng `cdc_activity_log`. Toàn bộ `188` dòng (nay là `350` dòng) của ngày `2026-10-30` đều mang `operation = 'transmute_poison_pill'` (`target_table = 'devtest.transaction_history_1'`, `master_binding_id = 75`, `triggered_by = 'kafka-consumer-hook'`).<br>2. Tuy nhiên, tại `cdc-cms-web/src/components/ReconPipelineGrid.tsx:486`, tab **`Log Transmute`** gọi `usePipelineActivityLog(historyMaster, 'transmute,transmute_sub', ...)` — **bỏ sót `'transmute_poison_pill'`**, khiến 100% các bản ghi `transmute_poison_pill` bị lọc mất khỏi UI!<br>3. Trên trang `ActivityLog.tsx:48-60, 103-132, 165`, bộ lọc `operationOptions` và cột `Scope` cũng chỉ nhận diện `'transmute'`, thiếu `'transmute_sub'` và `'transmute_poison_pill'`.<br>4. Tại `centralized-data-service/internal/handler/master/transmute_handler.go:703-709`, khi cô lập Poison Pill, `binarySearchSplit` ghi cứng `error_message = "poison pill isolated"` làm mất nguyên nhân lỗi gốc của DB Master (`Invalid $set :: caused by :: Unrecognized expression '$numberInt'`). | 1. Cập nhật `ReconPipelineGrid.tsx:486` truyền `'transmute,transmute_sub,transmute_poison_pill'` và hiển thị thêm `source_ids` / `operation` trong bảng `Log Transmute`.<br>2. Bổ sung `'transmute_sub'`, `'transmute_poison_pill'` vào `ActivityLog.tsx:48-60, 103-132, 165`.<br>3. Truyền lỗi gốc `causeErr` vào `binarySearchSplit` / `splitSingleTaskPayload` (`transmute_handler.go:635, 681-717, 728-797, 871`) để lưu đầy đủ thông điệp lỗi DB thật kèm `source_ids` vào `error_message` của `transmute_poison_pill`. |
| **F2** | Mở tab **`Lỗi đồng bộ (0)`** trên trang `Toàn vẹn dữ liệu` (`DataIntegrity.tsx` đọc từ `GET /api/failed-sync-logs` -> bảng `cdc_system.failed_sync_logs`) cũng báo **`Không có lỗi đồng bộ`**; đồng thời chỉ có **21 `_source_id` duy nhất** nhưng bị nhân bản **14–24 lần/bản ghi** thành `188 -> 350` dòng `transmute_poison_pill`. | 1. Trong `transmute_handler.go:681-717` (`binarySearchSplit`) và `783-790` (`splitSingleTaskPayload`), sau khi đã tách riêng 1 bản ghi Poison Pill (và cứu thành công các bản ghi lành lặn cùng mẻ), hàm vẫn trả `poisonErr` về `task.complete(poisonErr)` -> `finalizeJetStreamMessage` (`transmute_consumer_pool.go:344-378`) gọi `msg.NakWithDelay`. Hệ quả: NATS JetStream phát lại toàn bộ message đó `MaxDeliver = 5` lần (`server_setup.go:327`), mỗi lần lại chạy lại Binary Search Split từ đầu và ghi lặp thêm dòng `transmute_poison_pill` mới cho cùng 1 `_source_id`!<br>2. `GetDailyFlow` (`activity_log_read_repo_gorm.go:464`) đếm tổng số dòng `status = 'error'` thô mà không khử trùng lặp theo `_source_id` của `transmute_poison_pill` trong ngày.<br>3. Tại `DataIntegrity.tsx:95`, `useFailedLogs(50)` gọi `GET /api/failed-sync-logs?page_size=50` không truyền `status=failed`, lấy 50 dòng mới nhất của các bảng cũ đã tắt (`transaction_history`, `cdc_transaction_history_payload`) rồi lọc client-side theo `activePipelines` ra `0` dòng! | 1. Trong `activity_log_read_repo_gorm.go:464` (`GetDailyFlow`), đếm `dlq_count` chính xác theo số bản ghi Poison Pill duy nhất trong ngày (`COUNT(DISTINCT CASE WHEN e.operation = 'transmute_poison_pill' AND e.status = 'error' THEN COALESCE(NULLIF(e.details->'source_ids'->>0, ''), e.id::text) WHEN e.status = 'error' AND (e.details->>'split_retry')::boolean IS NOT TRUE THEN e.id::text END)`).<br>2. Trong `useReconStatus.ts:163-176` & `DataIntegrity.tsx:95`, truyền `status: 'failed'` và `page_size: 200` khi tải `failed-sync-logs`. |
| **F3** | Vì sao 21 bản ghi CDC của `transaction_history_1` bị lỗi khi Transmute sang MongoDB Master (`devtest.transaction_history_1`) trong khi `250,137` bản ghi Snapshot trước đó lại nạp thành công? | 1. Báo lỗi thực tế trong `cdc_activity_log` (`id=139707..139719`):<br>`mongo bulk write on transaction_history_1: bulk write exception: write errors: [Invalid $set :: caused by :: Unrecognized expression '$numberInt']`.<br>2. Tại `centralized-data-service/internal/service/master/transmuter_mongo.go:234-264`, khi chạy **Snapshot** (`isFullSync == true` hoặc `incomingTs == 0`), `BuildMongoOCCBulkModels` dùng Standard Update Document (`bson.M{"$set": setDoc, "$setOnInsert": setOnInsertDoc}`), nên MongoDB không parse biểu thức Aggregation.<br>3. Khi chạy **CDC Stream** (`!isFullSync && incomingTs > 0`), `BuildMongoOCCBulkModels` dùng **Aggregation Pipeline Update** (`[]bson.M{{"$set": setStage}}`) với `setStage[k] = bson.M{"$cond": bson.A{occCond, v, "$" + k}}`. Trong Aggregation Pipeline Update, mọi map con bên trong `v` có key bắt đầu bằng `$` (như `{"$numberInt": "0"}`, `{"$numberLong": "..."}`, `{"$oid": "..."}`, `{"$date": ...}` sinh ra từ Extended JSON trong cột JSONB của Shadow) bị MongoDB hiểu là toán tử Aggregation vì:<br>  - `convertMapToBSONM` (`mongo_type_converter.go:292-328`) chưa unwrap các wrapper Extended JSON (`$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, `$oid`, `$date`).<br>  - `BuildMongoOCCBulkModels` (`transmuter_mongo.go:250-258`) truyền trực tiếp `v` vào `$cond` / `$ifNull` mà không bọc `bson.M{"$literal": v}`! | 1. Trong `mongo_type_converter.go:292-328`, unwrap triệt để các map Extended JSON đơn khoá (`$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, `$oid`, `$date`) sang kiểu BSON chuẩn (`int32`, `int64`, `float64`, `primitive.Decimal128`, `primitive.ObjectID`, `primitive.DateTime`).<br>2. Trong `transmuter_mongo.go:250-258`, bọc giá trị `v` bằng `bson.M{"$literal": v}` trong nhánh `$ifNull` và `$cond` của Aggregation Pipeline Update (`bson.M{"$cond": bson.A{occCond, bson.M{"$literal": v}, "$" + k}}`) để khoá cứng `v` là hằng số dữ liệu, chặn đứng vĩnh viễn lỗi `Unrecognized expression '$...'`. |
| **F4** | Trên dòng `2026-10-30`, cột `1. SNAPSHOT / RE-SYNC -> Shadow Nạp = 250,137` nhưng **`Master Sync = —` (`0`)**, trong khi cột **`4. GHI VÀO DB MASTER (Từ CDC Stream)` lại vọt lên `250,634 Ops` (`250,420 dòng DB`)** dù Kafka CDC chỉ có `539 Ops`. | 1. Tại `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:73-75`, khi `snapshotBatchBuffer` (`bb.isSnapshotBuffer == true`) fanout sang Transmute Master, nó gán `triggeredBy = "snapshot"`.<br>2. Thế nhưng tại `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:438` và `454-462`:<br>  - Dòng `438` (`snapshot_master`) chỉ kiểm tra `e.triggered_by = 'snapshot-hook'`, **bỏ sót `'snapshot'`**, khiến `snapshot_master = 0` (`—`).<br>  - Dòng `454-462` (`master_written`, `master_inserted`, `master_updated`, `master_deleted`, `master_skipped`) chỉ loại trừ `e.triggered_by <> 'snapshot-hook'`, **không loại trừ `'snapshot'`**, khiến toàn bộ `250,137` dòng Snapshot nạp lên Master bị cộng dồn nhầm vào cột `4. GHI VÀO DB MASTER (Từ CDC Stream)`! | Cập nhật `activity_log_read_repo_gorm.go:438` và `454-462`: nhận diện đầy đủ `e.triggered_by IN ('snapshot', 'snapshot-hook') OR COALESCE((e.details->>'is_snapshot')::boolean, false)` cho `snapshot_master`, và loại trừ điều kiện này khỏi `master_written`, `master_inserted`, `master_updated`, `master_deleted`, `master_skipped`. |

---

## 2. Ràng Buộc Thực Thi (Technical Constraints từ Lessons)
- **`L005` (Strict Propose-Only Gate):** Giữ nguyên chế độ Propose-Only ở bước `S3`, trình bày đầy đủ phân tích + Code Demo và chờ anh `APPROVE` trước khi chạm vào source code.
- **`L018` (SQL Column Projection Sync):** Giữ nguyên thứ tự các tham số `?` và các cột projection của `GetDailyFlow` (`activity_log_read_repo_gorm.go:397-516`).
- **`L021` (Never Kill User Processes):** Không kill các tiến trình của anh trên máy.
- **`L031` / `L032` (Zero Direct DB CLI):** Tuyệt đối không dùng `psql` hay `docker exec`.

---

## 3. Code Demo Chi Tiết Cho Từng File

### Demo 1 (`F3`): `centralized-data-service/internal/service/master/mongo_type_converter.go` & `transmuter_mongo.go`
Unwrap Extended JSON (`$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, `$oid`, `$date`) trong `convertMapToBSONM` và bọc `bson.M{"$literal": v}` trong Aggregation Pipeline Update của `BuildMongoOCCBulkModels`:

```go
// File: centralized-data-service/internal/service/master/mongo_type_converter.go
func unwrapExtendedJSONMap(m map[string]any) (any, bool) {
	if len(m) != 1 {
		return nil, false
	}
	for k, rawVal := range m {
		switch k {
		case "$numberInt":
			if v, err := coerceInt32(rawVal); err == nil {
				return v, true
			}
		case "$numberLong":
			if v, err := coerceInt64(rawVal); err == nil {
				return v, true
			}
		case "$numberDouble":
			if f, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprintf("%v", rawVal)), 64); err == nil {
				return f, true
			}
		case "$numberDecimal":
			if dec, err := coerceDecimal128(rawVal); err == nil {
				return dec, true
			}
		case "$oid":
			if oid, err := coerceObjectID(rawVal); err == nil {
				return oid, true
			}
		case "$date":
			if subMap, ok := rawVal.(map[string]any); ok {
				if unwrapped, ok2 := unwrapExtendedJSONMap(subMap); ok2 {
					rawVal = unwrapped
				}
			}
			if dt, err := coerceDateTime(rawVal); err == nil {
				return dt, true
			}
		}
	}
	return nil, false
}

func convertNestedElement(v any) any {
	switch x := v.(type) {
	case map[string]any:
		if unwrapped, ok := unwrapExtendedJSONMap(x); ok {
			return unwrapped
		}
		return convertMapToBSONM(x)
	case []any:
		return convertSliceToBSONA(x)
	case json.Number:
		if i, err := x.Int64(); err == nil {
			if i >= math.MinInt32 && i <= math.MaxInt32 {
				return int32(i)
			}
			return i
		}
		if f, err := x.Float64(); err == nil {
			return f
		}
		return x.String()
	default:
		return v
	}
}
```

Và trong `centralized-data-service/internal/service/master/transmuter_mongo.go:245-258`:
```go
// File: centralized-data-service/internal/service/master/transmuter_mongo.go
			for k, v := range doc {
				if k == "_id" {
					continue
				}
				literalVal := bson.M{"$literal": v}
				if k == "_gpay_id" && targetKey != "_gpay_id" {
					setStage[k] = bson.M{
						"$ifNull": bson.A{"$_gpay_id", literalVal},
					}
					continue
				}
				setStage[k] = bson.M{
					"$cond": bson.A{occCond, literalVal, "$" + k},
				}
			}
```

### Demo 2 (`F1` & `F2`): `centralized-data-service/internal/handler/master/transmute_handler.go`
Truyền nguyên nhân lỗi DB thật (`causeErr`) xuống `binarySearchSplit` và `splitSingleTaskPayload` để bản ghi `transmute_poison_pill` hiển thị đúng lỗi gốc kèm `source_ids`:

```go
// File: centralized-data-service/internal/handler/master/transmute_handler.go
func (h *TransmuteHandler) binarySearchSplit(ctx context.Context, masterTable string, batch []NatsMsgTask, causeErr ...error) {
	var rootErr error
	if len(causeErr) > 0 {
		rootErr = causeErr[0]
	}
	if len(batch) == 1 {
		task := batch[0]
		if len(task.Req.Events) > 1 || (len(task.Req.Events) == 0 && len(task.Req.SourceIDs) > 1) {
			h.splitSingleTaskPayload(ctx, masterTable, task, rootErr)
			return
		}

		isolatedIDs := task.Req.SourceIDs
		if len(isolatedIDs) == 0 && len(task.Req.Events) == 1 {
			isolatedIDs = []string{task.Req.Events[0].SourceID}
		}

		errText := "poison pill isolated"
		if rootErr != nil && strings.TrimSpace(rootErr.Error()) != "" {
			errText = fmt.Sprintf("poison pill isolated (source_ids=%v): %s", isolatedIDs, rootErr.Error())
		}

		observability.Ctx(ctx, h.logger).Error("Poison pill isolated successfully in transmute master!",
			zap.String("master", masterTable),
			zap.Any("source_ids", isolatedIDs),
			zap.Error(rootErr))

		if h.activity != nil {
			eventDate := taskEventDate(task, time.Now())
			logEntry := h.activity.Start("transmute_poison_pill", masterTable, task.Req.TriggeredBy)
			h.activity.FailWithDetails(logEntry, errText, 1, map[string]interface{}{
				"master_binding_id": task.Req.MasterBindingID,
				"event_date":        eventDate,
				"poison_pill":       true,
				"source_ids":        isolatedIDs,
			})
		}
```

### Demo 3 (`F2` & `F4`): `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`
1. Tách chuẩn xác `snapshot_master` khi `e.triggered_by IN ('snapshot', 'snapshot-hook') OR COALESCE((e.details->>'is_snapshot')::boolean, false)` và loại trừ khỏi 5 cột `master_*` của CDC Stream (dòng `438, 454-462`).
2. Khử trùng lặp `transmute_poison_pill` theo `source_ids->>0` trong ngày khi tính `dlq_count` (dòng `464`):

```sql
-- File: cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:438-465
			SUM(CASE WHEN (e.triggered_by IN ('snapshot', 'snapshot-hook') OR COALESCE((e.details->>'is_snapshot')::boolean, false)) AND e.operation IN ('transmute','transmute_sub') AND e.status <> 'error'
			         THEN COALESCE((e.details->>'written')::bigint, NULLIF(e.rows_affected, 0), COALESCE((e.details->>'inserted')::bigint, 0) + COALESCE((e.details->>'updated')::bigint, 0) + COALESCE((e.details->>'deleted')::bigint, 0)) ELSE 0 END) AS snapshot_master,
			-- ...
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by NOT IN ('snapshot', 'snapshot-hook') AND NOT COALESCE((e.details->>'is_snapshot')::boolean, false)
			         THEN COALESCE((e.details->>'written')::bigint, NULLIF(e.rows_affected, 0), COALESCE((e.details->>'inserted')::bigint, 0) + COALESCE((e.details->>'updated')::bigint, 0) + COALESCE((e.details->>'deleted')::bigint, 0)) ELSE 0 END) AS master_written,
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by NOT IN ('snapshot', 'snapshot-hook') AND NOT COALESCE((e.details->>'is_snapshot')::boolean, false)
			         THEN COALESCE((e.details->>'inserted')::bigint, 0) ELSE 0 END) AS master_inserted,
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by NOT IN ('snapshot', 'snapshot-hook') AND NOT COALESCE((e.details->>'is_snapshot')::boolean, false)
			         THEN COALESCE((e.details->>'updated')::bigint, 0) ELSE 0 END) AS master_updated,
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by NOT IN ('snapshot', 'snapshot-hook') AND NOT COALESCE((e.details->>'is_snapshot')::boolean, false)
			         THEN COALESCE((e.details->>'deleted')::bigint, 0) ELSE 0 END) AS master_deleted,
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by NOT IN ('snapshot', 'snapshot-hook') AND NOT COALESCE((e.details->>'is_snapshot')::boolean, false)
			         THEN COALESCE((e.details->>'skipped')::bigint, 0) ELSE 0 END) AS master_skipped,
			COUNT(DISTINCT CASE
				WHEN e.operation = 'transmute_poison_pill' AND e.status = 'error'
				THEN COALESCE(NULLIF(e.details->'source_ids'->>0, ''), e.id::text)
				WHEN e.status = 'error' AND (e.details->>'split_retry')::boolean IS NOT TRUE
				THEN e.id::text
			END) AS dlq_count
```

### Demo 4 (`F1`): `cdc-cms-web/src/components/ReconPipelineGrid.tsx` & `cdc-cms-web/src/pages/ActivityLog.tsx`
Hiển thị đầy đủ `transmute_poison_pill` ngay trong tab **`Log Transmute`** của Drawer và trên trang **`Activity Log`**:

```tsx
// File: cdc-cms-web/src/components/ReconPipelineGrid.tsx:486
  const { data: transmuteLog, isLoading: isLoadingTransmute } = usePipelineActivityLog(
    historyMaster,
    'transmute,transmute_sub,transmute_poison_pill',
    sourceDb,
    historySchema,
    LOG_FETCH_LIMIT,
  );
```
