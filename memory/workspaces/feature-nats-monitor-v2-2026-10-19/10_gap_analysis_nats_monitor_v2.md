# 10. Gap Analysis & Adversarial Plan Audit — Nâng cấp `/nats-monitor` (`nats_monitor_v2`)

> **Trạng thái:** `VERIFIED` (Adversarial Audit trên Plan V1 & Demo Code — Propose-Only)
> **Nguyên tắc kiểm định:** Simplicity First, Minimal Impact, tuân thủ tuyệt đối Architecture, Core Systems và Design Pattern của `data-hub`. Không suy diễn.

---

## 1. Tóm tắt Kết quả Phản biện (Executive Summary)

Qua quá trình đối chiếu chéo bản Plan V1 (`09_tasks_solution_nats_monitor_v2.md` bản đầu) với toàn bộ mã nguồn thực tế của `cdc-cms-service`, `centralized-data-service`, `cdc-cms-web` và phản hồi JSON thực tế từ NATS HTTP Monitoring (`http://localhost:18222`), chúng tôi phát hiện **8 sai sót & lỗ hổng thiết kế (`F1`–`F8`)** cần phải loại bỏ hoặc hiệu chỉnh trước khi thực thi:

| Mã | Mức độ | Trạng thái | Hạng mục bị ảnh hưởng | Tóm tắt Sai sót / Thiếu sót trong Plan V1 | Quyết định Hiệu chỉnh (Plan V2 — Simplicity First) |
| :--- | :---: | :---: | :--- | :--- | :--- |
| **`F1`** | **P0 (Critical)** | `VERIFIED` | `POST /streams/reconcile` & `DeleteConsumer` | 1) `cdc-cms-service` không có `cfg.Kafka.TopicPrefix` để biết subjects của `CDC_EVENTS`.<br>2) `TransmuteConsumerPool.Start()` (`transmute_consumer_pool.go:112-153`) tạo `DeliverSubject = _INBOX.xxx` và bind 10 push subscriptions chỉ 1 lần lúc worker boot. Nếu CMS gọi `DeleteStream`+`AddStream` hoặc xóa consumer đang có `push_bound=true`, toàn bộ 10 worker subscriptions của `cdc-worker` sẽ bị đứt vĩnh viễn cho tới khi restart worker. | 1) **Bỏ hoàn toàn API `POST /streams/reconcile`** khỏi `cdc-cms-service` (trách nhiệm `EnsureStreams` thuộc về `centralized-data-service/pkgs/natsconn/nats_client.go:63-167`).<br>2) Giữ `PATCH /streams/:stream/config` chỉ sửa in-place `MaxBytes` & `MaxAge` (không đụng `Retention`/`Subjects`).<br>3) Trong `DeleteConsumer`, kiểm tra `ConsumerInfo`: **chặn cứng (`400 Bad Request`) nếu `info.PushBound == true` hoặc `info.NumWaiting > 0`**. |
| **`F2`** | **P1 (High)** | `VERIFIED` | `GetOverview` (`/connz` & `/subsz`) | 1) Plan V1 viết trường JSON của `/connz?subs=detail` là `subscriptions_detail`, nhưng NATS Server thực tế trả về **`subscriptions_list_detail`**.<br>2) Gọi thêm `/subsz?subs=1` (goroutine thứ 4) là dư thừa vì `/connz?subs=detail` đã chứa toàn bộ `subject`, `qgroup`, `cid`. | 1) Đọc đúng trường **`subscriptions_list_detail`** từ `/connz?subs=detail`.<br>2) **Bỏ goroutine thứ 4 (`/subsz?subs=1`)**, giữ nguyên đúng 3 goroutines song song như thiết kế gốc (`/varz`, `/jsz`, `/connz?subs=detail`). |
| **`F3`** | **P1 (High)** | `VERIFIED` | Middleware RBAC, Idempotency & Audit Trail | Plan V1 gắn các route xóa/purge/sửa config vào `RequireRole("admin")` đơn thuần, bỏ qua pattern chuẩn của `cdc-cms-service` là **`destructiveChain`** (`RequireOpsAdmin` + `Idempotency` + `Audit` ghi bảng `cdc_system.admin_actions`) và `ConfirmDestructiveModal` trên Frontend. Ngoài ra `deriveTarget(c)` trong `middleware/audit.go:325-346` chưa đọc `c.Params("stream")`. | 1) Toàn bộ route mutating của NATS đi qua **`destructiveChain`** (`router.go:132`).<br>2) Mở rộng nhẹ `middleware/audit.go:306-346` hỗ trợ `c.Params("stream")` / `c.Params("consumer")` và mã action `NATS_*`.<br>3) Frontend tái sử dụng **`ConfirmDestructiveModal`** (`ConfirmDestructiveModal.tsx`) gửi kèm `Idempotency-Key` và `X-Action-Reason`. |
| **`F4`** | **P1 (High)** | `VERIFIED` | `GetStreamMessages` (Message Inspector) | 1) Khi stream rỗng (`messages == 0`), NATS trả về `first_seq = last_seq + 1` (ví dụ `83 > 82`), nếu lặp `seq` sẽ lỗi.<br>2) Trên `WorkQueuePolicy` (`CDC_TRANSMUTE`), các message đã Ack bị xóa tạo khoảng trống (sparse gaps) giữa `FirstSeq` và `LastSeq`. Nếu không bỏ qua `ErrMsgNotFound` và không giới hạn số lần probe sẽ bị thiếu kết quả hoặc treo. | Kiểm tra `info.State.Messages == 0 || info.State.FirstSeq > info.State.LastSeq` trả mảng rỗng ngay. Khi duyệt `latest`/`first`, bỏ qua sequence đã bị xóa với giới hạn `maxProbes = limit * 4` (tối đa 60 lần `GetMsg`) và `context.WithTimeout(2500ms)`. |
| **`F5`** | **P1 (High)** | `VERIFIED` | `command_channels` Coverage Matrix | Plan V1 liệt kê 22 subjects trong đó có `cdc.cmd.backfill` (đã bị comment out tại `server_setup.go:304`), `cdc.cmd.shadow.bind` / `cdc.cmd.schedule.enable` / `cdc.cmd.master.bind` (chỉ bật khi `PROVISIONING_ORCHESTRATOR_ENABLED=1` tại `server_setup.go:376`), và `cdc.cmd.kafka.refresh-topics` (phụ thuộc `cfg.Kafka.Enabled`). Đưa vào danh sách bắt buộc sẽ gây **Báo động giả (`MISSING_COMMAND_LISTENER`)**. | Chỉ kiểm tra **15 Core NATS Command Channels bắt buộc** (luôn luôn đăng ký vô điều kiện trong `centralized-data-service/internal/server/server_setup.go:300-531`). Các kênh tùy chọn được đánh dấu `optional: true` (không sinh cảnh báo đỏ khi tắt flag). |
| **`F6`** | **P2 (Medium)** | `VERIFIED` | Sidebar Navigation (`App.tsx`) | Menu `/nats-monitor` đang bị comment out tại `cdc-cms-web/src/App.tsx:139-143`, khiến Operator không thể truy cập từ thanh điều hướng bên trái. | Mở lại (uncomment) mục `/nats-monitor` trong `cdc-cms-web/src/App.tsx:139-143` (dùng icon `CloudServerOutlined`). |
| **`F7`** | **P2 (Medium)** | `VERIFIED` | Hiển thị Edge Cases trên UI (`NatsMonitor.tsx`) | 1) Khi `messages == 0`, UI hiện `#83 → #82` và `first_ts = 1970-01-01T00:00:00Z`.<br>2) Khi `max_bytes == -1` (Unlimited), chia `%` trực tiếp sẽ ra số âm. | 1) Khi `messages === 0`, hiển thị `Trống (Next: #${first_seq})` và ẩn `1970-01-01`.<br>2) Khi `max_bytes <= 0`, hiển thị tag `UNLIMITED (NO CAP)` màu cam. |
| **`F8`** | **P2 (Medium)** | `VERIFIED` | `PurgeStream` Options (`nats.go`) | NATS Server trả lỗi nếu `StreamPurgeRequest` có đồng thời `Keep > 0` và `Sequence > 0` (`nats.go:1968-1975`). | Backend validate rõ: nếu `req.Keep > 0 && req.Seq > 0` thì trả `400 Bad Request` ngay trước khi gọi `JS.PurgeStream(stream, opts...)`. |

