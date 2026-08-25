# 12_implementation_plan_round2.md — Implementation Plan for Round 2 Fixes

## 1. Mục tiêu
Thực thi các chỉnh sửa phát hiện từ quá trình Adversarial QC (Tasks 5, 6, 7, 8 trong `08_tasks_schema_fix.md`).

## 2. Kế hoạch từng bước

### Bước 1: Sửa tầng CMS API
- File: `cdc-cms-service/internal/api/scheduler/transmute_schedule_handler.go`
- Thêm field `MasterSchema` vào `ScheduleCreateRequest`.
- Thêm validation `schedNameRe` cho `MasterSchema`.
- Map `MasterSchema` vào `CreateTransmuteScheduleCommand`.

### Bước 2: Sửa tầng CDS Worker Repository & Scheduler
- File: `centralized-data-service/internal/service/master/transmute_scheduler.go`
- File: `centralized-data-service/internal/repository/master/master_binding_repo.go`
- Chuyển `mb.master_schema || '.' || mb.master_table` thành `COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table`.

### Bước 3: Sửa tầng Persistence CMS
- File: `cdc-cms-service/internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go`
- Cập nhật câu lệnh `Save()` an toàn với `NULL`.

### Bước 4: Verification
- Chạy `go build ./internal/... ./cmd/...` cho cả 2 services.
- Kiểm tra diff từng dòng code.
