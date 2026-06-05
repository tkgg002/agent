# Bối cảnh (Context)

## Lỗi 1: Panic trong Kafka Consumer khi không có topic
- **Hiện tượng**: Worker bị restart liên tục kèm log crash: `panic: either Topic or GroupTopics must be specified with GroupID`.
- **Nguyên nhân**: Khi danh sách topic mới trống (`new: []`), hàm `buildReader` vẫn được gọi và cố khởi tạo `kafka.NewReader` của thư viện `github.com/segmentio/kafka-go` với danh sách topic rỗng, dẫn đến panic.
- **Mục tiêu**: Ngăn chặn panic này. Nếu không có topic nào được cấu hình, đóng reader hiện tại (nếu có) và không khởi tạo reader mới, đảm bảo worker không crash.

## Lỗi 2: Relation "failed_sync_logs" does not exist trong CMS Service
- **Hiện tượng**: Logs của `cdc-cms-service` báo lỗi `ERROR: relation "failed_sync_logs" does not exist (SQLSTATE 42P01)` liên tục khi chạy các câu lệnh sức khỏe hệ thống.
- **Nguyên nhân**: File `system_health_queries.go` trong CMS service đang truy vấn trực tiếp bảng `"failed_sync_logs"` mà không khai báo schema `"cdc_system".` một cách tường minh.
- **Mục tiêu**: Thêm prefix schema `"cdc_system".` cho các truy vấn sức khỏe hệ thống trong CMS Service.
