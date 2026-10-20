# 12. Implementation Plan (Line-by-Line Spec) — Nâng cấp `/nats-monitor` (`nats_monitor_v2`)

> **Trạng thái:** `PROPOSED` (Chờ User `APPROVE` trước khi thực thi pha Muscle S4)
> **Nguyên tắc thiết kế:** Simplicity First, Minimal Impact, tuân thủ 100% Architecture (`destructiveChain`, `ConfirmDestructiveModal`, `natsconn.NatsClient`) của `data-hub`.

---

## 1. Danh mục Tổng thể Các File & Dòng Code Cần Thực Hiện

| STT | Thành phần | File | Dòng hiện tại | Thay đổi cụ thể |
| :---: | :---: | :--- | :---: | :--- |
| 1 | Backend Middleware | `cdc-cms-service/internal/middleware/audit.go` | `L306-346` | Thêm 5 route NATS vào `defaultActionMap()` (`L306-316`) và bổ sung `"stream"`, `"consumer"`, `"seq"` vào `deriveTarget(c)` (`L325-346`). |
| 2 | Backend Server | `cdc-cms-service/internal/server/server.go` | `L326` | Truyền thêm `natsClient` vào `apisystem.NewNATSMonitorHandler(cfg.System.NatsMonitorURL, logger, natsClient)`. |
| 3 | Backend Router | `cdc-cms-service/internal/router/router.go` | `L316`, `L415` | Đăng ký route đọc `GET /system/nats/streams/:stream/messages` vào nhóm `shared` (`L316`) và 5 route điều khiển (`purge`, `config`, `delete msg`, `delete stream`, `delete consumer`) vào nhóm `admin` kèm `destructiveChain...` (`L415`). |
| 4 | Backend Handler | `cdc-cms-service/internal/api/system/nats_monitor_handler.go` | `L1-104` | Mở rộng `NATSMonitorHandler` với `natsClient *natsconn.NatsClient`, nâng cấp `GetOverview` (`/connz?subs=detail` + `governance`) và thêm 6 handler methods (`GetStreamMessages`, `PurgeStream`, `DeleteStreamMessage`, `UpdateStreamConfig`, `DeleteStream`, `DeleteConsumer`). |
| 5 | Backend Test | `cdc-cms-service/internal/api/system/nats_monitor_handler_test.go` | `L1-118` | Bổ sung unit tests kiểm chứng `governance` (drift detection, command channels `subscriptions_list_detail`), bảo vệ Core Stream (`DeleteStream` trả `400`), bảo vệ Active Consumer (`DeleteConsumer` khi `PushBound`), và `GetStreamMessages`. |
| 6 | Frontend Sidebar | `cdc-cms-web/src/App.tsx` | `L16-38`, `L139-143` | Import `CloudServerOutlined` và uncomment menu `/nats-monitor` (`Giám sát NATS`). |
| 7 | Frontend Page | `cdc-cms-web/src/pages/NatsMonitor.tsx` | `L1-1044` | Nâng cấp giao diện thành NATS Control Center (Alerts Banner, Stream Governance & Quota Bar, Message Inspector Drawer, Consumer Progress & In-Flight Window Drawer, Command Channels Matrix, tích hợp `ConfirmDestructiveModal`). |

---

## 2. Đặc tả Code Chi Tiết Từng File (`Line-by-Line`)

### 2.1. File 1: `cdc-cms-service/internal/middleware/audit.go` (`L306-346`)
- **Vị trí sửa:** `defaultActionMap()` (`L306-316`) và `deriveTarget(c *fiber.Ctx)` (`L325-346`).
- **Code thay thế chính xác:**
```go
func defaultActionMap() map[string]string {
	return map[string]string{
		"/api/reconciliation/heal/:table":                      "heal",
		"/api/reconciliation/check":                            "recon_check_all",
		"/api/reconciliation/check/:table":                     "recon_check",
		"/api/failed-sync-logs/:id/retry":                      "retry_failed",
		"/api/tools/reset-debezium-offset":                     "reset_offset",
		"/api/tools/trigger-snapshot/:table":                   "trigger_snapshot",
		"/api/tools/restart-debezium":                          "restart_connector",
		"/api/system/nats/streams/:stream/purge":               "nats_stream_purge",
		"/api/system/nats/streams/:stream/config":              "nats_stream_config_update",
		"/api/system/nats/streams/:stream/messages/:seq":       "nats_message_delete",
		"/api/system/nats/streams/:stream":                     "nats_stream_delete",
		"/api/system/nats/streams/:stream/consumers/:consumer": "nats_consumer_delete",
	}
}

// deriveTarget pulls the most specific URL segment that identifies the
// resource being acted on. Preference order:
//  1. :table param (heal, snapshot, check)
//  2. :id param    (retry)
//  3. :name param  (connector)
//  4. :stream param (+ optional :consumer or :seq for NATS control plane)
//  5. body.collection (reset-offset)
//  6. empty string
func deriveTarget(c *fiber.Ctx) string {
	if t := c.Params("table"); t != "" {
		return t
	}
	if id := c.Params("id"); id != "" {
		return id
	}
	if n := c.Params("name"); n != "" {
		return n
	}
	if stream := c.Params("stream"); stream != "" {
		if consumer := c.Params("consumer"); consumer != "" {
			return stream + "/" + consumer
		}
		if seq := c.Params("seq"); seq != "" {
			return stream + "#" + seq
		}
		return stream
	}
	// Best-effort body peek for reset-offset.
	var body struct {
		Collection string `json:"collection"`
		Database   string `json:"database"`
	}
	if err := json.Unmarshal(c.Body(), &body); err == nil {
		if body.Collection != "" {
			return body.Database + "." + body.Collection
		}
	}
	return ""
}
```

