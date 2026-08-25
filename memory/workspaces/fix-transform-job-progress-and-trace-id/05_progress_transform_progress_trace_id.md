# 05_PROGRESS: AUDIT LOG (APPEND ONLY)

## [2026-08-25T08:45] [Agent:Brain] KHỞI TẠO WORKSPACE & LẬP KẾ HOẠCH TOÀN TRÌNH
- Phát hiện root cause của 2 issue:
  1. `batch_transform_handler.go` đặt `heartbeatEvery := 50` làm các job < 50k rows không bao giờ cập nhật progress trong khi chạy. Đồng thời thiếu đếm trước `totalPendingRows` để tính %.
  2. `transform_jobs` có `trace_id` nhưng API và Read Model không trả về JSON, Frontend không hiển thị.
  3. SQL LATERAL JOIN trong Read Repo bị trượt FQN khiến F5 mất dữ liệu.
- Đã tạo implementation plan `implementation_plan.md` chờ user approve.

## [2026-08-25T09:00] [Agent:Muscle] HOÀN TẤT TRIỂN KHAI TOÀN DIỆN (FULL-LOOP GREEN)
- **Database Migration:** Đã tạo file migration `103_add_total_rows_to_jobs.sql` bổ sung cột `total_rows BIGINT` vào `cdc_system.transform_jobs` và `cdc_system.transmute_jobs`.
- **Worker Engine (`centralized-data-service`):**
  - Cập nhật `transform_job_repo.go` & `transmute_job_repo.go`: Bổ sung `TotalRows`, hỗ trợ lưu `total_rows` trong `UpdateStatus` & `UpdateProgress`.
  - Cập nhật `batch_transform_handler.go`: Bổ sung `TraceID` vào `BatchTransformPayload`, đếm trước `totalPendingRows` để tính live % chính xác, gọi `UpdateProgress` mỗi chunk.
  - Cập nhật `transmuter.go`: Đếm trước `totalShadowRows`, tính toán live % chính xác và cập nhật `UpdateProgress` mỗi batch.
  - Cập nhật unit test `batch_transform_handler_test.go` -> PASS 100%.
- **CMS Backend (`cdc-cms-service`):**
  - Cập nhật `persistence.TransformJob` & `persistence.TransmuteJob` với trường `TotalRows`.
  - Nâng cấp `GetLatestBySourceObjectID`: Kiểm tra `source_object_id` trực tiếp trước, fallback qua candidate target tables và schema FQN.
  - Cập nhật `source_object_actions_handler.go`: Gửi `TraceID` trong NATS payload `cdc.cmd.batch-transform`; trả `total_rows` và `trace_id` trong `TransformJobStatusV2`.
  - Cập nhật `master_transmute_job_handler.go`: Trả `total_rows` và `trace_id` trong `TransmuteJobStatus`.
  - Cập nhật Read Repos & CQRS models: `source_object_read_repo_gorm.go`, `master_read_repo_gorm.go`, `source_objects_read_models.go`, `list_masters.go` với SELECT `total_rows`, `trace_id` và LATERAL joins FQN-safe.
  - Sửa mock `stubReconReader` trong test suite -> `go test ./test/...` PASS 100%.
- **CMS Frontend (`cdc-cms-web`):**
  - Cập nhật `types/index.ts` với `last_transform_total_rows`, `last_transform_trace_id`, `last_transmute_total_rows`, `last_transmute_trace_id`.
  - Cập nhật `TableRegistry.tsx` (`TransformJobStatus`): Hiển thị live % + `<hoàn_thành> / <tổng_số> rows` khi chạy, hiển thị kết quả sau F5 và nút icon copy `<CopyOutlined />` SigNoz Trace ID siêu gọn.
  - Cập nhật `MasterRegistry.tsx` (`TransmuteJobStatus`): Hiển thị live % + `<hoàn_thành> / <tổng_số> rows` khi chạy, hiển thị kết quả sau F5 và nút icon copy `<CopyOutlined />` SigNoz Trace ID siêu gọn.
  - `npm run build` PASS 100% không warning/lỗi TypeScript.
