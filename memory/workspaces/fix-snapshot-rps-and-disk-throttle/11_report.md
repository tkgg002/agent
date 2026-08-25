# 11_report.md — Báo cáo thay đổi chi tiết sau thực thi (Post-Execution Report)

## Danh sách 8 file mã nguồn đã chỉnh sửa thực tế

| STT | File | Số dòng thay đổi | Nội dung thay đổi chi tiết |
| :--- | :--- | :---: | :--- |
| 1 | `cdc-cms-service/internal/app/queries/source/source_objects_read_models.go` | +2 lines | Bổ sung field `SnapshotMaxRPS *int` với JSON tag `snapshot_max_rps,omitempty` vào `SourceObjectListItem`. |
| 2 | `cdc-cms-service/internal/infra/persistence/source/source_object_read_repo_gorm.go` | +1 line | Bổ sung `so.snapshot_max_rps,` vào câu SQL SELECT trong hàm `ListEnriched`. |
| 3 | `cdc-cms-service/internal/app/commands/source/update_source_object_v2.go` | +22 lines | Bổ sung `SnapshotMaxRPS *int` vào struct, khai báo `ErrSourceObjectInvalidMaxRPS`, hằng số min/max, sửa `Validate()` cho phép update độc lập, và logic `Handle()` map `0` -> `NULL`. |
| 4 | `cdc-cms-service/internal/api/source/source_object_actions_handler.go` | +4 lines | Tiếp nhận `SnapshotMaxRPS` từ request body trong `UpdateMetadata`, map sang Command, và ánh xạ `ErrSourceObjectInvalidMaxRPS` sang HTTP 400. |
| 5 | `cdc-cms-service/internal/api/scheduler/snapshot_progress_handler.go` | +5 lines | Nạp `trace_id` gốc từ bảng `snapshot_progress` và truyền vào NATS payload khi dispatch Resume. |
| 6 | `cdc-cms-web/src/types/index.ts` | +3 lines | Bổ sung `snapshot_max_rps?: number | null;` vào interface `SourceObjectRow`. |
| 7 | `cdc-cms-web/src/pages/TableRegistry.tsx` | +23 lines | Thêm `'snapshot_max_rps'` vào `V2_EXCLUSIVE_FIELDS`, `openEdit`, `handleEdit` (gán 0 khi clear), và thêm `<Form.Item name="snapshot_max_rps">` với `InputNumber`. |
| 8 | `centralized-data-service/internal/handler/orchestration/snapshot_runner_state.go` | +3 lines | Cập nhật câu lệnh UPDATE trong `claimProgress` thành `SET status = 'running', trace_id = ?, error_msg = NULL, updated_at = NOW()`. |
