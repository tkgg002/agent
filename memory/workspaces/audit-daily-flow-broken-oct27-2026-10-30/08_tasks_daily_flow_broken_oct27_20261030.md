# 08. TASKS — `daily_flow_broken_oct27_20261030`

## Task T1 [DONE]: Kiểm tra `git log` / `git diff` từ ngày 2026-10-27 đến 2026-10-30 trên `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`
- **Mô tả & Context:** Rà soát toàn bộ lịch sử commit, working tree diff và các thay đổi liên quan đến đợt fix time 24h đêm (`ResolveEventDateHCM`, `splitRecordsByEventDate`, `splitTaskByEventDate`, `TableDebouncer`, `batch_buffer`, `transmute_handler`, `activity_log_read_repo_gorm.go`, `kafka_consumer`, v.v.).
- **Logs/Test:** Đối chiếu `git log --since="2026-10-26"` và `git diff` trên các repo trong `/Users/trainguyen/Documents/work/data-hub`.
- **Definition of Done:** Chỉ rõ chính xác commit/diff (`file:line`) nào đã làm gãy Thống kê đối soát 3 trạm & DLQ theo ngày hoặc làm đứt pipeline từ ngày 27, 28/10.

## Task T2 [DONE]: Phản biện DG0 & Thiết kế Giải pháp Khắc phục (`09_tasks_solution_daily_flow_broken_oct27_20261030.md`)
- **Mô tả & Context:** Chạy `laya_gate.py run --stage dg0`, lập giải pháp tối ưu duy nhất kèm Code Demo trong `09_tasks_solution_daily_flow_broken_oct27_20261030.md`, kiểm định qua `laya_gate.py run --stage dg1` và trình User chờ `APPROVE`.
- **Logs/Test:** Biên bản `DG0` và `DG1` đạt `PASS`.
- **Definition of Done:** Giải thích rõ ràng từng nguyên nhân kèm `file:line`, hiển thị Code Demo trực tiếp trên chat và giữ chế độ Propose-Only chờ `APPROVE`.

## Task T3 [DONE]: Thực thi bản vá F1..F4 sau khi User `APPROVE` & Kiểm định S5/DG3
- **Mô tả & Context:**
  - `F1`: Giới hạn `ResolveEventDateHCM` theo `MidnightFlushGraceWindow = 15 * time.Minute` (`cdc_event_date.go`), không tách/lùi ngày cho Snapshot (`batch_buffer.go`, `batch_buffer_event_date.go`, `batch_buffer_fanout.go`, `transmute_event_date_split.go`), bổ sung SQL defense-in-depth trong `GetDailyFlow` (`activity_log_read_repo_gorm.go`).
  - `F2`: Bổ sung `unwrapAvroUnion` cho `event["source"]` và các trường con trong `kafka_consumer.go` & `utils.go`.
  - `F3`: Bổ sung cell `totals.masterDeleted` tại `index={14}` trong `ReconPipelineGrid.tsx`.
  - `F4`: Khởi động lại daemon `centralized-data-service` (`:8082` / `:9090`).
- **Logs/Test:** Biên lai `EV-20261030T135438-7154`, `EV-20261030T135449-0f20`, `EV-20261030T135416-5296`, `EV-20261030T135622-bd24`.
- **Definition of Done:** Toàn bộ test và cổng `DG3` (`06_laya_summary_dg3_20261030T135718.json`) đạt `PASS`.
