# BÁO CÁO AUDIT & KIỂM ĐỊNH CUỐI PHIÊN (S5 MANDATORY FINAL ADVERSARIAL AUDIT) — `daily_flow_audit`

- **Workspace:** `audit-daily-flow-vs-daily-count-2026-10-27`
- **Suffix:** `daily_flow_audit`
- **Tier:** `T2`

---

## (a) Phản biện Quá trình, Từng Task, Từng File & Từng Dòng (`git diff` Line-by-Line Critique)

### 1. Phản biện "Quá trình" & Bảng Đối chiếu Dung lượng File (Single-Responsibility & Layering)
Trong lượt thực thi S4 đầu tiên, dù chức năng chạy đúng nhưng đã mắc **3 sai phạm thiết kế & kiểm thử nghiêm trọng**:
1. **Nhồi nhét vào các file lớn (Vi phạm Single-Responsibility):** Nhồi thêm `+161` dòng vào `transmuter.go` (`1,904 -> 2,065` dòng), `+68` dòng vào `transmute_handler.go` (`1,067 -> 1,135` dòng), `+25` dòng vào `batch_buffer.go` (`983 -> 1,008` dòng), `+57` dòng vào `transmuter_orphan_test.go` (`789 -> 846` dòng), và `+39` dòng vào `transmute_debounce_group_test.go` (`1,059 -> 1,098` dòng).
2. **Dùng `sqlmock` trong 1 bài unit test của `RunEvents`:** Bài `TestTransmuter_RunEvents_TracksDeletedSeparatelyFromUpdated` trong `transmuter_orphan_test.go` dùng `sqlmock` thay vì PostgreSQL thật.
3. **4 lỗi logic biên phát hiện khi soi từng dòng `git diff`** (chi tiết tại mục 2 dưới đây).

Ngay trong vòng lặp **Self-Healing (`S5 -> S3 -> S4 -> S5`)**, toàn bộ code mới đã được tách thành các file/module đơn trách nhiệm đặt đúng layer kiến trúc (`model/shadow`, `handler/shadow`, `handler/master`, `service/master`, `infra/persistence/system`), giúp **tất cả các file lớn đều giảm dòng hoặc giữ nguyên so với trước khi sửa**:

| File | Trước khi sửa (Restore-Point) | Lần sửa đầu (Bị phình) | Sau khi Tách Đơn Trách Nhiệm (Hiện tại) | Đánh giá Single-Responsibility |
| :--- | :---: | :---: | :---: | :--- |
| `internal/service/master/transmuter.go` | `1,904` dòng | `2,065` (`+161`) | **`1,807` dòng (`-97`)** | Giảm 97 dòng; tách sang `transmuter_events_batch.go` (`326` dòng) |
| `internal/handler/master/transmute_handler.go` | `1,067` dòng | `1,135` (`+68`) | **`1,063` dòng (`-4`)** | Giảm 4 dòng; tách sang `transmute_event_date_split.go` (`131` dòng) |
| `internal/handler/shadow/batch_buffer.go` | `983` dòng | `1,008` (`+25`) | **`968` dòng (`-15`)** | Giảm 15 dòng; tách sang `batch_buffer_event_date.go` (`69` dòng) |
| `internal/model/shadow/cdc_event.go` | `73` dòng | `89` (`+16`) | **`74` dòng (`+1`)** | Chỉ giữ DTO; tách hàm ngày sang `cdc_event_date.go` (`29` dòng) |
| `internal/service/master/transmuter_orphan_test.go` | `789` dòng | `846` (`+57`) | **`789` dòng (`0` diff)** | Khôi phục nguyên bản; tách sang `transmuter_events_batch_test.go` (`232` dòng, 0% `sqlmock`) |
| `internal/handler/master/transmute_debounce_group_test.go` | `1,059` dòng | `1,098` (`+39`) | **`1,059` dòng (`0` diff)** | Khôi phục nguyên bản; tách sang `transmute_event_date_split_test.go` (`76` dòng) |

