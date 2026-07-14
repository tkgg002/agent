# Yêu cầu chi tiết - Sửa lỗi hiển thị tổng record trong tab Pipeline

## 1. Bối cảnh & Mục tiêu
Hiện tại, khi hiển thị thông tin pipeline, các cột "Source (recs)", "Shadow (recs)", và "Master (recs)" đang hiển thị sai số lượng bản ghi (ví dụ hiển thị số 8 sau khi chạy Full Search/Full Diff, vốn là số lượng bản ghi bị lệch trong khung thời gian quét). Điều này là do hệ thống đang lấy nhầm dữ liệu từ `cdc_reconciliation_report` (vốn là report chạy theo khung thời gian cố định).

Yêu cầu là:
*   Tổng số lượng record (tổng record thật) chỉ được lấy từ bảng `cdc_recon_smoke_result` (bảng lưu kết quả check smoke toàn bộ table).
*   Không lấy/tham chiếu các trường đếm số lượng của `cdc_reconciliation_report` làm tổng số record của bảng.

## 2. Chi tiết yêu cầu
*   **Backend (`cdc-cms-service`):**
    *   Trong query `listLatestPrimary` ở file `recon_read_repo_gorm.go`, phần LEFT JOIN LATERAL với `cdc_recon_smoke_result` (`s`), loại bỏ các hàm `COALESCE` dùng để fallback sang các trường đếm của `cdc_reconciliation_report` (`r.total_source_count`, `r.source_count`, `r.total_dest_count`, `r.dest_count`). Mốc đếm thực tế của bảng phải lấy trực tiếp từ `s.source_total`, `s.source_active`, `s.shadow_total`, `s.shadow_active`.
*   **Frontend (`cdc-cms-web`):**
    *   Trong component `ReconPipelineGrid.tsx`, sửa các biến tính toán:
        *   `sourceTotal`: Chuyển từ `a.total_source_count ?? a.source_count` sang `a.source_active ?? a.source_total` (lấy từ smoke result).
        *   `shadowActive`: Chuyển sang sử dụng `a.shadow_active ?? a.shadow_total ?? b.source_active ?? b.source_total`.
        *   `masterActive`: Chuyển sang sử dụng `b.master_active ?? b.master_total`.
