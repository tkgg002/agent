# 13_analysis.md — Phân tích chi tiết của AI

## 1. Bản chất sự cố Snapshot `bank_requests` (12.6M rows)
- **Tải trọng**: 12,614,888 documents từ MongoDB sync sang Shadow Table PostgreSQL.
- **Tiến độ trước khi dừng**: 5,125,000 documents (40.63%).
- **Lỗi 1 (Frontend CMS)**: `Heartbeat timeout: progress was stuck in running state for too long (worker stopped)`.
- **Lỗi 2 (Network Trace)**: `dial tcp 10.200.185.20:5432: connect: connection refused`.
- **Nguyên nhân gốc rễ (Root Cause)**:
  - Do chạy unthrottled snapshot, tốc độ ghi hàng chục ngàn dòng/giây vào bảng có nhiều index làm I/O đĩa của PostgreSQL tăng vọt lên 95-100%.
  - PostgreSQL bị bão hòa WAL và Forced Checkpoint dẫn đến service PostgreSQL bị ngắt kết nối / crash / restart.
  - Khi Postgres sập, worker bị lỗi kết nối TCP và dừng lại.
  - Sau 5 phút không có heartbeat, CMS tự động đánh dấu tiến trình thành `Heartbeat timeout`.

## 2. Phân tích giải pháp điều tiết tốc độ (Rate Limiting)
- **Centralized Data Service** đã có sẵn logic `so.SnapshotMaxRPS > 0` tính toán `time.Sleep(expectedDuration - elapsed)` giữa các batch.
- **Cơ chế hoạt động**:
  - Với `batch_size = 3000` và `snapshot_max_rps = 1500`: mỗi batch được trải đều trong 2.0 giây.
  - Sau khi ghi xong 3000 dòng mất ~0.8s, worker sẽ ngủ ~1.2s.
  - Khoảng nghỉ 1.2s giúp PostgreSQL có đủ thời gian flush dirty buffers và xả WAL mà không gây nghẽn đĩa.
- **Tính khả thi của Resume**:
  - `last_seen_id = '69e999af803579b1447f9140'` đã lưu chính xác mốc 5,125,000.
  - Khi Resume với `snapshot_max_rps = 1500`, hệ thống nạp tiếp 7,489,888 dòng còn lại trong khoảng ~1.4 giờ một cách an toàn và tự động 100%.
