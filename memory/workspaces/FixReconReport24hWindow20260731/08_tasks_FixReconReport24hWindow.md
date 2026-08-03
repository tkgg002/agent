# Danh sách Task - Fix Recon Report 24h Window

- [x] **Task 1: Refactor `listLatestPrimary` dùng 24-hour Window**
  - Đổi `WHERE checked_at >= NOW() - INTERVAL '7 days'` thành `WHERE checked_at >= NOW() - INTERVAL '24 hours'` trong CTE `smoke_latest` của file `recon_read_repo_gorm.go`.

- [x] **Task 2: Build & Verify Test**
  - Chạy `go build ./cmd/server` thành công 100%.
