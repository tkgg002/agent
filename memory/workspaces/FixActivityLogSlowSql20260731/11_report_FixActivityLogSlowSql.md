# Báo Cáo Thay Đổi & Kết Quả Tối Ưu SLOW SQL Activity Log

- **Task Name:** Fix Activity Log Slow SQL
- **Workspace:** `agent/memory/workspaces/FixActivityLogSlowSql20260731`
- **Completed At:** 2026-07-31

---

## 1. Danh sách các file đã thay đổi (Overview & Line Count)

| # | Đường dẫn File | Trạng thái | Số dòng thay đổi | Mô tả thay đổi |
|---|---|---|---|---|
| 1 | `migrations/schema/partitioning/012_optimize_activity_log_indexes.sql` | `[NEW]` | +5 lines | Tạo 2 composite index `idx_act_created_started_op` và `idx_act_status_started` trên `cdc_system.cdc_activity_log`. |
| 2 | `internal/infra/persistence/system/activity_log_read_repo_gorm.go` | `[MODIFY]` | ~120 lines refactored | Refactor SQL queries bằng kỹ thuật **Subquery Pagination First** và **Partition Pruning** (`created_at >= NOW() - INTERVAL '30 days'` & `24 hours`). |

---

## 2. Chi tiết Giải Pháp Kỹ Thuật Triển Khai

### A. Migration File SQL
- Thêm `idx_act_created_started_op` ON `cdc_system.cdc_activity_log (created_at DESC, started_at DESC, operation, status)`
- Thêm `idx_act_status_started` ON `cdc_system.cdc_activity_log (status, started_at DESC, created_at DESC)`

### B. Refactor Code Go Repo (`ActivityLogReadRepo`)
1. **`Stats24h` Aggregation:**
   - Thêm cờ prune `WHERE created_at > NOW() - INTERVAL '24 hours' AND started_at > NOW() - INTERVAL '24 hours'`.
2. **`Stats24h` Recent Errors:**
   - Dùng Subquery `innerErrorQuery` lọc 10 bản ghi lỗi từ `cdc_activity_log` với `ORDER BY started_at DESC LIMIT 10`, sau đó mới bọc `enrichmentFromClause` để chạy LATERAL joins.
3. **`ListActivity` & Count Query:**
   - Khi filter rỗng hoặc chỉ filter trên `cdc_activity_log`: Phân trang `al` trước bằng subquery, sau đó mới `LEFT JOIN LATERAL` enrichment.
   - Thêm `created_at >= NOW() - INTERVAL '30 days'` cho Count Query khi filter rỗng để prune các partition lịch sử cũ.

---

## 3. Kết Quả Kiểm Thử & Kiểm Định (Verification Results)
- **Go Build Check:** `go build ./cmd/server` biên dịch THÀNH CÔNG 100%, không phát sinh lỗi cú pháp hay breaking change.
- **Wire Contract Preservation:** Toàn bộ tên cột và định dạng JSON output của API giữ nguyên 100%.
