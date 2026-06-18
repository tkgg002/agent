# 10_gap_analysis — Pipeline `export_jobs` lệch: ingest +2 / transmute −7

**Ngày**: 2026-06-11 14:2x ICT · **Agent**: Muscle (Claude-Opus-4.8) · **Trigger**: user gửi dashboard recon row + "Lệch phân tích nguyên nhân".

## Đối tượng
- Source (Mongo REMOTE, conn `dev000`): `centrallized-export-service.export_jobs` @ `mongodb://root:***@10.200.187.11/12/13:27017/?replicaSet=goopay` (mask — Rule 19). **Count=168**.
- Shadow: `shadow_dev000.export_jobs` (binding 82). **Count=170** (170 distinct _source_id, 0 dup, 0 _deleted).
- Master: `dw_centrallized_export_service.export_jobs_mt` (master_binding **11**, active/approved). **Count=163**. PK `_gpay_id`; upsert key UNIQUE(`_id`).
- Dashboard: 168 | 170 | 163 → ingest **+2** (shadow>source), transmute **−7** (master<shadow).

## Bằng chứng thu thập
1. Diff shadow↔master: master ⊂ shadow, thiếu **đúng 7** `_id`: `69523aa4d84ba5fa04cf6090, 69523aa6d94ba5fa04cf6090, 69533aa4d94ba5fa04cf6091, 69534aa4d94ba5fa04cf6090, 69633aa4d94ba5fa04cf6090` (5×, `_synced_at`=06-11 02:54) + `6a26936451c80c9c3855639a` (06-08), `6a28cbecec5b9378333d594a` (06-10).
2. **0 failed_sync_logs** cho 7 record này / target export_jobs_mt (chỉ export_jobs_2 có 906 fail — binding khác). ⇒ KHÔNG phải lỗi transform; 7 jobId cũng KHÔNG có trong master ⇒ không phải dedup-merge.
3. `transmute_schedule` binding 11: 2 row mode `immediate`(id5)/`post_ingest`(id3), **last_run=2026-06-05**, `next_run_at=NULL`, is_enabled=t. Master `_updated_at` max = **06-05 09:06**, max `_source_ts`=06-05 ⇒ **master đóng băng từ 06-05**.
4. Sibling master_binding **12** (`master_dev000.export_jobs_mt_02`, cùng shadow 82): last_run **06-11 02:33**, inserted=170, count=170, _updated_at 06-11 02:54 ⇒ **đã catch-up**. ⇒ data shadow ĐỦ & transmit được; lỗi riêng binding 11.
5. Scheduler poll (`transmute_scheduler.go:113-117`) **chỉ chạy `mode='cron'`** + `next_run_at<=NOW()`. Binding 11 mode immediate/post_ingest, next_run_at NULL ⇒ **scheduler không bao giờ quét nó**; chỉ chạy khi ingest bắn `publishTransmuteTrigger`→`cdc.cmd.transmute-shadow` (batch_buffer.go:91, fan-out mọi master is_active+approved).
6. `_synced_at` shadow: **168 record @ 06-11 02:54** (1 đợt snapshot re-sync) + 1 @ 06-10 + 1 @ 06-08. ⇒ 168 snapshot ≈ source 168; **2 record `6a2` (06-08/06-10) nằm NGOÀI snapshot**.
7. Worker recon log tự phát hiện: `segment B shadow↔master ... export_jobs_mt shadow_rows=170 master_rows=163` + `ReconDrift` alert (binding 12 = 170/170 ok).

## Root Cause
### Transmute −7 (master 163 < shadow 170) — CHẮC CHẮN
master_binding 11 transmute **không chạy từ 2026-06-05**. Mode event-driven (immediate/post_ingest) KHÔNG có cron fallback → scheduler (chỉ quét mode=cron) bỏ qua; chỉ phụ thuộc trigger lúc ingest. 7 `_id` MỚI xuất hiện sau 06-05 (đợt snapshot 06-11 + 2 record 06-08/10) không được materialize → master kẹt 163. Không phải lỗi transform (0 failed_logs). Sibling binding 12 sync đủ 170 vì được **re-provision/backfill 06-11**; binding 11 KHÔNG được re-trigger.
- "Vì sao chỉ binding 11": đợt refresh 06-11 đi qua **snapshot path** (168 record cùng _synced_at) — path này không auto-trigger transmute catch-up cho mọi master; master chỉ cập nhật khi có (re)provision/backfill thủ công (12 có, 11 không). Thiếu cron fallback ⇒ binding 11 stale âm thầm.

### Ingest +2 (shadow 170 > source 168) — CHẮC CHẮN ở mức suy luận (remote source không đọc được từ sandbox)
2 record dư = `6a26936451c80c9c3855639a` (06-08) + `6a28cbecec5b9378333d594a` (06-10) — là 2 record DUY NHẤT nằm ngoài đợt snapshot 168 của 06-11. Snapshot 06-11 kéo 168 từ source (≈ source hiện tại); 2 record cũ này tồn tại trong shadow nhưng KHÔNG còn trong source (đã bị xoá ở source sau 06-08/10; snapshot path chỉ ADD/UPDATE, **không prune** record source đã xoá) ⇒ **orphan/ghost** (lớp shadow>source đã biết, đã dựng orphan-prune phiên trước).

## Khuyến nghị xử lý (chưa thực thi — chờ duyệt vì ghi master/shadow)
- **Transmute −7**: re-trigger transmute binding 11 → publish `cdc.cmd.transmute-shadow {shadow_schema:shadow_dev000, shadow_table:export_jobs, source_ids:[7 ids hoặc all]}` HOẶC re-provision master_binding 11 (full backfill). → master 163→170. **Root fix**: thêm cron fallback cho transmute_schedule mode immediate/post_ingest, hoặc snapshot path bắn transmute-trigger cho mọi master binding.
- **Ingest +2**: orphan-prune `shadow_dev000.export_jobs` (POST /api/reconciliation/prune/... đã build) → soft-delete 2 ghost. **Root fix**: snapshot-with-prune (snapshot Δ phải soft-delete record source-đã-xoá). LƯU Ý: prune có safety source=0 → cần source remote đọc được khi chạy.

## Ràng buộc tuân thủ
- Chỉ ĐỌC (read-only) trong phân tích này; KHÔNG sửa data/config; KHÔNG commit/push. Credential remote đã mask. Trả lời tiếng Việt.
