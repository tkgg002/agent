# 08 Tasks — missing_core_stream

- [x] **T1 (S1 Intake):** Khởi tạo bộ hồ sơ workspace `nats-monitor-missing-core-stream-2026-10-23` (`01_requirements_missing_core_stream.md`, `05_progress_missing_core_stream.md`, `08_tasks_missing_core_stream.md`).
  - Context: `agent/GEMINI.md:68-72`
  - DoD: Đầy đủ 3 file S1 trước khi điều tra sâu và lập giải pháp.
- [x] **T2 (S2 Investigate):** Truy vết `MISSING_CORE_STREAM` trong `cdc-cms-web`, `cdc-cms-service` và `centralized-data-service` để tìm nguyên nhân gốc rễ (`file:line` `VERIFIED`) vì sao `/nats-monitor` báo thiếu `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG` trong khi `CDC_TRANSMUTE` vẫn chạy bình thường.
  - Context: `data-hub/cdc-cms-web/src/pages/NatsMonitor.tsx:1605-1638`, `data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go:674-847`, `data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:104-130`, `data-hub/centralized-data-service/pkgs/natsconn/nats_client.go:63-166`
  - DoD: Xác định 100% `VERIFIED` nguyên nhân kèm `file:line` cụ thể.
- [x] **T3 (S3 Solution & DG1/DG2):** Thiết kế 1 phương án kỹ thuật tối ưu duy nhất trong `09_tasks_solution_missing_core_stream.md` kèm Code Demo và chạy `laya_gate.py run --stage dg1` đạt `PASS`.
  - Context: `09_tasks_solution_missing_core_stream.md`, `04_decisions_missing_core_stream.md`
  - DoD: `laya_gate.py` trả `PASS`, cập nhật `05_progress_missing_core_stream.md` và `verify_governance.py --stage S3` đạt `PASSED`, trình anh chờ `APPROVE`.
- [x] **T4 (S4 Execute — Sau khi anh `APPROVE`):** Thực thi cơ chế Multi-Layer Stream Self-Healing (đã tinh chỉnh 5 điểm concurrency & kiến trúc):
  1. `centralized-data-service/pkgs/natsconn/nats_client.go:26-261`: Export `Default*StreamConfig()`, bỏ Fail-Fast trong `EnsureStreams`, log rõ `existing` vs `desired` khi `UpdateStream` lỗi, khởi tạo `client` trước `nats.Connect` và chạy `EnsureStreams` bất đồng bộ trong `ReconnectHandler` với `reconciling atomic.Bool`.
  2. `centralized-data-service/internal/handler/master/transmute_consumer_pool.go:62-288`: Bổ sung `unsubscribeSubsLocked()` dọn sạch `p.subs` cũ trước khi re-subscribe (giữ `Drain()` cho `Stop()`), dùng `natsconn.DefaultTransmuteStreamConfig()`, và nhả `p.mu.Unlock()` trước khi gọi `js.ConsumerInfo(..., nats.MaxWait(2*time.Second))` trong `watchdogLoop()`.
  3. `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:108-120`: Thoát ngay vòng retry (`break`) ở lần thử đầu tiên nếu gặp `nats.ErrNoStreamResponse`, `nats.ErrNoResponders`, hoặc `nats.ErrStreamNotFound` (`~1ms` fallback).
  4. `cdc-cms-service/pkgs/natsconn/nats_client.go:25-261`, `internal/server/server.go:124-126`, `internal/api/system/nats_monitor_handler.go:51-86,189-197,233-273`: Đồng bộ Single Source of Truth cho 4 Core Streams, gọi `EnsureStreams()` lúc boot & reconnect, và bọc `tryAutoHealMissingCoreStreams` với cooldown `30s` (`lastHealAttemptUnix atomic.Int64`).
  - Context: `09_tasks_solution_missing_core_stream.md`
  - DoD: Integration test với NATS thật (`evidence_run.py`: `EV-20261023T104653-8777`, `EV-20261023T104711-61bc`, `EV-20261023T104838-3f54`, `EV-20261023T105007-f372`) chứng minh xóa 3 stream `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG` sẽ tự động được khôi phục không rò rỉ goroutine, fast-fallback `~1ms` khi stream vắng mặt, và cảnh báo `MISSING_CORE_STREAM` biến mất.
- [x] **T5 (S5 Audit & S6 Close):** Thực hiện 6 bước Adversarial Audit, chạy `DG3` (`laya_gate.py`, `ts=20261023T105527`, `gate=PASS`) và `verify_governance.py --workspace nats-monitor-missing-core-stream-2026-10-23 --suffix missing_core_stream` đạt `PASSED`.
  - Context: `09_tasks_solution_missing_core_stream.md`, `audit_report_missing_core_stream.md`
  - DoD: Pass đủ G1-G8 và `verify_governance.py` `PASSED`.
- [x] **T6 (S5 → S3 → S4 → S5 Autonomous Self-Healing & Deep Adversarial Audit):** Phản biện từng dòng `git diff` và toàn bộ quá trình thực hiện, khắc phục dứt điểm các điểm phát hiện khi Audit sâu:
  1. `centralized-data-service/internal/handler/master/transmute_consumer_pool.go`: Lưu `boundDeliverSubject` & `boundConsumerCreated` để phát hiện consumer bị tạo lại bởi Pod khác (`DeliverSubject`/`Created` thay đổi hoặc `!PushBound`), chỉ `needsHeal` trên `ErrStreamNotFound`/`ErrConsumerNotFound` (không phá `p.subs` khi timeout mạng tạm thời), và đảm bảo stream/consumer sẵn sàng trước khi gọi `unsubscribeSubsLocked()`.
  2. `centralized-data-service/pkgs/natsconn/nats_client.go` & `cdc-cms-service/pkgs/natsconn/nats_client.go`: Bảo vệ việc gán và đọc `c.Conn`/`c.JS` bằng `c.mu` chống data race với `ReconnectHandler`.
  3. `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go`: Truyền `nats.RetryAttempts(0)` vào `bb.js.PublishMsg` để tắt ngủ ngầm `2x250ms` của `nats.go` khi gặp `ErrNoResponders`; bổ sung test trực tiếp `batch_buffer_fanout_test.go` (`publishTransmuteTrigger` fast-fallback `< 50ms`), kịch bản External Consumer Recreation trong `transmute_debounce_group_test.go`, và `cdc-cms-service/pkgs/natsconn/nats_client_test.go` (`preserveExistingSubjects = true`).
  - Context: `09_tasks_solution_missing_core_stream.md`, `04_decisions_missing_core_stream.md`, `audit_report_missing_core_stream.md`
  - DoD: Đã hoàn tất `DG1` (`PASS`, `ts=20261023T115409`) -> sửa code & test -> `evidence_run.py` (`EV-20261023T115458-6a9e`, `EV-20261023T115543-bbdf`, `EV-20261023T115558-b5f3` đều `exit=0`) -> `DG3` (`PASS`, `ts=20261023T115653`) -> cập nhật `audit_report_missing_core_stream.md`.

