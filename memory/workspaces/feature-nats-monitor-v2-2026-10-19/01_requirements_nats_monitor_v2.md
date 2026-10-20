tier: T2
environment: local_and_production

# 01. Requirements — Định nghĩa & Giải pháp Nâng cấp Toàn diện `/nats-monitor` (`nats_monitor_v2`)

## 1. Bối cảnh & Vấn đề cốt lõi (Problem Statement)
User yêu cầu:
> *"define cái `/nats-monitor` này, giải pháp nâng cấp toàn diện nó. hiện tại nó đang bị chung chung. khó kiểm soát"*

Qua điều tra thực chứng (`S2`) trên mã nguồn hiện tại:
- Frontend: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/NatsMonitor.tsx` (`1044` dòng)
- Backend CMS: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go` (`104` dòng)
- Kiến trúc NATS thực tế của Data-Hub:
  - `centralized-data-service/pkgs/natsconn/nats_client.go:63-167` (`EnsureStreams`: `CDC_EVENTS`, `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`)
  - `centralized-data-service/internal/server/server_setup.go:225-562` (Consumer Pools, `TransmuteConsumerPool`, Core NATS QueueGroups `cdc-worker-group`, `transmute-workers`, `cdc-snapshot-runner`)
  - `cdc-cms-service/internal/server/server.go:182-285` (`natsCommandBus` với 20 async command subjects `cdc.cmd.*`)

### Tại sao `/nats-monitor` hiện tại bị "chung chung" và "khó kiểm soát"?
1. **Chỉ là bản dump thụ động `/varz`, `/jsz`, `/connz` (Generic Passive Viewer):**
   - `nats_monitor_handler.go:46-62` chỉ proxy thô 3 endpoint HTTP `:18222` (`/varz`, `/jsz?streams=true&config=true&consumers=true`, `/connz` không có `?subs=1`) và không có bất kỳ liên kết nào với `natsClient.JS` / `natsClient.Conn` hay kiến trúc nghiệp vụ của Data-Hub.
2. **Mù ngữ cảnh nghiệp vụ Data-Hub (Zero Domain Context):**
   - Nhìn vào bảng Stream/Consumer hiện tại, Operator không biết:
     - Stream nào là **Core System Stream chuẩn** (`CDC_EVENTS`, `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`) và Stream nào là **Orphan / Rác** (như `TEST_WQ_CHECK` phát sinh khi test) hoặc bị **Lệch cấu hình (Config Drift)** so với chuẩn (`Retention: WorkQueuePolicy` vs `LimitsPolicy`, trần `MaxBytes`, `MaxAge`).
     - Không có thanh cảnh báo **Quota Đĩa / Bộ nhớ JetStream** (`storage / max_storage`, `bytes / max_bytes` của từng stream) — nguyên nhân trực tiếp gây sự cố đầy ổ `/data/jetstream` (`ENOSPC`) trên Production ngày `2026-10-19`.
     - Không biết 20+ Core NATS Command Subjects (`cdc.cmd.transmute`, `cdc.cmd.recon-check`, `cdc.cmd.execute-heal`, `cdc.cmd.snapshot.v2`...) có đang được `cdc-worker` lắng nghe đủ qua QueueGroup (`cdc-worker-group`, `transmute-workers`, `cdc-snapshot-runner`) hay không (vì `/connz` hiện tại thiếu `?subs=detail` và thiếu `/subsz`).
3. **Mù chẩn đoán độ trễ & điểm nghẽn Consumer (Blind Consumer Diagnostics):**
   - Drawer Consumer (`NatsMonitor.tsx:435-521`) chỉ hiện các con số rời rạc (`num_ack_pending`, `num_redelivered`, `num_pending`), thiếu hẳn:
     - **Tỷ lệ nghẽn cửa sổ In-Flight (`num_ack_pending / max_ack_pending`):** Khi chạm `1000/1000`, JetStream ngưng đẩy message mới (Head-of-Line Blocking).
     - **Độ lệch Sequence (`Stream Last Seq` vs `Delivered Seq` vs `Ack Floor Seq`):** Không thấy khoảng cách giữa tin mới nhất trên Stream và tin đã Ack hoàn tất.
     - **Cấu hình Consumer критическое:** Không hiển thị `deliver_group` (QueueGroup), `ack_wait` (quy đổi giây), `max_deliver`, `max_ack_pending`, `push_bound` (có worker đang gắn vào không), và `last_active` (consumer đã chết/treo bao lâu).
     - **Không xem được nội dung Message (No Message Peek/Inspector):** Khi `CDC_TRANSMUTE` bị kẹt 47,000 messages hoặc có poison pill redelivered liên tục, Operator không thể xem mẫu message ở `first_seq` / `last_seq` / `ack_floor + 1` đang thuộc `shadow_table` nào, `correlation_id` nào.
4. **Hoàn toàn KHÔNG CÓ khả năng điều khiển / vận hành (Zero Control Plane Actions — "Khó kiểm soát"):**
   - Trang `/nats-monitor` hiện tại **100% chỉ đọc (Read-Only)**:
     - Khi Stream bị đầy đĩa hoặc ứ đọng message cũ: **Không có nút Purge Stream** (xóa message, giữ stream) hay **Purge theo Subject**.
     - Khi Stream bị lệch cấu hình (ví dụ `CDC_TRANSMUTE` trên Prod vẫn ở `LimitsPolicy` hoặc chưa có `MaxBytes`): **Không có nút Reconcile / Chuẩn hóa Cấu hình Stream** hoặc **Cập nhật giới hạn (`MaxBytes`, `MaxAge`)**.
     - Khi phát sinh Stream rác (ví dụ `TEST_WQ_CHECK`): **Không có nút Xóa Stream Rác (Delete Non-System Stream)**.
     - Khi Consumer bị kẹt `num_ack_pending` hoặc lệch `DeliverGroup`: **Không có nút Reset / Recreate Consumer** hoặc xóa Consumer mồ côi.

