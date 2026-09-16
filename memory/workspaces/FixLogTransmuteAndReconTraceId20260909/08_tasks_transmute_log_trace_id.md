# 08 Tasks: Fix Log Transmute & Trace ID Đối soát

## Danh sách Task chi tiết

- [x] **TASK-01 (Backend Go - Trace ID)**:
  - File: `cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go`
  - Bổ sung trích xuất OTel trace ID từ `ctx` và trả về trường `"trace_id": traceID` trong hàm `TriggerCheckAll` (cả nhánh `table` đơn và nhánh `table = "*"`).

- [x] **TASK-02 (Backend Go - GORM Read Repo)**:
  - File: `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`
  - Sửa `countQuery` và `mainQuery`: đổi các điều kiện `so.source_database`, `so.source_object_name`, `sb.shadow_schema`, `sb.shadow_table` sang dùng `COALESCE(tm_so..., so...)` và `COALESCE(tm_sb..., sb...)`.

- [x] **TASK-03 (Frontend React - Hook Over-fetching Guard)**:
  - File: `cdc-cms-web/src/hooks/useReconStatus.ts`
  - Đổi `enabled: !!table || !!sourceDb` sang `enabled: Boolean(table)` trong `usePipelineActivityLog`.

- [x] **TASK-04 (Frontend React - Grid Transmute Bare Name & Log Binding)**:
  - File: `cdc-cms-web/src/components/ReconPipelineGrid.tsx`
  - Chuẩn hóa bare table name: `const rawMaster = pipeline.rowB?.target_table || pipeline.masterName || null; const historyMaster = rawMaster ? (rawMaster.split('.').pop() || null) : null;`
  - Truyền `historyMaster` vào `usePipelineActivityLog`.
  - Cập nhật tab `log_transmute` render `Empty` khi `!historyMaster`.

- [x] **TASK-05 (Frontend React - Toast Trace ID Normalization)**:
  - File: `cdc-cms-web/src/pages/DataIntegrity.tsx`
  - Sửa cả 2 vị trí `action.kind === 'check-table'` và `action.kind === 'heal'` dùng `traceId: res?.trace_id || undefined` và `traceId: healRes?.trace_id || undefined`.

- [x] **TASK-06 (Verification & Build Check)**:
  - Kiểm tra compile và cú pháp mã nguồn Go và TypeScript đã sửa đổi.

- [x] **TASK-07 (Worker Go - Day Chunk Progress Calculation)**:
  - File: `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go`
  - Thêm `jobRepo ReconJobRepository`, hàm `reportProgress`, tính `totalDays` và cập nhật `progress_percent` + `checkpoint_ts` sau mỗi ngày trong `Execute` và `executeSegmentB`.
  - File: `centralized-data-service/internal/server/server_setup.go`
  - Nối `chunkEngine.WithJobRepo(reconJobRepo)`.

