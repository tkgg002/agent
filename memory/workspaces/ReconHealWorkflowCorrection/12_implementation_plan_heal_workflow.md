# Kế hoạch triển khai - Sửa đổi Luồng và Trạng thái Chữa lành Đối soát

Kế hoạch này sửa đổi cách cập nhật trạng thái của các phiên đối soát (Reconciliation Reports) sau khi chạy chữa lành, đảm bảo phiên chỉ được đánh dấu là `healed` khi tất cả 3 loại lỗi (Thiếu ở đích, Lệch dữ liệu, Thừa ở đích) được xử lý triệt để.

## User Review Required

> [!IMPORTANT]
> - **Chữa lành một phần (Partial Healing)**: Báo cáo đối soát sẽ không bị đánh dấu `healed_at != NULL` nếu người dùng chỉ chọn chữa lành một số loại lỗi cụ thể. Report sẽ vẫn hiển thị trong tab "Phiên chưa xử lý" với các số liệu còn lại (Remaining) được giảm trừ tương ứng.
> - **Bảo vệ Trạng thái `healing`**: Logic giải phóng claim khi gặp lỗi được gia cố thêm để tránh các tiến trình bị kẹt vĩnh viễn ở trạng thái `healing`.

## Proposed Changes

### Centralized Data Service (`centralized-data-service`)

#### [MODIFY] [reconciliation_report_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go)
- Sửa hàm `ReleaseHealClaim`:
  ```go
  if prevStatus == "" || prevStatus == "healing" {
      prevStatus = "drift"
  }
  ```

### CDC CMS Service (`cdc-cms-service`)

#### [MODIFY] [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/handler/recon/recon_execute_heal_handler.go)
- Sửa hàm `finalizeReport`:
  ```go
  func (h *ExecuteHealHandler) finalizeReport(ctx context.Context, rpt *modelrecon.ReconciliationReport) {
      now := time.Now().UTC()
      
      isFullyHealed := rpt.HealedMissingDestCount >= rpt.MissingCount &&
          rpt.HealedMismatchedCount >= rpt.StaleCount &&
          rpt.PrunedMissingSrcCount >= rpt.OrphanCount

      updates := map[string]any{
          "healed_mismatched_count":         rpt.HealedMismatchedCount,
          "healed_mismatched_duration_ms":   rpt.HealedMismatchedDurationMs,
          "healed_missing_dest_count":       rpt.HealedMissingDestCount,
          "healed_missing_dest_duration_ms": rpt.HealedMissingDestDurationMs,
          "pruned_missing_src_count":        rpt.PrunedMissingSrcCount,
          "pruned_missing_src_duration_ms":  rpt.PrunedMissingSrcDurationMs,
      }

      if isFullyHealed {
          updates["healed_at"] = now
          updates["status"] = "healed"
      } else {
          updates["healed_at"] = nil
          updates["status"] = "partially_healed"
      }

      _ = h.reportRepo.UpdateByID(ctx, rpt.ID, updates)
  }
  ```

### CDC CMS Web (`cdc-cms-web`)

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- Cập nhật định dạng render của các cột `missing_count`, `stale_count`, `orphan_count` trong bảng `reportColumns` để hiển thị dạng `Remaining / Original`.
- Vô hiệu hóa checkbox chữa lành nếu loại lỗi tương ứng không còn bản ghi nào cần xử lý.
- Cập nhật `useEffect` khi modal mở để xác định động các checkbox được chọn theo thực trạng lỗi còn tồn đọng.

## Verification Plan

### Automated Tests
- Chạy biên dịch toàn bộ backend và frontend:
  - Backend: `make build` hoặc `go build ./...`
  - Frontend: `npm run build`
- Viết test/kiểm tra thủ công trên UI để đảm bảo:
  - Khi chỉ chọn chữa lành 1 phần, phiên đối soát vẫn nằm trong tab "Phiên chưa xử lý" và cập nhật đúng số còn lại.
  - Khi chữa lành hết 100%, phiên sẽ chuyển sang tab "Phiên đã xử lý".
