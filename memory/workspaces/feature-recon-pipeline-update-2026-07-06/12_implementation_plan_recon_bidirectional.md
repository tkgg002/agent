# Kế hoạch Triển khai - Đối soát hai chiều (Bidirectional Reconciliation)

Kế hoạch này chi tiết hóa các bước sửa đổi code để triển khai cơ chế đối soát hai chiều, giúp phát hiện cả bản ghi thiếu (missing) và bản ghi dư thừa (stale) trong quá trình đối soát full_diff.

## 1. Các file cần thay đổi

### Centralized Data Service

#### [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
- Sửa đổi hàm `TimeBoundedDiffMissingFromShadow`:
  - Thay đổi signature của hàm:
    `func (rc *ReconCore) TimeBoundedDiffMissingFromShadow(ctx context.Context, entry source.TableRegistry, startTime, endTime time.Time) (missing []string, stale []string, srcCount int, err error)`
  - Trong logic xử lý:
    - Khi nhận được ID từ source stream:
      - Tăng `srcCount`.
      - Nếu ID tồn tại trong `shadowSet`, gọi `delete(shadowSet, id)`.
      - Nếu không tồn tại, append ID vào `missing`.
    - Sau khi kết thúc stream (nếu không có lỗi):
      - Duyệt qua `shadowSet` và thu thập tất cả các ID còn lại vào slice `stale`.
      - Trả về `missing, stale, srcCount, nil`.

#### [MODIFY] [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
- Cập nhật lệnh gọi `TimeBoundedDiffMissingFromShadow`:
  - `missingIDs, staleIDs, srcCount, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctx, *entry, startTimeVal, endTimeVal)`
- Cập nhật logic tính toán `status`:
  - `if len(missingIDs) > 0 || len(staleIDs) > 0 { status = "drift" }`
- Cập nhật logic tạo `ReconciliationReport`:
  - `StaleCount: len(staleIDs)`
  - `StaleIDs: json.RawMessage(staleIDsBytes)`
  - `Diff: int64(len(missingIDs) + len(staleIDs))`

#### [MODIFY] [recon_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_handler.go)
- Cập nhật dòng 574 để nhận 4 giá trị trả về:
  `missing, stale, srcTotal, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctxDiff, *entry, start, end)`
- Ghi log nếu phát hiện bản ghi stale để hỗ trợ vận hành.

## 2. Kịch bản kiểm thử (Verification Plan)
- Chạy biên dịch toàn bộ codebase để đảm bảo không lỗi cú pháp:
  `go build ./cmd/...`
- Chạy unit test trong package `internal/service/recon/...` và `internal/handler/recon/...` để đảm bảo logic hoạt động chính xác.
