# Báo Cáo Kiểm Định & Phản Biện Cuối Cùng (S5 Audit Report — `missing_core_stream`)

- **Workspace:** `nats-monitor-missing-core-stream-2026-10-23`
- **Tier:** `T2`
- **Suffix:** `missing_core_stream`
- **Restore-Point Git Commit SHAs (Local, chưa `git push`):**
  - `centralized-data-service`: `778dbadf5ade9166daebf2b139ee7130ccdeea13`
  - `cdc-cms-service`: `818064d137a324ec893f891486d2b1945eefe917`
- **Biên lai thực thi hiện hành (`06_evidence/`, khớp 100% `src_fingerprint` sau vòng lặp tự chữa lành `S5 → S3 → S4 → S5`):**
  - `EV-20261023T115458-6a9e` (`integration` — `centralized-data-service`: `go vet` & `go test -race -count=1 -v` trên `./pkgs/natsconn/... ./internal/handler/master/... ./internal/handler/shadow/...` đạt `PASS`).
  - `EV-20261023T115543-bbdf` (`integration` — `cdc-cms-service`: `go vet` & `go test -race -count=1 -v` trên `./pkgs/natsconn/... ./internal/api/system/...` kết nối trực tiếp NATS thật đạt `PASS`).
  - `EV-20261023T115558-b5f3` (`runtime` — truy vấn trực tiếp NATS JetStream `http://localhost:18222/jsz?streams=true&config=true` xác nhận đủ 4 Core Streams đang hoạt động đạt `PASS`).

---

## Mục 1: Phản biện từng dòng update (`git diff` Line-by-Line Critique) & Từng Task

Trong quá trình kiểm định phản biện S5, em đã tự soi lại toàn bộ các task (`T1` → `T6`) và từng dòng `git diff` trên cả 2 repository, phát hiện **6 lỗ hổng kỹ thuật & điểm suy diễn từ bản triển khai đầu tiên**, và đã kích hoạt vòng lặp tự chữa lành khép kín (`S5 → S3 → S4 → S5`) để xử lý triệt để:

### 1.1. `centralized-data-service/pkgs/natsconn/nats_client.go` (Task T4.1 & T6.2)
- **Line 16–23 (`struct NatsClient`):** Bổ sung `logger *zap.Logger`, `mu sync.RWMutex`, `cdcSubjects []string`, và `reconciling atomic.Bool`. Không chứa bất kỳ hàm closure giả lập (`*Fn`) nào.
- **Line 26–81 (`Default*StreamConfig`):** Export 4 hàm khởi tạo cấu hình chuẩn (`DefaultCDCEventsStreamConfig`, `DefaultSchemaDriftStreamConfig`, `DefaultSchemaConfigStreamConfig`, `DefaultTransmuteStreamConfig`) làm nguồn sự thật duy nhất (`Single Source of Truth`).
- **Line 83–140 (`NewNatsClient`) — Phát hiện & Tự sửa #3 (Lock Synchronization trên `client.Conn` / `client.JS`):**
  - *Phản biện:* Ở bản sửa đầu tiên, dù `client := &NatsClient{logger: logger}` đã được khởi tạo trước `nats.Connect(...)`, việc gán `client.Conn = nc` và `client.JS = js` ở dòng 133–135 diễn ra **không có khóa `client.mu.Lock()`**, trong khi `ReconnectHandler` có thể chạy bất đồng bộ trên goroutine của `nats.go` và gọi `EnsureStreams` đọc `c.JS`.
  - *Khắc phục:* Bọc `client.Conn = nc; client.JS = js` trong `client.mu.Lock() / client.mu.Unlock()` (dòng 133–136) và đọc `c.mu.RLock(); js := c.JS; c.mu.RUnlock()` ở đầu `ensureSingleStream` (dòng 179–181).
- **Line 153–267 (`EnsureStreams` & `ensureSingleStream`):** Tách xử lý từng stream độc lập và gom lỗi bằng `errors.Join(errs...)` thay vì `return` sớm ở `CDC_EVENTS`. Khi `UpdateStream` gặp lỗi (ví dụ xung đột subject hoặc thông số), ghi log `WARN` chi tiết đối chiếu các trường `existing_*` và `desired_*` để truy vết nguyên nhân gốc mà không chặn 3 stream còn lại.