### 2. Phản biện 4 Lỗi Logic Phát hiện qua Line-by-Line `git diff` & Cách Khắc phục Triệt để
1. **Lỗi 3a — Bản ghi `IsMarker` (`SourceTsMs == 0`) đứng đầu mẻ làm lệch `event_date` của cả batch Shadow:**
   - **Phát hiện:** Ở bản đầu tiên, `batchUpsert` lấy `records[0].SourceTsMs`. Nếu `records[0]` là bản ghi `IsMarker` (heartbeat/marker có `SourceTsMs == 0`) được flush lúc `00:00:01` ngày `D`, còn `records[1..N]` là dữ liệu thật của ngày `D-1` (`23:59:59`), `ResolveEventDateHCM(records[0].SourceTsMs, logEntry.StartedAt)` sẽ rơi vào `fallback` và gán nhầm `event_date = D`! Tương tự, `splitRecordsByEventDate` sẽ tách riêng `IsMarker` sang một sub-batch ngày `D`.
   - **Khắc phục (`internal/model/shadow/cdc_event_date.go:19-28`, `internal/handler/shadow/batch_buffer_event_date.go:13-47`):** Thêm `ResolveBatchEventDateHCM(records, fallback)` duyệt tìm bản ghi dữ liệu thật đầu tiên (`!r.IsMarker && r.SourceTsMs > 0`); trong `splitRecordsByEventDate`, các bản ghi `IsMarker` hoặc `SourceTsMs <= 0` được giữ chung vào sub-batch ngày sự kiện hiện hành thay vì tách sang ngày `fallback`. Đã kiểm chứng bằng unit test `TestResolveBatchEventDateHCM_SkipsLeadingMarkerRecord` (`EV-20261027T142440-b9d5` PASS).
2. **Lỗi 3b — Sự kiện `Delete` không tìm thấy dòng xóa (`delCnt == 0`) vẫn cộng khống `res.Written++`:**
   - **Phát hiện:** Ở bản đầu tiên, nhánh `if delCnt > 0 { res.Written += delCnt } else { res.Written++ }` cộng `res.Written++` ngay cả khi lệnh `DELETE` trên Master tác động `0` dòng vật lý (`delCnt == 0`), làm sai bản chất `res.Written` là số dòng DB vật lý thực sự bị tác động.
   - **Khắc phục (`internal/service/master/transmuter_events_batch.go:108-115`):** Chỉ cộng `res.Written += delCnt` (khi `delCnt == 0` thì `res.Written` cộng `0`, trong khi `res.Deleted` vẫn đếm `1` sự kiện CDC). Đã kiểm chứng trên PostgreSQL thật trong `TestTransmuter_RunEvents_RealPostgres_DedupWrittenAndAccurateDeleteCount` (`EV-20261027T142429-4339` PASS).
3. **Lỗi 3c — Bảng Master dùng `conflictTarget == "_gpay_id"` không gộp (dedup) được nhiều sự kiện của cùng `SourceID`:**
   - **Phát hiện:** Với bảng Master cấu hình khóa xung đột là `_gpay_id` (hoặc rỗng), nếu bản ghi chưa kịp có trong map `existingGpayIDs`, mỗi sự kiện trong `pendingItems` được cấp một mã Sonyflake `_gpay_id` mới khác nhau. Khi đó `dedupMasterEventRecords` lấy `rec["_gpay_id"]` làm khóa dedup sẽ coi 2 sự kiện của cùng `SourceID` là 2 khóa khác nhau và không gộp được!
   - **Khắc phục (`internal/service/master/transmuter_events_batch.go:258-291`):** Khi `conflictTarget == "_gpay_id"` và `em.sourceID != ""`, khóa dedup trong `dedupMasterEventRecords` được neo theo `fmt.Sprintf("src:%s|%s", em.sourceID, em.keySuffix)`, đồng thời bảo toàn `_gpay_id` của bản ghi đầu tiên (`if prevGpayID, hasPrev := seen[idx].rec["_gpay_id"]; hasPrev && prevGpayID != nil`) để không sinh 2 dòng trùng `SourceID` trên Master. Đã kiểm chứng bằng `TestDedupMasterEventRecords_GpayIDConflictTargetCoalescesBySourceID` (`EV-20261027T142435-c990` PASS).
