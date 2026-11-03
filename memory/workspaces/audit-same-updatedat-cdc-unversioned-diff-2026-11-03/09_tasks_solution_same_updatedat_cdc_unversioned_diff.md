---
tier: T2
workspace: audit-same-updatedat-cdc-unversioned-diff-2026-11-03
suffix: same_updatedat_cdc_unversioned_diff
mode: propose-only
---

# 09. Technical Solution — Audit & Khắc phục Lệch `unversioned_diff` Khi 2 Lệnh Ghi Cùng Giây Oplog / `updatedAt` (Case `261029KHHDUH` — `MERCHANT_PAYMENT`)

## 1. Bảng Tổng Hợp Issues (Executive Summary Table)

| ID | Mức độ | Component & `file:line` | Vấn đề & Căn nguyên Vật lý (Root Cause) | Tác động Runtime (Case `261029KHHDUH`) | Giải pháp Tối ưu Nhất (1-1) |
| :--- | :---: | :--- | :--- | :--- | :--- |
| **`F1`** | **CRITICAL (P0)** | `centralized-data-service`<br>- `internal/service/shadow/schema_adapter.go:771-783`<br>- `internal/handler/shadow/utils.go:72-105`<br>- `internal/handler/shadow/batch_buffer.go:674-750` | **(1) Mất thứ tự trong cùng giây Oplog:** Debezium MongoDB gán `source.ts_ms = seconds * 1000` (từ BSON Oplog `Timestamp(seconds, ord)`), còn thứ tự trong cùng 1 giây nằm ở `source.ord`. Hàm `extractSourceTsMs` (`utils.go:72-105`) chỉ đọc `ts_ms` mà bỏ qua `ord`, làm 2 sự kiện cách nhau `23 ms` trong cùng giây `05:17:04Z` có cùng `_source_ts = 1761715024000`.<br>**(2) Cổng OCC dùng dấu `<` ngặt:** `buildOCCWhereClause` (`schema_adapter.go:774,781`) ép `_source_ts < EXCLUDED._source_ts` (trừ khi `_source = 'snapshot:v2'`). Khi 2 sự kiện cùng giây rơi vào 2 đợt flush/SQL khác nhau, sự kiện thứ 2 (`PENDING`) bị chặn (`0 rows affected`) và bị nuốt mất vĩnh viễn. | Postgres Shadow & Master giữ kẹt bản ghi đầu tiên (`PROCESSING`) suốt gần 5 giờ (`05:17:04Z` $\rightarrow$ `10:59:26Z`), gây 34 lần `unversioned_diff` khi Admin xem danh sách `MERCHANT_PAYMENT`. Chỉ tự hết lệch nhờ có lệnh `SUCCESS` lúc `10:59:26Z` mang `_source_ts` lớn hơn; nếu không có lệnh ghi sau đó, Postgres sẽ **sai mãi mãi**. | **(1A)** Tại `utils.go:72-105`: Kết hợp `source.ord` (giới hạn `1..999`) vào `ts_ms` khi `ts_ms % 1000 == 0` để bảo toàn thứ tự nhân quả sub-second của MongoDB Oplog.<br>**(1B)** Tại `schema_adapter.go:771-783`: Cho phép `_source_ts = EXCLUDED._source_ts AND EXCLUDED._source <> 'snapshot:v2'` (cổng `_hash IS DISTINCT FROM EXCLUDED._hash` đã chặn replay trùng nội dung), đồng bộ với `dedupByConflictKey` (`>=`) và `transmuter.go:1698` (`>=`). |
| **`F2`** | **HIGH (P1)** | `core-trans-proxy-service`<br>- `publish/cashin.publish.ts:34-44, 514-564`<br>`core-trans-proxy-history-service`<br>- `start.ts:50-55`<br>- `models/transHis.model.ts:28-32, 92-114` | **(1) Fire-and-Forget không `await` & Mutate chung object:** `CashInPublish.createTransHis` và `updateTransHis` gọi `handlePromise(ctx.call(...))` không `await` và gán đè trực tiếp lên cùng biến tham chiếu `transHisParam`.<br>**(2) Độ trễ bất đối xứng tạo log ngược `23 ms`:** `createTransHis` trong `start.ts:50-53` chạy thêm `createIpAddress(ctx)` + `$save` (validate full schema + ghi mọi index + trả full document), nặng hơn `$updateOne` của `updateTransHisStatus`. Trên MongoDB, `insertOne(PROCESSING)` thực thi trước vài ms nên `updateOne(PENDING)` tìm thấy bản ghi (`n:1, nModified:1`), nhưng `updateTransHisStatus` trả response về trước `createTransHis` `23 ms`. | - Giải thích trọn vẹn vì sao log `updateTransHisStatus` xuất hiện trước `createTransHis` `23 ms` mà `n:1, nModified:1` và `insertOne` không bị lỗi trùng khóa `unique: true` (`transHis.model.ts:31`).<br>- Rủi ro tiềm ẩn: nếu gói mạng `updateOne` tới MongoDB trước `insertOne`, `updateOne` sẽ trả `n:0` (mất lệnh cập nhật trạng thái) làm cả Mongo v1 lẫn Postgres v2 kẹt ở `PROCESSING`. | Tại `cashin.publish.ts:34-44`: Clone nông payload `{ ...transHisParam, status, error: generateTransHisError(error) }` để tránh race condition mutate chung tham chiếu, và trả về Promise để các luồng có thể `await` hoặc chuỗi hóa thứ tự `createTransHis` $\rightarrow$ `updateTransHis`. |