### 1.2. `centralized-data-service/internal/handler/master/transmute_consumer_pool.go` (Task T4.2 & T6.1)
- **Line 54–66 (`TransmuteConsumerPoolConfig` & `TransmuteConsumerPool`):** Thêm `WatchdogInterval time.Duration` (mặc định `15s`) và 2 trường theo dõi định danh consumer đang gắn kết: `boundDeliverSubject string` và `boundConsumerCreated time.Time`.
- **Line 123–146 (`unsubscribeSubsLocked` & `ensureStreamExistsLocked`):** Dùng `_ = sub.Unsubscribe()` khi tự hồi phục giữa chừng để giải phóng ngay goroutine nội bộ của `nats.go`; giữ `sub.Drain()` cho `Stop()` khi đóng tiến trình. Nếu `CDC_TRANSMUTE` bị xóa (`ErrStreamNotFound`), tự động gọi `p.js.AddStream(natsconn.DefaultTransmuteStreamConfig())`.
- **Line 148–240 (`ensureSubscribedLocked`) — Phát hiện & Tự sửa #2 (Chống tự hủy `p.subs` khi mạng chập chờn):**
  - *Phản biện:* Ở bản sửa đầu tiên, `ensureSubscribedLocked()` gọi `p.unsubscribeSubsLocked()` **ngay đầu hàm** trước khi gọi `ensureStreamExistsLocked()` và `ConsumerInfo()`. Nếu một lỗi mạng thoáng qua kích hoạt `ensureSubscribedLocked()`, toàn bộ `p.subs` đang sống sẽ bị hủy trước khi biết chắc NATS server có phản hồi hay không!
  - *Khắc phục:* Đưa `ensureStreamExistsLocked()` và kiểm tra/khởi tạo consumer trên NATS lên trước (dòng 156–206); chỉ khi NATS xác nhận stream và consumer sẵn sàng mới gọi `p.unsubscribeSubsLocked()` (dòng 210) và tạo bộ `WorkerCount` subscriptions mới (dòng 213–223), đồng thời lưu lại `p.boundDeliverSubject = info.Config.DeliverSubject` và `p.boundConsumerCreated = info.Created` (dòng 229–232).
- **Line 242–311 (`watchdogLoop`) — Phát hiện & Tự sửa #1 (Zombie `_INBOX` khi Multi-Pod hoặc `EnsureStreams` tạo lại Consumer):**
  - *Phản biện:* Push subscription của `nats.go` thực chất là một Core NATS subscription lắng nghe trên một subject nội bộ `_INBOX.*` (`DeliverSubject` của consumer). Khi stream `CDC_TRANSMUTE` bị xóa trên NATS server, `sub.IsValid()` phía client **vẫn trả về `true`**. Nếu một Pod khác (hoặc `EnsureStreams` + Pod khác) kịp tạo lại stream `CDC_TRANSMUTE` và consumer `transmute-shadow-consumer-v2` với một `DeliverSubject` mới (`_INBOX.new`) trước khi `watchdogLoop` của Pod hiện tại chạy tick tiếp theo, lời gọi `js.ConsumerInfo` sẽ trả về `err == nil`! Bản code đầu tiên chỉ kiểm tra `if _, err := js.ConsumerInfo(...); err != nil`, dẫn tới Pod hiện tại không biết consumer đã đổi sang `_INBOX.new` và các worker của Pod đó sẽ lắng nghe mãi mãi trên `_INBOX.old` (zombie subscriptions)!
  - *Khắc phục:* Trong `watchdogLoop` (dòng 280–292), sau khi gọi `js.ConsumerInfo` bên ngoài `p.mu.Lock()` với `nats.MaxWait(2*time.Second)`:
    1. Nếu `err != nil`: khi các `p.subs` cục bộ vẫn hợp lệ, chỉ bật `needsHeal = true` nếu lỗi là `nats.ErrStreamNotFound` hoặc `nats.ErrConsumerNotFound` (không phá hủy kết nối khi chỉ bị timeout mạng thoáng qua).
    2. Nếu `err == nil && info != nil`: kiểm tra `!info.PushBound || info.Config.DeliverGroup != queueGroup || (boundDeliverSubject != "" && info.Config.DeliverSubject != boundDeliverSubject) || (!boundConsumerCreated.IsZero() && !info.Created.Equal(boundConsumerCreated))`. Bất kỳ sự thay đổi nào của `DeliverSubject`, `Created`, hoặc mất trạng thái `PushBound` đều lập tức kích hoạt `needsHeal = true` để re-bind toàn bộ workers sang consumer mới (`EV-20261023T115458-6a9e` `PASS`).

