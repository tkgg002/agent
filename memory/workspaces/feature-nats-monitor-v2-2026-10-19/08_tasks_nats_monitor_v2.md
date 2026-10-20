# 08. Tasks Checklist — Nâng cấp Toàn diện `/nats-monitor` (`nats_monitor_v2`)

## Pha 1: Brain — Điều tra, Định nghĩa & Thiết kế Giải pháp (S1 → S3 → DG1/DG2)
- [x] **T1 (S1 Intake & S2 Read-Only Investigation):**
  - **[Mô tả]:** Khảo sát toàn diện hiện trạng `/nats-monitor` trên Frontend (`cdc-cms-web/src/pages/NatsMonitor.tsx`) và Backend (`cdc-cms-service/internal/api/system/nats_monitor_handler.go`), đối chiếu với kiến trúc NATS JetStream & Core NATS thực tế của `data-hub` (`centralized-data-service` & `cdc-cms-service`) và bài học sự cố `transmute-js-trigger-stall-2026-10-19`.
  - **[Context & Logs]:**
    - `cdc-cms-web/src/pages/NatsMonitor.tsx:1-1044`
    - `cdc-cms-service/internal/api/system/nats_monitor_handler.go:1-104`
    - `cdc-cms-service/internal/router/router.go:126-127, 316`
    - `cdc-cms-service/internal/server/server.go:182-326`
    - `centralized-data-service/pkgs/natsconn/nats_client.go:58-167`
    - `centralized-data-service/internal/server/server_setup.go:225-562`
  - **[Definition of Done]:** Xác định đầy đủ các lỗ hổng khiến `/nats-monitor` bị chung chung và khó kiểm soát kèm bằng chứng `file:line`.

- [x] **T2 (S3 Solution Design & Decision Gate DG1/DG2):**
  - **[Mô tả]:** Lập hồ sơ phân tích định nghĩa `13_analysis_nats_monitor_v2.md` và giải pháp kỹ thuật duy nhất `09_tasks_solution_nats_monitor_v2.md` (đủ 5 phần chuẩn `/brain-delegate`), chạy kiểm định qua `/laya_router` (`DG1` và `DG2`).
  - **[Context & Logs]:**
    - `agent/skills/brain-delegate/SKILL.md`
    - `agent/skills/laya-router/SKILL.md`
  - **[Definition of Done]:** `laya_gate.py run --stage dg1` và `--stage dg2` đạt `PASS`; trình User và dừng chờ `APPROVE`.

- [x] **T2b (Adversarial Plan Audit & Line-by-Line Implementation Spec):**
  - **[Mô tả]:** Phản biện toàn diện Plan V1, phát hiện và khắc phục 8 lỗ hổng thiết kế (`F1`–`F8`) trong `10_gap_analysis_nats_monitor_v2.md`, lập đặc tả code chi tiết từng dòng trong `12_implementation_plan_nats_monitor_v2.md`, cập nhật `09_tasks_solution_nats_monitor_v2.md` và kiểm định lại qua `/laya_router` (`DG1`, `DG2`, `DG3`).
  - **[Context & Logs]:**
    - `10_gap_analysis_nats_monitor_v2.md`
    - `12_implementation_plan_nats_monitor_v2.md`
    - `09_tasks_solution_nats_monitor_v2.md`
  - **[Definition of Done]:** Toàn bộ `DG1`, `DG2`, `DG3` cho Plan V2 đạt `PASS`; trình User và dừng chờ `APPROVE`.

---