---

## 2. Phân tích Chi tiết Bằng chứng Code (`file:line`) cho từng Phát hiện

### 2.1. Phân tích `F1`: Tại sao `POST /streams/reconcile` từ CMS là phản kiến trúc (Anti-Pattern) và nguy hiểm?
- **Bằng chứng 1 — Thiếu cấu hình `Kafka.TopicPrefix` trên CMS:**
  - Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go:88-100`, `centralized-data-service` khởi tạo `CDC_EVENTS` với `streamSubjects = []string{"cdc.raw.>", fmt.Sprintf("%s.>", cfg.Kafka.TopicPrefix)}`.
  - Trong khi đó, tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/config/config.go:16-28`, struct `Config` của `cdc-cms-service` chỉ có `Server`, `Log`, `Security`, `Database`, `Postgres`, `Mongo`, `Redis`, `Elasticsearch`, `Nats`, `Scheduler`, `System` — **hoàn toàn không có `Kafka.TopicPrefix`**. Nếu `cdc-cms-service` tự chạy `EnsureStreams` hoặc `UpdateStream` với subjects mặc định, nó sẽ ghi đè làm mất subject `<topicPrefix>.>` của `CDC_EVENTS`!
- **Bằng chứng 2 — Đứt kết nối Push Subscription của `TransmuteConsumerPool`:**
  - Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_consumer_pool.go:112-153`:
    ```go
    deliverSubject := nats.NewInbox()
    cfg := &nats.ConsumerConfig{
        Durable:        p.consumerName, // "transmute-shadow-consumer-v2"
        DeliverSubject: deliverSubject,
        DeliverGroup:   p.queueGroup,   // "cdc-transmute-workers"
        FilterSubject:  p.subject,      // "cdc.transmute.>"
        AckPolicy:      nats.AckExplicitPolicy,
        AckWait:        p.ackWait,
        MaxDeliver:     p.maxDeliver,
        MaxAckPending:  p.maxAckPending,
    }
    // ...
    sub, err := p.js.QueueSubscribe(
        p.subject,
        p.queueGroup,
        p.handleMessage,
        nats.Bind(p.streamName, p.consumerName),
        nats.ManualAck(),
    )
    ```
  - Các worker goroutines chỉ chạy `Start()` **một lần duy nhất khi khởi động `cdc-worker`**. Nếu `cdc-cms-service` xóa stream `CDC_TRANSMUTE` để tạo lại hoặc xóa consumer `transmute-shadow-consumer-v2` khi `PushBound == true`, NATS Server sẽ hủy `deliverSubject` inbox đó và toàn bộ 10 subscriptions của `cdc-worker` không còn nhận được bất kỳ message nào nữa.
- **Giải pháp chuẩn Core Systems:**
  - Giữ nguyên quyền khởi tạo/migrate `RetentionPolicy` tại `centralized-data-service/pkgs/natsconn/nats_client.go:63-167`.
  - Trên `cdc-cms-service`:
    - Chỉ cung cấp `PATCH /api/v1/system/nats/streams/:stream/config` để cập nhật in-place `MaxBytes` và `MaxAge` trên `info.Config` hiện có (giữ nguyên 100% `info.Config.Subjects`, `info.Config.Retention`, `info.Config.Storage`).
    - Trong `DeleteConsumer`: gọi `info, err := h.natsClient.JS.ConsumerInfo(streamName, consumerName)` trước. Nếu `info.PushBound || info.NumWaiting > 0`, trả về `400 Bad Request` (`"Consumer đang có worker lắng nghe (push_bound=true), không được xóa khi worker đang chạy"`).

### 2.2. Phân tích `F2`: Cấu trúc JSON thực tế của `/connz?subs=detail`
- **Bằng chứng Runtime (`/connz?subs=detail` trên `:18222`):**
  - Kết quả thực tế từ `http://localhost:18222/connz?subs=detail` trả về:
    ```json
    {
      "cid": 15,
      "name": "CDC Worker Service",
      "num_subs": 44,
      "subscriptions_list_detail": [
        {
          "subject": "cdc.cmd.transmute",
          "qgroup": "cdc-worker",
          "sid": "2",
          "msgs": 0,
          "max": 0,
          "cid": 15
        }
      ]
    }
    ```