### 1.3. `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go` (Task T4.3 & T6.3)
- **Line 106–122 (`publishTransmuteTrigger`) — Phát hiện & Tự sửa #4 (`nats.RetryAttempts(0)` chống ngủ ngầm `500ms` bên trong `nats.go`):**
  - *Phản biện (Bắt quả tang suy diễn bằng Test thật):* Ban đầu em nhận định rằng chỉ cần thêm `break` khi gặp `ErrNoStreamResponse` / `ErrNoResponders` / `ErrStreamNotFound` trong vòng lặp `for attempt := 1; attempt <= 3` là thời gian fallback sẽ giảm từ `300ms` xuống `~1ms`. Tuy nhiên, khi viết test tích hợp trực tiếp `TestPublishTransmuteTrigger_FastFallbackWhenStreamMissing` bắn vào NATS JetStream thật (khi không có stream `CDC_TRANSMUTE`), phép đo thời gian thực tế báo **FAIL ở mức `503.63ms`**!
  - *Nguyên nhân gốc:* Bên trong thư viện `github.com/nats-io/nats.go`, hàm `js.PublishMsg(msg)` mặc định có `DefaultPubRetryAttempts = 2` và `DefaultPubRetryWait = 250ms` khi gặp `ErrNoResponders` (`503 No Responders`). Nghĩa là trước khi sửa, mỗi lần gọi `publishTransmuteTrigger` khi mất stream thực chất bị treo `3 * 500ms + 300ms = 1800ms` (`1.8s`), và nếu chỉ thêm `break` ở vòng lặp ngoài mà không truyền `nats.RetryAttempts(0)`, mỗi lần gọi vẫn tốn `2 * 250ms = 500ms`!
  - *Khắc phục:* Truyền `nats.RetryAttempts(0)` vào `bb.js.PublishMsg(msg, nats.RetryAttempts(0))` (dòng 109) vì vòng lặp bên ngoài đã chủ động điều khiển retry cho các lỗi mạng tạm thời và thoát ngay khi gặp `ErrNoStreamResponse` / `ErrNoResponders` / `ErrStreamNotFound`. Sau khi sửa, `TestPublishTransmuteTrigger_FastFallbackWhenStreamMissing` hoàn tất toàn bộ trong `< 50ms` (`0.04s` bao gồm cả dựng server, chứng minh qua biên lai `EV-20261023T115458-6a9e` `PASS`).

