# 09_solution — InvestigateTransformSchedulerStuck

## TL;DR
"Worker không chạy" thực sự có **2 bug ĐỘC LẬP** chồng nhau, không phải
1. Cả 2 đều đã có bằng chứng từ DB; chưa fix code (user yêu cầu chỉ
document).

| # | Tên | Triệu chứng user thấy | Lớp ảnh hưởng |
|---|---|---|---|
| A | Telemetry split (fire-and-forget) | "OK rows=1" trong list dù thực sự fail | Hiển thị / quan sát — *không chặn data* |
| B | TZ bug `last_run_at` | Schedule fire 1 lần rồi đứng 7 giờ | Lập lịch — *chặn data* |

## Bug A — Telemetry split

### Cơ chế
File: `centralized-data-service/internal/server/worker_server.go:695-716`
```go
func (s *WorkerServer) runTransformCycle(now time.Time, targetTable string) {
    entries, _ := s.registryRepo.GetAllActive(...)
    dispatched := 0
    for _, entry := range entries {
        if targetTable != "" && entry.TargetTable != targetTable { continue }
        s.nats.Conn.Publish("cdc.cmd.batch-transform", []byte(entry.TargetTable))
        dispatched++
    }
    s.activityLogger.Quick("transform", logTarget, "scheduler", "success",
        int64(dispatched), map[string]interface{}{"tables": dispatched}, "")
}
```
- `rows_affected = dispatched` = **số lệnh đã `nats.Publish`**, KHÔNG
  phải số row data được transform.
- Status `success` chỉ phản ánh kết quả `nats.Publish` (in-memory
  enqueue → NATS server). **KHÔNG** chờ worker handler
  `HandleBatchTransform` execute SQL.

### Kết quả: 2 entry tách biệt mỗi lần fire
| field | id=27 (scheduler) | id=28 (worker handler) |
|---|---|---|
| operation | `transform` | `cmd-batch-transform` |
| triggered_by | `scheduler` | `nats-command` |
| status | `success` | `error` |
| rows_affected | `1` (1 table publish) | `0` |
| error_message | (none) | `no active mapping rules ...` |
| started_at | 02:38:22.747 UTC | 02:38:22.803 UTC (cách 56ms) |
| details | `{"tables":1}` | `{"error":"...", "command":"batch-transform"}` |

### Tại sao user thấy "OK"
UI/list filter theo `operation='transform'` → chỉ thấy id=27 (success).
Entry id=28 (operation='cmd-batch-transform') không match filter →
ẩn trong list mà user xem. Phải mở filter khác mới thấy lỗi thực sự.

### Match với lessons cũ
- lesson 844 (line 844): "NATS fire-and-forget pattern cho phép parallel
  refactor mà không cần sync."
- lesson 1297 (line 1297): "publisher A 'fire-and-forget' rồi mong
  handler tự về close (handler không có context schedule_id)" → **đây
  chính xác là pattern `runTransformCycle` đang mắc**. Lesson đã
  cảnh báo.

## Bug B — Timezone bug `cdc_worker_schedule.last_run_at`

### Cơ chế
1. Schema: `last_run_at | timestamp without time zone` (no TZ).
2. Go ghi: GORM Updates với `time.Now()` (Time có Location = Local =
   Asia/Ho_Chi_Minh +07). Driver pq strip TZ → PG store wall clock
   của local. → `09:38:22` được lưu khi instant thực = `02:38:22 UTC`.
3. Go đọc: GORM scan column TIMESTAMP-no-TZ → trả `time.Time` với
   Location = UTC, value = wall clock đọc nguyên xi → `09:38:22 UTC`
   (sai 7 giờ về tương lai).
4. Gating (line 624-633):
   ```go
   intervalDur := time.Duration(sched.IntervalMinutes) * time.Minute
   if sched.LastRunAt != nil && now.Sub(*sched.LastRunAt) < intervalDur {
       continue // Not due yet
   }
   ```
   `now` = `time.Now()` = instant `06:12 UTC`. `lastRunAt` parsed =
   `09:38 UTC`. `now.Sub(lastRunAt) = -3h26m`. Negative duration
   `< intervalDur (1min)` → continue → SKIP forever, hoặc until UTC
   wall clock catch up `lastRunAt + interval` ≈ 7 giờ kế.