---

### 2.2. File 2: `cdc-cms-service/internal/server/server.go` (`L325-327`)
- **Hiện tại (`L325-327`):**
```go
	systemDiagHandler := apisystem.NewDiagnosticsHandler(db, pgPool, redisClient, esClient, natsClient, cfg.System.ConnectURL, cfg.System.PrometheusURL)
	natsMonitorHandler := apisystem.NewNATSMonitorHandler(cfg.System.NatsMonitorURL, logger)
```
- **Code cập nhật (`L325-327`):**
```go
	systemDiagHandler := apisystem.NewDiagnosticsHandler(db, pgPool, redisClient, esClient, natsClient, cfg.System.ConnectURL, cfg.System.PrometheusURL)
	natsMonitorHandler := apisystem.NewNATSMonitorHandler(cfg.System.NatsMonitorURL, logger, natsClient)
```

---

### 2.3. File 3: `cdc-cms-service/internal/router/router.go` (`L218-220` & `L315-318`)
- **Tại nhóm `4.1 DESTRUCTIVE ROUTES (OPS-ADMIN ONLY)` (`L218-220`, bảo vệ bằng `destructiveChain` gồm `RequireOpsAdmin`, `Idempotency`, `Audit`, hỗ trợ cả `/api/system/nats/*` và `/api/v1/system/nats/*`):**
```go
	// NATS JetStream Control Plane (OpsAdmin + Idempotency + Audit)
	if h.System.NATSMonitor != nil {
		registerDestructive("/system/nats/streams/:stream/purge", h.System.NATSMonitor.PurgeStream)
		api.Patch("/system/nats/streams/:stream/config", append(destructiveChain, h.System.NATSMonitor.UpdateStreamConfig)...)
		api.Patch("/v1/system/nats/streams/:stream/config", append(destructiveChain, h.System.NATSMonitor.UpdateStreamConfig)...)
		api.Delete("/system/nats/streams/:stream/messages/:seq", append(destructiveChain, h.System.NATSMonitor.DeleteStreamMessage)...)
		api.Delete("/v1/system/nats/streams/:stream/messages/:seq", append(destructiveChain, h.System.NATSMonitor.DeleteStreamMessage)...)
		api.Delete("/system/nats/streams/:stream", append(destructiveChain, h.System.NATSMonitor.DeleteStream)...)
		api.Delete("/v1/system/nats/streams/:stream", append(destructiveChain, h.System.NATSMonitor.DeleteStream)...)
		api.Delete("/system/nats/streams/:stream/consumers/:consumer", append(destructiveChain, h.System.NATSMonitor.DeleteConsumer)...)
		api.Delete("/v1/system/nats/streams/:stream/consumers/:consumer", append(destructiveChain, h.System.NATSMonitor.DeleteConsumer)...)
	}
```
- **Tại nhóm `4.2 SHARED ROUTES (ADMIN + OPERATOR)` (`L315-318`, dùng helper `dual` có sẵn cho phép cả `admin` và `operator` xem overview & soi message read-only):**
```go
	// System / Activity / Recon reads
	dual("GET", shared, "/system/nats/overview", h.System.NATSMonitor.GetOverview)
	dual("GET", shared, "/system/nats/streams/:stream/messages", h.System.NATSMonitor.GetStreamMessages)
```

---

