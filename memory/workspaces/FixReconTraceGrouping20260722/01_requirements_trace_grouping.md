# 01 Requirements: Chuẩn hoá cấu trúc và lồng ghép OTel Trace Spans cho CDC Recon Jobs

## 1. Bối cảnh & Yêu cầu Phân cấp từ User
Trong Jaeger UI / OpenTelemetry Tracing, luồng đối soát `cdc.recon.chunk_stream_bucket` phải tuân thủ phân cấp 4 tầng chính xác sau:

```text
cdc.recon.chunk_stream_bucket: schedule_histories [2026-07-15 01:38:00 -> 2026-07-22 01:38:00]
 └── cdc.recon.chunk_day_01: schedule_histories [2026-07-15 01:38:00 -> 2026-07-16 01:38:00]
      ├── cdc.recon.hash_window: schedule_histories [09:23:00 -> 09:38:00]
      │    ├── recon.source.hash_window: schedule_histories [09:23:00 -> 09:38:00]
      │    ├── pg.hash_window: schedule_histories [09:23:00 -> 09:38:00]
      │    └── cdc.recon.drift_drill_down: schedule_histories [09:23:00 -> 09:38:00]  (chỉ sinh ra khi có drift)
      │         ├── recon.source.diff_idts: schedule_histories [09:23:00 -> 09:38:00]
      │         └── pg.diff_idts: schedule_histories [09:23:00 -> 09:38:00]
```

## 2. Quy tắc lồng ghép Context (Context Propagation Rules)
1. **`cdc.recon.hash_window`**: Sinh ra cho mỗi 15 phút (sub-window), là Child Span trực tiếp của `cdc.recon.chunk_day_x`.
2. **`recon.source.hash_window` & `pg.hash_window`**: Sinh ra khi thực hiện so khớp XOR Hash 15 phút, là Child Spans trực tiếp của `cdc.recon.hash_window`.
3. **`cdc.recon.drift_drill_down`**: Khi phát hiện chênh lệch Hash/Count trong 15 phút, span này được khởi tạo làm Child Span trực tiếp của `cdc.recon.hash_window`.
4. **`recon.source.diff_idts` & `pg.diff_idts`**: Khi drill-down lấy danh sách ID chênh lệch, hai span này được khởi tạo làm Child Spans trực tiếp của `cdc.recon.drift_drill_down`.

## 3. Definition of Done
- Mã nguồn `recon_stream_bucket_engine.go` truyền đúng context phân cấp theo 4 tầng trên.
- Mã nguồn `recon_hash.go`, `recon_dest_hash.go`, `recon_stream.go`, `recon_dest_query.go` sử dụng định dạng tiêu đề span động kèm tên bảng và mốc thời gian `[HH:MM:SS -> HH:MM:SS]`.
- Verify biên dịch thành công (`go test ./...`) và không có Goroutine/Context leak.
