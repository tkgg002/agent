# Phân tích kỹ thuật (Analysis) - Sửa lỗi hiển thị tổng record trong tab Pipeline

## 1. Nguyên nhân gốc rễ (Root Cause)
Trước thay đổi này, tab Pipeline trên giao diện Dashboard lấy dữ liệu tổng số lượng record của các trạm (Source, Shadow, Master) bằng cách kết hợp:
1. Trường `total_source_count` và `source_count` của `cdc_reconciliation_report`.
2. Trường `dest_count` của `cdc_reconciliation_report`.

Tuy nhiên, `cdc_reconciliation_report` là báo cáo ghi lại kết quả của các phiên kiểm tra đối soát, thường chạy theo một khoảng thời gian cố định hoặc một window/khung thời gian nhất định (ví dụ check hash window hoặc check full diff trong vòng 30 ngày qua). Vì vậy, `source_count` hay `dest_count` trong report này chỉ phản ánh số lượng record được kiểm tra trong khung thời gian đó (ví dụ chỉ có 8 record phát sinh và được đối soát), không phải là tổng số lượng thực tế của toàn bộ bảng.

Trong khi đó, kết quả kiểm tra smoke (`cdc_recon_smoke_result`) là nơi lưu trữ tổng số lượng record thực tế của toàn bộ bảng tại thời điểm gần nhất.

## 2. Giải pháp khắc phục
1. Loại bỏ hoàn toàn fallback sang dữ liệu của `cdc_reconciliation_report` ở Backend:
   * Sửa câu query lấy danh sách pipeline (`listLatestPrimary`) để các trường `source_total`, `source_active`, `shadow_total`, `shadow_active` trả về trực tiếp giá trị của lateral join với `cdc_recon_smoke_result`, không coalesce sang các trường tương ứng của `cdc_reconciliation_report`.
2. Điều chỉnh Frontend hiển thị:
   * Cập nhật `ReconPipelineGrid.tsx` để lấy `sourceTotal`, `shadowActive`, `masterActive` dựa trên các trường `source_active`/`source_total`, `shadow_active`/`shadow_total`, và `master_active`/`master_total` (chỉ lấy kết quả smoke check) thay vì dùng các trường window-based (`source_count`, `dest_count`).

Nhờ giải pháp này, cột record count hiển thị đúng số lượng toàn bộ bảng từ kết quả smoke check, độc lập hoàn toàn với kết quả kiểm tra của các report đối soát có khung thời gian cố định.
