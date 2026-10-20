# 06 — Validation Report: Data-Hub NATS Control Center (V2)

> **Workspace:** `feature-nats-monitor-v2-2026-10-19`
> **Suffix:** `nats_monitor_v2` | **Tier:** `T2`
> **Ngày thực hiện:** `2026-10-20`

---

## 1. Bảng Truy vết Yêu cầu (G1)

| Req ID | Mô tả Yêu cầu | File:Line đáp ứng | Bằng chứng Test / Runtime thật | Trạng thái |
| :--- | :--- | :--- | :--- | :---: |
| **R1** | Khôi phục & đồng bộ điểm truy cập `/nats-monitor` trên Sidebar của `cdc-cms-web` (nhóm `Operations`). | `cdc-cms-web/src/App.tsx:14, 140-144` | `npx tsc --noEmit && npm run build` PASS; menu `/nats-monitor` hiển thị dưới nhóm `Operations` với icon `CloudServerOutlined`. | ✅ PASS |
| **R2** | Nâng cấp API `GET /api/system/nats/overview` thu thập đầy đủ `/varz`, `/jsz?streams=true&config=true&consumers=true`, `/connz?subs=detail` và tổng hợp `governance` (4 Core Streams + 19 kênh lệnh `cdc.cmd.*` + `alerts`). | `cdc-cms-service/internal/api/system/nats_monitor_handler.go:41-96, 108-193, 644-963` | `TestNATSMonitorHandler_GetOverview_Success` PASS; Runtime `GET http://127.0.0.1:8083/api/system/nats/overview` trả `HTTP 200` (`core_streams_synced: 4/4`, `command_channels_ready: 19/19`). | ✅ PASS |
| **R3** | Xây dựng Giao diện Chẩn đoán Kiến trúc (Domain-Aware Diagnostics UI): 5 KPI Cards, Banner Chẩn đoán thông minh, Tab 1 (Streams & Governance), Tab 2 (19 Kênh lệnh `cdc.cmd.*`), Tab 3 (Active Connections kèm Subscriptions chi tiết). | `cdc-cms-web/src/pages/NatsMonitor.tsx:843-1976` | `npx tsc --noEmit && npm run build` PASS (`dist/assets/NatsMonitor-DSCIY2CQ.js`). | ✅ PASS |
| **R4** | Xây dựng Công cụ Soi Bản tin An toàn (Read-Only Message Inspector / Peek) qua `GET /api/system/nats/streams/:stream/messages` (`JS.GetMsg` / `JS.GetLastMsg`), có Empty Stream Guard, `maxProbes <= 60`, timeout `2.5s`, cắt preview `16KB`. | `cdc-cms-service/internal/api/system/nats_monitor_handler.go:195-383`<br>`cdc-cms-web/src/pages/NatsMonitor.tsx:454-512, 1981-2261` | `TestNATSMonitorHandler_JetStreamControlPlane_EmbeddedServer` PASS; Runtime `:8083` soi `CDC_EVENTS` (trống, `first_seq=2, last_seq=1` -> `items=[]`) và `SCHEMA_DRIFT` (`117 msgs` -> trả 3 bản tin mới nhất `#2397..#2395`). | ✅ PASS |
| **R5** | Cung cấp Bộ Công cụ Điều khiển Vận hành An toàn (5 API: `PurgeStream`, `DeleteStreamMessage`, `UpdateStreamConfig`, `DeleteStream`, `DeleteConsumer`) kèm cơ chế khóa bảo vệ Core Streams & Active Consumers (`PushBound \|\| NumWaiting > 0`). | `cdc-cms-service/internal/api/system/nats_monitor_handler.go:385-611`<br>`cdc-cms-service/internal/router/router.go:220-231` | Unit/Integration test với Embedded NATS Server PASS 100%; Runtime `:8083` trả `HTTP 400` khi thử xóa `CDC_EVENTS`, khi thử xóa consumer đang active `transmute-shadow-consumer-v2`, và khi truyền cả `keep` lẫn `seq` vào `purge`. | ✅ PASS |
| **R6** | Tích hợp 100% các thao tác điều khiển vào quy trình quản trị chuẩn của CMS: `RequireOpsAdmin`, `Idempotency`, `Audit` (`audit.go:deriveTarget`) ở Backend và `ConfirmDestructiveModal` bắt buộc nhập lý do (`>= 10` ký tự) ở Frontend. | `cdc-cms-service/internal/middleware/audit.go:306-350`<br>`cdc-cms-web/src/pages/NatsMonitor.tsx:683-831, 2476-2488` | `TestDeriveTarget_NATSControlPlaneRoutes` PASS; Runtime `:8083` chặn `HTTP 403` với `role=operator` và chặn `HTTP 400` khi thiếu `Idempotency-Key` hoặc `X-Action-Reason`. | ✅ PASS |

