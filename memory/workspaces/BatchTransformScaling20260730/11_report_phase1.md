# 11_report_phase1.md — Báo cáo Thay đổi Phase 1

**Ngày:** 2026-07-31  
**Phase:** Backend Async Transform Engine

---

## Danh Sách File Đã Thay Đổi

| File | Loại | Dòng | Thay đổi |
|------|------|------|----------|
| `cdc-cms-service/migrations/schema/recon_dlq/100_create_transform_jobs.sql` | NEW | 27 | DDL tạo bảng `cdc_system.transform_jobs` + 2 index |
| `cdc-cms-service/internal/infra/persistence/transform_job_repo.go` | NEW | 134 | GORM repo: Create, GetLatestByTable, GetByID, UpdateStatus, UpdateProgress, IsCancelRequested, RequestCancel |
| `centralized-data-service/internal/repository/transform_job_repo.go` | NEW | 91 | Worker-side repo: UpdateStatus, UpdateProgress, IsCancelRequested |
| `centralized-data-service/internal/handler/shadow/batch_transform_handler.go` | MODIFIED | 376→486 (+110) | Refactor sang async goroutine, dynamic chunk, cancel check, finishJob |
| `cdc-cms-service/internal/api/source/source_object_actions_handler.go` | MODIFIED | ~740→852 (+112) | Thêm TransformV2, TransformJobStatusV2, CancelTransformV2, SetTransformJobRepo |
| `cdc-cms-service/internal/router/router.go` | MODIFIED | +2 dòng | Routes: transform-cancel, transform-job-status |
| `cdc-cms-service/internal/server/server.go` | MODIFIED | +2 dòng | Import persistence, inject TransformJobRepo |
| `centralized-data-service/internal/server/server_setup.go` | MODIFIED | +1 dòng | Inject TransformJobRepo vào batchTransformHandler |
| `centralized-data-service/internal/handler/shadow/batch_transform_handler_test.go` | MODIFIED | ~195 | Fix test cho async handler + BUG-01 fix |

---

## Tổng Số Dòng Thay Đổi

- **New files:** +252 lines
- **Modified files:** ~+130 lines net
- **Total net addition:** ~+382 lines

---

## Bug Fixes Trong Phase

| Bug | File | Mô tả |
|-----|------|--------|
| BUG-01 | `batch_transform_handler.go` L320 | Break condition dùng `requestedChunkSize` thay vì `chunkSize` đã tune → loop không exit sớm |
| BUG-02 | `transform_job_repo.go` (CMS) L111 | Align `IsCancelRequested` signature → `bool` (không phải `(bool, error)`) |
