# 13. Analysis — Định nghĩa Toàn diện `/nats-monitor` & Phân tích Hạn chế Hiện trạng (`nats_monitor_v2`)

## 1. Chẩn đoán Hiện trạng: Tại sao `/nats-monitor` đang "Chung chung & Khó kiểm soát"? (`VERIFIED`)

| Khía cạnh | Hiện trạng trong Code (`VERIFIED`) | Hệ lụy Vận hành Thực tế ("Chung chung, Khó kiểm soát") |
| :--- | :--- | :--- |
| **1. Tầng Backend (`nats_monitor_handler.go:36-85`)** | Chỉ gọi HTTP GET tới `:18222` (`/varz`, `/jsz?streams=true&config=true&consumers=true`, `/connz` không có `?subs=1`) rồi trả về `map[string]any` thô. Không kết nối với `*natsconn.NatsClient` (`Conn` / `JS`) dù `cdc-cms-service/internal/server/server.go:120` đã khởi tạo sẵn `natsClient`. | - Chỉ là một trang JSON Viewer thụ động.<br>- Không thể thực hiện bất kỳ thao tác điều khiển nào lên JetStream (`Purge`, `DeleteMsg`, `DeleteStream`, `UpdateStream`, `DeleteConsumer`, `GetMsg`).<br>- `/connz` thiếu `?subs=detail` và không gọi `/subsz?subs=1` nên không biết `cdc-worker` đang subscribe những subject nào. |
| **2. Tầng Quản trị Stream (`NatsMonitor.tsx:308-430`)** | Hiển thị danh sách Stream phẳng (`name`, `storage`, `subjects`, `messages`, `bytes`, `consumers`, `seq`, `created`). Không hiển thị `retention` (`limits` vs `workqueue`), `max_bytes`, `max_age`, `discard`. | - **Mù Quota & Đầy ổ đĩa:** Khi `CDC_TRANSMUTE` phình lên ~1GB / 47k msgs (sự cố PROD `2026-10-19`), UI không hiện `% MaxBytes` hay cảnh báo `No MaxBytes Cap`.<br>- **Mù Config Drift:** Không biết Stream trên Broker có bị lệch cấu hình so với chuẩn Data-Hub (`centralized-data-service/pkgs/natsconn/nats_client.go:67-109`) hay không (ví dụ `CDC_TRANSMUTE` bị để `limits` thay vì `workqueue`).<br>- **Mù Stream Rác (Orphan):** Các stream test như `TEST_WQ_CHECK` trộn lẫn với 4 Core Streams mà không được gắn cờ `ORPHAN` và không có nút xóa. |
| **3. Tầng Chẩn đoán Consumer (`NatsMonitor.tsx:435-521`)** | Chỉ hiển thị 8 cột cơ bản trong Drawer (`name`, `filter_subject`, `ack_policy`, `num_ack_pending`, `num_redelivered`, `num_waiting`, `num_pending`, `delivered.stream_seq`, `created`). | - **Mù Độ lệch Sequence (`Ack Floor` vs `Delivered` vs `Stream Last Seq`):** Không thấy `ack_floor.stream_seq` đang tụt lại bao xa so với `delivered.stream_seq` và `stream.state.last_seq`.<br>- **Mù Bão hòa In-Flight:** Không hiển thị `num_ack_pending / config.max_ack_pending` (khi chạm `1000/1000` consumer sẽ bị khóa cứng).<br>- **Thiếu thông số cốt lõi:** Không hiện `ack_wait` (30s), `max_deliver` (5), `deliver_group` (`transmute-shadow-consumer-v2-group`), `push_bound` (có worker đang gắn vào không), và `last_active`. |
| **4. Tầng Soi dữ liệu trong Queue (Message Inspector)** | **Hoàn toàn không có.** | - Khi `CDC_TRANSMUTE` hoặc `SCHEMA_DRIFT` ứ đọng hàng chục nghìn message, Operator mù tịt không biết message đang kẹt ở `first_seq` hay `last_seq` là của bảng `shadow_table` nào, `_source_ids` nào, `Nats-Msg-Id` nào, hay bị kẹt từ bao giờ. |
| **5. Tầng Kiểm soát Kênh lệnh Core NATS (`cdc.cmd.*`)** | **Hoàn toàn không có.** | - Data-Hub dùng >25 subjects Core NATS (`cdc.cmd.transmute`, `cdc.cmd.recon-check`, `cdc.cmd.execute-heal`, `cdc.cmd.snapshot.v2`...) để giao tiếp giữa `cdc-cms-service` và `centralized-data-service` (`server.go:265-285`, `server_setup.go:299-539`). Hiện tại nếu `cdc-worker` chết hoặc rớt subscription trên 1 subject, `/nats-monitor` vẫn báo `NATS ONLINE` xanh lè mà không cảnh báo kênh lệnh bị mất người nghe (`0 listeners`). |

---

## 2. Định nghĩa Chuẩn `/nats-monitor` V2: Trung tâm Giám sát & Điều khiển NATS Data-Hub (4 Trụ cột)

Để `/nats-monitor` thực sự **rõ ràng, sát nghiệp vụ Data-Hub và kiểm soát được 100%**, hệ thống được định nghĩa lại theo **4 Trụ cột (4 Pillars)**:

