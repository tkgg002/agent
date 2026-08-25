# Solution & Task Breakdown: Transform Job Isolation

## Danh sách Task Thực thi (Muscle Role)

- [ ] **Task 1:** Tạo migration `cdc-cms-service/migrations/schema/recon_dlq/088_add_source_object_id_to_transform_jobs.sql`.
- [ ] **Task 2:** Cập nhật `TransformJob` struct và method `Create` / `GetLatestBySourceObjectID` trong `cdc-cms-service/internal/infra/persistence/transform_job_repo.go`.
- [ ] **Task 3:** Cập nhật `source_object_actions_handler.go` để truyền `id` (`sourceObjectID`) khi gọi `transformJobRepo.Create`.
- [ ] **Task 4:** Cập nhật SQL LATERAL JOIN trong `cdc-cms-service/internal/infra/persistence/source/source_object_read_repo_gorm.go` (`ListEnriched` & `ListEnrichedBindings`).
- [ ] **Task 5:** Compile và kiểm thử `cdc-cms-service`.
