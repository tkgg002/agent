# 13 Analysis — js_stall (Audit, Propose-Only)

## 1. VERIFIED (runtime local)
- SigNoz local 16/10: không có log chứa `eb59ab2d`, `5958909`, `transmute complete`; transmute realtime vẫn chạy sau 13:43:24 (vd 06:45:19 UTC `activity completed ... master_payment_bill_service.payment_bills`).
- `cdc_dw` local: payment_bills = master_binding 46; schedule immediate lần cuối 2026-09-29; `transmute_jobs` không có job từ 15/10.
- NATS local: consumer `transmute-shadow-consumer-v2` pending=0, ack_pending=0 (trạng thái sau restart).
=> Sự cố thuộc môi trường khác (server), chưa có dữ liệu runtime.

## 2. Luồng code (VERIFIED từ source)
- Sync ngay: core `cdc.cmd.transmute` → `HandleTransmute` → goroutine `svc.Run` full sync, timeout 24h. REF: centralized-data-service/internal/handler/master/transmute_handler.go:231-332
- Realtime: JetStream pool (`AckWait 30s`, `MaxDeliver 5`) → `HandleTransmuteShadow` (DB lookup timeout 5s, lỗi → return, message vẫn Ack) → core publish `cdc.cmd.transmute` → `TableDebouncer`. REF: internal/server/server_setup.go:332-362; transmute_handler.go:101-161; transmute_consumer_pool.go:126-153

## 3. HYPOTHESIS (chưa xác minh)
- H1 — Cạn pool systemDB: lookup 5s timeout → message Ack mà không fan-out → mất toàn bộ trigger master (chỉ warn log). REF: transmute_handler.go:105-134, transmute_consumer_pool.go:142-152
- H2 — Debouncer bypass semaphore: `flushCh` (cap 100) đầy → `default: go td.runTransmute(...)` không giới hạn concurrency → nổ goroutine/DB conn. REF: internal/handler/master/debounce.go:128-135
- H3 — Core NATS slow consumer bị drop im lặng: không có `nats.ErrorHandler`. REF: pkgs/natsconn/nats_client.go:19-29

## 4. Dữ liệu cần để chốt
- Log worker môi trường sự cố 16/10 13:40–13:50 theo keywords tại 08_tasks T4.
- Trạng thái NATS consumer + pg_stat_activity tại thời điểm đó.
- Trigger có tự hồi phục sau 15:04:51 hay cần restart worker.

> ⚠️ §1 là số liệu LOCAL — không dùng làm bằng chứng cho PROD (User xác nhận sự cố ở production).

## 5. ROOT CAUSE — VERIFIED (log PROD do User cung cấp, ts 1792377018 = 2026-10-19 09:30:18 +07)
Log: `failed to publish transmute trigger to JetStream, falling back to core NATS ... write /data/jetstream/$G/streams/CDC_TRANSMUTE/msgs/135.blk: no space left on device` (bảng `transhis`, `bank_requests_bvb`).

- RC1 — Ổ JetStream của NATS PROD đầy (ENOSPC tại `/data/jetstream`).
- RC2 — Stream không có trần dung lượng: `CDC_TRANSMUTE` tạo với `LimitsPolicy`, `MaxAge 24h`, không đặt `MaxBytes`/`MaxMsgs` → giữ MỌI message (kể cả đã ack) 24h; payload chứa `events[].RawData`. `CDC_EVENTS`/`SCHEMA_DRIFT`/`SCHEMA_CONFIG` MaxAge 7 ngày, cũng không MaxBytes. REF: centralized-data-service/pkgs/natsconn/nats_client.go:77-96; internal/handler/shadow/batch_buffer_fanout.go:73-81
- RC3 — Fallback core NATS rơi vào hư không: core subscriber `cdc.cmd.transmute-shadow` CHỈ đăng ký khi `natsClient.JS == nil`; PROD có JS → không ai nhận. Core `Publish` không có subscriber vẫn trả nil → không log lỗi → toàn bộ trigger master bị mất im lặng. REF: internal/server/server_setup.go:359-362; batch_buffer_fanout.go:118-124
- RC4 — Các publisher khác (`sinkworker`, `batch_transform_handler`, `bridge_handler`) publish core thẳng vào `cdc.cmd.transmute-shadow`: chỉ đến được worker qua việc stream capture subject → khi disk đầy cũng mất. REF: internal/sinkworker/worker.go:385; internal/handler/shadow/batch_transform_handler.go:610; internal/handler/source/bridge_handler.go:590
- HYPOTHESIS: mốc 13:43:24 16/10 = thời điểm disk đầy. Luồng "sync ngay" (full sync) KHÔNG publish trigger (transmute_handler.go:231-332) nên trùng giờ chưa chứng minh nhân quả; cần `df`/NATS server log PROD để chốt.