### Trụ cột 1: Broker Health, Storage Quota & Smart Diagnostics Banner (Giám sát Áp lực Hạ tầng)
1. **KPIs Có Ngưỡng Kiểm Soát (Threshold-Aware KPIs):**
   - **Broker Status & Slow Consumers:** `version`, `uptime`, `cpu`, `mem`, `slow_consumers` (Cảnh báo đỏ nếu `slow_consumers > 0` — dấu hiệu consumer đọc không kịp bị NATS drop).
   - **JetStream Disk & Memory Quota:** Hiển thị `jetstream.storage / jetstream.config.max_storage` và `jetstream.memory / jetstream.config.max_memory` kèm Progress Bar và `store_dir` (`/data/jetstream`).
   - **In-Flight & Backlog Pressure:** Tổng `messages` đang tồn trên các Stream `workqueue` (`CDC_TRANSMUTE`), tổng `num_ack_pending` và `num_redelivered` toàn hệ thống, và `jetstream.api.errors`.
2. **Smart Alert Banner (Cảnh báo Chẩn đoán Tự động):**
   - Tự động phát hiện và hiển thị ngay đầu trang nếu có bất kỳ vi phạm nào:
     - `STREAM_DRIFT`: Stream cốt lõi lệch `Retention` (`limits` vs `workqueue`) hoặc thiếu trần `MaxBytes`.
     - `ORPHAN_STREAM`: Tồn tại Stream lạ ngoài 4 Core Streams (`CDC_EVENTS`, `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`).
     - `CONSUMER_STALLED`: Consumer có `num_pending > 0` nhưng `push_bound == false` hoặc `num_ack_pending >= max_ack_pending * 0.8`.
     - `MISSING_WORKER_SUB`: Kênh lệnh Core NATS trọng yếu (`cdc.cmd.transmute`, `cdc.cmd.recon-check`, `cdc.cmd.snapshot.v2`...) không có `cdc-worker` nào lắng nghe.

### Trụ cột 2: JetStream Governance, Config Drift & Active Control Plane (Quản trị & Điều khiển Stream)
Định nghĩa rõ **Bảng Quy chuẩn 4 Core Streams của Data-Hub** (đối chiếu trực tiếp với `centralized-data-service/pkgs/natsconn/nats_client.go:67-109`):

| Stream Name | Phân loại | Vai trò Nghiệp vụ trong Data-Hub | Chuẩn `Retention` | Chuẩn `MaxBytes` | Chuẩn `MaxAge` |
| :--- | :---: | :--- | :---: | :---: | :---: |
| `CDC_EVENTS` | `CORE` | Luồng sự kiện CDC/Sink (`cdc.<prefix>.>`) đổ vào `ConsumerPool` ghi bảng Shadow | `limits` | `512 MiB` | `24h` |
| `CDC_TRANSMUTE` | `CORE` | Hàng đợi Trigger Realtime Post-Ingest (`cdc.cmd.transmute-shadow`) kích hoạt `TableDebouncer` ghi bảng Master | `workqueue` | `256 MiB` | `1h` |
| `SCHEMA_DRIFT` | `CORE` | Sự kiện phát hiện trường mới / lệch kiểu dữ liệu (`schema.drift.detected`) | `limits` | `64 MiB` | `7d` |
| `SCHEMA_CONFIG` | `CORE` | Tín hiệu đồng bộ/làm mới cache cấu hình (`schema.config.reload`) | `limits` | `64 MiB` | `7d` |
| *Stream khác* | `ORPHAN` | Stream phát sinh ngoài quy chuẩn (ví dụ: `TEST_WQ_CHECK`) | - | - | - |

**Các thông số & Hành động Kiểm soát trên từng Stream:**
- **Thông số hiển thị:**
  - Tên Stream + Tag Phân loại (`CORE` / `ORPHAN`) + Mô tả vai trò Data-Hub.
  - Trạng thái Chuẩn hóa (`SYNCED` 🟢 / `CONFIG DRIFT` 🟠 kèm chi tiết lệch / `ORPHAN` 🔴).
  - Chính sách Lưu trữ & Giữ tin: `Storage` (`FILE`/`MEMORY`), `Retention` (`WORKQUEUE`/`LIMITS`), `Discard` (`OLD`/`NEW`), `MaxAge` (quy đổi `1h`, `24h`, `7d`).
  - **Áp lực Dung lượng (`Bytes / MaxBytes`):** Hiển thị cả số tuyệt đối lẫn thanh Progress `%` (`state.bytes / config.max_bytes`).
  - **Khoảng Sequence & Thời gian (`FirstSeq -> LastSeq`, `FirstTs -> LastTs`):** Cho biết tin cũ nhất trong hàng đợi đang nằm từ thời điểm nào.
