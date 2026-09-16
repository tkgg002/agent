# 13 AI Analysis — Timezone Handling & Large Scale Transform

## Phân tích kỹ thuật chuyên sâu

### 1. Tại sao snapshot ban đầu đúng nhưng ALTER TYPE sau đó bị sai?
- **Khi Snapshot từ đầu với `timestamptz`:**
  - Debezium / Snapshot runner đọc timestamp từ source (ví dụ: `2026-08-27T03:08:19.555+00:00`).
  - Hàm `coerceToTimeOrNull` parse chuẩn ISO 8601 (RFC3339Nano), có timezone UTC rõ ràng.
  - Sau đó insert trực tiếp vào cột `timestamptz` của PostgreSQL -> PostgreSQL lưu mốc thời gian UTC tuyệt đối chính xác (khi query hiển thị `+07` đúng theo client session).
- **Khi ALTER TYPE `timestamp without time zone` -> `timestamptz`:**
  - Cột ban đầu lưu dạng naive timestamp (không timezone, ví dụ `2026-08-27 02:46:59.793`).
  - Câu lệnh DDL `ALTER TABLE ... TYPE timestamptz USING col::timestamptz` chạy qua connection pool (có timezone UTC).
  - PostgreSQL coi giá trị naive đó là giờ UTC (`2026-08-27 02:46:59.793 UTC`), dẫn đến việc khi client +07 query ra sẽ thấy `2026-08-27 09:46:59.793 +07`, bị lệch đi 7 tiếng so với thực tế.
- **Giải pháp triệt để:**
  - Dùng `Force Transform` để bóc tách lại từ chuỗi raw gốc trong `_raw_data`.
  - `_raw_data` chứa chuỗi gốc kèm timezone offset đầy đủ hoặc epoch milliseconds chuẩn.
  - Biểu thức `BuildCastExpr` cho `timestamptz` ép kiểu `::TIMESTAMPTZ` bảo toàn offset chính xác.

### 2. An toàn tải cho hệ thống 100M+ bản ghi
- Force Transform **không chạy 1 câu UPDATE duy nhất**.
- Hệ thống áp dụng cơ chế **Chunked CTE UPDATE** với cursor-based pagination theo Primary Key (1000 rows/batch).
- Giữa các chunk có đo thời gian thực thi để auto-tune chunk size, không làm cạn kiệt connection pool, không gây lock contention với CDC sink worker.
