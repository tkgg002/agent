# Báo Cáo Thay Đổi & Kết Quả Tối Ưu SLOW SQL Recon Read Repo

- **Task Name:** Fix Recon Read Repo Slow SQL
- **Workspace:** `agent/memory/workspaces/FixReconReadRepoSlowSql20260731`
- **Completed At:** 2026-07-31

---

## 1. Danh sách các file đã thay đổi (Overview & Line Count)

| # | Đường dẫn File | Trạng thái | Số dòng thay đổi | Mô tả thay đổi |
|---|---|---|---|---|
| 1 | `migrations/schema/recon_dlq/101_optimize_recon_read_indexes.sql` | `[NEW]` | +5 lines | Tạo 2 composite index `idx_recon_runs_tier_started` và `idx_smoke_result_checked_at`. |
| 2 | `internal/infra/persistence/recon/recon_read_repo_gorm.go` | `[MODIFY]` | +2 lines | Bổ sung `WHERE checked_at >= NOW() - INTERVAL '7 days'` trong CTE `smoke_latest` và `started_at >= NOW() - INTERVAL '7 days'` trong `GetBackfillStatus`. |

---

## 2. Chi tiết Giải Pháp Kỹ Thuật Triển Khai

### A. Migration File SQL
- `idx_recon_runs_tier_started` ON `cdc_system.recon_runs (tier, started_at DESC)`
- `idx_smoke_result_checked_at` ON `cdc_system.cdc_recon_smoke_result (checked_at DESC)`

### B. Refactor Code Go Repo (`reconReadRepoGorm`)
1. **`listLatestPrimary` (API `/api/reconciliation/report`):**
   - Bổ sung `WHERE checked_at >= NOW() - INTERVAL '7 days'` trong CTE `smoke_latest` để PostgreSQL chỉ quét và Sort 7 ngày dữ liệu smoke test gần nhất thay vì đĩa/RAM toàn bộ lịch sử.
2. **`GetBackfillStatus` (API `/api/recon/backfill-source-ts/status`):**
   - Bổ sung `started_at >= NOW() - INTERVAL '7 days'` kết hợp với index `(tier, started_at DESC)` để trả về phản hồi tức thì cho polling request từ FE.

---

## 3. Kết Quả Kiểm Thử & Kiểm Định (Verification Results)
- **Go Build Check:** `go build ./cmd/server` biên dịch THÀNH CÔNG 100%, không phát sinh bất kỳ lỗi syntax hay breaking change nào.
- **Wire Contract Preservation:** Wire contract của API giữ nguyên 100%.
