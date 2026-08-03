# Task Breakdown: Fix Execute Heal Segment B ID Resolution

- [ ] Task 1: Refactor ID Resolution helper trong `recon_execute_heal_handler.go` để chuyển đổi linh hoạt giữa `_source_id` và `_gpay_id`.
- [ ] Task 2: Fix Prune Master DELETE SQL hỗ trợ cả `_source_id` lẫn `_gpay_id`.
- [ ] Task 3: Chạy test suite `go test ./internal/handler/recon/...` để verify.
