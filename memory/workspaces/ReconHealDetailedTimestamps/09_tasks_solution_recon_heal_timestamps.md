# Hồ Sơ Giải Pháp Kỹ Thuật - Bổ sung Thời gian Chữa lành Từng Loại Lỗi

Tài liệu thiết kế chi tiết các thay đổi trong database, backend và frontend cho tính năng tách biệt thời gian chữa lành.

## 1. Database Migration
Thực hiện chạy câu lệnh SQL trên container `gpay-postgres-cdc`:
```sql
ALTER TABLE cdc_system.cdc_reconciliation_report 
  ADD COLUMN healed_mismatched_at timestamp with time zone,
  ADD COLUMN healed_missing_src_at timestamp with time zone,
  ADD COLUMN healed_missing_dest_at timestamp with time zone;
```

## 2. Thay đổi Backend

### centralized-data-service
- **Model (`internal/model/recon/reconciliation_report.go`)**:
  Khai báo thêm 3 cột:
  ```go
  HealedMismatchedAt  *time.Time `gorm:"column:healed_mismatched_at" json:"healed_mismatched_at"`
  HealedMissingSrcAt   *time.Time `gorm:"column:healed_missing_src_at" json:"healed_missing_src_at"`
  HealedMissingDestAt  *time.Time `gorm:"column:healed_missing_dest_at" json:"healed_missing_dest_at"`
  ```
- **Execution logic (`internal/handler/recon/recon_execute_heal_handler.go`)**:
  Trong hàm `finalizeReport`:
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

### cdc-cms-service
- **Model (`internal/model/recon/reconciliation_report.go`)**:
  Khai báo thêm 3 cột tương tự model của centralized-data-service.
- **Repository (`internal/infra/persistence/recon/recon_read_repo_gorm.go`)**:
  Cập nhật `baseQuery` trong `GetTableHistory` để select thêm 3 cột:
  `healed_mismatched_at`, `healed_missing_src_at`, `healed_missing_dest_at`.
  Trong câu select thứ 2 (Smoke Check), trả về `NULL::timestamp without time zone` tương ứng cho cả 3 cột này.

---

## 3. Thay đổi Frontend (cdc-cms-web)

### Hook definitions (`src/hooks/useReconStatus.ts`)
- Thêm các trường vào interface `UnhealedReport` và `ReconReport`:
  ```typescript
  healed_mismatched_at?: string | null;
  healed_missing_src_at?: string | null;
  healed_missing_dest_at?: string | null;
  ```

### Modal UI columns (`src/components/ExecuteHealModal.tsx`)
- Thay thế các cột của bảng lịch sử `healedReportColumns`:
  - Loại bỏ cột "Kết quả xử lý" và "Thời gian xử lý" cũ.
  - Thêm 3 cột mới:
    - **Lệch dữ liệu (Mismatched)**:
      - Render: Hiển thị `{r.healed_mismatched_count}/{r.stale_count} ({r.healed_mismatched_duration_ms}ms)`.
      - Hiển thị ngày giờ `healed_mismatched_at` (nếu có) định dạng `YYYY-MM-DD HH:mm:ss`.
    - **Thừa ở Master (Orphan)**:
      - Render: Hiển thị `{r.pruned_missing_src_count}/{r.orphan_count} ({r.pruned_missing_src_duration_ms}ms)`.
      - Hiển thị ngày giờ `healed_missing_src_at` (nếu có).
    - **Thiếu ở Master (Missing Dest)**:
      - Render: Hiển thị `{r.healed_missing_dest_count}/{r.missing_count} ({r.healed_missing_dest_duration_ms}ms)`.
      - Hiển thị ngày giờ `healed_missing_dest_at` (nếu có).
