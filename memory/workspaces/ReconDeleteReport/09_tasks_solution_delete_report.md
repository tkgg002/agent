# Hồ sơ Giải pháp: Thêm Chức năng Xoá Phiên Đối Soát (cdc_reconciliation_report) & Chọn Phiên Chữa Lành

Hồ sơ thiết kế chi tiết cho việc sửa đổi mã nguồn ở cả backend và frontend.

## A. Backend (cdc-cms-service)

### 1. `internal/api/recon/reconciliation_handler.go`
- Import `"gorm.io/gorm"`.
- Thêm thuộc tính `db *gorm.DB` vào struct `ReconciliationHandler`.
- Thay đổi hàm khởi tạo để tiêm `db *gorm.DB`:
```go
type ReconciliationHandler struct {
	reader         recon.ReconReader
	bus            ports.CommandBus
	listLatestQ    *recon.ListLatestReportsHandler
	getHistoryQ    *recon.GetTableHistoryHandler
	listFailedQ    *recon.ListFailedLogsHandler
	listUnhealedQ  *recon.ListUnhealedReportsHandler
	activityLogger ports.ActivityLogger
	logger         *zap.Logger
	db             *gorm.DB // <-- Thêm
}

func NewReconciliationHandler(
	reader recon.ReconReader,
	bus ports.CommandBus,
	listLatestQ *recon.ListLatestReportsHandler,
	getHistoryQ *recon.GetTableHistoryHandler,
	listFailedQ *recon.ListFailedLogsHandler,
	listUnhealedQ *recon.ListUnhealedReportsHandler,
	activityLogger ports.ActivityLogger,
	db *gorm.DB, // <-- Thêm
	loggers ...*zap.Logger,
) *ReconciliationHandler {
	logger := zap.NewNop()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &ReconciliationHandler{
		reader:         reader,
		bus:            bus,
		listLatestQ:    listLatestQ,
		getHistoryQ:    getHistoryQ,
		listFailedQ:    listFailedQ,
		listUnhealedQ:  listUnhealedQ,
		activityLogger: activityLogger,
		logger:         logger,
		db:             db, // <-- Thêm
	}
}
```

### 2. Tạo file `internal/api/recon/reconciliation_handler_delete_report.go`
Tạo file mới chứa handler DELETE:
```go
package recon

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// DeleteReport deletes a cdc_reconciliation_report entry by ID.
// DELETE /api/reconciliation/report/:id
func (h *ReconciliationHandler) DeleteReport(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid report ID"})
	}

	if h.db == nil {
		return c.Status(500).JSON(fiber.Map{"error": "database connection not wired"})
	}

	// Xoá bản ghi khỏi cdc_system.cdc_reconciliation_report
	if err := h.db.WithContext(c.UserContext()).Exec("DELETE FROM cdc_system.cdc_reconciliation_report WHERE id = ?", id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Report deleted successfully",
		"id":      id,
	})
}
```

### 3. `internal/server/server.go`
Truyền `db` vào hàm khởi tạo Reconciliation Handler tại dòng 346:
```go
	h.Recon = apirecon.NewReconciliationHandler(reconReader, cmdBus, listLatestReportsH, getTableHistoryH, listFailedLogsH, listUnhealedReportsH, activityLogger, db, logger)
```

### 4. `internal/router/router.go`
Đăng ký các route DELETE sau nhóm destructive check:
```go
	registerDestructive("/reconciliation/check", h.Recon.TriggerCheckAll)
    ...
	api.Delete("/reconciliation/report/:id", append(destructiveChain, h.Recon.DeleteReport)...)
	api.Delete("/v1/reconciliation/report/:id", append(destructiveChain, h.Recon.DeleteReport)...)
```

---

## B. Frontend (cdc-cms-web)

### 1. `cdc-cms-web/src/hooks/useReconStatus.ts`
- Cập nhật hàm helper `auditHeaders` để encode header value chống lỗi setRequestHeader trong browser khi có ký tự tiếng Việt:
```typescript
function auditHeaders(reason: string) {
  return {
    'Idempotency-Key': newIdempotencyKey(),
    'X-Action-Reason': encodeURIComponent(reason),
  };
}
```

- Bổ sung `useDeleteReportMutation` (sử dụng lý do mặc định `"Xóa phiên đối soát"` để bỏ qua bắt buộc nhập lý do thủ công):
```typescript
export function useDeleteReportMutation() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, { id: number }>({
    mutationFn: async ({ id }) => {
      await cmsApi.delete(`/api/reconciliation/report/${id}`, {
        headers: auditHeaders("Xóa phiên đối soát"),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['unhealed-reports'] });
      queryClient.invalidateQueries({ queryKey: ['recon-history'] });
      queryClient.invalidateQueries({ queryKey: ['recon-report'] });
    },
    retry: 0,
  });
}
```