---

## 2. Yêu cầu Chức năng Nâng cấp Toàn diện (`R1` – `R6`)
- **R1 — Định nghĩa Kiến trúc 4 Tầng cho `/nats-monitor` (Data-Hub NATS Control Center):**
  - Chuyển đổi `/nats-monitor` từ "trang xem JSON `/varz` chung chung" thành **Trung tâm Giám sát & Điều khiển NATS/JetStream đặc thù cho Data-Hub** gồm 4 trụ cột:
    1. **Tầng 1 — Sức khỏe Hạ tầng & Áp lực Đĩa/Bộ nhớ (Broker & Storage Pressure KPI + Health Rules)**
    2. **Tầng 2 — Quản trị & Kiểm soát JetStream Streams (Stream Governance, Drift Detection, Message Inspector & Control Actions)**
    3. **Tầng 3 — Chẩn đoán Sâu & Điều khiển Consumers (Consumer Lag/In-Flight Saturation, AckFloor vs StreamSeq & Consumer Control)**
    4. **Tầng 4 — Bản đồ Kênh Lệnh Core NATS & Kết nối Worker (Data-Hub Command Subject Coverage Matrix & Connection Subscriptions)**
- **R2 — Nâng cấp Backend `NATSMonitorHandler` (`cdc-cms-service`):**
  - Kết hợp cả **NATS HTTP Monitoring (`:18222`)** (`/varz`, `/jsz?streams=true&config=true&consumers=true`, `/connz?subs=detail`, `/subsz?subs=1`) **VÀ** **`*natsconn.NatsClient` (`Conn` + `JS`)** sẵn có trong `cdc-cms-service`:
    - `GET /api/v1/system/nats/overview`: Bổ sung `connz?subs=detail`, `subsz`, bảng đối chiếu **Data-Hub Stream Spec & Drift Analysis** (so sánh cấu hình thực tế trên broker với cấu hình chuẩn Data-Hub), và **Command Subject Coverage** (kiểm tra các subject `cdc.cmd.*` cốt lõi có active subscriber từ `cdc-worker` hay không).
    - `GET /api/v1/system/nats/streams/:stream/messages`: **Message Inspector (Read-Only Peek)** qua `JS.GetMsg(stream, seq)` hoặc `JS.GetLastMsg(stream, subject)` để xem header (`Nats-Msg-Id`, `traceparent`, `Cdc-Job-Id`), subject, timestamp, kích thước và JSON payload (đã cắt gọn an toàn nếu quá lớn) mà không làm tiêu thụ (consume/ack) message trong `WorkQueue`.
    - `POST /api/v1/system/nats/streams/:stream/purge`: Làm sạch message trong stream (`JS.PurgeStream`, hỗ trợ lọc theo `subject` hoặc giữ lại `keep` message mới nhất).
    - `POST /api/v1/system/nats/streams/reconcile`: Đồng bộ/chuẩn hóa lại cấu hình 4 Core Streams (`CDC_EVENTS`, `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`) về đúng chuẩn `MaxBytes`, `MaxAge`, `Retention`, `Discard`.
    - `DELETE /api/v1/system/nats/streams/:stream`: Xóa stream rác/orphan (chặn cứng không cho xóa 4 Core Streams nếu không có cờ recreate/reconcile).
    - `DELETE /api/v1/system/nats/streams/:stream/consumers/:consumer`: Xóa/Reset consumer bị kẹt hoặc consumer mồ côi để worker tự tái tạo hoặc recreate tại chỗ.
- **R3 — Nâng cấp Giao diện `NatsMonitor.tsx` (`cdc-cms-web`):**
  - Hiển thị trực quan % sử dụng Storage/Memory của JetStream và từng Stream (`Progress` bar theo `bytes / max_bytes`), gắn nhãn phân loại `CORE STREAM` vs `ORPHAN/CUSTOM`, cảnh báo `CONFIG DRIFT` hoặc `STALLED CONSUMER`.
  - Drawer Consumer nâng cấp: hiển thị `AckFloor Seq` → `Delivered Seq` → `Stream Last Seq`, thanh bão hòa `Ack Pending / Max Ack Pending`, `AckWait (s)`, `MaxDeliver`, `DeliverGroup`, `PushBound`, `Last Active` (tính thời gian trôi qua), và nút **Delete/Reset Consumer** có `Popconfirm`.
  - Drawer/Modal **Message Inspector (Soi Message thực tế)**: cho phép xem nhanh 10 messages đầu/cuối hoặc tra cứu theo `Seq` / `Subject` của Stream để biết chính xác bảng nào (`shadow_table`), bản ghi nào (`_source_ids`) đang nằm trong hàng đợi.
  - Tab mới **Data-Hub Command Channels (`cdc.cmd.*`)**: đối chiếu danh sách toàn bộ kênh lệnh RPC/Async của Data-Hub với các `subscriptions` thực tế của `cdc-worker` để phát hiện ngay kênh nào bị mất subscriber (`0 listeners`).
- **R4 — Kỷ luật Quy trình (`Propose-Only` tại S3):**
  - Hoàn thành đầy đủ hồ sơ thiết kế `09_tasks_solution_nats_monitor_v2.md`, kiểm định qua `/laya_router` (`DG1` & `DG2`), trình User và **DỪNG CHỜ `APPROVE`** trước khi sửa bất kỳ file source code nào.