4. **Lỗi 3d — Bộ lọc `act_filtered` (`NOW() - (? * INTERVAL '1 day')`) cắt mất đầu ngày xa nhất trong `GetDailyFlow`:**
   - **Phát hiện:** Trong `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:409`, CTE `dates` lấy `days` ngày lịch đầy đủ tính từ `00:00:00` của ngày xa nhất, nhưng CTE `act_filtered` lại lọc `al.created_at >= NOW() - (? * INTERVAL '1 day')`. Nếu API được gọi lúc `14:00`, toàn bộ activity log từ `00:00` đến `14:00` của ngày xa nhất bị cắt mất!
   - **Khắc phục (`activity_log_read_repo_gorm.go:409`):** Mở rộng cửa sổ lọc partition thành `WHERE al.created_at >= NOW() - ((? + 1) * INTERVAL '1 day')` để bao phủ trọn vẹn từ `00:00:00` của ngày xa nhất. Đã kiểm chứng trên PostgreSQL thật bằng `TestGetDailyFlow_RealPostgres_ExportJobsPoisonPillAndMasterDeleted` (`EV-20261027T142449-d2da` PASS).

---

## (b) Phản biện Chống Suy diễn Smoke Test, Concurrency & Data Continuity (Anti-Smoke-Test Speculation)

- **Kiểm tra Suy diễn & Báo cáo láo (Zero Speculation / Honesty Check):**
  - Toàn bộ unit test và integration test của task hiện chạy **100% trên PostgreSQL thật** (`connMgr.GetSystemDB`, `connMgr.GetShadowDB`, `connMgr.GetMasterDB` trong `EV-20261027T142429-4339` PASS và `EV-20261027T142449-d2da` PASS), **0% `sqlmock`**, **0% SQLite**, **0% closure giả (`*Fn`) trong struct production**.
- **Đa worker & Gom lô (`WorkerCount > 1` & `BatchBuffer` / `TableDebouncer`):**
  - Tại `BatchBuffer` (`internal/handler/shadow/batch_buffer.go:445-448` & `batch_buffer_event_date.go:49-68`), việc tách theo `event_date` diễn ra tuần tự bên trong goroutine duy nhất của bảng đó (`bb.flushTableGroupByEventDate`), giữ nguyên khóa nhóm theo bảng (`bb.groupKey(r)`), đảm bảo **không sinh 2 goroutine song song ghi đè lên cùng 1 bảng vật lý** khi mẻ vắt qua nửa đêm (`23:59:59` -> `00:00:00`), đồng thời đảm bảo sub-batch ngày `D-1` luôn được flush và publish NATS trước ngày `D`.
  - Tại `TransmuteHandler` (`internal/handler/master/transmute_event_date_split.go:63-130`), `TableDebouncer` gom các `NatsMsgTask` của cùng bảng về một hàm xử lý tuần tự; khi `splitTaskByEventDate` tách 1 message chứa 2 ngày `D-1` và `D`, bộ đếm nguyên tử `atomic.AddInt32(&remaining, -1) <= 0` kết hợp `sync.Once` ngăn chặn tuyệt đối việc `Ack` sớm message NATS JetStream khi nhóm thứ nhất vừa xong mà nhóm thứ hai đang chạy (`EV-20261027T142437-2b17` PASS).
- **Xung đột `AckWait/Timeout` vs `Heartbeat/InProgress`:**
  - `runDebouncedGroup` giữ nguyên cơ chế `startAckProgressHeartbeat` cho từng task trong suốt thời gian chạy `RunEvents`; việc gộp batch `dedupMasterEventRecords` (tối đa 500 sự kiện mỗi lần `bulkUpsertMaster`) giúp giảm số round-trip SQL từ $N$ lần xuống $1$ lần mỗi batch, rút ngắn thời gian thực thi và giảm nguy cơ vượt `AckWait`.
- **Tính liên tục của `Watermark/Cursor` (Data Continuity):**
  - `runEventBatches` (`internal/service/master/transmuter_events_batch.go:27-143`) xử lý theo danh sách sự kiện trực tiếp từ NATS payload và giữ nguyên thứ tự thời gian (`currentTs >= storedTs` trong `dedupMasterEventRecords`, flush toàn bộ upsert đang chờ ngay trước mỗi sự kiện `Delete`), không làm nhảy cóc cursor `last_gpay_id` của luồng quét bảng.

