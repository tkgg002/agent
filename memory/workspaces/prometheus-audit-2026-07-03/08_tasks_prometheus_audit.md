# Danh sách Tasks chi tiết

- [x] Lập kế hoạch phân tích và tìm kiếm các metric trong code Go
- [x] Phân tích các metric được định nghĩa trong `prometheus.go`
- [x] Tìm các metric thực tế được sử dụng trong codebase Go (grep các biến metric trong `pkgs/metrics/prometheus.go` để xem chỗ nào gọi)
- [x] Phân tích các metric trong `signoz-dashboard-recon.json`
- [x] So sánh chéo để tìm:
  1. Metric định nghĩa trong code nhưng không được cập nhật/sử dụng ở đâu trong Go codebase.
  2. Metric định nghĩa trong code nhưng không hiển thị trên Dashboard.
  3. Metric hiển thị trên Dashboard nhưng không tồn tại trong code.
- [x] Viết báo cáo chi tiết và cập nhật tiến độ.