### 2.4. File 4: `cdc-cms-service/internal/api/system/nats_monitor_handler.go`
- **Cấu trúc các hàm bổ sung trong `nats_monitor_handler.go`:**
  1. **Quy chuẩn 4 Core Streams (`canonicalCoreStreams`):**
     - `CDC_EVENTS`: `retention = "limits"`, `max_bytes = 512 MiB`, `max_age_ns = 24h`
     - `CDC_TRANSMUTE`: `retention = "workqueue"`, `max_bytes = 256 MiB`, `max_age_ns = 1h`
     - `SCHEMA_DRIFT`: `retention = "limits"`, `max_bytes = 64 MiB`, `max_age_ns = 7d`
     - `SCHEMA_CONFIG`: `retention = "limits"`, `max_bytes = 64 MiB`, `max_age_ns = 7d`
  2. **Danh mục 19 Kênh lệnh Core NATS bắt buộc (`canonicalCommandChannels`):**
     - Đối chiếu với mảng `subscriptions_list_detail` trả về từ `/connz?subs=detail`.
     - Tính số lượng `active_listeners`, danh sách `worker_cids`, `msgs_delivered` và trạng thái `READY` / `NO_LISTENER`.
  3. **`GetStreamMessages(c *fiber.Ctx) error` (`GET /api/v1/system/nats/streams/:stream/messages`):**
     - Hỗ trợ `mode`: `latest` (mặc định), `first`, `seq`, `subject`, `limit` (`1..50`, mặc định `10`).
     - Dùng `context.WithTimeout(c.UserContext(), 2500*time.Millisecond)`.
     - Nếu `info.State.Messages == 0 || info.State.FirstSeq > info.State.LastSeq`: trả về `items: []` ngay lập tức (Fix `F4`).
     - Với `mode=latest` / `mode=first`: duyệt tối đa `maxProbes = limit * 4` (cap ở `60`) để vượt qua các sparse gaps trên `WorkQueuePolicy`, đọc bằng `JS.GetMsg(streamName, seq, nats.Context(ctx))` (không tiêu thụ message). Cắt payload preview tối đa `16KB` kèm cờ `truncated: true/false`.
  4. **`PurgeStream(c *fiber.Ctx) error` (`POST /api/v1/system/nats/streams/:stream/purge`):**
     - Đọc body `{"subject": string, "keep": uint64, "seq": uint64}`.
     - Chặn `400 Bad Request` nếu `req.Keep > 0 && req.Seq > 0` (Fix `F8`).
     - Gọi `h.natsClient.JS.PurgeStream(streamName, &nats.StreamPurgeRequest{Subject: req.Subject, Keep: req.Keep, Sequence: req.Seq})`.
  5. **`DeleteStreamMessage(c *fiber.Ctx) error` (`DELETE /api/v1/system/nats/streams/:stream/messages/:seq`):**
     - Parse `:seq` (`uint64 > 0`), gọi `h.natsClient.JS.DeleteMsg(streamName, seq)`.
  6. **`UpdateStreamConfig(c *fiber.Ctx) error` (`PATCH /api/v1/system/nats/streams/:stream/config`):**
     - Đọc body `{"max_bytes": int64, "max_age_seconds": int64}`.
     - Validate `max_bytes >= 1048576` (`>= 1 MiB`) và `max_age_seconds >= 60` (`>= 1 phút`).
     - Lấy `info, err := h.natsClient.JS.StreamInfo(streamName)`, gán `cfg := info.Config`, cập nhật `cfg.MaxBytes` và `cfg.MaxAge`, rồi gọi `h.natsClient.JS.UpdateStream(&cfg)`. Giữ nguyên 100% `Subjects`, `Retention`, `Storage` (Fix `F1`).
  7. **`DeleteStream(c *fiber.Ctx) error` (`DELETE /api/v1/system/nats/streams/:stream`):**
     - Kiểm tra `if _, isCore := canonicalCoreStreams[streamName]; isCore` $\rightarrow$ trả về `400 Bad Request` (không cho xóa 4 Core Streams).
     - Với stream `ORPHAN`, gọi `h.natsClient.JS.DeleteStream(streamName)`.
  8. **`DeleteConsumer(c *fiber.Ctx) error` (`DELETE /api/v1/system/nats/streams/:stream/consumers/:consumer`):**
     - Gọi `info, err := h.natsClient.JS.ConsumerInfo(streamName, consumerName)` trước.
     - Nếu `info.PushBound || info.NumWaiting > 0` $\rightarrow$ trả về `400 Bad Request` (`"Consumer đang có worker lắng nghe (active subscription), không được xóa để tránh làm đứt kết nối worker"` — Fix `F1`).
     - Chỉ xóa khi consumer thực sự không còn kết nối lắng nghe (`!info.PushBound && info.NumWaiting == 0`).

