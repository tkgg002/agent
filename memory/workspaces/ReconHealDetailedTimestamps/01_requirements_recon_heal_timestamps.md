# Yêu cầu - Bổ sung Thời gian Chữa lành Từng Loại Lỗi & Cải tiến Giao diện Lịch sử

Bổ sung các trường lưu trữ thời gian chữa lành độc lập cho 3 loại lỗi (`mismatched`, `missing_from_src`, `missing_from_dest`) và cập nhật giao diện hiển thị chi tiết lịch sử xử lý tương ứng trên CMS.

## Yêu cầu chi tiết
1. **Database Migration**:
   - Thêm các trường sau vào bảng `cdc_system.cdc_reconciliation_report`:
     - `healed_mismatched_at` (timestamp with time zone, nullable): Thời điểm hoàn thành chữa lành lệch dữ liệu.
     - `healed_missing_src_at` (timestamp with time zone, nullable): Thời điểm hoàn thành dọn dẹp bản ghi thừa ở Master.
     - `healed_missing_dest_at` (timestamp with time zone, nullable): Thời điểm hoàn thành chữa lành bản ghi thiếu ở Master.

2. **Backend Model (Go)**:
   - Cập nhật struct `ReconciliationReport` trong cả `cdc-cms-service` và `centralized-data-service` để khai báo thêm 3 trường này.

3. **Heal Execution updates (`centralized-data-service`)**:
   - Trong logic chữa lành của transmuter worker, cập nhật gán timestamp tương ứng cho các trường khi thực hiện chữa lành thành công:
     - Nếu `rpt.HealedMismatchedCount > 0` -> Gán `healed_mismatched_at = time.Now()`.
     - Nếu `rpt.HealedMissingDestCount > 0` -> Gán `healed_missing_dest_at = time.Now()`.
     - Nếu `rpt.PrunedMissingSrcCount > 0` -> Gán `healed_missing_src_at = time.Now()`.

4. **API Union Query (`cdc-cms-service`)**:
   - Cập nhật câu truy vấn `unionQuery` trong `GetTableHistory` (`recon_read_repo_gorm.go`) để select thêm 3 trường này. Trong phần select của Smoke Check trả về `NULL::timestamp without time zone` tương ứng.

5. **Frontend UI (`cdc-cms-web`)**:
   - Cập nhật interface `ReconReport` và `UnhealedReport` trong `useReconStatus.ts` để bổ sung 3 trường thời gian mới.
   - Ở modal `ExecuteHealModal.tsx` -> tab "Phiên đã xử lý" -> Cập nhật các cột hiển thị:
     - Thay thế cột "Kết quả xử lý" và "Thời gian xử lý" cũ thành 3 cột hiển thị độc lập:
       - **Lệch dữ liệu (Mismatched)**: Hiển thị kết quả dạng `healed_mismatched_count/stale_count (duration_ms)` và mốc thời gian xử lý `healed_mismatched_at` tương ứng (nếu có).
       - **Thừa ở Master (Missing from Src)**: Hiển thị kết quả dạng `pruned_missing_src_count/orphan_count (duration_ms)` và mốc thời gian xử lý `healed_missing_src_at` tương ứng (nếu có).
       - **Thiếu ở Master (Missing from Dest)**: Hiển thị kết quả dạng `healed_missing_dest_count/missing_count (duration_ms)` và mốc thời gian xử lý `healed_missing_dest_at` tương ứng (nếu có).
