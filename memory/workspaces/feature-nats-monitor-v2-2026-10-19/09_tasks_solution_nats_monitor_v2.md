# 09. Tasks Solution — Giải pháp Nâng cấp Toàn diện `/nats-monitor` (`nats_monitor_v2`)

> **Trạng thái:** `PROPOSED` (Chờ User `APPROVE` trước khi thực thi sang pha Muscle S4)

---

## 1. Root Cause / Vấn đề cốt lõi (`VERIFIED`)

Trang `/nats-monitor` hiện tại bị **"chung chung"** và **"khó kiểm soát"** do 4 nguyên nhân gốc rễ trong thiết kế hiện tại:

1. **Backend `NATSMonitorHandler` hoàn toàn tách rời khỏi `natsClient.JS` & Ngữ cảnh Data-Hub:**
   - Tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go:16-33`, `NATSMonitorHandler` chỉ nhận `natsMonitorURL string` và `logger *zap.Logger`, chỉ gọi HTTP GET thô tới `:18222` (`/varz`, `/jsz?streams=true&config=true&consumers=true`, `/connz` không có `?subs=detail`) tại `nats_monitor_handler.go:46-61`.
   - Trong khi đó, tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go:120`, `cdc-cms-service` đã khởi tạo sẵn `natsClient` (`*natsconn.NatsClient` gồm cả `Conn *nats.Conn` và `JS nats.JetStreamContext` tại `cdc-cms-service/pkgs/natsconn/nats_client.go:14-17`), nhưng `server.go:326` không truyền `natsClient` vào `NewNATSMonitorHandler`.