---

### 2.5. File 5: `cdc-cms-web/src/App.tsx` (`L16-38` & `L139-143`)
- **Bổ sung import icon (`L16-38`):** Thêm `CloudServerOutlined` từ `@ant-design/icons`.
- **Uncomment menu item (`L139-143`):**
```tsx
      {
        key: '/nats-monitor',
        icon: <CloudServerOutlined />,
        label: <Link to="/nats-monitor">Giám sát NATS</Link>,
      },
```

---

### 2.6. File 6: `cdc-cms-web/src/pages/NatsMonitor.tsx`
- **Tích hợp `ConfirmDestructiveModal` (`src/components/ConfirmDestructiveModal.tsx`):**
  - Dùng chung 1 state `destructiveAction: { title, description, severity, confirmText, requireTypeMatch, onConfirm } | null` cho 4 thao tác:
    1. **Purge Stream** (`requireTypeMatch: false` khi purge theo subject, hoặc `requireTypeMatch: true` với `confirmText = stream.name` khi purge toàn bộ stream).
    2. **Xóa Poison Message (`#seq`)** (`requireTypeMatch: false`, nhập lý do $\ge 10$ ký tự).
    3. **Xóa Orphan Stream** (`requireTypeMatch: true`, `confirmText = stream.name`).
    4. **Xóa Consumer Mồ côi** (`requireTypeMatch: true`, `confirmText = consumer.name`).
  - Modal **Cập nhật Quota (`MaxBytes` & `MaxAge`)** có trường nhập `Lý do thay đổi (>= 10 ký tự)` và gửi kèm headers `'Idempotency-Key'` + `'X-Action-Reason'`.
- **Nâng cấp 3 Tabs chính + 2 Drawers chuyên sâu:**
  1. **Tab 1 — `JetStream Streams & Governance`:**
     - Hiển thị badge `CORE` / `ORPHAN`, vai trò nghiệp vụ, badge `WORKQUEUE` / `LIMITS`, trạng thái `SYNCED` 🟢 / `DRIFT` 🟠 (kèm Tooltip chi tiết `drift_reasons`).
     - Cột **Quota Usage**: thanh Progress `bytes / max_bytes` (hiển thị `UNLIMITED` màu cam nếu `max_bytes <= 0`).
     - Cột **Sequence Range**: xử lý chuẩn trường hợp `messages === 0` hiển thị `Trống (Next: #first_seq)` thay vì `#83 → #82` và ẩn `1970-01-01` (Fix `F7`).
     - Các nút thao tác: **Soi Message (Peek)**, **Consumers**, **Chỉnh Quota**, **Purge**, **Xóa Stream** (chỉ bật cho `ORPHAN`).
  2. **Tab 2 — `Kênh lệnh Data-Hub (cdc.cmd.*)`:**
     - Bảng đối soát 19 kênh lệnh Core NATS bắt buộc của hệ thống (lọc theo nhóm nghiệp vụ & trạng thái `READY` / `NO_LISTENER`), hiển thị rõ `Queue Group`, số lượng `Active Listeners`, `Worker CIDs`, và tổng số message đã giao (`msgs_delivered`).
  3. **Tab 3 — `Active Connections & Subscriptions`:**
     - Hiển thị danh sách kết nối kèm `RTT`, phân loại role (`Worker`, `CMS`, `Other`) và cho phép mở rộng (`expandable`) để xem chi tiết bảng `subscriptions_list_detail` (`subject`, `qgroup`, `sid`, `msgs`) của từng kết nối.
  4. **Drawer 1 — `Consumer Diagnostics & Control`:**
     - Hiển thị trạng thái `PushBound` (`Đang gắn Worker` 🟢 vs `Mồ côi / Ngắt kết nối` 🟠), `DeliverGroup`, tiến độ `AckFloor → Delivered / StreamLastSeq`, thanh bão hòa cửa sổ `AckPending / MaxAckPending`, `AckWait (s)`, `MaxDeliver`, `LastActive`, và nút **Xóa Consumer** (chỉ cho phép khi `push_bound === false`).
  5. **Drawer 2 — `JetStream Message Inspector (Read-Only Peek)`:**
     - Chế độ tra cứu: `10 tin mới nhất`, `10 tin đầu hàng đợi (cũ nhất)`, `Theo Sequence #`, `Tin cuối theo Subject`.
     - Hiển thị bảng message kèm `Seq`, `Subject`, `Timestamp`, `Size`, xem chi tiết `Headers` & `Payload JSON` đã format, và nút **Xóa Message (`#seq`)**.