---

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)

- **`L013` (`LS-GLOBAL-001-propose-only-gate`):** Giữ chế độ **100% Read-Only (Propose-Only)** đối với toàn bộ source code dự án; hiển thị trực tiếp **Code Demo** trên khung chat và dừng chờ `APPROVE`.
- **`L004` (`Verifiable Code Demo Snippets`):** Mọi đoạn `Code Demo (Before)` trích xuất nguyên văn 100% từ `HEAD` kèm số dòng `file:line`.
- **`L025` & `L028` (`Timestamp Precision & CDC Ordering`):** Mọi cơ chế OCC trên luồng CDC phải tính đến độ phân giải giây của BSON Oplog `Timestamp(seconds, ord)` và đồng bộ toán tử so sánh giữa tầng In-Memory Batch (`dedupByConflictKey`), tầng Shadow (`schema_adapter.go`) và tầng Master (`transmuter.go`).

---

## 3. Giải Mã Toàn Trình Case `261029KHHDUH` & Kiểm Định Giả Thuyết Của User

### 3.1. Giải mã Hiện tượng #1: Tại sao log `updateTransHisStatus` (`PENDING`, `error: 1172`, `n:1, nModified:1`) xuất hiện TRƯỚC `createTransHis` (`insertOne`, `PROCESSING`) `23 ms`, nhưng `insertOne` vẫn thành công và Mongo v1 giữ `PENDING`?

1. **Bác bỏ giả định `insertOne` chạy sau `updateOne` trên MongoDB hoặc bản ghi đã tồn tại từ 3 giờ trước:**
   - Tại `/Users/trainguyen/Documents/work/core-trans-proxy-history-service/models/transHis.model.ts:28-32`, trường `transId` có ràng buộc **`unique: true`** (và index `{ transId: 1 }` tại dòng `79`):
     ```ts
     transId: {
         type: String,
         required: [true, "transId required"],
         unique: true,
     },
     ```
   - Nếu bản ghi `transId = "261029KHHDUH"` thực sự đã có trong MongoDB trước khi `createTransHis` chạy `insertOne` (`MongoFuncHelper.$save` tại `transHis.model.ts:92-94`), lệnh `insertOne` **bắt buộc phải văng lỗi `E11000 duplicate key error`** chứ không thể trả thành công. Ngược lại, nếu `insertOne(PROCESSING)` thực thi trên MongoDB *sau* `updateOne(PENDING)`, trạng thái cuối trên MongoDB phải là `PROCESSING` chứ không thể là `PENDING`.
