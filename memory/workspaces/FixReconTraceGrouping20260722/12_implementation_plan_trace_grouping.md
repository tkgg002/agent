# 12 Implementation Plan: Fix Recon Trace Grouping

## 1. Kế hoạch sửa đổi chi tiết
- **`recon_stream_bucket_engine.go`**: Cập nhật `drillSubWindows` khởi tạo `cdc.recon.hash_window` cho từng cửa sổ 15-phút. Lồng `recon.source.hash_window` và `pg.hash_window` làm con của `cdc.recon.hash_window`. Nếu có drift, khởi tạo `cdc.recon.drift_drill_down` làm con của `cdc.recon.hash_window`, và `recon.source.diff_idts`/`pg.diff_idts` làm con của `cdc.recon.drift_drill_down`.
- **`recon_hash.go` & `recon_dest_hash.go`**: Cập nhật format tiêu đề `recon.source.hash_window: <col> [HH:MM:SS -> HH:MM:SS]` và `pg.hash_window: <table> [HH:MM:SS -> HH:MM:SS]`.
- **`recon_stream.go` & `recon_dest_query.go`**: Cập nhật format tiêu đề `recon.source.diff_idts: <col> [HH:MM:SS -> HH:MM:SS]` và `pg.diff_idts: <table> [HH:MM:SS -> HH:MM:SS]`.

## 2. Kế hoạch Kiểm thử
- Chạy unit test trong package recon: `go test -v ./internal/service/recon/...`.
- Xác nhận biên dịch và không sót lặp span context.