- **4 Hành động Điều khiển Trực tiếp (Control Actions):**
  1. **🔍 Soi Message (Message Inspector / Peek):** Mở Drawer xem các message đang nằm trong Stream (lấy mẫu 20 messages đầu/cuối hoặc tra cứu theo `Seq` / lọc theo `Subject` qua `JS.GetMsg` / `JS.GetLastMsg` — **Read-Only 100%, không làm mất/ack message**). Hỗ trợ **Xóa 1 Message lỗi (Delete Poison Pill Msg by Seq)** qua `JS.DeleteMsg(stream, seq)`.
  2. **🧹 Làm sạch Hàng đợi (Purge Stream):** Mở Modal cho phép xóa toàn bộ message tồn đọng trong Stream hoặc chỉ xóa theo `Subject` / giữ lại `Keep N` message mới nhất (`JS.PurgeStream`).
  3. **⚙️ Chuẩn hóa / Sửa Giới hạn Stream (Reconcile / Tune Limits):**
     - Nút **"Chuẩn hóa toàn bộ Core Streams"** (`POST /api/v1/system/nats/streams/reconcile`): Tự động đưa 4 Core Streams về đúng chuẩn `Retention`, `MaxBytes`, `MaxAge`, `Discard` (kể cả migrate `CDC_TRANSMUTE` từ `limits` sang `workqueue`).
     - Nút **"Chỉnh Quota (`MaxBytes` / `MaxAge`)"** (`PATCH /api/v1/system/nats/streams/:stream/config`): Cho phép Admin điều chỉnh nhanh trần `MaxBytes` (MB) và `MaxAge` (giờ) ngay trên UI khi lưu lượng tăng đột biến.
  4. **🗑️ Xóa Stream Rác (Delete Orphan Stream):** Chỉ bật cho các stream `ORPHAN` (khóa bảo vệ 4 Core Streams không bị bấm xóa nhầm).

### Trụ cột 3: Deep Consumer Diagnostics & Recovery (Kiểm soát Consumer & Chống Kẹt)
Nâng cấp toàn diện Drawer/Bảng Consumer của từng Stream:
- **Bộ chỉ số Chẩn đoán Điểm nghẽn (Bottleneck Telemetry):**
  - **Trạng thái Gắn kết (`Push Bound` / `Active`):** Cho biết hiện tại có goroutine/subscription của `cdc-worker` đang gắn vào Consumer này không (`push_bound: true/false`), kèm `deliver_group` (`transmute-shadow-consumer-v2-group`).
  - **Độ trễ & Vị trí Sequence 3 Tầng:**
    - `Ack Floor Seq` (`#82`) $\le$ `Delivered Seq` (`#82`) $\le$ `Stream Last Seq` (`#82`).
    - Tính rõ: `Unacked Gap = Delivered.StreamSeq - AckFloor.StreamSeq` và `Pending Lag = num_pending`.
  - **Độ bão hòa Cửa sổ In-Flight (`Ack Pending Saturation`):**
    - Hiển thị `num_ack_pending / config.max_ack_pending` (ví dụ `0 / 1000` kèm Progress Bar; đổi màu cam khi $\ge 50\%$, đỏ khi $\ge 80\%$).
  - **Chính sách Retry & Timeout:**
    - `AckWait` (quy đổi từ nanoseconds sang giây: `30s`), `MaxDeliver` (`5` lần), `NumRedelivered` (số lần gửi lại do timeout/NAK).
  - **Hoạt động gần nhất (`Last Active`):** Thời gian kể từ lần cuối `delivered.last_active` và `ack_floor.last_active`.
- **Hành động Điều khiển Consumer (Consumer Control):**
  - **🔄 Reset / Xóa Consumer (`DELETE /api/v1/system/nats/streams/:stream/consumers/:consumer`):** Khi Consumer bị treo trạng thái hoặc là Consumer mồ côi cũ, cho phép xóa/reset (với Core Consumer như `transmute-shadow-consumer-v2`, hỗ trợ recreate lại đúng `ConsumerConfig` chuẩn ngay lập tức).

### Trụ cột 4: Data-Hub Command Channels Matrix & Connection Subscriptions (Kiểm soát Kênh Lệnh Nghiệp vụ)
- Bổ sung Tab **"Kênh lệnh Data-Hub (`cdc.cmd.*`)"**:
  - Backend trích xuất danh sách `subscriptions_list` từ `/connz?subs=detail` và `/subsz?subs=1`, đối chiếu với **Danh mục Kênh lệnh Chuẩn của Data-Hub** (Transmute, Recon Check/Heal, Snapshot V2, Debezium Signal, Schema DDL, Introspect, Reload Config).
  - Hiển thị rõ từng kênh lệnh:
    - `Subject` (vd: `cdc.cmd.transmute`, `cdc.cmd.transmute-shadow.core-fallback`, `cdc.cmd.recon-check`, `cdc.cmd.execute-heal`, `cdc.cmd.snapshot.v2`...)
    - `Nhóm Nghiệp vụ` (`Master Transmute`, `Reconciliation`, `Source & Snapshot`, `Shadow & DDL`, `System Broadcast`)
    - `Queue Group` (`transmute-workers`, `cdc-worker-group`, `cdc-snapshot-runner`, `broadcast`)
    - `Active Listeners` (Số lượng subscription thực tế từ `cdc-worker` đang lắng nghe + tên client CID)
    - `Trạng thái` (`READY` 🟢 khi có $\ge 1$ worker lắng nghe vs `NO LISTENER` 🔴 khi không có worker nào lắng nghe).
