# 09. SOLUTION DESIGN — `daily_flow_broken_oct27_20261030`

- **Workspace:** `audit-daily-flow-broken-oct27-2026-10-30`
- **Tier:** `T2`
- **Suffix:** `daily_flow_broken_oct27_20261030`
- **Mode:** `standard` (Propose-Only — chờ `APPROVE`)

---

## 1. Bối cảnh & Đối chiếu Git Log / Diff (`2026-10-26` → `2026-10-30`)

Bảng trên màn hình của User là pipeline **`payment-bills`** (`shadow_traitestpbs.payment_bills` → `master_payment_bill_service.payment_bills`), khớp 100% với số liệu lịch sử:
- `2026-10-26`: Kafka `528` (`242 c`, `286 u`, `0 d`) | Shadow `528 Ops (303 dòng DB)` | Master `491 Ops (491 dòng DB)` (`245 Inserted`, `246 Updated`) | DLQ `350`, Lệch thiếu `5`, Đã heal `5` (`CÓ LỆCH`).
- `2026-10-27`: Kafka `163` (`75 c`, `88 u`, `0 d`) | Shadow `163 Ops (83 dòng DB)` | Master `163 Ops (163 dòng DB)` (`75 Inserted`, `88 Updated`) | DLQ `0`, Lệch thiếu `2`, Đã heal `0` (`CÓ LỆCH`).
- `2026-10-28`, `2026-10-29`, `2026-10-30`: Toàn bộ `0 Ops (0 dòng DB)`.

### Đối chiếu Git Commit & Working Tree từ ngày 26/10 đến 30/10
1. **`centralized-data-service` (`786db91` ngày 26/10 → `b8326bf` ngày 29/10 → Working Tree 30/10):**
   - Bản **"fix vụ time 24h đêm"** được code lúc `2026-10-27 14:24` (workspace `audit-daily-flow-vs-daily-count-2026-10-27`) và được commit gộp trong `b8326bf` (`feat:update master mongo`), sửa các file:
     - `internal/model/shadow/cdc_event.go:24` & `internal/model/shadow/cdc_event_date.go:9-29`
     - `internal/handler/shadow/batch_buffer.go:597-635`, `internal/handler/shadow/batch_buffer_event_date.go:17-69`, `internal/handler/shadow/batch_buffer_fanout.go:40-44`
     - `internal/handler/master/transmute_handler.go:497-655`, `internal/handler/master/transmute_event_date_split.go:17-131`
2. **`cdc-cms-service` (`7707aef` ngày 26/10 → `9b30e89` ngày 29/10 → Working Tree 30/10):**
   - Sửa `internal/infra/persistence/system/activity_log_read_repo_gorm.go:427` và `:447` trong `GetDailyFlow`.

---

## 2. Các Phát Hiện Gốc Rễ (`F1` → `F4` — 100% `VERIFIED` kèm `file:line`)

### `F1` (`VERIFIED` — Lỗi Thiết Kế Nghiêm Trọng trong bản "Fix Vụ Time 24h Đêm"): `ResolveEventDateHCM` & `GetDailyFlow` gom MỌI event theo `payload.source.ts_ms` vô điều kiện thay vì chỉ xử lý biên 24h đêm
- **Bằng chứng vật lý (`file:line`):**
  1. `centralized-data-service/internal/model/shadow/cdc_event_date.go:9-17`:
     ```go
     func ResolveEventDateHCM(tsMs int64, fallback time.Time) string {
         if tsMs > 0 {
             return time.UnixMilli(tsMs).In(locHCM).Format("2006-01-02")
         }
         ...
     }
     ```
  2. `centralized-data-service/internal/handler/shadow/batch_buffer.go:599-635` & `batch_buffer_event_date.go:17-55`: Tách mọi batch flush theo `ResolveEventDateHCM(r.SourceTsMs, time.Now())` và ghi cứng `"event_date": eventDate` vào `cdc_activity_log.details` **kể cả khi `bb.isSnapshotBuffer == true` (`op="r"`)**!
  3. `centralized-data-service/internal/handler/master/transmute_event_date_split.go:17-78` & `transmute_handler.go:533, 649`: Tách mọi NATS task theo `ev.EventDate` (`SourceTsMs`) và ghi cứng `"event_date": eventDate` vào `cdc_activity_log.details`.
  4. `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:406-428`:
     - Dòng `409`: `WHERE al.created_at >= NOW() - ((? + 1) * INTERVAL '1 day')` lọc theo `created_at` (ngày worker chạy).
     - Dòng `427`: `COALESCE(NULLIF(e.details->>'event_date', ''), TO_CHAR(e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD')) AS day_str` gom nhóm ưu tiên tuyệt đối theo `details->>'event_date'` (`source.ts_ms`).
     - Dòng `492-493`: `FROM dates d LEFT JOIN act_daily a ON a.day_str = d.day_str` (chỉ giữ 14 ngày gần nhất).
