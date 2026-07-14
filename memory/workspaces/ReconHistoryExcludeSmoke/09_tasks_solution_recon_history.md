# Hồ Sơ Giải Pháp Kỹ Thuật - Lọc Bỏ Smoke Check Trong Lịch Sử Chữa Lành

Tài liệu này mô tả chi tiết thiết kế kỹ thuật để loại bỏ toàn bộ bản ghi Smoke Check (đối soát nhanh) khỏi tab "Phiên đã xử lý" (Healed Reports) của modal Chữa lành.

## 1. Thay đổi tầng Backend (cdc-cms-service)

### Interface & Repository (recon_reader.go & recon_read_repo_gorm.go)
- Sửa đổi phương thức `GetTableHistory`:
  `GetTableHistory(ctx context.Context, table, shadowSchema, masterTable string, excludeSmoke bool, page, pageSize int) ([]reconmodel.ReconciliationReport, int64, error)`
- Trong implementation tại `recon_read_repo_gorm.go`:
  - Nếu `excludeSmoke` là `true`: Không sử dụng `unionQuery` (không UNION với `cdc_recon_smoke_result`), mà chỉ sử dụng `baseQuery` (query trực tiếp từ bảng `cdc_reconciliation_report`).
  - Nếu `excludeSmoke` là `false`: Sử dụng `unionQuery` như cũ.

### Application Query (get_table_history.go)
- Thêm trường `ExcludeSmoke bool` vào struct `GetTableHistoryQuery`.
- Gọi hàm repository với tham số `q.ExcludeSmoke`.

### HTTP Controller (reconciliation_handler_reports.go)
- Trong hàm `TableHistory`:
  `ExcludeSmoke: c.Query("exclude_smoke") == "true",`

### Unit Tests Mock (queries_test.go)
- Cập nhật định nghĩa `stubReconReader.GetTableHistory` để thêm tham số `excludeSmoke bool` cho khớp với signature interface.

---

## 2. Thay đổi tầng Frontend (cdc-cms-web)

### Hook API (useReconStatus.ts)
- Cập nhật `useTableHistory` hook:
  ```typescript
  export function useTableHistory(
    table: string | null,
    shadowSchema?: string | null,
    masterTable?: string | null,
    pageSize = 30,
    excludeSmoke = false
  ) {
    return useQuery<{ data: ReconReport[]; total: number }>({
      queryKey: ['recon-history', table, shadowSchema, masterTable, pageSize, excludeSmoke],
      queryFn: async () => {
        const res = await cmsApi.get(`/api/reconciliation/report/${encodeURIComponent(table!)}`, {
          params: {
            page: 1,
            page_size: pageSize,
            ...(shadowSchema ? { shadow_schema: shadowSchema } : {}),
            ...(masterTable ? { master_table: masterTable } : {}),
            ...(excludeSmoke ? { exclude_smoke: 'true' } : {}),
          },
        });
        return res.data;
      },
      enabled: !!table,
      refetchInterval: 30000,
    });
  }
  ```

### Modal UI (ExecuteHealModal.tsx)
- Cập nhật cách gọi `useTableHistory`:
  `const { data: historyData, isLoading: isHistoryLoading } = useTableHistory(open ? table : null, shadowSchema, undefined, 100, true);`
- Bộ lọc `healedReports` lọc bỏ các bản ghi `ok`, `error`, và chỉ giữ lại những bản ghi thỏa mãn `isReportFullyHealed(r)`.
