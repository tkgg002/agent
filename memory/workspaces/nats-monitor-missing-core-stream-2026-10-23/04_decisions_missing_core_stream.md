---
tier: T2
suffix: missing_core_stream
---

# 04 Architecture Decisions & Laya Flag Resolutions — missing_core_stream

## ADR-01: Giải trình các claim `FLAG` trong Decision Gate `DG1` / `DG2` (`20261023T094818`)
- **`S-RC`**: Đã xác minh trực tiếp trong source code:
  - [`batch_buffer_fanout.go:105-125`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go#L105-L125): `publishTransmuteTrigger` thử `bb.js.PublishMsg(msg)` tối đa 3 lần, khi thất bại sẽ fallback sang Core NATS qua subject `"cdc.cmd.transmute-shadow.core-fallback"`.
  - [`nats_client.go:26-29,159-164`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/natsconn/nats_client.go#L26-L29): `ReconnectHandler` không gọi lại `EnsureStreams`, và `EnsureStreams` return sớm ngay khi stream đầu tiên gặp lỗi.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:105-125`
- **`S-FIX`**: Đã xác minh trực tiếp trong source code:
  - [`nats_client.go:71-80`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/natsconn/nats_client.go#L71-L80): `EnsureStreams()` trong `cdc-cms-service` chỉ định nghĩa 3 stream (thiếu `CDC_TRANSMUTE`) và chưa được gọi lúc khởi động.
  - [`nats_monitor_handler.go:835-847`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go#L835-L847): `buildNATSGovernance` sinh cảnh báo `MISSING_CORE_STREAM` cho mọi stream trong `canonicalCoreStreamOrder` vắng mặt khỏi `seenCoreStreams`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/natsconn/nats_client.go:71-80`
- **`D-API`**: Đã xác minh trực tiếp trong source code:
  - [`nats_monitor_handler.go:154-175`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go#L154-L175): `NATSMonitorHandler.Overview` đọc `/jsz?streams=true&config=true&consumers=true` và có thể gọi `h.natsClient.EnsureStreams()` để tự động khôi phục Core Stream bị thiếu trước khi làm mới `/jsz` mà không làm thay đổi schema JSON trả về cho Frontend.
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go:154-175`
- **Ghi chú `DG3` (`20261023T095139`):** Cùng bộ đối chứng `S-RC`, `S-FIX`, `D-API` đã được xác minh và resolve ở chặng kiểm định phân tích (`DG3 = PASS`).

## ADR-02: Tinh chỉnh 5 Điểm Gợn Kỹ Thuật (Edge Cases & Concurrency) trong S3 Solution (`20261023T102835`)
1. **Lock-Free Network I/O & Drain Old Subscriptions trong `watchdogLoop` (`transmute_consumer_pool.go`):**
   - Nhả `p.mu.Unlock()` trước khi gọi `js.ConsumerInfo(..., nats.MaxWait(2*time.Second))` để không giữ mutex qua network RPC (không chặn `Stop()` khi Graceful Shutdown).
   - Bổ sung `drainSubsLocked()` dọn sạch toàn bộ `p.subs` cũ (`Drain()` / `Unsubscribe()`) trước khi tạo lại `WorkerCount` subscriptions mới, chống nhân bản goroutine.
2. **Non-Blocking `ReconnectHandler` (`reconciling atomic.Bool`) & Log chi tiết xung đột `UpdateStream` (`nats_client.go`):**
   - Bọc `EnsureStreams` trong goroutine riêng với `client.reconciling.CompareAndSwap(false, true)`.
   - Khi `UpdateStream` thất bại trên stream đã tồn tại (như `CDC_EVENTS`), log đầy đủ `existing_*` vs `desired_*` và tiếp tục tạo 3 stream còn lại.
3. **Cooldown `30s` (`lastHealAttemptUnix atomic.Int64`) cho Auto-Heal trong `GET /overview` (`nats_monitor_handler.go`):**
   - Dùng `CompareAndSwap` giới hạn tối đa 30s mới cho phép `tryAutoHealMissingCoreStreams` chạy 1 lần, tránh race condition khi nhiều tab trình duyệt cùng polling.
   - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go:154-175`
4. **Single Source of Truth cho `StreamConfig` (`DefaultTransmuteStreamConfig`):**
   - Dùng chung hàm dựng cấu hình chuẩn từ `pkgs/natsconn` giữa `EnsureStreams`, `TransmuteConsumerPool`, và `canonicalCoreStreams`, triệt tiêu nguy cơ `CONFIG_DRIFT`.
5. **Fast-Break `~1ms` khi gặp `nats.ErrNoStreamResponse` (`batch_buffer_fanout.go`):**
   - Thoát ngay vòng retry ở `attempt == 1` khi gặp `nats.ErrNoStreamResponse` hoặc `nats.ErrStreamNotFound` để chuyển sang Core NATS fallback tức thì mà không chờ `300ms`.
   - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:105-116`

## ADR-03: Khắc phục 5 Điểm Phản Biện Sâu ở Vòng Lặp Tự Chữa Lành `S5 → S3 → S4 → S5` (`20261023T114330`)
1. **Chống Zombie `_INBOX` trong môi trường Multi-Pod (`transmute_consumer_pool.go`):**
   - Lưu `boundDeliverSubject` và `boundConsumerCreated` khi bind consumer; trong `watchdogLoop()`, nếu `ConsumerInfo` trả về `err == nil` nhưng `!info.PushBound || info.Config.DeliverGroup != queueGroup || info.Config.DeliverSubject != boundDeliverSubject || !info.Created.Equal(boundConsumerCreated)` thì kích hoạt `needsHeal = true` để re-subscribe vào `_INBOX` mới.
2. **Không hủy `p.subs` khi `ConsumerInfo` bị Timeout mạng tạm thời (`transmute_consumer_pool.go`):**
   - Chỉ kích hoạt `needsHeal = true` từ lỗi `ConsumerInfo` khi `errors.Is(err, nats.ErrStreamNotFound) || errors.Is(err, nats.ErrConsumerNotFound)`, và trong `ensureSubscribedLocked()` đảm bảo stream/consumer sẵn sàng trước khi gọi `p.unsubscribeSubsLocked()`.
3. **Khóa `c.mu` bảo vệ `c.Conn` & `c.JS` (`nats_client.go`):**
   - Đồng bộ hóa việc gán `client.Conn`, `client.JS` trong `NewNatsClient` và truy cập `c.JS` trong `ensureSingleStream` bằng `c.mu` ở cả 2 repo.
4. **Phủ 100% Test Thực Tế cho mọi khẳng định (`batch_buffer_fanout_test.go`, `transmute_debounce_group_test.go`, `cdc-cms-service/pkgs/natsconn/nats_client_test.go`):**
   - Bổ sung test trực tiếp cho `publishTransmuteTrigger` (`< 50ms` fast-fallback), kịch bản External Consumer Recreation, và `preserveExistingSubjects` trên `CDC_EVENTS`.