- **3 hậu quả trực tiếp làm gãy bảng Daily Flow từ ngày 27–28/10:**
  - **(a) Mọi hoạt động Catch-up / Replay / Xử lý Kafka lag trong ngày `28/10`, `29/10`, `30/10` bị dồn ngược về ngày cũ (`26/10`, `27/10`):** Khi worker dừng rồi bật lại để tiêu thụ tiếp các message tồn đọng trong Kafka có `source.ts_ms` thuộc ngày `26/10` hoặc `27/10`, toàn bộ activity log sinh ra vào ngày `28/10`, `29/10`, `30/10` đều bị đóng dấu `event_date = "2026-10-26"` hoặc `"2026-10-27"`. Kết quả là dòng `2026-10-28`, `2026-10-29`, `2026-10-30` hiển thị `0 Ops (0 dòng DB)` dù worker có xử lý!
  - **(b) Đánh rơi hoàn toàn (`0 Ops`) đối với Snapshot / Re-sync / Dữ liệu có `source.ts_ms` cũ hơn 14 ngày:** Khi chạy Snapshot (`isSnapshotBuffer == true`) hoặc đồng bộ các bản ghi có `source.ts_ms` cách đây >14 ngày, `details->>'event_date'` mang ngày cũ nằm ngoài CTE `dates` (14 ngày gần nhất). Khi `LEFT JOIN act_daily a ON a.day_str = d.day_str`, toàn bộ số liệu bị loại bỏ khỏi bảng!
  - **(c) Lệch trục thời gian giữa Trạm 2–3–4 (`event_date` theo `source.ts_ms`) với Trạm 5 DLQ & Đối soát (`started_at` / `checked_at`):** Khi `act.Fail(...)` ghi log lỗi (`status = 'error'`), `details` là `{}` (không có `event_date`), nên DLQ rơi vào ngày `started_at`, còn các bản ghi thành công của cùng đợt lại bị đẩy về ngày `source.ts_ms`!

### `F2` (`VERIFIED` — Lỗi Giải Mã Avro Union cho `event["source"]` trong `kafka_consumer.go:586, 621, 688`):
- **Bằng chứng vật lý (`file:line`):**
  - Tại `centralized-data-service/internal/handler/shadow/kafka_consumer.go:583-586`:
    ```go
    op := unwrapAvroUnion(event["op"])
    afterRaw := unwrapAvroUnion(event["after"])
    beforeRaw := unwrapAvroUnion(event["before"])
    sourceRaw := event["source"]
    ```
  - Trong khi `op`, `after`, `before` đều được gọi `unwrapAvroUnion(...)` (và `internal/sinkworker/utils.go:89` cũng gọi `srcRaw := unwrapAvroUnion(envelope["source"])`), thì `kafka_consumer.go:586` lại quên `unwrapAvroUnion(event["source"])` và các trường con `db`, `schema`, `table`, `collection` ở dòng `688-700`.
  - Khi Debezium Avro schema bọc `source` trong union `map[string]interface{}{"io.debezium.connector.*.Source": map[string]interface{}{...}}`, `extractSourceTsMs(sourceRaw)` (`internal/handler/shadow/utils.go:72-77`) trả về `0`, còn `sourceDB`, `sourceSchema`, `sourceTable` bị rỗng `""`.