- Nâng cấp Tab **"Active Connections"**:
  - Hiển thị thêm cột `RTT` (độ trễ mạng giữa client và NATS broker, vd `1.38ms`), `Role` (`Worker`, `CMS API`, `Other`), và nút mở xem chi tiết danh sách **Subscriptions thực tế (`subscriptions_list`)** của từng kết nối.

---

## 5. Phân tích Root-Cause Runtime: Tại sao khi Update 1 Record (`buffer_type: cdc-stream`, `kafka-consumer-hook`), kiểm tra `CDC_EVENTS` và `CDC_TRANSMUTE` đều thấy `0 msgs` & `0 B`? (Ngày `2026-10-20`)

### 5.1. Câu trả lời trực diện: `CDC_EVENTS` và `CDC_TRANSMUTE` KHÔNG PHẢI MOCKUP
Toàn bộ số liệu hiển thị trên `/nats-monitor` đều được đọc trực tiếp thời gian thực 100% từ NATS Monitoring HTTP API (`http://127.0.0.1:18222/jsz?streams=true&config=true&consumers=true`) và NATS JetStream SDK (`127.0.0.1:14222`).

Tuy nhiên, việc cả 2 stream `CDC_EVENTS` và `CDC_TRANSMUTE` đều hiển thị **`0 msgs` (`Trống`)** và **`0 B` (`JS Storage & Quota` không tăng)** ngay sau khi anh update 1 bản ghi (lúc `10:31:32`, `10:32:16`, `10:32:44`) xuất phát từ **2 nguyên nhân kiến trúc thực tế khác nhau hoàn toàn** giữa `CDC_EVENTS` và `CDC_TRANSMUTE`, cộng với **1 điểm mù hiển thị (UI Blind Spot)** trên bảng `NatsMonitor.tsx`:

---

### 5.2. Nguyên nhân 1 — Stream `CDC_EVENTS`: Đang bị luồng `KafkaConsumer` đi tắt qua RAM (In-Memory Bypass), không có Publisher nào bắn vào NATS!
1. **Dấu vết trong Activity Log của anh:**
   - `10:32:16 20/10/2026 kafka-consumer centrallized-export-service.export-jobs -> shadow_traitestces.export_jobs ... "buffer_type": "cdc-stream"`
   - `10:32:44 20/10/2026 kafka-consumer wallet-service.events -> shadow_walletservice.events ... "buffer_type": "cdc-stream"`
2. **Sự thật trong Code (`centralized-data-service`) — Kèm `file:line`:**
   - Tại `centralized-data-service/internal/handler/shadow/kafka_consumer.go:748`:
     Khi `KafkaConsumer.processMessage` đọc bản tin Debezium từ Kafka (`wallet-service.events`, `centrallized-export-service.export-jobs`, `payment-bill-service.payment-bills`), nó **gọi trực tiếp hàm Go trong bộ nhớ (In-Memory)**:
     ```go
     return kc.eventHandler.HandleRaw(ctx, subject, cdcJSON)
     ```
     chứ **HOÀN TOÀN KHÔNG gọi `natsClient.JS.Publish` vào stream `CDC_EVENTS`**!
   - Sau đó `eventHandler.HandleRaw` đẩy thẳng vào `cdcBatchBuffer` trong RAM (`centralized-data-service/internal/handler/shadow/batch_buffer.go:622, 650`). Chuỗi `"buffer_type": "cdc-stream"` trong log là tên của bộ đệm Go `cdcBatchBuffer` (để phân biệt với bộ đệm `"snapshot"` tại `server_setup.go:158, 172`), **không phải NATS JetStream `CDC_EVENTS`**.
   - Trong khi đó, tại `centralized-data-service/pkgs/natsconn/nats_client.go:69-77` và `internal/server/server_setup.go:225-235`, hàm `NewWorkerServer` vẫn tạo stream `CDC_EVENTS` (`subjects: ["cdc.gpaylocal.>", "cdc.goopaylocal.>", "cdc.mariadblocal.>", "cdc.sftplocal.>"]`) và vẫn khởi chạy 4 `ConsumerPool` (`cdc-worker-group-local-cdc-*`, tổng 40 worker goroutines đứng `Fetch`), nhưng trên toàn bộ codebase `centralized-data-service` hiện **không có bất kỳ dòng code nào publish vào các subject của `CDC_EVENTS`**!
3. **Bằng chứng Runtime trực tiếp từ `:18222/jsz`:**
   ```json
   "name": "CDC_EVENTS",
   "state": {
     "messages": 0,
     "bytes": 0,
     "first_seq": 2,
     "last_seq": 1,
     "last_ts": "2026-09-04T07:18:31.609392132Z",
     "consumer_count": 4
   }
   ```
   $\rightarrow$ Bản tin cuối cùng đi vào `CDC_EVENTS` là `seq #1` từ ngày `04/09/2026`. Kể từ khi `KafkaConsumer` chuyển sang gọi thẳng `kc.eventHandler.HandleRaw` in-memory, `CDC_EVENTS` trở thành một **Legacy/Idle Stream** (có Stream + 4 Consumers đứng chờ nhưng không có Publisher).

---

