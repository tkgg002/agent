---
tier: T2
type: Bugfix
project: data-hub / centralized-data-service
---
# 01 Requirements — JetStream trigger → Master ngừng do ENOSPC & Cấu hình giới hạn dung lượng + xóa ngay khi xử lý xong

## Yêu cầu User (2026-10-19)
1. **Triệu chứng ban đầu (PRODUCTION):**
   - Sau khi chạy "sync ngay" cho `master_payment_bill_service.payment_bills` (`shadow_payment_bill`, binding `#86`, `conn: default_master`, hoàn tất `2026-10-16 15:04:51`, `5,958,909` rows, trace `eb59ab2d…`), toàn bộ tác vụ NATS JetStream trigger → Master bị ngừng từ `13:43:24 16/10/2026`, và **không có bất cứ 1 record nào được ghi vào Master từ ngày 16/10 tới nay (19/10)**.
   - Trên `/nats-monitor` PROD: stream `CDC_TRANSMUTE` giữ ~47,612 messages (~1 GB), consumer `transmute-shadow-consumer-v2` có `Ack Pending = 0`, `Pending = 0`, `Stream Seq = #47612` (tạo lúc `16:54:54 15/10/2026`).
   - Log worker PROD báo lỗi liên tục:
     `{"level":"error","msg":"failed to publish transmute trigger to JetStream, falling back to core NATS","error":"nats: write /data/jetstream/$G/streams/CDC_TRANSMUTE/msgs/135.blk: no space left on device"}`
2. **Yêu cầu thực thi của User:**
   - Đặt giới hạn dung lượng (`MaxBytes`) cho stream.
   - Cho xóa message ngay sau khi xử lý xong (`WorkQueuePolicy` — xóa ngay khi `Ack()`/`Term()`) thay vì giữ 24h (`LimitsPolicy`).
   - Giải thích rõ vì sao từ 16/10 tới nay không có bất cứ 1 record nào được ghi vào Master và sửa luôn lỗ hổng fallback core NATS bị rớt 100% trigger khi JetStream lỗi.
3. **Yêu cầu bổ sung của User (Đảm bảo an toàn cho `WorkQueuePolicy`):**
   - **Chỉ dùng 1 consumer group:** Xác nhận pool consumer cùng chia sẻ `transmute-shadow-consumer-v2` trên `cdc.cmd.transmute-shadow`.
   - **Bắt buộc bắt lỗi và gọi `msg.Nak()` (kèm delay) — Cấm gọi `msg.Ack()` bừa bãi khi lỗi ghi Master:** Nếu tra cứu binding lỗi hoặc ghi DB Master thất bại (`runDebouncedGroup` / `processSubBatch`), tuyệt đối không được `msg.Ack()`, phải trả lỗi về `TransmuteConsumerPool` để gọi `msg.NakWithDelay(...)`. Chỉ gọi `msg.Ack()` sau khi đã ghi Master thành công.
   - **Cấu hình `MaxDeliver = 5` & xử lý Poison Pill / DLQ:** Khi lỗi vĩnh viễn (schema/dữ liệu sai) đã retry hết `MaxDeliver` (5 lần), dừng redeliver (`msg.Term()`) và ghi nhận lỗi rõ ràng để không lặp vô tận.
4. **Yêu cầu tự động suy diễn mốc `lastGpayID` từ bảng Master khi bấm "Chạy ngay (tiếp theo)":**
   - Giữ nguyên việc xóa `LastCursorJSON` về `{}` ở cuối `Run()` để nút **"Chạy ngay (toàn bộ)"** (`runType == "full"`) không bao giờ bị kẹt ở mốc cũ.
   - Khi nhấn **"Chạy ngay (tiếp theo)"** (`runType != "full"`, `len(chunkIDs) == 0`) mà `lastGpayID == 0`, tự động suy diễn mốc `lastGpayID` từ bảng Master (`inferLastGpayIDFromMaster`) và chạy tiếp từ mốc đó (ví dụ từ 4 triệu lên 4.1 triệu).

## Phạm vi
- `centralized-data-service/pkgs/natsconn/nats_client.go` (`EnsureStreams`)
- `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go` (`publishTransmuteTrigger`)
- `centralized-data-service/internal/server/server_setup.go` (core NATS fallback subscription)
- `centralized-data-service/internal/handler/master/transmute_consumer_pool.go` (`processMessage`: `NakWithDelay` & `MaxDeliver` check)
- `centralized-data-service/internal/handler/master/debounce.go` & `transmute_handler.go` (`HandleTransmuteShadowTask` End-to-End Ack/Nak sau khi ghi Master)
- `centralized-data-service/internal/service/master/transmuter.go` (`Run` & `inferLastGpayIDFromMaster`)

## Definition of Done
- Root cause có bằng chứng runtime từ PROD + `file:line`.
- Giải pháp kỹ thuật 1 phương án duy nhất trong `09_tasks_solution_js_stall.md` đạt `PASS` qua Decision Gate `DG1` & `DG2`.
- Sau khi User `APPROVE`: triển khai code, pass unit/integration test + verify thực tế trên NATS JetStream broker (`WorkQueuePolicy` xóa message ngay sau `Ack()`, `NakWithDelay` giữ lại message để redeliver khi ghi Master lỗi, dừng sau `MaxDeliver = 5` lần, `MaxBytes` giới hạn dung lượng, core NATS fallback hoạt động, và `"Chạy ngay (tiếp theo)"` tự suy diễn `lastGpayID` từ bảng Master để chỉ chạy phần thiếu).
