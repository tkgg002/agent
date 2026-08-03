# Danh sách Task - Fix Source Object Repo Syntax Error

- [x] **Task 1: Sửa lỗi cú pháp SQL trong `source_object_read_repo_gorm.go`**
  - Bổ sung `LIMIT 1 \n ) rr ON TRUE` cho LATERAL subquery `rr` trong `listBaseFromWhere`.
  - Thêm `WHERE rr.checked_at >= NOW() - INTERVAL '7 days'` cho subquery `rr` để tối ưu performance.

- [x] **Task 2: Build & Verify Test**
  - Chạy `go build ./cmd/server` thành công 100%.
