# Phân Tích Sâu Nguyên Nhân SLOW SQL Recon Read Repo & Giải Pháp Tối Ưu

## I. Phân Tích Chi Tiết 2 Câu Query Chậm

### 1. Query `GetBackfillStatus` (690.289ms)
```sql
SELECT * FROM "recon_runs" WHERE tier = 4 AND instance_id LIKE 'backfill:%' ORDER BY started_at DESC LIMIT 30
```
- **Vấn đề:** 
  1. API `/api/recon/backfill-source-ts/status` được giao diện Frontend gọi (polling) định kỳ 5 giây/lần.
  2. Bảng `recon_runs` chứa toàn bộ lịch sử các lượt chạy recon của hệ thống.
  3. Mệnh đề `WHERE tier = 4 AND instance_id LIKE 'backfill:%' ORDER BY started_at DESC` không có composite index `(tier, started_at DESC)`.
  4. Mặc dù có `instance_id LIKE 'backfill:%'`, việc tìm kiếm chuỗi tiền tố không dùng index thích hợp nếu không có index composite, dẫn đến việc PostgreSQL phải quét toàn bộ các bản ghi trong `recon_runs` rồi sort `started_at DESC`.
- **Giải pháp:**
  1. Thêm migration index: `CREATE INDEX IF NOT EXISTS idx_recon_runs_tier_started ON cdc_system.recon_runs (tier, started_at DESC);`
  2. Thêm cờ giới hạn cửa sổ thời gian gần đây `started_at >= NOW() - INTERVAL '7 days'` khi lấy status backfill.

---

### 2. Query `ListLatest` / Reconciliation Report (1648.323ms = 1.65s!)
```sql
WITH active_bindings AS (...),
smoke_latest AS (
    SELECT DISTINCT ON (COALESCE(shadow_schema, ''), shadow_table, COALESCE(NULLIF(master_schema, ''), ''), COALESCE(NULLIF(master_table, ''), ''), COALESCE(segment, 'source_shadow'))
           id, run_id, trace_id, cycle_id, segment, source_type, source_host, source_db,
           ...
    FROM cdc_system.cdc_recon_smoke_result
    ORDER BY COALESCE(shadow_schema, ''), shadow_table, COALESCE(NULLIF(master_schema, ''), ''), COALESCE(NULLIF(master_table, ''), ''), COALESCE(segment, 'source_shadow'), checked_at DESC
)
SELECT s.id, ... FROM smoke_latest s ...
```
- **Vấn đề:** 
  1. Bảng `cdc_recon_smoke_result` lưu giữ kết quả từng đợt smoke check (chạy liên tục).
  2. CTE `smoke_latest` dùng `SELECT DISTINCT ON (...) ... ORDER BY ..., checked_at DESC` để chọn kết quả mới nhất của từng bảng.
  3. Việc **không khoanh vùng thời gian `checked_at`** ép PostgreSQL phải quét và sắp xếp (Sort) **TOÀN BỘ dữ liệu lịch sử** của bảng `cdc_recon_smoke_result` trên đĩa/RAM!
  4. Đồng thời bảng `cdc_recon_smoke_result` thiếu index trên `checked_at DESC`.
- **Giải pháp:**
  1. Thêm cờ Time Window Pruning `WHERE checked_at >= NOW() - INTERVAL '7 days'` trong CTE `smoke_latest`. Kết quả smoke check mới nhất của các bảng luôn nằm trong vòng 7 ngày gần đây. Việc này giúp Postgres loại bỏ 99%+ dữ liệu lịch sử cũ khi Sort!
  2. Tạo index composite `CREATE INDEX IF NOT EXISTS idx_smoke_result_checked_at ON cdc_system.cdc_recon_smoke_result (checked_at DESC);`