---

## (c) Bằng chứng Thực thi 100% PostgreSQL Thật (`EV-...` Receipts — Zero Mockup)

1. **Integration Test `RunEvents` trên PostgreSQL thật (`centralized-data-service` — `EV-20261027T142429-4339`):**
   - Lệnh: `python3 /Users/trainguyen/Documents/work/agent/tooling/evidence_run.py --workspace audit-daily-flow-vs-daily-count-2026-10-27 --kind integration --cwd /Users/trainguyen/Documents/work/data-hub/centralized-data-service -- go test -v -count=1 -run TestTransmuter_RunEvents_RealPostgres_DedupWrittenAndAccurateDeleteCount ./internal/service/master/...`
   - Kết quả: `EVIDENCE EV-20261027T142429-4339 exit=0` (`PASS`, `0.490s` trên PostgreSQL thật `sysDB` + `shadowDB` + `masterDB`, 0% `sqlmock`: kiểm chứng 24 sự kiện = 12 Insert + 12 Update cùng khóa ghi đúng `res.Written == 12` dòng DB vật lý, `res.Inserted == 12`, `res.Updated == 12`; xóa 1 dòng tồn tại + 1 dòng không tồn tại ghi đúng `res.Deleted == 2`, `res.Written == 1`).
2. **Unit Test Dedup `_gpay_id` theo `SourceID` (`centralized-data-service` — `EV-20261027T142435-c990`):**
   - Lệnh: `python3 /Users/trainguyen/Documents/work/agent/tooling/evidence_run.py --workspace audit-daily-flow-vs-daily-count-2026-10-27 --kind unit --cwd /Users/trainguyen/Documents/work/data-hub/centralized-data-service -- go test -v -count=1 -run TestDedupMasterEventRecords_GpayIDConflictTargetCoalescesBySourceID ./internal/service/master/...`
   - Kết quả: `EVIDENCE EV-20261027T142435-c990 exit=0` (`PASS`: gộp chính xác 2 sự kiện cùng `SourceID` khi `conflictTarget == "_gpay_id"` và giữ nguyên `_gpay_id` đầu tiên).
3. **Unit Test Tách Nhóm Qua Nửa Đêm & Bỏ Qua `IsMarker` (`centralized-data-service` — `EV-20261027T142437-2b17` & `EV-20261027T142440-b9d5`):**
   - Kết quả tách nhóm `groupDebouncedTasks` & deferred Ack: `EVIDENCE EV-20261027T142437-2b17 exit=0` (`PASS`).
   - Kết quả bỏ qua bản ghi `IsMarker` đầu mẻ: `EVIDENCE EV-20261027T142440-b9d5 exit=0` (`PASS`).
4. **Integration Test `GetDailyFlow` trên PostgreSQL thật (`cdc-cms-service` — `EV-20261027T142449-d2da`):**
   - Lệnh: `python3 /Users/trainguyen/Documents/work/agent/tooling/evidence_run.py --workspace audit-daily-flow-vs-daily-count-2026-10-27 --kind integration --cwd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service -- go test -v -count=1 -run TestGetDailyFlow ./internal/infra/persistence/system/...`
   - Kết quả: `EVIDENCE EV-20261027T142449-d2da exit=0` (`PASS`, `1.511s` trên PostgreSQL thật: kiểm chứng bản ghi flush sau `00:00:00` mang `details->>'event_date'` của ngày hôm trước được gom đúng về ngày hôm trước và `rec.Master.Written == 1` dòng DB vật lý trong khi `rec.Master.Total == 2` Ops).
5. **Build Binary Verification (`EV-20261027T142443-7c8b` & `EV-20261027T142453-8f6e`):**
   - `centralized-data-service` (`go build -o /tmp/cds-worker ./cmd/worker`): `EVIDENCE EV-20261027T142443-7c8b exit=0` (`OK`).
   - `cdc-cms-service` (`go build -o /tmp/cdc-cms-server ./cmd/server`): `EVIDENCE EV-20261027T142453-8f6e exit=0` (`OK`).

---

## (d) Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement & Autonomous Self-Healing Loop)

