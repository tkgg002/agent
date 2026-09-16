# 12 Implementation Plan: Fix Log Transmute & Trace ID Đối soát

## 1. Mục tiêu
Khắc phục triệt để 2 vấn đề tại `http://localhost:5173/data-integrity`:
1. Tab Log Transmute không hiển thị log của phiên mới.
2. Popup Toast hiển thị Trace ID ngẫu nhiên không đúng OTel trace của worker.

## 2. Kế hoạch triển khai chi tiết & Từng line code

### Bước 1: Sửa Backend Go — `cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go`
- **Vị trí**: Hàm `TriggerCheckAll` (dòng 113 - 171).
- **Thực hiện**:
  - Dòng 114: Trích xuất `traceID` từ context OTel:
    ```go
    var traceID string
    if sc := oteltrace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
        traceID = sc.TraceID().String()
    }
    ```
  - Dòng 145: Thêm `"trace_id": traceID` vào JSON map trả về cho nhánh single table.
  - Dòng 171: Thêm `"trace_id": traceID` vào JSON map trả về cho nhánh all tables (`table = "*"`).

### Bước 2: Sửa Backend Go — `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`
- **Vị trí**:
  - `countQuery` (dòng 169 - 180): Sửa filter `f.SourceDatabase`, `f.SourceTable`, `f.ShadowSchema`, `f.ShadowTable`.
  - `mainQuery` (dòng 223 - 234): Sửa filter tương tự.
- **Thực hiện**: Thay thế các điều kiện `so.` và `sb.` bằng `COALESCE(tm_so..., so...)` và `COALESCE(tm_sb..., sb...)`.

### Bước 3: Sửa Frontend React — `cdc-cms-web/src/hooks/useReconStatus.ts`
- **Vị trí**: Dòng 588 trong `usePipelineActivityLog`.
- **Thực hiện**: Thay `enabled: !!table || !!sourceDb` bằng `enabled: Boolean(table)`.

### Bước 4: Sửa Frontend React — `cdc-cms-web/src/components/ReconPipelineGrid.tsx`
- **Vị trí**:
  - Dòng 270 - 280:
    ```typescript
    const rawMaster = pipeline.rowB?.target_table || pipeline.masterName || null;
    const historyMaster = rawMaster ? (rawMaster.split('.').pop() || null) : null;
    ```
    Truyền `historyMaster` vào `usePipelineActivityLog(historyMaster, 'transmute', sourceDb)`.
  - Dòng 771: Tab `log_transmute` bọc điều kiện `!historyMaster ? <Empty ... /> : ...`.

### Bước 5: Sửa Frontend React — `cdc-cms-web/src/pages/DataIntegrity.tsx`
- **Vị trí**:
  - Dòng 312 (`action.kind === 'check-table'`): `traceId: res?.trace_id || undefined`.
  - Dòng 332 (`action.kind === 'heal'`): `traceId: healRes?.trace_id || undefined`.

### Bước 6: Verification & Build
- `cd cdc-cms-service && go build ./...`
- `cd cdc-cms-web && npm run build` (hoặc check build)
