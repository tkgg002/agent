# 08_tasks_fix_recon_istimestamptz.md: Danh sách Task Chi tiết

- [x] **Task 1:** Thêm logic phân giải `entry.ShadowSchema` từ `ResolveShadowTable` ở đầu hàm `Execute` trong `internal/service/recon/recon_stream_bucket_engine.go`.
- [x] **Task 2:** Chuẩn hoá việc truyền `entryCopy` có `ShadowSchema` sang `ExecuteSegment` trong `internal/service/recon/recon_job_worker.go`.
- [x] **Task 3:** Chạy `go test -v ./internal/service/recon/...` verify 100% Pass.