### 5.3. Nguyên nhân 2 — Stream `CDC_TRANSMUTE`: CÓ nhận và xử lý bản tin thật lúc `10:32:16` (`seq #200`) và `10:40:16` (`seq #204`), nhưng cơ chế `WorkQueuePolicy` xóa ngay lập tức sau khi ACK (`39ms`) + UI đang ẩn `last_seq` & `last_ts` khi `messages == 0`!
1. **Dấu vết trong Activity Log của anh:**
   - `10:32:16 20/10/2026 transmute shadow_traitestces.export_jobs -> master_centrallized_export_service.export_jobs* success 1 39ms kafka-consumer-hook`
2. **Luồng chạy thực tế trong Code (`centralized-data-service`) — Kèm `file:line`:**
   - Sau khi `cdcBatchBuffer` ghi xong bảng `shadow_traitestces.export_jobs`, hàm `publishTransmuteTrigger` tại `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:86-109` **CÓ bắn bản tin thật vào NATS JetStream `CDC_TRANSMUTE`** (`Subject: "cdc.cmd.transmute-shadow"`).
   - Consumer `transmute-shadow-consumer-v2` (`server_setup.go:338-350`) nhận bản tin này, đi vào `HandleTransmuteShadow` (`internal/handler/master/transmute_handler.go:189`, nơi gán `hookTrigger := "kafka-consumer-hook"`), thực thi đồng bộ sang 3 bảng Master (`export_jobs`, `export_jobs_1`, `export_jobs_2`) trong **`24ms–39ms`**, rồi gọi **`msg.Ack()`** (`tracker.completeTask` tại `transmute_handler.go:215`).
3. **Bằng chứng Runtime trực tiếp từ `:18222/jsz` đối chiếu đúng từng giây với log của anh:**
   - Ngay tại thời điểm `10:32:16` (giờ VN, tức `03:32:16Z` UTC) khi anh update bản ghi `export_jobs`, trạng thái thật của `CDC_TRANSMUTE` trên NATS `:18222/jsz` là:
     ```json
     "name": "CDC_TRANSMUTE",
     "config": { "retention": "workqueue", "subjects": ["cdc.cmd.transmute-shadow"], "max_bytes": 268435456 },
     "state": {
       "messages": 0,
       "bytes": 0,
       "first_seq": 201,
       "last_seq": 200,
       "last_ts": "2026-10-20T03:32:16.77305913Z",
       "consumer_count": 1
     }
     ```
   - Và đến `10:40:16` (giờ VN, tức `03:40:16Z` UTC), `last_seq` tiếp tục tăng lên **`204`** (`delivered.consumer_seq = 212`, `ack_floor.stream_seq = 204`, `last_active = 2026-10-20T03:40:16.886Z`)!
4. **Tại sao `CDC_TRANSMUTE` vừa xử lý tới `seq #204` mà trên giao diện `/nats-monitor` anh lại thấy `0 msgs`, `0 B` và tưởng nó không hoạt động?**
   - **Lý do 1 (Đặc tính `WorkQueuePolicy` của NATS JetStream tại `nats_client.go:101`):**
     Khác với `SCHEMA_DRIFT` (`Retention: LimitsPolicy` — giữ lại bản tin sau khi đọc nên có `119 msgs = 294.78 KB`), `CDC_TRANSMUTE` dùng `Retention: WorkQueuePolicy` (**xóa bản tin khỏi ổ đĩa ngay lập tức ở mili-giây thứ 39 khi Worker gọi `msg.Ack()`**). Vì Worker xử lý mất có `25ms–39ms` trong khi UI poll mỗi `5000ms` (`5 giây`), nên khi UI chụp trạng thái thì bản tin `#200..#204` đã được ACK và xóa khỏi đĩa từ lâu $\rightarrow$ số bản tin **đang tồn đọng chưa xử lý (`state.messages`) = `0`** và **dung lượng chiếm đĩa hiện tại (`state.bytes`) = `0 B`**.
   - **Lý do 2 (Điểm mù hiển thị cực lớn tại `cdc-cms-web/src/pages/NatsMonitor.tsx:939-966`):**
     Trong hàm render cột `Messages & Sequence` (`NatsMonitor.tsx:949-964`), code hiện tại viết:
     ```tsx
     {msgs === 0 ? (
       <Text type="secondary" style={{ fontSize: 12 }}>
         Trống (Next: #{first > 0 ? first : last + 1})
       </Text>
     ) : (
       <>
         <Text type="secondary" style={{ fontSize: 12 }}>
           Seq: #{first} → #{last}
         </Text>
         {lastTs !== '-' && (
           <Text type="secondary" style={{ fontSize: 11 }}>
             Cuối: {lastTs}
           </Text>
         )}
       </>
     )}
     ```
     Chính câu lệnh `if (msgs === 0)` này đã **GIẤU MẤT `last_seq` (`Đã xử lý: #204`) và GIẤU MẤT `last_ts` (`Cuối: 20/10/2026 10:40:16`)** khi `msgs === 0`!
     Ngoài ra:
     - Ở thẻ KPI số 2 (**`JS Storage & Quota`** tại `NatsMonitor.tsx:1638-1668`) và cột **`Dung lượng & Quota`** (`L970-1002`), UI chỉ hiển thị dung lượng **đang nằm chờ trên đĩa** (`0 B / 256 MB`), không hiển thị tổng số bản tin đã đi qua luồng (`Processed Throughput / Last Seq`).
     - Ở cột **`Consumers & Backpressure`** (`NatsMonitor.tsx:1005-1039`), UI chỉ hiển thị `In-Flight: 0` mà không hiển thị `AckFloor: #204` (`Delivered: 212`) và `Last Active: 10:40:16`.
     - Khi bấm **`Soi Msg` (Message Inspector)** trên `CDC_TRANSMUTE`, UI chỉ báo *"Stream hiện đang trống"* mà không giải thích rằng đây là stream `WORKQUEUE` đã tiêu thụ và xóa ngay toàn bộ `#204` bản tin sau khi ACK (lần cuối lúc `10:40:16`).

