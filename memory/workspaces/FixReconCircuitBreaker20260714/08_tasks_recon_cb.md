# Danh sách Task chi tiết

- [x] Phân tích và truy vết nguyên nhân gây Circuit Breaker Open
  - [x] Truy vấn registry để kiểm tra URL MongoDB của `scheduler-service`
  - [x] Kiểm tra các index hiện có của collection `schedule_histories` trong MongoDB
  - [x] Xem xét logs của `cdc-worker` hoặc NATS liên quan tới lỗi `source max ts`
- [x] Đề xuất thiết kế giải pháp
  - [x] Tạo index trên `lastUpdatedAt` của `schedule_histories`
  - [x] Tối ưu hóa `pickScanRangeWithLag` hoặc cơ chế fallback khi `MaxWindowTs` gặp lỗi circuit breaker
- [x] Thực hiện giải pháp (Ủy quyền cho Muscle)
  - [x] Viết script migrate/tạo index cho MongoDB
  - [x] Chạy lệnh tạo index và xác thực index đã được tạo
  - [x] Cập nhật code Go (nếu cần cải tiến khả năng chịu tải hoặc fallback)
- [x] Kiểm tra và nghiệm thu
  - [x] Chạy thử lệnh đối soát 7 ngày cho `source_shadow` của `schedule_histories`
  - [x] Kiểm tra trạng thái Circuit Breaker chuyển sang CLOSED / hoạt động bình thường
  - [x] Sửa lỗi giao diện Master / Shadow Registry reset trạng thái Collapse khi reload trang
  - [x] Viết báo cáo nghiệm thu và cập nhật bài học rút ra

