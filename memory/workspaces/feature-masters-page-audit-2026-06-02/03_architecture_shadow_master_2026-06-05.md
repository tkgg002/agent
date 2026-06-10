# 03_architecture_shadow_master_2026-06-05.md — Audit chặng 2 (Shadow→Master) theo Safety/Completeness/Performance + bổ sung giải pháp

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-05 | Đối chiếu phân tích kiến trúc của anh với SOURCE THẬT (file:line).
> Chặng 1 (Source→Shadow = Debezium CDC) KHÔNG đụng tới. Chỉ audit chặng 2 (Shadow→Master = CDC Worker WK-E1..E5).

## Map engine → code thật
| Engine | Code |
|--------|------|
| WK-E1 Transmute | `transmuter.go` Run/loadRules/processBatch/upsertMaster (gjson, transform_fn, gate, OCC upsert) |
| WK-E2 3-Way Trigger | run-now (`transmute_schedule_handler.RunNow`), cron (`transmute_scheduler.go` FOR UPDATE SKIP LOCKED + fencing), post_ingest (`sinkworker.publishTransmuteTrigger` + gate `hasPostIngestSchedule`) |
| WK-E3 Shadow Introspection | `command_handler.go HandleScanArrayFields` + introspection (shadow-columns) |
| WK-E4 Master DDL Apply | `master_ddl_generator.go Generate/Apply` (CREATE/ALTER + RLS) |
| WK-E5 Close-loop | `transmute_handler.publishCompleted` → `cdc.evt.transmute.completed` → `JobMonitor.HandleCompleted` |

---

## 1. PERFORMANCE 🟢 ĐẠT — 1 tối ưu còn lại
- ✅ Cache rule/mask O(1) (Registry in-memory) — đúng như phân tích, transmute không query DB per-record.
- ✅ Chunk fetch keyset (batchSize=500, ORDER BY _gpay_id, cursor `> ?`).
- ✅ Hash short-circuit (`_hash IS DISTINCT` → không ghi khi không đổi).
- 🟡 **GAP-PERF-1 (Mini-batching WRITE)**: `processBatch` gọi `upsertMaster` **per-row** (`transmuter.go:413`) → high-throughput sẽ nghẽn I/O master. **Giải pháp (đúng đề xuất anh)**: gom 100-500 record → 1 câu `INSERT ... ON CONFLICT` multi-row (VALUES nhiều dòng) trong 1 round-trip. Cần giữ guard OCC `_source_ts` per-row (dùng `DISTINCT ON (_source_id) ... ORDER BY _source_ts DESC` để chọn bản mới nhất trong batch trước khi upsert).

## 2. COMPLETENESS 🟡 CÓ ĐIỀU KIỆN — thiếu watermark
- ✅ 3-Way Trigger bao phủ realtime/cron-bù/run-now. Cron là lưới an toàn nếu realtime lỗi.
- 🟡 **GAP-COMP-1 (Watermark/Offset)** — đúng phân tích anh: `Run` khởi tạo `lastGpayID=0` MỖI lần (`transmuter.go:173`) → **full re-scan** toàn shadow mỗi cron/run-now (hash-dedup nên KHÔNG sai data, nhưng tốn I/O + không phải incremental thật). Run-now/cron không lưu high-water-mark.
  - **Giải pháp**: lưu `last_gpay_id` vào `sync_runtime_state` per master; `Run` đọc cursor khởi đầu từ đó (full lần đầu, incremental sau). Hoặc (mạnh hơn) **WK-E2 post_ingest nghe trực tiếp Kafka topic Debezium của bảng shadow** thay vì quét DB → thừa hưởng at-least-once của Debezium. (post_ingest hiện đã incremental theo `_source_ids` — đã tốt; cron/run-now mới cần HWM.)

## 3. SAFETY 🟡 — đã siết phần lớn phiên này
- ✅ OCC upsert + **GAP-02 guard `_source_ts`** (vừa thêm `transmuter.go:546`) → event cũ KHÔNG đè bản mới (test-first SQL verified).
- ✅ **GAP-01 RLS phase-1 (A)** (vừa thêm `master_ddl_generator.go:220`) → ENABLE RLS mọi master schema (no FORCE → worker owner bypass, non-owner chặn). Verified live b2 `relrowsecurity=t`.
- 🟡 **GAP-SAFE-1 (DDL ALTER block WK-E1)** — đúng phân tích anh: ALTER lấy AccessExclusiveLock, block transmute upsert. **Đã giảm thiểu phiên này**: thêm `SET LOCAL lock_timeout='5s'` + `statement_timeout='30s'` vào tx Apply (`master_ddl_generator.go:199`) → DDL fail-fast + retry thay vì block vô hạn/crash WK-E1. **Hoàn thiện (future)**: pause WK-E1 trước DDL → ALTER → resume (orchestration qua NATS pause/resume signal).
- 🟡 **GAP-SAFE-2 (Cache lag DDL↔rule)** — đúng phân tích anh: sau DDL, nếu rule cache (loadRules `cacheTTL`, `transmuter.go:281`) chưa reload → WK-E1 dùng rule cũ trên schema mới. **Giải pháp**: sau `Apply` thành công → invalidate cache transmuter cho master đó (xoá entry `cacheKey`), HOẶC bus event `cdc.evt.master.ddl-applied` → worker reload. Quy trình bắt buộc: **Approve → ALTER → Reload cache OK → mới cho WK-E1 chạy rule mới**.

---

## Tổng hợp gap kiến trúc mới (bổ sung) + trạng thái
| Mã | Hạng mục | Mức | Trạng thái |
|----|----------|-----|-----------|
| GAP-PERF-1 | Mini-batch write upsert | MED | 🟡 chưa (per-row) — giải pháp sẵn |
| GAP-COMP-1 | Watermark/HWM cron+run-now (hoặc Kafka shadow topic) | MED | 🟡 chưa (full re-scan, dedup an toàn) |
| GAP-SAFE-1 | DDL lock block WK-E1 | HIGH | 🟢 giảm thiểu (lock_timeout) — pause/resume = future |
| GAP-SAFE-2 | Cache lag sau DDL | MED | 🟡 chưa — giải pháp: invalidate cache sau Apply |

## Đã EXECUTE phiên này (cho doc audit)
- GAP-01 RLS phase-1 (A): RLS mọi schema (verified live). 
- GAP-SAFE-1: lock_timeout/statement_timeout cho DDL Apply.
- (Trước đó cùng mạch: GAP-02 OCC `_source_ts` guard, I2/I6/GAP-03/04/05.)
- worker go build=0, test PASS.

## Khuyến nghị thứ tự đóng nốt (gap kiến trúc)
1. GAP-SAFE-2 (cache invalidate sau DDL) — bug đúng/an toàn data, fix nhỏ (xoá cacheKey sau Apply).
2. GAP-PERF-1 (mini-batch) — khi throughput tăng; DISTINCT ON giữ OCC.
3. GAP-COMP-1 (HWM/Kafka) — incremental thật cho cron; lớn hơn, lên ADR.
4. GAP-SAFE-1 pause/resume — sau khi có orchestration signal.