2. **Không có Quản trị Quy chuẩn Stream (Stream Governance & Drift Detection):**
   - Trong kiến trúc Data-Hub (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/natsconn/nats_client.go:67-109`), hệ thống có **4 Core Streams** với quy chuẩn nghiêm ngặt:
     - `CDC_EVENTS`: `LimitsPolicy`, `MaxAge = 24h`, `MaxBytes = 512 MiB`, `DiscardOld`
     - `CDC_TRANSMUTE`: `WorkQueuePolicy`, `MaxAge = 1h`, `MaxBytes = 256 MiB`, `DiscardOld`, `Duplicates = 2m`
     - `SCHEMA_DRIFT`: `LimitsPolicy`, `MaxAge = 7d`, `MaxBytes = 64 MiB`, `DiscardOld`
     - `SCHEMA_CONFIG`: `LimitsPolicy`, `MaxAge = 7d`, `MaxBytes = 64 MiB`, `DiscardOld`
   - Giao diện `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/NatsMonitor.tsx:308-430` chỉ liệt kê danh sách stream chung chung, không hiển thị `retention` (`workqueue` vs `limits`), không tính `% MaxBytes` (`state.bytes / config.max_bytes`), không cảnh báo khi Stream bị **Lệch cấu hình (`CONFIG_DRIFT`)** (như `CDC_TRANSMUTE` bị để `limits` không có `max_bytes` gây đầy ổ `/data/jetstream`), và không phân biệt **Core Stream** với **Orphan Stream** (như `TEST_WQ_CHECK`).
3. **Mù Chẩn đoán Điểm nghẽn Consumer & Mù Nội dung Message (`NatsMonitor.tsx:435-521`):**
   - Bảng Consumer trong Drawer không hiển thị tương quan 3 mốc sequence (`ack_floor.stream_seq` $\rightarrow$ `delivered.stream_seq` $\rightarrow$ `stream.state.last_seq`), không hiển thị tỷ lệ bão hòa cửa sổ In-Flight (`num_ack_pending / config.max_ack_pending`), không hiển thị `ack_wait` (30s), `max_deliver` (5), `deliver_group`, `push_bound` và `last_active`.
   - Không có tính năng **Message Inspector (Peek Message)** để xem các message đang ứ đọng trong Stream thuộc bảng `shadow_table` nào, `_source_ids` nào mà không làm mất message.
4. **Không có Control Plane Actions & Không kiểm soát được Kênh lệnh Core NATS (`cdc.cmd.*`):**
   - Giao diện hiện tại 100% Read-Only: không thể **Purge Stream**, không thể **Xóa 1 Poison Message theo Seq**, không thể **Reconcile / Chuẩn hóa Config Stream**, không thể **Chỉnh Quota `MaxBytes`/`MaxAge`**, không thể **Xóa Stream Rác (Orphan)**, và không thể **Reset/Delete Consumer**.
   - Không đối chiếu danh sách `subscriptions` thực tế của `cdc-worker` với **Danh mục Kênh lệnh Core NATS (`cdc.cmd.*`)** (`cdc-cms-service/internal/server/server.go:265-285` & `centralized-data-service/internal/server/server_setup.go:299-560`), nên khi worker mất kết nối hoặc thiếu listener trên 1 subject, Operator không thể phát hiện trên `/nats-monitor`.

---

## 2. Giải pháp Kỹ thuật Duy nhất & Tối ưu nhất (Single Best Architecture — Đã qua Adversarial Audit `10_gap_analysis_nats_monitor_v2.md`)

Nâng cấp `/nats-monitor` thành **Data-Hub NATS Control Center (V2)** theo nguyên tắc **Simplicity First, Minimal Impact**, tuân thủ 100% Architecture (`destructiveChain`, `ConfirmDestructiveModal`, `natsconn.NatsClient`) của `data-hub`:

### 2.1. Backend (`cdc-cms-service`)
1. **Tiêm `*natsconn.NatsClient` vào `NATSMonitorHandler` (`server.go:326` & `nats_monitor_handler.go`):**
   - Cập nhật `NewNATSMonitorHandler(natsMonitorURL string, logger *zap.Logger, natsClients ...*natsconn.NatsClient) *NATSMonitorHandler` (dùng variadic `natsClients ...*natsconn.NatsClient` để giữ tương thích ngược 100% với các unit test cũ chỉ truyền 2 tham số).
2. **Nâng cấp `GET /api/v1/system/nats/overview` (`GetOverview`):**
   - Giữ nguyên **3 goroutines song song** với timeout `3s` (không gọi thừa `/subsz?subs=1` — Fix `F2`):
     - `/varz`
     - `/jsz?streams=true&config=true&consumers=true`
     - `/connz?subs=detail` (trích xuất chính xác trường **`subscriptions_list_detail`** chứa `subject`, `qgroup`, `sid`, `msgs`, `cid` — Fix `F2`)
   - Bổ sung bộ phân tích **`governance`** ngay trong response của `GetOverview`:
     - **`stream_specs`**: Đối chiếu từng stream trên broker với quy chuẩn 4 Core Streams (`CDC_EVENTS`, `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`):
       - Phân loại `tier`: `"CORE"` hoặc `"ORPHAN"`.
       - Mô tả `role`: vai trò nghiệp vụ trong Data-Hub.
       - Tính `quota_used_pct`: `state.bytes / config.max_bytes * 100` (nếu `max_bytes <= 0` thì đánh dấu `unlimited: true` — Fix `F7`).
       - Phát hiện `drift_reasons []string`: kiểm tra `retention` (`workqueue` cho `CDC_TRANSMUTE`, `limits` cho 3 stream còn lại), `max_bytes <= 0` hoặc lệch trần, `max_age`.
     - **`command_channels`**: Đối chiếu danh mục **19 Core NATS Command Subjects bắt buộc** (luôn luôn đăng ký vô điều kiện trong `centralized-data-service/internal/server/server_setup.go:281-530` — Fix `F5`, loại bỏ các subject đã comment out hoặc phụ thuộc feature flag để tránh báo động giả) với mảng `subscriptions_list_detail` từ `/connz?subs=detail` để trả về `active_listeners`, `queue_group`, `worker_cids`, `msgs_delivered` và trạng thái `READY` / `NO_LISTENER`.
     - **`alerts`**: Mảng cảnh báo tự động (`STREAM_DRIFT`, `ORPHAN_STREAM`, `STORAGE_PRESSURE`, `CONSUMER_STALLED`, `MISSING_COMMAND_LISTENER`, `SLOW_CONSUMER`).
3. **Bổ sung 1 API Đọc chuyên sâu & 5 API Điều khiển an toàn (Control Plane Endpoints) trên `NATSMonitorHandler`:**
   - **Read-Only Message Inspector (`shared` role: `admin`, `operator`):**
     - `GET /api/v1/system/nats/streams/:stream/messages` (`GetStreamMessages`):
       - Query params: `mode=latest|first|seq|subject`, `seq=<uint64>`, `subject=<string>`, `limit=<int>` (mặc định `10`, tối đa `50`).
       - Kiểm tra `info.State.Messages == 0 || info.State.FirstSeq > info.State.LastSeq` trả về mảng rỗng ngay lập tức; khi duyệt `latest`/`first` trên `WorkQueuePolicy` có sparse sequence gaps, tự động bỏ qua các sequence đã bị xóa với giới hạn `maxProbes = limit * 4` (tối đa `60` probes) và `context.WithTimeout(2500ms)` (Fix `F4`).
       - Đọc bằng `h.natsClient.JS.GetMsg(stream, seq)` hoặc `GetLastMsg(stream, subject)`, cắt payload preview tối đa `16KB`. **Hoàn toàn không làm tiêu thụ hay Ack message trên `WorkQueue`!**
   - **Control Actions (`admin` group + `destructiveChain`: `RequireOpsAdmin` + `Idempotency` + `Audit` — Fix `F1`, `F3`, `F8`):**
     - *(Lưu ý kiến trúc `F1`: Loại bỏ hoàn toàn `POST /streams/reconcile` khỏi `cdc-cms-service` vì `cdc-cms-service` không có `cfg.Kafka.TopicPrefix` và việc xóa/tạo lại stream lúc runtime sẽ làm đứt kết nối 10 push subscriptions của `TransmuteConsumerPool` bên `cdc-worker`. Trách nhiệm `EnsureStreams` thuộc về `centralized-data-service`).*
     - `POST /api/v1/system/nats/streams/:stream/purge` (`PurgeStream`):
       - Body: `{"subject": "...", "keep": 0, "seq": 0}`. Chặn `400 Bad Request` nếu truyền đồng thời `keep > 0` và `seq > 0` (Fix `F8`).
       - Gọi `h.natsClient.JS.PurgeStream(stream, opts...)` để làm sạch message trong stream mà vẫn giữ nguyên stream & consumer subscriptions.
     - `DELETE /api/v1/system/nats/streams/:stream/messages/:seq` (`DeleteStreamMessage`):
       - Gọi `h.natsClient.JS.DeleteMsg(stream, seq)` để loại bỏ đích danh 1 poison-pill message bị kẹt.
     - `PATCH /api/v1/system/nats/streams/:stream/config` (`UpdateStreamConfig`):
       - Body: `{"max_bytes": 268435456, "max_age_seconds": 3600}`.
       - Lấy `info, err := h.natsClient.JS.StreamInfo(stream)`, chỉ cập nhật in-place `MaxBytes` và `MaxAge` trên `info.Config` (giữ nguyên 100% `Subjects`, `Retention`, `Storage`), rồi gọi `h.natsClient.JS.UpdateStream(&info.Config)`.
     - `DELETE /api/v1/system/nats/streams/:stream` (`DeleteStream`):
       - Chặn cứng (`HTTP 400`) nếu `:stream` nằm trong 4 Core Streams (`CDC_EVENTS`, `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`). Chỉ cho phép xóa các stream `ORPHAN` (như `TEST_WQ_CHECK`).
     - `DELETE /api/v1/system/nats/streams/:stream/consumers/:consumer` (`DeleteConsumer`):
       - Gọi `info, err := h.natsClient.JS.ConsumerInfo(stream, consumer)` trước. **Chặn cứng (`HTTP 400`) nếu `info.PushBound || info.NumWaiting > 0`** (đang có worker lắng nghe) để bảo vệ kết nối của `TransmuteConsumerPool` (Fix `F1`). Chỉ cho phép xóa consumer mồ côi (`PushBound == false` & `NumWaiting == 0`).
   - **Mở rộng `cdc-cms-service/internal/middleware/audit.go:306-346` (Fix `F3`):**
     - Bổ sung 5 route NATS vào `defaultActionMap()` (`NATS_STREAM_PURGE`, `NATS_STREAM_CONFIG_UPDATE`, `NATS_MESSAGE_DELETE`, `NATS_STREAM_DELETE`, `NATS_CONSUMER_DELETE`) và mở rộng `deriveTarget(c)` đọc `c.Params("stream")`, `c.Params("consumer")`, `c.Params("seq")` để tự động lưu vết vào bảng `cdc_system.admin_actions`.

### 2.2. Frontend (`cdc-cms-web/src/App.tsx` & `cdc-cms-web/src/pages/NatsMonitor.tsx`)
1. **Mở lại Menu `/nats-monitor` trên Sidebar (`cdc-cms-web/src/App.tsx:139-143` — Fix `F6`):**
   - Import `CloudServerOutlined` và uncomment mục `/nats-monitor` (`Giám sát NATS`).
2. **Tái sử dụng `ConfirmDestructiveModal` (`cdc-cms-web/src/components/ConfirmDestructiveModal.tsx` — Fix `F3`):**
   - Mọi thao tác xóa/purge trên `NatsMonitor.tsx` đều mở `ConfirmDestructiveModal` chuẩn của hệ thống (bắt buộc nhập lý do `>= 10` ký tự, tự động gửi kèm `'Idempotency-Key'` và `'X-Action-Reason'`).
3. **Thiết kế giao diện NATS Control Center 4 khu vực (`cdc-cms-web/src/pages/NatsMonitor.tsx`):**
   - **Khu vực 1 — Smart Diagnostics Banner & 5 KPI Cards:** Hiển thị cảnh báo thông minh (`alerts`), thanh `%` JetStream Storage Quota, Slow Consumers, Pending/AckPending/Redelivered, và Command Channels Coverage (`19/19 Ready`).
   - **Khu vực 2 — Tab 1 (`JetStream Streams & Governance`):** Hiển thị vai trò nghiệp vụ, badge `CORE` / `ORPHAN`, trạng thái `SYNCED` / `DRIFT`, thanh `Bytes / MaxBytes` (xử lý chuẩn `max_bytes <= 0` thành `UNLIMITED` và `messages === 0` thành `Trống (Next: #seq)` — Fix `F7`), cùng các nút **Soi Message (Peek)**, **Consumers**, **Chỉnh Quota**, **Purge**, **Xóa Stream Rác**.
   - **Khu vực 3 — Drawer `Consumer Diagnostics` & Drawer `Message Inspector`:**
     - **Consumer Drawer:** Hiển thị `PushBound` (`Đang gắn Worker` vs `Mồ côi`), `DeliverGroup`, tiến độ `AckFloor → Delivered / StreamLastSeq`, thanh bão hòa `AckPending / MaxAckPending`, `AckWait (s)`, `MaxDeliver`, `LastActive`, và nút **Xóa Consumer Mồ côi**.
     - **Message Inspector Drawer:** Xem message theo `latest` / `first` / `seq` / `subject`, xem `Headers` & `Payload JSON` đã format và nút **Xóa Poison Message (`#seq`)**.
   - **Khu vực 4 — Tab 2 (`Kênh lệnh Data-Hub cdc.cmd.*`) & Tab 3 (`Active Connections & Subscriptions`):**
     - **Tab 2:** Bảng đối soát 19 kênh lệnh Core NATS bắt buộc (`READY` / `NO_LISTENER`, `Queue Group`, `Active Listeners`, `Worker CIDs`, `Msgs Delivered`).
     - **Tab 3:** Bảng kết nối kèm `RTT`, role và expandable row xem chi tiết bảng `subscriptions_list_detail` của từng connection.

---

## 3. Blast Radius & Rollback Plan

### 3.1. Blast Radius (Phạm vi ảnh hưởng — Minimal Impact)
1. `cdc-cms-service/internal/middleware/audit.go:306-346`: Thêm 5 mã action `NATS_*` vào `defaultActionMap()` và đọc thêm param `stream`/`consumer`/`seq` trong `deriveTarget(c)`.
2. `cdc-cms-service/internal/server/server.go:326`: Truyền thêm `natsClient` vào `apisystem.NewNATSMonitorHandler(cfg.System.NatsMonitorURL, logger, natsClient)`.
3. `cdc-cms-service/internal/router/router.go:316, 415`: Đăng ký 1 route đọc (`shared`) và 5 route điều khiển (`admin` + `destructiveChain`) dưới `/system/nats/*`.
4. `cdc-cms-service/internal/api/system/nats_monitor_handler.go`: Mở rộng struct `NATSMonitorHandler` và thêm các method quản trị NATS. Giữ nguyên tương thích ngược 100% cho `NewNATSMonitorHandler(url, logger)` và các trường cũ của `GET /api/system/nats/overview`.
5. `cdc-cms-service/internal/api/system/nats_monitor_handler_test.go`: Bổ sung unit test cho `governance`, `GetStreamMessages` (kể cả empty stream & sparse gaps), `PurgeStream`, `UpdateStreamConfig`, `DeleteStream` (chặn xóa Core Stream), `DeleteConsumer` (chặn xóa khi `PushBound == true`).
6. `cdc-cms-web/src/App.tsx:16-38, 139-143`: Uncomment menu `/nats-monitor`.
7. `cdc-cms-web/src/pages/NatsMonitor.tsx`: Nâng cấp giao diện trang `/nats-monitor` (tái sử dụng `ConfirmDestructiveModal`).

### 3.2. Rollback Plan
- Trước khi sửa code tại S4, tạo local git restore-point.
- Nếu cần rollback: `git checkout` lại các file trên về commit trước đó. Không có thay đổi schema DB (Zero DB Migration).

---

## 4. Code Demo

### 4.1. Backend Demo (`cdc-cms-service/internal/api/system/nats_monitor_handler.go`)

```go
// DeleteConsumer blocks deletion if the consumer is currently bound to active push workers or waiting pull calls.
func (h *NATSMonitorHandler) DeleteConsumer(c *fiber.Ctx) error {
	if h.natsClient == nil || h.natsClient.JS == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "JetStream client is not available on CMS server",
		})
	}
	streamName := strings.TrimSpace(c.Params("stream"))
	consumerName := strings.TrimSpace(c.Params("consumer"))
	if streamName == "" || consumerName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "stream and consumer parameters are required",
		})
	}
	info, err := h.natsClient.JS.ConsumerInfo(streamName, consumerName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("consumer %s on stream %s not found: %v", consumerName, streamName, err),
		})
	}
	if info.PushBound || info.NumWaiting > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("consumer %s is actively bound to worker subscriptions (push_bound=%v, num_waiting=%d) and cannot be deleted while active", consumerName, info.PushBound, info.NumWaiting),
		})
	}
	if err := h.natsClient.JS.DeleteConsumer(streamName, consumerName); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to delete consumer %s on stream %s: %v", consumerName, streamName, err),
		})
	}
	return c.JSON(fiber.Map{
		"status":   "deleted",
		"stream":   streamName,
		"consumer": consumerName,
		"message":  fmt.Sprintf("Đã xóa consumer %s trên stream %s thành công", consumerName, streamName),
	})
}
```

### 4.2. Frontend Demo (`cdc-cms-web/src/pages/NatsMonitor.tsx`)

```tsx
export default function NatsMonitor() {
  // Toàn bộ React Hooks được khai báo vô điều kiện ở đầu hàm (Tuân thủ tuyệt đối Rules of Hooks - L013)
  const [pollInterval, setPollInterval] = useState<number>(5000);
  const [connSearch, setConnSearch] = useState<string>('');
  const [cmdSearch, setCmdSearch] = useState<string>('');
  const [cmdGroupFilter, setCmdGroupFilter] = useState<string>('ALL');
  const [selectedStream, setSelectedStream] = useState<NatsStreamDetail | null>(null);
  const [consumerDrawerOpen, setConsumerDrawerOpen] = useState<boolean>(false);
  const [msgInspectorOpen, setMsgInspectorOpen] = useState<boolean>(false);
  const [inspectStreamName, setInspectStreamName] = useState<string>('');
  const [inspectMode, setInspectMode] = useState<'latest' | 'first' | 'seq' | 'subject'>('latest');
  const [inspectSeq, setInspectSeq] = useState<number>(1);
  const [inspectSubject, setInspectSubject] = useState<string>('');
  const [quotaModalStream, setQuotaModalStream] = useState<NatsStreamDetail | null>(null);
  const [destructiveTarget, setDestructiveTarget] = useState<DestructiveTargetState | null>(null);
  const [rawModalOpen, setRawModalOpen] = useState<boolean>(false);
  // ...
}
```

---

## 5. Bảng kết quả Decision Gate (`DG1`, `DG2` & `DG3` — Sau Adversarial Plan Audit)

### 5.1. Kết quả `DG1` (`06_laya_summary_dg1_20261019T165839.json` — Gate: `PASS`)
| Claim ID | Loại | Nội dung Claim | R1 | R2 | R3 | Quyết định | Giải trình `REF:` |
| :--- | :---: | :--- | :---: | :---: | :---: | :---: | :--- |
| `S-RC` | `solution` | Struct `NATSMonitorHandler` hiện tại chỉ lưu `natsMonitorURL`, `httpClient` và `logger` mà không có trường `natsClient` để tương tác với JetStream. | `CORRECT` (`0.665`) | `INCORRECT` (`0.0856`) | `YES` (`0.1407`) | `FLAG -> RESOLVED` | `REF: nats_monitor_handler.go:16-33` |
| `S-FIX` | `solution` | Struct `NatsClient` trong `cdc-cms-service` gồm hai trường là `Conn` kiểu `*nats.Conn` và `JS` kiểu `nats.JetStreamContext`. | `UNSUPPORTED` (`0.6241`) | `INCORRECT` (`0.3136`) | `YES` (`0.8059`) | `FLAG -> RESOLVED` | `REF: cdc-cms-service/pkgs/natsconn/nats_client.go:14-17` |
| `S-IMPACT` | `solution` | Hàm `Start` trong `transmute_consumer_pool.go` khởi tạo `consumerCfg` với `DeliverSubject` là `nats.NewInbox()` và tạo vòng lặp `WorkerCount` subscriptions. | `CORRECT` (`0.3903`) | `INCORRECT` (`0.321`) | `YES` (`0.9175`) | `FLAG -> RESOLVED` | `REF: transmute_consumer_pool.go:110-147` |
| `S-PATTERN` | `solution` | Trong `router.go` dòng 132, biến `destructiveChain` được tạo bằng `chain(middleware.RequireOpsAdmin(), destructive.Idempotency, destructive.Audit)`. | `CORRECT` (`0.77`) | `CORRECT` (`0.3953`) | `YES` (`0.965`) | `FLAG -> RESOLVED` | `REF: cdc-cms-service/internal/router/router.go:130-134` |

### 5.2. Kết quả `DG2` (`06_laya_summary_dg2_20261019T165957.json` — Gate: `PASS`)
| Claim ID | Loại | Nội dung Claim | R1 | R2 | R3 | Quyết định | Giải trình `REF:` |
| :--- | :---: | :--- | :---: | :---: | :---: | :---: | :--- |
| `D-BEHAV` | `demo` | Hàm `GetOverview` trong `nats_monitor_handler.go` khởi tạo context với timeout 3 giây và dùng `sync.WaitGroup` chờ 3 goroutine lấy `/varz`, `/jsz` và `/connz`. | `CORRECT` (`0.2072`) | `CORRECT` (`0.5648`) | `YES` (`0.9364`) | `ACCEPT` | `REF: nats_monitor_handler.go:36-63` |
| `D-EDGE` | `demo` | Hàm `deriveTarget` trong `audit.go` hiện tại chỉ kiểm tra các tham số URL là `table`, `id`, `name` và trường `collection` trong body JSON. | `CORRECT` (`0.8198`) | `CORRECT` (`0.3762`) | `YES` (`0.8572`) | `ACCEPT` | `REF: cdc-cms-service/internal/middleware/audit.go:325-346` |
| `D-API` | `demo` | Component `NatsMonitor` trên frontend khai báo các state bằng `useState` ở đầu hàm và gọi `cmsApi.get('/api/system/nats/overview')`. | `CORRECT` (`0.6524`) | `CORRECT` (`0.5761`) | `YES` (`0.9204`) | `ACCEPT` | `REF: cdc-cms-web/src/pages/NatsMonitor.tsx:237-260` |

### 5.3. Kết quả `DG3` (`06_laya_summary_dg3_20261019T165845.json` — Gate: `PASS`)
| Claim ID | Loại | Nội dung Claim | R1 | R2 | R3 | Quyết định | Giải trình `REF:` |
| :--- | :---: | :--- | :---: | :---: | :---: | :---: | :--- |
| `A-F1` | `audit` | Trong file `App.tsx` dòng 139-143, mục menu có key `'/nats-monitor'` đang bị comment out. | `CORRECT` (`0.7057`) | `CORRECT` (`0.1626`) | `YES` (`0.8323`) | `FLAG -> RESOLVED` | `REF: cdc-cms-web/src/App.tsx:135-145` |
| `A-R1` | `audit` | Hàm `defaultActionMap` trong `audit.go` dòng 306-316 định nghĩa map ánh xạ đường dẫn API sang tên hành động audit. | `INCORRECT` (`0.0048`) | `INCORRECT` (`0.3588`) | `YES` (`0.8329`) | `FLAG -> RESOLVED` | `REF: cdc-cms-service/internal/middleware/audit.go:306-316` |
| `A-SCOPE` | `audit` | Tại `server.go` dòng 326, `h.System.NATSMonitor` đang được khởi tạo bằng `apisystem.NewNATSMonitorHandler(cfg.System.NatsMonitorURL, logger)` chưa truyền `natsClient`. | `CORRECT` (`0.6974`) | `CORRECT` (`0.5398`) | `YES` (`0.8921`) | `ACCEPT` | `REF: cdc-cms-service/internal/server/server.go:324-327` |

---

## 6. Đề xuất Nâng cấp Bổ sung (Propose-Only): Xóa Điểm mù Hiển thị cho Stream `WorkQueuePolicy` (`CDC_TRANSMUTE`) & Làm rõ Kiến trúc `CDC_EVENTS`

Để người vận hành nhìn vào `/nats-monitor` thấy ngay **luồng `CDC_TRANSMUTE` vừa chạy thật lúc `10:32:16` / `10:40:16` (`#204 msgs`)** thay vì tưởng `0 msgs / 0 B` là chết/mockup, và hiểu rõ tại sao `CDC_EVENTS` không tăng khi `kafka-consumer` chạy:

### 6.1. Cải tiến 1 — Hiển thị đầy đủ `Đã xử lý (LastSeq)` & `Hoạt động cuối (LastTs)` ngay cả khi `messages == 0` (`cdc-cms-web/src/pages/NatsMonitor.tsx:935-1039`)
1. **Cột `Messages & Sequence` (`NatsMonitor.tsx:935-967`):**
   - Thay vì ẩn `last_seq` và `last_ts` khi `msgs === 0`, hiển thị tách bạch:
     - **Hàng 1 (Tồn đọng / Pending):** `0 pending` (màu xanh khi `retention === 'workqueue' && last > 0`, vì `WorkQueue` xóa ngay sau khi ACK).
     - **Hàng 2 (Tổng bản tin đã qua Stream):** `Đã qua stream: #204 msgs` (`last_seq`) — tăng tức thì mỗi khi anh update 1 record!
     - **Hàng 3 (Thời điểm bản tin cuối):** `Cuối: 20/10/2026 10:40:16` (`record.state?.last_ts`) — luôn hiển thị kể cả khi `msgs === 0`!
2. **Cột `Consumers & Backpressure` (`NatsMonitor.tsx:1005-1039`):**
   - Bổ sung dòng hiển thị trực tiếp tổng số bản tin Consumer đã xử lý thành công:
     - `Đã ACK: #204 (Delivered: 212)` kèm thời gian `Active: 10:40:16` (lấy từ `consumer_detail[].ack_floor.stream_seq`, `delivered.consumer_seq`, `delivered.last_active`).
3. **Drawer `Soi Msg (Message Inspector)` (`NatsMonitor.tsx:2150-2157`):**
   - Khi `msgs === 0` nhưng `last_seq > 0` trên stream `workqueue`, hiển thị hộp giải thích rõ:
     - *"Stream `CDC_TRANSMUTE` sử dụng `WorkQueuePolicy` (tự động xóa bản tin khỏi đĩa ngay khi Worker ACK thành công sau ~25–39ms). Tổng số bản tin đã đi qua luồng: **#204** (bản tin gần nhất lúc **20/10/2026 10:40:16**)."*

### 6.2. Cải tiến 2 — Ghi chú rõ kiến trúc thực tế của `CDC_EVENTS` (`cdc-cms-service/internal/api/system/nats_monitor_handler.go:54-60`)
- Cập nhật `Role` của `CDC_EVENTS` tại `nats_monitor_handler.go:57` để phản ánh đúng kiến trúc hiện tại của `centralized-data-service` (`kafka_consumer.go:748` đang gọi trực tiếp `kc.eventHandler.HandleRaw` qua `cdcBatchBuffer` trong RAM, nên `CDC_EVENTS` chỉ đóng vai trò hàng đợi dự phòng / NATS ingest bridge khi bật publisher riêng).
