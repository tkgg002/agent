# 08_tasks_recon_no_index.md

## Danh sách công việc (Implementation Tasks)

- [ ] Task 1 [Code thêm / Platform Enhancement]: Cho phép cấu hình `QueryTimeout` và `BatchSize` động cho `ReconSourceAgent` & `ReconDestAgent` thông qua `WorkerConfig` / Environment variables (`RECON_QUERY_TIMEOUT`, `RECON_BATCH_SIZE`).
- [ ] Task 2 [Code thêm / Platform Enhancement]: Nâng cấp `HashWindow` trong `recon_hash.go`:
  - Thêm cơ chế Safe ObjectId Check (Type Guard: chỉ áp dụng khi `_id` là `primitive.ObjectID` VÀ bảng không có in-place update).
  - Tối ưu Find options (`SetBatchSize(5000)`).
- [ ] Task 3 [Vận hành CMS]: Kiểm tra metadata của `order_updates_bvb` trong `source_object_registry` để xác định chính xác kiểu ID và đặc tính append-only.
- [ ] Task 4 [Verification]: Chạy test đối soát và đo đạc latency trên SigNoz / OpenTelemetry.
