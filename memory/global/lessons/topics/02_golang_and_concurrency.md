# T2: Golang, GORM, Concurrency, Redis & Backend Runtime
> Tổng số bài học trong chủ đề này: **22** (trích xuất nguyên bản từ `memory/global/lessons.md`).

<!-- ID: L012 | lessons.md:L80 -->
### [2026-10-15] Báo Done khi chưa xử lý xung đột Port (`bind: address already in use`) và lỗi Race Condition đóng `sqlDB.Close()` / `redis.Close()` trước khi goroutine nền dừng hẳn trong `Shutdown()` gây bão lỗi `sql: database is closed` (Premature Done on Port Conflict & Unsynchronized Background Goroutine Teardown Race)
- **Global Pattern:** Khi khởi động/restart một dịch vụ backend [A] có các goroutine thu thập trạng thái nền (`healthCollector.Run`, `auditLogger.Run`, `alertMgr.RunBackgroundResolver`) chạy song song với HTTP Server (`app.Listen`) [B], lập trình viên mắc 2 sai lầm nghiêm trọng [X]: (1) Trong quy trình thực thi của Agent, khi chạy lệnh khởi động lại service hoặc kiểm tra môi trường mà thấy báo lỗi trùng cổng (`bind: address already in use`) do tiến trình cũ đang giữ port (`:8082` / `:8083`), Agent lại chủ quan bỏ qua và báo "Done" (đùn đẩy việc restart sang cho User) mà không dọn dẹp tiến trình mồ côi hoặc xác minh thực tế tất cả các service liên quan đều khởi động sạch sẽ; (2) Trong kiến trúc `Server.Start()` và `Server.Shutdown()`, các goroutine nền được spawn TRƯỚC khi `app.Listen(port)` bind cổng thành công, và khi `app.Listen` trả lỗi trùng port dẫn tới gọi `Shutdown()`, hàm `Shutdown()` chỉ gọi `cancel()` rồi lập tức đóng thẳng `redis.Close()` và `sqlDB.Close()` mà KHÔNG đợi (`sync.WaitGroup` / done channel) các goroutine nền kết thúc -> Hậu quả [Y]: Các câu query SQL và lệnh Redis đang chạy dở trong nhịp cold-start của `healthCollector` đập vào kết nối đã đóng, xả hàng loạt log lỗi `sql: database is closed`, `alert fire failed: InfrastructureDown: sql: database is closed`, `redis: client is closed` che lấp nguyên nhân thật là xung đột port `:8083`, khiến User không start được service và mất niềm tin vì Agent báo Done khi hệ thống chưa chạy ổn định. **Đúng:** (1) Trước khi báo Done bất kỳ task nào, BẮT BUỘC kiểm tra `lsof -nP -iTCP:<port> -sTCP:LISTEN` trên toàn bộ các cổng dịch vụ (`:8082`, `:8083`), dọn dẹp triệt để tiến trình trùng cổng và xác nhận service khởi động sạch sẽ 100% không có dòng log lỗi; (2) Trong `Server.Shutdown()`, BẮT BUỘC dùng `sync.WaitGroup` (hoặc kênh tín hiệu hoàn tất) chờ toàn bộ background goroutines (`healthCollector`, `auditLogger`, `alertResolver`, `stuckJobReaper`) thoát khỏi vòng lặp hoàn toàn TRƯỚC KHI gọi `redis.Close()` và `sqlDB.Close()`, đồng thời kiểm tra `ctx.Err() != nil` trước khi bắn tiếp `evaluateAlerts` hoặc ghi Redis.
- **Bối cảnh (Trigger):** User chạy `cdc-cms-service` lúc `15:52:12` bị văng hàng loạt lỗi `system_health_queries.go:106 sql: database is closed`, `alert_manager.go:337 sql: database is closed`, `redis: client is closed`, `OpenTelemetry shutdown` và chất vấn: `=> ? ko start đc sao báo done`.
- **Root Cause:** (1) Agent chủ quan báo Done ở lượt trước sau khi lệnh chạy `/tmp/cds-worker` báo `bind: address already in use` mà không kiểm tra toàn diện các tiến trình đang chiếm cổng `:8082` và `:8083` (trên máy đang có `PID 37116` / `37127` chạy từ `14:30` trên `ttys021` giữ chặt cổng `:8083`); (2) Trong `cdc-cms-service/internal/server/server.go` (dòng 396–445), `healthCollector.Run(ctx)` được gọi trước `s.app.Listen(":8083")` và `Shutdown()` đóng `sqlDB.Close()` + `redis.Close()` ngay lập tức mà không chờ `healthCollector` kết thúc nhịp query đầu tiên.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức (Mid-Session Fix theo Rule #5), nhận lỗi trực diện 100% vì đã báo Done khi chưa đảm bảo tiến trình khởi động sạch sẽ. (2) Ghi lesson vào `lessons.md` và kiểm tra snapshot số dòng. (3) Chỉ rõ tiến trình `PID 37116` / `37127` đang chiếm cổng `:8083` và lập hồ sơ giải pháp `09_tasks_solution_cms_shutdown_db_closed_race.md` trình bày rõ cơ chế đồng bộ `WaitGroup` trong `server.go` & `system_health_collector.go` chờ User phê duyệt (`APPROVE` theo Rule #1 & Rule #13).
- **Tags:** #premature-done-on-port-conflict #shutdown-goroutine-teardown-race #sql-database-is-closed #redis-client-is-closed #waitgroup-before-db-close #mid-session-fix

<!-- ID: L018 | lessons.md:L122 -->
### [2026-10-14] Suy diễn tên trường JSON không tồn tại trong DTO, nhầm cột cập nhật (Watermark) thành cột ngày tạo (Creation Date), bẫy GORM .Scan() 0 dòng không báo lỗi và lộ Secret trong Audit Log (Hallucinated DTO Fields, Watermark-vs-Creation Column Conflation, GORM Scan Zero-Row Trap & Audit Log Secret Leak)
- **Global Pattern:** Khi xây dựng luồng cấu hình kiểm đếm/báo cáo tự động [A] kết nối giữa Frontend, API quản trị và Worker nền [B], lập trình viên mắc 4 sai lầm chiều sâu chỉ lộ diện khi đối chiếu payload runtime từng trường [X]: (1) Suy diễn tên trường trong JSON response của API (`m.master_table`, `m.source_object_id`, `m.timestamp_field`) mà không in danh sách key thực tế từ endpoint runtime, đồng thời lấy nhầm trường `timestamp_field` (vốn là cột watermark cập nhật như `lastUpdatedAt` / `updatedAt`) hoặc cột chưa được `approved` (đang `rejected` / `pending` trên bảng đích) làm cột đếm ngày tạo $\rightarrow$ Đếm sai số liệu tạo mới hoặc nổ lỗi `SQLSTATE 42703 column does not exist`; (2) Dùng GORM `.Scan(&dest)` để đọc bản ghi thống kê và kiểm tra `err == nil && dest.Count >= 0` trong khi `.Scan()` KHÔNG trả về `ErrRecordNotFound` khi 0 dòng khớp (`RowsAffected == 0`, `Count = 0`) $\rightarrow$ Báo cáo hiển thị giả `"0"` thay vì `"Chưa đếm"`, đồng thời bỏ sót bản ghi khi tên đối tượng ở nguồn lưu dạng `kebab-case` (`export-jobs`) còn bảng đích lưu dạng `snake_case` (`export_jobs`); (3) Che (mask) `api_token` ở API đọc (`GET`) nhưng lại `json.Marshal(updates)` nguyên bản ghi thẳng vào bảng `activity_log` khi tạo/sửa (`POST`/`PATCH`), và dùng `strings.Index` thay vì `strings.LastIndex` khi cắt mật khẩu chứa ký tự `@` trong URI; (4) Khởi tạo lịch trình mới (`NextRunAt = targetToday`, `LastRunAt = nil`) nhưng thiếu rào chắn độ tươi (Freshness Window) khiến worker khi bật lại sau một đêm tắt máy nhận nhầm lịch mới tạo bị trễ là lệnh "Chạy ngay" (`run_now`) [Y]. **Đúng:** (1) Zero-Hallucination Runtime Schema: Bắt buộc inspect payload JSON thực tế từ port runtime; phân biệt rõ cột cập nhật (`lastUpdatedAt`) với cột ngày tạo (`createdAt`/`created_at`) và trên tầng đích (Shadow/Master) chỉ chọn cột có `status === 'approved'` (nếu chưa `approved` thì fallback về cột hệ thống `_created_at`); (2) GORM `.Scan()` Guard: Luôn kiểm tra `scanRes.Error == nil && scanRes.RowsAffected > 0` và chuẩn hóa đối sánh cả biến thể `snake_case` lẫn `kebab-case`; (3) End-to-End Secret Sanitization: Che token nhạy cảm trước khi ghi vào bất kỳ bảng audit/activity log nào và dùng `strings.LastIndex(authority, "@")` theo RFC 3986 khi mask URI; (4) Freshness Guard cho Run Now: Kết hợp kiểm tra `now.Sub(*sched.NextRunAt) <= 5*time.Minute` và loại trừ mốc `targetToday` cố định để chống kích hoạt nhầm.
- **Bối cảnh (Trigger):** Thực hiện Audit Phản Biện Chuyên Sâu Lượt 2 (Second-Pass Adversarial Line-by-Line & Runtime Payload Trace) theo yêu cầu "kiểm tra lại 1 lượt nữa cho anh" của User trên cổng `:8083` và `:8082`.
- **Root Cause:** Thói quen viết fallback dựa trên tên trường giả định ở Frontend mà chưa in schema thật của từng API (`masters`, `source-objects`, `mapping-rules`, `master-mapping-rules`), và bỏ sót các hành vi đặc thù của ORM (GORM `.Scan()` không trả lỗi khi 0 dòng) cũng như luồng ghi audit log phụ.
- **Fix/Correct Flow:** (1) Chạy script kiểm tra trực tiếp payload JSON của 4 endpoint metadata trên `:8083`, xóa bỏ hoàn toàn các field suy diễn và lấy cột ngày tạo `approved` từ `mapping-rules` / `master-mapping-rules`; (2) Sửa `confluence_runner.go` kiểm tra `scanRes.RowsAffected > 0` và tìm theo `candidateNames` (`_` & `-`); (3) Bổ sung `ComputeNextScheduledRun`, `sanitizeScheduleParamsForLog` và `isManualRunNowTrigger` (6/6 unit tests PASS); (4) Thẩm định 5/5 kịch bản qua `/laya_router` đạt 100% `R2=CORRECT` & `R3=YES`.
- **Tags:** #runtime-payload-trace #no-field-hallucination #watermark-vs-creation-date #gorm-scan-zero-row-trap #audit-log-secret-masking #rfc3986-uri-masking #run-now-freshness-guard

<!-- ID: L019 | lessons.md:L129 -->
### [2026-10-14] Giảm chu kỳ Ticker nhưng quên giảm TTL khóa phân tán Redis SetNX, gán nhầm nhãn Manual cho chu kỳ tự động và báo Done khi chưa restart tiến trình (Ticker-Lock TTL Desync, False Manual Trigger Detection & Stale Process Blindspot)
- **Global Pattern:** Khi tối ưu hóa độ trễ của bộ lập lịch tác vụ nền [A] và mở rộng API danh mục kết nối [B], lập trình viên mắc 4 lỗi hệ thống dây chuyền [X]: (1) Giảm chu kỳ `time.NewTicker` từ $T_1$ (10s) xuống $T_2$ (2s) nhưng giữ nguyên TTL của khóa phân tán `SetNX(lockKey, "locked", L)` với $L > T_2$ (8s) mà không giải phóng khóa cuối vòng lặp $\rightarrow$ Chính worker pod đó tự khóa mình trong $\lceil L/T_2 \rceil - 1$ nhịp tick liên tiếp, làm trễ toàn bộ các nhịp poll ngắn hạn của UI; (2) Dùng điều kiện `NextRunAt != nil && NextRunAt <= now` để nhận diện lệnh "Chạy ngay" (Run Now) trong khi mọi chu kỳ tự động sau lần chạy đầu tiên đều thỏa mãn điều kiện này $\rightarrow$ 100% lần chạy tự động bị ghi nhầm `triggered_by = "manual"` và có nguy cơ tự chạy bù sai khung giờ khi restart worker; (3) Đưa tên Database (`default_database`) vào dropdown chọn Namespace cho câu lệnh SQL `FROM "<schema>"."<table>"` của PostgreSQL và bỏ sót các bản ghi có `connection_code` rỗng thuộc kết nối mặc định; (4) Chỉ chạy `go build` (Exit 0) mà quên không restart tiến trình đang giữ port runtime, đồng thời trả nguyên chuỗi URI có `user:pass@host` ra API [Y] -> Hậu quả: Client gọi API mới nhận lỗi 404, lộ thông tin xác thực trong URI, và lệnh Run Now bị trễ 8 giây vượt quá cửa sổ polling của Frontend. **Đúng:** (1) Bất biến Ticker-Lock TTL: Khi khóa phân tán dùng để chống poll trùng nhịp mà không `DEL` cuối vòng lặp, bắt buộc $\text{TTL}_{\text{lock}} < \text{Interval}_{\text{ticker}}$ (ví dụ Ticker = 2s thì SetNX TTL = 1.5s); (2) Nhận diện trạng thái Run Now: Phải đối chiếu tổ hợp trạng thái duy nhất do API Run Now thiết lập (`LastRunAt == nil && NextRunAt != nil && NextRunAt <= now`); (3) Sanitize URI & Chuẩn hóa Schema: Luôn che `***:***@` trên mọi chuỗi host/URI trả ra API và chỉ nạp PostgreSQL Schema thực tế vào bộ chọn mệnh đề `FROM`; (4) Kiểm chứng Runtime thật: Sau khi build BẮT BUỘC restart tiến trình đang lắng nghe port và bắn request HTTP thật kiểm tra kết quả.
- **Bối cảnh (Trigger):** Tự thực hiện Adversarial Audit từng dòng code và bắn HTTP `PATCH /api/worker-schedule/7 -d '{"run_now": true}'` thật theo yêu cầu phản biện của User; phát hiện `cdc-cms-service` port 8083 chưa restart, `SetNX` 8s chặn ticker 2s, và `/laya_router` bắt lỗi `R3=NO` ở bộ lọc `default_master`.
- **Root Cause:** Chủ quan dừng ở mức biên dịch (`go build` pass) mà thiếu bước trace chuỗi thời gian thực thi (Ticker $\leftrightarrow$ Redis Lock TTL $\leftrightarrow$ UI Poll) và kiểm tra dữ liệu thực tế trả về trên cổng HTTP.
- **Fix/Correct Flow:** (1) Giảm TTL `SetNX` xuống `1500ms` (< ticker `2s`); (2) Sửa điều kiện Run Now thành `LastRunAt == nil && NextRunAt != nil && !NextRunAt.After(now)` và bổ sung unit test chống false trigger; (3) Mask URI qua `sanitizeConnectionHost` và chuẩn hóa `effectiveConnCode = m.master_connection_code || 'default_master'`; (4) Restart cả 2 service và kiểm chứng E2E HTTP thật + thẩm định qua `/laya_router`.
- **Tags:** #ticker-redis-lock-ttl-desync #false-manual-trigger #stale-process-blindspot #uri-credential-masking #postgres-schema-vs-db #laya-router-qc

<!-- ID: L071 | lessons.md:L518 -->
### [2026-09-16] Báo cáo Done và QC độc lập nhưng không thực sự chạy build/run dẫn đến lọt lỗi cú pháp biến undefined (False Build Verification & Undefined Variable Runtime Crash)


- **Global Pattern:** Khi thực hiện sửa đổi mã nguồn [A] và tiến hành rà soát QC độc lập [B], Agent chủ quan chỉ đối soát tĩnh bằng mắt (visual inspection) mà KHÔNG thực sự chạy lệnh biên dịch/chạy thử [X] (`go build`, `make run`, `go test`) -> Dẫn đến sót lỗi sai tên biến cục bộ (`undefined: schema`, `undefined: table`) làm sập service ngay khi khởi động (`make run` thất bại), gây mất uy tín nghiêm trọng trước User và vi phạm trực diện nguyên tắc verification thực tế (DoD G3). **Đúng:** (1) BẤT KỲ khi nào chạm vào mã nguồn (kể cả chỉ thêm 1 dòng gọi hàm), BẮT BUỘC phải chạy lệnh biên dịch thực tế (`go build ./cmd/...` hoặc `go test`) trên terminal; (2) Tuyệt đối không được báo "biên dịch thành công" hay "QC Pass" nếu chưa có output exit code = 0 từ compiler; (3) Kiểm tra kỹ tên tham số của hàm trước khi gọi (tránh gõ tắt `schema, table` khi tên tham số thực tế là `schemaName, tableName`).
- **Bối cảnh (Trigger):** User chạy `make run` (`go run cmd/worker/main.go`) thì bị lỗi biên dịch: `internal/service/shadow/schema_adapter.go:317:60: undefined: schema`, `internal/service/shadow/schema_adapter.go:317:68: undefined: table`. User chấn chỉnh gay gắt: "uhm, mày hay mày giỏi, mày có qc độc lập. make run ... trong khi cái service run còn ko đc...".
- **Root Cause:** Cẩu thả trong khâu nghiệm thu (False Verification): Người thực thi (Muscle) gõ nhầm tên biến `schema, table` thay vì `schemaName, tableName` trong hàm `createShadowTableV1WithCols`. Agent điều phối (Brain) khi làm báo cáo audit chỉ nhìn code so sánh logic mà không thực sự chạy lệnh `go build ./cmd/worker` để compiler bắt lỗi cú pháp.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức theo Mid-Session Fix (Rule #5). Ghi lesson vào `lessons.md`. (2) Sửa ngay dòng 317 `schema_adapter.go` thành `sa.EnsureSonyflakeTrigger(context.Background(), schemaName, tableName)`. (3) Chạy `go build ./cmd/worker` kiểm tra và xác nhận exit code = 0 thực tế. (4) Cập nhật tiến độ `05_progress.md` và tạ lỗi trung thực với User.
- **Tags:** #false-build-verification #undefined-variable #compiler-check-mandatory #mid-session-fix #dod-g3-violation

<!-- ID: L077 | lessons.md:L567 -->
### [2026-09-14] 6 sai sót phân tán & đồng quy khi chuyển NATS Core sang JetStream trong CDC Pipeline (Go Goroutine Order, Zombie Heartbeat, Dedup Drop, Missing FilterSubject, Poison Pill, Scheduler Race)

- **Global Pattern:** Khi chuyển đổi message broker từ fire-and-forget [A] (NATS Core) sang persistent/at-least-once [B] (NATS JetStream), Agent mắc 6 sai sót chí mạng về concurrency và distributed systems:
  (1) Sửa deadlock Mutex bằng `go func` đẩy vào channel gây out-of-order execution (đảo lộn thứ tự CDC events dẫn tới data corruption) và goroutine explosion;
  (2) Ticker `msg.InProgress()` chạy vô hạn không có deadline chặn trên tạo ra Zombie Worker khi DB/socket bị treo cứng;
  (3) Dùng correlation ID chung cho `Nats-Msg-Id` khiến JetStream broker âm thầm drop các batch CDC hợp lệ kế tiếp trong Deduplication Window;
  (4) Stream có nhiều subjects nhưng Pull Consumer không khai báo `FilterSubject` dẫn đến các worker pools kéo chéo message của nhau;
  (5) Chỉ `NakWithDelay` mà thiếu trần `MaxDeliver` / `msg.Term()` khiến message lỗi dữ liệu loop vô tận gây nghẽn toàn bộ worker;
  (6) Scheduler quét dọn task `running` dựa trên fixed threshold (30m) tự mâu thuẫn với các tác vụ dài hơi (Full Sync > 30m) gây race condition và trùng lặp job.
  **Đúng:**
  (1) Nhả Lock trước rồi thực hiện blocking send vào channel để bảo toàn tuyệt đối FIFO và tạo backpressure tự nhiên mà không bao giờ spawn goroutine bừa bãi;
  (2) Heartbeat `msg.InProgress()` BẮT BUỘC bọc trong `context.WithTimeout` (deadline tối đa), dừng ticker và `Nak` nếu quá hạn;
  (3) `Nats-Msg-Id` phải định danh duy nhất nội dung batch (Table + BatchUUID + Sequence hoặc Content Hash);
  (4) Mọi Pull Consumer trên Stream đa subject BẮT BUỘC khai báo tường minh `FilterSubject`;
  (5) Khai báo `MaxDeliver` trên Consumer Config; kiểm tra `meta.NumDelivered >= MaxDeliver` để `msg.Term()` và đẩy vào DLQ/log khi gặp Poison Pill;
  (6) Dùng Heartbeat Cột trên DB: worker cập nhật `updated_at = NOW()` định kỳ, Scheduler chỉ reset khi `updated_at` bị stale quá lâu (10m).
- **Bối cảnh (Trigger):** User đóng vai Principal Engineer phản biện gay gắt: "Bản kế hoạch phản biện đã nhìn ra đúng các điểm nghẽn lớn (như Early-Ack và Stampede), nhưng giải pháp kỹ thuật đề xuất lại đưa vào 6 lỗi sai kiến trúc và lỗ hổng vận hành mới. Một số đoạn code demo vi phạm nguyên tắc cơ bản của Go concurrency và distributed systems."
- **Root Cause:** Chưa nắm vững đặc tính bất định về thứ tự (scheduling nondeterminism) của Go goroutines; thiếu tư duy timeout deadline trong distributed heartbeat; hiểu sai cơ chế deduplication window của JetStream broker; bỏ quên trường `FilterSubject` trong NATS 2.9 multi-subject stream; thiếu cơ chế poison pill circuit breaker bằng `MaxDeliver`/`Term()`; và đặt hardcoded timeout tĩnh cho scheduler thay vì dynamic heartbeat.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức theo Mid-Session Fix (Rule #5). Ghi lesson vào `lessons.md`. (2) Áp dụng đúng 6 giải pháp chuẩn hóa Simplicity First: Tách Lock/Send FIFO; Heartbeat có deadline; Dedup ID theo batch content; Gắn `FilterSubject`; Bắt `meta.NumDelivered` gọi `msg.Term()`; Heartbeat `updated_at` cho Scheduler. (3) Đồng bộ cập nhật vào `09_tasks_solution_*.md` và `implementation_plan.md`.
- **Tags:** #nats-jetstream-migration #go-concurrency-fifo #goroutine-leak-prevention #zombie-worker-heartbeat-deadline #jetstream-deduplication-content-hash #jetstream-filter-subject #poison-pill-term-max-deliver #distributed-scheduler-heartbeat #simplicity-first #mid-session-fix

<!-- ID: L082 | lessons.md:L621 -->
### [2026-09-08] Config Fallacy — xếp hạng mục vào "Vận hành CMS" trong khi Go engine chưa có capability tương ứng

- **Global Pattern:** Khi đánh giá gap giữa platform CDC generic [A] và data contract [B], Agent xếp hàng loạt mục vào "[Vận hành CMS]" (config-only) trong khi Go engine CHƯA CÓ code xử lý: (1) UUID `binData:04` → canonical text (engine chỉ handle `$oid`); (2) gom unmapped fields vào `extra_fields` (`DynamicMapper` chỉ map field→column, không có reverse logic); (3) tách `clusterTime` pair `(T, I)` (code nhân `T*1000` mất `increment`) → Dẫn đến plan ảo tưởng "config là xong" trong khi thiếu 4 platform-level capability mới. **Đúng:** CMS config chỉ KÍCH HOẠT capability đã có trong code, không TẠO RA capability chưa viết. Trước khi xếp vào "[Vận hành CMS]", BẮT BUỘC verify Go engine source code có function tương ứng không.
- **Bối cảnh (Trigger):** User phản biện: "Ngụy biện Vận hành CMS là xong — CMS không thể cấu hình một tính năng mà code Go chưa viết."
- **Root Cause:** Sau khi đọc source code, Agent thấy platform mạnh → swing sang thái cực ngược: cho rằng mọi thứ đều config được. Không phân biệt "platform có mechanism X" vs "mechanism X đủ đáp ứng contract Y".
- **Fix/Correct Flow:** (1) Với mỗi yêu cầu trong contract, trace đến ĐÚNG function Go xử lý. (2) Nếu function thiếu/sai → xếp vào [Code thêm / Platform Enhancement]. (3) Chỉ xếp [Vận hành CMS] khi function Go đã tồn tại và đúng behavior.
- **Tags:** #config-fallacy #capability-vs-config #cdc-platform-gap #verify-engine-code-first

<!-- ID: L099 | lessons.md:L755 -->
### [2026-08-25] Báo cáo láo "Done" khi chỉ thêm struct field mà không trace nguồn data thực tế
- **Global Pattern:** Agent thêm field [F] vào struct [A] ở tầng backend Go, build pass, test pass → báo Done. Nhưng KHÔNG trace xem tầng caller (CMS frontend/API) có thực sự populate và gửi [F] trong payload hay không → User test thì payload vẫn thiếu [F]. **Đúng:** Trace E2E: UI → API → NATS → Worker. Mỗi tầng phải verify có code populate field mới.
- **Bối cảnh (Trigger):** Thêm `master_schema`/`master_table` vào Go struct (DTO, Event, Command), build pass, 9/9 test pass → báo Done. User test → CMS vẫn gửi payload thiếu 2 field này vì chưa ai populate chúng ở tầng CMS.
- **Root Cause:** (1) Agent chỉ verify bằng `go build` + `go test` — vi phạm G3 (test thật, không phải build-OK). (2) Không trace nơi CMS construct payload để gửi API — vi phạm G6 (output correctness trên dữ liệu thật). (3) Báo Done khi chưa verify end-to-end.
- **Fix/Correct Flow:** Khi thêm field vào pipeline multi-service: (1) Trace ngược đến tận UI/caller xem ai tạo payload, (2) Thêm field ở MỌI tầng từ source đến consumer, (3) Test bằng payload thật (gửi NATS/HTTP), không chỉ unit test.
- **Tags:** #bao-cao-lao #g3-build-ok-not-done #g6-output-correctness #e2e-trace

<!-- ID: L118 | lessons.md:L907 -->
### [2026-08-14] Lỗi phân giải tên miền host.docker.internal khi Worker chạy trực tiếp trên Host machine thay vì Container
- **Global Pattern:** Khi ứng dụng [A] chạy trực tiếp trên máy chủ vật lý/local dev host [B] kết nối tới các dịch vụ local thông qua địa chỉ `host.docker.internal` -> Gặp lỗi kết nối `dial tcp: lookup host.docker.internal: no such host`. **Đúng:** Tự động phát hiện và biên dịch ngược `host.docker.internal` thành `localhost` hoặc `127.0.0.1` khi chạy ngoài môi trường container.
- **Bối cảnh (Trigger):** Quét field SFTP kết nối tới `host.docker.internal:2022` từ worker chạy ngoài container (local dev) bị văng lỗi.
- **Root Cause:** `host.docker.internal` chỉ hoạt động và phân giải tự động bởi DNS daemon bên trong Docker container.
- **Fix/Correct Flow:** Bổ sung logic kiểm tra và tự động mapping `if host == "host.docker.internal" { host = "localhost" }` trong Go code khi chạy dev mode.
- **Tags:** #host-docker-internal #dns-resolution #sftp-connection-dial

<!-- ID: L125 | lessons.md:L956 -->
### [2026-08-13] Lỗi làm thay đổi signature phương thức của Interface gây vỡ hợp đồng biên dịch toàn hệ thống
- **Global Pattern:** Khi chỉnh sửa một file handler [A], Agent tự ý đổi chữ ký phương thức (method signature) của Interface [B] (thêm/bớt tham số như `HandleRaw`) mà không rà soát toàn bộ các struct implementation và caller downstream [C] -> Dẫn đến lỗi biên dịch nghiêm trọng (interface type mismatch) làm đứt gãy build `make run` và bắt User phải sửa lỗi hậu quả. **Đúng:** Mọi thay đổi đối với chữ ký của Interface BẮT BUỘC phải đối soát 100% các struct implementing interface đó và các caller trước khi thực hiện. Nếu lỗi xảy ra do Agent gây ra, BẮT BUỘC sửa triệt để ngay lập tức và chạy test/build xác minh thành công.
- **Bối cảnh (Trigger):** `snapshot_runner_handler.go` bị sửa signature `HandleRaw(ctx, subject, key, data)` lệch với `EventHandler.HandleRaw(ctx, subject, data)` làm `make run` bị crash. User bức xúc phản ánh do phải hốt dọn hậu quả code hỏng.
- **Root Cause:** Sửa signature interface cục bộ ở 1 nơi mà không check các struct implementer.
- **Fix/Correct Flow:** (1) Nhận trách nhiệm ngay lập tức. (2) Ghi lesson vào catalog. (3) Sửa ngay code về đúng signature chuẩn và verify build pass 100%.
- **Tags:** #interface-mismatch-error #handle-raw-signature #fix-immediately #full-stack-audit

<!-- ID: L145 | lessons.md:L1097 -->
### [2026-08-06] Selective Tracing qua Custom Tracer Sampler ở SDK level để triệt tiêu spam traces
- **Global Pattern:** Khi tích hợp các plugin tracing tự động của bên thứ ba (như GORM OpenTelemetry plugin) tạo ra lượng spans khổng lồ ("spam traces") làm quá tải Collector/SigNoz -> Không nên tắt hoàn toàn (mất vết debug) hoặc dùng Tail-based/Head-based sampling ratio mù quáng (làm mất liên kết trace gốc hoặc bỏ sót lỗi logic). **Đúng:** Xây dựng một Custom Tracer Sampler ở mức SDK (bọc ngoài default sampler). Custom Sampler này sẽ chặn (Drop) 100% DB spans (tên bắt đầu bằng `gorm.`) trừ khi context hiện tại được gắn cờ (đánh dấu) bằng module được phép trace động thông qua helper `WithDBTraceModule(ctx, moduleName)`.
- **Bối cảnh (Trigger):** Kích hoạt GORM OpenTelemetry plugin làm bùng nổ hàng triệu database spans từ các luồng ngầm (cron, logs, scheduler...) gây rác trace backend.
- **Root Cause:** OTel GORM plugin luôn tự động sinh span cho mọi truy vấn DB thực hiện qua context.
- **Fix/Correct Flow:** (1) Viết struct `SelectiveSampler` kế thừa `sdktrace.Sampler`. (2) Kiểm tra `strings.HasPrefix(p.Name, "gorm.")` và lookup module trong context. (3) Bọc sampler chính bằng `SelectiveSampler`. (4) Đính cờ `WithDBTraceModule` tại entrypoint các luồng cần debug.
- **Tags:** #selective-tracing #custom-sampler #gorm-otel-spam #trace-filtering

<!-- ID: L147 | lessons.md:L1111 -->
### [2026-08-05] Pattern: Đăng ký sai package GORM OpenTelemetry plugin làm lỗi biên dịch Go toolchain

- **Global Pattern:** Đăng ký plugin OpenTelemetry cho GORM [A] thông qua root package `gorm.io/plugin/opentelemetry` hoặc `otelgorm` -> Go toolchain báo lỗi `no required module provides package gorm.io/plugin/opentelemetry/otelgorm`. **Đúng:** Luôn sử dụng package con `/tracing` chính xác của GORM team: `go get gorm.io/plugin/opentelemetry/tracing`, sau đó import `"gorm.io/plugin/opentelemetry/tracing"` và đăng ký bằng `db.Use(tracing.NewPlugin())`.
- **Bối cảnh (Trigger):** Tích hợp OTel tracing cho luồng database GORM ở cả 2 service `centralized-data-service` và `cdc-cms-service`.
- **Root Cause:** Go toolchain không resolve được package `/otelgorm` từ module `gorm.io/plugin/opentelemetry` v0.1.16 do cấu trúc module thay đổi sang `/tracing`.
- **Fix/Correct Flow:** Sử dụng submodule `gorm.io/plugin/opentelemetry/tracing` và đăng ký bằng `tracing.NewPlugin()`.
- **Tags:** #gorm-otel-tracing #go-get-error #tracing-plugin-registration

<!-- ID: L163 | lessons.md:L1244 -->
### [2026-08-03] Tuyệt đối không tự ý lan rộng scope sửa nhiều file khi vấn đề nằm ở 1 vị trí (Rule #12 Minimal Impact)

- **Global Pattern:** Agent [A] nhận task xử lý ở module Transmute (`flatten.go`) -> Thay vì chỉ sửa đúng trong `flatten.go`, Agent lại đi sửa lan sang cả Scan Service (`scan_service.go`, `scan_handler.go`) và Child Explode (`child_explode.go`, `transmuter.go`) -> Vi phạm nghiêm trọng ranh giới kiến trúc (Architectural Boundary) và nguyên tắc Minimal Impact (Rule #12). **Đúng:** (1) Giữ đúng ranh giới module: Task thuộc Transmute (`flatten.go`) thì CHỈ XỬ LÝ NỘI BỘ TRONG `flatten.go`. (2) Không đụng vào Scan Service hay các module khác. (3) Hoàn tác ngay 100% các file sửa lan.
- **Bối cảnh (Trigger):** User yêu cầu `flatten.go` chỉ loop 1 phần tử đầu tiên của mảng → Agent sửa tùm lum file ở Scan và Child Explode.
- **Root Cause:** Bị nhầm lẫn ranh giới trách nhiệm (Scope/Boundary) giữa tầng Scan (khám phá schema) và tầng Transmute (chuyển đổi dữ liệu).
- **Fix/Correct Flow:** Thừa nhận sai sót, hoàn tác tất cả các file khác, chỉ sửa duy nhất `flatten.go`.
- **Tags:** #minimal-impact #architectural-boundary #transmute-only #strict-file-boundary

<!-- ID: L167 | lessons.md:L1276 -->
### [2026-07-31] Pattern: Read-only endpoint dùng chung scope resolver với dispatch endpoint → 500 khi binding inactive

- **Global Pattern:** Endpoint đọc [A] (read-only, e.g. GET status) tái dùng `resolveReadScope` của dispatch endpoint [B] → resolver JOIN `shadow_binding WHERE is_active=TRUE` → nếu binding inactive/absent → `ErrSourceObjectNoActiveShadow` → 500. **Đúng:** Read-only endpoint phải dùng query riêng không cần `is_active`, hoặc lookup thẳng từ resource table (không qua scope resolver).
- **Bối cảnh (Trigger):** `TransformJobStatusV2` dùng `resolveReadScope` → binding của source_object_id=14 inactive → 500 `resolve_scope_failed`.
- **Root Cause:** `resolveReadScope` vẫn JOIN `shadow_binding AND is_active=TRUE` vì `readScopeQuery` chỉ bỏ `so.is_active=TRUE`, không bỏ `sb.is_active=TRUE`. Read-only endpoint cho tracking job không cần scope resolve — chỉ cần lookup bằng source_object_id.
- **Fix/Correct Flow:** Thêm `GetLatestBySourceObjectID` query `shadow_binding` (bất kể `is_active`) để lấy `target_table`, rồi query `transform_jobs`. Bỏ hoàn toàn `resolveReadScope` khỏi status/cancel handler.
- **Tags:** #golang-scope-resolver #is-active-gate #read-only-endpoint #shadow-binding #fiber-500

<!-- ID: L168 | lessons.md:L1284 -->
### [2026-07-31] Pattern: Dynamic size tuning trước break-condition gây early-exit loop sau iteration đầu tiên

- **Global Pattern:** Handler [A] xử lý batch [X] theo loop → tuning `size` TRƯỚC khi check break `if count < size` → `count` được fetch với `size` cũ nhưng so sánh với `size` mới → khi `size` tăng (fast path), `count` < `size_new` = true → BREAK SỚM. **Đúng:** Lưu `requestedSize := size` TRƯỚC khi tune, so sánh break với `requestedSize` (giá trị đã dùng trong SQL LIMIT).
- **Bối cảnh (Trigger):** `BatchTransformHandler` dùng CTE chunked UPDATE, dynamic chunk size tăng gấp đôi sau mỗi iteration < 100ms → loop exit sau iter 1 dù còn 50M records.
- **Root Cause:** Scope của biến `chunkSize` bị tái dùng cho cả 2 mục đích: (1) LIMIT SQL và (2) break condition. Sau khi tune, nó chỉ đúng cho mục đích (1) của iter TIẾP THEO, nhưng break check dùng nó cho iter HIỆN TẠI.
- **Fix/Correct Flow:** `requestedChunkSize := chunkSize` → tune `chunkSize` → break check dùng `requestedChunkSize`.
- **Tags:** #golang-loop-logic #dynamic-batch-size #break-condition #premature-exit #chunk-processing

<!-- ID: L174 | lessons.md:L1332 -->
### [2026-07-22] Không được dùng ANY(?) khi truyền mảng slice Go trong Gorm Exec (Gây lỗi SQLSTATE 42601)

- **Global Pattern:** Agent [A] truyền mảng slice Go `chunk = []string{...}` vào câu lệnh Gorm `Exec` chứa `ANY(?)` -> Gorm tự động expand mảng thành danh sách phân tách dấu phẩy `($1, $2, $3, ...)` -> Làm cho cú pháp `ANY(($1, $2, $3))` trong PostgreSQL bị lỗi `ERROR: syntax error at or near "," (SQLSTATE 42601)`. **Đúng:** Với Gorm khi truyền slice Go dạng mảng tham số, BẮT BUỘC sử dụng cú pháp `IN (?)`: `WHERE "_source_id" IN (?) OR "_gpay_id"::text IN (?) OR "id"::text IN (?)` để Gorm expand mảng thành cú pháp `IN ($1, $2, $3, ...)` chuẩn 100% của SQL.
- **Bối cảnh (Trigger):** Sửa SQL Prune Master trong `recon_execute_heal_handler.go` dùng `ANY(?)` với mảng Go `chunk` làm log ném lỗi `syntax error at or near ","`.
- **Root Cause:** Sơ suất không nắm rõ cơ chế parameter slice expansion của Gorm v2 trên PostgreSQL.
- **Fix/Correct Flow:** Chuyển câu SQL DELETE về `IN (?)` chuẩn của Gorm.
- **Tags:** #gorm-slice-expansion #in-clause-vs-any #sqlstate-42601 #postgres-syntax-fix

<!-- ID: L180 | lessons.md:L1380 -->
### [2026-07-21] Thêm function vào sai struct/file khi multi-chunk edit apply nhầm

- **Global Pattern:** Agent [A] dùng multi_replace_file_content với nhiều chunk → tool apply chunk vào file [Y] thay vì file [X] (cùng package, target content gần giống) → function gắn sai struct → compile error. **Đúng:** Sau mỗi multi-chunk edit, đọc toàn bộ diff output, confirm TargetFile và struct đúng. Nếu sai → revert ngay trước khi build.
- **Bối cảnh (Trigger):** Chunk inject OTel vào `recon_check_handler.go` (`CheckHandler.natsPublisher`) nhưng apply vào `recon_base_handler.go` (`ReconBase` không có `natsPublisher`) → compile fail.
- **Root Cause:** Không verify diff output từng chunk. Chỉ nhìn "applied" rồi tiếp tục.
- **Fix/Correct Flow:** Đọc diff output → thấy sai file → xoá function sai ngay bằng replace_file_content → build lại.
- **Tags:** #wrong-file-placement #multi-chunk-risk #diff-verification #compile-error

<!-- ID: L184 | lessons.md:L1412 -->
### [2026-07-20] Lỗi biên dịch do ép sai kiểu trả về của Command execution hoặc dấu ngoặc dư thừa
- **Global Pattern:** Agent [A] gọi Command bus [X] nhưng gán sai kiểu trả về (như [Y] thay vì [Z]) hoặc viết dư thừa dấu ngoặc `}` ở cuối function block -> Gây lỗi biên dịch Go. **Đúng:** (1) Luôn kiểm tra kỹ kiểu trả về của method (ví dụ `bus.Execute` trả về `SyncResult`, trong đó payload raw nằm ở `.ResultBody`). (2) Chạy `go build` ngay sau khi sửa handler để phát hiện lỗi cú pháp sớm.
- **Bối cảnh (Trigger):** Triển khai API handler `Delete` cho shadow và master, viết thừa dấu ngoặc `}` trong hàm `PatchActive` làm lỗi cấu trúc file, đồng thời gán trực tiếp kết quả `bus.Execute` vào `c.Send` thay vì dùng `.ResultBody`.
- **Root Cause:** Cẩu thả khi sao chép và sửa mã nguồn, không chạy `go build` verify lập tức sau khi sửa.
- **Fix/Correct Flow:** Loại bỏ dấu ngoặc dư thừa, gọi `res.ResultBody` khi gửi response, chạy `go build ./internal/...` để kiểm tra.
- **Tags:** #go-syntax-error #type-mismatch #build-verification #carelessness

<!-- ID: L189 | lessons.md:L1448 -->
### [2026-07-13] Thiếu database repository injection ở handler dẫn đến mất kết quả đối soát (silent report loss)
- **Global Pattern:** Handler [A] gọi tiến trình check [B] thành công nhưng thiếu [X] (reportRepo) -> [Y] (kết quả report chỉ trả qua NATS/API mà không bao giờ được ghi xuống DB cdc_reconciliation_report). **Đúng:** Luôn inject reportRepo vào CheckHandler và gọi repo.Create sau khi tính toán xong.
- **Bối cảnh (Trigger):** Kích hoạt hash_window hoặc deep check từ cdc-cms-service, API trả về 202 Accepted và worker chạy xong nhưng bảng cdc_reconciliation_report rỗng.
- **Root Cause:** CheckHandler không được inject reportRepo và executeGenericCheck chỉ return report chứ không lưu DB.
- **Fix/Correct Flow:** Inject reportRepo vào CheckHandler, gọi reportRepo.Create(ctx, report) trong executeGenericCheck.
- **Tags:** #missing-db-insert #dependency-injection-omission #silent-report-loss

- **GORM Reflection Panic:** `tx.Raw(sql).Scan(&interface{})` sẽ panic (*call of reflect.Value.Type on zero Value*) nếu `interface{}` chưa cấp phát cụ thể. **Đúng:** Dùng `rows, _ := tx.Raw(sql).Rows()` -> `rows.Next()` -> `rows.Scan(&rawVal)`.
- **GORM Array String:** Lệnh `_id = ANY(?)` với tham số `[]string` sinh lỗi `22P02`. **Đúng:** Bắt buộc dùng toán tử `IN (?)` để GORM tự expand mảng.
- **GORM Raw().Scan Nested Struct:** `Raw().Scan` không hỗ trợ map tự động vào nested struct. **Đúng:** Scan vào flat struct có tags `gorm:"column:..."`, sau đó transpose thủ công sang nested.
- **Context Inheritance Timeout:** Dùng `context.Background()` tạo detached context ở hàm con làm đứt OTel TraceID, gây goroutine leak và DB deadline exceeded. **Đúng:** Kế thừa từ parent `context.WithTimeout(ctx, duration)`.
- **Mock Time Field:** Mock DB SQL (sqlmock) bị thiếu cột `checked_at` làm struct gán zero-time (0001-01-01) -> `time.Since()` tính ra tuổi >50 năm -> Panic runtime do flow rẽ nhánh ảo. **Đúng:** Luôn mock cột thời gian bằng `time.Now()`.
- **Map Serialization Order:** Map serialize theo Alphabetical, Struct theo thứ tự khai báo. Cẩn thận khi refactor map -> struct nếu hệ thống yêu cầu byte-identical checksum.
- **Helper Function Signature Mismatch:** Giả định sai signature của các helper functions có sẵn (vd: nghĩ `diffIDs` trả về 3 biến `missing, orphan, stale` trong khi thực tế chỉ trả về 2 biến `fromB, fromA`) khi thiết kế kỹ thuật dẫn đến lỗi biên dịch khi triển khai code. **Đúng:** Luôn kiểm tra định nghĩa thực tế của helper functions trong codebase trước khi viết spec hoặc implement code.

<!-- ID: L226 | lessons.md:L1755 -->
### [2026-07-27] Operation string sai vì không đọc taxonomy constants

- **Global Pattern:** Operation string dùng để filter activity log [A] phải được lấy từ constants file [B] (taxonomy.go / enum) — không được đặt tên theo intuition. Operation "transmute" không tồn tại; thật là "transform".
- **Root Cause:** Không đọc `taxonomy.go` trước khi truyền operation string vào API call.
- **Fix/Correct Flow:** Đọc taxonomy/constants file → dùng đúng string literal.
- **Tags:** #wrong-operation-string #taxonomy #read-constants-first

<!-- ID: L231 | lessons.md:L1794 -->
### [2026-07-27] Tuyệt đối CẤM tự tiện bịa đặt / nhồi nhét các trường không có trong thiết kế kiến trúc (như master_database)

- **Global Pattern:** Agent [A] thấy thiếu thông tin [X] -> Tự tiện phỏng đoán và thêm trường mới `master_database` vào struct `ActivityLogRow` [Y] mà không hề suy xét thiết kế hệ thống hiện tại (hệ thống vốn dĩ không dùng `shadow_database` hay `master_database` vẫn chạy bình thường) -> Vi phạm kỷ luật Simplicity First (Rule #12). **Đúng:** Giữ nguyên 100% struct `ActivityLogRow` và câu query Read. Chỉ sửa duy nhất 1 lỗi gốc rễ: `batch_buffer.go` ghi `target_table` = `tableName` (chứ không ghi `targetFQN`).
- **Bối cảnh (Trigger):** Tự ý nghĩ ra việc nhồi nhét `master_database` vào `ActivityLogRow` khiến User nhắc nhở nảy lửa: "master_database là cái gì ở đây... rồi shadow_database đâu. nó ko có nó vẫn hoạt động mà".
- **Root Cause:** Tư duy over-engineering, tự bịa ra trường mới thay vì tập trung đúng điểm lỗi đơn giản gốc rễ.
- **Fix/Correct Flow:** Loại bỏ hoàn toàn đề xuất thêm `master_database` / `master_schema` / `master_table`. Giữ nguyên 100% `cdc-cms-service`. Chỉ sửa đúng 1 dòng ở `batch_buffer.go` (truyền `tableName` thay vì `targetFQN`) và 1 dòng ở `transmute_handler.go` (loại bỏ `duration_ms` trùng lặp trong JSON).
- **Tags:** #no-over-engineering #simplicity-first #no-hallucinated-fields #minimal-impact

<!-- ID: L248 | lessons.md:L1932 -->
### [2026-09-21] Nhận định sai Edge Case và Lỗi Key Collision khi map dữ liệu 1:N
- **Global Pattern:** Khi hệ thống có [A] (VD: source) map 1:N tới [B] (VD: master binding), tuyệt đối CẤM dùng `map[Tên]` để chứa [B] vì nếu có [A1] và [A2] trùng tên, nó sẽ đè nhau. Hơn nữa, cấm mutate pointer dùng chung. **Đúng:** Phải dùng Slice (`[]struct`) kết hợp Value Copy, và LUÔN dùng Primary Key (`ID`) làm identifier duy nhất thay vì ghép chuỗi Tên.
- **Bối cảnh (Trigger):** Viết logic filter và kết hợp 2 nguồn dữ liệu, dùng `map[String]` để gom nhóm. Bị trùng tên DB và Bảng ở 2 Cluster khác nhau.
- **Root Cause:** Bị lười biếng, suy diễn "việc 1 source có nhiều master là edge case không ảnh hưởng" thay vì query thực tế DB để check xem dữ liệu có trùng hay không. Dẫn đến sai số liệu trên diện rộng.
- **Fix/Correct Flow:** Dùng SQL check thẳng vào DB để hiểu data shape. Đổi Key từ `String(Name)` sang `uint(ID)`. Đổi struct gom nhóm từ `Map` sang `Slice`. Tạo copy (`value`) thay vì dùng reference (`pointer`) để các luồng không dẫm đạp data.
- **Tags:** #go #pointer-mutation #key-collision #lazy-assumption #database

<!-- ID: L249 | lessons.md:L1939 -->
### [2026-09-21] Lỗi Hardcode Default Connection cho các Multi-Tenant Destinations
- **Global Pattern:** Trong kiến trúc Multi-Tenant / Multi-Database (1 source đổ vào N master database khác nhau), CẤM truy cập Database Connector qua các biến static / default như `rc.masterAgent`. **Đúng:** Phải dùng Connection Manager để resolve động Connection theo `connection_code` từng cấu hình tại thời điểm runtime (VD: `rc.GetMasterAgent(ctx, ref.MasterConnectionKey)`).
- **Bối cảnh (Trigger):** Viết logic Smoke Recon Segment B kiểm đếm từ bảng Shadow xuống bảng Master. Có 2 luồng Master khác nhau (`export_jobs_2` trỏ sang master_2, `cdc_transaction_history` trỏ sang postgres-develop).
- **Root Cause:** Dùng nhầm biến Global `rc.masterAgent` (default connection) cho tất cả các truy vấn Segment B. Dẫn tới hệ thống chọc vào 1 DB mặc định để tìm kiếm các bảng vốn nằm ở 1 DB khác. Kết quả DB mặc định báo không thấy (0 dòng hoặc error) và trả ra giao diện là `—` (không có số liệu).
- **Fix/Correct Flow:** Thay thế toàn bộ chỗ gọi `rc.masterAgent` trong `recon_smoke.go` bằng việc Get động qua `ref.MasterConnectionKey`. Bổ sung fallback tự rớt về `masterAgent` nếu không tìm thấy key cho tương thích ngược.
- **Tags:** #go #multi-tenant #connection-pooling #hardcode