2. **Cơ chế vật lý thực sự tạo ra khoảng chênh log `23 ms`:**
   - Tại `/Users/trainguyen/Documents/work/core-trans-proxy-service/publish/cashin.publish.ts:34-44`, cả 2 hàm `createTransHis` và `updateTransHis` đều gọi `handlePromise(ctx.call(...))` **không có `await`** (Fire-and-Forget):
     ```ts
     private static createTransHis(ctx: Context, transHisParam, status, error: any = null) {
         transHisParam.error = generateTransHisError(error);
         transHisParam.status = status;
         handlePromise(ctx.call(`${SVC_ENV.get().CORE_TRANS_PROXY_HISTORY_SERVICE}.createTransHis`, transHisParam, { meta: ctx.meta }))
     }
     ```
   - Khi giao dịch `261029KHHDUH` bắt đầu lúc `05:17:04Z`, `core-trans` bắn `createTransHis(PROCESSING)` trước (không chờ `await`), rồi chỉ vài mili-giây sau bắn tiếp `updateTransHisStatus(PENDING, error: 1172)`.
   - Tại `/Users/trainguyen/Documents/work/core-trans-proxy-history-service/start.ts:50-55`:
     ```ts
     createTransHis: (ctx) => {
         this.ipAddressSavePublish.createIpAddress(ctx);
         return this.transHisSavePublish.createTrans(ctx);
     },
     updateTransHisStatus: (ctx) =>
         this.transHisSavePublish.updateTransStatus(ctx),
     ```
   - Trên MongoDB Primary:
     - **Bước 1 (`T0`):** Lệnh `insertOne(status: PROCESSING)` của `createTransHis` tới MongoDB trước, ghi bản ghi mới vào WiredTiger và Oplog tại giây `05:17:04Z` với thứ tự Oplog `ord = k`.
     - **Bước 2 (`T0 + vài ms`, cùng giây `05:17:04Z`):** Trong lúc `createTransHis` còn bận chạy `createIpAddress(ctx)` và serialize toàn bộ document lớn trả về qua Moleculer, lệnh `updateOne({ transId: "261029KHHDUH" }, { $set: { status: "PENDING", error: 1172 } })` của `updateTransHisStatus` tới MongoDB! Vì `insertOne` vừa commit ở Bước 1, `updateOne` tìm thấy ngay bản ghi đó (`n: 1, nModified: 1`), cập nhật MongoDB sang **`PENDING`** (cùng giây `05:17:04Z`, thứ tự Oplog `ord = k + 1`), và do payload response `{ n: 1, nModified: 1, ok: 1 }` cực nhẹ nên `updateTransHisStatus` kết thúc và ghi log response **sớm hơn `createTransHis` `23 ms`**!

---

### 3.2. Kiểm định Suy luận của User về CDC & Postgres giữ nhầm bản cũ `PROCESSING`

> **Kết luận Audit:** Suy luận của anh **CHÍNH XÁC 100% (`VERIFIED`)**. Khi 2 thao tác ghi diễn ra trong cùng một giây `05:17:04Z`, pipeline CDC hiện tại có khả năng giữ nhầm bản cũ `PROCESSING` trên PostgreSQL và **nếu không có lệnh `SUCCESS` lúc `10:59:26Z` thì bản ghi trong PostgreSQL sẽ sai mãi mãi**.

Dưới đây là 3 mắt xích code chính xác gây ra hiện tượng này:

1. **Mắt xích 1 — Độ phân giải giây của Debezium MongoDB `source.ts_ms` (`internal/handler/shadow/utils.go:72-105`):**
   - MongoDB Oplog lưu thời gian theo kiểu BSON `Timestamp(seconds, ord)` — trong đó `seconds` chỉ có độ phân giải **1 giây**, còn thứ tự các lệnh trong cùng 1 giây nằm ở `ord` (`1, 2, 3...`).
   - Debezium MongoDB Connector xuất `source.ts_ms = seconds * 1000` (luôn chẵn giây `...000`, cụ thể `05:17:04Z` ngày `29/10` là `1761715024000`) và đặt thứ tự trong giây ở `source.ord`.
   - Hàm `extractSourceTsMs` tại `internal/handler/shadow/utils.go:72-105` chỉ đọc `m["ts_ms"]` và bỏ qua hoàn toàn `m["ord"]`. Do đó cả 2 sự kiện CDC (`c`: `PROCESSING` và `u`: `PENDING`) đều mang **`_source_ts = 1761715024000`**.
2. **Mắt xích 2 — Cổng OCC tại `internal/service/shadow/schema_adapter.go:771-783` dùng dấu `<` ngặt (`_source_ts < EXCLUDED._source_ts`):**
   - Tại `schema_adapter.go:772-783` (`buildOCCWhereClause`), câu lệnh `ON CONFLICT DO UPDATE` trên bảng Shadow gắn điều kiện:
     ```sql
     WHERE (table."_hash" IS DISTINCT FROM EXCLUDED."_hash" OR table."_deleted" IS DISTINCT FROM EXCLUDED."_deleted")
       AND (
         table."_source_ts" IS NULL
         OR table."_source_ts" < EXCLUDED."_source_ts"
         OR (table."_source_ts" = EXCLUDED."_source_ts"
             AND table."_source" = 'snapshot:v2'
             AND EXCLUDED."_source" <> 'snapshot:v2')
       )
     ```
   - Khi sự kiện 1 (`PROCESSING`, `_source = 'debezium'`, `_source_ts = 1761715024000`) đã được INSERT vào bảng Shadow ở đợt flush trước, và sự kiện 2 (`PENDING`, `_source = 'debezium'`, `_source_ts = 1761715024000`) đến ở đợt flush tiếp theo:
     - `table."_source_ts" < EXCLUDED."_source_ts"` là `1761715024000 < 1761715024000` $\rightarrow$ **`FALSE`**!
     - `table."_source" = 'snapshot:v2'` là `'debezium' = 'snapshot:v2'` $\rightarrow$ **`FALSE`**!
   - PostgreSQL trả về `0 rows affected` (không báo lỗi), `BatchBuffer` đánh dấu `StatusPersisted` và commit Kafka offset. Sự kiện `PENDING` bị **bỏ qua vĩnh viễn tại tầng Shadow**, khiến cả Shadow lẫn Master PostgreSQL giữ `PROCESSING` từ `05:17:04Z` tới `10:59:26Z`!
3. **Mắt xích 3 — Tại sao trong 10 ngày chỉ có đúng 1 giao dịch bị lỗi này?**
   - Khi 2 sự kiện cùng giây nằm trong **cùng một đợt flush in-memory** của `BatchBuffer`, hàm `dedupByConflictKey` tại `internal/handler/shadow/batch_buffer_utils.go:100,127` lại dùng toán tử **`>=`** (`r.SourceTsMs >= records[prevIdx].SourceTsMs`), nên sự kiện đến sau (`PENDING`) ghi đè sự kiện trước (`PROCESSING`) ngay trong RAM trước khi xuống PostgreSQL!
   - Lỗi chỉ xảy ra khi 2 sự kiện cách nhau `23 ms` vô tình **rơi vắt ngang qua đúng ranh giới cắt batch (`FlushInterval` / `MaxBatchSize`)** của `BatchBuffer` (hoặc khi chunk đó rơi vào nhánh fallback chạy từng câu SQL tại `batch_buffer.go:674-750`), khiến chúng trở thành 2 câu lệnh SQL độc lập chạm vào `buildOCCWhereClause`. Xác suất 2 lệnh cách nhau `23 ms` cắt đúng ranh giới flush là rất nhỏ, nên trong 10 ngày chỉ xuất hiện đúng 1 giao dịch (`261029KHHDUH`).