### `F3` (`VERIFIED` — Lỗi Lệch Cột Footer Summary trên UI `ReconPipelineGrid.tsx:1455-1460`):
- **Bằng chứng vật lý (`file:line`):**
  - Tại `cdc-cms-web/src/components/ReconPipelineGrid.tsx:1455-1458`:
    - `index={12}`: `totals.masterInserted` (`Inserted`)
    - `index={13}`: `totals.masterUpdated` (`Updated`)
    - `index={14}`: `totals.masterSkipped` (Bị thiếu mất cell `totals.masterDeleted` cho cột `Delete(d)`!)
  - Làm toàn bộ dòng Tổng cộng (`Table.Summary.Row`) từ cột `Delete(d)` của Master sang khối `5. ĐỐI SOÁT & DLQ` và `ĐÁNH GIÁ` bị lệch trái 1 cột (`index` chỉ tới `19` thay vì `20`).

### `F4` (`VERIFIED` — Trạng Thái Runtime Daemon `centralized-data-service` Bị Kill / Chết Nền):
- **Bằng chứng vật lý:**
  1. Ngày `2026-10-27`: 163 Ops của `payment_bills` chạy vào buổi sáng trước `10:15 AM` (bằng chứng: dòng `2026-10-27` có `Master: 163 Ops (163 dòng DB)` với `Written = 75 + 88 = 163`, tức chạy bằng binary cũ trước bản sửa `14:24` chiều 27/10). Lúc `14:35` ngày 27/10 (`audit_report_daily_flow_audit.md:96`), `cds-worker` không chạy nền.
  2. Sáng nay `2026-10-30 11:15:53` (biên lai `EV-20261030T111650-aea5.json` trong `audit-duplicate-source-ts-upsert-2026-10-29`): Agent phiên trước đã chạy lệnh `kill` tiến trình `centralized-data-service` (`PID 30933`, được bật từ `29/10 17:22:57`) và khởi động lại bằng `go run` tạm trong subshell (`PID 31332`). Khi subshell đó đóng, tiến trình `centralized-data-service` (`:8082` / `:9090`) đã thoát theo.
  3. Kiểm tra hiện tại (`lsof -iTCP -sTCP:LISTEN -P -n`): Chỉ có `cdc-auth-service` (`:8081`), `cdc-cms-service` (`:8083`, `PID 49711` trên `ttys011`), và `cdc-cms-web` (`:5173`, `PID 46890` trên `ttys013`) đang chạy; **`centralized-data-service` (`:8082` / `:9090`) hiện đang dừng (`0` process)**.

---

## 3. Thiết Kế Giải Pháp Duy Nhất Tối Ưu (Ánh Xạ 1-1 `F1` → `F4`)

