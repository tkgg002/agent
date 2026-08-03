# Danh sách Task - Fix System Health Queries Slow SQL

- [x] **Task 1: Refactor `queryReconciliation` trong `system_health_queries.go`**
  - Bổ sung `WHERE checked_at >= NOW() - INTERVAL '7 days'` vào câu SQL `queryReconciliation`.

- [x] **Task 2: Build & Verify Test**
  - Chạy `go build ./cmd/server` thành công 100%.
