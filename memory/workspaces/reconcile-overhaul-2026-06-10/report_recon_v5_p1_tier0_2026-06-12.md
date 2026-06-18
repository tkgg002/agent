# report_recon_v5_p1_tier0_2026-06-12.md — V5-P1: Tier-0 count-only + bucket-on-drift

> Muscle:Claude-Opus-4.8 | 2026-06-12 | Verb: "làm đi" (approve V5 bản đơn giản hoá theo Boss)

## 1. Đã làm gì (đúng thiết kế đã duyệt)
| Đòn bẩy | Thực thi |
|---|---|
| **Tier-0 = count 3 trạm đúng nghĩa** | A: `EstimatedCount` (Mongo O(1) metadata) + `CountRows` PG; khớp (tolerance max(1, 0.1%)) → report `ok` `count_total`, **DỪNG — 0 window query**. B: totals + transmute-lag khớp → early-exit tương tự |
| **Bucket-aggregate chỉ-khi-lệch** | A: 1 aggregate Mongo (`$group` bucket-giờ, `$toLong($toDate)` chịu cả Date lẫn epoch-number) + 1 query PG — thay **1.344 round-trips**; B: 1 query/phía `count + bit_xor(hashtextextended)`; drill-down ID **chỉ bucket lệch** |
| **Phân xử estimate-noise** | bucket khớp + totals lệch → 1 lần `CountDocuments` exact (nhánh hiếm): exact khớp = noise → ok; lệch = drift NGOÀI lookback (việc Tier-3) |
| **Bỏ stagger-sleep** | CheckAll → pool goroutine: global-sem **8** + per-connection-sem **2** (thứ tự acquire cố định chống deadlock); giữ RunID-085 + per-table timeout 45s của bên kia |
| **Fail-fast mạng remote** | client recon: `serverSelectionTimeoutMS/connectTimeoutMS=5000` (thay default 30s — nguồn của run 2.4h); không đụng pkgs/mongodb chung |

## 2. Files đã sửa (git)
| File | Đổi |
|---|---|
| `recon_source_agent.go` | +EstimatedCount +BucketCounts (+~75) + getClient timeout params (+12) |
| `recon_dest_agent.go` | +BucketStat +BucketCounts (+~55) |
| `recon_core.go` | pickScanRangeWithLag (+wrapper); RunTier1 restructure (Tier-0→Tier-1); CheckAll pool (bỏ stagger); RunSegmentB early-exit + bucket; totals tái dùng; −rand +sync |

## 3. Benchmark (số thật — điều kiện: Mongo remote ĐANG DOWN toàn phần)
| Chỉ số | Trước (24h baseline) | Sau V5-P1 |
|---|---|---|
| Segment B per-bảng | avg 11.9s / max 1.078s | **0.0-0.1s** (7/7 success, Tier-0 early-exit) ✅ |
| Segment A per-bảng khi nguồn CHẾT | treo 30-45s+/bảng TUẦN TỰ (max 2.4h) | **fail-fast 10s, SONG SONG** — cycle toàn-fail ~30s, status `failed` trung thực |
| Segment A happy-path (<1s DoD) | 328.7s avg | ✅ **ĐẠT — mạng hồi, vòng schedule TỰ ĐỘNG: 9/9 success, avg 0.30s / max 0.98s** (events 188K rows = 0.98s; wallet_capsets 11K = 0.10s) — **~1.100× nhanh hơn** |
| Build/vet/test | | PASS (94+ tests) |

## 4. Trung thực — giới hạn & việc còn
- A happy-path benchmark chờ mạng remote (không chế số). Worker v5p1 PID 79453 đang chạy — vòng schedule kế khi mạng hồi sẽ tự cho số.
- Request-reply CheckAll qua NATS có thể timeout phía client khi cycle chen với schedule (reply vẫn về sau) — không ảnh hưởng kết quả ghi DB.
- V5-P2 (job-queue + backoff + multi-worker) + P3 (incremental watermark + bench 3K synthetic) theo roadmap — chưa làm.
- Breaker "skip cả nhóm connection khi down" chưa làm riêng (per-URL breaker sẵn có + fail-fast 5s đã giảm 6×; đưa vào P2 cùng job-queue).

## 5. Services
Worker `/tmp/cdc-worker-recon-v5p1` (PID 79453) RUNNING 8082; cms + FE không đổi trong phase này.