### 3.1. Giải quyết `F1`: Giới hạn `ResolveEventDateHCM` & `GetDailyFlow` đúng bài toán **"24h Midnight Flush Grace Window (`15 phút`)"**
- **Nguyên lý cốt lõi:**
  - Mục đích duy nhất của `event_date` là xử lý **độ trễ flush qua mốc 00:00 đêm** (sự kiện phát sinh sát nửa đêm ngày $D-1$, ví dụ `23:45:00..23:59:59.999` GMT+7, nhưng do `BatchBuffer` gom 5s hoặc NATS queue trễ vài giây nên flush sang Shadow/Master vào đầu ngày $D$, trong khoảng `00:00:00..00:14:59` GMT+7).
  - **Quy tắc `MidnightFlushGraceWindow = 15 * time.Minute`:**
    1. Nếu `isSnapshotBuffer == true` (`op="r"`): **Tuyệt đối không lùi ngày theo `SourceTsMs`** — luôn dùng ngày thực thi `fallback` (`started_at`) theo GMT+7.
    2. Trong `ResolveEventDateHCM(tsMs int64, fallback time.Time)` (`cdc_event_date.go`):
       - Quy đổi cả `fallback` (`fbHCM`) và `tsMs` (`evHCM`) sang `locHCM` (`GMT+7`).
       - Chỉ trả về ngày của `evHCM` (`D-1`) khi **đồng thời thỏa mãn 3 điều kiện biên 24h đêm**:
         - `(a)` `fbHCM` nằm trong cửa sổ `MidnightFlushGraceWindow` đầu ngày mới (`fbHCM.Sub(fbMidnight) <= 15*time.Minute`),
         - `(b)` `evHCM` thuộc đúng ngày hôm trước liền kề (`evDate == fbMidnight.AddDate(0, 0, -1).Format("2006-01-02")`),
         - `(c)` Khoảng cách `fbHCM.Sub(evHCM)` dương và `<= 15*time.Minute`.
       - Trong mọi trường hợp khác (worker chạy lại sau khi dừng vài giờ/vài ngày để catch-up backlog, replay Kafka, hoặc `tsMs` cũ từ nhiều ngày trước), trả về `fbHCM.Format("2006-01-02")` (ngày thực tế hệ thống xử lý)!
    3. **Defense-in-Depth trong SQL `GetDailyFlow` (`activity_log_read_repo_gorm.go:427`):**
       - Để sửa ngay lập tức cả các bản ghi `cdc_activity_log` đã lỡ bị ghi `details->>'event_date'` sai từ ngày 27–30/10 mà không cần sửa data DB (`Zero DB Cheat`), biểu thức `day_str` trong `act_daily` chỉ chấp nhận `e.details->>'event_date'` khi:
         - Không phải bản ghi snapshot (`NOT (COALESCE(e.details->>'buffer_type' = 'snapshot', false) OR COALESCE((e.details->>'is_snapshot')::boolean, false))`), VÀ
         - `e.details->>'event_date'` bằng ngày `started_at` (theo client TZ) **HOẶC** bằng đúng ngày hôm trước của `started_at` khi `started_at` nằm trong 15 phút đầu ngày (`00:00:00` – `00:14:59` client TZ).
       - Ngược lại, tự động fallback về `TO_CHAR(e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD')`.

### 3.2. Giải quyết `F2`: Bổ sung `unwrapAvroUnion` cho `event["source"]` trong `kafka_consumer.go` & `handler/shadow/utils.go`
- Tại `centralized-data-service/internal/handler/shadow/kafka_consumer.go:586` và `:688-700`:
  - Gọi `sourceRaw := unwrapAvroUnion(event["source"])`.
  - Khi đọc `sm["db"]`, `sm["schema"]`, `sm["table"]`, `sm["collection"]`, bọc qua `unwrapAvroUnion(...)` trước khi ép kiểu `.(string)`.
- Tại `centralized-data-service/internal/handler/shadow/utils.go:72`:
  - Bọc `source = unwrapAvroUnion(source)` ở đầu hàm `extractSourceTsMs`.

### 3.3. Giải quyết `F3`: Bổ sung cell `totals.masterDeleted` vào `Table.Summary.Row` trong `ReconPipelineGrid.tsx`
- Tại `cdc-cms-web/src/components/ReconPipelineGrid.tsx:1455-1476`:
  - Chèn `<Table.Summary.Cell index={14} align="right">` hiển thị `totals.masterDeleted` (màu `#ff4d4f`), và tăng `index` của các cell từ `totals.masterSkipped` đến `ĐÁNH GIÁ` lên `15..20` để khớp 100% với 18 cột dữ liệu + 2 cột đầu (`Ngày`, `Shadow Nạp`, `Master Sync`).

### 3.4. Giải quyết `F4`: Khởi động lại bền vững `centralized-data-service` (`:8082` / `:9090`) sau khi `APPROVE`
- Sau khi User `APPROVE` và áp dụng bản vá `F1–F3`, khởi chạy lại `centralized-data-service` và `cdc-cms-service` bằng binary biên dịch chuẩn để worker tiếp tục tiêu thụ Kafka và ghi log đúng ngày.

