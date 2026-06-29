# Context: bug-snapshot-v2-postgresql-zero-records-2026-06-24

## Vấn đề
- **Hiện tại**: Khi chạy snapshot v2 cho nguồn PostgreSQL (`failed_sync_logs`), snapshot runner báo đã started thành công nhưng không đồng bộ được record nào (rows_processed = 0 hoặc kết thúc sớm không có data).
- **Log detail**:
  ```text
  snapshot.v2 scoped to single shadow_binding ... source_object_id=52 shadow_binding_id=52 target_table=failed_sync_logs
  snapshot.v2 started ... progress_id=13 ... connection_code=pg_dev
  snapshot.v2 cluster time captured ... cluster_time_ms=1782267441646 method=pg-clock
  ```
  Nhưng không hề in ra log ghi nhận xử lý batch dữ liệu nào (`batchSize` hoặc `checkpoint`).

## Yêu cầu
- Điều tra tại sao loop cursor PostgreSQL kết thúc ngay lập tức hoặc không tìm thấy records để đồng bộ.
- Khắc phục lỗi và đảm bảo PostgreSQL snapshot v2 có thể đọc và đồng bộ đầy đủ các dòng dữ liệu vào shadow table.
