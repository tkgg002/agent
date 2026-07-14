# Báo Cáo Thay Đổi Mã Nguồn - Bổ sung Thời gian Chữa lành Từng Loại Lỗi

Báo cáo chi tiết các file đã sửa đổi, số lượng dòng code và tóm tắt thay đổi.

## 1. Tóm tắt các file đã thay đổi

| File Path | Dòng thay đổi (Ước lượng) | Nội dung thay đổi chính |
| :--- | :---: | :--- |
| `centralized-data-service/internal/model/recon/reconciliation_report.go` | +3 lines | Thêm 3 trường `HealedMismatchedAt`, `HealedMissingSrcAt`, `HealedMissingDestAt` vào struct `ReconciliationReport` để GORM ánh xạ. |
| `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go` | +10 lines | Cập nhật hàm `finalizeReport` để gán giá trị thời gian xử lý thực tế khi số bản ghi chữa lành tương ứng lớn hơn 0. |
| `cdc-cms-service/internal/model/recon/reconciliation_report.go` | +3 lines | Đồng bộ thêm 3 trường thời gian chữa lành tương ứng vào struct `ReconciliationReport` phía CMS. |
| `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go` | +6 lines | Cập nhật `baseQuery` và `unionQuery` (Smoke Check query) để select các trường mới từ DB (trả về NULL cho smoke check). |
| `cdc-cms-web/src/hooks/useReconStatus.ts` | +6 lines | Bổ sung 3 trường vào interface `UnhealedReport` và `ReconReport` trên frontend hook. |
| `cdc-cms-web/src/components/ExecuteHealModal.tsx` | +35 lines | Viết helper `formatTimestamp` hiển thị định dạng `YYYY-MM-DD HH:mm:ss` và cập nhật `healedReportColumns` để hiển thị 3 cột kết quả chữa lành chi tiết. |

---

## 2. Chi tiết các thay đổi

### 2.1 Backend Models & Logic
- **`centralized-data-service`**:
  - Struct `ReconciliationReport`:
    ```go
    HealedMismatchedAt  *time.Time `gorm:"column:healed_mismatched_at" json:"healed_mismatched_at"`
    HealedMissingSrcAt   *time.Time `gorm:"column:healed_missing_src_at" json:"healed_missing_src_at"`
    HealedMissingDestAt  *time.Time `gorm:"column:healed_missing_dest_at" json:"healed_missing_dest_at"`
    ```
  - Logic gán trong `finalizeReport`:
    ```go
    if rpt.HealedMismatchedCount > 0 {
        updates["healed_mismatched_at"] = now
    }
    if rpt.HealedMissingDestCount > 0 {
        updates["healed_missing_dest_at"] = now
    }
    if rpt.PrunedMissingSrcCount > 0 {
        updates["healed_missing_src_at"] = now
    }
    ```

- **`cdc-cms-service`**:
  - Đồng bộ struct model `ReconciliationReport` thêm 3 trường tương tự.
  - Cập nhật câu select query trong `GetTableHistory`:
    - `baseQuery`: select thêm `healed_mismatched_at`, `healed_missing_src_at`, `healed_missing_dest_at`.
    - `unionQuery` (smoke check): select `NULL::timestamp without time zone AS healed_mismatched_at`, `NULL::timestamp without time zone AS healed_missing_src_at`, `NULL::timestamp without time zone AS healed_missing_dest_at`.

### 2.2 Frontend (cdc-cms-web)
- Thêm thuộc tính vào interface `ReconReport` và `UnhealedReport`:
  ```typescript
  healed_mismatched_at?: string | null;
  healed_missing_src_at?: string | null;
  healed_missing_dest_at?: string | null;
  ```
- Thay thế các cột cũ trong bảng đã xử lý (`healedReportColumns`) bằng 3 cột:
  1. **Lệch dữ liệu (Mismatched)**: Hiển thị `{r.healed_mismatched_count}/{r.stale_count} ({r.healed_mismatched_duration_ms}ms)` và timestamp `healed_mismatched_at`.
  2. **Thừa ở Master (Missing from Src)**: Hiển thị `{r.pruned_missing_src_count}/{r.orphan_count} ({r.pruned_missing_src_duration_ms}ms)` và timestamp `healed_missing_src_at`.
  3. **Thiếu ở Master (Missing from Dest)**: Hiển thị `{r.healed_missing_dest_count}/{r.missing_count} ({r.healed_missing_dest_duration_ms}ms)` và timestamp `healed_missing_dest_at`.
- Định dạng ngày giờ hiển thị trên cột đã chuyển sang `YYYY-MM-DD HH:mm:ss` thông qua hàm helper JS `formatTimestamp`.

---

## 3. Trạng thái Kiểm thử & Biên dịch
- **`centralized-data-service` (`go build ./cmd/...`)**: Biên dịch thành công 🟢
- **`cdc-cms-service` (`go build ./cmd/server/...`)**: Biên dịch thành công 🟢
- **`cdc-cms-web` (`npx tsc --noEmit`)**: Biên dịch & kiểm tra kiểu thành công 🟢