## 6. AUDIT BỔ SUNG (2026-10-19) — Hành vi khi nhấn Transmute lại (Shadow 4.1M records, Master thiếu 100k records)
- **Kết luận (`VERIFIED`):** Khi nhấn nút Transmute trên UI (dù là **"Chạy ngay (tiếp theo)"** `run_type=""` hay **"Chạy ngay (toàn bộ)"** `run_type="full"` tại `MasterRegistry.tsx:668-696, 1540-1573`, hoặc **Run-now** tại `TransmuteSchedules.tsx:183-191`), hệ thống hiện tại **CHẠY LẠI TỪ ĐẦU (`_gpay_id > 0`, quét lại toàn bộ 4.1 triệu records)** chứ **KHÔNG chạy tiếp từ mốc 4 triệu**.
- **Bằng chứng `file:line` (`VERIFIED`):**
  1. **F1 (`P1`, `VERIFIED`) — Checkpoint `last_gpay_id` bị xóa trắng về `{}` ngay sau khi Full Sync hoàn tất:**
     - Trong lúc chạy vòng lặp `Run()`, sau mỗi batch 2,000 dòng, code lưu `{"last_gpay_id": lastGpayID}` vào `cdc_system.sync_runtime_state.last_cursor_json` (`transmuter.go:494-500`).
     - Tuy nhiên, ngay khi thoát vòng lặp (kết thúc 1 lần chạy Full Sync thành công với `len(onlySourceIDs) == 0`), tại `centralized-data-service/internal/service/master/transmuter.go:511-516`, code ghi đè `item.LastCursorJSON = []byte("{}")`. Do đó `last_cursor_json` chỉ đóng vai trò crash-recovery tạm thời khi job đang chạy dở bị ngắt giữa chừng; một khi job trước đó đã `COMPLETED`, `last_cursor_json` luôn là `{}` (`lastGpayID = 0`).
  2. **F2 (`P1`, `VERIFIED`) — Luồng Realtime CDC không cập nhật `last_cursor_json`:**
     - Cả `RunEvents` (`transmuter.go:546-723`) lẫn `Run` với `onlySourceIDs` (`transmuter.go:495`, điều kiện `if len(chunkIDs) == 0`) đều không ghi `last_gpay_id` vào `sync_runtime_state.last_cursor_json`.
  3. **F3 (`P2`, `VERIFIED`) — Đếm tiến độ `totalShadowRows` luôn quét từ `cursor = 0` và OCC `clockSkewToleranceMs = 2000` vẫn `DO UPDATE` lại 4 triệu dòng cũ:**
     - Tại `transmuter.go:205-211`, `t.countShadowRows(ctx, masterRow, 0)` được gọi trước khi đọc checkpoint (`transmuter.go:330`), luôn đếm toàn bộ bảng Shadow từ `cursor = 0`.
     - Tại `transmuter.go:1363-1372`, ngay cả khi `isFullSync = false` (`run_type = ""`), câu lệnh `ON CONFLICT DO UPDATE` dùng điều kiện `WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(<master>._source_ts, 0) - 2000`. Với 4 triệu dòng cũ có `_source_ts` bằng nhau giữa Shadow và Master, biểu thức `>= _source_ts - 2000` luôn `TRUE`, khiến PostgreSQL thực hiện `DO UPDATE` ghi đè lại cả 4 triệu dòng cũ trước khi `INSERT` 100k dòng mới.
- **Bảng kết quả Decision Gate `DG3` (`06_laya_summary_dg3_20261019T102241.json` — Gate: `PASS`):**
  | Claim ID | Nội dung | R1 | R2 | R3 | Quyết định |
  |---|---|---|---|---|---|
  | `A-F1` | Sau khi đồng bộ với `len(onlySourceIDs) == 0` hoàn tất, hàm `Run` ghi đè `item.LastCursorJSON` thành `[]byte("{}")` (`transmuter.go:511-516`) | `CORRECT` (`0.3168`) | `CORRECT` (`0.7691`) | `YES` (`0.8341`) | `ACCEPT` |
  | `A-F2` | Khi `runType == "full"`, hàm `Run` bỏ qua checkpoint (`lastGpayID = 0`); khi `runType != "full"` và `len(chunkIDs) == 0`, hàm đọc `last_gpay_id` từ `state.LastCursorJSON` (`transmuter.go:325-344`) | `CORRECT` (`0.4576`) | `CORRECT` (`0.6703`) | `YES` (`0.8621`) | `ACCEPT` |
  | `A-F3` | Khi `len(onlySourceIDs) == 0`, hàm `Run` gọi `t.countShadowRows(ctx, masterRow, 0)` với cursor bằng `0` (`transmuter.go:205-211`) | `CORRECT` (`0.6602`) | `CORRECT` (`0.8673`) | `YES` (`0.9401`) | `ACCEPT` |
