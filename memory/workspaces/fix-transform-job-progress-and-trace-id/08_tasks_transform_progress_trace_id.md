# 08_TASKS: DANH MỤC THỰC THI (FULL SCOPE)

- [ ] Task 1: Migration DDL `103_add_total_rows_to_jobs.sql` thêm `total_rows` cho `transform_jobs` và `transmute_jobs`.
- [ ] Task 2: Worker Engine (`centralized-data-service`):
  - [ ] 2.1 Cập nhật `transform_job_repo.go` và `transmute_job_repo.go` nhận `total_rows`.
  - [ ] 2.2 `batch_transform_handler.go`: Đếm trước `totalPendingRows`, cập nhật `UpdateProgress` mỗi chunk.
  - [ ] 2.3 `transmuter.go`: Đếm trước `totalShadowRows`, cập nhật `UpdateProgress` mỗi batch.
- [ ] Task 3: CMS Backend (`cdc-cms-service`):
  - [ ] 3.1 Cập nhật model và repo `transform_job_repo.go`, `transmute_job_repo.go`.
  - [ ] 3.2 `source_object_actions_handler.go` & `master_transmute_job_handler.go`: Trả `total_rows`, `trace_id` trong JSON.
  - [ ] 3.3 Tối ưu SQL LATERAL join trong `source_object_read_repo_gorm.go` và `master_read_repo_gorm.go`.
  - [ ] 3.4 Cập nhật DTOs `source_objects_read_models.go` và `list_masters.go`.
- [ ] Task 4: Frontend Web (`cdc-cms-web`):
  - [ ] 4.1 Cập nhật `types/index.ts`.
  - [ ] 4.2 Cập nhật `TableRegistry.tsx` (`TransformJobStatus`): Live % + rows/total + compact copy icon.
  - [ ] 4.3 Cập nhật `MasterRegistry.tsx` (`TransmuteJobStatus`): Live % + rows/total + compact copy icon.
- [ ] Task 5: Build, Test và Xác minh toàn diện trên 3 repository.