### Reproduce
```sql
-- pre:  schedule id=6 last_run_at='2026-05-13 09:38:22.743849' (local-as-UTC)
--       wall now UTC = 06:12, không fire suốt 3.5 giờ
UPDATE cdc_system.cdc_worker_schedule SET last_run_at = NULL WHERE id = 6;
-- đợi 65s
-- post: cdc_activity_log id=51 (scheduler) + id=52 (worker success rows=130) ✓
--       schedule id=6 last_run_at='2026-05-13 13:13:09.791229' (lại local)
--       run_count tăng 1 → 2
```

### Pattern lặp vô hạn
Sau mỗi lần fire, lưu `last_run_at = local-wall-clock`. Đọc lại nhìn
như `+7h trong tương lai UTC`. Schedule đông cứng 7 giờ. Cứ ~7 giờ
fire 1 lần thay vì interval định nghĩa (1 min hoặc 5 min).

### Trao đổi với scheduler khác (transmute)
Worker log:
```
"transmute scheduler started (60s poll, cron + FOR UPDATE SKIP LOCKED + fencing)"
```
→ Có scheduler thứ 2 dùng cơ chế khác (FOR UPDATE SKIP LOCKED + fencing
token). Nếu scheduler `cdc_worker_schedule` cũng bị TZ thì transmute có
thể đã được implement đúng. (Out of scope, chỉ note.)

## Suggest fix (chưa thực thi — user quyết)

### Fix Bug A — Reflect worker outcome lên scheduler entry
Option 1 (đơn giản, không thay đổi message bus):
- Đổi `runTransformCycle` không ghi `success rows=X` ngay sau publish.
  Thay bằng `accepted` status, `rows = số table queued`. Worker handler
  giữ nguyên ghi entry riêng. UI sẽ thấy `accepted` thay vì `success`
  → bớt nhầm lẫn.

Option 2 (đầy đủ, cần thêm event):
- Worker publish `cdc.evt.batch-transform.completed` sau khi handler
  done. Một consumer riêng update entry `transform` của scheduler
  thành `success/error` theo evt. Theo lesson 1297 (3-actor pattern
  publisher → handler → monitor).

### Fix Bug B — Eliminate TZ ambiguity
Option 1 (recommend, ít rủi ro):
- Migration đổi cột:
  ```sql
  ALTER TABLE cdc_system.cdc_worker_schedule
    ALTER COLUMN last_run_at TYPE timestamptz USING last_run_at AT TIME ZONE 'UTC',
    ALTER COLUMN next_run_at TYPE timestamptz USING next_run_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at  TYPE timestamptz USING created_at  AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at  TYPE timestamptz USING updated_at  AT TIME ZONE 'UTC';
  ```
  Lưu ý: nếu data hiện tại đã là local-wall-clock thì cần
  `AT TIME ZONE 'Asia/Ho_Chi_Minh'` thay vì `'UTC'` để dịch về đúng
  instant. Cần data audit trước khi migrate.

Option 2 (chỉ chạm code, không migrate):
- Trong `runScheduleTick` ghi lại bằng `time.Now().UTC()`. Khi đọc
  cũng wrap qua `.UTC()`. Tiết kiệm migration nhưng dễ regress.

### Audit liên quan
Tìm các bảng `cdc_system.*` có cột `timestamp` (no TZ) tương tự:
```sql
SELECT table_name, column_name FROM information_schema.columns
WHERE table_schema='cdc_system' AND data_type='timestamp without time zone'
ORDER BY table_name, column_name;
```

## Skills / Tools đã dùng
- **Read**: `worker_server.go` (runTransformCycle, schedule poller),
  `lessons.md` (lesson 844, 1297).
- **Bash**: `docker exec psql` (activity_log query, schedule query,
  experiment UPDATE), `curl /health`, `lsof`, `ps`, `date -u`.
- **Write**: workspace 4 file (00_context, 05_progress APPEND-only,
  09_solution, report).
- **TaskCreate / TaskUpdate**: track 1 task điều tra.
- **Governance**:
  - §3 Verify Before Done — experiment reproduce Bug B + activity_log
    id=51/52 chứng minh.
  - §6 Minimal scope — KHÔNG sửa code, chỉ document + experiment
    reversible (set lastRunAt=NULL).
  - §7 Workspace prefix bắt buộc.
  - §11 Memory APPEND only.
  - §12 Brain code prohibition không vi phạm (Muscle viết code; lần
    này không viết).
  - §13 Lesson abstract → 2 Global Pattern (xem report).
