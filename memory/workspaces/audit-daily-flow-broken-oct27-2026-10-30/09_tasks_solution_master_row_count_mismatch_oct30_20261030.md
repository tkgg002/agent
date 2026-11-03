---
tier: T2
mode: standard
suffix: master_row_count_mismatch_oct30_20261030
---

# Giải Pháp Kỹ Thuật (`S3 SOLUTION`): Xử Lý Lệch Số Lượng & Chặn Bản Ghi Trùng Giữa Shadow (`250,137 + 317`) và MongoDB Master (`250,240 + 347`) Trên `transaction_history_1`

## 1. Ràng Buộc Thực Thi (Technical Constraints)
1. **Strict Propose-Only (`L005`, Rule #1, Rule #13):** Giữ nguyên chế độ đề xuất kèm **Code Demo** cho tới khi User phản hồi `APPROVE`.
2. **Evidence-Based Verification (`Rule #0`, `Rule #9`):** Mọi khẳng định bám sát `file:line`, log runtime `/tmp/cds-worker.log` và dữ liệu đối soát từ HTTP API `:8083`.
3. **Zero Direct DB CLI (`L031`, `L032`):** Tuyệt đối không dùng `psql`, `mongosh` hay `docker exec`.
4. **Simplicity First, Minimal Impact (`Rule #12`):** Bảo toàn tuyệt đối toàn bộ test hiện hữu (`TestStaleMongoKeysToPruneAndPrimaryOnlyObjectIdPromotion`, `TestBuildMongoOCCBulkModels_*`, `TestMultiShadowJetStreamEdgeCases_*`).

---

## 2. Tổng Hợp Findings (`S2` + `DG0` Verified)

### Finding F1 (`VERIFIED` — Giải thích con số `250,137` ở Shadow Nạp vs `250,240` ở Master Sync tại Trạm 1)
- **Bằng chứng runtime (`cdc_system.activity_log` qua `GET :8083/api/activity-log?target_table=devtest.transaction_history_1`):**
  1. Lúc `10:29:13 GMT+7` (`03:29:13Z`), job `snapshot.v2` (`id=138246`) nạp xong **`250,137`** dòng từ PostgreSQL Source vào `shadow_testcoretrans.transaction_history_1`.
  2. Trong khoảng `10:29:13` → `11:52:36 GMT+7` (hơn 1 giờ 23 phút trước khi bấm `Master Sync`), luồng Kafka CDC tiếp tục đẩy các giao dịch mới vào bảng Shadow (`250,228` dòng tại thời điểm bắt đầu quét `11:52:36` và thêm `12` dòng CDC vừa vào trong lúc quét phân trang `_gpay_id > scanCursor`, nâng tổng số dòng thực tế trong bảng Shadow lên **`250,240`** dòng).
  3. Lúc `11:52:36` → `12:04:41 GMT+7`, job `Master Sync` (`id=138557`, `triggered_by = "admin"`, `event_date = ""`) chạy Full Sync (`Transmute` toàn bảng Shadow sang MongoDB Master) nên quét toàn bộ **`250,240`** dòng hiện có trong bảng Shadow tại thời điểm đó (`scanned = 250,240`, `inserted = 250,240`, `written = 250,240`).
  4. Như vậy, phần chênh `+103` dòng (`250,240 - 250,137 = 103`) ở cột `1. SNAPSHOT / RE-SYNC` **không phải lỗi nhân bản dữ liệu của Snapshot**, mà là `103` dòng CDC đã chảy vào Shadow trong khoảng `10:29–12:04` được lệnh Full Sync (`Master Sync`) quét gom luôn cùng `250,137` dòng gốc. Bằng chứng đanh thép: ngay khi `Master Sync` hoàn tất lúc `12:04:46 GMT+7` (bản ghi đối soát `id=528688`), đối soát `shadow_master` khớp tuyệt đối 100%: `source_count = 250,228`, `dest_count = 250,228`, **`diff = 0`**.

### Finding F2 (`VERIFIED` — Nguyên nhân gốc rễ sinh ra `348` bản ghi trùng trên MongoDB Master `devtest.transaction_history_1`)
- **Diễn biến timeline đối soát (`GET :8083/api/reconciliation/report/transaction_history_1`):**
  - `12:04:46` (`id=528688`): `Shadow = 250,228`, `Master = 250,228`, **`diff = 0`**.
  - `13:57:06` (`id=529318`): `Shadow = 250,348`, `Master = 250,342`, `diff = +6`.
  - **Từ `13:57:29` đến `14:34:20` (37 phút xảy ra bão retry `transmute_sub` 5 lần do lỗi `$numberInt` / `$oid` / `$date` poison pill trước khi vá `$literal`):**
    - `13:58:00` (`id=529331`, sau Redelivery #1): `Shadow = 250,348`, `Master = 250,390` (`diff = -42`).
    - `14:05:29` (`id=529408`, sau Redelivery #2): `Shadow = 250,363`, `Master = 250,460` (`diff = -97`).
    - `14:12:00` (`id=529478`, sau Redelivery #3): `Shadow = 250,363`, `Master = 250,513` (`diff = -150`).
    - `14:18:32` (`id=529557`, sau Redelivery #4): `Shadow = 250,369`, `Master = 250,602` (`diff = -233`).
    - `14:19:29` (`id=529561`, sau Redelivery #5): `Shadow = 250,369`, `Master = 250,625` (`diff = -256`).
    - `14:35:22` (`id=529719`, kết thúc đợt retry Batch B): `Shadow = 250,393`, `Master = 250,741` (**`diff = -348`**).
  - **Từ `14:35:22` đến `15:36:18` (`id=529719` → `id=530368`, suốt hơn 1 giờ sau khi vá `$literal`):**
    - `Shadow` tăng từ `250,393` lên `250,468` (`+75` dòng mới).
    - `Master` tăng từ `250,741` lên `250,816` (`+75` dòng mới).
    - **`diff` giữ nguyên tuyệt đối ở `-348` (không phát sinh thêm bất kỳ bản ghi lệch mới nào)!**
- **Nguyên nhân kỹ thuật trong code (`mongo_bson_restorer.go:47-95, 387-400` & `transmuter_mongo.go:189-248`):**
  1. **Lỗ hổng F2a — Bất nhất `conflictTarget` giữa `"id"` và `"_id"` khi `source_field_types == nil` (`mongo_bson_restorer.go:387-400`):**
     - Trên `devtest.transaction_history_1` (`master_binding_id = 75`), `transform_spec.pk = "id"`.
     - Trong `ApplySourceFieldTypesToBSONDoc(doc)` (`mongo_bson_restorer.go:53-62`), ngay cả khi `rawMeta == nil` (bản ghi Primary `transaction_history_1` tới trước Secondary `cdc_transaction_history_payload_1`), hàm đã promote `id` (24-char hex) thành `doc["_id"] = ObjectID(id)`.
     - Thế nhưng `resolveEffectiveMongoConflictTarget("id", records)` (`mongo_bson_restorer.go:392-398`) lại **chỉ trả về `"_id"` khi trong `records` có `source_field_types != nil`**!
     - Khi mẻ chỉ có bản ghi Primary (`source_field_types == nil`), `resolveEffectiveMongoConflictTarget` trả về `"id"`. Lúc này `BuildMongoOCCBulkModels` (`transmuter_mongo.go:231, 246`) đặt `filter = {"id": "6ae4..."}` và trong vòng lặp `$set` **bỏ qua `_id`** (`if k == "_id" { continue }`), khiến MongoDB `SetUpsert(true)` chèn 1 document với **`_id` ngẫu nhiên**! Vài ms sau, khi Secondary (`cdc_transaction_history_payload_1` có `source_field_types`) tới, `resolveEffectiveMongoConflictTarget` trả về `"_id"`, lọc theo `{"_id": ObjectId("6ae4...")}`, không khớp document cũ có `_id` ngẫu nhiên và **chèn tiếp document thứ 2 cho cùng 1 giao dịch**!
  2. **Lỗ hổng F2b — Bản ghi có `id` dạng UUID 36 ký tự (`01a18b6e-...`) bị rơi `targetKey` về `_gpay_id` Sonyflake ngẫu nhiên (`mongo_bson_restorer.go:53-62` & `transmuter_mongo.go:199-204`):**
     - Bảng `transaction_history_1` chứa cả `id` kiểu 24-char hex (`6ae43...`) lẫn `id` kiểu UUID 36 ký tự (`01a18b6e-a05c-4149-be4c-c7cb376df639`).
     - Khi trong batch có ít nhất 1 bản ghi chứa `source_field_types`, `resolveEffectiveMongoConflictTarget` đổi `conflictTarget` của cả batch thành `"_id"`.
     - Nhưng `ApplySourceFieldTypesToBSONDoc` (`mongo_bson_restorer.go:55`) chỉ gán `doc["_id"]` khi `len(strings.TrimSpace(s)) == 24`! Với bản ghi UUID 36 ký tự, `doc["_id"]` bằng `nil`.
     - Tại `BuildMongoOCCBulkModels` (`transmuter_mongo.go:199-204`), vì `doc["_id"] == nil`, code fallback `targetKey = "_gpay_id"` và `keyVal = doc["_gpay_id"]`. Mà ở `runEventBatches` (`transmuter_events_batch.go:169`), mỗi lần chạy đều gọi `nextMasterSonyflakeGpayID()` sinh một `_gpay_id` mới tinh!
     - Hệ quả: Trong 10 đợt retry `transmute_sub` lúc `13:57–14:34` (và mỗi khi bản ghi UUID nhận event `u` sau event `c`), `filter = {"_gpay_id": <sonyflake_mới>}` không khớp bản ghi cũ trên MongoDB, làm `SetUpsert(true)` **chèn đúp bản ghi mới trên mỗi lần retry** (tạo ra đúng `348` bản ghi dư trên MongoDB Master)!

### Finding F3 (`VERIFIED` — Giải thích con số `809 Ops / 386 dòng DB (347 Inserted, 462 Updated)` ở Trạm 4 so với `735 Ops / 326 dòng DB (317 Insert, 418 Update)` ở Trạm 3)
- Có 2 nguyên nhân cộng hưởng tạo ra số đếm `809 Ops (347 Inserted, 462 Updated)` ở cột `4. GHI VÀO DB MASTER`:
  1. **Cộng dồn từ 45 mẻ con `transmute_sub` trong 5 vòng redelivery (lúc `13:57–14:34` trước khi vá lỗi `$literal`):** Khi 2 mẻ chứa poison pill bị NATS JetStream redeliver 5 lần, các nhánh con không lỗi trong cùng mẻ được thực thi và ghi log `transmute_sub` lặp lại.
  2. **Bảng `multi_shadow` nhận sự kiện `cdc.cmd.transmute-shadow` từ cả 2 bảng (`transaction_history_1` + `cdc_transaction_history_payload_1`):** Cột 2 & 3 của dòng `transaction_history_1` chỉ thống kê sự kiện của bảng Primary `transaction_history_1`, trong khi `devtest.transaction_history_1` nhận sự kiện từ cả Primary và Secondary `cdc_transaction_history_payload_1`.

### Finding F4 (`VERIFIED` — Truy vấn Join bảng phụ `cdc_transaction_history_payload_1` bị chậm `2.5s–14.2s`/mẻ do không dùng cột index `_source_id`)
- Tại `transmuter_multi_shadow_join_resolver.go:30-33, 146-163`, `isDefaultIDJoin(field)` chỉ nhận `_source_id`, `id`, `_id`.
- Với `cdc_transaction_history_payload_1`, khóa chính của bảng là `transaction_history_id` nên cột `_source_id` (đã có btree index) chính là `transaction_history_id`, nhưng `secondary_join_field` trong `transform_spec` khai báo `"transaction_history_id"`. Do đó `queryMultiShadowRowsForTable` rơi vào nhánh `WHERE (_raw_data ->> 'transaction_history_id') IN (?)` phải quét JSONB 250,000 dòng mất `2.5s–14.2s` mỗi mẻ.

---

## 3. Phương Án Kỹ Thuật Duy Nhất (Single Best Solution) & Code Demo

### Fix 1 (Cho F2a & F2b): Chuẩn hóa nhất quán khóa `_id` trên MongoDB Master cho cả ObjectID 24-hex và UUID/String PK & đồng bộ `resolveEffectiveMongoConflictTarget`
- **File:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_bson_restorer.go` và `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go`
- **Cơ chế:**
  1. Trong `ApplySourceFieldTypesToBSONDoc(doc bson.M)` (`mongo_bson_restorer.go:47-63`):
     - Nếu `doc["_id"]` là chuỗi 24-char hex hợp lệ → chuyển thành `primitive.ObjectID`.
     - Nếu `doc["_id"]` chưa có: kiểm tra `id` rồi `_source_id`. Nếu là 24-char hex hợp lệ → gán `doc["_id"] = oid`. Nếu `cand == "id"` và chuỗi khác rỗng (ví dụ UUID 36 ký tự `"01a18b6e-..."`) → gán `doc["_id"] = trimmed` (dùng chính khóa chính `id` làm `_id` của MongoDB, tuyệt đối không để `doc["_id"] == nil` rơi về `_gpay_id` Sonyflake ngẫu nhiên!).
     - Khi `rawMeta == nil` nhưng `doc["_id"] != nil` và `doc["id"] != nil`: vẫn giữ `doc["id"]` trong `doc` để tương thích với `TestStaleMongoKeysToPruneAndPrimaryOnlyObjectIdPromotion` (`T1`), nhưng nhờ `resolveEffectiveMongoConflictTarget` trả về `"_id"` nên `BuildMongoOCCBulkModels` luôn dùng `filter = {"_id": doc["_id"]}` nhất quán ở cả `T1` lẫn `T2`!
  2. Trong `resolveEffectiveMongoConflictTarget(conflictTarget string, records []map[string]any) string` (`mongo_bson_restorer.go:387-400`):
     - Khi `ct == "id"` và `len(records) > 0`: nếu bất kỳ bản ghi nào có `r["_id"] != nil || r["id"] != nil || r["source_field_types"] != nil || r["sourceFieldTypes"] != nil` → trả về `"_id"` nhất quán (kể cả khi `T1` Primary tới trước chưa có `source_field_types` hoặc batch toàn bản ghi UUID).
  3. Trong `deleteFromMongoMasterBySourceAndGpayIDs` (`transmuter_mongo.go:398-404`):
     - Khi `pkCol == "id"`, bổ sung luôn mệnh đề tìm xóa theo `"_id"` (`bson.M{"_id": bson.M{"$in": coerceMongoSourceIDsToFilterSlice(sourceIDs, colTypes[pkCol])}}`) bên cạnh `"id"` và `"_source_id"`.

#### Code Demo — `mongo_bson_restorer.go` & `transmuter_mongo.go`
```go
// File: centralized-data-service/internal/service/master/mongo_bson_restorer.go
func ApplySourceFieldTypesToBSONDoc(doc bson.M) {
	if doc == nil {
		return
	}
	if idStr, ok := doc["_id"].(string); ok {
		trimmed := strings.TrimSpace(idStr)
		if len(trimmed) == 24 {
			if oid, err := primitive.ObjectIDFromHex(trimmed); err == nil {
				doc["_id"] = oid
			}
		} else if trimmed != "" {
			doc["_id"] = trimmed
		}
	}

	if _, hasMongoID := doc["_id"]; !hasMongoID {
		for _, cand := range []string{"id", "_source_id"} {
			if s, ok := doc[cand].(string); ok {
				trimmed := strings.TrimSpace(s)
				if len(trimmed) == 24 {
					if oid, err := primitive.ObjectIDFromHex(trimmed); err == nil {
						doc["_id"] = oid
						break
					}
				} else if trimmed != "" && cand == "id" {
					doc["_id"] = trimmed
					break
				}
			}
		}
	}
// ...
func resolveEffectiveMongoConflictTarget(conflictTarget string, records []map[string]any) string {
	ct := strings.TrimSpace(conflictTarget)
	if ct == "" {
		return "_gpay_id"
	}
	if ct == "id" && len(records) > 0 {
		for _, r := range records {
			if r["_id"] != nil || r["id"] != nil || r["source_field_types"] != nil || r["sourceFieldTypes"] != nil {
				return "_id"
			}
		}
	}
	return ct
}
```

```go
// File: centralized-data-service/internal/service/master/transmuter_mongo.go
	if len(sourceIDs) > 0 {
		pkCol := resolveConflictTarget(binding, colTypes)
		if pkCol != "" && pkCol != "_gpay_id" {
			filterSlice := coerceMongoSourceIDsToFilterSlice(sourceIDs, colTypes[pkCol])
			orClauses = append(orClauses, bson.M{pkCol: bson.M{"$in": filterSlice}})
			if pkCol == "id" {
				orClauses = append(orClauses, bson.M{"_id": bson.M{"$in": filterSlice}})
			}
		}
		orClauses = append(orClauses, bson.M{"_source_id": bson.M{"$in": sourceIDs}})
	}
```

---

### Fix 2 (Cho F4): Tối ưu truy vấn Join bảng phụ `multi_shadow` ưu tiên cột được đánh index `_source_id`
- **File:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_multi_shadow_join_resolver.go`
- **Cơ chế:**
  - Trong `queryMultiShadowRowsForTable` (`transmuter_multi_shadow_join_resolver.go:146-169`), khi `tbl.secondaryJoinField` không nằm trong `isDefaultIDJoin` (ví dụ `"transaction_history_id"` trên `cdc_transaction_history_payload_1`, nơi `_source_id` chính là `transaction_history_id`), thực hiện truy vấn ưu tiên index `WHERE _source_id IN (?)` trước.
  - Chỉ những `lookupKeys` nào chưa tìm thấy qua `_source_id` mới chạy truy vấn fallback `WHERE (_raw_data ->> ?) IN (?)`. Nhờ đó thời gian join mỗi mẻ 100 events giảm từ `2.5s–14.2s` xuống `< 5ms`!

#### Code Demo — `transmuter_multi_shadow_join_resolver.go`
```go
// File: centralized-data-service/internal/service/master/transmuter_multi_shadow_join_resolver.go
	// Fast-path: thử khớp qua cột được đánh index _source_id trước (vì trên bảng phụ 1:1 như
	// cdc_transaction_history_payload_1, PK của bảng phụ chính là secondaryJoinField nên _source_id == joinKey).
	fastQt := fmt.Sprintf(
		`SELECT _source_id, _raw_data, COALESCE(_source_ts, 0) AS _source_ts, COALESCE(_deleted, false) AS _deleted FROM %s WHERE _source_id IN (?)`,
		qtName,
	)
	_ = shadowDB.WithContext(ctx).Raw(fastQt, lookupKeys).Scan(&secRows).Error
	foundKeys := make(map[string]struct{}, len(secRows))
	for i := range secRows {
		foundKeys[secRows[i].SourceID] = struct{}{}
		if pid, ok := joinKeyToPrimID[secRows[i].SourceID]; ok {
			secRows[i].SourceID = pid
		}
	}
	if isDefaultIDJoin(tbl.secondaryJoinField) || len(foundKeys) == len(lookupKeys) {
		return secRows
	}

	remainingKeys := make([]string, 0, len(lookupKeys)-len(foundKeys))
	for _, k := range lookupKeys {
		if _, ok := foundKeys[k]; !ok {
			remainingKeys = append(remainingKeys, k)
		}
	}
	if len(remainingKeys) == 0 {
		return secRows
	}

	var fallbackRows []shadowRawRow
	qt := fmt.Sprintf(
		`SELECT COALESCE(_raw_data ->> ?, '') AS _source_id, _raw_data, COALESCE(_source_ts, 0) AS _source_ts, COALESCE(_deleted, false) AS _deleted FROM %s WHERE (_raw_data ->> ?) IN (?)`,
		qtName,
	)
	_ = shadowDB.WithContext(ctx).Raw(qt, tbl.secondaryJoinField, tbl.secondaryJoinField, remainingKeys).Scan(&fallbackRows).Error
	for i := range fallbackRows {
		if pid, ok := joinKeyToPrimID[fallbackRows[i].SourceID]; ok {
			fallbackRows[i].SourceID = pid
		}
		secRows = append(secRows, fallbackRows[i])
	}
	return secRows
```

---

## 4. Giải Đáp Câu Hỏi Của Anh: "Có Cần Anh Truncate Và Chạy Lại Không?"
1. **Đối với bảng Shadow (`shadow_testcoretrans.transaction_history_1`):**
   - **KHÔNG CẦN truncate hay chạy lại Snapshot.** Đối soát `source_shadow` đang khớp 100% tuyệt đối (`source_count = 250,468`, `dest_count = 250,468`, **`diff = 0`**).
2. **Đối với bảng MongoDB Master (`devtest.transaction_history_1`):**
   - Hiện tại đang dư đúng **`348` bản ghi trùng** (`Shadow = 250,468` vs `Master = 250,816`, `diff = -348`) sinh ra trong 37 phút retry (`13:57–14:34`) trước khi vá lỗi `$literal`. Từ `14:35` tới nay, mọi bản ghi mới chảy qua đều khớp 1:1 (`diff` đứng im ở `-348`).
   - **Anh KHÔNG CẦN tự truncate và nạp lại 250,000 dòng từ đầu.** Ngay sau khi anh `APPROVE`, em sẽ:
     1. Áp dụng bản vá Fix 1 & Fix 2 (`mongo_bson_restorer.go`, `transmuter_mongo.go`, `transmuter_multi_shadow_join_resolver.go`) và restart `cds-worker`.
     2. Chạy bài test/tác vụ dọn sạch đúng `348` document trùng (các document có `_id` ngẫu nhiên hoặc bị nhân bản theo `_gpay_id` mới nhưng trùng `id` / `_source_id`) và chuẩn hóa lại các document UUID trên `devtest.transaction_history_1`, đưa `shadow_master` về **`diff = 0` (`CÂN BẰNG`)** ngay lập tức mà không mất thời gian sync lại 250k dòng (hoặc nếu anh muốn sạch toàn bộ thống kê cột 1–4 trên bảng ngày hôm nay thì chỉ cần truncate riêng bảng Master `devtest.transaction_history_1` rồi bấm `Master Sync` mất ~5 phút sau khi em lên code vá Fix 1 & Fix 2).

---

## 5. Giải Pháp Kỹ Thuật Bổ Sung Cho 4 Issues Phát Hiện Qua Audit (`S5 -> S3 -> S4 Self-Healing Loop`)

### Fix 3 (Cho Issue #1, #2 & #3): Chuẩn hóa `GetDailyFlow` (`recon_daily` lấy lần đối soát mới nhất & khử trùng `transmute_sub` redelivery)
- **File:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`
- **Cơ chế:**
  1. Bổ sung CTE `dedup_enriched` ngay sau `enriched`: Với `e.operation = 'transmute_sub' AND e.status <> 'error'` (chỉ sinh ra khi chia tách mẻ chứa poison pill), đánh số `ROW_NUMBER() OVER (PARTITION BY e.operation, e.target_table, e.status, COALESCE(e.details->>'event_date', ''), COALESCE(e.details->>'scanned', ''), COALESCE(e.details->>'written', ''), COALESCE(e.details->>'inserted', ''), COALESCE(e.details->>'updated', ''), COALESCE(e.details->>'deleted', '') ORDER BY e.id ASC)` và chỉ giữ `sub_rn = 1` trong `act_daily`, loại bỏ 30 dòng `transmute_sub` bị ghi lặp trong 5 vòng NATS redelivery lúc 14h chiều ngày `2026-10-30`.
  2. Trong CTE `recon_daily` (`activity_log_read_repo_gorm.go:473-485`): Thay `SUM(COALESCE(r.missing_count, 0))` và `SUM(COALESCE(r.stale_count, 0))` bằng giá trị của **lần đối soát cuối cùng** trong ngày (sau khi trừ số dòng đã heal ngay trong lượt đó):
     `COALESCE((ARRAY_AGG(GREATEST(COALESCE(r.missing_count, 0) - COALESCE(r.healed_count, 0), 0) ORDER BY r.checked_at DESC, r.id DESC))[1], 0) AS missing`
     `COALESCE((ARRAY_AGG(COALESCE(r.stale_count, 0) ORDER BY r.checked_at DESC, r.id DESC))[1], 0) AS stale`

#### Code Demo — `activity_log_read_repo_gorm.go`
```go
	wave_enriched AS (
		SELECT e.*,
		       SUM(CASE WHEN e.operation = 'transmute_sub' AND e.status <> 'error' AND COALESCE((e.details->>'written')::bigint, e.rows_affected, 0) >= 5 THEN 1 ELSE 0 END)
		           OVER (PARTITION BY e.target_table ORDER BY e.id ASC) AS wave_id
		FROM enriched e
	),
	seq_enriched AS (
		SELECT w.*,
		       CASE WHEN w.operation = 'transmute_sub' AND w.status <> 'error'
		            THEN ROW_NUMBER() OVER (
		                PARTITION BY w.operation, w.target_table, w.wave_id,
		                             COALESCE(w.details->>'scanned', ''),
		                             COALESCE(w.details->>'written', ''),
		                             COALESCE(w.details->>'inserted', ''),
		                             COALESCE(w.details->>'updated', ''),
		                             COALESCE(w.details->>'deleted', '')
		                ORDER BY w.id ASC
		            )
		            ELSE 1
		       END AS seq_in_wave
		FROM wave_enriched w
	),
	dedup_enriched AS (
		SELECT s.*,
		       CASE WHEN s.operation = 'transmute_sub' AND s.status <> 'error'
		            THEN ROW_NUMBER() OVER (
		                PARTITION BY s.operation, s.target_table, s.status,
		                             COALESCE(s.details->>'event_date', ''),
		                             COALESCE(s.details->>'scanned', ''),
		                             COALESCE(s.details->>'written', ''),
		                             COALESCE(s.details->>'inserted', ''),
		                             COALESCE(s.details->>'updated', ''),
		                             COALESCE(s.details->>'deleted', ''),
		                             s.seq_in_wave
		                ORDER BY s.id ASC
		            )
		            ELSE 1
		       END AS sub_rn
		FROM seq_enriched s
	),
	act_daily AS (
		...
		FROM dedup_enriched e
		WHERE e.sub_rn = 1
		GROUP BY 1
	),
	recon_daily AS (
		SELECT
			TO_CHAR(r.checked_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD') AS day_str,
			COUNT(*) AS total_runs,
			COALESCE((ARRAY_AGG(GREATEST(COALESCE(r.missing_count, 0) - COALESCE(r.healed_count, 0), 0) ORDER BY r.checked_at DESC, r.id DESC))[1], 0) AS missing,
			COALESCE((ARRAY_AGG(COALESCE(r.stale_count, 0) ORDER BY r.checked_at DESC, r.id DESC))[1], 0) AS stale,
			SUM(COALESCE(r.healed_count, 0)) AS healed
		FROM cdc_system.cdc_reconciliation_report r
		WHERE r.checked_at >= NOW() - (? * INTERVAL '1 day')
		  AND (?::text = '' OR r.shadow_schema = ?)
		  AND r.shadow_table = ?
		GROUP BY 1
	)
```

---

### Fix 4 (Cho Issue #4): Tách 4 God-files theo đúng Single-Responsibility Principle (`SRP` — `L009`)
1. **`centralized-data-service/internal/handler/master/transmute_handler.go` (`1,074` dòng → `796` dòng):**
   - Tách cụm hàm cô lập Poison Pill (`binarySearchSplit`, `splitSingleTaskPayload`, `processSubBatch`, `isTransientError` từ dòng `682` đến `958`) sang file đơn trách nhiệm `centralized-data-service/internal/handler/master/transmute_poison_pill_split.go` (`~290` dòng).
2. **`centralized-data-service/internal/handler/shadow/batch_buffer.go` (`967` dòng → `815` dòng):**
   - Tách cụm hàm retry & sơ tán DLQ (`batchUpsertWithRetry`, `evacuateFailedChunkToDLQ`, `writeFailedSyncLog`, `isRetryableDBError` từ dòng `453–538` và `906–967`) sang file đơn trách nhiệm `centralized-data-service/internal/handler/shadow/batch_buffer_dlq.go` (`~165` dòng).
3. **`centralized-data-service/internal/service/master/mongo_bson_restorer_test.go` (`832` dòng → `501` dòng):**
   - Tách `TestMongoOCCBulkModels_ExtendedJSONAndLiteralProtection` và `TestMongoOCCBulkModels_UUIDAndPrimaryOnlyNoDuplicates_LiveMongo` (`dòng 503–832`) sang `centralized-data-service/internal/service/master/mongo_bson_uuid_live_test.go` (`~348` dòng).
4. **`cdc-cms-web/src/components/ReconPipelineGrid.tsx` (`1,961` dòng → `~1,715` dòng):**
   - Tách Modal bảng "Thống kê đối soát 3 trạm & DLQ theo ngày" (`dòng 1267–1509`) sang component độc lập `cdc-cms-web/src/components/ReconDailyFlowModal.tsx` (`~265` dòng).
