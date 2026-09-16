# 01 Requirements: Sửa lỗi Log Transmute & Trace ID Đối soát (Data Integrity)

## 1. Bối cảnh & Vấn đề
Tại trang `http://localhost:5173/data-integrity` (Nhật ký đối soát 30 phiên gần nhất):
1. **Tab "Log Transmute"** không lấy được phiên mới của pipeline bảng hiện tại do:
   - Frontend truyền `historyTable` (shadow table) thay vì Master Table.
   - Backend GORM lọc cứng `so.source_database = ?` nên khi `target_table` là master table thì `so` bị `NULL` (do không join qua `sb`), làm câu query trả về 0 dòng.
   - React Query hook `usePipelineActivityLog` bị over-fetching khi pipeline chưa có Master Binding (`enabled: !!table || !!sourceDb`).
2. **Popup Toast khi bấm đối soát / heal** hiển thị `trace_id` ngẫu nhiên sinh từ Frontend (`actionTrace.ts`), không khớp với OTel Trace ID thật được lưu trong backend và worker (`recon_jobs.trace_id`) do:
   - Endpoint `TriggerCheckAll` (`POST /api/reconciliation/check`) ở backend chưa trích xuất và chưa trả về `trace_id`.
   - Frontend fallback về `trace.traceId` (UUID client sinh ngẫu nhiên).

## 2. Phạm vi thay đổi (Scope)
- **Backend (`cdc-cms-service`)**:
  - `internal/api/recon/reconciliation_handler_commands.go`: Trích xuất OTel trace ID từ context trong `TriggerCheckAll` và trả về `trace_id` trong JSON response.
  - `internal/infra/persistence/system/activity_log_read_repo_gorm.go`: Đồng bộ `COALESCE(tm_so.source_database, so.source_database)` và các trường tương ứng trong `countQuery` và `mainQuery`.
- **Frontend (`cdc-cms-web`)**:
  - `src/hooks/useReconStatus.ts`: Cập nhật `enabled: Boolean(table)` trong `usePipelineActivityLog`.
  - `src/components/ReconPipelineGrid.tsx`: Chuẩn hóa bare table name qua `rawMaster.split('.').pop()`, truyền `historyMaster` vào `usePipelineActivityLog`, render `Empty` khi chưa có master binding.
  - `src/pages/DataIntegrity.tsx`: Đồng bộ loại bỏ fallback client UUID giả ở cả `check-table` và `heal`, ép kiểu `traceId: res?.trace_id || undefined`.