### 1.4. `cdc-cms-service/pkgs/natsconn/nats_client.go`, `internal/server/server.go`, `internal/api/system/nats_monitor_handler.go` (Task T4.4 & T6.2/T6.3)
- **`cdc-cms-service/pkgs/natsconn/nats_client.go:16-267`:** Bổ sung `mu sync.RWMutex` đồng bộ hóa `client.Conn` / `client.JS`, 4 hàm `Default*StreamConfig`, `ReconnectHandler` bất đồng bộ có cờ `reconciling atomic.Bool`, và `EnsureStreams()` bảo toàn `Subjects` hiện có của `CDC_EVENTS` (`preserveExistingSubjects = true`) đồng thời tạo/đồng bộ đủ 4 Core Streams (`CDC_EVENTS`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`, `CDC_TRANSMUTE`).
- **`cdc-cms-service/internal/server/server.go:124-126`:** Gọi `natsClient.EnsureStreams()` ngay khi `cdc-cms-service` khởi động.
- **`cdc-cms-service/internal/api/system/nats_monitor_handler.go:51-86, 189-197, 233-273`:** Ánh xạ `canonicalCoreStreams` từ `natsconn.Default*StreamConfig()`. Trong `GetOverview`, khi phát hiện thiếu Core Stream, `tryAutoHealMissingCoreStreams` kiểm tra khóa thời gian hồi `30s` (`lastHealAttemptUnix.CompareAndSwap`) trước khi gọi `h.natsClient.EnsureStreams()` và lấy lại `/jsz` + `/connz` mới (`EV-20261023T115543-bbdf` `PASS`).

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

- **Không dùng Mockup / In-Memory Fake:** Toàn bộ mã nguồn sản phẩm tuyệt đối không chứa hàm closure giả lập (`*Fn`). Mọi kiểm thử đều chạy trên tiến trình `nats-server` JetStream thật (cả embedded `natsserver.NewServer` với `JetStream: true` lẫn container NATS thực tế tại `localhost:14222` / `http://localhost:18222`), chứng minh qua biên lai `EV-20261023T115458-6a9e` (`PASS`) và `EV-20261023T115543-bbdf` (`PASS`).
- **Trace Runtime thật trên NATS JetStream Local (`localhost:18222`):** Truy vấn trực tiếp endpoint `/jsz?streams=true&config=true` xác nhận cả 4 Core Streams đang hoạt động (`runtime` `PASS` theo biên lai `EV-20261023T115558-b5f3`):
```text
LIVE_STREAMS: ['CDC_EVENTS', 'CDC_TRANSMUTE', 'SCHEMA_CONFIG', 'SCHEMA_DRIFT']
```
- **Minh bạch về Môi trường Production (`https://cdc-cms.goopay.vn/nats-monitor`) — Cần phối hợp với anh để Trace Runtime Value thật trên PROD:**
  - Từ máy local hiện tại, em **không có quyền truy cập SSH/K8s hay kết nối trực tiếp vào NATS порт `:4222`/`:8222` của môi trường Production** (`cdc-cms.goopay.vn`), nên em tuyệt đối không suy diễn rằng trên PROD đã tự hết cảnh báo trước khi code mới được deploy.
  - Để trace giá trị runtime thật trên PROD và xác nhận nguyên nhân chính xác vì sao `EnsureStreams` trên PROD từng thất bại ở `CDC_EVENTS` (hoặc sau khi NATS PROD restart), em đề xuất phối hợp cùng anh theo 2 bước:
    1. Anh có thể kiểm tra nhanh log khởi động của pod `centralized-data-service` hiện tại trên PROD (`kubectl logs <pod> | grep -iE "NATS|EnsureStreams|stream"`) hoặc chạy `curl -s http://<nats-prod>:8222/jsz?streams=true&config=true` để xem cấu hình `Subjects`/`Retention`/`Storage` hiện tại của `CDC_EVENTS` trên PROD.
    2. Sau khi anh deploy bản cập nhật `centralized-data-service` và `cdc-cms-service` này lên PROD, chỉ cần anh mở lại trang `https://cdc-cms.goopay.vn/nats-monitor` (hoặc khi pod khởi động), hệ thống sẽ tự động tạo ngay 3 stream `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG` (bất chấp `CDC_EVENTS` có lệch cấu hình hay không) và in log `WARN` kèm đầy đủ `existing_*` vs `desired_*` nếu `CDC_EVENTS` cần đối chiếu thêm.

---

## Mục 3: Kiểm tra Concurrency / Multi-Worker / Timeout-Heartbeat / Data Continuity

- **Multi-Worker (`WorkerCount = 3`) & Multi-Pod External Consumer Recreation trong `TransmuteConsumerPool`:** Trong `TestTransmuteConsumerPool_WatchdogSelfHealsDeletedStreamWithoutDuplicateSubs` (`EV-20261023T115458-6a9e` `PASS`), bài test kiểm chứng liên tiếp 2 kịch bản khắc nghiệt:
  1. Xóa stream `CDC_TRANSMUTE` khi 3 workers đang chạy: `watchdogLoop` tự động tạo lại stream + consumer, gọi `unsubscribeSubsLocked()` hủy 3 subscription cũ rồi tạo 3 subscription mới (`len(pool.subs) == 3`), xử lý bản tin thứ 2 đúng 1 lần (`processed == 2`).
  2. Xóa stream `CDC_TRANSMUTE` và **ngay lập tức tạo lại stream + consumer `transmute-shadow-consumer-v2` với một `DeliverSubject` mới (`externalInbox`) trước khi `watchdogLoop` chạy tick tiếp theo** (mô phỏng Pod khác vừa tạo lại consumer trên NATS): `watchdogLoop` phát hiện `info.Config.DeliverSubject != boundDeliverSubject`, tự động re-bind cả 3 workers sang `externalInbox` (`len(pool.subs) == 3`) và xử lý bản tin thứ 3 đúng 1 lần (`processed == 3`), không nhân bản worker và không rơi vào trạng thái zombie `_INBOX`.
- **Lock Contention & Graceful Shutdown (`AckWait` / `Timeout` vs `Stop()`):** `watchdogLoop` nhả `p.mu.Unlock()` trước khi gọi `js.ConsumerInfo(..., nats.MaxWait(2*time.Second))`, và không tự hủy `p.subs` khi gặp lỗi timeout mạng thoáng qua (`EV-20261023T115458-6a9e` `PASS`).
- **Debouncer, Heartbeat (`InProgress`) & Watermark Continuity:** Khi `CDC_TRANSMUTE` hoạt động ở chế độ `WorkQueuePolicy`, `TableDebouncer` duy trì gửi heartbeat `msg.InProgress()` định kỳ trong lúc gom lô và chỉ gọi `msg.Ack()` sau khi ghi xong bảng Master, xóa ngay bản tin khỏi đĩa NATS sau khi `Ack` (`TestTransmuteConsumerPool_WorkQueueImmediateDeleteOnAck` đạt `PASS` theo biên lai `EV-20261023T115458-6a9e`).
- **Bảo toàn Subjects của Worker & Cooldown Auto-Heal trên CMS:** `TestNatsClient_EnsureStreams_PreservesExistingCDCEventsSubjects` và `TestNatsClient_EnsureStreams_ContinuesWhenOneStreamFails` trên NATS thật xác nhận `cdc-cms-service` bảo toàn nguyên vẹn `Subjects` tùy chỉnh của `CDC_EVENTS` (`["cdc.goopay.users", "cdc.custom.orders"]`) và không dừng sớm khi 1 stream bị xung đột subject (`EV-20261023T115543-bbdf` `PASS`).

---

## Mục 4: Kiểm tra Suy diễn & Báo cáo láo (Anti-Speculation & Honesty Check)

- **Các điểm thiếu sót / suy diễn ở vòng đầu đã được tự phát hiện và khắc phục bằng bằng chứng thật:**
  1. *Điểm suy diễn #1 (Đã khắc phục):* Ở vòng đầu, em báo cáo rằng `publishTransmuteTrigger` đã giảm thời gian fallback xuống `~1ms` nhưng chưa có unit/integration test gọi trực tiếp `publishTransmuteTrigger`. Khi bổ sung `TestPublishTransmuteTrigger_FastFallbackWhenStreamMissing` (`batch_buffer_fanout_test.go`), em phát hiện `nats.go` vẫn ngủ ngầm `500ms` (`2 * 250ms`) nếu thiếu `nats.RetryAttempts(0)`. Sau khi thêm `nats.RetryAttempts(0)`, bài test chạy thật đạt `< 50ms` (`0.04s`, biên lai `EV-20261023T115458-6a9e` `PASS`).
  2. *Điểm thiếu sót #2 (Đã khắc phục):* Ở vòng đầu, package `cdc-cms-service/pkgs/natsconn` chưa có file `*_test.go` riêng để kiểm chứng `preserveExistingSubjects = true` trên `CDC_EVENTS` có custom worker subjects. Em đã bổ sung `cdc-cms-service/pkgs/natsconn/nats_client_test.go` chạy trực tiếp trên NATS thật (`localhost:14222`, xác thực qua `config-local.yml`, không `SKIP`) và đạt `PASS` (`EV-20261023T115543-bbdf`).
  3. *Điểm minh bạch #3 (Laya Mode B `dg3review` vs Mode A `dg3`):* Khi thử nghiệm cả 2 chế độ của `/laya_router`, chế độ `review` (Mode B — gửi nguyên khối unified diff 30KB vào câu hỏi trắc nghiệm `APPROVE/REJECT`) trả về xác suất thấp như nhau (`confidence = 0.4385 < 0.95`) cho cả bản `real` lẫn bản `control_corrupted` (tức model RL không phân biệt được khối diff lớn nguyên tảng ở Mode B). Ngược lại, ở chế độ chuẩn `run --stage dg3` (Mode A — kiểm định từng claim cụ thể kèm trích dẫn `REF: <file>:<line>` và bộ câu hỏi đối chứng `CTRL-TRUE-1`, `CTRL-FALSE-1`, `CTRL-FALSE-2`), vòng `r3` vượt qua toàn bộ ngưỡng hiệu chuẩn (`trusted_rounds: ["r3"]`, `gate: PASS`). Em báo cáo trung thực cả hai kết quả để không che giấu bất kỳ hành vi nào của công cụ.

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop)

