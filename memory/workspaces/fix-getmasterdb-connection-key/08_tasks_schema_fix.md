# 08_tasks_schema_fix.md — Detailed Task Checklist

## Danh sách công việc (Work Items)

- [x] **Task 1 (Round 1)**: Thêm `MasterSchema` vào `TransmuteScheduleHeader` (`repository.go`).
- [x] **Task 2 (Round 1)**: Cập nhật `GetHeaderByID` SELECT `master_schema` (`transmute_schedule_repository_gorm.go`).
- [x] **Task 3 (Round 1)**: Cập nhật `run_now.go` build FQN `schema.table`.
- [x] **Task 4 (Round 1)**: Cập nhật `CreateTransmuteScheduleCommand` (`create_schedule.go`).
- [x] **Task 5 (Round 2 - P0)**: Bổ sung `MasterSchema` vào `ScheduleCreateRequest` và controller trong `transmute_schedule_handler.go`.
- [x] **Task 6 (Round 2 - P0)**: Sửa PostgreSQL concat chống NULL: `COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table` trong `transmute_scheduler.go`.
- [x] **Task 7 (Round 2 - P0)**: Sửa PostgreSQL concat chống NULL trong `master_binding_repo.go` (2 queries).
- [x] **Task 8 (Round 2 - P0)**: Sửa `Save()` trong `transmute_schedule_repository_gorm.go` để query an toàn với `NULL`.
- [x] **Task 9 (Verification)**: Build kiểm tra toàn diện 2 services và xác nhận pass 100%.
- [x] **Task 10 (Governance)**: Cập nhật `05_progress.md` và trọn bộ workspace doc set.
