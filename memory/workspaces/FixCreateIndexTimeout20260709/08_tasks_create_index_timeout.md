# Danh sách Task chi tiết - Tối ưu hóa bất đồng bộ Create/Drop Index

- [x] Task 1: Nghiên cứu cấu trúc code và thiết kế giải pháp
  - [x] Đọc và hiểu cấu trúc `index_handler.go` và `index_manager.go`
  - [x] Thiết kế luồng xử lý bất đồng bộ bằng goroutine trong `IndexHandler`
- [x] Task 2: Viết kế hoạch triển khai chi tiết (`12_implementation_plan_create_index_timeout.md`) và gửi đề xuất cho User
- [x] Task 3: Ủy quyền Muscle thực thi thay đổi code
  - [x] Cập nhật logic `HandleCreateIndex` trong `index_handler.go` chạy bất đồng bộ trong một goroutine và phản hồi ngay lập tức cho NATS.
  - [x] Cập nhật logic `HandleDropIndex` trong `index_handler.go` chạy bất đồng bộ tương tự.
  - [x] Đảm bảo span được đóng đúng cách và xử lý logging khi chạy dưới nền.
  - [x] Thêm cache `ensuredShadowIndexes` vào `TransmuterModule` và tối ưu hóa bất đồng bộ `ensureShadowSourceIDIndex` trong `transmuter.go`.
  - [x] Chạy `go test` biên dịch thành công.
- [x] Task 4: Chạy linter quy trình và hoàn thành DoD
  - [x] Báo cáo thay đổi trong `11_report_create_index_timeout.md`
  - [x] Tạo walkthrough báo cáo kết quả `14_walkthrough_create_index_timeout.md`
  - [x] Chạy `python3 tooling/verify_governance.py`
