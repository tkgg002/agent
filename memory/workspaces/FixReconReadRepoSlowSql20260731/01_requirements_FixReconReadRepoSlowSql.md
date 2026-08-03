# Yêu cầu Tối ưu hóa SLOW SQL Recon Read Repo (FixReconReadRepoSlowSql)

## 1. Bối cảnh & Hiện trạng
Hệ thống ghi nhận 2 câu truy vấn SQL bị SLOW SQL (>= 200ms) tại file `internal/infra/persistence/recon/recon_read_repo_gorm.go`:

1. **Slow Query 1 (Line 479 - Backfill Status Query):** 690.289ms
   - API: `/api/recon/backfill-source-ts/status` (bị polling mỗi 5s từ FE)
   - SQL: `SELECT * FROM "recon_runs" WHERE tier = 4 AND instance_id LIKE 'backfill:%' ORDER BY started_at DESC LIMIT 30`
   - Nguyên nhân: Thiếu composite index `(tier, started_at DESC)` trên `recon_runs`.

2. **Slow Query 2 (Line 157 - Reconciliation Report Latest List):** 1648.323ms (1.65s!)
   - API: `/api/reconciliation/report`
   - SQL: CTE `smoke_latest` dùng `SELECT DISTINCT ON (...) ... FROM cdc_system.cdc_recon_smoke_result ORDER BY ..., checked_at DESC`.
   - Nguyên nhân: Quét và sort toàn bộ bảng `cdc_recon_smoke_result` không khoanh vùng thời gian (thiếu `checked_at > NOW() - INTERVAL '7 days'`) và thiếu index composite.

## 2. Mục tiêu (Definition of Done)
- [ ] Giảm latency của API `/api/recon/backfill-source-ts/status` từ **690ms -> < 10ms**.
- [ ] Giảm latency của API `/api/reconciliation/report` từ **1.65s -> < 50ms**.
- [ ] Tạo migration file SQL bổ sung composite indexes:
  - `idx_recon_runs_tier_started` ON `cdc_system.recon_runs (tier, started_at DESC)`
  - `idx_smoke_result_checked_at` ON `cdc_system.cdc_recon_smoke_result (checked_at DESC)`
- [ ] Refactor `listLatestPrimary` và `GetBackfillStatus` trong `recon_read_repo_gorm.go` để bổ sung cờ Partition / Time Window pruning (`checked_at >= NOW() - INTERVAL '7 days'`).
- [ ] Đảm bảo 100% wire contract dữ liệu API không bị thay đổi.