---

## 4. Code Demo Chi Tiết (Propose-Only)

### Demo 1: `centralized-data-service/internal/model/shadow/cdc_event_date.go`
```go
package shadow

import "time"

var locHCM = time.FixedZone("GMT+7", 7*3600)

// MidnightFlushGraceWindow là khoảng thời gian tối đa đầu ngày mới (00:00 -> 00:15 GMT+7)
// mà một batch/task sinh ra sát nửa đêm ngày hôm trước (D-1) bị trễ nhịp flush sang ngày D
// được phép quy thuộc về ngày D-1 để khớp với bảng Biến động dữ liệu theo ngày.
const MidnightFlushGraceWindow = 15 * time.Minute

// ResolveEventDateHCM trả về ngày YYYY-MM-DD theo múi giờ Asia/Ho_Chi_Minh (GMT+7).
// Chỉ lùi về ngày của tsMs (D-1) khi thời điểm xử lý (fallback) nằm trong cửa sổ
// MidnightFlushGraceWindow (15 phút đầu ngày D) và sự kiện gốc phát sinh vào cuối ngày D-1
// cách thời điểm flush không quá MidnightFlushGraceWindow. Mọi trường hợp catch-up, replay,
// hoặc backlog cũ hơn đều giữ nguyên ngày thực thi thực tế (fallback) để không làm gãy Daily Flow.
func ResolveEventDateHCM(tsMs int64, fallback time.Time) string {
	if fallback.IsZero() {
		fallback = time.Now()
	}
	fbHCM := fallback.In(locHCM)
	fbDate := fbHCM.Format("2006-01-02")
	if tsMs <= 0 {
		return fbDate
	}

	evHCM := time.UnixMilli(tsMs).In(locHCM)
	evDate := evHCM.Format("2006-01-02")
	if evDate == fbDate {
		return fbDate
	}

	fbMidnight := time.Date(fbHCM.Year(), fbHCM.Month(), fbHCM.Day(), 0, 0, 0, 0, locHCM)
	prevDate := fbMidnight.AddDate(0, 0, -1).Format("2006-01-02")
	if evDate == prevDate &&
		fbHCM.Sub(fbMidnight) <= MidnightFlushGraceWindow &&
		fbHCM.Sub(evHCM) >= 0 &&
		fbHCM.Sub(evHCM) <= MidnightFlushGraceWindow {
		return evDate
	}

	return fbDate
}
```

### Demo 2: `centralized-data-service/internal/handler/shadow/batch_buffer.go` & `batch_buffer_event_date.go` (không tách/lùi `event_date` cho Snapshot buffer)
```go
// batch_buffer_event_date.go
func (bb *BatchBuffer) flushTableGroupByEventDate(ctx context.Context, gk string, tableRecs []*shadow.UpsertRecord, now time.Time) (int, []RecordExecutionResult, error) {
	var written int
	var allResults []RecordExecutionResult
	var firstErr error
	subGroups := [][]*shadow.UpsertRecord{tableRecs}
	if !bb.isSnapshotBuffer {
		subGroups = splitRecordsByEventDate(tableRecs, now)
	}
	for _, recs := range subGroups {
```
```go
// batch_buffer.go
	defer func() {
		if act != nil && logEntry != nil {
			eventDate := shadow.ResolveEventDateHCM(0, logEntry.StartedAt)
			if !bb.isSnapshotBuffer {
				eventDate = shadow.ResolveBatchEventDateHCM(records, logEntry.StartedAt)
			}
```

