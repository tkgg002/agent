# 08_tasks.md — Danh sách Task chi tiết

## Phase 1: Backend Implementation (`cdc-cms-service`)
- [x] Task 1.1: Cập nhật `source_objects_read_models.go` thêm `SnapshotMaxRPS *int`.
- [x] Task 1.2: Cập nhật `source_object_read_repo_gorm.go` SELECT cột `so.snapshot_max_rps`.
- [x] Task 1.3: Cập nhật `update_source_object_v2.go` (Command struct, Validate, Handle).
- [x] Task 1.4: Cập nhật `source_object_actions_handler.go` nhận `SnapshotMaxRPS` từ request body.
- [x] Task 1.5: Cập nhật `snapshot_progress_handler.go` truyền `trace_id` khi resume.

## Phase 2: Frontend Implementation (`cdc-cms-web`)
- [x] Task 2.1: Cập nhật interface `SourceObjectRow` trong `types/index.ts`.
- [x] Task 2.2: Cập nhật `V2_EXCLUSIVE_FIELDS` trong `TableRegistry.tsx`.
- [x] Task 2.3: Cập nhật Form initialization (`openEdit`) và submit handler (`handleEdit`) trong `TableRegistry.tsx`.
- [x] Task 2.4: Thêm InputNumber UI component vào Modal *"Chỉnh sửa Source Object"*.

## Phase 3: Worker Implementation (`centralized-data-service`)
- [x] Task 3.1: Cập nhật `snapshot_runner_state.go` cập nhật `trace_id` và xóa `error_msg` khi resume.

## Phase 4: Validation & Handoff
- [x] Task 4.1: Chạy verify end-to-end cập nhật `snapshot_max_rps`.
- [x] Task 4.2: Viết báo cáo hoàn thành `11_report.md` và `14_walkthrough.md`.

