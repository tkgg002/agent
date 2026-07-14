# Kế hoạch triển khai kiểm tra Prometheus Metrics & Dashboard

## Mục tiêu
Kiểm tra chéo giữa code Go (định nghĩa metric & nơi sử dụng metric) và cấu hình Dashboard SigNoz để tìm các metric bị dư thừa, không sử dụng hoặc thiếu đồng bộ.

## Các bước thực hiện
1. **Thu thập danh sách metric định nghĩa trong code Go:**
   - Đọc các biến metric trong file `prometheus.go`.
2. **Tìm nơi sử dụng (cập nhật) các metric này trong codebase:**
   - Dùng ripgrep quét toàn bộ codebase Go để tìm các tham chiếu đến các metric này (ví dụ: `metrics.EventsProcessed.WithLabelValues`, v.v.).
3. **Thu thập danh sách metric cấu hình trong SigNoz Dashboard:**
   - Trích xuất tên metric từ trường `"metricName"` trong file `signoz-dashboard-recon.json`.
4. **Đối chiếu và phát hiện:**
   - Metric định nghĩa trong code Go nhưng không có nơi nào sử dụng trong code Go (Unused in Code).
   - Metric có nơi sử dụng trong code Go nhưng không có trong Dashboard (Missing from Dashboard).
   - Metric có trong Dashboard nhưng không tồn tại hoặc không được sử dụng trong code Go (Dead Dashboard Metric).
5. **Tổng hợp kết quả báo cáo:**
   - Viết báo cáo tiếng Việt chi tiết chỉ ra rõ các metric dư thừa hoặc bất hợp lý.