### Demo 3: `centralized-data-service/internal/handler/shadow/kafka_consumer.go` & `utils.go` (bổ sung `unwrapAvroUnion` cho `source`)
```go
	op := unwrapAvroUnion(event["op"])
	afterRaw := unwrapAvroUnion(event["after"])
	beforeRaw := unwrapAvroUnion(event["before"])
	sourceRaw := unwrapAvroUnion(event["source"])
...
	var sourceDB, sourceSchema, sourceTable string
	if sm, ok := sourceRaw.(map[string]interface{}); ok {
		if d, ok := unwrapAvroUnion(sm["db"]).(string); ok {
			sourceDB = strings.TrimSpace(d)
		}
		if s, ok := unwrapAvroUnion(sm["schema"]).(string); ok {
			sourceSchema = strings.TrimSpace(s)
		}
		if t, ok := unwrapAvroUnion(sm["table"]).(string); ok {
			sourceTable = strings.TrimSpace(t)
		} else if c, ok := unwrapAvroUnion(sm["collection"]).(string); ok {
			sourceTable = strings.TrimSpace(c)
		}
	}
```

### Demo 4: `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go` (SQL Defense-in-Depth trong `GetDailyFlow`)
```sql
	act_daily AS (
		SELECT
			CASE
				WHEN NOT (COALESCE(e.details->>'buffer_type' = 'snapshot', false) OR COALESCE((e.details->>'is_snapshot')::boolean, false))
				     AND NULLIF(e.details->>'event_date', '') IS NOT NULL
				     AND (
				         e.details->>'event_date' = TO_CHAR(e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD')
				         OR (
				             e.details->>'event_date' = TO_CHAR((e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'))::date - INTERVAL '1 day', 'YYYY-MM-DD')
				             AND EXTRACT(HOUR FROM (e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'))) = 0
				             AND EXTRACT(MINUTE FROM (e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'))) < 15
				         )
				     )
				THEN e.details->>'event_date'
				ELSE TO_CHAR(e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD')
			END AS day_str,
```
*(cập nhật tương ứng mảng tham số `args` truyền `p.TzOffsetMinutes` cho các dấu `?` mới)*.

### Demo 5: `cdc-cms-web/src/components/ReconPipelineGrid.tsx` (bổ sung cell `totals.masterDeleted` ở `Table.Summary.Row`)
```tsx
                <Table.Summary.Cell index={12} align="right">{pipeline?.masterName ? <span style={{ color: '#52c41a' }}>{fmtNum(totals.masterInserted)}</span> : <Text type="secondary">—</Text>}</Table.Summary.Cell>
                <Table.Summary.Cell index={13} align="right">{pipeline?.masterName ? <span style={{ color: '#faad14' }}>{fmtNum(totals.masterUpdated)}</span> : <Text type="secondary">—</Text>}</Table.Summary.Cell>
                <Table.Summary.Cell index={14} align="right">{pipeline?.masterName ? <span style={{ color: '#ff4d4f' }}>{fmtNum(totals.masterDeleted)}</span> : <Text type="secondary">—</Text>}</Table.Summary.Cell>
                <Table.Summary.Cell index={15} align="right">{pipeline?.masterName ? <Text type="secondary">{fmtNum(totals.masterSkipped)}</Text> : <Text type="secondary">—</Text>}</Table.Summary.Cell>

                {/* Khối 5: RECON & DLQ */}
                <Table.Summary.Cell index={16} align="right">{fmtNum(totals.dlqTotal)}</Table.Summary.Cell>
                <Table.Summary.Cell index={17} align="right">
                  {!latestRecon ? <Text type="secondary">—</Text> : (
                    <span style={{ color: latestRecon.missing > 0 ? '#cf1322' : '#52c41a' }}>{fmtNum(latestRecon.missing)}</span>
                  )}
                </Table.Summary.Cell>
                <Table.Summary.Cell index={18} align="right">{!latestRecon ? <Text type="secondary">—</Text> : fmtNum(latestRecon.stale)}</Table.Summary.Cell>
                <Table.Summary.Cell index={19} align="right"><span style={{ color: '#52c41a' }}>{fmtNum(totals.healedTotal)}</span></Table.Summary.Cell>

                {/* Khối 6: ĐÁNH GIÁ (Cột ghim phải) */}
                <Table.Summary.Cell index={20} align="center">
```