### 4.4. Khi nào dữ liệu thực sự đi vào `CDC_EVENTS`? (Code Trace 100%)
1. **Cấu hình Subject & Consumer của `CDC_EVENTS`:**
   - Tại `centralized-data-service/internal/server/server_setup.go:87-101` và `pkgs/natsconn/nats_client.go:68-77`, `CDC_EVENTS` được cấu hình bắt các wildcard subjects suy ra từ `cfg.Kafka.TopicPrefix`: `["cdc.goopaylocal.>", "cdc.goopay.>", "cdc.gpay.>", "cdc.mariadb.>", "sftp.>"]`.
   - Tại `server_setup.go:225-235`, Worker tạo các `ConsumerPool` (`PullSubscribe` tại `internal/handler/shadow/consumer_pool.go:45-51`) lắng nghe từng nhánh subject trên và gọi `cdcEventHandler.Handle(msgCtx, msg)` (`event_handler.go:509-516`).
2. **Hai luồng thực tế bắn message vào `CDC_EVENTS` (Chỉ kích hoạt khi có lỗi cần Replay từ DLQ):**
   - **Luồng 1 (Chính yếu — `DLQStateMachine` tự động Replay từ `cdc_system.failed_sync_logs`):**
     - Khi `KafkaConsumer` đọc một message từ Kafka mà bị lỗi (`schema_drift`, `missing_required_field`, hoặc lỗi `processMessage`), nó gọi `kc.writeDLQ(ctx, msg, err)` tại `internal/handler/shadow/dlq_helper.go:19-55` (được gọi từ `kafka_consumer.go:539, 557`). Hàm này lưu bản ghi lỗi vào bảng `cdc_system.failed_sync_logs` kèm cột **`KafkaTopic: msg.Topic`** (ví dụ: `cdc.goopaylocal.payment-bill-service.payment-bills`).
     - Định kỳ mỗi **5 phút**, tiến trình ngầm `DLQStateMachine` (`server_setup.go:213-214`, `internal/handler/dlq/dlq_state_machine.go:94-120, 155-163`) quét bảng `cdc_system.failed_sync_logs` lấy các bản ghi có `status IN ('pending','failed','retrying')` và `retry_count < 3`.
     - Tại `dlq_state_machine.go:234-244`, `retryOne` lấy `subject := row.KafkaTopic` (chính là tên topic Kafka bắt đầu bằng `cdc.goopaylocal.*` / `cdc.goopay.*`...) và bắn vào NATS:
       ```go
       subject := row.KafkaTopic
       ...
       msg := &nats.Msg{Subject: subject, Data: row.RawJSON, Header: make(nats.Header)}
       observability.InjectNATSHeader(ctx, msg.Header)
       publishErr := sm.nats.Conn.PublishMsg(msg)
       ```
     - Vì `subject` khớp với wildcard `cdc.goopaylocal.>` của stream **`CDC_EVENTS`**, NATS JetStream thu bản tin này vào `CDC_EVENTS`, và `ConsumerPool` (`consumer_pool.go:87-115`) sẽ kéo bản tin đó ra để chạy lại `cdcEventHandler.Handle` $\rightarrow$ đẩy vào `cdcBatchBuffer` ghi lại xuống Shadow DB.
   - **Luồng 2 (`DLQHandler.ReplayDLQ` tại `internal/handler/dlq/dlq_handler.go:243-255`):**
     - Khi gọi `ReplayDLQ`, hàm giải mã `DLQMessage`, lấy `msg.OriginalSubject` (thuộc nhánh `cdc.<prefix>.>`) và gọi `d.nats.Conn.PublishMsg(replayMsg)` để đẩy lại bản tin vào `CDC_EVENTS`.
3. **Tại sao bình thường (Happy-Path) `CDC_EVENTS` đứng yên?**
   - Ở luồng Happy-Path không lỗi, `KafkaConsumer.processMessage` (`kafka_consumer.go:748`) gọi thẳng `kc.eventHandler.HandleRaw(ctx, subject, cdcJSON)` trong RAM để gom vào `cdcBatchBuffer` (tránh nhân đôi I/O đĩa vừa ghi Kafka vừa ghi NATS). Do đó, `CDC_EVENTS` chỉ tăng message khi có bản ghi lỗi trong `cdc_system.failed_sync_logs` (có `kafka_topic != ''`) được `DLQStateMachine` hoặc `ReplayDLQ` bắn ngược lại để xử lý lại.