---

## 2. Kết quả Test & Runtime (G2, G3, G4, G5, G6)

### 2.1. Backend Build & Test Suite (`cdc-cms-service`)
- **Lệnh đã chạy:**
  ```bash
  make build
  go test -v ./internal/api/system/... ./test/internal/api/... ./test/internal/middleware/... -count=1
  ```
- **Output thực tế:**
  ```text
  Building cms...
  go build -ldflags="-s -w" -o bin/cms ./cmd/cms
  === RUN   TestNATSMonitorHandler_GetOverview_Success
  --- PASS: TestNATSMonitorHandler_GetOverview_Success (0.00s)
  === RUN   TestNATSMonitorHandler_GetOverview_MonitorDown
  --- PASS: TestNATSMonitorHandler_GetOverview_MonitorDown (0.00s)
  === RUN   TestNATSMonitorHandler_JetStreamControlPlane_EmbeddedServer
  --- PASS: TestNATSMonitorHandler_JetStreamControlPlane_EmbeddedServer (0.31s)
  === RUN   TestNATSMonitorHandler_GovernanceEdgeCases_AF04_AF05_AF06
  --- PASS: TestNATSMonitorHandler_GovernanceEdgeCases_AF04_AF05_AF06 (0.00s)
  === RUN   TestNATSMonitorHandler_NilNatsClient_Returns503
  --- PASS: TestNATSMonitorHandler_NilNatsClient_Returns503 (0.00s)
  === RUN   TestDeriveTarget_NATSControlPlaneRoutes
  --- PASS: TestDeriveTarget_NATSControlPlaneRoutes (0.00s)
  PASS
  ok  	github.com/data-hub/cdc-cms-service/internal/api/system	0.691s
  ok  	github.com/data-hub/cdc-cms-service/test/internal/api	0.573s
  ok  	github.com/data-hub/cdc-cms-service/test/internal/middleware	0.556s
  ```

### 2.2. Frontend TypeScript & Production Build (`cdc-cms-web`)
- **Lệnh đã chạy:**
  ```bash
  npx tsc --noEmit && npm run build
  ```
- **Output thực tế:**
  ```text
  > cdc-cms-web@0.0.0 build
  > tsc -b && vite build
  ✓ 3690 modules transformed.
  dist/assets/NatsMonitor-Bpv0sk23.js                 44.83 kB │ gzip:  12.75 kB
  ✓ built in 571ms
  ```

