# Plan: Bug Snapshot Limit 5000 Records / Kế hoạch: Lỗi Snapshot giới hạn 5000 records

## English
### Phase 1: Investigation & Research
1. Locate `snapshot_runner` component and trace the snapshot v2 implementation in `centralized-data-service`.
2. Inspect how batches are fetched, query limits, or hardcoded limits (like 5000 records or 2500 per batch since `batches_total` is 2).
3. Identify why `rows_total` stopped at 5000.
   - *Result*: MongoDB `_id` is numeric (int32/int64/float64), but resume filter query uses string: `{ "_id": { "$gt": "5816" } }`, causing MongoDB to return 0 records on subsequent batches.
4. Check compatibility for other databases (PostgreSQL, MySQL).
   - *Result*: PostgreSQL already handles type casting properly. MySQL is currently blocked/not enabled in `snapshot.v2`, but will use SQL logic similar to PostgreSQL when enabled.

### Phase 2: Implementation Plan
1. Retrieve a sample document at start to detect the type of `_id` in MongoDB.
2. Implement `buildResumeFilterWithSample` to dynamically cast `lastSeen` to numeric types if `_id` type is numeric.
3. Update `snapshot_runner_handler.go` and `snapshot_runner_utils.go`.

### Phase 3: Verification
1. Run existing unit tests:
   ```bash
   go test -v ./internal/handler/orchestration/...
   ```
2. Write a dedicated unit test for `buildResumeFilterWithSample`.

---

## Tiếng Việt
### Giai đoạn 1: Điều tra & Nghiên cứu
1. Định vị component `snapshot_runner` và tìm logic thực thi snapshot v2 trong `centralized-data-service`.
2. Kiểm tra cách lấy dữ liệu theo batch, các cấu hình giới hạn query (limit), hoặc giới hạn hardcode (ví dụ giới hạn 5000 records hoặc 2500 mỗi batch vì `batches_total` là 2).
3. Xác định tại sao `rows_total` dừng ở 5000.
   - *Kết quả*: `_id` của MongoDB là kiểu số (int32/int64/float64), nhưng câu truy vấn resume filter lại dùng chuỗi: `{ "_id": { "$gt": "5816" } }`, khiến MongoDB trả về 0 bản ghi ở các batch tiếp theo.
4. Kiểm tra khả năng tương thích với các loại database khác (PostgreSQL, MySQL).
   - *Kết quả*: PostgreSQL đã được xử lý ép kiểu phù hợp. MySQL hiện tại bị chặn/chưa kích hoạt trong `snapshot.v2`, nhưng khi kích hoạt sẽ dùng logic truy vấn SQL tương tự PostgreSQL.

### Giai đoạn 2: Kế hoạch thực thi
1. Lấy một bản ghi mẫu khi bắt đầu để phát hiện kiểu dữ liệu của `_id` trong MongoDB.
2. Triển khai `buildResumeFilterWithSample` để tự động ép kiểu `lastSeen` về đúng kiểu số nếu kiểu của `_id` là kiểu số.
3. Cập nhật `snapshot_runner_handler.go` và `snapshot_runner_utils.go`.

### Giai đoạn 3: Xác minh
1. Chạy các unit test hiện tại:
   ```bash
   go test -v ./internal/handler/orchestration/...
   ```
2. Viết unit test chuyên biệt cho `buildResumeFilterWithSample`.