### 4.5. Giải đáp: (1) Xem toàn bộ lịch sử sự kiện ở đâu? và (2) Vì sao `CDC_TRANSMUTE` hiển thị `1 (1/1 Active)` nhưng thực tế có 10 Worker điều phối song song?

1. **Về việc xem toàn bộ lịch sử bản tin đã chạy qua `CDC_EVENTS` và `CDC_TRANSMUTE`:**
   - **Trên NATS JetStream (`Soi Msg`):**
     - Với `CDC_TRANSMUTE`, do cấu hình `Retention: nats.WorkQueuePolicy` (`pkgs/natsconn/nats_client.go:101`), NATS JetStream xóa bản tin khỏi ổ đĩa ngay khi Worker gọi `msg.Ack()` (`~25–39ms`). Vì vậy `Soi Msg` trên `CDC_TRANSMUTE` chỉ hiển thị các bản tin **đang xếp hàng chờ xử lý (`pending` / `in-flight`)** tại thời điểm kiểm tra. (Nếu đổi sang `LimitsPolicy` thì NATS mới giữ lại bản tin sau khi ACK, nhưng kiến trúc hệ thống chủ đích dùng `WorkQueuePolicy` tại `nats_client.go:123-127` để giải phóng đĩa ngay và đảm bảo ngữ nghĩa hàng đợi công việc).
     - Với `CDC_EVENTS` (`Retention: nats.LimitsPolicy`, giữ 24h / 512MB), mọi bản tin DLQ Replay đi vào đây đều được giữ lại trên đĩa 24h và xem lại được qua `Soi Msg`. Còn các bản tin Happy-Path đi tắt qua RAM thì bản gốc nằm trên **Kafka Broker** (topic `cdc.<prefix>.<db>.<table>`).
   - **Nơi lưu trữ toàn bộ 100% lịch sử thực thi (Audit Trail đầy đủ):**
     - **Bảng `cdc_system.activity_logs` (Trang Nhật ký Hoạt động trên CMS):**
       - Lưu mọi batch `kafka-consumer` (`internal/handler/shadow/batch_buffer.go:617-661`) kèm `c`, `u`, `d`, `written`, `batch_size`, `trace_id`, `shadow_schema`.
       - Lưu mọi lệnh `transmute` (`internal/handler/master/transmute_handler.go:230-260`) kèm `triggered_by = "kafka-consumer-hook"`, `scanned`, `inserted`, `updated`, `skipped`, `duration_ms`, `master_binding_id`, `trace_id`.
     - **Bảng `cdc_system.transmute_jobs` (`internal/service/master/transmuter.go:575-625`):** Lưu chi tiết từng job Transmute đã thực thi.

2. **Về việc tại sao `CDC_TRANSMUTE` hiển thị `Consumers & Backpressure: 1 (1/1 Active)` — có phải chỉ có 1 con điều phối không?**
   - **Thực tế có ĐÚNG 10 WORKER SUBSCRIPTIONS (`PoolSize = 10`) đang chia tải song song, KHÔNG phải chỉ có 1 worker đơn lẻ!**
   - **Sự khác biệt giữa số lượng `Durable Consumer` (`1/1`) và số lượng `Worker Subscriptions` (`10 workers`):**
     - Tại `internal/server/server_setup.go:334-348` và `internal/handler/master/transmute_consumer_pool.go:102-153`:
       - Trên NATS JetStream, stream `CDC_TRANSMUTE` chỉ có 1 subject (`cdc.cmd.transmute-shadow`), nên hệ thống đăng ký **1 Durable Consumer Object** tên là `transmute-shadow-consumer-v2` với `DeliverGroup: "transmute-shadow-consumer-v2-group"`. Vì chỉ có 1 đối tượng Durable Consumer trong `/jsz`, cột `Consumers` hiển thị `1 (1/1 Active)` (nghĩa là **1/1 Durable Consumer Group đang Active**).
       - Tuy nhiên, tại `transmute_consumer_pool.go:143-153`, vòng lặp `for i := 0; i < p.cfg.WorkerCount; i++` (`WorkerCount = 10` từ `cfg.Worker.PoolSize` trong `config-local.yml:79`) tạo ra **10 `QueueSubscribe` worker độc lập** cùng gắn vào queue group `transmute-shadow-consumer-v2-group`!
     - **Bằng chứng runtime thật từ `http://127.0.0.1:18222/connz?subs=detail` (trên kết nối `cid=58`, `name="cdc-worker"`):**
       Có đúng **10 worker subscriptions (`sid: 48` đến `sid: 57`)** cùng gắn vào `qgroup: "transmute-shadow-consumer-v2-group"`, và cả 10 con đều đang chia tải xử lý bản tin thật:
       - `sid=48`: `msgs=3` | `sid=49`: `msgs=4` | `sid=50`: `msgs=4` | `sid=51`: `msgs=2` | `sid=52`: `msgs=1`
       - `sid=53`: `msgs=6` | `sid=54`: `msgs=6` | `sid=55`: `msgs=2` | `sid=56`: `msgs=1` | `sid=57`: `msgs=4`
     - **So sánh với `CDC_EVENTS` (`4/4 Active`):**
       Tại `server_setup.go:226-235`, `CDC_EVENTS` lặp qua **4 `cdcPrefixes`** (`cdc.goopaylocal`, `cdc.goopay`, `cdc.gpay`, `cdc.mariadb`), tạo ra **4 Durable Consumers** riêng biệt (mỗi consumer bên dưới cũng có `PoolSize = 10` goroutine chạy `sub.Fetch(100)` tại `consumer_pool.go:71-87`). Vì có 4 Durable Consumers nên `/jsz` báo `4 (4/4 Active)`.

