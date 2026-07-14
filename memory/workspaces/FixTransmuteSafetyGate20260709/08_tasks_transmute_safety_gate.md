# Danh sách Task chi tiết - Khắc phục transmute safety gate batchSize

- [x] Task 1: Nghiên cứu cấu trúc code và thiết kế phương án chunking
  - [x] Đọc và hiểu logic tại `transmuter.go`
  - [x] Xác minh sự phụ thuộc của `onlySourceIDs` và cách `fetchShadowBatch` truy vấn shadow DB
  - [x] Lập kế hoạch thiết kế giải pháp phân lô (chunking)
- [x] Task 2: Viết kế hoạch triển khai chi tiết (`12_implementation_plan_transmute_safety_gate.md`) và xin approval của User
- [x] Task 3: Ủy quyền Muscle thực thi thay đổi code
  - [x] Tạo test tái hiện lỗi / hoặc test case kiểm thử trước khi sửa (Red)
  - [x] Sửa logic hàm `Run` của `TransmuterModule` trong `transmuter.go` để chia nhỏ `onlySourceIDs` thành nhiều chunk có kích thước tối đa `batchSize` (2000)
  - [x] Sửa / Xóa safety gate kiểm tra kích thước `onlySourceIDs` ở đầu hàm `Run`
  - [x] Chạy test case để kiểm thử kết quả (Green)
- [x] Task 4: Chạy linter quy trình và hoàn thành DoD
  - [x] Chạy test suite của `master` package
  - [x] Rà soát lại bài học cũ
  - [x] Ghi nhận bài học mới (nếu có)
  - [x] Tạo walkthrough báo cáo kết quả
  - [x] Chạy `python3 agent/tooling/verify_governance.py`
