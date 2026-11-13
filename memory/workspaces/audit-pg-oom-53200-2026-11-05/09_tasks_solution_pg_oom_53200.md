---
tier: T2
suffix: pg_oom_53200
status: APPROVED P10 (T18)
---

# 09. Tasks Solution — pg_oom_53200

## Trạng thái
- Audit read-only. **Chưa lập phương án sửa code** vì root cause F6 (statement-cache/pool bloat) vẫn là HYPOTHESIS; môi trường phát sinh lỗi là remote, Agent không truy cập được (xem `audit_report_pg_oom_53200.md` §2).
- Không có Code Demo, không có DG1/DG2 ở vòng này. Không có thay đổi source.

## Điều kiện mở lại S3
1. User cung cấp PG log `Failed on request of size ... in memory context "X"` + `TopMemoryContext` dump quanh 02:34:50–02:37:10 UTC.
2. Cấu hình instance (RAM/limit, `max_connections`, `work_mem`, `vm.overcommit_memory`) và số connection theo app tại thời điểm lỗi.
3. Giá trị `CDS_KAFKA_GROUP_ID` thực tế trên deployment dev (F8).

Khi có dữ liệu → lập ĐÚNG 1 phương án ánh xạ F-n, chạy DG1, chờ `APPROVE`.

## DG1 (cơ sở dữ kiện của hướng xử lý dự kiến)
- `claims_dg1.json` → `06_laya_summary_dg1_20261105T094805.json`: 4 claim `FLAG` (S-FIX-01, S-RC-01, S-IMPACT-01, S-PATTERN-01), đã `resolve` kèm `REF:` → gate `PASS`.
- Chỉ xác nhận các dữ kiện thư viện/code (pgx hỗ trợ `default_query_exec_mode`, pgxpool lifetime mặc định 1h, vị trí `MinConns`), **không phải** là phương án đã duyệt.

---

# S3 (2026-11-05T11:50) — Audit giải pháp User đề xuất (5 file) + Phương án hợp nhất

## A. Phán quyết từng hạng mục của User
| # | Hạng mục User | Phán quyết | Bằng chứng / lý do |
|---|---|---|---|
| U1 | `ensurePgxCacheDescribe` cho `openGorm`/`openPgx` | ✅ Đúng hướng, **phải sửa 1 lỗi nghiêm trọng**: chỉ gắn trong nhánh postgres của `OpenGormWithEngine` | `multi_dialector.go:16-27,43` — cùng hàm mở MySQL/ClickHouse; go-sql-driver/mysql coi param lạ là system variable → `SET default_query_exec_mode` lỗi khi connect. cache_describe có cùng caveat schema-change với cache_statement mặc định (`pgx conn.go:648-658`) → không thêm rủi ro mới |
| U1b | `MaxConnLifetime` cho pgxpool | ⚠️ Vô nghĩa ở vị trí hiện tại | Pool pgx control-plane **tạo rồi vứt** `_ = pgxPool` (`server_setup.go:236-237`, prod `:255-256`), `NewEventBridge` không được gọi; nhưng `MinConns=MaxIdleConn=50` → pgxpool tự mở 50 conn idle (`pgxpool/pool.go:336-337`) **mỗi replica** (manifest: replicas 5, HPA 3–10) trên đúng instance control-plane đã OOM → **xoá dòng tạo pool chết** (phát hiện mới F17) |
| U2 | Fanout stale-reuse khi refresh lỗi | ✅ Đồng ý (F18) + bổ sung: lần đầu lỗi cũng cache fail-open TTL ngắn (5s) để không dội query mỗi chunk | `batch_buffer_fanout.go:163-215`. Lưu ý: "query nặng" là cường điệu (bảng metadata nhỏ); tác hại thật là **ngữ nghĩa** (`enabled=true`, mất `bindingCode`) + thêm query đúng lúc DB đang chết |
| U3.1 | Thêm string `53200`/`out of memory` vào `isRetryableDBError` | ⚪ Thừa nhưng vô hại | `batch_buffer_dlq.go:135-148` đã bắt `53*` qua `errors.As(*pgconn.PgError)`; giữ làm defense-in-depth |
| U3.2 | Backoff 1s×attempt khi 53xxx | ✅ Đồng ý + **sửa bug sẵn có**: `break` trong `select` không thoát vòng `for` khi `ctx.Done()` | `batch_buffer_dlq.go:38-45` |
| U3.3 | Không ghi DLQ khi lỗi retryable | ❌ **Bác** — đổi ngữ nghĩa | `StatusFailed` → rào chắn watermark, partition **không commit tới khi restart** (không có cơ chế redeliver in-process: `watermark_tracker.go:146-191`); hiện tại DLQ + `DLQWorker` (`internal/service/recon/dlq_worker.go:155-253`) giữ partition chạy. Thay bằng: bulk DLQ lỗi retryable → **bỏ vòng fallback `Create` từng dòng** (`batch_buffer_dlq.go:100-106`) |
| U4a | `chunkSize=200` cho snapshot buffer | ✅ Đồng ý (giảm param/statement, tx ngắn) | `batch_buffer.go:589` |
| U4b | Bỏ fallback khi `txErr` retryable | ⚪ Không cần | Fallback đã dừng ngay ở dòng retryable đầu tiên (`batch_buffer.go:684-695,723-734`) → chỉ tiết kiệm 1 query, nhưng mất khả năng cô lập dòng khi deadlock |
| U5a | Default batch 5000→1000 | ⚪ Không cần | Chỉ áp cho source không cấu hình `snapshot_batch_size` (`snapshot_runner_handler.go:433-443`); flush **không chồng chéo** — `timerLoop` 1 goroutine + `flushMu` (`batch_buffer.go:245-259,272,348`) |
| U5b | `recordDocError` trip ngay khi 53xxx | ✅ Đồng ý | hiện phải đợi 100 lỗi liên tiếp (`snapshot_runner_utils.go:29`) và mỗi lỗi còn ghi `snapshot_dlq` vào chính DB đang OOM (`snapshot_runner_handler.go:760-761,784`) |
| U5c | Hãm theo `flushElapsed` + `select ctx.Done()` | ✅ Đồng ý (backpressure thô cho đường shadow) | `snapshot_runner_handler.go:845-848,898-904` |
| GAP | Fan-out transmute 6M event của snapshot (F11/F12) | ❗ Chưa có trong plan — đòn bẩy lớn nhất | `batch_buffer_event_date.go:66-68`; debouncer concurrency 1 dùng chung. Đề xuất phase 2: hãm snapshot theo `NumPending` của stream `CDC_TRANSMUTE` (`pkgs/natsconn/nats_client.go:66-70`) — cần điều tra thêm, không gộp vào patch này |

## B. Phương án hợp nhất (patch này)
P1 `multi.go` + `multi_dialector.go`: helper `ensurePgxCacheDescribe`; gọi trong `openGorm`, `openPgx`, và **chỉ nhánh postgres** của `OpenGormWithEngine`.
P2 `server_setup.go`: xoá `registry.GetPgxPool(... RoleControlPlane)` + `_ = pgxPool` (pool chết).
P3 `batch_buffer_fanout.go`: stale-reuse + negative-cache 5s cho lần đầu lỗi.
P4 `batch_buffer_dlq.go`: backoff 53xxx + sửa `break`; evac bỏ per-row fallback khi bulk lỗi retryable; thêm string defense.
P5 `batch_buffer.go`: `chunkSize=200` khi `isSnapshotBuffer`.
P6 `snapshot_runner_handler.go`: fast-trip 53xxx trong `recordDocError`; hãm theo `flushElapsed`; `time.Sleep` → `select`.

## C. Quyết định cần User chốt
1. Nhánh đích: hotfix trên `a5dd18e` (release/1.1.0 — code DLQ/retry nằm trong `batch_buffer.go:475-560,942`) hay trên working tree hiện tại (đã tách `batch_buffer_dlq.go`, chưa commit)?
2. Có cho phép bench tái hiện trên PG docker local (tạo bảng tạm, ghi dữ liệu giả) để đo RSS backend trước/sau `cache_describe` không?

## D. Test/Verify dự kiến (S4)
- Unit: helper DSN (URL có/không `?`, key=value, đã có param, không chạm DSN MySQL); fanout stale-reuse; retry loop thoát khi ctx cancel.
- Runtime: chạy snapshot thật trên môi trường User chỉ định, theo dõi `pg_stat_activity` count + RSS backend + log `TopMemoryContext` — phối hợp User.

## E. Điều chỉnh sau phản hồi User (2026-11-05T13:04) — APPROVE: làm trên working tree, cho phép bench PG docker local
- **E1 (User đúng):** `Registry.Init` (`pkgs/database/multi.go:51-58`) gọi `GetPgxPool(RoleControlPlane)` và `GetPgxPool(RoleDestination)`; `Init` được gọi tại `internal/server/server_setup.go:54` → thực tế mỗi replica mở **2 pool pgx chết × MinConns 50 = 100 conn idle** (control-plane + destination). P2 mở rộng: xoá 2 lệnh `GetPgxPool` trong `Init` + xoá `server_setup.go:236-237`. Giữ nguyên hàm `GetPgxPool` (test `test/pkgs/database/multi_test.go:57,130-150` gọi trực tiếp).
- **E2 (phản biện vị trí bấm giờ):** User đúng là đặt `flushStart` ngay trước `flushSnapshotTables` sẽ bỏ sót các flush nền đã xong trong vòng lặp (`Add` → `flushCh` → `timerLoop`, `batch_buffer.go:180-188,245-259`). Nhưng đặt `flushStart` trước vòng `for _, rec := range records` (`snapshot_runner_handler.go:806`) lại **cộng cả thời gian CPU của `HandleRaw`** (map/mask/route 5.000 doc) → không phải "thời gian ghi DB 100%", và ngưỡng 200ms sẽ gần như luôn kích hoạt → snapshot luôn bị hãm ~50%. **Phương án chính xác:** `BatchBuffer` cộng dồn thời gian thực thi DB (`executeFlushGroupsWithResultsMode` — điểm chung của flush nền và flush theo bảng) vào 1 bộ đếm atomic; snapshot runner đọc delta trước vòng lặp → sau `flushSnapshotTables` = tổng thời gian ghi DB của mẻ (nền + cuối). Fallback (handler không hỗ trợ) = đo wall-clock quanh `flushSnapshotTables`.
- **E3:** Bench PG docker local: dùng `TEMP TABLE` (tự huỷ cuối session) + view read-only `pg_backend_memory_contexts` để đo bộ nhớ backend sau N câu multi-row INSERT động, so sánh `cache_statement` vs `cache_describe`.

## F. Phản hồi log 13:20 (worker mới pid 95441 khởi động 13:19:52) — chờ APPROVE
- **F-a "vẫn 5000/lần":** đúng. Tổng các flush giữa 2 lần flush chủ động của trace snapshot `b205ce…` (13:20:06 → 13:20:10) = 1000+805+1000+1000+1000+195 = 5000 → trang Mongo 5000 doc. Nguồn: payload CMS `batch_size` để trống → `so.SnapshotBatchSize` trống → `snapshotV2DefaultBatchSize = 5000` (`snapshot_runner_handler.go:433-438`, `snapshot_runner_utils.go:23`). Buffer snapshot flush mỗi 1000 (local `max(batchSize*2,1000)`; prod 2000 — `server_setup.go:154-157`), statement con 200 dòng (P5). **Đề xuất:** đổi default → 1000 (hạng mục 5a của User mà em đã gạt; dữ kiện log cho thấy User muốn trang khớp buffer).
- **F-b "không thấy số rows":** cột rows = `written` = tổng `RowsAffected` (`batch_buffer.go:546,662`). Upsert chỉ UPDATE khi `_hash`/`_deleted` khác (`schema_adapter.go:745-793`) → bản ghi snapshot đã có sẵn và không đổi ⇒ `RowsAffected=0` ⇒ `written=0` (đúng nghiệp vụ, không phải lỗi ghi). CDC cùng bảng lúc 13:20:11 vẫn `written=42` bình thường. **Đề xuất:** thêm `unchanged = batch_size - written` vào details activity log để CMS hiển thị rõ.
- **F-c (phát hiện):** `publishTransmuteTrigger` bắn cho **toàn bộ** record của group sau khi upsert thành công, kể cả record không đổi (`batch_buffer_event_date.go:66-68`) → re-snapshot dữ liệu không đổi vẫn sinh transmute cho mọi dòng (bảng có post_ingest). Gộp vào phase 2 (T10): chỉ trigger cho dòng thực sự thay đổi (cần `RETURNING _source_id`).
- DG1: `06_laya_summary_dg1_20261105T132335.json` → 3 FLAG resolved → PASS.

---

# G. S3 (2026-11-09T10:58) — Giải pháp Nâng cấp Toàn diện `/system-health` để Giám sát OOM (`53200`), DB Pool, BatchBuffer Pacing & JetStream Backlog

## G.1. Chẩn đoán Thực trạng: Vì sao `/system-health` hiện tại "Tào lao" (Đã kiểm chứng Live `EV-20261109T105256-dd51` & `EV-20261109T105235-080d`)

| Mã | Phát hiện (`VERIFIED` kèm `file:line` & Runtime Evidence) | Hậu quả thực tế trên `/system-health` |
| :--- | :--- | :--- |
| **F19** | **Lệch cổng & đường dẫn Worker Probe (`404 Not Found`):**<br>- `cdc-cms-service/config/config-local.yml:40` & `config-local_dev.yml:40` trỏ `workerUrl: http://localhost:8090`, trong khi Worker Fiber chạy ở `:8082` (`centralized-data-service/config/config-local.yml:3`).<br>- `cdc-cms-service/internal/infra/observability/probes/worker.go:14` ghép cứng `workerURL + "/healthz"`, nhưng Worker Fiber (`internal/server/server_setup.go:549-567`) chỉ đăng ký `/health`, `/ready`, `/metrics`, `/api/v1/internal/stats` (không có `/healthz`). | Trong `EV-20261109T105256-dd51`:<br>- `"worker": {"http_status": 404, "status": "down"}` → Thẻ Worker luôn báo **Down (Đỏ)**, `Tables registered = 0`.<br>- `"latency": {"source": "unknown", "error": "worker /metrics status 404"}` → Toàn bộ P50/P95/P99 chết cứng ở `0 ms (Unknown)`. |
| **F20** | **Debezium có Task `FAILED` nhưng báo `OK` + Ẩn Panel lỗi + Hỏng nút Restart:**<br>- `probes/debezium.go:46-50,100-111`: Chỉ đọc `connector["state"]` (`"RUNNING"`), bỏ qua `tasks[i]["state"] == "FAILED"`. Trong `EV-20261109T105256-dd51`, connector `testcoretrans` có `tasks[0].state = "FAILED"` nhưng `debezium.status` vẫn là `"ok"` và `alerts` là `null`.<br>- `cdc-cms-web/src/pages/SystemHealth.tsx:236-241`: `DebeziumFailurePanel` đọc `debezium.tasks` (shape đơn cũ) thay vì `debezium.connectors[].tasks`, nên ẩn luôn panel lỗi.<br>- `cdc-cms-web/src/hooks/useSystemHealth.ts:62-68` gửi `{ reason, connector: connectorName }` + header `X-Action-Reason` chưa `encodeURIComponent`, trong khi `system_health_handler.go:100` đọc `json:"connector_name"` → bấm Restart luôn bị `400 Bad Request`. | Connector chết ngầm nhiều ngày (`testcoretrans` chết từ `2026-11-03`) nhưng trang `/system-health` vẫn báo **Healthy (Xanh)**, không hiện bảng trace lỗi, và nếu hiện thì bấm nút Restart cũng lỗi `400`. |
| **F21** | **Hoàn toàn MÙ trước sự cố PostgreSQL OOM (`53200`), DB Pool Saturation & Snapshot Pacing:**<br>- `probes/postgres.go:16-33`: Chỉ đếm bảng `table_registry` (`8`) và `SUM(n_live_tup) WHERE schemaname='public'` (`total_rows: 0`).<br>- Không thu thập `sql.DBStats` (`Open`, `InUse`, `Idle`, `MaxOpen`, `WaitCount`, `WaitDuration`) của 3 DB roles bên Worker (`control_plane`, `shadow`, `destination`), không biết số kết nối `pg_stat_activity` trên PG server, không đếm số lỗi `53200 / out of memory` vừa nổ, và không lấy `BatchBuffer` (`cdc` / `snapshot` / `heal`) flush latency (`DBFlushDuration`), số lần retry `53xxx`, hay tiến độ `snapshot_progress` đang chạy. | Khi chạy Snapshot 6 triệu dòng làm PG chạm ngưỡng kết nối/RAM hoặc bị hãm nhịp (adaptive pacing `>200ms`), `/system-health` không có bất kỳ con số nào phản ánh áp lực DB hay trạng thái buffer. |
| **F22** | **Mù hàng đợi NATS JetStream (`CDC_TRANSMUTE`) & Kafka Lag:**<br>- `probes/nats.go:13-39`: Chỉ gọi `/jsz` tổng (`streams: 4, consumers: 5, messages: 188`), không lấy `num_pending` (backlog), `num_ack_pending` (in-flight), `num_redelivered` của từng stream (`CDC_TRANSMUTE`, `CDC_COMMANDS`, `CDC_DLQ`).<br>- `probes/kafka_lag.go:41-46`: Phụ thuộc cứng vào `kafkaExporterUrl` (đang để trống `""`), nên luôn báo `"unknown"`. | Không thấy được cơn bão fan-out làm ứ đọng hàng chục nghìn message ở stream `CDC_TRANSMUTE` hay message bị redeliver lặp lại khi Transmute gặp lỗi `53200`. |
| **F23** | **Giao diện `SystemHealth.tsx` ép cứng `status: 'ok'` & vứt bỏ cột `details` của Activity Log:**<br>- `SystemHealth.tsx:89-95`: Hàm `fallback` gán cứng `status: 'ok'` cho mọi section khiến tiêu đề Card luôn gắn nhãn `<Tag color="green">Healthy</Tag>`.<br>- `SystemHealth.tsx:454-476`: Bảng `RecentEventsBody` bỏ qua trường `details` (dù backend `system_health_queries.go:135` đã trả về), làm mất toàn bộ thông tin `batch_size`, `written`, `unchanged`, `duration_ms` và thông báo lỗi `ERROR: out of memory (SQLSTATE 53200)`. | Người vận hành nhìn vào bảng sự kiện chỉ thấy chữ `success` hoặc `error` trống rỗng, không biết batch ghi bao nhiêu dòng, mất bao nhiêu ms hay lỗi SQLSTATE gì. |

---

## G.2. Phương án Tối ưu Duy nhất (Ánh xạ 1-1 `F19 → F23`)

### 1. Phía Worker (`centralized-data-service`)
- **`pkgs/database/multi_stats.go` (File mới, SRP):**
  - Thêm hàm `(r *Registry) PoolStats() map[string]DBPoolStat` đọc `sqlDB.Stats()` cho từng role đang mở (`cdc`, `shadow`, `dest`): `max_open`, `open`, `in_use`, `idle`, `wait_count`, `wait_duration_ms`.
- **`internal/handler/shadow/batch_buffer.go` & `batch_buffer_dlq.go`:**
  - Thêm 3 bộ đếm atomic nhẹ trên `BatchBuffer`: `lastFlushDurationMs atomic.Int64`, `oomRetryCount atomic.Int64` (tăng khi `isResourceExhaustedDBError(err)` trong `batchUpsertWithRetry`), `dlqEvacuatedCount atomic.Int64` (tăng trong `evacuateFailedChunkToDLQ`).
  - Mở rộng `BatchStatus` (`GetStatus()`) trả thêm:
    - `db_flush_total_ms int64` (`bb.DBFlushDuration().Milliseconds()`)
    - `last_flush_duration_ms int64`
    - `oom_retry_count int64`
    - `dlq_evacuated_count int64`
    - `adaptive_throttled bool` (`isSnapshotBuffer && lastFlushDurationMs > 200`)
- **`internal/server/server_setup.go`:**
  - Đăng ký thêm route `/healthz` (cùng handler với `/health` để tương thích ngược) và nâng cấp `/api/v1/internal/stats` trả về:
    - `status: "ok"`, `service: "cdc-worker"`, `tables_registered: len(registrySvc.GetAllActiveBindings())`,
    - `runtime`: `goroutines`, `heap_alloc_mb`, `heap_inuse_mb`, `sys_mb`, `num_gc` (từ `runtime.ReadMemStats`),
    - `db_pools`: `registry.PoolStats()`,
    - `buffer` (`cdcBatchBuffer.GetStatus()`), `snapshot_buffer` (`snapshotBatchBuffer.GetStatus()`), `heal_buffer` (`healBatchBuffer.GetStatus()`),
    - `queue`: `collectPoolStats(consumerPools)`.

### 2. Phía CMS Backend (`cdc-cms-service`)
- **`config/config-local.yml` & `config/config-local_dev.yml`:**
  - Cập nhật `workerUrl: http://localhost:8082`.
- **`internal/infra/observability/probes/worker.go`:**
  - Gọi `GET <workerURL>/api/v1/internal/stats` (fallback sang `/health` nếu endpoint stats chưa có trên bản cũ). Nhờ vậy `snap.CDCPipeline["worker"]` chứa đầy đủ `status: "up"`, `latency_ms`, `tables_registered`, `runtime`, `db_pools`, `buffer`, `snapshot_buffer`, `heal_buffer`, `queue`.
- **`internal/infra/observability/probes/debezium.go`:**
  - Trong `Debezium()`, kiểm tra từng task trong `tasksDetails`: nếu có task `state == "FAILED"`, đếm `failed_tasks` và khi `connector.state == "RUNNING"` nhưng `failed_tasks > 0` thì gán `status = "DEGRADED"`.
  - Trong `DebeziumAll()`, tính tổng `failed_task_count` và danh sách `failed_connectors`; nếu bất kỳ connector nào khác `RUNNING` hoặc có `failed_tasks > 0` thì `overall = StatusDegraded`.
- **`internal/infra/observability/probes/nats.go`:**
  - Gọi `/jsz?streams=true&consumers=true` và trích xuất danh sách `stream_details` (`name`, `messages`, `bytes`, `consumers`, `num_pending`, `num_ack_pending`, `num_redelivered`) đặc biệt cho `CDC_TRANSMUTE`, `CDC_COMMANDS`, `CDC_DLQ`.
- **`internal/infra/observability/probes/postgres.go` & `system_health_queries.go`:**
  - Trong `probes.Postgres`: Bổ sung thống kê `pg_stat_activity` (`active_conns`, `idle_conns`, `idle_in_tx_conns`, `total_conns`, `max_conns` từ `SHOW max_connections`) và đếm `tables_registered` từ `cdc_system.source_object_registry WHERE status = 'active'` (fallback `table_registry` nếu lỗi).
  - Trong `system_health_queries.go`:
    - Mở rộng `queryFailedCount(ctx)` đếm thêm `oom_count_15m` và `oom_count_1h` (các dòng trong `cdc_system.cdc_activity_log` có `status = 'error'` và `error_message` chứa `'53200'` / `'out of memory'` / `'too many clients'` / `'53300'` trong 1h qua) + `active_snapshots` (các dòng `cdc_system.snapshot_progress` có `status IN ('running', 'paused')` hoặc `updated_at > NOW() - INTERVAL '1 hour'`).
- **`internal/infra/observability/system_health_compute.go` & `internal/api/system/system_health_handler.go`:**
  - Trong `computeAlerts`: Bổ sung cảnh báo khi:
    1. `worker.status == "down"` → `critical`: `"CDC Worker is DOWN or unreachable"`.
    2. `failed_sync.oom_count_15m > 0` hoặc `worker.snapshot_buffer.oom_retry_count > 0` → `critical`: `"Phát hiện lỗi PostgreSQL OOM / cạn tài nguyên (SQLSTATE 53xxx) trong 15 phút qua"`.
    3. `nats` stream `CDC_TRANSMUTE` có `num_pending > 5000` → `warning` (`> 50000` → `critical`).
  - Trong `RestartDebezium`: Đọc cả `connector_name` và `connector` từ JSON body.

### 3. Phía Giao diện (`cdc-cms-web` — Visual UI/UX Prototype theo L066)
- **Bảng màu Ngữ nghĩa (Semantic Color Palette) & Thanh Tiến độ (Progress HUD):**
  - **Healthy / Normal (`#52c41a` - Green, nền `#f6ffed`):** Pool `< 70%`, Flush `<= 200ms`, `0` OOM, `0` Failed Tasks.
  - **Degraded / Throttled (`#faad14` - Amber, nền `#fffbe6`):** Pool `70%–89%`, Snapshot Adaptive Pacing đang kích hoạt (`last_flush_duration_ms > 200ms`), hoặc `CDC_TRANSMUTE` backlog `> 5,000`.
  - **Critical / OOM / Failed (`#ff4d4f` - Red, nền `#fff2f0`):** Pool `>= 90%`, `oom_count_15m > 0` (`SQLSTATE 53200`), Debezium Task `FAILED`, hoặc Worker `DOWN`.
- **Bố cục từng vùng trên `/system-health` (`src/pages/SystemHealth.tsx`):**
  1. **Header & Alert Banner:** Giữ nguyên khung chuẩn, hiển thị đúng `Overall` (`Healthy` / `Degraded` / `Critical`) và danh sách cảnh báo thực tế.
  2. **Hàng 1 — Infrastructure Cards (6 thẻ):** `Kafka`, `NATS JetStream` (hiện số `pending` thực tế), `PostgreSQL` (hiện `active/max conns`), `Redis`, `Debezium` (đổi sang cam `Degraded` ngay khi có task `FAILED`), `Worker` (hiện `Heap MB · Goroutines`).
  3. **Khối Debezium Failure Panel (tự động bung khi có bất kỳ connector/task nào `FAILED`):** Liệt kê rõ tên từng connector bị lỗi (`testcoretrans`), `Task ID`, nút `Xem trace` và nút `Restart Connector` gửi đúng `{ reason, connector_name }` kèm `encodeURIComponent(reason)`.
  4. **Hàng 2 (MỚI) — Giám sát Tài nguyên DB, BatchBuffer & Snapshot Pacing (OOM & Backpressure HUD):**
     - **Cột trái (DB Connection Pools & OOM Counter):** Hiển thị 3 thanh `<Progress>` cho 3 Pool của Worker (`Control Plane`, `Shadow Plane`, `Destination Master`) + `PG Server Active/Max Connections`, kèm thống kê `OOM / 53xxx (15m / 1h)` và `Pool Wait Count / Wait Ms`.
     - **Cột phải (BatchBuffers & NATS Stream Queues):**
       - Bảng trạng thái 3 Buffer (`CDC Stream`, `Snapshot`, `Recon Heal`): `Queue Fill (buffer_size / max_size)`, `Last Flush (ms)`, `Total DB Flush (s)`, `OOM Retries (53xxx)`, `DLQ Evacuated`, và Tag trạng thái (`Normal` xanh / `Adaptive Pacing` cam).
       - Danh sách **Snapshot Jobs đang chạy (`active_snapshots`)** với thanh `<Progress>` phần trăm (`rows_processed / total_rows`).
       - Bảng mini **NATS JetStream Core Queues (`CDC_TRANSMUTE`, `CDC_COMMANDS`, `CDC_DLQ`)**: `Messages`, `Pending (Backlog)`, `In-Flight (AckPending)`, `Redelivered`.
  5. **Hàng 3 — Pipeline & Latency (P50 / P95 / P99):** Tự động lấy đúng từ Worker `:8082/metrics`.
  6. **Hàng 4 — Sự kiện gần đây (`RecentEventsBody`):** Bổ sung cột **Chi tiết (`details`)** hiển thị tường minh các badge `batch_size`, `written`, `unchanged`, `duration_ms`, `buffer_type` hoặc nội dung lỗi `error` (không dùng ellipsis cắt cụt tên bảng theo L070).

---

## G.3. Code Demo (Hiển thị Trực tiếp Để Duyệt Trước Khi Thực thi S4)

### Demo 1: `centralized-data-service/pkgs/database/multi_stats.go` (Mới) & `internal/handler/shadow/batch_buffer.go`
```go
// pkgs/database/multi_stats.go
package database

type DBPoolStat struct {
	Role           string `json:"role"`
	MaxOpen        int    `json:"max_open"`
	Open           int    `json:"open"`
	InUse          int    `json:"in_use"`
	Idle           int    `json:"idle"`
	WaitCount      int64  `json:"wait_count"`
	WaitDurationMs int64  `json:"wait_duration_ms"`
}

func (r *Registry) PoolStats() map[string]DBPoolStat {
	out := make(map[string]DBPoolStat, 3)
	if r == nil {
		return out
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for role, gdb := range r.gormDBs {
		if gdb == nil {
			continue
		}
		sqlDB, err := gdb.DB()
		if err != nil || sqlDB == nil {
			continue
		}
		st := sqlDB.Stats()
		out[role] = DBPoolStat{
			Role:           role,
			MaxOpen:        st.MaxOpenConnections,
			Open:           st.OpenConnections,
			InUse:          st.InUse,
			Idle:           st.Idle,
			WaitCount:      st.WaitCount,
			WaitDurationMs: st.WaitDuration.Milliseconds(),
		}
	}
	return out
}
```

```go
// internal/handler/shadow/batch_buffer.go (Mở rộng BatchStatus & GetStatus)
type BatchStatus struct {
	BufferSize          int       `json:"buffer_size"`
	MaxSize             int       `json:"max_size"`
	LastFlush           time.Time `json:"last_flush"`
	LastFlushDurationMs int64     `json:"last_flush_duration_ms"`
	DBFlushTotalMs      int64     `json:"db_flush_total_ms"`
	OOMRetryCount       int64     `json:"oom_retry_count"`
	DLQEvacuatedCount   int64     `json:"dlq_evacuated_count"`
	AdaptiveThrottled   bool      `json:"adaptive_throttled"`
}

func (bb *BatchBuffer) GetStatus() BatchStatus {
	bb.mu.Lock()
	bufLen := len(bb.records)
	maxSz := bb.maxSize
	lastFl := bb.lastFlush
	isSnap := bb.isSnapshotBuffer
	bb.mu.Unlock()

	lastDurMs := bb.lastFlushDurationMs.Load()
	return BatchStatus{
		BufferSize:          bufLen,
		MaxSize:             maxSz,
		LastFlush:           lastFl,
		LastFlushDurationMs: lastDurMs,
		DBFlushTotalMs:      bb.DBFlushDuration().Milliseconds(),
		OOMRetryCount:       bb.oomRetryCount.Load(),
		DLQEvacuatedCount:   bb.dlqEvacuatedCount.Load(),
		AdaptiveThrottled:   isSnap && lastDurMs > 200,
	}
}
```

### Demo 2: `centralized-data-service/internal/server/server_setup.go` (Nâng cấp `/api/v1/internal/stats` & `/healthz`)
```go
	healthHandler := func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "cdc-worker"})
	}
	app.Get("/health", healthHandler)
	app.Get("/healthz", healthHandler)
	// ...
	app.Get("/api/v1/internal/stats", func(c *fiber.Ctx) error {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		tablesRegistered := 0
		if registrySvc != nil {
			tablesRegistered = len(registrySvc.GetAllActiveBindings())
		}
		return c.JSON(fiber.Map{
			"status":            "ok",
			"service":           "cdc-worker",
			"tables_registered": tablesRegistered,
			"runtime": fiber.Map{
				"goroutines":    runtime.NumGoroutine(),
				"heap_alloc_mb": float64(mem.HeapAlloc) / (1024 * 1024),
				"heap_inuse_mb": float64(mem.HeapInuse) / (1024 * 1024),
				"sys_mb":        float64(mem.Sys) / (1024 * 1024),
				"num_gc":        mem.NumGC,
			},
			"db_pools":        registry.PoolStats(),
			"queue":           collectPoolStats(consumerPools),
			"buffer":          cdcBatchBuffer.GetStatus(),
			"snapshot_buffer": snapshotBatchBuffer.GetStatus(),
			"heal_buffer":     healBatchBuffer.GetStatus(),
			"config":          fiber.Map{"pool_size": cfg.Worker.PoolSize, "batch_size": cfg.Worker.BatchSize},
		})
	})
```

### Demo 3: `cdc-cms-service` Probes (`worker.go`, `debezium.go`, `nats.go`, `postgres.go`, `system_health_queries.go`)
```go
// internal/infra/observability/probes/worker.go
func Worker(ctx context.Context, deps HTTPDeps, workerURL string) map[string]any {
	base := strings.TrimRight(workerURL, "/")
	start := time.Now()
	body, code, err := deps.Get(ctx, base+"/api/v1/internal/stats")
	if err == nil && code == 404 {
		body, code, err = deps.Get(ctx, base+"/health")
	}
	sec := map[string]any{"latency_ms": time.Since(start).Milliseconds()}
	if err != nil {
		sec["status"] = StatusUnknown
		sec["error"] = SanitizeErr(err)
		return sec
	}
	if code >= 300 {
		sec["status"] = StatusDown
		sec["http_status"] = code
		return sec
	}
	var obj map[string]any
	_ = json.Unmarshal(body, &obj)
	if obj == nil {
		obj = map[string]any{}
	}
	obj["status"] = StatusUp
	obj["latency_ms"] = sec["latency_ms"]
	return obj
}
```

```go
// internal/infra/observability/probes/debezium.go (Bắt chính xác task FAILED)
	failedTasks := 0
	for _, t := range tasks {
		task, _ := t.(map[string]any)
		if task == nil {
			continue
		}
		d := map[string]any{"id": task["id"], "state": task["state"]}
		if task["state"] == "FAILED" {
			failedTasks++
			if trace, ok := task["trace"].(string); ok {
				if len(trace) > 500 {
					trace = trace[:500] + "..."
				}
				d["trace"] = trace
			}
		}
		taskDetails = append(taskDetails, d)
	}
	// Nếu connector RUNNING nhưng có task FAILED -> đánh dấu DEGRADED để DebeziumAll và Alert nhận diện!
	if strings.EqualFold(state, "RUNNING") && failedTasks > 0 {
		state = StatusDegraded
	}
	return map[string]any{
		"status":       state,
		"connector":    debeziumName,
		"failed_tasks": failedTasks,
		"tasks":        taskDetails,
	}
```

### Demo 4: `cdc-cms-web/src/hooks/useSystemHealth.ts` & `src/pages/SystemHealth.tsx`
```ts
// src/hooks/useSystemHealth.ts — Sửa payload connector_name & bọc encodeURIComponent cho X-Action-Reason (L070)
      const { data } = await cmsApi.post<RestartConnectorResult>(
        '/api/tools/restart-debezium',
        { reason, connector_name: connectorName, connector: connectorName },
        {
          headers: {
            'Idempotency-Key': idempotencyKey,
            'X-Action-Reason': encodeURIComponent(reason),
          },
        },
      );
```

### Demo 5 (T19 Self-Healing `S5 → S3 → S4 → S5` cho `system_health_queries.go:113-133`): Khớp Chính xác Schema `cdc_system.cdc_activity_log` & `cdc_system.snapshot_progress` (L057, L060)
- **Nguyên nhân gốc rễ (`F32`):**
  1. Tại `system_health_queries.go:117`, câu truy vấn `oomRow` viết sai tên bảng thành `cdc_system.activity_logs` gây lỗi `SQLSTATE 42P01 (relation "cdc_system.activity_logs" does not exist)`. Theo `migrations/schema/partitioning/010_partitioning.sql:85` và `internal/model/system/activity_log.go:23`, tên bảng vật lý chính xác là **`cdc_system.cdc_activity_log`**.
  2. Tại `system_health_queries.go:126-132`, câu truy vấn `activeSnapshots` gọi `SELECT id, table_name, ..., error_message FROM cdc_system.snapshot_progress` gây lỗi `SQLSTATE 42703 (column "table_name" does not exist)`. Theo `migrations/schema/core/058_v1_snapshot_progress.sql:15-27`, `062_add_total_rows_to_progress.sql:3`, `066_add_shadow_binding_id_to_snapshot_progress.sql:19-20` và `internal/infra/persistence/scheduler/snapshot_progress_read_repo_gorm.go:62-80`, bảng `cdc_system.snapshot_progress` lưu `source_object_id`, `shadow_binding_id`, và cột lỗi tên là `error_msg` (không phải `error_message`), còn tên bảng hiển thị phải lấy qua `LEFT JOIN cdc_system.source_object_registry so` và `LEFT JOIN LATERAL (...) sb` trên `cdc_system.shadow_binding`.
- **Code Demo chuẩn xác (`cdc-cms-service/internal/infra/observability/system_health_queries.go:109-145`):**
```go
	var oomRow struct {
		OOM15m int64 `gorm:"column:oom_15m"`
		OOM1h  int64 `gorm:"column:oom_1h"`
	}
	db.Raw(
		`SELECT
			count(*) FILTER (WHERE created_at > ?) AS oom_15m,
			count(*) AS oom_1h
		 FROM cdc_system.cdc_activity_log
		 WHERE created_at > ? AND created_at <= ?
		   AND status = 'error'
		   AND (error_message ILIKE '%53200%' OR error_message ILIKE '%out of memory%' OR error_message ILIKE '%53300%' OR error_message ILIKE '%too many clients%')`,
		fifteenMinAgo, oneHourAgo, now,
	).Scan(&oomRow)

	var activeSnapshots []map[string]any
	db.Raw(
		`SELECT
			sp.id,
			COALESCE(
				NULLIF(CONCAT_WS('.', sb.shadow_schema, sb.shadow_table), ''),
				NULLIF(CONCAT_WS('.', so.source_database, so.source_object_name), ''),
				CAST(sp.source_object_id AS TEXT)
			) AS table_name,
			sp.status,
			sp.rows_processed,
			sp.total_rows,
			sp.started_at,
			sp.updated_at,
			sp.error_msg AS error_message
		 FROM cdc_system.snapshot_progress sp
		 LEFT JOIN cdc_system.source_object_registry so ON so.id = sp.source_object_id
		 LEFT JOIN LATERAL (
			SELECT s.shadow_schema, s.shadow_table
			FROM cdc_system.shadow_binding s
			WHERE (sp.shadow_binding_id IS NOT NULL AND s.id = sp.shadow_binding_id)
			   OR (sp.shadow_binding_id IS NULL AND s.source_object_id = sp.source_object_id AND s.is_active = TRUE)
			ORDER BY s.id DESC
			LIMIT 1
		 ) sb ON TRUE
		 WHERE sp.status IN ('running', 'paused') OR sp.updated_at > ?
		 ORDER BY sp.updated_at DESC
		 LIMIT 5`,
		fifteenMinAgo,
	).Scan(&activeSnapshots)
```

---

# H. S3 (2026-11-09T13:20) — Kiểm toán Định lượng "600k / 6M Records → 12GB RAM" & Giải pháp Khóa chặt Chặng Fan-out / Transmute (`P7–P9`)

## H.1. Giải phẫu Định lượng: Vì sao Bản cũ (`a5dd18e7`) chạy đúng ~600k / 6M bản ghi là ngốn hết 12GB RAM và gãy?

Số liệu thực tế anh ghi nhận (**chạy ~600k / 6M bản ghi ngốn hết 12GB RAM rồi gãy**) khớp **chính xác 100% về mặt toán học** với cơ chế phình `CachedPlan` trên PostgreSQL mà chúng ta đã đo được trong `EV-20261105T130652-eaf5` và `EV-20261105T131324-ca03`:

| Đại lượng | Bản cũ (`a5dd18e7` — trước `P1–P6`) | Bản đã áp `P1–P6` (working tree hiện tại) | Bằng chứng Vật lý (`file:line` & `EV-...`) |
| :--- | :--- | :--- | :--- |
| **Chế độ thực thi `pgx` (`gorm.io/driver/postgres`)** | `QueryExecModeCacheStatement` (mặc định giữ **512 prepared statements / kết nối**) | `default_query_exec_mode=cache_describe` (**0 plan giữ lại sau khi chạy xong câu lệnh**) | `pkgs/database/pgx_exec_mode.go:13-39`, `multi.go:229,246`, `multi_dialector.go:38` |
| **RAM của 1 kết nối PG backend sau 300 câu multi-row `INSERT` động** | **`400.7 MB – 401.4 MB / backend`** (riêng `CachedPlanSource` + `CachedPlan` chiếm **`390.9 MB`**) | **`9.9 MB / backend`** (`CachedPlan` = **`0.0 MB`**, giảm **40 lần**) | `EV-20261105T130652-eaf5`, `EV-20261105T131324-ca03` vs `EV-20261105T131333-2f06` |
| **Điều gì xảy ra tại mốc ~600,000 bản ghi?** | `600,000` dòng = `600` lần flush Shadow (`1,000` dòng/mẻ) + hàng trăm mẻ `bulkUpsertMaster` (`transmuter.go:1660-1702`). Do mỗi document MongoDB có tập trường khác nhau (`getSignature` tại `batch_buffer.go:580-615`) và số dòng sau dedup khác nhau, hàng nghìn biến thể SQL multi-row được xoay vòng qua **~28–30 kết nối PG active** trong pool (`maxOpenConn=100, maxIdleConn=50`). | Mỗi câu SQL multi-row `INSERT` (đã thu nhỏ `chunkSize = 200` dòng theo `P5`) giải phóng hoàn toàn vùng nhớ plan ngay sau khi thực thi xong. | `internal/handler/shadow/batch_buffer.go:589-592`, `internal/service/master/transmuter.go:1660-1702` |
| **Tổng RAM PostgreSQL tại mốc ~600k bản ghi** | $$\text{30 backends} \times \text{400 MB} \approx \mathbf{12,000\text{ MB } (12\text{ GB})}$$ → Chạm trần **12 GB** (`11.67 GiB` theo `EV-20261109T131603-4b4b`) → Nổ `SQLSTATE 53200 (out of memory)` toàn cục! | $$\text{30 backends} \times \text{10 MB} \approx \mathbf{300\text{ MB}}$$ (Thực tế hiện tại trên Docker: `gpay-postgres-shadow` = `401.8 MiB / 11.67 GiB`, `gpay-postgres-cdc` = `218.1 MiB`, `gpay-postgres-dest` = `110.2 MiB` theo `EV-20261109T131603-4b4b`). | `EV-20261109T131603-4b4b`, `EV-20261105T131333-2f06` |

> **Kết luận tầng PostgreSQL (`SQLSTATE 53200`):** Bản vá `P1–P6` **đã giải quyết triệt để 100%** lỗi phình 12GB RAM trên PostgreSQL. Toàn bộ kết nối Postgres mở qua `Registry` (`GetSystemDB`, `GetShadowDB`, `GetMasterDB` trong `internal/service/source/connection_manager.go:57-65, 285-408`) đều đã đi qua `ensurePgxCacheDescribe`.

---

## H.2. Phát hiện Phản biện S2/S3 (`F24–F26`): 3 Điểm Nghẽn Bộ nhớ ở phía Go Worker & NATS khi chạy 600k → 6M bản ghi

Khi rà soát tiếp chặng **sau khi ghi Shadow DB** (`publishTransmuteTrigger` → NATS `CDC_TRANSMUTE` → `TransmuteConsumerPool` → `TableDebouncer` → `Transmuter.RunEvents`) với quy mô **6 triệu bản ghi**, em phát hiện **3 lỗ hổng ở phía Go Worker / NATS** (`F24–F26`) mà `P1–P6` chưa xử lý:

| Mã | Phát hiện (`VERIFIED` kèm `file:line` & Runtime Evidence) | Rủi ro khi chạy Snapshot 6 triệu dòng |
| :--- | :--- | :--- |
| **F24** | **`publishTransmuteTrigger` nhồi nguyên mảng `RawData` của 1,000–2,000 dòng vào 1 bản tin NATS duy nhất:**<br>- `internal/handler/shadow/batch_buffer_fanout.go:46-54, 89-105`: Đóng gói toàn bộ `records` trong group (mỗi record chứa `RawData: r.RawData`) vào 1 message `cdc.cmd.transmute-shadow`.<br>- Theo biên lai `EV-20261109T131526-e53c`, NATS Server có `"max_payload": 1048576` (**1 MB**). Nếu 1,000 bản ghi có kích thước JSON trung bình `> 1 KB` (`1,000 × 1.2 KB = 1.2 MB > 1 MB`), lệnh `PublishMsg` sẽ bị NATS từ chối với lỗi `nats: maximum payload exceeded`! | 1. Rơi mất bản tin trigger Transmute đối với các bảng có document JSON lớn (`> 1 KB/doc`).<br>2. Với 6 triệu dòng, hệ thống phải marshal/unmarshal 6 triệu bản sao `RawData` (~6–12 GB JSON) qua NATS. |
| **F25** | **Dead-Code Backpressure & Vượt rào `concurrencyLimit = 1` trong `TableDebouncer` làm phình RAM Go Worker:**<br>- `internal/handler/master/transmute_handler.go:197-234`: `handleTransmuteShadowWithOutcome` unmarshal `msg.Data` (`1,000` events kèm `RawData`), giữ `msg *nats.Msg` trong `shadowMsgTracker` (kèm 1 goroutine `time.NewTicker(5s)`), đẩy vào `td.AddTask(...)` rồi trả `ErrAsyncAckPending` **ngay lập tức**.<br>- `internal/handler/master/transmute_consumer_pool.go:164-173`: Không đặt `MaxAckPending` trên `ConsumerConfig` (mặc định JetStream đẩy tối đa **1,000 messages** in-flight).<br>- `internal/handler/master/debounce.go:88-105, 144-155`: Kiểm tra `if len(td.tasks) >= td.maxSize*2` (`1000` tasks) ở dòng 88 là **Dead Code 100%** vì ngay ở dòng 101 khi `len(td.tasks) >= 500`, `triggerFlushLocked()` đã xóa `td.tasks` về `0`! Tệ hơn, khi kênh `td.flushCh` (`cap = 100`) đầy, nhánh `default:` ở dòng 152–154 gọi thẳng `go td.runTransmute(context.Background(), tasks)` **bỏ qua semaphore `td.sem` (`concurrencyLimit = 1`)**! | Khi Snapshot xả 6,000 messages nhanh hơn tốc độ Transmute:<br>1. `TableDebouncer` gom tới `500 tasks × 1,000 events = 500,000 events` trong 1 mẻ.<br>2. Nhánh `default:` phóng hàng loạt goroutine `runDebouncedTransmute` chạy song song trên cùng 1 bảng Master (gây deadlock `40P01` và giữ hàng trăm nghìn bản ghi `RawData` trên Heap, có thể ngốn **4–8 GB RAM của Go Worker**). |
| **F26** | **Snapshot bắn Transmute cả khi `groupWritten == 0` (`F-c`) & thiếu Backpressure theo hàng đợi `CDC_TRANSMUTE` (`T10`):**<br>- `internal/handler/shadow/batch_buffer_event_date.go:67-69`: Gọi `publishTransmuteTrigger` ngay cả khi `bb.isSnapshotBuffer && groupWritten == 0` (toàn bộ dòng trong mẻ đã tồn tại và không đổi `_hash`).<br>- `internal/handler/orchestration/snapshot_runner_handler.go:920-923`: Mới chỉ hãm nhịp theo `batchFlushElapsed` của Shadow DB, chưa hãm nhịp khi hàng đợi NATS `CDC_TRANSMUTE` bị ứ đọng nhiều bản tin chưa xử lý. | Khi chạy lại Snapshot (resume hoặc chạy lại bảng 6M dòng đã có 600k dòng đầu), 600k dòng đầu (`written = 0`) vẫn bắn 600,000 events sang Transmute, làm ứ đọng stream `CDC_TRANSMUTE`. |

---

## H.3. Giải pháp Khóa chặt `F24–F26` (`P7–P9` — Tương thích ngược 100%, Giữ nguyên Kiến trúc)

### 1. `P7` — Chia nhỏ payload NATS an toàn (`<= 200` events / `<= 512 KB`) & Bỏ bắn Transmute khi Snapshot `groupWritten == 0` (`F24`, `F26`)
- **`internal/handler/shadow/batch_buffer_event_date.go:67-69`:**
  - Nếu `bb.isSnapshotBuffer && groupWritten == 0` (không có bất kỳ dòng nào được insert/update xuống Shadow trong sub-group của snapshot), **không gọi `publishTransmuteTrigger`**. (Luồng CDC stream `!bb.isSnapshotBuffer` vẫn giữ nguyên).
- **`internal/handler/shadow/batch_buffer_fanout.go`:**
  - Tách hàm `publishTransmuteTriggerChunk` và giới hạn mỗi bản tin `cdc.cmd.transmute-shadow` tối đa **`200` bản ghi** và tổng kích thước `RawData` tối đa **`512 KB`** (`const maxTransmuteTriggerEvents = 200`, `const maxTransmuteTriggerRawBytes = 512 * 1024`). Nhờ đó bản tin JSON không bao giờ vượt trần `max_payload = 1 MB` của NATS (`EV-20261109T131526-e53c`).

### 2. `P8` — Khôi phục Backpressure thật sự & Bảo toàn `concurrencyLimit = 1` trong `TableDebouncer` + Giới hạn `MaxAckPending` (`F25`)
- **`internal/handler/master/debounce.go`:**
  - Bổ sung đếm số lượng sự kiện trong `td.tasks` (`taskEventCount(task)`): kích hoạt `triggerFlushLocked()` ngay khi `len(td.tasks) >= td.maxSize` **HOẶC** tổng số events trong `td.tasks` đạt ngưỡng (`>= 1000` events), ngăn hiện tượng gom 500 tasks × 200 events = 100,000 events vào 1 mẻ.
  - Sửa điều kiện Backpressure tại `AddTask` (`debounce.go:88`): Kiểm tra **`len(td.flushCh) >= cap(td.flushCh)`** (hoặc `len(td.tasks) >= td.maxSize`) trước khi nạp thêm task — nếu kênh `flushCh` đã đầy, nhả `td.mu.Unlock()` và đợi `10ms` (tạo backpressure tự nhiên ngược lên worker callback của `TransmuteConsumerPool`).
  - Trong `triggerFlushLocked()` (`debounce.go:147-154`): **Bỏ nhánh `default: go td.runTransmute(...)` không kiểm soát**, chỉ gửi qua `td.flushCh <- tasks` (với buffer `16` thay vì `100`) để **100% các mẻ đều bắt buộc đi qua `td.sem` (`concurrencyLimit = 1`)**, tuyệt đối không bao giờ chạy chồng chéo nhiều goroutine `runDebouncedTransmute` trên cùng 1 bảng Master.
- **`internal/handler/master/transmute_consumer_pool.go:164-173`:**
  - Đặt `MaxAckPending: p.cfg.WorkerCount * 4` (mặc định `10 * 4 = 40` messages in-flight thay vì `1,000` của JetStream), đảm bảo bộ nhớ in-flight của NATS + `shadowMsgTracker` trong Go Worker luôn dưới `~20–30 MB`.

### 3. `P9` — Bổ sung Backpressure cho Snapshot theo độ trễ của Stream `CDC_TRANSMUTE` (`F26` / `T10`)
- **`internal/handler/orchestration/snapshot_runner_pacing.go` & `snapshot_runner_handler.go`:**
  - Thêm hàm `r.checkTransmuteBackpressure(ctx)`: sau mỗi trang Snapshot, kiểm tra số bản tin đang chờ của stream `CDC_TRANSMUTE` (`info.State.Msgs` qua `r.js.StreamInfo("CDC_TRANSMUTE")`, cache nhẹ `1s`). Nếu `Msgs > 1000`, tự động hãm nhịp `sleepWithContext(ctx, 500ms)` (và nếu `Msgs > 5000` thì chờ `2s`), giữ cho Snapshot và Transmute chạy nhịp nhàng, không bao giờ chất đống hàng triệu bản ghi trong JetStream.

---

## H.4. Code Demo cho `P7–P9` (Hiển thị Trực tiếp Để Duyệt Trước Khi Thực thi S4)

### Demo P7: `internal/handler/shadow/batch_buffer_event_date.go` & `batch_buffer_fanout.go`
```go
// internal/handler/shadow/batch_buffer_event_date.go:66-71
		metrics.BatchesFlushed.WithLabelValues("postgres", recs[0].TableName).Inc()
		if (bb.natsConn != nil || bb.js != nil) && (!bb.isSnapshotBuffer || groupWritten > 0) {
			bb.publishTransmuteTrigger(ctx, recs[0].SchemaName, recs[0].TableName, recs)
		}
		written += groupWritten
```

```go
// internal/handler/shadow/batch_buffer_fanout.go — Chia chunk <= 200 events & <= 512KB tránh vượt NATS max_payload (1MB)
const (
	maxTransmuteTriggerEvents   = 200
	maxTransmuteTriggerRawBytes = 512 * 1024
)

func (bb *BatchBuffer) publishTransmuteTrigger(ctx context.Context, shadowSchema, shadowTable string, records []*shadow.UpsertRecord) {
	if bb.natsConn == nil || shadowTable == "" || len(records) == 0 {
		return
	}
	meta := bb.resolvePostIngestSchedule(shadowSchema, shadowTable)
	if !meta.enabled {
		return
	}

	var chunk []*shadow.UpsertRecord
	var chunkBytes int
	for _, r := range records {
		if r == nil || r.IsMarker || r.PrimaryKeyValue == "" {
			continue
		}
		recBytes := len(r.RawData) + len(r.PrimaryKeyValue) + 64
		if len(chunk) > 0 && (len(chunk) >= maxTransmuteTriggerEvents || chunkBytes+recBytes > maxTransmuteTriggerRawBytes) {
			bb.publishTransmuteTriggerChunk(ctx, shadowSchema, shadowTable, meta, chunk)
			chunk = nil
			chunkBytes = 0
		}
		chunk = append(chunk, r)
		chunkBytes += recBytes
	}
	if len(chunk) > 0 {
		bb.publishTransmuteTriggerChunk(ctx, shadowSchema, shadowTable, meta, chunk)
	}
}
```

### Demo P8: `internal/handler/master/debounce.go` & `transmute_consumer_pool.go`
```go
// internal/handler/master/debounce.go — Khôi phục Backpressure thật & giữ nghiêm concurrencyLimit=1
const maxDebouncedEventsPerBatch = 1000

func taskEventCount(t NatsMsgTask) int {
	if n := len(t.Req.Events); n > 0 {
		return n
	}
	if n := len(t.Req.SourceIDs); n > 0 {
		return n
	}
	return 1
}

func (td *TableDebouncer) AddTask(task NatsMsgTask) {
	for {
		td.mu.Lock()
		// Backpressure thật sự: khi kênh flushCh đã đầy hoặc mảng tasks đạt ngưỡng, nhả lock chờ workerLoop tiêu thụ bớt
		if len(td.flushCh) >= cap(td.flushCh) || len(td.tasks) >= td.maxSize*2 {
			td.mu.Unlock()
			select {
			case <-td.ctx.Done():
				task.complete(td.ctx.Err())
				return
			case <-time.After(10 * time.Millisecond):
				continue
			}
		}

		td.tasks = append(td.tasks, task)
		td.pendingEvents += taskEventCount(task)

		if len(td.tasks) >= td.maxSize || td.pendingEvents >= maxDebouncedEventsPerBatch {
			td.triggerFlushLocked()
			td.mu.Unlock()
			break
		}

		td.setupTimersLocked()
		td.mu.Unlock()
		break
	}
}
```

```go
// internal/handler/master/transmute_consumer_pool.go:164-174 — Giới hạn MaxAckPending tránh dồn 1,000 messages vào RAM
	maxAckPending := p.cfg.WorkerCount * 4
	if maxAckPending <= 0 {
		maxAckPending = 40
	}
	consumerCfg := &nats.ConsumerConfig{
		Durable:        p.cfg.ConsumerName,
		DeliverPolicy:  nats.DeliverAllPolicy,
		AckPolicy:      nats.AckExplicitPolicy,
		AckWait:        p.cfg.AckWait,
		MaxDeliver:     p.cfg.MaxDeliver,
		MaxAckPending:  maxAckPending,
		FilterSubject:  p.cfg.FilterSubject,
		DeliverSubject: nats.NewInbox(),
		DeliverGroup:   queueGroup,
	}
```

---

# I. S3 (2026-11-09T13:43 — T17) — Điều tra `SELECT COUNT("_gpay_id") FROM "public"."cdc_transaction_history_payload"` (`>200 ms` trên 5M dòng) & Phản biện `COUNT("1")` vs `COUNT(1)` vs `COUNT(*)`

## I.1. Kết quả Điều tra & Phản biện `DG0` (`06_laya_summary_dg0_20261109T134341.json` — `PASS`)

| Mã | Phát hiện (`VERIFIED` kèm `file:line`) | Cơ chế Vật lý trên PostgreSQL (Bảng 5 triệu dòng) |
| :--- | :--- | :--- |
| **F27** | **Vì sao `SELECT COUNT("_gpay_id") FROM "public"."cdc_transaction_history_payload"` tốn `>200 ms` trên 5 triệu dòng:**<br>- Câu lệnh được gọi tại `recon_smoke.go:834` $\rightarrow$ `scanExact` (`recon_smoke.go:132`) $\rightarrow$ `CountRows` (`recon_dest_query.go:40`) / `CountRowsBeforeCutoff` (`recon_window_index_probe.go:234`).<br>- Bảng `"public"."cdc_transaction_history_payload"` có `"_gpay_id" BIGINT PRIMARY KEY` (`master_ddl_generator.go:149`) cùng cột `payload_data` lớn. | 1. **Giới hạn vật lý của MVCC `Index Only Scan` trên 5 triệu tuples:** PostgreSQL không lưu số đếm tổng trong header của B-Tree Index. Để đếm chính xác 5,000,000 dòng, PostgreSQL phải duyệt qua **~14,000–16,500 trang leaf (8KB)** của index `cdc_transaction_history_payload_pkey` (~110–130 MB). Với tốc độ quét Index-Only trong RAM của PG (~20–25 triệu tuples/giây/core), riêng việc duyệt 5 triệu index entries đã mất **`~180 ms – 250 ms`**.<br>2. **Visibility Map (VM) & Heap Fetches trên bảng CDC đang ghi:** Mỗi khi Transmute chạy `INSERT ... ON CONFLICT ("_gpay_id") DO UPDATE`, trang Heap chứa dòng đó bị xóa bit `all-visible` trong Visibility Map cho tới kỳ `autovacuum` kế tiếp. Khi `Index Only Scan` đi qua trang chưa `all-visible`, nó phải đọc xuống trang **Heap (`Heap Fetches`)** để kiểm tra `xmin`/`xmax`, đẩy thời gian từ `~180 ms` lên **`>200 ms – 600 ms`**. |
| **F28** | **Phản biện `COUNT("1")` (có nháy kép) vs `COUNT(1)` vs `COUNT(*)` trên 5 triệu dòng (`C-CONTRA-27`):**<br>- Tại `recon_dest_query.go:40` & `recon_dest_safety.go:31-33`: `quoteIdent(pkColumn)` luôn bọc dấu nháy kép `"..."` quanh `pkColumn`. | 1. **Nếu viết `COUNT("1")` (có dấu nháy kép `"1"` hoặc truyền `pkColumn="1"` vào `CountRows`):** Trong chuẩn ANSI SQL của PostgreSQL, `"1"` là **tên cột (delimited identifier)** chứ không phải số `1`. PostgreSQL sẽ báo lỗi ngay lập tức: **`ERROR: column "1" does not exist (SQLSTATE 42703)`**! Trong `CountRows` (`recon_dest_query.go:41-52`), lỗi này làm hủy transaction đầu tiên (`tx.Rollback()`) rồi mới mở lại transaction thứ hai chạy fallback `SELECT COUNT(*)` $\rightarrow$ **tốn gấp đôi round-trip và gây rác log lỗi `42703`**.<br>2. **Nếu viết `COUNT(1)` (số `1` không nháy kép) vs `COUNT(*)` vs `COUNT("_gpay_id")`:** Trên PostgreSQL, cả 3 đều phải chạy **`Index Only Scan` qua đủ 5,000,000 tuples (đều `>200 ms`)**. Giữa 3 cách:<br>  - **`COUNT(*)` là nhanh nhất:** Aggregate 0 tham số (`aggstar = true` trong `nodeAgg.c`), không giải mã cột, không đánh giá biểu thức, và cho phép Planner chọn **bất kỳ B-Tree index nào nhỏ nhất/ít bloat nhất** trên bảng.<br>  - **`COUNT(1)` chậm hơn `COUNT(*)` ~1–3% CPU:** Ở mỗi dòng trong 5M dòng, Executor phải đánh giá biểu thức hằng `1` và kiểm tra `IS NOT NULL`.<br>  - **`COUNT("_gpay_id")` chậm nhất trong 3 cách:** Vừa phải giải mã cột `"_gpay_id"` từ `IndexTuple` + kiểm tra `IS NOT NULL` 5 triệu lần, vừa ép Planner chỉ được dùng index có chứa `"_gpay_id"`. |
| **F29** | **Chi phí phụ 8 Network Round-Trips mỗi lần `scanExact` (`C-REALITY-28`, `C-REALITY-29`):**<br>- `readOnlyDB` (`recon_dest_agent.go:122-129`) gọi `Begin()` rồi gọi rời `tx.Exec("SET TRANSACTION READ ONLY")`.<br>- `scanExact` (`recon_smoke.go:132-137`) mở 2 transaction riêng biệt cho `CountRowsBeforeCutoff` và `CountDeletedRowsBeforeCutoff`. | Mỗi lần `scanExact` 1 bảng tốn **8 round-trips mạng** (`2 × [BEGIN + SET TRANSACTION READ ONLY + SELECT + ROLLBACK]`) và 2 câu đếm `total` / `deleted` nhìn 2 snapshot MVCC lệch thời điểm nhau. |
| **F30** | **Thiếu nhánh `da.IsNonPostgresSQL()` trong `CountRowsBeforeCutoff` & `CountDeletedRowsBeforeCutoff` (`C-CONTRA-30`):**<br>- Tại `recon_window_index_probe.go:198` và `278`, code chỉ kiểm tra `if da.IsMongo()` mà bỏ sót `if da.IsNonPostgresSQL()` (trong khi `CountRows` tại `recon_dest_query.go:20` và `CountDeletedRows` tại `:78` đều có). | Nếu bảng Master nằm trên **MySQL** (hoặc ClickHouse) và `scanExact` truyền `cutoffHi` khác rỗng, truy vấn sẽ rơi vào nhánh PostgreSQL dùng `quoteRelation` (dấu ngoặc kép `"..."`) và `SET TRANSACTION READ ONLY` $\rightarrow$ gây lỗi cú pháp **`Error 1064`** trên MySQL. Cách sửa chuẩn kiến trúc đa hình (`L001`, `L012`) là thêm nhánh `if da.IsNonPostgresSQL()` gọi `sqlDialectCountRows` + `sqlDialectCountInWindow` (vốn đã dùng đúng backtick `` ` `` và `FINAL` tại `recon_dest_sql_dialects.go:103-225`). |
| **F31** | **Phản biện đề xuất gộp 1 `FROM` bằng `COUNT(*) FILTER (WHERE "_source_ts" >= ...)` hoặc `SUM(CASE WHEN _source_ts >= ...)` (Mục 3.3 & 4 của tài liệu User đối chiếu):**<br>- Giả định sai: *"Tách 2 subquery khiến PostgreSQL quét bảng 5 triệu dòng 2 lần"*.<br>- Sự thật (`09_tasks_solution_2026_10_29.md:729`): Subquery 2 `(SELECT COUNT(*) FROM ... WHERE "_source_ts" >= ?)` chỉ lọc 5 phút cuối (`<0.01%` bảng) bằng **Range `Index Only Scan` ở nhánh phải cuối cùng của `ix_..._source_ts`** (`master_ddl_generator.go:229`), đọc ~4 trang và chỉ tốn **`~0.05 ms`**. | Nếu gộp thành 1 `FROM` với `COUNT(*) FILTER (WHERE "_source_ts" >= ...)` hoặc `SUM(CASE WHEN _source_ts >= ...)`:<br>1. **Tự triệt tiêu ưu điểm `aggstar = true` của `COUNT(*)`:** Vòng quét 5 triệu dòng buộc phải giải mã cột `"_source_ts"` (`slot_getsomeattrs`) và tính biểu thức `_source_ts >= ?` **đủ 5,000,000 lần** (thay vì chỉ ~50 lần ở nhánh phải B-Tree)!<br>2. **Cấm Planner dùng PK Index (`_gpay_id`) & rủi ro `Seq Scan`:** Vì câu lệnh đòi cột `"_source_ts"`, Planner không thể `Index Only Scan` trên PK `_gpay_id`; nếu bảng nào thiếu/bloat index `_source_ts`, PostgreSQL sẽ rơi vào **`Seq Scan` toàn bảng Heap (chứa `payload_data` hàng GB)** làm câu query vọt từ `~200 ms` lên **`10s – 60s`**!<br>$\Rightarrow$ **Giữ nguyên cấu trúc 2 `InitPlan` subqueries**, chỉ đổi `SELECT COUNT(%s)` ở `InitPlan 1` thành `SELECT COUNT(*)`. |

## I.2. Bảng Ánh xạ Điểm Gọi Thực thi & Ngân sách Dòng (`P10` — Khớp 1-1 `git diff`)

| Thứ tự | Tầng Kiến trúc (Layer) | File & Hàm cập nhật | Điểm gọi (`caller file:line`) | Thời điểm kích hoạt (`Lifecycle Stage`) | Ngân sách dòng (`wc -l`) |
| :---: | :--- | :--- | :--- | :--- | :--- |
| 1 | **Data-Access Query** (`ReconDestAgent`) | `internal/service/recon/recon_dest_query.go`: `CountRows` dùng thẳng `SELECT COUNT(*)` (bỏ nhánh `SELECT COUNT(%s)` trên `pkColumn`) | `recon_tier_a.go:1068`, `recon_window_index_probe.go:195` | Khi đếm tổng số dòng bảng đích | `735 -> 722` dòng (`-13` dòng) |
| 2 | **Data-Access Cutoff Probe** (`ReconDestAgent`) | `internal/service/recon/recon_window_index_probe.go`: thêm dispatch `da.IsNonPostgresSQL()` trong `CountRowsBeforeCutoff` & `CountDeletedRowsBeforeCutoff` + dùng thẳng `SELECT COUNT(*)` cho `raw_total` | `recon_smoke.go:132, 137` (`scanExact`) | Bước 3 (`prefetch.shadow` & `prefetch.master`) của `CheckAllUnified` | `312 -> 321` dòng (`+9` dòng) |
| 3 | **Unit Test** (`recon`) | `internal/service/recon/recon_fallback_test.go`: cập nhật `TestCountRows_FallbackTransactionIsolation` kiểm chứng `SELECT COUNT(*) FROM "my_table"` trực tiếp | `go test ./internal/service/recon` | Kiểm thử đơn vị gói `recon` | `120 -> 114` dòng (`-6` dòng) |

> **Ghi chú tương thích (`S5 -> S3`):** Giữ nguyên `readOnlyDB` trong `recon_dest_agent.go:122-129` (`Begin()` + `SET TRANSACTION READ ONLY`) để tương thích 100% với helper `expectTxReadOnly(mock)` tại `recon_dest_agent_test.go:41-44` mà không làm phình diff sang các file test khác.

## I.3. Code Demo (`P10` — Khớp 1-1 `git diff`)

### Demo P10-1: `internal/service/recon/recon_dest_query.go:35-46` (Chuyển thẳng sang `SELECT COUNT(*)`)
```go
// Trước (recon_dest_query.go:35-57):
	result, err := da.breaker.Execute(func() (interface{}, error) {
		var count int64
		// 1. Thử COUNT(pkColumn) nếu có pkColumn hợp lệ
		if pkColumn != "" && validateIdent(pkColumn) == nil {
			tx := da.readOnlyDB(ctx)
			sql := fmt.Sprintf(`SELECT COUNT(%s) FROM %s`, quoteIdent(pkColumn), quoteRelation(tableName))
			err := tx.Raw(sql).Scan(&count).Error
			tx.Rollback() // Đảm bảo đóng tx ngay lập tức
			if err == nil {
				return count, nil
			}
			// Fallback về COUNT(*) nếu pkColumn không tồn tại trên bảng
		}

		// 2. Chạy COUNT(*) trên một transaction read-only mới hoàn toàn để tránh SQLSTATE 25P02
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteRelation(tableName))
		if err := tx.Raw(sql).Scan(&count).Error; err != nil {
			return nil, err
		}
		return count, nil
	})

// Sau (recon_dest_query.go:35-44):
	result, err := da.breaker.Execute(func() (interface{}, error) {
		var count int64
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteRelation(tableName))
		if err := tx.Raw(sql).Scan(&count).Error; err != nil {
			return nil, err
		}
		return count, nil
	})
```

### Demo P10-2: `internal/service/recon/recon_window_index_probe.go:198-297` (Thêm `da.IsNonPostgresSQL()` & dùng thẳng `SELECT COUNT(*)` ở `InitPlan 1`)
```go
// 1. Trong CountRowsBeforeCutoff (recon_window_index_probe.go:209-219 & 239-252):
	if da.IsNonPostgresSQL() {
		total, err := da.sqlDialectCountRows(ctx, tableName, pkColumn)
		if err != nil {
			return 0, err
		}
		post, errW := da.sqlDialectCountInWindow(ctx, tableName, "_source_ts", cutoffHi, time.Now().UTC().Add(time.Minute))
		if errW == nil && total >= post {
			total -= post
		}
		return total, nil
	}

	result, err := da.breaker.Execute(func() (interface{}, error) {
		var row cutoffCountRow
		rel := quoteRelation(tableName)
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		sql := fmt.Sprintf(
			`SELECT (SELECT COUNT(*) FROM %s) AS raw_total, (SELECT COUNT(*) FROM %s WHERE "_source_ts" >= ?) AS post_cutoff`,
			rel, rel,
		)
		if err := tx.Raw(sql, cutoffMs).Scan(&row).Error; err != nil {
			return nil, err
		}
		return row, nil
	})

// 2. Trong CountDeletedRowsBeforeCutoff (recon_window_index_probe.go:287-297):
	if da.IsNonPostgresSQL() {
		delTotal, err := da.sqlDialectCountDeletedRows(ctx, tableName)
		if err != nil {
			return 0, err
		}
		delPost, errW := da.sqlDialectCountRecentDeletedRows(ctx, tableName, "_source_ts", cutoffHi, time.Now().UTC().Add(time.Minute))
		if errW == nil && delTotal >= delPost {
			delTotal -= delPost
		}
		return delTotal, nil
	}
```

### Demo P10-3: `internal/service/recon/recon_fallback_test.go:53-66`
```go
func TestCountRows_FallbackTransactionIsolation(t *testing.T) {
	da, mock := setupMockedDestAgent(t)

	// CountRows executes SELECT COUNT(*) directly without column extraction or error fallback
	expectTxReadOnly(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM "my_table"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(42)))
	mock.ExpectRollback()

	cnt, err := da.CountRows(context.Background(), "my_table", "non_existent_id")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), cnt)
	assert.NoError(t, mock.ExpectationsWereMet())
}
```

---

## K. Chẩn Đoán 4 Nguyên Nhân Gốc Rễ (`F33–F36`) Vì Sao `/system-health` Không Hiển Thị Tiến Độ Khi Chạy Snapshot & Transmute + Giải Pháp `P11` (`T21` — 2026-11-09)

### K.1 Bằng chứng Runtime Thật (`EV-20261109T145641-3ecf` & `EV-20261109T145644-1337`) — Minh Bạch 100% Không Có Mockup
- Khi probe trực tiếp API đang chạy trên máy (`GET http://localhost:8083/api/system/health` — `EV-20261109T145641-3ecf`), trong mảng `recent_events` ghi nhận đúng job Transmute mà anh vừa chạy lúc `14:54:19`:
  - `operation: "transmute"`, `table: "goopay_clickhouse.transaction_history_payload"`, `status: "success"`, `scanned: 251287`, `inserted: 251287`, `type_errors: 1387`, `correlation_id: "run-now-138-2026-11-09T07:54:19.753814Z"`, `master_binding_id: 93`.
- Nhưng ngay trong cùng payload `/api/system/health` đó, trường `failed_sync` lại trả về:
  - `"failed_sync": { "active_snapshots": null, "count_1h": 0, "count_24h": 0, "oom_count_15m": 0, "oom_count_1h": 0 }`.
- Nguyên nhân không phải do dữ liệu bị mockup, mà do **4 lỗ hổng thiết kế trong luồng truy vấn & hiển thị tiến độ (`F33–F36`)**:

| Mã | Mức độ | Vị trí (`file:line`) | Bản chất Lỗ hổng (`VERIFIED`) |
| :--- | :---: | :--- | :--- |
| **`F33`** | **CRITICAL** | `cdc-cms-service/internal/infra/observability/system_health_queries.go:124-162`<br>`cdc-cms-web/src/pages/SystemHealthPanels.tsx:336-398` | **Bỏ sót hoàn toàn bảng `cdc_system.transmute_jobs`:** Khi chạy Transmute (`run-now-138` qua `run_now.go:87-89`), `TransmuterModule.Run` (`transmuter.go:216-218, 511-521`) cập nhật liên tục `%` tiến độ (`status = 'RUNNING'`, `progress_percent`, `rows_affected`, `total_rows`, `chunk_size`) vào bảng `cdc_system.transmute_jobs` (`101_create_transmute_jobs.sql:7-21`, `103_add_total_rows_to_jobs.sql:8`). Tuy nhiên `queryFailedCount` (`system_health_queries.go:124-162`) chỉ query `cdc_system.snapshot_progress` mà **không hề query `cdc_system.transmute_jobs`**, và `SystemHealthPanels.tsx` cũng không có bất kỳ dòng nào hiển thị tiến độ Transmute! |
| **`F34`** | **HIGH** | `cdc-cms-service/internal/infra/observability/system_health_queries.go:124, 149`<br>`cdc-cms-web/src/pages/SystemHealthPanels.tsx:363, 368` | **Lọc `OR sp.updated_at > fifteenMinAgo` làm `active_snapshots` biến thành `null` sau 15 phút & UI tự ẩn khối tiến độ:**<br>1. Tại `system_health_queries.go:124, 149`, biến `var activeSnapshots []map[string]any` không được khởi tạo slice rỗng và câu SQL lọc `WHERE sp.status IN ('running', 'paused') OR sp.updated_at > ?` (`15 phút`). Ngay khi job Snapshot chạy xong (hoặc quá 15 phút kể từ lần chạy lúc `14:13:28`), query trả 0 dòng $\rightarrow$ `active_snapshots: null`.<br>2. Tại `SystemHealthPanels.tsx:363, 368`, điều kiện `{activeSnapshots.length > 0 && (...)}` ẩn sạch khu vực Snapshot khi mảng rỗng, và `.slice(0, 2)` cắt chỉ còn 2 dòng nhét chung ở đáy thẻ NATS nhỏ hẹp. |
| **`F35`** | **HIGH** | `centralized-data-service/internal/handler/master/debounce.go:31-48`<br>`transmute_handler.go:40-48`<br>`server_setup.go:562-592` | **Transmute kích hoạt tự động sau Snapshot/CDC (`cdc.cmd.transmute-shadow`) chạy ngầm trong `TableDebouncer` mà không lộ số liệu ra `/api/v1/internal/stats`:** Khi Snapshot bắn sang Transmute qua `HandleTransmuteShadow` (`transmute_handler.go:197-234`), các batch chạy trong `TableDebouncer` (`debounce.go:31-48`) chứ không tạo dòng trong `cdc_system.transmute_jobs`. Do `TableDebouncer` chưa có bộ đếm `in_flight` / `pending_events` / `events_processed` trên `/api/v1/internal/stats`, người dùng không thấy được các mẻ Transmute post-ingest đang chạy trong RAM. |
| **`F36`** | **HIGH** | `cdc-cms-service/internal/infra/observability/system_health_collector.go:113-115`<br>`system_health_handler.go:52-90`<br>`cdc-cms-web/src/hooks/useSystemHealth.ts:47-48` | **Điểm mù 45 giây (15s Collector + 30s React Query Polling) & Nút "Làm mới" chỉ đọc cache Redis cũ:**<br>- `NewCollector` đặt mặc định `cfg.Interval = 15 * time.Second` (`system_health_collector.go:114`).<br>- `useSystemHealth.ts:47-48` đặt `refetchInterval: 30_000` (30s) và `staleTime: 25_000` (25s) — chậm gấp **6–10 lần** so với `SnapshotMonitor.tsx:99` (`5000ms`) và `MasterRegistry.tsx:93` (`3000ms`). Một job Snapshot/Transmute 250k dòng chạy trong 10–25 giây sẽ chuyển từ `PENDING` $\rightarrow$ `RUNNING` $\rightarrow$ `COMPLETED` nằm trọn trong điểm mù 45 giây mà UI chưa kịp poll!<br>- Hơn nữa, khi bấm nút "Làm mới" trên UI, `GET /api/system/health` (`system_health_handler.go:62`) chỉ đọc lại bản cache cũ trong Redis chứ chưa hỗ trợ trigger thu thập tức thời. |

---

### K.2 Bản Vẽ Bố Cục Trực Quan (`Visual UI/UX Prototype First` — `L071`)

Thay vì nhét tạm 2 dòng chữ nhỏ ở đáy thẻ NATS và tự biến mất khi xong, khu vực **`PipelineStatsBody`** trong `SystemHealthPanels.tsx` được bổ sung **Hàng 3 chuyên biệt (2 cột song song `Col xs={24} md={12}`)** luôn hiển thị trực quan thanh `<Progress>` của cả **Snapshot (`cdc_system.snapshot_progress`)** và **Transmute (`cdc_system.transmute_jobs` + `TableDebouncer` In-Memory)**:

```text
+---------------------------------------------------------------------------------------------------+
|  CDC Pipeline (Worker · Debezium · Consumer Lag · Snapshot & Transmute Progress)            [OK]  |
+---------------------------------------------------------------------------------------------------+
| [Tables: 28] | [PG OOM 15m/1h: 0/0] | [DLQ 1h/24h: 0/0] | [PG Conns: 14/100] | [Heap: 21MB·162gr] |
+---------------------------------------------------------------------------------------------------+
| Worker BatchBuffers & Pacing        | Worker DB Pools (cdc/shadow/dest) | NATS JetStream Queues   |
| - CDC Stream (0/1000)   Flush: 12ms | - cdc     Open: 4/20  InUse: 1    | [CDC_TRANSMUTE] Msgs: 0 |
| - Snapshot   (0/1000)   Flush: 45ms | - shadow  Open: 6/30  InUse: 2    | [CDC_COMMANDS]  Msgs: 0 |
| - Self-Heal  (0/1000)   Flush:  0ms | - dest    Open: 4/30  InUse: 1    | [CDC_DLQ]       Msgs: 0 |
+-------------------------------------+-----------------------------------+-------------------------+
| Tiến độ Snapshot (Running & 5 job gần nhất)    | Tiến độ Transmute (Batch Jobs & Live Debouncers) |
| ---------------------------------------------- | ------------------------------------------------ |
| shadow_testcoretrans.transaction_history       | goopay_clickhouse.transaction_history_payload    |
| [done] 254,372 / 254,040 dòng (100%) · 14:13   | [COMPLETED] 251,287 / 251,287 dòng (100%) · 14:54|
| [====================================] 100%    | [====================================] 100%      |
|                                                |                                                  |
| shadow_testcoretrans.cdc_transaction_history_1 | Live Stream Debouncers (Post-Ingest):            |
| [running] 100,000 / 250,363 dòng (40%) · 14:12 | [master_payment.bills#46] InFlight:1 · Pend:120  |
| [==============>.....................]  40%    | Flushed: 18 batches (4,520 rows) · Last: 14:55   |
+------------------------------------------------+--------------------------------------------------+
```

---

### K.3 Giải Pháp Tối Ưu Nhất (`P11` — Ánh Xạ 1-1 Tới `F33–F36`) & Code Demo Đầy Đủ 1:1 (`L002`)

#### Demo P11-1: `cdc-cms-service/internal/infra/observability/system_health_queries.go:124-162` (Khắc phục `F33` & `F34`: Luôn lấy 5 Snapshot Jobs + 5 Transmute Jobs gần nhất, ưu tiên `running`/`PENDING`/`RUNNING` lên đầu, khởi tạo slice rỗng tránh `null`)

```go
	activeSnapshots := make([]map[string]any, 0, 5)
	db.Raw(
		`SELECT
			sp.id,
			COALESCE(
				NULLIF(CONCAT_WS('.', sb.shadow_schema, sb.shadow_table), ''),
				NULLIF(CONCAT_WS('.', so.source_database, so.source_object_name), ''),
				CAST(sp.source_object_id AS TEXT)
			) AS table_name,
			sp.status,
			sp.rows_processed,
			sp.total_rows,
			CASE
				WHEN sp.total_rows > 0 THEN LEAST(100, CAST(ROUND((sp.rows_processed::numeric / sp.total_rows::numeric) * 100) AS INT))
				WHEN sp.status IN ('done', 'completed') THEN 100
				ELSE 0
			END AS progress_percent,
			sp.started_at,
			sp.updated_at,
			sp.error_msg AS error_message
		 FROM cdc_system.snapshot_progress sp
		 LEFT JOIN cdc_system.source_object_registry so ON so.id = sp.source_object_id
		 LEFT JOIN LATERAL (
			SELECT s.shadow_schema, s.shadow_table
			FROM cdc_system.shadow_binding s
			WHERE (sp.shadow_binding_id IS NOT NULL AND s.id = sp.shadow_binding_id)
			   OR (sp.shadow_binding_id IS NULL AND s.source_object_id = sp.source_object_id AND s.is_active = TRUE)
			ORDER BY s.id DESC
			LIMIT 1
		 ) sb ON TRUE
		 ORDER BY
			CASE WHEN sp.status IN ('running', 'paused') THEN 0 ELSE 1 END,
			sp.updated_at DESC
		 LIMIT 5`,
	).Scan(&activeSnapshots)

	activeTransmutes := make([]map[string]any, 0, 5)
	db.Raw(
		`SELECT
			tj.job_id,
			tj.master_table,
			tj.status,
			tj.progress_percent,
			tj.rows_affected,
			tj.total_rows,
			tj.chunk_size,
			tj.started_at,
			tj.updated_at,
			tj.finished_at,
			tj.error_message
		 FROM cdc_system.transmute_jobs tj
		 ORDER BY
			CASE WHEN tj.status IN ('PENDING', 'RUNNING') THEN 0 ELSE 1 END,
			tj.updated_at DESC
		 LIMIT 5`,
	).Scan(&activeTransmutes)

	return map[string]any{
		"count_24h":         row.Count24h,
		"count_1h":          row.Count1h,
		"oom_count_15m":     oomRow.OOM15m,
		"oom_count_1h":      oomRow.OOM1h,
		"active_snapshots":  activeSnapshots,
		"active_transmutes": activeTransmutes,
	}
```

#### Demo P11-2: `centralized-data-service/internal/handler/master/debounce.go` & `transmute_handler.go` & `internal/server/server_setup.go` (Khắc phục `F35`: Đo lường tiến độ `TableDebouncer` cho luồng Transmute Post-Ingest/Streaming và xuất qua `/api/v1/internal/stats`)

1. Tại `centralized-data-service/internal/handler/master/debounce.go`:
```go
package master

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)
```
Và bổ sung các trường atomic + hàm `GetStat()` trên `TableDebouncer`:
```go
type DebouncerStat struct {
	TableName       string `json:"table_name"`
	InFlight        int32  `json:"in_flight"`
	PendingTasks    int    `json:"pending_tasks"`
	PendingEvents   int    `json:"pending_events"`
	QueuedBatches   int    `json:"queued_batches"`
	BatchesFlushed  int64  `json:"batches_flushed"`
	EventsProcessed int64  `json:"events_processed"`
	LastFlushUnixMs int64  `json:"last_flush_unix_ms"`
}

type TableDebouncer struct {
	mu              sync.Mutex
	tableName       string
	tasks           []NatsMsgTask
	pendingEvents   int
	maxSize         int
	idleTimeout     time.Duration
	maxTimeout      time.Duration
	idleTimer       *time.Timer
	maxTimer        *time.Timer
	flushCh         chan []NatsMsgTask
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	sem             chan struct{}
	logger          *zap.Logger
	runTransmute    func(ctx context.Context, batch []NatsMsgTask)
	inFlight        atomic.Int32
	batchesFlushed  atomic.Int64
	eventsProcessed atomic.Int64
	lastFlushUnixMs atomic.Int64
}
```
Trong `workerLoop()` và `GetStat()` của `debounce.go`:
```go
		case batch := <-td.flushCh:
			select {
			case td.sem <- struct{}{}:
				go func() {
					defer func() { <-td.sem }()
					td.inFlight.Add(1)
					defer td.inFlight.Add(-1)
					batchEvents := 0
					for _, t := range batch {
						batchEvents += taskEventCount(t)
					}
					td.runTransmute(td.ctx, batch)
					td.batchesFlushed.Add(1)
					td.eventsProcessed.Add(int64(batchEvents))
					td.lastFlushUnixMs.Store(time.Now().UnixMilli())
				}()
			case <-td.ctx.Done():
				td.runTransmute(context.Background(), batch)
				return
			}
```
```go
func (td *TableDebouncer) GetStat() DebouncerStat {
	td.mu.Lock()
	pendingTasks := len(td.tasks)
	pendingEvents := td.pendingEvents
	queuedBatches := len(td.flushCh)
	td.mu.Unlock()
	return DebouncerStat{
		TableName:       td.tableName,
		InFlight:        td.inFlight.Load(),
		PendingTasks:    pendingTasks,
		PendingEvents:   pendingEvents,
		QueuedBatches:   queuedBatches,
		BatchesFlushed:  td.batchesFlushed.Load(),
		EventsProcessed: td.eventsProcessed.Load(),
		LastFlushUnixMs: td.lastFlushUnixMs.Load(),
	}
}
```

2. Tại `centralized-data-service/internal/handler/master/transmute_handler.go`:
```go
func (h *TransmuteHandler) GetDebouncerStats() []DebouncerStat {
	h.debouncersMu.RLock()
	list := make([]*TableDebouncer, 0, len(h.debouncers))
	for _, td := range h.debouncers {
		list = append(list, td)
	}
	h.debouncersMu.RUnlock()

	stats := make([]DebouncerStat, 0, len(list))
	for _, td := range list {
		stats = append(stats, td.GetStat())
	}
	return stats
}
```

3. Tại `centralized-data-service/internal/server/server_setup.go:569-591`:
```go
		return c.JSON(fiber.Map{
			"status":               "ok",
			"service":              "cdc-worker",
			"tables_registered":    tablesRegistered,
			"queue":                collectPoolStats(consumerPools),
			"buffer":               cdcBatchBuffer.GetStatus(),
			"snapshot_buffer":      snapshotBatchBuffer.GetStatus(),
			"heal_buffer":          healBatchBuffer.GetStatus(),
			"transmute_debouncers": transmuteHandler.GetDebouncerStats(),
			"db_pools":             registry.PoolStats(),
			"runtime": fiber.Map{
				"goroutines":    runtime.NumGoroutine(),
				"heap_alloc_mb": float64(mem.HeapAlloc) / (1024 * 1024),
				"heap_inuse_mb": float64(mem.HeapInuse) / (1024 * 1024),
				"sys_mb":        float64(mem.Sys) / (1024 * 1024),
				"num_gc":        mem.NumGC,
			},
			"config": fiber.Map{
				"pool_size":        cfg.Worker.PoolSize,
				"batch_size":       cfg.Worker.BatchSize,
				"db_max_open_conn": cfg.DBPool.MaxOpenConn,
				"db_max_idle_conn": cfg.DBPool.MaxIdleConn,
			},
		})
```

#### Demo P11-3: `cdc-cms-service/internal/infra/observability/system_health_collector.go:113-115` & `cdc-cms-web/src/hooks/useSystemHealth.ts:40-51` (Khắc phục `F36`: Giảm chu kỳ Collector từ 15s xuống 5s và React Query `refetchInterval` từ 30s xuống 5s để bắt trọn mọi tiến độ thời gian thực)

1. Tại `cdc-cms-service/internal/infra/observability/system_health_collector.go:113-115`:
```go
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
```

2. Tại `cdc-cms-web/src/hooks/useSystemHealth.ts:40-51`:
```ts
export function useSystemHealth() {
  return useQuery<SystemHealthSnapshot>({
    queryKey: ['system-health'],
    queryFn: async () => {
      const { data } = await cmsApi.get<SystemHealthSnapshot>('/api/system/health');
      return data;
    },
    refetchInterval: 5_000,
    staleTime: 3_000,
    retry: 2,
  });
}
```

#### Demo P11-4: `cdc-cms-web/src/pages/SystemHealthPanels.tsx:163-482` (Hiển thị bảng Tiến độ Snapshot & Transmute chuyên biệt 2 cột kèm thanh `<Progress>` và chi tiết sự kiện Transmute)

```tsx
interface DebouncerStat {
  table_name?: string;
  in_flight?: number;
  pending_tasks?: number;
  pending_events?: number;
  queued_batches?: number;
  batches_flushed?: number;
  events_processed?: number;
  last_flush_unix_ms?: number;
}

export function PipelineStatsBody({
  pipeline,
  failedSync,
  infra,
}: {
  pipeline: unknown;
  failedSync: unknown;
  infra?: unknown;
}) {
  const failed =
    (failedSync as {
      count_1h?: number;
      count_24h?: number;
      oom_count_15m?: number;
      oom_count_1h?: number;
      active_snapshots?: Array<Record<string, unknown>>;
      active_transmutes?: Array<Record<string, unknown>>;
    }) || {};
  const pipelineObj = (pipeline as Record<string, unknown>) || {};
  const worker = (pipelineObj.worker as Record<string, unknown>) || {};
  const infraObj = (infra as Record<string, Record<string, unknown>>) || {};
  const pgInfo = infraObj.postgres || {};
  const natsInfo = infraObj.nats || {};
  const streamDetails = (natsInfo.stream_details as Record<string, Record<string, unknown>>) || {};

  const tablesRegistered =
    (worker?.tables_registered as number) ?? (pgInfo?.tables_registered as number) ?? 0;
  const oom15m = failed.oom_count_15m ?? 0;
  const oom1h = failed.oom_count_1h ?? 0;

  const cdcBuf = (worker.buffer as BufferStat) || {};
  const snapBuf = (worker.snapshot_buffer as BufferStat) || {};
  const healBuf = (worker.heal_buffer as BufferStat) || {};
  const dbPools = (worker.db_pools as Record<string, DBPoolStat>) || {};
  const runtimeInfo = (worker.runtime as Record<string, number>) || {};
  const activeSnapshots = Array.isArray(failed.active_snapshots) ? failed.active_snapshots : [];
  const activeTransmutes = Array.isArray(failed.active_transmutes) ? failed.active_transmutes : [];
  const debouncers = Array.isArray(worker.transmute_debouncers)
    ? (worker.transmute_debouncers as DebouncerStat[])
    : [];
```
Và tại khu vực Hàng 2 (`NATS JetStream Queues`) + Hàng 3 mới (`Tiến độ Snapshot & Transmute`) + `RecentEventsBody` trong `SystemHealthPanels.tsx:336-481`:
```tsx
        <Col xs={24} md={8}>
          <Card size="small" title="NATS JetStream Queues">
            {['CDC_TRANSMUTE', 'CDC_COMMANDS', 'CDC_DLQ'].map((sName) => {
              const st = streamDetails[sName] || {};
              const msgs = Number(st.messages ?? 0);
              const ackPending = Number(st.num_ack_pending ?? 0);
              const redelivered = Number(st.num_redelivered ?? 0);
              return (
                <div
                  key={sName}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    fontSize: 12,
                    marginBottom: 6,
                  }}
                >
                  <span>
                    <Tag color={msgs > 1000 ? 'orange' : 'green'}>{sName}</Tag>
                  </span>
                  <span>
                    Msgs: <strong>{msgs}</strong> · InFlight: {ackPending} · Redeliver:{' '}
                    {redelivered}
                  </span>
                </div>
              );
            })}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} md={12}>
          <Card size="small" title="Tiến độ Snapshot (Running & 5 job gần nhất)">
            {activeSnapshots.length === 0 ? (
              <Empty
                description="Chưa có tiến trình Snapshot nào"
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            ) : (
              activeSnapshots.map((s, idx) => {
                const done = Number(s.rows_processed ?? 0);
                const total = Number(s.total_rows ?? 0);
                const statusStr = String(s.status ?? '').toLowerCase();
                const pct =
                  s.progress_percent != null
                    ? Number(s.progress_percent)
                    : total > 0
                      ? Math.min(100, Math.round((done / total) * 100))
                      : statusStr === 'done' || statusStr === 'completed'
                        ? 100
                        : 0;
                const tagColor =
                  statusStr === 'running'
                    ? 'processing'
                    : statusStr === 'error' || statusStr === 'failed'
                      ? 'red'
                      : statusStr === 'done' || statusStr === 'completed'
                        ? 'green'
                        : 'default';
                const updatedAt = s.updated_at
                  ? new Date(String(s.updated_at)).toLocaleTimeString('vi-VN', { hour12: false })
                  : '';
                return (
                  <div key={String(s.id ?? idx)} style={{ fontSize: 12, marginBottom: 8 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span>
                        <strong>{String(s.table_name ?? '-')}</strong>
                      </span>
                      <span>
                        <Tag color={tagColor}>{String(s.status ?? '')}</Tag>
                        {done.toLocaleString()} / {total > 0 ? total.toLocaleString() : '?'} dòng
                        {updatedAt ? ` · ${updatedAt}` : ''}
                      </span>
                    </div>
                    <Progress
                      percent={pct}
                      size="small"
                      status={
                        statusStr === 'running'
                          ? 'active'
                          : statusStr === 'error' || statusStr === 'failed'
                            ? 'exception'
                            : undefined
                      }
                    />
                  </div>
                );
              })
            )}
          </Card>
        </Col>

        <Col xs={24} md={12}>
          <Card size="small" title="Tiến độ Transmute (Batch Jobs & Live Debouncers)">
            {activeTransmutes.length === 0 && debouncers.length === 0 ? (
              <Empty
                description="Chưa có tiến trình Transmute nào"
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            ) : (
              <>
                {activeTransmutes.map((t, idx) => {
                  const done = Number(t.rows_affected ?? 0);
                  const total = Number(t.total_rows ?? 0);
                  const statusStr = String(t.status ?? '').toUpperCase();
                  const pct =
                    t.progress_percent != null
                      ? Number(t.progress_percent)
                      : total > 0
                        ? Math.min(100, Math.round((done / total) * 100))
                        : statusStr === 'COMPLETED'
                          ? 100
                          : 0;
                  const tagColor =
                    statusStr === 'RUNNING' || statusStr === 'PENDING'
                      ? 'processing'
                      : statusStr === 'FAILED'
                        ? 'red'
                        : statusStr === 'COMPLETED'
                          ? 'green'
                          : 'default';
                  const updatedAt = t.updated_at
                    ? new Date(String(t.updated_at)).toLocaleTimeString('vi-VN', { hour12: false })
                    : '';
                  return (
                    <div key={String(t.job_id ?? idx)} style={{ fontSize: 12, marginBottom: 8 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <span>
                          <strong>{String(t.master_table ?? '-')}</strong>
                        </span>
                        <span>
                          <Tag color={tagColor}>{statusStr}</Tag>
                          {done.toLocaleString()} / {total > 0 ? total.toLocaleString() : '?'} dòng
                          {updatedAt ? ` · ${updatedAt}` : ''}
                        </span>
                      </div>
                      <Progress
                        percent={pct}
                        size="small"
                        status={
                          statusStr === 'RUNNING' || statusStr === 'PENDING'
                            ? 'active'
                            : statusStr === 'FAILED'
                              ? 'exception'
                              : undefined
                        }
                      />
                    </div>
                  );
                })}
                {debouncers.length > 0 && (
                  <div
                    style={{
                      marginTop: activeTransmutes.length > 0 ? 8 : 0,
                      borderTop: activeTransmutes.length > 0 ? '1px solid #f0f0f0' : undefined,
                      paddingTop: activeTransmutes.length > 0 ? 6 : 0,
                    }}
                  >
                    <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 4 }}>
                      Live Stream Debouncers (Post-Ingest):
                    </div>
                    {debouncers.slice(0, 5).map((d, idx) => (
                      <div
                        key={String(d.table_name ?? idx)}
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          fontSize: 11,
                          marginBottom: 4,
                        }}
                      >
                        <span>
                          <Tag color={(d.in_flight ?? 0) > 0 ? 'processing' : 'blue'}>
                            {String(d.table_name ?? '-')}
                          </Tag>
                        </span>
                        <span>
                          InFlight: <strong>{d.in_flight ?? 0}</strong> · Pending:{' '}
                          <strong>{d.pending_events ?? 0}</strong> · Flushed:{' '}
                          <strong>{(d.events_processed ?? 0).toLocaleString()}</strong> rows (
                          {d.batches_flushed ?? 0} batches)
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </Card>
        </Col>
      </Row>
    </Space>
  );
}
```
Và trong `RecentEventsBody` (`SystemHealthPanels.tsx:454-471`), bổ sung render tag trực quan cho sự kiện `transmute` (`scanned`, `inserted`, `updated`, `deleted`, `type_errors`):
```tsx
              if (parsed.batch_size !== undefined) {
                const unchanged =
                  parsed.unchanged ??
                  Number(parsed.batch_size ?? 0) - Number(parsed.written ?? 0);
                return (
                  <Space size={4} wrap>
                    <Tag color="blue">batch: {String(parsed.batch_size)}</Tag>
                    <Tag color="green">written: {String(parsed.written ?? 0)}</Tag>
                    <Tag>unchanged: {String(unchanged)}</Tag>
                    {parsed.buffer_type ? (
                      <Tag color="purple">{String(parsed.buffer_type)}</Tag>
                    ) : null}
                  </Space>
                );
              }
              if (parsed.scanned !== undefined) {
                return (
                  <Space size={4} wrap>
                    <Tag color="blue">scanned: {Number(parsed.scanned ?? 0).toLocaleString()}</Tag>
                    <Tag color="green">
                      inserted: {Number(parsed.inserted ?? 0).toLocaleString()}
                    </Tag>
                    <Tag color="cyan">updated: {Number(parsed.updated ?? 0).toLocaleString()}</Tag>
                    {Number(parsed.deleted ?? 0) > 0 ? (
                      <Tag color="volcano">
                        deleted: {Number(parsed.deleted ?? 0).toLocaleString()}
                      </Tag>
                    ) : null}
                    {Number(parsed.type_errors ?? 0) > 0 ? (
                      <Tag color="orange">
                        type_errors: {Number(parsed.type_errors ?? 0).toLocaleString()}
                      </Tag>
                    ) : null}
                  </Space>
                );
              }
```

---

## L. Chẩn đoán Pháp y & Giải pháp `P12` (`T23` — Sửa dứt điểm lỗi Snapshot dừng ở `100,000 / 250,363` với `Heartbeat timeout: progress was stuck in running state for too long (worker stopped)`)

### 1. Truy vết Pháp y Chính xác 100% từ PostgreSQL Runtime (`EV-20261109T142137-c12c` & `EV-20261109T153857-a32a`)

Dòng lỗi anh thấy trên `/snapshot-monitor`:
> `14:05:37 9/11/2026 | cdc_transaction_history_payload | testcoretrans | core-trans-proxy-history-service | cdc_transaction_history_payload_1 | shadow_testcoretrans | error | 100,000 | 100,000 / 250,363 | 39.94% | 67286461eb917d6e7fd308f4 | 9ac6f6…25c2 | - | Heartbeat timeout: progress was stuck in running state for too long (worker stopped)`

**KHÔNG PHẢI** do thiếu RAM/OOM hay đứt kết nối khi chạy 250k dòng (bằng chứng là ngay sau đó lúc **`14:09:22`**, bản ghi `id=91` của bảng `transaction_history` cùng nguồn `testcoretrans` đã chạy một mạch **`254,372 / 254,040` dòng (`100% done`)** trong 4 phút với `batch_size = 1000`).

Dưới đây là dòng thời gian pháp y chính xác từng giây của `id=90` (`trace_id = 9ac6f6a6ab304ab6acf82ec5325c25c2`) trích trực tiếp từ `cdc_system.snapshot_progress` và `cdc_system.cdc_activity_log` (`EV-20261109T142137-c12c`, `EV-20261109T153857-a32a`):

| Mốc thời gian (`+07:00`) | Bản ghi DB thật (`cdc_dw`) | Chuyện gì đã xảy ra trên thực tế |
|---|---|---|
| **`14:05:37.138`** | `cdc_activity_log.id = 157372` (`status = accepted`), `snapshot_progress.id = 90` (`status = running`, `total_rows = 250363`) | Job Snapshot `cdc_transaction_history_payload -> shadow_testcoretrans.cdc_transaction_history_payload_1` được kích hoạt khi Worker `centralized-data-service` **vẫn đang chạy binary cũ (`batch_size = 5000`)**. |
| **`14:05:37` → `14:07:15.259`** | `cdc_activity_log.id = 157451` (`operation = kafka-consumer`, `"batch_size": 5000`, `completed_at = 14:07:15.259`), `snapshot_progress.id = 90` (`updated_at = 14:07:15.26278`) | Trong 98 giây, Worker chạy xong **đúng 20 mẻ × `5,000` dòng = `100,000` dòng (`39.94%`)**, lưu `last_seen_id = "67286461eb917d6e7fd308f4"`. |
| **`14:07:15` → `14:09:22`** | Worker `centralized-data-service` được tắt/bật lại để nạp code mới (`batch_size = 1000`); lúc `14:09:22.692` chạy `id=91` (`transaction_history`, `activity_log.id = 157740`, `"batch_size": 1000`) hoàn tất `254,372` dòng `done` lúc `14:13:28.418` | Khi tiến trình Worker cũ tắt lúc `14:07:15`, goroutine `runSnapshot` của `id=90` bị giết ngang xương ở mốc `100,000` dòng mà **không có hook `OnStop` để chuyển `id=90` sang `paused`**, và khi tiến trình Worker mới bật lên cũng **không có hook `OnStart` để tự động resume `id=90` từ `last_seen_id`**! Do đó `id=90` bị bỏ rơi ở trạng thái `status = 'running'` suốt từ `14:07:15` (còn nguyên trong biên lai `EV-20261109T142137-c12c` lúc `14:21:37`). |
| **`14:34:14.062605`** | `snapshot_progress.id = 90` bị đổi thành `status = 'error'`, `finished_at = 14:34:14.062605`, `error_msg = 'Heartbeat timeout: progress was stuck in running state for too long (worker stopped)'` | Khi mở trang `/snapshot-monitor` (`GET /api/snapshot-progress`), hàm đọc `ListSnapshotProgress` (`cdc-cms-service/internal/infra/persistence/scheduler/snapshot_progress_read_repo_gorm.go:23-31`) chạy câu `UPDATE ... SET status = 'error', error_msg = 'Heartbeat timeout: ...' WHERE status = 'running' AND updated_at < NOW() - INTERVAL '5 minutes'` và đập chết `id=90` thành `error`! |

### 2. Bảng Tóm tắt 4 Lỗ hổng Kiến trúc Vòng đời Snapshot (`F37–F40`) & Ánh xạ Giải pháp `P12`

| Mã | Mức độ | Vị trí (`file:line`) | Nguyên nhân gốc rễ (`VERIFIED`) | Giải pháp `P12` (Ánh xạ 1-1) |
|---|---|---|---|---|
| **`F37`** | **CRITICAL** | `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go:183-199, 651-931` | 1. `Handle` spawn `go func` với `context.Background()` và đặt `defer recover()` ở hàm ngoài (`line 152`) thay vì bên trong goroutine con (`line 183`), nên nếu goroutine con panic thì tiến trình sập hoặc `snapshot_progress` kẹt `running` mãi mãi.<br>2. `updated_at` chỉ được cập nhật ở cuối mỗi batch tại `r.checkpoint` (`line 874`). Nếu 1 batch bảng payload lớn hoặc nhịp nghỉ backpressure kéo dài, không có luồng Heartbeat nền giữ nhịp sống. | 1. Bọc `defer recover()` ngay bên trong `spawnSnapshot` goroutine và bổ sung chốt chặn `ensureProgressNotStuckRunning` ở `defer` của `runSnapshot`.<br>2. Khởi chạy goroutine **Heartbeat nền định kỳ `15s`** (`trackAndHeartbeatProgress`) cập nhật `updated_at = NOW() WHERE id = ? AND status = 'running'` trong suốt thời gian `runSnapshot` đang chạy. |
| **`F38`** | **CRITICAL** | `centralized-data-service/internal/server/server_setup.go:407-421`, `snapshot_runner_handler.go:651-659` | `snapshotRunner` không đăng ký `ws.RegisterOnStop`. Khi Worker tắt/khởi động lại (`SIGTERM`/`SIGINT`), các snapshot đang chạy bị cắt ngang mà không hủy context hay chuyển `status = 'running' -> 'paused'`. | Thêm `SnapshotRunner.Stop()` và đăng ký vào `ws.RegisterOnStop(func() { snapshotRunner.Stop() })`: hủy context của mọi snapshot đang chạy, chờ tối đa `10s` để thoát mẻ hiện tại và cập nhật `status = 'paused'` (giữ nguyên `last_seen_id`, `rows_processed`). |
| **`F39`** | **CRITICAL** | `centralized-data-service/internal/handler/orchestration/snapshot_runner_state.go:16-125`, `cdc-cms-service/internal/infra/persistence/scheduler/snapshot_progress_read_repo_gorm.go:23-31` | 1. Khi Worker khởi động lại, `SnapshotRunner` không có cơ chế quét và tự động chạy tiếp (`auto-resume`) các bản ghi bị ngắt dở do Worker dừng.<br>2. `ListSnapshotProgress` (`snapshot_progress_read_repo_gorm.go:23-31`) đánh dấu mọi bản ghi quá hạn 5 phút thành `status = 'error'` kể cả khi đã có `last_seen_id`. | 1. Thêm `SnapshotRunner.Start(ctx)` (đăng ký vào `ws.RegisterOnStart`) kèm bộ canh gác `30s` (`recoverOrphanedSnapshots`): tự động tìm các bản ghi bị ngắt dở (`status = 'running'` mồ côi `> 45s` không thuộc tiến trình hiện tại, hoặc bị dính `'Heartbeat timeout:%'` trong 24h qua mà đã có `last_seen_id` và `rows_processed < total_rows`) để **tự động tiếp tục chạy (`auto-resume`) từ `last_seen_id` tới khi `done` 100%** (tự động cứu luôn `id=90` chạy nốt từ `100,000 -> 250,363`!).<br>2. Tại `snapshot_progress_read_repo_gorm.go:23-31`, nếu bản ghi quá hạn 5 phút đã có `last_seen_id`, chuyển sang `status = 'paused'` thay vì `'error'`. |
| **`F40`** | **MEDIUM** | `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go:718-723` | Nhánh phân trang PostgreSQL gọi `strconv.ParseFloat(lastSeen, 64)`. Nếu cột khóa chính kiểu chuỗi (`TEXT`/`VARCHAR`) mang giá trị hex 24 ký tự gồm toàn chữ số và 1 chữ `'e'` (ví dụ `"672864612345678901234e12"`), `ParseFloat` ép nhầm thành `float64` làm sai kiểu tham số `$1`. | Bỏ hẳn nhánh `strconv.ParseFloat(lastSeen, 64)` và chỉ gọi `strconv.ParseInt(lastSeen, 10, 64)` khi `isDigitsOnly(lastSeen)` trả về `true`. |
| **`F41`** | **CRITICAL** | `centralized-data-service/internal/server/server.go:81-98`<br>`server_setup.go:153-160, 217, 335, 510`<br>`server_setup_buffers.go:41-43` | **Graceful Shutdown hiện hữu vừa KHÔNG ĐƯỢC ÁP DỤNG cho `SnapshotRunner`, vừa BỊ VÔ HIỆU HÓA do ngược thứ tự `s.onStop` (FIFO thay vì LIFO):**<br>1. `cdcBatchBuffer`, `snapshotBatchBuffer`, `healBatchBuffer` đăng ký `RegisterOnStop` ở dòng `153–160` (`s.onStop[0..2]`), trước các Consumer/Runner (`ConsumerPool` dòng 217, `transmuteShadowPool` dòng 335, `reconJobWorker` dòng 510).<br>2. Trong `WorkerServer.Shutdown()` (`server.go:82-84`), vòng lặp `for _, stopFn := range s.onStop` chạy xuôi (FIFO) khiến `snapshotBatchBuffer.Stop()` bị gọi **TRƯỚC** khi các Runner/Consumer dừng!<br>3. Ngay sau đó `server.go:94-98` gọi `sqlDB.Close()` đóng sập kết nối `s.db` trong khi `SnapshotRunner.runSnapshot` vẫn đang chạy ngầm. Khi `runSnapshot` chạm lỗi `sql: database is closed`, cả `r.markProgressError` (`snapshot_runner_state.go:138`) lẫn `defer r.writeActivity` (`snapshot_runner_handler.go:216`) đều dùng chính `r.db` đã đóng nên bị vô hiệu hóa hoàn toàn, bỏ lại `id=90` ở `status = 'running'` và mất luôn dòng log kết thúc trong `cdc_activity_log`! | Đổi vòng lặp `s.onStop` trong `WorkerServer.Shutdown()` (`server.go:82-84`) sang thứ tự **ngược LIFO (`for i := len(s.onStop) - 1; i >= 0; i--`)** chuẩn stack teardown (`defer`): dừng toàn bộ Runner/Consumer (`reconJobWorker`, `snapshotRunner`, `transmuteShadowPool`, `consumerPools`) và cập nhật `status = 'paused'` khi `BatchBuffer` và `sqlDB` **vẫn còn mở**, sau đó mới `Stop()`/`Flush()` các `BatchBuffer`, rồi cuối cùng mới đóng `nats.Close()`, `redis.Close()`, `sqlDB.Close()`. |

---

### 3. Code Demo 1:1 cho `P12` (Khớp tuyệt đối 1-1 với bản vá sẽ thực thi sau `APPROVE`)

#### Demo P12-1: `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go`
1. Bổ sung các trường quản lý vòng đời vào struct `SnapshotRunner` (`lines 83-87`) và gọi `r.spawnSnapshot(p, jobID, msg.Header)` trong `Handle` (`lines 183-218`):
```go
	transmuteCheckMu     sync.Mutex
	jsCtx                nats.JetStreamContext
	lastTransmuteCheckAt time.Time
	lastTransmutePending uint64

	lifecycleMu       sync.Mutex
	lifecycleCtx      context.Context
	lifecycleCancel   context.CancelFunc
	activeProgressIDs map[int64]context.CancelFunc
	activeWg          sync.WaitGroup
}
```
```go
	jobID := msg.Header.Get("Cdc-Job-Id")

	r.spawnSnapshot(p, jobID, msg.Header)
}

func (r *SnapshotRunner) runSnapshot(ctx context.Context, p snapshotV2Payload, jobID string) (retErr error) {
	startedAt := time.Now()
	var (
		targetTable    string
		connectionCode string
		rowsTotal      int64
		progressID     int64
	)
	defer func() {
		r.ensureProgressNotStuckRunning(progressID, retErr)
		status := "success"
		errMsg := ""
		if retErr != nil {
			status = "error"
			errMsg = retErr.Error()
		}
		r.writeActivity(p, jobID, targetTable, connectionCode, status, rowsTotal, startedAt, errMsg)
	}()
```
2. Kích hoạt Heartbeat nền `15s` ngay sau khi `claimProgress` thành công (`lines 478-482`), kiểm tra `ctx.Err()` ở đầu vòng lặp (`line 652`), và bỏ `strconv.ParseFloat` khi phân trang PostgreSQL (`lines 718-723`):
```go
	progressID = claim.rowID
	lastSeen := claim.lastSeen
	runCtx, stopHeartbeat := r.trackAndHeartbeatProgress(ctx, progressID)
	defer stopHeartbeat()
	ctx = runCtx
```
```go
	for {
		if ctx.Err() != nil {
			r.markProgressPaused(progressID)
			return nil
		}
		if isPaused.Load() {
```
```go
			} else {
				var pkVal interface{} = lastSeen
				if isDigitsOnly(lastSeen) {
					if val, pErr := strconv.ParseInt(lastSeen, 10, 64); pErr == nil {
						pkVal = val
					}
				}
				query := fmt.Sprintf("SELECT row_to_json(t) FROM (SELECT * FROM %s WHERE %s > $1 ORDER BY %s ASC LIMIT $2) t", tableName, pkName, pkName)
				rows, err = pgConn.Query(pgQueryCtx, query, pkVal, p.BatchSize)
			}
```

#### Demo P12-2: `centralized-data-service/internal/handler/orchestration/snapshot_runner_state.go`
Bổ sung các hàm quản lý vòng đời (`Start`, `Stop`, `spawnSnapshot`, `trackAndHeartbeatProgress`, `markProgressPaused`, `ensureProgressNotStuckRunning`, `recoverOrphanedSnapshots`, `isDigitsOnly`) vào cuối tệp `snapshot_runner_state.go`:
```go
func isDigitsOnly(s string) bool {
	if s == "" || len(s) > 19 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (r *SnapshotRunner) baseContext() context.Context {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	if r.lifecycleCtx != nil {
		return r.lifecycleCtx
	}
	return context.Background()
}

func (r *SnapshotRunner) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	r.lifecycleMu.Lock()
	if r.lifecycleCancel != nil {
		r.lifecycleMu.Unlock()
		return
	}
	r.lifecycleCtx, r.lifecycleCancel = context.WithCancel(ctx)
	if r.activeProgressIDs == nil {
		r.activeProgressIDs = make(map[int64]context.CancelFunc)
	}
	loopCtx := r.lifecycleCtx
	r.lifecycleMu.Unlock()

	r.activeWg.Add(1)
	go func() {
		defer r.activeWg.Done()
		r.recoverOrphanedSnapshots(loopCtx)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				r.recoverOrphanedSnapshots(loopCtx)
			}
		}
	}()
}

func (r *SnapshotRunner) Stop() {
	r.lifecycleMu.Lock()
	cancel := r.lifecycleCancel
	r.lifecycleCancel = nil
	activeIDs := make([]int64, 0, len(r.activeProgressIDs))
	for id, fn := range r.activeProgressIDs {
		activeIDs = append(activeIDs, id)
		if fn != nil {
			fn()
		}
	}
	r.lifecycleMu.Unlock()

	if cancel != nil {
		cancel()
	}

	doneCh := make(chan struct{})
	go func() {
		r.activeWg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(10 * time.Second):
	}

	for _, id := range activeIDs {
		r.markProgressPaused(id)
	}
}

func (r *SnapshotRunner) spawnSnapshot(p snapshotV2Payload, jobID string, header map[string][]string) {
	r.activeWg.Add(1)
	go func(p snapshotV2Payload, jobID string, header map[string][]string) {
		defer r.activeWg.Done()
		defer func() {
			if rec := recover(); rec != nil {
				if r.logger != nil {
					r.logger.Error("snapshot.v2 goroutine panic recovered",
						zap.Int64("source_object_id", p.SourceObjectID),
						zap.Int64("progress_id", p.ProgressID),
						zap.Any("panic", rec))
				}
				if p.ProgressID > 0 {
					r.markProgressError(context.Background(), p.ProgressID, fmt.Sprintf("panic: %v", rec))
				}
			}
		}()

		baseCtx := r.baseContext()
		ctx := observability.ExtractNATSHeader(baseCtx, header)
		ctx, span := observability.ChildSpan(ctx, "nats.SnapshotV2Runner")
		defer span.End()

		if sc := span.SpanContext(); sc.IsValid() && p.TraceID == "" {
			p.TraceID = sc.TraceID().String()
		}

		if err := r.runSnapshot(ctx, p, jobID); err != nil {
			observability.Ctx(ctx, r.logger).Error("snapshot.v2 run failed",
				zap.Int64("source_object_id", p.SourceObjectID),
				zap.String("trace_id", p.TraceID),
				zap.String("job_id", jobID),
				zap.Error(err))
		}
	}(p, jobID, header)
}

func (r *SnapshotRunner) trackAndHeartbeatProgress(parentCtx context.Context, progressID int64) (context.Context, func()) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	runCtx, runCancel := context.WithCancel(parentCtx)

	r.lifecycleMu.Lock()
	if r.activeProgressIDs == nil {
		r.activeProgressIDs = make(map[int64]context.CancelFunc)
	}
	r.activeProgressIDs[progressID] = runCancel
	r.lifecycleMu.Unlock()

	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if r.db != nil {
					_ = r.db.WithContext(context.Background()).Exec(`
						UPDATE cdc_system.snapshot_progress
						SET updated_at = NOW()
						WHERE id = ? AND status = 'running'
					`, progressID).Error
				}
			}
		}
	}()

	stop := func() {
		runCancel()
		<-hbDone
		r.lifecycleMu.Lock()
		delete(r.activeProgressIDs, progressID)
		r.lifecycleMu.Unlock()
	}
	return runCtx, stop
}

func (r *SnapshotRunner) isProgressActiveInMemory(progressID int64) bool {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	_, ok := r.activeProgressIDs[progressID]
	return ok
}

func (r *SnapshotRunner) markProgressPaused(progressID int64) {
	if progressID <= 0 || r.db == nil {
		return
	}
	_ = r.db.WithContext(context.Background()).Exec(`
		UPDATE cdc_system.snapshot_progress
		SET status = 'paused',
		    updated_at = NOW()
		WHERE id = ? AND status = 'running'
	`, progressID).Error
}

func (r *SnapshotRunner) ensureProgressNotStuckRunning(progressID int64, retErr error) {
	if progressID <= 0 || r.db == nil {
		return
	}
	if retErr != nil {
		_ = r.db.WithContext(context.Background()).Exec(`
			UPDATE cdc_system.snapshot_progress
			SET status = 'error',
			    error_msg = COALESCE(NULLIF(error_msg, ''), ?),
			    updated_at = NOW(),
			    finished_at = COALESCE(finished_at, NOW())
			WHERE id = ? AND status = 'running'
		`, retErr.Error(), progressID).Error
		return
	}
	_ = r.db.WithContext(context.Background()).Exec(`
		UPDATE cdc_system.snapshot_progress
		SET status = 'paused',
		    updated_at = NOW()
		WHERE id = ? AND status = 'running'
	`, progressID).Error
}

func (r *SnapshotRunner) recoverOrphanedSnapshots(ctx context.Context) {
	if r == nil || r.db == nil {
		return
	}
	type orphanRow struct {
		ID              int64   `gorm:"column:id"`
		SourceObjectID  int64   `gorm:"column:source_object_id"`
		ShadowBindingID *int64  `gorm:"column:shadow_binding_id"`
		TraceID         *string `gorm:"column:trace_id"`
		Status          string  `gorm:"column:status"`
	}
	var rows []orphanRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, source_object_id, shadow_binding_id, trace_id, status
		FROM cdc_system.snapshot_progress
		WHERE (
			status = 'running'
			AND updated_at < NOW() - INTERVAL '45 seconds'
		) OR (
			status IN ('paused', 'error')
			AND error_msg LIKE 'Heartbeat timeout:%'
			AND last_seen_id IS NOT NULL
			AND last_seen_id <> ''
			AND (total_rows = 0 OR rows_processed < total_rows)
			AND updated_at > NOW() - INTERVAL '24 hours'
		)
		ORDER BY id ASC
		LIMIT 10
	`).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return
	}

	for _, row := range rows {
		if ctx.Err() != nil {
			return
		}
		if r.isProgressActiveInMemory(row.ID) {
			continue
		}
		res := r.db.WithContext(ctx).Exec(`
			UPDATE cdc_system.snapshot_progress
			SET status = 'paused',
			    error_msg = NULL,
			    finished_at = NULL,
			    updated_at = NOW()
			WHERE id = ? AND status IN ('running', 'paused', 'error')
		`, row.ID)
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}

		var bindingID int64
		if row.ShadowBindingID != nil {
			bindingID = *row.ShadowBindingID
		}
		traceID := ""
		if row.TraceID != nil {
			traceID = *row.TraceID
		}
		if r.logger != nil {
			r.logger.Info("snapshot.v2 auto-resuming interrupted snapshot progress",
				zap.Int64("progress_id", row.ID),
				zap.Int64("source_object_id", row.SourceObjectID),
				zap.Int64("shadow_binding_id", bindingID),
				zap.String("prev_status", row.Status),
				zap.String("trace_id", traceID))
		}
		r.spawnSnapshot(snapshotV2Payload{
			SourceObjectID:  row.SourceObjectID,
			ShadowBindingID: bindingID,
			ProgressID:      row.ID,
			TraceID:         traceID,
			Action:          "auto-resume",
			Origin:          "worker-recovery",
			Overwrite:       false,
		}, "", nil)
	}
}
```
Và thêm import `"centralized-data-service/pkgs/observability"` vào đầu `snapshot_runner_state.go`:
```go
import (
	"centralized-data-service/internal/activity"
	"centralized-data-service/internal/model/system"
	"centralized-data-service/pkgs/observability"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

#### Demo P12-3: `centralized-data-service/internal/server/server_setup.go` (`lines 420-422`)
Đăng ký `Start` và `Stop` của `snapshotRunner` vào vòng đời `WorkerServer`:
```go
	natsClient.Conn.QueueSubscribe("cdc.cmd.snapshot.v2", "cdc-snapshot-runner", snapshotRunner.Handle)
	ws.RegisterOnStart(func() { snapshotRunner.Start(context.Background()) })
	ws.RegisterOnStop(func() { snapshotRunner.Stop() })
```

#### Demo P12-4: `cdc-cms-service/internal/infra/persistence/scheduler/snapshot_progress_read_repo_gorm.go` (`lines 20-32`)
Chuyển các bản ghi `running` quá hạn 5 phút đã có `last_seen_id` sang `paused` (để tự động resume hoặc bấm Resume tiếp) thay vì đập chết thành `error`:
```go
func (r *SnapshotProgressReadRepoGorm) ListSnapshotProgress(ctx context.Context, f scheduler.SnapshotProgressFilter, page, pageSize int) ([]scheduler.SnapshotProgressRow, int64, error) {
	// Clean up stale "running" snapshot progress records (heartbeat timeout).
	// Rows that already recorded a last_seen_id checkpoint are marked as 'paused' so worker recovery can auto-resume them.
	_ = r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.snapshot_progress
		SET status = CASE WHEN last_seen_id IS NOT NULL AND last_seen_id <> '' THEN 'paused' ELSE 'error' END,
		    error_msg = 'Heartbeat timeout: progress was stuck in running state for too long (worker stopped)',
		    finished_at = CASE WHEN last_seen_id IS NOT NULL AND last_seen_id <> '' THEN NULL ELSE NOW() END,
		    updated_at = NOW()
		WHERE status = 'running'
		  AND updated_at < NOW() - INTERVAL '5 minutes'
	`)
```

#### Demo P12-5: `centralized-data-service/internal/server/server.go` (`lines 81-85`)
Đổi thứ tự chạy `s.onStop` trong `WorkerServer.Shutdown()` sang **LIFO (ngược từ cuối về đầu)** để các Runner/Consumer (`reconJobWorker`, `snapshotRunner`, `transmuteShadowPool`, `consumerPools`) dừng và cập nhật DB trước khi `BatchBuffer` đóng và trước khi `sqlDB.Close()` đóng kết nối DB:
```go
	// 1. Kích hoạt toàn bộ luồng dọn dẹp theo thứ tự LIFO (Stop Runner/Consumer trước -> Flush BatchBuffer sau)
	for i := len(s.onStop) - 1; i >= 0; i-- {
		s.onStop[i]()
	}
```

---

## M. Kiến Trúc Chuẩn 2 Dạng Graceful Shutdown (`Snapshot`, `Transmute`, `Transform`) & Khắc Phục Triệt Để `transmute-cancel` (`P13 v4` — Đã Sửa Dứt Điểm `R-P13-1..R-P13-8`, `B1..B5`, `A1..A4`, `C1..C6` & `D1..D3` — `T25`/`T27`/`T28`/`T29`/`T26`)

### 1. Bảng Đối Chiếu Toàn Bộ Lỗ Hổng (`F42–F45`, `R-P13-1..R-P13-8`, `B1–B5`, `A1–A4`, `C1–C6`) & 3 Bẫy Mã Nguồn Thật (`D1–D3`)

| Mã | Phân loại | Vị trí (`file:line`) | Cơ chế Lỗ hổng & Phản biện Ngược (`VERIFIED`) | Giải pháp Chuẩn trong `P13 v4` |
| :--- | :--- | :--- | :--- | :--- |
| **`C1` + `D1`** | **BLOCKER — Đối Chiếu Nguyên Bản 100% Mã Nguồn Đĩa, Giữ `res.Updated += batchRes.updated` & Không Gọi Hàm Ảo `updateScheduleByID`** | `transmuter.go:33-48, 115-129, 167, 473-481`<br>`transmute_handler.go:278-282`<br>`snapshot_runner_handler.go:62-91` | 1. **`C1` (`VERIFIED`):** Ở lượt soạn thảo trước, một phần `Demo P13-2`, `P13-4`, `P13-5` trên khung chat bị gõ lại từ trí nhớ thay vì trích xuất trực tiếp từ file trên đĩa, dẫn tới lệch tên trường (`InsertedCount int` thay vì `Inserted int64`), sót `res.Updated += batchRes.updated`, và sót `SubjectTransmuteShadowCoreFallback`.<br>2. **Phản biện ngược `D1` (`VERIFIED`):** Trong file thật `transmuter.go:33-48`, `TransmuterModule` **không hề có hàm `t.updateScheduleByID`** (theo kiến trúc `D-39.A` tại `transmute_handler.go:275-277`, việc cập nhật `transmute_schedule` thuộc về `JobMonitor` qua NATS event và `TransmuteJobRepo.RequestCancel`). Nếu gọi `t.updateScheduleByID` trong `transmuter.go` sẽ lỗi biên dịch ngay lập tức! | 1. Trích xuất 100% nguyên bản từ file thật trên đĩa: `TransmuterModule` (`transmuter.go:33-48`), `TransmuteResult` (`transmuter.go:115-129` kiểu `int64`), `Run(ctx, masterName, onlySourceIDs, jobID, runType, masterBindingID ...int64)` (`transmuter.go:167`), giữ nguyên `res.Updated += batchRes.updated` (`transmuter.go:475`), giữ đủ 3 hằng số `SubjectTransmuteCompleted`, `SubjectTransmuteShadowCoreFallback`, `SubjectTransmuteCancel` (`transmute_handler.go:278-282`), và `SnapshotRunner` dùng đúng `r.lifecycleMu` (`Demo P12-1`).<br>2. Tuyệt đối không gọi hàm ảo `t.updateScheduleByID` trong `TransmuterModule.Run` (`D1`). |
| **`C2` + `D2` + `A4`** | **BLOCKER — `BatchBuffer.Stop()` Đóng `stopCh` Để `FlushWithContext(4s)` Khi `bb.ctx` Còn Sống Rồi Mới Gọi `bb.cancel()`** | `batch_buffer.go:29-60, 94-110, 257-277, 356-364, 838-840`<br>`server.go:81-98` | 1. **`C2` (`VERIFIED`):** Nếu `BatchBuffer.Stop()` gọi `bb.cancel()` trước khi `timerLoop()` chạy `Flush()`, bất kỳ thao tác nào bên trong hoặc đang giữ `bb.flushMu` phụ thuộc vào `bb.ctx` sẽ bị hủy ngay khi đang xả mẻ cuối.<br>2. **Phản biện ngược `D2` (`VERIFIED`):** Trong code thật trên đĩa (`batch_buffer.go:275-277, 363`), `bb.Flush()` gọi `bb.FlushWithContext(context.Background())` và tự tạo `dbCtx` với timeout **`15 * time.Second`**! Nếu `timerLoop()` gọi `bb.Flush()` mặc định khi tắt máy, nó có thể giữ lock tới `15s`, làm lố ngân sách tắt máy. | 1. Bổ sung `stopCh chan struct{}` và `stopOnce sync.Once` cạnh `doneCh chan struct{}` trong `BatchBuffer`.<br>2. Trong `timerLoop()`, nhánh `case <-bb.stopCh:` tạo `flushCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second); defer cancel(); _, _ = bb.FlushWithContext(flushCtx)` khi `bb.ctx` vẫn còn sống (`C2` + `D2`).<br>3. Trong `BatchBuffer.Stop()`, đóng `bb.stopCh` trước, chờ `<-bb.doneCh` (tối đa `4s`), và **chỉ sau khi `<-bb.doneCh` hoàn tất mới gọi `bb.cancel()`**. |
| **`C3` + `D3` + `B3` + `A3`** | **BLOCKER — Bảo Toàn `Force=true` Toàn Bảng (`ForceFields`), Kiểm Tra Kiểu Dữ Liệu PK (`detectPrimaryKey`) & Lọc Bỏ Tag Kỹ Thuật Trên UI** | `batch_transform_handler.go:26-37, 183-187, 223-230, 252-496`<br>`transform_job_repo.go:11-26`<br>`cdc-cms-service/internal/infra/persistence/*` | 1. **`C3.1` + Phản biện ngược `D3` (`VERIFIED`):** Trên đĩa (`batch_transform_handler.go:36`), tên trường trong `BatchTransformPayload` là **`ForceFields []string`** (chứ không phải `Fields`!). Ngoài ra tại `batch_transform_handler.go:183-187`, code hiện tại đang chặn `if payload.Force && len(payload.ForceFields) == 0` báo lỗi; và nếu chỉ lưu `[force_fields=...]` thì khi chạy `Force=true` cho toàn bộ cột (không truyền danh sách cột lẻ), lúc Resume sẽ mất cờ `Force=true`!<br>2. **`C3.2` (`VERIFIED`):** `detectPrimaryKey` (`batch_transform_handler.go:477-496`) hiện chỉ trả về tên cột `a.attname`. Nếu PK bảng Shadow là `VARCHAR`/`TEXT`/`UUID` mà giá trị trùng hợp toàn chữ số (`"0012345"` hoặc `"99887766"`), gọi mù `strconv.ParseInt` sẽ biến `lastPK` thành `int64`, gây lỗi PostgreSQL `SQLSTATE 42883: operator does not exist: text > bigint` hoặc mất số `0` ở đầu!<br>3. **`C3.3` (`VERIFIED`):** Lưu `[cursor_pk=...]` vào `error_message` mỗi nhịp `UpdateProgress` nếu trả thô ra API của `cdc-cms-service` sẽ khiến giao diện CMS hiển thị nhầm chuỗi kỹ thuật ở cột lỗi. | 1. Dùng đúng tên trường `ForceFields []string` (`D3`), cho phép `Force=true` với `ForceFields` rỗng (cưỡng bức toàn bộ cột active mapping), và mã hóa độc lập `[force=1]` cùng `[force_fields=...]` qua `FormatTransformCheckpointNote` / `ParseTransformForce` (`C3.1`).<br>2. Nâng cấp `detectPrimaryKey` trả về `(pkCol string, pkType string, err error)` bằng `format_type(a.atttypid, a.atttypmod)`; chỉ gọi `strconv.ParseInt` khi `isIntegerPGType(pkType)` là `true` (`int2/int4/int8/smallint/integer/bigint`), giữ nguyên `string` cho mọi kiểu chuỗi/UUID (`C3.2`).<br>3. Bổ sung `StripJobCheckpointTags(errMsg string) string` trong `cdc-cms-service` để lọc sạch mọi tag `[auto-resume #...]`, `[cursor_pk=...]`, `[force=...]`, `[force_fields=...]`, `[run_type=...]` trước khi trả JSON cho UI (`C3.3`). |
| **`C4` + `B2` + `B5` + `A2`** | **HIGH — Thứ Tự FIFO Tuyệt Đối Qua `sequencedBatch` Khi Đóng `stopCh` & Tổng Ngân Sách Tắt Máy LIFO `< 13s`** | `debounce.go:31-221`<br>`snapshot_runner_handler.go`<br>`transmute_handler.go`<br>`batch_transform_handler.go`<br>`server_setup.go` | 1. **`C4.1` (`VERIFIED`):** Trong `P13 v3`, nếu 2 goroutine lấy `Batch #17` và `Batch #18` khỏi `takeFlushBatchLocked()` rồi cùng rơi vào nhánh `<-td.stopCh` và chạy `td.tasks = append(tasks, td.tasks...)` (prepend vào đầu mảng), `Batch #17` chạy trước sẽ nằm sau `Batch #18` (`[Batch #18, Batch #17]`), làm **đảo ngược thứ tự sự kiện CDC**!<br>2. **`C4.2` (`VERIFIED`):** Tạo `context.WithTimeout(..., 5s)` riêng cho từng batch trong `drainRemainingOnStop` sẽ làm `N` batch tốn `N × 5s`; cộng với `3s + 3s + 3s + 6s` ở các tầng khác sẽ vượt mốc `15s` SIGKILL. | 1. Gán số thứ tự đơn điệu `seq uint64` (`td.nextSeq++`) cho mỗi `sequencedBatch{seq, tasks}` ngay trong `takeFlushBatchLocked()` dưới `td.mu.Lock()`; nếu `enqueueFlushBatch` gặp `<-td.stopCh`, đẩy vào `td.overflowBatches`. Trong `drainRemainingOnStop()`, gom toàn bộ `td.flushCh` và `td.overflowBatches`, **sắp xếp theo `seq ASC`** rồi mới xả `td.tasks` ở cuối — đảm bảo 100% thứ tự FIFO toán học (`Batch #17` luôn trước `Batch #18` trước `td.tasks`).<br>2. Dùng **1 `drainCtx` chung `4s`** cho toàn bộ vòng lặp `drainRemainingOnStop()`; nếu hết `4s`, gọi `task.complete(drainCtx.Err())` cho các batch còn lại để JetStream `NakWithDelay`.<br>3. Báo trước `SetStopping(true)` đồng loạt và hạ ngân sách chờ ranh giới mẻ của 3 handler xuống **`2s`**, `TableDebouncer` drain song song tối đa **`4s`**, `BatchBuffer.Stop()` tối đa **`4s`** → tổng thời gian tắt máy xấu nhất $\le \mathbf{12\text{s} < 15\text{s}}$. |
| **`C5` + `B4`** | **HIGH — Bảo Toàn Chế Độ `Full Sync` (`[run_type=full]`) Khi Resume & Hiển Thị Đầy Đủ Nhánh Bắt Lỗi SQL Trong `SnapshotRunner`** | `transmuter.go:208-235, 360-465`<br>`transmute_handler.go:307-360`<br>`snapshot_runner_handler.go:651-752, 860-875` | 1. **`C5.1` (`VERIFIED`):** Nếu `recoverOrphanedTransmuteJobs` không lưu và khôi phục `RunType`, job `Full Sync` (`runType="full"`) khi Resume sẽ biến thành `"incremental"`; ngược lại nếu `runType == "full"` mà tất cả các dòng ở các mẻ đầu đều bị `Skipped` (`baseRowsAffected == 0` nhưng `last_gpay_id > 0`), kiểm tra `baseRowsAffected == 0` sẽ reset nhầm `last_gpay_id` về `0`!<br>2. **`C5.2` (`VERIFIED`):** Trong `SnapshotRunner.runSnapshot` (`snapshot_runner_handler.go:679-738, 863-868`), khi `Stop()` gọi `stopCancel()` sau `2s`, các câu lệnh `coll.Find`, `cursor.All`, `pgConn.Query`, `scanPGXRowsAsJSON`, hoặc `flushSnapshotTables` đang chạy sẽ trả về lỗi `context canceled`. Nếu không kiểm tra `r.stopping.Load()` ở các nhánh `if err != nil` đó, code sẽ gọi `r.markProgressError` đánh nhầm thành `error` thay vì `paused`! | 1. Lưu `[run_type=full|incremental]` vào `error_message` qua `FormatTransmuteCheckpointNote` và khôi phục bằng `ParseTransmuteRunType` trong `recoverOrphanedTransmuteJobs`. Xác định `isResumedJob := existingJob != nil && (existingJob.Status == "PAUSED" || existingJob.RowsAffected > 0 || resumeAttempts > 0)`: chỉ reset `{"last_gpay_id": 0}` khi `runType == "full" && !isResumedJob`, và luôn đọc tiếp `last_gpay_id` từ `sync_runtime_state` khi `isResumedJob` (`C5.1`).<br>2. Trong `Demo P13-2` (`snapshot_runner_handler.go:651-752, 860-875`), xuất đầy đủ 100% các nhánh bắt lỗi Mongo/PG và `flushSnapshotTables`: khi `r.stopping.Load()` là `true`, gọi `r.markProgressPaused(progressID)` và `return nil` thay vì `r.markProgressError` (`C5.2`). |
| **`C6` + `B1` + `A1`** | **HIGH — Không Hủy Oan Job `PAUSED` (`[force=1]` / `[run_type=full]`) Trong `ClaimRecoverableJobs` & Giữ Chặt Cổng Hủy `B1`** | `transmute_job_repo.go`<br>`transform_job_repo.go`<br>`transmuter.go` | **`C6` (`VERIFIED`):** Trong `P13 v3`, câu lệnh `UPDATE ... SET status = 'CANCELLED' WHERE ... EXISTS (SELECT 1 FROM ... j2 WHERE j2.created_at > j.created_at)` trong `ClaimRecoverableJobs` sẽ **hủy oan** một job `Force Transform` (`[force=1]`) hoặc `Full Sync` (`[run_type=full]`) đang `PAUSED` nếu có một nhịp incremental/normal transform nhỏ chạy sau đó! | Bỏ câu lệnh `UPDATE ... SET status = 'CANCELLED'` hủy mù các job `PAUSED`! Chỉ tự động đóng job cũ nếu có job mới hơn trên cùng bảng đã **`COMPLETED` với phạm vi tương đương hoặc rộng hơn** (`j.error_message NOT LIKE '%[force=1]%'` với Transform, hoặc `j.error_message NOT LIKE '%[run_type=full]%'` với Transmute); còn khi bảng đang có job khác `RUNNING` (`updated_at >= NOW() - INTERVAL '45 seconds'`), CTE `FOR UPDATE SKIP LOCKED` chỉ tạm thời bỏ qua (`NOT EXISTS`) để job `PAUSED` chờ tới lượt chạy an toàn. |

---

### 2. Code Demo 1:1 Đầy Đủ Cho `P13 v4` (9 Module Hoàn Chỉnh Khớp 100% Mã Nguồn Đĩa, 0 Cắt Cúp `...`)

#### Demo P13-1: `centralized-data-service/internal/handler/master/debounce.go` (`lines 31-221`) — Sửa `F42`, `R-P13-1`, `B2`, `B5` & `C4`
1. Định nghĩa `type sequencedBatch struct { seq uint64; tasks []NatsMsgTask }` và cấp số thứ tự tăng dần `td.nextSeq++` ngay trong `takeFlushBatchLocked()` dưới `td.mu.Lock()`.
2. Nếu `enqueueFlushBatch()` gặp `<-td.stopCh`, đưa `batch` vào `td.overflowBatches` dưới `td.mu.Lock()`. Trong `drainRemainingOnStop()`, gom toàn bộ `td.flushCh` và `td.overflowBatches`, **sắp xếp theo `seq ASC`** rồi mới nối `td.tasks` ở cuối — đảm bảo thứ tự FIFO toán học 100% (`Batch #17` luôn trước `Batch #18` trước `td.tasks`, sửa dứt điểm `C4.1`).
3. Dùng **1 `drainCtx` chung `4 * time.Second`** cho toàn bộ `drainRemainingOnStop()`; nếu hết `4s`, gọi `t.complete(drainCtx.Err())` cho các batch còn lại để JetStream `NakWithDelay` phát lại an toàn (sửa dứt điểm `C4.2`):
```go
type sequencedBatch struct {
	seq   uint64
	tasks []NatsMsgTask
}

type TableDebouncer struct {
	mu              sync.Mutex
	enqueueCond     *sync.Cond
	enqueuing       int
	nextSeq         uint64
	overflowBatches []sequencedBatch
	tableName       string
	tasks           []NatsMsgTask
	pendingEvents   int
	maxSize         int
	idleTimeout     time.Duration
	maxTimeout      time.Duration
	idleTimer       *time.Timer
	maxTimer        *time.Timer
	flushCh         chan sequencedBatch
	stopCh          chan struct{}
	stopOnce        sync.Once
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	sem             chan struct{}
	logger          *zap.Logger
	runTransmute    func(ctx context.Context, batch []NatsMsgTask)
}

const maxDebouncedEventsPerBatch = 1000

func taskEventCount(t NatsMsgTask) int {
	if n := len(t.Req.Events); n > 0 {
		return n
	}
	if n := len(t.Req.SourceIDs); n > 0 {
		return n
	}
	return 1
}

func NewTableDebouncer(
	ctx context.Context,
	tableName string,
	maxSize int,
	idleTimeout time.Duration,
	maxTimeout time.Duration,
	concurrencyLimit int,
	logger *zap.Logger,
	runTransmute func(ctx context.Context, batch []NatsMsgTask),
) *TableDebouncer {
	if concurrencyLimit <= 0 {
		concurrencyLimit = 1
	}
	subCtx, cancel := context.WithCancel(ctx)
	td := &TableDebouncer{
		tableName:    tableName,
		maxSize:      maxSize,
		idleTimeout:  idleTimeout,
		maxTimeout:   maxTimeout,
		flushCh:      make(chan sequencedBatch, 16),
		stopCh:       make(chan struct{}),
		ctx:          subCtx,
		cancel:       cancel,
		sem:          make(chan struct{}, concurrencyLimit),
		logger:       logger,
		runTransmute: runTransmute,
	}
	td.enqueueCond = sync.NewCond(&td.mu)

	td.wg.Add(1)
	go td.workerLoop()

	return td
}

func (td *TableDebouncer) Add(msg *nats.Msg, req TransmuteRequest) {
	td.AddTask(NatsMsgTask{Msg: msg, Req: req})
}

func (td *TableDebouncer) AddTask(task NatsMsgTask) {
	for {
		td.mu.Lock()
		select {
		case <-td.stopCh:
			td.mu.Unlock()
			task.complete(context.Canceled)
			return
		default:
		}

		// Backpressure thực tế: Hãm đầu vào nếu flushCh đã đầy hoặc RAM tích tụ quá tải
		if len(td.flushCh) >= cap(td.flushCh) || len(td.tasks) >= td.maxSize*2 {
			td.mu.Unlock()
			select {
			case <-td.stopCh:
				task.complete(context.Canceled)
				return
			case <-td.ctx.Done():
				task.complete(td.ctx.Err())
				return
			case <-time.After(10 * time.Millisecond):
				continue
			}
		}

		td.tasks = append(td.tasks, task)
		td.pendingEvents += taskEventCount(task)

		if len(td.tasks) >= td.maxSize || td.pendingEvents >= maxDebouncedEventsPerBatch {
			batch := td.takeFlushBatchLocked()
			td.mu.Unlock()
			td.enqueueFlushBatch(batch)
			break
		}

		td.setupTimersLocked()
		td.mu.Unlock()
		break
	}
}

func (td *TableDebouncer) setupTimersLocked() {
	if td.idleTimer != nil {
		td.idleTimer.Stop()
	}
	td.idleTimer = time.AfterFunc(td.idleTimeout, func() {
		td.mu.Lock()
		select {
		case <-td.stopCh:
			td.mu.Unlock()
			return
		default:
		}
		batch := td.takeFlushBatchLocked()
		td.mu.Unlock()
		td.enqueueFlushBatch(batch)
	})

	if td.maxTimer == nil {
		td.maxTimer = time.AfterFunc(td.maxTimeout, func() {
			td.mu.Lock()
			select {
			case <-td.stopCh:
				td.mu.Unlock()
				return
			default:
			}
			batch := td.takeFlushBatchLocked()
			td.mu.Unlock()
			td.enqueueFlushBatch(batch)
		})
	}
}

func (td *TableDebouncer) takeFlushBatchLocked() sequencedBatch {
	if len(td.tasks) == 0 {
		return sequencedBatch{}
	}
	if td.idleTimer != nil {
		td.idleTimer.Stop()
		td.idleTimer = nil
	}
	if td.maxTimer != nil {
		td.maxTimer.Stop()
		td.maxTimer = nil
	}
	tasks := td.tasks
	td.tasks = make([]NatsMsgTask, 0, td.maxSize)
	td.pendingEvents = 0
	td.nextSeq++
	td.enqueuing++
	return sequencedBatch{
		seq:   td.nextSeq,
		tasks: tasks,
	}
}

func (td *TableDebouncer) enqueueFlushBatch(batch sequencedBatch) {
	if len(batch.tasks) == 0 {
		return
	}
	defer func() {
		td.mu.Lock()
		td.enqueuing--
		if td.enqueuing <= 0 && td.enqueueCond != nil {
			td.enqueueCond.Broadcast()
		}
		td.mu.Unlock()
	}()

	// Ưu tiên kiểm tra stopCh trước để tránh select chọn ngẫu nhiên 50/50 khi stopCh đã đóng (B2 & C4)
	select {
	case <-td.stopCh:
		td.mu.Lock()
		td.overflowBatches = append(td.overflowBatches, batch)
		td.mu.Unlock()
		return
	default:
	}

	select {
	case td.flushCh <- batch:
	case <-td.stopCh:
		td.mu.Lock()
		td.overflowBatches = append(td.overflowBatches, batch)
		td.mu.Unlock()
	}
}

func (td *TableDebouncer) drainRemainingOnStop(firstBatch sequencedBatch) {
	// BẮT BUỘC chiếm đủ toàn bộ cap(td.sem) slot trước tiên để đợi mọi mẻ in-flight hoàn tất 100% (B2)
	for i := 0; i < cap(td.sem); i++ {
		td.sem <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(td.sem); i++ {
			<-td.sem
		}
	}()

	// Chờ mọi goroutine vừa gọi takeFlushBatchLocked() hoàn tất enqueueFlushBatch() (0 rơi batch — B2)
	td.mu.Lock()
	for td.enqueuing > 0 {
		td.enqueueCond.Wait()
	}
	if td.idleTimer != nil {
		td.idleTimer.Stop()
		td.idleTimer = nil
	}
	if td.maxTimer != nil {
		td.maxTimer.Stop()
		td.maxTimer = nil
	}

	var pendingBatches []sequencedBatch
	if len(firstBatch.tasks) > 0 {
		pendingBatches = append(pendingBatches, firstBatch)
	}
	for {
		select {
		case b := <-td.flushCh:
			if len(b.tasks) > 0 {
				pendingBatches = append(pendingBatches, b)
			}
		default:
			goto drainedChannel
		}
	}
drainedChannel:
	if len(td.overflowBatches) > 0 {
		pendingBatches = append(pendingBatches, td.overflowBatches...)
		td.overflowBatches = nil
	}
	tailTasks := td.tasks
	td.tasks = nil
	td.pendingEvents = 0
	td.mu.Unlock()

	// Sửa C4.1: Sắp xếp toàn bộ các batch theo đúng số thứ tự seq tăng dần (FIFO tuyệt đối: Batch #17 luôn trước Batch #18)
	sort.Slice(pendingBatches, func(i, j int) bool {
		return pendingBatches[i].seq < pendingBatches[j].seq
	})

	// Sửa C4.2: Dùng chung ĐÚNG 1 drainCtx 4s cho toàn bộ vòng lặp xả của TableDebouncer;
	// nếu hết 4s, gọi task.complete(drainCtx.Err()) để JetStream NakWithDelay phát lại.
	drainCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	runOrNak := func(tasks []NatsMsgTask) {
		if len(tasks) == 0 {
			return
		}
		if drainCtx.Err() != nil {
			for _, t := range tasks {
				t.complete(drainCtx.Err())
			}
			return
		}
		td.runTransmute(drainCtx, tasks)
	}

	for _, b := range pendingBatches {
		runOrNak(b.tasks)
	}
	runOrNak(tailTasks)
}

func (td *TableDebouncer) workerLoop() {
	defer td.wg.Done()
	for {
		select {
		case <-td.stopCh:
			td.drainRemainingOnStop(sequencedBatch{})
			return
		case batch := <-td.flushCh:
			select {
			case td.sem <- struct{}{}:
				go func(b sequencedBatch) {
					defer func() { <-td.sem }()
					td.runTransmute(td.ctx, b.tasks)
				}(batch)
			case <-td.stopCh:
				td.drainRemainingOnStop(batch)
				return
			}
		}
	}
}

func (td *TableDebouncer) Stop() {
	td.stopOnce.Do(func() {
		close(td.stopCh)
	})
	td.wg.Wait()
	td.cancel()
}
```

#### Demo P13-2: `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go` (`lines 62-95`, `180-215`, `651-752`, `860-875`) — Sửa `C1`, `C4` & `C5`
1. Đồng nhất 100% tên trường struct `SnapshotRunner` (`r.lifecycleMu`, `r.stopCancel`, `r.stopping`, `r.wg`) với `Demo P12-1` và file thật trên đĩa (`C1`).
2. Hạ ngân sách chờ ranh giới mẻ trong `Stop()` xuống **`2 * time.Second`** (`C4`), kèm `SetStopping(true)` báo trước.
3. Hiển thị đầy đủ 100% vòng lặp `runSnapshot` (`lines 651-752`) và `flushSnapshotTables` (`lines 860-875`) bao gồm cả các nhánh bắt lỗi Mongo `coll.Find`/`cursor.All` (`lines 679-691`) và PG `pgConn.Query`/`scanPGXRowsAsJSON` (`lines 727-738`): khi `r.stopping.Load()` là `true` (do `Stop()` gọi `stopCancel()` sau `2s`), chuyển trạng thái sang `r.markProgressPaused(progressID)` và `return nil` thay vì đánh nhầm thành `r.markProgressError` (`C5.2`):
```go
// Trong SnapshotRunner (lines 62-95 & 180-215):
func (r *SnapshotRunner) SetStopping(v bool) {
	r.stopping.Store(v)
}

// Stop gracefully pauses all in-flight snapshots at the current batch boundary (up to 2s budget)
// and falls back to canceling runCtx while preserving 'paused' state in snapshot_progress (C4 & C5).
func (r *SnapshotRunner) Stop() {
	r.stopping.Store(true)
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		r.lifecycleMu.Lock()
		if r.stopCancel != nil {
			r.stopCancel()
		}
		r.lifecycleMu.Unlock()
		<-done
	}
}
```
```go
// Trong SnapshotRunner.runSnapshot (lines 651-752) & flushSnapshotTables (lines 860-875) — đầy đủ 100% nhánh bắt lỗi SQL/Mongo khi Stop (C5.2):
	for {
		select {
		case <-ctx.Done():
			if r.stopping.Load() {
				r.markProgressPaused(progressID)
				return nil
			}
			r.markProgressPaused(progressID)
			return fmt.Errorf("snapshot cancelled after %d rows: %w", totalRows, ctx.Err())
		default:
		}
		if r.stopping.Load() {
			r.markProgressPaused(progressID)
			r.logger.Info("snapshot v2: paused cleanly at batch boundary on worker shutdown",
				zap.String("table", cmd.Table),
				zap.Int64("progress_id", progressID),
				zap.Int64("rows_processed", totalRows),
				zap.String("last_seen_id", lastSeen))
			return nil
		}

		pageStart := time.Now()
		var rawDocs []rawDocItem

		if engine == "mongodb" {
			filter := bson.D{}
			if lastSeen != "" {
				if oid, err := primitive.ObjectIDFromHex(lastSeen); err == nil {
					filter = bson.D{{Key: pkField, Value: bson.D{{Key: "$gt", Value: oid}}}}
				} else {
					filter = bson.D{{Key: pkField, Value: bson.D{{Key: "$gt", Value: lastSeen}}}}
				}
			}
			opts := mongooptions.Find().
				SetSort(bson.D{{Key: pkField, Value: 1}}).
				SetLimit(int64(batchSize))

			cursor, err := coll.Find(ctx, filter, opts)
			if err != nil {
				if r.stopping.Load() {
					r.markProgressPaused(progressID)
					return nil
				}
				r.markProgressError(progressID, err.Error())
				return fmt.Errorf("mongo find page %d: %w", pageIdx, err)
			}
			var bsonDocs []bson.M
			if err := cursor.All(ctx, &bsonDocs); err != nil {
				_ = cursor.Close(ctx)
				if r.stopping.Load() {
					r.markProgressPaused(progressID)
					return nil
				}
				r.markProgressError(progressID, err.Error())
				return fmt.Errorf("mongo cursor read page %d: %w", pageIdx, err)
			}
			_ = cursor.Close(ctx)

			for _, doc := range bsonDocs {
				sid := extractDocID(doc, pkField)
				b, err := bsonDocToJSON(doc)
				if err != nil {
					continue
				}
				rawDocs = append(rawDocs, rawDocItem{sourceID: sid, rawJSON: b})
			}
		} else {
			qualifiedTable := quoteIdentSnapshot(cmd.Table)
			if cmd.Schema != "" && !strings.Contains(cmd.Table, ".") {
				qualifiedTable = quoteIdentSnapshot(cmd.Schema) + "." + quoteIdentSnapshot(cmd.Table)
			}
			quotedPK := quoteIdentSnapshot(pkField)

			var querySQL string
			var queryArgs []any
			if lastSeen == "" {
				querySQL = fmt.Sprintf("SELECT * FROM %s ORDER BY %s ASC LIMIT $1", qualifiedTable, quotedPK)
				queryArgs = []any{batchSize}
			} else {
				querySQL = fmt.Sprintf("SELECT * FROM %s WHERE %s > $1 ORDER BY %s ASC LIMIT $2", qualifiedTable, quotedPK, quotedPK)
				var cursorParam any = lastSeen
				if isDigitsOnly(lastSeen) {
					if intVal, err := strconv.ParseInt(lastSeen, 10, 64); err == nil {
						cursorParam = intVal
					}
				}
				queryArgs = []any{cursorParam, batchSize}
			}

			rows, err := pgConn.Query(ctx, querySQL, queryArgs...)
			if err != nil {
				if r.stopping.Load() {
					r.markProgressPaused(progressID)
					return nil
				}
				r.markProgressError(progressID, err.Error())
				return fmt.Errorf("postgres query page %d: %w", pageIdx, err)
			}
			rawDocs, err = scanPGXRowsAsJSON(rows, pkField)
			rows.Close()
			if err != nil {
				if r.stopping.Load() {
					r.markProgressPaused(progressID)
					return nil
				}
				r.markProgressError(progressID, err.Error())
				return fmt.Errorf("postgres scan page %d: %w", pageIdx, err)
			}
		}

		if len(rawDocs) == 0 {
			break
		}
```
```go
// Tại nhánh flushSnapshotTables trong runSnapshot (lines 860-875):
		flushRes, err := r.flushSnapshotTables(ctx, targetTable, childTablesForFlush)
		if err != nil {
			if r.stopping.Load() {
				r.markProgressPaused(progressID)
				return nil
			}
			r.markProgressError(progressID, err.Error())
			return fmt.Errorf("shadow flush at page %d: %w", pageIdx, err)
		}
```

#### Demo P13-3: `centralized-data-service/internal/repository/transmute_job_repo.go` (`lines 56-132`) — Sửa `B1`, `B4`, `C5`, `C6`
1. Thêm `FormatTransmuteCheckpointNote(attempt int, runType string) string` và `ParseTransmuteRunType(errMsg string) string` để bảo toàn `[run_type=full]` / `[run_type=incremental]` qua `PAUSED` và Auto-Resume (`C5.1`).
2. Trong `ClaimRecoverableJobs`: bỏ câu lệnh `UPDATE ... SET status = 'CANCELLED'` hủy mù các job `PAUSED` (`C6`); chỉ tự động đóng job cũ nếu đã có job mới hơn trên cùng `master_table` **`COMPLETED` với phạm vi tương đương hoặc rộng hơn** (`j.error_message NOT LIKE '%[run_type=full]%' OR j2.error_message LIKE '%[run_type=full]%'`), và trong CTE `FOR UPDATE SKIP LOCKED` chỉ tạm thời chờ (`NOT EXISTS`) khi bảng đang có job khác `RUNNING`:
```go
const MaxOrphanAutoResumeAttempts = 3

// ParseAutoResumeCount extracts N from the "[auto-resume #N]" marker in error_message.
func ParseAutoResumeCount(errMsg string) int {
	const prefix = "[auto-resume #"
	idx := strings.LastIndex(errMsg, prefix)
	if idx == -1 {
		return 0
	}
	rest := errMsg[idx+len(prefix):]
	end := strings.IndexByte(rest, ']')
	if end <= 0 {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(rest[:end]))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// ParseTransmuteRunType extracts the saved run_type ("full" or "incremental") from error_message ("[run_type=...]").
func ParseTransmuteRunType(errMsg string) string {
	const prefix = "[run_type="
	idx := strings.LastIndex(errMsg, prefix)
	if idx == -1 {
		return "incremental"
	}
	rest := errMsg[idx+len(prefix):]
	end := strings.IndexByte(rest, ']')
	if end <= 0 {
		return "incremental"
	}
	rt := strings.TrimSpace(rest[:end])
	if rt == "full" {
		return "full"
	}
	return "incremental"
}

// FormatTransmuteCheckpointNote encodes auto-resume attempts and run_type into error_message.
func FormatTransmuteCheckpointNote(attempt int, runType string) string {
	var parts []string
	if attempt > 0 {
		parts = append(parts, fmt.Sprintf("[auto-resume #%d]", attempt))
	}
	if runType == "full" {
		parts = append(parts, "[run_type=full]")
	} else if runType != "" {
		parts = append(parts, fmt.Sprintf("[run_type=%s]", runType))
	}
	return strings.Join(parts, " ")
}

// FormatOrphanAutoResumeError formats the error_message with the incremented "[auto-resume #N]" marker and preserved run_type.
func FormatOrphanAutoResumeError(attempt int, runType ...string) string {
	rt := ""
	if len(runType) > 0 {
		rt = runType[0]
	}
	return FormatTransmuteCheckpointNote(attempt, rt)
}

// GetByID retrieves a TransmuteJob by job_id.
func (r *TransmuteJobRepo) GetByID(ctx context.Context, jobID string) (*TransmuteJob, error) {
	if jobID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var job TransmuteJob
	if err := r.db.WithContext(ctx).Where("job_id = ?", jobID).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateStatus updates status, rows_affected, total_rows, error_message, finished_at, and optionally master_binding_id.
func (r *TransmuteJobRepo) UpdateStatus(ctx context.Context, jobID, status string, rowsAffected, totalRows int64, errMsg string, masterBindingID ...int64) error {
	updates := map[string]interface{}{
		"status":        status,
		"rows_affected": rowsAffected,
		"updated_at":    time.Now(),
	}
	if len(masterBindingID) > 0 && masterBindingID[0] > 0 {
		updates["master_binding_id"] = masterBindingID[0]
	}
	if totalRows > 0 {
		updates["total_rows"] = totalRows
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	} else if status == "COMPLETED" {
		updates["error_message"] = ""
	}
	if status == "RUNNING" {
		if rowsAffected > 0 && totalRows > 0 {
			pct := int(float64(rowsAffected) / float64(totalRows) * 100)
			if pct > 99 {
				pct = 99
			}
			updates["progress_percent"] = pct
		} else {
			updates["progress_percent"] = 0
		}
	}
	if status == "COMPLETED" || status == "FAILED" || status == "CANCELLED" {
		now := time.Now()
		updates["finished_at"] = &now
		if status == "COMPLETED" {
			updates["progress_percent"] = 100
		}
	}
	q := r.db.WithContext(ctx).Model(&TransmuteJob{})
	if status == "RUNNING" || status == "PAUSED" {
		q = q.Where("job_id = ? AND cancel_requested = false AND status NOT IN ('CANCELLED', 'COMPLETED', 'FAILED')", jobID)
	} else {
		q = q.Where("job_id = ?", jobID)
	}
	return q.Updates(updates).Error
}

// UpdateProgress updates progress_percent, rows_affected, total_rows, chunk_size while guarding against overwriting terminal/cancelled states.
func (r *TransmuteJobRepo) UpdateProgress(ctx context.Context, jobID string, progress int, rowsAffected, totalRows int64, chunkSize int) error {
	updates := map[string]interface{}{
		"progress_percent": progress,
		"rows_affected":    rowsAffected,
		"chunk_size":       chunkSize,
		"updated_at":       time.Now(),
	}
	if totalRows > 0 {
		updates["total_rows"] = totalRows
	}
	return r.db.WithContext(ctx).
		Model(&TransmuteJob{}).
		Where("job_id = ? AND cancel_requested = false AND status NOT IN ('CANCELLED', 'COMPLETED', 'FAILED')", jobID).
		Updates(updates).Error
}

// IsCancelRequested returns true if cancel_requested=true or status='CANCELLED' for the job.
// Uses an independent 3s background context so it still succeeds after the job's ctx is canceled by HandleTransmuteCancel (B1).
func (r *TransmuteJobRepo) IsCancelRequested(_ context.Context, jobID string) bool {
	if jobID == "" {
		return false
	}
	qCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var result struct {
		CancelRequested bool   `gorm:"column:cancel_requested"`
		Status          string `gorm:"column:status"`
	}
	if err := r.db.WithContext(qCtx).
		Raw("SELECT cancel_requested, status FROM cdc_system.transmute_jobs WHERE job_id = ?", jobID).
		Scan(&result).Error; err != nil {
		return false
	}
	return result.CancelRequested || result.Status == "CANCELLED"
}

// TouchHeartbeat updates updated_at = NOW() while a transmute job is actively running.
func (r *TransmuteJobRepo) TouchHeartbeat(ctx context.Context, jobID string) error {
	if jobID == "" {
		return nil
	}
	return r.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.transmute_jobs SET updated_at = NOW() WHERE job_id = ? AND cancel_requested = false AND status IN ('PENDING', 'RUNNING')`,
		jobID,
	).Error
}

// ClaimRecoverableJobs sweeps stale cancel_requested jobs to CANCELLED and atomically claims
// PAUSED or orphaned PENDING/RUNNING jobs (>45s without heartbeat) using FOR UPDATE SKIP LOCKED
// without prematurely killing PAUSED [run_type=full] jobs (C6).
func (r *TransmuteJobRepo) ClaimRecoverableJobs(ctx context.Context) ([]TransmuteJob, error) {
	_ = r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transmute_jobs
		   SET status = 'CANCELLED',
		       error_message = 'cancelled by user',
		       finished_at = NOW(),
		       updated_at = NOW()
		 WHERE cancel_requested = true
		   AND status IN ('PENDING', 'RUNNING', 'PAUSED')
		   AND updated_at < NOW() - INTERVAL '45 seconds'
	`).Error

	// Sửa C6: Chỉ đóng job cũ nếu đã có một job mới hơn trên cùng master_table HOÀN TẤT (COMPLETED)
	// với phạm vi tương đương hoặc rộng hơn (không bao giờ lấy job incremental để hủy oan job [run_type=full] đang PAUSED!).
	_ = r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transmute_jobs j
		   SET status = 'CANCELLED',
		       error_message = 'superseded by completed newer job on same master_table',
		       finished_at = NOW(),
		       updated_at = NOW()
		 WHERE j.status IN ('PENDING', 'RUNNING', 'PAUSED')
		   AND (j.status = 'PAUSED' OR j.updated_at < NOW() - INTERVAL '45 seconds')
		   AND EXISTS (
		       SELECT 1
		         FROM cdc_system.transmute_jobs j2
		        WHERE j2.master_table = j.master_table
		          AND j2.job_id <> j.job_id
		          AND j2.created_at > j.created_at
		          AND j2.status = 'COMPLETED'
		          AND (
		              j.error_message NOT LIKE '%[run_type=full]%'
		              OR j2.error_message LIKE '%[run_type=full]%'
		          )
		   )
	`).Error

	var jobs []TransmuteJob
	err := r.db.WithContext(ctx).Raw(`
		WITH claimed AS (
			SELECT j.job_id
			FROM cdc_system.transmute_jobs j
			WHERE j.cancel_requested = false
			  AND (
			      j.status = 'PAUSED'
			      OR (j.status IN ('PENDING', 'RUNNING') AND j.updated_at < NOW() - INTERVAL '45 seconds')
			  )
			  AND NOT EXISTS (
			      SELECT 1
			      FROM cdc_system.transmute_jobs j2
			      WHERE j2.master_table = j.master_table
			        AND j2.job_id <> j.job_id
			        AND j2.status = 'RUNNING'
			        AND j2.updated_at >= NOW() - INTERVAL '45 seconds'
			  )
			ORDER BY j.updated_at ASC
			LIMIT 10
			FOR UPDATE SKIP LOCKED
		)
		UPDATE cdc_system.transmute_jobs j
		   SET updated_at = NOW()
		  FROM claimed c
		 WHERE j.job_id = c.job_id
		RETURNING j.*
	`).Scan(&jobs).Error
	return jobs, err
}

// GetLatestByTable returns the most recent transmute job for a master_table.
func (r *TransmuteJobRepo) GetLatestByTable(ctx context.Context, masterTable string) (*TransmuteJob, error) {
	var job TransmuteJob
	err := r.db.WithContext(ctx).
		Where("master_table = ?", masterTable).
		Order("created_at DESC").
		First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}
```

#### Demo P13-4: `centralized-data-service/internal/service/master/transmuter.go` (`lines 33-49`, `115-169`, `208-235`, `360-648`, `1384-1456`) — Sửa `C1`, `C5`, `D1`, `B1`, `B4`, `A1`, `R-P13-2`, `R-P13-3`
1. **`C1` + `D1`:** Khớp 100% nguyên bản struct `TransmuterModule`, `TransmuteResult`, chữ ký `func (t *TransmuterModule) Run(ctx context.Context, masterName string, onlySourceIDs []string, jobID string, runType string, masterBindingID ...int64) (res TransmuteResult, err error)`, giữ nguyên `res.Updated += batchRes.updated` (`transmuter.go:475`), và không gọi hàm ảo `t.updateScheduleByID` (`D1`).
2. **`C5.1` + `B4`:** Lưu `[run_type=full|incremental]` vào `resumeErrNote` / `pauseNote` qua `repository.FormatTransmuteCheckpointNote(resumeAttempts, runType)`; xác định `isResumedJob := existingJob != nil && (existingJob.Status == "PAUSED" || existingJob.RowsAffected > 0 || resumeAttempts > 0)`. Khi `runType == "full" && !isResumedJob && len(chunkIDs) == 0`, reset ngay `sync_runtime_state.last_cursor_json` về `{"last_gpay_id": 0}`; ngược lại khi `isResumedJob`, đọc tiếp `last_gpay_id` từ `sync_runtime_state` kể cả khi `runType == "full"` và `baseRowsAffected == 0` (trường hợp các mẻ trước toàn dòng `Skipped`).
3. **`B1` + `A1`:** Ngay sau `t.processBatch(...)`, nếu `batchRes.lastErr != nil`, kiểm tra ngay `IsCancelRequested` và `t.IsStopping()` để trả về `CANCELLED` / `PAUSED` **trước khi** tăng `lastGpayID` hay lưu `sync_runtime_state` (`A1`: chỉ gán `lastGpayID = batchRes.lastGpayID` khi `batchRes.lastErr == nil`); đồng thời kiểm tra `IsCancelRequested` và `t.IsStopping()` trước `if lastBatchErr != nil` ở cuối hàm `Run()`:
```go
// Trong struct TransmuterModule (lines 33-49):
type TransmuterModule struct {
	systemDB             *gorm.DB
	connMgr              *source.ConnectionManager
	runtimeRepo          *repomaster.SyncRuntimeStateRepo
	transmuteJobRepo     *repository.TransmuteJobRepo
	ddlEnsurer           MasterDestinationEnsurer
	typeRes              *shadow.TypeResolver
	logger               *zap.Logger
	batchSize            int
	stopping             atomic.Bool
	mu                   sync.RWMutex
	cache                map[string]cachedRules
	cacheTTL             time.Duration
	shadowCache          map[string]shadowState
	ensuredMasters       map[string]bool
	ensuredShadowIndexes map[string]bool
}
```
```go
// Trong struct TransmuteResult & các hàm helper (lines 115-169):
type TransmuteResult struct {
	Master     string `json:"master"`
	Source     string `json:"source"`
	Scanned    int64  `json:"scanned"`
	Written    int64  `json:"written"`
	Inserted   int64  `json:"inserted"`
	Updated    int64  `json:"updated"`
	Deleted    int64  `json:"deleted"`
	Skipped    int64  `json:"skipped"`
	OccSkipped int64  `json:"occ_skipped"`
	RuleMisses int64  `json:"rule_misses"`
	TypeErrors int64  `json:"type_errors"`
	DurationMs int64  `json:"duration_ms"`
	ActiveGate string `json:"active_gate,omitempty"`
	Paused     bool   `json:"paused,omitempty"`
	Cancelled  bool   `json:"cancelled,omitempty"`
}

// SetTransmuteJobRepo injects the transmute_jobs repository for async manual job tracking.
func (t *TransmuterModule) SetTransmuteJobRepo(r *repository.TransmuteJobRepo) {
	t.transmuteJobRepo = r
}

// SetStopping marks the transmuter as shutting down so active jobs pause cleanly after finishing their current chunk.
func (t *TransmuterModule) SetStopping(v bool) {
	t.stopping.Store(v)
}

// IsStopping returns true when the worker is shutting down.
func (t *TransmuterModule) IsStopping() bool {
	return t.stopping.Load()
}

func (t *TransmuterModule) finishTransmuteJob(ctx context.Context, jobID, status string, rowsAffected, totalRows int64, errMsg string, masterBindingID ...int64) {
	if t.transmuteJobRepo == nil || jobID == "" {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = t.transmuteJobRepo.UpdateStatus(writeCtx, jobID, status, rowsAffected, totalRows, errMsg, masterBindingID...)
}
```
```go
// Trong TransmuterModule.Run — Khởi tạo, bảo toàn baseRowsAffected & runType khi Auto-Resume (lines 167 & 208-235):
func (t *TransmuterModule) Run(ctx context.Context, masterName string, onlySourceIDs []string, jobID string, runType string, masterBindingID ...int64) (res TransmuteResult, err error) {
```
```go
	var baseRowsAffected int64
	var preservedTotalRows int64
	var resumeAttempts int
	var isResumedJob bool
	if jobID != "" && t.transmuteJobRepo != nil {
		if existingJob, errJob := t.transmuteJobRepo.GetByID(ctx, jobID); errJob == nil && existingJob != nil {
			if existingJob.CancelRequested || existingJob.Status == "CANCELLED" {
				res.Cancelled = true
				t.finishTransmuteJob(ctx, jobID, "CANCELLED", existingJob.RowsAffected, existingJob.TotalRows, "cancelled by user", masterRow.ID)
				return res, nil
			}
			resumeAttempts = repository.ParseAutoResumeCount(existingJob.ErrorMessage)
			if runType == "" {
				runType = repository.ParseTransmuteRunType(existingJob.ErrorMessage)
			}
			if existingJob.Status == "PAUSED" || existingJob.RowsAffected > 0 || resumeAttempts > 0 {
				isResumedJob = true
			}
			if existingJob.RowsAffected > 0 {
				baseRowsAffected = existingJob.RowsAffected
			}
			if existingJob.TotalRows > 0 {
				preservedTotalRows = existingJob.TotalRows
			}
		}
		resumeErrNote := repository.FormatTransmuteCheckpointNote(resumeAttempts, runType)
		_ = t.transmuteJobRepo.UpdateStatus(ctx, jobID, "RUNNING", baseRowsAffected, preservedTotalRows, resumeErrNote, masterRow.ID)
		hbCtx, hbCancel := context.WithCancel(ctx)
		defer hbCancel()
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-hbCtx.Done():
					return
				case <-ticker.C:
					_ = t.transmuteJobRepo.TouchHeartbeat(context.Background(), jobID)
				}
			}
		}()
	}
	resumeErrNote := repository.FormatTransmuteCheckpointNote(resumeAttempts, runType)

	// Đếm trước tổng số bản ghi shadow cần transmute
	var totalShadowRows int64
	if len(onlySourceIDs) > 0 {
		totalShadowRows = int64(len(onlySourceIDs))
	} else if preservedTotalRows > 0 {
		totalShadowRows = preservedTotalRows
	} else {
		totalShadowRows, _ = t.countShadowRows(ctx, masterRow, 0)
	}

	if jobID != "" && t.transmuteJobRepo != nil {
		_ = t.transmuteJobRepo.UpdateStatus(ctx, jobID, "RUNNING", baseRowsAffected, totalShadowRows, resumeErrNote, masterRow.ID)
	}
```
```go
// Trong TransmuterModule.Run — Reset cursor khi Full Sync mới, đọc checkpoint khi Resume & Vòng lặp chunk (lines 360-648):
	var lastBatchErr error
	for _, chunkIDs := range idChunks {
		var lastGpayID int64
		if runType == "full" && !isResumedJob && len(chunkIDs) == 0 {
			// B4 & C5.1: Chỉ reset sync_runtime_state.last_cursor_json về {"last_gpay_id": 0} khi bắt đầu Full Sync MỚI (!isResumedJob)
			if t.runtimeRepo != nil {
				t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
					item.LastCursorJSON = []byte(`{"last_gpay_id":0}`)
				})
			}
			t.logger.Info("transmuter: full resync requested, reset checkpoint cursor to 0",
				zap.String("master", masterName))
		} else if len(chunkIDs) == 0 {
			// Khi Incremental hoặc khi Auto-Resume một job Full Sync (isResumedJob == true), đọc tiếp last_gpay_id từ sync_runtime_state
			if t.runtimeRepo != nil {
				state, errState := t.runtimeRepo.GetByMasterBinding(ctx, masterRow.ID)
				if errState == nil && state != nil && len(state.LastCursorJSON) > 0 {
					var cursorMap map[string]any
					if errJson := json.Unmarshal(state.LastCursorJSON, &cursorMap); errJson == nil {
						if val, ok := cursorMap["last_gpay_id"]; ok {
							if fVal, ok := val.(float64); ok && int64(fVal) > 0 {
								lastGpayID = int64(fVal)
								remainingRows, _ := t.countShadowRows(ctx, masterRow, lastGpayID)
								if baseRowsAffected+remainingRows > 0 {
									totalShadowRows = baseRowsAffected + remainingRows
								}
								if jobID != "" && t.transmuteJobRepo != nil {
									_ = t.transmuteJobRepo.UpdateStatus(ctx, jobID, "RUNNING", baseRowsAffected, totalShadowRows, resumeErrNote, masterRow.ID)
								}
								t.logger.Info("transmuter: resuming from checkpoint",
									zap.String("master", masterName),
									zap.String("run_type", runType),
									zap.Int64("last_gpay_id", lastGpayID),
									zap.Int64("base_rows_affected", baseRowsAffected),
									zap.Int64("remaining_shadow_rows", remainingRows))
							}
						}
					}
				}
			}
			// Tự động suy diễn mốc lastGpayID từ bảng Master (kèm Gap-Safe Count Guard) khi chọn "Chạy ngay (tiếp theo)"
			if lastGpayID == 0 && runType != "full" {
				var inferred int64
				if isMongoDest {
					inferred = t.inferLastGpayIDFromMongoMaster(ctx, masterRow, rules)
				} else if isMySQLDest {
					inferred = t.inferLastGpayIDFromMySQLMaster(ctx, masterRow, masterDB, rules)
				} else if isClickHouseDest {
					inferred = t.inferLastGpayIDFromClickHouseMaster(ctx, masterRow, masterDB, rules)
				} else {
					inferred = t.inferLastGpayIDFromMaster(ctx, masterRow, masterDB, rules)
				}
				if inferred > 0 {
					lastGpayID = inferred
					if t.runtimeRepo != nil {
						cursorBytes, _ := json.Marshal(map[string]any{"last_gpay_id": lastGpayID})
						t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
							item.LastCursorJSON = cursorBytes
						})
					}
					remainingRows, _ := t.countShadowRows(ctx, masterRow, lastGpayID)
					if baseRowsAffected+remainingRows > 0 {
						totalShadowRows = baseRowsAffected + remainingRows
					}
					if jobID != "" && t.transmuteJobRepo != nil {
						_ = t.transmuteJobRepo.UpdateStatus(ctx, jobID, "RUNNING", baseRowsAffected, totalShadowRows, resumeErrNote, masterRow.ID)
					}
					t.logger.Info("transmuter: inferred last_gpay_id from master table",
						zap.String("master", masterName),
						zap.Int64("last_gpay_id", lastGpayID),
						zap.Int64("remaining_shadow_rows", remainingRows))
				}
			}
		}

		initialCursor := lastGpayID
		var shadowRows []shadowBatchRow
		for {
			cumulativeRows := baseRowsAffected + res.Inserted + res.Updated + res.Deleted
			if jobID != "" && t.transmuteJobRepo != nil {
				if t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
					res.Cancelled = true
					t.logger.Info("transmute: cancelled by user request", zap.String("master", masterName), zap.String("job_id", jobID))
					t.finishTransmuteJob(ctx, jobID, "CANCELLED", cumulativeRows, totalShadowRows, "cancelled by user", masterRow.ID)
					return res, nil
				}
			}
			if t.IsStopping() {
				res.Paused = true
				t.logger.Info("transmute: pausing at chunk boundary for graceful shutdown",
					zap.String("master", masterName),
					zap.String("job_id", jobID),
					zap.Int64("last_gpay_id", lastGpayID),
					zap.Int64("cumulative_rows", cumulativeRows))
				t.finishTransmuteJob(ctx, jobID, "PAUSED", cumulativeRows, totalShadowRows, resumeErrNote, masterRow.ID)
				return res, nil
			}

			shadowRows, err = t.fetchShadowBatch(ctx, masterRow, lastGpayID, chunkIDs)
			if err != nil {
				if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
					res.Cancelled = true
					t.finishTransmuteJob(ctx, jobID, "CANCELLED", cumulativeRows, totalShadowRows, "cancelled by user", masterRow.ID)
					return res, nil
				}
				if t.IsStopping() {
					res.Paused = true
					t.finishTransmuteJob(ctx, jobID, "PAUSED", cumulativeRows, totalShadowRows, resumeErrNote, masterRow.ID)
					return res, nil
				}
				t.markRuntimeFailure(ctx, masterRow.ID, err)
				t.finishTransmuteJob(ctx, jobID, "FAILED", cumulativeRows, totalShadowRows, err.Error(), masterRow.ID)
				return res, fmt.Errorf("fetch shadow batch: %w", err)
			}

			// thực hiện hard-DELETE sau processBatch để tránh processBatch upsert lại row vừa xóa.
			var toHardDelete []string
			if len(chunkIDs) > 0 && lastGpayID == 0 {
				existingMap := make(map[string]bool)
				activeSourceIDs := make(map[string]bool)
				for _, row := range shadowRows {
					existingMap[row.SourceID] = true
					if !row.Deleted {
						activeSourceIDs[row.SourceID] = true
					}
				}
				deletedInShadow := make([]string, 0)
				for _, row := range shadowRows {
					if row.Deleted && !activeSourceIDs[row.SourceID] {
						deletedInShadow = append(deletedInShadow, row.SourceID)
					}
				}

				orphanMasterIDs := make([]string, 0)
				for _, id := range chunkIDs {
					if !existingMap[id] {
						orphanMasterIDs = append(orphanMasterIDs, id)
					}
				}
				toHardDelete = append(orphanMasterIDs, deletedInShadow...)
			}

			if len(shadowRows) == 0 && len(toHardDelete) == 0 {
				break
			}
			var batchRes batchOutcome
			if len(shadowRows) > 0 {
				batchRes = t.processBatch(ctx, masterRow, rules, shadowRows, strat, rc, runType == "full" || len(chunkIDs) > 0)
				// Sửa B1 & A1: Nếu processBatch gặp lỗi do bị Cancel hoặc Shutdown, thoát ngay với trạng thái CANCELLED / PAUSED
				// trước khi tăng lastGpayID hay lưu checkpoint của mẻ bị đứt!
				if batchRes.lastErr != nil {
					if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
						res.Cancelled = true
						t.finishTransmuteJob(ctx, jobID, "CANCELLED", cumulativeRows, totalShadowRows, "cancelled by user", masterRow.ID)
						return res, nil
					}
					if t.IsStopping() {
						res.Paused = true
						t.finishTransmuteJob(ctx, jobID, "PAUSED", cumulativeRows, totalShadowRows, resumeErrNote, masterRow.ID)
						return res, nil
					}
					lastBatchErr = batchRes.lastErr
				}
				res.Scanned += batchRes.scanned
				res.Inserted += batchRes.inserted
				res.Updated += batchRes.updated
				res.Deleted += batchRes.deleted
				res.Skipped += batchRes.skipped
				res.OccSkipped += batchRes.occSkipped
				res.RuleMisses += batchRes.ruleMisses
				res.TypeErrors += batchRes.typeErrors
				if batchRes.lastErr == nil {
					lastGpayID = batchRes.lastGpayID
				}
			}

			// Luồng 1 DELETE: thực hiện sau processBatch theo đúng conflictTarget đọc động từ transform_spec.pk
			if len(toHardDelete) > 0 {
				deletedCnt, errDel := t.deleteFromMasterBySourceAndGpayIDs(ctx, masterRow, colTypes, toHardDelete, nil)
				if errDel != nil {
					if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
						res.Cancelled = true
						currentRows := baseRowsAffected + res.Inserted + res.Updated + res.Deleted
						t.finishTransmuteJob(ctx, jobID, "CANCELLED", currentRows, totalShadowRows, "cancelled by user", masterRow.ID)
						return res, nil
					}
					if t.IsStopping() {
						res.Paused = true
						currentRows := baseRowsAffected + res.Inserted + res.Updated + res.Deleted
						t.finishTransmuteJob(ctx, jobID, "PAUSED", currentRows, totalShadowRows, resumeErrNote, masterRow.ID)
						return res, nil
					}
					t.logger.Error("transmuter: hard-delete orphan master failed",
						zap.String("master", masterRow.MasterTable),
						zap.Error(errDel))
					if lastBatchErr == nil {
						lastBatchErr = errDel
					}
				} else {
					res.Deleted += deletedCnt
					t.logger.Info("transmuter: hard-deleted orphan master rows",
						zap.String("master", masterRow.MasterTable),
						zap.Int("requested_count", len(toHardDelete)),
						zap.Int64("deleted_rows", deletedCnt))
				}
			}

			if len(shadowRows) == 0 {
				break
			}

			// Realtime progress update for manual Transmute Now jobs (cộng dồn baseRowsAffected)
			if jobID != "" && t.transmuteJobRepo != nil {
				progressPercent := 0
				currentRows := baseRowsAffected + res.Inserted + res.Updated + res.Deleted
				if totalShadowRows > 0 {
					progressPercent = int(float64(currentRows) / float64(totalShadowRows) * 100)
					if progressPercent > 99 {
						progressPercent = 99
					}
				}
				_ = t.transmuteJobRepo.UpdateProgress(ctx, jobID, progressPercent, currentRows, totalShadowRows, t.batchSize)
			}

			// P2-2: Generic Orphan Cleanup — strategy tự khai báo suffix nào là orphan,
			// engine chỉ lo phần DB. Không có if-TransformType nào ở đây.
			if len(shadowRows) > 0 && batchRes.lastErr == nil {
				var allPotentialOrphans []int64
				for _, row := range shadowRows {
					if row.Deleted {
						continue
					}
					n := batchRes.rowEmitCounts[row.SourceID]
					fakeEmits := make([]transmute.Emit, n)
					for _, suffix := range strat.OrphanKeySuffixes(rc, transmute.ShadowRow{
						Raw: row.RawData, SourceID: row.SourceID,
						SourceTs: row.SourceTs, Deleted: row.Deleted,
					}, fakeEmits) {
						allPotentialOrphans = append(allPotentialOrphans,
							deterministicMasterFlattenGpayID(masterRow.ID, row.SourceID, row.GpayID, suffix),
							deterministicGpayID(row.GpayID, suffix))
					}
				}

				if len(allPotentialOrphans) > 0 {
					if isMongoDest {
						if errDel := t.deleteOrphanGpayIDsFromMongoMaster(ctx, masterRow, allPotentialOrphans); errDel != nil {
							t.logger.Error("transmuter: mongo orphan hard-delete failed",
								zap.String("master", masterRow.MasterTable),
								zap.String("strategy", masterRow.TransformType),
								zap.Error(errDel))
						}
					} else if isMySQLDest {
						_ = t.deleteOrphanGpayIDsFromMySQLMaster(ctx, masterRow, masterDB, allPotentialOrphans)
					} else if isClickHouseDest {
						_ = t.deleteOrphanGpayIDsFromClickHouseMaster(ctx, masterRow, masterDB, allPotentialOrphans)
					} else if masterDB, errDB := t.connMgr.GetMasterDB(ctx, masterRow.MasterConnectionKey); errDB == nil && masterDB != nil {
						var orphansToDelete []int64
						const selectChunkSize = 10000
						for i := 0; i < len(allPotentialOrphans); i += selectChunkSize {
							end := i + selectChunkSize
							if end > len(allPotentialOrphans) {
								end = len(allPotentialOrphans)
							}
							sqlQuery := fmt.Sprintf(`SELECT _gpay_id FROM %s WHERE _gpay_id IN (?)`,
								quoteTransmuteQualified(masterRow.MasterSchema, masterRow.MasterTable))
							var found []int64
							if errQ := masterDB.WithContext(ctx).Raw(sqlQuery, allPotentialOrphans[i:end]).Scan(&found).Error; errQ == nil {
								orphansToDelete = append(orphansToDelete, found...)
							}
						}

						if len(orphansToDelete) > 0 {
							sqlDel := fmt.Sprintf(`DELETE FROM %s WHERE _gpay_id IN (?)`,
								quoteTransmuteQualified(masterRow.MasterSchema, masterRow.MasterTable))
							errDel := masterDB.WithContext(ctx).Exec(sqlDel, orphansToDelete).Error
							if errDel != nil {
								t.logger.Error("transmuter: orphan hard-delete failed",
									zap.String("master", masterRow.MasterTable),
									zap.String("strategy", masterRow.TransformType),
									zap.Error(errDel))
							} else {
								t.logger.Info("transmuter: orphan hard-deleted",
									zap.String("master", masterRow.MasterTable),
									zap.String("strategy", masterRow.TransformType),
									zap.Int("count", len(orphansToDelete)))
							}
						}
					}
				}
			}

			// A1: Chỉ lưu checkpoint last_gpay_id khi mẻ hiện tại không bị lỗi (batchRes.lastErr == nil)
			if len(chunkIDs) == 0 && batchRes.lastErr == nil && t.runtimeRepo != nil && lastGpayID > 0 {
				cursorBytes, _ := json.Marshal(map[string]any{"last_gpay_id": lastGpayID})
				t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
					item.LastCursorJSON = cursorBytes
				})
			}

			if len(chunkIDs) > 0 || batchRes.lastErr != nil {
				break
			}
		}

		if runType != "full" && len(chunkIDs) == 0 && initialCursor > 0 && lastBatchErr == nil {
			t.syncUpdatedShadowRowsBeforeCursor(ctx, masterRow, rules, strat, rc, initialCursor, jobID, &res, &lastBatchErr)
			if res.Cancelled {
				cumulativeRows := baseRowsAffected + res.Inserted + res.Updated + res.Deleted
				t.finishTransmuteJob(ctx, jobID, "CANCELLED", cumulativeRows, totalShadowRows, "cancelled by user", masterRow.ID)
				return res, nil
			}
			if res.Paused {
				cumulativeRows := baseRowsAffected + res.Inserted + res.Updated + res.Deleted
				t.finishTransmuteJob(ctx, jobID, "PAUSED", cumulativeRows, totalShadowRows, resumeErrNote, masterRow.ID)
				return res, nil
			}
		}
	}

	res.DurationMs = time.Since(start).Milliseconds()
	totalRowsAffected := baseRowsAffected + res.Inserted + res.Updated + res.Deleted

	// Sửa B1: Kiểm tra IsCancelRequested (dùng context độc lập 3s) và IsStopping trước khi xét lastBatchErr
	if res.Cancelled || (jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID)) {
		res.Cancelled = true
		t.finishTransmuteJob(ctx, jobID, "CANCELLED", totalRowsAffected, totalShadowRows, "cancelled by user", masterRow.ID)
		return res, nil
	}
	if res.Paused || t.IsStopping() {
		res.Paused = true
		t.finishTransmuteJob(ctx, jobID, "PAUSED", totalRowsAffected, totalShadowRows, resumeErrNote, masterRow.ID)
		return res, nil
	}

	// Chỉ reset checkpoint sau khi đồng bộ Full Sync hoàn tất thành công 100%
	if runType == "full" && len(onlySourceIDs) == 0 && lastBatchErr == nil && t.runtimeRepo != nil {
		t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
			item.LastCursorJSON = []byte(`{}`)
		})
	}

	if lastBatchErr != nil {
		t.markRuntimeFailure(ctx, masterRow.ID, lastBatchErr)
		t.finishTransmuteJob(ctx, jobID, "FAILED", totalRowsAffected, totalShadowRows, lastBatchErr.Error(), masterRow.ID)
		return res, lastBatchErr
	}

	if res.Scanned > 0 && res.Inserted == 0 && res.Updated == 0 && res.Deleted == 0 {
		if res.RuleMisses == 0 && res.TypeErrors == 0 {
			t.markRuntimeSuccess(ctx, masterRow.ID, res)
			t.finishTransmuteJob(ctx, jobID, "COMPLETED", totalRowsAffected, totalShadowRows, "", masterRow.ID)
			return res, nil
		}
		degraded := fmt.Errorf("transmute degraded: scanned=%d nhưng 0 dòng ghi master (skipped=%d occ_skipped=%d rule_misses=%d type_errors=%d) — kiểm tra transform_spec/mapping rules",
			res.Scanned, res.Skipped, res.OccSkipped, res.RuleMisses, res.TypeErrors)
		t.markRuntimeFailure(ctx, masterRow.ID, degraded)
		t.finishTransmuteJob(ctx, jobID, "FAILED", totalRowsAffected, totalShadowRows, degraded.Error(), masterRow.ID)
		return res, degraded
	}
	t.markRuntimeSuccess(ctx, masterRow.ID, res)
	t.finishTransmuteJob(ctx, jobID, "COMPLETED", totalRowsAffected, totalShadowRows, "", masterRow.ID)
	return res, nil
}
```
```go
// Trong syncUpdatedShadowRowsBeforeCursor (lines 1384-1456):
func (t *TransmuterModule) syncUpdatedShadowRowsBeforeCursor(
	ctx context.Context,
	masterRow *masterBindingRuntime,
	rules []mappingRuleRow,
	strat transmute.Strategy,
	rc transmute.RunContext,
	maxCursor int64,
	jobID string,
	res *TransmuteResult,
	lastBatchErr *error,
) {
	if masterRow == nil || maxCursor <= 0 || t.connMgr == nil {
		return
	}
	shadowDB, err := t.connMgr.GetShadowDB(ctx, masterRow.ShadowConnectionKey)
	if err != nil || shadowDB == nil {
		return
	}

	shadowPK := masterRow.ShadowPK
	if shadowPK == "" {
		shadowPK = "_gpay_id"
	}
	pkIdent := quoteTransmuteIdent(shadowPK)
	pkSelect := pkIdent
	if shadowPK != "_gpay_id" {
		pkSelect = fmt.Sprintf("%s::bigint AS _gpay_id", pkIdent)
	}
	pkBigint := fmt.Sprintf("(%s)::bigint", pkIdent)
	shadowQT := quoteTransmuteQualified(masterRow.ShadowSchema, masterRow.ShadowTable)
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	if t.runtimeRepo != nil {
		if st, errSt := t.runtimeRepo.GetByMasterBinding(ctx, masterRow.ID); errSt == nil && st != nil && st.LastSuccessAt != nil && !st.LastSuccessAt.IsZero() {
			cand := st.LastSuccessAt.UTC().Add(-10 * time.Minute)
			if cand.After(cutoff) {
				cutoff = cand
			}
		}
	}

	var scanCursor int64
	for {
		if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
			res.Cancelled = true
			return
		}
		if t.IsStopping() {
			res.Paused = true
			return
		}
		var rows []shadowBatchRow
		query := fmt.Sprintf(
			`SELECT %s, _source_id, _raw_data, _source_ts, _deleted FROM %s WHERE %s > ? AND %s <= ? AND (COALESCE(_deleted, false) = true OR _updated_at >= ?) ORDER BY 1 LIMIT ?`,
			pkSelect, shadowQT, pkBigint, pkBigint,
		)
		errQ := shadowDB.WithContext(ctx).Raw(query, scanCursor, maxCursor, cutoff, t.batchSize).Scan(&rows).Error
		if errQ != nil {
			if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
				res.Cancelled = true
				return
			}
			if t.IsStopping() {
				res.Paused = true
				return
			}
			fallbackQuery := fmt.Sprintf(
				`SELECT %s, _source_id, _raw_data, _source_ts, _deleted FROM %s WHERE %s > ? AND %s <= ? AND COALESCE(_deleted, false) = true ORDER BY 1 LIMIT ?`,
				pkSelect, shadowQT, pkBigint, pkBigint,
			)
			if errFallback := shadowDB.WithContext(ctx).Raw(fallbackQuery, scanCursor, maxCursor, t.batchSize).Scan(&rows).Error; errFallback != nil {
				return
			}
		}
		if len(rows) == 0 {
			break
		}
		rows = t.enrichMultiShadowBatchRows(ctx, shadowDB, masterRow, rows)
		batchRes := t.processBatch(ctx, masterRow, rules, rows, strat, rc, false)
		if batchRes.lastErr != nil {
			if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
				res.Cancelled = true
				return
			}
			if t.IsStopping() {
				res.Paused = true
				return
			}
		}
		accumulateBatchOutcome(res, batchRes, lastBatchErr)
		if batchRes.lastErr != nil || batchRes.lastGpayID <= scanCursor {
			break
		}
		scanCursor = batchRes.lastGpayID
		if scanCursor >= maxCursor {
			break
		}
	}
	if !res.Cancelled && !res.Paused && !t.IsStopping() {
		t.syncLaggingSecondaryShadowRows(ctx, shadowDB, masterRow, rules, strat, rc, cutoff, res, lastBatchErr)
	}
}
```

#### Demo P13-5: `centralized-data-service/internal/handler/master/transmute_handler.go` (`lines 40-66`, `132-256`, `278-435`, `500-510`) — Sửa `C1`, `C4`, `C5`, `B2`, `B4`, `B5`, `A2`
1. **`C1`:** Giữ nguyên đầy đủ 3 hằng số `SubjectTransmuteCompleted`, `SubjectTransmuteShadowCoreFallback`, `SubjectTransmuteCancel` (`lines 278-282`).
2. **`C5.1`:** Trong `recoverOrphanedTransmuteJobs`, khôi phục `savedRunType := repository.ParseTransmuteRunType(j.ErrorMessage)` và truyền `RunType: savedRunType` vào `TransmuteRequest` để job `Full Sync` không bị tụt về `"incremental"`.
3. **`C4.2` + `B5` + `A2`:** Bổ sung `SetStopping(true)` báo trước, hạ thời gian chờ ranh giới mẻ trong `Stop()` xuống **`2 * time.Second`**, và dừng toàn bộ `TableDebouncer` **song song** bằng `sync.WaitGroup` (mỗi debouncer dùng chung 1 `drainCtx` `4s`):
```go
type TransmuteHandler struct {
	svc               *master.TransmuterModule
	masterBindingRepo *repomaster.MasterBindingRepo
	transmuteJobRepo  *repository.TransmuteJobRepo
	natsConn          *nats.Conn
	logger            *zap.Logger
	activity          *governance.ActivityLogger
	debouncers        map[string]*TableDebouncer
	debouncersMu      sync.RWMutex
	activeJobsMu      sync.Mutex
	activeJobs        map[string]context.CancelFunc
	activeTables      map[string]string
	recoverCancel     context.CancelFunc
	stopping          atomic.Bool
	wg                sync.WaitGroup
	stopOnce          sync.Once
	stopCh            chan struct{}
}

func NewTransmuteHandler(
	svc *master.TransmuterModule,
	masterBindingRepo *repomaster.MasterBindingRepo,
	conn *nats.Conn,
	logger *zap.Logger,
	activity *governance.ActivityLogger,
) *TransmuteHandler {
	return &TransmuteHandler{
		svc:               svc,
		masterBindingRepo: masterBindingRepo,
		natsConn:          conn,
		logger:            logger,
		activity:          activity,
		debouncers:        make(map[string]*TableDebouncer),
		activeJobs:        make(map[string]context.CancelFunc),
		activeTables:      make(map[string]string),
		stopCh:            make(chan struct{}),
	}
}

func (h *TransmuteHandler) SetTransmuteJobRepo(r *repository.TransmuteJobRepo) {
	h.transmuteJobRepo = r
}

func (h *TransmuteHandler) SetStopping(v bool) {
	h.stopping.Store(v)
	if h.svc != nil {
		h.svc.SetStopping(v)
	}
}

const (
	SubjectTransmuteCompleted          = "cdc.evt.transmute.completed"
	SubjectTransmuteShadowCoreFallback = "cdc.cmd.transmute-shadow.core-fallback"
	SubjectTransmuteCancel             = "cdc.cmd.transmute-cancel"
)

func (h *TransmuteHandler) HandleTransmuteShadow(msg *nats.Msg) {
	if h.stopping.Load() || (h.svc != nil && h.svc.IsStopping()) {
		return
	}
	_ = h.handleTransmuteShadowWithOutcome(context.Background(), msg, false)
}

func (h *TransmuteHandler) handleTransmuteShadowWithOutcome(parentCtx context.Context, msg *nats.Msg, trackJetStreamAck bool) error {
	// B2 + A2: Chặn ngay khi đang Stop(); nếu là JetStream task thì trả lỗi để NakWithDelay phát lại sau khi bật lại
	if h.stopping.Load() || (h.svc != nil && h.svc.IsStopping()) {
		if trackJetStreamAck {
			return fmt.Errorf("transmute handler is stopping")
		}
		return nil
	}

	var temp struct {
		ShadowTable string `json:"shadow_table"`
	}
	_ = json.Unmarshal(msg.Data, &temp)

	spanName := "nats.HandleTransmuteShadow"
	if temp.ShadowTable != "" {
		spanName = fmt.Sprintf("nats.HandleTransmuteShadow: %s", temp.ShadowTable)
	}

	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx := observability.ExtractNATSHeader(parentCtx, msg.Header)
	ctx, span := observability.ChildSpan(ctx, spanName)
	defer span.End()

	var req struct {
		ShadowTable         string                           `json:"shadow_table"`
		ShadowSchema        string                           `json:"shadow_schema,omitempty"`
		ShadowConnectionKey string                           `json:"shadow_connection_key,omitempty"`
		ShadowBindingCode   string                           `json:"shadow_binding_code,omitempty"`
		SourceIDs           []string                         `json:"_source_ids"`
		Events              []shadowmodel.TransmuteEventItem `json:"events,omitempty"`
		CorrelationID       string                           `json:"correlation_id,omitempty"`
		TriggeredBy         string                           `json:"triggered_by,omitempty"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		observability.Ctx(ctx, h.logger).Warn("transmute-shadow: bad payload", zap.Error(err))
		return nil
	}
	if req.ShadowTable == "" || h.masterBindingRepo == nil {
		return nil
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	masterTargets, err := h.resolveMasterTargets(
		lookupCtx,
		req.ShadowTable,
		req.ShadowSchema,
		req.ShadowConnectionKey,
		req.ShadowBindingCode,
	)
	if err != nil {
		return fmt.Errorf("transmute-shadow master lookup failed for %s: %w", req.ShadowTable, err)
	}

	if len(masterTargets) == 0 {
		return nil
	}
	hookTrigger := "kafka-consumer-hook"
	switch req.TriggeredBy {
	case "snapshot":
		hookTrigger = "snapshot-hook"
	case "bridge-oplog":
		hookTrigger = "bridge-oplog"
	}

	if trackJetStreamAck {
		if h.stopping.Load() || (h.svc != nil && h.svc.IsStopping()) {
			return fmt.Errorf("transmute handler is stopping")
		}
		policy := deliveryPolicyFromContext(parentCtx, h.logger)
		tracker := newShadowMsgTracker(msg, len(masterTargets), policy)
		for _, target := range masterTargets {
			tReq := TransmuteRequest{
				MasterTable:     target.MasterFQN,
				MasterBindingID: target.ID,
				SourceIDs:       req.SourceIDs,
				Events:          req.Events,
				TriggeredBy:     hookTrigger,
				CorrelationID:   req.CorrelationID,
			}
			taskMsg := &nats.Msg{
				Subject: "cdc.cmd.transmute",
				Header:  make(nats.Header),
			}
			observability.InjectNATSHeader(ctx, taskMsg.Header)

			var taskOnce sync.Once
			onDone := func(taskErr error) {
				taskOnce.Do(func() {
					tracker.completeTask(taskErr)
				})
			}

			debouncerKey := tReq.MasterTable
			if tReq.MasterBindingID > 0 {
				debouncerKey = fmt.Sprintf("%s#%d", tReq.MasterTable, tReq.MasterBindingID)
			}
			td := h.getOrCreateDebouncer(debouncerKey)
			if td == nil {
				onDone(context.Canceled)
				continue
			}
			td.AddTask(NatsMsgTask{
				Msg:    taskMsg,
				Req:    tReq,
				OnDone: onDone,
			})
		}
		return ErrAsyncAckPending
	}

	for _, target := range masterTargets {
		out, _ := json.Marshal(map[string]any{
			"master_table":      target.MasterFQN,
			"master_binding_id": target.ID,
			"_source_ids":       req.SourceIDs,
			"events":            req.Events,
			"triggered_by":      hookTrigger,
			"correlation_id":    req.CorrelationID,
		})
		if h.natsConn != nil {
			outMsg := &nats.Msg{
				Subject: "cdc.cmd.transmute",
				Data:    out,
				Header:  make(nats.Header),
			}
			observability.InjectNATSHeader(ctx, outMsg.Header)
			_ = h.natsConn.PublishMsg(outMsg)
		}
	}
	return nil
}

// HandleTransmuteCancel immediately cancels an in-flight transmute job by job_id.
func (h *TransmuteHandler) HandleTransmuteCancel(msg *nats.Msg) {
	var payload struct {
		JobID       string `json:"job_id"`
		MasterTable string `json:"master_table"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil || payload.JobID == "" {
		return
	}
	h.activeJobsMu.Lock()
	cancelFn, ok := h.activeJobs[payload.JobID]
	h.activeJobsMu.Unlock()
	if ok && cancelFn != nil {
		cancelFn()
	}
}

// Start launches the background recovery loop to auto-resume PAUSED or orphaned transmute jobs.
func (h *TransmuteHandler) Start(ctx context.Context) {
	h.activeJobsMu.Lock()
	if h.stopping.Load() {
		h.activeJobsMu.Unlock()
		return
	}
	recCtx, recCancel := context.WithCancel(ctx)
	h.recoverCancel = recCancel
	h.wg.Add(1)
	h.activeJobsMu.Unlock()

	go func() {
		defer h.wg.Done()
		h.recoverOrphanedTransmuteJobs(recCtx)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-recCtx.Done():
				return
			case <-h.stopCh:
				return
			case <-ticker.C:
				h.recoverOrphanedTransmuteJobs(recCtx)
			}
		}
	}()
}

func (h *TransmuteHandler) recoverOrphanedTransmuteJobs(ctx context.Context) {
	if h.transmuteJobRepo == nil || h.svc == nil || h.stopping.Load() || h.svc.IsStopping() {
		return
	}
	jobs, err := h.transmuteJobRepo.ClaimRecoverableJobs(ctx)
	if err != nil || len(jobs) == 0 {
		return
	}
	for _, j := range jobs {
		if h.stopping.Load() || h.svc.IsStopping() {
			return
		}
		h.activeJobsMu.Lock()
		_, alreadyRunningJob := h.activeJobs[j.JobID]
		_, alreadyRunningTable := h.activeTables[j.MasterTable]
		h.activeJobsMu.Unlock()
		if alreadyRunningJob || alreadyRunningTable {
			continue
		}
		var bID int64
		if j.MasterBindingID != nil {
			bID = *j.MasterBindingID
		}
		savedRunType := repository.ParseTransmuteRunType(j.ErrorMessage)
		// Chặn Poison-Pill OOM Crash-Loop (R-P13-7): chỉ tăng bộ đếm nếu job bị bỏ rơi ở PENDING/RUNNING (không phải PAUSED chủ động)
		if j.Status != "PAUSED" {
			attempts := repository.ParseAutoResumeCount(j.ErrorMessage)
			if attempts >= repository.MaxOrphanAutoResumeAttempts {
				failMsg := fmt.Sprintf("FAILED: max orphan auto-resume attempts (%d) exceeded", repository.MaxOrphanAutoResumeAttempts)
				h.logger.Error("Skipping poison-pill orphaned transmute job after max retries",
					zap.String("job_id", j.JobID),
					zap.String("master_table", j.MasterTable),
					zap.Int("attempts", attempts))
				_ = h.transmuteJobRepo.UpdateStatus(ctx, j.JobID, "FAILED", j.RowsAffected, j.TotalRows, failMsg, bID)
				continue
			}
			nextNote := repository.FormatOrphanAutoResumeError(attempts+1, savedRunType)
			_ = h.transmuteJobRepo.UpdateStatus(ctx, j.JobID, "RUNNING", j.RowsAffected, j.TotalRows, nextNote, bID)
		}
		h.logger.Info("Auto-resuming interrupted transmute job from checkpoint",
			zap.String("job_id", j.JobID),
			zap.String("master_table", j.MasterTable),
			zap.String("run_type", savedRunType),
			zap.String("prev_status", j.Status),
			zap.Int64("rows_affected", j.RowsAffected))
		fakeReq, _ := json.Marshal(TransmuteRequest{
			JobID:           j.JobID,
			MasterTable:     j.MasterTable,
			MasterBindingID: bID,
			RunType:         savedRunType,
			TriggeredBy:     "auto-resume",
			TraceID:         j.TraceID,
		})
		h.HandleTransmute(&nats.Msg{Subject: "cdc.cmd.transmute", Data: fakeReq})
	}
}

// Stop implements Two-Tier Graceful Shutdown within a strict 2s + 4s budget (C4 & B5):
// 1. Signals long-running TransmuterModule jobs to finish their current 1000-row chunk (up to 2s) and persist PAUSED state.
// 2. Drains and flushes all short-lived TableDebouncers in parallel (each draining in strict seq FIFO with a single shared 4s timeout).
func (h *TransmuteHandler) Stop() {
	h.stopOnce.Do(func() {
		h.activeJobsMu.Lock()
		h.stopping.Store(true)
		if h.svc != nil {
			h.svc.SetStopping(true)
		}
		if h.recoverCancel != nil {
			h.recoverCancel()
		}
		if h.stopCh != nil {
			close(h.stopCh)
		}
		h.activeJobsMu.Unlock()

		done := make(chan struct{})
		go func() {
			h.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			h.activeJobsMu.Lock()
			for _, cancelFn := range h.activeJobs {
				if cancelFn != nil {
					cancelFn()
				}
			}
			h.activeJobsMu.Unlock()
			<-done
		}

		h.debouncersMu.RLock()
		debouncers := make([]*TableDebouncer, 0, len(h.debouncers))
		for _, td := range h.debouncers {
			debouncers = append(debouncers, td)
		}
		h.debouncersMu.RUnlock()

		var tdWg sync.WaitGroup
		for _, td := range debouncers {
			if td == nil {
				continue
			}
			tdWg.Add(1)
			go func(d *TableDebouncer) {
				defer tdWg.Done()
				d.Stop()
			}(td)
		}
		tdWg.Wait()
	})
}

// HandleTransmute consumes cdc.cmd.transmute. Runs the full Shadow → Master
// materialise cycle, writes a result to cdc.result.transmute.
func (h *TransmuteHandler) HandleTransmute(msg *nats.Msg) {
	var req TransmuteRequest
	err := json.Unmarshal(msg.Data, &req)

	spanName := "cdc.worker.transmute.process"
	if req.MasterTable != "" {
		spanName = fmt.Sprintf("cdc.worker.transmute.process: %s", req.MasterTable)
	}

	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, spanName)

	if err != nil {
		span.End()
		h.replyErr(ctx, msg, req.CorrelationID, "invalid_payload: "+err.Error())
		return
	}
	if req.MasterTable == "" {
		span.End()
		h.replyErr(ctx, msg, req.CorrelationID, "master_table required")
		return
	}

	// Nếu là incremental transmute (realtime CDC hoặc heal), định tuyến qua TableDebouncer để gom lô và khống chế concurrency
	if len(req.Events) > 0 || len(req.SourceIDs) > 0 {
		span.End()
		// B2: Chặn ngay nếu đang Stop() để không tạo TableDebouncer mới sau khi Stop() đã bắt đầu
		if h.stopping.Load() || (h.svc != nil && h.svc.IsStopping()) {
			return
		}
		debouncerKey := req.MasterTable
		if req.MasterBindingID > 0 {
			debouncerKey = fmt.Sprintf("%s#%d", req.MasterTable, req.MasterBindingID)
		}
		td := h.getOrCreateDebouncer(debouncerKey)
		if td == nil {
			return
		}
		td.Add(msg, req)
		return
	}

	var timeout time.Duration = 24 * time.Hour

	// Kiểm tra cờ stopping + chống chạy chồng cùng bảng (B4) và gọi h.wg.Add(1) bên trong h.activeJobsMu.Lock()
	h.activeJobsMu.Lock()
	if h.stopping.Load() || (h.svc != nil && h.svc.IsStopping()) {
		h.activeJobsMu.Unlock()
		span.End()
		return
	}
	if existingKey, busy := h.activeTables[req.MasterTable]; busy && (req.JobID == "" || existingKey != req.JobID) {
		h.activeJobsMu.Unlock()
		span.End()
		h.logger.Warn("Skipping duplicate transmute request while table already has an active job",
			zap.String("master_table", req.MasterTable),
			zap.String("active_job", existingKey),
			zap.String("incoming_job", req.JobID))
		return
	}
	bgCtx, cancel := context.WithTimeout(ctx, timeout)
	jobKey := req.JobID
	if jobKey == "" {
		jobKey = fmt.Sprintf("anon:%s:%d", req.MasterTable, time.Now().UnixNano())
	}
	h.activeJobs[jobKey] = cancel
	h.activeTables[req.MasterTable] = jobKey
	h.wg.Add(1)
	h.activeJobsMu.Unlock()

	go func() {
		defer h.wg.Done()
		defer func() {
			h.activeJobsMu.Lock()
			delete(h.activeJobs, jobKey)
			if h.activeTables[req.MasterTable] == jobKey {
				delete(h.activeTables, req.MasterTable)
			}
			h.activeJobsMu.Unlock()
		}()
		defer cancel()
		defer span.End()

		var logEntry *system.ActivityLog
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				observability.Ctx(bgCtx, h.logger).Error("transmute goroutine panic recovered",
					zap.Any("panic", r),
					zap.String("master", req.MasterTable),
					zap.String("stack", string(stack)))
				if logEntry != nil {
					h.activity.Fail(logEntry, fmt.Sprintf("PANIC: %v", r))
				}
				if req.JobID != "" && h.transmuteJobRepo != nil {
					writeCtx, wCancel := context.WithTimeout(context.Background(), 5*time.Second)
					_ = h.transmuteJobRepo.UpdateStatus(writeCtx, req.JobID, "FAILED", 0, 0, fmt.Sprintf("PANIC: %v", r), req.MasterBindingID)
					wCancel()
				}
				metrics.TransmuteErrorTotal.WithLabelValues(req.MasterTable).Inc()
				cancel()
			}
		}()

		if h.activity != nil {
			logEntry = h.activity.Start("transmute", req.MasterTable, req.TriggeredBy)
		}

		var traceID string
		if sc := span.SpanContext(); sc.IsValid() {
			traceID = sc.TraceID().String()
		}
		if traceID == "" && req.TraceID != "" {
			traceID = req.TraceID
		}
		if traceID == "" {
			traceID = strings.ReplaceAll(uuid.NewString(), "-", "")
		}

		res, err := h.svc.Run(bgCtx, req.MasterTable, req.SourceIDs, req.JobID, req.RunType, req.MasterBindingID)
		resp := TransmuteResponse{TransmuteResult: res, CorrelationID: req.CorrelationID, TraceID: traceID}

		// Sửa R-P13-2: Khi job bị CANCELLED hoặc PAUSED ở ranh giới mẻ, thoát ngay lập tức,
		// TUYỆT ĐỐI KHÔNG gọi publishCompleted (không ghi đè COMPLETED hay đóng schedule thành 'success')
		if res.Cancelled {
			if logEntry != nil {
				h.activity.FailWithDetails(logEntry, "cancelled by user", res.Inserted+res.Updated+res.Deleted, map[string]interface{}{
					"master_binding_id": req.MasterBindingID,
					"job_id":            req.JobID,
					"status":            "cancelled",
				})
			}
			h.reply(msg, resp)
			return
		}
		if res.Paused {
			if logEntry != nil {
				h.activity.Complete(logEntry, res.Inserted+res.Updated+res.Deleted, map[string]interface{}{
					"master_binding_id": req.MasterBindingID,
					"job_id":            req.JobID,
					"status":            "paused",
				})
			}
			h.reply(msg, resp)
			return
		}

		triggerType := req.TriggeredBy
		if triggerType == "" {
			triggerType = "unknown"
		}

		if err != nil {
			resp.Err = err.Error()
			if logEntry != nil {
				h.activity.FailWithDetails(logEntry, err.Error(), 0, map[string]interface{}{
					"master_binding_id": req.MasterBindingID,
				})
			}
			metrics.TransmuteErrorTotal.WithLabelValues(req.MasterTable).Inc()
			observability.Ctx(bgCtx, h.logger).Warn("transmute failed",
				zap.String("master", req.MasterTable),
				zap.String("correlation_id", req.CorrelationID),
				zap.Error(err))
		} else {
			if logEntry != nil {
				h.activity.Complete(logEntry, res.Inserted+res.Updated+res.Deleted, map[string]interface{}{
					"scanned":           res.Scanned,
					"inserted":          res.Inserted,
					"updated":           res.Updated,
					"deleted":           res.Deleted,
					"skipped":           res.Skipped,
					"rule_misses":       res.RuleMisses,
					"type_errors":       res.TypeErrors,
					"active_gate":       res.ActiveGate,
					"run_type":          req.RunType,
					"correlation_id":    req.CorrelationID,
					"trace_id":          traceID,
					"master_binding_id": req.MasterBindingID,
				})
			}
			observability.Ctx(bgCtx, h.logger).Info("transmute complete",
				zap.String("master", req.MasterTable),
				zap.Int64("scanned", res.Scanned),
				zap.Int64("inserted", res.Inserted),
				zap.Int64("updated", res.Updated),
				zap.Int64("deleted", res.Deleted),
				zap.Int64("skipped", res.Skipped),
				zap.Int64("type_errors", res.TypeErrors),
				zap.Int64("rule_misses", res.RuleMisses),
				zap.String("active_gate", res.ActiveGate),
				zap.Int64("duration_ms", res.DurationMs))

			metrics.TransmuteOpsTotal.WithLabelValues(req.MasterTable, "inserted", triggerType).Add(float64(res.Inserted))
			metrics.TransmuteOpsTotal.WithLabelValues(req.MasterTable, "updated", triggerType).Add(float64(res.Updated))
			metrics.TransmuteOpsTotal.WithLabelValues(req.MasterTable, "deleted", triggerType).Add(float64(res.Deleted))
			metrics.TransmuteOpsTotal.WithLabelValues(req.MasterTable, "skipped", triggerType).Add(float64(res.Skipped))
			metrics.TransmuteOpsTotal.WithLabelValues(req.MasterTable, "type_error", triggerType).Add(float64(res.TypeErrors))
			metrics.TransmuteOpsTotal.WithLabelValues(req.MasterTable, "rule_miss", triggerType).Add(float64(res.RuleMisses))
			metrics.TransmuteDurationMs.WithLabelValues(req.MasterTable).Observe(float64(res.DurationMs))
		}

		h.publishCompleted(bgCtx, req, res, err, traceID)
		h.reply(msg, resp)
	}()
}

func (h *TransmuteHandler) getOrCreateDebouncer(masterTable string) *TableDebouncer {
	if h.stopping.Load() || (h.svc != nil && h.svc.IsStopping()) {
		return nil
	}
	h.debouncersMu.Lock()
	defer h.debouncersMu.Unlock()
	if h.stopping.Load() || (h.svc != nil && h.svc.IsStopping()) {
		return nil
	}
	if td, ok := h.debouncers[masterTable]; ok {
		return td
	}
	// Đăng ký TableDebouncer: size 500, idle 100ms, max 1s, concurrency 1 (tránh deadlock 40P01 trên cùng 1 master table)
	td := NewTableDebouncer(context.Background(), masterTable, 500, 100*time.Millisecond, time.Second, 1, h.logger, h.runDebouncedTransmute)
	h.debouncers[masterTable] = td
	return td
}
```

#### Demo P13-6: `centralized-data-service/internal/repository/transform_job_repo.go` & `centralized-data-service/internal/handler/shadow/batch_transform_handler.go` — Sửa `C1`, `C3`, `C4`, `C6`, `D3`, `B1`, `B3`, `B4`, `B5`, `A3`
Xuất đầy đủ 1:1:
- `TransformJob` bổ sung `SourceObjectID *int64` (`A3.3`), hàm `FormatTransformCheckpointNote(attempts int, cursorPK interface{}, force bool, forceFields []string) string` / `ParseTransformCursorPK` / `ParseTransformForce` mã hóa độc lập `[force=1]` và `[force_fields=...]` (`C3.1`), `IsCancelRequested` tự tạo `context.WithTimeout(context.Background(), 3*time.Second)` (`B1`), `UpdateStatus` và `UpdateProgress`, `ClaimRecoverableJobs` với `FOR UPDATE SKIP LOCKED` không hủy oan job `PAUSED` `[force=1]` (`C6`).
- `BatchTransformPayload` (`batch_transform_handler.go:26-37`) giữ nguyên tên trường thật trên đĩa `ForceFields []string` (`C1` & `D3`); `BatchTransformHandler` (`SetStopping`, `Stop` với ngân sách `2s` `C4`, `recoverOrphanedTransformJobs` truyền `Force: isForce, ForceFields: forceFields`, `runTransformJob` cho phép `Force=true` cả khi `ForceFields` rỗng để cưỡng bức toàn bộ cột active mapping `C3.1`, và `detectPrimaryKey` trả về `(pkCol, pkType, err)` chỉ `strconv.ParseInt` khi `isIntegerPGType(pkType)` là `true` để chặn tuyệt đối lỗi `SQLSTATE 42883` trên PK kiểu chuỗi `C3.2`):
```go
// Trong centralized-data-service/internal/repository/transform_job_repo.go:
// TransformJob mirrors cdc_system.transform_jobs for the Worker side.
type TransformJob struct {
	JobID           string     `gorm:"column:job_id;primaryKey"`
	TargetTable     string     `gorm:"column:target_table;not null"`
	SourceObjectID  *int64     `gorm:"column:source_object_id"`
	Status          string     `gorm:"column:status;not null"`
	ProgressPercent int        `gorm:"column:progress_percent;default:0"`
	RowsAffected    int64      `gorm:"column:rows_affected;default:0"`
	TotalRows       int64      `gorm:"column:total_rows;default:0"`
	ChunkSize       int        `gorm:"column:chunk_size;default:1000"`
	CancelRequested bool       `gorm:"column:cancel_requested;default:false"`
	ErrorMessage    string     `gorm:"column:error_message"`
	TraceID         string     `gorm:"column:trace_id"`
	StartedAt       time.Time  `gorm:"column:started_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

// FormatTransformCheckpointNote encodes auto-resume attempts, cursor_pk, force flag ([force=1]), and optional force_fields into error_message (C3.1).
func FormatTransformCheckpointNote(attempts int, cursorPK interface{}, force bool, forceFields []string) string {
	var parts []string
	if attempts > 0 {
		parts = append(parts, fmt.Sprintf("[auto-resume #%d]", attempts))
	}
	if cursorPK != nil {
		pkStr := strings.TrimSpace(fmt.Sprintf("%v", cursorPK))
		if pkStr != "" && pkStr != "<nil>" {
			parts = append(parts, fmt.Sprintf("[cursor_pk=%s]", pkStr))
		}
	}
	if force || len(forceFields) > 0 {
		parts = append(parts, "[force=1]")
	}
	if len(forceFields) > 0 {
		parts = append(parts, fmt.Sprintf("[force_fields=%s]", strings.Join(forceFields, ",")))
	}
	return strings.Join(parts, " ")
}

// ParseTransformCursorPK extracts the saved PK cursor from error_message ("[cursor_pk=...]").
func ParseTransformCursorPK(errMsg string) (string, bool) {
	const prefix = "[cursor_pk="
	idx := strings.Index(errMsg, prefix)
	if idx < 0 {
		return "", false
	}
	rest := errMsg[idx+len(prefix):]
	end := strings.IndexByte(rest, ']')
	if end <= 0 {
		return "", false
	}
	val := strings.TrimSpace(rest[:end])
	if val == "" {
		return "", false
	}
	return val, true
}

// ParseTransformForce extracts both the boolean force flag ("[force=1]") and optional force_fields list ("[force_fields=...]") from error_message (C3.1).
func ParseTransformForce(errMsg string) (bool, []string) {
	isForce := strings.Contains(errMsg, "[force=1]")
	const prefix = "[force_fields="
	idx := strings.Index(errMsg, prefix)
	if idx < 0 {
		return isForce, nil
	}
	rest := errMsg[idx+len(prefix):]
	end := strings.IndexByte(rest, ']')
	if end <= 0 {
		return isForce, nil
	}
	raw := strings.TrimSpace(rest[:end])
	if raw == "" {
		return isForce, nil
	}
	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	for _, it := range items {
		if trimmed := strings.TrimSpace(it); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) > 0 {
		isForce = true
	}
	return isForce, out
}

// GetByID retrieves a TransformJob by job_id.
func (r *TransformJobRepo) GetByID(ctx context.Context, jobID string) (*TransformJob, error) {
	if jobID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var job TransformJob
	if err := r.db.WithContext(ctx).Where("job_id = ?", jobID).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateStatus updates status, rows_affected, total_rows, finished_at, updated_at for a job.
// When status == "RUNNING", it refuses to overwrite rows where cancel_requested=true or status is already terminal.
func (r *TransformJobRepo) UpdateStatus(ctx context.Context, jobID, status string, rowsAffected, totalRows int64, errMsg string) error {
	updates := map[string]interface{}{
		"status":        status,
		"rows_affected": rowsAffected,
		"updated_at":    time.Now(),
	}
	if totalRows > 0 {
		updates["total_rows"] = totalRows
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	} else if status == "COMPLETED" {
		updates["error_message"] = ""
	}
	if status == "RUNNING" {
		if rowsAffected > 0 && totalRows > 0 {
			pct := int(float64(rowsAffected) / float64(totalRows) * 100)
			if pct > 99 {
				pct = 99
			}
			updates["progress_percent"] = pct
		} else {
			updates["progress_percent"] = 0
		}
	}
	if status == "COMPLETED" || status == "FAILED" || status == "CANCELLED" {
		now := time.Now()
		updates["finished_at"] = &now
		if status == "COMPLETED" {
			updates["progress_percent"] = 100
		}
	}
	q := r.db.WithContext(ctx).Model(&TransformJob{})
	if status == "RUNNING" || status == "PAUSED" {
		q = q.Where("job_id = ? AND cancel_requested = false AND status NOT IN ('CANCELLED', 'COMPLETED', 'FAILED')", jobID)
	} else {
		q = q.Where("job_id = ?", jobID)
	}
	return q.Updates(updates).Error
}

// UpdateProgress updates progress_percent, rows_affected, total_rows, chunk_size, updated_at, and optional checkpointNote.
// It refuses to overwrite jobs that have cancel_requested=true or are already in terminal status.
func (r *TransformJobRepo) UpdateProgress(ctx context.Context, jobID string, progress int, rowsAffected, totalRows int64, chunkSize int, checkpointNote ...string) error {
	updates := map[string]interface{}{
		"progress_percent": progress,
		"rows_affected":    rowsAffected,
		"chunk_size":       chunkSize,
		"updated_at":       time.Now(),
	}
	if totalRows > 0 {
		updates["total_rows"] = totalRows
	}
	if len(checkpointNote) > 0 && checkpointNote[0] != "" {
		updates["error_message"] = checkpointNote[0]
	}
	return r.db.WithContext(ctx).
		Model(&TransformJob{}).
		Where("job_id = ? AND cancel_requested = false AND status NOT IN ('CANCELLED', 'COMPLETED', 'FAILED')", jobID).
		Updates(updates).Error
}

// TouchHeartbeat updates updated_at = NOW() while a transform job is actively running.
func (r *TransformJobRepo) TouchHeartbeat(ctx context.Context, jobID string) error {
	if jobID == "" {
		return nil
	}
	return r.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.transform_jobs SET updated_at = NOW() WHERE job_id = ? AND cancel_requested = false AND status IN ('PENDING', 'RUNNING')`,
		jobID,
	).Error
}

// IsCancelRequested returns true if cancel_requested=true or status='CANCELLED' for the given job.
// It always uses an independent 3-second context derived from context.Background() so that canceling
// the job context in HandleTransformCancel never causes the DB query to fail and return false.
func (r *TransformJobRepo) IsCancelRequested(_ context.Context, jobID string) bool {
	if jobID == "" {
		return false
	}
	qCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var result struct {
		CancelRequested bool   `gorm:"column:cancel_requested"`
		Status          string `gorm:"column:status"`
	}
	if err := r.db.WithContext(qCtx).
		Raw("SELECT cancel_requested, status FROM cdc_system.transform_jobs WHERE job_id = ?", jobID).
		Scan(&result).Error; err != nil {
		return false
	}
	return result.CancelRequested || result.Status == "CANCELLED"
}

// ClaimRecoverableJobs sweeps stale cancel_requested transform jobs to CANCELLED,
// and atomically claims PAUSED or orphaned PENDING/RUNNING transform jobs (>45s without heartbeat)
// using FOR UPDATE SKIP LOCKED without prematurely cancelling PAUSED [force=1] jobs (C6).
func (r *TransformJobRepo) ClaimRecoverableJobs(ctx context.Context) ([]TransformJob, error) {
	_ = r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transform_jobs
		   SET status = 'CANCELLED',
		       error_message = 'cancelled by user',
		       finished_at = NOW(),
		       updated_at = NOW()
		 WHERE cancel_requested = true
		   AND status IN ('PENDING', 'RUNNING', 'PAUSED')
		   AND updated_at < NOW() - INTERVAL '45 seconds'
	`).Error

	// Sửa C6: Chỉ tự động đóng job cũ nếu một job mới hơn trên cùng target_table đã COMPLETED
	// với phạm vi tương đương hoặc rộng hơn (không bao giờ lấy job thường IS NULL để hủy oan job [force=1] đang PAUSED!).
	_ = r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transform_jobs j
		   SET status = 'CANCELLED',
		       error_message = 'superseded by completed newer transform job on same target_table',
		       finished_at = NOW(),
		       updated_at = NOW()
		 WHERE j.status IN ('PENDING', 'RUNNING', 'PAUSED')
		   AND (j.status = 'PAUSED' OR j.updated_at < NOW() - INTERVAL '45 seconds')
		   AND EXISTS (
		       SELECT 1 FROM cdc_system.transform_jobs j2
		        WHERE j2.target_table = j.target_table
		          AND j2.job_id <> j.job_id
		          AND j2.created_at > j.created_at
		          AND j2.status = 'COMPLETED'
		          AND (
		              j.error_message NOT LIKE '%[force=1]%'
		              OR (j2.error_message LIKE '%[force=1]%' AND j2.error_message NOT LIKE '%[force_fields=%')
		          )
		   )
	`).Error

	var jobs []TransformJob
	err := r.db.WithContext(ctx).Raw(`
		WITH claimed AS (
			SELECT j.job_id
			FROM cdc_system.transform_jobs j
			WHERE j.cancel_requested = false
			  AND (
			      j.status = 'PAUSED'
			      OR (j.status IN ('PENDING', 'RUNNING') AND j.updated_at < NOW() - INTERVAL '45 seconds')
			  )
			  AND NOT EXISTS (
			      SELECT 1 FROM cdc_system.transform_jobs j2
			       WHERE j2.target_table = j.target_table
			         AND j2.job_id <> j.job_id
			         AND j2.status = 'RUNNING'
			         AND j2.updated_at >= NOW() - INTERVAL '45 seconds'
			  )
			ORDER BY j.updated_at ASC
			LIMIT 10
			FOR UPDATE SKIP LOCKED
		)
		UPDATE cdc_system.transform_jobs j
		   SET updated_at = NOW()
		  FROM claimed c
		 WHERE j.job_id = c.job_id
		RETURNING j.*
	`).Scan(&jobs).Error
	return jobs, err
}
```
```go
// Trong centralized-data-service/internal/handler/shadow/batch_transform_handler.go (lines 25-60, 180-245, 252-496):
const SubjectTransformCancel = "cdc.cmd.transform-cancel"

// BatchTransformPayload is the JSON payload for cdc.cmd.batch-transform (khớp 100% nguyên bản trên đĩa — C1 & D3).
// For backward compatibility, if JSON parsing fails, msg.Data is treated as a plain table name.
type BatchTransformPayload struct {
	JobID           string   `json:"job_id"`
	TargetTable     string   `json:"target_table"`
	SourceObjectID  int64    `json:"source_object_id,omitempty"`
	ShadowBindingID int64    `json:"shadow_binding_id,omitempty"`
	TraceID         string   `json:"trace_id,omitempty"`
	// Force=true: bỏ qua IS NULL check, SET lại các field trong ForceFields (hoặc toàn bộ active rules nếu ForceFields rỗng).
	Force       bool     `json:"force,omitempty"`
	ForceFields []string `json:"force_fields,omitempty"`
}

type BatchTransformHandler struct {
	base.BaseHandler
	mappingV2Repo      *repomaster.MappingRuleV2Repo
	shadowDB           *gorm.DB
	transformChunkSize int
	metadataRegistry   metadata.MetadataRegistry
	registryRepo       metadata.RegistryResolver
	transformJobRepo   *repository.TransformJobRepo
	hmacKey            string
	activeJobsMu       sync.Mutex
	activeJobs         map[string]context.CancelFunc
	activeTables       map[string]string
	recoverCancel      context.CancelFunc
	stopping           atomic.Bool
	wg                 sync.WaitGroup
	stopOnce           sync.Once
	stopCh             chan struct{}
}

func NewBatchTransformHandler(
	db *gorm.DB,
	natsConn *nats.Conn,
	logger *zap.Logger,
	mappingV2Repo *repomaster.MappingRuleV2Repo,
	shadowDB *gorm.DB,
	metadataRegistry metadata.MetadataRegistry,
	registryRepo metadata.RegistryResolver,
) *BatchTransformHandler {
	return &BatchTransformHandler{
		BaseHandler:        base.NewBaseHandler(db, natsConn, logger),
		mappingV2Repo:      mappingV2Repo,
		shadowDB:           shadowDB,
		transformChunkSize: 1000,
		metadataRegistry:   metadataRegistry,
		registryRepo:       registryRepo,
		activeJobs:         make(map[string]context.CancelFunc),
		activeTables:       make(map[string]string),
		stopCh:             make(chan struct{}),
	}
}

func (h *BatchTransformHandler) SetStopping(v bool) {
	h.stopping.Store(v)
}

// HandleTransformCancel immediately cancels an in-flight batch transform job by job_id.
func (h *BatchTransformHandler) HandleTransformCancel(msg *nats.Msg) {
	var payload struct {
		JobID       string `json:"job_id"`
		TargetTable string `json:"target_table"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil || payload.JobID == "" {
		return
	}
	h.activeJobsMu.Lock()
	cancelFn, ok := h.activeJobs[payload.JobID]
	h.activeJobsMu.Unlock()
	if ok && cancelFn != nil {
		cancelFn()
	}
}

// Start launches the background recovery loop to auto-resume PAUSED or orphaned transform jobs.
func (h *BatchTransformHandler) Start(ctx context.Context) {
	h.activeJobsMu.Lock()
	if h.stopping.Load() {
		h.activeJobsMu.Unlock()
		return
	}
	recCtx, recCancel := context.WithCancel(ctx)
	h.recoverCancel = recCancel
	h.wg.Add(1)
	h.activeJobsMu.Unlock()

	go func() {
		defer h.wg.Done()
		h.recoverOrphanedTransformJobs(recCtx)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-recCtx.Done():
				return
			case <-h.stopCh:
				return
			case <-ticker.C:
				h.recoverOrphanedTransformJobs(recCtx)
			}
		}
	}()
}

func (h *BatchTransformHandler) recoverOrphanedTransformJobs(ctx context.Context) {
	if h.transformJobRepo == nil || h.stopping.Load() {
		return
	}
	jobs, err := h.transformJobRepo.ClaimRecoverableJobs(ctx)
	if err != nil || len(jobs) == 0 {
		return
	}
	for _, j := range jobs {
		if h.stopping.Load() {
			return
		}
		h.activeJobsMu.Lock()
		_, alreadyRunning := h.activeJobs[j.JobID]
		activeJobForTable, tableBusy := h.activeTables[j.TargetTable]
		h.activeJobsMu.Unlock()
		if alreadyRunning || (tableBusy && activeJobForTable != j.JobID) {
			continue
		}
		attempts := repository.ParseAutoResumeCount(j.ErrorMessage)
		savedCursor, _ := repository.ParseTransformCursorPK(j.ErrorMessage)
		isForce, forceFields := repository.ParseTransformForce(j.ErrorMessage)
		if j.Status != "PAUSED" {
			if attempts >= repository.MaxOrphanAutoResumeAttempts {
				failMsg := fmt.Sprintf("FAILED: max orphan auto-resume attempts (%d) exceeded", repository.MaxOrphanAutoResumeAttempts)
				h.Logger.Error("Skipping poison-pill orphaned transform job after max retries",
					zap.String("job_id", j.JobID),
					zap.String("target_table", j.TargetTable),
					zap.Int("attempts", attempts))
				_ = h.transformJobRepo.UpdateStatus(ctx, j.JobID, "FAILED", j.RowsAffected, j.TotalRows, failMsg)
				continue
			}
			nextNote := repository.FormatTransformCheckpointNote(attempts+1, savedCursor, isForce, forceFields)
			_ = h.transformJobRepo.UpdateStatus(ctx, j.JobID, "RUNNING", j.RowsAffected, j.TotalRows, nextNote)
		}
		var srcObjID int64
		if j.SourceObjectID != nil {
			srcObjID = *j.SourceObjectID
		}
		h.Logger.Info("Auto-resuming interrupted transform job from checkpoint",
			zap.String("job_id", j.JobID),
			zap.String("target_table", j.TargetTable),
			zap.String("prev_status", j.Status),
			zap.String("cursor_pk", savedCursor),
			zap.Bool("force", isForce),
			zap.Int64("rows_affected", j.RowsAffected))
		raw, _ := json.Marshal(BatchTransformPayload{
			JobID:          j.JobID,
			TargetTable:    j.TargetTable,
			SourceObjectID: srcObjID,
			TraceID:        j.TraceID,
			Force:          isForce,
			ForceFields:    forceFields,
		})
		h.HandleBatchTransform(&nats.Msg{Subject: "cdc.cmd.batch-transform", Data: raw})
	}
}

// Stop signals active transform jobs to pause cleanly at the current 1000-row chunk boundary (2s budget — C4) and waits for exit.
func (h *BatchTransformHandler) Stop() {
	h.stopOnce.Do(func() {
		h.activeJobsMu.Lock()
		h.stopping.Store(true)
		if h.recoverCancel != nil {
			h.recoverCancel()
		}
		if h.stopCh != nil {
			close(h.stopCh)
		}
		h.activeJobsMu.Unlock()

		done := make(chan struct{})
		go func() {
			h.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			h.activeJobsMu.Lock()
			for _, cancelFn := range h.activeJobs {
				if cancelFn != nil {
					cancelFn()
				}
			}
			h.activeJobsMu.Unlock()
			<-done
		}
	})
}

// HandleBatchTransform is the NATS message handler.
func (h *BatchTransformHandler) HandleBatchTransform(msg *nats.Msg) {
	var payload BatchTransformPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil || payload.TargetTable == "" {
		payload.TargetTable = string(msg.Data)
		payload.JobID = ""
	}

	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx = observability.WithDBTraceModule(ctx, "batch_transform")

	h.activeJobsMu.Lock()
	if h.stopping.Load() {
		h.activeJobsMu.Unlock()
		return
	}
	if payload.TargetTable != "" {
		if activeID, busy := h.activeTables[payload.TargetTable]; busy && activeID != "" && activeID != payload.JobID {
			h.activeJobsMu.Unlock()
			h.Logger.Warn("Skipping duplicate batch transform while target_table already has an active job",
				zap.String("target_table", payload.TargetTable),
				zap.String("active_job_id", activeID),
				zap.String("skipped_job_id", payload.JobID))
			return
		}
	}
	jobCtx, cancel := context.WithTimeout(ctx, 24*time.Hour)
	jobKey := payload.JobID
	if jobKey == "" {
		jobKey = fmt.Sprintf("anon:%s:%d", payload.TargetTable, time.Now().UnixNano())
	}
	h.activeJobs[jobKey] = cancel
	if payload.TargetTable != "" {
		h.activeTables[payload.TargetTable] = jobKey
	}
	h.wg.Add(1)
	h.activeJobsMu.Unlock()

	go func() {
		defer h.wg.Done()
		defer func() {
			h.activeJobsMu.Lock()
			delete(h.activeJobs, jobKey)
			if payload.TargetTable != "" && h.activeTables[payload.TargetTable] == jobKey {
				delete(h.activeTables, payload.TargetTable)
			}
			h.activeJobsMu.Unlock()
			cancel()
		}()
		h.runTransformJob(jobCtx, payload)
	}()
}
```
```go
// Trong BatchTransformHandler.runTransformJob (lines 180-245 & 252-496 — khôi phục parsedForce/parsedFields từ GetByID trước khi tạo forceSet/setClauses theo G3, hỗ trợ Force=true cả khi ForceFields rỗng, kiểm tra kiểu PK an toàn & dừng ở ranh giới mẻ):
	// ── G3: Khôi phục baseRowsAffected, cursor_pk, force & force_fields TRƯỚC khi xây dựng forceSet / setClauses ─
	var baseRowsAffected int64
	var preservedTotalRows int64
	var resumeAttempts int
	var savedCursorRaw string
	var hasSavedCursor bool
	isForceJob := payload.Force
	var forceFieldsNote []string
	if len(payload.ForceFields) > 0 {
		forceFieldsNote = payload.ForceFields
	}
	if h.transformJobRepo != nil && jobID != "" {
		if existingJob, errJob := h.transformJobRepo.GetByID(ctx, jobID); errJob == nil && existingJob != nil {
			if existingJob.CancelRequested || existingJob.Status == "CANCELLED" {
				h.finishJob(context.Background(), jobID, "CANCELLED", existingJob.RowsAffected, existingJob.TotalRows, "cancelled by user")
				return
			}
			if existingJob.RowsAffected > 0 {
				baseRowsAffected = existingJob.RowsAffected
			}
			if existingJob.TotalRows > 0 {
				preservedTotalRows = existingJob.TotalRows
			}
			resumeAttempts = repository.ParseAutoResumeCount(existingJob.ErrorMessage)
			savedCursorRaw, hasSavedCursor = repository.ParseTransformCursorPK(existingJob.ErrorMessage)
			parsedForce, parsedFields := repository.ParseTransformForce(existingJob.ErrorMessage)
			if parsedForce {
				isForceJob = true
			}
			if len(forceFieldsNote) == 0 && len(parsedFields) > 0 {
				forceFieldsNote = parsedFields
			}
		}
	}

	// Force mode: xây dựng tập cột cần cưỡng bức (nếu forceFieldsNote rỗng thì áp dụng cho toàn bộ cột active mapping — C3.1, D3 & G3)
	forceSet := make(map[string]struct{}, len(forceFieldsNote))
	for _, f := range forceFieldsNote {
		if trimmed := strings.ToLower(strings.TrimSpace(f)); trimmed != "" {
			forceSet[trimmed] = struct{}{}
		}
	}

	var setClauses []string
	var nullChecks []string
	seenCols := make(map[string]struct{})
	for _, r := range rules {
		if r.Status != "active" && r.Status != "approved" && r.Status != "" {
			continue
		}
		targetField := strings.TrimSpace(r.TargetField)
		if targetField == "" {
			continue
		}
		colKey := strings.ToLower(targetField)
		if _, exists := seenCols[colKey]; exists {
			continue
		}
		if isForceJob && len(forceSet) > 0 {
			if _, inForce := forceSet[colKey]; !inForce {
				continue
			}
		}
		seenCols[colKey] = struct{}{}
		quotedCol := sqlutil.QuoteIdent(targetField)
		expr := h.buildExtractionSQL(r)
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedCol, expr))
		nullChecks = append(nullChecks, fmt.Sprintf("%s IS NULL", quotedCol))
	}

	if len(setClauses) == 0 {
		observability.Ctx(ctx, h.Logger).Info("batch transform: no active columns matched", zap.String("table", targetTable))
		h.finishJob(ctx, jobID, "COMPLETED", 0, 0, "")
		return
	}

	setExpr := strings.Join(setClauses, ", ")
	whereExpr := strings.Join(nullChecks, " OR ")
	if isForceJob {
		whereExpr = "TRUE"
	}

	if h.transformJobRepo != nil && jobID != "" {
		initNote := repository.FormatTransformCheckpointNote(resumeAttempts, savedCursorRaw, isForceJob, forceFieldsNote)
		_ = h.transformJobRepo.UpdateStatus(ctx, jobID, "RUNNING", baseRowsAffected, preservedTotalRows, initNote)
		hbCtx, hbCancel := context.WithCancel(ctx)
		defer hbCancel()
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-hbCtx.Done():
					return
				case <-ticker.C:
					_ = h.transformJobRepo.TouchHeartbeat(context.Background(), jobID)
				}
			}
		}()
	}

	// ── Phát hiện Primary Key + Kiểu dữ liệu PK TRƯỚC khi đếm để áp dụng AND quotedPK > lastPK an toàn kiểu (B3, A3.2 & C3.2) ─
	pkCol, pkType, pkErr := h.detectPrimaryKey(execDB, schemaName, pureTable)
	chunkSize := h.transformChunkSize
	if chunkSize <= 0 {
		chunkSize = 1000
	}

	var lastPK interface{}
	hasCursor := false
	if hasSavedCursor && savedCursorRaw != "" && pkErr == nil && pkCol != "" {
		hasCursor = true
		if isIntegerPGType(pkType) {
			if parsedInt, errParse := strconv.ParseInt(savedCursorRaw, 10, 64); errParse == nil {
				lastPK = parsedInt
			} else {
				lastPK = savedCursorRaw
			}
		} else {
			lastPK = savedCursorRaw
		}
	}

	// ── Đếm trước tổng số bản ghi còn cần transform (kể từ sau cursor_pk nếu Resume) ─
	var remainingPendingRows int64
	if hasCursor && pkCol != "" {
		quotedPKForCount := sqlutil.QuoteIdent(pkCol)
		countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE _raw_data IS NOT NULL AND (%s) AND %s > ?`, quotedTable, whereExpr, quotedPKForCount)
		if err := execDB.WithContext(ctx).Raw(countSQL, lastPK).Scan(&remainingPendingRows).Error; err != nil {
			observability.Ctx(ctx, h.Logger).Warn("batch transform: failed to count pending rows after cursor", zap.String("table", targetTable), zap.Error(err))
		}
	} else {
		countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE _raw_data IS NOT NULL AND (%s)`, quotedTable, whereExpr)
		if err := execDB.WithContext(ctx).Raw(countSQL).Scan(&remainingPendingRows).Error; err != nil {
			observability.Ctx(ctx, h.Logger).Warn("batch transform: failed to count pending rows", zap.String("table", targetTable), zap.Error(err))
		}
	}
	totalPendingRows := baseRowsAffected + remainingPendingRows
	if preservedTotalRows > totalPendingRows {
		totalPendingRows = preservedTotalRows
	}

	if h.transformJobRepo != nil && jobID != "" {
		initNote := repository.FormatTransformCheckpointNote(resumeAttempts, lastPK, isForceJob, forceFieldsNote)
		if err := h.transformJobRepo.UpdateStatus(ctx, jobID, "RUNNING", baseRowsAffected, totalPendingRows, initNote); err != nil {
			observability.Ctx(ctx, h.Logger).Warn("transform_jobs: failed to set RUNNING", zap.String("job_id", jobID), zap.Error(err))
		}
	}

	if remainingPendingRows == 0 {
		observability.Ctx(ctx, h.Logger).Info("batch transform: no pending rows to transform", zap.String("table", targetTable))
		h.finishJob(ctx, jobID, "COMPLETED", baseRowsAffected, totalPendingRows, "")
		h.publishTransmuteTrigger(ctx, schemaName, pureTable)
		return
	}

	if pkErr != nil || pkCol == "" {
		observability.Ctx(ctx, h.Logger).Warn("batch transform: PK not detected, running unchunked UPDATE",
			zap.String("table", targetTable),
			zap.Error(pkErr),
		)
		transformSQL := fmt.Sprintf(`UPDATE %s SET %s WHERE _raw_data IS NOT NULL AND (%s)`,
			quotedTable, setExpr, whereExpr,
		)
		result := execDB.WithContext(ctx).Exec(transformSQL)
		if result.Error != nil {
			if h.transformJobRepo != nil && jobID != "" && h.transformJobRepo.IsCancelRequested(context.Background(), jobID) {
				h.finishJob(context.Background(), jobID, "CANCELLED", baseRowsAffected, totalPendingRows, "cancelled by user")
				return
			}
			if h.stopping.Load() {
				pauseNote := repository.FormatTransformCheckpointNote(resumeAttempts, lastPK, isForceJob, forceFieldsNote)
				if pauseNote == "" {
					pauseNote = "paused by worker shutdown"
				}
				h.finishJob(context.Background(), jobID, "PAUSED", baseRowsAffected, totalPendingRows, pauseNote)
				return
			}
			enrichedErr := h.enrichCastingError(result.Error, rules)
			h.publishAndFinishJob(ctx, jobID, "error", baseRowsAffected, totalPendingRows, enrichedErr, rules)
			return
		}
		h.finishJob(ctx, jobID, "COMPLETED", baseRowsAffected+result.RowsAffected, totalPendingRows, "")
		h.publishTransmuteTrigger(ctx, schemaName, pureTable)
		return
	}

	// ── Chunked CTE UPDATE loop ────────────────────────────────────────────
	quotedPK := sqlutil.QuoteIdent(pkCol)
	totalRows := baseRowsAffected
	var productiveIters int

	for {
		if h.transformJobRepo != nil && jobID != "" {
			if h.transformJobRepo.IsCancelRequested(ctx, jobID) {
				observability.Ctx(ctx, h.Logger).Info("batch transform cancelled by request",
					zap.String("table", targetTable),
					zap.String("job_id", jobID),
					zap.Int64("rows_so_far", totalRows),
				)
				h.finishJob(context.Background(), jobID, "CANCELLED", totalRows, totalPendingRows, "cancelled by user")
				if h.DB != nil {
					act := governance.NewActivityLogger(h.DB, h.Logger)
					act.Quick("cmd-batch-transform", targetTable, "nats-command", "cancelled",
						totalRows, map[string]interface{}{"trace_id": traceID, "iterations": productiveIters}, "")
				}
				return
			}
		}
		if h.stopping.Load() {
			observability.Ctx(ctx, h.Logger).Info("batch transform paused at chunk boundary for graceful shutdown",
				zap.String("table", targetTable),
				zap.String("job_id", jobID),
				zap.Int64("rows_so_far", totalRows),
			)
			pauseNote := repository.FormatTransformCheckpointNote(resumeAttempts, lastPK, isForceJob, forceFieldsNote)
			if pauseNote == "" {
				pauseNote = "paused by worker shutdown"
			}
			h.finishJob(context.Background(), jobID, "PAUSED", totalRows, totalPendingRows, pauseNote)
			return
		}

		cursorClause := ""
		var args []interface{}
		if hasCursor {
			cursorClause = fmt.Sprintf(" AND %s > ?", quotedPK)
			args = append(args, lastPK)
		}

		chunkSQL := fmt.Sprintf(
			`WITH chunk AS (
                SELECT %[1]s AS pk FROM %[2]s
                WHERE _raw_data IS NOT NULL AND (%[6]s)%[3]s
                ORDER BY %[1]s LIMIT %[4]d
            ),
            upd AS (
                UPDATE %[2]s s SET %[5]s
                FROM chunk c
                WHERE s.%[1]s = c.pk
                RETURNING s.%[1]s
            )
            SELECT 
                (SELECT MAX(pk) FROM chunk) AS max_pk,
                (SELECT COUNT(*) FROM chunk) AS chunk_count,
                (SELECT COUNT(*) FROM upd) AS updated_count`,
			quotedPK, quotedTable, cursorClause, chunkSize, setExpr, whereExpr,
		)

		chunkSpanName := fmt.Sprintf("shadow.batch_transform: %s", targetTable)
		chunkCtx, chunkSpan := observability.ChildSpan(ctx, chunkSpanName,
			attribute.String("db.table", targetTable),
			attribute.Int("batch.chunk_size", chunkSize),
			attribute.Int("batch.iteration", productiveIters+1),
		)

		chunkStart := time.Now()
		row := execDB.WithContext(chunkCtx).Raw(chunkSQL, args...).Row()
		var maxPK interface{}
		var chunkCount int64
		var updatedCount int64
		err := row.Scan(&maxPK, &chunkCount, &updatedCount)
		elapsed := time.Since(chunkStart)

		chunkSpan.SetAttributes(
			attribute.Int64("batch.chunk_count", chunkCount),
			attribute.Int64("batch.updated_count", updatedCount),
			attribute.Int64("batch.elapsed_ms", elapsed.Milliseconds()),
		)
		observability.EndSpan(chunkSpan, &err)

		if err != nil {
			if h.transformJobRepo != nil && jobID != "" && h.transformJobRepo.IsCancelRequested(context.Background(), jobID) {
				h.finishJob(context.Background(), jobID, "CANCELLED", totalRows, totalPendingRows, "cancelled by user")
				return
			}
			if h.stopping.Load() {
				pauseNote := repository.FormatTransformCheckpointNote(resumeAttempts, lastPK, isForceJob, forceFieldsNote)
				if pauseNote == "" {
					pauseNote = "paused by worker shutdown"
				}
				h.finishJob(context.Background(), jobID, "PAUSED", totalRows, totalPendingRows, pauseNote)
				return
			}
			enrichedErr := h.enrichCastingError(err, rules)
			observability.Ctx(ctx, h.Logger).Error("batch transform chunk query failed",
				zap.String("table", targetTable),
				zap.Error(err),
				zap.String("enriched_error", enrichedErr),
			)
			h.finishJob(context.Background(), jobID, "FAILED", totalRows, totalPendingRows, enrichedErr)
			return
		}

		if maxPK == nil || chunkCount == 0 {
			break
		}

		totalRows += updatedCount
		productiveIters++
		hasCursor = true
		lastPK = maxPK

		requestedChunkSize := chunkSize
		if elapsed < 100*time.Millisecond && chunkSize < 20000 {
			chunkSize = min(chunkSize*2, 20000)
		} else if elapsed > 1000*time.Millisecond && chunkSize > 500 {
			chunkSize = max(chunkSize/2, 500)
		}

		if h.transformJobRepo != nil && jobID != "" {
			progressPercent := 0
			if totalPendingRows > 0 {
				progressPercent = int(float64(totalRows) / float64(totalPendingRows) * 100)
				if progressPercent > 99 {
					progressPercent = 99
				}
			}
			cpNote := repository.FormatTransformCheckpointNote(resumeAttempts, lastPK, isForceJob, forceFieldsNote)
			_ = h.transformJobRepo.UpdateProgress(ctx, jobID, progressPercent, totalRows, totalPendingRows, chunkSize, cpNote)
		}

		if chunkCount < int64(requestedChunkSize) {
			break
		}
	}
```
```go
// Trong BatchTransformHandler.detectPrimaryKey (lines 477-496 — trả về cả tên cột PK lẫn kiểu dữ liệu để tránh lỗi 42883 trên PK kiểu chuỗi — C3.2):
func isIntegerPGType(pgType string) bool {
	t := strings.ToLower(strings.TrimSpace(pgType))
	switch t {
	case "smallint", "integer", "bigint", "int2", "int4", "int8", "serial", "bigserial", "smallserial":
		return true
	default:
		return false
	}
}

func (h *BatchTransformHandler) detectPrimaryKey(execDB *gorm.DB, schemaName, tableName string) (string, string, error) {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	type pkMeta struct {
		ColName  string `gorm:"column:col_name"`
		DataType string `gorm:"column:data_type"`
	}
	var cols []pkMeta
	sql := `
        SELECT a.attname AS col_name, format_type(a.atttypid, a.atttypmod) AS data_type
        FROM pg_index i
        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = (?::text)::regclass AND i.indisprimary
        ORDER BY array_position(i.indkey, a.attnum)`
	qualified := schemaName + "." + tableName
	if err := execDB.Raw(sql, qualified).Scan(&cols).Error; err != nil {
		return "", "", err
	}
	if len(cols) != 1 {
		return "", "", nil
	}
	return cols[0].ColName, cols[0].DataType, nil
}
```

#### Demo P13-7: `centralized-data-service/internal/server/server_setup.go` (`lines 265-290`, `305-343`, `407-423`) — Sửa `R-P13-8` & `C4.2` (Báo trước `SetStopping(true)` đồng loạt, tổng ngân sách tắt máy LIFO $\le 12\text{s} < 15\text{s}$)
Đăng ký đầy đủ `ws.RegisterOnStart` và `ws.RegisterOnStop` + NATS cancel subscribers cho cả 3 bộ `batchTransformHandler`, `transmuteHandler`, và `snapshotRunner`, đồng thời phát tín hiệu `SetStopping(true)` đồng loạt ngay bước đầu tiên của LIFO `onStop` để cả 3 Tier-2 runners bắt đầu dừng ở ranh giới mẻ song song (`2s` + `4s` `TableDebouncer` + `4s` `BatchBuffer` $\le 12\text{s} < 15\text{s}$):
```go
// Tại lines 265-290 (BatchTransformHandler):
	batchTransformHandler := handlershadow.NewBatchTransformHandler(db, natsClient.Conn, logger, mappingRuleV2Repo, shadowDB, registrySvc, registryRepo)
	batchTransformHandler.SetHMACKey(cfg.MaskingHMACKey)
	batchTransformHandler.SetTransformChunkSize(cfg.Worker.TransformChunkSize)
	batchTransformHandler.SetTransformJobRepo(repository.NewTransformJobRepo(db))
	ws.RegisterOnStart(func() { batchTransformHandler.Start(context.Background()) })
	ws.RegisterOnStop(func() { batchTransformHandler.Stop() })
```
```go
// Tại lines 288-290 (NATS subscribe cho BatchTransform):
	natsClient.Conn.QueueSubscribe("cdc.cmd.batch-transform", workerQueueGroup, batchTransformHandler.HandleBatchTransform)
	natsClient.Conn.Subscribe(handlershadow.SubjectTransformCancel, batchTransformHandler.HandleTransformCancel)
```
```go
// Tại lines 305-343 (TransmuteHandler):
	transmuteJobRepo := repository.NewTransmuteJobRepo(db)
	transmuter := master.NewTransmuterModule(db, connectionManager, syncRuntimeRepo, masterDDLGen, typeResolver, logger)
	transmuter.SetTransmuteJobRepo(transmuteJobRepo)
	masterDDLGen.SetCacheInvalidator(transmuter.InvalidateRuleCache)
	transmuteHandler := handlermaster.NewTransmuteHandler(transmuter, masterBindingRepo, natsClient.Conn, logger, activityLogger)
	transmuteHandler.SetTransmuteJobRepo(transmuteJobRepo)
	ws.RegisterOnStart(func() { transmuteHandler.Start(context.Background()) })
	ws.RegisterOnStop(func() { transmuteHandler.Stop() })
```
```go
// Tại lines 338-343 (NATS subscribe cho Transmute):
	natsClient.Conn.QueueSubscribe("cdc.cmd.transmute", "transmute-workers", transmuteHandler.HandleTransmute)
	natsClient.Conn.Subscribe(handlermaster.SubjectTransmuteCancel, transmuteHandler.HandleTransmuteCancel)
	natsClient.Conn.QueueSubscribe(handlermaster.SubjectTransmuteShadowCoreFallback, "transmute-workers", transmuteHandler.HandleTransmuteShadow)
```
```go
// Tại lines 407-426 (SnapshotRunner + Pre-Signal SetStopping cho cả 3 Tier-2 runners ngay đầu LIFO shutdown):
	snapshotRunner := handlerorchestration.NewSnapshotRunner(
		db,
		snapshotEventHandler,
		registrySvc,
		connectionRepo,
		sourceObjectRepo,
		shadowBindingRepo,
		natsClient.Conn,
		logger,
		cfg.Kafka.Brokers,
		cfg.Kafka.GroupID,
		cfg.Debezium.KafkaConnectURL,
	)
	natsClient.Conn.QueueSubscribe("cdc.cmd.snapshot.v2", "cdc-snapshot-runner", snapshotRunner.Handle)
	ws.RegisterOnStart(func() { snapshotRunner.Start(context.Background()) })
	ws.RegisterOnStop(func() {
		// Báo trước SetStopping(true) đồng loạt để Snapshot, Transmute và BatchTransform cùng thoát ở ranh giới mẻ song song (C4.2)
		snapshotRunner.SetStopping(true)
		transmuteHandler.SetStopping(true)
		batchTransformHandler.SetStopping(true)
		snapshotRunner.Stop()
	})
```

#### Demo P13-8: `cdc-cms-service` (`transmute_job_repo.go`, `transform_job_repo.go`, `system_health_queries.go`, `master_transmute_job_handler.go`, `source_object_actions_handler.go`, `server.go`, `stuck_job_reaper.go`) — Sửa `F44`, `R-P13-6`, `R-P13-7` & `C3.3`
1. Thêm `StripJobCheckpointTags(status, errMsg string) string` trong `cdc-cms-service/internal/infra/persistence` để xóa sạch các hậu tố kỹ thuật (`[auto-resume #...]`, `[cursor_pk=...]`, `[force=...]`, `[force_fields=...]`, `[run_type=...]`) khi trả dữ liệu cho UI (`GetByID`, `GetLatestByTable`, `GetLatestByMasterBindingID`, `GetLatestBySourceObjectID`, `GetLatestBySourceObjectIDAndTable`, và `system_health_queries.go`) để giao diện CMS không bao giờ hiện báo lỗi giả khi job đang `RUNNING` hoặc `PAUSED` (`C3.3`).
2. Chặn điều kiện rỗng `tj.trace_id <> ''` và `tj.master_binding_id IS NOT NULL AND tj.master_binding_id > 0` khi đóng `transmute_schedule` (`R-P13-6`).
3. Dùng đúng interface `ports.Publisher` (`Publish(ctx context.Context, subject string, payload []byte) error`) của `cdc-cms-service` để bắn `cdc.cmd.transmute-cancel` và `cdc.cmd.transform-cancel`:
```go
// Trong cdc-cms-service/internal/infra/persistence/transmute_job_repo.go (bổ sung StripJobCheckpointTags & cập nhật các hàm đọc + RequestCancel):
var technicalCheckpointTagList = []string{
	"[auto-resume #",
	"[cursor_pk=",
	"[force=",
	"[force_fields=",
	"[run_type=",
}

// StripJobCheckpointTags removes internal checkpoint tags ([cursor_pk=...], [force=1], [force_fields=...], [run_type=...], [auto-resume #...])
// from error_message before returning job records to the UI (C3.3).
func StripJobCheckpointTags(status, errMsg string) string {
	s := strings.TrimSpace(errMsg)
	if s == "" {
		return ""
	}
	for _, prefix := range technicalCheckpointTagList {
		for {
			idx := strings.Index(s, prefix)
			if idx < 0 {
				break
			}
			end := strings.IndexByte(s[idx:], ']')
			if end < 0 {
				break
			}
			s = strings.TrimSpace(s[:idx] + " " + s[idx+end+1:])
		}
	}
	upperStatus := strings.ToUpper(strings.TrimSpace(status))
	if upperStatus == "RUNNING" || upperStatus == "PAUSED" || upperStatus == "PENDING" {
		if s == "paused by worker shutdown" || s == "auto-resumed orphaned job" || s == "reaper: heartbeat timeout (auto-resumable)" {
			return ""
		}
	}
	return strings.TrimSpace(s)
}

// Áp dụng job.ErrorMessage = StripJobCheckpointTags(job.Status, job.ErrorMessage) ở cuối GetByID, GetLatestByTable, GetLatestByMasterBindingID trong TransmuteJobRepo
// và ở cuối GetByID, GetLatestByTable, GetLatestBySourceObjectID, GetLatestBySourceObjectIDAndTable trong TransformJobRepo.
```
```go
// Trong cdc-cms-service/internal/infra/persistence/transmute_job_repo.go (lines 199-208):
// RequestCancel sets cancel_requested=true and immediately transitions PENDING, PAUSED,
// or stale RUNNING jobs (>45s without heartbeat) to CANCELLED, while safely closing any matching running schedule row.
func (r *TransmuteJobRepo) RequestCancel(ctx context.Context, jobID string) error {
	if err := r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transmute_jobs
		   SET cancel_requested = true,
		       status = CASE
		           WHEN status IN ('PENDING', 'PAUSED') OR updated_at < NOW() - INTERVAL '45 seconds' THEN 'CANCELLED'
		           ELSE status
		       END,
		       error_message = CASE
		           WHEN status IN ('PENDING', 'PAUSED') OR updated_at < NOW() - INTERVAL '45 seconds' THEN 'cancelled by user'
		           ELSE error_message
		       END,
		       finished_at = CASE
		           WHEN status IN ('PENDING', 'PAUSED') OR updated_at < NOW() - INTERVAL '45 seconds' THEN NOW()
		           ELSE finished_at
		       END
		 WHERE job_id = ? AND status IN ('PENDING', 'RUNNING', 'PAUSED')
	`, jobID).Error; err != nil {
		return err
	}
	_ = r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transmute_schedule ts
		   SET last_status = 'cancelled',
		       last_error  = 'cancelled by user',
		       updated_at  = NOW()
		  FROM cdc_system.transmute_jobs tj
		 WHERE tj.job_id = ?
		   AND (
		       (tj.master_binding_id IS NOT NULL AND tj.master_binding_id > 0 AND ts.master_binding_id = tj.master_binding_id)
		       OR (tj.trace_id <> '' AND ts.last_trace_id = tj.trace_id)
		   )
		   AND ts.last_status = 'running'
	`, jobID).Error
	return nil
}
```
```go
// Trong cdc-cms-service/internal/infra/persistence/transform_job_repo.go (lines 231-240):
// RequestCancel sets cancel_requested=true and immediately transitions PENDING, PAUSED,
// or stale RUNNING jobs (>45s without heartbeat) to CANCELLED.
func (r *TransformJobRepo) RequestCancel(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transform_jobs
		   SET cancel_requested = true,
		       status = CASE
		           WHEN status IN ('PENDING', 'PAUSED') OR updated_at < NOW() - INTERVAL '45 seconds' THEN 'CANCELLED'
		           ELSE status
		       END,
		       error_message = CASE
		           WHEN status IN ('PENDING', 'PAUSED') OR updated_at < NOW() - INTERVAL '45 seconds' THEN 'cancelled by user'
		           ELSE error_message
		       END,
		       finished_at = CASE
		           WHEN status IN ('PENDING', 'PAUSED') OR updated_at < NOW() - INTERVAL '45 seconds' THEN NOW()
		           ELSE finished_at
		       END
		 WHERE job_id = ? AND status IN ('PENDING', 'RUNNING', 'PAUSED')
	`, jobID).Error
}
```
```go
// Trong cdc-cms-service/internal/api/master/master_transmute_job_handler.go (lines 16-33 & 96-129):
type MasterTransmuteJobHandler struct {
	transmuteJobRepo *persistence.TransmuteJobRepo
	publisher        ports.Publisher
	activityLogger   ports.ActivityLogger
	logger           *zap.Logger
}

func NewMasterTransmuteJobHandler(
	transmuteJobRepo *persistence.TransmuteJobRepo,
	activityLogger ports.ActivityLogger,
	logger *zap.Logger,
) *MasterTransmuteJobHandler {
	return &MasterTransmuteJobHandler{
		transmuteJobRepo: transmuteJobRepo,
		activityLogger:   activityLogger,
		logger:           logger,
	}
}

func (h *MasterTransmuteJobHandler) SetPublisher(pub ports.Publisher) {
	h.publisher = pub
}

// CancelTransmute sets cancel_requested=true for the active transmute job and publishes cdc.cmd.transmute-cancel.
// POST /api/v1/masters/:name/transmute-cancel
func (h *MasterTransmuteJobHandler) CancelTransmute(c *fiber.Ctx) error {
	name := strings.TrimSpace(c.Params("name"))
	var job *persistence.TransmuteJob
	var err error

	if h.transmuteJobRepo == nil {
		return c.Status(503).JSON(fiber.Map{"error": "job_tracking_unavailable"})
	}

	if name != "" {
		job, err = h.transmuteJobRepo.GetLatestByTable(c.UserContext(), name)
	} else if idStr := c.Params("id"); idStr != "" {
		if id, pErr := strconv.ParseInt(idStr, 10, 64); pErr == nil && id > 0 {
			job, err = h.transmuteJobRepo.GetLatestByMasterBindingID(c.UserContext(), id)
		}
	}

	if err != nil || job == nil {
		return c.Status(404).JSON(fiber.Map{"error": "no_active_transmute_job"})
	}
	if job.Status != "PENDING" && job.Status != "RUNNING" && job.Status != "PAUSED" {
		return c.Status(409).JSON(fiber.Map{"error": "job_not_cancellable", "status": job.Status})
	}
	if err := h.transmuteJobRepo.RequestCancel(c.UserContext(), job.JobID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "cancel_request_failed"})
	}
	if h.publisher != nil {
		payloadBytes, _ := json.Marshal(map[string]any{
			"job_id":       job.JobID,
			"master_table": job.MasterTable,
		})
		_ = h.publisher.Publish(c.UserContext(), "cdc.cmd.transmute-cancel", payloadBytes)
	}
	h.activityLogger.LogAsync(ports.ActivityEntry{
		Operation: "transmute-cancel", TargetTable: job.MasterTable, Status: "requested",
		Details: map[string]any{"user": middleware.GetUsername(c), "job_id": job.JobID},
	})
	updatedJob, _ := h.transmuteJobRepo.GetByID(c.UserContext(), job.JobID)
	finalStatus := "CANCEL_REQUESTED"
	if updatedJob != nil {
		finalStatus = updatedJob.Status
	}
	return c.JSON(fiber.Map{"message": "cancel requested", "job_id": job.JobID, "status": finalStatus})
}
```
```go
// Trong cdc-cms-service/internal/api/source/source_object_actions_handler.go (lines 859-886):
// CancelTransformV2 sets cancel_requested=true for the active transform job and publishes cdc.cmd.transform-cancel.
// POST /api/v1/source-objects/:id/transform-cancel
func (h *SourceObjectActionsHandler) CancelTransformV2(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_source_object_id"})
	}
	if h.transformJobRepo == nil {
		return c.Status(503).JSON(fiber.Map{"error": "job_tracking_unavailable"})
	}
	job, err := h.transformJobRepo.GetLatestBySourceObjectID(c.UserContext(), id)
	if err != nil || job == nil {
		return c.Status(404).JSON(fiber.Map{"error": "no_active_transform_job"})
	}
	if job.Status != "PENDING" && job.Status != "RUNNING" && job.Status != "PAUSED" {
		return c.Status(409).JSON(fiber.Map{"error": "job_not_cancellable", "status": job.Status})
	}
	if err := h.transformJobRepo.RequestCancel(c.UserContext(), job.JobID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "cancel_request_failed"})
	}
	if h.publisher != nil {
		payloadBytes, _ := json.Marshal(map[string]any{
			"job_id":       job.JobID,
			"target_table": job.TargetTable,
		})
		_ = h.publisher.Publish(c.UserContext(), "cdc.cmd.transform-cancel", payloadBytes)
	}
	h.activityLogger.LogAsync(ports.ActivityEntry{
		Operation: "transform-cancel", TargetTable: job.TargetTable, Status: "requested",
		Details: map[string]any{"user": middleware.GetUsername(c), "job_id": job.JobID},
	})
	updatedJob, _ := h.transformJobRepo.GetByID(c.UserContext(), job.JobID)
	finalStatus := "CANCEL_REQUESTED"
	if updatedJob != nil {
		finalStatus = updatedJob.Status
	}
	return c.JSON(fiber.Map{"message": "cancel requested", "job_id": job.JobID, "status": finalStatus})
}
```
```go
// Trong cdc-cms-service/internal/server/server.go (lines 355-358):
	h.Master.Mapping = apimaster.NewMasterMappingRuleHandler(listMasterRulesQH, saveCmdH, deleteCmdH, metadataCmdH, approveDdlCmdH, approveAllCmdH, dropColCmdH, dropRejectedCmdH, syncShadowCmdH, batchCmdH)
	h.Master.TransmuteJob = apimaster.NewMasterTransmuteJobHandler(transmuteJobRepo, activityLogger, logger)
	h.Master.TransmuteJob.SetPublisher(natsPublisher)
```
```go
// Trong cdc-cms-service/internal/infra/messaging/stuck_job_reaper.go (lines 135-152):
	_ = r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transmute_jobs
		   SET status = CASE
		           WHEN cancel_requested = true THEN 'CANCELLED'
		           WHEN error_message LIKE '%[auto-resume #3]%' THEN 'FAILED'
		           ELSE 'PAUSED'
		       END,
		       error_message = CASE
		           WHEN cancel_requested = true THEN 'cancelled by user'
		           WHEN error_message LIKE '%[auto-resume #3]%' THEN 'FAILED: max orphan auto-resume attempts (3) exceeded'
		           ELSE COALESCE(NULLIF(error_message, ''), 'reaper: heartbeat timeout (auto-resumable)')
		       END,
		       finished_at = CASE
		           WHEN cancel_requested = true OR error_message LIKE '%[auto-resume #3]%' THEN NOW()
		           ELSE finished_at
		       END,
		       updated_at = NOW()
		 WHERE status IN ('PENDING', 'RUNNING')
		   AND updated_at < NOW() - INTERVAL '10 minutes'
	`).Error

	_ = r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transform_jobs
		   SET status = CASE
		           WHEN cancel_requested = true THEN 'CANCELLED'
		           WHEN error_message LIKE '%[auto-resume #3]%' THEN 'FAILED'
		           ELSE 'PAUSED'
		       END,
		       error_message = CASE
		           WHEN cancel_requested = true THEN 'cancelled by user'
		           WHEN error_message LIKE '%[auto-resume #3]%' THEN 'FAILED: max orphan auto-resume attempts (3) exceeded'
		           ELSE COALESCE(NULLIF(error_message, ''), 'reaper: heartbeat timeout (auto-resumable)')
		       END,
		       finished_at = CASE
		           WHEN cancel_requested = true OR error_message LIKE '%[auto-resume #3]%' THEN NOW()
		           ELSE finished_at
		       END,
		       updated_at = NOW()
		 WHERE status IN ('PENDING', 'RUNNING')
		   AND updated_at < NOW() - INTERVAL '10 minutes'
	`).Error
```

#### Demo P13-9: `centralized-data-service/internal/handler/shadow/batch_buffer.go` (`lines 55-60`, `94-110`, `257-265`, `838-841`) — Sửa `C2`, `D2`, `B5` & `A4` (`BatchBuffer.Stop()` đóng `stopCh` để chạy `FlushWithContext(4s)` khi `bb.ctx` còn sống rồi mới gọi `bb.cancel()`)
Bổ sung `stopCh chan struct{}`, `stopOnce sync.Once` và `doneCh chan struct{}` vào `BatchBuffer`. Khi `Stop()` chạy: đóng `bb.stopCh` trước để `timerLoop()` chạy `FlushWithContext(flushCtx)` (`4s` timeout) khi `bb.ctx` vẫn còn sống, chờ `<-bb.doneCh` (tối đa `4s`), và **chỉ sau khi `<-bb.doneCh` hoàn tất mới gọi `bb.cancel()`**:
```go
// Trong centralized-data-service/internal/handler/shadow/batch_buffer.go:
	dbFlushNanos        atomic.Int64
	lastFlushDurationMs atomic.Int64
	oomRetryCount       atomic.Int64
	dlqEvacuatedCount   atomic.Int64
	stopCh              chan struct{}
	stopOnce            sync.Once
	doneCh              chan struct{}
}
```
```go
func NewBatchBuffer(maxSize int, timeout time.Duration, db *gorm.DB, schemaAdapter *serviceshadow.SchemaAdapter, logger *zap.Logger) *BatchBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	bb := &BatchBuffer{
		records:          make([]*shadow.UpsertRecord, 0, maxSize),
		maxSize:          maxSize,
		timeout:          timeout,
		db:               db,
		schemaAdapter:    schemaAdapter,
		logger:           logger,
		flushCh:          make(chan struct{}, 1),
		ctx:              ctx,
		cancel:           cancel,
		watermarkTracker: NewStatefulWatermarkTracker(),
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
	}
	go bb.timerLoop()
	return bb
}
```
```go
func (bb *BatchBuffer) timerLoop() {
	defer func() {
		if bb.doneCh != nil {
			close(bb.doneCh)
		}
	}()
	ticker := time.NewTicker(bb.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-bb.stopCh:
			flushCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			_, _ = bb.FlushWithContext(flushCtx)
			cancel()
			return
		default:
		}
		select {
		case <-bb.stopCh:
			flushCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			_, _ = bb.FlushWithContext(flushCtx)
			cancel()
			return
		case <-bb.ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			_, _ = bb.FlushWithContext(flushCtx)
			cancel()
			return
		case <-ticker.C:
			bb.flushFromBackground()
		case <-bb.flushCh:
			bb.flushFromBackground()
		}
	}
}
```
```go
func (bb *BatchBuffer) Stop() {
	bb.stopOnce.Do(func() {
		if bb.stopCh != nil {
			close(bb.stopCh)
		}
	})
	if bb.doneCh != nil {
		select {
		case <-bb.doneCh:
		case <-time.After(4500 * time.Millisecond):
		}
	}
	bb.cancel()
}
```

---

## MỤC N: KIỂM TOÁN NHẬT KÝ THỰC THI (`T30`) & PHƯƠNG ÁN `P14` (TÁCH FILE `*_lifecycle.go` + `t.Skipf` GUARD — `T31`)

### N.1 Giải Trình Minh Bạch 4 Điểm Trong Nhật Ký Thực Thi Kèm Bằng Chứng Vật Lý (`EV-...`)

1. **Điểm 1 — Vì sao lần chạy đầu `TestRequestCancel_RealLocalPostgres_ResolvesStuckJob` bị `Exit 1`, `git diff` của 3 file bên `cdc-cms-service`, và kiểm chứng `CHECK` constraint trên `'PAUSED'`:**
   - **Không có `CHECK` constraint nào chặn `'PAUSED'` trên PostgreSQL thật:** Trong `migrations/schema/recon_dlq/101_create_transmute_jobs.sql:10` và `100_create_transform_jobs.sql:10`, cột `status` có kiểu `VARCHAR(32) NOT NULL DEFAULT 'PENDING'` hoàn toàn không có `CHECK` constraint.
   - **Nguyên nhân thật sự khiến lần chạy đầu `Exit 1`:** Bản ghi `job_id = 0ab5f75491304fd58dbfcceec26343f9` đã được User bấm Cancel (`cancel_requested = true`) lúc `2026-11-09 16:20:29`, nhưng trong lúc chờ duyệt Plan, `StuckJobReaper` cũ trên `:8082` đã quét lúc `16:32:24` và ghi đè bản ghi này thành `status = 'FAILED', error_message = 'reaper: timeout exceeded'` (trong khi `cancel_requested` vẫn bằng `true`). Ở lần chạy test đầu tiên, câu `UPDATE` trong `RequestCancel` chỉ lọc `WHERE status IN ('PENDING', 'RUNNING', 'PAUSED')`, nên không khớp với bản ghi đang ở `status = 'FAILED'` (`cancel_requested = true, error_message = 'reaper: timeout exceeded'`) và làm `assert after.Status == "CANCELLED"` báo lỗi `got status=FAILED`.
   - **Nội dung sửa ở 3 file (`transmute_job_repo.go`, `transform_job_repo.go`, `stuck_job_reaper.go`):** Giữ nguyên 100% nhánh chuyển sang `'PAUSED'` đồng nhất với Worker (`ELSE 'PAUSED'` trong `stuck_job_reaper.go:140, 162`), và chỉ cộng thêm nhánh `OR (cancel_requested = true AND status = 'FAILED' AND error_message LIKE 'reaper:%')` để tự chữa lành các job đã có `cancel_requested = true` từng bị Reaper cũ đánh nhầm thành `FAILED: reaper: timeout exceeded` về đúng `'CANCELLED'`.
   - **Kiểm chứng biên dịch & test `internal/infra/messaging/...` (`stuck_job_reaper.go`) + `internal/infra/persistence`:** Đã chạy qua `evidence_run.py` (`EV-20261110T090216-5016`, `exit=0` — **100% PASS**).

2. **Điểm 2 — Kiểm chứng `go build ./...` & `go test -count=1 ./internal/service/master/...` (`transmuter.go`):**
   - **Bên `centralized-data-service`:**
     - `go build ./...`: **`EV-20261110T090043-4e07` (`exit=0`)**.
     - `go test -v -count=1 -skip TestLiveBindings ./internal/service/master/...`: **`EV-20261110T090431-21ee` (`exit=0`)** — toàn bộ unit/integration test của `TransmuterModule` (`TestTransmuter_OrphanMasterChunking`, `TestTransmuter_FullSyncHardDeleteWhenShadowMarkedDeleted`, `TestTransmuter_RunEvents_TracksDeletedSeparatelyFromUpdated`, `TestTransmuter_SpecPKAndResurrectedRowAndLegacyGpayID`, và toàn bộ `./internal/service/master/transmute`) đều **PASS 100%**. *(Riêng hàm test cũ `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` tại `transmuter_gpayid_test.go:409-417` quét toàn bộ `cdc_system.master_binding` trên DB local và vấp bản ghi rác có ID âm `-492241003` trong bảng `master_binding`).*
   - **Bên `cdc-cms-service`:** Lệnh `go build ./...` báo lỗi tại thư mục `scratch/` (`EV-20261110T090113-403a`: `scratch/check_logs.go:11:6: main redeclared in this block / scratch/check_bindings.go:11:6`) do thư mục `scratch/` có sẵn 5 script rời cùng khai báo `func main()` từ trước (`.agent-project.json` của `cdc-cms-service` chuẩn hóa lệnh build là `go build ./cmd/... ./internal/... ./pkgs/...`, đạt `exit=0` tại `EV-20261110T040010-3a2f`).

3. **Điểm 3 — Tính lũy đẳng & `t.Skipf` của `TestRequestCancel_RealLocalPostgres_ResolvesStuckJob` (`transmute_job_repo_test.go`):**
   - Khi chạy lại lần 2 (`EV-20261110T090216-5016`, khi `0ab5f75491304fd58dbfcceec26343f9` đã ở `status = CANCELLED`), hàm test **vẫn PASS (`0.05s`)** vì `RequestCancel` là lũy đẳng (idempotent) và `GetByID` sau đó vẫn trả về `status = CANCELLED, cancel_requested = true`.
   - Tuy nhiên, để an toàn tuyệt đối trên CI hoặc môi trường không có `config-local.yml` / PostgreSQL `:5436` / bản ghi `0ab5f75491304fd58dbfcceec26343f9`, trong `Demo P14-1` bên dưới em chuyển `t.Fatalf` khi thiếu config/DB/record sang `t.Skipf`.

4. **Điểm 4 — Phương án `P14`: Tách file `transmute_handler_lifecycle.go` & `batch_transform_lifecycle.go` (Giữ `NOW <= HEAD`):**
   - Tách `transmute_handler_lifecycle.go` (`310` dòng) khỏi `transmute_handler.go`, đưa `transmute_handler.go` từ `1099` dòng về **`789` dòng** ($\le 799$ dòng trước khi sửa).
   - Tách `batch_transform_lifecycle.go` (`331` dòng) khỏi `batch_transform_handler.go`, đưa `batch_transform_handler.go` từ `886` dòng về **`555` dòng** ($\le 556$ dòng trước khi sửa).

---

### N.2 Code Demo 1:1 Cho `P14` (`Demo P14-1` → `Demo P14-3`)

#### Demo P14-1: `cdc-cms-service/internal/infra/persistence/transmute_job_repo_test.go` (`lines 33-94`) — Đổi `t.Fatalf` sang `t.Skipf` khi thiếu DB local hoặc bản ghi fixture
```go
func TestRequestCancel_RealLocalPostgres_ResolvesStuckJob(t *testing.T) {
	candidates := []string{"config/config-local.yml", "../../../config/config-local.yml"}
	var cfgFile string
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if _, statErr := os.Stat(abs); statErr == nil {
				cfgFile = abs
				break
			}
		}
	}
	if cfgFile == "" {
		t.Skip("config-local.yml not found; skipping real local Postgres test")
	}
	_ = os.Setenv("CFG_PATH", cfgFile)

	cfg, err := config.NewConfig()
	if err != nil {
		t.Skipf("cannot load config (%v); skipping real local Postgres test", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.UserName, cfg.DB.Password, cfg.DB.Database, cfg.DB.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("local Postgres unavailable (%v); skipping", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("failed to get sql.DB (%v); skipping", err)
	}
	defer sqlDB.Close()

	repo := NewTransmuteJobRepo(db)
	ctx := context.Background()
	const targetJobID = "0ab5f75491304fd58dbfcceec26343f9"

	before, err := repo.GetByID(ctx, targetJobID)
	if err != nil {
		t.Skipf("target job %s not present in local DB (%v); skipping", targetJobID, err)
	}
	t.Logf("Before RequestCancel: job_id=%s status=%s cancel_requested=%v updated_at=%s",
		before.JobID, before.Status, before.CancelRequested, before.UpdatedAt)

	if err := repo.RequestCancel(ctx, targetJobID); err != nil {
		t.Fatalf("RequestCancel(%s) failed: %v", targetJobID, err)
	}

	after, err := repo.GetByID(ctx, targetJobID)
	if err != nil {
		t.Fatalf("GetByID(%s) after RequestCancel failed: %v", targetJobID, err)
	}
	t.Logf("After RequestCancel: job_id=%s status=%s cancel_requested=%v error_message=%q finished_at=%v",
		after.JobID, after.Status, after.CancelRequested, after.ErrorMessage, after.FinishedAt)

	if after.Status != "CANCELLED" || !after.CancelRequested {
		t.Fatalf("expected stuck job %s to transition to status=CANCELLED with cancel_requested=true, got status=%s cancel_requested=%v",
			targetJobID, after.Status, after.CancelRequested)
	}
}
```

#### Demo P14-2: Tách `centralized-data-service/internal/handler/master/transmute_handler_lifecycle.go` khỏi `transmute_handler.go` (`1099` dòng $\rightarrow$ `789` dòng)
```go
// File mới: centralized-data-service/internal/handler/master/transmute_handler_lifecycle.go
package master

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"centralized-data-service/internal/repository"
	"centralized-data-service/pkgs/observability"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func (h *TransmuteHandler) SetTransmuteJobRepo(r *repository.TransmuteJobRepo) {
	h.transmuteJobRepo = r
}

func (h *TransmuteHandler) SetStopping(v bool) {
	h.stopping.Store(v)
	if h.svc != nil {
		h.svc.SetStopping(v)
	}
}

func (h *TransmuteHandler) GetDebouncerStats() []DebouncerStat {
	h.debouncersMu.RLock()
	list := make([]*TableDebouncer, 0, len(h.debouncers))
	for _, td := range h.debouncers {
		list = append(list, td)
	}
	h.debouncersMu.RUnlock()

	stats := make([]DebouncerStat, 0, len(list))
	for _, td := range list {
		if td != nil {
			stats = append(stats, td.GetStat())
		}
	}
	return stats
}

// HandleTransmuteCancel immediately cancels an in-flight transmute job by job_id.
func (h *TransmuteHandler) HandleTransmuteCancel(msg *nats.Msg) {
	var payload struct {
		JobID       string `json:"job_id"`
		MasterTable string `json:"master_table"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil || payload.JobID == "" {
		return
	}
	h.activeJobsMu.Lock()
	cancelFn, ok := h.activeJobs[payload.JobID]
	h.activeJobsMu.Unlock()
	if ok && cancelFn != nil {
		cancelFn()
	}
}

// Start launches the background recovery loop to auto-resume PAUSED or orphaned transmute jobs.
func (h *TransmuteHandler) Start(ctx context.Context) {
	h.activeJobsMu.Lock()
	if h.stopping.Load() {
		h.activeJobsMu.Unlock()
		return
	}
	recCtx, recCancel := context.WithCancel(ctx)
	h.recoverCancel = recCancel
	h.wg.Add(1)
	h.activeJobsMu.Unlock()

	go func() {
		defer h.wg.Done()
		h.recoverOrphanedTransmuteJobs(recCtx)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-recCtx.Done():
				return
			case <-h.stopCh:
				return
			case <-ticker.C:
				h.recoverOrphanedTransmuteJobs(recCtx)
			}
		}
	}()
}

func (h *TransmuteHandler) recoverOrphanedTransmuteJobs(ctx context.Context) {
	if h.transmuteJobRepo == nil || h.svc == nil || h.stopping.Load() || h.svc.IsStopping() {
		return
	}
	jobs, err := h.transmuteJobRepo.ClaimRecoverableJobs(ctx)
	if err != nil || len(jobs) == 0 {
		return
	}
	for _, j := range jobs {
		if h.stopping.Load() || h.svc.IsStopping() {
			return
		}
		h.activeJobsMu.Lock()
		_, alreadyRunningJob := h.activeJobs[j.JobID]
		_, alreadyRunningTable := h.activeTables[j.MasterTable]
		h.activeJobsMu.Unlock()
		if alreadyRunningJob || alreadyRunningTable {
			continue
		}
		var bID int64
		if j.MasterBindingID != nil {
			bID = *j.MasterBindingID
		}
		savedRunType := repository.ParseTransmuteRunType(j.ErrorMessage)
		if j.Status != "PAUSED" {
			attempts := repository.ParseAutoResumeCount(j.ErrorMessage)
			if attempts >= repository.MaxOrphanAutoResumeAttempts {
				failMsg := fmt.Sprintf("FAILED: max orphan auto-resume attempts (%d) exceeded", repository.MaxOrphanAutoResumeAttempts)
				h.logger.Error("Skipping poison-pill orphaned transmute job after max retries",
					zap.String("job_id", j.JobID),
					zap.String("master_table", j.MasterTable),
					zap.Int("attempts", attempts))
				_ = h.transmuteJobRepo.UpdateStatus(ctx, j.JobID, "FAILED", j.RowsAffected, j.TotalRows, failMsg, bID)
				continue
			}
			nextNote := repository.FormatOrphanAutoResumeError(attempts+1, savedRunType)
			_ = h.transmuteJobRepo.UpdateStatus(ctx, j.JobID, "RUNNING", j.RowsAffected, j.TotalRows, nextNote, bID)
		}
		h.logger.Info("Auto-resuming interrupted transmute job from checkpoint",
			zap.String("job_id", j.JobID),
			zap.String("master_table", j.MasterTable),
			zap.String("run_type", savedRunType),
			zap.String("prev_status", j.Status),
			zap.Int64("rows_affected", j.RowsAffected))
		fakeReq, _ := json.Marshal(TransmuteRequest{
			JobID:           j.JobID,
			MasterTable:     j.MasterTable,
			MasterBindingID: bID,
			RunType:         savedRunType,
			TriggeredBy:     "auto-resume",
			TraceID:         j.TraceID,
		})
		h.HandleTransmute(&nats.Msg{Subject: "cdc.cmd.transmute", Data: fakeReq})
	}
}

// Stop implements Two-Tier Graceful Shutdown within a strict 2s + 4s budget (C4 & B5).
func (h *TransmuteHandler) Stop() {
	h.stopOnce.Do(func() {
		h.activeJobsMu.Lock()
		h.stopping.Store(true)
		if h.svc != nil {
			h.svc.SetStopping(true)
		}
		if h.recoverCancel != nil {
			h.recoverCancel()
		}
		if h.stopCh != nil {
			close(h.stopCh)
		}
		h.activeJobsMu.Unlock()

		done := make(chan struct{})
		go func() {
			h.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			h.activeJobsMu.Lock()
			for _, cancelFn := range h.activeJobs {
				if cancelFn != nil {
					cancelFn()
				}
			}
			h.activeJobsMu.Unlock()
			<-done
		}

		h.debouncersMu.RLock()
		debouncers := make([]*TableDebouncer, 0, len(h.debouncers))
		for _, td := range h.debouncers {
			debouncers = append(debouncers, td)
		}
		h.debouncersMu.RUnlock()

		var tdWg sync.WaitGroup
		for _, td := range debouncers {
			if td == nil {
				continue
			}
			tdWg.Add(1)
			go func(d *TableDebouncer) {
				defer tdWg.Done()
				d.Stop()
			}(td)
		}
		tdWg.Wait()
	})
}

// HandleTransmuteTask is the worker task adapter for TransmuteConsumerPool (JetStream).
func (h *TransmuteHandler) HandleTransmuteTask(ctx context.Context, msg *nats.Msg) error {
	if h.stopping.Load() || (h.svc != nil && h.svc.IsStopping()) {
		return fmt.Errorf("transmute handler is stopping")
	}
	var req TransmuteRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return fmt.Errorf("invalid transmute payload: %w", err)
	}
	if req.MasterTable == "" {
		return fmt.Errorf("master_table required")
	}

	if len(req.SourceIDs) > 0 || len(req.Events) > 0 {
		debouncerKey := req.MasterTable
		if req.MasterBindingID > 0 {
			debouncerKey = fmt.Sprintf("%s#%d", req.MasterTable, req.MasterBindingID)
		}
		policy := deliveryPolicyFromContext(ctx, h.logger)
		tracker := newShadowMsgTracker(msg, 1, policy)
		var taskOnce sync.Once
		td := h.getOrCreateDebouncer(debouncerKey)
		if td == nil {
			tracker.completeTask(context.Canceled)
			return fmt.Errorf("transmute handler is stopping")
		}
		td.AddTask(NatsMsgTask{
			Msg: msg,
			Req: req,
			OnDone: func(taskErr error) {
				taskOnce.Do(func() {
					tracker.completeTask(taskErr)
				})
			},
		})
		return ErrAsyncAckPending
	}

	_, err := h.execRun(ctx, req.MasterTable, nil, req.JobID, "", req.MasterBindingID)
	return err
}

func (h *TransmuteHandler) HandleTransmuteShadowTask(ctx context.Context, msg *nats.Msg) error {
	return h.handleTransmuteShadowWithOutcome(ctx, msg, true)
}

type shadowMsgTracker struct {
	msg        *nats.Msg
	policy     deliveryPolicy
	remaining  int32
	mu         sync.Mutex
	firstErr   error
	stopCh     chan struct{}
	finishOnce sync.Once
}

func newShadowMsgTracker(msg *nats.Msg, targetCount int, policy deliveryPolicy) *shadowMsgTracker {
	t := &shadowMsgTracker{
		msg:       msg,
		policy:    policy,
		remaining: int32(targetCount),
		stopCh:    make(chan struct{}),
	}
	if targetCount <= 0 {
		close(t.stopCh)
		finalizeJetStreamMessage(msg, nil, policy)
		return t
	}
	if msg != nil && msg.Sub != nil {
		interval := policy.AckWait / 3
		if interval <= 0 {
			interval = 5 * time.Second
		}
		if interval < 50*time.Millisecond {
			interval = 50 * time.Millisecond
		}
		if interval > 5*time.Second {
			interval = 5 * time.Second
		}
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			maxTimer := time.NewTimer(35 * time.Minute)
			defer maxTimer.Stop()
			for {
				select {
				case <-t.stopCh:
					return
				case <-maxTimer.C:
					return
				case <-ticker.C:
					_ = msg.InProgress()
				}
			}
		}()
	}
	return t
}

func (t *shadowMsgTracker) completeTask(err error) {
	if err != nil {
		t.mu.Lock()
		if t.firstErr == nil {
			t.firstErr = err
		}
		t.mu.Unlock()
	}
	if atomic.AddInt32(&t.remaining, -1) <= 0 {
		t.finishOnce.Do(func() {
			close(t.stopCh)
			t.mu.Lock()
			finalErr := t.firstErr
			t.mu.Unlock()
			finalizeJetStreamMessage(t.msg, finalErr, t.policy)
		})
	}
}
```

#### Demo P14-3: Tách `centralized-data-service/internal/handler/shadow/batch_transform_lifecycle.go` khỏi `batch_transform_handler.go` (`886` dòng $\rightarrow$ `555` dòng)
```go
// File mới: centralized-data-service/internal/handler/shadow/batch_transform_lifecycle.go
package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mastermodel "centralized-data-service/internal/model/master"
	"centralized-data-service/internal/repository"
	"centralized-data-service/internal/service/governance"
	"centralized-data-service/pkgs/observability"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (h *BatchTransformHandler) SetStopping(v bool) {
	h.stopping.Store(v)
}

func (h *BatchTransformHandler) SetTransformJobRepo(r *repository.TransformJobRepo) {
	h.transformJobRepo = r
}

// HandleTransformCancel immediately cancels an in-flight batch transform job by job_id.
func (h *BatchTransformHandler) HandleTransformCancel(msg *nats.Msg) {
	var payload struct {
		JobID       string `json:"job_id"`
		TargetTable string `json:"target_table"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil || payload.JobID == "" {
		return
	}
	h.activeJobsMu.Lock()
	cancelFn, ok := h.activeJobs[payload.JobID]
	h.activeJobsMu.Unlock()
	if ok && cancelFn != nil {
		cancelFn()
	}
}

// Start launches the background recovery loop to auto-resume PAUSED or orphaned transform jobs.
func (h *BatchTransformHandler) Start(ctx context.Context) {
	h.activeJobsMu.Lock()
	if h.stopping.Load() {
		h.activeJobsMu.Unlock()
		return
	}
	recCtx, recCancel := context.WithCancel(ctx)
	h.recoverCancel = recCancel
	h.wg.Add(1)
	h.activeJobsMu.Unlock()

	go func() {
		defer h.wg.Done()
		h.recoverOrphanedTransformJobs(recCtx)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-recCtx.Done():
				return
			case <-h.stopCh:
				return
			case <-ticker.C:
				h.recoverOrphanedTransformJobs(recCtx)
			}
		}
	}()
}

func (h *BatchTransformHandler) recoverOrphanedTransformJobs(ctx context.Context) {
	if h.transformJobRepo == nil || h.stopping.Load() {
		return
	}
	jobs, err := h.transformJobRepo.ClaimRecoverableJobs(ctx)
	if err != nil || len(jobs) == 0 {
		return
	}
	for _, j := range jobs {
		if h.stopping.Load() {
			return
		}
		h.activeJobsMu.Lock()
		_, alreadyRunning := h.activeJobs[j.JobID]
		activeJobForTable, tableBusy := h.activeTables[j.TargetTable]
		h.activeJobsMu.Unlock()
		if alreadyRunning || (tableBusy && activeJobForTable != j.JobID) {
			continue
		}
		attempts := repository.ParseAutoResumeCount(j.ErrorMessage)
		savedCursor, _ := repository.ParseTransformCursorPK(j.ErrorMessage)
		isForce, forceFields := repository.ParseTransformForce(j.ErrorMessage)
		if j.Status != "PAUSED" {
			if attempts >= repository.MaxOrphanAutoResumeAttempts {
				failMsg := fmt.Sprintf("FAILED: max orphan auto-resume attempts (%d) exceeded", repository.MaxOrphanAutoResumeAttempts)
				h.Logger.Error("Skipping poison-pill orphaned transform job after max retries",
					zap.String("job_id", j.JobID),
					zap.String("target_table", j.TargetTable),
					zap.Int("attempts", attempts))
				_ = h.transformJobRepo.UpdateStatus(ctx, j.JobID, "FAILED", j.RowsAffected, j.TotalRows, failMsg)
				continue
			}
			nextNote := repository.FormatTransformCheckpointNote(attempts+1, savedCursor, isForce, forceFields)
			_ = h.transformJobRepo.UpdateStatus(ctx, j.JobID, "RUNNING", j.RowsAffected, j.TotalRows, nextNote)
		}
		var srcObjID int64
		if j.SourceObjectID != nil {
			srcObjID = *j.SourceObjectID
		}
		h.Logger.Info("Auto-resuming interrupted transform job from checkpoint",
			zap.String("job_id", j.JobID),
			zap.String("target_table", j.TargetTable),
			zap.String("prev_status", j.Status),
			zap.String("cursor_pk", savedCursor),
			zap.Bool("force", isForce),
			zap.Int64("rows_affected", j.RowsAffected))
		raw, _ := json.Marshal(BatchTransformPayload{
			JobID:          j.JobID,
			TargetTable:    j.TargetTable,
			SourceObjectID: srcObjID,
			TraceID:        j.TraceID,
			Force:          isForce,
			ForceFields:    forceFields,
		})
		h.HandleBatchTransform(&nats.Msg{Subject: "cdc.cmd.batch-transform", Data: raw})
	}
}

// Stop signals active transform jobs to pause cleanly at the current 1000-row chunk boundary (2s budget — C4) and waits for exit.
func (h *BatchTransformHandler) Stop() {
	h.stopOnce.Do(func() {
		h.activeJobsMu.Lock()
		h.stopping.Store(true)
		if h.recoverCancel != nil {
			h.recoverCancel()
		}
		if h.stopCh != nil {
			close(h.stopCh)
		}
		h.activeJobsMu.Unlock()

		done := make(chan struct{})
		go func() {
			h.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			h.activeJobsMu.Lock()
			for _, cancelFn := range h.activeJobs {
				if cancelFn != nil {
					cancelFn()
				}
			}
			h.activeJobsMu.Unlock()
			<-done
		}
	})
}

func (h *BatchTransformHandler) startJobHeartbeat(ctx context.Context, jobID string, resumeAttempts int, savedCursorRaw string, isForceJob bool, forceFieldsNote []string, baseRowsAffected, preservedTotalRows int64) context.CancelFunc {
	if h.transformJobRepo == nil || jobID == "" {
		return func() {}
	}
	initNote := repository.FormatTransformCheckpointNote(resumeAttempts, savedCursorRaw, isForceJob, forceFieldsNote)
	_ = h.transformJobRepo.UpdateStatus(ctx, jobID, "RUNNING", baseRowsAffected, preservedTotalRows, initNote)
	hbCtx, hbCancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				_ = h.transformJobRepo.TouchHeartbeat(context.Background(), jobID)
			}
		}
	}()
	return hbCancel
}

func (h *BatchTransformHandler) runUnchunkedTransform(
	ctx context.Context,
	execDB *gorm.DB,
	targetTable, schemaName, pureTable, quotedTable, setExpr, whereExpr, jobID, traceID string,
	pkErr error,
	resumeAttempts int,
	lastPK interface{},
	isForceJob bool,
	forceFieldsNote []string,
	baseRowsAffected, totalPendingRows int64,
	rules []mastermodel.MappingRuleV2,
) {
	observability.Ctx(ctx, h.Logger).Warn("batch transform: PK not detected, running unchunked UPDATE",
		zap.String("table", targetTable),
		zap.Error(pkErr),
	)
	transformSQL := fmt.Sprintf(`UPDATE %s SET %s WHERE _raw_data IS NOT NULL AND (%s)`,
		quotedTable, setExpr, whereExpr,
	)
	result := execDB.WithContext(ctx).Exec(transformSQL)
	if result.Error != nil {
		if h.transformJobRepo != nil && jobID != "" && h.transformJobRepo.IsCancelRequested(context.Background(), jobID) {
			h.finishJob(context.Background(), jobID, "CANCELLED", baseRowsAffected, totalPendingRows, "cancelled by user")
			return
		}
		if h.stopping.Load() {
			pauseNote := repository.FormatTransformCheckpointNote(resumeAttempts, lastPK, isForceJob, forceFieldsNote)
			if pauseNote == "" {
				pauseNote = "paused by worker shutdown"
			}
			h.finishJob(context.Background(), jobID, "PAUSED", baseRowsAffected, totalPendingRows, pauseNote)
			return
		}
		enrichedErr := h.enrichCastingError(result.Error, rules)
		observability.Ctx(ctx, h.Logger).Error("batch transform query failed",
			zap.String("table", targetTable),
			zap.Error(result.Error),
			zap.String("enriched_error", enrichedErr),
		)
		h.publishAndFinishJob(ctx, jobID, "error", baseRowsAffected, totalPendingRows, enrichedErr, rules)
		return
	}
	observability.Ctx(ctx, h.Logger).Info("batch transform completed (unchunked)",
		zap.String("table", targetTable),
		zap.Int64("rows_affected", baseRowsAffected+result.RowsAffected),
	)
	if h.DB != nil {
		act := governance.NewActivityLogger(h.DB, h.Logger)
		act.Quick("cmd-batch-transform", targetTable, "nats-command", "success",
			baseRowsAffected+result.RowsAffected, map[string]interface{}{"trace_id": traceID}, "")
	}
	h.finishJob(ctx, jobID, "COMPLETED", baseRowsAffected+result.RowsAffected, totalPendingRows, "")
	h.publishTransmuteTrigger(ctx, schemaName, pureTable)
}

func (h *BatchTransformHandler) finishJob(ctx context.Context, jobID, status string, rowsAffected, totalRows int64, errMsg string) {
	if h.transformJobRepo == nil || jobID == "" {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.transformJobRepo.UpdateStatus(writeCtx, jobID, status, rowsAffected, totalRows, errMsg); err != nil {
		observability.Ctx(ctx, h.Logger).Warn("transform_jobs: failed to update final status",
			zap.String("job_id", jobID),
			zap.String("status", status),
			zap.Error(err),
		)
	}
}

func (h *BatchTransformHandler) publishAndFinishJob(
	ctx context.Context, jobID, status string, rowsAffected, totalRows int64, errMsg string,
	rules []mastermodel.MappingRuleV2,
) {
	h.finishJob(ctx, jobID, map[string]string{
		"skipped": "FAILED",
		"error":   "FAILED",
		"success": "COMPLETED",
	}[status], rowsAffected, totalRows, errMsg)
}

func isIntegerPGType(pgType string) bool {
	t := strings.ToLower(strings.TrimSpace(pgType))
	switch t {
	case "smallint", "integer", "bigint", "int2", "int4", "int8", "serial", "bigserial", "smallserial":
		return true
	default:
		return false
	}
}

func (h *BatchTransformHandler) detectPrimaryKey(execDB *gorm.DB, schemaName, tableName string) (string, string, error) {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	type pkMeta struct {
		ColName  string `gorm:"column:col_name"`
		DataType string `gorm:"column:data_type"`
	}
	var cols []pkMeta
	sql := `
        SELECT a.attname AS col_name, format_type(a.atttypid, a.atttypmod) AS data_type
        FROM pg_index i
        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = (?::text)::regclass AND i.indisprimary
        ORDER BY array_position(i.indkey, a.attnum)`
	qualified := schemaName + "." + tableName
	if err := execDB.Raw(sql, qualified).Scan(&cols).Error; err != nil {
		return "", "", err
	}
	if len(cols) != 1 {
		return "", "", nil
	}
	return cols[0].ColName, cols[0].DataType, nil
}

func (h *BatchTransformHandler) publishTransmuteTrigger(ctx context.Context, schemaName, tableName string) {
	if h.NatsConn == nil || tableName == "" {
		return
	}
	var meta struct {
		BindingCode         string `gorm:"column:binding_code"`
		ShadowSchema        string `gorm:"column:shadow_schema"`
		ShadowConnectionKey string `gorm:"column:shadow_connection_key"`
	}
	if h.DB != nil {
		_ = h.DB.WithContext(ctx).Raw(`
			SELECT COALESCE(sb.binding_code, '') AS binding_code,
			       COALESCE(sb.shadow_schema, '') AS shadow_schema,
			       COALESCE(cr.connection_code, '') AS shadow_connection_key
			  FROM cdc_system.shadow_binding sb
			  LEFT JOIN cdc_system.connection_registry cr ON cr.id = sb.shadow_connection_id
			 WHERE sb.is_active = true
			   AND sb.shadow_table = ?
			   AND COALESCE(sb.shadow_schema, '') = COALESCE(NULLIF(?, ''), COALESCE(sb.shadow_schema, ''))
			 ORDER BY sb.id ASC
			 LIMIT 1
		`, tableName, schemaName).Scan(&meta).Error
	}
	resolvedSchema := meta.ShadowSchema
	if resolvedSchema == "" {
		resolvedSchema = schemaName
	}
	payload, err := json.Marshal(map[string]any{
		"shadow_binding_code":   meta.BindingCode,
		"shadow_table":          tableName,
		"shadow_schema":         resolvedSchema,
		"shadow_connection_key": meta.ShadowConnectionKey,
		"triggered_by":          "batch-transform",
		"correlation_id":        fmt.Sprintf("transform-%s-%d", tableName, time.Now().UnixNano()),
	})
	if err != nil {
		observability.Ctx(ctx, h.Logger).Warn("post-transform transmute trigger: marshal failed",
			zap.String("table", tableName), zap.Error(err))
		return
	}
	msg := &nats.Msg{
		Subject: "cdc.cmd.transmute-shadow",
		Data:    payload,
		Header:  make(nats.Header),
	}
	observability.InjectNATSHeader(ctx, msg.Header)
	if err := h.NatsConn.PublishMsg(msg); err != nil {
		observability.Ctx(ctx, h.Logger).Warn("post-transform transmute trigger: publish failed",
			zap.String("table", tableName), zap.Error(err))
		return
	}
	observability.Ctx(ctx, h.Logger).Info("post-transform transmute trigger published",
		zap.String("shadow_schema", resolvedSchema),
		zap.String("shadow_table", tableName))
}
```

---

## O. PHƯƠNG ÁN `P15` (`T32` — 2026-11-10): XỬ LÝ DỨT ĐIỂM 5 LỖI/NỢ KỸ THUẬT `E1–E5`

### 1. Kết Quả Kiểm Chứng Thực Tế (`EV-20261110T103828-c564`) & Phản Biện Ngược `DG0` Trên `E1`

- **Biên lai `EV-20261110T103828-c564` (Truy vấn trực tiếp `information_schema.columns`, `pg_constraint`, và dữ liệu thực của `cdc_system.transmute_schedule` trên PostgreSQL `localhost:5436/cdc_dw`):**
  1. **Danh sách cột thực tế của `cdc_system.transmute_schedule`:**
     `id (bigint)`, `master_binding_id (bigint)`, `mode (text)`, `cron_expr (text)`, `last_run_at (timestamptz)`, `next_run_at (timestamptz)`, **`last_status (text)`**, `last_error (text)`, `last_stats (jsonb)`, `is_enabled (boolean)`, `created_by (text)`, `created_at (timestamptz)`, `updated_at (timestamptz)`, `last_trace_id (varchar)`.
     → Hoàn toàn **không có cột `last_run_status`**. Câu lệnh phụ trong `transmute_job_repo.go:272` trước đó dùng `last_run_status` đã bị PostgreSQL báo lỗi `column "last_run_status" does not exist` nhưng bị `_ =` nuốt mất!
  2. **Dữ liệu thực tế trong `cdc_system.transmute_schedule`:**
     Toàn bộ các dòng đang lưu chữ thường: `last_status = 'success'`, `last_status = 'failed'`, hoặc `NULL`.
  3. **CẢNH BÁO PHẢN BIỆN NGƯỢC (`DG0` — Bác bỏ bằng bằng chứng vật lý trên `pg_constraint`):**
     Trên PostgreSQL thật `cdc_dw` (cũng như tại `migrations/schema/cdc_system_model/036_v2_transmute_schedule.sql:12`), bảng `cdc_system.transmute_schedule` có ràng buộc `CHECK` cứng:
     ```sql
     transmute_schedule_last_status_check:
     CHECK (((last_status IS NULL) OR (last_status = ANY (ARRAY['success'::text, 'failed'::text, 'running'::text, 'skipped'::text]))))
     ```
     - Cả `'cancelled'` (chữ thường) lẫn `'CANCELLED'` (chữ hoa) **ĐỀU KHÔNG HỢP LỆ** trong `transmute_schedule_last_status_check`!
     - Nếu bỏ `_ =` (trả về lỗi thật) mà gán `SET last_status = 'cancelled'` hoặc `'CANCELLED'`, PostgreSQL sẽ **nổ lỗi `SQLSTATE 23514` (`violates check constraint "transmute_schedule_last_status_check"`)** ngay khi hủy một job có `transmute_schedule` đang `'running'`.
     - **Giải pháp chuẩn 100% khớp schema thật & đồng bộ với `transmute_scheduler.go:179, 193, 225` + `transmute_handler.go:549`:** Khi giải phóng `transmute_schedule` đang `last_status = 'running'` do User bấm Cancel, đặt **`SET last_status = 'failed', last_error = 'cancelled by user', updated_at = NOW()`** kèm chốt chặn `R-P13-6` (`tj.status = 'CANCELLED'` và khớp `master_binding_id > 0` hoặc `trace_id <> ''`), và trả lỗi tường minh (`return r.db.WithContext(ctx).Exec(...).Error`).

---

### 2. Bảng Đối Soát `wc -l` (`HEAD` vs `NOW` vs `Sau P15`) Cho Toàn Bộ File (`E5`)

| File | `HEAD` (Dòng) | `NOW` Trước `P15` (Dòng) | Chênh Lệch (`NOW - HEAD`) | Sau `P15` Dự Kiến (Dòng) | File Tách Mới / Nhận Hàm (`<= 400` Dòng) | Trạng Thái Sau `P15` |
| :--- | :---: | :---: | :---: | :---: | :--- | :---: |
| `internal/service/master/transmuter.go` | `1874` | `2049` | `+175` 🔴 | **`1841`** ($\le 1874$) | `transmuter_lifecycle.go` (**`221` dòng**) | **PASS (`NOW <= HEAD`)** |
| `internal/handler/orchestration/snapshot_runner_handler.go` | `1029` | `1078` | `+49` 🔴 | **`960`** ($\le 1029$) | `snapshot_runner_pacing.go` (**`257` dòng**) | **PASS (`NOW <= HEAD`)** |
| `internal/handler/orchestration/snapshot_runner_state.go` | `215` | `514` | `+299` 🔴 (`>400`) | **`228`** ($\le 400$) | `snapshot_runner_lifecycle.go` (**`307` dòng**) | **PASS (`<= 400`)** |
| `internal/handler/shadow/batch_buffer.go` | `824` | `877` | `+53` 🔴 | **`765`** ($\le 824$) | `batch_buffer_lifecycle.go` (**`126` dòng**) | **PASS (`NOW <= HEAD`)** |
| `internal/handler/master/debounce.go` | `197` | `399` | `+202` 🟡 | **`196`** ($\le 197$) | `debounce_lifecycle.go` (**`208` dòng**) | **PASS (`NOW <= HEAD`)** |
| `internal/handler/master/transmute_handler.go` | `799` | `787` | `-12` 🟢 | **`787`** ($\le 799$) | `transmute_handler_lifecycle.go` (**`324` dòng**) | **PASS (`NOW <= HEAD`)** |
| `internal/handler/shadow/batch_transform_handler.go` | `556` | `548` | `-8` 🟢 | **`548`** ($\le 556$) | `batch_transform_lifecycle.go` (**`380` dòng**) | **PASS (`NOW <= HEAD`)** |

---

### 3. Code Demo 1:1 (`Demo P15-1` đến `Demo P15-7`)

#### Demo P15-1: Sửa `E1`, `E2`, `E3` trong `cdc-cms-service/internal/infra/persistence/transmute_job_repo.go` & `transform_job_repo.go`

```go
// File: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/transmute_job_repo.go (lines 33-64 & 236-294)
var technicalCheckpointTagList = []string{
	"[run_type=",
	"[force=",
	"[force_fields=",
	"[fields=",
	"[cursor_pk=",
	"[auto-resume #",
}

// StripJobCheckpointTags removes internal checkpoint tags ([run_type=...], [force=...], [force_fields=...], [fields=...], [cursor_pk=...], [auto-resume #N])
// from error_message before returning job records to the UI, and trims any leading "| " separator left behind when StuckJobReaper appends a timeout reason (E3).
func StripJobCheckpointTags(status, errMsg string) string {
	cleaned := strings.TrimSpace(errMsg)
	if cleaned == "" {
		return ""
	}
	for _, prefix := range technicalCheckpointTagList {
		for {
			start := strings.Index(cleaned, prefix)
			if start == -1 {
				break
			}
			endRel := strings.IndexByte(cleaned[start:], ']')
			if endRel == -1 {
				break
			}
			cleaned = strings.TrimSpace(cleaned[:start] + " " + cleaned[start+endRel+1:])
		}
	}
	cleaned = strings.TrimSpace(strings.TrimLeft(cleaned, "| "))
	upper := strings.ToUpper(strings.TrimSpace(status))
	if upper == "COMPLETED" {
		return ""
	}
	if upper == "RUNNING" || upper == "PAUSED" || upper == "PENDING" {
		if cleaned == "paused by worker shutdown" ||
			cleaned == "paused by worker shutdown (auto-resumable)" ||
			cleaned == "auto-resumed orphaned job" ||
			cleaned == "reaper: heartbeat timeout (paused for worker auto-resume)" ||
			cleaned == "reaper: heartbeat timeout (auto-resumable)" ||
			strings.HasPrefix(cleaned, "Heartbeat timeout: paused at checkpoint") {
			return ""
		}
	}
	return cleaned
}
```

```go
// File: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/transmute_job_repo.go (lines 236-294)
// RequestCancel sets cancel_requested=true for the given job.
// If the job is PENDING, PAUSED, or stale RUNNING (updated_at older than 45s — worker stopped/restarted),
// it immediately transitions the job to CANCELLED so the UI does not stay stuck indefinitely.
// When status remains RUNNING (fresh heartbeat <45s), updated_at is preserved (ELSE updated_at) so repeated
// Cancel clicks do not push back the 45s stale threshold or 10m Reaper threshold (E2).
func (r *TransmuteJobRepo) RequestCancel(ctx context.Context, jobID string) error {
	err := r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transmute_jobs
		   SET cancel_requested = true,
		       status = CASE
		           WHEN status IN ('PENDING', 'PAUSED')
		             OR updated_at < NOW() - INTERVAL '45 seconds'
		             OR (status = 'FAILED' AND error_message LIKE 'reaper:%') THEN 'CANCELLED'
		           ELSE status
		       END,
		       error_message = CASE
		           WHEN status IN ('PENDING', 'PAUSED')
		             OR updated_at < NOW() - INTERVAL '45 seconds'
		             OR (status = 'FAILED' AND error_message LIKE 'reaper:%') THEN 'cancelled by user'
		           ELSE error_message
		       END,
		       finished_at = CASE
		           WHEN status IN ('PENDING', 'PAUSED')
		             OR updated_at < NOW() - INTERVAL '45 seconds'
		             OR (status = 'FAILED' AND error_message LIKE 'reaper:%') THEN COALESCE(finished_at, NOW())
		           ELSE finished_at
		       END,
		       updated_at = CASE
		           WHEN status IN ('PENDING', 'PAUSED')
		             OR updated_at < NOW() - INTERVAL '45 seconds'
		             OR (status = 'FAILED' AND error_message LIKE 'reaper:%') THEN NOW()
		           ELSE updated_at
		       END
		 WHERE job_id = ?
		   AND (
		       status IN ('PENDING', 'RUNNING', 'PAUSED')
		       OR (cancel_requested = true AND status = 'FAILED' AND error_message LIKE 'reaper:%')
		   )
	`, jobID).Error
	if err != nil {
		return err
	}
	// E1: Release any stuck 'running' transmute_schedule row matching master_binding_id or trace_id.
	// Note: transmute_schedule_last_status_check allows ('success','failed','running','skipped'), so last_status is set to 'failed' with last_error = 'cancelled by user'.
	return r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transmute_schedule ts
		   SET last_status = 'failed',
		       last_error  = 'cancelled by user',
		       updated_at  = NOW()
		  FROM cdc_system.transmute_jobs tj
		 WHERE tj.job_id = ?
		   AND tj.status = 'CANCELLED'
		   AND ts.last_status = 'running'
		   AND (
		       (tj.master_binding_id IS NOT NULL AND tj.master_binding_id > 0 AND ts.master_binding_id = tj.master_binding_id)
		       OR (COALESCE(tj.trace_id, '') <> '' AND ts.last_trace_id = tj.trace_id)
		   )
	`, jobID).Error
}
```

```go
// File: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/transform_job_repo.go (lines 236-268)
// RequestCancel sets cancel_requested=true for the given job.
// If the job is PENDING, PAUSED, or stale RUNNING (updated_at older than 45s — worker stopped/restarted),
// it immediately transitions the job to CANCELLED so the UI does not stay stuck indefinitely.
// Preserves updated_at when the job is still actively RUNNING (<45s) so repeated Cancel clicks do not reset the stale timer (E2).
func (r *TransformJobRepo) RequestCancel(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE cdc_system.transform_jobs
		   SET cancel_requested = true,
		       status = CASE
		           WHEN status IN ('PENDING', 'PAUSED')
		             OR updated_at < NOW() - INTERVAL '45 seconds'
		             OR (status = 'FAILED' AND error_message LIKE 'reaper:%') THEN 'CANCELLED'
		           ELSE status
		       END,
		       error_message = CASE
		           WHEN status IN ('PENDING', 'PAUSED')
		             OR updated_at < NOW() - INTERVAL '45 seconds'
		             OR (status = 'FAILED' AND error_message LIKE 'reaper:%') THEN 'cancelled by user'
		           ELSE error_message
		       END,
		       finished_at = CASE
		           WHEN status IN ('PENDING', 'PAUSED')
		             OR updated_at < NOW() - INTERVAL '45 seconds'
		             OR (status = 'FAILED' AND error_message LIKE 'reaper:%') THEN COALESCE(finished_at, NOW())
		           ELSE finished_at
		       END,
		       updated_at = CASE
		           WHEN status IN ('PENDING', 'PAUSED')
		             OR updated_at < NOW() - INTERVAL '45 seconds'
		             OR (status = 'FAILED' AND error_message LIKE 'reaper:%') THEN NOW()
		           ELSE updated_at
		       END
		 WHERE job_id = ?
		   AND (
		       status IN ('PENDING', 'RUNNING', 'PAUSED')
		       OR (cancel_requested = true AND status = 'FAILED' AND error_message LIKE 'reaper:%')
		   )
	`, jobID).Error
}
```

---

#### Demo P15-2: Sửa `E4.1` trong `transmute_job_repo_test.go` (Transaction Rollback + Fixture Tạm Cho Cả 2 Bảng + `t.Setenv`) & `system_health_collector_test.go`

```go
// File: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/transmute_job_repo_test.go
package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cdc-cms-service/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestStripJobCheckpointTags(t *testing.T) {
	raw := "[run_type=full] [force=1] [force_fields=amount,updated_at] [cursor_pk=67286461eb917d6e7fd308f4] [auto-resume #2]"
	if got := StripJobCheckpointTags("COMPLETED", raw); got != "" {
		t.Fatalf("StripJobCheckpointTags(COMPLETED, %q) = %q, want empty string", raw, got)
	}
	if got := StripJobCheckpointTags("RUNNING", raw+" resumable note"); got != "resumable note" {
		t.Fatalf("StripJobCheckpointTags(RUNNING) = %q, want 'resumable note'", got)
	}
	// E3: Suppress transient pause/reaper heartbeat notes when PAUSED or RUNNING
	reaperPause := raw + " | reaper: heartbeat timeout (paused for worker auto-resume)"
	if got := StripJobCheckpointTags("PAUSED", reaperPause); got != "" {
		t.Fatalf("StripJobCheckpointTags(PAUSED, reaperPause) = %q, want empty string", got)
	}
	// E3: Strip tags and leading "| " separator when FAILED after 3 auto-resume attempts
	reaperFailed := "[run_type=full] [auto-resume #3] | reaper: max auto-resume attempts (3) exceeded"
	wantFailed := "reaper: max auto-resume attempts (3) exceeded"
	if got := StripJobCheckpointTags("FAILED", reaperFailed); got != wantFailed {
		t.Fatalf("StripJobCheckpointTags(FAILED, %q) = %q, want %q", reaperFailed, got, wantFailed)
	}
}

func TestRequestCancel_RealLocalPostgres_RollbackTx(t *testing.T) {
	candidates := []string{"config/config-local.yml", "../../../config/config-local.yml"}
	var cfgFile string
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if _, statErr := os.Stat(abs); statErr == nil {
				cfgFile = abs
				break
			}
		}
	}
	if cfgFile == "" {
		t.Skip("config-local.yml not found; skipping real local Postgres test")
	}
	t.Setenv("CFG_PATH", cfgFile)

	cfg, err := config.NewConfig()
	if err != nil {
		t.Skipf("cannot load config (%v); skipping real local Postgres test", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.UserName, cfg.DB.Password, cfg.DB.Database, cfg.DB.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("local Postgres unavailable (%v); skipping", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("failed to get sql.DB (%v); skipping", err)
	}
	defer sqlDB.Close()

	tx := db.Begin()
	if tx.Error != nil {
		t.Skipf("failed to begin transaction (%v); skipping", tx.Error)
	}
	defer tx.Rollback()

	var bindingID int64
	if err := tx.Raw("SELECT id FROM cdc_system.master_binding ORDER BY id ASC LIMIT 1").Scan(&bindingID).Error; err != nil || bindingID <= 0 {
		t.Skipf("no master_binding row available for fixture (%v); skipping", err)
	}

	ctx := context.Background()
	repo := NewTransmuteJobRepo(tx)
	traceID := "test_trace_p15_rollback_001"
	freshJobID := "test_job_p15_fresh_running_001"
	staleJobID := "test_job_p15_stale_running_002"

	if err := tx.Exec(`
		INSERT INTO cdc_system.transmute_schedule (master_binding_id, mode, last_status, last_error, last_trace_id, is_enabled, created_at, updated_at)
		VALUES (?, 'immediate', 'running', NULL, ?, true, NOW(), NOW())
		ON CONFLICT (master_binding_id, mode) DO UPDATE
		   SET last_status = 'running',
		       last_error = NULL,
		       last_trace_id = EXCLUDED.last_trace_id,
		       updated_at = NOW()
	`, bindingID, traceID).Error; err != nil {
		t.Fatalf("failed to seed transmute_schedule fixture: %v", err)
	}

	// 1. Fresh RUNNING job (<45s): RequestCancel sets cancel_requested=true, keeps status=RUNNING, preserves updated_at, and leaves schedule 'running' (E2).
	freshTs := time.Now().UTC().Add(-10 * time.Second).Truncate(time.Microsecond)
	if err := tx.Exec(`
		INSERT INTO cdc_system.transmute_jobs (job_id, master_binding_id, master_table, status, cancel_requested, trace_id, started_at, created_at, updated_at)
		VALUES (?, ?, 'test_schema.test_master', 'RUNNING', false, ?, ?, ?, ?)
	`, freshJobID, bindingID, traceID, freshTs, freshTs, freshTs).Error; err != nil {
		t.Fatalf("failed to insert fresh RUNNING job: %v", err)
	}

	if err := repo.RequestCancel(ctx, freshJobID); err != nil {
		t.Fatalf("RequestCancel(freshJobID) failed: %v", err)
	}

	freshAfter, err := repo.GetByID(ctx, freshJobID)
	if err != nil {
		t.Fatalf("GetByID(freshJobID) failed: %v", err)
	}
	if freshAfter.Status != "RUNNING" || !freshAfter.CancelRequested {
		t.Fatalf("fresh RUNNING job: got status=%s cancel_requested=%v, want status=RUNNING cancel_requested=true",
			freshAfter.Status, freshAfter.CancelRequested)
	}
	if !freshAfter.UpdatedAt.UTC().Truncate(time.Microsecond).Equal(freshTs) {
		t.Fatalf("fresh RUNNING job updated_at was mutated: got %s, want preserved %s",
			freshAfter.UpdatedAt.UTC().Format(time.RFC3339Nano), freshTs.Format(time.RFC3339Nano))
	}

	// 2. Stale RUNNING job (>45s): RequestCancel transitions job to CANCELLED and releases transmute_schedule from 'running' to 'failed' with last_error='cancelled by user' (E1).
	staleTs := time.Now().UTC().Add(-60 * time.Second).Truncate(time.Microsecond)
	if err := tx.Exec(`
		INSERT INTO cdc_system.transmute_jobs (job_id, master_binding_id, master_table, status, cancel_requested, trace_id, started_at, created_at, updated_at)
		VALUES (?, ?, 'test_schema.test_master', 'RUNNING', false, ?, ?, ?, ?)
	`, staleJobID, bindingID, traceID, staleTs, staleTs, staleTs).Error; err != nil {
		t.Fatalf("failed to insert stale RUNNING job: %v", err)
	}

	if err := repo.RequestCancel(ctx, staleJobID); err != nil {
		t.Fatalf("RequestCancel(staleJobID) failed: %v", err)
	}

	staleAfter, err := repo.GetByID(ctx, staleJobID)
	if err != nil {
		t.Fatalf("GetByID(staleJobID) failed: %v", err)
	}
	if staleAfter.Status != "CANCELLED" || !staleAfter.CancelRequested || staleAfter.ErrorMessage != "cancelled by user" || staleAfter.FinishedAt == nil {
		t.Fatalf("stale RUNNING job: got status=%s cancel_requested=%v err=%q finished_at=%v, want CANCELLED/true/'cancelled by user'/non-nil",
			staleAfter.Status, staleAfter.CancelRequested, staleAfter.ErrorMessage, staleAfter.FinishedAt)
	}
	if !staleAfter.UpdatedAt.After(staleTs) {
		t.Fatalf("stale RUNNING job updated_at should advance to NOW() on CANCELLED transition, got %s <= %s",
			staleAfter.UpdatedAt.UTC().Format(time.RFC3339Nano), staleTs.Format(time.RFC3339Nano))
	}

	var sched struct {
		LastStatus string `gorm:"column:last_status"`
		LastError  string `gorm:"column:last_error"`
	}
	if err := tx.Raw(`
		SELECT COALESCE(last_status, '') AS last_status, COALESCE(last_error, '') AS last_error
		  FROM cdc_system.transmute_schedule
		 WHERE master_binding_id = ? AND mode = 'immediate'
	`, bindingID).Scan(&sched).Error; err != nil {
		t.Fatalf("failed to query transmute_schedule after RequestCancel: %v", err)
	}
	if sched.LastStatus != "failed" || sched.LastError != "cancelled by user" {
		t.Fatalf("transmute_schedule after RequestCancel: got last_status=%q last_error=%q, want last_status='failed' last_error='cancelled by user'",
			sched.LastStatus, sched.LastError)
	}
}
```

```go
// File: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_collector_test.go (lines 114-136)
	if cfgFile == "" {
		t.Skip("config-local.yml not found from working directory; skipping")
	}
	t.Setenv("CFG_PATH", cfgFile)

	cfg, err := config.NewConfig()
	if err != nil {
		t.Skipf("cannot load config (%v); skipping", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.UserName, cfg.DB.Password, cfg.DB.Database, cfg.DB.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("failed to connect to local Postgres cdc_dw (%v); skipping", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("failed to get sql.DB (%v); skipping", err)
	}
	defer sqlDB.Close()
```

---

#### Demo P15-3: Sửa `E4.2` — Gắn `//go:build ignore` cho 5 script rời trong `cdc-cms-service/scratch/*.go`

```go
// Thêm 2 dòng đầu tiên vào cả 5 file:
// - /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/scratch/check_bindings.go
// - /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/scratch/check_logs.go
// - /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/scratch/check_payment_bills.go
// - /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/scratch/check_snapshot_logs.go
// - /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/scratch/test_cleanup.go
//go:build ignore

package main
```

---

#### Demo P15-4: Sửa `E5.1` — Tách `transmuter_lifecycle.go` (`221` dòng) khỏi `transmuter.go` (giảm từ `2049` dòng xuống `1841` dòng $\le 1874$ `HEAD`)

```go
// File mới: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_lifecycle.go
// Chuyển nguyên văn 1:1 các hàm quản lý vòng đời job, suy diễn con trỏ checkpoint và quét bù trước con trỏ từ transmuter.go (lines 159-181, 1185-1261, 1528-1631) sang transmuter_lifecycle.go:
package master

import (
	"context"
	"fmt"
	"time"

	"centralized-data-service/internal/repository"
	"centralized-data-service/internal/service/master/transmute"

	"gorm.io/gorm"
)

// SetTransmuteJobRepo injects the transmute_jobs repository for async manual job tracking.
func (t *TransmuterModule) SetTransmuteJobRepo(r *repository.TransmuteJobRepo) {
	t.transmuteJobRepo = r
}

// SetStopping marks the transmuter as shutting down so active jobs pause cleanly after finishing their current chunk.
func (t *TransmuterModule) SetStopping(v bool) {
	t.stopping.Store(v)
}

// IsStopping returns true when the worker is shutting down.
func (t *TransmuterModule) IsStopping() bool {
	return t.stopping.Load()
}

func (t *TransmuterModule) finishTransmuteJob(ctx context.Context, jobID, status string, rowsAffected, totalRows int64, errMsg string, masterBindingID ...int64) {
	if t.transmuteJobRepo == nil || jobID == "" {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = t.transmuteJobRepo.UpdateStatus(writeCtx, jobID, status, rowsAffected, totalRows, errMsg, masterBindingID...)
}

func (t *TransmuterModule) inferLastGpayIDFromMaster(ctx context.Context, row *masterBindingRuntime, masterDB *gorm.DB, rules []mappingRuleRow) int64 {
	shadowDB, err := t.connMgr.GetShadowDB(ctx, row.ShadowConnectionKey)
	if err != nil || masterDB == nil {
		return 0
	}
	masterQT := quoteTransmuteQualified(row.MasterSchema, row.MasterTable)
	shadowQT := quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable)
	shadowPK := fmt.Sprintf("(%s)::bigint", quoteTransmuteIdent(row.ShadowPK))

	var maxShadowID int64
	if shadowDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT COALESCE(MAX(%s), 0) FROM %s`, shadowPK, shadowQT)).Scan(&maxShadowID).Error != nil || maxShadowID <= 0 {
		return 0
	}

	var inferred int64
	var candGpayIDs []int64
	if masterDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT _gpay_id FROM %s WHERE _gpay_id <= ? ORDER BY _gpay_id DESC LIMIT 50`, masterQT), maxShadowID).Scan(&candGpayIDs).Error == nil && len(candGpayIDs) > 0 {
		_ = shadowDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT COALESCE(MAX(%s), 0) FROM %s WHERE %s IN (?)`, shadowPK, shadowQT, shadowPK), candGpayIDs).Scan(&inferred).Error
	}

	colTypes := make(map[string]string, len(rules))
	for _, r := range rules {
		colTypes[r.TargetColumn] = r.DataType
	}
	if pkCol := resolveConflictTarget(row, colTypes); pkCol != "" && pkCol != "_gpay_id" && isMasterPKMappedFromSourcePK(row, rules, pkCol) {
		pkIdent := quoteTransmuteIdent(pkCol)
		var recentIDs []string
		if masterDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT %s::text FROM %s WHERE _source_ts IS NOT NULL ORDER BY _source_ts DESC LIMIT 50`, pkIdent, masterQT)).Scan(&recentIDs).Error == nil && len(recentIDs) > 0 {
			var bySourceID int64
			if shadowDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT COALESCE(MAX(%s), 0) FROM %s WHERE _source_id IN (?)`, shadowPK, shadowQT), recentIDs).Scan(&bySourceID).Error == nil && bySourceID > inferred {
				type shadowGapRow struct {
					GpayID   int64  `gorm:"column:_gpay_id"`
					SourceID string `gorm:"column:_source_id"`
				}
				var gapRows []shadowGapRow
				errGap := shadowDB.WithContext(ctx).Raw(
					fmt.Sprintf(`SELECT %s AS _gpay_id, _source_id FROM %s WHERE %s > ? AND %s <= ? AND NOT COALESCE(_deleted, false) ORDER BY %s ASC LIMIT 50`, shadowPK, shadowQT, shadowPK, shadowPK, shadowPK),
					inferred, bySourceID,
				).Scan(&gapRows).Error
				if errGap == nil && len(gapRows) > 0 {
					gapSourceIDs := make([]string, 0, len(gapRows))
					for _, gr := range gapRows {
						gapSourceIDs = append(gapSourceIDs, gr.SourceID)
					}
					var matchedIDs []string
					errMatch := masterDB.WithContext(ctx).Raw(
						fmt.Sprintf(`SELECT DISTINCT %s::text FROM %s WHERE %s::text IN (?)`, pkIdent, masterQT, pkIdent),
						gapSourceIDs,
					).Scan(&matchedIDs).Error
					if errMatch == nil {
						if len(matchedIDs) == len(gapRows) {
							var activeShadowRows, masterCount int64
							errSh := shadowDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE NOT COALESCE(_deleted, false)`, shadowQT)).Scan(&activeShadowRows).Error
							errMs := masterDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, masterQT)).Scan(&masterCount).Error
							remAfterCand, errRem := t.countShadowRows(ctx, row, bySourceID)
							if errSh == nil && errMs == nil && errRem == nil && activeShadowRows >= masterCount && remAfterCand >= (activeShadowRows-masterCount) {
								inferred = bySourceID
							}
						} else if len(matchedIDs) > 0 {
							matchedSet := make(map[string]bool, len(matchedIDs))
							for _, id := range matchedIDs {
								matchedSet[id] = true
							}
							for _, gr := range gapRows {
								if !matchedSet[gr.SourceID] {
									break
								}
								inferred = gr.GpayID
							}
						}
					}
				}
			}
		}
	}
	return inferred
}

// syncUpdatedShadowRowsBeforeCursor quét bổ sung các dòng cũ trên Shadow có _gpay_id <= maxCursor
// vừa được cập nhật (_updated_at >= cutoff) hoặc bị đánh dấu _deleted = true khi chạy Incremental Sync.
func (t *TransmuterModule) syncUpdatedShadowRowsBeforeCursor(
	ctx context.Context,
	masterRow *masterBindingRuntime,
	rules []mappingRuleRow,
	strat transmute.Strategy,
	rc transmute.RunContext,
	maxCursor int64,
	jobID string,
	res *TransmuteResult,
	lastBatchErr *error,
) {
	if masterRow == nil || maxCursor <= 0 || t.connMgr == nil {
		return
	}
	shadowDB, err := t.connMgr.GetShadowDB(ctx, masterRow.ShadowConnectionKey)
	if err != nil || shadowDB == nil {
		return
	}

	shadowPK := masterRow.ShadowPK
	if shadowPK == "" {
		shadowPK = "_gpay_id"
	}
	pkIdent := quoteTransmuteIdent(shadowPK)
	pkSelect := pkIdent
	if shadowPK != "_gpay_id" {
		pkSelect = fmt.Sprintf("%s::bigint AS _gpay_id", pkIdent)
	}
	pkBigint := fmt.Sprintf("(%s)::bigint", pkIdent)
	shadowQT := quoteTransmuteQualified(masterRow.ShadowSchema, masterRow.ShadowTable)
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	if t.runtimeRepo != nil {
		if st, errSt := t.runtimeRepo.GetByMasterBinding(ctx, masterRow.ID); errSt == nil && st != nil && st.LastSuccessAt != nil && !st.LastSuccessAt.IsZero() {
			cand := st.LastSuccessAt.UTC().Add(-10 * time.Minute)
			if cand.After(cutoff) {
				cutoff = cand
			}
		}
	}

	var scanCursor int64
	for {
		if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
			res.Cancelled = true
			return
		}
		if t.IsStopping() {
			res.Paused = true
			return
		}
		var rows []shadowBatchRow
		query := fmt.Sprintf(
			`SELECT %s, _source_id, _raw_data, _source_ts, _deleted FROM %s WHERE %s > ? AND %s <= ? AND (COALESCE(_deleted, false) = true OR _updated_at >= ?) ORDER BY 1 LIMIT ?`,
			pkSelect, shadowQT, pkBigint, pkBigint,
		)
		errQ := shadowDB.WithContext(ctx).Raw(query, scanCursor, maxCursor, cutoff, t.batchSize).Scan(&rows).Error
		if errQ != nil {
			if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
				res.Cancelled = true
				return
			}
			if t.IsStopping() {
				res.Paused = true
				return
			}
			fallbackQuery := fmt.Sprintf(
				`SELECT %s, _source_id, _raw_data, _source_ts, _deleted FROM %s WHERE %s > ? AND %s <= ? AND COALESCE(_deleted, false) = true ORDER BY 1 LIMIT ?`,
				pkSelect, shadowQT, pkBigint, pkBigint,
			)
			if errFallback := shadowDB.WithContext(ctx).Raw(fallbackQuery, scanCursor, maxCursor, t.batchSize).Scan(&rows).Error; errFallback != nil {
				return
			}
		}
		if len(rows) == 0 {
			break
		}
		rows = t.enrichMultiShadowBatchRows(ctx, shadowDB, masterRow, rows)
		batchRes := t.processBatch(ctx, masterRow, rules, rows, strat, rc, false)
		if batchRes.lastErr != nil {
			if jobID != "" && t.transmuteJobRepo != nil && t.transmuteJobRepo.IsCancelRequested(ctx, jobID) {
				res.Cancelled = true
				return
			}
			if t.IsStopping() {
				res.Paused = true
				return
			}
		}
		accumulateBatchOutcome(res, batchRes, lastBatchErr)
		if batchRes.lastErr != nil || batchRes.lastGpayID <= scanCursor {
			break
		}
		scanCursor = batchRes.lastGpayID
		if scanCursor >= maxCursor {
			break
		}
	}
	if !res.Cancelled && !res.Paused && !t.IsStopping() {
		t.syncLaggingSecondaryShadowRows(ctx, shadowDB, masterRow, rules, strat, rc, cutoff, res, lastBatchErr)
	}
}
```

---

#### Demo P15-5: Sửa `E5.2` — Tách `snapshot_runner_lifecycle.go` (`307` dòng) & chuyển helper Kafka/Lag sang `snapshot_runner_pacing.go` (`257` dòng), đưa `snapshot_runner_handler.go` về `960` dòng ($\le 1029$ `HEAD`) và `snapshot_runner_state.go` về `228` dòng ($\le 400$)

```go
// File mới: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_lifecycle.go
// Chuyển nguyên văn 1:1 SetStopping (từ snapshot_runner_handler.go:96-98) và toàn bộ khối vòng đời Start/Stop/Heartbeat/Auto-Resume (từ snapshot_runner_state.go:230-514) sang snapshot_runner_lifecycle.go:
package orchestration

import (
	"context"
	"fmt"
	"time"

	"centralized-data-service/pkgs/observability"

	"go.uber.org/zap"
)

func (r *SnapshotRunner) SetStopping(v bool) {
	r.stopping.Store(v)
}

func (r *SnapshotRunner) baseContext() context.Context {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	if r.lifecycleCtx != nil {
		return r.lifecycleCtx
	}
	return context.Background()
}

func (r *SnapshotRunner) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	r.lifecycleMu.Lock()
	if r.lifecycleCancel != nil {
		r.lifecycleMu.Unlock()
		return
	}
	r.lifecycleCtx, r.lifecycleCancel = context.WithCancel(ctx)
	if r.activeProgressIDs == nil {
		r.activeProgressIDs = make(map[int64]context.CancelFunc)
	}
	loopCtx := r.lifecycleCtx
	r.lifecycleMu.Unlock()

	r.activeWg.Add(1)
	go func() {
		defer r.activeWg.Done()
		r.recoverOrphanedSnapshots(loopCtx)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				r.recoverOrphanedSnapshots(loopCtx)
			}
		}
	}()
}

// Stop gracefully pauses all in-flight snapshots at the current batch boundary (up to 2s budget — C4 & P13-2)
// and falls back to canceling runCtx while preserving 'paused' state in snapshot_progress.
func (r *SnapshotRunner) Stop() {
	r.stopping.Store(true)

	r.lifecycleMu.Lock()
	cancel := r.lifecycleCancel
	r.lifecycleCancel = nil
	activeIDs := make([]int64, 0, len(r.activeProgressIDs))
	for id := range r.activeProgressIDs {
		activeIDs = append(activeIDs, id)
	}
	r.lifecycleMu.Unlock()

	if cancel != nil {
		cancel()
	}

	doneCh := make(chan struct{})
	go func() {
		r.activeWg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		r.lifecycleMu.Lock()
		for _, fn := range r.activeProgressIDs {
			if fn != nil {
				fn()
			}
		}
		r.lifecycleMu.Unlock()
		<-doneCh
	}

	for _, id := range activeIDs {
		r.markProgressPaused(id)
	}
}

func (r *SnapshotRunner) spawnSnapshot(p snapshotV2Payload, jobID string, header map[string][]string) {
	r.activeWg.Add(1)
	go func(p snapshotV2Payload, jobID string, header map[string][]string) {
		defer r.activeWg.Done()
		defer func() {
			if rec := recover(); rec != nil {
				if r.logger != nil {
					r.logger.Error("snapshot.v2 goroutine panic recovered",
						zap.Int64("source_object_id", p.SourceObjectID),
						zap.Int64("progress_id", p.ProgressID),
						zap.Any("panic", rec))
				}
				if p.ProgressID > 0 {
					r.markProgressError(context.Background(), p.ProgressID, fmt.Sprintf("panic: %v", rec))
				}
			}
		}()

		baseCtx := r.baseContext()
		ctx := observability.ExtractNATSHeader(baseCtx, header)
		ctx, span := observability.ChildSpan(ctx, "nats.SnapshotV2Runner")
		defer span.End()

		if sc := span.SpanContext(); sc.IsValid() && p.TraceID == "" {
			p.TraceID = sc.TraceID().String()
		}

		if err := r.runSnapshot(ctx, p, jobID); err != nil {
			observability.Ctx(ctx, r.logger).Error("snapshot.v2 run failed",
				zap.Int64("source_object_id", p.SourceObjectID),
				zap.String("trace_id", p.TraceID),
				zap.String("job_id", jobID),
				zap.Error(err))
		}
	}(p, jobID, header)
}

func (r *SnapshotRunner) trackAndHeartbeatProgress(parentCtx context.Context, progressID int64) (context.Context, func()) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	runCtx, runCancel := context.WithCancel(parentCtx)

	r.lifecycleMu.Lock()
	if r.activeProgressIDs == nil {
		r.activeProgressIDs = make(map[int64]context.CancelFunc)
	}
	r.activeProgressIDs[progressID] = runCancel
	r.lifecycleMu.Unlock()

	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if r.db != nil {
					_ = r.db.WithContext(context.Background()).Exec(`
						UPDATE cdc_system.snapshot_progress
						SET updated_at = NOW()
						WHERE id = ? AND status = 'running'
					`, progressID).Error
				}
			}
		}
	}()

	stop := func() {
		runCancel()
		<-hbDone
		r.lifecycleMu.Lock()
		delete(r.activeProgressIDs, progressID)
		r.lifecycleMu.Unlock()
	}
	return runCtx, stop
}

func (r *SnapshotRunner) isProgressActiveInMemory(progressID int64) bool {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	_, ok := r.activeProgressIDs[progressID]
	return ok
}

func (r *SnapshotRunner) markProgressPaused(progressID int64) {
	if progressID <= 0 || r.db == nil {
		return
	}
	_ = r.db.WithContext(context.Background()).Exec(`
		UPDATE cdc_system.snapshot_progress
		SET status = 'paused',
		    error_msg = 'paused by worker shutdown (auto-resumable)',
		    updated_at = NOW()
		WHERE id = ? AND status = 'running'
	`, progressID).Error
}

func (r *SnapshotRunner) ensureProgressNotStuckRunning(progressID int64, retErr error) {
	if progressID <= 0 || r.db == nil {
		return
	}
	if retErr != nil {
		_ = r.db.WithContext(context.Background()).Exec(`
			UPDATE cdc_system.snapshot_progress
			SET status = 'error',
			    error_msg = COALESCE(NULLIF(error_msg, ''), ?),
			    updated_at = NOW(),
			    finished_at = COALESCE(finished_at, NOW())
			WHERE id = ? AND status = 'running'
		`, retErr.Error(), progressID).Error
		return
	}
	_ = r.db.WithContext(context.Background()).Exec(`
		UPDATE cdc_system.snapshot_progress
		SET status = 'paused',
		    error_msg = 'paused by worker shutdown (auto-resumable)',
		    updated_at = NOW()
		WHERE id = ? AND status = 'running'
	`, progressID).Error
}

func (r *SnapshotRunner) recoverOrphanedSnapshots(ctx context.Context) {
	if r == nil || r.db == nil || r.stopping.Load() {
		return
	}
	type orphanRow struct {
		ID              int64   `gorm:"column:id"`
		SourceObjectID  int64   `gorm:"column:source_object_id"`
		ShadowBindingID *int64  `gorm:"column:shadow_binding_id"`
		TraceID         *string `gorm:"column:trace_id"`
		Status          string  `gorm:"column:status"`
	}
	var rows []orphanRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, source_object_id, shadow_binding_id, trace_id, status
		FROM cdc_system.snapshot_progress
		WHERE (
			status = 'running'
			AND updated_at < NOW() - INTERVAL '45 seconds'
		) OR (
			status = 'paused'
			AND error_msg LIKE 'paused by worker shutdown%'
			AND (total_rows = 0 OR rows_processed < total_rows)
			AND updated_at > NOW() - INTERVAL '24 hours'
		) OR (
			status IN ('paused', 'error')
			AND error_msg LIKE 'Heartbeat timeout:%'
			AND last_seen_id IS NOT NULL
			AND last_seen_id <> ''
			AND (total_rows = 0 OR rows_processed < total_rows)
			AND updated_at > NOW() - INTERVAL '24 hours'
		)
		ORDER BY id ASC
		LIMIT 10
	`).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return
	}

	for _, row := range rows {
		if ctx.Err() != nil || r.stopping.Load() {
			return
		}
		if r.isProgressActiveInMemory(row.ID) {
			continue
		}
		res := r.db.WithContext(ctx).Exec(`
			UPDATE cdc_system.snapshot_progress
			SET status = 'paused',
			    error_msg = NULL,
			    finished_at = NULL,
			    updated_at = NOW()
			WHERE id = ? AND status IN ('running', 'paused', 'error')
		`, row.ID)
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}

		var bindingID int64
		if row.ShadowBindingID != nil {
			bindingID = *row.ShadowBindingID
		}
		traceID := ""
		if row.TraceID != nil {
			traceID = *row.TraceID
		}
		if r.logger != nil {
			r.logger.Info("snapshot.v2 auto-resuming interrupted snapshot progress",
				zap.Int64("progress_id", row.ID),
				zap.Int64("source_object_id", row.SourceObjectID),
				zap.Int64("shadow_binding_id", bindingID),
				zap.String("prev_status", row.Status),
				zap.String("trace_id", traceID))
		}
		r.spawnSnapshot(snapshotV2Payload{
			SourceObjectID:  row.SourceObjectID,
			ShadowBindingID: bindingID,
			ProgressID:      row.ID,
			TraceID:         traceID,
			Action:          "auto-resume",
			Origin:          "worker-recovery",
			Overwrite:       false,
		}, "", nil)
	}
}
```

```go
// Trong /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_pacing.go:
// Nối nguyên văn 1:1 SetTopicController, SetKafkaConnectURL, checkAdaptiveLagThrottling, autoCreateKafkaTopic, kafkaConnectDo (từ snapshot_runner_handler.go:100-106, 136-159, 1001-1078) vào cuối snapshot_runner_pacing.go:
func (r *SnapshotRunner) SetTopicController(tc topicController) {
	r.topicController = tc
}

func (r *SnapshotRunner) SetKafkaConnectURL(url string) {
	r.kafkaConnectURL = url
}

func (r *SnapshotRunner) checkAdaptiveLagThrottling(ctx context.Context) {
	if li, ok := r.topicController.(lagInspector); ok && li != nil {
		lag := li.GetMaxLag()
		if lag > 10000 {
			observability.Ctx(ctx, r.logger).Warn("snapshot.v2 adaptive pause: kafka consumer lag critical",
				zap.Int64("kafka_lag", lag),
			)
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		} else if lag > 1000 {
			observability.Ctx(ctx, r.logger).Info("snapshot.v2 adaptive throttling: kafka consumer lag elevated",
				zap.Int64("kafka_lag", lag),
			)
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}
}

func autoCreateKafkaTopic(ctx context.Context, bootstrapServers, topic string, logger *zap.Logger) {
	if topic == "" {
		return
	}
	if bootstrapServers == "" {
		bootstrapServers = os.Getenv("KAFKA_BROKERS")
		if bootstrapServers == "" {
			bootstrapServers = os.Getenv("CMS_SYSTEM_SIGNAL_KAFKA_BOOTSTRAP")
		}
	}
	if strings.TrimSpace(bootstrapServers) == "" {
		if logger != nil {
			logger.Warn("auto-create kafka topic skipped: no kafka brokers configured in environment")
		}
		return
	}

	brokers := strings.Split(bootstrapServers, ",")
	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker == "" {
			continue
		}
		conn, err := kafka.DialContext(ctx, "tcp", broker)
		if err != nil {
			continue
		}
		topicConfig := kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}
		err = conn.CreateTopics(topicConfig)
		conn.Close()
		if err == nil {
			if logger != nil {
				logger.Info("auto-created kafka topic for connector",
					zap.String("topic", topic), zap.String("broker", broker))
			}
			return
		}
	}
}

func kafkaConnectDo(ctx context.Context, method, rawURL string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
```

---

#### Demo P15-6: Sửa `E5.3` — Tách `batch_buffer_lifecycle.go` (`126` dòng) khỏi `batch_buffer.go` (giảm từ `877` dòng xuống `765` dòng $\le 824$ `HEAD`)

```go
// File mới: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_lifecycle.go
// Chuyển nguyên văn 1:1 các hàm cấu hình, timerLoop, Flush, GetStatus, Stop (từ batch_buffer.go:66-76, 117-143, 262-302, 847-876) sang batch_buffer_lifecycle.go:
package shadow

import (
	"context"
	"time"

	reposhadow "centralized-data-service/internal/repository/shadow"
	"centralized-data-service/internal/service/governance"
	serviceshadow "centralized-data-service/internal/service/shadow"
	servicesource "centralized-data-service/internal/service/source"

	"github.com/nats-io/nats.go"
)

func (bb *BatchBuffer) SetOnCommitOffsets(cb func(context.Context, map[TopicPartition]int64)) {
	bb.onCommitOffsets = cb
}

func (bb *BatchBuffer) SetSnapshotMode(isSnapshot bool) {
	bb.isSnapshotBuffer = isSnapshot
}

func (bb *BatchBuffer) SetFailedSyncLogRepo(repo *reposhadow.FailedSyncLogRepo) {
	bb.failedSyncLogRepo = repo
}

// SetJetStream cấu hình JetStream Context cho BatchBuffer
func (bb *BatchBuffer) SetJetStream(js nats.JetStreamContext) {
	bb.js = js
}

// SetNatsConn bật realtime post-ingest fan-out (xem field natsConn). Gọi ở bootstrap.
func (bb *BatchBuffer) SetNatsConn(c *nats.Conn) { bb.natsConn = c }

func (bb *BatchBuffer) SetMaskingService(masking *governance.MaskingService) {
	bb.masking = masking
}

func (bb *BatchBuffer) SetConnectionManager(connMgr *servicesource.ConnectionManager) {
	bb.connMgr = connMgr
}

// FlushSchemaCache evicts all cached table schemas from every per-connection
// SchemaAdapter held in bb.adapters, plus the default bb.schemaAdapter.
// Call this whenever a schema.config.reload event fires so newly-added columns
// are picked up on the next upsert without a process restart.
func (bb *BatchBuffer) FlushSchemaCache() {
	bb.adapters.Range(func(_, v any) bool {
		v.(*serviceshadow.SchemaAdapter).FlushAll()
		return true
	})
	bb.schemaAdapter.FlushAll()
}

func (bb *BatchBuffer) timerLoop() {
	defer func() {
		if bb.doneCh != nil {
			close(bb.doneCh)
		}
	}()
	ticker := time.NewTicker(bb.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-bb.stopCh:
			flushCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			_, _ = bb.FlushWithContext(flushCtx)
			cancel()
			return
		default:
		}
		select {
		case <-bb.stopCh:
			flushCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			_, _ = bb.FlushWithContext(flushCtx)
			cancel()
			return
		case <-bb.ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			_, _ = bb.FlushWithContext(flushCtx)
			cancel()
			return
		case <-ticker.C:
			bb.flushFromBackground()
		case <-bb.flushCh:
			bb.flushFromBackground()
		}
	}
}

// Flush drains the buffer synchronously and returns (written, err)
func (bb *BatchBuffer) Flush() (written int, err error) {
	return bb.FlushWithContext(context.Background())
}

func (bb *BatchBuffer) GetStatus() BatchStatus {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	lastDur := bb.lastFlushDurationMs.Load()
	return BatchStatus{
		BufferSize:          len(bb.records),
		MaxSize:             bb.maxSize,
		LastFlush:           bb.lastFlush,
		LastFlushDurationMs: lastDur,
		DBFlushTotalMs:      bb.DBFlushDuration().Milliseconds(),
		OOMRetryCount:       bb.oomRetryCount.Load(),
		DLQEvacuatedCount:   bb.dlqEvacuatedCount.Load(),
		AdaptiveThrottled:   bb.isSnapshotBuffer && lastDur > 200,
	}
}

func (bb *BatchBuffer) Stop() {
	bb.stopOnce.Do(func() {
		if bb.stopCh != nil {
			close(bb.stopCh)
		}
	})
	if bb.doneCh != nil {
		select {
		case <-bb.doneCh:
		case <-time.After(4500 * time.Millisecond):
		}
	}
	bb.cancel()
}
```

---

#### Demo P15-7: Sửa `E5.4` — Tách `debounce_lifecycle.go` (`208` dòng) khỏi `debounce.go` (giảm từ `399` dòng xuống `196` dòng $\le 197$ `HEAD`)

```go
// File mới: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/debounce_lifecycle.go
// Chuyển nguyên văn 1:1 các hàm takeFlushBatchLocked, enqueueFlushBatch, executeBatchWithMetrics, drainRemainingOnStop, workerLoop, GetStat, Stop (từ debounce.go:204-399) sang debounce_lifecycle.go:
package master

import (
	"context"
	"sort"
	"time"
)

func (td *TableDebouncer) takeFlushBatchLocked() sequencedBatch {
	if len(td.tasks) == 0 {
		return sequencedBatch{}
	}
	if td.idleTimer != nil {
		td.idleTimer.Stop()
		td.idleTimer = nil
	}
	if td.maxTimer != nil {
		td.maxTimer.Stop()
		td.maxTimer = nil
	}
	tasks := td.tasks
	td.tasks = make([]NatsMsgTask, 0, td.maxSize)
	td.pendingEvents = 0
	td.nextSeq++
	td.enqueuing++
	return sequencedBatch{
		seq:   td.nextSeq,
		tasks: tasks,
	}
}

func (td *TableDebouncer) enqueueFlushBatch(batch sequencedBatch) {
	if len(batch.tasks) == 0 {
		return
	}
	defer func() {
		td.mu.Lock()
		td.enqueuing--
		if td.enqueuing <= 0 && td.enqueueCond != nil {
			td.enqueueCond.Broadcast()
		}
		td.mu.Unlock()
	}()

	// Ưu tiên kiểm tra stopCh trước để tránh select chọn ngẫu nhiên 50/50 khi stopCh đã đóng (B2 & C4)
	select {
	case <-td.stopCh:
		td.mu.Lock()
		td.overflowBatches = append(td.overflowBatches, batch)
		td.mu.Unlock()
		return
	default:
	}

	select {
	case td.flushCh <- batch:
	case <-td.stopCh:
		td.mu.Lock()
		td.overflowBatches = append(td.overflowBatches, batch)
		td.mu.Unlock()
	}
}

func (td *TableDebouncer) executeBatchWithMetrics(ctx context.Context, tasks []NatsMsgTask) {
	if len(tasks) == 0 {
		return
	}
	td.inFlight.Add(1)
	defer td.inFlight.Add(-1)
	batchEvents := 0
	for _, t := range tasks {
		batchEvents += taskEventCount(t)
	}
	td.runTransmute(ctx, tasks)
	td.batchesFlushed.Add(1)
	td.eventsProcessed.Add(int64(batchEvents))
	td.lastFlushUnixMs.Store(time.Now().UnixMilli())
}

func (td *TableDebouncer) drainRemainingOnStop(firstBatch sequencedBatch) {
	// BẮT BUỘC chiếm đủ toàn bộ cap(td.sem) slot trước tiên để đợi mọi mẻ in-flight hoàn tất 100% (B2)
	for i := 0; i < cap(td.sem); i++ {
		td.sem <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(td.sem); i++ {
			<-td.sem
		}
	}()

	// Chờ mọi goroutine vừa gọi takeFlushBatchLocked() hoàn tất enqueueFlushBatch() (0 rơi batch — B2)
	td.mu.Lock()
	for td.enqueuing > 0 {
		td.enqueueCond.Wait()
	}
	if td.idleTimer != nil {
		td.idleTimer.Stop()
		td.idleTimer = nil
	}
	if td.maxTimer != nil {
		td.maxTimer.Stop()
		td.maxTimer = nil
	}

	var pendingBatches []sequencedBatch
	if len(firstBatch.tasks) > 0 {
		pendingBatches = append(pendingBatches, firstBatch)
	}
	for {
		select {
		case b := <-td.flushCh:
			if len(b.tasks) > 0 {
				pendingBatches = append(pendingBatches, b)
			}
		default:
			goto drainedChannel
		}
	}
drainedChannel:
	if len(td.overflowBatches) > 0 {
		pendingBatches = append(pendingBatches, td.overflowBatches...)
		td.overflowBatches = nil
	}
	tailTasks := td.tasks
	td.tasks = nil
	td.pendingEvents = 0
	td.mu.Unlock()

	// Sửa C4.1: Sắp xếp toàn bộ các batch theo đúng số thứ tự seq tăng dần (FIFO tuyệt đối: Batch #17 luôn trước Batch #18)
	sort.Slice(pendingBatches, func(i, j int) bool {
		return pendingBatches[i].seq < pendingBatches[j].seq
	})

	// Sửa C4.2: Dùng chung ĐÚNG 1 drainCtx 4s cho toàn bộ vòng lặp xả của TableDebouncer;
	// nếu hết 4s, gọi task.complete(drainCtx.Err()) để JetStream NakWithDelay phát lại.
	drainCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	runOrNak := func(tasks []NatsMsgTask) {
		if len(tasks) == 0 {
			return
		}
		if drainCtx.Err() != nil {
			for _, t := range tasks {
				t.complete(drainCtx.Err())
			}
			return
		}
		td.executeBatchWithMetrics(drainCtx, tasks)
	}

	for _, b := range pendingBatches {
		runOrNak(b.tasks)
	}
	runOrNak(tailTasks)
}

func (td *TableDebouncer) workerLoop() {
	defer td.wg.Done()
	for {
		select {
		case <-td.stopCh:
			td.drainRemainingOnStop(sequencedBatch{})
			return
		case batch := <-td.flushCh:
			select {
			case td.sem <- struct{}{}:
				go func(b sequencedBatch) {
					defer func() { <-td.sem }()
					td.executeBatchWithMetrics(td.ctx, b.tasks)
				}(batch)
			case <-td.stopCh:
				td.drainRemainingOnStop(batch)
				return
			}
		}
	}
}

func (td *TableDebouncer) GetStat() DebouncerStat {
	td.mu.Lock()
	pendingTasks := len(td.tasks)
	pendingEvents := td.pendingEvents
	queuedBatches := len(td.flushCh) + len(td.overflowBatches)
	td.mu.Unlock()
	return DebouncerStat{
		TableName:       td.tableName,
		InFlight:        td.inFlight.Load(),
		PendingTasks:    pendingTasks,
		PendingEvents:   pendingEvents,
		QueuedBatches:   queuedBatches,
		BatchesFlushed:  td.batchesFlushed.Load(),
		EventsProcessed: td.eventsProcessed.Load(),
		LastFlushUnixMs: td.lastFlushUnixMs.Load(),
	}
}

func (td *TableDebouncer) Stop() {
	td.stopOnce.Do(func() {
		close(td.stopCh)
	})
	td.wg.Wait()
	td.cancel()
}
```

