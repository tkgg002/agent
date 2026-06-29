# Implementation Plan: bug-worker-crash-on-start-2026-06-24

## Kế hoạch triển khai

### Phase 1: Research
- [ ] Chạy `git status` và `git diff` để kiểm tra toàn bộ các thay đổi chưa commit.
- [ ] Kiểm tra file khởi tạo worker: `internal/server/worker_server_init.go` hoặc `main.go`.
- [ ] Tái hiện lỗi bằng cách chạy thử worker local: `go run cmd/worker/main.go` hoặc tương đương để xem output log/panic thực tế.

### Phase 2: Implementation
- [ ] Sửa lỗi gây crash dựa trên kết quả research.

### Phase 3: Verification
- [ ] Chạy thử worker local và kiểm tra xem service có duy trì hoạt động ổn định không.