### 4.6. Khi `Transmute` bị lỗi thì bản tin và thông tin lỗi được quăng vào đâu? (Code Trace 100%)
1. **Cơ chế xử lý trên NATS JetStream (`CDC_TRANSMUTE`) & `TableDebouncer`:**
   - **Bước 1 — Phân loại lỗi Tạm thời (`Transient`) vs Lỗi Dữ liệu độc (`Poison Pill`) & Chia để trị (`Binary Search Split`):**
     - Tại `internal/handler/master/transmute_handler.go:621-653`:
       - Nếu là lỗi mạng/DB tạm thời (`isTransientError(err)` tại `transmute_handler.go:826-845`: `deadlock`, `connection refused`, `statement timeout`, `too many clients`...), hệ thống trả lỗi ngay cho `shadowMsgTracker` để retry cả mẻ.
       - Nếu là lỗi dữ liệu/mapping (`Poison Pill`), hệ thống kích hoạt **`binarySearchSplit` (`transmute_handler.go:652, 690-720`)**: chia đôi mẻ đệ quy (`processSubBatch` `L722-824`) để các bản ghi lành trong cùng mẻ vẫn được `ACK` và ghi vào Master DB bình thường, đồng thời cô lập chính xác 1 task duy nhất chứa bản ghi lỗi (`len(batch) == 1` tại `L691-712`).
   - **Bước 2 — Retry có trễ (`NakWithDelay`) & Kết thúc (`Term`) trên `CDC_TRANSMUTE`:**
     - Tại `internal/handler/master/transmute_consumer_pool.go:198-235` (`finalizeJetStreamMessage`):
       - Nếu `meta.NumDelivered < MaxDeliver (5 lần)`: Gọi **`msg.NakWithDelay(delay)`** (`delay = NumDelivered * 2s`: `2s`, `4s`, `6s`, `8s`). Trong lúc chờ retry, bản tin vẫn nằm trong `CDC_TRANSMUTE` (trên `/nats-monitor` sẽ hiện `pending > 0`, `Retry > 0` và xem được qua `Soi Msg`).
       - Nếu đã thử đủ **`5 lần (`meta.NumDelivered >= 5`)` mà vẫn lỗi**: Gọi **`msg.Term()`** (`L204-215`) để loại bỏ bản tin khỏi hàng đợi `CDC_TRANSMUTE` (chống nghẽn vĩnh viễn).
2. **4 Bảng trong PostgreSQL (`cdc_system`) nơi thông tin lỗi Transmute được quăng vào:**
   *(Lưu ý: Khác với tầng Shadow `KafkaConsumer` ghi vào `cdc_system.failed_sync_logs`, tầng Master không cần copy lại payload thô vì bản ghi gốc đã nằm an toàn trong bảng Shadow DB `shadow_<schema>.<table>`).*
   - **(1) Bảng `cdc_system.activity_logs` (Trang Nhật ký Hoạt động trên CMS):**
     - Ghi dòng `operation = 'transmute'` (hoặc `'transmute_sub'`), `status = 'error'`, kèm `error_message` và `{"master_binding_id": ..., "split_retry": true}` (`transmute_handler.go:631-647, 774-789`).
     - Khi cô lập xong Poison Pill (`transmute_handler.go:697-704`), ghi dòng **`operation = 'transmute_poison_pill'`**, `status = 'error'` chỉ đích danh danh sách ID lỗi: `{"master_binding_id": ..., "poison_pill": true, "source_ids": ["..."]}`.
   - **(2) Bảng `cdc_system.sync_runtime_state` (Trạng thái sức khỏe của từng `master_binding_id`):**
     - Tại `internal/service/master/transmuter_state.go:35-44` (`markRuntimeFailure`), cập nhật dòng `runtime_scope = 'master'` của `master_binding_id` với `last_error_at = NOW()` và `last_error_message` (kể cả khi `TypeErrors > 0` hoặc `RuleMisses > 0` làm 0 dòng được ghi tại `transmuter.go:545-556`).
   - **(3) Bảng `cdc_system.transmute_schedule` (Nếu chạy từ lịch định kỳ):**
     - `JobMonitor.HandleCompleted` (`internal/service/master/job_monitor.go:72-102`) nhận sự kiện `cdc.evt.transmute.completed` và cập nhật `last_status = 'error'`, `last_error = ...`, `last_trace_id = ...`.
   - **(4) Bảng `cdc_system.transmute_jobs` (Nếu chạy Manual / Async Job có `job_id`):**
     - Tại `internal/service/master/transmuter.go:196-207`, cập nhật `status = 'failed'` và `error_message = err.Error()`.