- **Hai bài học đã được ghi nhận vào `memory/global/lessons.md` (`2226 → 2240` dòng) và đồng bộ `lessons/INDEX.md` (`290` lessons):**
  1. `[2026-10-23] NATS Stream Reconciliation & Self-Heal Watchdog`: Không giữ `mu.Lock()` xuyên suốt lời gọi RPC mạng trong vòng lặp watchdog; phải `Unsubscribe()` các subscription cũ trước khi tạo lại slice subscriptions mới để tránh nhân bản worker; không gọi đồng bộ chuỗi RPC trong `nats.ReconnectHandler`; đặt cooldown nguyên tử (`atomic.Int64`) cho cơ chế auto-heal trên HTTP GET endpoint; và thoát sớm vòng lặp retry JetStream khi gặp `ErrNoStreamResponse` / `ErrNoResponders` / `ErrStreamNotFound`.
  2. `[2026-10-23] NATS Push Consumer External Recreation & Non-Destructive Watchdog Probe`: Khi stream/consumer trên NATS bị xóa rồi được tạo lại bởi một Pod khác với `DeliverSubject` (`_INBOX`) mới trước khi watchdog của Pod hiện tại chạy, `sub.IsValid()` vẫn là `true` và `js.ConsumerInfo` vẫn trả về `err == nil`. Do đó watchdog bắt buộc phải đối chiếu `info.PushBound`, `info.Config.DeliverGroup`, `info.Config.DeliverSubject` và `info.Created` với giá trị lúc bind, đồng thời không được hủy `p.subs` trước khi xác nhận stream/consumer trên NATS sẵn sàng.

