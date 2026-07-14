# Yêu cầu Kiểm tra Metric Prometheus & SigNoz Dashboard

## Mục tiêu
- Kiểm tra các metric định nghĩa trong `centralized-data-service/pkgs/metrics/prometheus.go`.
- Kiểm tra các metric sử dụng trong SigNoz Dashboard configuration `centralized-data-service/deployments/signoz-dashboard-recon.json`.
- Xác định xem có metric nào dư thừa, không được sử dụng ở cả hai nơi, hoặc có sự không đồng bộ (ví dụ: metric có trong code nhưng không có trong dashboard, hoặc ngược lại).
- Xuất báo cáo chi tiết.
