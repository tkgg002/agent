# 09_tasks_solution_recon_v5_scale.md — Tính toán lại Recon cho 200-300 DBs / 10-50M rows

> Muscle:Claude-Opus-4.8 | 2026-06-12 | Verb Boss: "recon đang quá lâu… 200-300 source DB, bảng 10-50tr… tính toán lại"
> **Trạng thái: DESIGN + CAPACITY MATH — chờ Boss approve. CHƯA code.**

---

## 1. Đo hiện trạng (số THẬT 24h qua, recon_runs)

| Segment | avg/run | max/run | Vì sao |
|---|---|---|---|
| A (source↔shadow, Tier1) | **328.7s** | **8.592s (2.4h)** | **672 windows × 2 count = 1.344 round-trips/bảng** tới Mongo REMOTE (~200ms RTT) ≈ 270s + CountDocuments FULL (collscan) + stagger/jitter |
| B (shadow↔master) | 11.9s | 1.078s | 672×2 query PG local (nhanh hơn) + 2 COUNT(*) full |
| Cycle hiện tại | **~13' cho 19 bảng** | | đúng quan sát của Boss |

**Điểm chết kiến trúc hiện tại:** chi phí tỷ lệ theo **SỐ ROUND-TRIP** (672 windows × 2 phía × N bảng, tuần tự + sleep cố ý), không phải data size. Bảng 600 rows cũng mất 5'+ vì RTT.

## 2. Quy mô mục tiêu & con số pipeline

200-300 source DBs × (ước 5-20 bảng CDC/DB) → **1.500-6.000 pipelines** (mỗi pipeline = 1 dòng A + có thể 1+ dòng B). Tính 3 kịch bản: **1K / 3K / 5K**.

### Nếu GIỮ kiến trúc hiện tại
| Kịch bản | Tuần tự 1 worker (328s/bảng) | Song song 16 (bỏ sleep) |
|---|---|---|
| 1.000 | 3.8 ngày/cycle | ~5.7h |
| 3.000 | **11.4 ngày/cycle** | ~17h |
| 5.000 | 19 ngày/cycle | ~28h |
+ bảng 50M: CountDocuments full = collscan 30-120s/bảng/cycle; HashWindow B stream toàn bộ window. → **VỠ HOÀN TOÀN, không cứu bằng tăng máy.**

## 3. Thiết kế V5 — 5 đòn bẩy (giải pháp duy nhất)

### Đòn 1 — Diệt round-trip: 672×2 queries → **2 queries/bảng** (aggregate server-side)
Tier1-A thay vòng lặp window bằng **1 aggregate mỗi phía**, so 168 bucket/giờ trong memory:
```js
// Mongo (1 round-trip, dùng index {tsField:1}, chỉ rows trong lookback):
db.coll.aggregate([
 {$match:{updated_at:{$gte:lo,$lt:hi}}},
 {$group:{_id:{$dateTrunc:{date:"$updated_at",unit:"hour"}}, c:{$sum:1}}}])
```
```sql
-- PG shadow (1 round-trip):
SELECT date_trunc('hour', to_timestamp(_source_ts/1000)) AS b, count(*)
FROM shadow_x.t WHERE _source_ts >= $lo AND _source_ts < $hi GROUP BY 1;
```
Bucket lệch → drill-down CHỈ bucket đó (như Tier2 hiện tại). Segment B tương tự — PG15 có `bit_xor` aggregate:
```sql
SELECT date_trunc('hour', to_timestamp(_source_ts/1000)) AS b, count(*),
       bit_xor(hashtextextended(_gpay_id::text||'|'||_source_ts::text,0))
FROM rel WHERE _source_ts >= $lo AND _source_ts < $hi AND NOT COALESCE(_deleted,false) GROUP BY 1;
```
→ **A: ~270s round-trip → 0.3-1s. B: ~10s → 0.3-1s.** Độ nhạy GIỮ NGUYÊN (count per bucket + xor per bucket ⊇ thông tin 672-window count).

