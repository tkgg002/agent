# Kế hoạch triển khai - Sửa lỗi hiển thị tổng record trong tab Pipeline

## 1. Thành phần thay đổi

### Backend: `recon_read_repo_gorm.go`
*   **Mục tiêu:** Loại bỏ các hàm `COALESCE` dùng để fallback sang dữ liệu của `cdc_reconciliation_report` trong query `listLatestPrimary`.
*   **Thay đổi chi tiết:**
    ```sql
    -- Trước:
    COALESCE(s.source_total, r.total_source_count) AS source_total,
    COALESCE(s.source_active, r.source_count) AS source_active,
    COALESCE(s.shadow_total, r.total_dest_count) AS shadow_total,
    COALESCE(s.shadow_active, r.dest_count) AS shadow_active,

    -- Sau:
    s.source_total AS source_total,
    s.source_active AS source_active,
    s.shadow_total AS shadow_total,
    s.shadow_active AS shadow_active,
    ```

### Frontend: `ReconPipelineGrid.tsx`
*   **Mục tiêu:** Thay đổi cách lấy các mốc đếm sang các trường `source_active`, `source_total`, `shadow_active`, `shadow_total`, `master_active`, `master_total`.
*   **Thay đổi chi tiết:**
    1.  Trong nhánh có cả `a` và `b`:
        ```typescript
        const sourceTotal = aCountable ? (a.source_active ?? a.source_total ?? null) : null;
        const shadowActive = (aCountable ? (a.shadow_active ?? a.shadow_total ?? null) : null) ?? b.source_active ?? b.source_total ?? null;
        const masterActive = b.master_active ?? b.master_total ?? null;
        ```
    2.  Trong nhánh chỉ có `a` (chưa có master binding):
        ```typescript
        const sourceTotal = aCountable ? (a.source_active ?? a.source_total ?? null) : null;
        const shadowTotal = aCountable ? (a.shadow_active ?? a.shadow_total ?? null) : null;
        ```

## 2. Kế hoạch kiểm thử & Xác minh
*   Chạy biên dịch Backend: `go build ./...` trong `cdc-cms-service`.
*   Chạy biên dịch Frontend: `npx tsc --noEmit` trong `cdc-cms-web`.
*   Chạy linter quy trình: `python3 agent/tooling/verify_governance.py`.