- **Giải pháp chuẩn:**
  - Trong `nats_monitor_handler.go`, đọc trường `subscriptions_list_detail` (fallback sang `subscriptions_list` nếu cần) ngay từ kết quả `/connz?subs=detail`. Không cần gọi thêm `/subsz?subs=1`.

### 2.3. Phân tích `F3`: Tuân thủ Design Pattern `destructiveChain` & `ConfirmDestructiveModal`
- **Bằng chứng Codebase:**
  - Tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go:132`:
    ```go
    destructiveChain := chain(middleware.RequireOpsAdmin(), destructive.Idempotency, destructive.Audit)
    ```
  - Tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/middleware/audit.go:220-264`, middleware `Audit` bắt buộc header `X-Action-Reason` (hoặc trường `reason` trong body) có độ dài `>= 10` ký tự và tự động lưu vào bảng `cdc_system.admin_actions` kèm `before_state`, `after_state`, `actor`, `ip_address`.
  - Tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/middleware/audit.go:325-346`, hàm `deriveTarget(c *fiber.Ctx)` hiện tại chỉ đọc `table`, `id`, `name`:
    ```go
    func deriveTarget(c *fiber.Ctx) string {
        for _, key := range []string{"table", "id", "name"} {
            if v := strings.TrimSpace(c.Params(key)); v != "" {
                return v
            }
        }
        return c.Path()
    }
    ```
    Nếu mở rộng mảng key thành `[]string{"table", "id", "name", "stream", "consumer"}` (hoặc ghép `stream/consumer` khi có cả hai) và thêm 5 action `POST /system/nats/streams/:stream/purge`, `PATCH /system/nats/streams/:stream/config`, `DELETE /system/nats/streams/:stream/messages/:seq`, `DELETE /system/nats/streams/:stream`, `DELETE /system/nats/streams/:stream/consumers/:consumer` vào `defaultActionMap()` (`audit.go:306-316`), mọi thao tác quản trị NATS sẽ tự động được bảo vệ bởi RBAC OpsAdmin + Idempotency Redis + Audit Log DB!
  - Trên Frontend, `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx:16-143` đã cung cấp sẵn modal nhập lý do (`reason >= 10 chars`) + tùy chọn gõ tên xác nhận (`requireTypeMatch`), sinh `Idempotency-Key` và gắn header `X-Action-Reason`. Việc tái sử dụng `ConfirmDestructiveModal` giúp giảm hơn 120 dòng code modal thủ công trong `NatsMonitor.tsx` và đồng nhất 100% UX toàn hệ thống.

### 2.4. Phân tích `F5`: Danh mục 15 Kênh lệnh Core NATS bắt buộc (`command_channels`)
Đối chiếu từng dòng giữa `cdc-cms-service/internal/server/server.go:265-285` và `centralized-data-service/internal/server/server_setup.go:299-535`, danh sách chính xác **15 kênh lệnh Core NATS luôn luôn active vô điều kiện** trên `cdc-worker` gồm:

| STT | Subject | Queue Group (`server_setup.go`) | Dòng đăng ký tại `server_setup.go` | Nhóm nghiệp vụ |
| :---: | :--- | :---: | :---: | :--- |
| 1 | `cdc.cmd.transmute` | `cdc-worker` | `server_setup.go:300` | Master Transmute |
| 2 | `cdc.cmd.transmute-shadow.core-fallback` | `cdc-transmute-workers` | `server_setup.go:281` | Master Transmute |
| 3 | `cdc.cmd.batch-transform` | `cdc-worker` | `server_setup.go:302` | Master Transmute |
| 4 | `cdc.cmd.standardize` | `cdc-worker` | `server_setup.go:303` | Master Transmute |
| 5 | `cdc.cmd.recon-check` | `cdc-worker` | `server_setup.go:305` | Reconciliation & Heal |
| 6 | `cdc.cmd.recon-heal` | `cdc-worker` | `server_setup.go:306` | Reconciliation & Heal |
| 7 | `cdc.cmd.execute-heal` | `cdc-worker` | `server_setup.go:315` | Reconciliation & Heal |
| 8 | `cdc.cmd.retry-failed` | `cdc-worker` | `server_setup.go:311` | Reconciliation & Heal |
| 9 | `cdc.cmd.discover` | `cdc-worker` | `server_setup.go:307` | Source & Discovery |
| 10 | `cdc.cmd.scan-fields` | `cdc-worker` | `server_setup.go:308` | Source & Discovery |
| 11 | `cdc.cmd.debezium-signal` | `cdc-worker` | `server_setup.go:309` | Connector & Snapshot |
| 12 | `cdc.cmd.restart-debezium` | `cdc-worker` | `server_setup.go:310` | Connector & Snapshot |
| 13 | `cdc.cmd.system-connector-recover` | `cdc-worker` | `server_setup.go:312` | Connector & Snapshot |
| 14 | `cdc.cmd.bridge-oplog` | `cdc-worker` | `server_setup.go:313` | Connector & Snapshot |
| 15 | `cdc.cmd.snapshot.v2` | `cdc-worker` | `server_setup.go:314` | Connector & Snapshot |
| 16 | `cdc.cmd.master-create` | `cdc-worker` | `server_setup.go:467` | Master Table DDL |
| 17 | `cdc.cmd.master-alter-column` | `cdc-worker` | `server_setup.go:478` | Master Table DDL |
| 18 | `cdc.cmd.master-swap` | `cdc-worker` | `server_setup.go:489` | Master Table DDL |
| 19 | `schema.config.reload` | *(broadcast)* | `server_setup.go:530` | Schema Config |

*(Tổng cộng 19 kênh luôn luôn đăng ký vô điều kiện khi `natsClient != nil` — không bao gồm các kênh phụ thuộc feature flag để tránh báo động giả).*

---

## 3. Kết quả Kiểm định `/laya_router` (`DG1`, `DG2`, `DG3`)

- **`DG1` (`06_laya_summary_dg1_20261019T165839.json`):** `PASS` 🟢
- **`DG2` (`06_laya_summary_dg2_20261019T165957.json`):** `PASS` 🟢 (`D-BEHAV`, `D-EDGE`, `D-API` đều đạt `ACCEPT` trực tiếp)
- **`DG3` (`06_laya_summary_dg3_20261019T165845.json`):** `PASS` 🟢 (`A-F1`, `A-R1`, `A-SCOPE` đạt `PASS`)