- Cập nhật `useExecuteHealMutation` để tự động invalidates cache của các queries sau khi heal thành công (giúp danh sách phiên tự động refresh):
```typescript
export function useExecuteHealMutation() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, ExecuteHealPayload & { reason: string }>({
    mutationFn: async ({
      table,
      segment,
      report_ids,
      heal_mismatched,
      heal_missing_dest,
      prune_missing_src,
      force_heal,
      reason,
    }) => {
      await cmsApi.post(
        '/api/reconciliation/execute-heal',
        {
          table,
          segment: segment || undefined,
          report_ids,
          heal_mismatched,
          heal_missing_dest,
          prune_missing_src,
          force_heal: force_heal || false,
        },
        { headers: auditHeaders(reason) },
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['unhealed-reports'] });
      queryClient.invalidateQueries({ queryKey: ['recon-history'] });
      queryClient.invalidateQueries({ queryKey: ['recon-report'] });
    },
    retry: 0,
  });
}
```

### 2. `cdc-cms-web/src/components/ExecuteHealModal.tsx`
- Định nghĩa mảng rỗng tĩnh ngoài component để tránh tạo mới reference gây re-render loop:
  ```typescript
  const EMPTY_ARRAY: any[] = [];
  ```
- Khai báo gán reports sử dụng `EMPTY_ARRAY`:
  ```typescript
  const reports = data?.data || EMPTY_ARRAY;
  ```
- Cập nhật bộ lọc `healedReports` để lấy các phiên có trạng thái `healed` hoặc `partially_healed` (hoặc đã ghi nhận số bản ghi đã xử lý/dọn dẹp > 0) thay vì chỉ lọc theo `healed_at != null` (do các phiên `partially_healed` có `healed_at = null` trong DB):
  ```typescript
  const healedReports = (historyData?.data || []).filter(
    (r: any) =>
      r.healed_at != null ||
      r.status === 'healed' ||
      r.status === 'partially_healed' ||
      (r.healed_count ?? 0) > 0 ||
      (r.pruned_missing_src_count ?? 0) > 0
  );
  ```
- Thêm state `selectedRowKeys` để quản lý danh sách ID phiên được chọn chữa lành:
  ```typescript
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  ```
- Khởi tạo mặc định chọn tất cả các phiên khi danh sách `reports` thay đổi, sử dụng `data` ổn định làm dependency:
  ```typescript
  useEffect(() => {
    if (open && reports.length > 0) {
      setSelectedRowKeys(reports.map(r => r.id));
    } else {
      setSelectedRowKeys([]);
    }
  }, [open, data]); // Dùng data thay vì reports để tránh re-render loop!
  ```
- Thiết lập checkbox mặc định cũng dùng `data` ổn định làm dependency:
  ```typescript
  useEffect(() => {
    if (open && reports.length > 0) {
      const hasRemainingMismatched = reports.some(
        (r) => r.stale_count - (r.healed_mismatched_count || 0) > 0
      );
      const hasRemainingMissingDest = reports.some(
        (r) => r.missing_count - (r.healed_missing_dest_count || 0) > 0
      );

      setHealMismatched(hasRemainingMismatched);
      setHealMissingDest(hasRemainingMissingDest);
      setPruneMissingSrc(false);
    }
  }, [open, data]); // Dùng data thay vì reports để tránh re-render loop!
  ```
- Bổ sung cấu hình `rowSelection` cho Ant Design `<Table />`:
  ```typescript
  const rowSelection = {
    selectedRowKeys,
    onChange: (newSelectedRowKeys: React.Key[]) => {
      setSelectedRowKeys(newSelectedRowKeys);
    },
  };
  ```
- Cập nhật logic `isFormValid` yêu cầu `selectedRowKeys.length > 0`:
  ```typescript
  const isFormValid = isReasonValid && hasCheckbox && selectedRowKeys.length > 0;
  ```
- Cập nhật logic `executeHeal` để truyền đúng danh sách ID đã chọn:
  ```typescript
  const reportIds = selectedRowKeys.map(Number);
  ```
- Truyền `rowSelection` vào `<Table />` của tab `"unhealed"`:
  ```typescript
  <Table
    size="small"
    rowKey="id"
    rowSelection={rowSelection}
    dataSource={reports}
    columns={reportColumns}
    pagination={false}
    scroll={{ y: 200 }}
    style={{ marginBottom: 4 }}
  />
  ```
- Cập nhật hàm `handleDeleteReport` để không yêu cầu lý do thủ công và gọi thẳng `deleteMutation.mutateAsync({ id })`.