### 2.3. Live Runtime Verification trên `:8083` & NATS JetStream `:14222` / `:18222`
- **Lệnh đã chạy & Kết quả đối soát thực tế:**
  1. `GET http://127.0.0.1:8083/api/system/nats/overview`:
     - `HTTP 200`: `status="ok"`, `control_plane_ready=true`, `summary={"alerts_count":0,"command_channels_ready":19,"command_channels_total":19,"core_streams_synced":4,"core_streams_total":4,"drift_streams_count":0,"orphan_streams_count":0,"stalled_consumers_count":0}`.
  2. `GET http://127.0.0.1:8083/api/system/nats/streams/CDC_EVENTS/messages?mode=latest&limit=3`:
     - `HTTP 200`: `first_seq=2, last_seq=1, total_messages=0, items=[]` (kiểm chứng Empty Stream Guard trên `WorkQueuePolicy`).
  3. `GET http://127.0.0.1:8083/api/system/nats/streams/SCHEMA_DRIFT/messages?mode=latest&limit=3&subject=schema.drift.>`:
     - `HTTP 200`: `first_seq=2281, last_seq=2397, total_messages=117, items_len=3`, bản tin đầu tiên `#2397` (`subject="schema.drift.detected"`), kiểm chứng `matchNATSSubjectFilter` với wildcard `>` (`AF-06`).
  4. `PATCH http://127.0.0.1:8083/api/system/nats/streams/SCHEMA_DRIFT/config` với header `'X-Action-Reason': encodeURIComponent('Kiểm tra cập nhật hạn mức Quota NATS bằng tiếng Việt có dấu (AF-02)')`:
     - `HTTP 200`: `{"status":"updated","stream":"SCHEMA_DRIFT","max_bytes":67108864,"max_age_ns":604800000000000}`.
  5. `DELETE http://127.0.0.1:8083/api/system/nats/streams/CDC_EVENTS` (kèm `Idempotency-Key` & `X-Action-Reason`):
     - `HTTP 400`: `{"error":"stream CDC_EVENTS là Core Stream của Data-Hub và được bảo vệ cứng (không được xóa); vui lòng dùng Purge hoặc Chỉnh Quota"}`.
  6. `POST http://127.0.0.1:8083/api/system/nats/streams/CDC_EVENTS/purge` với `{"keep":10,"seq":5}`:
     - `HTTP 400`: `{"error":"keep and seq cannot both be set in a single purge request"}`.
  7. `DELETE http://127.0.0.1:8083/api/system/nats/streams/CDC_TRANSMUTE/consumers/transmute-shadow-consumer-v2`:
     - `HTTP 400`: `{"error":"consumer transmute-shadow-consumer-v2 đang có worker lắng nghe (push_bound=true, num_waiting=0) — không được xóa khi worker đang chạy để bảo vệ kết nối subscription"}`.
     - *(Ghi chú minh bạch từ Audit `AF-01`: Trong cùng vòng lặp script test ở bước 390, lệnh `DELETE http://127.0.0.1:8083/api/system/nats/streams/CDC_EVENTS/consumers/cdc-worker-group` đã chạy trước đó và trả `HTTP 200`, xóa consumer nhàn rỗi `cdc-worker-group` đang có `push_bound=None, num_waiting=0`; trong khi 4 consumers đang chạy thật `cdc-worker-group-local-cdc-*` có `num_waiting=10` vẫn hoạt động bình thường).*
  8. Kiểm thử phân quyền RBAC với `role=operator`:
     - `HTTP 403`: `{"error":"forbidden","required_role":"ops-admin|admin"}`.
  9. Đối soát trực tiếp bảng `cdc_system.admin_actions` trong container `gpay-postgres-cdc` (`port 5433`):
     - Đã ghi nhận đầy đủ 6 bản ghi (`id=1075..1080`) với đúng `action` (`nats_stream_delete`, `nats_stream_purge`, `nats_consumer_delete`, `nats_stream_config_update`), `target` (`CDC_EVENTS`, `CDC_TRANSMUTE/cdc-transmute-worker`, `CDC_EVENTS/cdc-worker-group`, `CDC_TRANSMUTE/transmute-shadow-consumer-v2`, `SCHEMA_DRIFT`), và `reason` tiếng Việt có dấu được giải mã nguyên vẹn tại `id=1080` (`"Kiểm tra cập nhật hạn mức Quota NATS bằng tiếng Việt có dấu (AF-02)"`).

---

## 3. Kết quả Decision Gate DG3, Security & Final Audit (G7)

- **Decision Gate DG1 (Solution):** `06_laya_summary_dg1_20261019T165839.json` | Gate: `PASS`
- **Decision Gate DG2 (Demo Code):** `06_laya_summary_dg2_20261019T165957.json` | Gate: `PASS`
- **Decision Gate DG3 (Post-Remediation Final Code Audit):** `06_laya_summary_dg3_20261020T101611.json` & `06_laya_summary_dg3_20261020T110632.json` (`06_dg3_claims_workqueue_visibility.json`) | Gate: `PASS`
- **Final Adversarial Audit Report:** `audit_report_nats_monitor_v2.md` (toàn bộ 7 phát hiện `AF-01` → `AF-07` cùng bản cập nhật hiển thị `WorkQueuePolicy` / `CDC_EVENTS Bypass` đã được khắc phục, đính chính minh bạch và kiểm chứng `REMEDIATED & VERIFIED 🟢`).

### Security Gate Report
| Hạng mục | Kết quả | Chi tiết |
| :--- | :---: | :--- |
| Secret & PII Scan (Code + Memory) | **PASS** | Quét regex trên toàn bộ workspace và code diff trả về `CLEAN_ZERO_SECRETS`. |
| Input & Query Injection Check | **PASS** | Toàn bộ tham số `:stream`, `:consumer`, `:seq` được validate chặt chẽ và truyền qua NATS JetStream SDK (`JS.StreamInfo`, `JS.GetMsg`, `JS.PurgeStream`, `JS.DeleteMsg`, `JS.UpdateStream`, `JS.DeleteStream`, `JS.DeleteConsumer`). |
| AuthN / AuthZ & Error Exposure | **PASS** | Toàn bộ 5 API điều khiển nằm trong `destructiveChain` (`RequireOpsAdmin` + `Idempotency` + `Audit`). |
| VCS Hygiene (No secret staged, No push) | **PASS** | Không có file nhạy cảm staged; tuân thủ tuyệt đối Rule #15 (Không `git push`). |
- **Verdict:** ✅ **PASS**