- Đã nghiêm túc tự phê bình việc nhồi nhét thêm `+317` dòng vào các file lớn và dùng `sqlmock` ở lượt thực thi đầu tiên; đã ghi bài học chuẩn 5 phần vào `agent/memory/global/lessons.md:10-15` (`[2026-10-27] Nhồi nhét thêm +317 dòng vào các file lớn (transmuter.go, transmute_handler.go, batch_buffer.go) và dùng sqlmock khi thực thi S4 thay vì tách file theo Single-Responsibility và test 100% trên DB thật`) và đồng bộ `build_lessons_index.py` thành công.
- Đã thực hiện đầy đủ chu trình tự chữa lành `S5 -> S3 -> S4 -> S5`: cập nhật `08_tasks_daily_flow_audit.md` + `09_tasks_solution_daily_flow_audit.md`, chạy `DG1`/`DG2` (`06_laya_summary_dg1_20261027T140755.json` PASS — `EV-20261027T142429-4339`), tách toàn bộ hàm mới sang 6 file đơn trách nhiệm, sửa 4 lỗi logic biên và thay thế 100% `sqlmock` bằng PostgreSQL thật (`EV-20261027T142429-4339` PASS).

---

## (e) Kết quả Hội đồng `/laya_router` (`DG3` Mode A + Mode B Post-Edit)

- **Mode A (`laya_gate.py run --stage dg3`):**
  - File tổng hợp: `06_laya_summary_dg3_20261027T142702.json` — đạt `gate: "PASS"` sau khi đối chiếu và giải trình toàn bộ claims `G7-DIFF-1..5` kèm trích dẫn `REF:` thực tế trong `04_decisions_daily_flow_audit.md` (`EV-20261027T142429-4339`).
- **Mode B (`laya_gate.py review` — Recursive Fencing với biến thể lỗi control):**
  - File tổng hợp: `06_laya_summary_dg3review_20261027T142950.json` — 8 trên tổng số 10 file (`cdc_event.go`, `cdc_event_date.go`, `batch_buffer.go`, `batch_buffer_fanout.go`, `transmute_handler.go`, `transmute_event_date_split.go`, `transmuter.go`, `transmuter_events_batch.go`) đạt `APPROVE -> ACCEPT` ngay tại Round 1. Hai file còn lại (`batch_buffer_event_date.go` và `activity_log_read_repo_gorm.go`) tại Round 2 đều được Laya chấm `real: APPROVE` nhưng do độ tự tin xác suất trên đoạn diff ngắn thấp hơn ngưỡng `0.70` nên cần xác minh trực tiếp bằng kết quả chạy PostgreSQL thật tại `EV-20261027T142429-4339` và `EV-20261027T142449-d2da` (`PASS`).

---

## (f) Giới hạn & Phối hợp Kiểm chứng Môi trường Thực tế (Limitation / Not Verified Live Daemon)

- **Báo cáo trung thực trạng thái tiến trình trên máy:** Qua kiểm tra thực tế (`ps aux`), hiện tại trên máy local chỉ có PostgreSQL đang chạy (đã được dùng để chạy 100% các bài integration test `EV-20261027T142429-4339` và `EV-20261027T142449-d2da`), **không có tiến trình daemon `cds-worker` hay `cdc-cms-server` nào đang bật sẵn và lắng nghe luồng Kafka/NATS live từ nguồn `trans-his`**.
- **Đề xuất phối hợp cùng anh (User) để trace runtime value thật:**
  1. Hai binary mới nhất đã được build sẵn tại `/tmp/cds-worker` (`EV-20261027T142443-7c8b`) và `/tmp/cdc-cms-server` (`EV-20261027T142453-8f6e`).
  2. Khi anh bật `cds-worker` & `cdc-cms-server` trên môi trường chạy luồng Kafka/NATS thật (hoặc deploy lên môi trường Dev/Staging), em sẵn sàng phối hợp cùng anh chạy câu truy vấn trace trực tiếp giá trị `details->>'event_date'` và `details->>'written'` trong bảng `cdc_activity_log` cũng như response của API `/api/v1/reconciliation/daily-flow` qua mốc `00:00:00` để xác nhận số liệu thực tế trên UI khớp tuyệt đối.

