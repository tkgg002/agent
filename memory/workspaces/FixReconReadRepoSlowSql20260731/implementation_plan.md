# Kế hoạch Tối ưu hóa SLOW SQL Recon Read Repo (cdc-cms-service)

## Mô tả Bài toán & Nguyên nhân Gốc rễ
Hệ thống ghi nhận 2 câu truy vấn SQL bị SLOW SQL (>= 200ms) tại file `internal/infra/persistence/recon/recon_read_repo_gorm.go`:

1. **Slow Query 1 (`GetBackfillStatus` - 690.289ms):**
   - *API:* `/api/recon/backfill-source-ts/status` (bị polling định kỳ từ Frontend)
   - *SQL:* `SELECT * FROM "recon_runs" WHERE tier = 4 AND instance_id LIKE 'backfill:%' ORDER BY started_at DESC LIMIT 30`
   - *Nguyên nhân:* Mệnh đề lọc `WHERE tier = 4 AND instance_id LIKE ... ORDER BY started_at DESC` không có index composite `(tier, started_at DESC)` khiến PostgreSQL phải Seq Scan toàn bộ bảng `recon_runs` rồi mới sort.

2. **Slow Query 2 (`ListLatest` / Reconciliation Report - 1648.323ms = 1.65s!):**
   - *API:* `/api/reconciliation/report`
   - *SQL:* CTE `smoke_latest` dùng `SELECT DISTINCT ON (...) ... FROM cdc_system.cdc_recon_smoke_result ORDER BY ..., checked_at DESC`.
   - *Nguyên nhân:* Việc `DISTINCT ON` không khoanh vùng thời gian `checked_at` buộc Postgres quét và sort TOÀN BỘ dữ liệu lịch sử smoke test trong bảng `cdc_recon_smoke_result` trên đĩa/RAM.

---

## Giải Pháp Tối Ưu Duy Nhất (Single Best Approach)

### 1. Bổ sung Composite Indexes (Migration SQL 101)
Tạo file migration `101_optimize_recon_read_indexes.sql`:
- `idx_recon_runs_tier_started` ON `cdc_system.recon_runs (tier, started_at DESC)`
- `idx_smoke_result_checked_at` ON `cdc_system.cdc_recon_smoke_result (checked_at DESC)`

### 2. Time Window Pruning (Giảm latency 1.65s -> < 20ms)
- **`ListLatest` (`smoke_latest` CTE):** Bổ sung cờ lọc khoanh vùng thời gian `WHERE checked_at >= NOW() - INTERVAL '7 days'`. Chỉ lấy kết quả smoke test trong 7 ngày gần nhất, giúp Postgres loại bỏ 99%+ dữ liệu lịch sử cũ trước khi `DISTINCT ON`.
- **`GetBackfillStatus`:** Bổ sung cờ `started_at >= NOW() - INTERVAL '7 days'` kết hợp với composite index `(tier, started_at DESC)`.

---

## Proposed Changes

### Database Migration

#### [NEW] `migrations/schema/recon_dlq/101_optimize_recon_read_indexes.sql`
- Thêm 2 composite index tối ưu cho `cdc_system.recon_runs` và `cdc_system.cdc_recon_smoke_result`.

---

### cdc-cms-service (Go Backend)

#### [MODIFY] `internal/infra/persistence/recon/recon_read_repo_gorm.go`
- Refactor `listLatestPrimary`: Thêm `WHERE checked_at >= NOW() - INTERVAL '7 days'` trong CTE `smoke_latest`.
- Refactor `GetBackfillStatus`: Thêm `started_at >= NOW() - INTERVAL '7 days'` và tận dụng index composite `(tier, started_at DESC)`.

---

## Verification Plan

### Automated Tests
- Chạy biên dịch kiểm tra syntax Go: `go build ./cmd/server` tại repo `cdc-cms-service`.

### Manual Verification
- Kiểm tra các API:
  - `GET /api/recon/backfill-source-ts/status`
  - `GET /api/reconciliation/report`
- Kiểm tra log latency xem cả 2 truy vấn SQL đã giảm xuống dưới 50ms (kỳ vọng < 15ms) hay chưa.
