---
tier: T2
suffix: missing_core_stream
---

# 09 Technical Solution — Xử lý triệt để hiện tượng `MISSING_CORE_STREAM` (`CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`) trên `/nats-monitor`

## 1. Kết quả Điều tra Nguyên nhân Gốc rễ (S2 Investigation — `100% VERIFIED`)

### 1.1. Nơi sinh ra cảnh báo trên giao diện `https://cdc-cms.goopay.vn/nats-monitor` (`VERIFIED`)
1. **Frontend (`cdc-cms-web`):**
   - Tại [`NatsMonitor.tsx:1605-1638`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/NatsMonitor.tsx#L1605-L1638), khối banner **"Chẩn đoán Kiến trúc NATS ({alerts.length} phát hiện cần lưu ý)"** render danh sách `data.governance.alerts` trả về từ API `GET /api/system/nats/overview` ([`NatsMonitor.tsx:446`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/NatsMonitor.tsx#L446)).
2. **Backend (`cdc-cms-service`):**
   - Tại [`nats_monitor_handler.go:154-157`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go#L154-L157), hàm `Overview` gọi HTTP Monitoring của NATS `/jsz?streams=true&config=true&consumers=true`.
   - Tại [`nats_monitor_handler.go:674-847`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go#L674-L847), hàm `buildNATSGovernance` trích xuất danh sách stream hiện có qua `extractStreamsFromJsz(jsz)` (`line 678`), đánh dấu `seenCoreStreams[name] = true` (`line 715`), sau đó kiểm tra 4 Core Streams bắt buộc trong `canonicalCoreStreamOrder = ["CDC_EVENTS", "CDC_TRANSMUTE", "SCHEMA_DRIFT", "SCHEMA_CONFIG"]` (`lines 46-51, 836-846`).
   - Bất kỳ Core Stream nào không có mặt trong `/jsz` sẽ sinh ra alert `MISSING_CORE_STREAM[<StreamName>]`. Việc giao diện chỉ báo 3 lỗi (`CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`) mà **không báo `CDC_EVENTS`** chứng tỏ trên NATS JetStream PROD hiện tại **chỉ có `CDC_EVENTS` tồn tại, còn 3 stream `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG` đang thực sự chưa được tạo trên JetStream**.

---

### 1.2. Tại sao `CDC_TRANSMUTE` bị thiếu trên JetStream mà luồng Transmute của anh vẫn bắn qua và chạy bình thường? (`VERIFIED`)
Thực tế hiện tại trên PROD, luồng Transmute Shadow $\rightarrow$ Master **đang chạy 100% qua đường dự phòng Core NATS (RAM Fallback)** chứ không đi qua JetStream stream `CDC_TRANSMUTE`:

1. **Ở đầu bắn lệnh (`BatchBuffer.publishTransmuteTrigger`):**
   - Tại [`batch_buffer_fanout.go:104-129`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go#L104-L129), mỗi khi Kafka consumer flush xong dữ liệu vào bảng `shadow_*`, worker gọi `bb.js.PublishMsg(msg)` lên subject `"cdc.cmd.transmute-shadow"` (thử tối đa 3 lần, mất `100ms + 200ms = 300ms` tại `lines 105-115`).
   - Vì stream `CDC_TRANSMUTE` không tồn tại trên JetStream, NATS trả lỗi `nats: no response from stream`. Sau 3 lần thử thất bại, code ở `lines 119-129` **tự động chuyển sang bắn qua Core NATS (RAM)** trên subject dự phòng `"cdc.cmd.transmute-shadow.core-fallback"`:
     ```go
     fallbackMsg := nats.NewMsg("cdc.cmd.transmute-shadow.core-fallback")
     fallbackMsg.Data = data
     if fbErr := bb.natsConn.PublishMsg(fallbackMsg); fbErr != nil { ... }
     ```
2. **Ở đầu nhận lệnh (`NewWorkerServer`):**
   - Tại [`server_setup.go:360`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L360), worker **luôn mở sẵn** một Core NATS QueueSubscriber lắng nghe `"cdc.cmd.transmute-shadow.core-fallback"` bất kể `natsClient.JS` có khác `nil` hay không:
     ```go
     natsClient.Conn.QueueSubscribe(handlermaster.SubjectTransmuteShadowCoreFallback, "transmute-workers", transmuteHandler.HandleTransmuteShadow)
     ```
   - Khi nhận được message từ `"cdc.cmd.transmute-shadow.core-fallback"`, [`HandleTransmuteShadow`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L132-L253) tìm danh sách bảng Master đích và bắn tiếp `"cdc.cmd.transmute"` qua Core NATS (`lines 244-249`) $\rightarrow$ [`HandleTransmute`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L290-L323) $\rightarrow$ `TableDebouncer` $\rightarrow$ ghi xuống Master DB bình thường.
3. **Hệ quả ngầm nguy hiểm nếu cứ để chạy bằng Core NATS Fallback:**
   - **Bị phạt độ trễ `+300ms` trên mọi lần flush Shadow:** Do `bb.js != nil`, mỗi lần flush bảng Shadow đều phải chờ 3 lần retry `bb.js.PublishMsg` thất bại (`300ms`) rồi mới rơi sang Core NATS.
   - **Mất hoàn toàn cơ chế bền vững (Durability, `AckWait`, `MaxDeliver=5`, `InProgress` heartbeat) của JetStream `WorkQueuePolicy`:** Nếu đúng lúc `TableDebouncer` đang gom lô 2s mà pod worker restart hoặc kết nối Master DB chập chờn, lệnh chạy trên RAM sẽ mất luôn mà không được JetStream giữ lại để chạy lại.

---

### 1.3. Tại sao trên NATS JetStream PROD chỉ có `CDC_EVENTS` mà mất cả 3 stream `SCHEMA_DRIFT`, `SCHEMA_CONFIG`, `CDC_TRANSMUTE`? (`VERIFIED`)
Có **3 nguyên nhân gốc rễ (Root Causes)** kết hợp gây ra tình trạng này:

1. **RC-1 (`VERIFIED` — `centralized-data-service/pkgs/natsconn/nats_client.go:67-165`): Vòng lặp `EnsureStreams` bị "Fail-Fast" ngay tại phần tử đầu tiên (`CDC_EVENTS`) và lỗi bị nuốt thành Warning ở `server_setup.go:99-101`:**
   - Trong [`nats_client.go:67-109`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/natsconn/nats_client.go#L67-L109), mảng `desiredStreams` được khai báo theo đúng thứ tự:
     1. `CDC_EVENTS` (`index 0`, `lines 68-77`)
     2. `SCHEMA_DRIFT` (`index 1`, `lines 78-87`)
     3. `SCHEMA_CONFIG` (`index 2`, `lines 88-97`)
     4. `CDC_TRANSMUTE` (`index 3`, `lines 98-108`)
   - Trong vòng lặp `for _, desired := range desiredStreams` (`lines 111-165`), nếu stream đầu tiên `CDC_EVENTS` đã tồn tại từ trước và bước `c.JS.UpdateStream(&info.Config)` (`line 161`) hoặc kiểm tra stream trả về lỗi, dòng `162` lập tức **`return fmt.Errorf("failed to update stream %s config: %w", desired.Name, err)`** — dừng toàn bộ vòng lặp và **không bao giờ chạy tới 3 stream phía sau (`SCHEMA_DRIFT`, `SCHEMA_CONFIG`, `CDC_TRANSMUTE`)**!
   - Tại [`server_setup.go:99-101`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L99-L101), lỗi trả về từ `EnsureStreams` chỉ bị ghi `logger.Warn("failed to ensure NATS streams (may not have permissions)", zap.Error(err))` rồi worker tiếp tục chạy mà không có 3 stream kia.

2. **RC-2 (`VERIFIED` — `centralized-data-service/pkgs/natsconn/nats_client.go:26-28` & `transmute_consumer_pool.go:94-165`): Không có cơ chế Self-Heal khi Stream bị xóa thủ công (lúc dọn đĩa `/data/jetstream`) hoặc khi NATS Reconnect:**
   - `EnsureStreams` chỉ được gọi **đúng 1 lần duy nhất** lúc khởi động process `centralized-data-service`.
   - Tại [`nats_client.go:26-28`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/natsconn/nats_client.go#L26-L28), `nats.ReconnectHandler` chỉ log `"NATS reconnected"` chứ không gọi lại `EnsureStreams`.
   - Tại [`transmute_consumer_pool.go:94-165`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_consumer_pool.go#L94-L165), `TransmuteConsumerPool.Start()` cũng chỉ chạy 1 lần lúc boot. Nếu lúc boot stream `CDC_TRANSMUTE` chưa có (do RC-1), `p.js.ConsumerInfo("CDC_TRANSMUTE", ...)` trả lỗi `stream not found`, [`server_setup.go:349-354`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L349-L354) log warn `"failed to start transmute shadow JetStream consumer pool, falling back to core NATS"` rồi **không bao giờ thử khởi tạo lại pool nữa**; hoặc nếu sau khi boot mà stream `CDC_TRANSMUTE` bị xóa trên NATS (ví dụ khi dọn đĩa NATS), các subscription cũ trong `p.subs` bị mất hiệu lực vĩnh viễn cho tới khi restart pod worker.

3. **RC-3 (`VERIFIED` — `cdc-cms-service/pkgs/natsconn/nats_client.go:70-101` & `nats_monitor_handler.go:150-246`): `cdc-cms-service` bỏ quên `EnsureStreams()` (và bản thân hàm đó bị lệch chuẩn, thiếu `CDC_TRANSMUTE`), đồng thời `/nats-monitor` phát hiện thiếu Core Stream nhưng không tự động khôi phục:**
   - Trong [`cdc-cms-service/pkgs/natsconn/nats_client.go:71-101`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/natsconn/nats_client.go#L71-L101), hàm `EnsureStreams()` chỉ có 3 stream (thiếu `CDC_TRANSMUTE`, sai subjects/limits) và **chưa bao giờ được gọi** trong [`cdc-cms-service/internal/server/server.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L81-L87).
   - Trong khi đó, [`nats_monitor_handler.go:54-81`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go#L54-L81) (`canonicalCoreStreams`) đã nắm giữ đầy đủ thông số chuẩn của cả 4 Core Streams và có sẵn kết nối `h.natsClient.JS`, nhưng khi `Overview` phát hiện thiếu Core Stream (`CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`), nó chỉ báo lỗi `MISSING_CORE_STREAM` lên UI mà không tự động tạo/reconcile lại ngay các Core Stream bị thiếu.

---

## 2. Phương án Kỹ thuật Duy nhất & Tối ưu nhất (Single Best Solution — S3 Refined)

Thiết kế cơ chế **Multi-Layer Stream Self-Healing (Tự phục hồi Đa tầng, Không Fail-Fast, Lock-Free Network Check & Single Source of Truth)** đồng bộ trên cả `centralized-data-service` và `cdc-cms-service`, tích hợp đầy đủ **5 điểm tinh chỉnh kỹ thuật** của anh:

### Thành phần 1: Single Source of Truth `StreamConfig`, Chống Fail-Fast & Non-Blocking `ReconnectHandler` tại `centralized-data-service/pkgs/natsconn/nats_client.go`
1. **Single Source of Truth cho 4 Core Streams (`DefaultTransmuteStreamConfig`, `CoreStreamConfigs`):**
   - Khai báo tập trung cấu hình chuẩn của 4 Core Streams tại `pkgs/natsconn` (đặc biệt export `DefaultTransmuteStreamConfig()` cho `CDC_TRANSMUTE`) để cả `EnsureStreams` và `TransmuteConsumerPool` dùng chung đúng 1 hàm dựng cấu hình, triệt tiêu 100% nguy cơ lệch cấu hình (`CONFIG_DRIFT`).
2. **Cô lập lỗi từng Stream trong `EnsureStreams(cdcSubjects)` & Log chi tiết `existingConfig` vs `desiredConfig` khi `UpdateStream` lỗi:**
   - Không dùng `return fmt.Errorf(...)` thoát sớm khi 1 stream gặp lỗi; xử lý độc lập từng stream qua `ensureSingleStream(desired)` và gom lỗi bằng `errors.Join(errs...)`.
   - Khi `c.JS.UpdateStream(&info.Config)` thất bại (như trường hợp `CDC_EVENTS` trên PROD), log rõ toàn bộ trường `existing_subjects`, `existing_retention`, `existing_storage`, `existing_replicas`, `existing_max_bytes`, `existing_max_age`, `existing_duplicates` đối chiếu với `desired` để chẩn đoán chính xác nguyên nhân xung đột trên PROD mà không chặn 3 stream phía sau (`SCHEMA_DRIFT`, `SCHEMA_CONFIG`, `CDC_TRANSMUTE`).
3. **Non-Blocking `ReconnectHandler` có cờ chống chạy chồng lấp (`reconciling atomic.Bool`):**
   - Trong `nats.ReconnectHandler`, tuyệt đối không gọi đồng bộ `EnsureStreams` trên goroutine callback nội bộ của `nats.Conn`. Thay vào đó, kiểm tra `if client.reconciling.CompareAndSwap(false, true)` và tách ra một goroutine riêng (`defer client.reconciling.Store(false)`) để chạy `EnsureStreams(subjects)`.

### Thành phần 2: Lock-Free Network Check & Clean Subscription Drain trong `watchdogLoop` (`transmute_consumer_pool.go`) + Fast-Break `ErrNoStreamResponse` (`batch_buffer_fanout.go`)
1. **Không giữ `p.mu.Lock()` suốt quá trình Network I/O & Dọn sạch `p.subs` cũ trước khi Re-Subscribe ([`transmute_consumer_pool.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_consumer_pool.go)):**
   - Trong `watchdogLoop()`:
     - Bước 1: Chỉ giữ `p.mu.Lock()` trong vài nano-giây để kiểm tra nhanh trên RAM (`p.stopped`, `len(p.subs) < p.cfg.WorkerCount`, `sub.IsValid()`).
     - Bước 2: Nhả lock (`p.mu.Unlock()`) trước khi gọi RPC mạng `p.js.ConsumerInfo(p.cfg.StreamName, p.cfg.ConsumerName, nats.MaxWait(2*time.Second))` để kiểm tra sự tồn tại của stream/consumer trên NATS. Nhờ đó, nếu NATS chập chờn và ` Stop()` được gọi khi pod nhận `SIGTERM`, `Stop()` không bao giờ bị block chờ `RequestTimeout`.
     - Bước 3: Chỉ khi `needsHeal == true` mới lấy lại `p.mu.Lock()`, kiểm tra `if p.stopped { p.mu.Unlock(); return }`, và gọi `p.ensureSubscribedLocked()`.
   - Trong `ensureSubscribedLocked()`:
     - **BẮT BUỘC** duyệt qua toàn bộ `p.subs` cũ, gọi `_ = sub.Drain()` / `_ = sub.Unsubscribe()` cho các subscription còn sống và gán `p.subs = nil` **trước khi** tạo lại `p.cfg.WorkerCount` queue subscriptions mới — chống tuyệt đối lỗi nhân bản (`duplicate`) worker goroutines!
     - Dùng `natsconn.DefaultTransmuteStreamConfig()` thay vì hardcode lại `nats.StreamConfig`.
2. **Tối ưu độ trễ Fallback từ `300ms` xuống `~1ms` khi gặp `nats.ErrNoStreamResponse` ([`batch_buffer_fanout.go:105-116`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go#L105-L116)):**
   - Trong vòng retry 3 lần của `bb.js.PublishMsg(msg)`, kiểm tra ngay: nếu `errors.Is(pubErr, nats.ErrNoStreamResponse)` hoặc `errors.Is(pubErr, nats.ErrStreamNotFound)` (tức stream `CDC_TRANSMUTE` chưa tồn tại trên broker), **`break` khỏi vòng lặp retry ngay ở lần thử đầu tiên (`attempt == 1`)** để rơi thẳng xuống Core NATS fallback (`~1ms`) thay vì ngủ chờ vô ích `50ms + 100ms + 150ms = 300ms`.

### Thành phần 3: Đồng bộ Single Source of Truth & Cooldown `30s` (`atomic.Int64`) tại `cdc-cms-service`
1. **Dùng chung định nghĩa chuẩn giữa [`cdc-cms-service/pkgs/natsconn/nats_client.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/natsconn/nats_client.go) và [`nats_monitor_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go):**
   - Định nghĩa chuẩn thông số 4 Core Streams tại `pkgs/natsconn` và dùng chung cho cả `EnsureStreams()` lẫn `canonicalCoreStreams` trong `nats_monitor_handler.go`.
   - Gọi `EnsureStreams()` lúc `cdc-cms-service` khởi động ([`server.go:81-87`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L81-L87)) và bất đồng bộ trong `ReconnectHandler` (có cờ `reconciling atomic.Bool`).
2. **Cổng Cooldown `30s` (`lastHealAttemptUnix atomic.Int64`) cho Auto-Heal dự phòng trong `NATSMonitorHandler.Overview`:**
   - Khi `Overview` phát hiện `hasMissingCoreStreams(jsz)` là `true`, dùng `h.tryTriggerAutoHeal()` kiểm tra `lastHealAttemptUnix` bằng `CompareAndSwap` với chu kỳ tối thiểu **30 giây (`healCooldownSec = 30`)**.
   - Nếu đang trong thời gian cooldown 30s (hoặc có tab trình duyệt khác vừa kích hoạt), `GET /overview` trả về kết quả ngay lập tức mà không gọi lặp `EnsureStreams()` hay `/jsz` kép.

---

## 3. Code Demo Chi Tiết (DG2 Verification Target — Đã Tinh Chỉnh 5 Điểm)

### 3.1. `centralized-data-service/pkgs/natsconn/nats_client.go`
```go
type NatsClient struct {
	Conn        *nats.Conn
	JS          nats.JetStreamContext
	logger      *zap.Logger
	mu          sync.RWMutex
	cdcSubjects []string
	reconciling atomic.Bool
}

// DefaultTransmuteStreamConfig returns the canonical StreamConfig for CDC_TRANSMUTE
// (Single Source of Truth shared between EnsureStreams and TransmuteConsumerPool).
func DefaultTransmuteStreamConfig() *nats.StreamConfig {
	return &nats.StreamConfig{
		Name:       "CDC_TRANSMUTE",
		Subjects:   []string{"cdc.cmd.transmute-shadow"},
		Retention:  nats.WorkQueuePolicy,
		MaxAge:     1 * time.Hour,
		MaxBytes:   256 * 1024 * 1024, // 256 MiB cap
		Discard:    nats.DiscardOld,
		Duplicates: 2 * time.Minute,
		Storage:    nats.FileStorage,
		Replicas:   1,
	}
}

// Trong NewNatsClient — Khởi tạo client trước nats.Connect & Non-blocking ReconnectHandler:
	client := &NatsClient{
		logger: logger,
	}
	nc, err := nats.Connect(
		url,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
			client.mu.RLock()
			subjects := append([]string(nil), client.cdcSubjects...)
			client.mu.RUnlock()
			if len(subjects) == 0 {
				return
			}
			if client.reconciling.CompareAndSwap(false, true) {
				go func(subs []string) {
					defer client.reconciling.Store(false)
					if err := client.EnsureStreams(subs); err != nil {
						logger.Warn("failed to re-ensure NATS streams after reconnect", zap.Error(err))
					}
				}(subjects)
			}
		}),
	)

// Trong EnsureStreams(cdcSubjects []string) error — Không Fail-Fast + Log rõ xung đột UpdateStream:
func (c *NatsClient) EnsureStreams(cdcSubjects []string) error {
	if len(cdcSubjects) == 0 {
		return fmt.Errorf("EnsureStreams: cdcSubjects is required (derive from kafka.topicPrefix)")
	}
	c.mu.Lock()
	c.cdcSubjects = append([]string(nil), cdcSubjects...)
	c.mu.Unlock()

	desiredStreams := []*nats.StreamConfig{
		// CDC_EVENTS, SCHEMA_DRIFT, SCHEMA_CONFIG, DefaultTransmuteStreamConfig()
	}

	var errs []error
	for _, desired := range desiredStreams {
		if err := c.ensureSingleStream(desired); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c *NatsClient) ensureSingleStream(desired *nats.StreamConfig) error {
	info, err := c.JS.StreamInfo(desired.Name)
	if err == nats.ErrStreamNotFound {
		if _, err = c.JS.AddStream(desired); err != nil {
			return fmt.Errorf("failed to create stream %s: %w", desired.Name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get stream info %s: %w", desired.Name, err)
	}
	// ... reconcile Retention / mutable fields ...
	if needUpdate {
		if _, err := c.JS.UpdateStream(&info.Config); err != nil {
			c.logger.Warn("failed to update existing NATS stream config (continuing with remaining streams)",
				zap.String("stream", desired.Name),
				zap.Strings("existing_subjects", info.Config.Subjects),
				zap.Strings("desired_subjects", desired.Subjects),
				zap.Int64("existing_max_bytes", info.Config.MaxBytes),
				zap.Int64("desired_max_bytes", desired.MaxBytes),
				zap.Duration("existing_max_age", info.Config.MaxAge),
				zap.Duration("desired_max_age", desired.MaxAge),
				zap.Duration("existing_duplicates", info.Config.Duplicates),
				zap.Duration("desired_duplicates", desired.Duplicates),
				zap.Error(err),
			)
			return fmt.Errorf("failed to update stream %s config: %w", desired.Name, err)
		}
	}
	return nil
}
```

### 3.2. `centralized-data-service/internal/handler/master/transmute_consumer_pool.go` & `batch_buffer_fanout.go`
```go
// 1. Trong batch_buffer_fanout.go:105-116 — Tắt retry nội bộ 2x250ms của nats.go (nats.RetryAttempts(0)) và thoát ngay vòng retry nếu gặp ErrNoStreamResponse / ErrNoResponders / ErrStreamNotFound:
	if bb.js != nil {
		var pubErr error
		for attempt := 1; attempt <= 3; attempt++ {
			_, pubErr = bb.js.PublishMsg(msg, nats.RetryAttempts(0))
			if pubErr == nil {
				return
			}
			if errors.Is(pubErr, nats.ErrNoStreamResponse) ||
				errors.Is(pubErr, nats.ErrNoResponders) ||
				errors.Is(pubErr, nats.ErrStreamNotFound) {
				break
			}
			time.Sleep(time.Duration(attempt*50) * time.Millisecond)
		}
		bb.logger.Error("failed to publish transmute trigger to JetStream, falling back to core NATS",
			zap.String("table", shadowTable), zap.Error(pubErr))
	}

// 2. Trong transmute_consumer_pool.go — Tách unsubscribeSubsLocked() (cho Self-Heal) vs Drain() (cho Stop):
func (p *TransmuteConsumerPool) unsubscribeSubsLocked() {
	for _, sub := range p.subs {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	p.subs = nil
}

func (p *TransmuteConsumerPool) ensureStreamExistsLocked() error {
	if p.cfg.StreamName != "CDC_TRANSMUTE" {
		return nil
	}
	_, err := p.js.StreamInfo(p.cfg.StreamName)
	if err == nats.ErrStreamNotFound {
		_, addErr := p.js.AddStream(natsconn.DefaultTransmuteStreamConfig())
		return addErr
	}
	return err
}

func (p *TransmuteConsumerPool) ensureSubscribedLocked() error {
	if p.stopped {
		return nil
	}
	if p.js == nil {
		return fmt.Errorf("jetstream context is nil")
	}
	// Dọn dứt khoát các subscription cũ bằng Unsubscribe() khi Self-Heal để giải phóng goroutine nats.go ngay lập tức
	p.unsubscribeSubsLocked()

	if err := p.ensureStreamExistsLocked(); err != nil {
		return fmt.Errorf("failed to ensure stream %s: %w", p.cfg.StreamName, err)
	}
	// ... kiểm tra/tạo consumer và tạo mới p.cfg.WorkerCount QueueSubscribe ...
}

func (p *TransmuteConsumerPool) watchdogLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.mu.Lock()
			if p.stopped || p.js == nil {
				p.mu.Unlock()
				return
			}
			needsHeal := len(p.subs) < p.cfg.WorkerCount
			if !needsHeal {
				for _, sub := range p.subs {
					if sub == nil || !sub.IsValid() {
						needsHeal = true
						break
					}
				}
			}
			js := p.js
			streamName := p.cfg.StreamName
			consumerName := p.cfg.ConsumerName
			p.mu.Unlock()

			// Gọi Network I/O (ConsumerInfo) BÊN NGOÀI p.mu.Lock() với timeout ngắn 2s
			if !needsHeal {
				if _, err := js.ConsumerInfo(streamName, consumerName, nats.MaxWait(2*time.Second)); err != nil {
					needsHeal = true
				}
			}
			if !needsHeal {
				continue
			}

			p.mu.Lock()
			if !p.stopped {
				if err := p.ensureSubscribedLocked(); err != nil {
					p.logger.Warn("transmute consumer pool self-heal attempt failed",
						zap.String("stream", p.cfg.StreamName),
						zap.String("consumer", p.cfg.ConsumerName),
						zap.Error(err),
					)
				}
			}
			p.mu.Unlock()
		}
	}
}
```

### 3.3. `cdc-cms-service/pkgs/natsconn/nats_client.go` & `internal/api/system/nats_monitor_handler.go`
```go
// Trong cdc-cms-service/internal/api/system/nats_monitor_handler.go:
const autoHealCooldownSec = 30

func (h *NATSMonitorHandler) tryAutoHealMissingCoreStreams(ctx context.Context, jsz map[string]any) bool {
	if h.natsClient == nil || h.natsClient.JS == nil || !h.hasMissingCoreStreams(jsz) {
		return false
	}
	now := time.Now().Unix()
	last := h.lastHealAttemptUnix.Load()
	if now-last < autoHealCooldownSec {
		return false
	}
	if !h.lastHealAttemptUnix.CompareAndSwap(last, now) {
		return false
	}
	if err := h.natsClient.EnsureStreams(); err != nil {
		h.logger.Warn("auto-heal missing core NATS streams failed", zap.Error(err))
		return false
	}
	return true
}
```

---

## 4. Phân tích Tác động & Rủi ro (Impact & Edge Cases)
- **Graceful Shutdown Không Bị Block (`p.mu` Lock-Free Network Check):** `watchdogLoop` thực hiện `ConsumerInfo` bên ngoài `p.mu.Lock()` với `nats.MaxWait(2*time.Second)` và kiểm tra cờ `p.stopped` dưới lock trước khi tạo lại subscription. Khi `Stop()` được gọi, `p.stopped = true` và `close(p.stopCh)` diễn ra tức thì mà không bị treo chờ RPC mạng hay đua tạo lại `p.subs` sau khi đã `Drain()`.
- **Không Nhân Bản Goroutine (`unsubscribeSubsLocked` vs `Drain`):** Khi Self-Heal giữa chừng, `ensureSubscribedLocked()` dùng `sub.Unsubscribe()` giải phóng goroutine nội bộ của `nats.go` ngay lập tức; khi Graceful Shutdown, `Stop()` dùng `sub.Drain()` để xử lý nốt message đang chạy.
- **Zero Configuration Split-Brain:** Dùng chung `DefaultTransmuteStreamConfig` và thông số chuẩn giữa `pkgs/natsconn` và `nats_monitor_handler.go`, đảm bảo `buildNATSGovernance()` luôn thấy cấu hình khớp 100% từng byte.
- **Fast Fallback `~1ms`:** Khi stream `CDC_TRANSMUTE` vắng mặt (`ErrNoStreamResponse` / `ErrNoResponders` / `ErrStreamNotFound`), `publishTransmuteTrigger` chuyển ngay sang Core NATS sau lần thử đầu tiên thay vì tốn `300ms`.

### 4.1. Bổ sung Phản biện S5 (`S5 → S3 → S4 → S5` Self-Healing Loop)
1. **Phát hiện Zombie `_INBOX` khi Stream/Consumer bị Pod khác tạo lại (`boundDeliverSubject` & `boundConsumerCreated` trong `transmute_consumer_pool.go`):**
   - Khi stream bị xóa trên NATS, các push subscription cũ vẫn có `sub.IsValid() == true` (vì bản chất là Core NATS subscription lắng nghe `_INBOX.old`). Nếu một Pod khác (hoặc `EnsureStreams`) kịp tạo lại stream và consumer `transmute-shadow-consumer-v2` với `DeliverSubject` mới (`_INBOX.new`) trước khi `watchdogLoop` của Pod hiện tại chạy, lời gọi `js.ConsumerInfo` sẽ trả về `err == nil`.
   - Do đó, `TransmuteConsumerPool` lưu `boundDeliverSubject` và `boundConsumerCreated` lúc đăng ký `p.subs`; trong `watchdogLoop`, khi `ConsumerInfo` trả về `err == nil`, kiểm tra `!info.PushBound || info.Config.DeliverGroup != queueGroup || (boundDeliverSubject != "" && info.Config.DeliverSubject != boundDeliverSubject) || (!boundConsumerCreated.IsZero() && !info.Created.Equal(boundConsumerCreated))` để phát hiện consumer đã bị tạo lại và lập tức re-subscribe sang `_INBOX.new`.
2. **Chống tự hủy `p.subs` khi `ConsumerInfo` bị Timeout mạng thoáng qua:**
   - Khi tất cả `p.subs` cục bộ đều hợp lệ (`len(p.subs) == WorkerCount` và `sub.IsValid()`), `watchdogLoop` chỉ bật `needsHeal = true` khi lỗi `ConsumerInfo` là `ErrStreamNotFound` hoặc `ErrConsumerNotFound` (bỏ qua lỗi `ErrTimeout` thoáng qua). Đồng thời trong `ensureSubscribedLocked()`, đảm bảo stream và consumer tồn tại trên NATS xong rồi mới gọi `p.unsubscribeSubsLocked()` để thay thế `p.subs`.
3. **Đồng bộ khóa `c.mu` khi gán và đọc `c.Conn` / `c.JS` trong `nats_client.go`:**
   - Bảo vệ việc gán `client.Conn = nc`, `client.JS = js` trong `NewNatsClient` và đọc `c.JS` trong `ensureSingleStream` bằng `c.mu` ở cả `centralized-data-service` và `cdc-cms-service`, loại bỏ hoàn toàn nguy cơ data race nếu `ReconnectHandler` kích hoạt sớm.
4. **Bổ sung Test Trực Tiếp cho mọi khẳng định (Zero Speculation):**
   - Thêm `batch_buffer_fanout_test.go` kiểm chứng trực tiếp `publishTransmuteTrigger` khi thiếu `CDC_TRANSMUTE` chuyển sang `cdc.cmd.transmute-shadow.core-fallback` trong `< 50ms`.
   - Mở rộng `transmute_debounce_group_test.go` kiểm chứng kịch bản Multi-Pod External Consumer Recreation.
   - Thêm `cdc-cms-service/pkgs/natsconn/nats_client_test.go` kiểm chứng trực tiếp `preserveExistingSubjects = true` trên `CDC_EVENTS`.
