# 08_tasks_phase_3.md - Danh sách Task chi tiết Phase 3

## 1. Hạng mục TX-C3: Silent Rule Drop
- [x] Định nghĩa Prometheus metric `cdc_transmute_rule_dropped_total` trong `prometheus.go`.
- [x] Thêm Warn log chi tiết và tăng metric counter trong `transmuter.go` ở hàm `loadRules()`.

## 2. Hạng mục SINK-H5: Fallback Protection
- [x] Thêm import `"strings"` trong `batch_buffer.go`.
- [x] Viết helper `isRetryableDBError` trong `batch_buffer.go`.
- [x] Sửa đổi vòng lặp sequential fallback trong `batch_buffer.go`: Nếu gặp lỗi transient, lập tức trả về error thay vì ghi DLQ.

## 3. Hạng mục Phân tích & Nghiên cứu (TX-H3 & TX-H6)
- [x] Nghiên cứu giải pháp cho OCC clock skew và FNV-1a collision.
- [x] Lưu báo cáo nghiên cứu chi tiết vào `13_analysis_risks_phase_3.md`.