---

## 4. Giải Pháp Đề Xuất Duy Nhất (Single Best Solution) & Code Demo

### 4.1. Giải pháp cho `F1` (`centralized-data-service`)
1. **Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/utils.go:72-105`:**
   - Trong `extractSourceTsMs`, sau khi trích xuất `ts_ms`, đọc thêm `m["ord"]` (MongoDB Oplog ordinal trong cùng giây). Nếu `tsMs > 0 && tsMs%1000 == 0` và `ord > 0`, cộng `min(ord, 999)` vào `tsMs`. Nhờ đó các sự kiện Oplog trong cùng 1 giây có `_source_ts` tăng dần nghiêm ngặt (`1761715024001 < 1761715024002`) mà không bao giờ vượt sang giây kế tiếp (`1761715025000`).
2. **Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go:771-783`:**
   - Cập nhật `buildOCCWhereClause` để cho phép cập nhật khi `_source_ts = EXCLUDED._source_ts` miễn là sự kiện đến không phải bản snapshot cũ (`EXCLUDED."_source" <> 'snapshot:v2'`). Vì mệnh đề `changeClause` (`_hash IS DISTINCT FROM EXCLUDED._hash OR _deleted IS DISTINCT FROM EXCLUDED._deleted` tại dòng `751-766`) luôn đứng trước `AND`, nếu Kafka replay lại đúng sự kiện đó (cùng `_hash` và `_deleted`) thì `changeClause` vẫn là `FALSE` (`0 rows affected` — đảm bảo idempotency), còn nếu có thay đổi thật trong cùng mốc `_source_ts` thì PostgreSQL sẽ cập nhật bản mới nhất (đồng nhất với `dedupByConflictKey` ở `batch_buffer_utils.go:100` và `transmuter.go:1698`).

#### Code Demo 1A — `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/utils.go:72-105`

**Before (nguyên văn từ `HEAD`):**
```go
func extractSourceTsMs(source interface{}) int64 {
	source = unwrapAvroUnion(source)
	m, ok := source.(map[string]interface{})
	if !ok {
		return 0
	}
	raw := m["ts_ms"]
	if mm, ok := raw.(map[string]interface{}); ok && len(mm) == 1 {
		for _, v := range mm {
			raw = v
			break
		}
	}
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}
```

**After (đề xuất):**
```go
func parseAnyInt64(raw interface{}) int64 {
	if mm, ok := raw.(map[string]interface{}); ok && len(mm) == 1 {
		for _, v := range mm {
			raw = v
			break
		}
	}
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

func extractSourceTsMs(source interface{}) int64 {
	source = unwrapAvroUnion(source)
	m, ok := source.(map[string]interface{})
	if !ok {
		return 0
	}
	tsMs := parseAnyInt64(m["ts_ms"])
	if tsMs > 0 && tsMs%1000 == 0 {
		if ord := parseAnyInt64(m["ord"]); ord > 0 {
			if ord > 999 {
				ord = 999
			}
			tsMs += ord
		}
	}
	return tsMs
}
```

#### Code Demo 1B — `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go:768-789`

**Before (nguyên văn từ `HEAD`):**
```go
	// 2. Cổng kiểm soát thứ tự thời gian (OCC Time Guard): Chống out-of-order stale write
	if hasSourceTs && hasPositiveTs {
		var timeClause string
		if _, hasSourceCol := schema.Columns["_source"]; hasSourceCol {
			timeClause = fmt.Sprintf(
				`(%s."_source_ts" IS NULL `+
					`OR %s."_source_ts" < EXCLUDED."_source_ts" `+
					`OR (%s."_source_ts" = EXCLUDED."_source_ts" `+
					`    AND %s."_source" = 'snapshot:v2' `+
					`    AND EXCLUDED."_source" <> 'snapshot:v2'))`,
				qualifiedTable, qualifiedTable, qualifiedTable, qualifiedTable,
			)
		} else {
			timeClause = fmt.Sprintf(`(%s."_source_ts" IS NULL OR %s."_source_ts" < EXCLUDED."_source_ts")`,
				qualifiedTable, qualifiedTable)
		}

		if changeClause != "" {
			return fmt.Sprintf(`WHERE %s AND %s`, changeClause, timeClause)
		}
		return fmt.Sprintf(`WHERE %s`, timeClause)
	}
```