---

## Mục 6: Kết quả QC & Nhận xét từ Hội đồng `/laya_router` (`DG3`)

- **Kết quả chạy `laya_gate.py run --stage dg3` (sau lần sửa code cuối cùng ở bước T6):**
  - File tổng hợp: `06_laya_summary_dg3_20261023T115653.json` (`ts: 20261023T115653`).
  - Trạng thái cổng `DG3`: `gate = PASS` (với `trusted_rounds: ["r3"]`, vượt qua toàn bộ các câu đối chứng `CTRL-TRUE-1`, `CTRL-FALSE-1`, `CTRL-FALSE-2`).
  - Chi tiết 5 khẳng định kiểm định code thật (`REF:` khớp 100% từng dòng trên đĩa sau khi hoàn tất Self-Healing Loop):
    - `A-1` (`EnsureStreams` cô lập lỗi từng stream bằng `errors.Join`): `ACCEPT` (`r3: YES, conf=0.6068`).
    - `A-2` (`ReconnectHandler` bất đồng bộ bảo vệ bởi `reconciling.CompareAndSwap`): `ACCEPT` (`r3: YES, conf=0.7968`).
    - `A-3` (`watchdogLoop` nhả khóa `p.mu` trước khi gọi `ConsumerInfo`, kiểm tra `PushBound`/`DeliverGroup`/`DeliverSubject`/`Created`, và gọi `ensureSubscribedLocked`): `ACCEPT` (`r3: YES, conf=0.8239`).
    - `A-4` (`publishTransmuteTrigger` truyền `nats.RetryAttempts(0)`, thoát sớm vòng lặp khi gặp `ErrNoStreamResponse`/`ErrNoResponders`/`ErrStreamNotFound` và bắn sang `cdc.cmd.transmute-shadow.core-fallback`): `ACCEPT` (`r3: YES, conf=0.7404`).
    - `A-5` (`GetOverview` tự động hồi phục Core Stream thiếu với cooldown `30s` và làm mới `/jsz` + `/connz`): `ACCEPT` (`r3: YES, conf=0.7793`).

