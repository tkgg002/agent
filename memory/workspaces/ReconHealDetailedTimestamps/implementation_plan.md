# Kế hoạch Bổ sung Thời gian Chữa lành Từng Loại Lỗi & Cải tiến Giao diện Lịch sử

Tài liệu này mô tả kế hoạch thiết kế và triển khai bổ sung 3 trường thời gian chữa lành độc lập (`healed_mismatched_at`, `healed_missing_src_at`, `healed_missing_dest_at`) cho 3 loại lỗi (`mismatched`, `missing_from_src`, `missing_from_dest`) ở cả database, backend và frontend.

---

## Proposed Changes

### 1. Database Migration (Postgres)
Chạy DDL cập nhật schema của bảng `cdc_system.cdc_reconciliation_report`:
```sql
ALTER TABLE cdc_system.cdc_reconciliation_report 
  ADD COLUMN healed_mismatched_at timestamp with time zone,
  ADD COLUMN healed_missing_src_at timestamp with time zone,
  ADD COLUMN healed_missing_dest_at timestamp with time zone;
```

### 2. Backend (centralized-data-service)

#### [MODIFY] [reconciliation_report.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go)
- Khai báo thêm 3 trường:
  - `HealedMismatchedAt` `*time.Time` (`gorm:"column:healed_mismatched_at" json:"healed_mismatched_at"`)
  - `HealedMissingSrcAt` `*time.Time` (`gorm:"column:healed_missing_src_at" json:"healed_missing_src_at"`)
  - `HealedMissingDestAt` `*time.Time` (`gorm:"column:healed_missing_dest_at" json:"healed_missing_dest_at"`)

#### [MODIFY] [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
- Trong hàm `finalizeReport`, bổ sung cập nhật các trường thời gian tương ứng nếu số lượng lỗi xử lý thực tế tương ứng > 0:
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

### 3. Backend (cdc-cms-service)

#### [MODIFY] [reconciliation_report.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/model/recon/reconciliation_report.go)
- Khai báo thêm 3 trường `HealedMismatchedAt`, `HealedMissingSrcAt`, `HealedMissingDestAt` tương ứng.

#### [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
- Cập nhật hàm `GetTableHistory`:
  - Trong `baseQuery`, select thêm 3 cột: `healed_mismatched_at`, `healed_missing_src_at`, `healed_missing_dest_at`.
  - Trong phần select thứ 2 (Smoke Check), trả về `NULL::timestamp without time zone` tương ứng cho cả 3 cột này để đồng bộ kiểu dữ liệu của `UNION ALL`.

### 4. Frontend (cdc-cms-web)

#### [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- Thêm định nghĩa 3 trường thời gian vào interface `UnhealedReport` và `ReconReport`:
  ```typescript
  healed_mismatched_at?: string | null;
  healed_missing_src_at?: string | null;
  healed_missing_dest_at?: string | null;
  ```

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- Cập nhật cấu hình `healedReportColumns` của bảng lịch sử:
  - Loại bỏ các cột "Kết quả xử lý" và "Thời gian xử lý" cũ.
  - Thêm 3 cột mới hiển thị độc lập:
    1. **Lệch dữ liệu (Mismatched)**:
       - Hiển thị: `{healed_mismatched_count}/{stale_count} ({healed_mismatched_duration_ms}ms)`.
       - Hiển thị ngày giờ `healed_mismatched_at` (nếu có) dưới định dạng `YYYY-MM-DD HH:mm:ss`.
    2. **Thừa ở Master (Missing from Src)**:
       - Hiển thị: `{pruned_missing_src_count}/{orphan_count} ({pruned_missing_src_duration_ms}ms)`.
       - Hiển thị ngày giờ `healed_missing_src_at` (nếu có).
    3. **Thiếu ở Master (Missing from Dest)**:
       - Hiển thị: `{healed_missing_dest_count}/{missing_count} ({healed_missing_dest_duration_ms}ms)`.
       - Hiển thị ngày giờ `healed_missing_dest_at` (nếu có).

---

## Verification Plan

### Automated Tests
1. Chạy các lệnh biên dịch để đảm bảo không lỗi cú pháp:
   ```bash
   # Centralized data service
   go build ./...
   # CDC CMS service
   go build ./cmd/server/...
   # Frontend React
   npx tsc --noEmit
   ```

### Manual Verification
1. Reset trạng thái report `export_jobs` về `drift`, gán lại các count lỗi.
2. Thực hiện heal từng phần (ví dụ chỉ chọn heal missing dest).
3. Xác nhận trên DB chỉ có `healed_missing_dest_at` được set timestamp, còn các trường khác vẫn `NULL`. Trạng thái tổng chuyển sang `partially_healed`.
4. Xem bảng lịch sử trên giao diện CMS. Xác nhận các cột Lệch dữ liệu, Thừa/Thiếu ở Master hiển thị chính xác kết quả xử lý (ví dụ: `6/6 (10ms)`) kèm thời gian hoàn thành cụ thể.
