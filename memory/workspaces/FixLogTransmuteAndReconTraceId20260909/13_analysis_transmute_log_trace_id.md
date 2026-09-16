# 13 Analysis: Phân tích kỹ thuật & Đánh giá rủi ro (Adversarial Audit)

## 1. Phân tích nguyên nhân gốc rễ (Root Cause Analysis)

### Vấn đề 1: Tab Log Transmute không hiển thị log
1. **Lệch Target Table giữa UI và DB**:
   - Transmute worker ghi nhận `target_table = req.MasterTable` (bảng Master).
   - Component `ReconPipelineGrid.tsx` lại truyền `historyTable` (bảng Shadow) vào hook `usePipelineActivityLog`.
2. **Lệch pha giữa Projection và Filter trong GORM repo**:
   - `projectionColumns()` trong `activity_log_read_repo_gorm.go` đã dùng `COALESCE(tm_so..., so...)`.
   - Nhưng `countQuery` và `mainQuery` lọc cứng `AND so.source_database = ?`. Vì tác vụ `transmute` không join được qua `sb` (shadow) nên `so` bị `NULL`, khiến câu query trả về 0 dòng.
3. **React Query Over-fetching / Data Leak**:
   - Hook `usePipelineActivityLog` có `enabled: !!table || !!sourceDb`. Khi một pipeline không có master binding (`table = null`), điều kiện `!!sourceDb` vẫn kích hoạt query tải về 30 log của các bảng khác trong cùng database.
4. **Nguy cơ Schema-Qualified Table Name (FQN)**:
   - Nếu `pipeline.rowB?.target_table` hoặc `pipeline.masterName` chứa FQN (ví dụ `master_payment.payment_bills`) mà backend lại lọc `(al.target_table = ? OR al.target_table LIKE '%.?')`, việc không strip về bare name sẽ làm hỏng mệnh đề so khớp khi DB chỉ lưu bare name `payment_bills`.

### Vấn đề 2: Trace ID hiển thị sai trên Toast khi trigger đối soát
1. **Backend chưa trả OTel Trace ID**:
   - Endpoint `TriggerCheckAll` (`POST /api/reconciliation/check`) trong `reconciliation_handler_commands.go` bỏ quên việc trích xuất và trả về `trace_id` trong JSON response.
2. **Frontend fallback sai**:
   - `DataIntegrity.tsx` fallback sang `trace.traceId` (UUID client sinh ngẫu nhiên), hoàn toàn lệch với OTel Trace ID của worker trên SigNoz.
   - Nhánh `action.kind === 'heal'` cũng mắc lỗi tương tự.

## 2. Đánh giá tính an toàn và không gây hồi quy (Zero Regression)
- Không thay đổi DDL/schema.
- Không thay đổi interface công khai của API.
- Hook `usePipelineActivityLog` chỉ được dùng tại `ReconPipelineGrid.tsx`.
- Tất cả các sửa đổi bám sát 100% design pattern hiện có.
