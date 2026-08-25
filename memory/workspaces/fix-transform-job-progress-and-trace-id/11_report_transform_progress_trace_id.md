# 11_REPORT: DANH SÁCH FILE THAY ĐỔI & OVERVIEW

## 1. Tổng quan thay đổi
Bổ sung tính năng hiển thị tiến độ realtime (live % và số rows `<hoàn_thành> / <tổng_số>`) và icon copy SigNoz Trace ID cho cả 2 luồng:
1. **Batch Transform Job** trên màn hình Shadow (`/shadow`)
2. **Transmute Job** trên màn hình Masters (`/masters`)
Đồng thời đảm bảo dữ liệu kết quả và Trace ID vẫn được lưu trữ và hiển thị đầy đủ ngay lập tức khi người dùng tải lại trang (F5).

---

## 2. Danh sách File và Số dòng thay đổi

| Service / Project | File | Số dòng thay đổi | Mô tả thay đổi |
| :--- | :--- | :---: | :--- |
| `cdc-cms-service` | `migrations/schema/recon_dlq/103_add_total_rows_to_jobs.sql` | +7 lines | DDL thêm `total_rows` vào `transform_jobs` và `transmute_jobs` |
| `centralized-data-service` | `internal/repository/transform_job_repo.go` | ~25 lines | Bổ sung `TotalRows`, cập nhật `UpdateStatus` & `UpdateProgress` |
| `centralized-data-service` | `internal/repository/transmute_job_repo.go` | ~25 lines | Bổ sung `TotalRows`, cập nhật `UpdateStatus` & `UpdateProgress` |
| `centralized-data-service` | `internal/handler/shadow/batch_transform_handler.go` | ~60 lines | Thêm `TraceID`, đếm `totalPendingRows`, cập nhật % progress mỗi chunk |
| `centralized-data-service` | `internal/service/master/transmuter.go` | ~45 lines | Thêm `countShadowRows`, tính % tiến độ và cập nhật % mỗi batch |
| `centralized-data-service` | `internal/handler/shadow/batch_transform_handler_test.go` | ~12 lines | Cập nhật mock test `SELECT COUNT(*)` |
| `cdc-cms-service` | `internal/infra/persistence/transform_job_repo.go` | ~30 lines | Thêm `TotalRows`, nâng cấp candidate tables lookup |
| `cdc-cms-service` | `internal/infra/persistence/transmute_job_repo.go` | ~10 lines | Thêm `TotalRows` vào struct |
| `cdc-cms-service` | `internal/api/source/source_object_actions_handler.go` | ~15 lines | Gửi `trace_id` trong NATS, trả `total_rows`, `trace_id` trong status API |
| `cdc-cms-service` | `internal/api/master/master_transmute_job_handler.go` | ~10 lines | Trả `total_rows`, `trace_id` trong status API |
| `cdc-cms-service` | `internal/infra/persistence/source/source_object_read_repo_gorm.go` | ~20 lines | SELECT `total_rows`, `trace_id` và sửa LATERAL join FQN-safe |
| `cdc-cms-service` | `internal/infra/persistence/master/master_read_repo_gorm.go` | ~15 lines | SELECT `total_rows`, `trace_id` và sửa LATERAL join FQN-safe |
| `cdc-cms-service` | `internal/app/queries/source/source_objects_read_models.go` | ~10 lines | Bổ sung DTO fields `last_transform_total_rows`, `last_transform_trace_id` |
| `cdc-cms-service` | `internal/app/queries/master/list_masters.go` | ~8 lines | Bổ sung DTO fields `last_transmute_total_rows`, `last_transmute_trace_id` |
| `cdc-cms-service` | `test/internal/app/queries/queries_test.go` | +3 lines | Bổ sung mock method `GetActiveReconJobs` |
| `cdc-cms-web` | `src/types/index.ts` | ~10 lines | Bổ sung types `last_transform_total_rows`, `last_transform_trace_id` |
| `cdc-cms-web` | `src/pages/TableRegistry.tsx` | ~50 lines | Cập nhật UI live %, `<hoàn_thành> / <tổng_số>` và icon copy Trace ID |
| `cdc-cms-web` | `src/pages/MasterRegistry.tsx` | ~50 lines | Cập nhật UI live %, `<hoàn_thành> / <tổng_số>` và icon copy Trace ID |