### Đòn 2 — Totals O(1): `estimatedDocumentCount()` (Mongo metadata) + `pg_class.reltuples` (PG estimate); exact COUNT chỉ khi bảng < 1M hoặc 1 lần/6h (cache). Hết collscan 50M mỗi vòng.

### Đòn 3 — Bỏ stagger-sleep → **worker pool + semaphore per-connection**
Concurrency 12 bảng song song; semaphore ≤3 bảng đồng thời/1 source connection (không đập 1 cluster Mongo bởi 50 bảng cùng lúc — thay thế đúng vai trò của stagger nhưng không đốt thời gian chờ).

### Đòn 4 — Job-queue thay CheckAll-quét-hết: **`next_check_at` + adaptive backoff**
- Registry per-bảng: `next_check_at`, `consec_ok`. Scheduler kéo `WHERE next_check_at <= now ORDER BY next_check_at FOR UPDATE SKIP LOCKED LIMIT K` (pattern đã có ở transmute scheduler).
- ok liên tiếp → backoff: 15' → 1h → 6h → 24h (cap); có drift/event mới (ingest lag>0) → reset 10-15'.
- **Multi-worker scale ngang tự nhiên** (SKIP LOCKED chia job; bỏ leader-election-per-cycle → hết nghẽn 1 instance + hết trap leader-TTL).

### Đòn 5 — Incremental watermark: lưu `last_verified_ts` per bảng — vòng sau chỉ scan từ đó (thay lookback 7d cố định). Bảng 50M nhưng delta vài giờ → bucket aggregate chạy trên delta. Tier3/row-diff full giữ off-peak đêm.

## 4. Tính lại sau V5 (throughput math)

Per-bảng: A = 2 aggregate (0.3-1s remote) + O(1) totals; B = 2 query (~0.3s). Pool 12, semaphore 3/connection, 200-300 connections → đủ độ rộng song song; throughput thận trọng **4-6 bảng/s/worker**.

| Kịch bản | 1 worker | 3 workers | Hot-set hiệu dụng (backoff) |
|---|---|---|---|
| 1.000 | **3-4'** | ~1.5' | <1' (chỉ ~10-20% hot mỗi vòng) |
| 3.000 | **9-13'** | 3-5' | ~2-3' |
| 5.000 | 15-21' | **5-7'** | ~3-5' |

→ Mục tiêu vận hành: **hot tables được đối soát mỗi 10-15'; full sweep mỗi giờ; bảng ổn lâu năm tự giãn 6-24h** — đạt với 1-3 worker, KHÔNG phụ thuộc size bảng (50M hay 500M chỉ ảnh hưởng aggregate-on-delta + index ts).

## 5. Roadmap thực thi (sau approve)

| Phase | Nội dung | Ước lượng | DoD đo được |
|---|---|---|---|
| **V5-P1** | Bucket-aggregate 2-query (A & B) + estimate totals + bỏ stagger→pool/semaphore | 2.5d | avg run/bảng < 2s (từ 328s); cycle 19 bảng < 1' (từ 13') |
| **V5-P2** | Job-queue next_check_at + SKIP LOCKED multi-worker + adaptive backoff (bỏ leader-per-cycle) | 2d | 2 worker chia job không trùng; bảng ok 5 lần liên tiếp giãn lịch |
| **V5-P3** | Incremental watermark + totals cache 6h + load-test giả lập 3K pipelines (synthetic registry) | 1.5d | cycle 3K < 15'/1 worker trên bench |

Tổng ~6 ngày. Mỗi phase build+test+benchmark trước/sau bằng chính `recon_runs.duration`.

## 6. Những gì KHÔNG làm
- Không thêm máy để cứu kiến trúc round-trip (vô ích — RTT × 1.344 không giảm theo CPU).
- Không hạ độ nhạy (bucket/giờ giữ độ phát hiện tương đương 15'-window cho mục đích drift-detect; drill-down giữ nguyên độ chính xác ID).
- Không đụng pipeline ingest source→shadow.

## Verb chờ Boss
`approve recon v5` → thực thi V5-P1 | `revise <điểm>`.
