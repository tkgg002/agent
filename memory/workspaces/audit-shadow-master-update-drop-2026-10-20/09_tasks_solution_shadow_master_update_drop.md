---
tier: T2
suffix: shadow_master_update_drop
workspace: audit-shadow-master-update-drop-2026-10-20
status: PROPOSE_ONLY_WAITING_APPROVE
created_at: 2026-10-20T14:08:00+07:00
---

# 09. Giải pháp Kỹ thuật Tối ưu: Triệt tiêu Rớt UPDATE/DELETE Chặng Shadow -> Master

> [!IMPORTANT]
> Tài liệu này tuân thủ nghiêm ngặt quy tắc **Single-Best Solution (Rule #9 S3)** và **Propose-Only (Rule #1A, #13)**. Không có bất kỳ file source code nào bị chỉnh sửa cho đến khi nhận được lệnh `APPROVE` từ User.

---

## 1. Tổng quan Kiến trúc Giải pháp Theo Từng Sơ Hở (`F1` $\rightarrow$ `F5`)

Để triệt tiêu hoàn toàn hiện tượng rớt `UPDATE`/`DELETE` ở chặng Shadow $\rightarrow$ Master và khôi phục 100% khả năng tự đối soát/chữa lành (Recon/Heal Segment B), giải pháp được thiết kế **ánh xạ 1:1 trực tiếp vào từng sơ hở (`F1` $\rightarrow$ `F5`)** theo nguyên tắc **Minimal Impact & Đúng DNA Dự án**:

1. **Giải pháp cho Sơ hở `F1` (Triệt tiêu lỗi SQL `42703` khi xóa trên Master & Cô lập Poison Pill đến cấp độ từng Event):**
   - **Xóa bản ghi trên Master theo đúng khóa cấu hình động từ `master_binding.transform_spec` (`parsePKFromSpec` / `resolveConflictTarget`):**
     - Trên mỗi bảng Master, cấu hình JSON `master_binding.transform_spec` đã định nghĩa tường minh cột khóa chính thông qua trường `"pk"` (được trích xuất động bởi hàm [`parsePKFromSpec(binding.TransformSpec)`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go#L465-L478) trong `resolveConflictTarget(binding, colTypes)`): bảng nào cấu hình `{"pk": "_id"}` thì `conflictTarget` là `"_id"`, bảng nào cấu hình `{"pk": "id"}` thì `conflictTarget` là `"id"`, nếu không khai báo `"pk"` thì mặc định là `"_gpay_id"`.
     - Trong cả [`TransmuterModule.RunEvents()`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L672-L688) và [`TransmuterModule.Run()`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L429-L431), loại bỏ hoàn toàn câu lệnh hardcode `DELETE FROM %s WHERE _source_id = ?`. Thay bằng hàm `deleteFromMasterBySourceIDs` sử dụng trực tiếp cột `conflictTarget` được đọc động từ `binding.TransformSpec["pk"]` (`DELETE FROM %s WHERE <quoteIdent(conflictTarget)> IN (?) OR "_gpay_id" IN (?)` khi `conflictTarget != "_gpay_id"`, hoặc `WHERE "_gpay_id" IN (?)` khi `conflictTarget == "_gpay_id"`).
   - **Bọc Retry với `isRetryableDBError` trong `RunEvents()` & Đồng bộ `isTransientError`:**
     - Bọc thao tác `DELETE` và `bulkUpsertMaster` trong `RunEvents()` bằng vòng lặp retry (`maxRetries = 3`) sử dụng [`isRetryableDBError`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1607-L1639) giống hệt `processBatch`.
     - Mở rộng [`isTransientError`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L826-L845) đồng bộ với `isRetryableDBError` (`"driver: bad connection"`, `"broken pipe"`, `"unexpected eof"`, `"conn closed"`, `"08006"`, `"57p01"`, `"40001"`, v.v.) để không bao giờ quy kết nhầm lỗi kết nối DB tạm thời thành Poison Pill.
   - **Chia đôi đệ quy đến tận cấp độ `Events` / `SourceIDs` trong [`TransmuteHandler.binarySearchSplit`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L690-L720):**
     - Khi `len(batch) == 1`, nếu `len(batch[0].Req.Events) > 1` (hoặc `len(batch[0].Req.SourceIDs) > 1`), tiếp tục chia đôi mảng `Events` / `SourceIDs` bên trong task đó (`splitSingleTaskPayload`) để thực thi riêng rẽ từng nửa, đảm bảo **chỉ cô lập đúng 1 bản ghi Poison Pill duy nhất** và **cứu sống 100% các bản ghi `UPDATE` hợp lệ còn lại trong cùng message NATS**!

2. **Giải pháp cho Sơ hở `F2` (Xóa bỏ Split-Brain `_gpay_id` giữa `Run` và `RunEvents` & Khôi phục Recon/Heal Segment B):**
   - **Đồng nhất `_gpay_id` giữa Shadow và Master trong [`TransmuterModule.RunEvents()`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L718-L724):** Trước khi xử lý các sự kiện trong `RunEvents()`, thực hiện 1 câu truy vấn batch nhẹ (`lookupShadowGpayIDs`: `SELECT "_source_id", "_gpay_id" FROM <shadow_table> WHERE "_source_id" IN (?)`) để lấy đúng `_gpay_id` (Sonyflake) thực tế đang lưu trên bảng Shadow và truyền vào `deterministicGpayID(shadowGID, e.KeySuffix)` giống hệt `Run()`. Chỉ khi bản ghi chưa tồn tại trên Shadow mới fallback về `deterministicSourceGpayID(ev.SourceID, e.KeySuffix)`.
   - **Tự động đồng bộ lại `_gpay_id` bị lệch khi Upsert trong [`bulkUpsertMaster()`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1438-L1443):** Khi `conflictTarget != "_gpay_id"` (tức bảng Master dùng khóa chính lấy từ `transform_spec["pk"]`), bỏ điều kiện chặn `(conflictTarget != "_gpay_id" && k == "_gpay_id")`, cho phép `DO UPDATE SET "_gpay_id" = EXCLUDED."_gpay_id"` để tự động chữa lành toàn bộ các dòng đang bị lệch `_gpay_id` giữa Shadow và Master.
   - **Ánh xạ khóa động theo `transform_spec["pk"]` khi Heal Segment B tại [`resolveSourceIDsForSegmentB`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go#L653-L734):** Nếu một số `_gpay_id` lịch sử trên Master (dạng SHA-256 cũ) không tìm thấy trực tiếp trong `shadow."_gpay_id"`, hệ thống đọc cột `"pk"` từ `master_binding.transform_spec` (`parsePKFromSpec`), truy vấn bảng Master lấy giá trị cột `<pk>` của các dòng `_gpay_id` đó để đưa vào danh sách `sourceIDs` cần transmute, chữa lành 100% các dòng bị Split-Brain `_gpay_id` lịch sử!

3. **Giải pháp cho Sơ hở `F3` (Vét đầy đủ các bản ghi cũ vừa được `UPDATE` trên Shadow khi chạy Incremental & Bắn NATS Transmute sau khi Heal Segment A):**
   - **Quét Incremental bắt được cả `INSERT` mới lẫn `UPDATE` mới trong [`TransmuterModule.Run()`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L330-L370):** Khi chạy incremental (`runType != "full"` và `len(onlySourceIDs) == 0`), ngoài các dòng mới có `"_gpay_id"::bigint > currentLastID`, truy vấn lấy thêm các dòng cũ (`"_gpay_id"::bigint <= currentLastID`) nhưng vừa được cập nhật trên Shadow dựa theo mốc `_synced_at >= lookback` (hoặc có `_source_ts` trên Shadow lớn hơn Master) để transmute sang Master ngay trong cùng chu kỳ.
   - **Kích hoạt Transmute ngay sau khi Heal Segment A tại [`ReconHealer.HealMissingIDs`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go#L548-L570) & [`HealWindow`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go#L720-L737):** Sau khi `schemaAdapter.UpsertRecord` ghi thành công các bản ghi được heal từ MongoDB vào bảng Shadow, xuất bản ngay message NATS `cdc.cmd.transmute` chứa `source_ids` vừa heal để đẩy thẳng từ Shadow sang Master.

4. **Giải pháp cho Sơ hở `F4` (Chống đảo thứ tự sự kiện `UPDATE` trong cùng cửa sổ Flush tại [`BatchBuffer.batchUpsert`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L728-L753)):**
   - **Gộp cột & Khử trùng lặp theo `PrimaryKeyValue` TRƯỚC KHI chia nhóm theo `getSignature(r)`:** Trong `batchUpsert()`, gọi `dedupByConflictKey(sc.Records)` (nâng cấp để merge các cột không `nil` từ sự kiện trước sang sự kiện sau của cùng `PrimaryKeyValue`) **trước khi** chia vào `groups[sig]`, và duyệt `groups` theo mảng thứ tự xác định `sigOrder` thay vì duyệt Go `map` ngẫu nhiên. Nhờ đó, mỗi `PrimaryKeyValue` chỉ xuất hiện ở đúng 1 nhóm `sig` duy nhất với đầy đủ cột mới nhất.

5. **Giải pháp cho Sơ hở `F5` (Chống rớt cột khi `records[0]` thiếu key trong [`bulkUpsertMaster`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1412) & Đồng bộ hóa `cron` qua `TableDebouncer`):**
   - Trong `bulkUpsertMaster()`, thu thập tập hợp (union) tất cả các tên cột từ mọi phần tử trong `records` thay vì chỉ lấy `sortedKeysAny(records[0])`.

---

## 2. Thiết kế Chi tiết & Code Demo Theo Từng Sơ Hở (`F1` $\rightarrow$ `F5`)

### 2.1. Thiết kế & Code Demo cho Sơ hở `F1`: Xóa trên Master theo `conflictTarget` (đọc động từ `transform_spec.pk`), Bọc Retry & Chia đôi Poison Pill cấp độ `Events`

#### A. Xóa bản ghi trên Master theo `conflictTarget` (`resolveConflictTarget` từ `transform_spec.pk`) và bọc Retry trong [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L670-L746) & [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L424-L440)

Hàm [`resolveConflictTarget(binding, colTypes)`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go#L465-L478) hiện có đã đọc động trường `"pk"` từ JSON `binding.TransformSpec` (`pk := parsePKFromSpec(binding.TransformSpec)` — bảng cấu hình `{"pk": "_id"}` trả về `"_id"`, bảng cấu hình `{"pk": "id"}` trả về `"id"`, mặc định trả về `"_gpay_id"`). Ta bổ sung helper `deleteFromMasterBySourceIDs` dùng chính `conflictTarget` này thay vì hardcode `_source_id`:

```go
// Helper mới trong internal/service/master/transmuter.go:
// Xóa bản ghi trên bảng Master dựa trên cột conflictTarget được đọc động từ binding.TransformSpec["pk"]
func (t *TransmuterModule) deleteFromMasterBySourceIDs(
	ctx context.Context,
	masterDB *gorm.DB,
	masterRow *masterBindingRuntime,
	colTypes map[string]string,
	sourceIDs []string,
	shadowGpayMap map[string]int64,
) error {
	if len(sourceIDs) == 0 {
		return nil
	}
	conflictTarget := resolveConflictTarget(masterRow, colTypes)
	quotedTarget := quoteTransmuteQualified(masterRow.MasterSchema, masterRow.MasterTable)

	// Tập hợp đầy đủ _gpay_id (cả Sonyflake từ Shadow lẫn deterministicSourceGpayID)
	gpayIDs := make([]int64, 0, len(sourceIDs)*2)
	for _, sid := range sourceIDs {
		if gid, ok := shadowGpayMap[sid]; ok && gid > 0 {
			gpayIDs = append(gpayIDs, deterministicGpayID(gid, ""))
		}
		gpayIDs = append(gpayIDs, deterministicSourceGpayID(sid, ""))
	}

	var execErr error
	for attempt := 0; attempt <= 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*200) * time.Millisecond)
		}
		if conflictTarget != "" && conflictTarget != "_gpay_id" && masterRow.Strategy != "flatten" {
			// Dùng trực tiếp cột PK lấy từ transform_spec["pk"] của bảng Master đó (như "_id" hoặc "id")
			delSQL := fmt.Sprintf(`DELETE FROM %s WHERE %s IN ? OR "_gpay_id" IN ?`,
				quotedTarget, quoteTransmuteIdent(conflictTarget))
			execErr = masterDB.WithContext(ctx).Exec(delSQL, sourceIDs, gpayIDs).Error
		} else {
			delSQL := fmt.Sprintf(`DELETE FROM %s WHERE "_gpay_id" IN ?`, quotedTarget)
			execErr = masterDB.WithContext(ctx).Exec(delSQL, gpayIDs).Error
		}
		if execErr == nil || !isRetryableDBError(execErr) {
			break
		}
	}
	return execErr
}
```

Và bọc retry `isRetryableDBError` cho cả nhánh `DELETE` lẫn `bulkUpsertMaster` trong `RunEvents()` ([transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L672-L740)) cũng như nhánh `toHardDelete` trong `Run()` ([transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L424-L440)).

#### B. Chia đôi đệ quy mảng `Events` / `SourceIDs` trong [`transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L690-L720) & Mở rộng [`isTransientError`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L826-L845)

```go
func (h *TransmuteHandler) binarySearchSplit(ctx context.Context, masterTable string, batch []NatsMsgTask) {
	if len(batch) == 0 {
		return
	}
	if len(batch) == 1 {
		task := batch[0]
		// Nếu trong 1 message NATS có nhiều Events hoặc SourceIDs, tiếp tục chia đôi mảng bên trong task
		// để cứu sống toàn bộ các bản ghi UPDATE hợp lệ và chỉ cô lập đúng 1 event lỗi duy nhất!
		if len(task.Req.Events) > 1 || len(task.Req.SourceIDs) > 1 {
			h.splitSingleTaskPayload(ctx, masterTable, task)
			return
		}

		observability.Ctx(ctx, h.logger).Error("Poison pill isolated successfully in transmute master!",
			zap.String("master", masterTable),
			zap.Any("source_ids", task.Req.SourceIDs))

		if h.activity != nil {
			logEntry := h.activity.Start("transmute_poison_pill", masterTable, task.Req.TriggeredBy)
			h.activity.FailWithDetails(logEntry, "poison pill isolated", 1, map[string]interface{}{
				"master_binding_id": task.Req.MasterBindingID,
				"poison_pill":       true,
				"source_ids":        task.Req.SourceIDs,
			})
		}

		poisonErr := fmt.Errorf("poison pill isolated")
		h.replyErr(ctx, task.Msg, task.Req.CorrelationID, "poison_pill_isolated")
		h.publishCompleted(ctx, task.Req, master.TransmuteResult{}, poisonErr, "")
		task.complete(poisonErr)
		return
	}

	mid := len(batch) / 2
	left := batch[:mid]
	right := batch[mid:]

	h.processSubBatch(ctx, masterTable, left)
	h.processSubBatch(ctx, masterTable, right)
}

func (h *TransmuteHandler) splitSingleTaskPayload(ctx context.Context, masterTable string, task NatsMsgTask) {
	if len(task.Req.Events) > 1 {
		mid := len(task.Req.Events) / 2
		leftReq := task.Req
		leftReq.Events = append([]shadowmodel.TransmuteEventItem(nil), task.Req.Events[:mid]...)
		rightReq := task.Req
		rightReq.Events = append([]shadowmodel.TransmuteEventItem(nil), task.Req.Events[mid:]...)

		h.processSubBatch(ctx, masterTable, []NatsMsgTask{{Req: leftReq}})
		h.processSubBatch(ctx, masterTable, []NatsMsgTask{{Req: rightReq}})
		task.complete(nil)
		return
	}
	if len(task.Req.SourceIDs) > 1 {
		mid := len(task.Req.SourceIDs) / 2
		leftReq := task.Req
		leftReq.SourceIDs = append([]string(nil), task.Req.SourceIDs[:mid]...)
		rightReq := task.Req
		rightReq.SourceIDs = append([]string(nil), task.Req.SourceIDs[mid:]...)

		h.processSubBatch(ctx, masterTable, []NatsMsgTask{{Req: leftReq}})
		h.processSubBatch(ctx, masterTable, []NatsMsgTask{{Req: rightReq}})
		task.complete(nil)
		return
	}
}
```

---

### 2.2. Thiết kế & Code Demo cho Sơ hở `F2`: Đồng nhất `_gpay_id` giữa Shadow và Master & Phục hồi Recon/Heal Segment B theo `transform_spec.pk`

#### A. Tra cứu `_gpay_id` thực tế từ bảng Shadow trong `RunEvents()` ([`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L668-L725))
```go
// Helper mới trong internal/service/master/transmuter.go:
// Tra cứu batch _source_id -> _gpay_id (Sonyflake) thực tế đang lưu trên bảng Shadow
func (t *TransmuterModule) lookupShadowGpayIDs(ctx context.Context, masterRow *masterBindingRuntime, sourceIDs []string) map[string]int64 {
	res := make(map[string]int64, len(sourceIDs))
	if len(sourceIDs) == 0 || t.shadowDB == nil {
		return res
	}
	quotedShadow := quoteTransmuteQualified(masterRow.ShadowSchema, masterRow.ShadowTable)
	type pair struct {
		SourceID string `gorm:"column:_source_id"`
		GpayID   int64  `gorm:"column:_gpay_id"`
	}
	var rows []pair
	err := t.shadowDB.WithContext(ctx).
		Table(quotedShadow).
		Select(`"_source_id", "_gpay_id"`).
		Where(`"_source_id" IN ?`, sourceIDs).
		Scan(&rows).Error
	if err != nil {
		t.logger.Warn("transmuter: lookup shadow _gpay_id failed, fallback to deterministic hash",
			zap.String("shadow", quotedShadow), zap.Error(err))
		return res
	}
	for _, r := range rows {
		if r.SourceID != "" && r.GpayID > 0 {
			res[r.SourceID] = r.GpayID
		}
	}
	return res
}
```
Trong `RunEvents()` ([transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L718-L724)):
```go
		records := make([]map[string]any, 0, len(emits))
		for _, e := range emits {
			rec := e.Cols
			if shadowGID, ok := shadowGpayMap[ev.SourceID]; ok && shadowGID > 0 {
				rec["_gpay_id"] = deterministicGpayID(shadowGID, e.KeySuffix)
			} else {
				rec["_gpay_id"] = deterministicSourceGpayID(ev.SourceID, e.KeySuffix)
			}
			rec["_source_ts"] = ev.SourceTs
			rec["_deleted"] = false
			records = append(records, rec)
		}
```

#### B. Cho phép cập nhật `"_gpay_id" = EXCLUDED."_gpay_id"` khi `conflictTarget != "_gpay_id"` trong [`bulkUpsertMaster`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1437-L1444)
```go
	conflictTarget := resolveConflictTarget(binding, colTypes)

	sets := make([]string, 0, len(keys))
	for _, k := range keys {
		// Chỉ bỏ qua đúng cột conflictTarget (được lấy động từ transform_spec["pk"]);
		// khi conflictTarget != "_gpay_id", cho phép cập nhật lại "_gpay_id" = EXCLUDED."_gpay_id" để đồng bộ 100% với Shadow!
		if k == conflictTarget {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", quoteTransmuteIdent(k), quoteTransmuteIdent(k)))
	}
	sets = append(sets, `"_updated_at" = NOW()`)
```

#### C. Ánh xạ ngược `_gpay_id` qua cột `transform_spec["pk"]` khi Heal Segment B tại [`recon_execute_heal_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go#L653-L734)
Trong `resolveSourceIDsForSegmentB()`: Nếu tra cứu `SELECT "_source_id"::text FROM shadow WHERE "_gpay_id"::text IN (?)` chưa tìm thấy hết các ID (do trên Master đang mang `_gpay_id` SHA-256 lịch sử), hàm đọc động tên cột `"pk"` từ `master_binding.transform_spec` và truy vấn trên bảng Master `SELECT <quoteIdent(specPK)>::text FROM master WHERE "_gpay_id"::text IN (?)` để lấy chính xác danh sách khóa gốc cần transmute lại từ Shadow sang Master.

---

### 2.3. Thiết kế & Code Demo cho Sơ hở `F3`: Quét bổ sung các bản ghi cũ vừa được `UPDATE` trên Shadow & Bắn NATS Transmute sau khi Heal Segment A

1. **Trong [`TransmuterModule.Run()`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L330-L370):**
   Khi chạy chế độ incremental (`!req.FullResync && len(req.OnlySourceIDs) == 0 && lastGpayID > 0`), sau khi quét các dòng mới `_gpay_id > lastGpayID`, thực hiện truy vấn lấy danh sách `_source_id` của các bản ghi cũ (`"_gpay_id"::bigint <= lastGpayID`) nhưng có `"_synced_at" >= lookbackWindow` trên bảng Shadow để chạy `fetchShadowBatch(ctx, masterRow, 0, recentUpdatedSourceIDs)` và đẩy toàn bộ các bản cập nhật đó sang Master.
2. **Trong [`ReconHealer.HealMissingIDs`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go#L548-L570) & [`HealWindow`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go#L720-L737):**
   Ngay sau khi upsert thành công các bản ghi được heal từ MongoDB vào bảng Shadow (`healedIDs`), tự động xuất bản message NATS `cdc.cmd.transmute` với `SourceIDs: healedIDs` cho các `master_binding` đang active của bảng Shadow đó.

---

### 2.4. Thiết kế & Code Demo cho Sơ hở `F4`: Khử trùng lặp & Gộp cột theo `PrimaryKeyValue` TRƯỚC KHI chia nhóm `getSignature(r)` trong [`batch_buffer.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L724-L753)

Tại [`batch_buffer.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L724-L753) và [`batch_buffer_utils.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_utils.go#L26-L42):
```go
				if !sc.IsDelete {
					// Khử trùng lặp và gộp cột theo PrimaryKeyValue TRƯỚC KHI chia nhóm theo getSignature(r)
					// để cùng 1 document không bao giờ bị tách vào 2 nhóm signature khác nhau và chạy ngược thứ tự do duyệt map!
					dedupedRecords := dedupByConflictKey(sc.Records)

					groups := make(map[string][]*shadow.UpsertRecord)
					var sigOrder []string
					for _, r := range dedupedRecords {
						sig := getSignature(r)
						if _, exists := groups[sig]; !exists {
							sigOrder = append(sigOrder, sig)
						}
						groups[sig] = append(groups[sig], r)
					}

					for _, sig := range sigOrder {
						subChunk := groups[sig]
						query, values := schemaAdapter.BuildBatchUpsertSQLInSchema(
							schema, schemaName, tableName, effectivePK, subChunk,
						)
						if query == "" {
							continue
						}
						res := tx.Exec(query, values...)
						if res.Error != nil {
							return res.Error // Rollback entire chunk on first error
						}
						txWritten += res.RowsAffected
					}
				}
```

---

### 2.5. Thiết kế & Code Demo cho Sơ hở `F5`: Hợp nhất (Union) toàn bộ Keys của `records` trong [`bulkUpsertMaster`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1412)

Tại [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1412):
```go
	// Thu thập union toàn bộ keys từ tất cả các phần tử trong records thay vì chỉ lấy records[0]
	keySet := make(map[string]struct{})
	for _, rec := range records {
		for k := range rec {
			keySet[k] = struct{}{}
		}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)
```

---

## 3. Kế hoạch Kiểm thử & Xác minh Runtime (Verification Plan sau khi có `APPROVE`)

1. **Unit & Integration Test:**
   - Kiểm chứng `RunEvents()` khi nhận hỗn hợp `INSERT`, `UPDATE`, `DELETE` trên bảng Master không có cột `_source_id` (với `transform_spec` cấu hình `{"pk": "_id"}` hoặc `{"pk": "id"}`): đảm bảo `DELETE` xóa đúng dòng theo cột `pk` trong `transform_spec`, `UPDATE` giữ nguyên `_gpay_id` Sonyflake của Shadow.
   - Kiểm chứng `binarySearchSplit` với `len(batch) == 1` và `len(Events) = 10` trong đó có 1 event lỗi ở giữa: đảm bảo 9 event `UPDATE` hợp lệ được ghi thành công vào Master và chỉ 1 event lỗi bị đánh dấu Poison Pill.
   - Kiểm chứng `BatchBuffer.batchUpsert` với 2 sự kiện khác signature của cùng 1 `PrimaryKeyValue` trong cùng cửa sổ flush.
2. **Runtime Verification trên môi trường thật (`docker` containers):**
   - Build và restart container `cdc-ingestion-service`.
   - Kích hoạt Transmute / Execute Heal cho toàn bộ 5 Active Bindings: `Binding 46` (`payment_bills`), `Binding 47` (`payment_bills_1`), `Binding 48` (`export_jobs`), `Binding 51` (`export_jobs_2`), `Binding 52` (`export_jobs_1`).
   - Truy vấn đối soát trực tiếp giữa `gpay-postgres-shadow` (`:5436`) và `gpay-postgres-dest` (`:5434`) / `gpay-postgres-master-2` (`:5437`):
     - Số dòng rớt `DELETE` (`55853, 55854`): giảm từ **`2` $\rightarrow$ `0`**.
     - Số dòng rớt `UPDATE` (`s._source_ts > m._source_ts`): giảm từ **`14` $\rightarrow$ `0`** trên `payment_bills` và từ **`344` $\rightarrow$ `0`** trên `payment_bills_1`.
     - Số dòng lệch `_gpay_id` (`s._gpay_id <> m._gpay_id`): giảm từ **`1,482` $\rightarrow$ `0`** trên `payment_bills`/`payment_bills_1` và từ **`520` $\rightarrow$ `0`** trên `export_jobs`.
     - Chạy lại Recon Segment B: `missing_in_shadow = 0`, `missing_in_master = 0`, `mismatched_Hash = 0`.

---

## 6. Kế Hoạch Bổ Sung Từ Phản Biện Cuối Workflow (`S5 → S3 Closed-Loop Re-Plan` Cho `AF-01..AF-05` — Đã Được User `APPROVE`)

### 6.1. Giải pháp cho `AF-01`: Quét động toàn bộ `is_active = true` trong `cdc_system.master_binding` (bao gồm `Binding 51, 52`) & Phối hợp Restart 3 Daemon Runtime (`:8081/:8082/:8083`)
- **Tại `internal/service/master/transmuter_gpayid_test.go:402`:** Thay vì hardcode `for _, bID := range []int64{46, 47, 48}`, truy vấn động `SELECT id FROM cdc_system.master_binding WHERE is_active = true ORDER BY id` để chạy đồng bộ và kiểm chứng `droppedDelete = 0, droppedUpdate = 0, gpayMismatch = 0` trên toàn bộ **5 active bindings (`46, 47, 48, 51, 52`)** (lấy đúng `masterDB` theo `row.MasterConnectionKey` cho cả `default_master` `:5434` lẫn `master_2` `:5437`).
- **Phối hợp Runtime:** Sau khi hoàn tất code & test, phối hợp với User rebuild binary và restart 3 tiến trình `main` đang lắng nghe trên `:8081` (`PID 34453`), `:8082` (`PID 50249`), `:8083` (`PID 50156`).

### 6.2. Giải pháp cho `AF-02`: Sửa lỗi nuốt lỗi 1 nửa trong `splitSingleTaskPayload` & Tránh chạy lặp các `Events` đã thành công trong `RunEvents`
- **Tại `internal/handler/master/transmute_handler.go:782-795` (`splitSingleTaskPayload`):**
  - Khi `leftErr != nil || rightErr != nil` (dù chỉ 1 nửa thất bại), bắt buộc lấy lỗi khác `nil` (`firstErr`) để gọi `h.replyErr(ctx, task.Msg, task.Req.CorrelationID, firstErr.Error())` và `task.complete(firstErr)` (thay vì `if leftErr != nil && rightErr != nil`), đảm bảo JetStream `Nak()`/redeliver đúng phần tử lỗi và không treo RPC reply. Khi cả 2 nửa thành công (`leftErr == nil && rightErr == nil`), gọi `h.reply(task.Msg, ...)` trước `task.complete(nil)`.
- **Code Demo (`internal/handler/master/transmute_handler.go`):**
  ```go
  if isTransientError(leftErr) {
      task.complete(leftErr)
      return
  }
  if isTransientError(rightErr) {
      task.complete(rightErr)
      return
  }
  if leftErr != nil || rightErr != nil {
      firstErr := leftErr
      if firstErr == nil {
          firstErr = rightErr
      }
      h.replyErr(ctx, task.Msg, task.Req.CorrelationID, firstErr.Error())
      task.complete(firstErr)
      return
  }
  h.reply(task.Msg, map[string]any{
      "status":         "COMPLETED",
      "correlation_id": task.Req.CorrelationID,
      "table":          masterTable,
  })
  task.complete(nil)
  ```

### 6.3. Giải pháp cho `AF-03`: Bảo toàn `LastCursorJSON` khi chạy Incremental Sync & Giới hạn `cutoff` của `syncUpdatedShadowRowsBeforeCursor` theo `LastSuccessAt`
- **Tại `internal/service/master/transmuter.go:552-557`:** Chỉ reset `item.LastCursorJSON = []byte("{}")` khi `runType == "full" && len(onlySourceIDs) == 0 && lastBatchErr == nil`. Với Incremental Sync (`runType != "full"`), giữ nguyên `last_gpay_id` vừa cập nhật!
- **Tại `internal/service/master/transmuter.go:1475-1525` (`syncUpdatedShadowRowsBeforeCursor`):**
  - Đọc `last_success_at` từ `t.runtimeRepo.GetByMasterID(ctx, masterRow.ID)` làm mốc `cutoff` (lùi biên an toàn 10 phút: `state.LastSuccessAt.Add(-10 * time.Minute)`), và chỉ khi chưa có `LastSuccessAt` mới fallback `24h` kèm điều kiện `_updated_at > _created_at` để không quét lại 54,754 dòng mới `INSERT`.
- **Code Demo (`internal/service/master/transmuter.go`):**
  ```go
  // Chỉ reset checkpoint khi đồng bộ Full Sync hoàn tất thành công
  if runType == "full" && len(onlySourceIDs) == 0 && lastBatchErr == nil && t.runtimeRepo != nil {
      t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
          item.LastCursorJSON = []byte(`{}`)
      })
  }
  ```

### 6.4. Giải pháp cho `AF-04`: Chia chunk & lấy động `shadow_connection_key` trong `publishTransmuteShadowForSegA` + Bổ sung publish Transmute cho `executeLegacyHeal`
- **Tại `internal/handler/recon/recon_execute_heal_handler.go:339-381` (`publishTransmuteShadowForSegA`):**
  - Lấy `shadowConnKey` từ `MasterBindingRef` (hoặc mặc định `"default_shadow"` thay vì `"default"`), khử trùng lặp `uniqueStrings(sourceIDs)` và chia lô theo `healChunkSize = 200` giống `publishTransmuteChunked`.
- **Tại `internal/handler/recon/recon_check_heal_handler.go:302-360` (`executeLegacyHeal`):**
  - Sau khi `healSvc.HealMissingIDs` hoặc `healSvc.HealWindow` hoàn tất với `healed > 0`, publish thông điệp `cdc.cmd.transmute-shadow` để đồng bộ sang Master.

### 6.5. Giải pháp cho `AF-05`: Khử trùng lặp `sourceIDs` trong `resolveSourceIDsForSegmentB` & So sánh `SourceTsMs` trong `dedupByConflictKey`
- **Tại `internal/handler/recon/recon_execute_heal_handler.go:733-753` (`resolveSourceIDsForSegmentB`):**
  - Khử trùng lặp `return uniqueStrings(out)` ở cuối hàm `resolveSourceIDsForSegmentB` để không bắn trùng `sourceIDs` khi vừa khớp một phần trên Shadow vừa fallback lên Master.
- **Tại `internal/handler/shadow/batch_buffer_utils.go:90-136` (`dedupByConflictKey`):**
  - Khi hai bản ghi trong cùng `sc.Records` có cùng `PrimaryKeyValue`, so sánh `r.SourceTsMs >= records[prevIdx].SourceTsMs` trước khi cập nhật `lastIdx[pk]` và chỉ cho phép giá trị cột của bản ghi có `SourceTsMs` mới hơn (hoặc đứng sau khi `SourceTsMs` bằng nhau) ghi đè lên `mergedData[pk]`.
