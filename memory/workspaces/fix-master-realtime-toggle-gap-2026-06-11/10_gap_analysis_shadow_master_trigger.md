# 10_gap_analysis — Độ tin cậy luồng Shadow → Master (tổng quát)

**Ngày**: 2026-06-11 · **Agent**: Muscle (Claude-Opus-4.8) · **Trigger**: user "ko chỉ 1 vụ đâu, rất nhiều thứ ngắt luồng trigger shadow→master, audit lại, nghĩ solution tổng quan hơn". (Fix toggle off→on trước đó là point-fix quá hẹp.)

## Audit — cách transmute shadow→master được kích hoạt (4 nguồn, đọc code thật)
| Nguồn trigger | File | Subject | Loại |
|---|---|---|---|
| Realtime ingest (batch flush) | `batch_buffer.go:109,302` | `cdc.cmd.transmute-shadow` | **Core NATS** |
| Snapshot/sink path | `sinkworker.go:239,272` | `cdc.cmd.transmute-shadow` | **Core NATS** |
| Fan-out per master binding | `transmute_handler.go:129` | `cdc.cmd.transmute` | **Core NATS** |
| Scheduler (CHỈ mode='cron') | `transmute_scheduler.go:155` | `cdc.cmd.transmute` | **Core NATS** |
| Manual RunNow / heal-B | `transmute_schedule_handler.go`, `recon_heal_v4.go:161` | `cdc.cmd.transmute` | **Core NATS** |

**Subscribe**: `worker_server.go:446-447` dùng `natsClient.Conn.Subscribe(...)` = **Core NATS** (JetStream chỉ dùng cho Kafka consumer pool, KHÔNG cho transmute). ⇒ **at-most-once, không persist, không redeliver**.

## Root cause TỔNG QUÁT (1 câu)
Materialise shadow→master **chỉ** dựa trên **trigger event at-most-once (Core NATS), forward-only**, KHÔNG có **catch-up định kỳ đảm bảo** cho binding realtime (mode immediate/post_ingest — scheduler poll CHỈ mode='cron', hiện 0 row cron). Recon Segment B **phát hiện** drift mỗi chu kỳ NHƯNG **không auto-heal** (heal-B chỉ chạy khi operator bắn `cdc.cmd.recon-heal`). ⇒ BẤT KỲ trigger nào mất/bị skip → record kẹt ở shadow VĨNH VIỄN.

## "Rất nhiều thứ ngắt luồng" — liệt kê đầy đủ (cùng 1 gốc)
| # | Đường đứt | Vì sao record mất ở master |
|---|---|---|
| G1 | Realtime toggle off→on | Trigger tắt trong cửa sổ off; bật lại chỉ forward, không quét lại |
| G2 | Worker restart/down lúc publish | Core NATS rớt message → không redeliver |
| G3 | NATS slow-consumer / disconnect / overflow | Core NATS drop âm thầm |
| G4 | Snapshot/backfill lớn bị hiccup | Trigger qua Core NATS, mất giữa chừng (case `export_jobs_mt`: 168 snapshot 06-11 nhưng binding 11 master kẹt 163) |
| G5 | Master binding tạo/approve SAU khi shadow đã có data | Trigger forward-only, không backfill record cũ |
| G6 | Gate flap (master tạm not-approved/inactive lúc trigger) | Transmute skip, không retry |
| G7 | Schedule realtime (immediate/post_ingest) | Scheduler poll CHỈ mode='cron' ⇒ 0 safety-net định kỳ |
| G8 | Transmute fail lẻ (DLQ) | Không guaranteed retry → kẹt |

Mẫu số chung: **trigger-driven, at-most-once, thiếu reconciling safety-net**. (Đo thực tế hiện tại: `export_jobs_mt` shadow 170 / master 163 / diff 7 — recon đã DETECT nhưng không tự đóng.)

## GIẢI PHÁP TỔNG QUÁT (1 hướng — đóng tất cả G1–G8)
**Close-loop AUTO-HEAL trên recon Segment B định kỳ.**
Recon Segment B ĐÃ chạy định kỳ (runReconcileCycle ~mỗi 30') và ĐÃ tính `missing_from_master` cho mọi binding active+approved. Chỉ cần wire: **khi Segment B thấy shadow record thiếu/stale ở master → tự dispatch heal-B (re-transmute) ngay trong chu kỳ**, thay vì chỉ fire `ReconDrift` alert.

**Vì sao đây là "tổng quát"**: cơ chế đối soát **END-STATE** (shadow vs master) nên KHÔNG quan tâm trigger nào bị mất — đóng đồng thời G1–G8 trong ≤1 chu kỳ. Idempotent (OCC/LWW `_source_ts` đã có — transmuter.go:589-594). **Reuse 100%** Segment B (detect) + heal-B (re-transmute) đã tồn tại; chỉ thêm bước detect→heal + an toàn.

**An toàn bắt buộc (tránh heal-storm — lesson [2026-05-22] DLQ no-circuit-breaker)**:
- Throttle: mỗi binding chỉ auto-heal lại sau cooldown (vd ≥1 chu kỳ / N phút).
- Circuit-breaker: heal fail liên tiếp K lần cho 1 binding → ngừng auto, chỉ alert (tránh chạy điên khi lỗi deterministic).
- Bound per cycle: heal tối đa top-M binding drift mỗi vòng (tránh tải).
- Leader-gated (đã có ở CheckAllSegmentB).
- Chỉ heal hướng shadow→master (re-transmute missing); orphan-in-master xử lý riêng (đã có prune).

## Roadmap (phase)
- **P1 (lõi, đóng G1–G8)**: auto-heal close-loop trong runReconcileCycle: sau CheckAllSegmentB, với report `diff>0 & missing_count>0` → dispatch heal-B (throttle + circuit-breaker). ~1-2 file worker, không schema.
- **P2 (defense-in-depth realtime, giảm G2/G3/G4 tại nguồn)**: migrate transmute subjects Core NATS → **JetStream** (persist + redeliver) ⇒ realtime gần zero-loss; chu kỳ heal là lưới cuối.
- **P3 (đóng G5 tức thì)**: hook one-shot backfill khi tạo/approve master binding (và optional khi re-enable realtime — point-fix cũ trở thành tối ưu latency, không còn là "giải pháp chính").
- **P4 (tối ưu)**: watermark per-binding (`last_transmuted__gpay_id/_source_ts`) để sweep incremental thay vì full-scan.

## Trạng thái
- Đây là **design deliverable** — source CHƯA sửa. Chờ user chọn phase/approve mới thực thi (Rule 12).
- Point-fix toggle (09_tasks_solution.md cũ) → hạ xuống P3 optional, KHÔNG phải giải pháp chính.
