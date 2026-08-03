# 08 Tasks: Fix Recon Trace Grouping

- [x] Task 1: Chuẩn hoá tên Span `recon.source.hash_window` & `pg.hash_window` bao gồm `<table/collection> [<start> -> <end>]` trong `recon_hash.go` & `recon_dest_hash.go`.
- [x] Task 2: Điều chỉnh truyền `ctx` trong `recon_stream_bucket_engine.go` tạo sub-window span `cdc.recon.hash_window` 15-phút làm Parent Span cho `recon.source.hash_window`, `pg.hash_window`, và `cdc.recon.drift_drill_down`.
- [x] Task 3: Đảm bảo các hàm gọi `ListIDTsInWindow` (`recon.source.diff_idts`, `pg.diff_idts`) kế thừa `ctxDiff` để nằm trong `cdc.recon.drift_drill_down`.
- [x] Task 4: Sửa signature mock trong `recon_job_handler_test.go` đảm bảo tương thích interface repo.
- [x] Task 5: Chạy unit test & verify compilation (`go test ./internal/...` PASS 100%).
