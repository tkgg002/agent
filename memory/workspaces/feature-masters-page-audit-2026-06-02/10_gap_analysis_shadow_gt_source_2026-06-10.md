# Audit: Shadow > Source (một số pipeline shadow nhiều record hơn source)

**Ngày**: 2026-06-10 · **Người**: Muscle/Claude-Opus-4.8 · **Loại**: data-integrity CDC audit (source→shadow).

## 1. Bằng chứng (PROVEN — query thật)
Source Mongo `centralized-export-service.export-jobs` = **133** records (lưu ý registry ghi `centrallized` 2-L là typo display; db thật 1-L).

| shadow table | total | distinct _source_id | dup | _deleted | so với source 133 |
|---|---|---|---|---|---|
| shadow_aaaa_*.export_jobs (63) | 465 | 465 | 0 | 0 | +332 |
| shadow_aaaaa.export_jobs_2/4/5 | 465 | 465 | 0 | 0 | +332 |
| shadow_aaaaa.export_jobs (66, inactive) | 463 | 463 | 0 | 0 | +330 |
| shadow_dev000.export_jobs (82) | 170 | 170 | 0 | 0 | +37 |
| shadow_dev000.export_jobs_test (90) | 169 | 169 | 0 | 0 | +36 |
| shadow_aaaaa.export_jobs_3 (72, inactive) | 0 | 0 | 0 | 0 | — |

- **KHÔNG trùng** (distinct == total) → không phải lỗi nhân bản/re-snapshot.
- **`_deleted = 0` toàn bộ** → KHÔNG có soft-delete nào được áp.
- Lấy 170 `_source_id` của dev000.export_jobs → **0/170 còn tồn tại trong source** (170 ghost). Mongo `_id` là ObjectId, sanity query OK → 0-match là thật.
- shadow vẫn sync gần đây (max `_synced_at` = 2026-06-10 03:57) → pipeline LIVE, không phải treo.
- **Không binding nào có `explode_path`** → KHÔNG phải flatten 1→N by-design. Shadow>source ở đây là BUG.
- wallet-service (wallets/events/wallet-capsets): **không tồn tại trong Mongo** (db `wallet-service` không có) → các shadow đó orphan hoàn toàn (source=0 < shadow>0) — case khác (source bị xoá/đổi tên).

## 2. Root cause
1. **Code CÓ xử lý delete per-doc**: `event_handler.go:175` `op=="d" → handleDelete` → tombstone soft-delete `ON CONFLICT (_source_id) ... SET _deleted=TRUE`. transmute (`fetchShadowBatch`) đọc + propagate `_deleted` sang master. ⇒ delete per-doc nếu có SẼ chạy đúng.
2. **Nhưng `_deleted=0`** ⇒ source removals KHÔNG đến dưới dạng per-doc Debezium delete event. Nguyên nhân: source bị **drop/replace (re-seed cả collection)** — môi trường dev/smoke-test ghi đè `export-jobs` liên tục. **MongoDB-Debezium KHÔNG emit per-document delete khi drop collection** ⇒ shadow giữ lại doc cũ (ghost), chỉ cộng thêm doc mới (insert) ⇒ shadow tích luỹ vượt source.
3. **Không có cơ chế PRUNE orphan**: snapshot là insert-only; recon Segment A (source→shadow) chỉ **heal MISSING** (source có/shadow thiếu → insert), KHÔNG prune **ORPHAN** (shadow có/source không = ghost). recon Segment B (shadow↔master) có `diffIDTs` tính cả orphan nhưng A thì không ⇒ ghost không bao giờ bị dọn ⇒ shadow lớn dần vô hạn.

## 3. Giải pháp
### A. Dọn ngay (reset shadow bị ghost) — tác vụ vận hành, có duyệt
- TRUNCATE + re-snapshot shadow table bị ghost (đưa về khớp source), HOẶC one-time soft-delete ghost: `UPDATE <shadow> SET _deleted=TRUE WHERE _source_id NOT IN (<source ids hiện tại>)` rồi để transmute lan delete sang master.

### B. Fix gốc — thêm ORPHAN-PRUNE vào recon Segment A (đề xuất chính)
- Mở rộng recon source→shadow: liệt kê `_source_id` ở shadow KHÔNG còn trong source (dùng debezium-signal/list ids hoặc count+sample) → **soft-delete** (`_deleted=TRUE`) (KHÔNG hard-delete để giữ audit + để transmute propagate). Tái dùng khung `diffIDTs` (đã có cho Segment B).
- Đặt lịch prune định kỳ (recon-heal mở rộng) ⇒ shadow ≤ source, tự hồi phục sau mỗi đợt re-seed.

### C. Snapshot-with-prune (xử lý drop/re-seed triệt để)
- Khi re-snapshot: đánh dấu `_deleted=TRUE` cho shadow row KHÔNG có trong snapshot mới (snapshot = source of truth tại thời điểm đó). Giải quyết case Debezium không emit delete khi drop.

### D. Vận hành
- KHÔNG drop/recreate collection source đang CDC; dùng per-doc delete (handleDelete đã chạy đúng → soft-delete tự áp).
- wallet-service: source object đăng ký nhưng Mongo db không tồn tại → dọn/re-point binding orphan (deactivate hoặc trỏ lại db đúng).

### E. Đảm bảo read-side (đã OK)
- transmute đọc `_deleted` + propagate; master/query phải lọc `WHERE NOT _deleted` để ghost (sau khi soft-delete) không lọt master.

## 4. Verify đã làm
- Query count source (Mongo) vs 8 shadow export_jobs; distinct vs total; _deleted; 170 _source_id vs Mongo (0 match); max _synced_at; explode_path; event_handler delete path; transmute _deleted propagate; recon thiếu prune A.
