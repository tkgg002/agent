# report_recon_actions_inventory_2026-06-12.md — Thống kê TẤT CẢ loại recon + 328s đi đâu

> Muscle:Claude-Opus-4.8 | 2026-06-12 | Boss: "thống kê các loại recon; action gì mà 1 table 300s; chỉ count 3 bảng thôi mà"

## 1. Inventory — mọi loại recon đang tồn tại trong hệ

| # | Loại | Trigger | Chạy gì (số round-trip) | Chi phí đo được |
|---|------|---------|-------------------------|------------------|
| 1 | **Tier 1 — A count_windowed** | scheduler mỗi cycle + NATS | per bảng: MaxTs Mongo (1) + MaxTs PG (1) + **672 window × 2 COUNT = 1.344 queries** + CountDocuments FULL Mongo (1, collscan) + COUNT(*) PG (1) + upsert lag/report (2) | **avg 328.7s, max 2.4h** |
| 2 | Tier 2 — A hash_window | NATS/heal-A khi cần ID | per window lệch: HashWindow stream 2 phía + ListIDs 2 phía | avg 2.3s (ít window lệch) |
| 3 | Tier 3 — A bucket_hash | off-peak 02-05h | stream TOÀN BỘ bảng 2 phía, 256 bucket XOR | chưa chạy ở prod-scale |
| 4 | **Segment B — shadow↔master** | scheduler (mới thêm) + NATS | MaxTs ×2 + **672 window × 2 HashWindow (stream)** + COUNT(*) ×2 + ListIDTs khi lệch | avg 11.9s (PG local), max 1078s |
| 5 | Orphan-prune A (bên khác đang dev) | NATS tier=prune | đọc id source vs shadow → soft-delete ghost | mới |
| 6 | Heal-A (re-trigger) | NATS recon-heal | đọc report + publish dbz-signal chunks | <1s dispatch |
| 7 | Heal-B (re-trigger) | NATS recon-heal | map gpay→source_id (PG) + publish transmute chunks | <1s dispatch |
| 8 | Row-diff L3-B (deep) | NATS deep:true | fetch raw 2 phía theo ID (cap 200) + re-derive | ~1-3s |
| 9 | Backfill _source_ts | API | batch UPDATE từ Mongo | theo batch |
| 10 | TimestampDetector / FullCountAgg | phụ trợ | sample/đếm | disabled phần lớn |

## 2. "Chỉ count 3 bảng thôi mà" — 328s đi đâu? (phân rã Tier 1 per bảng)

| Thành phần | Round-trips | Thời gian (Mongo REMOTE ~200-250ms RTT, đo gián tiếp từ avg) |
|---|---|---|
| 672 window-COUNT Mongo | 672 | **~135-170s** |
| 672 window-COUNT PG | 672 | ~2-5s (local) |
| CountDocuments FULL Mongo (totals — collscan) | 1 | 2-10s (bảng nhỏ) → **30-120s @ 10-50M** |
| MaxTs ×2 + COUNT PG + report/lag | ~5 | ~1-2s |
| **Khi mạng remote chập chờn** (đang xảy ra: connect timeout 30s) | mỗi lần đứt | +30s/lần, breaker mở → max **2.4h** |

→ Boss đúng 100%: **mục tiêu chỉ cần 3 con số COUNT, nhưng Tier 1 hiện trả lời câu hỏi đắt hơn nhiều: "lệch Ở KHOẢNG THỜI GIAN NÀO"** bằng cách bắn 1.344 phát COUNT-theo-window NGAY CẢ KHI KHÔNG LỆCH. Đó là thiết kế ngược: trả tiền drill-down trước khi biết có drift.

## 3. Cấu trúc lại theo đúng quan điểm Boss (gộp vào V5 — còn đơn giản hơn bản trước)

**TIER-0 (mặc định, mọi cycle) = ĐÚNG NGHĨA "count 3 bảng":**
```
source_total  = Mongo estimatedDocumentCount()        — 1 query, O(1) metadata
shadow_total  = PG count(*) (hoặc reltuples nếu >10M) — 1 query
master_total  = PG count(*) (hoặc reltuples)          — 1 query
(+ MaxTs ×3 cho lag — 3 query O(1) theo index)
```
→ **~6 query O(1)/pipeline ≈ 0.3-0.8s** (kể cả remote RTT). 3.000 pipelines × 0.5s / pool 12 ≈ **2-3 phút/cycle**. Khớp = xong, KHÔNG chạy gì thêm.

**TIER-1 (chỉ khi Tier-0 lệch hoặc lag vượt ngưỡng):** bucket-aggregate 2 query/phía (thiết kế V5 đòn 1) → định vị khoảng giờ lệch.
**TIER-2 (chỉ bucket lệch):** drill-down ID. **TIER-3/row-diff:** off-peak/theo yêu cầu — giữ nguyên.

Kèm 2 lá chắn cho mạng remote chập chờn (nguồn run 2.4h): connect-timeout hạ 30s→5s + breaker per-connection đánh dấu connection DOWN → skip cả nhóm bảng của connection đó trong cycle (1 lần timeout thay vì N bảng × timeout), alert `ReconSourceDown`.

## 4. Số sau cấu trúc lại (cập nhật bảng V5)
| Pipelines | Cycle Tier-0 (1 worker, pool 12) | Khi có drift (chạy thêm Tier-1 cho bảng lệch) |
|---|---|---|
| 1.000 | ~1' | +0.5-1s/bảng lệch |
| 3.000 | **2-3'** | nt |
| 5.000 | 4-5' | nt |

→ Roadmap V5-P1 đổi nội dung thành: **Tier-0 count-only + demote window-count thành Tier-1-on-drift + timeout/breaker per-connection** (vẫn 2.5d). P2/P3 giữ nguyên (job-queue/backoff/watermark).

## 5. Verb chờ Boss
`approve recon v5` (bản đã đơn giản hoá theo ý anh) | `revise`.
