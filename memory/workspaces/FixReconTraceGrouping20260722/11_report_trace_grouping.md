# 11 Report: Báo Cáo Thay Đổi Mã Nguồn Fix Recon Trace Grouping

## 1. Overview
Đã hoàn thành chuẩn hoá phân cấp 4 tầng OpenTelemetry Trace Spans cho luồng đối soát CDC trong `centralized-data-service`.

## 2. Thống Kê Các File Thay Đổi

| File | Số dòng thay đổi | Nội dung thay đổi |
| --- | --- | --- |
| `internal/service/recon/recon_stream_bucket_engine.go` | ~45 lines | Tạo `cdc.recon.hash_window` cho từng cửa sổ 15-phút làm parent span. Lồng `recon.source.hash_window` và `pg.hash_window` vào `cdc.recon.hash_window`. Lồng `cdc.recon.drift_drill_down` vào `cdc.recon.hash_window` khi phát hiện chênh lệch. |
| `internal/service/recon/recon_hash.go` | ~7 lines | Đặt tên động `recon.source.hash_window: <col> [15:04:05 -> 15:04:05]` |
| `internal/service/recon/recon_dest_hash.go` | ~7 lines | Đặt tên động `pg.hash_window: <table> [15:04:05 -> 15:04:05]` |
| `internal/service/recon/recon_stream.go` | ~7 lines | Đặt tên động `recon.source.diff_idts: <col> [15:04:05 -> 15:04:05]` |
| `internal/service/recon/recon_dest_query.go` | ~7 lines | Đặt tên động `pg.diff_idts: <table> [15:04:05 -> 15:04:05]` |
| `internal/handler/recon/recon_job_handler_test.go` | ~4 lines | Bổ sung `UpdateStatusExtended` vào `mockJobRepoHandler` |

## 3. Kết Quả Kiểm Thử (Verification Output)
- `go test ./internal/...`: **PASS** (100% test pass).