**After (đề xuất):**
```go
	// 2. Cổng kiểm soát thứ tự thời gian (OCC Time Guard): Chống out-of-order stale write
	if hasSourceTs && hasPositiveTs {
		var timeClause string
		if _, hasSourceCol := schema.Columns["_source"]; hasSourceCol {
			timeClause = fmt.Sprintf(
				`(%s."_source_ts" IS NULL `+
					`OR %s."_source_ts" < EXCLUDED."_source_ts" `+
					`OR (%s."_source_ts" = EXCLUDED."_source_ts" `+
					`    AND EXCLUDED."_source" <> 'snapshot:v2'))`,
				qualifiedTable, qualifiedTable, qualifiedTable,
			)
		} else {
			timeClause = fmt.Sprintf(`(%s."_source_ts" IS NULL OR %s."_source_ts" <= EXCLUDED."_source_ts")`,
				qualifiedTable, qualifiedTable)
		}

		if changeClause != "" {
			return fmt.Sprintf(`WHERE %s AND %s`, changeClause, timeClause)
		}
		return fmt.Sprintf(`WHERE %s`, timeClause)
	}
```

---

### 4.2. Giải pháp cho `F2` (`core-trans-proxy-service/publish/cashin.publish.ts:34-44`)
- Tránh mutate trực tiếp lên biến tham chiếu dùng chung `transHisParam` và trả về Promise từ `createTransHis` / `updateTransHis` để đảm bảo thứ tự nhân quả `createTransHis` hoàn tất trước khi `updateTransHisStatus` được gọi.

#### Code Demo 2 — `/Users/trainguyen/Documents/work/core-trans-proxy-service/publish/cashin.publish.ts:34-44`

**Before (nguyên văn từ `HEAD`):**
```ts
	private static createTransHis(ctx: Context, transHisParam, status, error: any = null) {
		transHisParam.error = generateTransHisError(error);
		transHisParam.status = status;
		handlePromise(ctx.call(`${SVC_ENV.get().CORE_TRANS_PROXY_HISTORY_SERVICE}.createTransHis`, transHisParam, { meta: ctx.meta }))
	}

	private static updateTransHis(ctx: Context, transHisParam, status, error: any = null) {
		transHisParam.error = generateTransHisError(error);
		transHisParam.status = status;
		handlePromise(ctx.call(`${SVC_ENV.get().CORE_TRANS_PROXY_HISTORY_SERVICE}.updateTransHisStatus`, transHisParam, { meta: ctx.meta }))
	}
```

**After (đề xuất):**
```ts
	private static createTransHis(ctx: Context, transHisParam, status, error: any = null) {
		const payload = {
			...transHisParam,
			error: generateTransHisError(error),
			status,
		};
		return handlePromise(ctx.call(`${SVC_ENV.get().CORE_TRANS_PROXY_HISTORY_SERVICE}.createTransHis`, payload, { meta: ctx.meta }));
	}

	private static updateTransHis(ctx: Context, transHisParam, status, error: any = null) {
		const payload = {
			...transHisParam,
			error: generateTransHisError(error),
			status,
		};
		return handlePromise(ctx.call(`${SVC_ENV.get().CORE_TRANS_PROXY_HISTORY_SERVICE}.updateTransHisStatus`, payload, { meta: ctx.meta }));
	}
```
