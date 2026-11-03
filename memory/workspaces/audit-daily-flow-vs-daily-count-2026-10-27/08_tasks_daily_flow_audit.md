# 08. TASKS — `daily_flow_audit`

## [x] Task 1: Neo mốc ngày sự kiện gốc (`event_date` từ `SourceTsMs` / Shadow Ingest) xuyên suốt `BatchBuffer` -> `TransmuteHandler` (`centralized-data-service`)
- **Context (`file:line`):**
  - `centralized-data-service/internal/model/shadow/cdc_event.go:70-85` (`TransmuteEventItem`)
  - `centralized-data-service/internal/model/shadow/cdc_event_date.go:1-45` (`ResolveEventDateHCM`, `ResolveBatchEventDateHCM` — tách file đơn trách nhiệm)
  - `centralized-data-service/internal/handler/shadow/batch_buffer.go:420-470, 640-660` (`executeFlushGroupsWithResults`, `batchUpsert`)
  - `centralized-data-service/internal/handler/shadow/batch_buffer_event_date.go:1-55` (`splitRecordsByEventDate` — tách file đơn trách nhiệm)
  - `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:32-55` (`publishTransmuteTrigger`)
  - `centralized-data-service/internal/handler/master/transmute_handler.go:508-720` (`groupDebouncedTasks`, `runDebouncedGroup`)
  - `centralized-data-service/internal/handler/master/transmute_event_date_split.go:1-95` (`taskEventDate`, `splitTaskByEventDate` — tách file đơn trách nhiệm)
  - `centralized-data-service/internal/handler/master/transmute_event_date_split_test.go:1-80` (Test đơn trách nhiệm cho `groupDebouncedTasks` & `splitTaskByEventDate`)
  - `centralized-data-service/internal/service/master/transmuter.go:115-120, 704-775` (`TransmuteResult`, `RunEvents`)
  - `centralized-data-service/internal/service/master/transmuter_events_batch.go:1-260` (`runEventBatches`, `dedupMasterEventRecords`, `lookupShadowGpayIDs` — tách file đơn trách nhiệm)
  - `centralized-data-service/internal/service/master/transmuter_events_batch_test.go:1-210` (Test tích hợp 100% PostgreSQL thật cho `RunEvents`, 0% `sqlmock` / 0% SQLite)
- **Mô tả:**
  - Xác định `event_date` (YYYY-MM-DD theo múi giờ `Asia/Ho_Chi_Minh`) cho từng sự kiện dựa trên `SourceTsMs` (hoặc fallback về thời điểm `BatchBuffer` nhận mẻ nếu `SourceTsMs <= 0`, bỏ qua bản ghi `IsMarker` khi lấy `event_date` của batch).
  - Tại `BatchBuffer`, tách/ghi nhận `event_date` vào `cdc_activity_log.details["event_date"]` và truyền `EventDate` trong `TransmuteEventItem` qua NATS `cdc.cmd.transmute-shadow`.
  - Tại `TransmuteHandler` (`TableDebouncer`), tách nhóm `groupDebouncedTasks` theo `(master_binding_id, triggered_by, event_date)` để các sự kiện của ngày `D-1` (`23:59:59`) không bị trộn vào mẻ ngày `D` (`00:00:00`), đồng thời ghi `details["event_date"]` và `details["written"]` (số dòng DB thực tế sau dedup trong `RunEvents`) vào `cdc_activity_log`.
- **Logs/Test:** `EV-20261027T142429-4339`, `EV-20261027T142435-c990`, `EV-20261027T142437-2b17`, `EV-20261027T142440-b9d5`, `EV-20261027T142443-7c8b`
- **Definition of Done:** [x] Các sự kiện sinh ra ở `23:59:59` ngày `D-1` dù được Kafka/BatchBuffer hay `TableDebouncer` xử lý lúc `00:00:01` ngày `D` vẫn mang cùng `event_date = D-1` ở cả Kafka, Shadow, Master và khớp với Báo cáo Đối soát `00:05`.

## [x] Task 2: Gom nhóm Daily Flow theo `event_date` và đọc `details->>'written'` cho Master (`cdc-cms-service`)
- **Context (`file:line`):**
  - `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:366-540` (`GetDailyFlow`)
  - `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm_test.go:1-235`