## Pha 2: Muscle — Thực thi sau khi có `APPROVE` (S4 → S6)
- [x] **T3 (Backend Control Plane & Deep Telemetry APIs trong `cdc-cms-service`):**
  - **[Mô tả]:** Thực thi chính xác theo `12_implementation_plan_nats_monitor_v2.md`:
    1. Mở rộng `cdc-cms-service/internal/middleware/audit.go:306-346` (`defaultActionMap` + `deriveTarget` hỗ trợ `:stream`, `:consumer`, `:seq`).
    2. Tiêm `natsClient` vào `NewNATSMonitorHandler` tại `cdc-cms-service/internal/server/server.go:326`.
    3. Đăng ký route đọc `GET /system/nats/streams/:stream/messages` vào nhóm `shared` (`router.go:316`) và 5 route điều khiển vào nhóm `admin` kèm `destructiveChain` (`router.go:415`).
    4. Trong `nats_monitor_handler.go`:
       - `GetOverview`: giữ đúng 3 goroutines (`/varz`, `/jsz?streams=true&config=true&consumers=true`, `/connz?subs=detail`), đọc `subscriptions_list_detail`, tính toán `governance` (`stream_specs` 4 Core Streams, `command_channels` 19 kênh bắt buộc, `alerts`).
       - `GetStreamMessages`: đọc message không tiêu thụ (`JS.GetMsg` / `JS.GetLastMsg`), xử lý stream rỗng (`first_seq > last_seq`) và sparse sequence gaps trên `WorkQueuePolicy` (`maxProbes = limit * 4`, timeout `2.5s`).
       - `PurgeStream`: làm sạch stream (validate không truyền đồng thời `keep > 0` và `seq > 0`).
       - `DeleteStreamMessage`: xóa 1 poison message theo `seq`.
       - `UpdateStreamConfig`: cập nhật in-place `MaxBytes` và `MaxAge` (giữ nguyên `Subjects` & `Retention`).
       - `DeleteStream`: chặn cứng 4 Core Streams (`400 Bad Request`), chỉ xóa stream `ORPHAN`.
       - `DeleteConsumer`: kiểm tra `ConsumerInfo`, chặn cứng (`400 Bad Request`) nếu `PushBound == true` hoặc `NumWaiting > 0`.
  - **[Context & Logs]:** `cdc-cms-service/internal/middleware/audit.go:306-346`, `cdc-cms-service/internal/server/server.go:326`, `cdc-cms-service/internal/router/router.go:316, 415`, `cdc-cms-service/internal/api/system/nats_monitor_handler.go:1-104`.
  - **[Definition of Done]:** `go build ./...` và `go test ./internal/api/system/...` PASS; verify runtime thật trên `:8083` và NATS `:18222`/`:14222`.

- [x] **T4 (Frontend NATS Control Center UI trong `cdc-cms-web`):**
  - **[Mô tả]:** Thực thi chính xác theo `12_implementation_plan_nats_monitor_v2.md`:
    1. Mở lại menu `/nats-monitor` trong `cdc-cms-web/src/App.tsx:139-143`.
    2. Nâng cấp `cdc-cms-web/src/pages/NatsMonitor.tsx`:
       - Hàng KPI & Cảnh báo thông minh (`alerts`).
       - Tab **JetStream Streams & Governance**: hiển thị vai trò nghiệp vụ, thanh dung lượng `bytes / max_bytes` (xử lý `max_bytes <= 0` $\rightarrow$ `UNLIMITED`, `messages === 0` $\rightarrow$ `Trống (Next: #seq)`), trạng thái `SYNCED` / `DRIFT` / `ORPHAN`, các nút **Soi Message (Peek)**, **Consumers**, **Chỉnh Quota**, **Purge**, **Xóa Stream Rác**.
       - Drawer **Consumer Diagnostics & Control**: hiển thị trạng thái `PushBound`, tiến độ `AckFloor → Delivered / StreamLastSeq`, thanh bão hòa `AckPending / MaxAckPending`, `AckWait (s)`, `DeliverGroup`, `LastActive`, nút **Xóa Consumer Mồ côi**.
       - Drawer **Message Inspector**: xem danh sách message theo `latest` / `first` / `seq` / `subject`, xem `Headers` & `Payload JSON` và nút **Xóa Poison Message**.
       - Tab **Kênh lệnh Data-Hub (`cdc.cmd.*`)**: bảng đối soát 19 kênh lệnh Core NATS bắt buộc (`READY` / `NO_LISTENER`, `Queue Group`, `Active Listeners`, `Worker CIDs`, `Msgs Delivered`).
       - Tab **Active Connections & Subscriptions**: hiển thị danh sách kết nối kèm `RTT` và expandable row xem chi tiết `subscriptions_list_detail`.
       - Tích hợp `ConfirmDestructiveModal` cho toàn bộ thao tác xóa/purge.
  - **[Context & Logs]:** `cdc-cms-web/src/App.tsx:139-143`, `cdc-cms-web/src/pages/NatsMonitor.tsx:1-1044`, `cdc-cms-web/src/components/ConfirmDestructiveModal.tsx:1-143`.
  - **[Definition of Done]:** `npx tsc --noEmit` và `npm run build` PASS; không vi phạm Rules of Hooks; layout chuẩn Ant Design.

- [x] **T5 (S5 Mandatory Final Adversarial Audit & S6 Close):**
  - **[Mô tả]:** Thực hiện 6 bước QC phản biện cuối workflow (`git diff` line-by-line, chống suy diễn smoke test, trace runtime thật, `DG3` qua `laya_gate.py`, `/security-agent`), xuất `audit_report_nats_monitor_v2.md`, `06_validation_nats_monitor_v2.md`, `11_report_nats_monitor_v2.md` và chạy `verify_governance.py`.
  - **[Definition of Done]:** Toàn bộ Gates `G1–G8` và `verify_governance.py` đạt `PASSED` 🟢.
