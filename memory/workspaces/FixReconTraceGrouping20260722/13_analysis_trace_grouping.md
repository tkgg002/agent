# 13 Analysis: Phân Phân Cấp OTel Trace Spans Đã Được Đánh Giá Đầy Đủ

## Cấu Trúc Cây Trace Spans Đã Cập Nhật Theo Đúng Phản Hồi Của User:

```text
cdc.recon.chunk_stream_bucket: schedule_histories [2026-07-15 01:38:00 -> 2026-07-22 01:38:00]
 └── cdc.recon.chunk_day_01: schedule_histories [2026-07-15 01:38:00 -> 2026-07-16 01:38:00]
      ├── cdc.recon.hash_window: schedule_histories [09:23:00 -> 09:38:00]
      │    ├── recon.source.hash_window: schedule_histories [09:23:00 -> 09:38:00]
      │    ├── pg.hash_window: schedule_histories [09:23:00 -> 09:38:00]
      │    └── cdc.recon.drift_drill_down: schedule_histories [09:23:00 -> 09:38:00]  (chỉ sinh ra khi phát hiện drift)
      │         ├── recon.source.diff_idts: schedule_histories [09:23:00 -> 09:38:00]
      │         └── pg.diff_idts: schedule_histories [09:23:00 -> 09:38:00]
```