- **Mô tả:**
  - Trong CTE `act_filtered`, mở rộng cửa sổ lọc partition thành `al.created_at >= NOW() - ((? + 1) * INTERVAL '1 day')` để không cắt cụt các dòng log đầu ngày của ngày xa nhất trong `dates` hoặc dòng log qua biên `00:00:00`.
  - Trong CTE `act_daily` của `GetDailyFlow`, thay biểu thức cắt ngày `TO_CHAR(e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD')` bằng `COALESCE(NULLIF(e.details->>'event_date', ''), TO_CHAR(e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD'))`.
  - Bổ sung `master_written` đọc từ `COALESCE((e.details->>'written')::bigint, NULLIF(e.rows_affected, 0), ...)` để hiển thị đúng số dòng DB vật lý được ghi tại Master tương ứng với Shadow.
- **Logs/Test:** `EV-20261027T142449-d2da`, `EV-20261027T142453-8f6e`
- **Definition of Done:** [x] Cột Kafka, Shadow và Master trên Daily Flow khớp số lượng `Ops` (`c`, `u`, `d`) 100% theo ngày sự kiện, không còn lệch `+1 c / +1 u` qua biên nửa đêm `00:00:00`.

## [x] Task 3: Self-Healing Refactor (Tách module đúng Single-Responsibility, sửa 4 lỗi Line-by-Line Audit, 100% Real-PG Test)
- **Context (`file:line`):**
  - `agent/tooling/build_lessons_index.py:240-255`
  - `centralized-data-service/internal/model/shadow/cdc_event.go` & `cdc_event_date.go`
  - `centralized-data-service/internal/handler/shadow/batch_buffer.go` & `batch_buffer_event_date.go`
  - `centralized-data-service/internal/handler/master/transmute_handler.go`, `transmute_event_date_split.go`, `transmute_debounce_group_test.go`, `transmute_event_date_split_test.go`
  - `centralized-data-service/internal/service/master/transmuter.go`, `transmuter_events_batch.go`, `transmuter_orphan_test.go`, `transmuter_events_batch_test.go`
  - `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:409`
- **Mô tả:**
  - Tách toàn bộ hàm mới ra các file đơn trách nhiệm (`cdc_event_date.go`, `batch_buffer_event_date.go`, `transmute_event_date_split.go`, `transmuter_events_batch.go`, `transmute_event_date_split_test.go`, `transmuter_events_batch_test.go`) để không nhồi nhét vào các file lớn; khôi phục `transmuter_orphan_test.go` và `transmute_debounce_group_test.go` về nguyên bản `HEAD` và giảm kích thước `transmuter.go` nhỏ hơn trước khi sửa (`1,807 < 1,904` dòng).
  - Sửa 4 lỗi logic phát hiện khi Line-by-Line Audit:
    1. `ResolveBatchEventDateHCM` & `splitRecordsByEventDate` bỏ qua bản ghi `IsMarker` / `SourceTsMs <= 0` khi xác định `event_date` của sub-batch.
    2. `runEventBatches` khi xử lý sự kiện xóa (`IsDelete`/`Op == "d"`): chỉ cộng `res.Written += delCnt` (tuyệt đối không cộng `res.Written++` khi `delCnt == 0`).
    3. `dedupMasterEventRecords`: khi `conflictTarget == "_gpay_id"`, dedup theo `sourceID + "|" + keySuffix` (thay vì `_gpay_id` Sonyflake sinh mới mỗi event) để gộp chính xác nhiều sự kiện của cùng `SourceID`.
    4. `activity_log_read_repo_gorm.go:409`: dùng `((? + 1) * INTERVAL '1 day')` trong `act_filtered` để không cắt mất đầu ngày xa nhất.
  - Thay thế hoàn toàn `sqlmock` trong test của `RunEvents` bằng PostgreSQL thật 100% (`sysDB` + `shadowDB` + `masterDB` thật) trong `transmuter_events_batch_test.go`.
- **Logs/Test:** `EV-20261027T142429-4339`, `EV-20261027T142435-c990`, `EV-20261027T142437-2b17`, `EV-20261027T142440-b9d5`, `EV-20261027T142443-7c8b`, `EV-20261027T142449-d2da`, `EV-20261027T142453-8f6e`
- **Definition of Done:** [x] Không có file lớn nào bị phình dòng do nhồi nhét hàm mới; 100% test chạy trên PostgreSQL thật (0% `sqlmock`); đạt `PASS` cổng `DG1` và `DG3` (`Mode A` + `Mode B` `review`).


